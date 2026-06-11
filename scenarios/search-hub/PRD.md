# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md`
> **Validation**: Enforced by `prd-control-tower` + `scenario-auditor`
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)
> **Plan of record**: `docs/plans/unified-search-hub-plan.md` (this scenario is its implementation)

## 🎯 Overview
- **Purpose**: Add the project's **federated AI search router** — one query surface over every registered knowledge corpus, with automatic routing and unified ranking, extensible to any source (project-internal today; web / papers / personal inventory via the identical contract later). Turns "which of N searches do I call?" into "just search."
- **Primary users/verticals**: Every agent and scenario that needs to discover something (commands, components, records, docs, initiatives, code, …). It is interface-enabling infrastructure, not a standalone vertical.
- **Deployment surfaces**: Connect + CLI (`query`, `providers register|list|remove`, `status`) and a direct search UI with type facets, expand-search, provenance, and per-provider freshness/health.
- **Value promise**: Eliminates the meta-discovery problem (knowing which tool holds the answer). Federation makes search *reachable* across corpora; the metrics surface lets us *measure* whether discovery actually works and where it is under-used.

## 🏛️ Architecture invariants (non-negotiable, guarded by tests)
- **Thin router.** It owns only registry · classifier · fan-out · reranker · metrics. It stores **no vectors and no corpus content** — only the provider registry + query telemetry (SQLite). Architectural test asserts: no qdrant import, no corpus-content tables.
- **No conditional monolith.** Zero provider-specific code in the router. A new provider = one declarative registry row (descriptor + `ResultMapping`), no router source change.
- **Non-destructive federation.** Registering a corpus never removes, replaces, or degrades that scenario's own in-scenario search (per-type or group-unified). Two layers coexist permanently.
- **Retrieval lives in providers / `packages/ai-go/search`, not here.** The router federates providers' existing search endpoints as-is; it never indexes on a provider's behalf.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Provider registry + self-registration | `RegisterProvider`/`ListProviders`/`DeregisterProvider` (Connect + CLI) persist provider descriptors in SQLite; a new provider is added by one declarative row.
- [ ] OT-P0-002 | Explicit-type federation | `query "<text>" --type a,b` (or `--all`) fans out to matching registered providers with bounded concurrency, per-provider timeout, and partial results; returns provenance-tagged hits grouped by provider.
- [ ] OT-P0-003 | Graceful degradation | A down/stale provider is skipped with a surfaced warning and never fails the whole query; partial results return within timeout.
- [ ] OT-P0-004 | Operator-friendly output | CLI names the corpora searched, per-corpus counts, expansion hints (`--all`/`--type`/`--limit`), and provenance on every hit; `--json` shape is stable for scripting.
- [ ] OT-P0-005 | Routing accuracy (make-or-break) | Against labeled `testdata/routing_queries.json`, automatic routing achieves **recall ≥ 0.85** (uncertain ⇒ widen, not narrow); precision reported. `--type`/`--all` always override.
- [ ] OT-P0-006 | Thin-router boundary held | Architectural tests prove no qdrant import / no corpus tables, and that adding a fixture provider requires only a registry row (no router source change).

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Unified cross-provider ranking | Cross-encoder/LLM reranker fuses heterogeneous candidates into one comparable ranked list; rerank-on vs rerank-off MRR reported to justify its cost. Falls back to by-provider grouping + `degraded` flag when the reranker is unavailable.
- [ ] OT-P1-002 | Measurement backbone | Per-query telemetry (classified types, providers hit, counts, latency p50/p95, zero-result rate, re-query/"again" count); insights surface flags under-utilized providers (registered but never routed-to) and zero-result queries.
- [x] OT-P1-003 | All live providers federated | cli-health.commands, ui-health.surfaces/.widgets, swarm-manager.records/.backlog/.initiative, prompt-manager.skill/.action, knowledge-observatory.docs (when its cutover lands) registered.
- [ ] OT-P1-004 | Gap corpora tracked | Every corpus with no search yet (scenarios, resources, code, contracts, runs, git-provenance, requirements, config, domain-map, metrics) is a `CAPABILITY_GAP` registry stub visible in `providers list`/`status` as the live Track-A checklist.
- [x] OT-P1-005 | Search UI | Query box, bucket/type facets, expand-search, per-result provenance + provider freshness, loading/error/empty states.

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | External-scope providers | The descriptor carries `scope=EXTERNAL` from day one; register paid/external corpora (papers, web) through the identical contract.
- [ ] OT-P2-002 | `--group <scenario>` scoping | Reproduce a scenario's group-unified search *through* the hub without the scenario losing its own.
- [ ] OT-P2-003 | Federation-coverage metric | Track registered live leaves ÷ total known corpora toward 100% (the Track-A adoption gauge, §9 of the plan).

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: Go API (Connect RPC + mux fallback), React + Vite + Tailwind UI, Go CLI.
- Data + storage expectations: **SQLite** (modernc.org/sqlite) for registry + metrics (no qdrant — the router holds no vectors). SQLite is the permanent, intentional choice: it keeps the router dependency-free from external database services and suits a federation-router whose data (registry rows + query telemetry) is local and append-friendly.
- Integration strategy: federate providers over **HTTP + JSON only** (Connect RPCs are reachable via `POST {service}/{method}` with `application/json`; plain REST likewise) plus a CLI fallback. The router links **zero** provider Go clients; cross-scenario base URLs resolve at call-time via the backend resolver (never client-computed).
- Models: classifier `qwen3:1.7b`, reranker = LLM-as-reranker via `qwen3:4b` (both already pulled), each behind a swappable interface — drop in a true cross-encoder later without changing the contract.
- Non-goals / guardrails: does NOT index/cache corpus content, does NOT create `packages/ai-go/search` (the KO cutover plan does), does NOT build the gap providers themselves (their home scenarios do), does NOT remove or alter any scenario's existing search.

## 🤝 Dependencies & Launch Plan
- Required resources: `ollama` (classifier + reranker). Storage is SQLite (embedded, no external database service required).
- Scenario dependencies (soft, degrade-if-absent): `cli-health`, `ui-health`, `swarm-manager`, `knowledge-observatory`, `prompt-manager`.
- Operational risks: classifier mis-routing (mitigate: widen-on-uncertainty + measured recall gate); fan-out latency (bounded concurrency + per-provider timeout + partial results); data accretion (architectural test).
- Launch sequencing: registry → explicit-type fan-out (useful from here) → classifier → rerank → metrics → register all live + stub gaps → UI/validation. Functional and useful from explicit-type federation onward.

## 🎨 UX & Branding
- Look & feel: vrooli-default design kit (react-vite-tailwind adapter); light/dark via design tokens.
- Accessibility: WCAG AA; keyboard-navigable facets and results; clear empty/error/degraded states.
- Voice & messaging: utilitarian and honest — always show what was searched, what was skipped and why, and how to expand.
- Branding hooks: standard vrooli AppShell + tokens.

## 📎 Appendix
- Plan: `docs/plans/unified-search-hub-plan.md` (Appendix A = the locked contract: endpoint survey, result-mapping spec, proto draft, per-provider adapter rows, gap stubs).
- Taxonomy: `docs/reference/ai-search-routing.md` (corpus→intent→provider map this scenario operationalizes).
- Companion: `docs/plans/knowledge-observatory-search-cutover-plan.md` (creates `packages/ai-go/search`, the shared retrieval library; Track B).
