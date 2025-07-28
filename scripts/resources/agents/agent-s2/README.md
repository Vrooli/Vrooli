# Agent S2 - Autonomous Computer Interaction Service

Agent S2 is an open-source framework for autonomous computer interaction, enabling AI agents to observe, reason, and perform tasks on digital interfaces using mouse and keyboard control. This integration provides Agent S2 as a secure, Docker-based service within the Vrooli ecosystem with **dual-mode operation** for both secure sandbox and extended host system access.

## 🚀 Features

### 🧠 AI-Powered Capabilities
- **Natural Language Commands**: Execute tasks using human language ("take a screenshot and analyze what's on screen")
- **Intelligent Planning**: AI breaks down complex goals into actionable steps
- **Screen Understanding**: AI can analyze and reason about screen content
- **Autonomous Decision Making**: AI agent can adapt to different applications and contexts
- **LLM Integration**: Support for OpenAI GPT-4, Anthropic Claude, and other providers

### ⚙️ Core Automation
- **Screenshot Capture**: Full screen or region-based screenshots  
- **GUI Automation**: Mouse control, keyboard input, and complex workflows
- **Multi-Step Sequences**: Execute pre-programmed automation workflows
- **Virtual Display**: Secure X11 virtual display with VNC access
- **RESTful API**: Clean HTTP interface for both AI and core automation operations
- **Docker Security**: Sandboxed execution with non-root user

## 📋 Prerequisites

- Docker installed and running
- Port 4113 (API) and 5900 (VNC) available
- **For AI Features**: API key for your preferred LLM provider (OpenAI, Anthropic, etc.)

## 🤖 AI-Driven vs Core Automation

Agent S2 operates with two complementary layers:

### 🧠 AI Layer (Intelligence & Planning)
- **Requires**: Valid API key for OpenAI or Anthropic
- **Capabilities**: Natural language understanding, intelligent planning, screen analysis
- **Use Case**: "Open Chrome and search for cats" or "Organize my desktop files"
- **How it works**: AI interprets commands and uses core automation functions to execute them

### ⚙️ Core Automation Layer (Execution & Control)
- **Requires**: No API key needed - always available
- **Capabilities**: Direct GUI control (click, type, move mouse, screenshots)
- **Use Case**: Precise, deterministic automation with specific coordinates
- **Reliability**: Foundation layer that AI uses for all physical interactions

The AI layer intelligently orchestrates core automation functions to achieve complex goals.

### 🏗️ Architecture: How AI Uses Core Automation

```
User: "Take a screenshot and click the Chrome icon"
                    ↓
┌─────────────────────────────────────────────────┐
│              AI Layer (Intelligence)             │
├─────────────────────────────────────────────────┤
│ 1. Understand natural language command           │
│ 2. Plan sequence of actions:                    │
│    - Take screenshot to see screen              │
│    - Analyze screenshot to find Chrome icon     │
│    - Calculate Chrome icon coordinates          │
│    - Click at those coordinates                 │
└────────────────┬────────────────────────────────┘
                 ↓ Calls Core Functions
┌─────────────────────────────────────────────────┐
│         Core Automation Layer (Execution)        │
├─────────────────────────────────────────────────┤
│ • execute_screenshot() → PyAutoGUI screenshot    │
│ • analyze_image() → Process screenshot           │
│ • execute_click(x,y) → PyAutoGUI click          │
└─────────────────────────────────────────────────┘
                 ↓
            Screen Action Performed
```

**Key Point**: AI doesn't replace core automation - it orchestrates it intelligently!

## 🚀 Dual-Mode Operation

Agent S2 supports two distinct operation modes to balance security and capability:

### 🔒 Sandbox Mode (Default)
- **Security**: High isolation with restricted access
- **Environment**: Containerized desktop with pre-installed applications
- **Access**: Limited to container filesystem and approved applications
- **Use Cases**: General automation, web browsing, document editing
- **Network**: Restricted to external HTTPS connections only

### 🖥️ Host Mode (Advanced)
- **Security**: Medium isolation with controlled host access
- **Environment**: Access to host desktop and applications
- **Access**: Controlled access to host filesystem and applications
- **Use Cases**: System administration, development workflows, native app automation
- **Network**: Access to localhost and private networks (with validation)

### Mode Comparison

