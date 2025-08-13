# Agent Metareasoning Manager API Documentation

## Overview
The Metareasoning Manager provides a comprehensive REST API for managing AI reasoning workflows, executing them through n8n or Windmill, and tracking performance metrics.

**Base URL**: `http://localhost:{SERVICE_PORT}`  
**Authentication**: Bearer token in Authorization header  
**Version**: 3.0.0

## Authentication
All endpoints except `/health` require authentication:

```bash
curl -H "Authorization: Bearer agent_metareasoning_manager_cli_default_2024" \
     http://localhost:${SERVICE_PORT}/workflows
```

## API Endpoints

### Health & System

#### GET /health
Check system health and service status.

**Response**:
```json
{
  "status": "healthy",
  "version": "3.0.0",
  "database": true,
  "services": {
    "n8n": true,
    "windmill": true
  },
  "workflows_count": 6,
  "timestamp": "2024-01-01T00:00:00Z"
}
```

#### GET /platforms
List available execution platforms.

**Response**:
```json
{
  "platforms": [
    {
      "name": "n8n",
      "description": "Visual workflow automation",
      "status": true,
      "url": "http://localhost:5678"
    },
    {
      "name": "windmill",
      "description": "Code-based workflows with UI",
      "status": true,
      "url": "http://localhost:5681"
    }
  ]
}
```

#### GET /models
List available AI models from Ollama.

**Response**:
```json
{
  "models": [
    {
      "name": "llama3.2",
      "size_mb": 4500,
      "modified_at": "2024-01-01T00:00:00Z"
    }
  ],
  "count": 3
}
```

#### GET /stats
Get system-wide statistics.

**Response**:
```json
{
  "total_workflows": 25,
  "active_workflows": 20,
  "total_executions": 1500,
  "successful_execs": 1400,
  "failed_execs": 100,
  "avg_execution_time": 3500,
  "most_used_workflow": "pros-cons",
  "most_used_model": "llama3.2",
  "last_execution": "2024-01-01T00:00:00Z"
}
```

### Workflow Management

#### GET /workflows
List all workflows with optional filtering.

**Query Parameters**:
- `platform` (string): Filter by platform (n8n, windmill)
- `type` (string): Filter by workflow type
- `active` (boolean): Show only active workflows (default: true)
- `page` (integer): Page number for pagination
- `page_size` (integer): Items per page (default: 20)

**Response**:
```json
{
  "workflows": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "pros-cons",
      "description": "Weighted advantages vs disadvantages",
      "type": "pros-cons",
      "platform": "n8n",
      "webhook_path": "analyze-pros-cons",
      "is_active": true,
      "usage_count": 45,
      "success_count": 42,
      "tags": ["decision-making", "analytical"]
    }
  ],
  "total": 25,
  "page": 1,
  "page_size": 20
}
```

#### GET /workflows/{id}
Get a specific workflow by ID.

