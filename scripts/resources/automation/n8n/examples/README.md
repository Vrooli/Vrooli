# n8n Example Workflows

This directory contains example workflows that demonstrate n8n capabilities and integration patterns.

## Available Examples

### example-notification-workflow.json
A simple workflow that demonstrates:
- Manual trigger activation
- Executing system commands
- Writing notification files to the host system

**Use case**: Basic system notifications and file operations

### webhook-workflow.json
An API-executable workflow that shows:
- Webhook trigger configuration
- REST API integration
- Response handling with `lastNode` mode

**Use case**: External service integration and API automation

## How to Import

### Method 1: Via Web Interface (Manual)
1. Access n8n web interface at http://localhost:5678
2. Click "Workflows" → "Import from File"
3. Select one of these JSON files
4. Customize the workflow for your needs
5. Save and activate the workflow

### Method 2: Via API (Programmatic)
```bash
# Get your API key from the configuration
API_KEY=$(jq -r '.services.automation.n8n.apiKey' ~/.vrooli/resources.local.json)

# Import a workflow
curl -X POST http://localhost:5678/api/v1/workflows \
  -H "X-N8N-API-KEY: $API_KEY" \
  -H "Content-Type: application/json" \
  -d @webhook-workflow.json

# Activate the imported workflow (replace WORKFLOW_ID with the returned ID)
curl -X POST http://localhost:5678/api/v1/workflows/WORKFLOW_ID/activate \
  -H "X-N8N-API-KEY: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{}'
```

### Method 3: Using the Management Script
```bash
# Execute a workflow (if it has a webhook trigger)
./scripts/resources/automation/n8n/manage.sh --action execute --workflow-id WORKFLOW_ID
```

## Tips

- Update webhook paths to avoid conflicts
- Test workflows in a safe environment first
- Use n8n's built-in error handling for production workflows
- Check the [n8n documentation](https://docs.n8n.io) for more examples