| Feature | Sandbox Mode | Host Mode |
|---------|--------------|-----------|
| **Security Level** | High | Medium |
| **Isolation** | Full container isolation | Controlled host access |
| **File Access** | Container only (`/home/agents2`, `/tmp`, `/opt/agent-s2`) | Host filesystem (with security constraints) |
| **Applications** | Container apps only | Host applications + container apps |
| **Network Access** | External HTTPS only | Localhost + private networks |
| **Display Access** | Virtual display (X11) | Host display (X11 forwarding) |
| **Risk Level** | Minimal | Controlled |

## 🔧 Installation

### Docker Service Installation

#### Sandbox Mode (Recommended)
```bash
# Install in sandbox mode with AI enabled (default)
./scripts/resources/agents/agent-s2/manage.sh --action install \
  --mode sandbox \
  --llm-provider anthropic \
  --llm-model claude-3-7-sonnet-20250219

# Install with OpenAI instead
./scripts/resources/agents/agent-s2/manage.sh --action install \
  --mode sandbox \
  --llm-provider openai \
  --llm-model gpt-4o

# Install with core automation only (no AI features)
./scripts/resources/agents/agent-s2/manage.sh --action install \
  --mode sandbox \
  --enable-ai no
```

#### Host Mode (Advanced)
⚠️ **Warning**: Host mode provides broader system access. Only enable on trusted systems.

```bash
# Enable host mode during installation
export AGENT_S2_HOST_MODE_ENABLED=true

# Install with host mode capabilities
./scripts/resources/agents/agent-s2/manage.sh --action install \
  --mode sandbox \
  --llm-provider anthropic \
  --llm-model claude-3-7-sonnet-20250219 \
  --host-mode-enabled yes

# Install with specific host applications allowed
./scripts/resources/agents/agent-s2/manage.sh --action install \
  --mode sandbox \
  --host-mode-enabled yes \
  --allowed-host-apps "firefox,code,gimp" \
  --host-mounts "/home/user/Documents,/home/user/Projects"
```

#### Custom Configuration
```bash
# Install with comprehensive custom settings
./scripts/resources/agents/agent-s2/manage.sh --action install \
  --mode sandbox \
  --llm-provider anthropic \
  --llm-model claude-3-7-sonnet-20250219 \
  --enable-ai yes \
  --enable-search yes \
  --host-mode-enabled yes \
  --vnc-password mysecurepassword \
  --audit-logging yes
```

### Python Client Installation
```bash
# Install the Python package for client usage
cd scripts/resources/agents/agent-s2
pip install -e .

# Or install with AI support
pip install -e ".[ai]"

# For development
pip install -e ".[dev]"
```

### Environment Variables
Set these before installation to configure AI providers:

```bash
# For Anthropic Claude (recommended)
export ANTHROPIC_API_KEY="your_anthropic_api_key_here"

# For OpenAI GPT-4
export OPENAI_API_KEY="your_openai_api_key_here"

# Optional: Disable AI if no keys available
export AGENTS2_ENABLE_AI=false

# Host mode configuration
export AGENT_S2_HOST_MODE_ENABLED=true
export AGENT_S2_HOST_AUDIT_LOGGING=true
export AGENT_S2_HOST_SECURITY_PROFILE=agent-s2-host
```

## 🔄 Mode Management

### Mode Switching

Agent S2 supports runtime mode switching between sandbox and host modes:

```bash
# Switch to host mode (if enabled)
./scripts/resources/agents/agent-s2/lib/modes.sh switch_mode host

# Switch back to sandbox mode
./scripts/resources/agents/agent-s2/lib/modes.sh switch_mode sandbox

# Check current mode
./scripts/resources/agents/agent-s2/lib/modes.sh current_mode

# Validate mode configuration
./scripts/resources/agents/agent-s2/lib/modes.sh validate_mode host
```

### API-Based Mode Management

```bash
# Get current mode information
curl http://localhost:4113/modes/current

# Switch modes via API
curl -X POST http://localhost:4113/modes/switch \
  -H "Content-Type: application/json" \
  -d '{"new_mode": "host"}'

# Get mode-specific capabilities
curl http://localhost:4113/modes/environment

# Get security constraints for current mode
curl http://localhost:4113/modes/security

# List available applications in current mode
curl http://localhost:4113/modes/applications
```

### Host Mode Prerequisites

Before using host mode, ensure security components are installed:

