# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview
- **Purpose**: Central code tidiness orchestrator that prevents scenarios from decaying into unmaintainable chaos through progressive, multi-tier scanning (cheap static + expensive AI) and campaign-based cleanup
- **Primary users/verticals**: Development agents, maintenance scenarios, human developers managing code health across 100+ scenarios
- **Deployment surfaces**: CLI (agent integration), API (programmatic access), UI (human management dashboard), auto-campaigns (background enforcement)
- **Value promise**: Surfaces refactor opportunities before they become emergencies; ensures comprehensive maintainability coverage without redundant static-quality ownership; prevents "1000-line file syndrome" across the ecosystem

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Maintainability scanning | Detect long files, technical debt markers, coupling, complexity, and duplication, then return normalized tidiness findings for agents and Test Genie
- [ ] OT-P0-002 | Code quality metrics | Detect languages (Go/TS/JS/Python/Rust), compute line counts, technical debt markers (TODO/FIXME/HACK), import/function metrics, cyclomatic complexity (via gocyclo), and code duplication (via dupl/jscpd) with graceful tool degradation
- [x] OT-P0-003 | Light scan performance | Complete light scans for typical scenarios in under 60-120 seconds or surface clear timeout status
- [ ] OT-P0-004 | AI batch scanning | Process files in batches using resource-claude-code/resource-codes with configurable limits
- [ ] OT-P0-005 | visited-tracker integration | Create/attach to campaigns and prioritize unvisited/least-visited files for smart scans
- [ ] OT-P0-006 | No file hammering | Prevent analyzing the same file twice within a session or beyond configurable max visits
- [ ] OT-P0-007 | Agent API | Expose HTTP/CLI interface for other agents to request top N tidiness issues by scenario/file/folder/category
- [ ] OT-P0-008 | Issue storage | Record AI-generated issues with scenario, file path, category, severity, agent notes, remediation steps, and campaign metadata
- [ ] OT-P0-009 | Global dashboard | Display per-scenario counts of light issues, AI issues, long files, visit %, and campaign status
- [ ] OT-P0-010 | Scenario detail view | Show file table with paths, line counts, issue counts, visit counts, and sortable columns

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Auto-tidiness campaigns | Run automatic agent scan campaigns across up to K scenarios with session limits and priority rules
- [ ] OT-P1-002 | Campaign lifecycle | Auto-complete campaigns when all files visited or max sessions reached; support pause/resume/terminate
- [ ] OT-P1-003 | Campaign safety | Mark campaigns as "error" on repeated failures; enforce global concurrency limit K
- [ ] OT-P1-004 | Issue management UI | Allow mark-as-resolved, mark-as-ignored, filter by status, view agent notes and suggested remediation
- [ ] OT-P1-005 | Issue de-duplication | Group/link same logical issue from multiple sources (lint + type + AI) to reduce clutter
- [ ] OT-P1-006 | Trigger controls | Enable one-off light scans, one-off smart scans, and campaign enable/disable from UI
- [ ] OT-P1-007 | Scan history | Track which resource was used, when issues were created, which campaign/session produced them
- [ ] OT-P1-008 | Configurable thresholds | Centrally configure long file lines, max scans per file, max concurrent campaigns (no hard-coding)
- [ ] OT-P1-009 | Read-only agent access | Agent read calls return existing data without triggering new scans unless force flag set
- [ ] OT-P1-010 | Force scan queueing | Enqueue force-scan requests in controlled queue respecting global concurrency limits

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Trend analysis | Display code health metrics over time (% files long, % files with issues, avg issues per file)
- [ ] OT-P2-002 | Issue-tracker integration | Create tasks in app-issue-tracker for high-severity tidiness issues
- [ ] OT-P2-004 | Remediation automation | Auto-apply safe fixes (e.g., dead import removal) with approval workflow
- [ ] OT-P2-005 | Custom rule engine | Allow humans to define custom tidiness rules (e.g., "no files >X lines in api/handlers/")
- [ ] OT-P2-006 | Multi-scenario reports | Generate fleet-wide tidiness reports showing worst offenders across all scenarios
- [ ] OT-P2-007 | CI/CD integration | Webhook/API hooks for blocking PRs with new tidiness violations
- [ ] OT-P2-008 | Smart prioritization | Use file criticality (e.g., main.go more important than test fixtures) to rank issues

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: Go API (Makefile execution, language detection, file traversal, AI orchestration), React UI (dashboard, campaign management, metrics visualization), CLI (agent integration)
- Data + storage expectations: PostgreSQL (issue tracking, campaign state, audit trail), file-based JSON (light scan caching for portability), optional Redis (expensive operation caching)
- Integration strategy: CLI-first for agents → HTTP API for UI/external → visited-tracker campaigns → resource-claude-code/codes for AI analysis → optional static analysis tools (gocyclo, dupl, jscpd)
- Non-goals / guardrails: Not auto-fixing code; not pattern/anti-pattern detection; not standards enforcement (that's scenario-auditor's domain); not lint/type/static-quality contract enforcement (that's quality-health's domain); not real-time IDE integration (batch-oriented)
- **Unowned capability gap (2026-08-18):** auto-fix and pattern-based smell detection previously belonged to the `code-smell` scenario, which was retired as abandoned (no dependents, no registry entry, source stale since 2025-12; the planned integration was never built). These two capabilities now have **no owner**. Tidiness Manager still declines them by design — adopting them would be a deliberate scope expansion requiring a PRD revision, not a bugfix.

