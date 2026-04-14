package models

// OllamaRequest is the payload sent to the local LLM
type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Format string `json:"format"`
	Stream bool   `json:"stream"`
}

// OllamaGenerateResponse handles the envelope returned by Ollama
type OllamaGenerateResponse struct {
	Response string `json:"response"`
	Error    string `json:"error"`
}

// EfficiencyMetrics is the parsed intelligence from the paper
type EfficiencyMetrics struct {
	EfficiencyScore int    `json:"efficiency_score"`
	Speedup         string `json:"speedup"`
	Hardware        string `json:"hardware"`
	IsImplementable bool   `json:"is_implementable"`
	EdgeType        string `json:"edge_type"`
	IsTheoretical   bool   `json:"is_theoretical"`
}

// Paper represents a full row from the PostgreSQL database
type Paper struct {
	ArxivID         string  `json:"arxiv_id"`
	Title           string  `json:"title"`
	Abstract        string  `json:"abstract"`
	EfficiencyScore int     `json:"efficiency_score"`
	Speedup         string  `json:"speedup"`
	Hardware        string  `json:"hardware"`
	IsImplementable bool    `json:"is_implementable"`
	GithubURL       *string `json:"github_url"`
	EdgeType        string  `json:"edge_type"`
	IsTheoretical   bool    `json:"is_theoretical"`
	Category        string  `json:"category"`
}

// OllamaEmbeddingRequest is the payload for the embedding API
type OllamaEmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// OllamaEmbeddingResponse catches the array of floats
type OllamaEmbeddingResponse struct {
	Embedding []float32 `json:"embedding"`
}

// --- GROQ LLM STRUCTS ---
type GroqRequest struct {
	Model          string `json:"model"`
	Messages       []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
	ResponseFormat struct {
		Type string `json:"type"`
	} `json:"response_format"`
	Temperature float32 `json:"temperature"`
}

type GroqResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// --- COHERE EMBEDDING STRUCTS ---
type CohereRequest struct {
	Texts     []string `json:"texts"`
	Model     string   `json:"model"`
	InputType string   `json:"input_type"`
}

type CohereResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}
