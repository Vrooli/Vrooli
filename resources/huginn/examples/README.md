# Huginn Examples

This directory contains example configurations for Huginn agents, scenarios, and complete workflows.

## Directory Structure

- `agents/` - Individual agent configurations
- `scenarios/` - Multi-agent scenario configurations  
- `workflows/` - Complete workflow implementations

## Agents

Individual agent examples demonstrating specific functionality:

### event-relay.json
Relays events between different systems using webhooks.

### price-tracker.json
Monitors product prices and detects changes.

### rss-aggregator.json
Aggregates RSS feeds from multiple sources.

### website-monitor.json
Monitors website availability and content changes.

## Scenarios

Multi-agent scenarios showing how agents work together:

### intelligence-hub.json
Comprehensive intelligence gathering from multiple sources including RSS feeds, APIs, and websites.

### monitoring-suite.json
Complete monitoring solution for websites, APIs, and system health.

## Workflows

End-to-end workflow implementations:

### daily-digest.json
Collects news and weather data to create a daily email digest.

### website-change-alert.json
Monitors websites for changes and sends notifications via webhooks.

### vrooli-integration.json
Demonstrates integration with other Vrooli resources (Redis, Ollama, MinIO, Node-RED).

## Usage

1. **Import via Web UI:**
   - Navigate to http://localhost:4111
   - Go to Scenarios > Import
   - Upload or paste the JSON content

2. **Import via API:**
   ```bash
   # Import an agent
   curl -X POST http://localhost:4111/api/agents \
     -H "Content-Type: application/json" \
     -d @agents/rss-aggregator.json

   # Import a scenario
   curl -X POST http://localhost:4111/api/scenarios \
     -H "Content-Type: application/json" \
     -d @scenarios/monitoring-suite.json
   ```

3. **Via Management Script:**
   ```bash
   # Future feature - import examples
   ./manage.sh --action import --file examples/workflows/daily-digest.json
   ```

## Customization

All examples use placeholder values that need to be customized:

- API keys: Replace `YOUR_API_KEY` with actual keys
- URLs: Update webhook URLs and endpoints
- Email addresses: Change to your actual email
- Schedules: Adjust timing to your needs

## Creating Your Own

Use these examples as templates:

1. Copy an existing example
2. Modify the agent type and options
3. Update connections between agents
4. Test in development before production

## Best Practices

- Start with simple agents before building complex scenarios
- Use descriptive names for agents
- Add notes to document agent purpose
- Test each agent individually first
- Use deduplication agents to prevent duplicate events
- Monitor agent logs for debugging