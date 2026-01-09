# Quick Start Guide

Get your first Vrooli scenario deployed to a VPS in minutes.

## Prerequisites

Before you begin, ensure you have:

- **A VPS** with SSH access (Ubuntu 22.04+ recommended)
- **A domain** pointing to your VPS IP address
- **SSH key** configured for passwordless login
- **Vrooli running** locally with your scenario

## Step 1: Start the Wizard

From the Dashboard, click **Start New Deployment** to launch the deployment wizard.

## Step 2: Configure Your Manifest

The manifest defines what gets deployed and where. At minimum, you need:

```json
{
  "scenario": {
    "id": "your-scenario-name"
  },
  "target": {
    "vps": {
      "host": "your-server.com",
      "user": "root"
    }
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

## Step 4: Generate Plan

The plan shows exactly what will happen during deployment:
1. Bundle creation
2. File transfer to VPS
3. Vrooli setup
4. Resource startup
5. Scenario deployment
6. HTTPS configuration

## Step 5: Build Bundle

The bundle is a minimal Vrooli installation containing:
- Core Vrooli scripts
- Your scenario files
- Required resources
- Configuration files

## Step 6: Preflight Checks

Preflight verifies your VPS is ready:
- SSH connectivity
- Disk space
- Required tools
- Port availability

## Step 7: Deploy!

Click **Deploy to VPS** to start the deployment. This typically takes 2-5 minutes.

Once complete, your scenario will be live at `https://your-domain.com`!

## Next Steps

- [Manifest Reference](guides/manifest-reference.md) - Full configuration options
- [Troubleshooting](guides/troubleshooting.md) - Common issues and fixes
- [Deployment Lifecycle](reference/deployment-lifecycle.md) - Understanding status transitions
