# Scenario Deployment Guide

> **The single entry point for deploying Vrooli scenarios to any platform tier.**
>
> This guide provides a complete, linear walkthrough. Agents can follow it start-to-finish. Developers can use it as a checklist and dive into linked docs for details.

---

## Current Implementation Status

**Read this first** to understand what works today vs. what requires workarounds.

### Status Legend

| Status | Meaning |
|--------|---------|
| **Working** | Fully automated, no manual steps |
| **API Only** | Works but requires REST API calls (no CLI command yet) |
| **Manual** | Components exist but require manual assembly |
| **Not Yet** | Not implemented; documented workaround below |

### Capability Matrix

| Capability | Status | Workaround |
|------------|--------|------------|
| Thin client desktop (UI only) | **Working** | N/A |
| Bundled desktop (UI + API + resources) | **Manual** | See [Phase 6B](#option-b-bundled-app-manual-assembly) |
| Fitness scoring | **Working** | N/A |
| Dependency swap analysis | **Working** | N/A |
| Profile management | **Working** | N/A |
| Secrets identification | **Working** | N/A |
| Bundle manifest assembly | **API Only** | See [Phase 5](#phase-5-generate-bundle-manifest) |
| Bundle manifest export | **API Only** | See [Phase 5](#phase-5-generate-bundle-manifest) |
| Automated binary compilation | **Not Yet** | Manual `GOOS/GOARCH go build` per platform |
| Asset procurement | **Not Yet** | Manually download Chromium, models, etc. |
| Code signing | **Not Yet** | Configure certs in `package.json` manually |
| End-to-end bundled validation | **Not Yet** | Use thin client for production today |

### What This Means for You

**If you need a desktop app today:**
- **Use thin client mode** (Phase 6A) - Fully automated, production-ready
- Requires your users to have network access to a Tier 1 Vrooli server

**If you need offline/standalone:**
- **Use bundled mode** (Phase 6B) - Manual assembly required
- Follow the detailed steps in Phase 6B; expect 30-60 minutes of setup
- Not yet validated end-to-end; report issues to [PROBLEMS.md](PROBLEMS.md)

---

## Desktop Deployment (Tier 2)

Deploy a scenario as a standalone Windows, macOS, or Linux application.

### Prerequisites

```bash
# Verify Vrooli CLI
vrooli --version

# Start required scenarios
vrooli scenario start deployment-manager
vrooli scenario start scenario-to-desktop

# Verify they're running
deployment-manager status
```

### Phase 1: Check Compatibility

```bash
# Analyze dependencies
deployment-manager analyze <scenario-name>

# Check desktop fitness (tier 2)
deployment-manager fitness <scenario-name> --tier 2
```

**Interpret the score:**
- **80-100**: Ready - proceed to Phase 3
- **60-79**: Good - review warnings, proceed
- **40-59**: Fair - address blockers in Phase 2
- **0-39**: Poor - significant swaps required

### Phase 2: Address Blocking Issues

If you have blockers (common: postgres, redis), apply swaps:

```bash
# List available swaps
deployment-manager swaps list <scenario-name>

# Analyze a specific swap
deployment-manager swaps analyze postgres sqlite

# Check cascading effects
deployment-manager swaps cascade postgres sqlite
```

### Phase 3: Create Deployment Profile

```bash
# Create profile targeting desktop
deployment-manager profile create <profile-name> <scenario-name> --tier 2

# Note the returned profile ID (e.g., profile-1234567890)

# Apply required swaps
deployment-manager swaps apply <profile-id> postgres sqlite --show-fitness

# Configure environment
deployment-manager profile set <profile-id> env LOG_LEVEL info

# Verify profile
deployment-manager profile show <profile-id>
```

### Phase 4: Configure Secrets

```bash
# Identify required secrets
deployment-manager secrets identify <profile-id>

# Generate template
deployment-manager secrets template <profile-id> --format env

# Validate secrets configuration
deployment-manager secrets validate <profile-id>
```

**Secret classifications:**
- `per_install_generated` - Auto-created on first run (no action needed)
- `user_prompt` - User provides during first-run wizard (no action needed)
- `infrastructure` - **Blocked** - ensure swaps removed these

### Phase 5: Generate Bundle Manifest

> **Note**: CLI command not yet available. Use REST API.

```bash
# Validate profile first
deployment-manager validate <profile-id> --verbose

# Get API port
API_PORT=$(vrooli scenario port deployment-manager API_PORT)

# Export bundle manifest with checksum
curl -X POST "http://localhost:${API_PORT}/api/v1/bundles/export" \
  -H "Content-Type: application/json" \
  -d '{"scenario": "<scenario-name>", "tier": "tier-2-desktop"}' \
  > bundle.json

# Validate the manifest
curl -X POST "http://localhost:${API_PORT}/api/v1/bundles/validate" \
  -H "Content-Type: application/json" \
  -d @bundle.json
```

### Phase 6: Build Desktop Installers

#### Option A: Thin Client (Working Today)

Bundles UI only; requires connection to Tier 1 server.

```bash
STD_PORT=$(vrooli scenario port scenario-to-desktop API_PORT)

curl -X POST "http://localhost:${STD_PORT}/api/v1/desktop/generate/quick" \
  -H "Content-Type: application/json" \
  -d '{
    "scenario_name": "<scenario-name>",
    "deployment_mode": "external-server",
    "server_url": "https://your-vrooli-server.example.com"
  }'
```

#### Option B: Bundled App (Manual Assembly)

> **Current Gap**: Full automation pending. Follow these manual steps.

```bash
# Navigate to Electron app
cd scenarios/<scenario-name>/platforms/electron

# Create bundle directory
mkdir -p bundle
cp /path/to/bundle.json bundle/

# Build API binaries (from scenario's api/ directory)
cd scenarios/<scenario-name>/api
GOOS=linux GOARCH=amd64 go build -o ../platforms/electron/bundle/services/api/linux-x64/<api-binary>
GOOS=darwin GOARCH=amd64 go build -o ../platforms/electron/bundle/services/api/macos-x64/<api-binary>
GOOS=darwin GOARCH=arm64 go build -o ../platforms/electron/bundle/services/api/macos-arm64/<api-binary>
GOOS=windows GOARCH=amd64 go build -o ../platforms/electron/bundle/services/api/windows-x64/<api-binary>.exe

# Build UI
cd scenarios/<scenario-name>/ui
pnpm run build
cp -r dist ../platforms/electron/bundle/ui/

# Build Electron installers
cd scenarios/<scenario-name>/platforms/electron
pnpm install
pnpm run dist:all
```

**Output in `dist-electron/`:**
- `<app>-1.0.0.msi` - Windows installer
- `<app>-1.0.0-mac.pkg` - macOS installer
- `<app>-1.0.0.AppImage` - Linux portable
- `<app>-1.0.0_amd64.deb` - Debian/Ubuntu package

### Phase 7: Configure Auto-Updates (Optional)

```json
{
  "update_config": {
    "channel": "stable",
    "provider": "github",
    "github": {"owner": "your-org", "repo": "<app>-desktop"},
    "auto_check": true
  }
}
```

See [Auto-Updates Guide](guides/auto-updates.md) for full configuration.

### Phase 8: Monitor Deployed Apps

```bash
# View telemetry logs
deployment-manager logs <profile-id> --format table

# Filter by error level
deployment-manager logs <profile-id> --level error

# Search logs
deployment-manager logs <profile-id> --search "migration"
```

Telemetry is stored in `~/.config/<app-name>/telemetry/deployment-telemetry.jsonl`.

---

## Quick Reference

### Essential Commands

```bash
# Check compatibility
deployment-manager fitness <scenario> --tier 2

# Create profile
deployment-manager profile create <name> <scenario> --tier 2

# Apply swap
deployment-manager swaps apply <profile-id> postgres sqlite

# Check secrets
deployment-manager secrets identify <profile-id>

# Validate
deployment-manager validate <profile-id> --verbose

# Package (via scenario-to-desktop)
deployment-manager package <profile-id> --packager scenario-to-desktop
```

### Tier Mapping

| Input | Tier |
|-------|------|
| `local`, `1` | Tier 1 (Local Dev) |
| `desktop`, `2` | Tier 2 (Desktop) |
| `mobile`, `ios`, `android`, `3` | Tier 3 (Mobile) |
| `saas`, `cloud`, `web`, `4` | Tier 4 (SaaS) |
| `enterprise`, `on-prem`, `5` | Tier 5 (Enterprise) |

### Common Swaps for Desktop

| From | To | Fitness Impact |
|------|----|----------------|
| `postgres` | `sqlite` | +25 desktop |
| `redis` | `in-process` | +10 desktop |
| `ollama` | packaged models | Enables offline |
| `browserless` | `playwright-driver` | +Chromium bundled |

---

## Troubleshooting

### "Fitness score is 0"

**Cause**: Critical blocker preventing deployment

**Solution**:
```bash
deployment-manager fitness <scenario> --tier 2
# Check blockers array in output
# Apply required swaps
```

### "Cannot connect to deployment-manager API"

**Solution**:
```bash
vrooli scenario start deployment-manager
API_PORT=$(vrooli scenario port deployment-manager API_PORT)
deployment-manager configure api_base "http://localhost:${API_PORT}"
```

### "Secrets validation failed"

**Cause**: Infrastructure secrets cannot be bundled

**Solution**: Apply swaps to remove dependencies that require infrastructure secrets (e.g., postgres → sqlite removes POSTGRES_PASSWORD).

### "Bundle manifest validation failed"

**Cause**: Manifest doesn't match schema v0.1

**Solution**: Ensure required fields exist:
- `schema_version`: "v0.1"
- `target`: "desktop"
- `app.name`, `app.version`
- At least one service

### "App starts but can't reach API"

**For thin client**: Verify `server_url` points to running Tier 1 server

**For bundled app**:
1. Check runtime supervisor started (look for `runtime_start` in telemetry)
2. Verify IPC port: `curl http://127.0.0.1:<ipc_port>/healthz`
3. Check service logs in app data directory

---

## Deep Dive Documentation

| Topic | Document |
|-------|----------|
| Full desktop workflow | [workflows/desktop-deployment.md](workflows/desktop-deployment.md) |
| CLI command reference | [cli/README.md](cli/README.md) |
| REST API reference | [api/README.md](api/README.md) |
| Fitness scoring details | [guides/fitness-scoring.md](guides/fitness-scoring.md) |
| Dependency swapping | [guides/dependency-swapping.md](guides/dependency-swapping.md) |
| Secrets management | [guides/secrets-management.md](guides/secrets-management.md) |
| Auto-updates | [guides/auto-updates.md](guides/auto-updates.md) |
| Bundle manifest examples | [examples/manifests/](examples/manifests/) |
| Tier 2 technical reference | [tiers/tier-2-desktop.md](tiers/tier-2-desktop.md) |
| Implementation roadmap | [ROADMAP.md](ROADMAP.md) |
| Known issues | [PROBLEMS.md](PROBLEMS.md) |

---

## For Agents

When helping users deploy scenarios:

1. **Start here** - This guide provides the exact command sequence
2. **Check status table** - Know what works vs. what needs workarounds
3. **Use troubleshooting** - Common errors and solutions are documented
4. **Reference deep dives** - Link to detailed docs when users need more info
5. **Note the gaps** - Be upfront about what's not yet automated

### Key Implementation Files

| Purpose | Location |
|---------|----------|
| CLI entry point | `scenarios/deployment-manager/cli/app.go` |
| API routes | `scenarios/deployment-manager/api/server/routes.go` |
| Bundle handler | `scenarios/deployment-manager/api/bundles/handler.go` |
| Manifest schema | `docs/examples/manifests/desktop-happy.json` |
| Runtime supervisor | `scenarios/scenario-to-desktop/runtime/supervisor.go` |
| Bundle packager | `scenarios/scenario-to-desktop/api/bundle_packager.go` |
