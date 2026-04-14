package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"resmon/internal/database"
	"resmon/pkg/models"
)

// 1. COHERE EMBEDDINGS (Replaces local Nomic)
func GenerateEmbedding(text string) (string, error) {
	reqBody := models.CohereRequest{
		Texts:     []string{text},
		Model:     "embed-english-v3.0",
		InputType: "search_document",
	}
	jsonData, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "https://api.cohere.ai/v1/embed", bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+os.Getenv("COHERE_API_KEY"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var cohereResp models.CohereResponse
	if err := json.NewDecoder(resp.Body).Decode(&cohereResp); err != nil {
		return "", err
	}

	if len(cohereResp.Embeddings) == 0 {
		return "[]", fmt.Errorf("no embeddings returned")
	}

	vecBytes, _ := json.Marshal(cohereResp.Embeddings[0])
	return string(vecBytes), nil
}

// 2. GROQ ANALYSIS (Replaces local Gemma)
// Notice we added 'category string' here to feed the frontend UI!
func ProcessPaper(id string, title string, abstract string, published string, category string) {
	if abstract == "" {
		log.Printf("Skipping %q: empty abstract", title)
		return
	}

	prompt := fmt.Sprintf(`You are an expert AI systems engineer. Analyze the following research paper abstract. 
Extract efficiency metrics and return ONLY a valid JSON object. Do not use markdown blocks.

{
  "efficiency_score": 0-100,
  "speedup": "e.g. 2x or N/A",
  "hardware": "e.g. A100 or Edge",
  "edge_type": "Speed" or "Efficiency" or "General",
  "is_theoretical": true or false
}

Abstract: %s`, abstract)

	// GROQ REQUEST PAYLOAD
	groqReq := models.GroqRequest{
		Model: "llama-3.1-8b-instant",
		Messages: []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{Role: "system", Content: "You output strict JSON only."},
			{Role: "user", Content: prompt},
		},
		ResponseFormat: struct {
			Type string `json:"type"`
		}{Type: "json_object"},
		Temperature: 0.1,
	}

	jsonData, _ := json.Marshal(groqReq)

	req, _ := http.NewRequest("POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+os.Getenv("GROQ_API_KEY"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Groq error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var groqResp models.GroqResponse
	if err := json.NewDecoder(resp.Body).Decode(&groqResp); err != nil || len(groqResp.Choices) == 0 {
		log.Printf("Failed to parse Groq response for %q", title)
		return
	}

	var metrics models.EfficiencyMetrics
	rawJSON := groqResp.Choices[0].Message.Content
	if err := json.Unmarshal([]byte(rawJSON), &metrics); err != nil {
		log.Printf("Failed to parse metrics JSON for %q: %v raw=%s", title, err, rawJSON)
		return
	}

	fmt.Printf("\n🚀 [ANALYZED by GROQ] %s\n", title)

	githubURL := searchGithubForPaper(title)
	if githubURL != "" {
		fmt.Printf("Found Code: %s\n", githubURL)
	}

	fmt.Printf("Generating Semantic Vector via Cohere..\n")
	vectorString, err := GenerateEmbedding(abstract)
	if err != nil {
		log.Printf("Failed to generate embedding: %v\n", err)
		vectorString = "[]" // Fallback so DB doesn't crash
	}

	if database.Pool == nil {
		return
	}

	// 3. THE FIXED DATABASE INSERT (Includes Category & Edge Types)
	_, err = database.Pool.Exec(context.Background(),
		`INSERT INTO papers (arxiv_id, title, abstract, efficiency_score, speedup, hardware, is_implementable, github_url, embedding, edge_type, is_theoretical, category) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::vector, $10, $11, $12)
		 ON CONFLICT (arxiv_id) DO UPDATE SET
			 title = EXCLUDED.title,
			 abstract = EXCLUDED.abstract,
			 efficiency_score = EXCLUDED.efficiency_score,
			 speedup = EXCLUDED.speedup,
			 hardware = EXCLUDED.hardware,
			 is_implementable = EXCLUDED.is_implementable,
			 github_url = EXCLUDED.github_url,
			 embedding = EXCLUDED.embedding,
			 edge_type = EXCLUDED.edge_type,
			 is_theoretical = EXCLUDED.is_theoretical,
			 category = EXCLUDED.category`,
		id, title, abstract, metrics.EfficiencyScore, metrics.Speedup, metrics.Hardware, metrics.IsImplementable, githubURL, vectorString, metrics.EdgeType, metrics.IsTheoretical, category)

	if err != nil {
		log.Printf("❌ Failed to insert to DB: %v\n", err)
	} else {
		log.Printf("💾 Saved to DB with Score: %d | Edge: %s\n", metrics.EfficiencyScore, metrics.EdgeType)
	}
}

func searchGithubForPaper(title string) string {
	query := url.QueryEscape(fmt.Sprintf(`"%s"`, title))
	reqURL := fmt.Sprintf("https://api.github.com/search/repositories?q=%s", query)

	resp, err := http.Get(reqURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		return ""
	}
	defer resp.Body.Close()

	var result struct {
		Items []struct {
			HtmlUrl string `json:"html_url"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
		if len(result.Items) > 0 {
			return result.Items[0].HtmlUrl
		}
	}
	return ""
}
