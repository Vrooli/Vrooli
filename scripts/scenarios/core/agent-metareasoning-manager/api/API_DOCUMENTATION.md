# Metareasoning API Documentation

## 🚀 Overview

The Metareasoning API is a high-performance REST API for managing and executing AI-powered workflows across multiple automation platforms. It provides comprehensive CRUD operations, AI-powered workflow generation, and advanced execution tracking.

### Key Features
- **Multi-Platform Support**: Execute workflows on n8n, Windmill, and other platforms
- **AI Integration**: Generate workflows using large language models (Ollama)
- **High Performance**: <100ms response times for most endpoints
- **Comprehensive Search**: Text-based and semantic workflow discovery
- **Import/Export**: Seamless workflow portability between platforms
- **Detailed Metrics**: Execution history and performance analytics

## 🔐 Authentication

All endpoints (except `/health`) require Bearer token authentication.

```bash
# Add authentication header to all requests
Authorization: Bearer metareasoning_cli_default_2024
```

**Available Tokens:**
- `metareasoning_cli_default_2024` - CLI access
- `admin_token_2024` - Administrative access
- `test_token_2024` - Testing access

## 📊 Performance Headers

The API includes performance monitoring headers in all responses:

- `X-Response-Time`: Server-side processing time (e.g., "25.3ms")
- `X-Cache`: Cache status ("HIT", "MISS", or "SKIP")
- `X-Cache-Age`: Age of cached response (e.g., "120s")

## 🌐 Base URL

```
Development: http://localhost:8093
Production:  https://api.metareasoning.example.com
```

## 📋 API Endpoints

### System Endpoints

#### Quick Health Check
```http
GET /health
```

Fast health check endpoint (<10ms target). No authentication required.

**Response:**
```json
{
  "status": "healthy",
  "version": "3.0.0",
  "database": true,
  "response_time_ms": 5.23,
  "timestamp": 1640995200
}
```

**Example:**
```bash
curl http://localhost:8093/health
```

#### Comprehensive Health Check
```http
GET /health/full
```

Detailed health check with service status information.

**Response:**
```json
{
  "status": "healthy",
  "version": "3.0.0",
  "database": true,
  "services": {
    "n8n": true,
    "windmill": false
  },
  "workflows_count": 42,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

#### List Available Models
```http
GET /models
```

Get available AI models from Ollama.

**Response:**
```json
{
  "models": [
    {
      "name": "llama2:latest",
      "size_mb": 3825,
      "modified_at": "2024-01-15T10:30:00Z"
    },
    {
      "name": "codellama:latest", 
      "size_mb": 3825,
      "modified_at": "2024-01-15T09:15:00Z"
    }
  ],
  "count": 2
}
```

#### List Execution Platforms
```http
GET /platforms
```

Get status of available execution platforms.

**Response:**
```json
{
  "platforms": [
    {
      "name": "n8n",
      "description": "n8n workflow automation",
      "status": true,
      "url": "http://localhost:5678"
    },
    {
      "name": "windmill",
      "description": "Windmill script execution",
      "status": true,
      "url": "http://localhost:8000"
    }
  ]
}
```

#### System Statistics
```http
GET /stats
```

Get system-wide statistics and metrics.

**Response:**
```json
{
  "total_workflows": 150,
  "active_workflows": 142,
  "total_executions": 2458,
  "successful_execs": 2301,
  "failed_execs": 157,
  "avg_execution_time": 1250,
  "most_used_workflow": "Data Processing Pipeline",
  "most_used_model": "llama2:latest",
  "last_execution": "2024-01-15T10:29:45Z"
}
```

### Workflow Management

#### List Workflows
```http
GET /workflows[?platform={platform}&type={type}&active={boolean}&page={int}&page_size={int}]
```

Get paginated list of workflows with filtering options.

**Query Parameters:**
- `platform` (optional): Filter by platform (`n8n`, `windmill`, `both`)
- `type` (optional): Filter by workflow type
- `active` (optional): Show only active workflows (default: `true`)
- `page` (optional): Page number (default: `1`)
- `page_size` (optional): Items per page (default: `20`, max: `100`)

**Example Request:**
```bash
curl -H "Authorization: Bearer metareasoning_cli_default_2024" \
  "http://localhost:8093/workflows?platform=n8n&page=1&page_size=5"
