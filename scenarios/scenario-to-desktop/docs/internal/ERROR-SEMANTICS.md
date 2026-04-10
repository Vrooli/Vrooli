# Error Semantics

## Last Updated
2026-04-06

## Error Type Hierarchy

All backend errors use `DomainError` from [CODE: api/shared/errors/errors.go].

### Base Error Codes

| Code | HTTP Status | Meaning |
|------|------------|---------|
| `INTERNAL_ERROR` | 500 | Unexpected server failure |
| `NOT_FOUND` | 404 | Resource does not exist |
| `BAD_REQUEST` | 400 | Malformed or invalid request |
| `UNAUTHORIZED` | 401 | Missing or invalid credentials |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `CONFLICT` | 409 | Resource state conflict |
| `TIMEOUT` | 408 | Operation exceeded time limit |
| `VALIDATION_ERROR` | 422 | Input failed validation rules |

### Domain-Specific Codes

**Bundle errors**: `BUNDLE_NOT_FOUND`, `BUNDLE_INVALID`, `BUNDLE_MANIFEST_ERROR`, `BUNDLE_COMPILE_ERROR`, `BUNDLE_RUNTIME_ERROR`, `BUNDLE_SECRETS_ERROR`

**Pipeline errors**: `CodePipelineFailed`, `CodePipelineTimeout`, `CodePipelineScenarioRequired`, `CodePipelineOrchestratorNotConfigured`

## Recovery Actions

Errors carry machine-readable recovery hints via `.WithRecovery(action, message)`:

| Recovery Action | When Used | User Guidance |
|----------------|-----------|---------------|
| `RecoveryFixInput` | Validation failures | Fix the input and retry |
| `RecoveryRetry` | Transient failures | Wait and try again |
| `RecoveryRetryWithBackoff` | Rate limits, resource contention | Exponential backoff |
| `RecoveryInstallDependency` | Missing tools (Wine, electron-builder) | Install prerequisite |
| `RecoveryWaitForResource` | Resource not ready | Wait for resource health |
| `RecoveryProvideCredentials` | Auth required | Provide credentials |
| `RecoveryContactSupport` | Unrecoverable | Escalate |
| `RecoveryNone` | Informational | No action possible |

## Frontend Error Handling

UI uses `ApiError` class and `throwIfNotOk()` helper in [CODE: ui/src/lib/api.ts]:
- All 60+ API functions use structured error handling
- 404 responses are non-errors for state/logs queries (expected "not found" condition)
- Error utilities in [CODE: ui/src/lib/error-utils.ts] provide `createErrorInfo()` for display

## Error Response Format

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "scenario_name is required",
    "recovery": "fix_input",
    "recovery_hint": "Provide a valid scenario name",
    "details": { "field": "scenario_name" }
  }
}
```
