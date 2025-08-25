# n8n - Business Workflow Automation

n8n is a powerful workflow automation platform that connects various services and automates business processes. This resource provides automated installation, configuration, and management of n8n with enhanced host system access for the Vrooli project.

## 🎯 Quick Reference

- **Category**: Automation
- **Port**: 5678 (Editor & API)
- **Container**: n8n
- **API Docs**: [Complete API Reference](docs/API.md)
- **Status**: Production Ready

## 🚀 Quick Start

### Prerequisites
- Docker installed and running
- 1GB+ RAM available
- Port 5678 available
- (Optional) PostgreSQL for production use

### Installation
```bash
# Install with custom image (recommended for host access)
./manage.sh --action install --build-image yes

# Install with standard n8n image
./manage.sh --action install --build-image no

# Install with PostgreSQL database
./manage.sh --action install --database postgres --build-image yes

# Force reinstall with custom settings
./manage.sh --action install --build-image yes --basic-auth yes --username admin --password mypass --force yes
```

### Basic Usage
```bash
# Check service status with comprehensive information  
./manage.sh --action status

# Test all functionality
./manage.sh --action test

# Execute workflow by ID (recommended method)
./manage.sh --action execute --workflow-id WORKFLOW_ID

# List all workflows
./manage.sh --action list-workflows

# View service logs
./manage.sh --action logs
```

### Verify Installation
```bash
# Check service health and functionality
./manage.sh --action status

# Test workflow management
./manage.sh --action list-workflows

# Access web interface: http://localhost:5678
# API base URL: http://localhost:5678/api/v1/
```

## 🔧 Core Features

- **🔄 Business Workflow Automation**: Visual workflow builder with 400+ integrations
- **💻 Host Command Access**: Run system commands directly from n8n workflows
- **🐳 Docker Integration**: Control other Docker containers from within n8n
- **📁 Workspace Access**: Direct access to the Vrooli project files
- **🔐 Security**: Basic authentication with encrypted credential storage
- **💾 Database Options**: SQLite (default) or PostgreSQL for production
- **🌐 Webhook Support**: External webhook endpoints for triggers
- **📊 API Access**: Full REST API for programmatic control

## 📖 Documentation

- **[API Reference](docs/API.md)** - REST API, CLI commands, and workflow management
- **[Configuration Guide](docs/CONFIGURATION.md)** - Installation options, environment variables, and setup
- **[Troubleshooting](docs/TROUBLESHOOTING.md)** - Common issues, diagnostics, and solutions

## 🎯 When to Use n8n

### Use n8n When:
- Building complex business workflow automation
- Integrating with 400+ SaaS services and APIs
- Need workflow execution history and audit trails
- Require scheduled and triggered data processing
- Want comprehensive credential management
- Building data transformation pipelines

### Consider Alternatives When:
- Need real-time monitoring and dashboards → [Node-RED](../node-red/)
- Want event-driven IoT integrations → [Node-RED](../node-red/)
- Building simple REST APIs → [Node-RED](../node-red/)
- Prefer continuous execution flows → [Node-RED](../node-red/)

## 🔗 Integration Examples

### Workflow Management
```bash
# Export workflows for backup (recommended)
./manage.sh --action export-workflows --output backup.json

# Import workflows from file
./manage.sh --action import-workflows --input backup.json

# Activate/deactivate workflows
./manage.sh --action activate-workflow --workflow-id WORKFLOW_ID
./manage.sh --action deactivate-workflow --workflow-id WORKFLOW_ID

# Database operations
./manage.sh --action backup-database --output db-backup.sql
./manage.sh --action restore-database --input db-backup.sql
```

### API Setup (Required for CLI Execution)
```bash
# Get API setup instructions (recommended)
./manage.sh --action api-setup

# Save API key to configuration (persists across sessions)
./manage.sh --action save-api-key --api-key YOUR_API_KEY

# Execute workflows with saved API key
./manage.sh --action execute --workflow-id WORKFLOW_ID
```

