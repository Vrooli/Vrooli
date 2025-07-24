# Local Resource Management System

This directory provides a comprehensive framework for setting up and managing local development resources for the Vrooli project. The system supports automatic installation, configuration, and integration of AI models, automation tools, storage services, and agent platforms.

## 🎯 Overview

The resource management system is designed to:
- **Automate setup** of local development resources
- **Integrate seamlessly** with the existing Vrooli resource discovery system
- **Scale to support** multiple resource categories and types
- **Provide consistent** installation and management patterns
- **Enable reproducible** development environments

## 📁 Directory Structure

```
scripts/resources/
├── README.md              # This documentation
├── index.sh               # Main orchestrator script
├── common.sh              # Shared utilities and functions
├── ai/                    # AI and machine learning resources
│   ├── ollama.sh          # Ollama local inference server
│   ├── localai.sh         # LocalAI alternative (planned)
│   └── llamacpp.sh        # llama.cpp server (planned)
├── automation/            # Workflow automation tools
│   ├── n8n.sh             # n8n workflow automation (planned)
│   └── nodered.sh         # Node-RED flow editor (planned)
├── storage/               # Data storage solutions
│   ├── minio.sh           # MinIO object storage (planned)
│   └── ipfs.sh            # IPFS distributed storage (planned)
└── agents/                # Browser automation agents
    ├── browserless/       # Browser automation service
    │   └── manage.sh      # Browserless Chrome-as-a-Service
    └── claude-code/       # Claude CLI tool
        └── manage.sh      # Claude Code installation manager
```

## 🚀 Quick Start

### Automatic Setup (Recommended)

The easiest way to set up resources is through the main setup script:

```bash
# Install Ollama only
./scripts/main/setup.sh --target native-linux --resources ollama

# Install all AI resources
./scripts/main/setup.sh --target native-linux --resources ai-only

# Install all resources
./scripts/main/setup.sh --target native-linux --resources all
```

### Manual Resource Management

You can also use the resource manager directly:

```bash
# List available resources
./scripts/resources/index.sh --action list

# Discover running resources
./scripts/resources/index.sh --action discover

# Install specific resources
./scripts/resources/index.sh --action install --resources ollama
./scripts/resources/index.sh --action install --resources "ollama,n8n,minio"

# Check resource status
./scripts/resources/index.sh --action status --resources ollama

# Start/stop resources
./scripts/resources/index.sh --action start --resources ollama
./scripts/resources/index.sh --action stop --resources ollama
```

## 📋 Available Resources

### AI Resources (`ai`)
| Resource | Status | Description | Default Port |
|----------|--------|-------------|--------------|
| `ollama` | ✅ Available | Local inference server for LLMs | 11434 |
| `localai` | 🚧 Planned | OpenAI-compatible local AI server | 8080 |
| `llamacpp` | 🚧 Planned | llama.cpp inference server | 8081 |

### Automation Resources (`automation`)
| Resource | Status | Description | Default Port |
|----------|--------|-------------|--------------|
| `n8n` | 🚧 Planned | Workflow automation platform | 5678 |
| `nodered` | 🚧 Planned | Visual flow-based programming | 1880 |

### Storage Resources (`storage`)
| Resource | Status | Description | Default Port |
|----------|--------|-------------|--------------|
| `minio` | 🚧 Planned | S3-compatible object storage | 9000 |
| `ipfs` | 🚧 Planned | Distributed file storage | 5001 |

### Agent Resources (`agents`)
| Resource | Status | Description | Default Port |
|----------|--------|-------------|--------------|
| `browserless` | ✅ Implemented | Browser automation service (Chrome-as-a-Service) | 4110 |
| `claude-code` | ✅ Implemented | Anthropic's official CLI for Claude | N/A (CLI) |

## 🔧 Resource Categories

You can install resources by category:

- `ai-only` - All AI and machine learning resources
- `automation-only` - All workflow automation tools
- `storage-only` - All storage solutions
- `agents-only` - All browser automation agents
- `all` - All available resources
- `none` - Skip resource installation

## 📖 Individual Resource Usage

### Ollama (AI Inference Server)

```bash
# Install with default models (llama3.1:8b, codellama:7b, llama2:7b)
./scripts/resources/ai/ollama.sh --action install

# Install without models
./scripts/resources/ai/ollama.sh --action install --skip-models

# Install with specific models
./scripts/resources/ai/ollama.sh --action install --models "llama3.1:8b,codellama:13b"

# Check status
./scripts/resources/ai/ollama.sh --action status

# List installed models
./scripts/resources/ai/ollama.sh --action models

# Start/stop service
./scripts/resources/ai/ollama.sh --action start
./scripts/resources/ai/ollama.sh --action stop

# Uninstall completely
./scripts/resources/ai/ollama.sh --action uninstall
```

## ⚙️ Configuration

### Automatic Configuration

Resources are automatically configured in `~/.vrooli/resources.local.json` when installed. This configuration is automatically discovered by the Vrooli resource system.

