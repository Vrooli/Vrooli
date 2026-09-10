# Release notes: cloud delivery ramp (plan `scenario-to-cloud-professional-vps-delivery-certification`)

Candidate: the `agi` working tree on 2026-09-09 (execution
`6f482be3-0c38-45be-aff7-8744b174c925`). Every claim below names the
evidence file that backs it under
`~/.vrooli/plan-artifacts/scenario-to-cloud-professional-vps-delivery-certification/evidence/`
("evidence/…") or a test in this repository. A phase without a row did not
ship. Nothing here is a certification claim; see **Standing**.

## Standing

**Not certified.** The `api/certification` readiness computation over the
package-lane receipts (`evidence/certification-report-2026-09-09-final.json`,
119 receipts) reports `not_ready`:

| Required (case, lane) cells | passed | missing | stale | unavailable | failed |
|---|---|---|---|---|---|
| 266 | 0 | 158 | 55 | 53 | 0 |

The 55 stale cells hold package-lane receipts bound to an earlier candidate
digest (they pass again when re-run against the final tree; see evidence
freshness in the support policy). The 53 unavailable cells are QEMU-lane
receipts recorded honestly as `verdict: unavailable` (host tools absent).
Every required real, BAS, review and soak cell is missing. No production or staging
publication was performed; the production deployment was only read.
Evidence freshness and what a re-certification requires:
`docs/reference/support-policy.md` §"Compatibility policy and evidence freshness".

## What shipped, by phase

| Phase | Shipped | Evidence |
|---|---|---|
| 01 Contract | PRD OT-P0-007..017, 32 requirements (`requirements/10-18`), support policy, four ledgers | `evidence/P01-contract.md` |
| 02 Validation plant | Certification matrix registry (94 cases, `certification/matrix.json`) with a refusal gate, generic workload fixtures with oracles (`fixtures/`), test-mode-only fault injection | `evidence/P02-validation-plant.md` |
| 03 Contracts | Typed identities and selectors (`api/identity`), `api/apierrors` codes → HTTP status → exit codes, deployment identity columns with legacy conversion, `cloud_operations` request-key semantics, proto identity/errors/deployments + Connect DeploymentsService | `evidence/P03-contracts.md`, `docs/reference/identity-and-selectors.md` |
| 04 Management authority | Default-deny authorization on every route with shared authn, origin/host/websocket enforcement, revocation recheck, generated matrix (`docs/reference/authorization-matrix.md`) | `evidence/P04-management-authority.md` |
| 05 Closure | Declaration-derived component closure with reasons, digest, unsupported entries; setup/v1 Selection parity | `evidence/P05-closure.md`, `docs/reference/closure.md` |
| 06 Executable plan | One typed plan with a semantic digest; preview = apply; `plan_digest` required on apply; hidden skips removed | `evidence/P06-executable-plan.md`, `docs/reference/executable-plan.md` |
| 07 Durable operations | Fenced, leased operations with startup reconciliation, wait/cancel, receipts read before replay; RUN-01..10 package receipts | `evidence/P07-durability.md`, `docs/reference/operation-lifecycle.md` |
| 08 Target owner | `vrooli cloud-target` verbs (receipt, release verify/stage/activate/rollback, data inventory, host repair through privilegebroker only) with fenced idempotent receipts | `evidence/P08-target-owner.md` |
| 09 Reach | `api/reach` seam with Bridge and bounded SSH adapters, explicit transport, revocation refusal, artifact `Deliver`, platform negotiation | `evidence/P09-shared-reach.md` |
| 10 Release trust | One release identity (bundle + reproducible native CLI + closure + configuration), signed provenance, `.complete` staging, lease-aware GC | `evidence/P10-release-trust.md`, `docs/reference/release-identity.md` |
| 11 Safe activation | Executor on reach with `--operation --step --fence`, strategy selection, runtime → edge → pointer ordering with intent file, predecessor retention, legacy data inventory/adopt | `evidence/P11-safe-activation.md`, `docs/reference/activation-and-reconciliation.md` |
| 12 Data recovery | Recovery points with consistency providers, schema-aware rollback admission, protected retention, fresh-host restore drill (package lane) | `evidence/P12-data-recovery.md`, `docs/guides/recovery-runbook.md` (test-rendered) |
| 13 Credentials | Bindings, versions, rotation state machine, revocation, recovery; canary scan 0 findings; secrets CRUD on the binding model | `evidence/P13-credentials.md`, `docs/reference/credential-lifecycle.md` |
| 14 Edge | Listener policy, per-deployment Caddy snippets with rollback, DNS/TLS lifecycle; a real renewal failure on the production domain was observed and logged | `evidence/P14-edge.md` |
| 15 Lifecycle | `api/reconcile` desired/observed model, desired-state contract (an intentional stop is never restarted), retirement plans, protected cleanup; private start/stop/repair pipelines removed | `evidence/P15-lifecycle.md` |
| 16 Health | Typed health observation; Deployment Manager's false-green consumer replaced by a fail-closed exact-deployment check | `evidence/P16-health-truth.md`, `docs/reference/health-contract.md` |
| 17 Governance | delivery-ramp adapters, `cloud-launch-v1` cells, exact review binding, signed bound receipts, publication apply re-checks approval | `evidence/P17-governance.md`, `docs/reference/governance-binding.md` |
| 18 API/CLI | CLI over generated Connect clients, one selector grammar, plan → review → apply → wait, durable wait/resume with the exit-code contract, `cli/manifest.json` with governance, generated command reference | `evidence/P18-api-cli.md`, `docs/reference/cli-commands.md` (test-generated) |
| 19 Operator experience | Deployment console (five operator questions), plan review wizard, durable operation resume, denied/interrupted states, a11y tests | `evidence/P19-operator-experience.md`, `docs/reference/console-experience.md` |
| 20 QEMU lane (prepared) | Readiness report, governed qualification program, lane manifest (54 cells), honest `unavailable` receipts | `evidence/P20-qemu-qualification.md`, `docs/guides/qemu-qualification.md` |
| 22 Security (deterministic) | Threat-to-test matrix over existing negative proofs; canary scans | `evidence/P22-security-review.md` |
| 23 Budgets (deterministic) | Frozen budgets, measurement harness, bounded-behaviour tests, per-host operation limit, alert emitter, OPS-04 package receipt | `evidence/P23-soak-operations.md`, `docs/reference/service-objectives.md`, `docs/guides/incident-runbook.md` |
| 24 Generality (deterministic) | Newcomer fixture deployed through declarations only, two-environment isolation, unsupported-architecture naming, routes across update, generic fixture golden | `evidence/P24-generality.md`, `docs/guides/next-scenario-onboarding.md` |
| 25 Consolidation | Route retirements (`/ssh/*`, `bundles/vps/list|delete`, `preflight/fix/ports`), `api/ssh` deleted with the runner inside `reach/sshadapter`, manifest `key_path` replaced by an SSH-key credential binding, `preflight/disk/*` on observation programs and broker actions, canonical documentation set, architecture conformance tests (`api/archtest`), operator runbooks (`docs/guides/runbooks/`), this file | `evidence/P25-consolidation.md` |

