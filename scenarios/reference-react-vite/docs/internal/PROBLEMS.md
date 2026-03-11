# Known Issues

This document tracks known issues, tech debt, and deferred work for future agents.

## Active Issues

### High Priority

| ID | Component | Description | Severity | Added |
|----|-----------|-------------|----------|-------|
| P009 | Scoring | Completeness score 45/100, target is 96+ | 4/5 | 2026-03-11 |

### Medium Priority

| ID | Component | Description | Severity | Added |
|----|-----------|-------------|----------|-------|
| P006 | API | No integration tests with testcontainers | 3/5 | 2026-03-11 |
| P010 | E2E | No E2E tests with Playwright | 3/5 | 2026-03-11 |

### Low Priority

| ID | Component | Description | Severity | Added |
|----|-----------|-------------|----------|-------|
| P011 | Auditor | 4 PRD-related auditor violations (read-only file) | 2/5 | 2026-03-11 |
| P012 | Validation | Monolithic test file penalty (2 points) - tasks_test.go validates 4 requirements | 2/5 | 2026-03-11 |

## Tech Debt

### API Layer
- [ ] Add request validation middleware
- [ ] Implement rate limiting
- [ ] Add OpenAPI/Swagger documentation generation

### UI Layer
- [x] ~~Replace template UI with task management interface~~ (Phase 11)
- [x] ~~Add react-router for navigation~~ (Phase 11)
- [x] ~~Implement proper error boundaries~~ (Phase 9)
- [x] ~~Add loading states for data fetching~~ (Phase 11)

### Testing
- [x] ~~Add unit tests for domain packages (tasks, projects, notes)~~ (Phase 3)
- [x] ~~Add handler tests with mock repositories~~ (Phase 3-4)
- [ ] Add integration tests with testcontainers
- [ ] Add E2E tests with Playwright or Cypress
- [ ] Add bats-based CLI integration tests

### Documentation
- [x] ~~Add CLI command reference~~ (Phase 8)
- [ ] Generate API docs from code

## Deferred Decisions

### ORM vs Raw SQL
**Decision**: Using raw SQL with repository pattern.
**Rationale**: Reference scenario should demonstrate the simplest pattern that works. ORMs add complexity without clear benefit for this use case.
**Revisit when**: Business logic becomes complex enough to benefit from ORM features.

### Multiple Database Support
**Decision**: PostgreSQL only.
**Rationale**: Reference scenario focuses on demonstrating patterns, not cross-database compatibility.
**Revisit when**: Need to demonstrate SQLite for testing or desktop deployments.

## Resolved Issues

| ID | Description | Resolution | Date |
|----|-------------|------------|------|
| R001 | SQL injection false positives from auditor | Refactored query builder to avoid fmt.Sprintf on SQL | 2026-03-11 |
| R002 | CORS wildcard security warning | Made CORS configurable via CORS_ALLOWED_ORIGINS env var | 2026-03-11 |
| P001 | No unit tests exist yet | 100+ Go tests, 30 UI tests implemented | 2026-03-11 |
| P002 | Template UI not replaced | Full task management interface with Dashboard, Tasks, Projects pages | 2026-03-11 |
| P003 | 0% requirement completion | Requirements structured with 40 items across 20 modules | 2026-03-11 |
| P004 | No routing implemented | react-router-dom with 3 routes (Dashboard, Tasks, Projects) | 2026-03-11 |
| P005 | Only status command in CLI | Full API parity: 15 commands for tasks/projects/notes CRUD | 2026-03-11 |
| P007 | service.json initialization | Setup steps configured with pnpm install --ignore-workspace | 2026-03-11 |
| P008 | 1:1 requirement mapping penalty | Decomposed to 40 requirements across 20 modules (0% 1:1 mapping) | 2026-03-11 |
| R003 | Type duplication in UI | Consolidated types in api.ts, factories.ts imports from canonical source | 2026-03-11 |
| R004 | meta vs pagination field mismatch | Fixed api.ts to use pagination matching actual API response | 2026-03-11 |

## Last Updated

2026-03-11 - Synced with current implementation state (Phase 15: Spec Sync)
