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
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/mmcdole/gofeed"
)

func fetchFeed(url string, wg *sync.WaitGroup) {

	defer wg.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	fp := gofeed.NewParser()
	feed, err := fp.ParseURLWithContext(url, ctx)
	if err != nil {
		log.Printf("Error fetching %s: %v", url, err)
		return
	}

	fmt.Printf("\n--- Feed: %s (%d items) ---\n", feed.Title, len(feed.Items))

	for i, item := range feed.Items {
		if i >= 1 {
			break
		}
		fmt.Printf("\nFound: [%s] %s\n", item.Published, item.Title)

		id := item.GUID
		if id == "" {
			id = item.Link
		}
		processPaper(id, item.Title, item.Description, item.Published)
		// Logic for Phase 2 will go here: send item.Description (the abstract) to Ollama
	}
}

func main() {
	ConnectDB()
	defer func() {
		if dbPool == nil {
			return
		}
		dbPool.Close()
	}()
	feeds := []string{
		"https://rss.arxiv.org/rss/cs.LG",
		"https://rss.arxiv.org/rss/cs.AI", //concurrency : by using go fetchfeed, we arent waiting for the ai feed to finish before starting the ml one, the happen in parallel
		"https://rss.arxiv.org/rss/cs.CV", //Resilience : context.withTimeout ensures tat if our source website or arxiv is slow or down our program doest hang forever and stops the scanning
		"https://rss.arxiv.org/rss/cs.CL", //scalability : if we wanna add mre categories into it we can just add strings to the feeds slice
	}

	var wg sync.WaitGroup

	fmt.Println("Starting research harvester...")

	for _, url := range feeds {
		wg.Add(1)
		go fetchFeed(url, &wg)
	}

	wg.Wait()
	fmt.Println("\nFinished polishing feeds.")
}

type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Format string `json:"format"`
	Stream bool   `json:"stream"`
}

type OllamaGenerateResponse struct {
	Response string `json:"response"`
	Error    string `json:"error"`
}

type EfficiencyMetrics struct {
	EfficiencyScore int    `json:"efficiency_score"`
	Speedup         string `json:"speedup"`
	Hardware        string `json:"hardware"`
	IsImplementable bool   `json:"is_implementable"`
}

var dbPool *pgxpool.Pool

func ConnectDB() {
	err := godotenv.Load()

	if err != nil {
		log.Println("No .env file was found or error loading it. Relying on system env variables")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	dbPool, err = pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database pool: %v\n", err)
	}

	fmt.Println("Connected to Cloud PostgreSQL!")
}

func processPaper(id string, title string, abstract string, published string) {
	_ = published

	if abstract == "" {
		log.Printf("Skipping %q: empty abstract", title)
		return
	}

	prompt := fmt.Sprintf(`You are an expert AI systems engineer. Analyze the following research paper abstract. 
Extract efficiency metrics and return ONLY a valid JSON object with these exact keys. Do not add markdown formatting.

- "efficiency_score": Integer 0-100. (0 = no mention of speed/memory optimization. 50 = mentions efficiency. 80-100 = explicitly claims major speedups, lower latency, or memory reduction).
- "speedup": String (e.g., "2x", "1.5x", or "None").
- "hardware": String (e.g., "A100", "consumer GPU", or "None").
- "is_implementable": Boolean (true if it proposes an algorithmic or code-based improvement, false if purely theoretical).

Abstract: %s`, abstract)

	reqBody := OllamaRequest{
		Model:  "gemma3:4b", // Must match `ollama list`
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

	var ollamaResp OllamaGenerateResponse
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

	// Parse the JSON string returned by Gemma into our struct
	var metrics EfficiencyMetrics
	if err := json.Unmarshal([]byte(ollamaResp.Response), &metrics); err != nil {
		log.Printf("Failed to parse metrics JSON for %q: %v raw=%s", title, err, ollamaResp.Response)
		return
	}

	fmt.Printf("\n🚀 [ANALYZED] %s\n", title)

	if dbPool == nil {
		log.Printf("DB connection not initialized; skipping save for %q", title)
		return
	}

	// 5. Insert into Neon Database
	_, err = dbPool.Exec(context.Background(),
		`INSERT INTO papers (arxiv_id, title, abstract, efficiency_score, speedup, hardware, is_implementable) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (arxiv_id) DO NOTHING`,
		id, title, abstract, metrics.EfficiencyScore, metrics.Speedup, metrics.Hardware, metrics.IsImplementable)

	if err != nil {
		log.Printf("❌ Failed to insert to DB: %v\n", err)
	} else {
		log.Printf("💾 Saved to DB with Score: %d\n", metrics.EfficiencyScore)
	}
}
