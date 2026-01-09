# Deployment Troubleshooting Guide

Common issues and solutions when deploying Vrooli scenarios across tiers.

## Quick Diagnostics

Run these commands first to identify the problem area:

```bash
# Check deployment-manager health
deployment-manager status

# Check scenario fitness for target tier
deployment-manager fitness <scenario> --tier <tier-number>

# Validate deployment profile
deployment-manager validate <profile-id> --verbose

# Check scenario dependencies
deployment-manager analyze <scenario>
```

---

## deployment-manager Issues

### "Cannot connect to deployment-manager API"

**Symptoms:**
- CLI commands fail with connection errors
- "Connection refused" or timeout errors

**Causes & Solutions:**

1. **Scenario not running**
   ```bash
   vrooli scenario status deployment-manager
   # If stopped:
   vrooli scenario start deployment-manager
   ```

2. **Wrong API port**
   ```bash
   # Get correct port
   API_PORT=$(vrooli scenario port deployment-manager API_PORT)

   # Update CLI configuration
   deployment-manager configure api_base "http://localhost:${API_PORT}"
   ```

3. **Port conflict**
   ```bash
   # Check what's using the port
   lsof -i :<port>

   # Restart scenario to get new port
   vrooli scenario stop deployment-manager
   vrooli scenario start deployment-manager
   ```

### "Scenario dependency analyzer not healthy"

**Symptoms:**
- `deployment-manager analyze` fails
- Status shows `scenario-dependency-analyzer: unhealthy`

**Solution:**
```bash
# Start the dependency analyzer
vrooli scenario start scenario-dependency-analyzer

# Verify it's healthy
vrooli scenario status scenario-dependency-analyzer
```

### "Secrets manager not available"

**Symptoms:**
- `deployment-manager secrets` commands fail
- Status shows `secrets-manager: unhealthy`

**Solution:**
```bash
vrooli scenario start secrets-manager
```

---

## Fitness Scoring Issues

### "Fitness score is 0"

**Symptoms:**
- `deployment-manager fitness <scenario> --tier 2` returns 0
- Blockers list contains critical issues

**Causes & Solutions:**

1. **Unsupported dependencies**
   ```bash
   # Check blockers
   deployment-manager fitness <scenario> --tier 2

   # View available swaps
   deployment-manager swaps list <scenario>

   # Apply recommended swaps
   deployment-manager profile swap <profile-id> add postgres sqlite
   ```

2. **Circular dependencies**
   ```bash
   deployment-manager analyze <scenario>
   # If circular_dependencies: true, refactor scenario dependencies
   ```

3. **Missing platform binaries**
   - Ensure API can be cross-compiled for target platforms
   - Check `go build` works with `GOOS=windows/darwin/linux`

### "Fitness score not improving after swaps"

**Symptoms:**
- Applied swaps but score unchanged

**Solution:**
```bash
# Verify swap was applied
deployment-manager profile show <profile-id>

# Recalculate fitness with swaps
deployment-manager swaps apply <profile-id> <from> <to> --show-fitness
```

---

## Profile Management Issues

### "Profile not found"

**Symptoms:**
- Commands fail with "profile not found" error

**Causes & Solutions:**

1. **Wrong profile ID**
   ```bash
   # List all profiles
   deployment-manager profiles

   # Use correct ID
   deployment-manager profile show <correct-id>
   ```

2. **Database not initialized**
   ```bash
   # Restart deployment-manager to reinitialize
   vrooli scenario stop deployment-manager
   vrooli scenario start deployment-manager
   ```

### "Profile swap failed"

**Symptoms:**
- `profile swap add` command fails

**Solution:**
```bash
# Verify swap is valid
deployment-manager swaps analyze <from> <to>

# Check if swap is applicable to tier
deployment-manager swaps list <scenario>
```

---

## Bundle Manifest Issues

### "Bundle manifest validation failed"