```bash
# Install security prerequisites
sudo ./scripts/resources/agents/agent-s2/security/install-security.sh install

# Verify AppArmor profile is loaded
sudo apparmor_status | grep docker-agent-s2-host

# Enable X11 forwarding (if not using headless mode)
./scripts/resources/agents/agent-s2/lib/modes.sh setup_x11_forwarding
```

## 🎯 Quick Start

### 1. Check Service Status
```bash
./scripts/resources/agents/agent-s2/manage.sh --action status
```

### 2. View the Virtual Display
Connect to the VNC server to watch Agent S2 in action:
- **VNC URL**: `vnc://localhost:5900`
- **Password**: `agents2vnc` (or your custom password)

### 3. Test the API

#### Basic Health Check
```bash
# Check service and AI status
curl http://localhost:4113/health

# Check capabilities
curl http://localhost:4113/capabilities
```

#### AI-Driven Commands (if AI enabled)
```bash
# Execute natural language command
curl -X POST http://localhost:4113/ai/action \
  -H "Content-Type: application/json" \
  -d '{
    "task": "take a screenshot and move mouse to center",
    "context": {"purpose": "testing AI capabilities"}
  }'

# Generate a plan
curl -X POST http://localhost:4113/ai/plan \
  -H "Content-Type: application/json" \
  -d '{
    "goal": "organize desktop workspace",
    "constraints": ["do not delete files"]
  }'

# Analyze screen content
curl -X POST http://localhost:4113/ai/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "question": "What applications are currently visible?"
  }'
```

#### Core Automation (always available)
```bash
# Take a screenshot (PNG format)
curl -X POST "http://localhost:4113/screenshot?format=png" \
  -H "Content-Type: application/json"

# Take a screenshot with region
curl -X POST "http://localhost:4113/screenshot?format=png" \
  -H "Content-Type: application/json" \
  -d '[100, 100, 500, 400]'

# Take a JPEG screenshot with quality
curl -X POST "http://localhost:4113/screenshot?format=jpeg&quality=85" \
  -H "Content-Type: application/json"

# Mouse operations
curl -X POST http://localhost:4113/mouse/click \
  -H "Content-Type: application/json" \
  -d '{"x": 500, "y": 300, "button": "left"}'

curl -X POST http://localhost:4113/mouse/move \
  -H "Content-Type: application/json" \
  -d '{"x": 600, "y": 400}'

# Keyboard operations
curl -X POST http://localhost:4113/keyboard/type \
  -H "Content-Type: application/json" \
  -d '{"text": "Hello World!", "interval": 0.1}'

curl -X POST http://localhost:4113/keyboard/press \
  -H "Content-Type: application/json" \
  -d '{"keys": ["ctrl", "c"]}'
```

### 4. Run Usage Examples
```bash
# Run all examples
./scripts/resources/agents/agent-s2/manage.sh --action usage --usage-type all

# Test specific functionality
./scripts/resources/agents/agent-s2/manage.sh --action usage --usage-type screenshot
./scripts/resources/agents/agent-s2/manage.sh --action usage --usage-type automation

# Or run examples directly with Python
cd scripts/resources/agents/agent-s2/examples

# Getting started examples
python 01-getting-started/check_health.py
python 01-getting-started/hello_screenshot.py
python 01-getting-started/simple_automation.py

# Advanced automation examples
python 02-basic-automation/screenshot.py
python 02-basic-automation/mouse_control.py
python 02-basic-automation/combined_automation.py

# AI-powered examples (requires API key)
python 04-ai-integration/natural_language_tasks.py
python 04-ai-integration/visual_reasoning.py
```

## 📝 Example Files Explained

The examples are now organized in numbered directories for progressive learning:

### 01-getting-started/
**Basic examples to get you started:**
- `check_health.py` - Service health checking and status monitoring
- `hello_screenshot.py` - Simple screenshot capture and saving
- `simple_automation.py` - Basic mouse and keyboard automation

**Key concepts:** REST API basics, health checking, simple automation tasks

### 02-basic-automation/
**Core automation features:**
- `screenshot.py` - Advanced screenshot techniques (regions, formats, quality)
- `mouse_control.py` - Mouse movements, clicks, and drag operations
- `keyboard_input.py` - Text typing, key combinations, and shortcuts
- `combined_automation.py` - Combining multiple automation actions

**Key concepts:** Direct automation control, coordinate systems, timing

