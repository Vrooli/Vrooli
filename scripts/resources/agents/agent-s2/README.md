# Agent S2 - Autonomous Computer Interaction Service

Agent S2 is an open-source framework for autonomous computer interaction, enabling AI agents to observe, reason, and perform tasks on digital interfaces using mouse and keyboard control. This integration provides Agent S2 as a secure, Docker-based service within the Vrooli ecosystem.

## 🎯 Quick Reference

- **Category**: Agents
- **Port**: 4113 (API), 5900 (VNC)
- **Container**: agent-s2
- **API Docs**: [Complete API Reference](docs/API.md)
- **Status**: Production Ready

## 🚀 Quick Start

### Prerequisites
- Docker installed and running
- 2GB+ RAM available
- Ports 4113 and 5900 available
- **For AI Features**: API key for OpenAI or Anthropic

### Installation
```bash
# Basic installation with AI enabled
./manage.sh --action install \
  --mode sandbox \
  --llm-provider anthropic \
  --llm-model claude-3-7-sonnet-20250219

# Core automation only (no AI features)
./manage.sh --action install \
  --mode sandbox \
  --enable-ai no

# Advanced installation with host mode
./manage.sh --action install \
  --mode sandbox \
  --host-mode-enabled yes \
  --audit-logging yes
```

### Basic Usage
```bash
# Check service status
./manage.sh --action status

# Test screenshot functionality
./manage.sh --action usage --usage-type screenshot

# Test automation capabilities
./manage.sh --action usage --usage-type automation

# Test AI planning (requires API key)
./manage.sh --action usage --usage-type planning

# View all available usage examples
./manage.sh --action usage --usage-type help
```

### Verify Installation
```bash
# Check service health and capabilities
./manage.sh --action status

# Test all functionality
./manage.sh --action usage --usage-type all

# Connect to VNC to see desktop
# VNC URL: vnc://localhost:5900
# Password: agents2vnc (or your custom password)
```

## 🔧 Core Features

- **🧠 AI-Powered Automation**: Execute tasks using natural language commands
- **📸 Screenshot Capture**: Full screen or region-based screenshots with multiple formats
- **🖱️ Mouse Control**: Click, move, drag operations with pixel precision
- **⌨️ Keyboard Input**: Text typing and key combinations
- **🖥️ Virtual Display**: Secure X11 virtual display with VNC access
- **🔒 Dual-Mode Operation**: Sandbox (high security) and Host (controlled access) modes
- **🐍 Python SDK**: Full-featured client library for programmatic access
- **🔄 Task Management**: Asynchronous task execution with status tracking

## 📖 Documentation

- **[API Reference](docs/API.md)** - Complete endpoint documentation and examples
- **[Configuration Guide](docs/CONFIGURATION.md)** - Environment variables, installation options, and settings
- **[Security Guide](docs/SECURITY.md)** - Security considerations for both sandbox and host modes
- **[Advanced Usage](docs/ADVANCED.md)** - Architecture details, integrations, and complex scenarios
- **[Troubleshooting](docs/TROUBLESHOOTING.md)** - Common issues and diagnostic procedures
- **[Examples](examples/)** - Progressive tutorials from basic to advanced usage

## 🎯 When to Use Agent S2

### Use Agent S2 When:
- Automating desktop applications (Excel, Photoshop, games, etc.)
- Performing GUI testing for applications
- Extracting visual data from any application
- Combining AI intelligence with precise automation
- Requiring human-like interaction patterns

### Consider Alternatives When:
- Working only with web pages → [Browserless](../browserless/)
- Building API-based workflows → [n8n](../../automation/n8n/)
- Performing simple web scraping → [SearXNG](../../search/searxng/)

## 🔗 Integration Examples

### With n8n Workflows
```json
{
  "method": "POST",
  "url": "http://agent-s2:4113/ai/action",
  "body": {
    "task": "capture desktop and click on visible Chrome icon",
    "context": {"workflow": "browser_automation"}
  }
}
```

### With Python SDK
```python
from agent_s2 import AgentS2Client

# Initialize client
client = AgentS2Client(base_url="http://localhost:4113")

# AI-powered automation
if client.get_capabilities().get("ai_available"):
    result = client.ai_action(
        task="Open notepad and type 'Hello World!'",
        context={"demo": True}
    )

# Core automation
client.click(500, 300)
client.type_text("Automated task complete")
screenshot = client.screenshot()
```

### Multi-Resource Workflow
See [Advanced Integration Patterns](docs/ADVANCED.md#advanced-integration-patterns) for complex workflows combining Agent S2 with other Vrooli resources.

## ⚡ Key Architecture

Agent S2 operates with a unique **two-layer architecture**:

- **AI Layer**: Interprets natural language, plans actions, analyzes screen content
- **Core Automation Layer**: Executes precise mouse/keyboard operations, captures screenshots

The AI layer intelligently orchestrates core automation functions to achieve complex goals, while core automation remains available independently for deterministic tasks.

**Dual Modes**:
- **Sandbox Mode** (default): High security, container isolation, external HTTPS only
- **Host Mode** (advanced): Controlled host access, localhost connectivity, enhanced capabilities

## 🆘 Getting Help

- Check [Troubleshooting Guide](docs/TROUBLESHOOTING.md) for common issues
- Run `./manage.sh --action status` for detailed diagnostics
- View logs: `./manage.sh --action logs`
- Test functionality: `./manage.sh --action usage --usage-type all`

## 📦 What's Included

```
agent-s2/
├── manage.sh                 # Management script
├── README.md                 # This overview
├── docs/                     # Detailed documentation
│   ├── API.md               # Complete API reference
│   ├── CONFIGURATION.md     # Setup and configuration
│   ├── SECURITY.md          # Security considerations
│   ├── ADVANCED.md          # Architecture and integrations
│   └── TROUBLESHOOTING.md   # Issue resolution
├── examples/                 # Progressive tutorials
│   ├── 01-getting-started/  # Basic examples
│   ├── 02-basic-automation/ # Core automation
│   ├── 03-advanced-features/# Complex workflows
│   └── 04-ai-integration/   # AI-powered examples
├── lib/                     # Helper scripts and functions
├── config/                  # Configuration files
├── security/               # Security profiles and scripts
└── agent_s2/              # Python SDK source
```

---

**🤖 Agent S2 bridges the gap between AI intelligence and precise desktop automation, making any application programmable through natural language commands while maintaining the reliability of traditional automation tools.**