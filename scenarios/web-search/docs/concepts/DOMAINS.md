# Domains — Web Search

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

> **Scaffold status (2026-06-09):** This map describes the *intended*
> bounded contexts derived from `PRD.md` and the requirements registry.
> No product domain is implemented yet — the `notes` worked example and
> the `health` infra domain are still present from the template and will
> be removed once the first real domain (`livesearch`) is green
> (orientation Gate 6/7). Domain source paths below are the planned
> layout, not yet on disk.

## Purpose Of This Document

Use this document to answer:

- What product capabilities does this scenario expose?
- Which domain owns each concept, table, proto, endpoint, UI feature,
  CLI command, and test surface?
- Which concepts are shared, deferred, or deliberately not domains?

System-level architecture belongs in [`ARCHITECTURE.md`](ARCHITECTURE.md).
Workflow details belong in [`FLOWS.md`](FLOWS.md). Storage details
belong in [`DATA.md`](DATA.md). Dependency contracts belong in
[`INTEGRATIONS.md`](INTEGRATIONS.md).

## The Shape In One Paragraph

web-search has two faces. The **live** face is an external passthrough:
it queries the local SearXNG resource for fresh web results (L0), can
synthesize a cited answer over the returned snippets (L1), and protects
external engines with a cache + budget governor. The **learnings** face
is a local, self-curating knowledge store: research runs (L2 single-pass;
L3 iterative, via agent-manager) distill **findings** (cited claims)
grouped into **briefs** (one run), persisted in the scenario's own
SQLite and semantically indexed via aisearch-go. Both faces register
with search-hub as separate scope-tagged providers, so a default
federated query surfaces internalized findings (project scope) without
ever firing a rate-limited live web call (external scope).

## Domain Inventory