### 03-advanced-features/
**Complex automation patterns:**
- `automation_sequences.py` - Multi-step automation workflows
- `error_recovery.py` - Error handling and recovery strategies
- `comprehensive_demo.py` - Full-featured demonstration combining all capabilities

**Key concepts:** Workflow orchestration, error handling, state management

### 04-ai-integration/
**AI-powered automation (requires API key):**
- `natural_language_tasks.py` - Execute tasks using natural language
- `visual_reasoning.py` - AI-based screen analysis and understanding
- `autonomous_planning.py` - Goal-oriented task planning
- `adaptive_automation.py` - Context-aware automation that adapts to UI changes

**Key concepts:** AI orchestration, natural language processing, visual understanding

## 🔌 API Reference

### Health Check
```http
GET /health
```
Returns service health status and display information.

### Get Capabilities
```http
GET /capabilities
```
Returns supported tasks and display configuration.

### Take Screenshot
```http
POST /screenshot?format={format}&quality={quality}
Body (optional): [x, y, width, height]  // Region array

Query Parameters:
- format: "png" or "jpeg" (default: "png")
- quality: 1-100 for JPEG (default: 95)
```

### Mouse Operations
```http
POST /mouse/click
{
  "x": 500,
  "y": 300,
  "button": "left|right|middle",
  "clicks": 1
}

POST /mouse/move
{
  "x": 600,
  "y": 400,
  "duration": 0.5  // Optional animation duration
}

GET /mouse/position
```

### Keyboard Operations
```http
POST /keyboard/type
{
  "text": "Hello World!",
  "interval": 0.1  // Delay between keystrokes
}

POST /keyboard/press
{
  "keys": ["ctrl", "c"]  // Key combination
}
```

### Task Management
```http
GET /tasks/{task_id}
GET /tasks
POST /tasks/cancel/{task_id}
```

## 🧠 AI API Endpoints

### Execute AI Action
```http
POST /ai/action
{
  "task": "Natural language task description",
  "screenshot": "Optional base64 screenshot data",
  "context": { /* Optional context object */ }
}
```

**Example:**
```bash
curl -X POST http://localhost:4113/ai/action \
  -H "Content-Type: application/json" \
  -d '{
    "task": "open a text editor and write hello world",
    "context": {"purpose": "demonstration"}
  }'
```

### Generate AI Plan
```http
POST /ai/plan
{
  "goal": "High-level goal to achieve",
  "constraints": ["list", "of", "constraints"]
}
```

**Example:**
```bash
curl -X POST http://localhost:4113/ai/plan \
  -H "Content-Type: application/json" \
  -d '{
    "goal": "organize my desktop files by type",
    "constraints": ["do not delete anything", "keep downloads folder visible"]
  }'
```

### Analyze Screen
```http
POST /ai/analyze
{
  "question": "What do you want to know about the screen?",
  "screenshot": "Optional base64 screenshot data"
}
```

**Example:**
```bash
curl -X POST http://localhost:4113/ai/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "question": "What applications are currently running?"
  }'
```

## 🐍 Python SDK Example

First, install the Python package:
```bash
cd scripts/resources/agents/agent-s2
pip install -e .  # or pip install -e ".[ai]" for AI support
```

Then use the client library:

