# Known Issues & Technical Debt

Agent-maintained document tracking issues, debt, and cleanup history.

## Last Updated
2025-01-25

---

## Code Quality Debt

### API

| Area | Issue | Severity | Recommended Fix |
|------|-------|----------|-----------------|
| Error handling | Inconsistent error response format across domains | Medium | Standardize with shared error types |
| Validation | Input validation scattered in handlers | Low | Extract to middleware or shared validators |

### CLI

| Area | Issue | Severity | Recommended Fix |
|------|-------|----------|-----------------|
| Output formatting | Duplication across domain commands | Low | Consolidate into `internal/output` |

### UI

_No significant debt identified._

---

## Test Gaps

### API Coverage

| Package | Coverage | Gaps |
|---------|----------|------|
| skills/ | Good | handlers_test.go, query_test.go exist |
| tags/ | None | Needs handler tests |
| members/ | None | Needs handler tests |
| testing/ | None | Needs handler tests with mock Ollama |
| search/ | None | Needs handler tests |
| metrics/ | None | Needs repository tests |

### CLI Coverage

| Package | Coverage | Gaps |
|---------|----------|------|
| All | None | No CLI tests yet |

### Recommended Test Priority

1. `api/skills/handlers_test.go` - Extend existing tests
2. `api/tags/handlers_test.go` - CRUD tests with mock repository
3. `api/search/handlers_test.go` - Search logic tests
4. CLI integration tests with mock API

---

## Stability Issues

_None identified._

---

## UX Issues

| Area | Issue | Impact | Recommendation |
|------|-------|--------|----------------|
| CLI | No completion support | Low | Add shell completion scripts |
| CLI | Long content truncated in list views | Low | Add pagination or --limit flag |

---

## Cleanup History

| Date | Change | Outcome |
|------|--------|---------|
| 2025-01-25 | Aligned API with screaming architecture | All domains now have interfaces |
| 2025-01-25 | Added CLI domains for all API endpoints | Full CLI coverage |

---

## Deferred Work

| Item | Reason | Priority |
|------|--------|----------|
| Qdrant integration | Optional feature, not core | Low |
| CLI shell completion | Nice-to-have | Low |
| Semantic search | Requires Qdrant | Low |
