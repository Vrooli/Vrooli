# Known Issues & Technical Debt

## Last Updated
2026-02-16

## Code Quality Debt

- **CLI JSON parsing fragility** (`cli/system-monitor`): Uses grep/cut for JSON parsing instead of jq. Will break on unexpected JSON structures or multiline values. Severity: medium. Fix: rewrite CLI in Go or add jq dependency.
- **CLI report endpoint bug** (`cli/system-monitor`): Calls `/api/reports/generate` (missing `/v1/` prefix). Will always 404. Severity: high. Fix: change to `/api/v1/reports/generate`.
- **CLI --quiet flag ignored** (`cli/system-monitor`): Flag is parsed but never checked — has no effect. Severity: low. Fix: implement quiet mode or remove the flag.
- **CLI entry point divergence**: `vrooli-system-monitor` is a separate file missing the `version` command and `-v` flag. Severity: low. Fix: make it a symlink like the `vrooli` entry point.
- **Script API placeholders** (`api/internal/handlers/investigations.go`): ListScripts returns empty array, GetScript and ExecuteScript return 404. 30 scripts exist on disk but aren't served. Severity: medium. Fix: implement filesystem-based script serving.

## Test Gaps

- **No unit tests**: Only `api/internal/collectors/collectors_test.go` and `api/internal/services/monitor_test.go` exist. No tests for handlers, repository, alert service, investigation service, report service, settings, or agent-manager integration.
- **Empty test/ directory**: Test phases defined in service.json via test-genie but no test phase scripts populated.
- **No integration tests**: No end-to-end API tests.
- **No UI tests**: No React component tests.
- **No CLI tests**: No bash test suite for CLI commands.

## Stability Issues

- **In-memory storage**: All data lost on API restart. PostgreSQL repository interface defined but not wired at runtime by default. Could cause data loss in production.
- **Missing API endpoints referenced by UI**: UI calls `/api/v1/metrics/timeline`, `/api/v1/metrics/disk/details`, and `POST /processes/{pid}/kill` — none exist. UI sparkline falls back to client-side accumulation; disk detail view cannot fetch partition data; process kill action silently fails.
- **No authentication**: All API endpoints are publicly accessible. No auth middleware enabled.

## UX Issues

- **simulate command broken**: CLI `simulate` command calls `GET /api/test/anomaly/cpu` which is not registered in the API router. Will always fail.
- **UI references non-existent endpoints**: Timeline and disk detail views degrade gracefully but cannot show full data.
- **No WebSocket**: UI uses HTTP polling (5s/60s/4s intervals), introducing latency vs real-time updates.

## Cleanup History

- 2026-02-16: Previous spec-sync sessions corrected script count (70+ → 30), removed non-existent timeline endpoint from API contract, documented all placeholder endpoints, corrected polling interval descriptions.
