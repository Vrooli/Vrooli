# errors - Structured Error Types

The `errors` package provides domain-specific error types for better error handling and user feedback.

## Overview

Generic errors make debugging difficult. This package defines structured error types that capture context (which service, what operation, underlying cause) and enable programmatic error handling.

## Error Types

| Type | Fields | Use Case |
|------|--------|----------|
| `ServiceError` | `ServiceID`, `Op`, `Err` | Service lifecycle failures |
| `SecretError` | `SecretID`, `Reason` | Secret management issues |
| `PortError` | `ServiceID`, `PortName`, `Reason` | Port allocation failures |
| `AssetError` | `Path`, `Reason` | Asset verification failures |
| `MigrationError` | `ServiceID`, `Version`, `Err` | Migration execution failures |
| `HealthCheckError` | `ServiceID`, `CheckType`, `Err` | Health check failures |

## Sentinel Errors

| Error | Description |
|-------|-------------|
| `ErrSecretsNotReady` | Required secrets not yet provided |
| `ErrServiceNotFound` | Unknown service ID |
| `ErrPortNotAllocated` | Port not yet allocated |

## Usage

### Creating Errors

```go
// Service error
err := &errors.ServiceError{
    ServiceID: "api",
    Op:        "start",
    Err:       fmt.Errorf("binary not found"),
}

// Secret error
err := &errors.SecretError{
    SecretID: "API_KEY",
    Reason:   "required but not provided",
}
```

### Checking Errors

```go
// Check specific service
if errors.IsServiceError(err, "api") {
    // Handle API service error
}

// Check error category
if errors.IsSecretError(err) {
    // Prompt user for missing secrets
}

if errors.IsAssetError(err) {
    // Asset verification failed
}

// Unwrap for details
var svcErr *errors.ServiceError
if stderrors.As(err, &svcErr) {
    log.Printf("Service %s failed during %s: %v",
        svcErr.ServiceID, svcErr.Op, svcErr.Err)
}
```

### Error Messages

All error types implement `Error()` with informative messages:

```
service api: start: binary not found
secret API_KEY: required but not provided
port api.http: no available port in range
asset models/llm.bin: checksum mismatch
migration api/v2: exit code 1
health check api (http): connection refused
```

## Integration

The Supervisor uses these errors throughout:

```go
func (s *Supervisor) startService(ctx context.Context, svc manifest.Service) error {
    bin, ok := s.opts.Manifest.ResolveBinary(svc)
    if !ok {
        return &errors.ServiceError{
            ServiceID: svc.ID,
            Op:        "resolve_binary",
            Err:       fmt.Errorf("no binary for platform %s", manifest.CurrentPlatform()),
        }
    }
    // ...
}
```

## Dependencies

- **Depends on**: Standard library only
- **Depended on by**: `bundleruntime`
