package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"

	"resmon/internal/database"
	"resmon/pkg/models"
)

func GenerateEmbedding(text string) (string, error) {
	reqBody := models.OllamaEmbeddingRequest{
		Model:  "nomic-embed-text",
		Prompt: text,
	}

	jsonData, _ := json.Marshal(reqBody)
	resp, err := http.Post("http://localhost:11434/api/embeddings", "application/json", bytes.NewBuffer(jsonData))

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	var result models.OllamaEmbeddingResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	vecBytes, _ := json.Marshal(result.Embedding)
	return string(vecBytes), nil
}

var ollamaMu sync.Mutex

// Capital P so it can be called from main.go
func ProcessPaper(id string, title string, abstract string, published string) {

	ollamaMu.Lock()

	defer ollamaMu.Unlock()
	if abstract == "" {
		log.Printf("Skipping %q: empty abstract", title)
		return
	}

	prompt := fmt.Sprintf(`You are an expert AI systems engineer. Analyze the following research paper abstract. 
Extract efficiency metrics and return ONLY a valid JSON object with these exact keys. Do not add markdown formatting.

- "efficiency_score": Integer 0-100. (0 = no mention of speed/memory optimization. 50 = mentions efficiency. 80-100 = explicitly claims major speedups, lower latency, or memory reduction).
- "speedup": e.g., "2.5x", "40%%", or "N/A".
- "hardware": e.g., "A100", "Edge Device", "8GB VRAM", or "N/A".
- "edge_type": "Speed" (if it's about latency/FPS), "Efficiency" (if it's about RAM/VRAM/Power), or "General".
- "is_theoretical": boolean (true if no empirical benchmarks are mentioned).


Abstract: %s`, abstract)

	// Using the struct from our models package
	reqBody := models.OllamaRequest{
		Model:  "gemma3:4b",
		Prompt: prompt,
		Format: "json",
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Printf("Failed to encode Ollama request for %q: %v", title, err)
		return
	}

	resp, err := http.Post("http://localhost:11434/api/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Ollama error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed reading Ollama response for %q: %v", title, err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Ollama returned non-200 for %q: status=%d body=%s", title, resp.StatusCode, string(body))
		return
	}

	var ollamaResp models.OllamaGenerateResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		log.Printf("Failed to parse Ollama envelope for %q: %v body=%s", title, err, string(body))
		return
	}

	if ollamaResp.Error != "" {
		log.Printf("Ollama reported error for %q: %s", title, ollamaResp.Error)
		return
	}

	if ollamaResp.Response == "" {
		log.Printf("Ollama response missing 'response' field for %q: body=%s", title, string(body))
		return
	}

	var metrics models.EfficiencyMetrics
	if err := json.Unmarshal([]byte(ollamaResp.Response), &metrics); err != nil {
		log.Printf("Failed to parse metrics JSON for %q: %v raw=%s", title, err, ollamaResp.Response)
		return
	}

	fmt.Printf("\n🚀 [ANALYZED] %s\n", title)

	githubURL := ""
	if metrics.IsImplementable {
		fmt.Printf("🔍 Hunting for Github Repository ...\n")
		githubURL = searchGithubForPaper(title)

		if githubURL != "" {
			fmt.Printf("Found Code: %s\n", githubURL)
		} else {
			fmt.Printf("No Code published yet.\n")
		}
	}

	fmt.Printf("Generating Semantic Vector..\n")

	vectorString, err := GenerateEmbedding(abstract)
	if err != nil {

		log.Printf("Failed to generate embedding: %v\n", err)

		vectorString = "[]"
	}

	if database.Pool == nil {
		log.Printf("DB connection not initialized; skipping save for %q", title)
		return
	}

	// Persist github_url so the frontend can render a code link when available.
	_, err = database.Pool.Exec(context.Background(),
		`INSERT INTO papers (arxiv_id, title, abstract, efficiency_score, speedup, hardware, is_implementable, github_url, embedding) 
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
				 ON CONFLICT (arxiv_id) DO UPDATE SET
					 title = EXCLUDED.title,
					 abstract = EXCLUDED.abstract,
					 efficiency_score = EXCLUDED.efficiency_score,
					 speedup = EXCLUDED.speedup,
					 hardware = EXCLUDED.hardware,
					 is_implementable = EXCLUDED.is_implementable,
					 github_url = EXCLUDED.github_url,
					 embedding = EXCLUDED.embedding`,
		id, title, abstract, metrics.EfficiencyScore, metrics.Speedup, metrics.Hardware, metrics.IsImplementable, githubURL, vectorString)

	if err != nil {
		log.Printf("❌ Failed to insert to DB: %v\n", err)
	} else {
		log.Printf("💾 Saved to DB with Score: %d\n", metrics.EfficiencyScore)
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
