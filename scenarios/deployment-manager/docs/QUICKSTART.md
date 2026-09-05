# Desktop deployment quickstart

Use this page to run the deployment-manager governance path for a desktop
target. The target-specific build, runtime, secrets, and evidence behavior is
owned by [scenario-to-desktop](../../scenario-to-desktop/docs/OVERVIEW.md).

## Prerequisites

- `deployment-manager` and `scenario-to-desktop` are running through the
  lifecycle manager;
- the scenario has an honest dependency declaration and a built UI;
- the selected target has the required build and native-validation tools.

The `--tier 2` examples mean **technical deployment Tier 2: desktop**. They do
not refer to the commercial monetization tiers.

## Run the path

```bash
# Start the governance and target ramps
vrooli scenario start deployment-manager
vrooli scenario start scenario-to-desktop

# Create and inspect a desktop profile
deployment-manager profile create my-profile my-scenario --tier 2
deployment-manager analyze my-scenario
deployment-manager fitness my-scenario --tier 2

# Apply a declared, reviewed swap when the target plan requires one
deployment-manager swaps list my-scenario
deployment-manager swaps apply <profile-id> postgres sqlite

# Validate and build the primary Linux target
deployment-manager validate my-profile --verbose
deployment-manager deploy-desktop \
  --profile my-profile \
  --platforms linux \
  --timeout 20m
```

Use `--dry-run` to inspect the plan without building. Add `win` or `mac` only
when the corresponding artifact and native-validation environment are
available. A generated package is not proof that the application works on the
target.

## Choose the deployment mode

| Mode | Use when | Offline claim |
| --- | --- | --- |
| `bundled` | Every required dependency has an eligible private artifact | Only after native dependency evidence passes |
| `external-server` | The desktop shell calls a configured Tier 1 scenario API | No |
| `cloud-api` | A future cloud API integration is explicitly being developed | Not currently claimable |

For bundled mode, an unsupported required resource is a named blocker. Do not
omit it to improve a score. For thin-client mode, validate the real route and
show server-unavailable and authentication states.

## What to inspect before release

1. The target plan identifies every bundled, remote, conditional, and
   unsupported dependency.
2. Secret classifications and provisioning routes are explicit; values are
   never printed into manifests, logs, or evidence.
3. The artifact, trust metadata, source revision, and target platform match.
4. Native launch, dependency operation, communication, fallback, and shutdown
   evidence are attached to the exact artifact.
5. The release gate records `pass`, `failed`, `degraded`, `unavailable`,
   `unsupported`, or `not_run` rather than converting missing evidence into a
   pass.

## Troubleshooting

```bash
deployment-manager logs <profile-id> --level error
deployment-manager secrets identify <profile-id>
deployment-manager secrets validate <profile-id>
```

For target-specific failures, continue with the [desktop workflow](workflows/desktop-deployment.md),
the [scenario-to-desktop quickstart](../../scenario-to-desktop/docs/QUICKSTART.md),
and the [troubleshooting guide](workflows/troubleshooting.md).

