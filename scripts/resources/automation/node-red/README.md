# Node-RED - Real-time Flow Programming

Node-RED is a flow-based programming tool for wiring together hardware devices, APIs, and online services. This resource provides automated installation, configuration, and management of Node-RED with enhanced host system access for the Vrooli project.

## 🎯 Quick Reference

- **Category**: Automation
- **Port**: 1880 (Editor), 1880/ui (Dashboard)
- **Container**: node-red
- **API Docs**: [Complete API Reference](docs/API.md)
- **Status**: Production Ready

## 🚀 Quick Start

### Prerequisites
- Docker installed and running
- 512MB+ RAM available
- Port 1880 available
- (Optional) Docker socket access for container management

### Installation
```bash
# Install with custom image (recommended for host access)
./manage.sh --action install --build-image yes

# Install with standard Node-RED image
./manage.sh --action install --build-image no

# Force reinstall if already exists
./manage.sh --action install --build-image yes --force yes
```

### Basic Usage
```bash
# Check service status
./manage.sh --action status

# Test all functionality
./manage.sh --action test

# Test host command access
./manage.sh --action validate-host

# Test Docker integration
./manage.sh --action validate-docker

# View performance metrics
./manage.sh --action metrics
```

### Verify Installation
```bash
# Check service health and functionality
./manage.sh --action status

# Test flow management
./manage.sh --action flow-list

# Access web interfaces:
# Editor: http://localhost:1880
# Dashboard: http://localhost:1880/ui
```

## 🔧 Core Features

- **🔄 Flow-based Programming**: Visual programming using flows and nodes
- **💻 Host Command Access**: Run system commands directly from Node-RED flows
- **🐳 Docker Integration**: Control other Docker containers from within Node-RED
- **📁 Workspace Access**: Direct access to the Vrooli project files
- **📊 Dashboard Support**: Built-in dashboard for creating UIs
- **🌐 HTTP Endpoints**: Create REST APIs and webhooks
- **📈 Real-time Processing**: Event-driven flows with continuous execution
- **🔧 Enhanced Tools**: Pre-installed nodes for common automation tasks

## 📖 Documentation

- **[API Reference](docs/API.md)** - HTTP endpoints, WebSocket API, and flow management
- **[Configuration Guide](docs/CONFIGURATION.md)** - Installation options, settings, and customization
- **[Troubleshooting](docs/TROUBLESHOOTING.md)** - Common issues and diagnostic procedures
- **[Flow Examples](flows/)** - Pre-built flows for monitoring and automation

## 🎯 When to Use Node-RED

### Use Node-RED When:
- Building real-time system monitoring and dashboards
- Creating event-driven automation with immediate response
- Developing REST APIs and webhooks quickly
- Integrating with Docker containers and host systems
- Setting up IoT and hardware integrations
- Building development and testing tools

### Consider Alternatives When:
- Need business workflow orchestration → [n8n](../n8n/)
- Require complex data transformations → [n8n](../n8n/)
- Want audit trails and execution history → [n8n](../n8n/)
- Need 300+ pre-built SaaS integrations → [n8n](../n8n/)

## 🔗 Integration Examples

### Flow Management
```bash
# Export flows for backup
./manage.sh --action flow-export --output ./my-flows.json

# Import flows from file
./manage.sh --action flow-import --flow-file ./my-flows.json

# Execute specific flow endpoint
./manage.sh --action flow-execute --endpoint "/api/resource-check"

# Execute flow with data
./manage.sh --action flow-execute --endpoint "/test/exec" --data '{"command": "docker ps"}'
```

### With Other Vrooli Resources
```javascript
// HTTP request node to check Ollama status
{
  "method": "GET",
  "url": "http://ollama:11434/api/tags",
  "name": "Check Ollama Models"
}

// Function node for Docker container management
exec('docker restart n8n', (error, stdout, stderr) => {
    msg.payload = {
        success: !error,
        output: stdout,
        error: error ? error.message : null
    };
    node.send(msg);
});
```

## ⚡ Key Architecture

### Flow Execution Model
Node-RED uses an asynchronous, event-driven architecture:

```
Message Object → Node Chain → Asynchronous Processing → Multiple Outputs
```

**Message Structure**:
```javascript
msg = {
    payload: {...},        // Main data content
    topic: "resource-name", // Message routing/identification  
    url: "http://...",     // Dynamic HTTP endpoints
    resource: {...},       // Custom metadata preservation
    statusCode: 200,       // HTTP response codes
    responseTime: 45       // Performance metrics
}
```

### Node-RED vs n8n Comparison

| Feature | Node-RED | n8n |
|---------|----------|-----|
| **Focus** | Real-time monitoring & APIs | Business workflows |
| **Execution** | Event-driven, continuous | Scheduled, triggered |
| **UI** | Flow editor + dashboard | Workflow editor only |
| **Integration** | Host system, Docker, IoT | 300+ SaaS services |
| **Best For** | Monitoring, APIs, IoT | Data pipelines, automation |

**Complementary Usage**: Node-RED for real-time monitoring → triggers n8n workflows for complex processing → results displayed in Node-RED dashboards.

## 🆘 Getting Help

- Check [Troubleshooting Guide](docs/TROUBLESHOOTING.md) for common issues
- Run `./manage.sh --action status` for detailed diagnostics
- View logs: `./manage.sh --action logs`
- Test functionality: `./manage.sh --action test`

## 📦 What's Included

```
node-red/
├── manage.sh                    # Management script
├── README.md                    # This overview
├── docs/                        # Detailed documentation
│   ├── API.md                  # Complete API reference
│   ├── CONFIGURATION.md        # Setup and configuration
│   └── TROUBLESHOOTING.md      # Issue resolution
├── flows/                       # Pre-built flow examples
│   ├── README.md               # Flow documentation
│   ├── test-basic.json         # Basic connectivity test
│   ├── test-exec.json          # Command execution test
│   └── vrooli-monitor.json     # Resource monitoring flow
├── lib/                        # Helper scripts and functions
├── config/                     # Configuration files
├── Dockerfile                  # Custom Node-RED image
└── docker-entrypoint.sh       # Enhanced entrypoint
```

---

**🔄 Node-RED excels at real-time system integration, making it perfect for monitoring Vrooli resources, creating responsive dashboards, and building event-driven automation that reacts immediately to system changes.**