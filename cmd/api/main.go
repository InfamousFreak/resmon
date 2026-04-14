package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"resmon/internal/database"
	"resmon/internal/llm"
	"resmon/pkg/models"
)

func getBadge(edgeType string, score int) string {
	normalized := strings.TrimSpace(edgeType)

	if score >= 70 {
		switch strings.ToLower(normalized) {
		case "speed":
			return "SPEED OPTIMIZED"
		case "memory", "efficiency":
			return "MEMORY OPTIMIZED"
		case "energy":
			return "ENERGY OPTIMIZED"
		default:
			return "EFFICIENCY FOCUSED"
		}
	}

	if score >= 40 {
		return "MENTIONS EFFICIENCY"
	}

	return "NOT EFFICIENCY FOCUSED"
}

// The handler to fetch the latest high-efficiency papers
func getLatestPapers(w http.ResponseWriter, r *http.Request) {
	// Query the database for the top 20 most recently parsed papers, ordered by efficiency
	rows, err := database.Pool.Query(context.Background(),
		`SELECT arxiv_id, title, abstract, efficiency_score, speedup, hardware, is_implementable, github_url, edge_type, is_theoretical
		 FROM papers 
		 ORDER BY efficiency_score DESC NULLS LAST, arxiv_id DESC
		 LIMIT 50`)

	if err != nil {
		http.Error(w, "Failed to query database", http.StatusInternalServerError)
		log.Printf("DB Query Error: %v", err)
		return
	}
	defer rows.Close()

	var papers []models.Paper

	// Loop through the rows and append them to our slice
	for rows.Next() {
		var p models.Paper
		err := rows.Scan(
			&p.ArxivID, &p.Title, &p.Abstract,
			&p.EfficiencyScore, &p.Speedup, &p.Hardware, &p.IsImplementable, &p.GithubURL, &p.EdgeType, &p.IsTheoretical,
		)
		if err != nil {
			log.Printf("Row scan error: %v", err)
			continue
		}
		p.EdgeType = getBadge(p.EdgeType, p.EfficiencyScore)
		papers = append(papers, p)
	}

	// Send the JSON response to the frontend
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(papers)
}

func interrogatePaper(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PaperID  string `json:"paper_id"`
		Question string `json:"question"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// 1. Fetch the abstract from the database
	var abstract string
	err := database.Pool.QueryRow(context.Background(),
		"SELECT abstract FROM papers WHERE arxiv_id = $1", req.PaperID).Scan(&abstract)
	if err != nil {
		http.Error(w, "Paper not found", http.StatusNotFound)
		return
	}

	// 2. Build the System Prompt for Groq
	systemPrompt := `You are a cynical, highly-skilled Senior AI Backend Engineer. 
	You are answering a junior developer's question about a research paper. 
	Be direct, technical, and concise (max 3 sentences). 
	If the abstract doesn't contain the answer, bluntly say the paper lacks empirical data for that.`

	userPrompt := fmt.Sprintf("Paper Context: %s\n\nQuestion: %s", abstract, req.Question)

	// 3. Fire the request to Groq
	groqReq := map[string]interface{}{
		"model": "llama-3.1-8b-instant",
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"temperature": 0.2,
	}

	jsonData, _ := json.Marshal(groqReq)
	reqHttp, _ := http.NewRequest("POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(jsonData))
	reqHttp.Header.Set("Authorization", "Bearer "+os.Getenv("GROQ_API_KEY"))
	reqHttp.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(reqHttp)
	if err != nil {
		http.Error(w, `{"error": "Groq API Network Failure"}`, http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		log.Printf("Groq Error: %s", string(bodyBytes))
		http.Error(w, `{"error": "Groq rejected the request"}`, http.StatusInternalServerError)
		return
	}

	var groqResp models.GroqResponse
	json.Unmarshal(bodyBytes, &groqResp)

	// 4. Return the answer to the Next.js UI
	if len(groqResp.Choices) > 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"answer": groqResp.Choices[0].Message.Content})
	} else {
		http.Error(w, "No answer generated", http.StatusInternalServerError)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	reqHttp, _ = http.NewRequestWithContext(ctx, "POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(jsonData))
}

func main() {
	// 1. Connect to the database
	database.ConnectDB()
	defer func() {
		if database.Pool != nil {
			database.Pool.Close()
		}
	}()

	// 2. Initialize the Chi Router
	r := chi.NewRouter()

	// 3. Add Middleware (Logging, Crash Recovery, and CORS)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		// Explicitly whitelist your Vercel domain and localhost
		AllowedOrigins: []string{
			"https://resmon-74i40n2oe-smarak-choudhurys-projects.vercel.app",
			"http://localhost:3000",
		},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Content-Type"},
	}))

	// 4. Define your API Routes
	r.Get("/api/papers", getLatestPapers)
	r.Get("/api/search", searchPapers)
	r.Post("/api/interrogate", interrogatePaper)

	// 5. Start the Server
	log.Println("🚀 API Server running on http://localhost:8080")
	err := http.ListenAndServe(":8080", r)
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func searchPapers(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	// 1. Convert the user's text search into a mathematical vector
	vectorString, err := llm.GenerateEmbedding(query)
	if err != nil {
		http.Error(w, "Failed to embed query", http.StatusInternalServerError)
		return
	}

	fmt.Printf("\n[DEBUG] Search Query: '%s' | Vector Length: %d characters\n", query, len(vectorString))

	// 2. Perform a Semantic Vector Search in Postgres using pgvector's `<=>` operator
	// This orders the results by how mathematically similar they are to the search query
	rows, err := database.Pool.Query(context.Background(),
		`SELECT arxiv_id, title, abstract, efficiency_score, speedup, hardware, is_implementable, github_url, edge_type, is_theoretical 
		 FROM papers 
		 WHERE embedding <=> $1::vector < 0.55
		 ORDER BY embedding <=> $1::vector 
		 LIMIT 15`, vectorString)

	if err != nil {
		http.Error(w, "Database search failed", http.StatusInternalServerError)
		log.Printf("Vector Search Error: %v", err)
		return
	}
	defer rows.Close()

	var papers []models.Paper
	for rows.Next() {
		var p models.Paper
		err := rows.Scan(
			&p.ArxivID, &p.Title, &p.Abstract,
			&p.EfficiencyScore, &p.Speedup, &p.Hardware, &p.IsImplementable, &p.GithubURL, &p.EdgeType, &p.IsTheoretical,
		)
		if err == nil {
			p.EdgeType = getBadge(p.EdgeType, p.EfficiencyScore)
			papers = append(papers, p)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(papers)
}
