# telemetry - Event Recording

The `telemetry` package records runtime events to a JSONL file for debugging, analytics, and crash reporting.

## Overview

Understanding runtime behavior requires visibility into what happened. This package records structured events (service starts, health check results, errors, etc.) to a local file that can be uploaded for analysis or reviewed locally for debugging.

## Key Types

| Type | Purpose |
|------|---------|
| `Recorder` | Interface for event recording |
| `FileRecorder` | Production implementation (writes to JSONL) |
| `Record` | Single telemetry event structure |

## Recorder Interface

```go
type Recorder interface {
    // Record writes an event with optional details
    Record(event string, details map[string]interface{}) error
}
```

## Record Structure

```go
type Record struct {
    Timestamp string                 `json:"ts"`      // ISO 8601 UTC
    Event     string                 `json:"event"`   // Event type
    Details   map[string]interface{} `json:"details"` // Event-specific data
}
```

## Standard Events

| Event | Description |
|-------|-------------|
| `runtime_start` | Supervisor started |
| `runtime_shutdown` | Supervisor shutting down |
| `runtime_error` | Fatal runtime error |
| `service_start` | Service process launched |
| `service_ready` | Service passed readiness check |
| `service_not_ready` | Service failed readiness check |
| `service_exit` | Service process exited |
| `service_blocked` | Service waiting on dependencies |
| `gpu_status` | GPU detection result |
| `secrets_missing` | Required secrets not provided |
| `secrets_updated` | Secrets submitted via API |
| `migration_start` | Migration execution started |
| `migration_applied` | Migration completed successfully |
| `migration_failed` | Migration failed |
| `asset_missing` | Required asset not found |
| `asset_checksum_mismatch` | Asset SHA256 mismatch |
| `asset_size_exceeded` | Asset larger than budget |

## Usage

```go
// Create recorder
recorder := telemetry.NewFileRecorder(telemetryPath, clock, fs)

// Record events
recorder.Record("runtime_start", nil)

recorder.Record("service_start", map[string]interface{}{
    "service_id": "api",
})

recorder.Record("runtime_error", map[string]interface{}{
    "error": err.Error(),
})
```

## Output Format (JSONL)

Each line is a complete JSON object:

```jsonl
{"ts":"2024-01-15T10:30:00Z","event":"runtime_start","details":{}}
{"ts":"2024-01-15T10:30:01Z","event":"service_start","details":{"service_id":"api"}}
{"ts":"2024-01-15T10:30:05Z","event":"service_ready","details":{"service_id":"api"}}
```

## File Location

Configured in manifest:

```json
{
  "telemetry": {
    "file": "runtime/telemetry.jsonl",
    "upload_url": "https://api.example.com/telemetry"
  }
}
```

Path is relative to app data directory.

## Privacy Considerations

- No secret values are recorded
- No user data beyond service status
- Local file only; upload is optional
- User can delete file at any time

## Dependencies

- **Depends on**: `infra` (Clock, FileSystem)
- **Depended on by**: `bundleruntime`, `assets`, `gpu`