**Symptoms:**
- `/api/v1/bundles/validate` returns errors
- Export fails with schema errors

**Common causes:**

1. **Missing required fields**
   ```json
   {
     "schema_version": "v0.1",  // Required
     "target": "desktop",        // Required
     "app": {
       "name": "...",           // Required
       "version": "..."         // Required
     }
   }
   ```

2. **Invalid secret class**
   - Valid classes: `per_install_generated`, `user_prompt`, `remote_fetch`, `infrastructure`

3. **No services defined**
   - At least one service must be in the `services` array

4. **Invalid IPC mode**
   - Must be `loopback-http` for desktop

**Debug approach:**
```bash
# Validate against schema
API_PORT=$(vrooli scenario port deployment-manager API_PORT)
curl -X POST "http://localhost:${API_PORT}/api/v1/bundles/validate" \
  -H "Content-Type: application/json" \
  -d @bundle.json
```

### "Failed to assemble bundle"

**Symptoms:**
- `/api/v1/bundles/assemble` returns error

**Causes:**

1. **scenario-dependency-analyzer unavailable**
   ```bash
   vrooli scenario start scenario-dependency-analyzer
   ```

2. **secrets-manager unavailable**
   ```bash
   vrooli scenario start secrets-manager
   ```

3. **Scenario doesn't exist**
   ```bash
   # Verify scenario name
   vrooli scenario list | grep <scenario-name>
   ```

---

## Desktop Build Issues

### "Electron build fails with missing dependencies"

**Symptoms:**
- `pnpm run dist:all` fails
- npm/node errors during build

**Solution:**
```bash
cd scenarios/<scenario>/platforms/electron

# Clean install
rm -rf node_modules
rm -rf dist-electron
pnpm install

# Retry build
pnpm run dist:all
```

### "Cannot find module 'electron'"

**Solution:**
```bash
# Ensure electron is installed
pnpm add -D electron electron-builder

# Verify versions
pnpm list electron electron-builder
```

### "Cross-platform build fails on Linux"

**Symptoms:**
- Building Windows MSI on Linux fails

**Solution:**
```bash
# Install Wine for Windows builds
sudo apt install wine64

# Or build only Linux targets
pnpm run dist:linux
```

### "macOS build requires code signing"

**Symptoms:**
- Build fails with signing errors
- "The application cannot be opened" on macOS

**Solutions:**

1. **For development/testing:**
   ```json
   // package.json
   {
     "build": {
       "mac": {
         "identity": null
       }
     }
   }
   ```

2. **For distribution:**
   - Obtain Apple Developer certificate
   - Configure in `package.json`:
   ```json
   {
     "build": {
       "mac": {
         "hardenedRuntime": true,
         "gatekeeperAssess": true,
         "identity": "Developer ID Application: Your Name (TEAMID)"
       }
     }
   }
   ```

### "Bundle directory not found"

**Symptoms:**
- Electron build can't find `bundle/` directory

**Solution:**
```bash
# Create bundle directory structure
mkdir -p scenarios/<scenario>/platforms/electron/bundle

# Copy manifest
cp bundle.json scenarios/<scenario>/platforms/electron/bundle/

# Ensure package.json includes bundle in extraResources
```

---

## Runtime Issues (Bundled Apps)

### "App starts but shows blank screen"

**Causes & Solutions:**

1. **UI bundle missing**
   ```bash
   # Verify UI files exist
   ls scenarios/<scenario>/platforms/electron/bundle/ui/

   # Rebuild UI if missing
   cd scenarios/<scenario>/ui
   pnpm run build
   cp -r dist ../platforms/electron/bundle/ui/
   ```

2. **Wrong API URL in UI**
   - Check `VITE_API_BASE_URL` points to bundled API port

### "App starts but API calls fail"

**For thin client:**
```bash
# Verify server URL is correct
# Check network connectivity to Tier 1 server
curl https://your-server.example.com/api/v1/health
```

