# Desktop Deployment Workflow (Tier 2)

Deploy a Vrooli scenario as a standalone Windows, macOS, or Linux desktop application.

---

## Which Desktop Mode Do You Need?

Before starting, determine which deployment mode fits your requirements:

```
Do you need the app to work offline / without a server?
│
├── YES → Bundled Mode (Recommended: deploy-desktop command)
│         • Self-contained: UI + API + resources in single installer
│         • Works offline after first-run setup
│         • Requires dependency swaps (postgres → sqlite, etc.)
│         • Status: ✅ Fully automated via deploy-desktop
│
└── NO → Thin Client Mode (Phase 6A)
          • Bundles UI only; connects to running Tier 1 server
          • Requires network access to Vrooli server
          • No dependency swaps needed
          • Status: ✅ Fully automated
```

### Quick Decision Matrix

| Requirement | Thin Client | Bundled |
|-------------|-------------|---------|
| Works offline | No | Yes |
| Self-contained installer | Yes (UI only) | Yes (full app) |
| Requires Tier 1 server | Yes | No |
| Automation status | **Fully automated** | **Fully automated** |
| Setup complexity | Low | Medium |
| Bundle size | Small (~50MB) | Large (100MB-1GB+) |
| Best for | Internal tools, connected environments | Distribution, offline use |

---

## Implementation Status

| Mode | Status | Description |
|------|--------|-------------|
| **Thin Client** | ✅ Working | UI bundled in Electron; connects to running Tier 1 server |
| **Bundled App** | ✅ Working | UI + API + resources via `deploy-desktop` command |

