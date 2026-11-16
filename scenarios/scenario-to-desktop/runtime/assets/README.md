# assets - Asset Verification and Playwright Setup

The `assets` package verifies bundled assets and configures Playwright-based browser automation services.

## Overview

Desktop bundles include binary executables, data files, and other assets. This package ensures all required assets exist and are valid before services start. It also handles special setup for Playwright-based services that need browser binaries.

## Key Components

### Asset Verification

| Type | Purpose |
|------|---------|
| `Verifier` | Validates asset existence, size, and checksums |

### Playwright Setup

| Function | Purpose |
|----------|---------|
| `ServiceUsesPlaywright(svc)` | Detect if service needs Playwright |
| `ApplyPlaywrightConventions(cfg, svc, env)` | Configure Playwright environment |

## Asset Verification

Assets can be verified for:

| Check | Description |
|-------|-------------|
| Existence | File must exist at specified path |
| Size Budget | File size must be within tolerance |
| Checksum | SHA256 hash must match |

```go
verifier := assets.NewVerifier(bundlePath, fs, telemetry)
if err := verifier.EnsureAssets(service); err != nil {
    // Asset verification failed
}
```

## Manifest Example

```json
{
  "assets": [
    {
      "path": "models/llm.bin",
      "size_budget_bytes": 4294967296,
      "size_budget_slack_percent": 5,
      "sha256": "abc123..."
    }
  ]
}
```

## Playwright Conventions

Services using Playwright can declare a package-local browser closure with
`PLAYWRIGHT_BROWSERS_PATH`. The runtime resolves that path against the bundle
root, requires it to be a directory, and passes the absolute path to the
driver. This keeps the driver's pinned Playwright revision inside the bundle
instead of inheriting a developer's user cache. Missing declared closures fail
closed. An explicit `BROWSER_EXECUTABLE_PATH` may name one compatible binary;
the only runtime fallback is an explicit `ELECTRON_CHROMIUM_PATH` supplied by
the launcher.

### Environment Variables Set

| Variable | Description |
|----------|-------------|
| `PLAYWRIGHT_BROWSERS_PATH` | Path to browser binaries |
| `BROWSER_EXECUTABLE_PATH` | Exact browser executable override consumed by BAS |
| `PWDEBUG_PORT` | Debug protocol port (if allocated) |

Browser acquisition is outside this runtime helper. The browser closure must be
provisioned by the governed dependency/artifact owner before packaging. No
system-wide browser search or browser download occurs here.

## Usage

```go
// Asset verification
verifier := assets.NewVerifier(bundlePath, fs, telemetry)
err := verifier.EnsureAssets(service)

// Playwright setup
cfg := assets.PlaywrightConfig{
    BundlePath: bundlePath,
    FS:         fs,
    EnvReader:  envReader,
    Ports:      portAllocator,
    Telemetry:  telemetry,
}
err := assets.ApplyPlaywrightConventions(cfg, service, envMap)
```

## Dependencies

- **Depends on**: `manifest`, `infra` (FileSystem, EnvReader), `ports`, `telemetry`
- **Depended on by**: `bundleruntime`
