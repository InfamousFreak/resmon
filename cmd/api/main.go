package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"resmon/internal/database"
	"resmon/internal/llm"
	"resmon/pkg/models"
)

// The handler to fetch the latest high-efficiency papers
func getLatestPapers(w http.ResponseWriter, r *http.Request) {
	// Query the database for the top 20 most recently parsed papers, ordered by efficiency
	rows, err := database.Pool.Query(context.Background(),
		`SELECT arxiv_id, title, abstract, efficiency_score, speedup, hardware, is_implementable, github_url
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
			&p.EfficiencyScore, &p.Speedup, &p.Hardware, &p.IsImplementable, &p.GithubURL,
		)
		if err != nil {
			log.Printf("Row scan error: %v", err)
			continue
		}
		papers = append(papers, p)
	}

	// Send the JSON response to the frontend
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(papers)
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
		AllowedOrigins:   []string{"http://localhost:3000"}, // Allows your Next.js frontend
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: true,
	}))

	// 4. Define your API Routes
	r.Get("/api/papers", getLatestPapers)
	r.Get("/api/search", searchPapers)

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
		`SELECT arxiv_id, title, abstract, efficiency_score, speedup, hardware, is_implementable, github_url 
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
			&p.EfficiencyScore, &p.Speedup, &p.Hardware, &p.IsImplementable, &p.GithubURL,
		)
		if err == nil {
			papers = append(papers, p)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(papers)
}
