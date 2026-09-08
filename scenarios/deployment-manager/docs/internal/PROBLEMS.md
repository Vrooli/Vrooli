# Current validation problems

This is the maintained problems register for deployment-manager; it records only
environmental or implementation findings that still reproduce.

| Finding | Cause | Scope |
|---|---|---|
| Go aggregate coverage | The final aggregate behavior-coverage runs meet the declared policy: 75.5% API (3,692/4,889 statements) and 75.1% CLI (1,597/2,127 statements). Per-package coverage and compatibility-orchestration branches remain advisory follow-up debt. | Remaining implementation |
| Test architecture debt | Unit-health still reports one injectable-seam warning for package-level process construction and advisory traceability/coverage debt. The declared thresholds remain unchanged. | Remaining implementation |

No suppression, waiver, or lowered threshold is used for these findings.

## Work ladder

- Rung: W3 (learning capture and measurement implementation).
- Evidence: the named readiness goal requires an attributable learning loop; Deployment Manager OT-P0-044 supplies that contract. The archived cross-ramp goal and desktop OT-P0-004 cover agent tooling for the desktop ramp. Business and requirements gates passed on 2026-09-04. This task implements the operator-approved learning recommendations without changing release promises.
- Blocker: none for the learning setup; live outcome baselines remain unearned. Shared Memory UI/attestation findings are recorded in the learning progress entry.
- Measured: 2026-09-04

## Work ladder — first commercial release review, 2026-09-07

- Rung: W0 (launch contract is incomplete against the current operator directive).
- Evidence: the operator requires “Updating from here should be proven to publish to the right location” and desktop apps “proven to be able to update from this.” PRD OT-P1-011 puts desktop/mobile update distribution under “Should have post-launch”; no P0 target explicitly requires a hosted-feed-to-installed-client upgrade receipt. OT-P0-043 does require current evidence at promotion, which remains a valid implementation obligation.
- Governing goals inspected: `deployment-manager-evidence-complete-readiness` (active) and `deployment-manager-greenfield-re-platform-to-a-proto-first` (archived), including the latter's referenced plan. The archived plan explicitly excluded real publication, fleet-wide ramp adoption, and macOS/Windows journey proof. Its completion cannot certify this launch.
- Blocker: reconcile launch-critical distribution, client-upgrade, recovery, and authorization acceptance with the P0 contract. This review does not authorize implementation or contract edits.
- Measured: 2026-09-07. No new W1–W3 gate was run below this contract gap. Source inspection and historical receipts below are diagnostic evidence, not gate passes.
- Visual assessment and proposed acceptance matrix: Commercial release readiness (preserved HTML).

### Source findings requiring focused reproduction

These findings describe the inspected working tree; no live publication or security exploit was attempted.

| ID | Finding | Source and required closure |
|---|---|---|
| DM-LAUNCH-01 | `RunDeploy` discards stage return codes and continues into later stages after load/gate/validation failures. | `api/deployments/orchestrator_release.go:40`. Stop before every downstream side effect on refusal or failure; prove via injected failures for each stage. |
| DM-LAUNCH-02 | The approved digest is checked and stored by release start, but not carried in `DeployRequest` or `PublishPipelineRequest`; the path builds again. | `api/releases/handlers.go:298,334`; `api/deployments/orchestrator_helpers.go:33`. Bind promotion to the immutable, signed, tested artifact set and destination; verify the actual uploaded bytes. |
| DM-LAUNCH-03 | Publish failures become warnings; update verification initializes `allMatch=true` and iterates only returned published versions. An empty set can mark a release published; nil results and missing requested targets are not rejected there. | `api/deployments/orchestrator_helpers.go:22`; `api/deployments/orchestrator_release.go:145`. Require exact nonempty target coverage, explicit valid receipts, matching hashes and identities, and durable persistence before success. |
| DM-LAUNCH-04 | Approval revalidates freshness, but release start reads the stored approved state; the promotion repository checks status rather than re-evaluating evidence. | `api/handlers/readiness/connect_handler.go:154,221`; `api/server/server.go:209`; `api/internal/readiness/repository.go:443`. Revalidate at the irreversible boundary and refuse expiry, revocation, policy change, or candidate/destination drift. |
| DM-LAUNCH-05 | The inspected server mounts mutating Connect handlers without an authentication interceptor; producer binding and actor are request fields. The HTTP listener binds all interfaces. | `api/server/connect_routes.go:21`; `api/server/server.go:223,264`; `api/handlers/readiness/connect_handler.go:37,154`. Establish the shared authenticated boundary and test direct and Bridge-routed access. Network reachability and exploitability were not measured. |
| DM-LAUNCH-06 | `CheckDeployReadiness` discards JSON decode errors and sets `Ready=true` for every HTTP 200. | `api/deployments/lpbs_release_client.go:152`. Reject malformed bodies and explicit negative readiness, with typed protocol tests. |
| DM-LAUNCH-07 | The release workflow runs synchronously on the request context; recovery program execute mode always refuses because no target-owner mutation is bound. | `api/releases/handlers.go:334`; `.vrooli/program-runtime/release-recover.py:step_act`. Use durable resumable jobs, idempotent publication, reconciliation after ambiguous outcomes, and an evidenced ramp-owned recovery operation. |
| DM-LAUNCH-08 | UI and workflow proof do not establish the full review/publish/update journey. | `ui/src/features/releases/Releases.tsx` lists records; `bas/registry.json` registers one observer case asserting a heading. Gate-named flows assert headings/input visibility, not approval/refusal effects. Add complete operator journeys with exact identity and publication/client receipts. |

### Recorded validation limitation

Latest run returned by Test Genie: `20260906-025035-951cb950`, comprehensive, failed on 2026-09-06 before phase execution. `runs findings` confirms no findings manifest. `runs status deployment-manager <run-id> --json` reports lifecycle start failure during a UI component build, following stale `secrets-manager` rebuild handling. The terminal receipt does not identify the underlying build error. This is an unavailable current suite verdict, not evidence that individual phases failed. No current green certification is established.

## Preserved visual sources

Historical HTML and editable canvas sources live beneath the protected
control-plane runtime home (`~/.vrooli` for the invoking user).
These are dated evidence or design supplements; this relocation does not
change plan status or establish release readiness.

- `plan-artifacts/docs-html-progress-20260908/scenarios/deployment-manager/docs/internal/commercial-release-readiness-2026-09-07.html`

The originating installation must retain these artifacts with its durable backups.
Exact originals and SHA-256 hashes are in `backups/docs-html-progress-20260908/manifest.json`.
