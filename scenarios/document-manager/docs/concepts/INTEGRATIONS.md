# Integrations — Document Manager

This document is the canonical dependency contract for resources,
other scenarios, and third-party services used by the scenario.

## Purpose Of This Document

Use this document to answer:

- What does the scenario depend on?
- Which dependencies are required versus optional?
- Which domain uses each dependency?
- What is the failure or degradation behavior?
- Where is the dependency declared or configured?

## Standing Rule

**No domain in this scenario may hold a provider credential, a provider
URL, a concrete model slug, a hard-coded context window, or a hard-coded
embedding dimension.** All inference flows through AI Gateway by role and
profile. All parsing runtimes are reached through resource CLIs. A
provider HTTP client appearing anywhere in this tree is an
`ai-conformance` ERROR finding (`ai.direct_ollama_http` /
`ai.direct_openrouter_http`) and fails the phase.

**And one narrower rule that carries the product claim:** every outbound
`GatewayRequest` is constructed at the single `internal/gatewayreq`
choke point, which applies the privacy-class→profile mapping and refuses
to emit a profile weaker than the document's class. The gateway's
fail-closed behavior attaches to the *profile*, not the privacy class —
`PROFILE_LOCAL_FIRST` is documented to fall back to a remote provider —
so a second construction site anywhere in the tree silently voids the
residency guarantee. An AST check asserts there is exactly one.

## Declaration Status

Dependencies below are **planned, not yet declared**.
`.vrooli/service.json` still reflects the generated default (SQLite,
standalone) because no domain implementation exists yet. Each row names
the gate that must pass before it is added, per Gate 4 of
[`../START-HERE.md`](../START-HERE.md): document the reason here first,
then edit the service manifest.

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes | API, all persistence-backed domains | `SQLITE_PATH` lifecycle env var | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario must be started through lifecycle commands. |
| Routed file-store seam | platform seam | yes | `intake`, `derivation`, `corpus` | `filerouting.RoutedRoots` picked per request | Test-mode isolation breaks if bypassed; enforced by the `storage` phase. |
| AI Gateway | scenario | yes (P0) | `enrichment`, `sensitivity`, `custody` | Connect-RPC; role + profile requests | Enrichment degrades to unavailable; parse tiers 1–2 keep working. |
| pdf-inspector | resource | yes (P0) | `derivation` (tier 1) | resource CLI | Tier 1 unavailable → escalate to tier 2 and record the reason. |
| `unstructured-io` | resource | yes (P0) | `derivation` (tier 2) | `resource-unstructured-io` CLI | Tier 2 unavailable → non-PDF formats cannot be derived; document enters an error state, bytes are retained. |
| `storage-manager` | scenario | yes (P0) | `corpus` | storage kind + retention declaration | Retention is unenforced; the corpus grows unbounded. Never a silent condition. |
| `search-hub` | scenario | yes (P0) | `retrieval` | `.vrooli/search.json` descriptor, self-registered via `searchregister-go` | The corpus stops being *discoverable* — a caller that does not already know it wants this corpus cannot find it, and no query spans sources and findings together. Direct Connect calls still work, and local `retrieval` still answers, so the scenario is fully functional standalone. Registration runs in a background goroutine and must never block startup. |
| ledger engine (via `vrooli-memory` today) | scenario | **no (P1)** | `handoff` | Connect-RPC; scoped append of *findings* | Publication queues locally and retries. **The corpus stays fully usable — this is an optional integration, not a dependency.** |
| `landing-page-business-suite` | scenario | no (P1) | `enrichment`, `derivation` (tier 3) | credit reserve/execute/finalize | Metered tier unavailable; free tiers and BYOK unaffected. |
| `file-tools` | scenario | no | `intake` | CLI | Archive expansion unavailable; single-file intake unaffected. |

