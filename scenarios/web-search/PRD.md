# Product Requirements Document (PRD)

## 🎯 Overview
Purpose: Add the missing "web" dimension to Vrooli's federated search and turn expensive web research into compounding, reusable local knowledge. `web-search` is a search-hub provider that (a) serves live web results from the local SearXNG resource and (b) maintains its own self-curating, citation-backed knowledge store of "findings" distilled from research runs, so the system answers more from what it has already internalized over time and reaches the rate-limited live web progressively less.

Target users: Vrooli agents operating mid-task and needing current or external information; human operators using unified search; the search-hub federation itself as a registered scope-aware provider. Also serves as the foundation for an in-Vrooli deep-research capability.

Deployment surfaces: Go API (proto-first Connect-RPC + REST `/health`), Go CLI (`web-search`), react-vite + Tailwind UI, and two registered search-hub providers (`web-search.live` and `web-search.learnings`).

Value proposition: Unified search can finally reach the live web rate-safely, and research stops being throwaway. Valuable findings persist locally, are semantically searchable, and self-curate. The result is compounding intelligence: more internalized knowledge over time, fewer external calls, and honest, citation-forward answers that surface uncertainty rather than conceal it.

## 🎯 Operational Targets

Operational targets are measurable outcomes; checkboxes may auto-update based on validation.

### 🔴 P0 – Must ship for viability
- [x] OT-P0-001 | L0 Live Web Search | Query the local SearXNG resource and normalize results into the search-hub `SearchHit` shape with no LLM involvement, serving as the fast path for external web queries.
- [x] OT-P0-002 | L1 Snippet Synthesis | Deliver an optional LLM-generated cited answer over the snippets SearXNG already returned (no page fetching), always additive to raw hits, always cited, and configured to abstain when sources conflict or are thin, never blocking raw results.
- [x] OT-P0-003 | Dual Search-Hub Provider Registration | Register `web-search.live` (SCOPE_EXTERNAL — rate-limited, reached only on explicit `--type web`/`--all` or fallback escalation) and `web-search.learnings` (SCOPE_PROJECT — local findings corpus, always safe to query, joins default routing) with self-registration at startup.
- [ ] OT-P0-004 | Scope-Aware Federated Blending | Ensure a default federated query routes to `web-search.learnings` and surfaces internalized findings without firing any live web request; live web joins only on explicit request or fallback escalation when project results are empty or weak.
- [x] OT-P0-005 | Findings Knowledge Store | Own a SQLite database (via api-core storage utils) with semantic indexing via aisearch-go, where the atomic unit is a "finding" (cited claim + source citations + retrieval date + confidence + status) and the container is a "brief" (one research run, holding many findings + provenance).
- [ ] OT-P0-006 | Findings Management CLI Commands | Expose CLI commands for browse/list, manual add, edit, supersede, flag, and prune of findings, usable by both human operators and (later) the L3 agent.
- [x] OT-P0-007 | Live-Web Result Cache and Budget Governor | Serve repeated queries from a TTL-based cache and enforce a token-bucket budget governor per time window on the live-web path, returning a graceful "rate-limited, try later" response rather than hammering SearXNG or external engines.
- [x] OT-P0-008 | Core UI | Ship a UI surface comprising a search box with results, query history, an ops panel (SearXNG engine reachability, cache hit-rate, budget remaining), and a findings management surface (browse, edit, supersede, flag, manual add).

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | L2 Fetch, Read, and Synthesize | Fetch top-N result pages via the browserless resource, extract readable text, and produce a single-pass cited synthesis, exposed as discrete API endpoints and agent tools.
- [ ] OT-P1-002 | L3 Iterative Research Agent | Spawn an agent-manager–managed research run that uses L2 endpoints and web-search CLI commands as tools, plans and decomposes the query, loops (search → read → find gaps → re-search), verifies across sources, and emits a cited brief, reusing existing agentic infrastructure with no hand-rolled loop plumbing.
- [x] OT-P1-003 | Research-and-Reconcile Loop | Configure the L3 agent as a librarian: Gather phase reads semantically near existing findings first (bounded sweep); Reconcile phase writes new findings, supersedes outdated ones, and flags contradictions; budget ordering answers the user first, then curates as a bounded post-step.
- [ ] OT-P1-004 | Finding Auto-Capture Policy | Auto-capture findings by default in L3 runs, offer opt-in capture via flag for L2, and never persist findings for L0/L1; an LLM distillation pass at run end emits structured findings (claim + citations + confidence).
- [x] OT-P1-005 | Contradiction Handling and Audit Trail | Supersede rather than hard-delete outdated findings (archived rows kept with provenance for recoverability and auditability); gate agent mutations (supersede/prune) behind confidence thresholds; surface flagged contradictions as first-class reviewable objects in a dispute review queue resolvable by human command, targeted research, or new evidence.
- [x] OT-P1-006 | Trust and Freshness Metadata | Attach citations, retrieval date, originating query/brief, and confidence to every finding; decay confidence score with age and display age visibly; enforce status = active / disputed / superseded with blended search returning active + disputed (disputed surfaces with a "sources conflict" warning and both sources) and excluding superseded by default, with superseded retrievable only via `--include-archived`.
- [x] OT-P1-007 | Dispute Review Queue UI Surface | Provide a dedicated UI panel listing all flagged contradictions with status, conflicting sources, and resolution controls (resolve, re-research, dismiss).

