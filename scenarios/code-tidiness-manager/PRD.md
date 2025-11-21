# Product Requirements Document (PRD)

> **Version**: 2.0.0
> **Last Updated**: 2025-11-18
> **Status**: Active
> **Template Source**: PRD Control Tower Canonical Template

## 🎯 Overview

**Purpose**: Code Tidiness Manager adds automated technical debt detection, cleanup suggestion generation, and maintenance intelligence to Vrooli. It continuously scans the codebase for inefficiencies, redundancies, and cleanup opportunities, generating both automated cleanup scripts for simple issues and detailed analysis for complex architectural improvements.

**Primary Users**:
- Developers maintaining Vrooli scenarios
- DevOps engineers managing infrastructure
- AI agents needing clean workspace guarantees
- Product teams tracking technical debt

**Deployment Surfaces**:
- **CLI**: `code-tidiness-manager` binary for manual scans and cleanup operations
- **API**: RESTful endpoints for programmatic access (`/api/v1/tidiness/*`)
- **UI**: Web dashboard showing cleanup categories, trends, and actionable insights
- **Events**: Message bus integration for cross-scenario coordination

**Intelligence Amplification**: This capability provides agents with clean workspace guarantees, pattern recognition from accepted/rejected cleanups, technical debt awareness for architectural decisions, resource optimization insights, and code quality baselines that all future code generation follows.

**Recursive Value**: Enables future scenarios like code-quality-enforcer (automated PR reviews), resource-optimizer (dynamic allocation), scenario-deduplication-manager (intelligent merging), legacy-code-modernizer (systematic upgrades), and performance-bottleneck-hunter (deep analysis).

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Full codebase scanning | Scan entire Vrooli codebase for cleanup opportunities (backup files, temp files, empty dirs)
- [ ] OT-P0-002 | Safe cleanup script generation | Generate safe cleanup scripts for automated issues with confidence scoring
- [ ] OT-P0-003 | Complex issue detection | Detect complex issues requiring human judgment (duplicate scenarios, architectural drift)
- [ ] OT-P0-004 | REST API access | Provide REST API for other scenarios to request targeted scans
- [ ] OT-P0-005 | Learning persistence | Store scan history and track accepted/rejected suggestions for learning
- [ ] OT-P0-006 | CLI interface | CLI interface for manual triggering and configuration

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Scheduled scanning | Scheduled automatic scanning with configurable frequency
- [ ] OT-P1-002 | Git integration | Integration with git hooks for pre-commit cleanup suggestions
- [ ] OT-P1-003 | Dashboard UI | Dashboard UI showing cleanup categories, counts, and trends
- [ ] OT-P1-004 | Batch operations | Batch operations for applying multiple similar cleanups
- [ ] OT-P1-005 | Custom rule API | Custom rule registration API for scenarios to define their own cleanup patterns
- [ ] OT-P1-006 | Notification system | Notification system for critical technical debt accumulation

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | ML-based predictions | Machine learning model to predict cleanup acceptance likelihood
- [ ] OT-P2-002 | Deep scan mode | "Spring cleaning mode" for comprehensive deep scans
- [ ] OT-P2-003 | CI/CD integration | Integration with CI/CD for automated cleanup in pipelines
- [ ] OT-P2-004 | Cost analysis | Cost analysis showing resource waste from inefficiencies
- [ ] OT-P2-005 | Gamification | Gamification with cleanliness scores and leaderboards

## 🧱 Tech Direction Snapshot

**Architecture Approach**:
- Go-based API server for performance and concurrency (scan engine, rule processing)
- React + Vite UI for dashboard and visualization
- CLI as Go binary wrapping core lib/ functions
- PostgreSQL for persistence (scan history, rules, learning data)
- Redis for caching scan results and temporary analysis data

**Data Storage**:
- Primary entities: ScanResult, CleanupAction, CleanupRule in PostgreSQL
- Cache layer in Redis for performance optimization
- Optional vector storage (Qdrant) for code similarity detection

**Integration Strategy**:
- **Shared workflows first**: Create code-scanner.json n8n workflow for reusable scanning logic
- **Resource CLI second**: Use resource-postgres and resource-redis CLI commands
- **Direct API only when needed**: Real-time scan progress updates require direct endpoints
- **Event-driven coordination**: Publish tidiness.scan.completed, tidiness.cleanup.executed events

**Performance Criteria**:
- Full scan < 60s for entire codebase
- API response < 500ms per suggestion
- Memory usage < 512MB during scan
- False positive rate < 5% for automated cleanups

**Non-Goals**:
- Security credential scanning (delegated to secrets-manager)
- Runtime monitoring (focus is static analysis)
- Cross-repository analysis (v1 scope is single Vrooli installation)

## 🤝 Dependencies & Launch Plan

**Required Local Resources**:
- **postgres**: Persistence layer for scan history, patterns, and learning data
- **redis**: Caching for performance optimization during large scans

**Optional Local Resources**:
- **qdrant**: Vector storage for advanced code similarity detection (fallback: skip duplicate detection features)
- **ollama**: AI-powered code analysis and suggestion refinement (fallback: rule-based analysis only)

**Scenario Dependencies**:
- **Upstream**: File system access, git for repository structure
- **Downstream enablement**: Unlocks code-quality-enforcer, resource-optimizer, scenario-health-monitor

**Cross-Scenario Interactions**:
- Provides to: git-manager (pre-commit suggestions), scenario-health-monitor (debt metrics), code-quality-enforcer (quality baseline)
- Consumes from: ecosystem-manager (new scenario notifications), resource-monitor (system load metrics)

**Technical Risks**:
- False positive cleanups → Mitigation: Confidence scoring, dry-run mode, human review gates
- Performance impact on large codebases → Mitigation: Incremental scanning, caching, throttling
- Data loss from cleanup → Mitigation: Backup before cleanup, rollback capability, audit trail

**Launch Sequencing**:
1. Core scanning engine + rule-based detection
2. API + CLI with dry-run mode
3. PostgreSQL/Redis integration
4. Web dashboard for visualization
5. Event publishing for cross-scenario coordination
6. Git hooks and scheduled scanning

## 🎨 UX & Branding

**Visual Palette**: Light theme with severity-based accent colors (green=clean, yellow=attention needed, red=action required). Modern, clean sans-serif typography. Dashboard layout with cards and charts inspired by GitHub Insights crossed with Marie Kondo minimalism.

**Tone & Personality**: Helpful and encouraging, calm and organized. Target feeling: "My codebase is under control." Non-judgmental presentation of technical debt with positive reinforcement for cleanup actions.

**Accessibility Commitments**:
- WCAG AA compliance for color contrast and text sizing
- Full keyboard navigation support
- Screen reader compatibility for all dashboard elements
- Responsive design (desktop-first, tablet supported)

**Motion Language**: Subtle transitions for state changes, satisfying progress animations during cleanup operations, before/after visualizations of codebase health improvements.

**Voice Characteristics**: Professional and data-driven, focusing on actionable insights rather than blame. Clean, organized interface mirroring the goal of clean code.

## 📎 Appendix

**External References**:
- Clean Code principles by Robert Martin
- Technical Debt Quadrant by Martin Fowler
- SonarQube rule definitions for inspiration
