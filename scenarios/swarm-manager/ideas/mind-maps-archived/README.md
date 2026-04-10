# Mind Maps

## Purpose
AI-enabled mind mapping tool for visual organization and semantic search of ideas and knowledge. This scenario serves as a critical building block for other scenarios that need to organize, search, and retrieve structured information.

## Scenario Usefulness
Mind Maps is a **foundational capability** that other scenarios can leverage via its CLI and API to:
- Store and organize structured knowledge in campaigns/projects/topics
- Perform semantic search across mind map nodes and connections
- Auto-organize unstructured information into visual maps
- Export knowledge in various formats for downstream processing

### Cross-Scenario Integration
- **stream-of-consciousness-analyzer**: Uses Mind Maps to organize analyzed thoughts
- **research-assistant**: Stores research findings as interconnected mind maps
- **study-buddy**: Organizes study materials into visual knowledge graphs
- **product-manager-agent**: Maps product features and dependencies
- **idea-generator**: Visualizes and connects generated ideas

## Dependencies
### Required Resources
- **n8n**: Workflow automation for intelligent processing
- **PostgreSQL**: Persistent storage of mind maps and metadata
- **Redis**: Real-time collaboration and caching
- **Qdrant**: Vector database for semantic search

### Shared Workflows
- `ollama.json`: LLM inference for auto-organization
- `embedding-generator.json`: Creates vector embeddings for semantic search
- `chain-of-thought-orchestrator.json`: Complex reasoning for node relationships
- `structured-data-extractor.json`: Extracts structured data from text
- `universal-rag-pipeline.json`: Advanced retrieval for context-aware suggestions

## API Endpoints
- `GET /api/mindmaps` - List all mind maps
- `POST /api/mindmaps` - Create new mind map
- `GET /api/mindmaps/:id` - Get specific mind map
- `PUT /api/mindmaps/:id` - Update mind map
- `DELETE /api/mindmaps/:id` - Delete mind map
- `POST /api/search` - Semantic search across mind maps
- `POST /api/organize` - Auto-organize unstructured content

## CLI Commands
```bash
# Create a new mind map
mind-maps create --title "Project Architecture" --description "System design overview"

# Search mind maps semantically
mind-maps search "distributed systems concepts"

# Auto-organize text into mind map
mind-maps organize --text "..." --campaign "research"

# Export mind map
mind-maps export --id <id> --format json
```

## UI Features
- **Interactive Canvas**: Drag-and-drop node manipulation
- **Neural View**: Advanced visualization showing weighted connections
- **Real-time Collaboration**: Multiple users can edit simultaneously
- **Semantic Clustering**: Automatically groups related concepts
- **Export Options**: JSON, Markdown, PNG formats

## Development Notes
- Workflows use variable substitution for n8n URLs (e.g., `${service.n8n.url}`)
- Multiple UI variants available (standard and neural-enhanced)
- Supports both public and private mind maps with user-based filtering