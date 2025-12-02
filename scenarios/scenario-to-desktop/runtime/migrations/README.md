# migrations - Migration State Tracking

The `migrations` package tracks which schema migrations have been applied to each service.

## Overview

Services may need to run migrations on startup (database schema changes, data transformations, etc.). This package tracks which migrations have been applied, enabling the runtime to skip already-run migrations and detect upgrade vs fresh install scenarios.

## Key Types

| Type | Purpose |
|------|---------|
| `State` | Tracks applied migrations per service |
| `Tracker` | Loads and persists migration state |

## State Structure

```go
type State struct {
    AppVersion string              // Current app version
    Applied    map[string][]string // service_id → [migration_versions]
}
```

## Helper Functions

| Function | Purpose |
|----------|---------|
| `NewState()` | Create empty state |
| `NewTracker(path, fs)` | Create tracker for load/persist |
| `Phase(state, version)` | Determine install phase |
| `ShouldRun(migration, phase)` | Check if migration should run |
| `BuildAppliedSet(versions)` | Convert slice to set for lookup |
| `MarkApplied(state, svcID, version)` | Record migration as applied |

## Install Phases

| Phase | Condition |
|-------|-----------|
| `first_install` | No previous app version recorded |
| `upgrade` | Previous version differs from current |
| `same_version` | Version unchanged |

## Migration Run Conditions

| `run_on` | When it runs |
|----------|--------------|
| `always` | Every startup |
| `first_install` | Only on fresh install |
| `upgrade` | Only when upgrading |

## Usage

```go
// Load state
tracker := migrations.NewTracker(migrationsPath, fs)
state, err := tracker.Load()

// Determine phase
phase := migrations.Phase(state, currentVersion)

// Check if migration should run
if migrations.ShouldRun(migration, phase) {
    // Execute migration
    // ...
    migrations.MarkApplied(&state, serviceID, migration.Version)
}

// Persist state
state.AppVersion = currentVersion
tracker.Persist(state)
```

## Storage Format

State is stored in `migrations.json`:

```json
{
  "app_version": "1.2.0",
  "applied": {
    "api": ["v1", "v2", "v3"],
    "db": ["v1"]
  }
}
```

## Manifest Example

```json
{
  "services": [{
    "id": "api",
    "migrations": [
      {"version": "v1", "command": ["./migrate", "up"], "run_on": "first_install"},
      {"version": "v2", "command": ["./migrate", "up"], "run_on": "upgrade"}
    ]
  }]
}
```

## Dependencies

- **Depends on**: `infra` (FileSystem)
- **Depended on by**: `bundleruntime`
