# Vrooli Resource Management System

Local AI and automation tools that extend Vrooli's capabilities through modular services.

## 🚀 Quick Start

```bash
# Discover what's running
vrooli resource status

# Install specific resources via CLI
vrooli resource install ollama
vrooli resource install agent-s2

# Check resource status
vrooli resource status ollama

# Use resource-specific CLIs (when available)
resource-ollama status
resource-ollama list-models
```

## 📦 Available Resources

| Category | Resources | Purpose |
|----------|-----------|---------|
| **AI** | ollama, whisper, unstructured-io, comfyui | Local AI inference and processing |
| **Automation** | node-red, huginn, comfyui | Workflow orchestration |
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

- **Discovery**: `vrooli resource status`
- **Installation/Start/Stop/Status (CLI)**: `vrooli resource <install|start|stop|status> <name>`
- **Resource-specific CLIs**: `resource-<name> <command>` (e.g., `resource-ollama list-models`)
- **Configuration**: `~/.vrooli/service.json`
- **Validation**: Three-layer testing system (`./tools/validate-interfaces.sh`)
- **Compliance**: Auto-fix interface issues (`./tools/fix-interface-compliance.sh`)

> **Note on CLI Installation**: Resource-specific CLIs (like `resource-ollama`) are automatically installed to `~/.local/bin/` when resources are set up. These provide direct access to resource functionality alongside the main `vrooli resource` commands.

> **Note on Legacy Scripts**: Some legacy resources may include `manage.sh` or top-level `install.sh` scripts. These are deprecated in favor of the unified CLI system. Use `vrooli resource` commands for all resource management operations.

## 🎯 Scenario Deployment

Resources support declarative deployment through the [injection system](../scenarios/injection/README.md), 
enabling complete application deployment from JSON specifications.

## 🛠️ Resource Structure

Each resource follows a consistent structure:
```
resource-name/
├── lib/               # Functionality 
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
./scripts/resources/index.sh --action logs --resources <resource-name>

# Check health across all resources
./index.sh --action discover
```

---

For detailed documentation, integration patterns, and troubleshooting, see **[docs/README.md](docs/README.md)**
