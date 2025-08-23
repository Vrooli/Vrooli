# n8n API Reference

n8n provides comprehensive REST API access and CLI capabilities for workflow automation and management. This guide covers both the manage.sh integration and direct API usage.

> **💡 Recommended Usage**: Use the `manage.sh` script for most operations. This guide shows direct API usage for advanced integration scenarios.

## Base Access

- **Web Interface**: `http://localhost:5678`
- **Recommended Management**: `./manage.sh --action [action]`
- **REST API Base**: `http://localhost:5678/api/v1/`
- **CLI Access**: `docker exec n8n n8n [command]`

## Management API (via manage.sh)

### Service Management

**Recommended Method:**
```bash
# Check service status with comprehensive information
./manage.sh --action status

# Start/stop/restart service
./manage.sh --action start
./manage.sh --action stop
./manage.sh --action restart

# View service logs
./manage.sh --action logs

# Test functionality
./manage.sh --action test
```

### Workflow Management

**Recommended Method:**
```bash
# List all workflows with details
./manage.sh --action workflow-list

# Execute specific workflow by ID
./manage.sh --action execute --workflow-id WORKFLOW_ID

# Import/export workflows
./manage.sh --action export-workflows --output backup.json
./manage.sh --action import-workflows --input backup.json

# Activate/deactivate workflows
./manage.sh --action activate-workflow --workflow-id WORKFLOW_ID
./manage.sh --action deactivate-workflow --workflow-id WORKFLOW_ID
```

### Database and Security

**Recommended Method:**
```bash
# Setup authentication
./manage.sh --action install --basic-auth yes --username admin --password secure123

# Database operations
./manage.sh --action install --database postgres
./manage.sh --action backup-database --output db-backup.sql
./manage.sh --action restore-database --input db-backup.sql
```

## n8n REST API

**API Key Required**: Create an API key through the web interface at Settings → n8n API.

### Authentication Setup

1. Access n8n web interface: `http://localhost:5678`
2. Go to Settings → n8n API
3. Create API key with appropriate scopes
4. Use the key in API requests:

```bash
# Set API key as environment variable (recommended)
export N8N_API_KEY="your-api-key-here"

# Use in requests
curl -H "X-N8N-API-KEY: $N8N_API_KEY" http://localhost:5678/api/v1/workflows
```

### Workflow Management

#### List Workflows
```http
GET /api/v1/workflows
```

**Recommended Method:**
```bash
./manage.sh --action workflow-list
```

**Direct API:**
```bash
curl -H "X-N8N-API-KEY: $N8N_API_KEY" http://localhost:5678/api/v1/workflows
```

**Response:**
```json
{
  "data": [
    {
      "id": "1",
      "name": "Example Workflow",
      "active": true,
      "nodes": [...],
      "connections": {...}
    }
  ]
}
```

#### Get Specific Workflow
```http
GET /api/v1/workflows/{id}
```

```bash
curl -H "X-N8N-API-KEY: $N8N_API_KEY" http://localhost:5678/api/v1/workflows/1
```

#### Create Workflow
```http
POST /api/v1/workflows
Content-Type: application/json
```

```bash
curl -X POST -H "X-N8N-API-KEY: $N8N_API_KEY" \
     -H "Content-Type: application/json" \
     -d @workflow.json \
     http://localhost:5678/api/v1/workflows
```

#### Execute Workflow
```http
POST /api/v1/workflows/{id}/execute
```

**Recommended Method:**
```bash
./manage.sh --action execute --workflow-id WORKFLOW_ID
```

**Direct API:**
```bash
curl -X POST -H "X-N8N-API-KEY: $N8N_API_KEY" \
     -H "Content-Type: application/json" \
     http://localhost:5678/api/v1/workflows/1/execute
```

#### Update Workflow Status
```http
PATCH /api/v1/workflows/{id}
Content-Type: application/json
```

```bash
# Activate workflow
curl -X PATCH -H "X-N8N-API-KEY: $N8N_API_KEY" \
     -H "Content-Type: application/json" \
     -d '{"active": true}' \
     http://localhost:5678/api/v1/workflows/1

# Deactivate workflow
curl -X PATCH -H "X-N8N-API-KEY: $N8N_API_KEY" \
     -H "Content-Type: application/json" \
     -d '{"active": false}' \
     http://localhost:5678/api/v1/workflows/1
```