```

**Response:**
```json
{
  "workflows": [
    {
      "id": "123e4567-e89b-12d3-a456-426614174000",
      "name": "Data Processing Pipeline",
      "description": "Processes incoming data and generates reports",
      "type": "data-processing",
      "platform": "n8n",
      "config": {
        "nodes": [{"id": "1", "type": "webhook"}],
        "connections": {}
      },
      "version": 2,
      "is_active": true,
      "tags": ["data", "processing"],
      "usage_count": 156,
      "success_count": 148,
      "failure_count": 8,
      "created_by": "admin",
      "created_at": "2024-01-10T14:30:00Z",
      "updated_at": "2024-01-15T09:20:00Z"
    }
  ],
  "total": 25,
  "page": 1,
  "page_size": 5
}
```

#### Get Specific Workflow
```http
GET /workflows/{id}
```

Retrieve detailed information about a specific workflow.

**Example Request:**
```bash
curl -H "Authorization: Bearer metareasoning_cli_default_2024" \
  http://localhost:8093/workflows/123e4567-e89b-12d3-a456-426614174000
```

**Response:**
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "name": "Data Processing Pipeline",
  "description": "Comprehensive data processing with validation and reporting",
  "type": "data-processing",
  "platform": "n8n",
  "config": {
    "nodes": [
      {"id": "1", "type": "webhook", "name": "Data Input"},
      {"id": "2", "type": "function", "name": "Validate Data"},
      {"id": "3", "type": "function", "name": "Process Data"},
      {"id": "4", "type": "webhook", "name": "Send Results"}
    ],
    "connections": {
      "1": {"main": [[{"node": "2", "type": "main", "index": 0}]]},
      "2": {"main": [[{"node": "3", "type": "main", "index": 0}]]},
      "3": {"main": [[{"node": "4", "type": "main", "index": 0}]]}
    }
  },
  "webhook_path": "/webhook/data-processing",
  "estimated_duration": 2500,
  "version": 2,
  "parent_id": "original-workflow-id",
  "is_active": true,
  "is_builtin": false,
  "tags": ["data", "processing", "validation"],
  "usage_count": 156,
  "success_count": 148,
  "failure_count": 8,
  "avg_execution_time_ms": 1847,
  "created_by": "admin",
  "created_at": "2024-01-10T14:30:00Z",
  "updated_at": "2024-01-15T09:20:00Z"
}
```

#### Create New Workflow
```http
POST /workflows
```

Create a new workflow.

**Request Body:**
```json
{
  "name": "Customer Notification System",
  "description": "Sends notifications to customers based on events",
  "type": "notification",
  "platform": "n8n",
  "config": {
    "nodes": [
      {"id": "1", "type": "webhook", "name": "Event Trigger"},
      {"id": "2", "type": "function", "name": "Format Message"},
      {"id": "3", "type": "email", "name": "Send Email"}
    ],
    "connections": {
      "1": {"main": [[{"node": "2", "type": "main", "index": 0}]]},
      "2": {"main": [[{"node": "3", "type": "main", "index": 0}]]}
    }
  },
  "tags": ["notification", "email", "customer"],
  "estimated_duration": 1500
}
```

**Example Request:**
```bash
curl -X POST \
  -H "Authorization: Bearer metareasoning_cli_default_2024" \
  -H "Content-Type: application/json" \
  -d '{"name":"Test Workflow","type":"test","platform":"n8n","config":{}}' \
  http://localhost:8093/workflows
```

**Response:** Returns the created workflow with assigned ID and timestamps.

#### Update Workflow
```http
PUT /workflows/{id}
```

Update an existing workflow (creates a new version).

**Example Request:**
```bash
curl -X PUT \
  -H "Authorization: Bearer metareasoning_cli_default_2024" \
  -H "Content-Type: application/json" \
  -d '{"name":"Updated Workflow","type":"test","platform":"n8n","config":{}}' \
  http://localhost:8093/workflows/123e4567-e89b-12d3-a456-426614174000
```

#### Delete Workflow
```http
DELETE /workflows/{id}
```

Soft delete a workflow (marks as inactive).

**Example Request:**
```bash
curl -X DELETE \
  -H "Authorization: Bearer metareasoning_cli_default_2024" \
  http://localhost:8093/workflows/123e4567-e89b-12d3-a456-426614174000
```

**Response:**
```json
{
  "message": "workflow deleted"
}
```

### Workflow Execution

#### Execute Workflow
```http
POST /workflows/{id}/execute
```

Execute a specific workflow with input data.

**Request Body:**
```json
{
  "input": {
    "data": "sample input data",
    "parameters": {
      "timeout": 30000,
      "retry_count": 3
    }
  },
  "context": "production",
  "model": "llama2:latest",
  "metadata": {
    "user_id": "user123",
    "session_id": "session456"
  }
}
```

**Example Request:**
```bash
curl -X POST \
  -H "Authorization: Bearer metareasoning_cli_default_2024" \
  -H "Content-Type: application/json" \
  -d '{"input":{"message":"Hello World"}}' \
  http://localhost:8093/workflows/123e4567-e89b-12d3-a456-426614174000/execute
```

