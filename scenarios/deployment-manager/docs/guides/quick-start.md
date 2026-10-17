# Deployment quickstart

This is the short governance path. It intentionally stops before target-native
implementation details.

## Prerequisites

```bash
vrooli scenario start deployment-manager
vrooli scenario start scenario-to-desktop
```

The target scenario must run successfully through the Tier 1 lifecycle.

## Create and inspect a desktop profile

```bash
deployment-manager profile create my-profile my-scenario --tier 2
deployment-manager profile show my-profile
deployment-manager analyze my-scenario
deployment-manager fitness my-scenario --tier 2
```

Review dependency limitations, host requirements, secret strategies, and
compatible swaps. A fitness score does not prove native runtime support.

## Validate and build

```bash
deployment-manager validate my-profile --verbose
deployment-manager deploy-desktop \
  --profile my-profile \
  --platforms linux \
  --timeout 20m
```

The resulting artifact can be non-promotable when evidence or release trust is
missing. Inspect the target plan and evidence before distribution.

## Next steps

- [Desktop workflow](../workflows/desktop-deployment.md)
- [Scenario-to-desktop quickstart](../../../scenario-to-desktop/docs/QUICKSTART.md)
- [Fitness scoring](fitness-scoring.md)
- [Dependency swapping](dependency-swapping.md)
- [Troubleshooting](../workflows/troubleshooting.md)
