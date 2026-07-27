# Integrations — Vrooli Memory

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
| SQLite | embedded storage | yes | journal, facets, forest, harness | `SQLITE_PATH` lifecycle env var | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario must be started through lifecycle commands. |
| ai-gateway | scenario | yes | journal (classify, embed), forest (summarize) | scenario CLI/API | **Writes degrade, never fail.** An entry is appended unclassified and queued; compaction pauses until inference is available. |
| search-hub | scenario | yes (for federated reach) | federation | `.vrooli/search.json` descriptor + `RegisterProvider` | Local `recall` keeps working; only cross-corpus federated query loses the memory provider. |
| vrooli-events | scenario | no | harness | api-core receipt publication (automatic) | Run correlation is absent; writes and recall are unaffected. |
| swarm-manager | scenario | no | harness (work-record migration) | records read for one-time import | Migration deferred; work-record memories still write normally. |

## Vrooli Resources

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| SQLite (embedded) | active | Journal, facet, and tree tables are single-writer, local, and modest in size. No shared resource is warranted. | If memory becomes multi-host or the vector index outgrows embedded storage. |
| ollama | indirect | Reached **through ai-gateway**, never directly. Embedding and summarization are inference calls, and ai-gateway owns model policy, routing, and capacity. | Never call ollama directly from this scenario — that would duplicate policy the gateway owns. |
| qdrant | not-applicable | Vector storage goes through the shared `aisearch-go` package used by every existing search provider, keeping this scenario's index shape identical to the rest of the fleet. | If corpus size makes an embedded index untenable. |

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| ai-gateway | required | Facet classification, facet-text derivation, embedding, and compaction summarization. | Scenario CLI/API. Model policy stays in the gateway. |
| search-hub | required | Federated retrieval. Memory registers a provider descriptor; the router holds no memory content and no vectors. | `.vrooli/search.json` validated against `.vrooli/schemas/search.schema.json`; boot self-registration. |
| vrooli-events | automatic | Run correlation for memories written inside an agent run. **No integration work** — `api-core/server.go` wraps every handler in `eventbus.AutomaticRuntime`, so receipts carrying run id, workflow execution id, actor kind, and identity token are published for every endpoint already. | Memory stores correlation ids only and never copies run payloads. |
| swarm-manager | migration-only | Work records are absorbed as a memory kind (`VROOLIME-P1-001`). One-time import of existing records; the `records create` write path is retired from agent-facing docs. | Read-only import. |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| None. | not-applicable | Every inference call routes through ai-gateway, and all storage is local. This scenario reaches no external network service. | Add only if a hosted embedding or summarization provider is ever introduced — which would go through ai-gateway, not here. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |
| ai-gateway (classify) | call error or timeout | Entry is appended with an unclassified facet and enqueued for retry. **A write is never lost to an inference failure.** | `VROOLIME-P0-002` |
| ai-gateway (summarize) | call error or timeout | Compaction pass aborts cleanly; the frontier stays over target until inference returns. No partial summary is written. | `VROOLIME-P0-007` |
| search-hub | registration failure at boot | Scenario starts and serves local recall; registration retries. Federated query loses the memory provider until it succeeds. | `VROOLIME-P0-009` |
| vrooli-events | receipt not published, or null run id | Memory is written with no correlation. Documented as expected: `run_id` is nullable for heartbeat-spawned agents. | `VROOLIME-P1-002` |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
