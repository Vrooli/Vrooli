# SmartNotes

## Purpose
Local AI-enabled note-taking system with intelligent organization, semantic search, and real-time suggestions. Acts as a foundational knowledge management tool that other scenarios can leverage for persistent memory and context.

## Usefulness to Vrooli
- **Knowledge Base**: Provides persistent storage for any scenario needing to maintain notes or documentation
- **Semantic Search**: Offers vector-based search that other scenarios can query via API
- **AI Processing**: Automatic tagging, summarization, and linking that enhances stored knowledge
- **Cross-Scenario Memory**: Other scenarios can store and retrieve contextual information through the notes API

## Dependencies
### Required Resources
- **n8n**: Workflow automation for note processing and AI operations
- **PostgreSQL**: Primary storage for notes, folders, and metadata
- **Qdrant**: Vector database for semantic search capabilities
- **Ollama**: AI model for note analysis and suggestions

### Shared Workflows Used
- `ollama.json`: AI text generation and analysis
- `embedding-generator.json`: Vector embedding creation
- `semantic-search.json`: Vector similarity search
- `structured-data-extractor.json`: Extract structured data from notes
- `chain-of-thought-orchestrator.json`: Complex reasoning workflows
- `react-loop-engine.json`: Iterative AI processing
- `smart-semantic-search.json`: Enhanced search with context

## Architecture
### API (Go)
- RESTful endpoints for CRUD operations
- Integration with n8n workflows for AI processing
- Direct connections to PostgreSQL and Qdrant

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
- **API**: `POST /api/notes` to store context, `GET /api/notes/search` for retrieval
- **CLI**: `notes new "Context from scenario X"` for quick storage
- **Workflows**: Execute note-processor workflow for AI-enhanced storage