**Response**:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "pros-cons",
  "description": "Weighted advantages vs disadvantages",
  "type": "pros-cons",
  "platform": "n8n",
  "config": {...},
  "webhook_path": "analyze-pros-cons",
  "schema": {
    "input": {"required": ["input"], "optional": ["context"]},
    "output": {"analysis": "object"}
  },
  "version": 1,
  "is_active": true,
  "is_builtin": true,
  "tags": ["decision-making"],
  "usage_count": 45,
  "created_at": "2024-01-01T00:00:00Z"
}
```

#### POST /workflows
Create a new workflow.

**Request Body**:
```json
{
  "name": "Custom Analysis",
  "description": "My custom workflow",
  "type": "custom",
  "platform": "n8n",
  "webhook_path": "custom-analysis",
  "schema": {...},
  "config": {...},
  "tags": ["custom", "analysis"],
  "estimated_duration_ms": 5000
}
```

#### PUT /workflows/{id}
Update a workflow (creates new version).

**Request Body**: Same as POST /workflows

#### DELETE /workflows/{id}
Soft delete a workflow (marks as inactive).

### Workflow Execution

#### POST /workflows/{id}/execute
Execute a workflow.

**Request Body**:
```json
{
  "input": "Should we migrate to microservices?",
  "context": "5-person team, legacy monolith",
  "model": "llama3.2",
  "metadata": {
    "user": "john.doe",
    "source": "api"
  }
}
```

**Response**:
```json
{
  "id": "execution-id",
  "workflow_id": "workflow-id",
  "status": "success",
  "data": {
    "pros": [...],
    "cons": [...],
    "recommendation": "..."
  },
  "execution_ms": 3500,
  "timestamp": "2024-01-01T00:00:00Z"
}
```

#### POST /analyze/{type}
Backward compatibility endpoint for quick analysis.

**Request Body**:
```json
{
  "input": "Should we use TypeScript?",
  "context": "New project",
  "model": "llama3.2"
}
```

### Advanced Features

#### GET /workflows/search
Search workflows using text search.

**Query Parameters**:
- `q` (string, required): Search query

**Response**:
```json
{
  "query": "risk",
  "results": [...],
  "count": 5
}
```

#### POST /workflows/generate
Generate a workflow from natural language prompt.

**Request Body**:
```json
{
  "prompt": "Create a workflow that analyzes customer sentiment",
  "platform": "n8n",
  "model": "llama3.2",
  "temperature": 0.7
}
```

#### POST /workflows/import
Import a workflow from n8n or Windmill export.

**Request Body**:
```json
{
  "platform": "n8n",
  "data": {...},
  "name": "Imported Workflow"
}
```

#### GET /workflows/{id}/export
Export a workflow to platform format.

**Query Parameters**:
- `format` (string): Export format (n8n, windmill, json)

#### POST /workflows/{id}/clone
Clone an existing workflow.

**Request Body**:
```json
{
  "name": "Cloned Workflow"
}
```

### Metrics & History

#### GET /workflows/{id}/history
Get execution history for a workflow.

**Query Parameters**:
- `limit` (integer): Max results (default: 50, max: 100)

**Response**:
```json
{
  "workflow_id": "id",
  "history": [
    {
      "id": "exec-id",
      "status": "success",
      "execution_time_ms": 3500,
      "model_used": "llama3.2",
      "created_at": "2024-01-01T00:00:00Z",
      "input_summary": "Should we...",
      "has_output": true
    }
  ],
  "count": 50
}
```

#### GET /workflows/{id}/metrics
Get performance metrics for a workflow.

**Response**:
```json
{
  "total_executions": 100,
  "success_count": 95,
  "failure_count": 5,
  "avg_execution_time": 3500,
  "min_execution_time": 2000,
  "max_execution_time": 8000,
  "last_execution": "2024-01-01T00:00:00Z",
  "success_rate": 95.0,
  "models_used": ["llama3.2", "mistral"]
}
```

## Error Handling

All errors follow a consistent format:

```json
{
  "error": "Detailed error message"
}
```

**Common Status Codes**:
- `200 OK`: Success
- `201 Created`: Resource created
- `400 Bad Request`: Invalid request
- `401 Unauthorized`: Missing/invalid token
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server error

## Rate Limiting

Currently no rate limiting is implemented, but this should be added for production use.

## Examples

### Create and Execute a Workflow

```bash
# Create workflow
curl -X POST http://localhost:${SERVICE_PORT}/workflows \
  -H "Authorization: Bearer agent_metareasoning_manager_cli_default_2024" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Analysis",
    "type": "custom",
    "platform": "n8n",
    "webhook_path": "my-analysis",
    "description": "Custom analysis workflow",
    "tags": ["custom"]
  }'

# Execute it
curl -X POST http://localhost:${SERVICE_PORT}/workflows/{id}/execute \
  -H "Authorization: Bearer agent_metareasoning_manager_cli_default_2024" \
  -H "Content-Type: application/json" \
  -d '{
    "input": "Test input",
    "model": "llama3.2"
  }'
```

### Generate Workflow from Prompt

```bash
curl -X POST http://localhost:${SERVICE_PORT}/workflows/generate \
  -H "Authorization: Bearer agent_metareasoning_manager_cli_default_2024" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Create a workflow that summarizes long documents",
    "platform": "n8n"
  }'
```

### Search and Clone

```bash
# Search for workflows
curl -X GET "http://localhost:${SERVICE_PORT}/workflows/search?q=risk" \
  -H "Authorization: Bearer agent_metareasoning_manager_cli_default_2024"

# Clone a workflow
curl -X POST http://localhost:${SERVICE_PORT}/workflows/{id}/clone \
  -H "Authorization: Bearer agent_metareasoning_manager_cli_default_2024" \
  -H "Content-Type: application/json" \
  -d '{"name": "Risk Assessment v2"}'
```

## CLI Usage

The CLI provides a convenient wrapper for all API endpoints:

```bash
# Basic operations
metareasoning health
metareasoning list
metareasoning get <workflow-id>
metareasoning execute <workflow-id> '{"input": "test"}'

# Advanced features
metareasoning search "risk assessment"
metareasoning generate "Create a sentiment analyzer"
metareasoning export <workflow-id> n8n
metareasoning history <workflow-id> 100
metareasoning metrics <workflow-id>
metareasoning stats

# Quick analysis (backward compatibility)
metareasoning analyze pros-cons "Should we adopt Kubernetes?"
```

## Database Schema

The API uses PostgreSQL with the following main tables:

- `workflows`: Workflow definitions with versioning
- `execution_history`: All workflow executions
- `workflow_metrics`: Aggregated performance metrics
- `reasoning_patterns`: Pattern library for semantic search
- `model_performance`: AI model performance tracking

Automatic triggers update metrics and statistics on every execution.

## Future Enhancements

- **Qdrant Integration**: Full semantic search using vector embeddings
- **WebSocket Support**: Real-time execution updates
- **Batch Operations**: Execute multiple workflows in parallel
- **Workflow Chains**: Define sequences of workflows
- **API Versioning**: Support multiple API versions
- **OAuth2**: More sophisticated authentication
- **Rate Limiting**: Protect against abuse
- **Caching**: Redis caching for frequent queries

---

*This API is designed to be a reference implementation demonstrating best practices for database integration, RESTful design, and comprehensive workflow management.*