### Execution Management

#### List Executions
```http
GET /api/v1/executions
```

**Query Parameters:**
- `filter`: JSON filter object
- `limit`: Number of results (default: 20)
- `includeData`: Include execution data (default: false)

```bash
curl -H "X-N8N-API-KEY: $N8N_API_KEY" \
     "http://localhost:5678/api/v1/executions?limit=10&includeData=false"
```

#### Get Execution Details
```http
GET /api/v1/executions/{id}
```

```bash
curl -H "X-N8N-API-KEY: $N8N_API_KEY" \
     http://localhost:5678/api/v1/executions/execution-id
```

#### Stop Running Execution
```http
POST /api/v1/executions/{id}/stop
```

```bash
curl -X POST -H "X-N8N-API-KEY: $N8N_API_KEY" \
     http://localhost:5678/api/v1/executions/execution-id/stop
```

### Credentials Management

#### List Credentials
```http
GET /api/v1/credentials
```

```bash
curl -H "X-N8N-API-KEY: $N8N_API_KEY" http://localhost:5678/api/v1/credentials
```

#### Create Credentials
```http
POST /api/v1/credentials
Content-Type: application/json
```

```bash
curl -X POST -H "X-N8N-API-KEY: $N8N_API_KEY" \
     -H "Content-Type: application/json" \
     -d '{
       "name": "My API Key",
       "type": "httpHeaderAuth",
       "data": {
         "name": "Authorization",
         "value": "Bearer token-here"
       }
     }' \
     http://localhost:5678/api/v1/credentials
```

## n8n CLI Commands

**Access via Docker:**
```bash
docker exec n8n n8n [command]
```

### Workflow Operations

#### List Workflows
```bash
# List all workflows
docker exec n8n n8n list:workflow

# List only active workflows
docker exec n8n n8n list:workflow --active=true

# List with IDs only
docker exec n8n n8n list:workflow --onlyId
```

#### Execute Workflows
```bash
# Execute workflow by ID
docker exec n8n n8n execute --id=WORKFLOW_ID

# Execute with raw JSON output
docker exec n8n n8n execute --id=WORKFLOW_ID --rawOutput
```

**Note**: CLI execution has known issues in n8n v1.93.0+. Use the manage.sh script or REST API instead.

#### Import/Export Workflows
```bash
# Export single workflow
docker exec n8n n8n export:workflow --id=WORKFLOW_ID --output=workflow.json --pretty

# Export all workflows
docker exec n8n n8n export:workflow --all --output=backup-dir/ --separate

# Create backup
docker exec n8n n8n export:workflow --backup --output=backups/latest/

# Import workflow
docker exec n8n n8n import:workflow --input=workflow.json

# Import multiple workflows
docker exec n8n n8n import:workflow --separate --input=backups/latest/
```

#### Update Workflows
```bash
# Activate workflow
docker exec n8n n8n update:workflow --id=WORKFLOW_ID --active=true

# Deactivate all workflows
docker exec n8n n8n update:workflow --all --active=false
```

### AI-Powered Workflow Generation

#### Generate from Text Prompt
```bash
# Generate workflow from description
docker exec n8n n8n ttwf:generate \
  --prompt "Create a telegram chatbot that can tell current weather in Berlin" \
  --output result.json

# Batch generate from dataset
docker exec n8n n8n ttwf:generate \
  --input dataset.jsonl \
  --output results.jsonl \
  --limit 10 \
  --concurrency 2
```

## Webhook API

### Trigger Workflows via Webhooks

#### Test Webhook
```http
GET /webhook-test/{path}
POST /webhook-test/{path}
```

```bash
# GET request
curl http://localhost:5678/webhook-test/my-trigger

# POST with data
curl -X POST -H "Content-Type: application/json" \
     -d '{"message": "Hello from webhook"}' \
     http://localhost:5678/webhook-test/my-trigger
```

#### Production Webhook
```http
GET /webhook/{path}
POST /webhook/{path}
```