## 🤝 Dependencies & Launch Plan
- Required resources: postgres (data storage), resource-claude-code (AI analysis), resource-codes (additional AI capabilities)
- Optional resources: redis (caching), visited-tracker (campaign management, file prioritization)
- Optional tools: gocyclo (Go complexity analysis), dupl (Go duplication detection), jscpd (TS/JS duplication detection)
- Scenario dependencies: visited-tracker (file tracking), scenario-auditor (complementary standards checks), app-issue-tracker (task creation integration for P2)
- Operational risks: AI cost runaway (mitigate with strict batching + session limits); false positive noise (mitigate with configurable thresholds); Makefile inconsistency across scenarios (document standards); campaign resource exhaustion (enforce global concurrency limit K)
- Launch sequencing: Phase 1 - Light scanning + language metrics + basic UI (2 weeks); Phase 2 - AI integration + visited-tracker + agent API (3 weeks); Phase 3 - Auto-campaigns + issue management (2 weeks); Phase 4 - Integrations + P2 features (ongoing)

## 🎨 UX & Branding
- Look & feel: Developer-focused dark theme dashboard inspired by scenario-auditor; clean data tables with sortable columns; split-pane for file details; minimalist campaign controls
- Accessibility: Keyboard navigation for all tables/filters; high contrast for issue severity indicators; screen reader support for campaign status announcements
- Voice & messaging: Calm, systematic, proactive - "Continuous tidiness prevents emergencies" / "Comprehensive coverage, zero redundancy" / "Your code's health monitor"
- Branding hooks: Severity indicators (🔴 Critical length, 🟠 High complexity, 🟢 Clean); Campaign status badges (🟢 Active, ⏸️ Paused, ✅ Complete, ❌ Error); Visit staleness indicators (🔥 Unvisited, ⚠️ Stale, ✅ Recent)

## 📎 Appendix

### Integration with Existing Scenarios

**visited-tracker**: tidiness-manager creates campaigns per scenario, uses visit counts to avoid redundant analysis, marks files as visited after smart scans

**scenario-auditor**: Complementary - auditor enforces standards compliance (security, schema, best practices); tidiness enforces cleanliness (length, organization, duplication)

**quality-health**: Complementary - quality-health owns lint/type/static-quality contracts, strict config policy, suppressions, and static-quality autofix. Tidiness Manager must not duplicate those findings.

**app-issue-tracker**: Consumer in P2 - high-severity tidiness issues can auto-create tasks for human/agent follow-up

### Performance Targets

- Light scan: <60s for scenarios <50 files, <120s for scenarios <200 files
- AI batch: Max 10 files per batch, max 5 concurrent batches
- Campaign: Max K=3 concurrent auto-campaigns initially
- API response: <500ms for read-only agent queries (cached)

### Staleness Algorithm

Files prioritized by score = (days_since_last_visit * 2) + (days_since_last_modification) - (total_visit_count * 0.5)

Higher score = higher priority for next smart scan

### Issue Categories

- **length**: Files exceeding line count thresholds (detected via line counting)
- **duplication**: Repeated logic across files (detected via dupl/jscpd)
- **complexity**: High cyclomatic complexity functions (detected via gocyclo)
- **technical_debt**: TODO/FIXME/HACK markers (detected via regex)
- **coupling**: Excessive imports/dependencies (detected via import counting)
- **dead_code**: Unused functions, imports, components (AI-powered)
- **style**: Inconsistent naming, patterns (AI-powered)

### Campaign State Machine

```
CREATED → ACTIVE → (PAUSED ⇄ ACTIVE) → COMPLETED/ERROR
```

Transitions:
- CREATED → ACTIVE: First session starts
- ACTIVE → PAUSED: Manual pause or error threshold
- PAUSED → ACTIVE: Manual resume
- ACTIVE → COMPLETED: All files visited OR max sessions reached
- ACTIVE/PAUSED → ERROR: Repeated failures exceed threshold

### Language Metrics System

Light scans automatically detect languages and collect metrics per language:

**Supported Languages**: Go, TypeScript, JavaScript, Python, Rust

**Universal Metrics** (no tools required):
- Technical debt markers: TODO, FIXME, HACK counts (case-insensitive regex)
- Import metrics: avg/max imports per file (language-aware parsing)
- Function metrics: avg/max functions per file (language-aware parsing)

**Tool-Enhanced Metrics** (graceful degradation):
- **Complexity** (Go via gocyclo): cyclomatic complexity, high-complexity function locations
- **Duplication** (Go via dupl, TS/JS via jscpd): duplicate block detection with file locations

**Design Principles**:
- Detection-based: adapts to scenario structure (not Makefile-dependent)
- Graceful: works without optional tools, richer with them installed
- Non-blocking: metric failures don't break light scans
- Actionable: directly supports refactor phase prioritization
