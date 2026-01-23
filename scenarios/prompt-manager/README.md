# Prompt Manager

AI maintenance task management system for Vrooli, organizing automated code quality and maintenance skills with web interface, API, and CLI access.

## Features

- **AI Maintenance Tasks**: Pre-configured maintenance skills for code quality, testing, performance, and security
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
   Open http://localhost:${UI_PORT} in your browser (port provided by lifecycle system)

3. **Use the CLI:**
   ```bash
   prompt-manager help
   prompt-manager add "My first skill" debugging
   prompt-manager list
   ```

4. **API access:**
   The REST API is available at http://localhost:${API_PORT} (port provided by lifecycle system)

## Architecture

### Components

- **Go API Server** (port allocated by lifecycle): RESTful backend with campaign and skill management
- **React UI** (port allocated by lifecycle): Web interface with campaign sidebar and skill editor  
- **Bash CLI**: Command-line tool for quick operations
- **PostgreSQL**: Primary data storage
- **Qdrant** (optional): Vector database for semantic search
- **Ollama** (optional): Local LLM for prompt testing

### Database Schema

- **campaigns**: Organizing containers (debugging, UX, coding, etc.)
- **skills**: Individual skill content with metadata
- **tags**: Labeling system for skills
- **templates**: Reusable skill patterns
- **test_results**: History of skill testing with LLMs

## API Endpoints

### Campaigns
- `GET /api/campaigns` - List all campaigns
- `POST /api/campaigns` - Create new campaign
- `GET /api/campaigns/{id}` - Get campaign details
- `GET /api/campaigns/{id}/skills` - Get skills in campaign

### Skills
- `GET /api/skills` - List skills with filters
- `POST /api/skills` - Create new skill
- `GET /api/skills/{id}` - Get skill details
- `PUT /api/skills/{id}` - Update skill
- `POST /api/skills/{id}/use` - Record usage

### Search & Discovery
- `GET /api/search/skills?q={query}` - Full-text search
- `POST /api/skills/semantic` - Vector similarity search
- `GET /api/skills/recent` - Recently used skills
- `GET /api/skills/favorites` - Favorite skills

### Export/Import
- `GET /api/v1/export` - Export all data to JSON
  - Query params: `campaign_id` (filter by campaign), `include_archived` (include archived skills)
- `POST /api/v1/import` - Import data from JSON
  - Body: JSON export file content
  - Returns: Summary of imported items and any errors

## CLI Commands

```bash
# Status and health
prompt-manager status

# Campaign management
prompt-manager campaigns list
prompt-manager campaigns create "My Campaign" "Description"

# Skill operations
prompt-manager add "Skill title" campaign-name
prompt-manager list [campaign] [filter]
prompt-manager search "query"
prompt-manager show <skill-id>
prompt-manager use <skill-id>  # Copy and record usage

# Version control (NEW)
prompt-manager versions <skill-id>           # View version history
prompt-manager revert <skill-id> <version>   # Revert to previous version

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
bash cli/install.sh
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
- **Network**: Ports allocated dynamically by lifecycle system

## Optional Enhancements

### Semantic Search (requires Qdrant)
- Vector embeddings for skill content
- Similarity-based discovery
- Related skill suggestions

### Skill Testing (requires Ollama)
- Test skills with local LLMs
- Performance and quality metrics
- Effectiveness ratings

## Use Cases

- **Developers**: Debug patterns, code review templates, architecture decisions
- **Designers**: UX research methods, design system components, user journey analysis
- **Writers**: Content templates, documentation patterns, communication frameworks
- **General**: Personal AI skill library with organized access

## Data Flow

```
CLI/UI → Go API → PostgreSQL (metadata) 
                 → Qdrant (embeddings)
                 → Ollama (testing)
```

All interfaces interact through the central Go API server, ensuring consistent data handling and business logic.