# Agent S2 - Autonomous Computer Interaction Service

Agent S2 is an open-source framework for autonomous computer interaction, enabling AI agents to observe, reason, and perform tasks on digital interfaces using mouse and keyboard control. This integration provides Agent S2 as a secure, Docker-based service within the Vrooli ecosystem.

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

## 🔧 Installation

```bash
# Install with AI enabled (default - requires API key)
./scripts/resources/agents/agent-s2/manage.sh --action install \
  --llm-provider anthropic \
  --llm-model claude-3-7-sonnet-20250219

# Install with OpenAI instead
./scripts/resources/agents/agent-s2/manage.sh --action install \
  --llm-provider openai \
  --llm-model gpt-4o

# Install with core automation only (no AI features)
./scripts/resources/agents/agent-s2/manage.sh --action install \
  --enable-ai no

# Install with web search enabled (requires Perplexica)
./scripts/resources/agents/agent-s2/manage.sh --action install \
  --enable-search yes

# Install with custom settings
./scripts/resources/agents/agent-s2/manage.sh --action install \
  --llm-provider anthropic \
  --llm-model claude-3-7-sonnet-20250219 \
  --enable-ai yes \
  --enable-search no \
  --vnc-password mysecurepassword
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
curl -X POST http://localhost:4113/execute/ai \
  -H "Content-Type: application/json" \
  -d '{
    "command": "take a screenshot and move mouse to center",
    "context": "testing AI capabilities"
  }'

# Generate a plan
curl -X POST http://localhost:4113/plan \
  -H "Content-Type: application/json" \
  -d '{
    "goal": "organize desktop workspace",
    "constraints": ["do not delete files"]
  }'

# Analyze screen content
curl -X POST http://localhost:4113/analyze-screen \
  -H "Content-Type: application/json" \
  -d '{
    "question": "What applications are currently visible?"
  }'
```

#### Core Automation (always available)
```bash
# Take a screenshot
curl -X POST http://localhost:4113/screenshot \
  -H "Content-Type: application/json" \
  -d '{"format": "png"}'

# Execute direct automation
curl -X POST http://localhost:4113/execute \
  -H "Content-Type: application/json" \
  -d '{
    "task_type": "click",
    "parameters": {"x": 500, "y": 300}
  }'
```

### 4. Run Usage Examples
```bash
# Run all examples
./scripts/resources/agents/agent-s2/manage.sh --action usage --usage-type all

# Test specific functionality
./scripts/resources/agents/agent-s2/manage.sh --action usage --usage-type screenshot
./scripts/resources/agents/agent-s2/manage.sh --action usage --usage-type automation
```

## 📝 Example Files Explained

### screenshot-demo.sh
A simple bash script demonstrating screenshot capture:
- Checks Agent S2 health status
- Takes a full-screen screenshot via REST API
- Extracts and saves the base64-encoded image
- Shows examples of region-based and JPEG screenshots

**Key concepts demonstrated:**
- Health checking pattern
- REST API interaction
- Base64 image handling
- Error handling

### basic-automation.py
A comprehensive Python script demonstrating both AI and core automation capabilities:

**AI-Driven Examples:**
- Natural language command execution ("take a screenshot and move mouse to center")
- AI planning and goal achievement ("organize desktop workspace")
- Intelligent screen analysis ("what applications are currently visible?")
- Shows how AI uses core automation functions to execute commands

**Core Automation Examples:**
- Health monitoring and status reporting
- Direct actions (mouse move, click, type)
- Complex multi-step automation sequences
- Precise coordinate-based control

**Key concepts demonstrated:**
- AI Layer → Core Automation Layer architecture
- How AI commands translate to core function calls
- Graceful fallback when AI unavailable
- Python SDK pattern for both AI and direct automation
- Interactive demonstration with mode selection
- Comprehensive error handling and status reporting

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
POST /screenshot
{
  "format": "png",      // png or jpeg
  "quality": 95,        // JPEG quality (1-100)
  "region": [x, y, width, height]  // Optional region
}
```

### Execute Task
```http
POST /execute
{
  "task_type": "click|type_text|key_press|mouse_move|automation_sequence",
  "parameters": { /* task-specific parameters */ },
  "async_execution": false
}
```

### Get Task Status
```http
GET /tasks/{task_id}
```

### Get Mouse Position
```http
GET /mouse/position
```

## 🧠 AI API Endpoints

### Execute AI Command
```http
POST /execute/ai
{
  "command": "Natural language instruction",
  "context": "Optional context for the command",
  "async_execution": false
}
```

**Example:**
```bash
curl -X POST http://localhost:4113/execute/ai \
  -H "Content-Type: application/json" \
  -d '{
    "command": "open a text editor and write hello world",
    "context": "demonstration"
  }'
```

### Generate AI Plan
```http
POST /plan
{
  "goal": "High-level goal to achieve",
  "constraints": ["list", "of", "constraints"]
}
```

**Example:**
```bash
curl -X POST http://localhost:4113/plan \
  -H "Content-Type: application/json" \
  -d '{
    "goal": "organize my desktop files by type",
    "constraints": ["do not delete anything", "keep downloads folder visible"]
  }'
