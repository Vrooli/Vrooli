# Problems & Known Issues

> Note (2026-02-14): Some historical entries below use legacy `ideas`/`recommendations` terminology from earlier iterations.

## Open Issues

No known open issues at this time. Use this file to track new defects or risks.

### PRD Template Deviations (Informational)

The PRD contains extra sections in the Appendix that are not part of the standard template:
- "Idea Backlog File Structure" - useful documentation, intentionally kept
- "spec.json Schema" - useful documentation, intentionally kept
- "Execution Controls and Policy Defaults" - useful documentation, intentionally kept

These provide valuable domain context and should remain unless PRD template rules change.

## Deferred Ideas

### P2 Features (Future)
- **OT-P2-001**: Advanced cost formulas for priority calculations
- **OT-P2-002**: Qdrant integration for pattern recognition
- **OT-P2-003**: Analytics dashboard with usage metrics
- **OT-P2-004**: Batch operations on ideas and scenarios
- **OT-P2-005**: Webhook support for external integrations

### Design Decisions Deferred
- Multi-user authentication (currently single-user local installation)
- Complex workflow builders (using simple queue-based processing)
- Kanban/Trello-style UI (using tabbed list interface instead)

## Resolved Issues

### Progress Phase (iter 3) - Standards Remediation (2026-01-28)

| ID | Component | Resolution |
|----|-----------|------------|
| TD-002 | cli/ | Added cli/app_test.go with tests for NewApp, resolveV1Endpoint, and app constants |
| DOC-001 | docs/PROGRESS.md | Fixed broken code reference by removing markdown link syntax from prose |

### Phase 6 - Error Semantics & Recovery Paths (2026-01-28)

| ID | Component | Resolution |
|----|-----------|------------|
| ES-001 | ui/App.tsx | Added ErrorBoundary to catch React runtime errors with full-page fallback |
| ES-002 | ui/pages | Added NotFoundPage for 404 routes - prevents blank page on invalid URLs |
| ES-003 | ui/lib/error-utils | Created error categorization with 8 distinct categories mapped to recovery paths |
| ES-004 | ui/lib/error-utils | Added structured logging with sanitization (URLs, paths stripped from messages) |
| ES-005 | docs/internal | Updated ERROR-SEMANTICS.md with complete category documentation |

### Phase 7 - Intent Clarification (2026-01-28)

| ID | Component | Resolution |
|----|-----------|------------|
| IC-001 | ui/pages/ExecutionPage | Added docblock explaining execution control behavior and run governance |
| IC-002 | ui/pages/SettingsPage | Added docblock explaining placeholder status, persistence gap, and future behavior |
| IC-003 | api/main.go | Added package-level docs explaining purpose, current minimal status, and planned endpoints |
| IC-004 | cli/app.go | Added package-level docs, renamed apiPath→resolveV1Endpoint with docblock |
| IC-005 | docs/internal | Created INTENT.md with scenario purpose, design philosophy, and modification guide |

### Phase 8 - Change Axis & Evolution Resilience Audit (2026-01-28)

| ID | Component | Resolution |
|----|-----------|------------|
| CA-001 | docs/internal/SEAMS.md | Added comprehensive Change Axes section documenting 8 primary axes with cost analysis |
| CA-002 | docs/internal/SEAMS.md | Added Stable vs Volatile areas classification for clear boundaries |
| CA-003 | docs/internal/SEAMS.md | Added Extension Points table for common modification patterns |
| CA-004 | docs/internal/SEAMS.md | Added guidance for P1 integrations and YOLO-mode safeguards |

**Note**: This phase was documentation-focused. The architecture was already well-structured for evolution (previous phases 1-7 established clean seams, config centralization, and type localization). The main deliverable is documenting these patterns for future maintainers.

### Phase 5 - Failure Topography & Graceful Degradation (2026-01-28)

| ID | Component | Resolution |
|----|-----------|------------|
| FT-001 | ui/pages | Fixed misleading "No ideas yet" error message - now shows dedicated ErrorState component |
| FT-002 | ui/lib/api-client | Added structured error differentiation (network/timeout/http/parse) with user-friendly messages |
| FT-003 | ui/lib/api-client | Added timeout enforcement via AbortController (uses apiConfig.requestTimeoutMs) |
| FT-004 | ui/components | Created reusable ErrorState component with retry support for recoverable errors |

### Phase 26 - Progress & Quality Fixes (2026-01-28)

| ID | Component | Resolution |
|----|-----------|------------|
| P26-001 | ui/pages/IdeaDetailsPage.test.tsx | Fixed TypeScript TS2345 error - QueueResponse mock was missing `idea` property |
| P26-002 | ui/pages/IdeaDetailsPage.tsx | Removed 3 non-null assertions by adding explicit null guards in queryFn callbacks |
| P26-003 | ui/pages/ScenarioDetailsPage.tsx | Removed 2 non-null assertions by adding explicit null guards in queryFn callbacks |
| P26-004 | ui/pages/IdeasPage.tsx | Added aria-label to icon-only filter button for accessibility (button-name audit) |
| P26-005 | ui/pages/*.tsx, components/*.tsx | Fixed color contrast by changing text-slate-500 to text-slate-400 for empty state text |
| P26-006 | Lighthouse accessibility | Score improved from 73% to 97% (above 90% threshold) |

### Phase 30 - Integration & CLI Parity (2026-01-28)

| ID | Component | Resolution |
|----|-----------|------------|
| TD-001 | cli/ | CLI binary present after setup; command surface expanded to cover scenarios/execution/settings/queue |
| TD-003 | api/ | Business endpoints implemented across backlog/scenarios/settings/execution/queue |
| TD-004 | api/ | Filesystem-first persistence confirmed; no database schema needed |
| TD-005 | integrations/ | Agent-manager + ecosystem-manager clients use api-core discovery |
