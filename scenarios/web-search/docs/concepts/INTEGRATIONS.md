# Integrations — Web Search

This document is the canonical dependency contract for resources,
other scenarios, and third-party services used by the scenario.

> **Current status (2026-09-06):** Dependencies are declared in
> `.vrooli/service.json` and wired through the scenario lifecycle. Optional
> SearXNG, Qdrant, Ollama, Agent Manager, and BAS paths report typed
> unavailable or degraded outcomes when they cannot be reached.


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
| browser-automation-studio | scenario | optional (`try_start`) | research (L2 JS-page escalation) | Discovery-resolved `/ux-capture/public` frozen-page bundle | L2 remains HTTP-only when the scenario or browser escalation is unavailable; L0/L1 are unaffected. |
| search-hub | scenario | optional (try_start) | federation (registration) | RegistryService self-registration from `.vrooli/search.json` | Retries briefly then serves locally; re-registers next boot. |
| agent-manager | scenario | optional (`try_start`) | research (L3 runs) | Durable Agent Manager run lifecycle | L3 is unavailable while Agent Manager is unavailable; L0/L1/L2 remain available. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Start through lifecycle commands. |

## Vrooli Resources

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| searxng | enabled (try_start) | The live-web engine; the whole external-search value (L0/L1/L2 source). | **Verify healthy/standards-current on this host before P0 work** — it exists and appears maintained. |
| ollama | enabled (try_start) | Embeddings for the findings index + LLM for L1/L2/L3 synthesis & distillation. | — |
| qdrant | enabled (try_start) | Semantic index of findings (collection `web-search-findings`). | — |
| reranker | enabled (try_start) | Reorders the findings shortlist for best ordering. | — |

## Scenario Dependencies

The scenario rows below mirror `.vrooli/service.json`. Resource-backed search
dependencies are listed separately above; browserless is no longer a resource
dependency. The browser leg is supplied by browser-automation-studio.

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| browser-automation-studio | enabled (`try_start`) | Browser escalation for HTTP failures and JS-shell pages in L2. | Web-search resolves the scenario URL, requests `/ux-capture/public`, validates and reads the frozen-page artifact, then removes the ephemeral bundle. |
| search-hub | enabled (try_start) | Federated routing. web-search self-registers `web-search.live` (SCOPE_EXTERNAL) and `web-search.learnings` (SCOPE_PROJECT). | Idempotent `RegisterProvider` upsert from `.vrooli/search.json`; control-token handshake gates mutations (overrides/reindex/config). |
| agent-manager | enabled (`try_start`) | Owns iterative L3 research-and-reconcile runs. | Starts a durable run using L2 tools; when unavailable, L3 is unavailable while L0–L2 remain usable. |
| vrooli-memory | enabled (`try_start`) | Optional attempt capture and learning-read advice. | When unavailable, web-search operates without advice and program-runtime queues captures for delivery. |

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
| Fetched web pages | indirect | L2 fetches result pages over HTTP and escalates difficult pages through browser-automation-studio. | Extracted text feeds synthesis/distillation; external page bundles are ephemeral. |

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
