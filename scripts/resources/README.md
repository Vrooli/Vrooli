# Vrooli Resource Management System

Local AI and automation tools that extend Vrooli's capabilities through modular services.

## 🚀 Quick Start

```bash
# Discover what's running
./index.sh --action discover

# Install specific resources via CLI
./scripts/resources/index.sh --action install --resources "ollama,n8n,agent-s2"

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
- **[Testing Strategy](docs/TESTING_STRATEGY.md)** - Three-layer validation system
- **[Integration Cookbook](docs/integration-cookbook.md)** - Multi-resource workflows
- **[Interface Standards](docs/interface-standards.md)** - Resource API contracts
- **[Port Registry](port_registry.sh)** - Service port allocations (`./port_registry.sh --action list`)
- **[Architecture Overview](docs/README.md#-integration-patterns)** - How resources work together

## 🔧 Management

- **Discovery**: `./index.sh --action discover`
- **Installation/Start/Stop/Status (CLI)**: `./scripts/resources/index.sh --action <install|start|stop|status> --resources <name1,name2>`
- **Configuration**: `~/.vrooli/service.json`
- **Validation**: Three-layer testing system (`./tools/validate-interfaces.sh`)
- **Compliance**: Auto-fix interface issues (`./tools/fix-interface-compliance.sh`)

> Note on `manage.sh`: Some legacy resources may include a `manage.sh` script. This is deprecated and not recommended; new resources should use the CLI entrypoints via `scripts/resources/index.sh` and resource-specific CLI functions.

## 🎯 Scenario Deployment

Resources support declarative deployment through the [injection system](../scenarios/injection/README.md), 
enabling complete application deployment from JSON specifications.

## 🛠️ Resource Structure

Each resource follows a consistent structure:
```
resource-name/
├── lib/               # Functionality libraries
├── config/            # Configuration files
├── docs/              # Resource-specific documentation
├── examples/          # Usage examples
└── cli/               # (Preferred) CLI entrypoints for actions
```

## 🔍 Common Operations

```bash
# Install and start a resource (CLI)
./scripts/resources/index.sh --action install --resources ollama
./scripts/resources/index.sh --action start --resources ollama

# View logs (resource-specific helper)
./scripts/resources/index.sh --action logs --resources n8n

# Check health across all resources
./index.sh --action discover
```

---

For detailed documentation, integration patterns, and troubleshooting, see **[docs/README.md](docs/README.md)**