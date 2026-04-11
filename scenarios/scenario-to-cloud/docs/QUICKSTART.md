# Quick Start Guide

Get your first Vrooli scenario deployed to a VPS in minutes.

## Prerequisites

Before you begin, ensure you have:

- **A VPS** with SSH access (Ubuntu 24.04 recommended; 22.04/20.04 compatible)
- **A domain** pointing to your VPS IP address
- **SSH key** configured for passwordless login
- **Vrooli running** locally with your scenario

## Step 1: Start the Wizard

From the Dashboard, click **Start New Deployment** to launch the deployment wizard.

## Step 2: Configure Your Manifest

Use `manifest init` to generate a starter manifest:

```bash
scenario-to-cloud manifest init \
  --scenario your-scenario-name \
  --host your-server.com \
  --domain app.your-domain.com \
  --out cloud-manifest.json
```

Equivalent minimal contract:

```json
{
  "version": "1.0.0",
  "scenario": {
    "id": "your-scenario-name"
  },
  "target": {
    "type": "vps",
    "vps": {
      "host": "your-server.com",
      "user": "root"
    }
  },
  "dependencies": {
    "scenarios": ["your-scenario-name"],
    "resources": []
  },
  "bundle": {
    "include_packages": true,
    "include_autoheal": true
  },
  "ports": {
    "ui": 3000,
    "api": 3001,
    "ws": 3002
  },
  "edge": {
    "domain": "app.your-domain.com"
  }
}
```

### Key Fields

| Field | Description |
|-------|-------------|
| `scenario.id` | The name of your scenario folder |
| `target.vps.host` | Your VPS hostname or IP |
| `target.vps.user` | SSH user (usually `root`) |
| `edge.domain` | The domain for HTTPS access |

## Step 3: Validate

Click **Validate** to check your manifest for errors. Common issues:
- Invalid JSON syntax
- Missing required fields
- Unreachable host

Optional: show canonical runtime policy:

```bash
scenario-to-cloud preflight requirements
```

## Step 4: Generate Plan

The plan shows exactly what will happen during deployment:
1. Bundle creation
2. File transfer to VPS
3. Native `vrooli` CLI upload to the deployment workdir
4. Vrooli setup
5. Resource startup
6. Scenario deployment
7. HTTPS configuration

## Step 5: Build Bundle

The bundle is a minimal Vrooli installation containing:
- Your scenario files
- Required resources
- Shared package modules required by the deployment
- Generated deployment metadata

The setup flow then uploads a deployment-local native `vrooli` binary to
`<workdir>/.vrooli/bin/vrooli` and uses that exact binary for all remote setup,
deploy, inspect, and stop operations. It does not rely on a legacy bootstrap
script or a preinstalled global CLI on the VPS.

## Step 6: Preflight Checks

Preflight verifies your VPS is ready:
- SSH connectivity
- Disk space
- Required tools
- Port availability

Before deploy, validate SSH access path:

```bash
scenario-to-cloud ssh bootstrap your-server.com --user root --non-interactive
```

## Step 7: Resolve + Deploy If Needed

Check current state by selector (no manifest required). If deployment is missing, create and validate a persistent manifest, then deploy:

```bash
scenario-to-cloud deployment health --host your-server.com --scenario your-scenario-name --json

scenario-to-cloud manifest init \
  --scenario your-scenario-name \
  --host your-server.com \
  --domain app.your-domain.com \
  --out scenarios/your-scenario-name/.vrooli/cloud/manifest.prod.json

scenario-to-cloud manifest validate scenarios/your-scenario-name/.vrooli/cloud/manifest.prod.json
scenario-to-cloud redeploy scenarios/your-scenario-name/.vrooli/cloud/manifest.prod.json --if-needed --preflight --wait
```

This typically takes 2-5 minutes when deployment is required.

Once complete, your scenario will be live at `https://your-domain.com`!

## Next Steps

- [Manifest Reference](guides/manifest-reference.md) - Full configuration options
- [Troubleshooting](guides/troubleshooting.md) - Common issues and fixes
- [Deployment Lifecycle](reference/deployment-lifecycle.md) - Understanding status transitions
