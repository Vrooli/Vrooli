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

### Step 3: Add @vrooli/api-base Dependency

Add `@vrooli/api-base` to `ui/package.json`:

```json
{
  "dependencies": {
    "@vrooli/api-base": "workspace:*",
    ...other dependencies
  }
}
```

### Step 4: Create Production Server

Create `ui/server.js` using the standard `@vrooli/api-base/server` utility:

```javascript
import { startScenarioServer } from '@vrooli/api-base/server'

startScenarioServer({
  uiPort: process.env.UI_PORT,
  apiPort: process.env.API_PORT,
  distDir: './dist',
  serviceName: 'my-app',
  corsOrigins: '*',
})
```

**Why use `@vrooli/api-base/server`?**
- **Standard Implementation**: All scenarios use the same server with consistent behavior
- **Built-in Features**: Health checks, CORS, proxy support, SPA routing, static file serving
- **Maintained**: Updates and bug fixes automatically propagate to all scenarios
- **Testing Support**: Consistent server means consistent test expectations

**Note:** The `startScenarioServer` function is ESM-only. If your `package.json` has `"type": "commonjs"` (or no type field), either:
1. Add `"type": "module"` to package.json (recommended), or
2. Rename the file to `server.cjs` and use CommonJS `require()` syntax (not recommended - @vrooli/api-base may not support this)

### Step 5: Ensure Build Step Uses VITE_API_BASE_URL

Verify your setup phase includes a `build-ui` step with the correct environment variable:

```json
{
  "setup": {
    "steps": [
      {
        "name": "install-ui-deps",
        "run": "if [ -f ui/package.json ]; then cd ui && pnpm install; else echo 'ui/ not present yet'; fi",
        "description": "Install UI dependencies"
      },
      {
        "name": "build-ui",
        "run": "if [ -f ui/package.json ]; then if [ -z \"${API_PORT:-}\" ]; then echo 'API_PORT must be provided by the lifecycle system'; exit 1; fi; cd ui && VITE_API_BASE_URL=\"http://localhost:${API_PORT}/api/v1\" pnpm run build; else echo 'ui/ not present yet'; fi",
        "description": "Build production UI (requires lifecycle-assigned API_PORT)"
      }
    ]
  }
}
```

**Important:**
- The `VITE_API_BASE_URL` environment variable must be set during build time for Vite to embed the correct API URL
- The `${API_PORT}` comes from the lifecycle system's port allocation
- The conditional checks ensure graceful handling when UI hasn't been generated yet

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

### Issue: "Cannot find module '@vrooli/api-base/server'"
**Cause:** @vrooli/api-base not installed in ui/package.json
**Fix:** Add `"@vrooli/api-base": "workspace:*"` to dependencies and run `pnpm install` from ui/

### Issue: "ReferenceError: require is not defined in ES module scope"
**Cause:** package.json has `"type": "module"` but server.js uses CommonJS syntax
**Fix:** Update server.js to use ESM `import` syntax (see Step 4)

### Issue: "SyntaxError: Cannot use import statement outside a module"
**Cause:** package.json is missing `"type": "module"` but server.js uses ESM syntax
**Fix:** Add `"type": "module"` to ui/package.json

### Issue: Binary check always reports "Missing"
**Cause:** Binary target path doesn't include subdirectory
**Fix:** Change `"my-app-api"` to `"api/my-app-api"` in condition checks

### Issue: UI bundle check not triggering rebuild
**Cause:** UI bundle check not configured in setup.condition
**Fix:** Add ui-bundle check to setup.condition.checks (see Step 1)

### Issue: "VITE_API_BASE_URL is undefined in production build"
**Cause:** Environment variable not passed during build step
**Fix:** Ensure build-ui step includes `VITE_API_BASE_URL="http://localhost:${API_PORT}/api/v1"` (see Step 5)

### Issue: Dev server still starting instead of production server
**Cause:** Old dev server step still present in develop phase
**Fix:** Remove any steps running `npm run dev`, `vite`, etc. (see Step 2)

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

Well-configured scenarios using production bundles with `@vrooli/api-base/server`:
- `prd-control-tower`: `/scenarios/prd-control-tower/.vrooli/service.json` and `ui/server.js`
- `landing-manager`: `/scenarios/landing-manager/.vrooli/service.json` and `ui/server.cjs`
- `deployment-manager`: `/scenarios/deployment-manager/.vrooli/service.json` and `ui/server.cjs`
- `tidiness-manager`: `/scenarios/tidiness-manager/.vrooli/service.json` and `ui/server.js`
- Template: `/scripts/scenarios/templates/react-vite/` - reference implementation for new scenarios

## Related Documentation

- [Phased Testing Architecture](/docs/testing/architecture/PHASED_TESTING.md)
- [Lifecycle System v2.0](/docs/scenarios/LIFECYCLE_V2.md)
- [Service Configuration Schema](/.vrooli/schemas/service.schema.json)

## Questions?

Run `vrooli scenario status <name>` to check if your scenario needs conversion to production bundles.
