# UI Smoke Testing Guide

**Status**: Active
**Last Updated**: 2025-12-03

---

## Overview

UI smoke testing validates that a scenario's UI is accessible, renders correctly, integrates with the iframe-bridge, and has no critical JavaScript errors. It runs during the **structure** phase as a fast sanity check before deeper testing.

The UI smoke test is implemented as a Go-native orchestrator that loads the scenario's UI in an iframe via Browserless, validates the iframe-bridge handshake, and captures artifacts.

## Quick Start

### Enable UI Smoke Testing

UI smoke testing is **enabled by default** for scenarios with a `ui/` directory. To customize settings, add configuration to `.vrooli/testing.json`:

```json
{
  "structure": {
    "ui_smoke": {
      "enabled": true,
      "timeout_ms": 90000,
      "handshake_timeout_ms": 15000,
      "handshake_signals": []
    }
  }
}
```

### Run Smoke Tests

```bash
# Run structure phase (includes UI smoke)
test-genie execute my-scenario --phases structure

# Check test artifacts
ls coverage/ui-smoke/
```

## Configuration Reference

### Full Configuration

```json
{
  "structure": {
    "ui_smoke": {
      "enabled": true,
      "timeout_ms": 90000,
      "handshake_timeout_ms": 15000,
      "handshake_signals": [
        "customApp.ready",
        "MY_APP_INITIALIZED"
      ]
    }
  }
}
```

### Configuration Options

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | boolean | `true` | Enable/disable UI smoke testing |
| `timeout_ms` | number | `90000` | Overall timeout for the test (ms) |
| `handshake_timeout_ms` | number | `15000` | Max time to wait for iframe-bridge handshake (ms) |
| `handshake_signals` | string[] | `[]` | Custom window property paths to check for readiness |

### Default Handshake Signals

When no custom signals are provided, the following signals are checked (in order):

1. `window.__vrooliBridgeChildInstalled`
2. `window.IFRAME_BRIDGE_READY`
3. `window.IframeBridge.ready`
4. `window.iframeBridge.ready`
5. `window.IframeBridge.getState().ready`

### Custom Handshake Signals

You can define custom signals for apps that use different readiness indicators:

```json
{
  "structure": {
    "ui_smoke": {
      "handshake_signals": [
        "myApp.initialized",
        "REACT_APP_READY",
        "store.getState().isReady"
      ]
    }
  }
}
```

Signal patterns supported:
- **Simple property**: `"MY_FLAG"` checks `window.MY_FLAG === true`
- **Nested property**: `"app.ready"` checks `window.app && window.app.ready === true`
- **Method call**: `"store.getState().ready"` checks `window.store && typeof window.store.getState === 'function' && window.store.getState().ready === true`

## Prerequisites

### 1. Browserless Resource

The UI smoke test requires Browserless to be running:

```bash
# Check status
resource-browserless manage status

# Start if needed
resource-browserless manage start
```

### 2. iframe-bridge Dependency

Your UI must have `@vrooli/iframe-bridge` as a dependency in `ui/package.json`:

```json
{
  "dependencies": {
    "@vrooli/iframe-bridge": "workspace:*"
  }
}
```

**What is iframe-bridge?**

The `@vrooli/iframe-bridge` package provides communication utilities between Vrooli's host environment and scenario UIs embedded in iframes. It handles:

- **Ready signaling**: Notifies the host when the UI has finished initializing
- **Message passing**: Enables secure cross-origin communication between host and iframe
- **Storage shimming**: Patches localStorage/sessionStorage for iframe compatibility

When a UI smoke test runs, Browserless loads your UI in a headless browser and waits for the iframe-bridge to signal that the app is ready. If the bridge never signals ready, the test fails.

For detailed implementation guidance, see the [iframe-bridge README](/packages/iframe-bridge/README.md).

### 3. UI Port Definition

Your scenario should define a UI port in `.vrooli/service.json`:

```json
{
  "ports": {
    "ui": {
      "env_var": "UI_PORT",
      "description": "UI development server port"
    }
  }
}
```

## Execution Flow

The UI smoke test follows this sequence:

1. **Check UI directory exists** - Skip if no `ui/` directory
2. **Check Browserless health** - Block if Browserless is offline
3. **Check bundle freshness** - Block if source files are newer than dist
4. **Check UI port defined** - Determine if scenario expects a UI
5. **Discover UI port** - Find the running UI server port
6. **Check iframe-bridge dependency** - Fail if missing
7. **Execute browser session** - Load UI in iframe via Browserless
8. **Evaluate handshake** - Wait for bridge readiness signal
9. **Write artifacts** - Save screenshot, console logs, etc.
10. **Build result** - Determine pass/fail status

## Test Results

### Status Values

| Status | Meaning |
|--------|---------|
| `passed` | UI loaded successfully with handshake |
| `failed` | Test encountered errors (JS errors, network failures, no handshake) |
| `skipped` | Test was skipped (no UI directory or no UI port defined) |
| `blocked` | Precondition failed (Browserless offline, bundle stale, port not running) |

### Result JSON

Results are stored in `coverage/ui-smoke/latest.json`:

```json
{
  "scenario": "my-scenario",
  "status": "passed",
  "message": "UI loaded successfully",
  "timestamp": "2025-12-03T10:30:00Z",
  "duration_ms": 3500,
  "ui_url": "http://localhost:3000",
  "handshake": {
    "signaled": true,
    "timed_out": false,
    "duration_ms": 1200
  },
  "artifacts": {
    "screenshot": "coverage/ui-smoke/screenshot.png",
    "console": "coverage/ui-smoke/console.json",
    "network": "coverage/ui-smoke/network.json",
    "html": "coverage/ui-smoke/dom.html",
    "raw": "coverage/ui-smoke/raw.json"
  }
}
```

