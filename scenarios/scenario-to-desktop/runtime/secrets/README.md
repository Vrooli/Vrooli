# secrets - Secret Management

The `secrets` package handles secure storage, retrieval, and injection of secrets into service environments.

## Overview

Desktop applications often need API keys, database passwords, and other sensitive values. This package manages the full secret lifecycle: prompting users, persisting to disk, and injecting into service processes via environment variables or files.

## Key Types

| Type | Purpose |
|------|---------|
| `Store` | Interface for secret storage operations |
| `Manager` | Production implementation of `Store` |
| `Injector` | Injects secrets into service environments |

## Store Interface

```go
type Store interface {
    // Load reads secrets from persistent storage
    Load() (map[string]string, error)

    // Persist saves secrets to persistent storage
    Persist(secrets map[string]string) error

    // Get returns current in-memory secrets
    Get() map[string]string

    // Set updates in-memory secrets
    Set(secrets map[string]string)

    // Merge combines new secrets with existing
    Merge(new map[string]string) map[string]string

    // MissingRequired returns IDs of missing required secrets
    MissingRequired() []string

    // MissingRequiredFrom checks a map for missing required secrets
    MissingRequiredFrom(secrets map[string]string) []string

    // FindSecret looks up a secret definition by ID
    FindSecret(id string) *manifest.Secret
}
```

## Usage

```go
// Create manager
secretsPath := filepath.Join(appData, "secrets.json")
manager := secrets.NewManager(manifest, fs, secretsPath)

// Load persisted secrets
loaded, err := manager.Load()
manager.Set(loaded)

// Check for missing required secrets
missing := manager.MissingRequired()
if len(missing) > 0 {
    // Prompt user for secrets via UI
}

// Update secrets
merged := manager.Merge(userInput)
manager.Persist(merged)
manager.Set(merged)

// Inject into service environment
injector := secrets.NewInjector(manager, fs, appData)
err := injector.Apply(envMap, service)
```

## Secret Classes

| Class | Description |
|-------|-------------|
| `user_prompt` | Prompted from user via UI |
| `generated` | Auto-generated (e.g., random tokens) |
| `derived` | Computed from other values |

## Target Types

Secrets can be injected as:

| Type | Description | Example |
|------|-------------|---------|
| `env` | Environment variable | `API_KEY=secret123` |
| `file` | Written to file | `/app/data/.credentials` |

## Storage Format

Secrets are stored in `secrets.json`:

```json
{
  "API_KEY": "sk-xxx...",
  "DB_PASSWORD": "hunter2"
}
```

File permissions are set to `0600` (owner read/write only).

## Security Considerations

- Secrets file is stored in user config directory
- File permissions restrict access to current user
- Secrets are never logged or included in telemetry
- Memory is not explicitly zeroed (Go limitation)

## Dependencies

- **Depends on**: `manifest`, `infra` (FileSystem)
- **Depended on by**: `bundleruntime`
