## Tools focus: Scenario to Desktop

Convert any Vrooli scenario into a cross-platform desktop application (Windows, macOS, Linux) using the scenario-to-desktop CLI. This skill covers the full pipeline: generation, building, signing, distribution, and telemetry.

The scenario-to-desktop tool creates Electron wrappers that connect to your running scenario. Currently, **thin client mode (external-server) is production-ready**; bundled mode is under development.

Required reading:
- `prompt-manager skills read skill-principles`

---

### **1. When to Use This Tool**

| Scenario | CLI Command |
|----------|-------------|
| Convert web app to desktop | `scenario-to-desktop pipeline-run <scenario> --platforms win,mac,linux` |
| Check build status | `scenario-to-desktop pipeline-status <id>` |
| Download built installers | `scenario-to-desktop download <scenario> <platform>` |
| Collect deployment telemetry | `scenario-to-desktop telemetry-ingest <scenario> --file <path>` |
| Configure code signing | `scenario-to-desktop signing-set <scenario> --config <json>` |
| Distribute to users | `scenario-to-desktop distribute <scenario> --artifacts <paths>` |

**When NOT to use this tool:**
- Scenarios without a UI (API-only scenarios)
- Mobile app generation (out of scope)
- Bundled offline apps (not production-ready yet)

---

### **1.1 Scope Boundaries**

**In scope:**
- Converting scenarios with web UIs to cross-platform desktop apps
- Building installers for Windows (NSIS), macOS (ZIP/DMG), Linux (AppImage/DEB)
- Code signing configuration and validation
- Distribution to GitHub Releases, S3, and other targets
- Deployment telemetry collection and analysis

**Out of scope:**
- API-only scenarios (no UI)
- Mobile app generation (iOS/Android)
- Bundled/offline apps (not production-ready)
- Web deployment or SaaS packaging

---

### **2. Prerequisites**

Before generating desktop apps:

1. **Target scenario is running and reachable**
   ```bash
   curl -s http://localhost:<SCENARIO_PORT>/api/health || \
   curl -s https://app-monitor.<domain>/apps/<scenario>/proxy/api/health
   ```

2. **Scenario has a built UI**
   - Check: `scenarios/<scenario>/ui/dist/` exists and contains built assets

3. **Start scenario-to-desktop**
   ```bash
   cd scenarios/scenario-to-desktop
   make start    # PREFERRED
   # or: vrooli scenario start scenario-to-desktop

   # Verify it's running
   scenario-to-desktop status
   ```

---

### **3. CLI Command Reference**

**Global Options** (apply to all commands):
- `--api-base <url>`: Override API base URL (default: auto-detected)
- `--auto-start`: Auto-start the scenario if not running
- `--no-color`: Disable ANSI color output
- `--color`: Force-enable ANSI color output

#### **3.1 Health & Status**

```bash
# Check API health and system status
scenario-to-desktop status [--json]
```

#### **3.2 Templates**

```bash
# List available desktop templates
scenario-to-desktop templates [--json]

# Get specific template details
scenario-to-desktop template <type>
# Types: universal (or 'basic' alias), advanced, multi_window, kiosk
# Note: 'basic' is a backward compatibility alias for 'universal'
```

#### **3.3 Pipeline (Main Workflow)**

The pipeline is the primary way to convert a scenario to desktop. It runs stages: bundle, preflight, generate, build, smoketest, distribution.

```bash
# Start a full pipeline run (async - returns immediately)
scenario-to-desktop pipeline-run <scenario> [--stages <stages>] [--platforms win,mac,linux]

# Start and wait for completion (recommended for agents/scripts)
scenario-to-desktop pipeline-run <scenario> --wait [--timeout 600]

# Check pipeline status
scenario-to-desktop pipeline-status <id> [--verbose] [--json]

# List all pipelines
scenario-to-desktop pipeline-list [--json]

# Get active pipeline for a scenario
scenario-to-desktop pipeline-active <scenario> [--no-create]

# Create a new pipeline for a scenario
scenario-to-desktop pipeline-create <scenario>

# Start the active pipeline (async)
scenario-to-desktop pipeline-start <scenario> [--stages <stages>] [--platforms <platforms>]

# Start the active pipeline and wait (recommended for agents/scripts)
scenario-to-desktop pipeline-start <scenario> --wait [--timeout 600]

# Resume a stopped pipeline
scenario-to-desktop pipeline-resume <id>

# Cancel a running pipeline
scenario-to-desktop pipeline-cancel <id>

# Reset active pipeline
scenario-to-desktop pipeline-reset <scenario>

# View pipeline history
scenario-to-desktop pipeline-history <scenario> [--limit N]
```

