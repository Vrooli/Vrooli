# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md`
> **Validation**: Enforced by `prd-control-tower` + `scenario-auditor`
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview
- **Purpose**: Persistent file visit tracking with staleness detection for systematic code analysis across large codebases, enabling agent loops to maintain perfect memory across conversations
- **Primary users/verticals**: Agent loops in ecosystem-manager (Progress, UX, Refactor, Test phases), maintenance scenarios, code quality automation, Claude Code agents performing systematic multi-file work across conversations
- **Deployment surfaces**: CLI (programmatic integration for agent loops), API (web interface and external integrations), UI (manual campaign management)
- **Value promise**: Enables agent loops to maintain perfect memory across conversations, ensuring comprehensive coverage without redundant work, with phase-specific metadata storage for handoff between analysis modes (UX → Refactor → Test) and staleness scoring to prioritize neglected files

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Campaign tracking system | Campaign-based file tracking with visit counts, staleness scoring, CLI interface, and JSON persistence
- [ ] OT-P0-002 | Zero-friction agent integration | Auto-creation shorthand with location + tag + glob pattern for seamless agent loop usage without manual campaign management
- [x] OT-P0-003 | Phase metadata and handoff context | Campaign-level and file-level notes for storing phase-specific metadata, work-in-progress tracking, and cross-phase handoff information
- [x] OT-P0-004 | Precise campaign control | Manual prioritization and exclusion controls for fine-tuning file coverage and handling exceptional cases
- [ ] OT-P0-005 | Clutter prevention and limits | Smart default exclusions (data/, tmp/, coverage/, dist/, build/) and configurable campaign size limits to maintain focused campaigns
- [x] OT-P0-006 | Smart campaign sync | Should gracefully handle new, renamed, and removed files in campaign target pattern, auto-syncing on campaign read (cached 30 seconds) and write commands to ensure campaign matches actual ground-truth files

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | HTTP API endpoints | HTTP API with CRUD operations, prioritization queries, and export/import capabilities
- [x] OT-P1-002 | Web interface | Web interface for manual campaign management and visualization

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Advanced analytics and scaling | Advanced analytics with staleness trend analysis and multi-project management

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: Go API (file tracking and staleness calculation), React UI (campaign management dashboard), CLI (agent integration)
- Data + storage expectations: File-based JSON storage for simplicity and portability, optional PostgreSQL for enhanced querying, optional Redis for caching
- Integration strategy: CLI-first for agent integration, HTTP API for web interface, file-based storage for transparency and manual intervention
- Non-goals / guardrails: Not an end-user application (internal developer tool), no complex database requirements (file-based is sufficient), no real-time collaboration features (single-agent focused)

## 🤝 Dependencies & Launch Plan
- Required resources: Local file system (for tracking file modifications and storing campaign data)
- Optional resources: postgres (for enhanced data storage), redis (for caching and performance optimization)
- Scenario dependencies: Used by api-manager, test-genie, and other maintenance scenarios
- Operational risks: Must handle large codebases (1000+ files) efficiently, must maintain state across multiple agent conversations
- Launch sequencing: Phase 1 - Deploy CLI and file-based storage (1 week), Phase 2 - Add HTTP API and web interface (2 weeks), Phase 3 - Integration with maintenance scenarios (ongoing)

## 🎨 UX & Branding
- Look & feel: Minimal developer-focused UI with dark theme, clean data tables, simple campaign management
- Accessibility: Keyboard navigation for all operations, high contrast for readability, screen reader support for campaign status
- Voice & messaging: Technical, systematic, focused on comprehensive coverage - "Never miss a file, never repeat work"
- Branding hooks: Staleness indicators (🔥 Critical staleness, ⚠️ High staleness, ✅ Recently visited)

## 📎 Appendix

Performance targets, staleness algorithm details, and integration patterns are maintained in the scenario's README.md and supporting documentation.
