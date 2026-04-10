# Error Semantics

This document describes the error handling architecture for the reference-react-vite scenario, following the error-semantics-recovery-path-design steer skill patterns.

## Overview

The scenario implements a structured error system with:
- **Typed error categories** for machine-readable classification
- **Recovery hints** guiding users/agents on next steps
- **Retryability flags** distinguishing transient from permanent failures
- **Request ID correlation** for debugging across logs and responses

## Error Categories

### Category Decision Criteria

Each error category has a distinct recovery path. Categories are mutually exclusive and user-recognizable:

| Category | HTTP Status | Recovery Path | Retryable |
|----------|-------------|---------------|-----------|
| `BAD_REQUEST` | 400 | Fix request format/syntax | No |
| `VALIDATION_ERROR` | 422 | Fix input values per field feedback | No |
| `NOT_FOUND` | 404 | Verify resource ID, list available resources | No |
| `INTERNAL_ERROR` | 500 | Retry with backoff | Yes |
| `CONFLICT` | 409 | Refresh resource, resolve state conflict | No |
| `UNAUTHORIZED` | 401 | Login or refresh session | No |

### Recovery Hint Examples

```json
{
  "code": "VALIDATION_ERROR",
  "message": "task title is required",
  "details": {"title": "required"},
  "recovery": "Review the field values in 'details' and correct invalid inputs.",
  "retryable": false,
  "request_id": "abc123"
}
```

```json
{
  "code": "INTERNAL_ERROR",
  "message": "failed to create task",
  "recovery": "This is a temporary server issue. Please retry after a short delay.",
  "retryable": true,
  "request_id": "xyz789"
}
```

## Error Flow Architecture

### Request → Handler → Domain → Repository

```
┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐
│ Client  │───→│ Handler │───→│ Domain  │───→│  Repo   │
└─────────┘    └─────────┘    └─────────┘    └─────────┘
     ▲              │              │              │
     │              │              │              │
     │       writeError()    validation      ErrNotFound
     │       + recovery      errors           (typed)
     │       + logging                           │
     │              │              │              │
     └──────────────┴──────────────┴──────────────┘
                    Error propagation
```

### Error Origins by Layer

| Layer | Error Types | Handling Pattern |
|-------|-------------|------------------|
| **Handler** | JSON decode failures, parameter parsing | `writeBadRequest()` |
| **Domain** | Validation failures, business rule violations | `writeValidationError()` |
| **Repository** | Not found, database errors | `IsNotFound()` check, then appropriate response |

## Repository Error Types

Location: `api/repository/repository.go`

The repository layer exposes typed sentinel errors for semantic error handling:

```go
// ErrNotFound is returned when a requested resource does not exist.
var ErrNotFound = errors.New("resource not found")

// IsNotFound checks if an error indicates a missing resource.
func IsNotFound(err error) bool {
    return errors.Is(err, ErrNotFound)
}
```

### Usage Pattern

```go
// In handler
if err := h.repo.Delete(ctx, id); err != nil {
    if repository.IsNotFound(err) {
        writeNotFound(w, r, "task")  // 404
        return
    }
    writeInternalError(w, r, "failed to delete task")  // 500
    return
}
```

This replaces fragile string matching:
```go
// OLD (fragile)
if err.Error() == "task not found" { ... }

// NEW (type-safe)
if repository.IsNotFound(err) { ... }
```

## Critical Flows & Failure Modes

### Task CRUD Operations

| Operation | Primary Failures | Recovery Path |
|-----------|-----------------|---------------|
| Create | Validation, DB write | Fix input, retry |
| Read | Not found, DB read | Verify ID |
| Update | Not found, validation | Refresh & retry |
| Delete | Not found, DB constraint | Verify ID, check dependencies |

### Project CRUD Operations

| Operation | Primary Failures | Recovery Path |
|-----------|-----------------|---------------|
| Create | Validation (name, color) | Fix input |
| Read | Not found | Verify ID |
| Update | Not found, validation | Refresh & retry |
| Delete | Not found, has tasks | Remove tasks first |

### Note CRUD Operations

| Operation | Primary Failures | Recovery Path |
|-----------|-----------------|---------------|
| Create | Task not found, validation | Verify task ID |
| Read | Not found | Verify note ID |
| Update | Not found, validation | Refresh & retry |
| Delete | Not found | Verify note ID |

## Observability

### Structured Error Logging

All errors are logged with structured context:
```
[ERROR] request_id=abc123 code=VALIDATION_ERROR status=422 path=/api/v1/tasks message="task title is required"
```

Log fields:
- `request_id`: Correlation ID for tracing
- `code`: Machine-readable error category
- `status`: HTTP status code
- `path`: Request path
- `message`: Human-readable error description

### Request ID Propagation

- Client can provide `X-Request-ID` header
- If absent, a UUID is generated
- ID is included in error responses and logs

## Graceful Degradation Patterns

### Transient vs Permanent Failures

| Failure Type | Behavior | Client Action |
|--------------|----------|---------------|
| Transient (DB timeout) | Return 500 with `retryable: true` | Retry with exponential backoff |
| Permanent (validation) | Return 422 with `retryable: false` | Fix input before retry |

### Safe Failure States

- **Validation failures**: No state change occurs; input is rejected before any writes
- **Not found**: Read-only check; no side effects
- **Create failures**: Transaction is not committed; no partial state

## Integration with UI

The error response format is designed for UI consumption:

```typescript
interface APIError {
  code: string;        // Machine-readable category
  message: string;     // User-facing message
  details?: Record<string, unknown>;  // Field-level errors
  recovery: string;    // Suggested next action
  retryable: boolean;  // Can retry without changes?
  request_id: string;  // For support/debugging
}
```

### UI Error Handling Example

```typescript
async function handleAPIError(error: APIError) {
  if (error.retryable) {
    // Show retry button, implement backoff
    showRetryableError(error.message, error.recovery);
  } else if (error.code === 'VALIDATION_ERROR') {
    // Highlight invalid fields
    showFieldErrors(error.details);
  } else if (error.code === 'NOT_FOUND') {
    // Navigate away or show missing resource message
    showNotFoundError(error.recovery);
  } else {
    // Generic error with support info
    showGenericError(error.message, error.request_id);
  }
}
```

## Related Documentation

- [DOC: docs/internal/SEAMS.md] - Architectural seams
- [DOC: docs/reference/api-endpoints.md] - API reference
- [DOC: docs/reference/configuration.md] - Configuration options
- [CODE: api/handlers/errors.go] - Error response utilities
- [CODE: api/repository/repository.go] - Repository error types

## Last Updated

2026-03-11 - Initial error semantics documentation with typed errors and recovery paths
