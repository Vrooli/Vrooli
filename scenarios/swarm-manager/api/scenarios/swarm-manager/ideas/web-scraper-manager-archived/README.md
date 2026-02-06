# Web Scraper Manager

Unified dashboard for managing web scraping across multiple platforms (Huginn, Browserless, Agent-S2) with data extraction monitoring, job orchestration, and analytics.

## Overview

Web Scraper Manager provides a centralized interface to:
- Manage scraping agents across multiple platforms
- Monitor job execution in real-time
- Analyze and export scraped data
- Configure scraping targets and schedules
- Track performance metrics and analytics

## Features

### Core Capabilities
- **Multi-Platform Support**: Integrate with Huginn, Browserless, and Agent-S2
- **Agent Management**: Create, configure, and manage scraping agents
- **Job Orchestration**: Schedule and execute scraping jobs with Redis queues
- **Data Storage**: PostgreSQL for metadata, MinIO for assets and exports
- **Vector Search**: Qdrant for content similarity detection (optional)
- **AI-Powered**: Ollama integration for intelligent content extraction (optional)

### Web Dashboard
- Real-time job monitoring
- Agent configuration interface
- Data exploration and export tools
- Performance analytics and reporting
- Platform capability comparison

## Quick Start

### Setup
```bash
# From the scenario directory
make setup

# Or using vrooli CLI
vrooli scenario setup web-scraper-manager
```

### Start Services
```bash
# Start all services (API + UI)
make run

# Or using vrooli CLI
vrooli scenario start web-scraper-manager
```

Services will be available at:
- Dashboard: `http://localhost:${UI_PORT}`
- API: `http://localhost:${API_PORT}`
- MinIO Console: `http://localhost:${MINIO_PORT}`

### CLI Usage
```bash
# Check status
web-scraper-manager status

# List agents
web-scraper-manager agents list

# Create a new agent
web-scraper-manager agents create "My Agent" browserless content

# Execute an agent
web-scraper-manager agents execute <agent-id>

# View results
web-scraper-manager results list

# Export data
web-scraper-manager export json
```

## Architecture

### Components
```
┌─────────────────────────────────────────────────┐
│              Web Dashboard (UI)                  │
│              Node.js + Vanilla JS                │
└─────────────────┬───────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────┐
│           Go API Server                          │
│   - Agent Management                             │
│   - Job Orchestration                            │
│   - Data Export                                  │
└─────────────────┬───────────────────────────────┘
                  │
         ┌────────┼────────┐
         │        │        │
    ┌────▼──┐ ┌──▼───┐ ┌─▼────┐
    │Postgres│ │Redis │ │MinIO │
    └────────┘ └──────┘ └──────┘
         │        │        │
    ┌────▼────────▼────────▼────┐
    │   External Platforms       │
    │  - Huginn                  │
    │  - Browserless             │
    │  - Agent-S2                │
    └────────────────────────────┘
```

### Data Flow
1. User creates agent via UI/CLI
2. Agent configuration stored in PostgreSQL
3. Jobs queued in Redis
4. API coordinates execution across platforms
5. Results stored in PostgreSQL
6. Assets (screenshots, files) stored in MinIO
7. Optional: Content vectors stored in Qdrant

## API Endpoints

### Health
- `GET /health` - API health check

### Agents
- `GET /api/agents` - List all agents
- `GET /api/agents?platform=<platform>` - List agents by platform
- `POST /api/agents` - Create new agent
- `GET /api/agents/:id` - Get agent details
- `PUT /api/agents/:id` - Update agent
- `DELETE /api/agents/:id` - Delete agent
- `POST /api/agents/:id/execute` - Execute agent

### Results
- `GET /api/results` - List all results
- `GET /api/agents/:id/results` - Get agent results
- `GET /api/results/:id` - Get result details

### Platforms
- `GET /api/platforms` - List supported platforms and capabilities

### Metrics
- `GET /api/metrics` - System metrics and statistics

### Export
- `POST /api/export` - Trigger data export (JSON, CSV, XML)

## Testing

### Run All Tests
```bash
make test
```

### Individual Test Suites
```bash
# API health check
curl -sf http://localhost:${API_PORT}/health

# Database connectivity
bash test/test-database-connection.sh

# Storage buckets
bash test/test-storage-buckets.sh

# CLI tests
cd cli && bats web-scraper-manager.bats
```

### Test Coverage
- Go build validation
- API endpoint health checks
- Database schema verification
- MinIO bucket accessibility
- CLI command validation

## Configuration

### Environment Variables
- `API_PORT` - API server port (auto-assigned)
- `UI_PORT` - UI server port (auto-assigned)
- `WEB_SCRAPER_MANAGER_API_URL` - Override API URL for CLI

