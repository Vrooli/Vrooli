# Error Semantics & Recovery Paths

This document describes the error handling design for the Lifestyle Dashboard scenario.

## Last Updated
2026-03-10

## Design Goals

1. **Categorized errors**: Every error belongs to one of 5 categories with distinct recovery paths
2. **Machine-readable codes**: Error codes enable programmatic handling
3. **Human-friendly messages**: User-facing messages without technical jargon
4. **Recovery hints**: Guidance on how to fix or work around the error
5. **Observable failures**: All errors are logged with consistent format

## Error Categories

| Category | HTTP | Description | Recovery Path |
|----------|------|-------------|---------------|
| `validation` | 400 | User input is invalid | Fix input and retry |
| `not_found` | 404 | Requested resource doesn't exist | Verify resource ID exists |
| `conflict` | 409 | Resource state prevents operation | Resolve conflict first |
| `internal` | 500 | Server error, cause unknown | Retry after brief wait |
| `unavailable` | 503 | External dependency down | Check scenario status |

### Category Selection Criteria

When categorizing an error, apply these criteria:

1. **Can the user fix it?** → `validation`
2. **Does the resource not exist?** → `not_found`
3. **Is it a state conflict?** → `conflict`
4. **Is a dependency unreachable?** → `unavailable`
5. **Everything else** → `internal`

## Error Codes

### Validation Errors
| Code | Description |
|------|-------------|
| `INVALID_JSON` | Request body is not valid JSON |
| `MISSING_FIELD` | Required field is missing |
| `INVALID_FIELD` | Field value is invalid |
| `INVALID_TIME_RANGE` | Time range parameters are invalid |

### Not Found Errors
| Code | Description |
|------|-------------|
| `EVENT_NOT_FOUND` | Event with given ID doesn't exist |
| `DOMAIN_NOT_FOUND` | Domain with given name doesn't exist |

### Internal Errors
| Code | Description |
|------|-------------|
| `DATABASE_ERROR` | Database operation failed |
| `HEALTH_CHECK_ERROR` | Health check operation failed |

### Unavailable Errors
| Code | Description |
|------|-------------|
| `DEPENDENCY_UNAVAILABLE` | Required dependency is not accessible |

## Response Format

```json
{
  "error": true,
  "category": "validation",
  "code": "MISSING_FIELD",
  "message": "Field 'domain' is required",
  "details": {
    "field": "domain"
  },
  "recovery": "Check the request body and fix validation errors"
}
```

### Fields

| Field | Type | Description |
|-------|------|-------------|
| `error` | boolean | Always `true` for error responses |
| `category` | string | Error category for recovery path |
| `code` | string | Machine-readable error code |
| `message` | string | Human-readable description |
| `details` | object | Additional context (optional) |
| `recovery` | string | Recovery guidance (optional) |

## Recovery Hints

| Hint | When Used |
|------|-----------|
| "Check the request body and fix validation errors" | Validation errors |
| "Verify the resource ID exists" | Not found errors |
| "Wait a moment and retry the request" | Internal errors |
| "Ensure the scenario is running with 'vrooli scenario status lifestyle-dashboard'" | Unavailable errors |

## Implementation

### API (Go)

Errors are defined in `api/errors/errors.go`:

```go
// Create a validation error
err := errors.NewValidationError(errors.CodeMissingField, "Field 'domain' is required")
err = err.WithDetails("field", "domain")

// Create a not found error
err := errors.NewNotFoundError(errors.CodeEventNotFound, "event", eventID)

// Create an internal error (logs detail, returns generic message)
log.Printf("[ERROR] CreateEvent: database error: %v", dbErr)
err := errors.NewInternalError(errors.CodeDatabaseError, "Failed to create event. Please try again.")
```

Write errors using `WriteAPIError`:

```go
func (h *Handler) CreateEvent(w http.ResponseWriter, r *http.Request) {
    if req.Domain == "" {
        WriteAPIError(w, errors.ErrMissingDomain)
        return
    }
    // ...
}
```

### UI (TypeScript)

The UI uses `APIError` class to parse structured errors:

```typescript
import { APIError } from "../lib/api";

try {
  await fetchEvent(id);
} catch (e) {
  if (e instanceof APIError) {
    console.log(e.category);  // "not_found"
    console.log(e.code);      // "EVENT_NOT_FOUND"
    console.log(e.isRetryable); // false
    console.log(e.recovery);  // "Verify the resource ID exists"
  }
}
```

The `ErrorAlert` component renders appropriate UI based on error category:

```tsx
<ErrorAlert
  error={error}
  onRetry={() => refetch()}
  onBack={() => navigate(-1)}
/>
```

## Logging

All errors are logged with consistent format:

```
[ERROR] <Handler>(<params>): <type>: <details>
[WARN] <Handler>(<params>): <type>: <details>
```

Examples:
```
[ERROR] CreateEvent: database error: SQLITE_BUSY: database is locked
[ERROR] GetEvent(abc-123): database error: connection reset
[WARN] GetDomainHealth(sleep-tracker): health check failed: timeout
```

## Failure Modes

### Critical Flows

| Flow | Failure Mode | Error Category | Impact |
|------|--------------|----------------|--------|
| Create Event | DB write fails | `internal` | Event not persisted |
| Create Event | Invalid JSON | `validation` | Request rejected |
| Query Events | DB read fails | `internal` | No events returned |
| Get Domain | Not exists | `not_found` | 404 response |
| Domain Health | Target timeout | N/A | Returns `unhealthy` status |

### Graceful Degradation

1. **Health checks**: Domain health checks that timeout return `unhealthy` status instead of failing the request
2. **Query limits**: MaxEventLimit (1000) prevents runaway queries
3. **Timeline limits**: MaxTimelineDays (365) caps historical queries
4. **Retry-after**: Internal errors suggest retry, giving transient issues time to resolve

## Adding New Error Codes

1. Add the code constant to `api/errors/errors.go`
2. Document it in this file
3. Update UI error handling if new category needed
4. Add tests for the new error path

## Related Documentation

- [SEAMS.md](SEAMS.md) - Observability surface section
- [configuration.md](../reference/configuration.md) - Tunable limits and timeouts
