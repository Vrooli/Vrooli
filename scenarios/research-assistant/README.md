# Research Assistant

## Purpose
An intelligent research assistant that helps users gather, analyze, and synthesize information from multiple sources. It provides automated research workflows, document analysis, and knowledge management capabilities.

## Key Features
- **Multi-source Research**: Gathers information from 70+ search engines via SearXNG
- **Advanced Search Filters**: Language, safe search, file type, site filtering, date ranges, and sorting
- **Intelligent Analysis**: Uses Ollama AI models to analyze and synthesize research findings
- **RAG Support**: Implements retrieval-augmented generation for contextual responses
- **Scheduled Reports**: Can generate periodic research summaries via n8n automation
- **Knowledge Storage**: Stores research data in Qdrant for semantic search
- **Privacy-First**: All data processing happens locally, no external API keys required

## Dependencies
- **Resources**: ollama, qdrant, postgres, minio, n8n, searxng
- **Shared Workflows**: 
  - `ollama.json` - For AI text generation
  - `embedding-generator.json` - For generating text embeddings
  
## Architecture
- **API**: Go-based REST API for research operations
- **CLI**: Command-line interface for local research tasks
- **Workflows**: N8n orchestration for complex research pipelines
- **Storage**: PostgreSQL for metadata, Qdrant for vectors, MinIO for documents

## Use Cases
- Academic research assistance
- Market research and competitive analysis
- Technical documentation research
- News and information aggregation
- Knowledge base building

## Integration with Other Scenarios
This scenario provides research capabilities that can be leveraged by:
- `product-manager` - For market research and competitive analysis
- `study-buddy` - For academic research support
- `stream-of-consciousness-analyzer` - For organizing research notes
- Other scenarios requiring information gathering and synthesis

## UI Style
Professional, clean interface with focus on information density and readability. Dark mode support for extended research sessions.