```bash
# Production webhook endpoint
curl -X POST -H "Content-Type: application/json" \
     -d '{"data": "production data"}' \
     http://localhost:5678/webhook/production-trigger
```

## Complete Workflow Example

### System Notification Workflow

This example demonstrates creating and executing a workflow programmatically.

#### 1. Workflow Definition (JSON)
```json
{
  "name": "System Notification Workflow",
  "active": false,
  "nodes": [
    {
      "parameters": {
        "path": "notify",
        "responseMode": "lastNode",
        "options": {}
      },
      "type": "n8n-nodes-base.webhook",
      "typeVersion": 1.1,
      "position": [0, 0],
      "id": "webhook-trigger",
      "name": "Webhook"
    },
    {
      "parameters": {
        "command": "echo \"Workflow executed at $(date)\" > /workspace/notification.txt"
      },
      "type": "n8n-nodes-base.executeCommand",
      "typeVersion": 1,
      "position": [300, 0],
      "id": "create-notification",
      "name": "Create Notification"
    }
  ],
  "connections": {
    "Webhook": {
      "main": [
        [
          {
            "node": "Create Notification",
            "type": "main",
            "index": 0
          }
        ]
      ]
    }
  }
}
```

#### 2. Create and Execute Workflow
```bash
# Save workflow to file
cat > notification-workflow.json << 'EOF'
[workflow JSON from above]
EOF

# Create workflow via API
WORKFLOW_ID=$(curl -X POST -H "X-N8N-API-KEY: $N8N_API_KEY" \
  -H "Content-Type: application/json" \
  -d @notification-workflow.json \
  http://localhost:5678/api/v1/workflows | jq -r '.id')

# Activate workflow
curl -X PATCH -H "X-N8N-API-KEY: $N8N_API_KEY" \
     -H "Content-Type: application/json" \
     -d '{"active": true}' \
     http://localhost:5678/api/v1/workflows/$WORKFLOW_ID

# Trigger workflow via webhook
curl -X POST -H "Content-Type: application/json" \
     -d '{"message": "Test notification"}' \
     http://localhost:5678/webhook/notify
```

## Error Handling

### Standard Error Response
```json
{
  "code": 404,
  "message": "Workflow not found",
  "hint": "Check if the workflow ID is correct"
}
```

### Common Error Codes
- `400`: Bad Request - Invalid request data
- `401`: Unauthorized - Invalid or missing API key
- `404`: Not Found - Resource doesn't exist
- `429`: Too Many Requests - Rate limit exceeded
- `500`: Internal Server Error - Server-side error

## Best Practices

### API Usage
1. **Use manage.sh for operations** - More reliable than direct API calls
2. **Store API keys securely** - Use environment variables, not hardcoded values
3. **Implement proper error handling** - Always check response status codes
4. **Rate limiting awareness** - Don't exceed API rate limits
5. **Use pagination** - For large result sets

### Workflow Design
1. **Error handling nodes** - Include error handling in workflows
2. **Credential management** - Use n8n's credential system for sensitive data
3. **Testing workflows** - Test with webhook-test endpoints first
4. **Documentation** - Document workflow purpose and parameters

### Security
1. **API key management** - Rotate keys regularly
2. **Network security** - Use HTTPS in production
3. **Input validation** - Validate all webhook inputs
4. **Access control** - Limit API key permissions

## Integration Examples

### With Vrooli Resources
```bash
# Execute workflow to check other resources
./manage.sh --action execute --workflow-id resource-check-workflow

# Create workflow via API to monitor Ollama
curl -X POST -H "X-N8N-API-KEY: $N8N_API_KEY" \
     -H "Content-Type: application/json" \
     -d @ollama-monitor-workflow.json \
     http://localhost:5678/api/v1/workflows
```

### With External Services
```bash
# Trigger workflow from external system
curl -X POST -H "Content-Type: application/json" \
     -d '{"source": "external-system", "data": {...}}' \
     http://localhost:5678/webhook/external-trigger
```

See the [examples directory](../examples/) for complete workflow definitions and the [configuration guide](CONFIGURATION.md) for advanced setup options.