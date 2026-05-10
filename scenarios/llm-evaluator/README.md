# LLM Evaluator

> Comprehensive framework for evaluating LLM outputs with quality metrics, benchmarking, and feedback loops.

## 🎯 Purpose

LLM Evaluator provides systematic measurement and continuous improvement of all LLM-powered scenarios within the Vrooli ecosystem. It fills the most critical gap in Vrooli's AI infrastructure by enabling unified quality measurement across all LLM interactions.

**Key Capabilities:**
- Calculate BLEU, ROUGE, and semantic similarity (BERTScore) metrics
- LLM-as-judge evaluation with customizable rubrics
- RAG quality assessment (context relevancy, faithfulness, answer relevancy)
- Model comparison and A/B testing framework
- Quality gates and CI/CD integration
- Real-time dashboard visualization

## 🚀 Quick Start

### Prerequisites
- PostgreSQL (for evaluation results storage)
- Ollama (recommended, for LLM-as-judge evaluations)
- Qdrant (optional, via knowledge-observatory for semantic metrics)

### Setup and Start
```bash
cd scenarios/llm-evaluator

# First time setup
make setup

# Start the service
make start

# OR via vrooli CLI
vrooli scenario start llm-evaluator
```

### Access
- **UI Dashboard**: http://localhost:${UI_PORT}
- **API**: http://localhost:${API_PORT}/api/v1
- **Health**: http://localhost:${API_PORT}/health

### CLI Usage
```bash
# Install CLI
cd cli && ./install.sh

# Check status
llm-evaluator status

# Submit evaluation job
llm-evaluator evaluate --input output.txt --reference expected.txt --metrics bleu,rouge

# Get job results
llm-evaluator results <job-id>
```

## 📋 Documentation

- [PRD.md](PRD.md) - Product Requirements Document with operational targets
- [docs/PROGRESS.md](docs/PROGRESS.md) - Development progress tracking
- [docs/PROBLEMS.md](docs/PROBLEMS.md) - Known issues and deferred ideas
- [docs/RESEARCH.md](docs/RESEARCH.md) - Research notes and external references
- [requirements/](requirements/) - Requirements registry mapping PRD targets to implementation

## 🧱 Architecture

```
llm-evaluator/
├── api/              # Go API server
│   ├── main.go       # Entry point
│   ├── handlers.go   # HTTP handlers
│   ├── jobs.go       # Evaluation job management
│   └── metrics/      # Metric calculators (BLEU, ROUGE, etc.)
├── ui/               # React + TypeScript dashboard
│   └── src/
│       ├── pages/    # Dashboard, Jobs, Results views
│       └── components/ # Charts, comparison tools
├── cli/              # Go CLI wrapper
├── .vrooli/          # Lifecycle and health configuration
├── requirements/     # Requirements registry
└── docs/             # Documentation
```

## 🧪 Testing

```bash
# Run all tests via test-genie
make test

# Run Go unit tests
cd api && go test -v ./...

# Run UI tests
cd ui && pnpm test
```

## 🔧 Configuration

### Environment Variables
| Variable | Purpose |
|----------|---------|
| `API_PORT` | Port for Go API server |
| `UI_PORT` | Port for React UI |
| `DATABASE_URL` | PostgreSQL connection string |
| `OLLAMA_URL` | Ollama endpoint for LLM-as-judge |
| `QDRANT_URL` | Qdrant endpoint for semantic similarity |

### Port Ranges
- **API**: 15000-19999
- **UI**: 35000-39999
- **WebSocket**: 25000-29999

## 🤝 Integration

LLM Evaluator integrates with other Vrooli scenarios:

- **agent-manager**: Evaluate agent outputs automatically
- **prompt-manager**: Feed evaluation data for prompt optimization
- **knowledge-observatory**: Access RAG context for faithfulness checks

## 📈 Recursive Value

This scenario enables the following improvements across Vrooli:
- Unified quality measurement for all LLM-powered scenarios
- Feedback loops that improve prompts, model selection, and output quality
- Cost-quality optimization to maximize AI ROI
- Quality gates that prevent poor outputs from reaching downstream systems

## 📚 External References

- [DeepEval - LLM Evaluation Framework](https://deepeval.com/)
- [RAGAs - RAG Evaluation](https://docs.ragas.io/en/stable/)
- [LLM Evaluation Metrics Guide](https://www.confident-ai.com/blog/llm-evaluation-metrics-everything-you-need-for-llm-evaluation)

## 📄 License

Part of the Vrooli ecosystem. See root LICENSE file.
