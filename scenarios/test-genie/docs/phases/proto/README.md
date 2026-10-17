# `proto` phase

The `proto` phase is a **thin delegating phase**: it calls the `proto-health`
scenario's shared `ScenarioValidationService` and maps scenario-scoped Protocol
Buffer contract findings into the shared `FINDING_SOURCE_PROTO` channel.
test-genie does not parse descriptors or validate proto organization itself;
those checks live in `proto-health`, alongside the proto-surface fact RPC.

This phase declares a [Phase Capability Contract](../../concepts/phase-capability-contract.md); the sections below follow the required remediation-doc skeleton.

## North Star

Every scenario proto surface is a clean, evolvable contract that is **provable, not merely present**: package layout is organized and clean, generated Go/Python/TypeScript/descriptor artifacts stay synchronized with their schema sources, transport usage is proto-backed with aligned REST-payload and health contracts, dependency and reuse posture is clean, and — the deepest rung — **endpoint proof is aligned**: Code Facts evidence confirms proto adoption and endpoint payloads are actually implemented, with no contradictions. At maximum maturity a downstream consumer can trust the proto contract without reading the implementation.

## The rungs and their gates

proto-health reports a ladder per capability. The rungs are monotone — each implies the one below.

| Capability | Rungs | Top rung (aspiration) | Key next unlock |
|---|---|---|---|
| Package Layout | L0–L2 | Proto package layout is clean. | Clean package/version/domain/import/stability/shared-type/annotation contracts. |
| Generation Sync | L0–L3 | Generation sync is clean. | Generated artifacts match schema sources, then toolchain provenance is current. |
| Transport Contracts | L0–L3 | Transport contracts are clean. | Proto-backed transport + health protos, then aligned REST payloads and protovalidate constraints. |
| Endpoint Proof | L0–L3 | Endpoint proof is aligned. | Code Facts proof present, then no contradictory implementation evidence. |
| Dependency Reuse | L0–L2 | Proto dependency and reuse posture is clean. | Clean cross-domain imports, dependency stability, and unused-type hygiene. |

## What each finding means

Each finding caps the capability it names at a rung; only ERROR/BLOCKER severities fail the phase, so most findings are honest, non-failing debt. Representative codes (full inventory in the descriptor):

| Code | Capability | Caps at | Severity | Fails phase? |
|---|---|---|---|---|
| `proto.gen_manifest_missing` | generation_sync | L0 | ERROR | Yes |
| `proto.gen_out_of_sync` | generation_sync | L2 | ERROR | Yes |
| `proto.gen_toolchain_drift` | generation_sync | L3 | WARNING | No |
| `proto.cycle` | package_layout | L1 | ERROR | Yes |
| `proto.package_mismatch` | package_layout | L1 | ERROR | Yes |
| `proto.hand_rolled_transport` | transport_contracts | L1 | WARNING | No |
| `proto.rest_payload_invalid_conformance` | transport_contracts | L2 | ERROR | Yes |
| `proto.endpoint_proof_missing` | endpoint_proof | L2 | WARNING | No |
| `proto.endpoint_proof_contradicted` | endpoint_proof | L3 | ERROR | Yes |
| `proto.cross_domain_import` | dependency_reuse | L2 | WARNING | No |

## The canonical fix

