# Agent S2 - Natural Language Computer Control

Agent S2 enables you to control your computer using natural language commands. Simply describe what you want to do - "go to Reddit", "search for tutorials", "open calculator" - and Agent S2 will execute the task using AI-powered visual understanding and automation. It combines the intelligence of LLMs with precise mouse/keyboard control to interact with any application or website.

> **⚠️ Important Security Update**: The transparent proxy feature is now **DISABLED by default** to prevent unintended system-wide traffic interception. See [Transparent Proxy Documentation](docs/TRANSPARENT_PROXY.md) for details.

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
- **For AI Features**: Local Ollama instance (default) or API key for OpenAI/Anthropic

### Installation
```bash
# Basic installation with AI enabled (uses Ollama by default)
./manage.sh --action install \
  --mode sandbox

# Alternative: with specific Anthropic provider
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
# Execute AI tasks with natural language (primary feature)
./manage.sh --action ai-task --task "go to reddit"
./manage.sh --action ai-task --task "search for Python tutorials on Google"
./manage.sh --action ai-task --task "open calculator and compute 25 * 4"
./manage.sh --action ai-task --task "take a screenshot of the desktop"

# Check service status
./manage.sh --action status

# Test specific capabilities
./manage.sh --action usage --usage-type screenshot    # Screenshot functionality
./manage.sh --action usage --usage-type automation    # Core automation
./manage.sh --action usage --usage-type planning      # AI planning
./manage.sh --action usage --usage-type help          # All examples
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
- **🛡️ Stealth Mode**: Advanced anti-bot detection with fingerprint randomization and session persistence

## 🤖 AI Task Execution

Agent S2 can understand and execute natural language commands. Simply describe what you want to do:

```bash
# Navigate to websites
./manage.sh --action ai-task --task "go to reddit and browse the front page"

# Search for information
./manage.sh --action ai-task --task "search Google for machine learning tutorials"

# Interact with applications
./manage.sh --action ai-task --task "open calculator and compute 150 divided by 3"

# Complex multi-step tasks
./manage.sh --action ai-task --task "take a screenshot, open text editor, and write a summary"
```

The AI understands context and can handle complex instructions that would normally require multiple manual steps.

## 📖 Documentation

- **[API Reference](docs/API.md)** - Complete endpoint documentation and examples
- **[Configuration Guide](docs/CONFIGURATION.md)** - Environment variables, installation options, and settings
- **[Security Guide](docs/SECURITY.md)** - Security considerations for both sandbox and host modes
- **[Stealth Mode Guide](docs/STEALTH_MODE.md)** - Anti-bot detection, fingerprint randomization, and session persistence
- **[Advanced Usage](docs/ADVANCED.md)** - Architecture details, integrations, and complex scenarios
- **[Troubleshooting](docs/TROUBLESHOOTING.md)** - Common issues and diagnostic procedures
- **[Examples](examples/)** - Progressive tutorials from basic to advanced usage

## 🎯 When to Use Agent S2

### Use Agent S2 When:
- **Natural language automation**: "Go to website X and do Y" - describe tasks in plain English
- **Web navigation with AI reasoning**: Navigate complex sites, handle popups, solve CAPTCHAs
- **Desktop application automation**: Control Excel, Photoshop, games, or any GUI application
- **Visual understanding required**: "Click the blue button", "Find the search box", etc.
- **Multi-step workflows**: Complex tasks requiring decision-making and adaptation
- **GUI testing**: Automated testing that requires visual verification

### Consider Alternatives When:
- Working with simple, predictable web pages → [Browserless](../browserless/)
- Building pure API-based workflows → [n8n](../../automation/n8n/)
- Performing basic web scraping → [SearXNG](../../search/searxng/)
- You know exact selectors/coordinates → Use core automation endpoints directly

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

## 🧪 Testing & Examples

### Individual Resource Tests
- **Test Location**: `scripts/__test/resources/single/agents/agent-s2.test.sh`
- **Test Coverage**: Service health, AI task execution, screenshot functionality, automation capabilities
- **Run Test**: `cd scripts/__test/resources && ./quick-test.sh agent-s2`

### Working Examples
- **Examples Folder**: [examples/](examples/)
- **Progressive Tutorials**: 
  - `01-getting-started/` - Basic usage and setup
  - `02-basic-automation/` - Core automation features
  - `03-advanced-features/` - Complex workflows and integrations
  - `04-ai-integration/` - AI-powered natural language automation
- **Integration Examples**: Multi-resource workflows combining Agent-S2 with Ollama, Whisper, and ComfyUI

### Integration with Scenarios
Agent-S2 is used in these business scenarios:
- **[Research Assistant](../../scenarios/core/research-assistant/)** - AI-powered research with automated data collection ($15k-30k projects)
- **[Analytics Dashboard](../../scenarios/core/analytics-dashboard/)** - Automated dashboard creation and data visualization ($12k-25k projects)

### Test Fixtures
- **Shared Test Data**: `scripts/__test/resources/fixtures/images/` (test screenshots and visual validation)
- **Integration Data**: `scripts/__test/resources/fixtures/workflows/` (automation scenarios)

### Quick Test Commands
```bash
# Test individual Agent-S2 functionality
./scripts/__test/resources/quick-test.sh agent-s2

# Test in business scenarios
cd ./scenarios/research-assistant && ./test.sh

# Run all tests using Agent-S2
./scripts/scenarios/tools/test-by-resource.sh --resource agent-s2
```

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