# Desktop deployment workflow

This workflow covers the governance path for a deployment Tier 2 desktop
release. It explains what deployment-manager decides. The target-specific build,
runtime, and evidence procedures live in [scenario-to-desktop](../../../scenario-to-desktop/docs/OVERVIEW.md).

## Choose the deployment shape

| Shape | Use when | Requires |
| --- | --- | --- |
| Bundled private | The app must run without a Vrooli server | Eligible local routes for every required capability |
| External-server thin client | A Tier 1 server remains the source of truth | Reachable, authenticated scenario API |
| Shared provider | A desktop app may reuse an existing provider | Broker consent, scoped lease, expiry, and provider evidence |
| Tier 2 peer | Desktop applications need direct communication | Not available until the peer protocol is implemented |

Do not call a thin client an offline app. Do not call a provider candidate a
peer protocol.

## Prerequisites

The target scenario must run through the Tier 1 lifecycle. The deployment
services should be available:

```bash
vrooli scenario start deployment-manager
vrooli scenario start scenario-to-desktop
```

The dependency analyzer and credential services are started according to the
local deployment flow. If a required classification or artifact decision is
unavailable, the release must show that state rather than silently omit it.

## 1. Create a profile

Use a named profile to make the target decision reproducible:

```bash
deployment-manager profile create my-profile my-scenario --tier 2
deployment-manager profile show my-profile
```

The profile is operator intent. It is not a release approval.

## 2. Analyze dependencies and fitness

```bash
deployment-manager analyze my-scenario
deployment-manager fitness my-scenario --tier 2
```

Review the complete dependency graph, not only the scenario’s direct imports.
The result should identify:

- required scenarios and resources;
- RAM, CPU, disk, GPU, network, and host-tool requirements;
- platform and architecture coverage;
- licensing or distribution limitations;
- secret classes and provisioning strategies;
- supported, conditional, degraded, and unsupported routes;
- compatible dependency swaps and their functional limitations.

A fitness score is a decision aid. It is not a substitute for target evidence.

## 3. Resolve blockers explicitly

When a dependency is not desktop-compatible, inspect the available swaps:

```bash
deployment-manager swaps list my-scenario
deployment-manager swaps analyze postgres sqlite
deployment-manager profile swap my-profile add postgres sqlite
```

Every swap must record its reason and limitations. A swap is not valid merely
because the replacement starts: the scenario must use the replacement correctly,
and migrations or data compatibility must be proven.

Infrastructure secrets must be removed by a swap or an explicit remote route.
They must never be copied into a desktop artifact.

## 4. Validate the target plan

Run a dry run before building a release artifact:

```bash
deployment-manager deploy-desktop \
  --profile my-profile \
  --platforms linux \
  --dry-run \
  --timeout 20m
```

Review the generated plan for each resource:

- selected deployment mode;
- platform and architecture;
- privilege and host requirements;
- bundling policy;
- artifact identity and checksum;
- fallback and limitation;
- health and readiness contract;
- secret treatment;
- promotability.

An unsupported required resource is a terminal runtime limitation. A conditional
resource must display its host requirement before installation. A build may be
useful for investigation while remaining non-promotable.

## 5. Build and exercise the artifact

Run the pipeline after the plan is understood:

```bash
deployment-manager deploy-desktop \
  --profile my-profile \
  --platforms linux \
  --timeout 20m
```

The ramp owns the following work:

1. generate the Electron wrapper;
2. stage the scenario and selected resource artifacts;
3. verify the manifest and checksums;
4. run preflight;
5. start the private runtime when bundled mode is selected;
6. validate API, dependency, secret, and readiness behavior;
7. run the native user journey;
8. build the platform installer or portable artifact;
9. persist the evidence references.

Follow [scenario-to-desktop quickstart](../../../scenario-to-desktop/docs/QUICKSTART.md),
[build and packaging](../../../scenario-to-desktop/docs/guides/build-and-packaging.md),
and [smoke-test pipeline](../../../scenario-to-desktop/docs/reference/smoke-test-pipeline.md)
for implementation details.

## 6. Review evidence

The following are separate claims:

| Claim | Required proof |
| --- | --- |
| Source/build correctness | Build result bound to an exact source revision |
| Package integrity | Artifact hashes and release-manifest trust |
| Dependency eligibility | Target plan and resource readiness results |
| Native behavior | App launch, interaction, service operation, and clean shutdown |
| Communication | Machine assertion of the route and provider identity |
| Visual journey | Reviewer-visible, ordered, redacted evidence |
| Update behavior | Update detection, replacement, relaunch, and failure handling |

Compile or package success does not prove native behavior. A recording does not
prove communication. `unsupported` and `unavailable` are terminal evidence
states, not degraded passes. The authoritative labels and sidecar requirements
are in the [desktop evidence contract](../../../../docs/reference/scenario-to-desktop-evidence-and-tier-contract.md).

## 7. Gate and publish

The ramp asks deployment-manager for the release decision using the exact target
identity and evidence set. Publish only after the gate permits it. The release
record must retain:

- source revision;
- target OS and architecture;
- profile and target-plan identity;
- artifact hashes;
- evidence references;
- warnings and limitations;
- approval and promotion state.

Use the [managed release authority](../../../../docs/configuration/release-authority.md)
for release-manifest signing. OS installer signing and release-manifest signing
are separate trust mechanisms.

## Failure handling

When the workflow stops, preserve the distinction between:

- build failure;
- target ineligibility;
- missing or unavailable credential provider;
- missing user credential;
- service transport failure;
- service readiness failure;
- communication/authentication failure;
- insufficient evidence;
- rejected release gate.

Do not “fix” a failed release by removing a dependency, downgrading a required
service to optional, or marking an unavailable environment as passed.

## Related references

- [Deployment Hub](../../../../docs/deployment/README.md)
- [Tier 2 desktop contract](../tiers/tier-2-desktop.md)
- [Bundle manifest schema](../guides/bundle-manifest-schema.md)
- [Secrets management](../guides/secrets-management.md)
- [Evidence contract](../guides/evidence-contract.md)
- [Troubleshooting](troubleshooting.md)
