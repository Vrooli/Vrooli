# Agent S2 Examples

Welcome to the Agent S2 examples! These examples are organized in a progressive learning path to help you master Agent S2's capabilities.

## 📚 Learning Path

Our examples follow a numbered progression from basics to advanced AI integration:

### 🎯 [01-getting-started](01-getting-started/)
Start here! Learn the fundamentals:
- Taking screenshots
- Checking service health
- Basic automation concepts
- Your first automation script

### 🔧 [02-basic-automation](02-basic-automation/)
Master core automation capabilities:
- Mouse control (movement, clicking, dragging)
- Keyboard automation (typing, shortcuts, special keys)
- Advanced screenshot techniques
- Combining mouse and keyboard actions

### 🚀 [03-advanced-features](03-advanced-features/)
Build robust, complex automations:
- Multi-step automation sequences
- Error handling and recovery
- Performance optimization
- State management and checkpoints

### 🧠 [04-ai-integration](04-ai-integration/)
Unlock intelligent automation with AI:
- Natural language commands
- Visual reasoning and screen understanding
- Autonomous planning and execution
- Adaptive automation that learns

## 🚀 Quick Start

### 1. Ensure Agent S2 is Running
```bash
../manage.sh --action start
../manage.sh --action status
```

### 2. Install Python Dependencies
```bash
pip install -e ../  # Install the agent_s2 package
```

### 3. Start with Hello World
```bash
cd 01-getting-started
python hello_screenshot.py
```

### 4. Progress Through Examples
Follow the numbered directories in order. Each section builds on the previous one.

## 📋 Prerequisites

### Always Required
- Agent S2 container running
- Python 3.8+ with pip
- Access to ports 4113 (API) and 5900 (VNC)

### For AI Examples (Section 04)
Choose one AI provider:

**Option A: Local Ollama (Recommended)**
```bash
# Install Ollama (if not already installed)
curl -fsSL https://ollama.ai/install.sh | sh
ollama pull llama3.2-vision:11b

# Agent S2 uses Ollama by default - no additional configuration needed!
```

**Option B: Anthropic Claude**
```bash
export ANTHROPIC_API_KEY="your_key_here"
export AGENT_S2_AI_ENABLED=true
```

**Option C: OpenAI**
```bash
export OPENAI_API_KEY="your_key_here"
export AGENT_S2_AI_ENABLED=true
```

## 🖥️ Watching Automation in Action

Connect via VNC to see the automation happening:
- **URL:** `vnc://localhost:5900`
- **Password:** `agents2vnc`

## 📁 Repository Structure

```
examples/
├── 01-getting-started/     # Begin here
│   ├── hello_screenshot.py # Simplest example
│   ├── check_health.py     # Service verification
│   └── simple_automation.py # Basic combined automation
├── 02-basic-automation/    # Core features
│   ├── mouse_control.py    # Mouse automation
│   ├── keyboard_input.py   # Keyboard automation
│   ├── screenshot.py       # Screenshot techniques
│   └── combined_automation.py # Real-world example
├── 03-advanced-features/   # Complex patterns
│   ├── automation_sequences.py # Multi-step workflows
│   ├── error_recovery.py   # Robust automation
│   └── comprehensive_demo.py # Full demonstration
├── 04-ai-integration/      # AI capabilities
│   ├── natural_language_tasks.py # NL commands
│   ├── visual_reasoning.py # Screen understanding
│   ├── autonomous_planning.py # Goal-driven automation
│   └── adaptive_automation.py # Learning automation
├── setup-demo-environment.sh # Launch demo applications
└── README.md              # This file
```

## 🏗️ Architecture Overview

Agent S2 provides two layers of automation:

```
┌─────────────────────────────────────────┐
│          AI Layer (Optional)            │ ← Section 04
│   Natural Language + Visual Reasoning   │
├─────────────────────────────────────────┤
│        Core Automation Layer            │ ← Sections 01-03
│    Mouse + Keyboard + Screenshots       │
├─────────────────────────────────────────┤
│         X11 Virtual Display             │
│      Xvfb + VNC + Window Manager       │
└─────────────────────────────────────────┘
```

**Key Insight:** AI enhances automation but isn't required. Core automation works standalone!

## 🛠️ Utility Scripts

### setup-demo-environment.sh
Launches GUI applications in the virtual display for testing:
```bash
./setup-demo-environment.sh
```
This creates windows and text to interact with during examples.

## 🔍 Testing Your Progress

After each section, verify your understanding:

### After Section 01
- Can take screenshots programmatically
- Understand Agent S2 health checking
- Perform basic mouse/keyboard actions

### After Section 02  
- Control mouse precisely (position, clicks, drags)
- Automate keyboard input and shortcuts
- Capture specific screen regions
- Combine actions into workflows

### After Section 03
- Build multi-step automation sequences
- Handle errors gracefully
- Optimize performance
- Create maintainable automation scripts

### After Section 04
- Use natural language for automation
- Let AI understand screen content
- Build adaptive automations
- Create goal-driven workflows

## 💡 Tips for Success

1. **Start Simple** - Don't skip to advanced examples
2. **Run Examples** - Reading isn't enough; execute the code
3. **Watch VNC** - See what's happening in the virtual display
4. **Check Outputs** - Screenshots are saved to `testing/test-outputs/`
5. **Experiment** - Modify examples to learn better
6. **Read Logs** - Understanding errors helps learning

## 🐛 Troubleshooting

### Agent S2 Not Running
```bash
../manage.sh --action status
../manage.sh --action start
../manage.sh --action logs
```

### Import Errors
```bash
# Install the agent_s2 package
cd ..
pip install -e .
```

### AI Not Available
```bash
# Check AI status
curl http://localhost:4113/health | jq .ai_status

# Verify API keys are set
echo $ANTHROPIC_API_KEY
echo $OPENAI_API_KEY
```

### VNC Connection Failed
- Verify port 5900 is accessible
- Check container is running
- Try alternative VNC clients

## 🤝 Contributing Examples

We welcome new examples! Guidelines:
1. Follow the numbered section structure
2. Include clear documentation
3. Add error handling
4. Test on fresh installations
5. Submit PR with description

## 📚 Additional Resources

- [Agent S2 Main Documentation](../README.md)
- [API Reference](../docs/api.md)
- [Client Library Docs](../agent_s2/client/README.md)
- [Troubleshooting Guide](../docs/troubleshooting.md)

---

**Ready to start?** Head to [01-getting-started](01-getting-started/) and begin your Agent S2 journey!