**⚠️ Important**: The n8n CLI command `n8n execute` is broken in versions 1.93.0+ (GitHub issue #15567). Use `./manage.sh --action execute` with API authentication instead.

### With Other Vrooli Resources
```javascript
// Execute Command node to check other resources
docker ps | grep ollama
docker exec ollama curl -s http://localhost:11434/api/tags

// HTTP Request node to monitor Node-RED
{
  "method": "GET",
  "url": "http://node-red:1880/flows",
  "name": "Check Node-RED Flows"
}
```

## ⚡ Key Architecture

### Custom Docker Image Benefits
When using `--build-image yes`, n8n gets enhanced capabilities:

```
Standard n8n → Custom Image → Host System Access
├── Basic workflow execution
├── Enhanced command execution
├── Docker container control
├── Workspace file access
└── Pre-installed tools (bash, git, curl, wget, jq, python3)
```

### Volume Mounts
```bash
# Data persistence
n8n-data:/home/node/.n8n                 # Workflows and settings

# Host integration (custom image only)
/var/run/docker.sock:/var/run/docker.sock # Docker control
${PWD}:/workspace:ro                      # Workspace access
/usr/bin:/host/usr/bin:ro                 # Host binaries
/home:/host/home                          # Home directory access
```

### n8n vs Node-RED Comparison

| Feature | n8n | Node-RED |
|---------|-----|----------|
| **Focus** | Business workflows | Real-time monitoring |
| **Integrations** | 400+ SaaS services | Host system, Docker, IoT |
| **Execution** | Scheduled, triggered | Event-driven, continuous |
| **UI** | Workflow editor only | Editor + dashboard |
| **Best For** | Data pipelines, automation | APIs, monitoring, IoT |

**Complementary Usage**: n8n for complex business logic → triggers Node-RED flows for real-time responses → results feed back to n8n workflows.

## 🆘 Getting Help

- Check [Troubleshooting Guide](docs/TROUBLESHOOTING.md) for common issues
- Run `./manage.sh --action status` for detailed diagnostics  
- View logs: `./manage.sh --action logs`
- Test functionality: `./manage.sh --action test`

## 🧪 Testing & Examples

### Individual Resource Tests
- **Test Location**: `__test/resources/single/automation/n8n.test.sh`
- **Test Coverage**: Service health, API functionality, workflow management, host system access
- **Run Test**: `cd __test/resources && ./quick-test.sh n8n`

### Working Examples
- **Examples Folder**: [examples/](examples/)
- **Available Examples**: 
  - `example-notification-workflow.json` - Notification automation workflow
  - `webhook-workflow.json` - Webhook-triggered processing
- **Integration Examples**: Multi-service workflows connecting n8n with Ollama, Agent-S2, and storage resources

### Integration with Scenarios
n8n is used in these business scenarios:
- **[Secure Document Processing](../../scenarios/secure-document-processing/)** - Compliant document workflows ($20k-40k projects)
- **[Analytics Dashboard](../../scenarios/analytics-dashboard/)** - Resource monitoring workflows ($15k-30k projects)

### Test Fixtures
- **Shared Test Data**: `__test/resources/fixtures/workflows/` (sample n8n workflows)
- **Integration Data**: `__test/resources/fixtures/documents/` (for document processing scenarios)

### Quick Test Commands
```bash
# Test individual n8n functionality
./__test/resources/quick-test.sh n8n

# Run all tests using n8n
./scripts/scenarios/tools/test-by-resource.sh --resource n8n
```

## 📦 What's Included

```
n8n/
├── manage.sh                    # Management script with API workaround
├── README.md                    # This overview
├── docs/                        # Detailed documentation
│   ├── API.md                  # Complete API reference
│   ├── CONFIGURATION.md        # Setup and configuration
│   └── TROUBLESHOOTING.md      # Issue resolution
├── lib/                        # Modular script components
├── config/                     # Configuration and defaults
├── docker/                     # Docker-related files
│   ├── Dockerfile              # Custom n8n image definition
│   └── docker-entrypoint.sh    # Enhanced entrypoint script
├── examples/                   # Example workflows
│   ├── example-notification-workflow.json
│   └── webhook-workflow.json
└── .bats files                 # Automated tests
```

---

**🔄 n8n excels at complex business workflow automation, making it perfect for connecting SaaS services, processing data pipelines, and orchestrating multi-step business processes with full audit trails and execution history.**