**Pipeline stages:** `bundle`, `preflight`, `generate`, `build`, `distribution`, `smoketest`

Note: The `distribution` stage is automatically skipped unless `--distribute` is specified. The `smoketest` stage runs after distribution to validate the final artifacts.

**Blocking mode flags:**
- `--wait`: Block until pipeline completes (recommended for agents and scripts)
- `--timeout N`: Max wait time in seconds (default: 600, i.e., 10 minutes)

**Example: Full pipeline run**
```bash
# Run full pipeline for all platforms (async)
scenario-to-desktop pipeline-run my-scenario --platforms win,mac,linux

# Run and wait for completion (blocking)
scenario-to-desktop pipeline-run my-scenario --platforms linux --wait --timeout 900

# Run only generate and build stages for Linux
scenario-to-desktop pipeline-run my-scenario --stages generate,build --platforms linux
```

**Note:** If `--platforms` is omitted, all platforms (win, mac, linux) are built by default. Verify platform selection in `pipeline-status` output under the `config.platforms` field.

#### **3.4 Download Built Packages**

```bash
# Download built installer for a platform
scenario-to-desktop download <scenario> <platform> [--output <path>]
# Platforms: win, mac, linux

# Examples
scenario-to-desktop download my-scenario win --output my-app-setup.exe
scenario-to-desktop download my-scenario linux --output my-app.AppImage
scenario-to-desktop download my-scenario mac --output my-app.zip
```

#### **3.5 Desktop Records**

```bash
# List all desktop generation records
scenario-to-desktop records [--json]

# Move desktop wrapper to destination
scenario-to-desktop records-move <id> [--target destination|custom] [--path <path>]

# Delete desktop app for a scenario
scenario-to-desktop records-delete <scenario>
```

#### **3.6 Telemetry**

Collect and analyze telemetry from deployed desktop apps.

```bash
# Ingest telemetry from file
scenario-to-desktop telemetry-ingest <scenario> --file <path> [--source cli]

# Get telemetry summary
scenario-to-desktop telemetry-summary <scenario> [--json]

# Get AI-generated insights
scenario-to-desktop telemetry-insights <scenario> [--json]

# Get recent telemetry entries
scenario-to-desktop telemetry-tail <scenario> [--limit N] [--json]

# Download telemetry file
scenario-to-desktop telemetry-download <scenario> [--output <path>]

# Delete telemetry data
scenario-to-desktop telemetry-delete <scenario>
```

**Telemetry file locations by platform:**
- Windows: `%APPDATA%\<App Name>\deployment-telemetry.jsonl`
- macOS: `~/Library/Application Support/<App Name>/deployment-telemetry.jsonl`
- Linux: `~/.config/<App Name>/deployment-telemetry.jsonl`

#### **3.7 Code Signing**

Configure code signing for trusted distribution.

```bash
# Get signing configuration
scenario-to-desktop signing-get <scenario> [--json]

# Set signing configuration
scenario-to-desktop signing-set <scenario> --config <json|@file.json>

# Delete signing configuration
scenario-to-desktop signing-delete <scenario>

# Validate signing configuration
scenario-to-desktop signing-validate <scenario>

# Check if signing is ready for all platforms
scenario-to-desktop signing-ready <scenario> [--json]

# List available signing tools
scenario-to-desktop signing-prerequisites [--json]

# Discover certificates on a platform
scenario-to-desktop signing-discover <platform>
# Platforms: windows, macos, linux

# Generate GPG key for Linux signing
scenario-to-desktop signing-generate-key <scenario> \
  --name "Developer Name" \
  --email "dev@example.com" \
  [--passphrase <pass>] [--passphrase-env <var>] [--force]
```

#### **3.8 Distribution**

Distribute built packages to various targets (GitHub Releases, S3, etc.).

```bash
# List distribution targets
scenario-to-desktop dist-targets [--json]

# Get distribution target details
scenario-to-desktop dist-target-get <name> [--json]

# Create distribution target
scenario-to-desktop dist-target-create --config <json|@file.json>

# Update distribution target
scenario-to-desktop dist-target-update <name> --config <json|@file.json>

# Delete distribution target
scenario-to-desktop dist-target-delete <name>

# Test distribution target connectivity
scenario-to-desktop dist-target-test <name>

# Validate all distribution targets
scenario-to-desktop dist-validate [--json]

# Check distribution credentials
scenario-to-desktop dist-check-credentials [--json]

# Start distribution
scenario-to-desktop distribute <scenario> --artifacts <paths> [--targets <names>]

# Check distribution status
scenario-to-desktop dist-status <id> [--json]

# Cancel distribution
scenario-to-desktop dist-cancel <id>

# List all distributions
scenario-to-desktop dist-list [--json]
```

