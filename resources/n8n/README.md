# n8n - Business Workflow Automation

n8n is a powerful workflow automation platform that connects various services and automates business processes. This resource provides automated installation, configuration, and management of n8n with enhanced host system access for the Vrooli project.

**v2.0 Contract Status**: ✅ Fully Compliant
**Last Updated**: 2025-01-10

## 🎯 Quick Reference

- **Category**: Automation
- **Port**: 5678 (Editor & API)
- **Container**: n8n
- **API Docs**: [Complete API Reference](docs/API.md)
- **Status**: Production Ready
- **v2.0 Features**: ✅ Full test suite, ✅ Secrets management, ✅ Content management

## 🚀 Quick Start

### Prerequisites
- Docker installed and running
- 1GB+ RAM available
- Port 5678 available
- (Optional) PostgreSQL for production use

### Installation
```bash
# Install with custom image (recommended for host access)
resource-n8n manage install --build-image yes

# Install with standard n8n image
resource-n8n manage install --build-image no

# Install with PostgreSQL database
resource-n8n manage install --database postgres --build-image yes

# Force reinstall with custom settings
resource-n8n manage install --build-image yes --basic-auth yes --username admin --password mypass --force yes
```

### Basic Usage
```bash
# Check service status with comprehensive information  
resource-n8n

# Test functionality (v2.0 compliant)
resource-n8n test all        # Run all tests
resource-n8n test smoke      # Quick health check
resource-n8n test integration # Full functionality test
resource-n8n test unit       # Library function tests

# Execute workflow by ID (recommended method)
resource-n8n content execute --workflow-id WORKFLOW_ID

# List all workflows
resource-n8n content list

# View service logs
resource-n8n logs

# Display credentials for integration
resource-n8n credentials --format json
```

### Verify Installation
```bash
# Check service health and functionality
resource-n8n

# Test workflow management
resource-n8n content list

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
resource-n8n content export --output backup.json

# Import workflows from file
resource-n8n content add --file backup.json

# Activate/deactivate workflows
resource-n8n content activate --workflow-id WORKFLOW_ID
resource-n8n content deactivate --workflow-id WORKFLOW_ID

# Database operations
resource-n8n manage backup --output db-backup.sql
resource-n8n manage restore --input db-backup.sql
```

### API Setup (Required for CLI Execution)
```bash
# Get API setup instructions (recommended)
resource-n8n content configure

# Save API key to configuration (persists across sessions)
resource-n8n content configure --api-key YOUR_API_KEY

# Execute workflows with saved API key
resource-n8n content execute --workflow-id WORKFLOW_ID
```

**⚠️ Important**: The n8n CLI command `n8n execute` is broken in versions 1.93.0+ (GitHub issue #15567). Use `resource-n8n content execute` with API authentication instead.

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
├── cli.sh                      # v2.0 CLI interface
├── README.md                   # This overview
├── PRD.md                      # Product Requirements Document
├── docs/                       # Detailed documentation
│   ├── API.md                  # Complete API reference
│   ├── CONFIGURATION.md        # Setup and configuration
│   └── TROUBLESHOOTING.md      # Issue resolution
├── lib/                        # Modular script components
│   ├── core.sh                 # Core functionality
│   ├── test.sh                 # Test orchestration
│   ├── content.sh              # Content management
│   ├── secrets.sh              # Secrets management
│   └── ...                     # Other libraries
├── config/                     # Configuration and defaults
│   ├── defaults.sh             # Default configuration
│   ├── runtime.json            # v2.0 runtime configuration
│   ├── schema.json             # Configuration schema
│   └── secrets.yaml            # Secrets declaration
├── test/                       # v2.0 test structure
│   ├── run-tests.sh            # Main test runner
│   └── phases/                 # Test phases
│       ├── test-smoke.sh       # Quick health check
│       ├── test-integration.sh # Full functionality
│       └── test-unit.sh        # Library validation
├── docker/                     # Docker-related files
│   ├── Dockerfile              # Custom n8n image definition
│   └── docker-entrypoint.sh   # Enhanced entrypoint script
├── examples/                   # Example workflows
│   ├── example-notification-workflow.json
│   └── webhook-workflow.json
└── .bats files                 # Automated tests
```

---

**🔄 n8n excels at complex business workflow automation, making it perfect for connecting SaaS services, processing data pipelines, and orchestrating multi-step business processes with full audit trails and execution history.**