```python
from agent_s2 import AgentS2Client, ScreenshotClient, AutomationClient, AIClient

# Initialize the main client
client = AgentS2Client(base_url="http://localhost:4113")

# Check service health and capabilities
if client.health_check():
    print("✅ Agent S2 is running")
    caps = client.get_capabilities()
    ai_available = caps.get("ai_available", False)
else:
    print("❌ Agent S2 is not responding")
    exit(1)

# Screenshot operations
screenshot_client = ScreenshotClient(client)

# Capture full screen
screenshot_data = screenshot_client.capture(format="png")
screenshot_client.save_to_file("desktop.png", screenshot_data)

# Capture specific region
region_screenshot = screenshot_client.capture_region(x=100, y=100, width=500, height=400)

# Capture with JPEG compression
jpeg_screenshot = screenshot_client.capture(format="jpeg", quality=85)

# Automation operations
automation = AutomationClient(client)

# Mouse control
automation.click(x=500, y=300)
automation.right_click(x=600, y=400)
automation.double_click(x=700, y=500)
automation.move_mouse(x=800, y=600, duration=0.5)
automation.drag(start_x=100, start_y=100, end_x=200, end_y=200)

# Keyboard control
automation.type_text("Hello, Agent S2!")
automation.press_key("Return")
automation.key_combination(["ctrl", "c"])

# Complex automation sequence
result = automation.execute_sequence([
    {"action": "move_mouse", "x": 100, "y": 100},
    {"action": "click"},
    {"action": "type_text", "text": "Automated task"},
    {"action": "press_key", "key": "Return"}
])

# AI-powered operations (if available)
if ai_available:
    ai = AIClient(client)
    
    # Natural language task execution
    result = ai.perform_task(
        task="Take a screenshot and move the mouse to the center of the screen",
        context={"purpose": "demonstration"}
    )
    print(f"AI Task Result: {result}")
    
    # Screen analysis
    analysis = ai.analyze_screen(
        question="What applications are currently visible on the desktop?"
    )
    print(f"Screen Analysis: {analysis}")
    
    # Goal-oriented planning
    plan = ai.create_plan(
        goal="Organize desktop files by type",
        constraints=["Don't delete any files", "Keep downloads folder visible"]
    )
    print(f"AI Plan: {plan}")
    
    # Navigate to UI element
    nav_result = ai.navigate_to("Chrome browser icon")
    print(f"Navigation Result: {nav_result}")
else:
    print("ℹ️ AI features not available - using core automation only")

# Context manager usage
with AgentS2Client() as client:
    # Client automatically handles connection lifecycle
    screenshot = client.screenshot()
    client.click(500, 300)
```

### Quick Start with the Client

```python
# Minimal example
from agent_s2 import AgentS2Client

# Create client and take a screenshot
client = AgentS2Client()
screenshot = client.screenshot()
client.save_screenshot("screen.png")

# Click and type
client.click(500, 300)
client.type_text("Hello World!")

# AI example (if enabled)
if client.get_capabilities().get("ai_available"):
    result = client.ai_action(
        task="Open notepad and type 'Hello from AI'",
        context={"demo": True}
    )
```

## 🔒 Security Considerations

Agent S2's dual-mode architecture provides different security levels for different use cases.

### Sandbox Mode Security (High Security)

#### Container Isolation
- **User**: Runs as non-root user (`agents2`) inside container
- **Filesystem**: Isolated container filesystem with read-only host mounts
- **Display**: Virtual X11 display with no host display access
- **Network**: External HTTPS only, no localhost or private network access
- **Resources**: Strict CPU and memory limits enforced

#### Restricted Access
- **Applications**: Container applications only (Firefox, LibreOffice, etc.)
- **File System**: Limited to `/home/agents2`, `/tmp`, `/opt/agent-s2`
- **Commands**: Whitelist-based command filtering
- **Network**: No access to host services or internal networks

### Host Mode Security (Medium Security)

#### Controlled Host Access
- **User**: Still runs as `agents2` user with controlled escalation
- **Display**: X11 forwarding with host display access (optional)
- **Network**: Controlled access to localhost and private networks
- **Resources**: Enhanced limits with host resource access

#### Security Monitoring
- **AppArmor**: Mandatory security profile (`docker-agent-s2-host`)
- **Audit Logging**: All actions logged to `/var/log/agent-s2/audit/`
- **Threat Detection**: Real-time monitoring for suspicious activities
- **Input Validation**: Enhanced validation for all inputs and commands

#### Host Mode Constraints
```bash
# Forbidden paths (enforced by AppArmor)
/etc/passwd, /etc/shadow, /root/*, /var/log/auth.log

# Blocked commands
sudo su -, passwd, usermod, chmod 4755, nc -l, bash -i >&

# Network restrictions
No access to: 127.0.0.1:22, sensitive internal services

# Application restrictions
Password managers, system settings still blocked
```

### Security Best Practices

#### For Sandbox Mode
```bash
# Keep container updated
./scripts/resources/agents/agent-s2/manage.sh --action update

# Monitor resource usage
docker stats agent-s2

# Review logs regularly
./scripts/resources/agents/agent-s2/manage.sh --action logs
```

#### For Host Mode
```bash
# Verify AppArmor profile is active
sudo apparmor_status | grep docker-agent-s2-host

# Monitor audit logs
sudo tail -f /var/log/agent-s2/audit/$(date +%Y-%m-%d).log

# Review security events
curl http://localhost:4113/modes/security | jq '.recent_events'

# Check for security violations
./scripts/resources/agents/agent-s2/security/check-violations.sh
```

