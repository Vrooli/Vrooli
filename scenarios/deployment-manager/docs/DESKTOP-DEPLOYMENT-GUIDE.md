# Desktop deployment guide

This file is retained as a stable link for existing references. The desktop
workflow is maintained at [workflows/desktop-deployment.md](workflows/desktop-deployment.md).
The desktop packager’s implementation and runtime behavior are maintained in
the [scenario-to-desktop documentation](../../scenario-to-desktop/docs/OVERVIEW.md).

## Quick path

```bash
deployment-manager profile create my-profile my-scenario --tier 2
deployment-manager deploy-desktop --profile my-profile --platforms linux --timeout 20m
```

This command starts the target pipeline. It does not, by itself, prove that the
artifact is safe to promote or that every target platform runs natively.

Read next:

- [Desktop workflow](workflows/desktop-deployment.md)
- [Tier 2 desktop contract](tiers/tier-2-desktop.md)
- [Scenario-to-desktop quickstart](../../scenario-to-desktop/docs/QUICKSTART.md)
- [Desktop evidence contract](../../../docs/reference/scenario-to-desktop-evidence-and-tier-contract.md)
