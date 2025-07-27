# Agent S2 - Autonomous Computer Interaction Service

Agent S2 is an open-source framework for autonomous computer interaction, enabling AI agents to observe, reason, and perform tasks on digital interfaces using mouse and keyboard control. This integration provides Agent S2 as a secure, Docker-based service within the Vrooli ecosystem.

## 🚀 Features

- **Screenshot Capture**: Full screen or region-based screenshots
- **GUI Automation**: Mouse control, keyboard input, and complex workflows
- **Multi-Step Planning**: Break down complex tasks into executable steps
- **Virtual Display**: Secure X11 virtual display with VNC access
- **RESTful API**: Clean HTTP interface for all operations
- **LLM Integration**: Support for OpenAI, Anthropic, and other providers
- **Docker Security**: Sandboxed execution with non-root user

## 📋 Prerequisites

- Docker installed and running
- Port 4113 (API) and 5900 (VNC) available
- API key for your preferred LLM provider (OpenAI, Anthropic, etc.)

## 🔧 Installation

```bash
# Install Agent S2 with default settings
./scripts/resources/agents/agent-s2/manage.sh --action install

# Install with specific LLM provider
./scripts/resources/agents/agent-s2/manage.sh --action install \
  --llm-provider anthropic \
  --llm-model claude-3-opus-20240229

# Install with custom VNC password
./scripts/resources/agents/agent-s2/manage.sh --action install \
  --vnc-password mysecurepassword
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
```bash
# Health check
curl http://localhost:4113/health

# Take a screenshot
curl -X POST http://localhost:4113/screenshot \
  -H "Content-Type: application/json" \
  -d '{"format": "png"}'

# Execute automation
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
A comprehensive Python script showing all major capabilities:
- Health monitoring and status reporting
- Individual actions (mouse move, click, type)
- Complex multi-step automation sequences
- Proper error handling

**Key concepts demonstrated:**
- Python SDK pattern for Agent S2
- Sequential automation steps
- Async task execution
- Integration best practices

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
        """Execute a task"""
        data = {
            "task_type": task_type,
            "parameters": parameters
        }
        response = requests.post(f"{self.base_url}/execute", json=data)
        response.raise_for_status()
        return response.json()

# Usage
agent = AgentS2Client()

# Take screenshot
img = agent.screenshot()
img.save("desktop.png")

# Click at position
agent.click(500, 300)

# Type text
agent.type_text("Hello from Agent S2!")

# Complex automation
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
- Automating legacy desktop software
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