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

Services using Playwright declare a package-local browser closure with
`PLAYWRIGHT_BROWSERS_PATH`. A staged `browser-closure.json` carries the shared
`runtime_identity` (engine, browser version, driver/version, deterministic
bundle version and whole-payload SHA-256) plus platform and executable binding.
The runtime validates the identity, required installed receipt and digest,
resolves the executable only within the packaged closure, and sets the single
`BROWSER_EXECUTABLE_PATH` consumed by BAS. Missing, floating, stale, mismatched
or conflicting closure data fails before the service process is created. There
is no user-cache, system-browser or Electron fallback.

### Environment Variables Set

| Variable | Description |
|----------|-------------|
| `PLAYWRIGHT_BROWSERS_PATH` | Path to browser binaries |
| `BROWSER_EXECUTABLE_PATH` | Exact browser executable override consumed by BAS |
| `PWDEBUG_PORT` | Debug protocol port (if allocated) |

Browser acquisition is outside this runtime helper. BAS's
`bundle/stage-browser-closure.mjs` consumes an explicit artifact tree after it
has been provisioned through the governed dependency/artifact owner. It stages
the exact `browsers.json` revisions and emits the closure manifest before
scenario-to-desktop copies the declared `asset_dirs`. The helper never
downloads or looks in a developer cache.

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