### Resource Dependencies

#### Required Resources
- **PostgreSQL**: Stores agent configs, results, and metadata
- **Redis**: Job queues and caching
- **MinIO**: Asset storage and exports

#### Optional Resources
- **Qdrant**: Content similarity detection
- **Ollama**: AI-powered content extraction
- **Huginn**: RSS/social media monitoring
- **Browserless**: JavaScript rendering and screenshots
- **Agent-S2**: AI-guided web interaction

### Database Schema
The PostgreSQL database includes:
- `scraping_agents` - Agent configurations
- `scraping_targets` - Target URLs and selectors
- `scraping_results` - Execution results
- `proxy_pool` - Proxy management
- `api_endpoints` - Platform endpoint tracking

### MinIO Buckets
- `scraper-assets` - Downloaded content and files
- `screenshots` - Page screenshots
- `exports` - Data export files

## Platform Integration

### Huginn
Best for: RSS feeds, social media monitoring, scheduled scraping
- Supports chaining multiple agents
- Template-based extraction
- Webhook integrations

### Browserless
Best for: JavaScript-heavy sites, screenshots, PDF generation
- Stealth mode support
- Parallel execution
- Cookie management
- Proxy support

### Agent-S2
Best for: Complex interactions, dynamic content, AI-guided scraping
- Natural language instructions
- Adaptive learning
- Visual recognition
- Complex workflow automation

## Development

### Build
```bash
# Build API
cd api && go build -o web-scraper-manager-api .

# Install UI dependencies
cd ui && npm install
```

### Project Structure
```
web-scraper-manager/
├── api/                    # Go API server
│   ├── main.go            # Main server
│   ├── scheduler.go       # Job scheduling
│   ├── scraper.go         # Platform integrations
│   └── test_*.go          # Unit tests
├── cli/                   # CLI tool
│   ├── web-scraper-manager # Main CLI script
│   ├── web-scraper-manager.bats # CLI tests
│   └── install.sh         # CLI installation
├── ui/                    # Web dashboard
│   ├── server.js          # Node.js UI server
│   └── public/            # Static assets
├── initialization/        # Database schemas
│   └── storage/postgres/
├── scripts/              # Setup scripts
├── test/                 # Integration tests
├── Makefile             # Build and test commands
└── PRD.md              # Product requirements
```

## Troubleshooting

### API Won't Start
```bash
# Check if port is available
lsof -i :${API_PORT}

# Check resource status
vrooli resource status postgres
vrooli resource status redis
vrooli resource status minio

# View logs
make logs
```

### Database Connection Issues
```bash
# Test database connection
bash test/test-database-connection.sh

# Verify schema
psql -U postgres -d vrooli -c "\dt"
```

### MinIO Access Issues
```bash
# Test bucket accessibility
bash test/test-storage-buckets.sh

# Check MinIO status
vrooli resource status minio
```

### Platform Integration Issues
- Verify platform is running and accessible
- Check platform health endpoints
- Review platform-specific logs
- Verify API credentials/configuration

## Performance

### Response Time Targets
- Health endpoint: < 500ms
- Agent list: < 1s
- Job execution trigger: < 500ms
- Data export: < 30s for standard datasets

### Scalability
- Supports multiple concurrent scraping jobs via Redis queues
- Horizontal scaling through platform distribution
- Asset storage scales with MinIO
- Database indexing for fast queries

## Security

### Best Practices
- Use environment variables for sensitive configuration
- Implement rate limiting on API endpoints
- Validate all user inputs
- Sanitize scraped content before storage
- Use proxy pools for anonymity
- Rotate credentials regularly

### Access Control
- API authentication (to be implemented)
- Role-based access control (planned)
- Audit logging (planned)

## Business Value

### Use Cases
1. **Market Research**: Monitor competitor pricing and product listings
2. **Content Aggregation**: Collect news and blog posts for analysis
3. **Lead Generation**: Extract business contact information
4. **Price Monitoring**: Track product prices across e-commerce sites
5. **Social Media Monitoring**: Aggregate social media mentions and trends

### Revenue Potential
- **SaaS Deployment**: $50-200/month per user
- **Enterprise Licensing**: $10K-50K annual contracts
- **Custom Integration Services**: $5K-20K per project
- **Estimated ARR**: $50K-150K with 100-500 users

## Contributing

See [CLAUDE.md](/CLAUDE.md) for development guidelines and contribution process.

## License

Part of the Vrooli ecosystem. See main repository for license details.

## Support

- Documentation: `/docs`
- Issues: File in main Vrooli repository
- CLI Help: `web-scraper-manager help`
