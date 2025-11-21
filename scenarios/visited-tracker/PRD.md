# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md`
> **Validation**: Enforced by `prd-control-tower` + `scenario-auditor`
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview
- **Purpose**: Persistent file visit tracking with staleness detection for systematic code analysis across large codebases
- **Primary users/verticals**: Maintenance scenarios (api-manager, test-genie), code quality scenarios, documentation scenarios, security audit scenarios, Claude Code agents performing systematic analysis
- **Deployment surfaces**: CLI (programmatic integration), API (web interface and external integrations), UI (manual campaign management)
- **Value promise**: Enables maintenance scenarios to efficiently track analyzed files, ensuring comprehensive coverage without redundant work, with staleness scoring to prioritize least-visited and most-modified files

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Campaign-based file tracking | Campaign-based file tracking with patterns (*.go, *.js, etc.)
- [ ] OT-P0-002 | Visit count tracking | Visit count tracking for each file in a campaign
- [ ] OT-P0-003 | Staleness scoring | Staleness scoring based on visit frequency and modification time
- [ ] OT-P0-004 | CLI interface | CLI interface for programmatic integration
- [ ] OT-P0-005 | JSON persistence | JSON file storage for persistence and portability

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | HTTP API | HTTP API for web interface and external integrations
- [ ] OT-P1-002 | Web interface | Web interface for manual campaign management
- [ ] OT-P1-003 | File synchronization | File synchronization with glob pattern matching
- [ ] OT-P1-004 | Prioritization | Least visited and most stale file prioritization
- [ ] OT-P1-005 | Campaign export/import | Campaign export/import capabilities

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Advanced analytics | Advanced analytics and staleness trend analysis
- [ ] OT-P2-002 | Git history integration | Integration with git history for enhanced staleness detection
- [ ] OT-P2-003 | Multi-project management | Multi-project campaign management
- [ ] OT-P2-004 | Automated discovery | Automated file discovery and pattern suggestions

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