- **Generation-sync findings** (`proto.gen_manifest_missing`, `proto.gen_out_of_sync`, `proto.gen_toolchain_drift`) → regenerate proto artifacts and commit the generation lockfile so committed Go/Python/TS/descriptor output and toolchain pins match the schema sources.
- **Package-layout findings** (`proto.cycle`, `proto.package_mismatch`, `proto.domain_mismatch`, `proto.shared_type_misplaced`, `proto.stability_*`, `proto.version_naming`, `proto.unsupported_annotation`) → move proto files into the correct scenario package/version/domain, break import cycles, relocate shared types, and align stability metadata per `packages/proto/STYLE_GUIDE.md`.
- **Transport-contract findings** (`proto.hand_rolled_transport`, `proto.missing_health_proto`, `proto.rest_payload_*`, `proto.constraint_missing_protovalidate`) → replace hand-rolled transport with generated Connect-RPC, declare required health protos, and align REST payload declarations and protovalidate constraints.
- **Endpoint-proof findings** (`proto.proto_adoption_missing`/`_contradicted`, `proto.endpoint_proof_missing`/`_contradicted`) → implement the declared endpoints so Code Facts evidence confirms adoption; a `*_contradicted` finding means the code contradicts the declaration — reconcile them.
- **Dependency-reuse findings** (`proto.cross_domain_import`, `proto.import_kind_unknown`, `proto.possibly_unused`, `proto.stability_dependency_mismatch`) → remove or relocate cross-domain imports, reuse shared types, and prune unused declarations.

## How to verify

```bash
# See the current rung, gaps, and next move for every capability:
proto-health validate scenario <scenario>

# Or drive it through Test Genie and read the per-phase scorecard:
test-genie execute <scenario> --phases proto
test-genie runs findings --scenario <scenario>
```

## What it runs

```text
scenario-validation/v1.ScenarioValidationService.ValidateScenario
```

Test Genie reads the shared `status`, `assessment.local`, and
`assessment.findings` fields. Each assessment finding is mapped to an
`ArchitectureFinding{Source: FINDING_SOURCE_PROTO}`, so it carries a
deterministic stable ID, normalized severity, and the per-source effort
default. The phase summary carries proto-health's `current_level`, `next_level`,
`clean`, and `unknown_count` convergence signals; phase pass/fail still comes
only from `status`.

## Severity contract

proto-health owns the rule semantics and severity tier. This phase only
normalizes the emitted severity string:

| proto-health severity | normalized | feeds the `proto-health` dimension as a gap? |
|---|---|---|
| `SEVERITY_ERROR` | ERROR | **yes** |
| `SEVERITY_WARNING` | WARNING | no |
| `SEVERITY_INFO` | INFO | no |

Only ERROR findings fail the phase. They flow into the swarm-manager
`proto-health` dimension, which is a soft R2 ladder input for evolvable
architecture and contract health.

proto-health also marks each finding with a `clean_requirement`. REQUIRED
findings gate proto-health's local rung even when they are WARNING-level debt;
UNCHECKABLE findings are excluded from the rung and surfaced as unknown. This
lets agents continue fixing a scenario toward a clean proto surface without
turning warning-level debt into a Test Genie failure.

Generated-artifact drift for the target scenario is still an ERROR
(`proto.gen_out_of_sync`). A proto-bearing scenario with no committed
generation lockfile emits `proto.gen_manifest_missing` as an ERROR.
Toolchain pin drift emits `proto.gen_toolchain_drift` as a WARNING, so it is
visible but does not fail the phase.

Evidence checks from code-facts are advisory unless they prove a contradiction:
`proto.proto_adoption_missing`, `proto.endpoint_proof_missing`, and
`*_unsupported` findings are WARNING. `*_contradicted` findings remain ERROR.
Absence of proof therefore does not make the phase outcome depend on whether
code-facts happens to be running.

## Classification

- **Optional**, included in every built-in preset. Curated presets reference it
  explicitly; `comprehensive` auto-joins it through the default catalog.
- **120s timeout** — descriptor and generated-artifact checks are file reads and
  hashing; the budget mainly protects RPC startup and slow worktrees.
- Unreachable `proto-health` API -> skip observation (never a hard failure).
- `proto.gen_toolchain_drift` -> advisory finding (never a hard failure).
- `TEST_GENIE_SKIP_PROTO=1` skips the phase entirely.

## See also

- `scenarios/proto-health/` — the producer scenario.
- `packages/proto/STYLE_GUIDE.md` — the proto organization standard.
- `packages/proto/schemas/architecture/v1/findings.proto` —
  `FINDING_SOURCE_PROTO = 12`.
- `scenarios/proto-health/.vrooli/test-genie.json` — the descriptor-owned
  `proto-health` dimension and phase coverage metadata.
