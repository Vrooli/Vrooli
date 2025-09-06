# Scenario Generator V1 - API Documentation

## Base URL
```
http://localhost:8080
```

## Authentication
Currently no authentication required (to be implemented in future versions)

## Endpoints

### Health Check

#### `GET /health`
Check API and service health status.

**Response:**
```json
{
  "status": "healthy",
  "timestamp": 1234567890,
  "services": {
    "database": "healthy",
    "pipeline": "active",
    "vrooli_resource": "claude"
  }
}
```

---

### Scenarios

#### `GET /api/scenarios`
Retrieve all scenarios (limited to 50 most recent).

**Response:**
```json
[
  {
    "id": "uuid",
    "name": "Customer Support Dashboard",
    "description": "AI-powered support system",
    "prompt": "Create a customer support dashboard...",
    "status": "completed",
    "complexity": "intermediate",
    "category": "customer-service",
    "estimated_revenue": 25000,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
]
```

#### `POST /api/scenarios`
Create a new scenario.

**Request Body:**
```json
{
  "name": "New Scenario",
  "description": "Description of the scenario",
  "prompt": "Detailed prompt for generation",
  "complexity": "intermediate",
  "category": "business-tool"
}
```

**Response:** `201 Created`
```json
{
  "id": "new-uuid",
  "name": "New Scenario",
  "status": "requested",
  "created_at": "2024-01-01T00:00:00Z"
}
```

#### `GET /api/scenarios/{id}`
Get a specific scenario by ID.

**Response:**
```json
{
  "id": "uuid",
  "name": "Scenario Name",
  "description": "Description",
  "prompt": "Full prompt",
  "files_generated": {},
  "resources_used": [],
  "status": "completed",
  "complexity": "intermediate",
  "category": "business-tool",
  "estimated_revenue": 25000,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z",
  "completed_at": "2024-01-01T01:00:00Z"
}
```

#### `PUT /api/scenarios/{id}` ✨ **NEW**
Update an existing scenario's metadata.

**Request Body (partial updates supported):**
```json
{
  "name": "Updated Name",
  "description": "Updated description",
  "complexity": "advanced",
  "category": "ai-automation",
  "estimated_revenue": 35000,
  "status": "completed",
  "generation_error": "Error message if any"
}
```

**Allowed Fields:**
- `name` - Scenario name
- `description` - Scenario description
- `prompt` - Generation prompt
- `complexity` - simple/intermediate/advanced
- `category` - Category classification
- `estimated_revenue` - Revenue estimate in dollars
- `status` - requested/generating/completed/failed
- `generation_error` - Error message
- `notes` - Additional notes

**Response:**
```json
{
  "id": "uuid",
  "name": "Updated Name",
  "description": "Updated description",
  "status": "completed",
  "updated_at": "2024-01-01T02:00:00Z"
}
```

**Error Responses:**
- `400 Bad Request` - No valid fields to update
- `404 Not Found` - Scenario doesn't exist
- `500 Internal Server Error` - Database error

#### `DELETE /api/scenarios/{id}` ✨ **NEW**
Delete a scenario and all associated data.

**Response:** `204 No Content` (empty body)

**Error Responses:**
- `404 Not Found` - Scenario doesn't exist
- `500 Internal Server Error` - Database error

**Notes:**
- Deletion is permanent and cascades to generation_logs
- Uses database transaction for data integrity
- Logs deletion for audit trail

---

### Generation

#### `POST /api/generate`
Start generating a new scenario asynchronously.

**Request Body:**
```json
{
  "name": "Scenario Name",
  "description": "Description",
  "prompt": "Detailed generation prompt",
  "complexity": "intermediate",
  "category": "business-tool",
  "resources": ["postgres", "ollama"]
}
```

**Response:** `202 Accepted`
```json
{
  "scenario_id": "uuid",
  "status": "generating",
  "prompt": "Your prompt",
  "estimated_time": "3-8 minutes",
  "message": "Scenario generation started using AI pipeline"
}
```

#### `GET /api/generate/status/{id}`
Get generation status for a scenario.

**Response:**
```json
{
  "scenario_id": "uuid",
  "status": "implementing",
  "progress": 50,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:30:00Z",
  "estimated_revenue": 25000
}
```

**Status Values:**
- `requested` - Initial state (0% progress)
- `planning` - Creating architecture (25% progress)
- `implementing` - Generating code (50% progress)
- `validating` - Testing scenario (75% progress)
- `completed` - Successfully done (100% progress)
- `failed` - Generation failed (0% progress)

---

### Templates

#### `GET /api/templates`
Get all active scenario templates.

