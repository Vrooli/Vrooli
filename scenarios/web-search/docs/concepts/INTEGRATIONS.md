# Integrations — Web Search

This document is the canonical dependency contract for resources,
other scenarios, and third-party services used by the scenario.

> **Scaffold status (2026-06-09):** Dependencies are declared in
> `.vrooli/service.json` and described here as the *intended* contract.
> Wiring is not implemented yet. `enabled: false` entries (browserless,
> agent-manager) are P1 dependencies turned on when their level ships.

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
| SQLite | embedded storage | yes | API (findings/briefs/audit) | resolved by `api-core/storage` from the scenario id | API reports unhealthy if unreachable. |
| SearXNG | resource | optional (try_start) | livesearch (L0/L1), research (L2 source) | JSON search API (`SEARXNG_URL`) | `web-search.live` degrades to unavailable; learnings unaffected. |
| Qdrant | resource | optional (try_start) | findings (semantic index) | aisearch-go collection `web-search-findings` | Findings recall falls back to text matching; reindex deferred. |
| Ollama | resource | optional (try_start) | findings (embeddings), livesearch/research (synthesis, distillation) | `embedding.default` + small chat role | Embeddings/synthesis unavailable; raw hits still returned. |
| reranker | resource | optional (try_start) | findings (ranking) | TEI cross-encoder (bge-reranker-v2-m3) | Falls back to raw dense order. |
| browserless | resource | optional, **P1** (disabled) | research (L2 fetch/extract) | headless browser fetch | L2/L3 fall back to snippet-only (L1) synthesis. |
| search-hub | scenario | optional (try_start) | federation (registration) | RegistryService self-registration from `.vrooli/search.json` | Retries briefly then serves locally; re-registers next boot. |
| agent-manager | scenario | optional, **P1** (disabled) | research (L3 runs) | agent run lifecycle | Only L0/L1/L2 available; L3 unavailable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Start through lifecycle commands. |

## Vrooli Resources

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| searxng | enabled (try_start) | The live-web engine; the whole external-search value (L0/L1/L2 source). | **Verify healthy/standards-current on this host before P0 work** — it exists and appears maintained. |
| ollama | enabled (try_start) | Embeddings for the findings index + LLM for L1/L2/L3 synthesis & distillation. | — |
| qdrant | enabled (try_start) | Semantic index of findings (collection `web-search-findings`). | — |
| reranker | enabled (try_start) | Reorders the findings shortlist for best ordering. | — |
| browserless | disabled (P1) | Page fetch + readable-text extraction for L2/L3. | Enable when OT-P1-001 (L2) lands. |

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| search-hub | enabled (try_start) | Federated routing. web-search self-registers `web-search.live` (SCOPE_EXTERNAL) and `web-search.learnings` (SCOPE_PROJECT). | Idempotent `RegisterProvider` upsert from `.vrooli/search.json`; control-token handshake gates mutations (overrides/reindex/config). |
| agent-manager | disabled (P1) | L3 iterative research runs reuse agent-manager's agentic loop (no hand-rolled plumbing). | Spawn a run that calls L2 endpoints + `web-search` CLI as tools; emits a cited brief. Enable when OT-P1-002 lands. |

### search-hub registration contract (the shape this scenario must honor)

Each provider descriptor in `.vrooli/search.json` declares:
`provider_id`, `provider_group` (`web-search`), `bucket`, `type`,
`description` (for classifier routing), `endpoint` (HTTP+JSON with
`{{query}}`/`{{limit}}` templating), `result_mapping` (JSON-path
selectors → unified `SearchHit`), `scope`, and `tuning`. web-search
declares **two**:

- `web-search.live` — `scope: SCOPE_EXTERNAL`, backed by the livesearch
  endpoint. Reached only on explicit `--type web`/`--all` or fallback
  escalation. **Never on a default federated query.**
- `web-search.learnings` — `scope: SCOPE_PROJECT`, backed by the findings
  semantic-search endpoint. Joins default routing like any local corpus.

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| External search engines | indirect | SearXNG aggregates Google/Bing/DuckDuckGo/Startpage. | web-search never calls them directly — only via the local SearXNG resource. Rate-limit risk is mitigated by the cache + budget governor (OT-P0-007). |
| Fetched web pages | indirect (P1) | L2 fetches result pages for extraction. | Via browserless; extracted text feeds synthesis/distillation, not stored wholesale. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests (planned) |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` unhealthy. | health handler tests |
| SearXNG | connection error / non-200 | `web-search.live` returns degraded + surfaced warning; governor "try later". | livesearch integration tests |
| Qdrant | unreachable | findings recall → text fallback. | findings integration tests |
| Ollama | unreachable | no embeddings/synthesis; raw hits only. | livesearch/findings tests |
| search-hub | unreachable at boot | retry briefly, serve locally, re-register next boot. | federation registration tests |
| Budget exhausted | governor token bucket empty | graceful "rate-limited, try later", no external call. | governor unit tests |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
