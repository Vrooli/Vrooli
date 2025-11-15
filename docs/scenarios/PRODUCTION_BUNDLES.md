# Production Bundles for Web Scenarios

## Overview

Web-based scenarios **must serve production-built UI bundles** in their develop phase, not development servers (like `vite dev`, `npm run dev`, etc.). This is a critical architectural requirement for the Vrooli ecosystem.

> **Note:** Converting `develop` to use production bundles does *not* eliminate the need for a dedicated deployment plan. The production phase remains where we package scenarios for other tiers (desktop, SaaS, etc.) per the [Deployment Hub](../deployment/README.md) and tier docs such as [Tier 2 Desktop](../deployment/tiers/tier-2-desktop.md). Treat the develop phase as "production-parity local runtime" and the production phase as "shipping the scenario." Keeping both phases prevents `vrooli scenario status` from flagging missing deployment automation and ensures deployments stay auditable.

## Why Production Bundles Are Required

### 1. **Auto-Rebuild Detection**
The Vrooli lifecycle system automatically detects stale code and rebuilds when you restart scenarios. This only works correctly with production bundles:

```bash
# Agent makes changes via app-issue-tracker
# Issue completion triggers restart
vrooli scenario restart my-app

# ✅ With production bundles: Detects stale UI, rebuilds automatically
# ❌ With dev server: No staleness detection, changes may not appear
```

### 2. **Cache-Busting**
Production builds use cache-busting filenames (e.g., `index-abc123.js`) which ensure browsers always load the latest code. Dev servers don't provide this guarantee when accessed through proxies or embedded contexts.

### 3. **Consistent Behavior Across Contexts**
Scenarios are often embedded in other apps (like app-monitor's iframe previews). Production bundles work reliably in all contexts:
- Direct access: `http://localhost:3000`
- Proxied access: `http://app-monitor.local/apps/my-app/proxy/`
- Embedded iframes: `<iframe src="..." />`

Dev servers may fail or behave inconsistently in these scenarios due to CORS, HMR websockets, and relative path assumptions.

### 4. **Predictable Performance**
Production bundles have consistent load times and resource sizes. Dev servers can vary wildly depending on module caching, making performance testing unreliable.

### 5. **True Integration Testing**
Testing against production bundles ensures your tests validate what users will actually experience, not an artificial development-only environment.

## How to Convert to Production Bundles

### Prerequisites
Check your scenario uses a dev server in develop phase:
```bash
# Look for these patterns in .vrooli/service.json develop phase:
npm run dev
vite
webpack-dev-server
react-scripts start
```

### Step 1: Add UI Bundle Staleness Check

Add the `ui-bundle` check to your setup conditions in `.vrooli/service.json`:

```json
{
  "lifecycle": {
    "setup": {
      "condition": {
        "checks": [
          {
            "type": "binaries",
            "targets": ["api/my-app-api"]
          },
          {
            "type": "ui-bundle",
            "bundle_path": "ui/dist/index.html",
            "source_dir": "ui/src"
          },
          {
            "type": "cli",
            "targets": ["my-app"]
          }
        ]
      }
    }
  }
}
```

**Configuration:**
- `bundle_path`: Path to production bundle entry point (usually `ui/dist/index.html` for Vite)
- `source_dir`: Directory containing source files to watch for changes

### Step 2: Update Develop Phase

Change the `start-ui` step to serve the built bundle:

**Before (dev server):**
```json
{
  "name": "start-ui",
  "run": "cd ui && npm run dev",
  "description": "Start React UI development server",
  "background": true,
  "condition": {
    "file_exists": "ui/package.json"
  }
}
```

**After (production server):**
```json
{
  "name": "start-ui",
  "run": "cd ui && node server.js",
  "description": "Start Express server with built React app",
  "background": true,
  "condition": {
    "file_exists": "ui/dist/index.html"
  }
}
```

### Step 3: Add Express Dependency

Add `express` to `ui/package.json`:

```json
{
  "dependencies": {
    "express": "^4.21.2",
    ...other dependencies
  }
}
```

### Step 4: Create Production Server

Create `ui/server.js` (or `ui/server.cjs` if `package.json` has `"type": "module"`):

```javascript
const express = require('express');
const path = require('path');

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT || 3000;
const API_PORT = process.env.API_PORT;

// Serve static files from dist directory
app.use(express.static(path.join(__dirname, 'dist')));

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({ status: 'healthy', service: 'my-app-ui' });
});

// Catch all route - serve index.html for SPA routing
app.get('*', (req, res) => {
    res.sendFile(path.join(__dirname, 'dist', 'index.html'));
});

app.listen(PORT, () => {
    console.log(`My App UI running on http://localhost:${PORT}`);
    if (API_PORT) {
        console.log(`API available at http://localhost:${API_PORT}`);
    }
});
```

**Note:** If your `package.json` has `"type": "module"`, rename the file to `server.cjs` and update the service.json accordingly.

### Step 5: Ensure Build Step Exists

Verify your setup phase includes a `build-ui` step:

```json
{
  "setup": {
    "steps": [
      {
        "name": "install-ui-deps",
        "run": "cd ui && npm install",
        "description": "Install React UI dependencies"
      },
      {
        "name": "build-ui",
        "run": "cd ui && npm run build",
        "description": "Build React UI for production"
      }
    ]
  }
}
```

### Step 6: Fix Binary Paths (If Applicable)

Ensure binary check targets include the subdirectory:

**Before (incorrect):**
```json
{
  "type": "binaries",
  "targets": ["my-app-api"]
}
```

**After (correct):**
```json
{
  "type": "binaries",
  "targets": ["api/my-app-api"]
}
```

## Testing the Conversion

Run these three test cases to validate the conversion:

### Test 1: Go API Changes Detection
```bash
vrooli scenario start my-app
echo "// test change" >> api/main.go
vrooli scenario restart my-app