**Response:**
```json
{
  "id": "exec-uuid-here",
  "workflow_id": "123e4567-e89b-12d3-a456-426614174000",
  "status": "success",
  "data": {
    "output": "Processed: Hello World",
    "metrics": {
      "nodes_executed": 4,
      "processing_time": 1847
    }
  },
  "execution_ms": 1847,
  "timestamp": "2024-01-15T10:35:22Z"
}
```

#### Get Execution History
```http
GET /workflows/{id}/history[?limit={int}]
```

Retrieve execution history for a workflow.

**Query Parameters:**
- `limit` (optional): Maximum entries to return (default: `50`, max: `100`)

**Example Request:**
```bash
curl -H "Authorization: Bearer metareasoning_cli_default_2024" \
  "http://localhost:8093/workflows/123e4567-e89b-12d3-a456-426614174000/history?limit=5"
```

**Response:**
```json
{
  "workflow_id": "123e4567-e89b-12d3-a456-426614174000",
  "history": [
    {
      "id": "exec-1",
      "status": "success",
      "execution_time_ms": 1847,
      "model_used": "llama2:latest",
      "created_at": "2024-01-15T10:35:22Z",
      "input_summary": "Hello World message",
      "has_output": true
    },
    {
      "id": "exec-2", 
      "status": "failed",
      "execution_time_ms": 892,
      "model_used": "llama2:latest",
      "created_at": "2024-01-15T09:22:15Z",
      "input_summary": "Invalid data format",
      "has_output": false
    }
  ],
  "count": 2
}
```

#### Get Workflow Metrics
```http
GET /workflows/{id}/metrics
```

Get performance metrics for a specific workflow.

**Response:**
```json
{
  "total_executions": 156,
  "success_count": 148,
  "failure_count": 8,
  "avg_execution_time": 1847,
  "min_execution_time": 892,
  "max_execution_time": 3456,
  "last_execution": "2024-01-15T10:35:22Z",
  "success_rate": 94.87,
  "models_used": ["llama2:latest", "codellama:latest"]
}
```

### AI-Powered Workflow Generation

#### Generate Workflow from Prompt
```http
POST /workflows/generate
```

Generate a workflow using AI from natural language description.

**Request Body:**
```json
{
  "prompt": "Create a workflow that processes incoming webhooks, validates the data, stores it in a database, and sends a confirmation email",
  "platform": "n8n",
  "model": "llama2:latest",
  "temperature": 0.7
}
```

**Example Request:**
```bash
curl -X POST \
  -H "Authorization: Bearer metareasoning_cli_default_2024" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Create a simple data processing workflow",
    "platform": "n8n",
    "model": "llama2",
    "temperature": 0.7
  }' \
  http://localhost:8093/workflows/generate
```

**Response:** Returns a `WorkflowCreate` object that can be used to create the workflow.

```json
{
  "name": "Data Processing Workflow",
  "description": "AI-generated workflow for data processing",
  "type": "data-processing",
  "platform": "n8n",
  "config": {
    "nodes": [
      {"id": "1", "type": "webhook", "name": "Data Input"},
      {"id": "2", "type": "function", "name": "Process Data"},
      {"id": "3", "type": "database", "name": "Store Results"}
    ],
    "connections": {
      "1": {"main": [[{"node": "2", "type": "main", "index": 0}]]},
      "2": {"main": [[{"node": "3", "type": "main", "index": 0}]]}
    }
  },
  "tags": ["generated", "data-processing"],
  "estimated_duration": 2000
}
```

### Import/Export Operations

#### Import Workflow
```http
POST /workflows/import
```

Import a workflow from external platform format.

**Request Body:**
```json
{
  "platform": "n8n",
  "name": "Imported Customer Workflow",
  "data": {
    "nodes": [
      {"id": "1", "type": "webhook"},
      {"id": "2", "type": "function"}
    ],
    "connections": {
      "1": {"main": [[{"node": "2", "type": "main", "index": 0}]]}
    },
    "settings": {
      "executionOrder": "v1"
    }
  }
}
```

**Example Request:**
```bash
curl -X POST \
  -H "Authorization: Bearer metareasoning_cli_default_2024" \
  -H "Content-Type: application/json" \
  -d '{
    "platform": "n8n",
    "name": "Imported Workflow",
    "data": {"nodes": [{"id": "1", "type": "webhook"}]}
  }' \
  http://localhost:8093/workflows/import
```

#### Export Workflow
```http
GET /workflows/{id}/export[?format={format}]
```

Export workflow to platform-specific format.

**Query Parameters:**
- `format` (optional): Export format (`json`, `yaml`) (default: `json`)

**Example Request:**
```bash
curl -H "Authorization: Bearer metareasoning_cli_default_2024" \
  "http://localhost:8093/workflows/123e4567-e89b-12d3-a456-426614174000/export?format=json"
```

