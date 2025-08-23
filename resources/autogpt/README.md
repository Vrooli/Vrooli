# AutoGPT Resource

AutoGPT is an autonomous AI agent framework that enables self-directed task execution and goal-oriented workflows.

## Overview

AutoGPT provides:
- **Autonomous Task Execution**: Agents that break down complex goals into actionable tasks
- **Memory Management**: Long-term and short-term memory for context retention
- **Tool Integration**: Connect to various APIs and services
- **Self-Improvement**: Agents can refine their own prompts and strategies

## Usage

### Basic Commands

```bash
# Install AutoGPT
resource-autogpt install

# Start the service
resource-autogpt start

# Check status
resource-autogpt status

# Create an agent
resource-autogpt create-agent "research-assistant" \
  --goal "Research market trends in renewable energy" \
  --model "gpt-4"

# List agents
resource-autogpt list-agents

# Run an agent
resource-autogpt run-agent "research-assistant"

# Stop the service
resource-autogpt stop
```

### Injecting Agent Configurations

You can inject pre-configured agents:

```bash
# Inject an agent configuration
resource-autogpt inject agents/market-researcher.yaml

# Inject a custom tool
resource-autogpt inject tools/web-scraper.py
```

## Integration with Vrooli

AutoGPT integrates seamlessly with other Vrooli resources:

1. **OpenRouter/Ollama**: Uses available LLMs for agent reasoning
2. **PostgreSQL**: Stores agent memory and task history
3. **Redis**: Manages task queues and agent state
4. **Browserless**: Enables web interaction for agents
5. **n8n/Huginn**: Triggers agents from workflows

## Configuration

AutoGPT uses environment variables for configuration:

- `AUTOGPT_AI_PROVIDER`: LLM provider (openrouter, ollama, openai)
- `AUTOGPT_MODEL`: Default model to use
- `AUTOGPT_MAX_ITERATIONS`: Maximum steps per task (default: 25)
- `AUTOGPT_MEMORY_BACKEND`: Memory storage (postgres, redis, local)

## Architecture

```
AutoGPT Resource
├── Agent Manager (REST API)
│   ├── Agent Creation & Management
│   ├── Task Queue Processing
│   └── Result Storage
├── Memory Systems
│   ├── Vector Memory (Qdrant)
│   ├── Conversation Memory (Redis)
│   └── Long-term Memory (PostgreSQL)
└── Tool Ecosystem
    ├── Web Browser (via Browserless)
    ├── Code Execution (via Judge0)
    └── Custom Tools (Python plugins)
```

## Examples

See the `examples/` directory for sample configurations:
- `research-agent.yaml`: Market research automation
- `code-reviewer.yaml`: Automated code review agent
- `content-creator.yaml`: Blog post and documentation writer

## Testing

Run the test suite:

```bash
resource-autogpt test
```

## Scenarios Enabled

AutoGPT enables powerful autonomous scenarios:
- **Research Automation**: Agents that continuously gather and synthesize information
- **Code Generation**: Self-improving coding assistants
- **Business Process Automation**: Agents that handle multi-step business workflows
- **Creative Content**: Autonomous content creation and curation
- **Data Analysis**: Self-directed data exploration and insight generation