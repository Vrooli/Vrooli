# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview
- **Purpose**: Central command center for managing the Vrooli scenario ecosystem - orchestrating backlog work, scenario lifecycle, execution control, and self-improvement insights
- **Primary users/verticals**: Vrooli operators, agents, and developers managing the scenario ecosystem
- **Deployment surfaces**: CLI, API, UI (React + Vite)
- **Value promise**: Single interface to manage scenarios, backlog pipelines, and governed execution of autonomous change work

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Backlog file structure | Git-tracked folder-per-item in scenarios/swarm-manager/{ideas,research,fix,execute}/
- [ ] OT-P0-002 | Backlog CRUD | Create, read, update, delete backlog items via API and CLI
- [ ] OT-P0-003 | Backlog details page | File tree view, drag-and-drop upload, preview for markdown/code/images
- [ ] OT-P0-004 | Backlog queue for processing | Queue idea backlog items for Swarm Manager planning and implementation workflows
- [ ] OT-P0-005 | Scenario catalog with priority | List all scenarios with priority ranking, search, and filter
- [ ] OT-P0-006 | Scenario metadata management | Greenfield/brownfield toggle
- [ ] OT-P0-007 | Scenario deletion with safeguards | Strong confirmation dialog + archive-to-backlog option
- [ ] OT-P0-008 | Tabbed navigation UI | Header tabs (desktop) / bottom-nav (mobile) with five tabs: Backlog, Scenarios, Execution, Prompts, Settings
- [ ] OT-P0-009 | agent-manager integration | Spawn agents for all automated work through agent-manager

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Execution control policy | Manual/scheduled/yolo defaults with configurable delay
- [ ] OT-P1-002 | Execution operations page | View pending/running/completed/failed and govern runs
- [ ] OT-P1-003 | Insights engine | Self-improvement suggestions based on system patterns
- [ ] OT-P1-004 | Research agent modal | Spawn research agents from backlog details page (Idea Agent: clarify/suggest/enhance workflow)
- [ ] OT-P1-005 | visited-tracker integration | Campaign management for context cleanup
- [ ] OT-P1-006 | knowledge-observatory integration | View and prune PROBLEMS.md files
- [ ] OT-P1-007 | scenario-completeness-scoring integration | Display completeness scores
- [ ] OT-P1-008 | app-issue-tracker integration | Open and track issues against scenarios
- [ ] OT-P1-009 | test-genie integration | Run tests and display results
- [ ] OT-P1-010 | Settings modal | Theme, execution policy config, insights config
- [ ] OT-P1-011 | Verified evidence ledger | Link verified Agent Manager work to Sessions and operating-mode executions with idempotent reconciliation and watermark-aware evidence gates

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Advanced cost formulas | Sophisticated priority calculations based on value/effort
- [ ] OT-P2-002 | Pattern recognition | Heuristic similarity and pattern detection (filesystem-only)
- [ ] OT-P2-003 | Analytics dashboard | Usage metrics, agent performance, scenario health trends
- [ ] OT-P2-004 | Batch operations | Bulk actions on backlog items and scenarios
- [ ] OT-P2-005 | Webhooks | External integrations for notifications and triggers

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: Go API (api-core/server with gorilla/mux), React UI (Vite, TypeScript, React Query, Zustand), Go CLI (cli-core with urfave/cli)
- Data + storage expectations: Filesystem only (git-tracked backlog folders, `.vrooli/settings.json`, `.vrooli/queue.json`)
- Integration strategy: All agent work via agent-manager; Swarm Manager owns backlog-to-plan-to-execution orchestration; prompt resolution flows through prompt-manager and validation through test-genie.
- Non-goals / guardrails: No kanban/Trello UI, no direct agent spawning, no embedded scenario implementation code, no complex workflow builders, no multi-user auth

## 🤝 Dependencies & Launch Plan
- Required resources: None (filesystem-only persistence)
- Scenario dependencies (required): agent-manager, prompt-manager
- Scenario dependencies (optional P1): knowledge-observatory, visited-tracker, scenario-completeness-scoring, app-issue-tracker, test-genie
- Operational risks: Agent-manager API stability; filesystem integrity for settings, queue, and execution policy
- Launch sequencing: P0 core CRUD and UI → P0 integrations → P1 execution control policy → P1 integrations → P2 analytics

## 🎨 UX & Branding
- Look & feel: Dark theme default, responsive mobile-first, clean minimal interface
- Tab structure: Backlog (icon: lightbulb), Scenarios (icon: package), Execution (icon: zap), Prompts (icon: scroll-text), Settings (icon: settings)
- Accessibility: WCAG AA compliance, keyboard navigation, screen reader support
- Voice & messaging: Technical but approachable, focused on efficiency and clarity
- Branding hooks: Consistent with Vrooli design system, scenario status indicators with color coding

## 📎 Appendix
- Additional implemented operator surfaces beyond the core launch targets include prompts management, backlog conversion, prompt tracing, scenario lifecycle controls, and spec-sync-archive orchestration.
- Backlog items live under `scenarios/swarm-manager/{ideas,research,fix,execute}/` as git-tracked folders with a required `spec.json` and optional supporting files.
- Settings can draw recommendation context from sources such as `PROBLEMS.md`, completeness scores, test phase results, coverage data, and operator-supplied focus text.
