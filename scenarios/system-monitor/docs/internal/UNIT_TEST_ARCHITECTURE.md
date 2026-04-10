# system-monitor Unit Testing Architecture

## Last Updated
2026-02-18

## Test Organization Status
- [x] Go tests co-located with source files
- [x] TypeScript tests configured to be co-located with source files (`src/**/*.test.ts(x)`)
- [x] Consistent naming conventions documented in config
- [x] Test utilities package exists (`api/internal/testutil`, `ui/src/test-utils`)

## Mock Organization Status
- [x] Centralized mock packages for API unit tests
- [x] Mock factory/builder patterns introduced
- [x] Inline mock definitions removed from core service and handler test files

## Testability Status
- [x] Dependency injection used in core services (`CommandRunner`, `Clock`, interfaces)
- [x] Interfaces defined for external dependencies
- [~] Time/environment/filesystem abstraction present in key paths but not yet universal

## Infrastructure Status
- [ ] Testcontainers configured for database tests
- [x] UI unit test setup/config files added (`ui/vitest.config.ts`, `ui/src/test-utils/setup.ts`)
- [ ] CI runs full UI+API unit tests successfully (UI test runtime dependencies are not fully wired yet)

## Issues Found
1. `scenarios/system-monitor/ui/package.json:13` - UI unit-test scripts exist, but scenario-local dev dependencies still need to include runnable `vitest` + `jsdom` in this environment.
2. `scenarios/system-monitor/api` - No `testcontainers` usage for database-backed behavior validation; architecture currently relies on in-memory repository tests.
3. `scenarios/system-monitor/api/internal/collectors/collectors_test.go` - Some tests still rely on wall-clock sleeps and real host metrics, which can increase non-determinism across machines/CI.

## Completed Architecture Improvements
1. Added reusable API test utilities:
   - `scenarios/system-monitor/api/internal/testutil/helpers.go`
2. Centralized API test doubles:
   - `scenarios/system-monitor/api/internal/services/mocks/command_runner.go`
   - `scenarios/system-monitor/api/internal/services/mocks/agent_executor.go`
   - `scenarios/system-monitor/api/internal/handlers/mocks/monitor_querier.go`
3. Refactored existing tests to consume centralized utilities/mocks:
   - `scenarios/system-monitor/api/internal/services/script_test.go`
   - `scenarios/system-monitor/api/internal/services/investigation_test.go`
   - `scenarios/system-monitor/api/internal/handlers/metrics_test.go`
4. Added UI unit-testing skeleton for greenfield growth under `src/`:
   - `scenarios/system-monitor/ui/vitest.config.ts`
   - `scenarios/system-monitor/ui/src/test-utils/setup.ts`
   - `scenarios/system-monitor/ui/src/test-utils/index.ts`
   - `scenarios/system-monitor/ui/src/test-utils/factories/uiFactory.ts`
   - `scenarios/system-monitor/ui/src/test-utils/mocks/http.ts`
5. Reduced test flakiness risk in monitor lifecycle tests:
   - `scenarios/system-monitor/api/internal/services/monitor_test.go` now waits on context cancellation instead of fixed sleeps.

## Priority Improvements
1. Install and lock UI unit-test runtime (`vitest` + `jsdom`) to activate the current UI architecture.
2. Introduce `testcontainers-go` for repository/integration tests that require real database semantics.
3. Add deterministic collector seams (clock or collector interfaces) for host-dependent collector test paths.
