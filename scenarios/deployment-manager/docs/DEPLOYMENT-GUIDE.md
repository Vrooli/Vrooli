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
under `.vrooli/program-runtime`. Recovery is read-only by default. A halt may be
executed only with the exact release, approved review, candidate, destination,
deployment, explicit confirmation, and dry-run reference; the owner must return
a durable effect receipt. Owner-specific rollback and forward repair require
their exact capability and compatibility fields; an owner without that
operation returns an explicit refusal.

For a cloud release, inspect the release dossier first:

```bash
deployment-manager releases dossier <release-id> --format json
deployment-manager releases recover <release-id> --review-key <review-key> --candidate-id <candidate-id> --destination-revision-id <destination-revision-id> --dry-run
deployment-manager releases recover <release-id> --review-key <review-key> --candidate-id <candidate-id> --destination-revision-id <destination-revision-id> --action halt --confirmation "halt <release-id>"
```

The dossier is the reviewer-facing read model. It joins the durable release
standing, receipt-backed health, candidate identity, destination revision, and
review binding. `missing_proof` is authoritative: an absent identity
repository or identity record means the release is not review-complete and
must not be promoted. The endpoint is read-only and is also available at
`GET /api/v1/releases/{release_id}/dossier`.

The first recovery command must produce a preview with no external effect. The
second command is successful only when the owner returns a matching durable
receipt; an unavailable owner, identity mismatch, or receipt persistence
failure leaves the release unresolved for reconciliation.

LPBS channels support `halt`, `withdraw`, `rollback`, and `forward_repair`
through the owner recovery endpoint. `rollback` requires the exact current
revision, its recorded predecessor revision, and `data_compatibility=compatible`.
`forward_repair` requires the same compatibility qualification plus a complete
replacement artifact set. Both create a new immutable channel revision, so an
installed client is never described as reverted by a feed change. Cloud
deployments support `halt`, rollback, and forward repair through
scenario-to-cloud. Cloud repair requires an exact current bundle identity, an
exact retained repair bundle, `data_compatibility=compatible`, and a durable
owner receipt after the repaired deployment is healthy. The production-like VPS
qualification remains open until this contract is exercised against the named
staging authority.

Cloud health is never inferred from an HTTP status. The release gate resolves
the exact cloud deployment from the manifest's scenario and domain (or
environment) through scenario-to-cloud's resolve endpoint, or uses the
deployment id from the cloud receipt, and then reads the typed observation at
`GET /api/v1/deployments/{id}/health/observation`. The step succeeds only when
the observation is `HEALTH_STATUS_HEALTHY`, `FRESHNESS_CURRENT`, about that
deployment id, observed within 120 seconds, running the expected bundle
digest, and not partial. Anything else fails the step with a reason code
(`status_not_healthy`, `stale_observation`, `release_mismatch`,
`deployment_mismatch`, `malformed_report`, …) and the cloud service's typed
next action. A cloud receipt whose `health` field says healthy is a
record-state claim and is also checked against the observation before the
receipt is accepted. Contract:
[scenario-to-cloud health contract](../../scenario-to-cloud/docs/reference/health-contract.md).

For the release ledger restore drill, use the existing data-backup-manager
qualification sink against the deployment-manager scenario:

```bash
data-backup-manager safety register-targets --scenario deployment-manager
data-backup-manager safety backup-now --scenario deployment-manager
data-backup-manager restores verify --target <ledger-target> --destination <backup-destination> --snapshot <snapshot-id>
```

Restore into an isolated qualification location, verify the snapshot before
using it, then re-read the deployment-manager release dossier and each owner’s
current channel/object/service receipt. A restored ledger never authorizes a
new promotion by itself; reconcile external owner state and create a fresh
review binding before resuming effects. Never restore over a live ledger or
replay a remote effect from a restored operation record.

For a release-trust compromise drill, inspect the managed authority before
taking action and keep the old trust root available for evidence review:

```bash
vrooli release-authority status --format json
# In an isolated qualification authority only:
vrooli release-authority regenerate --replace-trust-anchor
vrooli release-authority status --format json
```

Regeneration is a trust-root replacement. It requires explicit authority,
invalidates signatures from the prior key unless that public key remains
trusted, and must be followed by signing a known qualification stage and
verifying that the old signature is refused while the new signature is
accepted. Never run regeneration against the production authority as a
routine repair. Record the old/new key identifiers, refusal, acceptance, and
operator actions in the qualification receipt; private key material must never
appear in that receipt.

## Evidence-complete readiness workflow

One immutable review identity includes scenario, profile, candidate commit,
artifact digest, canonical target set, channel, and policy version. Evidence
owners report small typed observations with `readiness-reviews report-evidence`.
The report is accepted only for the criterion's policy-declared producer binding.
`readiness-reviews prepare` reads those exact observations, resolves the latest
published predecessor for the same profile, targets, and channel, persists the
decision, and opens only unresolved work in Swarm Manager.
When release shape is not inferable from the profile, pass declared facts such
as `--fact commercial_release=true`, `--fact paid_release=true`, or
`--fact schema_changed=false`; omitted facts remain unknown and fail closed.

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

Replacing a readiness evaluation archives the previous evidence snapshot with a
replacement revision and timestamp before updating the current evaluation.
Promotion reads only the current projection; the archived references remain
available for audit and do not preserve approval or goal-closure standing.

Target evidence follows the same rule: a passed target verdict must carry a
non-empty evidence class and at least one checksum-bearing owner reference.
The release gate refuses a passed boolean that has no proof, even if the
target and run identifiers are present. Repeated reports are immutable
attempts; reads expose the newest attempt first, and promotion evaluates that
attempt while retaining older references for audit.

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

