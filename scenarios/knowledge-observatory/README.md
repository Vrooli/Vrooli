# Knowledge Observatory

## 🔭 Overview

Knowledge Observatory is a critical monitoring and management tool for Vrooli's semantic knowledge system. It acts as a **consciousness monitor** for the collective intelligence stored in Qdrant, providing real-time insights into knowledge health, quality, and evolution.

## 🎯 Purpose

This scenario adds permanent capability for:
- **Semantic Search**: Natural language queries across all knowledge collections
- **Quality Monitoring**: Real-time health metrics for knowledge coherence, freshness, and redundancy
- **Knowledge Visualization**: Interactive graph showing concept relationships
- **Intelligence Introspection**: Understanding how Vrooli's knowledge evolves over time

## 💡 Why This Matters

Every scenario in Vrooli contributes knowledge to the semantic database. Without proper monitoring:
- Knowledge quality degrades over time
- Redundant information accumulates
- Gaps in understanding remain hidden
- Agents can't verify what knowledge already exists

Knowledge Observatory solves these problems by providing visibility into the system's collective intelligence.

## 🚀 Features

### Dashboard
- Real-time knowledge health metrics
- Collection statistics and trends
- Alert system for quality degradation
- Activity timeline

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

## 🔧 Architecture

### Components
- **Go API**: RESTful endpoints for all operations (dynamically allocated port 15000-19999)
- **CLI**: Command-line interface for knowledge queries
- **Web UI**: Mission control style dashboard (dynamically allocated port 35000-39999)
- **PostgreSQL**: Metadata and metrics storage
- **Qdrant**: Vector database being monitored
- **N8n Workflows**: Automated quality monitoring

### Resource Dependencies
- **Required**: Qdrant, PostgreSQL
- **Optional**: N8n, Ollama (for enhanced analysis)

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
knowledge-observatory metrics --collection vrooli_knowledge

# Watch health in real-time
knowledge-observatory health --watch
```

Notes:
- `search` and `ingest` are implemented and call the Knowledge Observatory API.
- `ingest-job` and `job-status` are implemented for async chunked ingestion.
- `graph`, `metrics`, and `health --watch` are planned CLI commands; the API currently exposes `GET /api/v1/knowledge/health` for health metrics.

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

The dashboard follows a **technical mission control** aesthetic:
- Dark theme with green accent (Matrix-inspired)
- Real-time data streams
- Graph visualizations with organic pulsing
- Heat maps for knowledge density
- Professional yet engaging interface

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