**Response:**
```json
[
  {
    "id": "uuid",
    "name": "Simple SaaS Dashboard",
    "description": "Basic business dashboard",
    "category": "business-tool",
    "prompt_template": "Create a SaaS dashboard...",
    "resources_suggested": ["postgres", "windmill"],
    "complexity_level": "simple",
    "estimated_revenue_min": 10000,
    "estimated_revenue_max": 20000,
    "usage_count": 15,
    "success_rate": 85.5
  }
]
```

#### `GET /api/templates/{id}`
Get a specific template by ID.

---

### Search

#### `GET /api/search/scenarios?q={query}`
Search scenarios by name, description, prompt, or category.

**Query Parameters:**
- `q` - Search term (required)

**Response:**
```json
[
  {
    "id": "uuid",
    "name": "Matching Scenario",
    "description": "Description",
    "category": "business-tool",
    "status": "completed"
  }
]
```

#### `GET /api/featured`
Get featured high-value scenarios (revenue > $20,000).

**Response:** Same format as `/api/scenarios`

---

### Logs

#### `GET /api/scenarios/{id}/logs`
Get generation logs for a specific scenario.

**Response:**
```json
[
  {
    "id": "log-uuid",
    "scenario_id": "scenario-uuid",
    "step": "planning_iteration_1",
    "prompt": "Generate architecture for...",
    "response": "Here's the architecture...",
    "success": true,
    "started_at": "2024-01-01T00:00:00Z",
    "completed_at": "2024-01-01T00:05:00Z",
    "duration_seconds": 300
  }
]
```

---

### Backlog Management

#### `GET /api/backlog`
Get all backlog items across all states.

**Response:**
```json
{
  "pending": [
    {
      "id": "invoice-saas",
      "name": "Invoice Management SaaS",
      "description": "Freelance invoice management",
      "priority": "high",
      "estimated_revenue": 25000,
      "filename": "001-invoice-management-saas.yaml",
      "status": "pending"
    }
  ],
  "in_progress": [],
  "completed": [],
  "failed": []
}
```

#### `POST /api/backlog`
Create a new backlog item.

**Request Body:**
```json
{
  "name": "New Backlog Item",
  "description": "Description",
  "prompt": "Detailed prompt",
  "complexity": "intermediate",
  "category": "business-tool",
  "priority": "medium",
  "estimated_revenue": 25000,
  "tags": ["saas", "automation"]
}
```

#### `GET /api/backlog/{id}`
Get a specific backlog item.

#### `PUT /api/backlog/{id}`
Update a backlog item.

#### `DELETE /api/backlog/{id}`
Delete a backlog item.

#### `POST /api/backlog/{id}/generate`
Move backlog item to generation queue and start processing.

**Response:**
```json
{
  "message": "Generation started via AI pipeline",
  "scenario_id": "uuid",
  "backlog_item": {
    "id": "item-id",
    "name": "Item Name",
    "status": "in_progress"
  }
}
```

#### `POST /api/backlog/{id}/move`
Move backlog item between states.

**Request Body:**
```json
{
  "to_state": "completed"
}
```

**Valid States:**
- `pending` - Waiting in queue
- `in_progress` - Currently generating
- `completed` - Successfully generated
- `failed` - Generation failed

---

## Error Handling

All endpoints follow standard HTTP status codes:

- `200 OK` - Successful GET request
- `201 Created` - Successful POST creation
- `202 Accepted` - Async operation started
- `204 No Content` - Successful DELETE
- `400 Bad Request` - Invalid request data
- `404 Not Found` - Resource not found
- `500 Internal Server Error` - Server error

Error responses include descriptive messages:
```json
{
  "error": "Scenario not found"
}
```

## Rate Limiting
Currently no rate limiting (to be implemented)

## Versioning
API version: 1.0.0
Future versions will use URL versioning (e.g., `/api/v2/`)

## Testing

Use the provided test scripts:
```bash
# Test all endpoints
./test.sh

# Test CRUD operations
./test-crud-endpoints.sh

# Test specific functionality
./test.sh api
./test.sh database
./test.sh create
```

## Example Usage

### Create and Update a Scenario
```bash
# Create scenario
RESPONSE=$(curl -X POST http://localhost:8080/api/scenarios \
  -H "Content-Type: application/json" \
  -d '{"name":"Test","description":"Test scenario","prompt":"Create a test"}')

# Extract ID
ID=$(echo $RESPONSE | jq -r '.id')

# Update scenario
curl -X PUT http://localhost:8080/api/scenarios/$ID \
  -H "Content-Type: application/json" \
  -d '{"name":"Updated Test","complexity":"advanced"}'

# Delete scenario
curl -X DELETE http://localhost:8080/api/scenarios/$ID
```

### Generate from Backlog
```bash
# Add to backlog
curl -X POST http://localhost:8080/api/backlog \
  -H "Content-Type: application/json" \
  -d '{"name":"New Feature","description":"AI feature","prompt":"Build an AI tool"}'

# Start generation
curl -X POST http://localhost:8080/api/backlog/new-feature/generate
```