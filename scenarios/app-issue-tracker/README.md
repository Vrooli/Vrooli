# App Issue Tracker

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.21%2B-blue.svg)](https://golang.org)
[![Node.js](https://img.shields.io/badge/Node.js-18%2B-green.svg)](https://nodejs.org)

A lightweight, file-based issue tracking system with AI-powered investigation capabilities. No database required – uses YAML files for storage.

## 🚀 Quick Start

### Prerequisites
- Go 1.21+ (for API)
- Node.js 18+ (for UI)
- Git (for version control)

### Installation
1. Clone the repository:
   ```bash
   git clone <repo-url>
   cd scenarios/app-issue-tracker
   ```

2. Start the scenario:
   ```bash
   make start
   ```

3. The service will be available at:
   - **Web UI**: http://localhost:[UI_PORT] (dynamically assigned, check `vrooli scenario status app-issue-tracker`)
   - **API**: http://localhost:[API_PORT]/health (dynamically assigned, typically 15000-19999 range)
   - **CLI**: `app-issue-tracker --help`

   Note: Ports are dynamically allocated. Use `vrooli scenario status app-issue-tracker` to see actual ports.

### Create Your First Issue
```bash
# Using CLI
app-issue-tracker create --title "Login button not working" --description "The login fails with error 500" --priority high --type bug

# View issues
app-issue-tracker list

# Investigate with AI
app-issue-tracker investigate "Login button not working"
```

## 🛠️ Features

### Core Functionality
- **Evidence Bundles**: Each issue lives in `data/issues/<status>/<issue-id>/metadata.yaml` with logs/screenshots under `artifacts/`
- **CLI Interface**: Full command-line issue management
- **REST API**: Programmatic access to all features
- **Web UI**: Visual interface for browsing and editing issues

### AI Integration
- **Automatic Investigation**: AI analyzes issues and suggests fixes
- **Semantic Search**: Find similar issues using vector embeddings
- **Codex Integration**: Leverages advanced AI for code-related issues
- **Workflow Automation**: Trigger investigations on issue creation/update

### Advanced Capabilities
- **Templates**: Pre-defined YAML templates for bugs, features, etc.
- **Status Management**: Track issues through open → active → completed → failed
- **Bulk Operations**: Search, filter, and act on multiple issues
- **Export/Import**: Generate reports in CSV, Markdown, or JSON

## 📁 Project Structure

```
app-issue-tracker/
├── api/                 # Go-based REST API server
│   ├── main.go         # API entrypoint
│   └── vector_search.go # Semantic search implementation
├── ui/                 # Web interface
│   ├── index.html      # Main UI page
│   ├── app.js          # Frontend logic
│   └── server.js       # Node.js dev server
├── cli/                # Command-line tools
│   └── app-issue-tracker # Main CLI executable
├── data/
│   └── issues/         # Folder-based issue storage
│       ├── open/               # Active issue bundles (`metadata.yaml` + artifacts)
│       ├── active/
│       ├── completed/
│       └── templates/          # YAML metadata templates
├── scripts/            # Utility scripts
│   ├── claude-investigator.sh
│   └── setup-vector-search.sh
├── test/               # Test suites
│   └── phases/         # Standardized test phases
└── docs/               # Documentation
    └── INVESTIGATION_WORKFLOW.md
```

## ⚙️ Configuration

### Environment Variables
- `API_PORT`: API server port (default: dynamically assigned)
- `UI_PORT`: Web UI port (default: dynamically assigned)
- `QDRANT_URL`: Vector database endpoint (optional)
- `OPENAI_API_KEY`: For Codex-powered investigations (optional)

### Security Configuration
For production deployments, configure security settings:
- `ENABLE_AUTH`: Enable API token authentication (default: false)
- `API_TOKENS`: Comma-separated list of valid API tokens
- `ALLOWED_ORIGINS`: Comma-separated list of allowed CORS origins (default: *)
- `RATE_LIMIT`: Maximum requests per time window (default: 100)

### GitHub Integration
To enable automatic PR creation:
- `GITHUB_TOKEN`: GitHub personal access token with `repo` scope
- `GITHUB_OWNER`: GitHub username or organization
- `GITHUB_REPO`: Repository name

**📖 See [docs/SECURITY_SETUP.md](docs/SECURITY_SETUP.md) for detailed security configuration guide.**

### Resources (Optional)
- **Qdrant**: For semantic search (enabled by default)
- **Redis**: For caching (enabled by default)

Disable in `.vrooli/service.json` if not needed.

## 🔧 Development

### Code Quality
```bash
# Format code
make fmt

# Lint code
make lint

# Run tests
make test

# Full check
make check
```

### Building
```bash
# Build API binary
make build

# Clean artifacts
make clean
```

### Testing
Tests are organized into phases:
- `test-unit.sh`: Unit tests for API and UI
- `test-integration.sh`: End-to-end integration
- `test-structure.sh`: File and directory validation
- `test-dependencies.sh`: Dependency checks
- `test-business.sh`: Business logic validation
- `test-performance.sh`: Performance benchmarks

Run all: `./test/run-tests.sh`

### Lifecycle Management
Always use the Makefile targets or Vrooli CLI:
```bash
make start    # Start services
make stop     # Stop services
make logs     # View logs
make status   # Check status
vrooli scenario run app-issue-tracker  # Alternative
```

**Never** run binaries directly (e.g., `./api/app-issue-tracker-api`) – this bypasses health checks and lifecycle management.

## 🔌 API Endpoints

Base URL: `http://localhost:${API_PORT}/api/v1`

### Issues
- `GET /issues` – List all issues
- `POST /issues` – Create new issue
- `GET /issues/{id}` – Get issue details
- `PUT /issues/{id}` – Update issue
- `DELETE /issues/{id}` – Delete issue

### Issue Creation Payload

```json
{
  "title": "Login button hangs",
  "description": "Users cannot sign in from mobile web",
  "type": "bug",
  "priority": "high",
  "app_id": "app-web",
  "tags": ["auth", "mobile"],
  "metadata_extra": {
    "report_source": "app-monitor",
    "logs_total": "18"
  },
  "reporter": {
    "name": "Casey QA",
    "email": "casey@example.com"
  },
  "artifacts": [
    {
      "name": "Lifecycle Logs",
      "category": "logs",
      "content": "...recent lifecycle output...",
      "encoding": "plain",
      "content_type": "text/plain"
    },
    {
      "name": "Preview Screenshot",
      "category": "screenshot",
      "content": "<base64>",
      "encoding": "base64",
      "content_type": "image/png"
    }
  ]
}
```

Artifacts are persisted on disk under `artifacts/` alongside `metadata.yaml`, and the API echoes back the recorded attachment metadata. Legacy fields such as `app_logs`, `console_logs`, `network_logs`, and `screenshot_data` are still accepted and automatically converted into artifacts for backwards compatibility.

### Search
- `GET /issues/search?q=query` – Keyword search
- `GET /issues/search/semantic?query=description` – Vector search (requires Qdrant)

### Investigation & Fixes
- `POST /investigate` – Trigger AI investigation for an issue
- `POST /generate-fix` – Generate fix for investigated issue
- `POST /issues/{id}/create-pr` – Create GitHub PR with fixes (requires GITHUB_TOKEN, GITHUB_OWNER, GITHUB_REPO)

### Export
- `GET /export?format=json&status=open` – Export issues (formats: json, csv, markdown)

### Health
- `GET /health` – Service status

See `api/main.go` for full documentation.

## 🎯 Usage Examples

### CLI Commands
```bash
# Create bug issue
app-issue-tracker create --template bug --title "UI crash on mobile" --priority critical

# Search issues
app-issue-tracker search "authentication error"

# Move issue to active state
app-issue-tracker update 001-auth --status active --assignee @john

# Run unified investigation + fix
app-issue-tracker investigate 001-auth

# Export report
app-issue-tracker export --format md --status open > weekly-report.md
```

### File Operations (Advanced)
Issues are stored as folder bundles:
```bash
# View raw issue metadata
cat data/issues/open/<issue-id>/metadata.yaml

# Inspect captured evidence
ls data/issues/open/<issue-id>/artifacts/

# Move status manually (preserves attachments)
mv data/issues/open/<issue-id> data/issues/completed/
```

### AI Workflow
1. Create issue: `app-issue-tracker create ...`
2. Trigger agent run: `app-issue-tracker investigate <id>`
3. Review AI suggestions in `data/issues/active/<issue-id>/metadata.yaml`
4. Agent auto-fixes or mark complete: `app-issue-tracker update <id> --status completed`

## 🐛 Troubleshooting

### Common Issues
- **Port conflicts**: Check `make status` and adjust ports in `.vrooli/service.json`
- **AI not working**: Ensure `OPENAI_API_KEY` is set or use mock mode
- **Search slow**: Install Qdrant or disable semantic search
- **YAML validation errors**: Check schema in `initialization/configuration/app-registry.json`

### Logs
View detailed logs:
```bash
make logs    # Last 50 lines
vrooli scenario logs app-issue-tracker --follow  # Live tail
```

### Debugging
- Run in dev mode: `make dev`
- Enable debug logging: Set `LOG_LEVEL=debug`
- Test phases individually: `bash test/phases/test-unit.sh`

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Commit changes: `git commit -m 'Add amazing feature'`
4. Run tests: `make check`
5. Push and create PR

See [CONTRIBUTING.md](../CONTRIBUTING.md) for details.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](../LICENSE) file for details.

## 🙏 Acknowledgments

- Built with [Vrooli](https://vrooli.com) scenario framework
- AI integration powered by [Codex](https://platform.openai.com/docs)
- Thanks to the open-source community for tools like Go, Node.js, and Qdrant

---

*For support, file an issue using this very tracker: `app-issue-tracker create --template bug`*