### Security Monitoring

Agent S2 includes comprehensive security monitoring:

#### Threat Detection
- **Suspicious file access**: `/etc/passwd`, `/root/*`, private keys
- **Privilege escalation**: `sudo`, `setuid`, `chmod 4755`
- **Network anomalies**: Reverse shells, suspicious connections
- **Rapid actions**: Automated attack patterns

#### Audit Logging
```bash
# View audit summary
curl http://localhost:4113/modes/security/audit

# Export security events
curl http://localhost:4113/modes/security/events > security-report.json

# Check threat indicators
curl http://localhost:4113/modes/security | jq '.threat_indicators'
```

### VNC Security
- **Authentication**: Password-protected VNC access
- **Binding**: Local-only by default (`127.0.0.1:5900`)
- **Encryption**: Optional SSL/TLS for remote access
- **Access Control**: VNC only accessible to authorized users

### API Security
- **Authentication**: API key support for production environments
- **Rate Limiting**: Configurable request rate limits
- **Input Validation**: All inputs validated and sanitized
- **CORS**: Configurable cross-origin request policies

### Production Deployment Security

#### Recommended Configuration
```bash
# Enable all security features
export AGENT_S2_HOST_AUDIT_LOGGING=true
export AGENT_S2_HOST_SECURITY_PROFILE=agent-s2-host
export AGENT_S2_API_RATE_LIMIT=100
export AGENT_S2_VNC_SSL=true

# Install with security hardening
./scripts/resources/agents/agent-s2/manage.sh --action install \
  --mode sandbox \
  --host-mode-enabled yes \
  --audit-logging yes \
  --security-hardening yes \
  --vnc-ssl yes
```

#### Security Checklist
- [ ] AppArmor profile installed and active (host mode)
- [ ] Audit logging enabled and monitored
- [ ] VNC password changed from default
- [ ] API rate limiting configured
- [ ] Security events monitoring set up
- [ ] Regular security log reviews scheduled
- [ ] Container images kept updated

## 🛠️ Configuration

### Environment Variables
- `AGENTS2_API_KEY`: API key for LLM provider
- `OPENAI_API_KEY`: OpenAI API key (if using OpenAI)
- `ANTHROPIC_API_KEY`: Anthropic API key (if using Anthropic)

### Advanced Options
```bash
# Enable host display access (security risk!)
./scripts/resources/agents/agent-s2/manage.sh --action install \
  --enable-host-display yes

# Custom resource limits
docker update agent-s2 \
  --memory 8g \
  --cpus 4.0
```

## 🐛 Troubleshooting

### Service Won't Start
```bash
# Check logs
./scripts/resources/agents/agent-s2/manage.sh --action logs

# Check Docker status
docker ps -a | grep agent-s2

# Rebuild image
./scripts/resources/agents/agent-s2/manage.sh --action uninstall
./scripts/resources/agents/agent-s2/manage.sh --action install --force yes
```

### VNC Connection Issues
1. Ensure VNC port 5900 is not blocked
2. Check VNC password is correct
3. Try alternative VNC clients (RealVNC, TightVNC, TigerVNC)

### API Not Responding
1. Check service health: `curl http://localhost:4113/health`
2. Verify port 4113 is accessible
3. Check container logs for errors

### Display Issues
- Virtual display runs at 1920x1080 by default
- Some applications may require specific display settings
- Use VNC to verify display is working

## 📊 Performance Tuning

### Memory Usage
Agent S2 requires at least 2GB RAM, recommended 4GB:
```bash
docker update agent-s2 --memory 4g
```

### CPU Allocation
For better performance with complex tasks:
```bash
docker update agent-s2 --cpus 2.0
```

### Shared Memory
For applications requiring more shared memory:
```bash
docker update agent-s2 --shm-size 4gb
```

## 🎯 How Agent S2 Differs from Other Automation Services

### Comparison Table

| Feature | Agent S2 | n8n | Node-RED | Browserless | Huginn |
|---------|----------|-----|----------|-------------|---------|
| **Primary Focus** | GUI automation & screenshots | Workflow automation | IoT/Flow programming | Web browser automation | Web scraping & monitoring |
| **Interface Type** | Desktop GUI (any app) | Web services/APIs | Hardware/APIs | Web pages only | Web pages/APIs |
| **Programming Model** | Task-based API | Visual workflows | Flow-based nodes | Browser scripting | Event-based agents |
| **AI Integration** | Built-in LLM planning | Via HTTP nodes | Custom nodes | None built-in | None built-in |
| **Execution Model** | Direct GUI control | Webhook/scheduled | Event-driven | Page automation | Periodic/triggered |

