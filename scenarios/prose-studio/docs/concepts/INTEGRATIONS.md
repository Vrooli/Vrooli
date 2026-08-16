# Integrations — Prose Studio

This document is the canonical dependency contract for resources,
other scenarios, and third-party services used by the scenario.

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
| SQLite | embedded storage | yes | API, all persistence-backed domains | `SQLITE_PATH` lifecycle env var | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario is started through lifecycle commands only. |
| ai-gateway | scenario | yes | `generation` | Typed inference `Run`; roles `write.default` and `write.diverse` | Generation refuses with a named error; every read path stays available. |
| Consumer declaration files | filesystem | no | `declarations` | `.vrooli/prose-studio/*.json` in a consuming scenario | A malformed file registers `invalid` with its parse error; the service still starts. |

## Vrooli Resources

Prose Studio declares **no external Vrooli resource**, and that is a decision
rather than a scaffold default. Every record it owns — styles, profiles,
sessions, candidates, documents, declarations — is scenario-local and read
almost exclusively by this scenario. Nothing needs cross-scenario transactional
storage, a vector index, or a queue: candidate sets are small and bounded, the
convergence graph is append-only, and federated reachability is served by
registering with `search-hub` rather than by sharing a datastore.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| SQLite (embedded) | in-use | All domain data is scenario-local, small, and append-only. | Never expected to change. |
| Postgres | not-applicable | No cross-scenario transactional need. | A second writer needs the same rows. |
| Qdrant / vector store | deferred | Semantic diversity is a P2 metric; the P0 tier is lexical and deterministic. | Embedding-based diversity is promoted from P2, *and* an embedding surface exists through ai-gateway. |
| Redis / queue | not-applicable | Generation is synchronous and bounded per request; long-form is section-at-a-time. | Batch document generation becomes a real workload. |
| Vault / secrets | not-applicable | This scenario holds no credential of any kind; all inference routes through ai-gateway, which owns provider credentials. | Never — holding a provider credential here would be an architectural error. |

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| ai-gateway | required | The only inference path. This scenario names no model, holds no credential, and calls no vendor directly, which is also what makes its metering Class A for free. | `InferenceService.Run` with `schema_json`; roles `write.default` (quality lane) and `write.diverse` (candidate-set lane). **Both roles and request-level sampling control must exist before this scenario's first generation slice.** |
| content-desk | consumer | First declaring consumer. Keeps editorial judgement — what to argue, which claims, which profile — while this scenario owns mechanism. | Declares profiles under its own `.vrooli/prose-studio/`; references committed outputs by identifier. |
| search-hub | optional | Federated retrieval of styles and committed documents. | Provider registration; degradation is loss of federated reach, never loss of local function. |
| react-component-library | build-time | UI primitives are adopted rather than hand-rolled. | Component adoption; no runtime coupling. |
| scenario-dependency-analyzer | governance | Every third-party package is found and installed through it. No raw package manager, no hand-edited governance JSON. | `deps approved search`, `deps install`. |
| vrooli-events | automatic | Run correlation through the shared event bus. | `packages/api-core/eventbus`; no scenario-local analysis stack. |
| brand-manager | optional | A style record may carry a brand reference. That is the entire coupling; this scenario owns prose voice and never visual identity. | Reference resolution only. |
| program-runtime | consumer | A program should reach generation as a bounded typed operation returning a handle, so candidate sets never materialise into an agent's context. | Manifest-bound governed binding. |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| None. | not-applicable | Model providers are reached exclusively through ai-gateway, which owns provider selection, credentials, locality, and metering. A direct provider call from this scenario would bypass the boundary ai-gateway's own conformance phase exists to enforce. | n/a |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |
| ai-gateway unavailable | transport error or typed `INFERENCE_ERROR_CODE_UNAVAILABLE` | Generation refuses with a named error naming the role; styles, profiles, declarations, and every committed document stay readable. Never a partial candidate set presented as complete. | generation service tests with a stubbed gateway seam |
| ai-gateway schema rejection | `INFERENCE_ERROR_CODE_UNSUPPORTED_SCHEMA` | Surfaced verbatim with the offending construct named; the profile is reported infeasible rather than silently retried with a weaker schema. | generation service tests |
| Context window exceeded | gateway context refusal | Treated as **advisory about token counts, never recorded as a measured fact**. The static feasibility check should have refused the profile earlier; a refusal reaching here is a bug in that check, not a normal path. | feasibility tests, documents domain |
| Silent local truncation | none — this is the danger | Until ai-gateway sets a context ceiling on the local provider, prompt overflow truncates without erroring and the warning never reaches this scenario. Local-model measurements are therefore **not trustworthy** and must not be reported as acceptance evidence. | tracked as a gateway prerequisite, not maskable here |
| Declaration file malformed | parse error at scan | Registers `invalid` carrying the error; startup is never blocked; generation against it refuses by name. | declarations tests |
| Declaration key collision | two files claim one key | Both raise `declaration_key_collision` naming both paths. Never resolved last-writer-wins. | declarations tests |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
