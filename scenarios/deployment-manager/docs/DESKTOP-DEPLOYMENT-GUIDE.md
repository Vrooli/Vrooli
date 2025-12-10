# Desktop Deployment Guide (Tier 2)

> **Quick path**: Use `deploy-desktop` for end-to-end bundled desktop builds in one command.

## Quick Start (Recommended)

```bash
# Create a profile for your scenario
deployment-manager profile create my-profile my-scenario --tier 2

# Build everything: manifest, binaries, Electron wrapper, installers
deployment-manager deploy-desktop --profile my-profile
```

This single command orchestrates the full 7-step pipeline. See [deploy-desktop CLI reference](cli/deployment-commands.md#deploy-desktop) for options.

## Quick Reference

```bash
# Full automated pipeline (recommended)
deployment-manager profile create my-desktop <scenario> --tier 2
deployment-manager deploy-desktop --profile my-desktop

# Dry-run to preview
deployment-manager deploy-desktop --profile my-desktop --dry-run

# Partial pipeline options
deployment-manager deploy-desktop --profile my-desktop --skip-installers  # No installers
deployment-manager deploy-desktop --profile my-desktop --platforms win,linux  # Specific platforms
```

## Documentation Map

| Topic | Location |
|-------|----------|
| **Tutorial** | [tutorials/hello-desktop-walkthrough.md](tutorials/hello-desktop-walkthrough.md) |
| **Full Workflow** | [workflows/desktop-deployment.md](workflows/desktop-deployment.md) |
| **CLI Commands** | [cli/deployment-commands.md](cli/deployment-commands.md) |
| **Bundle Schema** | [guides/bundle-manifest-schema.md](guides/bundle-manifest-schema.md) |
| **Auto-Updates** | [guides/auto-updates.md](guides/auto-updates.md) |
| **Secrets Management** | [guides/secrets-management.md](guides/secrets-management.md) |
| **Troubleshooting** | [workflows/troubleshooting.md](workflows/troubleshooting.md) |
| **Tier 2 Reference** | [tiers/tier-2-desktop.md](tiers/tier-2-desktop.md) |

## Thin Client vs Bundled

| Mode | Use When | Command |
|------|----------|---------|
| **Bundled** | Offline use, distribution | `deploy-desktop --profile <id>` |
| **Thin Client** | Internal tools, always online | scenario-to-desktop direct API |

See [workflow guide](workflows/desktop-deployment.md#which-desktop-mode-do-you-need) for detailed comparison.
