# 🔬 RESEARCH_MONITOR_v1.0

<div align="center">

![Status](https://img.shields.io/badge/STATUS-LIVE-00ff41?style=for-the-badge&labelColor=000000)
![Model](https://img.shields.io/badge/MODEL-GEMMA_4B-blue?style=for-the-badge&labelColor=000000)
![Backend](https://img.shields.io/badge/BACKEND-GO-00ADD8?style=for-the-badge&labelColor=000000&logo=go)
![Search](https://img.shields.io/badge/SEARCH-VECTOR_DB-purple?style=for-the-badge&labelColor=000000)
![License](https://img.shields.io/badge/LICENSE-MIT-green?style=for-the-badge&labelColor=000000)

**A live ArXiv intelligence engine that scores, ranks, and semantically searches research papers using on-device LLM inference — built with love and curiosity.**

[Report Bug](https://github.com/InfamousFreak/research-monitor/issues) · [Request Feature](https://github.com/InfamousFreak/research-monitor/issues)

</div>

---

```
> RESEARCH_MONITOR_v1.0
STATUS: LIVE DATA FEED | MODEL: GEMMA:4B
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
INGESTING  cs.AI · cs.CV · cs.LG · cs.CL
SCORING    relevance · implementability · hardware
EMBEDDING  nomic-embed-text · cosine similarity
SERVING    semantic search · distance threshold
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

---

## 📸 Dashboard

![Research Monitor Dashboard]()

> Terminal-aesthetic live feed — papers scored by relevance, flagged by implementability, tagged with hardware requirements.

---

## 🧠 What Is This

Most research aggregators are glorified RSS feeds. This is different.

**Research Monitor** ingests ArXiv papers in real-time across configurable category targets (`cs.AI`, `cs.CV`, `cs.LG` etc.), runs each abstract through a local **Gemma 4B** inference pipeline to generate relevance scores, extracts hardware and speedup metadata, and stores vector embeddings for semantic search — all powered by a concurrent Go backend.

The result: instead of scrolling through 50 papers manually, you get a ranked, scored, searchable intelligence feed with one question: *is this paper worth your time?*

---

## ⚡ Core Features

| Feature | Description |
|---|---|
| 🔴 **Live Data Feed** | Concurrent ArXiv ingestion across multiple category targets |
| 🤖 **LLM Scoring** | Gemma 4B scores each paper for relevance (0–100) |
| ✅ **Implementability Flag** | Model determines if paper is reproducible without massive compute |
| 🔧 **Hardware Extraction** | Detects hardware requirements and speedup claims from abstracts |
| 🔍 **Semantic Search** | Vector similarity search with distance threshold filtering |
| 📊 **Ranked Feed** | Papers sorted by score, not just recency |
| 💬 **Interrogate Mode** | Direct Gemma conversation per paper *(in progress)* |

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    RESEARCH MONITOR v1.0                    │
└─────────────────────────────────────────────────────────────┘
                              │
          ┌───────────────────▼──────────────────┐
          │           ArXiv OAI-PMH API           │
          │   cs.AI · cs.CV · cs.LG · cs.CL      │
          └───────────────────┬──────────────────┘
                              │
          ┌───────────────────▼──────────────────┐
          │         Go Concurrent Fetcher         │
          │    goroutines · buffered channels     │
          │         worker pool pattern           │
          └──────────┬──────────────┬────────────┘
                     │              │
        ┌────────────▼──┐    ┌──────▼────────────┐
        │  Gemma 4B     │    │  Nomic Embeddings  │
        │  (Ollama)     │    │  (nomic-embed-text)│
        │               │    │                    │
        │ · relevance   │    │ · 768-dim vectors  │
        │ · implementa- │    │ · cosine distance  │
        │   bility      │    │ · threshold filter │
        │ · hardware    │    │                    │
        │ · speedup     │    │                    │
        └────────┬──────┘    └──────┬─────────────┘
                 │                  │
          ┌──────▼──────────────────▼──────┐
          │         Vector Store           │
          │   scored papers + embeddings   │
          └──────────────┬─────────────────┘
                         │
          ┌──────────────▼─────────────────┐
          │        Terminal UI             │
          │   ranked cards · search bar    │
          │   score badges · flags         │
          └────────────────────────────────┘
```

---

## 🎯 Scoring System

Each paper receives a **composite score (0–100)** computed by Gemma 4B across four dimensions:

```
SCORE = f(relevance, novelty, implementability, hardware_accessibility)

IMPLEMENTABLE: TRUE   → paper can be reproduced on consumer hardware
IMPLEMENTABLE: FALSE  → requires H100/A100 scale compute
SPEEDUP: Nx           → extracted from abstract if claimed
HARDWARE: type        → GPU/TPU/CPU requirements detected
```

Papers with `SCORE: 0` are either outside the relevance threshold or flagged as not implementable without significant compute infrastructure.

---

## 🔍 Semantic Search

The search system uses **mathematical vector distances** rather than keyword matching:

```
Query → Nomic Embedding → 768-dim vector
                              ↓
              Cosine similarity against all stored papers
                              ↓
              Distance threshold filter (tuned empirically)
                              ↓
              Ranked results by semantic closeness
```

**Why distance threshold matters:** Without it, every query returns results regardless of actual semantic relevance. The threshold ensures "no result" is a valid and honest answer when nothing in the corpus genuinely matches your query.

---

## 🚀 Getting Started

### Prerequisites

```bash
# Go 1.21+
go version

# Ollama for local model serving
curl -fsSL https://ollama.ai/install.sh | sh

# Pull required models
ollama pull gemma:4b
ollama pull nomic-embed-text
```

### Installation

```bash
git clone https://github.com/InfamousFreak/research-monitor
cd research-monitor
go mod download
```

### Configuration

```bash
# config.yaml
targets:
  - cs.AI
  - cs.CV
  - cs.LG
  - cs.CL
  - stat.ML

fetch_interval: 3600        # seconds between harvests
max_results_per_category: 50
score_threshold: 40         # minimum score to display
distance_threshold: 0.35    # semantic search cutoff
worker_pool_size: 3         # concurrent inference workers
```

### Run

```bash
go run main.go
```

---

## 📁 Project Structure

```
research-monitor/
├── main.go                 # entry point
├── cmd/
│   ├── api           # ArXiv OAI-PMH client
│   └── harvester
│   └── processor         # Atom/XML feed parser
├── internal/
│   ├── database           # Gemma 4B scoring pipeline
│   └── llm        
├── pkg/          # vector store + similarity search
│   └── models              # paper persistence
├── frontend/
    ├── public
│   └── app        # terminal dashboard renderer         # configuration management
└── docker-compose.yaml            # targets + thresholds
```

---

## 🔬 Research Context

This project is directly informed by the emerging field of **adaptive test-time scaling** in AI research — the idea that inference compute should be allocated dynamically based on problem difficulty rather than uniformly.

Papers like *"From Scale to Speed: Adaptive Test-Time Scaling for Image Editing"* (Alibaba, Feb 2025) demonstrate 2x speedups through intelligent resource allocation at inference time. Research Monitor applies this philosophy at the **meta-level** — allocating your reading time adaptively based on paper relevance scores rather than processing every paper uniformly.

The scoring system is essentially a lightweight classifier that answers: *"Is this paper worth deeper compute (your time)?"*

---


## 🛠️ Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.21 · goroutines · channels · worker pools |
| LLM Inference | Gemma 4B via Ollama |
| Embeddings | Nomic-embed-text (768-dim) |
| Vector Search | Cosine similarity · distance threshold |
| Data Source | ArXiv OAI-PMH API (Atom/XML) |
| UI | Terminal renderer |
| Planned: Inference | Groq API |
| Planned: Deploy | Railway |

---

## 📊 Performance Notes

Running Gemma 4B + Nomic embeddings concurrently on CPU will saturate memory on most laptops. Current architecture uses a **worker pool pattern** to bound concurrent inference calls:

```go
// Bounded concurrency — tune pool size to your hardware
sem := make(chan struct{}, config.WorkerPoolSize)

for _, paper := range papers {
    sem <- struct{}{}
    go func(p Paper) {
        defer func() { <-sem }()
        result := inferenceWorker(p)
        results <- result
    }(paper)
}
```

For deployment, the Ollama calls will be swapped for **Groq API** which handles inference server-side, removing the local compute bottleneck entirely.

---

## 👤 Author

**Smarak Choudhury**

- GitHub: [@InfamousFreak](https://github.com/InfamousFreak)
- LinkedIn: [smarak-choudhury](https://www.linkedin.com/in/smarak-choudhury-423b39280/)
- Portfolio: https://slick-folio-magic.vercel.app/


---

<div align="center">

**Built by an engineer, for engineers.**

⭐ Star this repo if you find it useful

</div>