> **Quick Start**: For bundled desktop apps, use the single-command approach:
> ```bash
> deployment-manager profile create my-profile my-scenario --tier 2
> deployment-manager deploy-desktop --profile my-profile
> ```
> This orchestrates the entire 7-step pipeline automatically. See [deploy-desktop CLI reference](../cli/deployment-commands.md#deploy-desktop) for details.

> **Reference Implementation**: The `hello-desktop` scenario demonstrates a complete working bundled desktop build. See [Hello Desktop Tutorial](../tutorials/hello-desktop-walkthrough.md).

## Prerequisites

### Required Tools

```bash
# Verify Vrooli CLI is installed
vrooli --version

# Verify deployment-manager scenario is running
vrooli scenario status deployment-manager
# If not running:
vrooli scenario start deployment-manager

# Verify scenario-to-desktop is available
vrooli scenario status scenario-to-desktop
# If not running:
vrooli scenario start scenario-to-desktop

# Verify Node.js and npm/pnpm for Electron builds
node --version    # v18+ required
pnpm --version    # or npm --version
```

### Required Scenarios

| Scenario | Purpose | Required? |
|----------|---------|-----------|
| `deployment-manager` | Orchestration, fitness scoring, profiles | Yes |
| `scenario-to-desktop` | Electron wrapper generation | Yes |
| `scenario-dependency-analyzer` | Dependency DAG computation | Yes (auto-started) |
| `secrets-manager` | Secret classification | Yes (auto-started) |

### Target Scenario Requirements

The scenario you want to deploy must:
- Have a working UI (React/Vite preferred)
- Have a working API (Go preferred for cross-platform binaries)
- Be running successfully in Tier 1 (local dev stack)

---

## Phase 1: Check Compatibility

### Step 1.1: Verify API Health

```bash
deployment-manager status
```

**Expected output:**
```json
{
  "status": "healthy",
  "api_version": "1.1.0",
  "dependencies": {
    "scenario-dependency-analyzer": "healthy",
    "secrets-manager": "healthy"
  }
}
```

### Step 1.2: Analyze Dependencies

```bash
deployment-manager analyze <scenario-name>
```

**Example:**
```bash
deployment-manager analyze picker-wheel
```

**Expected output:**
```json
{
  "scenario": "picker-wheel",
  "dependencies": {
    "resources": ["postgres", "redis"],
    "scenarios": []
  },
  "circular_dependencies": false,
  "resource_requirements": {
    "ram_mb": 512,
    "disk_mb": 100,
    "gpu": false
  },
  "tiers": {
    "1": {"overall": 100, "portability": 100, "resources": 100},
    "2": {"overall": 45, "portability": 30, "resources": 60},
    ...
  }
}
```

### Step 1.3: Check Desktop Fitness Score

```bash
deployment-manager fitness <scenario-name> --tier 2
# OR
deployment-manager fitness <scenario-name> --tier desktop
```

**Expected output:**
```json
{
  "scenario": "picker-wheel",
  "tier": 2,
  "scores": {
    "overall": 45,
    "portability": 30,
    "resources": 60,
    "licensing": 100,
    "platform_support": 50
  },
  "blockers": [
    "postgres requires swap to sqlite for desktop bundling",
    "redis requires swap to in-process cache"
  ],
  "warnings": [
    "Large dependency tree may increase bundle size"
  ]
}
```

### Interpreting Fitness Scores

| Score Range | Meaning | Action |
|-------------|---------|--------|
| 80-100 | Excellent | Proceed to Phase 3 |
| 60-79 | Good | Review warnings, proceed |
| 40-59 | Fair | Address blockers in Phase 2 |
| 0-39 | Poor | Significant swaps required |

---

## Phase 2: Address Blocking Issues

If your fitness score has blockers, resolve them with dependency swaps.

### Step 2.1: List Available Swaps

```bash
deployment-manager swaps list <scenario-name>
```

**Expected output:**
```json
{
  "scenario": "picker-wheel",
  "suggestions": [
    {
      "from": "postgres",
      "to": "sqlite",
      "impact": {
        "desktop": "+25",
        "mobile": "+40"
      },
      "pros": ["No server required", "Single file database", "Cross-platform"],
      "cons": ["Single-user only", "No replication"],
      "migration_effort": "medium"
    },
    {
      "from": "redis",
      "to": "in-process",
      "impact": {"desktop": "+10"},
      "pros": ["No external process"],
      "cons": ["Memory-only, no persistence"],
      "migration_effort": "low"
    }
  ]
}
```

### Step 2.2: Analyze Swap Impact

```bash
deployment-manager swaps analyze postgres sqlite
```

**Expected output:**
```json
{
  "from": "postgres",
  "to": "sqlite",
  "fitness_delta": {
    "tier_1": 0,
    "tier_2": 25,
    "tier_3": 40,
    "tier_4": -5,
    "tier_5": -10
  },
  "affected_scenarios": ["picker-wheel"],
  "migration_steps": [
    "Update DATABASE_URL format",
    "Run schema migration to SQLite dialect",
    "Update connection pooling configuration"
  ]
}
```

### Step 2.3: Check Cascading Effects

```bash
deployment-manager swaps cascade postgres sqlite
```

This shows if other scenarios depend on the one you're modifying.

---

## Phase 3: Create Deployment Profile

### Step 3.1: Create a New Profile

```bash
deployment-manager profile create <profile-name> <scenario-name> --tier 2
```

**Example:**
```bash
deployment-manager profile create picker-wheel-desktop picker-wheel --tier 2
```

**Expected output:**
```json
{
  "id": "profile-1234567890",
  "name": "picker-wheel-desktop",
  "scenario": "picker-wheel",
  "tiers": [2],
  "version": 1,
  "created_at": "2025-01-15T12:00:00Z"
}
```

### Step 3.2: Apply Dependency Swaps

```bash
deployment-manager swaps apply <profile-id> postgres sqlite --show-fitness
```

**Or using profile name:**
```bash
deployment-manager profile swap <profile-id> add postgres sqlite
```

### Step 3.3: Configure Environment

```bash
# Set environment variables for the profile
deployment-manager profile set <profile-id> env LOG_LEVEL info
deployment-manager profile set <profile-id> env DEBUG false
```

### Step 3.4: View Profile Configuration

```bash
deployment-manager profile show <profile-id>
```

**Expected output:**
```json
{
  "id": "profile-1234567890",
  "name": "picker-wheel-desktop",
  "scenario": "picker-wheel",
  "tiers": [2],
  "swaps": {
    "postgres": "sqlite"
  },
  "settings": {
    "env": {
      "LOG_LEVEL": "info",
      "DEBUG": "false"
    }
  },
  "version": 3
}
```

### Step 3.5: Export Profile (Optional)

```bash
deployment-manager profile export <profile-id> --output ./picker-wheel-desktop-profile.json
```

---

## Phase 4: Configure Secrets

### Step 4.1: Identify Required Secrets

```bash
deployment-manager secrets identify <profile-id>
```

**Expected output:**
```json
{
  "profile_id": "profile-1234567890",
  "secrets": [
    {
      "id": "jwt_secret",
      "type": "required",
      "source": "per_install_generated",
      "description": "JWT signing key"
    },
    {
      "id": "session_secret",
      "type": "required",
      "source": "user_prompt",
      "description": "Session signing secret"
    },
    {
      "id": "postgres_password",
      "type": "infrastructure",
      "source": "infrastructure",
      "description": "Database password - WILL NOT BE BUNDLED"
    }
  ]
}
```

### Secret Classifications

| Class | Meaning | Bundle Behavior |
|-------|---------|-----------------|
| `per_install_generated` | Auto-generated on first run | Included in manifest with generator config |
| `user_prompt` | User provides during first-run wizard | Included in manifest with prompt config |
| `remote_fetch` | Fetched from external vault | Reference only, fetched at runtime |
| `infrastructure` | Never leaves Tier 1 | **Excluded from bundle** |

### Step 4.2: Generate Secret Template

```bash
# For desktop deployment (.env format)
deployment-manager secrets template <profile-id> --format env
```

**Expected output:**
```env
# Picker Wheel Desktop - Secrets Template
# Generated: 2025-01-15T12:00:00Z

# === PER-INSTALL GENERATED (auto-created on first run) ===
# JWT_SECRET will be auto-generated (32 char alphanumeric)

# === USER PROVIDED (prompted during first-run wizard) ===
SESSION_SECRET=  # Enter a 32+ character random string

# === INFRASTRUCTURE (excluded from bundle) ===
# POSTGRES_PASSWORD - Not included; using SQLite swap instead
```

### Step 4.3: Validate Secrets Configuration

```bash
deployment-manager secrets validate <profile-id>
```

---

## Phase 5: Generate Bundle Manifest

### Step 5.1: Validate Profile for Deployment

```bash
deployment-manager validate <profile-id> --verbose
```

**Expected output:**
```json
{
  "profile_id": "profile-1234567890",
  "valid": true,
  "checks": [
    {"name": "fitness_threshold", "status": "pass", "details": "Score 70 >= 50"},
    {"name": "secrets_complete", "status": "pass", "details": "All required secrets configured"},
    {"name": "licensing", "status": "pass", "details": "All dependencies OSS-compatible"},
    {"name": "resource_limits", "status": "pass", "details": "RAM 256MB within desktop limits"},
    {"name": "platform_requirements", "status": "pass", "details": "Binaries available for win/mac/linux"},
    {"name": "dependency_compatibility", "status": "pass", "details": "All swaps applied"}
  ]
}
```

### Step 5.2: Assemble Bundle Manifest

```bash
# Assemble the bundle manifest (preview without checksum)
deployment-manager bundle assemble picker-wheel --tier desktop

# Or output directly to a file
deployment-manager bundle assemble picker-wheel --tier desktop --output manifest.json
```

**Expected output:**
```json
{
  "status": "assembled",
  "schema": "desktop.v0.1",
  "manifest": {
    "schema_version": "v0.1",
    "target": "desktop",
    "app": {"name": "picker-wheel", "version": "1.2.3"},
    "services": [...],
    "swaps": [...],
    "secrets": [...]
  }
}
```

### Step 5.3: Export Bundle Manifest with Checksum

```bash
# Export production-ready manifest with SHA256 checksum
deployment-manager bundle export picker-wheel --output bundle.json
```

**Expected output:**
```
Bundle manifest exported to bundle.json
  Scenario:    picker-wheel
  Tier:        tier-2-desktop
  Schema:      desktop.v0.1
  Checksum:    a1b2c3d4e5f6...
  Generated:   2025-01-15T12:00:00Z
```

### Step 5.4: Validate Bundle Manifest

```bash
deployment-manager bundle validate ./bundle.json
```

**Expected output:**
```
Manifest is valid
  File:   ./bundle.json
  Schema: desktop.v0.1
```

<details>
<summary>Alternative: Using REST API directly</summary>

If you prefer using the REST API instead of the CLI:

```bash
# Get the deployment-manager API port
API_PORT=$(vrooli scenario port deployment-manager API_PORT)

# Assemble manifest
curl -X POST "http://localhost:${API_PORT}/api/v1/bundles/assemble" \
  -H "Content-Type: application/json" \
  -d '{"scenario": "picker-wheel", "tier": "tier-2-desktop", "include_secrets": true}'

# Export with checksum
curl -X POST "http://localhost:${API_PORT}/api/v1/bundles/export" \
  -H "Content-Type: application/json" \
  -d '{"scenario": "picker-wheel", "tier": "tier-2-desktop"}' > bundle.json

# Validate
curl -X POST "http://localhost:${API_PORT}/api/v1/bundles/validate" \
  -H "Content-Type: application/json" \
  -d @bundle.json
```

</details>

---

## Phase 6: Build Desktop Installers

### Option A: Bundled App (Recommended - Fully Automated)

Use the `deploy-desktop` command to orchestrate the entire pipeline:

```bash
# Create a profile if you haven't already
deployment-manager profile create picker-profile picker-wheel --tier 2

# Run the full pipeline (builds binaries, generates Electron, creates installers)
deployment-manager deploy-desktop --profile picker-profile
```

This single command:
1. Loads and validates the profile
2. Assembles the bundle manifest with profile swaps
3. Exports the manifest to the scenario
4. Cross-compiles API binaries for all platforms (linux-x64, darwin-arm64, darwin-x64, win-x64)
5. Generates the Electron wrapper via scenario-to-desktop
6. Builds platform installers (MSI, PKG, AppImage, DEB)

**Partial pipeline options:**

```bash
# Dry-run to preview
deployment-manager deploy-desktop --profile picker-profile --dry-run

# Skip installer builds (just manifest + binaries + Electron)
deployment-manager deploy-desktop --profile picker-profile --skip-installers

# Build for specific platforms only
deployment-manager deploy-desktop --profile picker-profile --platforms win,linux
```

**Output location:**

```
scenarios/picker-wheel/platforms/electron/dist-electron/
├── Picker Wheel Setup 1.0.0.exe     # Windows NSIS installer
├── Picker Wheel-1.0.0-mac.zip       # macOS app bundle
├── Picker Wheel-1.0.0.AppImage      # Linux portable
├── picker-wheel_1.0.0_amd64.deb     # Debian/Ubuntu package
├── latest.yml                        # Windows update manifest
├── latest-mac.yml                    # macOS update manifest
└── latest-linux.yml                  # Linux update manifest
```

### Option B: Thin Client (UI Only)

Bundles UI only; requires connection to Tier 1 server.

```bash
# Get scenario-to-desktop API port
STD_PORT=$(vrooli scenario port scenario-to-desktop API_PORT)

# Generate thin client
curl -X POST "http://localhost:${STD_PORT}/api/v1/desktop/generate/quick" \
  -H "Content-Type: application/json" \
  -d '{
    "scenario_name": "picker-wheel",
    "deployment_mode": "external-server",
    "server_url": "https://your-vrooli-server.example.com"
  }'
```

### Option C: Manual Assembly (Advanced/Debugging)

<details>
<summary>Click to expand manual steps (not recommended for normal use)</summary>

If automation fails or you need fine-grained control, you can manually assemble the bundle:

#### Step 6.1: Prepare the Bundle Directory

```bash
cd scenarios/picker-wheel/platforms/electron
mkdir -p bundle
cp /path/to/bundle.json bundle/
```

#### Step 6.2: Build Platform Binaries

```bash
cd scenarios/picker-wheel/api

# Linux
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ../platforms/electron/bundle/bin/linux-x64/picker-wheel-api

# macOS (Intel)
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o ../platforms/electron/bundle/bin/darwin-x64/picker-wheel-api

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o ../platforms/electron/bundle/bin/darwin-arm64/picker-wheel-api

# Windows
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o ../platforms/electron/bundle/bin/win-x64/picker-wheel-api.exe
```

#### Step 6.3: Build UI Bundle

```bash
cd scenarios/picker-wheel/ui
pnpm run build
cp -r dist ../platforms/electron/bundle/ui/
```

#### Step 6.4: Build Electron Installers

```bash
cd scenarios/picker-wheel/platforms/electron
pnpm install
pnpm run dist:all
```

</details>

---

## Phase 7: Configure Auto-Updates

### Step 7.1: Choose Update Provider

| Provider | Best For | Setup Effort |
|----------|----------|--------------|
| GitHub Releases | Open source apps | Low |
| Self-Hosted | Enterprise/private | Medium |
| None | Air-gapped environments | None |

### Step 7.2: GitHub Releases Setup

1. Create a GitHub repository for releases (e.g., `your-org/picker-wheel-desktop`)
2. Configure in generation request:

```json
{
  "update_config": {
    "channel": "stable",
    "provider": "github",
    "github": {
      "owner": "your-org",
      "repo": "picker-wheel-desktop"
    },
    "auto_check": true
  }
}
```

### Step 7.3: Publish a Release

```bash
# Tag the release
git tag v1.0.0
git push origin v1.0.0

# Upload artifacts to GitHub Release:
# - picker-wheel-1.0.0.msi
# - picker-wheel-1.0.0-mac.pkg
# - picker-wheel-1.0.0.AppImage
# - latest.yml, latest-mac.yml, latest-linux.yml
```

### Step 7.4: Update Channels

| Channel | GitHub Release Type | Users |
|---------|---------------------|-------|
| `dev` | Pre-release | Internal testing |
| `beta` | Pre-release | Beta testers |
| `stable` | Release | Production users |

See [Auto-Updates Guide](../guides/auto-updates.md) for detailed configuration.

---

## Phase 8: Monitor Deployed Apps

### Step 8.1: Telemetry Collection

Bundled apps emit telemetry to `deployment-telemetry.jsonl`:

```json
{"ts": "2025-01-15T12:00:00Z", "event": "app_start", "details": {"version": "1.0.0"}}
{"ts": "2025-01-15T12:00:05Z", "event": "ready", "details": {"startup_ms": 5000}}
{"ts": "2025-01-15T12:05:00Z", "event": "update_check_started", "details": {"channel": "stable"}}
```

### Step 8.2: Upload Telemetry

If `telemetry.upload_to` is configured in the manifest, telemetry uploads automatically.

For manual upload:
```bash
API_PORT=$(vrooli scenario port deployment-manager API_PORT)

curl -X POST "http://localhost:${API_PORT}/api/v1/telemetry/upload" \
  -H "Content-Type: application/json" \
  -d @deployment-telemetry.jsonl
```

### Step 8.3: View Telemetry in deployment-manager

```bash
# View logs for a profile
deployment-manager logs <profile-id> --format table

# Filter by level
deployment-manager logs <profile-id> --level error

# Search logs
deployment-manager logs <profile-id> --search "migration"
```

### Step 8.4: Access Telemetry UI

Navigate to the deployment-manager UI and click on "Telemetry" in the navigation to see:
- Total events per scenario
- Failure breakdown by type
- Recent events and errors
- Per-bundle statistics

---

## Known Gaps and Workarounds

### Resolved Issues

| Issue | Status | Resolution |
|-------|--------|------------|
| CLI bundle commands | ✅ Resolved | `bundle assemble`, `bundle export`, `bundle validate` available |
| Automated binary compilation | ✅ Resolved | `deploy-desktop` command handles cross-compilation |
| End-to-end pipeline validation | ✅ Resolved | `hello-desktop` scenario validates full workflow |
| Runtime supervisor bundling | ✅ Resolved | Automatically included by `deploy-desktop` |

### Outstanding Gaps

#### Gap 1: No Asset Procurement Automation

**Status**: Assets (model files, database seeds) must be manually placed

**Workaround**: Download/build assets before running `deploy-desktop`

**Tracking**: Add asset download/verification step to packager

#### Gap 2: Code Signing Not Automated

**Status**: Installers are unsigned by default

**Workaround**: Configure signing certificates manually in `package.json`:

```json
{
  "build": {
    "win": {
      "certificateFile": "./cert.pfx",
      "certificatePassword": "${WIN_CSC_KEY_PASSWORD}"
    },
    "mac": {
      "hardenedRuntime": true,
      "gatekeeperAssess": true
    }
  }
}
```

**Tracking**: Add certificate configuration to deploy-desktop flow

#### Gap 3: Clean-Machine Install Testing

**Status**: No automated clean-machine validation

**Workaround**: Manual testing on fresh VM/container

**Tracking**: Add CI step for clean-machine smoke test

---

## Troubleshooting

### "Fitness score is 0 for desktop"

**Cause**: Critical blocker preventing desktop deployment

**Solution**:
1. Run `deployment-manager fitness <scenario> --tier 2` to see blockers
2. Apply required swaps (usually postgres → sqlite)
3. Re-check fitness score

### "Cannot connect to deployment-manager API"

**Cause**: Scenario not running or wrong port

**Solution**:
```bash
vrooli scenario start deployment-manager
API_PORT=$(vrooli scenario port deployment-manager API_PORT)
deployment-manager configure api_base "http://localhost:${API_PORT}"
```

### "Bundle manifest validation failed"

**Cause**: Manifest doesn't match schema v0.1

**Solution**:
1. Check required fields: `schema_version`, `target`, `app.name`, `app.version`
2. Ensure at least one service is defined
3. Validate all secrets have valid `class` values

### "Electron build fails with missing dependencies"

**Cause**: Node modules not installed or wrong Electron version

**Solution**:
```bash
cd scenarios/<scenario>/platforms/electron
rm -rf node_modules
pnpm install
pnpm run dist:all
```

### "App starts but can't reach API"

**For thin client**: Verify `server_url` points to running Tier 1 server

**For bundled app**:
1. Check runtime supervisor started: look for `runtime_start` in telemetry
2. Verify IPC port matches manifest: `curl http://127.0.0.1:<ipc_port>/healthz`
3. Check service logs in app data directory

See [Troubleshooting Guide](troubleshooting.md) for more solutions.

---

## Quick Reference

### Essential Commands

```bash
# Check scenario compatibility
deployment-manager fitness <scenario> --tier 2

# Create deployment profile
deployment-manager profile create <name> <scenario> --tier 2

# Apply dependency swap
deployment-manager swaps apply <profile-id> postgres sqlite

# Configure secrets
deployment-manager secrets identify <profile-id>
deployment-manager secrets template <profile-id> --format env

# Validate before deployment
deployment-manager validate <profile-id> --verbose

# Build installers
cd scenarios/<scenario>/platforms/electron
pnpm run dist:all
```

### File Locations

| Artifact | Location |
|----------|----------|
| Bundle manifest | `scenarios/<scenario>/platforms/electron/bundle/bundle.json` |
| Built installers | `scenarios/<scenario>/platforms/electron/dist-electron/` |
| Update manifests | `dist-electron/latest*.yml` |
| Telemetry | `~/.config/<app-name>/telemetry/deployment-telemetry.jsonl` |
| Secrets | `~/.config/<app-name>/secrets.json` |
| Logs | `~/.config/<app-name>/logs/` |

---

## Related Documentation

- [Tier 2 Desktop Reference](../tiers/tier-2-desktop.md) - Technical details and roadmap
- [Bundled Runtime Plan](/docs/plans/bundled-desktop-runtime-plan.md) - Implementation plan
- [Auto-Updates Guide](../guides/auto-updates.md) - Update channel configuration
- [Dependency Swapping Guide](../guides/dependency-swapping.md) - Swap strategies
- [Secrets Management Guide](../guides/secrets-management.md) - Secret classification
- [Example Manifests](../examples/manifests/) - Reference bundle.json files
- [Picker Wheel Desktop Example](../examples/picker-wheel-desktop.md) - Real-world case study
