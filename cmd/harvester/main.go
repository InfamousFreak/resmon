package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"

	"resmon/internal/database"
	"resmon/internal/llm"
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
		if i >= 6 {
			break
		}
		fmt.Printf("\nFound: [%s] %s\n", item.Published, item.Title)

		id := item.GUID
		if id == "" {
			id = item.Link
		}

		// Call the exported LLM processor
		llm.ProcessPaper(id, item.Title, item.Description, item.Published)
	}
}

func main() {
	database.ConnectDB()
	defer func() {
		if database.Pool == nil {
			return
		}
		database.Pool.Close()
	}()

	feeds := []string{
		"https://rss.arxiv.org/rss/cs.LG",
		"https://rss.arxiv.org/rss/cs.AI",
		"https://rss.arxiv.org/rss/cs.CV",
		"https://rss.arxiv.org/rss/cs.CL",
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
