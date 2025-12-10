# Desktop Deployment Quickstart

> **Deploy a scenario as a standalone desktop app in under 5 minutes.**

This guide walks you through deploying a scenario as a bundled desktop application using the deployment-manager CLI.

## Prerequisites

1. **deployment-manager running**: `vrooli scenario start deployment-manager`
2. **scenario-to-desktop running**: `vrooli scenario start scenario-to-desktop`
3. **Your scenario has a built UI**: `cd scenarios/<your-scenario>/ui && pnpm run build`

## Reference Implementation

Use **`hello-desktop`** as the reference scenario for desktop deployment. It has:
- Zero external dependencies (no postgres, redis, ollama)
- Minimal Go API + static HTML UI
- Pre-configured desktop deployment metadata

```bash
# Quick validation that the pipeline works
deployment-manager profile create test-profile hello-desktop --tier 2
deployment-manager deploy-desktop --profile test-profile --dry-run
```

## Step 1: Create a Deployment Profile (30 seconds)

```bash
# Create a profile for your scenario targeting desktop
deployment-manager profile create my-desktop-profile <scenario-name> --tier 2
```

**Example:**
```bash
deployment-manager profile create picker-wheel-desktop picker-wheel --tier 2
```

## Step 2: Check Fitness & Apply Swaps (60 seconds)

```bash
# Check if your scenario is ready for desktop
deployment-manager fitness <scenario-name> --tier 2

# If fitness < 0.8, check what swaps are needed
deployment-manager swaps list <scenario-name>

# Apply recommended swaps
deployment-manager swaps apply <profile-id> postgres sqlite
deployment-manager swaps apply <profile-id> redis memory-cache
```

**Common Desktop Swaps:**
| From | To | When to use |
|------|-----|-------------|
| postgres | sqlite | Single-user file-based storage |
| redis | memory-cache | In-process caching |
| ollama | packaged-model | Offline AI (larger bundle) |

## Step 3: Deploy to Desktop (3 minutes)

```bash
# Full end-to-end deployment (manifest + binaries + Electron + installers)
deployment-manager deploy-desktop --profile my-desktop-profile
```

**Options:**
```bash
# Specific platforms only
deployment-manager deploy-desktop --profile my-desktop-profile --platforms win,mac

# Skip installer builds (faster iteration)
deployment-manager deploy-desktop --profile my-desktop-profile --skip-installers

# Thin client mode (connects to server, no bundled services)
deployment-manager deploy-desktop --profile my-desktop-profile --mode external-server

# Preview without executing
deployment-manager deploy-desktop --profile my-desktop-profile --dry-run
```

## Step 4: Get Your Installers

On success, you'll see:

```
✓ Desktop Deployment: success
  Profile:  picker-wheel-desktop
  Scenario: picker-wheel
  Duration: 4m32s

Steps:
  ✓ Load profile - loaded profile for scenario picker-wheel
  ✓ Validate profile - profile validation passed
  ✓ Assemble manifest - assembled manifest with 2 swaps
  ✓ Export manifest - wrote manifest to /home/.../platforms/electron/bundle/bundle.json
  ✓ Build binaries - built 2 service(s) for 3 platform(s)
  ✓ Generate desktop wrapper - generated Electron wrapper at /home/.../platforms/electron
  ✓ Build platform installers - built installers for 3 platform(s)

Installers:
  win:     /home/.../platforms/electron/dist-electron/picker-wheel-1.0.0.msi
  mac:     /home/.../platforms/electron/dist-electron/picker-wheel-1.0.0.pkg
  linux:   /home/.../platforms/electron/dist-electron/picker-wheel-1.0.0.AppImage
```

## Deployment Modes

| Mode | Use Case | Bundle Size |
|------|----------|-------------|
| `bundled` | Offline standalone app | Large (100MB+) |
| `external-server` | Thin client connecting to Vrooli server | Small (~50MB) |
| `cloud-api` | UI connecting to cloud-hosted API | Small (~50MB) |

## Troubleshooting

### "scenario-to-desktop not available"
Start the scenario-to-desktop service:
```bash
vrooli scenario start scenario-to-desktop
```

### "fitness score too low"
Apply dependency swaps to make the scenario desktop-compatible:
```bash
deployment-manager swaps list <scenario>
deployment-manager swaps apply <profile> postgres sqlite
```

### Build failures
Check the build logs:
```bash
deployment-manager logs <profile> --level error
```

### Missing UI
Build the UI first:
```bash
cd scenarios/<scenario>/ui && pnpm run build
```

## Full Workflow Example

```bash
# 1. Start required services
vrooli scenario start deployment-manager
vrooli scenario start scenario-to-desktop

# 2. Create profile
deployment-manager profile create myapp-desktop my-scenario --tier 2

# 3. Check and fix fitness
deployment-manager fitness my-scenario --tier 2
deployment-manager swaps apply myapp-desktop postgres sqlite

# 4. Deploy
deployment-manager deploy-desktop --profile myapp-desktop

# 5. Test the installer
# On Linux:
./scenarios/my-scenario/platforms/electron/dist-electron/my-scenario-1.0.0.AppImage

# On macOS:
open ./scenarios/my-scenario/platforms/electron/dist-electron/my-scenario-1.0.0.pkg

# On Windows:
msiexec /i scenarios\my-scenario\platforms\electron\dist-electron\my-scenario-1.0.0.msi
```

## Next Steps

- [Full CLI Reference](cli/deploy-desktop.md)
- [Bundle Manifest Schema](guides/bundle-manifest-schema.md)
- [Dependency Swapping Guide](guides/dependency-swapping.md)
- [Telemetry & Analytics](guides/telemetry-guide.md)
- [Troubleshooting Guide](workflows/troubleshooting.md)
