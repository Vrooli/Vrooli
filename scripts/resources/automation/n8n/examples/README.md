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

1. Access n8n web interface at http://localhost:5678
2. Click "Workflows" → "Import from File"
3. Select one of these JSON files
4. Customize the workflow for your needs
5. Save and activate the workflow

## Tips

- Update webhook paths to avoid conflicts
- Test workflows in a safe environment first
- Use n8n's built-in error handling for production workflows
- Check the [n8n documentation](https://docs.n8n.io) for more examples