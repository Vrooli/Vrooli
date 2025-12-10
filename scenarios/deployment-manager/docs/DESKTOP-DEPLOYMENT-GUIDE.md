# Desktop Deployment Guide (Tier 2)

> **This guide has been consolidated into the workflows directory.**
>
> See: **[workflows/desktop-deployment.md](workflows/desktop-deployment.md)** for the complete desktop deployment workflow.

## Quick Reference

```bash
# Automated workflow
./scripts/desktop-quick-start.sh <scenario-name>

# Manual workflow summary
deployment-manager fitness <scenario> --tier 2           # Check compatibility
deployment-manager profile create my-desktop <scenario> --tier 2  # Create profile
deployment-manager swaps apply <profile-id> postgres sqlite       # Apply swaps
deployment-manager bundle export <scenario> --output bundle.json  # Generate manifest
cd scenarios/<scenario>/platforms/electron && pnpm run dist:all   # Build installers
```

## Documentation Map

| Topic | Location |
|-------|----------|
| **Full Workflow** | [workflows/desktop-deployment.md](workflows/desktop-deployment.md) |
| **CLI Commands** | [cli/deployment-commands.md](cli/deployment-commands.md) |
| **Bundle Schema** | [guides/bundle-manifest-schema.md](guides/bundle-manifest-schema.md) |
| **Auto-Updates** | [guides/auto-updates.md](guides/auto-updates.md) |
| **Secrets Management** | [guides/secrets-management.md](guides/secrets-management.md) |
| **Troubleshooting** | [workflows/troubleshooting.md](workflows/troubleshooting.md) |
| **Tier 2 Reference** | [tiers/tier-2-desktop.md](tiers/tier-2-desktop.md) |

## Key Decision: Thin Client vs Bundled

```
Do you need the app to work offline?
├── YES → Bundled Mode (full workflow)
└── NO  → Thin Client Mode (simpler, connects to Tier 1 server)
```

See the [full workflow guide](workflows/desktop-deployment.md#which-desktop-mode-do-you-need) for detailed comparison.
