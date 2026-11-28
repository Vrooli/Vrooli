# SmartNotes

## Purpose
Local note-taking system with intelligent organization and real-time suggestions. It provides persistent storage for scenarios that need to retain notes or documentation.

## Usefulness to Vrooli
- **Knowledge Base**: Provides persistent storage for any scenario needing to maintain notes or documentation
- **Search**: Offers tenant-scoped text search across notes
- **AI Processing**: Automatic tagging, summarization, and linking that enhances stored knowledge
- **Cross-Scenario Memory**: Other scenarios can store and retrieve contextual information through the notes API

## Dependencies
### Required Resources
- **PostgreSQL**: Primary storage for notes, folders, and metadata
- **Ollama**: AI model for note analysis and suggestions

AI processing is orchestrated directly by the API, so no external workflow engine is required. Semantic embedding is not currently served.

### Automation Modules Used
- AI text generation and analysis pipelines (previously defined as `ollama.json`)
- Structured data extraction routines (formerly `structured-data-extractor.json`)
- Chain-of-thought coordination for reasoning (formerly `chain-of-thought-orchestrator.json`)
- Iterative AI processing control loops (formerly `react-loop-engine.json`)

Vector semantic search is currently removed from the API and CLI. The prior endpoint returned 503 because no admitted vector-store and embedding owner was wired. The durable semantic write feed and owner integration contract remain for a future qualified serving path.

## Architecture
### API (Go)
- RESTful endpoints for CRUD operations
- Integration with internal automation modules for AI processing
- PostgreSQL-backed storage and tenant-scoped text search
- Semantic-vector serving is removed until its embedding, collection, and admission owners are wired

### CLI
- Command-line interface for quick note operations
- Commands: `list`, `new`, `view`, `edit`, `delete`, `search`, `folders`, `tags`, `templates`, `summary`
- Environment-aware configuration

### UI
- Two interfaces: Standard and Zen mode
- Real-time note editing with markdown support
- AI-powered suggestions and auto-tagging
- Semantic search interface

## UX Style
**Standard Mode**: Clean, modern interface optimized for productivity with collapsible sidebar, markdown preview, and keyboard shortcuts

**Zen Mode**: Distraction-free writing environment with minimal UI, focus mode, and ambient features

## Key Features
1. **Intelligent Processing**: Automatic summarization, tagging, and linking
2. **Semantic Search**: Find notes by meaning, not just keywords
3. **Daily Summaries**: AI-generated overview of daily notes
4. **Smart Suggestions**: Context-aware writing assistance
5. **Template System**: Pre-built note structures for common use cases
6. **Folder Organization**: Hierarchical structure with smart categorization
7. **Cross-Linking**: Automatic discovery of related notes

## Usage by Other Scenarios
Other scenarios can interact with SmartNotes through:
- **API**: `POST /api/notes` to store context, `GET /api/notes/search` for retrieval, and `/api/notes/process` for AI enrichment
- **CLI**: `notes new "Context from scenario X"` for quick storage
