# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md`
> **Validation**: Enforced by `prd-control-tower` + `scenario-auditor`
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview
- **Purpose**: Read-only kiosk-style aggregator that composes dashboard payloads from Swarm Manager, Vrooli Core, and LPBS into six themed always-on displays.
- **Primary users/verticals**: Vrooli operators monitoring system health on a TV / Xbox browser; ambient awareness of agent throughput, scenario state, revenue, and broadcast analytics.
- **Deployment surfaces**: Go API, React + Vite UI, scenario CLI (`command-center`).
- **Value promise**: Single ambient surface that surfaces gap/partial/live status across all three upstream subsystems without ever mutating their state.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [x] OT-P0-001 | Dashboard aggregation API | Compose per-dashboard payloads from gap registry + per-source TTL cache + 3 upstream clients (CC-AGG-001..004).
- [x] OT-P0-002 | Mission Control vertical slice | Ground Control theme, real R3F starfield + satellites scene, live `/api/v1/dashboards/mission-control` fetch with gap/stale badges (CC-MC-001..003).

### 🟠 P1 – Should have post-launch
- [x] OT-P1-001 | UI shell for six themed routes | Router, DashboardLayout + ThemeProvider + lazy SceneCanvas, GapBadge + StaleBadge primitives, adapted but not auto-invoked kiosk hooks (CC-UI-001..004).

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Theming engine completion | Replace placeholder cube scenes + populate the five non-Mission-Control themes (delivered by `command-center-theming-engine`).
- [ ] OT-P2-002 | Kiosk UX policy | Auto-cycle, fade transitions, hidden settings panel, D-pad input, fullscreen-on-load (delivered by `command-center-kiosk-ux`).
- [ ] OT-P2-003 | LPBS dashboard endpoints | Production LPBS `/api/v1/admin/dashboard/*` so the LPBS upstream client can stop returning `ErrNotAvailable` (delivered by `lpbs-command-center-dashboard-endpoints`).

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: Go API (gorilla/mux + api-core/server), React 18 + Vite + TypeScript UI, React Three Fiber + drei for the 3D scenes, Tailwind for theming, react-router-dom + @tanstack/react-query for routing/data, scenario CLI on cli-core.
- Data + storage expectations: Filesystem-only (gap registry JSON loaded once at startup); per-source in-memory TTL cache for upstream payloads; in-memory ring buffer for R3F debug stats. No database.
- Integration strategy: Read-only HTTP clients against Swarm Manager, Vrooli Core, and LPBS; resolve upstream URLs via `${UPSTREAM}_BASE_URL` env vars assigned by the lifecycle manager. Never mutate upstreams.
- Non-goals / guardrails: No write paths to upstream services; no per-user auth (local-only debug surface); no metric-bound visuals on Mission Control; no replacement of the gap registry as the source of truth for `live`/`partial`/`gap`.

## 🤝 Dependencies & Launch Plan
- Required resources: None (filesystem + in-memory only).
- Scenario dependencies (optional `try_start`): `swarm-manager`, `landing-page-business-suite`. Vrooli Core read at `http://localhost:8092/scenarios` by default.
- Operational risks: Upstream availability can degrade dashboard freshness — surfaced via `staleness_ts` and gap-mode fallback. Ground Control scene uses GPU; constrained on low-power kiosks.
- Launch sequencing: Scaffold (this scenario) → theming-engine populates 5 stub themes → kiosk-ux wires hooks/auto-cycle → LPBS endpoints unlock LPBS-backed metrics.

## 🎨 UX & Branding
- Look & feel: Six dashboards with distinct themes (Ground Control / Bioluminescent / Foundry / Vault / Signal Tower / Cosmos). Mission Control is the reference electric-blue space aesthetic; the others ship as palette-only stubs filled in by the theming-engine sibling.
- Accessibility: Focus-visible rings on interactive elements, large legible typography for TV viewing distance, semantic landmarks on each layout.
- Voice & messaging: Sparse, glanceable copy; gap/stale state messaging surfaces "what is needed" rather than raw error text.
- Branding hooks: Theme tokens via CSS custom properties (`--cc-bg`, `--cc-accent`, …) so future themes plug in without component edits.

## 📎 Appendix
- Architecture deep-dive: `docs/ARCHITECTURE.md` (cache, registry, theming seams, Mission Control reference slice).
- Sibling execute items extending this scaffold: `command-center-theming-engine`, `command-center-kiosk-ux`, `lpbs-command-center-dashboard-endpoints`.
- Test commands for current targets: `vrooli scenario test command-center` runs structure + standards + lint + unit (Go + Vitest) + playbooks (BAS) + ui-smoke. Requirements live under `requirements/0{1,2,3}-*/module.json`.
