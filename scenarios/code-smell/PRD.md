# Product Requirements Document (PRD)

> **Version**: 2.0.0
> **Last Updated**: 2025-11-18
> **Status**: Active
> **Canonical Reference**: PRD Control Tower

## 🎯 Overview

A self-improving code quality guardian that continuously detects, tracks, and fixes code smell violations across the entire Vrooli codebase. This includes both general best practices and Vrooli-specific patterns, with intelligent auto-fixing for safe changes and a review queue for risky modifications.

**Purpose**: Add a permanent capability for code quality enforcement and improvement that learns from every fix.

**Primary Users**: Developers, DevOps engineers, AI agents building or maintaining Vrooli scenarios.

**Deployment Surfaces**: CLI, API, review queue UI, git hook integration, event-driven automations.

**Intelligence Amplification**: Every code smell pattern discovered and fixed becomes a permanent rule that prevents future scenarios from making the same mistakes. Agents learn what patterns work and what patterns break systems, accumulating institutional knowledge about code quality that compounds over time.

**Recursive Value**: Enables future scenarios like code-review-assistant, technical-debt-manager, scenario-quality-scorer, pattern-library-builder, and migration-assistant.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Hot-reloadable rules engine | Hot-reloadable rules engine that reads from JSON/YAML without restart
- [ ] OT-P0-002 | AI-powered pattern detection | Integration with resource-claude-code for AI-powered pattern detection
- [ ] OT-P0-003 | Avoid redundant checks | Integration with visited-tracker to avoid redundant checks
- [ ] OT-P0-004 | Auto-fix safe patterns | Auto-fix capability for safe patterns (whitespace, imports, etc.)
- [ ] OT-P0-005 | Review queue UI | Review queue UI for approving/denying dangerous changes
- [ ] OT-P0-006 | API endpoints for reviews | API endpoints for other scenarios to request code reviews
- [ ] OT-P0-007 | Vrooli-specific smells | Detect and fix Vrooli-specific smells (hard-coded ports, paths)

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Learning mode | Learning mode that tracks approved/rejected fixes to improve suggestions
- [ ] OT-P1-002 | Statistics dashboard | Dashboard showing smell statistics across codebase
- [ ] OT-P1-003 | Rule editor UI | Rule editor UI for adding patterns without editing files
- [ ] OT-P1-004 | Fix history view | History view tracking what's been fixed over time
- [ ] OT-P1-005 | Batch processing | Batch processing mode for fixing multiple files
- [ ] OT-P1-006 | Configurable risk levels | Configurable risk levels for different fix categories

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Git hooks integration | Integration with git hooks for pre-commit checks
- [ ] OT-P2-002 | Smell trend analysis | Smell trend analysis over time
- [ ] OT-P2-003 | Team-specific rules | Team-specific rule sets
- [ ] OT-P2-004 | Export reports | Export reports in multiple formats

## 🧱 Tech Direction Snapshot

**Architecture**: Go API backend with React UI, hot-reloadable JSON/YAML rule engine, PostgreSQL for rule and history storage.

**AI Integration**: resource-claude-code CLI for complex pattern analysis beyond regex capabilities.

**Caching Strategy**: Optional Redis for analysis result caching; fallback to in-memory cache with shorter TTL.

**File Tracking**: visited-tracker integration to skip unchanged files; fallback to timestamp-based tracking.

**Risk Model**: Three-tier risk levels (safe/moderate/dangerous) determine auto-fix eligibility vs. manual review queue.

**Non-goals**: No n8n workflow orchestration (moving to direct CLI/API integrations per platform direction).

## 🤝 Dependencies & Launch Plan

**Required Resources**:
- resource-claude-code: AI-powered pattern detection beyond regex
- postgres: Persistent storage for rules, violations, fix history, learned patterns

**Optional Resources**:
- redis: Cache analysis results for performance (fallback: in-memory cache)
- visited-tracker: Avoid re-analyzing unchanged files (fallback: timestamp tracking)

**Downstream Enablement**:
- code-review-assistant: Uses learned patterns for automated PR reviews
- technical-debt-manager: Prioritizes refactoring based on smell metrics
- scenario-quality-scorer: Rates code quality before deployment
- ci-cd-healer: Fixes build issues caused by code smells

**Launch Sequencing**:
1. Core rule engine with hot-reload (P0-001)
2. Auto-fix infrastructure with risk levels (P0-004, P1-006)
3. AI integration for complex patterns (P0-002)
4. Review queue UI (P0-005)
5. API for scenario integration (P0-006)
6. Learning system post-launch (P1-001)

**Risks**:
- False positives in detection → mitigated by learning system + manual review queue
- Breaking code with auto-fix → mitigated by risk levels + comprehensive testing
- Claude-code unavailability → fallback to regex-only rules
- Performance impact on large codebases → incremental analysis with visited-tracker

## 🎨 UX & Branding

**Visual Palette**: Dark theme with modern code review interface inspired by GitHub PR view. Monospace font for code, modern sans for UI elements.

**Accessibility**: WCAG AA compliance, full keyboard navigation, high-contrast mode support.

**Tone**: Technical but approachable. Focused and efficient. Confidence-building rather than punitive.

**Layout**: Dashboard with sidebar navigation, split-pane diff viewer for review queue, Monaco editor for rule editing.

**Animation Language**: Subtle transitions only—no flashy effects that distract from code review tasks.

**Target Feeling**: Users feel confident about code quality without being overwhelmed by technical debt.

## 📎 Appendix

### References

- README.md – Quick start guide and implementation details
- docs/rules.md – Rule writing guide
- docs/api.md – API specification and endpoint documentation
- docs/integration.md – Integration patterns for other scenarios