```

### Analyze Screen
```http
POST /analyze-screen
{
  "question": "What do you want to know about the screen?"
}
```

**Example:**
```bash
curl -X POST http://localhost:4113/analyze-screen \
  -H "Content-Type: application/json" \
  -d '{
    "question": "What applications are currently running?"
  }'
```

## 🐍 Python SDK Example

```python
import requests
import base64
from PIL import Image
import io

class AgentS2Client:
    def __init__(self, base_url="http://localhost:4113"):
        self.base_url = base_url
    
    def screenshot(self, region=None):
        """Take a screenshot"""
        data = {"format": "png"}
        if region:
            data["region"] = region
        
        response = requests.post(f"{self.base_url}/screenshot", json=data)
        if response.ok:
            # Extract base64 image
            img_data = response.json()["data"].split(",")[1]
            img_bytes = base64.b64decode(img_data)
            return Image.open(io.BytesIO(img_bytes))
        raise Exception(f"Screenshot failed: {response.text}")
    
    def click(self, x, y, button="left"):
        """Click at position"""
        return self.execute("click", {"x": x, "y": y, "button": button})
    
    def type_text(self, text, interval=0.1):
        """Type text"""
        return self.execute("type_text", {"text": text, "interval": interval})
    
    def execute(self, task_type, parameters):
        """Execute a core automation task"""
        data = {
            "task_type": task_type,
            "parameters": parameters
        }
        response = requests.post(f"{self.base_url}/execute", json=data)
        response.raise_for_status()
        return response.json()
    
    # AI Methods
    def ai_command(self, command, context=None, async_execution=False):
        """Execute natural language command using AI"""
        data = {
            "command": command,
            "context": context,
            "async_execution": async_execution
        }
        response = requests.post(f"{self.base_url}/execute/ai", json=data)
        response.raise_for_status()
        return response.json()
    
    def ai_plan(self, goal, constraints=None):
        """Generate AI plan for achieving a goal"""
        data = {
            "goal": goal,
            "constraints": constraints or []
        }
        response = requests.post(f"{self.base_url}/plan", json=data)
        response.raise_for_status()
        return response.json()
    
    def ai_analyze_screen(self, question=None):
        """Analyze screen content using AI"""
        data = {"question": question} if question else {}
        response = requests.post(f"{self.base_url}/analyze-screen", json=data)
        response.raise_for_status()
        return response.json()
    
    def get_capabilities(self):
        """Get service capabilities"""
        response = requests.get(f"{self.base_url}/capabilities")
        response.raise_for_status()
        return response.json()

# Usage Examples
agent = AgentS2Client()

# Check what's available
caps = agent.get_capabilities()
ai_available = caps.get("ai_status", {}).get("initialized", False)

if ai_available:
    print("🧠 AI Mode Available - Using intelligent automation")
    
    # AI-driven examples
    result = agent.ai_command("take a screenshot and move mouse to center")
    print(f"AI Command Result: {result}")
    
    plan = agent.ai_plan("organize desktop files", ["don't delete anything"])
    print(f"AI Plan: {plan}")
    
    analysis = agent.ai_analyze_screen("What applications are visible?")
    print(f"Screen Analysis: {analysis}")
    
else:
    print("⚙️ Core Automation Mode - Direct control available")

# Core automation functions (always available)
# Take screenshot
img = agent.screenshot()
img.save("desktop.png")

# Click at position
agent.click(500, 300)

# Type text
agent.type_text("Hello from Agent S2!")

# Complex automation sequence
agent.execute("automation_sequence", {
    "steps": [
        {"type": "mouse_move", "parameters": {"x": 100, "y": 100}},
        {"type": "click", "parameters": {}},
        {"type": "type_text", "parameters": {"text": "Automated task"}},
        {"type": "key_press", "parameters": {"keys": ["Return"]}}
    ]
})
```

## 🔒 Security Considerations

### Container Security
- Runs as non-root user (`agents2`)
- Isolated virtual display (no host display access by default)
- Sandboxed execution environment
- Resource limits enforced (memory, CPU)

### VNC Security
- Password-protected VNC access
- Local-only binding by default
- Optional SSL/TLS for remote access

### API Security
- API key authentication support
- Rate limiting recommended for production
- Input validation on all endpoints

### Restricted Applications
The following applications are blocked by default:
- Password managers (passwords, keychain, 1password, bitwarden)
- System settings
- Terminal/shell access

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
    "command": "take a screenshot and move mouse to center"
}

# What the AI layer does:
1. Parse command → identifies "screenshot" and "move mouse"
2. Plans execution sequence
3. Calls core functions:
   - await execute_screenshot({})
   - screen_width, screen_height = pyautogui.size()
   - await execute_mouse_move({"x": width//2, "y": height//2})

# What gets returned:
{
    "command": "take a screenshot and move mouse to center",
    "actions_taken": [
        {"action": "screenshot", "status": "completed"},
        {"action": "mouse_move", "parameters": {"x": 960, "y": 540}, "status": "completed"}
    ],
    "core_functions_used": 2
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
  "url": "http://agent-s2:4113/execute",
  "authentication": "none",
  "jsonParameters": true,
  "body": {
    "task_type": "screenshot",
    "parameters": {}
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