#### **3.9 Wine (Windows Builds on Linux)**

Wine is required for building Windows installers on Linux.

```bash
# Check Wine installation status
scenario-to-desktop wine-check [--json]

# Install Wine
scenario-to-desktop wine-install --method <flatpak|flatpak-auto|appimage>

# Check Wine installation status
scenario-to-desktop wine-status <install_id> [--json]
```

#### **3.10 Configuration**

```bash
# Configure CLI settings
scenario-to-desktop configure

# Set API base URL
scenario-to-desktop configure api_base

# Set API token
scenario-to-desktop configure token
```

---

### **4. Common Workflows**

#### **4.1 Basic Desktop App Generation**

```bash
# 1. Start the scenario-to-desktop service
cd scenarios/scenario-to-desktop && make start

# 2. Verify system status
scenario-to-desktop status

# 3. Run the full pipeline
scenario-to-desktop pipeline-run my-scenario --platforms win,mac,linux

# 4. Monitor progress
scenario-to-desktop pipeline-status <pipeline-id> --verbose

# 5. Download built installers
scenario-to-desktop download my-scenario win
scenario-to-desktop download my-scenario mac
scenario-to-desktop download my-scenario linux

# 6. Verify downloads
ls -la *.exe *.AppImage *.zip  # Check files exist
file my-scenario.AppImage      # Should show: ELF 64-bit LSB executable
```

**Pipeline success verification:**
```bash
# Confirm pipeline completed
scenario-to-desktop pipeline-status <id> --json | jq '.status'
# Expected: "completed"
```

#### **4.2 Collect User Telemetry**

```bash
# After users run the desktop app, collect their telemetry
# 1. Get the telemetry file from user (platform-specific paths above)

# 2. Ingest the telemetry
scenario-to-desktop telemetry-ingest my-scenario --file user-telemetry.jsonl

# 3. View summary
scenario-to-desktop telemetry-summary my-scenario

# 4. Get AI insights
scenario-to-desktop telemetry-insights my-scenario
```

#### **4.3 Set Up Code Signing**

```bash
# 1. Check what signing tools are available
scenario-to-desktop signing-prerequisites

# 2. Discover existing certificates
scenario-to-desktop signing-discover windows
scenario-to-desktop signing-discover macos
scenario-to-desktop signing-discover linux

# 3. For Linux, generate a GPG key if needed
scenario-to-desktop signing-generate-key my-scenario \
  --name "My Company" --email "dev@mycompany.com"

# 4. Set signing configuration
scenario-to-desktop signing-set my-scenario --config @signing-config.json

# 5. Validate configuration
scenario-to-desktop signing-validate my-scenario

# 6. Check readiness
scenario-to-desktop signing-ready my-scenario
```

#### **4.4 Distribute to GitHub Releases**

```bash
# 1. Create GitHub distribution target
scenario-to-desktop dist-target-create --config '{
  "name": "github-releases",
  "type": "github",
  "enabled": true,
  "config": {
    "repo": "owner/repo",
    "token_env": "GITHUB_TOKEN"
  }
}'

# 2. Test the target
scenario-to-desktop dist-target-test github-releases

# 3. Distribute built artifacts
scenario-to-desktop distribute my-scenario \
  --artifacts "dist/my-app.exe,dist/my-app.AppImage,dist/my-app.zip" \
  --targets github-releases

# 4. Monitor distribution
scenario-to-desktop dist-status <distribution-id>

# 5. Verify distribution completed
scenario-to-desktop dist-status <distribution-id> --json | jq '.status'
# Expected: "completed"

# For GitHub Releases, verify via gh CLI
gh release view v<version> --repo owner/repo
```

#### **4.5 Agent/Scripted Workflows**

For automated pipelines (agents, CI/CD, scripts), always use `--wait` to block until completion:

```bash
# Run and wait - blocks until complete, fail, or timeout
scenario-to-desktop pipeline-run my-scenario --platforms linux --wait --timeout 900

# Check exit code
if [ $? -eq 0 ]; then
    echo "Success - downloading artifacts"
    scenario-to-desktop download my-scenario linux
else
    echo "Pipeline failed"
    exit 1
fi
```

**Why use `--wait` for agents:**
1. Single command to start and wait for completion
2. Proper exit codes for scripting (0 = success, non-zero = failure)
3. No need to poll `pipeline-status` manually
4. Timeout protection prevents indefinite hangs

**Exit codes (with `--wait`):**

| Code | Meaning |
|------|---------|
| 0 | Completed successfully |
| Non-zero | Failed (check error in output) |