## Vrooli Resources

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| pdf-inspector | **planned — not packaged** | Tier-1 native PDF classification and extraction. Rust library with Node, Python and CLI bindings; a CLI-shaped resource is the light path. This is the resource that makes the free tier viable, since roughly half of PDFs never need OCR. | Package the resource, then declare. Blocks `DOC-P0-004`. |
| `unstructured-io` | **exists — unverified** | Tier-2 structural parse for DOCX, HTML, EPUB and tables. Its README states it is mid-migration to the current `docker-service` structure. | Verify it starts and responds before relying on it. Blocks `DOC-P0-005`. |
| `qdrant` | deferred | Embeddings live in SQLite beside the units they describe as `float32` blobs, scanned in-process — sufficient at single-host corpus sizes and it keeps the scenario dependency-free. This is a *scale* decision, not a boundary one: the boundary is that `retrieval` answers over this corpus only, and no store choice changes that. | If linear scan latency becomes unacceptable. The migration path is `ai-go/search`'s `VectorStore` abstraction, which `signal-inbox` already uses against Qdrant — swapping the semantic half behind it changes no ownership rule and no privacy filter. |
| `postgres` | not-applicable | SQLite covers metadata, derivations, anchors and custody. The retired scenarios used Postgres for CRUD that no longer exists. | Only if multi-writer concurrency becomes real. |
| `vault` | deferred | The retired `secure-document-processing` used Vault for an encryption story that was never the differentiator. Encryption at rest is table stakes, not the wedge. | If a buyer requires managed key custody beyond OS-level disk encryption. |
| `minio` | deferred | The artifact store is filesystem-backed through the routed seam. Object storage matters only for multi-host deployment. | At `DOC-P2-005` (air-gapped/deployed profiles) or multi-host operation. |

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| `ai-gateway` | **planned — required** | Every inference call. Also the source of the `RouteEvidence` records the custody receipt is assembled from, and the enforcer of fail-closed routing for confidential documents. | Connect-RPC. Requests carry `role`, `profile`, `privacy_class`, and a `required_vram_bytes` footprint for local vision work. |
| `storage-manager` | **planned — required** | Governs artifact-store retention. Raw bytes are `regenerable: false`; derivation outputs and the retrieval index are `regenerable: true`. | Storage kind declarations plus budgets. |
| `search-hub` | **planned — required** | The only **discovery** path into this corpus, and the only place a caller can query sources and findings together. Promoted from P1: without it the corpus is undiscoverable, even though it is not unreachable. A caller that already knows it wants this corpus can and should call the Connect API directly — `DOC-P0-020` exists precisely so that path is a first-class contract. | `.vrooli/search.json` provider descriptor; boot-time self-registration through `packages/searchregister-go`. |
| ledger engine | **planned — optional (P1)** | A consumer may record a *finding* about this corpus into a ledger scope, citing an anchor. Not a dependency: this scenario neither imports the ledger nor requires it to be running. | Scoped, idempotent append of small entries with an anchor URI as provenance. Needs per-scope CRUD, which does not exist yet — and no P0 waits on it. |
| `landing-page-business-suite` | **planned — P1** | Credits for the metered tier. | Reserve → execute → finalize; HTTP 402 handled with an upgrade path. |
| `file-tools` | **planned — optional** | Archive and container expansion during intake rather than reimplementing it. | CLI. |
| `text-tools` | evaluate | Some normalization may already exist there. Check before writing our own. | CLI. |

### Known Gaps In Upstream Contracts

These are dependencies on work that does not exist yet. They are risks
recorded in `PRD.md` and repeated here because they are integration
facts, not scenario-internal ones.

| Gap | Impact | Owner |
|---|---|---|
| AI Gateway has no vision or multimodal role. Baseline roles cover chat, summarize, classify, extract, embedding, rerank, code and agent — nothing maps page image to text. | **Blocks `DOC-P1-001`** (tier-3 vision parse) entirely. Building it without the role means a direct provider call, which fails `ai-conformance`. | `ai-gateway` + `resource-ollama` / `resource-openrouter` policy. |
| `RouteEvidence` carries no caller correlation key, and `ListRouteEvidence` filters only by `limit` and `scenario`. | A receipt built today is self-attested by this scenario, with no independent gateway-side record to corroborate it. Weakens `DOC-P1-013` (attestation export) from evidence to self-report. | `ai-gateway`. |
| OpenRouter `embedding.default` is unverified. | BYOK embeddings — this scenario's most frequent paid operation — are local-only. | `resource-openrouter`. |
| The ledger engine is not yet its own scenario — the engine still lives inside `vrooli-memory`, which has no verb that creates, lists, or seeds a scope. | Blocks `DOC-P1-023` only. **No P0 depends on it**, and the extraction is deliberately off this scenario's critical path. Extracting it should wait until `vrooli-memory`'s own production-readiness work lands, since that plan's phase order is load-bearing. | `vrooli-memory`, then a new ledger-engine scenario. |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| None. | not-applicable | Every external capability is reached through a resource or another scenario. This scenario opens no outbound connection of its own — which is the product claim, so it is enforced rather than incidental. | — |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` reports unhealthy dependency status. | health handler tests |
| pdf-inspector | non-zero exit, missing binary, malformed output | Escalate to tier 2; record the escalation reason in the derivation and the receipt. Never silently produce an empty parse. | derivation tier-router tests |
| `unstructured-io` | unreachable, timeout, malformed response | Document enters an explicit error state; bytes are retained so a later retry costs nothing. | derivation integration tests |
| AI Gateway unreachable | connect error | Enrichment unavailable; tiers 1–2 continue. Documents stay in the corpus, unenriched and visibly so. **Retrieval degrades rather than fails:** units ingested during the outage have no vectors, so the semantic half cannot see them and the lexical (FTS5) half answers alone. The response must mark the index partial — a silently narrowed result set reads as "no matches," which is the wrong answer, not a degraded one. | enrichment service tests; retrieval degraded-mode tests |
| AI Gateway rejects a route | `no_eligible_route`, `insufficient_capacity` | Surface the rejection reason verbatim. **For a confidential document this is the correct outcome, not an error to work around** — it is `DOC-P0-013` behaving. | sensitivity + enrichment tests |
| Ledger engine unreachable | connect error | Publication queues and retries; the corpus remains fully usable. Nothing about ingest, derivation, anchoring or retrieval degrades. | handoff service tests |
| `search-hub` unreachable | registration or query error | Local `retrieval` continues to answer; the corpus is simply not federated. Surfaced as a warning, never silent. | retrieval provider tests |
| `storage-manager` absent | no enforcement | Retention unenforced. Surfaced as a warning, never silent. | corpus retention tests |
| LPBS insufficient credits | HTTP 402 | Short-circuit with a clear top-up path. Never partially deliver. Free tiers unaffected. | metering tests |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DOMAINS.md`](DOMAINS.md) — which domain uses each dependency
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
