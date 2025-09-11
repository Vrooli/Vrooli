# Parlant LLM Agent Framework

Production-ready Python SDK for controlled multi-agent development with behavioral guidelines and journey definitions.

## Overview

Parlant is a framework that transforms how AI agents make decisions in customer-facing scenarios. Unlike traditional hope-based compliance approaches, Parlant ensures predictable agent behavior through:

- **Behavioral Guidelines**: Natural language rules with contextual matching
- **Journey Definitions**: Multi-step customer interaction flows
- **Tool Registration**: Decorator-based external service integration
- **Self-Critique Engine**: Real-time guideline adherence validation
- **Audit Logging**: Complete decision trail for compliance

## Quick Start

```bash
# Install Parlant
vrooli resource parlant manage install

# Start the service
vrooli resource parlant manage start --wait

# Create an agent
vrooli resource parlant content create-agent \
    --name "CustomerSupport" \
    --description "Handle customer inquiries"

# Add behavioral guidelines
vrooli resource parlant content add-guideline \
    --agent "agent_1" \
    --condition "User asks about refund" \
    --action "Explain refund policy and offer assistance" \
    --priority 1

# Check status
vrooli resource parlant status
```

## Architecture

Parlant decomposes agents into three layers:

1. **Policy Layer**: Guidelines and journeys as declarative objects
2. **Tool Layer**: External service integrations via decorators  
3. **Inference Layer**: LLM model abstraction supporting multiple providers

This separation enables:
- Policy-model decoupling for easy updates
- Improved testability with mocked components
- Better governance through audit trails
- Maintainability across model upgrades

## Usage Examples

### Creating an Agent

```bash
# Create a customer service agent
vrooli resource parlant content create-agent \
    --name "SupportBot" \
    --description "24/7 customer support assistant" \
    --model "gpt-4"
```

### Adding Behavioral Guidelines

```bash
# Add a guideline for handling complaints
vrooli resource parlant content add-guideline \
    --agent "agent_1" \
    --condition "customer expresses frustration" \
    --action "acknowledge feelings, apologize, and offer solution" \
    --priority 1

# Add a guideline for upselling
vrooli resource parlant content add-guideline \
    --agent "agent_1" \
    --condition "customer shows interest in premium features" \
    --action "explain benefits and provide pricing information" \
    --priority 2
```

### Interacting with Agents

```bash
# Send a message to an agent (via curl)
curl -X POST http://localhost:11458/agents/agent_1/chat \
    -H "Content-Type: application/json" \
    -d '{"agent_id": "agent_1", "message": "I need help with my refund"}'
```

## API Endpoints

- `POST /agents` - Create new agent
- `GET /agents` - List all agents
- `GET /agents/{id}` - Get agent details
- `POST /agents/{id}/guidelines` - Add behavioral guideline
- `POST /agents/{id}/journeys` - Define customer journey
- `POST /agents/{id}/tools` - Register tool/function
- `POST /agents/{id}/chat` - Send message to agent
- `GET /agents/{id}/history` - Retrieve conversation history
- `GET /health` - Health check endpoint

## Configuration

Environment variables:
- `PARLANT_PORT` - API server port (default: 11458)
- `PARLANT_HOST` - API server host (default: 127.0.0.1)
- `PARLANT_WORKERS` - Number of worker processes (default: 4)
- `PARLANT_MAX_AGENTS` - Maximum concurrent agents (default: 50)
- `PARLANT_USE_POSTGRES` - Enable PostgreSQL persistence (default: false)
- `PARLANT_USE_REDIS` - Enable Redis session state (default: false)

## Testing

```bash
# Run all tests
vrooli resource parlant test all

# Run specific test types
vrooli resource parlant test smoke       # Quick health check (<30s)
vrooli resource parlant test integration # Full functionality (<120s)
vrooli resource parlant test unit        # Library functions (<60s)
```

## Troubleshooting

### Service Won't Start
```bash
# Check if port is in use
netstat -tlnp | grep 11458

# Check logs
vrooli resource parlant logs --tail 100

# Verify installation
ls -la ~/.parlant/venv/
```

### Health Check Failing
```bash
# Test health endpoint directly
curl -v http://localhost:11458/health

# Check service status
vrooli resource parlant status --json
```

### Python Dependencies Issues
```bash
# Reinstall with force flag
vrooli resource parlant manage uninstall
vrooli resource parlant manage install --force
```

## Integration with Vrooli

Parlant integrates seamlessly with other Vrooli resources:

- **With Ollama**: Use local LLMs for agent inference
- **With PostgreSQL**: Persist conversation history
- **With Redis**: Manage session state
- **With LiteLLM**: Route between multiple LLM providers

Example scenario integration:
```bash
# Use Parlant in a customer service scenario
vrooli scenario customer-service develop
# The scenario can leverage Parlant for agent behavior control
```

## Security Considerations

- API keys managed through environment variables
- Session isolation between agents
- Audit logging for compliance requirements
- No direct prompt injection vulnerabilities
- Guidelines validated before application

## Performance

- Supports 50+ concurrent agents
- Handles 100+ guidelines per agent efficiently
- Processes 100 messages/second throughput
- Response time <500ms for guideline matching
- Memory usage <512MB baseline

## License

Apache 2.0 - Open source

## External Resources

- [Official Documentation](https://www.parlant.io/docs)
- [GitHub Repository](https://github.com/emcie-co/parlant)
- [PyPI Package](https://pypi.org/project/parlant/)