Whole-module `go test ./...` for `api/` passed on 2026-09-09 (all packages);
`cli/` 12 packages pass; `ui/` 44 test files pass with type-check clean
(`evidence/P26-certification.md`).

## Explicit limitations (copied from `evidence/P26-certification.md`)

Not delivered:

| Item | Standing |
|---|---|
| Phase 21 real-VPS lane | no disposable target (EXT-01) |
| Phase 20 execution | host tools absent (EXT-06); the program is ready |
| Phase 22 independent review and live adversarial journeys | EXT-07 / EXT-01 |
| Phase 23 soak, real pressure, live alert delivery | EXT-09 / EXT-01 |
| Phase 24 on a target; hosted-consumer journey | EXT-01 |
| Phase 25 remainder | operator walkthrough (DL-09) and the legacy data-inventory adoption on the production row (DL-04); the `api/ssh` package, manifest `key_path` and the `preflight/disk/*` shell were retired the same day (`evidence/P25-consolidation.md`; standing per row in `ledgers/deletion-ledger.md`) |
| Phase 26 | staging publication and production promotion inputs (EXT-01, EXT-11) |

Deletion-ledger rows still partial at the time of writing: DL-02 (cloud
SSH transport authority), DL-03 (private host repair for disk cleanup),
DL-04 (directory-name heuristic for unmapped legacy paths, retires once the
production inventory is adopted), DL-08 (SSH error classifiers), DL-09
(docs; closed by this phase except for the walkthrough), DL-11
(`preflight/disk/*` routes). Closed: DL-01, 05, 06, 07, 10, 12, 13, 14, 15.

Known functional limitations carried in the reference docs:

- Bridge artifact delivery is delegated to Bridge's authenticated
  ArtifactsService and waits for durable placement confirmation. Missing
  service configuration or unconfirmed placement remains a typed failure with
  no SSH fallback; live disposable-target qualification is still pending
  (activation doc §12).
- `side_by_side` activation runs no concurrent candidate; production
  manifests pin ports, so activations are `maintenance`.
- `data.retire delete` and `artifacts.retire` deletion have no target
  owner verb; both are honest refusals or reports.
- The production domain's TLS-ALPN renewal failure (phase 14) must be
  investigated before 2026-12-07.
- No scheduler invokes terminal-operation pruning; notification-hub delivery
  to an operator is unexercised until the soak.

## Supported platform matrix (from `docs/reference/support-policy.md`)

| OS | Architecture | Classification |
|---|---|---|
| Ubuntu 24.04 LTS | linux/amd64 | Certified (launch baseline) — **evidence pending** on the QEMU and real lanes |
| Ubuntu 24.04 LTS | linux/arm64 | Certified (launch baseline) — **evidence pending** |
| Ubuntu 22.04 LTS | linux/amd64, arm64 | Compatibility-only (preflight warning; no certification cells) |
| Ubuntu 20.04 LTS | any | Compatibility-only, deprecated |
| Debian 12 | any | Unsupported |
| Other Linux, macOS, Windows | any | Unsupported as a VPS target |

Transports: vrooli-bridge / nodereach is the certified path; direct SSH
through the bounded reach adapter is supported with explicit transport
selection and never a silent fallback. Cloud provider APIs, Kubernetes and
multi-region are out of scope.

## Operator handoff (in order)

1. Designate a disposable Ubuntu 24.04 VPS and a delegated DNS sub-zone
   (EXT-01, EXT-03); install `qemu` and `cloud-localds` through `vrooli setup`
   (EXT-06).
2. Provision the cloud service principal's Bridge token, enroll the target
   through `vrooli-bridge onboard`, bind the deployment to `transport: bridge`.
3. Run the QEMU journeys (phase 20) and the real-VPS journeys (phase 21)
   through the operations API; receipts land in `certification/evidence/`
   and the readiness report turns cells from missing to passed.
4. Investigate the production TLS-ALPN renewal failure.
5. Name the independent security reviewer and the walkthrough participant
   (EXT-07/08); schedule the 24 h soak (EXT-09).

Runbooks for every lifecycle task: `docs/guides/runbooks/README.md`.
