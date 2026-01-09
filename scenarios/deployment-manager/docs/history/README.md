# Historical Documentation

Legacy documentation preserved for reference. This content describes approaches that have been superseded by the current deployment-manager system.

> **Warning**: The guidance in these documents is outdated. Use the current [workflows](../workflows/README.md) and [guides](../guides/README.md) instead.

## Archived Documents

| Document | Era | Notes |
|----------|-----|-------|
| [K8s Legacy](k8s-legacy.md) | Pre-tiered | Old Kubernetes deployment approach |
| [Packaging Script](packaging-script.md) | Pre-tiered | Original package-scenario-deployment.sh |
| [Vault Legacy](vault-legacy.md) | Pre-tiered | Old secrets approach |

## Why These Are Archived

The deployment-manager scenario replaces these approaches with:

1. **Tiered model** - Different deployment targets with fitness scoring
2. **Manifest-driven** - `bundle.json` instead of shell scripts
3. **Orchestrated** - Coordinated via deployment-manager API/CLI
4. **Secret classification** - Infrastructure vs service vs user secrets

## When to Reference

These docs may still be useful for:

- Understanding historical decisions
- Migrating old deployments to new system
- Reference when adding SaaS/enterprise tiers

## Current Approach

See the main [documentation hub](../README.md) for the current deployment system.
