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

Services using Playwright need browser binaries. The package:

1. Detects Playwright usage via `service.playwright_browsers`
2. Sets `PLAYWRIGHT_BROWSERS_PATH` to bundle's browser directory
3. Allocates debug port if needed
4. Falls back to system Chromium if bundled browsers missing

### Environment Variables Set

| Variable | Description |
|----------|-------------|
| `PLAYWRIGHT_BROWSERS_PATH` | Path to browser binaries |
| `PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH` | Fallback Chromium path |
| `PWDEBUG_PORT` | Debug protocol port (if allocated) |

### Fallback Detection

If bundled browsers aren't found, the package searches for system Chromium:

```
/usr/bin/chromium-browser
/usr/bin/chromium
/snap/bin/chromium
/Applications/Chromium.app/Contents/MacOS/Chromium
```

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