**Response:**
```json
{
  "format": "json",
  "platform": "n8n",
  "data": {
    "nodes": [...],
    "connections": {...},
    "settings": {...}
  }
}
```

#### Clone Workflow
```http
POST /workflows/{id}/clone
```

Create a copy of an existing workflow.

**Request Body:**
```json
{
  "name": "Cloned Customer Workflow"
}
```

### Search and Discovery

#### Search Workflows
```http
GET /workflows/search?q={query}
```

Search workflows using text-based search.

**Query Parameters:**
- `q` (required): Search query string

**Example Request:**
```bash
curl -H "Authorization: Bearer metareasoning_cli_default_2024" \
  "http://localhost:8093/workflows/search?q=data%20processing"
```

**Response:**
```json
{
  "query": "data processing",
  "results": [
    {
      "id": "123e4567-e89b-12d3-a456-426614174000",
      "name": "Data Processing Pipeline",
      "description": "Processes incoming data...",
      "type": "data-processing",
      "platform": "n8n",
      "tags": ["data", "processing"],
      "usage_count": 156
    }
  ],
  "count": 1
}
```

## 📝 Error Responses

All error responses follow a consistent format:

```json
{
  "error": "Error message describing what went wrong",
  "code": 400,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### Common Error Codes

- **400 Bad Request**: Invalid request data or parameters
- **401 Unauthorized**: Missing or invalid authentication token
- **404 Not Found**: Resource not found
- **422 Unprocessable Entity**: Valid format but invalid data
- **500 Internal Server Error**: Server-side error

## 🚀 Performance Guidelines

### Response Time Targets
- **Health endpoints**: <10ms
- **List operations**: <50ms  
- **CRUD operations**: <100ms
- **Search operations**: <100ms
- **AI generation**: <5000ms
- **Execution**: Varies by workflow complexity

### Optimization Features
- **Caching**: GET requests cached for 5 minutes
- **Compression**: Automatic gzip compression
- **Connection Pooling**: Optimized database connections
- **Performance Headers**: Response timing in headers

### Rate Limiting
Currently not implemented, but recommended for production:
- 100 requests per minute per API key
- Burst allowance of 20 requests
- Different limits for different endpoint categories

## 📊 Monitoring and Metrics

### Performance Headers
Every response includes performance metrics:
```
X-Response-Time: 25.3ms
X-Cache: HIT
X-Cache-Age: 120s
```

### Health Monitoring
Use `/health` for automated health checks:
```bash
#!/bin/bash
response=$(curl -s http://localhost:8093/health)
status=$(echo $response | jq -r '.status')
if [ "$status" != "healthy" ]; then
  echo "API is not healthy: $response"
  exit 1
fi
```

## 🛠️ Development Tools

### OpenAPI Specification
Full OpenAPI 3.0 specification available at `/openapi.yaml`

### Testing
```bash
# Run all tests
make test

# Run benchmarks
make benchmark

# Performance testing
make performance-test
```

### Local Development
```bash
# Start API server
make run

# Monitor performance
./monitor_performance.sh

# View logs
tail -f api.log
```

## 🔗 Integration Examples

### Python Client
```python
import requests

class MetareasoningAPI:
    def __init__(self, base_url, token):
        self.base_url = base_url
        self.headers = {"Authorization": f"Bearer {token}"}
    
    def list_workflows(self, platform=None, page=1, page_size=20):
        params = {"page": page, "page_size": page_size}
        if platform:
            params["platform"] = platform
        
        response = requests.get(
            f"{self.base_url}/workflows",
            headers=self.headers,
            params=params
        )
        return response.json()
    
    def execute_workflow(self, workflow_id, input_data):
        response = requests.post(
            f"{self.base_url}/workflows/{workflow_id}/execute",
            headers=self.headers,
            json={"input": input_data}
        )
        return response.json()

# Usage
api = MetareasoningAPI("http://localhost:8093", "metareasoning_cli_default_2024")
workflows = api.list_workflows(platform="n8n")
```

### cURL Examples
```bash
# List all n8n workflows
curl -H "Authorization: Bearer metareasoning_cli_default_2024" \
  "http://localhost:8093/workflows?platform=n8n"

# Execute a workflow
curl -X POST \
  -H "Authorization: Bearer metareasoning_cli_default_2024" \
  -H "Content-Type: application/json" \
  -d '{"input": {"message": "test"}}' \
  http://localhost:8093/workflows/{workflow-id}/execute

# Generate new workflow
curl -X POST \
  -H "Authorization: Bearer metareasoning_cli_default_2024" \
  -H "Content-Type: application/json" \
  -d '{"prompt": "Create email notification workflow", "platform": "n8n"}' \
  http://localhost:8093/workflows/generate
```

---

For more information, see the [OpenAPI specification](./openapi.yaml) or contact the development team.