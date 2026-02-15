# Known Issues & Technical Debt

Agent-maintained document tracking issues, debt, and cleanup history.

## Last Updated
2026-02-15

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
| heartbeat/ | Good | scheduler, executor, handlers, prompt builder, team execution queue, member context all covered |
| tags/ | None | Needs handler tests |
| agents/ | None | Needs handler tests |
| teams/ | Partial | Heartbeat cleanup + handler coverage exists; expand org chart + messaging tests |
| store/ | None | Needs relation and index tests |
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

## E2E Issues

| Area | Issue | Impact | Recommendation |
|------|-------|--------|----------------|
| UI smoke coverage | Smoke tests cover load, scene switching, and new-skill editor open only | Medium | Add BAS cases for skill editing (save/discard), search filtering, and agent creation flows |
| Requirements linkage | BAS workflows not linked to requirements JSON | Low | Add automation validation entries once requirements are formalized |

### Missing data-testid attributes in production bundle (Fixed)
- **Execution ID:** ebf858ab-d3d0-4b9a-b2f6-6e4a11c11a87
- **Output path:** `/tmp/bas/prompt-manager/world-ui-loads`
- **Screenshot:** `/tmp/bas/prompt-manager/world-ui-loads/screenshots/step-02-wait-world-canvas.png`
- **Root cause:** prompt-manager UI was serving a stale production bundle built before data-testid attributes were added, so BAS selectors could not resolve.
- **Fix:** rebuilt the UI bundle (`pnpm run build`) and restarted the scenario to serve the updated `ui/dist`.
- **Status:** Fixed (validated by successful BAS runs: `7d145946-73b2-4562-9378-b6363a6dd499`, `d9baa7b8-a171-4cd8-a2ff-ff794b57ecfb`, `1cfc4bc0-869a-4d18-ba43-2afadb6c450e`)

---

## Stability Issues

_No open crash issues identified. Team editor org chart now guards against self-connections and missing manager labels; relationship panels handle empty states gracefully._

---

## Root Cause Analyses (Resolved)

### Heartbeat trigger crashes when config is missing

**Hypotheses**
1. Trigger handler skips config checks, leading to nil dereference inside executor.
2. Executor assumes heartbeat config exists because scheduling only happens for enabled configs.

**Test/Verification**
- Manual trigger without `heartbeat.json` returns `404` and does not panic.
- Unit test: `api/heartbeat/executor_test.go#TestExecutorExecuteFailsWhenConfigMissing`.

**Root Cause**
- `Executor.Execute` and `TriggerManual` assumed `GetHeartbeatConfig` never returns `nil`, causing unsafe dereferences.

**Fix**
- Guard against nil configs and return a not-found error before updating state.

**Prevention**
- Added explicit nil checks and regression tests.

---

### Scheduler ignores per-member profileKey

**Hypotheses**
1. Scheduler uses a hard-coded default profile key for all runs.
2. Profile key is only honored in manual trigger path.

**Test/Verification**
- Scheduler uses `profileKey` from config in `api/heartbeat/scheduler_test.go#TestSchedulerUsesConfigProfileKey`.

**Root Cause**
- Scheduler did not consult per-member heartbeat config when executing scheduled runs.

**Fix**
- Introduced a config-store seam in the scheduler and resolved profile key at execution time.

**Prevention**
- Added scheduler unit tests covering default and custom profile keys.

---

### Removing a team member leaves scheduled heartbeats and member data

**Hypotheses**
1. Member removal only deletes the relation file and does not clean related member files.
2. Scheduler has no lifecycle hook to unschedule per-member entries when membership changes.

**Test/Verification**
- Member removal now unschedules and deletes the member directory in `api/teams/handlers_cleanup_test.go#TestRemoveMemberCleansDataAndUnschedules`.

**Root Cause**
- Cleanup responsibilities were split across domains with no explicit boundary for member teardown.

**Fix**
- Added `FileTeamStore.DeleteMemberData` and a handler-level cleanup step to unschedule + remove member data.

**Prevention**
- Centralized cleanup logic and added regression test.

---

### Deleting a team leaves scheduled heartbeats active

**Hypotheses**
1. Team delete path does not inform the heartbeat scheduler.
2. Scheduler retains cron entries independently of file store deletion.

**Test/Verification**
- Team deletion now unschedules all member heartbeats in `api/teams/handlers_cleanup_test.go#TestDeleteTeamUnschedulesHeartbeats`.

**Root Cause**
- Team deletion flow skipped scheduler teardown and relied solely on file deletion.

**Fix**
- Unschedule all team heartbeats before deleting team files.

**Prevention**
- Added explicit scheduler cleanup and a regression test.

---

### Incomplete scenario registration in CLIDetector (Fixed)

**Root Cause**
- `NewCLIDetector([]string{"prompt-manager"})` only registered `vrooli` and `prompt-manager` as known CLIs. Scenario CLIs like `visited-tracker`, `app-monitor`, etc. were classified as `CodeExternalTool` instead of `CodeScenarioCLI`.

**Fix**
- Added `discoverScenarioNames()` in `main.go` that reads all scenario directory names at startup and passes them to `NewCLIDetector`. All 86+ scenario CLIs are now correctly classified.

**Prevention**
- Scenario names are discovered dynamically from the filesystem — no hardcoding required when scenarios are added or removed.

---

### Multi-line backtick commands missed by CLIDetector (Fixed)

**Root Cause**
- `Detect()` processed backtick patterns per-line, so multi-line backtick spans (with `\` continuation) were never matched.

**Fix**
- Backtick matching now runs on full content (after stripping code fences). Code fences are replaced with equivalent newlines to preserve line numbering.

**Prevention**
- Tests: `TestCLIDetector_MultiLineBacktick`, `TestCLIDetector_CodeFenceStripped`, `TestCLIDetector_CodeFencePreservesLineNumbers`.

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

## Wiring Gaps

_No open wiring gaps._

---

## Deferred Work

| Item | Reason | Priority |
|------|--------|----------|
| Graph `recent-activity` scoring unwired | `RecentActivityScoreFromTimestamp` is implemented in [CODE: api/graph/scoring.go:RecentActivityScoreFromTimestamp] but `Node` lacks a timestamp field to feed it, so `recentActivityScore` returns a neutral 0.5. Wire it when nodes gain `updatedAt` metadata. | Medium |
| Qdrant integration | Optional feature, not core | Low |
| CLI shell completion | Nice-to-have | Low |
| Semantic search | Requires Qdrant | Low |
