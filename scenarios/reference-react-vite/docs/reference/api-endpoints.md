# API Reference

REST API documentation for the reference-react-vite scenario.

**Base URL**: `http://localhost:15000/api/v1`

## Authentication

No authentication required. This is a reference implementation.

## Common Response Formats

### Success Response
```json
{
  "data": { ... },
  "meta": {
    "request_id": "uuid",
    "timestamp": "2026-03-11T14:30:00Z"
  }
}
```

### List Response with Pagination
```json
{
  "data": [ ... ],
  "meta": {
    "total": 100,
    "page": 1,
    "limit": 20,
    "total_pages": 5
  }
}
```

### Error Response
[CODE: api/handlers/errors.go#APIError]

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Human-readable description",
    "details": { "field": "specific error" },
    "request_id": "uuid"
  }
}
```

## Health

### GET /health

Check API and dependency health.

**Response**: 200 OK
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "checks": {
    "database": {
      "status": "healthy",
      "latency_ms": 5
    }
  }
}
```

---

## Projects

[CODE: api/handlers/projects.go]

### GET /api/v1/projects

List all projects with optional filtering.

**Query Parameters**:
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | int | 1 | Page number |
| limit | int | 20 | Items per page (max 100) |
| status | string | - | Filter by status (active, archived, completed) |

**Response**: 200 OK
```json
{
  "data": [
    {
      "id": "uuid",
      "name": "Project Name",
      "description": "Optional description",
      "color": "#FF5733",
      "status": "active",
      "created_at": "2026-03-11T14:30:00Z",
      "updated_at": "2026-03-11T14:30:00Z"
    }
  ],
  "meta": {
    "total": 10,
    "page": 1,
    "limit": 20,
    "total_pages": 1
  }
}
```

### POST /api/v1/projects

Create a new project.

**Request Body**:
```json
{
  "name": "Project Name",
  "description": "Optional description",
  "color": "#FF5733"
}
```

**Validation Rules**:
[CODE: api/domain/projects/project.go#Validate]
- `name`: Required, 1-200 characters
- `color`: Optional, must match `#[0-9A-Fa-f]{6}` format

**Response**: 201 Created
```json
{
  "data": {
    "id": "uuid",
    "name": "Project Name",
    "description": "Optional description",
    "color": "#FF5733",
    "status": "active",
    "created_at": "2026-03-11T14:30:00Z",
    "updated_at": "2026-03-11T14:30:00Z"
  }
}
```

### GET /api/v1/projects/{id}

Get a single project by ID.

**Response**: 200 OK

**Errors**:
- 404: Project not found

### PATCH /api/v1/projects/{id}

Update a project.

**Request Body**: (all fields optional)
```json
{
  "name": "Updated Name",
  "description": "Updated description",
  "color": "#00FF00",
  "status": "archived"
}
```

**Response**: 200 OK

### DELETE /api/v1/projects/{id}

Delete a project.

**Response**: 204 No Content

---

## Tasks

[CODE: api/handlers/tasks.go]

### GET /api/v1/tasks

List all tasks with optional filtering.

**Query Parameters**:
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | int | 1 | Page number |
| limit | int | 20 | Items per page (max 100) |
| status | string | - | Filter by status |
| project_id | uuid | - | Filter by project |
| priority | int | - | Filter by priority (1-5) |

**Response**: 200 OK
```json
{
  "data": [
    {
      "id": "uuid",
      "title": "Task Title",
      "description": "Task description",
      "status": "pending",
      "priority": 3,
      "due_date": "2026-03-15T00:00:00Z",
      "project_id": "uuid",
      "created_at": "2026-03-11T14:30:00Z",
      "updated_at": "2026-03-11T14:30:00Z"
    }
  ],
  "meta": { ... }
}
```

### POST /api/v1/tasks

Create a new task.

**Request Body**:
```json
{
  "title": "Task Title",
  "description": "Optional description",
  "priority": 3,
  "due_date": "2026-03-15T00:00:00Z",
  "project_id": "uuid"
}
```

**Validation Rules**:
[CODE: api/domain/tasks/task.go#Validate]
- `title`: Required, 1-500 characters
- `priority`: Optional, 1-5 (default: 3)
- `status`: pending, in_progress, completed, blocked

**Response**: 201 Created

### GET /api/v1/tasks/{id}

Get a single task by ID.

### PATCH /api/v1/tasks/{id}

Update a task.

### DELETE /api/v1/tasks/{id}

Delete a task (cascades to notes).

---

## Notes

[CODE: api/handlers/notes.go]

### GET /api/v1/tasks/{task_id}/notes

List all notes for a task.

**Query Parameters**:
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | int | 1 | Page number |
| limit | int | 20 | Items per page |

**Response**: 200 OK
```json
{
  "data": [
    {
      "id": "uuid",
      "content": "Note content",
      "author": "user@example.com",
      "task_id": "uuid",
      "created_at": "2026-03-11T14:30:00Z",
      "updated_at": "2026-03-11T14:30:00Z"
    }
  ],
  "meta": { ... }
}
```

### POST /api/v1/tasks/{task_id}/notes

Add a note to a task.

**Request Body**:
```json
{
  "content": "Note content",
  "author": "optional@example.com"
}
```

**Validation Rules**:
[CODE: api/domain/notes/note.go#Validate]
- `content`: Required, 1-10000 characters
- `task_id`: Must reference existing task

**Response**: 201 Created

### GET /api/v1/notes/{id}

Get a single note by ID.

### PATCH /api/v1/notes/{id}

Update a note.

### DELETE /api/v1/notes/{id}

Delete a note.

---

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| VALIDATION_ERROR | 400 | Request body failed validation |
| NOT_FOUND | 404 | Resource not found |
| INVALID_REQUEST | 400 | Malformed request |
| INTERNAL_ERROR | 500 | Server error |

## Rate Limiting

Not implemented. This is a reference scenario for local development.

## Related

- [Data Model](data-model.md) - Database schema
- [Configuration](configuration.md) - Environment variables
