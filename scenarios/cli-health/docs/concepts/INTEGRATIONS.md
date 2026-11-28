# Integrations — CLI Health

## Purpose Of This Document

This guide records CLI Health's declared resources, scenario dependencies,
integration contracts, and failure behavior.

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes | API, notes reference | resolved by `api-core/storage` from the scenario id | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario should be started through lifecycle commands. |

## Vrooli Resources

## Vrooli Resources

| Resource | Enabled | Required | Startup policy | Purpose | Degraded behavior |
|---|---:|---:|---|---|---|
| qdrant | yes | no | `try_start` | Vector database for cross-scenario CLI command semantic search (collection: `cli-health-commands`). | Semantic search falls back to BM25-style text matching; reindex is deferred. |
| ollama | yes | no | `try_start` | Local embedding role for command index generation (`embedding.default`). | Embedding generation is unavailable; text-mode search still works. |
| reranker | yes | no | `try_start` | Cross-encoder reranker (BAAI/bge-reranker-v2-m3 via TEI) reorders the dense command-search shortlist. | Reranking falls back to raw dense (cosine) order; search still returns results, only the final ordering is less precise. Local-LLM fallback is best-effort: qwen3:1.7b cannot discriminate and qwen3:4b is too slow (~38s/query), so the cross-encoder is effectively required for the rerank quality win. |

## Scenario Dependencies

| Scenario | Enabled | Required | Startup policy | Purpose | Degraded behavior |
|---|---:|---:|---|---|---|
| search-hub | yes | no | `try_start` | CLI Health self-registers its `cli-health.commands` provider from `.vrooli/search.json`, allowing federated command search and quality sweeps. | Registration retries briefly and then gives up if Search Hub is down; CLI Health continues serving command search and retries on its next boot. |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| None yet. | not-applicable | Generated scenario has no third-party dependency. | Add when PRD/requirements require external APIs, webhooks, auth, payments, or data feeds. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
