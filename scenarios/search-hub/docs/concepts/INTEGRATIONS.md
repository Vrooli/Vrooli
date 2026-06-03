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
  the template's embedded SQLite (`modernc.org/sqlite`, CGO-clean) at
  `SQLITE_PATH`. There is no `postgres` dependency: Phase 2 proposed one,
  but Phase 3 stayed on SQLite (registry data is tiny; the pure-Go test
  harness needs no external service). See the Intentional Deviations
  table in [`ARCHITECTURE.md`](ARCHITECTURE.md).
- **`ollama` — required (`try_start` + `degraded_behavior`).** Serves
  the classifier (`qwen3:1.7b`) and reranker (`qwen3:4b`), both already
  pulled. Declared required so the quality layers have a home, but with
  explicit degradation: classifier down ⇒ fall back to explicit
  `--type`/`--all`; reranker down ⇒ by-provider grouping + `degraded`
  flag. (The `try_start` + `degraded_behavior` pairing is the
  intentional-degradation shape the schema validator requires.)
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
| SQLite (`SQLITE_PATH`) | embedded store | n/a (in-process) | n/a | registry, metrics | API unhealthy if the DB file is unwritable; no external service. |
| `ollama` | shared resource | yes | `try_start` + `degraded_behavior` | routing (classifier `qwen3:1.7b`), rerank (`qwen3:4b`) | Classifier down ⇒ explicit `--type`/`--all`; reranker down ⇒ by-provider grouping + `degraded` flag. |
| `cli-health` | scenario (provider) | no | `ignore` | providers / routing fan-out | Absent ⇒ provider skipped with warning. |
| `ui-health` | scenario (provider) | no | `ignore` | providers / routing fan-out | Absent ⇒ provider skipped with warning. |
| `swarm-manager` | scenario (provider) | no | `ignore` | providers / routing fan-out | Absent ⇒ provider skipped with warning. |
| `knowledge-observatory` | scenario (provider) | no | `ignore` | providers / routing fan-out | Absent ⇒ provider skipped with warning. |
| `prompt-manager` | scenario (provider) | no | `ignore` | providers / routing fan-out | Absent ⇒ provider skipped with warning. |
| Vrooli lifecycle | local platform | yes | n/a | API, UI, CLI | Start through lifecycle commands. |

> **Storage note (2026-06-03, Phase 3).** `.vrooli/service.json` keeps
> the template's `SQLITE_PATH` — it now backs the live `registry` store
> (the `notes` worked example was removed). The cross-scenario base URL
> for each provider is resolved at call-time via the backend resolver
> (never client-computed) — descriptors store a logical
> `{scenario_id, path}`, not a host:port.

## Vrooli Resources

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| SQLite (embedded) | in-process | Registry + metrics persistence (no external service). | If the registry outgrows single-writer SQLite. |
| `ollama` | required (degradable) | Classifier + reranker. | Swap reranker model when a cross-encoder lands (KO plan). |
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
| SQLite (`SQLITE_PATH`) | `PingContext` error (unwritable/corrupt DB file) | `/health` returns unhealthy dependency status. | health handler tests |
| `ollama` | classify/rerank call error or timeout | Degrade per policy above; surface in `status` (`classifier_available` / `reranker_available`). | routing/rerank degradation tests (Phase 5/6) |
| Provider scenario | unreachable within per-provider timeout | Skip provider, surface warning, return partial results. | federation-correctness tests (Phase 4) |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
