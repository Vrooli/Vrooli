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

### 🚧 Advanced Features (P1 - Infrastructure Ready)
- **Vector Search**: Qdrant connected, implementation pending
- **AI Analysis**: Ollama connected, integration pending
- **Real-time Updates**: Redis connected, pub/sub pending
- **Batch Operations**: Infrastructure ready, implementation pending

## Architecture

```
┌─────────────────┐     ┌──────────────┐     ┌──────────────┐
│   Web UI        │────▶│   Go API     │────▶│  PostgreSQL  │
│  (port 38106)   │     │ (port 17810) │     │  (port 5433) │
└─────────────────┘     └──────────────┘     └──────────────┘
                              │
                    ┌─────────┴─────────┐
                    ▼                   ▼
              ┌──────────┐       ┌──────────┐
              │  Qdrant  │       │  Ollama  │
              │ (vectors)│       │   (AI)   │
              └──────────┘       └──────────┘
```

## API Endpoints

### System Status
- `GET /health` - Service health check
- `GET /api/system/db-status` - Database connectivity
- `GET /api/system/vector-status` - Vector database status
- `GET /api/system/ai-status` - AI model status

### Applications
- `GET /api/applications` - List all applications
- `POST /api/applications` - Create new application
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