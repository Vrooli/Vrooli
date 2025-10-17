# Document Manager

AI-powered documentation management SaaS platform for comprehensive analysis and quality maintenance.

## Quick Start

```bash
# Start the scenario (auto-starts required resources)
make run

# Run tests
make test

# View logs
make logs

# Stop everything
make stop
```

## Features

### ✅ Core Functionality (P0 - 100% Complete)
- **API Health Check**: Sub-2ms response times
- **Application Management**: Full CRUD operations for tracking multiple repositories
- **Agent System**: Configure AI agents for documentation tasks
- **Improvement Queue**: Track and manage documentation improvements by severity
- **Database Integration**: PostgreSQL for persistent storage
- **Web Interface**: Professional UI for managing applications and agents
- **CLI Tool**: Command-line interface for all operations

### ✅ Advanced Features (P1 - 100% Complete)
- **Vector Search**: ✅ Production-ready similarity search with Ollama embeddings (`/api/search` endpoint)
- **AI Integration**: ✅ Ollama nomic-embed-text model integrated for semantic embeddings (768 dimensions)
- **Document Indexing**: ✅ POST `/api/index` endpoint for batch document indexing
- **Data Management**: ✅ DELETE endpoints for applications, agents, and queue items
- **Real-time Updates**: ✅ Redis pub/sub for live event notifications (graceful degradation if Redis unavailable)
- **Batch Operations**: ✅ POST `/api/queue/batch` endpoint for bulk approve/reject/delete operations

## Architecture

```
┌─────────────────┐     ┌──────────────┐     ┌──────────────┐
│   Web UI        │────▶│   Go API     │────▶│  PostgreSQL  │
│  (dynamic port) │     │(dynamic port)│     │  (port 5433) │
└─────────────────┘     └──────────────┘     └──────────────┘
                              │
                    ┌─────────┴─────────┐
                    ▼                   ▼
              ┌──────────┐       ┌──────────┐
              │  Qdrant  │       │  Ollama  │
              │ (port 6333) │     │(port 11434)│
              └──────────┘       └──────────┘
```

**Note**: API and UI ports are dynamically allocated by Vrooli. Use `make status` to see current ports.

## API Endpoints

### System Status
- `GET /health` - Service health check
- `GET /api/system/db-status` - Database connectivity
- `GET /api/system/vector-status` - Vector database status
- `GET /api/system/ai-status` - AI model status

### Applications
- `GET /api/applications` - List all applications
- `POST /api/applications` - Create new application
- `DELETE /api/applications?id={uuid}` - Delete application (cascades to agents and queue items)
```json
{
  "name": "My App",
  "repository_url": "https://github.com/user/repo",
  "documentation_path": "/docs",
  "active": true
}
```

### Agents
- `GET /api/agents` - List all agents
- `POST /api/agents` - Create new agent
- `DELETE /api/agents?id={uuid}` - Delete agent (cascades to queue items)
```json
{
  "name": "Doc Analyzer",
  "type": "documentation_analyzer",
  "application_id": "uuid",
  "configuration": "{}",
  "enabled": true
}
```

### Improvement Queue
- `GET /api/queue` - List improvement items
- `POST /api/queue` - Add improvement item
- `DELETE /api/queue?id={uuid}` - Delete queue item
```json
{
  "agent_id": "uuid",
  "application_id": "uuid",
  "type": "documentation_improvement",
  "title": "Fix typo in README",
  "description": "Description here",
  "severity": "low",
  "status": "pending"
}
```

### Vector Search & Document Indexing (AI-Powered)

#### Index Documents
- `POST /api/index` - Index documents into Qdrant for semantic search

**Request:**
```json
{
  "application_id": "uuid-of-application",
  "documents": [
    {
      "id": "doc_readme_001",
      "path": "/README.md",
      "content": "Document content here...",
      "metadata": {"type": "readme", "section": "overview"}
    }
  ]
}
```

**Response:**
```json
{
  "indexed": 3,
  "failed": 0,
  "errors": []
}
```

#### Search Documents
- `POST /api/search` - Semantic search for similar documentation using Ollama embeddings

**Request:**
```json
{
  "query": "database architecture design patterns",
  "limit": 10
}
```