| Domain | Purpose | Primary Archetype | Owns Data | Surfaces | Requirements | Source Paths (planned) |
|---|---|---|---|---|---|---|
| livesearch | Live web results from SearXNG (L0) + optional cited snippet synthesis (L1), guarded by a TTL cache and a token-bucket budget governor. | External query / passthrough | Cache entries only (ephemeral, TTL'd). No durable corpus. | API, UI | OT-P0-001, OT-P0-002, OT-P0-007 | `api/internal/livesearch/`, `api/handlers/livesearch/`, `ui/src/features/search/`, `packages/proto/schemas/web-search/v1/livesearch/` |
| findings | The learnings store: durable `finding` + `brief` records in own SQLite (api-core storage) with an aisearch-go semantic index; CRUD, status lifecycle, freshness decay, visibility filtering, audit trail. | Knowledge store / entity | Findings, briefs, citations, audit log. | API, CLI, UI | OT-P0-005, OT-P0-006, OT-P1-006 | `api/internal/findings/`, `api/handlers/findings/`, `cli/domains/findings/`, `ui/src/features/findings/`, `packages/proto/schemas/web-search/v1/findings/` |
| research | Orchestrates L2 (fetch top-N via browserless → extract → single-pass cited synthesis) and L3 (agent-manager research-and-reconcile run); LLM distillation into findings; contradiction handling + dispute queue. | Orchestration / agentic workflow | Research run state (briefs are co-owned with findings). | API, CLI, UI | OT-P1-001, OT-P1-002, OT-P1-003, OT-P1-004, OT-P1-005, OT-P1-007 | `api/internal/research/`, `api/handlers/research/`, `cli/domains/research/`, `ui/src/features/research/`, `packages/proto/schemas/web-search/v1/research/` |
| federation | search-hub provider descriptors + self-registration of `web-search.live` (SCOPE_EXTERNAL) and `web-search.learnings` (SCOPE_PROJECT); the scope-aware blending contract. | Integration / registration | Provider descriptors (`.vrooli/search.json`); control token (in-memory). | API (the two provider Search endpoints), boot registration | OT-P0-003, OT-P0-004 | `api/internal/federation/`, `.vrooli/search.json`, `api/handlers/{livesearch,findings}/` (the provider endpoints) |
| curation *(deferred, P2)* | Usage-telemetry-driven finding curation, classifier auto-routing to live web, periodic full-store consistency GC. | Background maintenance | Usage/effectiveness telemetry. | API, CLI | OT-P2-001, OT-P2-002, OT-P2-003 | `api/internal/curation/` (not yet) |
| health *(infra)* | Report runtime readiness and dependency reachability (SearXNG/Qdrant/Ollama/search-hub). | Reporting / query | No product data. | API, UI | Scaffold health. | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/web-search/v1/health/` |

## Domain Details

### livesearch

- Purpose: turn a query into fresh web results and (optionally) a cited
  answer, without overrunning external rate limits.
- Primary archetype: external query / passthrough.
- Owns: SearXNG JSON client, result→SearchHit normalization, L1 snippet
  synthesis (always cited, additive, abstains on conflict), the TTL
  result cache, and the token-bucket budget governor.
- Does not own: any durable corpus (findings live in `findings`), routing
  decisions (those are search-hub's), page fetching/extraction (that is
  `research`/L2).
- Storage: ephemeral cache entries only (see [`DATA.md`](DATA.md)).
- Key invariants: synthesis never blocks raw hits; the governor returns a
  graceful "rate-limited, try later" rather than hammering SearXNG.
- Requirements: OT-P0-001 (L0), OT-P0-002 (L1), OT-P0-007 (cache+governor).

### findings

- Purpose: persist and semantically serve what the system has learned
  from web research, with full provenance and honest trust signals.
- Primary archetype: knowledge store / entity.
- Owns: `finding` records (cited claim + citations + retrieval date +
  confidence + status), `brief` containers, the per-finding audit log,
  the aisearch-go index (collection `web-search-findings`), the status
  state machine (`active`/`disputed`/`superseded`), age-based score decay,
  and the default visibility filter (`status != superseded`).
- Does not own: how findings are produced (that is `research`), how they
  are routed externally (that is `federation`).
- Surfaces: API (read/write), CLI (`web-search findings ...` management:
  list/add/edit/supersede/flag/prune), UI findings-management surface.
- Key invariants: findings are never hard-deleted (supersede/archive
  only); every mutation is audited (what/why/which brief); disputed
  findings are surfaced *with* a warning, never silently resolved.
- Requirements: OT-P0-005 (store), OT-P0-006 (CLI mgmt), OT-P1-006
  (trust & freshness).

### research

- Purpose: deepen a query beyond raw results and feed the learnings
  store, while keeping the store consistent.
- Primary archetype: orchestration / agentic workflow.
- Owns: L2 pipeline (fetch top-N via browserless → extract readable text →
  single-pass cited synthesis), L3 run orchestration (delegated to
  agent-manager), the research-and-reconcile loop (gather-existing-first
  → reconcile), the LLM distillation pass that emits structured findings,
  auto-capture policy (L3 on by default, L2 opt-in, L0/L1 never),
  confidence-gated contradiction handling, and the dispute review queue.
- Does not own: the agentic loop machinery itself (reused from
  agent-manager — OT-P1-002 is explicitly "no hand-rolled loop plumbing");
  finding persistence (writes through `findings`).
- Surfaces: API (start run, run status, dispute queue), CLI
  (`web-search research ...`), UI (research view + dispute queue).
- Key invariants: answer the user's question first within budget, curate
  as a bounded post-step; act on contradictions only above a confidence
  threshold, otherwise flag.
- Requirements: OT-P1-001..005, OT-P1-007.

### federation

- Purpose: make both faces of web-search reachable through the unified
  search-hub router, with the scope split that keeps live web off the
  default path.
- Primary archetype: integration / registration.
- Owns: the `.vrooli/search.json` descriptors for `web-search.live`
  (SCOPE_EXTERNAL) and `web-search.learnings` (SCOPE_PROJECT), the
  idempotent self-registration client (mirrors cli-health/KO), the
  control-token handshake, and the ResultMapping for each provider.
- Does not own: the routing/classifier/rerank logic (search-hub's), the
  underlying search implementations (`livesearch` and `findings`).
- Key invariant: `web-search.live` is reachable only via explicit
  `--type web`/`--all` or fallback escalation; never on a default query.
- Requirements: OT-P0-003 (dual registration), OT-P0-004 (scope-aware
  blending).

### curation *(deferred — P2)*

- Purpose: keep the learnings store healthy at scale and let live-web
  routing become smarter over time.
- Status: deferred. Planned requirements OT-P2-001 (usage-telemetry
  curation), OT-P2-002 (classifier auto-routing to live web), OT-P2-003
  (periodic full-store consistency GC).
- Revisit trigger: once the findings store has meaningful volume and
  per-query reconcile + age-decay prove insufficient on their own.

### health *(infra, from scaffold)*

- Purpose: expose API/dependency readiness; show the UI can read live
  backend state. Probes SearXNG, Qdrant, Ollama, and search-hub
  reachability in addition to the local DB.

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Finding | An atomic, cited claim distilled from research; the indexed unit. | `findings` |
| Brief | The container for one research run; holds many findings + provenance. | `findings` (produced by `research`) |
| Provider | A scope-tagged search-hub registration backed by one of the two faces. | `federation` |
| Scope | `SCOPE_PROJECT` (learnings, always-on) vs `SCOPE_EXTERNAL` (live web, gated). | `federation` (declared); search-hub (enforced) |
| Level (L0–L3) | Depth ladder: raw → snippet-synth → fetch+synth → iterative research. | `livesearch` (L0/L1), `research` (L2/L3) |
| Seam | Test-substitutable boundary wired once in production. | [`../internal/SEAMS.md`](../internal/SEAMS.md) |

## Deferred Domains

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| curation | Needs a populated store + evidence that reconcile/decay are insufficient. | P2; store reaches meaningful volume. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database infrastructure.
- `api/internal/testutil/` — cross-domain test harnesses.
- `ui/src/components/` — shared presentation primitives.

The SearXNG client, browserless client, Ollama/embeddings client, and
aisearch-go wiring are **infrastructure adapters** used by the domains
above, not domains themselves.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — research-and-reconcile loop, scope routing, dispute state machine
- [`DATA.md`](DATA.md) — findings/briefs storage and the index
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — SearXNG / search-hub / agent-manager / browserless contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