### 🟢 P2 – Future / expansion
- [x] OT-P2-001 | Usage-Telemetry-Driven Curation | Track whether each finding is actually surfaced or used (effectiveness-ledger style), allowing proven findings to persist while never-surfaced findings decay out automatically.
- [ ] OT-P2-002 | Classifier Auto-Routing to Live Web | Train the search-hub routing classifier to detect "web-shaped" queries and automatically include the live web provider, gated behind the proven cache and budget governor to manage rate-limit risk.
- [x] OT-P2-003 | Periodic Full-Store Consistency GC Run | Execute a scheduled, store-wide garbage-collection and consistency pass separate from the per-query reconcile step to prune permanently unresolvable or fully decayed findings.

## 🧱 Tech Direction Snapshot
Preferred stacks: Proto-first Go API with Connect-RPC; react-vite + Tailwind UI using the vrooli-default design kit and operational console visual language; declarative Go CLI. Mirror the cli-health and knowledge-observatory search adoption patterns throughout.

Preferred storage: Own SQLite database via api-core storage utils for findings and briefs metadata; aisearch-go + Qdrant for the semantic findings index using nomic embeddings; reranker resource for result ranking. The live web path is a passthrough to the SearXNG resource — nothing is stored except via the TTL cache and distilled findings written by L2/L3 capture.

Integration strategy: Shared workflows preferred over resource CLI over direct API. Reuse search-hub for provider registration and scope-aware routing; agent-manager for L3 research runs; aisearch-go for the learnings corpus; SearXNG resource for live web queries; browserless resource for L2 page fetch and text extraction; ollama for L1, L2, and L3 synthesis and distillation.

Non-goals: This scenario is not a replacement for, and does not write into, knowledge-observatory's curated documentation corpus — no doc or markdown files are created in KO. `web-search` owns its findings store exclusively. Live web is never on the default federated query path. Synthesis is always additive, always cited, and never blocks raw hits. Findings are never hard-deleted (supersede/archive only). Disputed findings are surfaced with a visible warning and are never silently resolved.

## 🤝 Dependencies & Launch Plan
Required resources: `searxng` (verify healthy and standards-current on this host before any P0 work begins — it exists and appears maintained); `ollama` (L1/L2/L3 synthesis); `qdrant` (semantic index); `reranker` (ranking); `browserless` (P1/L2 page fetch and extraction).

Scenario dependencies: `search-hub` (provider registration and scope-aware routing — must be available before either provider can self-register); `agent-manager` (required for P1/L3 iterative research runs).

Operational risks: SearXNG or external-engine rate limits and bans (mitigated by TTL cache, token-bucket budget governor, and scope-gating so default queries never reach the live web); trust of unvetted external content (mitigated by mandatory citations, retrieval-date stamps, disputed labels, surface-with-warning policy, and no silent resolution); findings store growth and rot over time (mitigated by reconcile-on-use, age-based confidence decay, and P2 telemetry-driven curation).

Launch sequencing: **P0** — confirm SearXNG resource is healthy, then ship L0/L1, both search-hub providers, findings store and CLI management commands, cache/governor, and core UI. **P1** — ship L2 page fetch and synthesis, L3 agent-manager research-and-reconcile loop, contradiction handling, trust/freshness metadata, and dispute review queue UI. **P2** — ship usage-telemetry-driven curation, classifier auto-routing to live web, and periodic full-store GC run.

## 🎨 UX & Branding
User experience: Search box with immediate results display; query history panel for replay and comparison; an ops panel exposing SearXNG engine reachability, cache hit-rate, and current budget remaining; a findings management surface for browse, edit, supersede, flag, and manual add. Disputed findings always carry a visible "sources conflict" warning with both conflicting sources shown. A dedicated dispute review queue surfaces all flagged contradictions with resolution controls. The interface is honest about uncertainty — explicit abstain messages and "sources disagree" labels rather than false confidence.

Visual design: Vrooli-default operational console design kit with full light and dark theme support. Vrooli-default design tokens govern color, typography, and spacing. PWA install surface is maintained via the seeded `ui/public/site.webmanifest`, `apple-icon-180.png`, `favicon-196.png`, and maskable manifest icons; generic placeholder icons are replaced when final product branding is available.

Accessibility: WCAG AA floor enforced per the vrooli-default design kit; all interactive controls keyboard-accessible; conflict warnings and status labels conveyed through both color and text so they are perceivable without color vision.