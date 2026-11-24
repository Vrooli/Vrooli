# Integrations — Search Hub

This document is the canonical dependency contract for resources,
other scenarios, and third-party services used by the scenario.


Use this document to answer:

- What does the scenario depend on?
- Which dependencies are required versus optional?
- Which domain uses each dependency?
- What is the failure or degradation behavior?
- Where is the dependency declared or configured?

## Dependency decisions (orientation rationale)

Search Hub's dependency posture follows from the thin-router invariant
([`ARCHITECTURE.md`](ARCHITECTURE.md)): it persists its registry, telemetry,
and evaluation state, may use a derived provider-description index for owner
selection, and reaches provider corpora at runtime without owning their data.

- **Storage — embedded SQLite (no resource dependency).** The `registry`
  (`providers`) and, later, `metrics` (`query_telemetry`) stores live in
  the template's embedded SQLite (`modernc.org/sqlite`, CGO-clean) at `api-core/storage`. There is no `postgres` dependency: Phase 2 proposed one,
  but Phase 3 stayed on SQLite (registry data is tiny; the pure-Go test
  harness needs no external service). See the Intentional Deviations
  table in [`ARCHITECTURE.md`](ARCHITECTURE.md).
- **`ollama` — degradable.** The interactive classifier role is retired.
  Ollama remains available for the LLM reranker fallback and optional eval
  corpus generation. If Ollama is unavailable, a healthy TEI reranker remains
  usable; if all rerank legs fail, results remain grouped by provider.
- **`reranker` — optional (`try_start`).** Dedicated TEI cross-encoder
  primary for unified federation rerank. It is preferred over Ollama because it
  is the right primitive for short-list relevance scoring and avoids cold LLM
  startup on the query hot path. If absent or unhealthy, the router falls back
  to the Ollama LLM rerank leg, then honest grouping.
- **Federated provider scenarios — soft (`startup_policy: ignore`).**
  Registered scenarios such as `agent-manager`, `cli-health`, `ui-health`,
  `swarm-manager`, `knowledge-observatory`, and `prompt-manager` are *runtime
  federation targets*, not boot
  prerequisites. `ignore` (not `try_start`) is deliberate: `try_start`
  force-boots all five when search-hub starts (observed a 4+ min hang),
  which is wrong for an optional federation. The hub degrades
  gracefully — a down/absent provider is skipped with a surfaced
  warning, never failing the query.
- **Qdrant stores no provider corpus.** The gated fleet path can store derived
  provider-description vectors and metadata for candidate selection. Owner
  operation, rights, quality evidence, capacity admission, and explicit index
  lifecycle govern that path. Provider search corpora and result vectors stay
  with their owner scenarios; no index cutover is implied by this dependency
  description.

## Dependency Inventory

| Dependency | Type | Required? | Startup Policy | Used By | Failure / Degradation Behavior |
|---|---|---|---|---|---|
| SQLite | embedded store | n/a (in-process) | n/a | registry, metrics | API unhealthy if the DB file is unwritable; no external service. |
| `ollama` | shared resource | no (degradable) | `try_start` + `degraded_behavior` | LLM rerank fallback, eval corpusgen, owner-gated description embedding | Interactive classifier is retired; when unavailable, TEI remains the primary rerank path and lexical provider selection remains available. |
| `qdrant` | shared resource | no (operation-gated) | `try_start` | versioned provider-description fleet index and embedding-arm generation | If semantic index or capacity admission is unavailable, do not run index mutation; the query path can report and use its supported fallback. |
| `reranker` | shared resource | no | `try_start` | routing rerank primary | Unavailable ⇒ Ollama LLM rerank fallback; all rerank legs unavailable ⇒ by-provider grouping + `degraded` flag. |
| `cli-health` | scenario (provider) | no | `ignore` | providers / routing fan-out | Absent ⇒ provider skipped with warning. |
| `ui-health` | scenario (provider) | no | `ignore` | providers / routing fan-out | Absent ⇒ provider skipped with warning. |
| `swarm-manager` | scenario (provider) | no | `ignore` | providers / routing fan-out | Absent ⇒ provider skipped with warning. |
| `knowledge-observatory` | scenario (provider) | no | `ignore` | providers / routing fan-out | Absent ⇒ provider skipped with warning. |
| `prompt-manager` | scenario (provider) | no | `ignore` | providers / routing fan-out | Absent ⇒ provider skipped with warning. |
| `agent-manager` | scenario (provider) | no | `ignore` | `agent-manager.runs` conversation-history fan-out and operator-reviewed caller harvesting | Run metadata/content reads use generated Connect `GetRun`/`GetRunEvents`. The privacy-filtered caller-source inventory keeps only Agent Manager's `GET /api/v1/runs/{id}/events/metadata` as REST because the owner has no RPC for that projection; its payload-free shape is checked by `TestAgentManagerMetadataHTTPReaderRequiresMetadataOnlyResponse`. Absent, stale, rebuilding, or lexical-only ⇒ the provider group reports its native coverage/degradation evidence; other groups remain available. |
| Vrooli lifecycle | local platform | yes | n/a | API, UI, CLI | Start through lifecycle commands. |

> **Storage note (2026-06-03, Phase 3).** `.vrooli/service.json` keeps
> the template's scenario database — it now backs the live `registry` store
> (the `notes` worked example was removed). The cross-scenario base URL
> for each provider is resolved at call-time via the backend resolver
> (never client-computed) — descriptors store a logical
> `{scenario_id, path}`, not a host:port.

## Vrooli Resources

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| SQLite (embedded) | in-process | Registry + metrics persistence (no external service). | If the registry outgrows single-writer SQLite. |
| `ollama` | optional/degradable | LLM rerank fallback, eval corpusgen, and owner-gated description embedding. | If the fallback role changes or corpusgen gets a dedicated low-cost role. |
| `reranker` | optional (degradable) | TEI cross-encoder primary for unified federation rerank. | If Search Hub no longer uses cross-provider rerank. |
| qdrant | operation-gated | Stores derived provider-description vectors, not owner corpus vectors; writes require rights evidence and disk admission. | If the semantic selection path is retired or its owner changes. |

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| Registered provider scenarios, including `agent-manager`, `cli-health`, `ui-health`, `swarm-manager`, `knowledge-observatory`, and `prompt-manager` | soft (federated) | Provider corpora are reached at runtime through self-registered descriptors. Search Hub stores descriptor and hashed-query telemetry only; corpus content remains with its owner. | Each leaf's declared search and status RPCs. Indexed providers additionally expose token-gated control RPCs. |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| None yet. | not-applicable | v1 federates project-internal providers only. The descriptor carries `scope = EXTERNAL` for future paid/web corpora. | Add when an external provider registers. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error (unwritable/corrupt DB file) | `/health` returns unhealthy dependency status. | health handler tests |
| `reranker` | TEI `/health` or `/rerank` error/timeout | Fall back to Ollama LLM rerank within the router budget. | shared-reranker chain adapter tests |
| `ollama` | rerank/embedding call error or timeout | Keep provider-description lexical selection; fall back from Ollama reranking to TEI or honest provider grouping; retain per-part degradation evidence. | routing/rerank degradation tests |
| Provider scenario | unreachable within per-provider timeout | Skip provider, surface warning, return partial results. | federation-correctness tests (Phase 4) |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness


Provider HTTP 401 and 403 responses remain degraded caller-visible results;
they do not by themselves establish a shared provider outage. Provider
authority is evaluated by the provider. Search Hub does not infer an empty
corpus from a refusal or automatically grant itself access.
