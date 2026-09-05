# Integrations — Scenario to Plugin

This document is the canonical dependency contract for resources,
other scenarios, and third-party services used by the scenario.

It is kept in agreement with `.vrooli/service.json` by hand. If this page
and that manifest disagree, one of them is a bug: declare the dependency
or correct the doc, never leave them inconsistent.

## Purpose Of This Document

Use this document to answer:

- What does the scenario depend on?
- Which dependencies are required versus optional?
- Which domain uses each dependency?
- What is the failure or degradation behavior?
- Where is the dependency declared or configured?

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes | API, all pipeline domains | resolved by `api-core/storage` from the scenario id | API reports unhealthy if unreachable. |
| Capture store | local filesystem | yes | `composition`, `attestation`, `rehearsal` | Scenario-owned artifact directory | Artifact writes fail closed; no package advances. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario should be started through lifecycle commands. |
| `deployment-manager` | scenario | yes | `distribution` | Release-gate request; `EvidenceService.ReportTargetVerdict` | Publication and revocation refused; upstream stages unaffected. |
| `workspace-sandbox` | scenario | yes | `rehearsal` | Sandbox lifecycle and command execution | Rehearsal refused rather than run unisolated. |
| `cli-health` | scenario | yes | `conformance`, `declaration` | Pinned `cli-manifest` command surface | Drift gate fails closed. |
| `secrets-manager` | scenario | yes | `attestation`, `distribution` | Credential references, never literals | Channel calls needing a credential are refused with the reference named. |
| `scenario-dependency-analyzer` | scenario | no | `attestation` (SBOM), build | Governed package installation; dependency facts | SBOM falls back to manifest inspection with reduced provenance. |
| `offer-desk` | scenario | no | `distribution` | Per-plugin channel evidence | Attribution buffered locally; no stage blocks. |
| `knowledge-observatory` | scenario | no | `distribution` | Published-skill documentation indexing | Indexing deferred; packages unaffected. |

## Vrooli Resources

This scenario declares **no shared resources**, and that is a deliberate
product decision rather than an unfilled default.

Scenario to Plugin exists to publish capabilities that install *without*
the Vrooli resource fleet. A ramp that itself required Postgres, Redis,
Qdrant, or Ollama to produce a standalone package would contradict the
contract it enforces, and it would make the rehearsal's isolation claim
harder to trust. SQLite plus a local capture store is sufficient: the
data is low-volume, single-writer, and scoped to one host's publication
history.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| postgres | not-applicable | Publication history is single-host, low-volume, and single-writer. | Multi-host publication coordination, or a shared fleet-wide plugin registry. |
| redis | not-applicable | No cross-process cache or queue; rehearsals are serialized per package. | Rehearsal concurrency that outgrows in-process scheduling. |
| ollama | not-applicable | Every gate is deterministic. A model-judged conformance check would make the gate unreproducible. | Never for gating. Possible for advisory drift-repair suggestions (`OT-P2-003`), and only as advisory output. |
| qdrant | not-applicable | No semantic retrieval in the pipeline. | Cross-plugin similarity search, if composite packaging arrives. |

## Scenario Dependencies

| Scenario | Status | Required | Startup | Reason | Contract |
|---|---|---|---|---|---|
| `deployment-manager` | active | yes | `must_start` | ADR-005 assigns the release decision to the governance plane. This ramp asks; it never decides. | Release-gate request bound to a source commit; `TargetVerdict` carrying evidence references only. |
| `workspace-sandbox` | active | yes | `must_start` | A rehearsal that can reach the developer's running scenarios proves nothing about an external machine. | Sandbox creation, command execution, teardown, and an accountability record per run. |
| `cli-health` | active | yes | `must_start` | The drift check must compare against a *pinned* command surface, not a live one, so a failure is attributable. | `cli-manifest` retrieval by scenario and revision. |
| `secrets-manager` | active | yes | `try_start` | Registry tokens and signing credentials must never be held by this scenario. | Reference resolution at publish time; no literal ever enters an artifact, SBOM, or verdict. |
| `scenario-dependency-analyzer` | active | no | `try_start` | Third-party packages are governed fleet-wide; SBOM accuracy improves when dependency facts come from the authority. | `deps install` at build time; dependency facts for SBOM generation. |
| `offer-desk` | active | no | `ignore` | The `skill-registries` channel needs per-plugin evidence to be activated from measurement rather than argument. | Channel evidence submission; this scenario never reads pricing or offer state. |
| `knowledge-observatory` | active | no | `ignore` | Published skill documentation should be discoverable inside the fleet. | Documentation indexing; advisory only. |

### Why `agent-manager` is not a dependency

Other ramps declare `agent-manager` for agent-assisted pipeline
investigation. This ramp deliberately does not. Every gate here is
deterministic and must be reproducible by a third party from the emitted
evidence — a registry, an auditor, or a consuming agent. Introducing a
model-judged step into a supply-chain gate would make the verdict
unreproducible and the trust claim unverifiable.

## Third-Party Services

These are reached only during `attestation` and `distribution`, always
through a credential reference, and always with the outcome recorded.

| Service | Status | Required | Reason | Contract |
|---|---|---|---|---|
| OCI registry (`ghcr.io`) | planned | yes at publish | Primary distribution channel; carries the artifact plus signature, provenance, and SBOM as referrers. | Push by digest; referrer attach; retrieval confirmation before a publication is recorded. |
| Sigstore (Fulcio / Rekor) | planned | yes at publish | Keyless Cosign signing and transparency-log inclusion. | Signing through the managed release authority; this scenario never holds a key. |
| Skill/plugin scanners | planned | yes at publish | A critical or high finding fails the package. | Container-invoked scan with a machine-readable verdict. |
| Curated registries | deferred | no | Submission is a manual, relationship-bearing process today (`OT-P2-001`). | Submission and review-state tracking, once one package has been accepted by hand. |

Network access is confined to these calls. No pipeline stage before
`distribution` reaches the network, which is what allows composition,
conformance, and attestation ordering to be verified offline.

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |
| Capture store | Write or digest mismatch | Package marked failed; no partial artifact is recorded as complete. | composition storage tests |
| `deployment-manager` | Unreachable, or no decision for this commit | Publication refused with the missing decision named. Never a default-allow. | `PLG-DIST-GATE` |
| `workspace-sandbox` | Sandbox creation failure | Rehearsal refused. The ramp does not fall back to host-local install. | `PLG-REHEARSE-ISOLATE` |
| `cli-health` | Manifest unavailable for the pinned revision | Drift check fails closed and names the unavailable revision. | `PLG-CONF-DRIFT`, `PLG-CONF-DRIFT-PIN` |
| `secrets-manager` | Reference unresolvable | Channel call refused; the reference name is reported, never the value. | `PLG-DIST-GATE`, `PLG-ATTEST-SIGN` |
| OCI registry | Push succeeds, retrieval fails | Publication recorded as unconfirmed, not published. | `PLG-DIST-CONFIRM` |
| Scanner | Critical or high finding | Package fails. A scanner finding is a gate, not a warning. | `PLG-ATTEST-SCAN` |
| Channel (revocation) | Withdrawal not supported or fails | Revocation marked incomplete with the channel named. | `PLG-DIST-REVOKE-PARTIAL` |

The pattern is uniform: **every dependency failure closes a gate, and no
dependency failure opens one.** Degradation reduces what the ramp will
do, never what it will claim.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DOMAINS.md`](DOMAINS.md) — which domain uses each dependency
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — credential handling and threat model