# Expected: "Running setup before develop" + "build-api" step runs
```

### Test 2: UI Changes Detection
```bash
vrooli scenario start my-app
touch ui/src/App.tsx
vrooli scenario restart my-app

# Expected: "Running setup before develop" + "build-ui" step runs
```

### Test 3: Fast Idempotency Path
```bash
vrooli scenario start my-app
vrooli scenario start my-app

# Expected: "✓ Scenario 'my-app' is already running and healthy" (immediate return)
```

## Common Issues

### Issue: "Cannot find module 'express'"
**Cause:** Express not installed in ui/package.json
**Fix:** Add `"express": "^4.21.2"` to dependencies and run setup

### Issue: "ReferenceError: require is not defined in ES module scope"
**Cause:** package.json has `"type": "module"` but server uses CommonJS
**Fix:** Rename `server.js` to `server.cjs` and update service.json

### Issue: Binary check always reports "Missing"
**Cause:** Binary target path doesn't include subdirectory
**Fix:** Change `"my-app-api"` to `"api/my-app-api"` in condition checks

### Issue: UI bundle check not triggering rebuild
**Cause:** UI bundle check not registered in setup.sh
**Fix:** Ensure `ui-bundle` case exists in `/scripts/lib/utils/setup.sh` (should already be there)

## Architecture Details

### How Auto-Rebuild Works

1. **Scenario Start/Restart**: Lifecycle system calls `setup::is_needed()`
2. **Condition Checks**: Each check script returns exit code:
   - `0` = Setup needed (stale code detected)
   - `1` = Not needed (code is current)
3. **Staleness Detection**:
   - **Binaries**: Compares `.go` file timestamps to binary timestamp using `find ... -newer`
   - **UI Bundles**: Compares source files in `ui/src/` to `ui/dist/index.html` timestamp
4. **Rebuild Triggered**: If any check returns `0`, full setup runs before starting

### Why Dev Servers Don't Work

Dev servers bypass the filesystem-based staleness detection:
- Hot Module Replacement (HMR) updates in-memory, not on disk
- No bundle timestamp to compare against source changes
- Lifecycle system can't detect when source has changed

## Examples in Codebase

Well-configured scenarios using production bundles:
- `app-monitor`: Full proxy support with production bundles
- `feature-request-voting`: Clean production bundle setup
- `system-monitor`: Recently converted, serves as good reference

## Related Documentation

- [Phased Testing Architecture](/docs/testing/architecture/PHASED_TESTING.md)
- [Lifecycle System v2.0](/docs/scenarios/LIFECYCLE_V2.md)
- [Service Configuration Schema](/.vrooli/schemas/service.schema.json)

## Questions?

Run `vrooli scenario status <name>` to check if your scenario needs conversion to production bundles.
