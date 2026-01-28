# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md`
> **Validation**: Enforced by `prd-control-tower` + `scenario-auditor`
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview
- **Purpose**: Central command center for managing the Vrooli scenario ecosystem - orchestrating idea backlogs, scenario lifecycle, autonomous recommendations, and self-improvement insights
- **Primary users/verticals**: Vrooli operators, agents, and developers managing the scenario ecosystem
- **Deployment surfaces**: CLI, API, UI (React + Vite)
- **Value promise**: Single interface to manage what scenarios exist, what ideas are in the pipeline, and how the system prioritizes autonomous improvement efforts

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [x] OT-P0-001 | Idea backlog file structure | Git-tracked folder-per-idea with spec.json + context files in scenarios/swarm-manager/ideas/
- [x] OT-P0-002 | Idea backlog CRUD | Create, read, update, delete ideas via API and CLI
- [x] OT-P0-003 | Idea details page | File tree view, drag-and-drop upload, preview for markdown/code/images
- [x] OT-P0-004 | Idea queue for processing | Queue ideas for initialization/implementation via ecosystem-manager
- [x] OT-P0-005 | Scenario catalog with priority | List all scenarios with priority ranking, search, and filter
- [x] OT-P0-006 | Scenario metadata management | Greenfield/brownfield toggle, enable/disable from recommendations
- [x] OT-P0-007 | Scenario deletion with safeguards | Strong confirmation dialog + archive-to-backlog option
- [x] OT-P0-008 | Tabbed navigation UI | Header tabs (desktop) / bottom-nav (mobile) with four tabs: Ideas, Scenarios, Recommendations, Settings
- [ ] OT-P0-009 | agent-manager integration | Spawn agents for all automated work through agent-manager
- [x] OT-P0-010 | ecosystem-manager integration | Initialize and improve scenarios from ideas via ecosystem-manager

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Recommendation engine (3-state) | Off/suggestions/yolo modes with configurable data sources
- [ ] OT-P1-002 | Recommendation management page | View pending recommendations, approve or reject
- [ ] OT-P1-003 | Insights engine | Self-improvement suggestions based on system patterns
- [ ] OT-P1-004 | Research agent modal | Spawn research agents from idea details page
- [ ] OT-P1-005 | visited-tracker integration | Campaign management for context cleanup
- [ ] OT-P1-006 | knowledge-observatory integration | View and prune PROBLEMS.md files
- [ ] OT-P1-007 | scenario-completeness-scoring integration | Display completeness scores
- [ ] OT-P1-008 | app-issue-tracker integration | Open and track issues against scenarios
- [ ] OT-P1-009 | test-genie integration | Run tests and display results
- [ ] OT-P1-010 | Settings modal | Theme, recommendation engine config, insights config

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Advanced cost formulas | Sophisticated priority calculations based on value/effort
- [ ] OT-P2-002 | Pattern recognition | Heuristic similarity and pattern detection (filesystem-only)
- [ ] OT-P2-003 | Analytics dashboard | Usage metrics, agent performance, scenario health trends
- [ ] OT-P2-004 | Batch operations | Bulk actions on ideas and scenarios
- [ ] OT-P2-005 | Webhooks | External integrations for notifications and triggers

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: Go API (Gin), React UI (Vite, TypeScript), Go CLI (Cobra)
- Data + storage expectations: Filesystem only (git-tracked idea folders, `.vrooli/settings.json`, `.vrooli/queue.json`)
- Integration strategy: All agent work via agent-manager, all scenario ops via ecosystem-manager, never direct calls
- Non-goals / guardrails: No kanban/Trello UI, no direct agent spawning, no embedded scenario implementation code, no complex workflow builders, no multi-user auth

## 🤝 Dependencies & Launch Plan
- Required resources: None (filesystem-only persistence)
- Scenario dependencies (required): agent-manager, ecosystem-manager
- Scenario dependencies (optional P1): knowledge-observatory, visited-tracker, scenario-completeness-scoring, app-issue-tracker, test-genie, prompt-manager
- Operational risks: Tight coupling with ecosystem-manager API stability; filesystem integrity for settings/queue; recommendation engine complexity
- Launch sequencing: P0 core CRUD and UI → P0 integrations → P1 recommendation engine → P1 integrations → P2 analytics

## 🎨 UX & Branding
- Look & feel: Dark theme default, responsive mobile-first, clean minimal interface
- Tab structure: Ideas (icon: lightbulb), Scenarios (icon: package), Recommendations (icon: zap), Settings (icon: gear)
- Accessibility: WCAG AA compliance, keyboard navigation, screen reader support
- Voice & messaging: Technical but approachable, focused on efficiency and clarity
- Branding hooks: Consistent with Vrooli design system, scenario status indicators with color coding

## 📎 Appendix

### Idea Backlog File Structure
Location: `scenarios/swarm-manager/ideas/` (git-tracked)
```
ideas/
├── my-scenario-idea/
│   ├── spec.json        # Required: name, title, description, status, priority
│   ├── notes.md         # Optional context
│   ├── mockup.png       # Optional visuals
│   └── research/        # Optional supporting files
```

### spec.json Schema
```json
{
  "name": "my-scenario-idea",
  "title": "My Scenario Idea",
  "description": "Brief description",
  "status": "backlog|researching|ready|queued|in_progress|completed|archived",
  "priority": 1,
  "tags": ["ai", "automation"],
  "created": "2026-01-28T00:00:00Z",
  "updated": "2026-01-28T00:00:00Z"
}
```

### Recommendation Engine Data Sources
Configurable checkboxes in Settings:
- PROBLEMS.md scanning
- Completeness scores
- Test phase results
- Test coverage percentages
- Custom focus text input
