# Vrooli Resource API Documentation

## Overview

The Vrooli Resource API provides comprehensive endpoints for managing, monitoring, and orchestrating AI resources, automation workflows, and system integrations. This API enables dynamic resource discovery, health monitoring, task execution, and real-time coordination across the Vrooli ecosystem.

## Base URL

```
http://localhost:3000/api
```

## Authentication

All requests require a valid JWT token in the Authorization header:

```http
Authorization: Bearer <jwt_token>
```

## API Versioning

The API uses URL versioning:
- Current version: `v1`
- Example: `/api/v1/resources`

## Content Types

- **Request**: `application/json`
- **Response**: `application/json`
- **File uploads**: `multipart/form-data`

---

## Resources Management

### Discover Resources

Automatically discover available resources in the environment.

**Endpoint**: `GET /api/v1/resources/discover`

**Response**:
```json
{
  "success": true,
  "data": {
    "resources": [
      {
        "id": "ollama",
        "name": "Ollama Local LLM",
        "type": "ai",
        "status": "running",
        "baseUrl": "http://localhost:11434",
        "capabilities": ["text_generation", "code_analysis", "vision"],
        "models": ["llama3.1:8b", "qwen2.5-coder:7b"],
        "metadata": {
          "version": "0.1.20",
          "uptime": 3600,
          "memory_usage": "2.1GB"
        }
      }
    ],
    "summary": {
      "total": 8,
      "running": 7,
      "stopped": 1,
      "by_type": {
        "ai": 3,
        "automation": 3,
        "storage": 1,
        "agent": 1
      }
    }
  },
  "timestamp": "2025-07-29T14:30:00Z",
  "requestId": "req_123456"
}
```

### List Resources

Get paginated list of resources with filtering options.

**Endpoint**: `GET /api/v1/resources`

**Query Parameters**:
- `type` (string): Filter by resource type (ai, automation, storage, agent, search)
- `status` (string): Filter by status (running, stopped, error)
- `enabled` (boolean): Filter by enabled status
- `page` (integer): Page number (default: 1)
- `limit` (integer): Items per page (default: 10, max: 100)

**Example Request**:
```http
GET /api/v1/resources?type=ai&status=running&page=1&limit=5
```

