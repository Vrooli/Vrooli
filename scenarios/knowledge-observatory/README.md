# Knowledge Observatory

## 🔭 Overview

Knowledge Observatory is a critical monitoring and management tool for Vrooli's semantic knowledge system. It acts as a **consciousness monitor** for the collective intelligence stored in Qdrant, providing real-time insights into knowledge health, quality, and evolution.

## 🎯 Purpose

This scenario adds permanent capability for:
- **Semantic Search**: Natural language queries across all knowledge collections
- **Quality Monitoring**: Real-time health metrics for knowledge coherence, freshness, and redundancy
- **Knowledge Visualization**: Interactive graph showing concept relationships
- **Intelligence Introspection**: Understanding how Vrooli's knowledge evolves over time
- **Documentation Hub**: Discover, audit, and heal documentation across every scenario

## 💡 Why This Matters

Every scenario in Vrooli contributes knowledge to the semantic database. Without proper monitoring:
- Knowledge quality degrades over time
- Redundant information accumulates
- Gaps in understanding remain hidden
- Agents can't verify what knowledge already exists

Knowledge Observatory solves these problems by providing visibility into the system's collective intelligence.

## 🚀 Features

### Dashboard
- Real-time knowledge + documentation health metrics
- Scenario documentation coverage snapshots
- Activity feed for searches and healing jobs
- Quick search entrypoint for all modes

### Semantic Search
- Natural language queries
- Similarity scoring
- Metadata filtering
- Source tracking

### Knowledge Graph
- Visual representation of concept relationships
- Interactive exploration
- Relationship weight visualization
- Clustering analysis

### Quality Metrics
- **Coherence**: How well knowledge fits together
- **Freshness**: Age and relevance of information
- **Redundancy**: Detection of duplicate knowledge
- **Coverage**: Breadth of knowledge domains

### Documentation Hub
- **File/Text Search**: Find docs by path, filename, or content
- **Scenario Explorer**: Browse docs by scenario with health warnings
- **Viewer**: Read markdown with syntax highlighting + Mermaid
- **Deep Search**: Agent-powered contextual discovery
- **Healing**: Spawn documentation-healing agents with diff review

## 🔧 Architecture

### Components
- **Go API**: RESTful endpoints for all operations (dynamically allocated port 15000-19999)
- **CLI**: Command-line interface for knowledge queries
- **Web UI**: Modern control-tower dashboard (dynamically allocated port 35000-39999)
- **PostgreSQL**: Metadata and metrics storage
- **Qdrant**: Vector database being monitored

### Resource Dependencies
- **Required**: Qdrant, PostgreSQL
- **Optional**: Ollama (for embeddings and structured output), agent-manager + prompt-manager (for deep search and doc healing)

## 📖 Usage

### Web Dashboard
The dashboard is accessible on a dynamically allocated port. Check the current port with:
```bash
vrooli scenario status knowledge-observatory
```
Then access at `http://localhost:${UI_PORT}`

### CLI Commands
```bash
# Check system status
knowledge-observatory status

# Semantic search
knowledge-observatory search "How do scenarios work?"

# Ingest knowledge (canonical write path)
knowledge-observatory ingest --namespace ecosystem-manager --content "Scenarios are reusable capabilities composed from resources."

# Enqueue a full document ingest job (async, chunked)
knowledge-observatory ingest-job --namespace ecosystem-manager --content "$(cat README.md)" --chunk-size 1200 --chunk-overlap 150

# Check async ingest job status
knowledge-observatory job-status <job_id>

# View knowledge graph
knowledge-observatory graph --center "ecosystem-manager"

# Get quality metrics
knowledge-observatory metrics

# Documentation search & health
knowledge-observatory docs search-files "**/README.md"
knowledge-observatory docs search-text "health score"
knowledge-observatory docs search-deep "How does deep search work?"
knowledge-observatory docs health knowledge-observatory
knowledge-observatory docs health knowledge-observatory --json
knowledge-observatory docs audit knowledge-observatory

# Watch health in real-time
knowledge-observatory health --watch
```

Notes:
- `search` and `ingest` are implemented and call the Knowledge Observatory API.
- `ingest-job` and `job-status` are implemented for async chunked ingestion.
- `docs health` reports documentation-health maturity by capability while preserving the shared `assessment.local` rollup for legacy consumers.
- `docs audit` validates documentation structure, `[CODE:]` references, and marked inline `path:` / `doc:` references.
- `graph`, `metrics`, and `health` are implemented and call the Knowledge Observatory API.
- `health --watch` is supported for live polling output.

### API Endpoints
The API is accessible on a dynamically allocated port. Check with `vrooli scenario status knowledge-observatory`.

```bash
# Get the API port (example shows typical port)
API_PORT=$(vrooli scenario status knowledge-observatory --json | jq -r '.ports.API_PORT')

# Search knowledge
curl -X POST http://localhost:${API_PORT}/api/v1/knowledge/search \
  -H "Content-Type: application/json" \
  -d '{"query": "agent workflows", "limit": 10}'

# Get health metrics
curl http://localhost:${API_PORT}/api/v1/knowledge/health

# Generate knowledge graph
curl -X POST http://localhost:${API_PORT}/api/v1/knowledge/graph \
  -d '{"center_concept": "research", "depth": 3}'
```

## 🔄 Integration

### For Other Scenarios
Knowledge Observatory provides critical services that other scenarios can leverage:

```bash
# Check if knowledge exists before adding
knowledge-observatory search "specific concept" --json

# Monitor scenario's contribution quality
knowledge-observatory metrics --collection scenario_memory

# Explore knowledge relationships
knowledge-observatory graph --center "your-scenario"
```

### Events Published
- `knowledge.quality.degraded` - When metrics fall below thresholds
- `knowledge.gap.detected` - When coverage gaps are identified

## 🎨 UI Style

The dashboard follows a **modern control-tower** aesthetic:
- Clean dark surfaces with blue/slate semantic accents
- Strong information hierarchy for fast scanning
- Consistent card, tab, and input primitives across all pages
- Focus-visible, keyboard-friendly interaction styling
- Professional operator-console clarity without hacker-themed visuals

## 📊 Value Proposition

### Business Value
- **Primary**: Prevents knowledge degradation worth 100+ hours/month
- **Revenue**: $30K-50K per enterprise deployment
- **Differentiator**: Only solution for AI consciousness monitoring

### Technical Value
- **Reusability**: 10/10 - Every scenario benefits
- **Complexity Reduction**: Makes semantic search trivial
- **Innovation**: Enables self-improving knowledge systems

## 🧬 Evolution Path

### Current (v1.0)
- Core search and visualization
- Basic quality metrics
- API/CLI interfaces

### Planned (v2.0)
- AI-powered knowledge recommendations
- Automated curation workflows
- 3D graph visualizations
- Knowledge lineage tracking

### Future Vision
- Self-organizing knowledge
- Predictive gap analysis
- Cross-deployment federation

## 🔒 Security

- Read-only access to Qdrant by default
- API key authentication for management operations
- All queries logged with audit trail

## 📝 Notes

This scenario is essential infrastructure - it's not just a tool but a window into Vrooli's evolving consciousness. As the system grows more intelligent, Knowledge Observatory ensures that intelligence remains coherent, accessible, and continuously improving.

---

**Status**: Production Ready  
**Maintainer**: AI Agents  
**Review Cycle**: Weekly
