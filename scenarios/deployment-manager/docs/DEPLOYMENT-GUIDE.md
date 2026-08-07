# Deployment-manager guide

`deployment-manager` is the governance plane for scenario deployment. It does
not replace a target packager. It analyzes the dependency graph, stores target
profiles, evaluates readiness, gates promotion, and records what shipped.

For project-wide terminology and maturity, read the [Deployment Hub](../../../docs/deployment/README.md).
For desktop implementation, read the [scenario-to-desktop documentation](../../scenario-to-desktop/docs/OVERVIEW.md).

## Ownership

| Concern | deployment-manager | Target ramp |
| --- | --- | --- |
| Dependency graph and fitness | Owns | Consumes |
| Target profile and swaps | Owns | Consumes |
| Approval and release gate | Owns | Requests |
| Build and packaging | Does not own | Owns |
| Native runtime execution | Does not own | Owns |
| Evidence production | Validates references and decides | Produces |
| Publication | Records decision | Performs after approval |

The control direction is from the ramp to deployment-manager. A packager asks
for a decision; deployment-manager does not launch an unrequested packager.

## Current target model

These are technical deployment targets, not commercial delivery tiers:

| Target | Meaning | Status |
| --- | --- | --- |
| `1` / `local` | Full Vrooli stack on an operator-controlled host | Current reference path |
| `2` / `desktop` | Windows, macOS, or Linux desktop application | Implemented ramp; target claims are evidence-gated |
| `3` / `mobile` | iOS or Android application | Directional |
| `4` / `saas` | Hosted cloud installation | Reference/planning |
| `5` / `enterprise` | Appliance or enterprise deployment | Strategic framing |

Use descriptive names in prose. Numeric aliases remain for CLI compatibility.

## General workflow

1. Confirm the scenario runs through the Tier 1 lifecycle.
2. Create a target profile.
3. Analyze dependencies and target fitness.
4. Apply only documented, compatible swaps.
5. Validate secret strategies, licenses, host requirements, and artifact
   availability.
6. Ask the target ramp to build and exercise the artifact.
7. Review target evidence and exact source/artifact identity.
8. Approve or reject the release gate.
9. Publish only after approval and record the release.

The target ramp owns the implementation details for steps 6 and 9. Do not copy
those steps into a second deployment-manager workflow.

## Desktop entry point

```bash
deployment-manager profile create my-profile my-scenario --tier 2
deployment-manager analyze my-scenario
deployment-manager fitness my-scenario --tier 2
deployment-manager deploy-desktop --profile my-profile --platforms linux --timeout 20m
```

The desktop command can produce an artifact without producing a promotable
release. Inspect its dependency plan, warnings, evidence, and release-trust
state before distribution. See the [desktop workflow](workflows/desktop-deployment.md)
for the governance steps and the [desktop ramp quickstart](../../scenario-to-desktop/docs/QUICKSTART.md)
for target-specific operation.

## Other targets

Mobile, SaaS, and enterprise pages describe direction and constraints. They are
not promises of a working packager. Use the [Deployment Hub](../../../docs/deployment/README.md)
and the relevant target page to determine whether a path is current, conditional,
or merely planned.

## Troubleshooting and references

- [Troubleshooting](workflows/troubleshooting.md)
- [CLI overview](cli/overview-commands.md)
- [Profile commands](cli/profile-commands.md)
- [Fitness scoring](guides/fitness-scoring.md)
- [Dependency swapping](guides/dependency-swapping.md)
- [Bundle manifest schema](guides/bundle-manifest-schema.md)
- [Evidence contract](guides/evidence-contract.md)
- [Release authority](../../../docs/configuration/release-authority.md)
