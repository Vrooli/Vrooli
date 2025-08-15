# Prompt Manager

AI maintenance task management system for Vrooli, organizing automated code quality and maintenance prompts with web interface, API, and CLI access.

## Features

- **AI Maintenance Tasks**: Pre-configured maintenance prompts for code quality, testing, performance, and security
- **Campaign Organization**: Tasks grouped by purpose (Testing, Performance, Security, UX, Code Health, etc.)
- **Task ID System**: Track AI maintenance work with standardized IDs (TEST_QUALITY, REACT_PERF, etc.)
- **Multiple Interfaces**: Web UI, REST API, and command-line tool for accessing maintenance tasks
- **Quick Access Keys**: Fast access to frequently used maintenance tasks
- **Usage Tracking**: Monitor which maintenance tasks are performed most often

## Quick Start

1. **Start the application:**
   ```bash
   bash deployment/startup.sh
   ```

2. **Access the web interface:**
   Open http://localhost:3005 in your browser

3. **Use the CLI:**
   ```bash
   prompt-manager help
   prompt-manager add "My first prompt" debugging
   prompt-manager list
   ```

4. **API access:**
   The REST API is available at http://localhost:8085

## Architecture

### Components

- **Go API Server** (port 8085): RESTful backend with campaign and prompt management
- **React UI** (port 3005): Web interface with campaign sidebar and prompt editor  
- **Bash CLI**: Command-line tool for quick operations
- **PostgreSQL**: Primary data storage
- **Qdrant** (optional): Vector database for semantic search
- **Ollama** (optional): Local LLM for prompt testing

### Database Schema

- **campaigns**: Organizing containers (debugging, UX, coding, etc.)
- **prompts**: Individual prompt content with metadata
- **tags**: Labeling system for prompts
- **templates**: Reusable prompt patterns
- **test_results**: History of prompt testing with LLMs

## API Endpoints

### Campaigns
- `GET /api/campaigns` - List all campaigns
- `POST /api/campaigns` - Create new campaign
- `GET /api/campaigns/{id}` - Get campaign details
- `GET /api/campaigns/{id}/prompts` - Get prompts in campaign

### Prompts
- `GET /api/prompts` - List prompts with filters
- `POST /api/prompts` - Create new prompt
- `GET /api/prompts/{id}` - Get prompt details
- `PUT /api/prompts/{id}` - Update prompt
- `POST /api/prompts/{id}/use` - Record usage

### Search & Discovery
- `GET /api/search/prompts?q={query}` - Full-text search
- `POST /api/prompts/semantic` - Vector similarity search
- `GET /api/prompts/recent` - Recently used prompts
- `GET /api/prompts/favorites` - Favorite prompts

## CLI Commands

```bash
# Status and health
prompt-manager status

# Campaign management  
prompt-manager campaigns list
prompt-manager campaigns create "My Campaign" "Description"

# Prompt operations
prompt-manager add "Prompt title" campaign-name
prompt-manager list [campaign] [filter]
prompt-manager search "query"
prompt-manager show <prompt-id>
prompt-manager use <prompt-id>  # Copy and record usage

# Quick access
prompt-manager quick <key>      # Access by quick key
```

## Configuration

### App Configuration
Located in `initialization/configuration/app-config.json`:
- Port settings
- Database configuration  
- Feature toggles
- UI preferences
- Resource limits

### Campaign Templates
Located in `initialization/configuration/campaign-templates.json`:
- Pre-configured campaign types
- Color and icon schemes
- Quick setup options

## Development

### Prerequisites
- Go 1.21+
- Node.js 16+
- PostgreSQL
- (Optional) Qdrant, Ollama

### Setup
```bash
# Database
createdb prompt_manager
psql -d prompt_manager < initialization/storage/postgres/schema.sql
psql -d prompt_manager < initialization/storage/postgres/seed.sql

# API Server
cd api
go mod tidy
go run main.go

# UI Development
cd ui  
npm install
npm start

# CLI Installation
bash cli/install-cli.sh
```

### Testing
```bash
# Validation
bash deployment/validate.sh

# Integration tests
# (Test specification in scenario-test.yaml)
```

## Resource Requirements

- **Memory**: ~200MB
- **Storage**: ~50MB initial + data growth
- **CPU**: Minimal (1 core sufficient)
- **Network**: Ports 3005 (UI), 8085 (API)

## Optional Enhancements

### Semantic Search (requires Qdrant)
- Vector embeddings for prompt content
- Similarity-based discovery
- Related prompt suggestions

### Prompt Testing (requires Ollama)  
- Test prompts with local LLMs
- Performance and quality metrics
- Effectiveness ratings

## Use Cases

- **Developers**: Debug patterns, code review templates, architecture decisions
- **Designers**: UX research methods, design system components, user journey analysis  
- **Writers**: Content templates, documentation patterns, communication frameworks
- **General**: Personal AI prompt library with organized access

## Data Flow

```
CLI/UI → Go API → PostgreSQL (metadata) 
                 → Qdrant (embeddings)
                 → Ollama (testing)
```

All interfaces interact through the central Go API server, ensuring consistent data handling and business logic.