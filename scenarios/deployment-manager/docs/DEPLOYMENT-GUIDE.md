# Deployment-manager guide

`deployment-manager` is the governance plane for scenario deployment. It does
not replace a target packager. It analyzes the dependency graph, stores target
profiles, evaluates readiness, gates promotion, and records what shipped.

## Agent operating model

Deployment Manager is the durable release substrate. Its scenario-owned usage
skill teaches an agent how to choose and verify a deployment operation. Governed
programs compose repeated readiness, observation, and recovery workflows;
they must use declared bindings and must not duplicate release state or safety
invariants. The improve skill reads outcome and friction signals and routes the
repair to the cheapest correct layer:

- Repair the usage skill when selection, interpretation, or safety guidance was
  missing.
- Repair a governed program when a repeated composition is inefficient or loses
  evidence.
- Repair Deployment Manager when a program repeatedly compensates for a missing
  release primitive, invariant, migration rule, or recovery operation.

The governed programs are `readiness-review`, `release-preflight`,
`release-observe`, `release-recover`, and `setpoint-read`. Their contracts live
under `.vrooli/program-runtime`. Recovery is read-only by default and refuses
write mode when no target-owner mutation is bound; it never invents rollback
behavior inside the program.

## Evidence-complete readiness workflow

One immutable review identity includes scenario, profile, candidate commit,
artifact digest, canonical target set, channel, and policy version. Evidence
owners report small typed observations with `readiness-reviews report-evidence`.
The report is accepted only for the criterion's policy-declared producer binding.
`readiness-reviews prepare` reads those exact observations, resolves the latest
published predecessor for the same profile, targets, and channel, persists the
decision, and opens only unresolved work in Swarm Manager.

Independent customer-journey results use `readiness-reviews human-check`. After
the Swarm goal is actually archived, use `readiness-reviews sync`, then
`readiness-reviews approve` with the complete unchanged identity. Approval
revalidates evidence freshness and disposition, active waiver state, and passed
human checks. A release start must carry the exact review key and artifact digest;
publication marks that review promoted and supersedes the prior promoted review
for the same profile, targets, and channel.

Missing, stale, failed, unknown, or unavailable required evidence never becomes
a pass. A waiver must use `readiness-review-waivers create`; it is bound to one
review and criterion, records actor and reason, and expires within the policy
limit. The old caller-authored signal JSON remains only as a low-level producer
and test seam, not the agent workflow.

Readiness evidence is commit- and artifact-scoped. For a scenario with an actual
deployed predecessor, readiness also compares against that predecessor. Storage
conformance follows `storage-steer`: greenfield scenarios use declarative
per-domain schemas; greenfield scenarios with data use explicit out-of-tree
transformation scripts; production schema evolution earns ordered versioned
migrations. A release with no schema change requires no migration. When a schema
change exists, the complete migration delta must succeed against a representative
copy of the last deployed database without violating data invariants.

Test readiness requires current attributable suite evidence, behavioral evidence
for changed promises, and no unexplained regression against the previous deployed
release. Gherkin governs acceptance criteria and behavioral/e2e descriptions; it
does not require unit-test source to mimic feature files. Raw coverage is a trend
and floor signal, not sufficient proof by itself.

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
