# Integrations — Search Hub

This document is the canonical dependency contract for resources,
other scenarios, and third-party services used by the scenario.

## Purpose Of This Document

Use this document to answer:

- What does the scenario depend on?
- Which dependencies are required versus optional?
- Which domain uses each dependency?
- What is the failure or degradation behavior?
- Where is the dependency declared or configured?

## Dependency decisions (orientation rationale)

Search Hub's dependency posture follows directly from the thin-router
invariant ([`ARCHITECTURE.md`](ARCHITECTURE.md)): it persists only its
own registry + telemetry, calls a local LLM for classify/rerank, and
reaches providers at runtime — never owning their data.

- **Storage — embedded SQLite (no resource dependency).** The `registry`
  (`providers`) and, later, `metrics` (`query_telemetry`) stores live in
  the template's embedded SQLite (`modernc.org/sqlite`, CGO-clean) at `api-core/storage`. There is no `postgres` dependency: Phase 2 proposed one,
  but Phase 3 stayed on SQLite (registry data is tiny; the pure-Go test
  harness needs no external service). See the Intentional Deviations
  table in [`ARCHITECTURE.md`](ARCHITECTURE.md).
- **`ollama` — required (`try_start` + `degraded_behavior`).** Serves
  the classifier, the LLM fallback reranker, and optional eval corpus
  generation; Search Hub declares only `classify.routing` and
  `rerank.llm_fallback`, both resolving to the resident 4B model. Explicit
  degradation: classifier down ⇒ fall back to explicit `--type`/`--all`;
  rerank fallback down ⇒ by-provider grouping after TEI also fails.
- **`reranker` — optional (`try_start`).** Dedicated TEI cross-encoder
  primary for unified federation rerank. It is preferred over Ollama because it
  is the right primitive for short-list relevance scoring and avoids cold LLM
  startup on the query hot path. If absent or unhealthy, the router falls back
  to the Ollama LLM rerank leg, then honest grouping.
- **Federated provider scenarios — soft (`startup_policy: ignore`).**
  `cli-health`, `ui-health`, `swarm-manager`, `knowledge-observatory`,
  `prompt-manager` are *runtime federation targets*, not boot
  prerequisites. `ignore` (not `try_start`) is deliberate: `try_start`
  force-boots all five when search-hub starts (observed a 4+ min hang),
  which is wrong for an optional federation. The hub degrades
  gracefully — a down/absent provider is skipped with a surfaced
  warning, never failing the query.
- **No qdrant.** The router holds no vectors; retrieval lives in
  providers. Declaring qdrant would invite data accretion and break the
  thin-router boundary (guarded by an architectural test).

## Dependency Inventory

| Dependency | Type | Required? | Startup Policy | Used By | Failure / Degradation Behavior |
|---|---|---|---|---|---|
| SQLite | embedded store | n/a (in-process) | n/a | registry, metrics | API unhealthy if the DB file is unwritable; no external service. |
| `ollama` | shared resource | yes | `try_start` + `degraded_behavior` | routing classifier, LLM rerank fallback, eval corpusgen | Classifier down ⇒ explicit `--type`/`--all`; LLM rerank fallback down ⇒ by-provider grouping when TEI also cannot answer. |
| `reranker` | shared resource | no | `try_start` | routing rerank primary | Unavailable ⇒ Ollama LLM rerank fallback; all rerank legs unavailable ⇒ by-provider grouping + `degraded` flag. |
| `cli-health` | scenario (provider) | no | `ignore` | providers / routing fan-out | Absent ⇒ provider skipped with warning. |
| `ui-health` | scenario (provider) | no | `ignore` | providers / routing fan-out | Absent ⇒ provider skipped with warning. |
| `swarm-manager` | scenario (provider) | no | `ignore` | providers / routing fan-out | Absent ⇒ provider skipped with warning. |
| `knowledge-observatory` | scenario (provider) | no | `ignore` | providers / routing fan-out | Absent ⇒ provider skipped with warning. |
| `prompt-manager` | scenario (provider) | no | `ignore` | providers / routing fan-out | Absent ⇒ provider skipped with warning. |
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
| `ollama` | required (degradable) | Classifier + LLM rerank fallback + eval corpusgen. | If the fallback role changes or corpusgen gets a dedicated low-cost role. |
| `reranker` | optional (degradable) | TEI cross-encoder primary for unified federation rerank. | If Search Hub no longer uses cross-provider rerank. |
| qdrant | deliberately excluded | Thin-router invariant; router holds no vectors. | Never (would break the boundary). |

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| `cli-health`, `ui-health`, `swarm-manager`, `knowledge-observatory`, `prompt-manager` | soft (federated) | Provider corpora reached at runtime through registered descriptors. Non-destructive: their own search surfaces are unchanged. | Each leaf's existing search RPC/CLI (see plan Appendix A.5). |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| None yet. | not-applicable | v1 federates project-internal providers only. The descriptor carries `scope = EXTERNAL` for future paid/web corpora. | Add when an external provider registers. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error (unwritable/corrupt DB file) | `/health` returns unhealthy dependency status. | health handler tests |
| `reranker` | TEI `/health` or `/rerank` error/timeout | Fall back to Ollama LLM rerank within the router budget. | shared-reranker chain adapter tests |
| `ollama` | classify/rerank call error or timeout | Degrade per policy above; surface in `status` (`classifier_available` / `reranker_available`). | routing/rerank degradation tests (Phase 5/6) |
| Provider scenario | unreachable within per-provider timeout | Skip provider, surface warning, return partial results. | federation-correctness tests (Phase 4) |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
