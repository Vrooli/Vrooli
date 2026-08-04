# secrets - Secret Management

The `secrets` package handles secure storage, retrieval, and injection of secrets into service environments.

## Overview

Desktop applications often need API keys, database passwords, and other sensitive values. This package manages the full secret lifecycle: prompting users, persisting through Vrooli's native OS credential authority, and injecting values into service processes via environment variables or explicitly declared files.

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
// Create the production manager. It fails closed when the native credential
// authority is unavailable.
manager, err := secrets.NewNativeManager(manifest)
if err != nil {
    return err
}

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
| `file` | Materialized on ephemeral storage, removed once services start | `$XDG_RUNTIME_DIR/vrooli/bundle-secrets/cert.pem` |

## Storage

Production secrets are stored only by the Vrooli credential authority, which
uses the native OS credential store, or the encrypted file store on a host with
no native one. The runtime never reads or writes a `secrets.json` file during
normal operation.

A secret declaring `logical_id` and `field` resolves to the **same durable name
every other deployment tier uses**, so a credential provisioned once during
onboarding is the credential this bundle reads. A secret that declares neither
falls back to a bundle-private namespace derived from the app name — it still
works, but no Tier 1 install will find it and it is a separate value to back up.

A `file` target is materialized on ephemeral storage (`XDG_RUNTIME_DIR`, else
the system temp dir) and removed once the services that needed it have started.
Where the host has no ephemeral location the bundle refuses rather than writing
a durable plaintext credential an operator cannot see.

`MigrateLegacyFile` is the sole compatibility path for an older JSON file. It
requires an explicit source path, imports only secret IDs declared in the
desktop manifest, verifies the native write, and leaves the source intact
unless the caller explicitly requests deletion after a successful import.

## Security Considerations

- Production values are never persisted in the desktop app data directory, and
  a `file` target never lands on durable storage
- Native-store unavailability is a startup/configuration error; there is no
  plaintext fallback
- Secrets are never logged or included in telemetry
- Memory is not explicitly zeroed (Go limitation)

## Dependencies

- **Depends on**: `manifest`, `infra` (FileSystem)
- **Depended on by**: `bundleruntime`