**Response:**
```json
{
  "results": [
    {
      "id": "4d92c6d0-c36c-54c1-b050-43ed2f751215",
      "score": 0.697,
      "document_id": "doc_setup_001",
      "content": "To set up Document Manager, first install PostgreSQL...",
      "metadata": {
        "application_id": "uuid",
        "application_name": "Test Application",
        "type": "setup",
        "path": "/docs/SETUP.md"
      },
      "application_name": "Test Application"
    }
  ],
  "query": "database architecture design patterns",
  "total": 1
}
```

**Features:**
- **Production AI Integration**: Uses Ollama's nomic-embed-text model for 768-dimensional semantic embeddings
- **Automatic Collection Management**: Creates Qdrant collections automatically
- **Deterministic Document IDs**: Uses UUID v5 for consistent document identification
- **Batch Indexing**: Index multiple documents in a single request
- **Rich Metadata**: Store and retrieve custom metadata with documents
- **Real Similarity Search**: Actual Qdrant vector search with cosine similarity scoring
- **Graceful Fallbacks**: Handles Ollama unavailability with informative errors

### Batch Queue Operations
- `POST /api/queue/batch` - Perform bulk operations on queue items

**Request:**
```json
{
  "action": "approve",  // or "reject", "delete"
  "ids": ["uuid1", "uuid2", "uuid3"]
}
```

**Response:**
```json
{
  "succeeded": ["uuid1", "uuid2"],
  "failed": ["uuid3"],
  "total": 3
}
```

**Actions:**
- `approve`: Bulk approve queue items (sets status to 'approved')
- `reject`: Bulk reject queue items (sets status to 'denied')
- `delete`: Bulk delete queue items

**Features:**
- **Atomic Per-Item**: Each item is processed independently; partial failures don't affect successful operations
- **Error Tracking**: Clear reporting of which items succeeded vs. failed
- **Real-time Events**: Publishes batch operation events to Redis for live UI updates

## CLI Usage

```bash
# Install CLI (done automatically by make run)
cd cli && ./install.sh

# Basic commands
document-manager --help
document-manager status
document-manager list applications
document-manager list agents
document-manager list improvements

# Create resources
document-manager create app --name "MyApp" --repo "https://github.com/..."
document-manager create agent --name "Analyzer" --app-id "uuid"
```

## Testing

```bash
# Run all tests
make test

# Run specific test suites
./test/integration-test.sh     # Integration tests
./test/security-check.sh        # Security validation
cd cli && bats document-manager.bats  # CLI tests
```

## Development

### Prerequisites
- Go 1.21+
- Node.js 18+
- Docker (for resources)
- Vrooli CLI

### Project Structure
```
document-manager/
├── api/              # Go API server
├── cli/              # CLI tool
├── ui/               # Web interface
├── lib/              # Helper scripts
├── test/             # Test suites
├── initialization/   # Database schemas and seeds
└── .vrooli/         # Service configuration
```

### Environment Variables
- `API_PORT` - API server port (auto-assigned)
- `UI_PORT` - Web UI port (auto-assigned)
- `POSTGRES_URL` - Database connection
- `QDRANT_URL` - Vector database URL
- `OLLAMA_URL` - AI model server
- `REDIS_URL` - Cache/pub-sub
- `N8N_URL` - Workflow automation
- `CORS_ALLOWED_ORIGINS` - CORS allowed origins (defaults to `http://localhost:${UI_PORT}`)

## Production Deployment

### Recommendations
1. **Add Authentication**: Implement JWT/session auth
2. **Enable TLS**: Use HTTPS in production
3. **Add Rate Limiting**: Protect against abuse
4. **Configure Monitoring**: Set up logging and metrics
5. **Backup Strategy**: Regular database backups

### Revenue Model
- **Target**: $25,000 - $50,000
- **Model**: SaaS subscription tiers
- **Pricing**: Based on applications monitored and agent usage

## Troubleshooting

### Scenario won't start
```bash
# Ensure resources are running
./lib/ensure-resources.sh

# Check logs
make logs
```

### Tests failing
```bash
# Check API health
curl http://localhost:17810/health

# Check UI health
curl http://localhost:38106/ui-health
```

### Port conflicts
```bash
# Stop all processes
make stop

# Check for lingering processes
ps aux | grep document-manager
pkill -f document-manager
```

## Support

For issues, check:
1. `PROBLEMS.md` - Known issues and solutions
2. `PRD.md` - Product requirements and specifications
3. Logs: `make logs`