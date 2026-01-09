# Product Requirements Document (PRD)

> Scenario: api-library
> Template: Canonical PRD v2.0
> Last Reviewed: 2025-09-27

## 🎯 Overview
- **Purpose**: Preserve institutional knowledge about third-party APIs so agents can discover, compare, and operationalize external capabilities without manual research.
- **Primary users**: Scenario builders who need vetted integration options, operators tracking API health/cost, and automation surfaces that query the library programmatically.
- **Deployment surfaces**: REST API powering other scenarios, CLI helpers for batch operations, and a React web console for browsing/searching the catalog.

## 🎯 Operational Targets
The checklists below define the measurable outcomes needed for each release tier. Status is driven by requirements sync rather than manual edits.

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | API metadata inventory | Store and retrieve API metadata including endpoints, pricing, rate limits, and auth methods.
- [ ] OT-P0-002 | Semantic capability search | Provide semantic search across API descriptions and capabilities for instant discovery.
- [ ] OT-P0-003 | Credential tracking | Track which APIs have credentials configured so operators know what is ready to use.
- [ ] OT-P0-004 | API insights workspace | Capture and display notes, gotchas, and integration learnings for every API.
- [ ] OT-P0-005 | Research-assistant integration | Pull newly discovered APIs from research-assistant to keep the library fresh.
- [ ] OT-P0-006 | Metadata history | Track creation/update timestamps and source URLs for every API record.
- [ ] OT-P0-007 | Scenario-facing REST API | Expose REST endpoints so other scenarios can query, filter, and mutate the catalog.
- [ ] OT-P0-008 | Operator web UI | Provide a web UI for browsing, searching, and managing APIs.

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Redis caching | Cache frequently accessed APIs to reduce latency for popular queries.
- [ ] OT-P1-002 | Pricing refresh | Automatically refresh pricing data from source URLs on a 24-hour cadence.
- [ ] OT-P1-003 | Cost calculator | Offer usage-based cost calculators that estimate spend per scenario.
- [ ] OT-P1-004 | Categorization and tagging | Maintain API categories/tags plus supporting CRUD endpoints.
- [ ] OT-P1-005 | Version tracking | Track version history, breaking changes, and migration metadata per API.
- [ ] OT-P1-006 | Integration recipes | Store integration snippets/recipes with community voting and metadata.
- [ ] OT-P1-007 | Status monitoring | Monitor deprecations, sunsets, and general API status through dedicated endpoints.
- [ ] OT-P1-008 | Export tools | Export filtered catalogs as JSON or CSV for auditing and downstream automation.

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Comparison matrix | Generate API comparison matrices for capability/customer fit analysis.
- [ ] OT-P2-002 | Usage analytics | Surface usage analytics and recommendations tied to scenario behavior.
- [ ] OT-P2-003 | Update webhooks | Emit webhooks when API data changes so dependent scenarios can react.
- [ ] OT-P2-004 | Codegen integrations | Provide opinionated code-generation endpoints for scaffolding integrations.
- [ ] OT-P2-005 | Health monitoring | Track uptime/performance signals for APIs and expose them via monitoring endpoints.

## 🧱 Tech Direction Snapshot
The platform favors boring, observable components so future agents can safely extend it without relearning novel stacks.
- Preferred data stores: Postgres for normalized metadata plus Qdrant (with Ollama embeddings) for semantic search, backed by Redis caches for hot paths.
- Preferred service stack: Go REST API with typed DTOs, shared middleware, and CLI proxies that reuse the same handlers for predictable contracts.
- Preferred UI runtime: React + Vite console focused on search-first workflows, optimistic updates, and operator shortcuts that mirror other Vrooli consoles.
- Observability: standard system-monitor hooks (health endpoints, metrics) plus structured audit logs for catalog mutations.
- Non-goals: building end-user automation workflows or duplicating integration code generators beyond the minimal snippets service.

## 🤝 Dependencies & Launch Plan
- Requires local Postgres, Qdrant, and Redis resources plus optional Ollama for embedding refreshes; all orchestrated through the scenario lifecycle manager (make start/test/logs/stop).
- Depends on research-assistant scenario for continuous intake plus ecosystem-manager for publishing catalog metadata to other apps.
- Launch gating: seed baseline API catalog, verify semantic search accuracy, and enable monitoring hooks before exposing REST endpoints to dependent scenarios.
- Operational readiness: integrate with system-monitor alerts and ensure CLI + API auth policies align with deployment tier expectations.

## 🎨 UX & Branding
- Visual tone mirrors Vrooli ops consoles: dark-neutral palette, monospace data columns, and clear severity badges for API health/cost insights.
- UX emphasizes search-first workflows with keyboard shortcuts, semantic filters, and inline metadata cards; avoid multi-step modal flows.
- Copy voice is pragmatic and instructional, surfacing actionable guidance ("Configure credentials" / "Watch deprecation") instead of marketing fluff.
- Accessibility targets WCAG AA: focus outlines, contrast-compliant badges, and screen-reader labels for every action icon.

## 📎 Appendix
- Historical progress journals (sessions dated 2025-09-24 through 2025-09-27) plus detailed technical artifacts now live in `docs/PROGRESS.md` and `docs/PROBLEMS.md` to keep this PRD minimal.