Release start returns a durable operation handle when execution is queued or
running. Reattach to that operation with its bounded standing instead of
restarting the release:

```bash
deployment-manager releases operation <operation-id>
deployment-manager releases operation <operation-id> --format json
```

The same operation standing is available through the typed Releases API and
the Deployment Manager UI. An `ambiguous` result requires reconciliation or
recovery; it is not a successful publication claim.

Commercial desktop publication also requires the exact LPBS app key and remote
profile from the approved destination. The owner pipeline will not start when
either coordinate is missing. A release-bound desktop deploy also requires the
exact approved app key and channel, and derives a non-empty owner update URL.
Configured URLs are checked against that owner destination before artifacts are
collected or uploaded. The ready artifact platform set must also equal the
approved target set and carry a digest for every target; restore the owner
endpoint or restage the exact candidate when that check refuses the release.

## Commercial readiness criteria vs cloud-owned evidence

Two owners, one decision. Deployment Manager owns the commercial readiness
criteria: the readiness policy, the exact review identity, approval,
promotion, the publication receipt and the dossier. scenario-to-cloud owns
target execution and target evidence: the `cloud-launch-v1` capability
profile (the release lane of its certification matrix), the append-only
evidence records bound to release digest, target key and operation, the
publication effect and its reconciliation against the target's
`active-release.json`, and recovery.

What crosses the boundary, in both directions:

- The cloud reports one coverage `TargetVerdict` per target
  (`EvidenceService.ReportTargetVerdict`; platform = cloud target key, one
  ref per cell record) and asks `ReadinessService.PrepareReview` for the
  exact identity. The `ramp-evidence-complete` criterion consumes that
  verdict; the other criteria stay Deployment Manager's.
- Deployment Manager passes its review identity to the cloud in the
  deployment request and, after the receipt validates and the typed health
  observation is current and healthy, reads the cloud's per-cell evidence
  and publication. `deployCloud` refuses promotion when any required cell is
  not `passed` for the receipt's release and target, or when the evidence
  belongs to another release or target. A green build, a healthy observation
  or the receipt alone never promotes.
- The persisted publication receipt records the release the target actually
  activated, the predecessor the cloud resolved from the target pointer or
  its published history, the review key and the cloud evidence reference
  (`releases dossier` shows them under `publication_receipts`).
- Recovery requests carry the review key and the cloud's dry-run
  `preview_ref`; the cloud binds rollback to the published predecessor.

What never crosses it: Deployment Manager does not execute against targets,
author cloud evidence, or keep a second deployment state; the cloud does not
approve, waive, or promote. Missing, stale, failed, unknown or unavailable
cells are reported with their own word and are never a pass on either side.
Contract: `scenarios/scenario-to-cloud/docs/reference/governance-binding.md`.

## Other targets

Mobile, SaaS, and enterprise pages describe direction and constraints. They are
not promises of a working packager. Use the [Deployment Hub](../../../docs/deployment/README.md)
and the relevant target page to determine whether a path is current, conditional,
or merely planned.

## Release operations runbook

Start with the release dossier and health projection:

```bash
deployment-manager releases get <release-id> --format json
deployment-manager releases health <release-id> --format json
```

If health reports `ambiguous_effect`, `failed_verification`, or
`missing_publication_receipt`, stop promotion and reconcile the owner’s current
object, service, or client state before retrying. A `halted` standing stops
future offers and records the owner receipt; it does not remove an update that
is already installed.

For an LPBS destination, use the exact review, candidate, destination revision,
and release identities. Preview first, then execute only with an explicit
confirmation:

```bash
deployment-manager releases recover <release-id> \
  --review-key <review-key> \
  --candidate-id <candidate-id> \
  --destination-revision-id <destination-revision-id> \
  --action halt --dry-run --format json

deployment-manager releases recover <release-id> \
  --review-key <review-key> \
  --candidate-id <candidate-id> \
  --destination-revision-id <destination-revision-id> \
  --action halt --confirmation "halt <release-id>" --format json
```

The current owner implementations support halt for LPBS channels and cloud
deployments. LPBS also supports withdrawal, predecessor-bound rollback, and
compatibility-qualified forward repair. Cloud rollback and forward repair use
the retained bundle store and durable recovery operation endpoint. Do not
substitute a direct object-store, VPS, or feed mutation. If a schema migration
makes binary rollback unsafe, preserve the installed data and route a
compatibility-qualified forward repair through the owning target ramp.

To reconcile an uncertain release, run the owner observation and inspect the
returned evidence. Reconciliation preserves an ambiguous or failed durable
standing when the observation cannot prove a complete publication receipt:

```bash
deployment-manager releases reconcile <release-id> --format json
deployment-manager releases reconcile <release-id> --deep --format json
```

Health observations emit signed `deployment-manager.release.alert.v1` facts to
`vrooli-events` when that service is configured. `notification-hub` renders
those facts and records delivery attempts for its configured operator
destination. Event delivery is optional; the health projection and CLI remain
the canonical fallback. Alerts identify the release or target, severity,
reason, and next action. Signing-expiry observations come from the
scenario-to-desktop certificate owner and never include credentials or private
key material.

## Troubleshooting and references

- [Troubleshooting](workflows/troubleshooting.md)
- [CLI overview](cli/overview-commands.md)
- [Profile commands](cli/profile-commands.md)
- [Fitness scoring](guides/fitness-scoring.md)
- [Dependency swapping](guides/dependency-swapping.md)
- [Bundle manifest schema](guides/bundle-manifest-schema.md)
- [Evidence contract](guides/evidence-contract.md)
- [Release authority](../../../docs/configuration/release-authority.md)
