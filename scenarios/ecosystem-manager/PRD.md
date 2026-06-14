# Ecosystem Manager — Product Requirements Document

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md`
> **Validation**: Enforced by `prd-control-tower` + `scenario-auditor`

## 🎯 Overview
- **Purpose**: Provide the unified control plane that generates and improves Vrooli resources and scenarios — the self-improvement kernel that turns open findings into agent-driven work and learns which interventions pay off. It also serves as the reference scenario whose own migration (R0/R1 conformance + Connect-RPC) other old-template scenarios copy.
- **Primary users/verticals**: Vrooli platform engineers, the autosteer control loop itself (programmatic consumer), and operators monitoring ecosystem development across the fleet.
- **Deployment surfaces**: API (Go, transitioning gorilla/mux REST → Connect-RPC), CLI (`ecosystem-manager` via cli-core), UI (React + Vite kanban + insights), and the WebSocket live-update channel.
- **Value promise**: Consolidates four legacy tools (resource/scenario × generator/improver) into one intelligent platform; drives a measurable improvement loop (effectiveness-weighted skill selection, maturity ladder) instead of ad-hoc tool-switching.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [x] OT-P0-001 | Unified task queue | Create/track all four operation types from one surface with real-time WebSocket updates.
- [x] OT-P0-002 | Closed-loop autosteer | Diagnose findings → select skill → execute via agent-manager → re-measure → terminate on objective/diminishing-returns/budget.
- [x] OT-P0-003 | Settings control | Persisted slots/cooldown/agent settings governing queue concurrency, with validated partial updates and side-effects on the processor.
- [ ] OT-P0-004 | Health + recovery | `/health` on the root router, stale-task recovery on startup, automatic WebSocket reconnection.
- [ ] OT-P0-005 | Template conformance | EM itself passes the react-vite template standards/docs gates so it can be the migration reference for the rest of the fleet.
- [ ] OT-P0-006 | Shadow-safe improvement | Autosteer engagements run against a baseline shadow (start/promote/abandon via git-control-tower), routing agent runs **and measurement** to the shadow so edits and audits never leak to the live instance.

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Connect-RPC transport | Migrate domains off hand-rolled REST/Zod onto proto-first Connect-RPC, one domain at a time (discovery is the reference).
- [x] OT-P1-002 | Cross-type intelligence | Dependency analysis flags affected scenarios/resources and feeds smart prioritization.
- [x] OT-P1-003 | Effectiveness transparency | Per-(skill, dimension) ledger + decision trace exposed via API/CLI/UI as a glass box.
- [x] OT-P1-004 | Temporal-flow verification | Model the queue/task state machine with `flow-verifier` (Quint) and gate transitions.
- [x] OT-P1-005 | Toolchain trust/cost priors | development-toolchain-validator fitness snapshots seed selection priors and a Layer-1 eligibility gate vetoes red skills, failing open when DTV is degraded.
- [ ] OT-P1-006 | Fleet autosteer | Point EM's loop at other old-template scenarios and have it steer them to maturity unattended, with importance-aware scheduling across the fleet.

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Advanced analytics | Reporting dashboard, task templates/automation, external-tool integration.

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: Go API (`api-core`/`cli-core`/`connectrpc.com/connect`), React + Vite + Tailwind UI, proto-first contracts under `packages/proto`.
- Data + storage expectations: all runtime state resolves through `api-core/storage` classes under `<data-root>/vrooli/<namespace>/` — filesystem-backed queue (`queue/{pending,...}` under `ClassData`), `settings.json` under `ClassConfig`, and an embedded SQLite DB (`ClassData`) for findings/effectiveness/history. No Postgres.
- Integration strategy: proto-typed Connect-RPC clients consumed by UI + CLI; agent-manager for agent execution; shared workflows > resource CLI > direct API.
- Non-goals / guardrails: no big-bang multi-domain transport migration (one reference domain per plan); no compatibility shims within a migrated domain; no disabling lint/security rules to pass gates.

## 🤝 Dependencies & Launch Plan
- Required resources: `agent-manager` (agent execution), `scenario-auditor` (standards), `test-genie` (finding producers), `flow-verifier` (temporal verification), `git-control-tower` (regression baselines).
- Scenario dependencies: consumes the broader Vrooli fleet as both targets (scenarios it improves) and producers (test-genie phases feeding the ladder).
- Operational risks: external-agent availability (Claude Code), transport-migration state-tracking drift, pre-existing UI type-safety backlog under strict lint.
- Launch sequencing: (1) R0/R1 + standards/docs conformance; (2) Connect-RPC foundation; (3) discovery reference domain (settings proved too entangled to lead); (4) temporal-flow reference; (5) incremental per-domain migration of the remaining seven domains.

## 🎨 UX & Branding
- Look & feel: dense operator dashboard — kanban task board (Pending/In Progress/Review/Completed/Failed), insights panels, light/dark/system theme.
- Accessibility: focus-visible styling on interactive elements, keyboard navigation via the iframe-bridge spatial nav, WCAG-aligned color contrast.
- Voice & messaging: precise, engineering-operator tone; status is color-coded and quantitative (counts, timers, token costs).
- Branding hooks: consistent Vrooli iconography (lucide-react), monospace for IDs/paths, the kanban board as the signature surface.

---

## Detailed Requirements

### Executive Summary
The Ecosystem Manager consolidates four separate management tools (resource-generator, resource-improver, scenario-generator, scenario-improver) into a single platform with cross-type intelligence and a closed-loop improvement controller.

### Problem Statement
- **Fragmented Management**: four overlapping tools with siloed operations.
- **Limited Visibility**: no unified view of ecosystem development.
- **Resource Conflicts**: port allocation / process coordination issues.
- **Context Loss**: no cross-type dependency awareness.
- **Maintenance Overhead**: 4× deployment/monitoring/maintenance complexity.

### Epics & User Stories (acceptance criteria)

#### Epic 1: Unified Task Management
- **Unified task creation** — single modal supports all four operation types; form adapts; validation prevents invalid configs; tasks queue immediately.
- **Kanban visualization** — five columns, drag-and-drop, real-time WebSocket updates, filterable cards.
- **Advanced filtering** — text search + status/type/operation/category/priority filters (AND logic), session-persistent.

#### Epic 2: Intelligent Queue Processing
- **Smart queue coordination** — concurrent slot management, port coordination, live process monitoring, pause/resume, settings-controlled behavior.
- **Cross-type intelligence** — dependency flagging, impact analysis, smart prioritization, duplicate prevention.
- **Real-time process monitoring** — live counts, execution timers, termination controls, log viewing.

#### Epic 3: Enhanced Developer Experience
- **Comprehensive task details** — full info modal, phase progress, error/result viewing, edit for pending.
- **Smart prompt assembly** — assembled-prompt viewing with metadata, copy, cache status.
- **Natural CLI integration** — natural commands, dynamic port discovery, color-coded output, bulk ops.

#### Epic 4: System Reliability & Performance
- **Robust settings management** — persisted across restarts, backend API with localStorage fallback, theme support, validation/defaults, reset.
- **Health monitoring & recovery** — health endpoints, WebSocket reconnect, stale-task recovery, orphan cleanup, retry logic.

### Architecture & Non-functional Standards
- **API versioning**: endpoints prefixed `/api/v1/` (REST remainder); Connect-RPC procedure paths for migrated domains.
- **Performance**: API < 200ms p95; UI list updates < 100ms up to 1000 tasks; WebSocket latency < 50ms.
- **Reliability**: 99.5% availability; zero data loss in normal operation; graceful degradation; orphan cleanup.
- **Security**: input validation, XSS prevention, CORS restrictions, process isolation, no secrets in logs/client.
- **Scalability**: 10,000+ tasks; up to 10 concurrent agent executions; linear memory growth.

### Auto-Steer Controller (v1, effectiveness-weighted)
Drives a target scenario toward an objective profile: diagnose open findings → select the skill most likely to close the heaviest dimension → execute via agent-manager → re-measure → terminate on objective-met / diminishing returns / runtime thrashing / budget cap. Conceptual spec: [`docs/concepts/CONTROL-MODEL.md`](docs/concepts/CONTROL-MODEL.md); targets tracked in [`requirements/index.json`](requirements/index.json). Includes token-cost capture, a per-(skill, dimension) effectiveness ledger, credit assignment, effectiveness-weighted selection (shrinkage prior + epsilon-greedy), Layer-2 thrashing defense, and a decision-trace glass box. Development-toolchain-validator trust/cost priors and the Layer-1 eligibility gate are wired (fail open on DTV degradation).

### Risk Assessment
- **High** — external AI-service dependency (mitigation: robust error handling, fallback, monitoring).
- **Medium** — 4-tools→1 migration edge cases (mitigation: comprehensive testing, gradual migration, rollback).
- **Low** — UI performance on large datasets (mitigation: virtualization, pagination, monitoring).

---

**Document Version**: 2.1
**Last Updated**: 2026-06-12
**Owner**: Ecosystem Manager Development Team
