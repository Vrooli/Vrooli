# AI-Driven Examples

This directory contains examples demonstrating Agent S2's AI-powered capabilities using local Ollama models. These examples showcase true agentic behavior where AI interprets goals, plans actions, and executes them using core automation functions.

## 🧠 Overview

AI-driven automation provides intelligent, context-aware control:
- Natural language command interpretation
- Visual scene understanding and reasoning
- Autonomous goal planning and execution
- Adaptive behavior based on screen content

The AI layer orchestrates core automation functions to achieve complex objectives.

## 🛠️ Prerequisites

### Local Ollama Setup (Recommended)

These examples work best with local Ollama models, eliminating the need for external API keys:

1. **Install Ollama:**
```bash
curl -fsSL https://ollama.ai/install.sh | sh
```

2. **Pull recommended models:**
```bash
# For general tasks (good balance of speed/capability)
ollama pull llama2:7b

# For coding tasks
ollama pull codellama:7b

# For advanced reasoning (requires more RAM)
ollama pull llama2:13b
ollama pull mistral:7b
```

3. **Start Agent S2 with Ollama:**
```bash
../../manage.sh --action install \
  --llm-provider ollama \
  --llm-model llama2:7b \
  --enable-ai yes
```

### Alternative: External API Keys

You can also use cloud providers:
```bash
# For Anthropic Claude
export ANTHROPIC_API_KEY="your_key_here"
../../manage.sh --action install --llm-provider anthropic

# For OpenAI GPT-4
export OPENAI_API_KEY="your_key_here"  
../../manage.sh --action install --llm-provider openai
```

## 📁 Examples

### 1. **natural_language_tasks.py** - Command Interpretation
Demonstrates AI understanding natural language commands:
- Basic commands ("take a screenshot")
- Multi-step tasks ("create a todo list document")
- Contextual understanding
- Adaptive behavior
- Creative task execution

```bash
python natural_language_tasks.py
```

**Key Features:**
- Interprets vague commands and fills in details
- Adapts to current screen content
- Interactive conversation mode
- Demonstrates AI → core automation flow

### 2. **visual_reasoning.py** - Screen Understanding
Shows AI's visual reasoning capabilities:
- Screen content analysis
- Visual element identification
- Spatial reasoning
- Pattern recognition
- Dynamic scene adaptation

```bash
python visual_reasoning.py
```

**Key Features:**
- Analyzes screenshots with AI vision
- Identifies clickable elements
- Understands spatial relationships
- Detects screen changes
- Visual search capabilities

### 3. **autonomous_planning.py** - Goal-Driven Planning
Demonstrates autonomous task planning and execution:
- Complex goal decomposition
- Multi-step plan creation
- Constraint-aware planning
- Error recovery strategies
- Adaptive replanning

```bash
python autonomous_planning.py
```

**Key Features:**
- Breaks down high-level goals into steps
- Creates executable action sequences
- Handles constraints and limitations
- Recovers from errors gracefully
- Learning from previous attempts

## 🚀 Getting Started

1. **Ensure AI is enabled:**
```bash
../../manage.sh --action status
# Look for "AI Initialized: ✅"
```

2. **Connect to VNC to watch AI in action:**
- URL: `vnc://localhost:5900`
- Password: `agents2vnc`

3. **Run AI examples:**
```bash
cd ai/
python [example_name].py
```

## 🔧 AI API Endpoints Used

These examples demonstrate AI-specific endpoints:

- `POST /execute/ai` - Natural language command execution
- `POST /plan` - Goal-based planning
- `POST /analyze-screen` - Visual scene analysis
- `GET /capabilities` - AI capability detection

## 💡 Key Concepts

### Natural Language Processing
AI converts human language into executable actions:
```python
# User says: "take a screenshot and click the center"
# AI breaks this down into:
# 1. execute_screenshot()
# 2. calculate center coordinates  
# 3. execute_click(x=960, y=540)
```

### Visual Understanding
AI analyzes screen content to make informed decisions:
```python
# AI can identify:
# - Text elements and their content
# - UI components (buttons, menus, icons)
# - Spatial relationships
# - Visual patterns and changes
```

### Autonomous Planning
AI creates multi-step plans to achieve goals:
```python
# Goal: "Organize desktop files"
# AI Plan:
# 1. Analyze current desktop layout
# 2. Identify file types and locations
# 3. Create logical groupings
# 4. Move files to organized positions
# 5. Verify the new arrangement
```

## 🔄 AI → Core Architecture

The AI examples demonstrate this flow:

```
Natural Language → AI Understanding → Action Planning → Core Execution
     ↓                    ↓                    ↓              ↓
"Click the button"   Identify button    Plan click action   execute_click()
```

## 🧪 Example Scenarios

### Learning Scenario
```python
# Start simple, build complexity
python natural_language_tasks.py  # Learn basic commands
python visual_reasoning.py        # Add visual understanding  
python autonomous_planning.py     # Combine into autonomous goals
```

### Testing Scenario
```python
# Test AI capabilities
1. Run health check: "Take a screenshot"
2. Test reasoning: "Click on the largest empty area"  
3. Test planning: "Create a simple document"
4. Test adaptation: Change screen, see AI respond
```

## 🛠️ Troubleshooting

### AI Not Available
```bash
# Check Ollama is running
curl http://localhost:11434/api/version

# Check Agent S2 AI status
curl http://localhost:4113/health | jq .ai_status
```

### Model Performance Issues
```bash
# Use smaller models for faster response
ollama pull llama2:7b     # Faster, less capable
ollama pull llama2:13b    # Slower, more capable

# Restart Agent S2 with new model
../../manage.sh --action restart
```

### AI Command Failures
- Check that AI is properly initialized
- Verify the command is clear and specific
- Try breaking complex commands into simpler steps
- Watch VNC display to see what AI is interpreting

## 📊 Performance Tips

1. **Model Selection:**
   - `llama2:7b` - Fast, good for simple tasks
   - `mistral:7b` - Good reasoning, efficient
   - `codellama:7b` - Best for technical tasks
   - `llama2:13b` - Most capable, requires 8GB+ RAM

2. **Command Optimization:**
   - Be specific about desired outcomes
   - Provide context when helpful
   - Break complex goals into steps
   - Use visual references ("click the blue button")

3. **Resource Management:**
   - Monitor memory usage with larger models
   - Close unused applications during AI tasks
   - Use core automation for high-precision tasks

## 🔬 Advanced Features

### Context Persistence
AI can remember information across commands within a session:
```python
# Command 1: "Take a screenshot and remember the layout"
# Command 2: "Click on something that wasn't there before" 
# AI uses memory from Command 1 to execute Command 2
```

### Adaptive Behavior
AI adjusts strategies based on success/failure:
```python
# If clicking fails, AI might try:
# 1. Different coordinates
# 2. Alternative interaction method
# 3. Entirely different approach
```

### Creative Problem Solving
AI can interpret ambiguous requests creatively:
```python
# "Make the desktop look nice" could result in:
# - Arranging icons in patterns
# - Organizing by type or color
# - Creating visual symmetry
```

## 📝 Notes

- AI responses may vary between runs due to model behavior
- Some tasks may take longer as AI analyzes and plans
- Visual understanding quality depends on screen content clarity
- Models improve with use as they adapt to patterns

## 🤝 Integration Opportunities

These AI examples can be combined with other Vrooli services:
- **n8n workflows** - Trigger AI automation from business processes
- **Browserless** - Combine web automation with desktop AI control  
- **ComfyUI** - Process AI-captured screenshots for image generation
- **Node-RED** - IoT triggers for desktop automation