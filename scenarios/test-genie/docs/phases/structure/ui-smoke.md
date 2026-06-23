# iframe-bridge Handshake & UI Render Debugging Guide

**Status**: Active (debugging reference)
**Last Updated**: 2026-06-23

> **Moved:** The native `smoke` test-genie phase and the standalone `vrooli scenario ui-smoke` command were **retired**. BAS-driven UI render + iframe-bridge handshake validation now runs inside the **[ui-health phase](../ui-health/README.md)** (execution mode) — see `ui-health validate scenario <name>`. The `playbooks` phase also drives the bridge. This guide is retained as the iframe-bridge handshake / render-failure **debugging reference** that those phases' remediation messages point at; the legacy `.vrooli/testing.json` `structure.ui_smoke` config knob below is preserved where still honored.

---

## Overview

UI render validation checks that a scenario's UI is accessible, renders correctly, integrates with the iframe-bridge, and has no critical JavaScript errors.

It is driven through the **Browser Automation Studio (BAS)** workflow engine: the ui-health phase embeds the scenario's UI in a host iframe shell, validates the iframe-bridge handshake, and captures artifacts (the `playbooks` phase drives the bridge top-level instead). The verdict is produced by the shared `internal/evidence` analyzer.

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
# Run the ui-health phase (static UI checks + BAS render/handshake in execution mode)
test-genie execute my-scenario ui-health

# Or invoke ui-health directly
ui-health validate scenario my-scenario

# Check render artifacts
ls coverage/runs/<run-id>/ui-smoke/pages/
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

### 1. Browser Automation Studio (BAS)

The UI smoke test drives its capture on the BAS workflow engine, so the
`browser-automation-studio` scenario must be running:

```bash
# Start if needed (idempotent)
vrooli scenario start browser-automation-studio
```

When BAS is unreachable, the runnability gate **skips** the smoke phase
(resource unavailable) rather than failing it hard.

### 2. iframe-bridge Dependency

Your UI must have `@vrooli/iframe-bridge` as a dependency in `ui/package.json`:

```json
{
  "dependencies": {
    "@vrooli/iframe-bridge": "file:../../../packages/iframe-bridge"
  }
}
```

**What is iframe-bridge?**

The `@vrooli/iframe-bridge` package provides communication utilities between Vrooli's host environment and scenario UIs embedded in iframes. It handles:

- **Ready signaling**: Notifies the host when the UI has finished initializing
- **Message passing**: Enables secure cross-origin communication between host and iframe
- **Storage shimming**: Patches localStorage/sessionStorage for iframe compatibility

When a UI smoke test runs, the BAS workflow engine embeds your UI in a host iframe shell inside a headless browser and waits for the iframe-bridge to signal that the app is ready (via the bridge's READY/HELLO postMessage or a window-property readiness signal). If the bridge never signals ready, the test fails.

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
2. **Check bundle freshness** - Block if source files are newer than dist
3. **Discover UI port** - Find the running UI server port (or skip when no UI port is defined; auto-start when requested)
4. **Check iframe-bridge dependency** - Fail if missing
5. **Capture via BAS** - Run the inline smoke workflow (navigate host shell → inject iframe + arm handshake listener → assert the readiness marker → screenshot the frame)
6. **Analyze evidence** - The shared `internal/evidence` analyzer decides the verdict (handshake hard-fail, then network failures, then page errors; console errors counted, console warnings surfaced)
7. **Write artifacts** - Save screenshot, console logs, network failures, raw evidence
8. **Build result** - Determine pass/fail status

> BAS reachability is decided up front by the runnability gate, not inside this flow: when BAS is down the phase is skipped before step 1.

## Test Results

### Status Values

| Status | Meaning |
|--------|---------|
| `passed` | UI loaded successfully with handshake |
| `failed` | Test encountered errors (JS errors, network failures, no handshake) |
| `skipped` | Test was skipped (no UI directory or no UI port defined) |
| `blocked` | Precondition failed (bundle stale, or UI port defined but not running) |

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
| `network.json` | JSON | Failed network requests (4xx/5xx/transport errors) |
| `raw.json` | JSON | Raw engine-agnostic evidence (minus screenshot bytes) |
| `latest.json` | JSON | Complete result object with metadata |
| `README.md` | Markdown | Human-readable summary with troubleshooting |

## Troubleshooting

### BAS Unavailable

**Symptom**: Smoke phase **skipped** with a "requires unavailable resource(s):
browser-automation-studio" reason.

**Solution**:
```bash
vrooli scenario start browser-automation-studio
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
3. Check `screenshot.png` and `network.json` to see the actual rendered state and failed requests

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
