# Known Issues

This document tracks known issues, tech debt, and deferred work for future agents.

## Active Issues

### High Priority

| ID | Component | Description | Severity | Added |
|----|-----------|-------------|----------|-------|
| P001 | Testing | No unit tests exist yet | 4/5 | 2026-03-11 |
| P002 | UI | Template UI not replaced with scenario-specific interface | 4/5 | 2026-03-11 |
| P003 | Requirements | 0% requirement completion (no passing tests) | 4/5 | 2026-03-11 |

### Medium Priority

| ID | Component | Description | Severity | Added |
|----|-----------|-------------|----------|-------|
| P004 | UI | No routing implemented (single page app) | 3/5 | 2026-03-11 |
| P005 | CLI | Only `status` command implemented | 3/5 | 2026-03-11 |
| P006 | API | No integration tests with testcontainers | 3/5 | 2026-03-11 |

### Low Priority

| ID | Component | Description | Severity | Added |
|----|-----------|-------------|----------|-------|
| P007 | Config | service.json initialization files not auto-executed | 2/5 | 2026-03-11 |
| P008 | Scoring | 85% of operational targets have 1:1 requirement mapping (penalty) | 2/5 | 2026-03-11 |

## Tech Debt

### API Layer
- [ ] Add request validation middleware
- [ ] Implement rate limiting
- [ ] Add OpenAPI/Swagger documentation generation

### UI Layer
- [ ] Replace template UI with task management interface
- [ ] Add react-router for navigation
- [ ] Implement proper error boundaries
- [ ] Add loading states for data fetching

### Testing
- [ ] Add unit tests for domain packages (tasks, projects, notes)
- [ ] Add handler tests with mock repositories
- [ ] Add integration tests with testcontainers
- [ ] Add E2E tests with Playwright or Cypress
- [ ] Add bats-based CLI integration tests

### Documentation
- [ ] Add CLI command reference
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

## Last Updated

2026-03-11 - Initial problems documentation created