Example configuration:
```json
{
  \"version\": \"1.0.0\",
  \"enabled\": true,
  \"services\": {
    \"ai\": {
      \"ollama\": {
        \"enabled\": true,
        \"baseUrl\": \"http://localhost:11434\",
        \"healthCheck\": {
          \"intervalMs\": 60000,
          \"timeoutMs\": 5000
        },
        \"models\": {
          \"defaultModel\": \"llama3.1:8b\",
          \"supportsFunctionCalling\": true
        },
        \"api\": {
          \"version\": \"v1\",
          \"modelsEndpoint\": \"/api/tags\",
          \"chatEndpoint\": \"/api/chat\",
          \"generateEndpoint\": \"/api/generate\"
        }
      }
    }
  }
}
```

### Manual Configuration

You can manually edit `~/.vrooli/resources.local.json` to customize resource settings. The file will be created automatically if it doesn't exist.

## 🏗️ System Integration

### Resource Discovery

The Vrooli server automatically discovers resources configured in `~/.vrooli/resources.local.json`. Resources installed through this system are immediately available for use.

### Health Monitoring

All resources include health check endpoints and monitoring. The Vrooli resource registry will automatically monitor resource health and availability.

### Service Management

Resources are installed as systemd services (Linux) for proper lifecycle management:

```bash
# Check service status
systemctl status ollama

# View logs
journalctl -u ollama -f

# Manual service control
sudo systemctl start ollama
sudo systemctl stop ollama
sudo systemctl restart ollama
```

## 🔍 Troubleshooting

### Common Issues

**Resource script not found**
```bash
# Ensure you're in the project root
cd /path/to/vrooli

# Check if resource script exists
ls -la scripts/resources/ai/ollama.sh
```

**Port conflicts**
```bash
# Check what's using a port
sudo lsof -i :11434

# Kill conflicting process
sudo kill -9 <PID>
```

**Permission errors**
```bash
# Ensure script has execute permissions
chmod +x scripts/resources/ai/ollama.sh

# Check sudo access
sudo -v
```

**Service won't start**
```bash
# Check service status
systemctl status ollama

# View detailed logs
journalctl -u ollama -n 50

# Check port availability
./scripts/resources/index.sh --action discover
```

### Debug Mode

Enable verbose logging by setting debug environment variables:

```bash
# Enable debug logging
export LOG_LEVEL=debug

# Run resource setup with debug info
./scripts/resources/index.sh --action install --resources ollama
```

### Manual Cleanup

If automatic uninstall fails, you can manually clean up:

```bash
# Stop and disable service
sudo systemctl stop ollama
sudo systemctl disable ollama

# Remove service file
sudo rm -f /etc/systemd/system/ollama.service
sudo systemctl daemon-reload

# Remove binary
sudo rm -f /usr/local/bin/ollama

# Remove configuration
rm -f ~/.vrooli/resources.local.json

# Remove user (optional)
sudo userdel ollama
```

## 🧩 Extending the System

### Adding New Resources

To add a new resource:

1. **Create resource script** in the appropriate category directory
2. **Follow the pattern** established by existing resources (e.g., `ollama.sh`)
3. **Implement required functions**:
   - Installation and setup
   - Service management (start/stop/restart)
   - Status checking and health monitoring
   - Configuration management
   - Uninstallation
4. **Update the orchestrator** (`index.sh`) to include the new resource
5. **Test thoroughly** with all supported actions

### Resource Script Template

```bash
#!/usr/bin/env bash
set -euo pipefail

# Resource name and configuration
readonly RESOURCE_NAME=\"myresource\"
readonly RESOURCE_PORT=\"8080\"
readonly RESOURCE_BASE_URL=\"http://localhost:\${RESOURCE_PORT}\"

# Source common utilities
SCRIPT_DIR=$(cd \"$(dirname \"\${BASH_SOURCE[0]}\")\" && pwd)
RESOURCES_DIR=\"\${SCRIPT_DIR}/..\"
source \"\${RESOURCES_DIR}/common.sh\"

# Implement required functions:
# - myresource::install
# - myresource::uninstall  
# - myresource::start
# - myresource::stop
# - myresource::restart
# - myresource::status
```

## 📊 Resource Status

Check the status of all resources:

```bash
# Quick status check
./scripts/resources/index.sh --action discover

# Detailed status for specific resources
./scripts/resources/index.sh --action status --resources ollama

# Check configuration
cat ~/.vrooli/resources.local.json | jq .
```

## 🔗 Related Documentation

- [Vrooli Resource Provider System](/packages/server/src/services/resources/README.md)
- [Main Setup Script](/scripts/main/setup.sh)
- [Development Environment Setup](/scripts/main/develop.sh)
- [Project Documentation](/docs/README.md)

## 💡 Tips and Best Practices

1. **Always run setup scripts from the project root** to ensure correct path resolution
2. **Use the integrated setup approach** (`setup.sh --resources`) for new environments
3. **Check resource status** before reporting issues
4. **Keep default ports available** for automatic discovery
5. **Monitor logs** when troubleshooting service issues
6. **Use resource categories** for batch operations
7. **Test resource health** after installation

---

*This resource management system is designed to grow with the Vrooli ecosystem. As new resource types are needed, they can be easily added following the established patterns.*