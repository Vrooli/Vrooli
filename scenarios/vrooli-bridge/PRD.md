# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md`
> **Validation**: Enforced by `prd-control-tower` + `scenario-auditor`
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview
- **Purpose**: Vrooli Bridge enables external projects - no matter what machine they are located on - to become Vrooli-aware consumers, allowing them to leverage Vrooli's full suite of intelligent scenarios and capabilities through automated documentation injection and integration tracking
- **Primary users/verticals**: Developers using Claude Code on multiple projects, teams managing polyglot codebases, organizations integrating Vrooli into existing development workflows
- **Deployment surfaces**: CLI (project scanning and integration), API (project registry and health checks), UI (integration dashboard with status tracking)
- **Value promise**: Multiplies Vrooli's reach to entire development ecosystem, enabling every project to benefit from Vrooli's intelligence while creating a bidirectional learning loop where agents learn from diverse codebases

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Project discovery system | Scan filesystem to discover projects (git repos, package.json, Cargo.toml, etc.)
- [ ] OT-P0-002 | Documentation generator | Generate Vrooli integration documentation for external projects
- [ ] OT-P0-003 | CLAUDE.md injection | Update/create CLAUDE.md files to reference Vrooli documentation
- [ ] OT-P0-004 | Integration tracking | Track integration status and last update timestamps
- [ ] OT-P0-005 | CLI interface | Provide CLI interface for manual project registration
- [ ] OT-P0-006 | Project registry | Store project registry in PostgreSQL for persistence
- [ ] OT-P0-007 | Remote server connection | Securely connect to remote servers. Require sign-in only the first time

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Project type detection | Auto-detect project type and recommend relevant scenarios
- [ ] OT-P1-002 | Version tracking | Version tracking for Vrooli documentation updates
- [ ] OT-P1-003 | Bulk operations | Bulk operations for updating multiple projects
- [ ] OT-P1-004 | Health checks | Integration health checks (verify docs are still present)
- [ ] OT-P1-005 | Project categorization | Project categorization and tagging system

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Auto-update on new scenarios | Auto-update projects when new Vrooli scenarios are added
- [ ] OT-P2-002 | Dependency visualization | Project dependency graph visualization
- [ ] OT-P2-003 | Custom templates | Custom documentation templates per project type
- [ ] OT-P2-004 | Analytics | Integration analytics and usage tracking

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: Go API (project scanner and registry), React UI (dashboard with dark theme), PostgreSQL (project metadata), Redis (scan result caching)
- Data + storage expectations: PostgreSQL (project registry and integration history), Redis (optional cache for scan results), filesystem access (read/write to project directories)
- Integration strategy: File-based documentation injection (VROOLI_INTEGRATION.md, CLAUDE.md updates), CLI-first approach for manual registration, API for UI and automation integration
- Non-goals / guardrails: No code modification beyond documentation files, no system-wide daemon required, respects filesystem permissions, read-only mode for projects without write access

## 🤝 Dependencies & Launch Plan
- Required resources: postgres (project registry and metadata storage), filesystem access (read/write to project directories), git (detect repository boundaries)
- Scenario dependencies: None required initially, enables cross-project-refactor, dependency-orchestrator, project-health-monitor in future
- Operational risks: Filesystem permissions may block integration, large repositories slow scanning, outdated integrations if projects move/delete
- Launch sequencing: Phase 1 - Core discovery and integration (2 weeks), Phase 2 - Health checks and bulk operations (1 week), Phase 3 - Advanced features and analytics (ongoing)

## 🎨 UX & Branding
- Look & feel: Professional developer-focused aesthetic inspired by GitHub Desktop and VS Code Extensions, dark theme with accent colors per project type, modern monospace typography for file paths, dashboard layout with sidebar navigation
- Accessibility: WCAG AA compliance, keyboard navigation for all operations, high contrast mode support, clear status indicators
- Voice & messaging: Helpful and informative, focused and efficient, technical but approachable - "I have full control over my integrations"
- Branding hooks: Project type badges with color coding, integration status indicators (✓ active, ⚠ outdated, ✗ missing), clean modern interface emphasizing transparency and control

## 📎 Appendix

Supplementary context including intelligence amplification, recursive value, and performance targets are documented in README.md.