## Artifacts

All artifacts are stored in `coverage/ui-smoke/`:

| File | Format | Content |
|------|--------|---------|
| `screenshot.png` | PNG | UI screenshot at test completion |
| `console.json` | JSON | All console messages (log/warn/error/info) |
| `network.json` | JSON | Failed network requests (4xx/5xx/timeouts) |
| `dom.html` | HTML | Complete DOM snapshot |
| `raw.json` | JSON | Full Browserless response (minus screenshot) |
| `latest.json` | JSON | Complete result object with metadata |
| `README.md` | Markdown | Human-readable summary with troubleshooting |

## Troubleshooting

### Browserless Offline

**Symptom**: Test blocked with "Browserless resource is offline"

**Solution**:
```bash
resource-browserless manage start
```

### Bundle Stale

**Symptom**: Test blocked with "Source file newer than bundle"

**Solution**:
```bash
vrooli scenario restart my-scenario
```

### Handshake Timeout

**Symptom**: Test failed with "Iframe bridge never signaled ready"

**Why This Fails the Test**:

The iframe-bridge handshake is **required** for UI smoke tests to pass. This is by design because:

1. **Vrooli's architecture** relies on iframe embedding for scenario UIs
2. **Production readiness** requires proper host-iframe communication
3. **Silent failures** in the bridge would cause runtime issues

If the handshake times out, it indicates a fundamental integration problem that must be fixed.

**Causes**:
1. iframe-bridge not properly initialized in your app
2. App crashes before reaching ready state
3. Custom signals don't match your app's readiness indicators
4. JavaScript errors preventing the bridge from initializing
5. Missing or incorrect `@vrooli/iframe-bridge` import

**Solutions**:

1. **Verify iframe-bridge installation**:
   ```bash
   # Check if dependency exists
   grep iframe-bridge ui/package.json

   # Reinstall if needed
   cd ui && pnpm add @vrooli/iframe-bridge
   ```

2. **Ensure proper initialization** in your app entry point:
   ```typescript
   // App.tsx or index.tsx - must be called early!
   import { initIframeBridge } from '@vrooli/iframe-bridge';

   // Call before React renders
   initIframeBridge();
   ```

3. **Check console.json artifact** for JavaScript errors:
   ```bash
   cat coverage/ui-smoke/console.json | jq '.[] | select(.type == "error")'
   ```

4. **Use custom handshake signals** if your app uses different readiness indicators:
   ```json
   {
     "structure": {
       "ui_smoke": {
         "handshake_signals": ["myApp.ready", "STORE_INITIALIZED"]
       }
     }
   }
   ```

5. **Increase timeout** if your app legitimately takes longer to initialize:
   ```json
   {
     "structure": {
       "ui_smoke": {
         "handshake_timeout_ms": 30000
       }
     }
   }
   ```

For more information about iframe-bridge, see the [iframe-bridge README](/packages/iframe-bridge/README.md).

### UI Port Not Detected

**Symptom**: Test blocked with "UI port is defined in service.json but not detected"

**Solutions**:
1. Ensure your scenario is running: `vrooli scenario status my-scenario`
2. Check UI server logs: `vrooli scenario logs my-scenario --step start-ui`
3. Restart the scenario: `vrooli scenario restart my-scenario`

### Network Failures

**Symptom**: Test failed with "Network failures (N total)"

**Causes**:
1. API endpoints returning errors
2. Missing assets (CSS, JS, images)
3. CORS issues

**Solutions**:
1. Check `network.json` artifact for specific failed requests
2. Ensure all required services are running
3. Fix any 404s or 500s in your API

### Blank Screenshots

**Symptom**: Screenshot shows blank/white page

**Causes**:
1. Page hasn't fully rendered
2. CSS loading issues
3. JavaScript errors preventing render

**Solutions**:
1. Increase `handshake_timeout_ms` to give more time for rendering
2. Check `console.json` for errors
3. Check `dom.html` to see actual DOM state

## Storage Shim

The UI smoke test evaluates `window.__VROOLI_UI_SMOKE_STORAGE_PATCH__` to check if the iframe-bridge properly patches localStorage/sessionStorage APIs. This helps detect storage access issues in iframe contexts.

Results are included in the `storage_shim` field of the result JSON.

## Best Practices

### 1. Keep UI Smoke Tests Fast

UI smoke is meant to be a quick sanity check. If your UI takes too long to load:
- Optimize initial bundle size
- Defer non-critical resources
- Use the default timeouts (90s is generous)

### 2. Initialize iframe-bridge Early

Initialize iframe-bridge as early as possible in your app entry point so the readiness signal fires quickly:

```typescript
// App.tsx or index.tsx
import { initIframeBridge } from '@vrooli/iframe-bridge';

initIframeBridge();
```

### 3. Use Meaningful Handshake Signals

If using custom signals, choose ones that indicate your app is truly ready:
- After initial data fetches complete
- After authentication state is resolved
- After core components have mounted

### 4. Handle Errors Gracefully

Unhandled JavaScript errors will cause the test to fail. Ensure your app has proper error boundaries.

## See Also

- [Structure Phase](README.md) - Structure phase overview
- [UI Automation with BAS](../playbooks/ui-automation-with-bas.md) - Full UI testing
- [Phases Overview](../README.md) - Phase architecture