**For bundled app:**
1. Check runtime supervisor started
2. Verify IPC port matches manifest
3. Check service logs:
   ```bash
   # Logs are in app data directory
   # Windows: %APPDATA%/<app-name>/logs/
   # macOS: ~/Library/Application Support/<app-name>/logs/
   # Linux: ~/.config/<app-name>/logs/
   ```

### "Runtime failed to start services"

**Symptoms:**
- App launches but services never become ready
- Telemetry shows `dependency_unreachable` events

**Debug steps:**

1. **Check manifest service order**
   - Services start in dependency order
   - Verify `dependencies` arrays are correct

2. **Check port allocation**
   - Ports are allocated from `ports.default_range`
   - Ensure range has enough available ports

3. **Check health endpoints**
   - Each service needs working health endpoint
   - Verify `health.path` is correct

### "Secrets not available at runtime"

**Symptoms:**
- App prompts for secrets but they're not saved
- Services fail with "missing secret" errors

**Solutions:**

1. **Check secret storage location**
   ```bash
   # Secrets stored in app data
   cat ~/.config/<app-name>/secrets.json
   ```

2. **Verify manifest secret configuration**
   - `per_install_generated` secrets should have `generator` config
   - `user_prompt` secrets should have `prompt` config

---

## Auto-Update Issues

### "Update check fails"

**Symptoms:**
- "Update check failed" in app
- Telemetry shows `update_error` events

**Causes & Solutions:**

1. **No update config**
   - Verify `update_config` was provided during generation

2. **GitHub releases not accessible**
   ```bash
   # Verify releases exist
   curl https://api.github.com/repos/<owner>/<repo>/releases
   ```

3. **Wrong channel**
   - Pre-releases only show for `dev`/`beta` channels
   - Stable releases show for all channels

### "Update downloaded but won't install"

**Symptoms:**
- Update downloads but fails to install
- App shows "Update failed" error

**Solutions:**

1. **Code signing mismatch**
   - Ensure new version is signed with same certificate

2. **Permissions issue**
   - Windows: Run as administrator
   - macOS: Ensure app is in Applications folder
   - Linux: AppImage needs execute permission

---

## Telemetry Issues

### "Telemetry not uploading"

**Symptoms:**
- No data in deployment-manager telemetry view
- Local telemetry file exists but isn't syncing

**Solutions:**

1. **Check upload URL**
   ```json
   // In manifest
   {
     "telemetry": {
       "upload_to": "https://your-server/api/v1/telemetry/upload"
     }
   }
   ```

2. **Manual upload for debugging**
   ```bash
   curl -X POST "http://localhost:<port>/api/v1/telemetry/upload" \
     -H "Content-Type: application/json" \
     -d @~/.config/<app-name>/telemetry/deployment-telemetry.jsonl
   ```

3. **Network connectivity**
   - Verify app can reach telemetry endpoint
   - Check firewall rules

---

## Getting Help

If these solutions don't resolve your issue:

1. **Check scenario-specific docs**
   - `scenarios/<scenario>/docs/PROBLEMS.md` may have known issues

2. **Review deployment-manager logs**
   ```bash
   deployment-manager logs <profile-id> --level error
   ```

3. **Check progress and known issues**
   - `scenarios/deployment-manager/docs/PROGRESS.md`
   - `scenarios/deployment-manager/docs/PROBLEMS.md`

4. **File an issue**
   - Report issues at https://github.com/anthropics/claude-code/issues
   - Include: scenario name, tier, commands run, error messages

---

## Related Documentation

- [Desktop Deployment Workflow](desktop-deployment.md)
- [Mobile Deployment Workflow](mobile-deployment.md)
- [SaaS Deployment Workflow](saas-deployment.md)
- [Fitness Scoring Guide](../guides/fitness-scoring.md)
- [Dependency Swapping Guide](../guides/dependency-swapping.md)
- [Auto-Updates Guide](../guides/auto-updates.md)