**Full automated workflow example:**
```bash
#!/bin/bash
set -e

SCENARIO="my-scenario"
PLATFORMS="linux"
TIMEOUT=900

echo "Building desktop app for $SCENARIO..."
scenario-to-desktop pipeline-run "$SCENARIO" --platforms "$PLATFORMS" --wait --timeout "$TIMEOUT"

echo "Downloading artifacts..."
scenario-to-desktop download "$SCENARIO" linux --output "${SCENARIO}.AppImage"

echo "Build complete: ${SCENARIO}.AppImage"
```

---

### **5. Deployment Modes**

| Mode | Status | CLI Flag |
|------|--------|----------|
| `external-server` (Thin Client) | Production-ready | Default |
| `cloud-api` | Stub | Future |
| `bundled` | Stub | Future |

---

### **6. Cross-Platform Build Matrix**

Building from Linux (recommended for single-machine builds):

| Target | Format | Build on Linux | Notes |
|--------|--------|----------------|-------|
| Linux | AppImage, DEB | Native | Recommended |
| Windows | NSIS (.exe) | Via Wine | Works reliably |
| Windows | MSI | Via Wine (unstable) | Use CI for production |
| macOS | ZIP | Native | Contains .app bundle |
| macOS | DMG, PKG | macOS only | Requires `hdiutil` |

---

### **7. Troubleshooting**

#### **7.1 Wine Build Issues**

**"spawn wine ENOENT"**
```bash
# Check Wine status
scenario-to-desktop wine-check

# Install Wine if needed
scenario-to-desktop wine-install --method flatpak-auto
```

**"Wine: installed ... - not usable"**
Wine is installed but not properly configured for use.
```bash
# Try reinstalling with a different method
scenario-to-desktop wine-install --method appimage

# Or verify flatpak Wine configuration
flatpak run --command=wine org.winehq.Wine --version
```

#### **7.2 Pipeline Failures**

```bash
# Check detailed status
scenario-to-desktop pipeline-status <id> --verbose

# Reset and retry
scenario-to-desktop pipeline-reset my-scenario
scenario-to-desktop pipeline-start my-scenario
```

#### **7.3 Signing Issues**

```bash
# Validate signing configuration
scenario-to-desktop signing-validate my-scenario

# Check readiness per platform
scenario-to-desktop signing-ready my-scenario --json
```

#### **7.4 Bundled Mode Errors**

**"Error: Bundled payload is missing"**
This error indicates bundled mode was attempted, which is not production-ready.

```bash
# Ensure you're using thin client mode (default)
scenario-to-desktop pipeline-run my-scenario --platforms linux
# Do NOT use --deployment-mode bundled (not supported yet)
```

#### **7.5 Telemetry Issues**

**"Error: invalid telemetry format"**
```bash
# Validate JSONL format
head -1 telemetry.jsonl | jq .

# Check for corrupted lines
jq -c . telemetry.jsonl 2>&1 | grep -n "parse error"
```

**"No telemetry data available"**
- Verify the app has been run at least once
- Check platform-specific telemetry paths (Section 3.6)
- Ensure user has read permissions on the telemetry file

**Telemetry file not found**
```bash
# Platform-specific default locations:
# Windows: %APPDATA%\<App Name>\deployment-telemetry.jsonl
# macOS:   ~/Library/Application Support/<App Name>/deployment-telemetry.jsonl
# Linux:   ~/.config/<App Name>/deployment-telemetry.jsonl
```

---

### **8. Guardrails**

**Do:**
- Always use lifecycle commands: `make start`, `make stop`
- Verify target scenario is running before generation
- Use thin client mode for production deployments
- Collect telemetry from testers for debugging

**Do NOT:**
- Run API binary directly (`./api/scenario-to-desktop-api`)
- Use bundled mode in production (not ready yet)
- Expect MSI builds to work reliably via Wine
- Build DMG/PKG on Linux (requires macOS)

---

### **9. Output Expectations**

**You may:**
- Generate desktop wrappers for any scenario with a UI
- Build installers for Windows, macOS, and Linux
- Configure code signing for trusted distribution
- Distribute to various targets (GitHub, S3, etc.)
- Collect and analyze deployment telemetry

**You must:**
- Verify target scenario is running before generation
- Use appropriate deployment mode (thin client for production)
- Test generated apps before distribution
- Document any build issues in the scenario's PROBLEMS.md

**You must NOT:**
- Bypass lifecycle commands for starting/stopping services
- Use bundled mode in production (not ready)
- Distribute apps without testing on target platforms
- Ignore telemetry from user deployments