**Response**:
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": "ollama",
        "name": "Ollama Local LLM",
        "type": "ai",
        "status": "running",
        "baseUrl": "http://localhost:11434",
        "enabled": true,
        "lastChecked": "2025-07-29T14:29:30Z",
        "responseTime": 45,
        "metrics": {
          "requestCount": 1523,
          "errorRate": 0.02,
          "avgResponseTime": 1250
        }
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 5,
      "total": 3,
      "totalPages": 1,
      "hasNext": false,
      "hasPrev": false
    }
  }
}
```

### Get Resource Details

Get detailed information about a specific resource.

**Endpoint**: `GET /api/v1/resources/{resourceId}`

**Path Parameters**:
- `resourceId` (string): Unique resource identifier

**Response**:
```json
{
  "success": true,
  "data": {
    "id": "ollama",
    "name": "Ollama Local LLM",
    "type": "ai",
    "status": "running",
    "baseUrl": "http://localhost:11434",
    "enabled": true,
    "configuration": {
      "models": ["llama3.1:8b", "qwen2.5-coder:7b", "llama3.2-vision:11b"],
      "maxConcurrency": 4,
      "timeout": 30000,
      "retries": 3
    },
    "capabilities": [
      {
        "name": "text_generation",
        "description": "Generate text responses from prompts",
        "parameters": ["prompt", "model", "temperature", "max_tokens"]
      },
      {
        "name": "vision_analysis",
        "description": "Analyze images and documents",
        "parameters": ["image", "prompt", "model"]
      }
    ],
    "healthCheck": {
      "endpoint": "/api/tags",
      "interval": 60000,
      "timeout": 5000,
      "retries": 3,
      "lastCheck": "2025-07-29T14:29:30Z",
      "status": "healthy"
    },
    "metrics": {
      "uptime": 86400,
      "requestCount": 1523,
      "errorCount": 31,
      "avgResponseTime": 1250,
      "successRate": 0.98,
      "lastError": "2025-07-29T12:15:00Z",
      "resourceUsage": {
        "cpu": 45.2,
        "memory": "2.1GB",
        "gpu": 78.5
      }
    },
    "createdAt": "2025-07-28T10:00:00Z",
    "updatedAt": "2025-07-29T14:29:30Z"
  }
}
```

---

## Health Monitoring

### Check Resource Health

Perform health check on a specific resource.

**Endpoint**: `GET /api/v1/resources/{resourceId}/health`

**Response**:
```json
{
  "success": true,
  "data": {
    "resourceId": "ollama",
    "healthy": true,
    "responseTime": 42,
    "timestamp": "2025-07-29T14:30:00Z",
    "details": {
      "status": "running",
      "models_loaded": 3,
      "available_memory": "6.2GB",
      "active_connections": 2,
      "queue_size": 0
    },
    "checks": [
      {
        "name": "endpoint_reachable",
        "status": "pass",
        "responseTime": 15
      },
      {
        "name": "models_available",
        "status": "pass",
        "count": 3
      },
      {
        "name": "memory_usage",
        "status": "pass",
        "usage": "2.1GB/8GB"
      }
    ]
  }
}
```

### System Health Overview

Get comprehensive health status of all resources.

**Endpoint**: `GET /api/v1/health`

**Response**:
```json
{
  "success": true,
  "data": {
    "overall": "healthy",
    "timestamp": "2025-07-29T14:30:00Z",
    "summary": {
      "total_resources": 8,
      "healthy": 7,
      "unhealthy": 1,
      "unknown": 0
    },
    "resources": [
      {
        "id": "ollama",
        "type": "ai",
        "status": "healthy",
        "responseTime": 42,
        "lastCheck": "2025-07-29T14:29:30Z"
      },
      {
        "id": "node-red",
        "type": "automation",
        "status": "unhealthy",
        "error": "Connection timeout",
        "lastCheck": "2025-07-29T14:28:45Z"
      }
    ],
    "metrics": {
      "avgResponseTime": 156,
      "slowestResource": "whisper",
      "fastestResource": "redis",
      "errorRate": 0.125
    }
  }
}
```

---

## Task Execution

### Execute Task

Execute a task on a specific resource.

**Endpoint**: `POST /api/v1/resources/{resourceId}/execute`

**Request Body**:
```json
{
  "task": "generate_text",
  "parameters": {
    "prompt": "Explain artificial intelligence in simple terms",
    "model": "llama3.1:8b",
    "temperature": 0.7,
    "max_tokens": 500
  },
  "timeout": 30000,
  "priority": "medium",
  "metadata": {
    "userId": "user_123",
    "sessionId": "session_456"
  }
}
```

**Response**:
```json
{
  "success": true,
  "data": {
    "taskId": "task_789_20250729143000",
    "status": "completed",
    "result": {
      "text": "Artificial intelligence (AI) is a technology that enables computers to perform tasks that typically require human intelligence...",
      "usage": {
        "tokens": 342,
        "time": 2.3
      }
    },
    "executionTime": 2350,
    "startedAt": "2025-07-29T14:30:00Z",
    "completedAt": "2025-07-29T14:30:02.350Z",
    "metadata": {
      "model": "llama3.1:8b",
      "provider": "ollama",
      "version": "1.0"
    }
  }
}
```

### Get Task Status

Check the status of a running or completed task.

**Endpoint**: `GET /api/v1/tasks/{taskId}`

**Response**:
```json
{
  "success": true,
  "data": {
    "taskId": "task_789_20250729143000",
    "status": "running",
    "progress": 0.65,
    "estimatedTimeRemaining": 1200,
    "startedAt": "2025-07-29T14:30:00Z",
    "parameters": {
      "task": "process_document",
      "resourceId": "unstructured-io"
    },
    "logs": [
      {
        "timestamp": "2025-07-29T14:30:00Z",
        "level": "info",
        "message": "Starting document processing"
      },
      {
        "timestamp": "2025-07-29T14:30:01Z",
        "level": "info", 
        "message": "Extracted 15 pages of text"
      }
    ]
  }
}
```

### Cancel Task

Cancel a running task.

**Endpoint**: `POST /api/v1/tasks/{taskId}/cancel`

**Response**:
```json
{
  "success": true,
  "data": {
    "taskId": "task_789_20250729143000",
    "cancelled": true,
    "cancelledAt": "2025-07-29T14:30:05Z",
    "reason": "User requested cancellation"
  }
}
```

---

## Workflow Management

### List Workflows

Get available automation workflows.

**Endpoint**: `GET /api/v1/workflows`

**Response**:
```json
{
  "success": true,
  "data": {
    "workflows": [
      {
        "id": "doc_processing_pipeline",
        "name": "Document Processing Pipeline",
        "description": "Extract, analyze, and index documents using AI",
        "type": "node-red",
        "status": "active",
        "triggers": ["file_upload", "webhook", "schedule"],
        "resources": ["unstructured-io", "ollama", "minio"],
        "lastRun": "2025-07-29T12:00:00Z",
        "successRate": 0.94
      }
    ]
  }
}
```

### Execute Workflow

Trigger a workflow execution.

**Endpoint**: `POST /api/v1/workflows/{workflowId}/execute`

**Request Body**:
```json
{
  "inputs": {
    "document_url": "https://example.com/document.pdf",
    "analysis_type": "comprehensive",
    "output_format": "json"
  },
  "priority": "high"
}
```

---

## Error Handling

All API endpoints return consistent error responses:

```json
{
  "success": false,
  "error": {
    "code": "RESOURCE_NOT_FOUND",
    "message": "Resource with ID 'invalid_id' not found",
    "details": {
      "resourceId": "invalid_id",
      "availableResources": ["ollama", "node-red", "whisper"]
    }
  },
  "timestamp": "2025-07-29T14:30:00Z",
  "requestId": "req_123456"
}
```

### Common Error Codes

| Code | Description |
|------|-------------|
| `RESOURCE_NOT_FOUND` | Requested resource does not exist |
| `RESOURCE_UNAVAILABLE` | Resource is not running or accessible |
| `TASK_EXECUTION_FAILED` | Task execution encountered an error |
| `VALIDATION_ERROR` | Request validation failed |
| `AUTHENTICATION_REQUIRED` | Valid authentication token required |
| `AUTHORIZATION_DENIED` | Insufficient permissions |
| `RATE_LIMIT_EXCEEDED` | Too many requests |
| `TIMEOUT` | Request or operation timed out |
| `INTERNAL_ERROR` | Internal server error |

---

## Rate Limiting

API requests are rate limited:
- **Default**: 100 requests per minute per user
- **Burst**: Up to 20 requests in 10 seconds
- **Headers**: Rate limit info in response headers

```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1625097600
```

---

## WebSocket API

Real-time updates available via WebSocket connection:

```javascript
const ws = new WebSocket('ws://localhost:3000/api/v1/ws');

// Subscribe to resource health updates
ws.send(JSON.stringify({
  action: 'subscribe',
  topic: 'resource.health',
  resourceId: 'ollama'
}));

// Receive real-time updates
ws.onmessage = (event) => {
  const update = JSON.parse(event.data);
  console.log('Health update:', update);
};
```

---

## SDK and Client Libraries

- **JavaScript/TypeScript**: `@vrooli/api-client`
- **Python**: `vrooli-python-client`
- **Go**: `github.com/vrooli/go-client`

Example usage:
```javascript
import { VrooliApiClient } from '@vrooli/api-client';

const client = new VrooliApiClient({
  baseUrl: 'http://localhost:3000/api',
  apiKey: 'your-api-key'
});

const resources = await client.resources.discover();
```

---

## Testing

The API includes comprehensive test fixtures and examples in:
- `/scripts/resources/tests/fixtures/`
- Postman collection: `vrooli-api.postman_collection.json`
- OpenAPI spec: `/api/v1/openapi.json`

For detailed testing instructions, see the [Testing Guide](../testing/README.md).
