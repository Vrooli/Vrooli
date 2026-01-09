# Quick Start Guide

Deploy your first scenario in 5 minutes.

## Prerequisites

```bash
# Verify Vrooli is installed
vrooli --version

# Start deployment-manager
vrooli scenario start deployment-manager
```

## Step 1: Check Scenario Fitness (30 seconds)

```bash
# Replace 'picker-wheel' with your scenario name
deployment-manager fitness picker-wheel --tier desktop
```

**Expected output:**

```json
{
  "scenario": "picker-wheel",
  "tier": 2,
  "scores": { "overall": 45 },
  "blockers": ["postgres requires swap to sqlite"],
  "warnings": []
}
```

- **Score 80+**: Ready to deploy - skip to Step 4
- **Score 40-79**: Has blockers - continue to Step 2
- **Score < 40**: Significant work needed - see [Dependency Swapping Guide](dependency-swapping.md)

## Step 2: Create a Deployment Profile (30 seconds)

```bash
deployment-manager profile create my-desktop-app picker-wheel --tier 2
```

Save the returned `id` (e.g., `profile-1704067200`).

## Step 3: Apply Required Swaps (1 minute)

If you have blockers, apply the suggested swaps:

```bash
# Apply postgres → sqlite swap
deployment-manager swaps apply profile-1704067200 postgres sqlite --show-fitness

# Apply redis → in-process swap (if needed)
deployment-manager swaps apply profile-1704067200 redis in-process --show-fitness
```

Verify fitness improved:

```bash
deployment-manager profile analyze profile-1704067200
```

## Step 4: Configure Secrets (1 minute)

```bash
# See what secrets are needed
deployment-manager secrets identify profile-1704067200

# Generate a template
deployment-manager secrets template profile-1704067200 --format env
```

**Secret types:**
- `per_install_generated`: Auto-created (no action needed)
- `user_prompt`: User provides on first run (no action needed)
- `infrastructure`: **Blocked** - ensure swaps removed these

## Step 5: Validate Before Building (30 seconds)

```bash
deployment-manager validate profile-1704067200 --verbose
```

All checks should pass. If not, address the issues shown.

## Step 6: Generate Bundle Manifest (1 minute)

```bash
# Get API port
API_PORT=$(vrooli scenario port deployment-manager API_PORT)

# Generate and save the bundle manifest
curl -X POST "http://localhost:${API_PORT}/api/v1/bundles/export" \
  -H "Content-Type: application/json" \
  -d '{"scenario": "picker-wheel", "tier": "tier-2-desktop"}' \
  > bundle.json
```

## Step 7: Build Desktop Installers (3-5 minutes)

### Option A: Package via CLI

```bash
deployment-manager package profile-1704067200 --packager scenario-to-desktop
```

### Option B: Manual Build

```bash
# Navigate to Electron app directory
cd scenarios/picker-wheel/platforms/electron

# Copy bundle manifest
cp /path/to/bundle.json bundle/

# Install dependencies
pnpm install

# Build all platforms
pnpm run dist:all
```

## Step 8: Find Your Installers

```bash
ls -la scenarios/picker-wheel/platforms/electron/dist-electron/
```

Output:

```
picker-wheel-1.0.0.msi          # Windows
picker-wheel-1.0.0-mac.pkg      # macOS
picker-wheel-1.0.0.AppImage     # Linux (portable)
picker-wheel-1.0.0_amd64.deb    # Linux (Debian)
```

## What's Next?

- [Desktop Deployment Workflow](../workflows/desktop-deployment.md) - Full workflow with all options
- [Auto-Updates Guide](auto-updates.md) - Set up automatic updates
- [Troubleshooting](../workflows/troubleshooting.md) - Common issues and solutions

## Quick Reference

```bash
# Check fitness
deployment-manager fitness <scenario> --tier 2

# Create profile
deployment-manager profile create <name> <scenario> --tier 2

# Apply swap
deployment-manager swaps apply <profile-id> <from> <to>

# Check secrets
deployment-manager secrets identify <profile-id>

# Validate
deployment-manager validate <profile-id> --verbose

# Package
deployment-manager package <profile-id> --packager scenario-to-desktop
```

## Troubleshooting Quick Fixes

**"Cannot connect to deployment-manager API"**

```bash
vrooli scenario start deployment-manager
```

**"Fitness score is 0"**

Check blockers with `fitness` command and apply required swaps.

**"Secrets validation failed"**

Infrastructure secrets can't be bundled. Apply swaps to remove them (e.g., postgres → sqlite).

**"Package command failed"**

Ensure scenario-to-desktop is running:

```bash
vrooli scenario start scenario-to-desktop
```