### Key Differentiators

1. **Desktop Application Control**
   - Agent S2: Can control ANY desktop application (Excel, Photoshop, games, etc.)
   - Others: Limited to web/API interactions

2. **Visual Computer Use**
   - Agent S2: Takes screenshots, analyzes screen content, makes decisions
   - Others: Work with structured data/APIs

3. **Human-like Interaction**
   - Agent S2: Simulates mouse movements, typing, clicking like a human
   - Others: Direct API calls or DOM manipulation

### Use Cases

**Agent S2:**
- Automating traditional desktop software
- GUI testing for applications
- Visual data extraction from any app
- Game automation
- Accessibility testing

**n8n/Node-RED:**
- API integrations
- Data pipeline automation
- Business process automation
- IoT device control

**Browserless:**
- Web scraping
- PDF generation
- Website testing
- Headless browser tasks

### 🔄 How AI Commands Translate to Core Functions

When you send an AI command, here's what happens internally:

```python
# What you send:
{
    "task": "take a screenshot and move mouse to center",
    "context": {"purpose": "demonstration"}
}

# What the AI layer does:
1. Parse task → identifies "screenshot" and "move mouse"
2. Plans execution sequence
3. Calls core API endpoints:
   - POST /screenshot?format=png
   - GET /mouse/position (to get screen dimensions)
   - POST /mouse/move {"x": width//2, "y": height//2}

# What gets returned:
{
    "task": "take a screenshot and move mouse to center",
    "actions_taken": [
        {"action": "screenshot", "endpoint": "/screenshot", "status": "completed"},
        {"action": "mouse_move", "endpoint": "/mouse/move", "parameters": {"x": 960, "y": 540}, "status": "completed"}
    ],
    "reasoning": "Captured screen state, calculated center coordinates, moved mouse to center position",
    "success": true
}
```

### Architecture Differences

```
Agent S2:                          n8n/Node-RED:
┌─────────────────┐               ┌─────────────────┐
│   X11 Display   │               │   Web Server    │
├─────────────────┤               ├─────────────────┤
│  GUI Automation │               │ Workflow Engine │
├─────────────────┤               ├─────────────────┤
│   REST API      │               │   REST API      │
├─────────────────┤               ├─────────────────┤
│ Screenshot/OCR  │               │ Node Executor   │
└─────────────────┘               └─────────────────┘
```

## 🔗 Integration Examples

### With n8n Workflows
Create an n8n HTTP Request node:
```json
{
  "method": "POST",
  "url": "http://agent-s2:4113/screenshot?format=png",
  "authentication": "none",
  "jsonParameters": true
}
```

Or for AI-driven automation:
```json
{
  "method": "POST",
  "url": "http://agent-s2:4113/ai/action",
  "authentication": "none",
  "jsonParameters": true,
  "body": {
    "task": "capture screen and click on the first visible button",
    "context": {"workflow": "automated_testing"}
  }
}
```

### With ComfyUI
Use Agent S2 screenshots as input for image generation workflows.

### With Browserless
Combine Agent S2 for desktop automation with Browserless for web automation.

### Combined Workflows
1. **Agent S2 + n8n**: n8n workflow triggers Agent S2 to interact with desktop app, then processes the screenshot data
2. **Agent S2 + Browserless**: Agent S2 handles desktop apps while Browserless handles web automation in the same workflow
3. **Agent S2 + ComfyUI**: Agent S2 captures game screenshots, ComfyUI processes them for AI art generation

## 📚 Additional Resources

- [Agent S2 Official Docs](https://www.simular.ai/agent-s)
- [Agent S2 GitHub](https://github.com/simular-ai/Agent-S)
- [API Documentation](http://localhost:4113/docs) (when running)

## 🤝 Contributing

To contribute to the Agent S2 integration:

1. Follow Vrooli's contribution guidelines
2. Test changes thoroughly with Docker
3. Update documentation as needed
4. Submit pull requests with clear descriptions

## 📄 License

This integration follows Vrooli's licensing. Agent S2 itself is open source - check the [official repository](https://github.com/simular-ai/Agent-S) for its license terms.