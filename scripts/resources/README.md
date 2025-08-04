# Vrooli Resource Management System

Local AI and automation tools that extend Vrooli's capabilities through modular services.

## 🚀 Quick Start

```bash
# Discover what's running
./index.sh --action discover

# Install specific resources
./scripts/main/setup.sh --resources "ollama,n8n,agent-s2"

# Check resource status
./index.sh --action status --resources ollama
```

## 📦 Available Resources

| Category | Resources | Purpose |
|----------|-----------|---------|
| **AI** | ollama, whisper, unstructured-io, comfyui | Local AI inference and processing |
| **Automation** | n8n, node-red, windmill, huginn | Workflow orchestration |
| **Agents** | agent-s2, browserless, claude-code | Web/desktop automation |
| **Search** | searxng | Privacy-respecting search |
| **Storage** | postgres, redis, minio, vault, qdrant, questdb | Data persistence |
| **Execution** | judge0 | Code execution sandboxing |

## 📚 Documentation

- **[Complete Guide](docs/README.md)** - Detailed resource documentation
- **[Integration Cookbook](docs/integration-cookbook.md)** - Multi-resource workflows
- **[Interface Standards](docs/interface-standards.md)** - Resource API contracts
- **[Port Registry](port-registry.sh)** - Service port allocations (`./port-registry.sh --action list`)
- **[Architecture Overview](docs/README.md#-integration-patterns)** - How resources work together

## 🔧 Management

- **Discovery**: `./index.sh --action discover`
- **Installation**: See [setup documentation](../main/README.md)
- **Configuration**: `~/.vrooli/service.json`
- **Validation**: `./tools/validate-interfaces.sh`

## 🎯 Scenario Deployment

Resources support declarative deployment through the [injection system](_injection/README.md), 
enabling complete application deployment from JSON specifications.

## 🛠️ Resource Structure

Each resource follows a consistent structure:
```
resource-name/
├── manage.sh           # Main management script
├── lib/               # Functionality libraries
├── config/            # Configuration files
├── docs/              # Resource-specific documentation
└── examples/          # Usage examples
```

## 🔍 Common Operations

```bash
# Install and start a resource
./ai/ollama/manage.sh --action install
./ai/ollama/manage.sh --action start

# View logs
./automation/n8n/manage.sh --action logs

# Check health across all resources
./index.sh --action discover
```

---

For detailed documentation, integration patterns, and troubleshooting, see **[docs/README.md](docs/README.md)**