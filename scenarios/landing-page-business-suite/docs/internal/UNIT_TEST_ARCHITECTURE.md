# Landing Page Business Suite Unit Testing Architecture

## Last Updated
2026-02-04

## Test Organization Status
- Go unit tests are co-located with source files under `api/` and `cli/` using `*_test.go`.
- TypeScript unit tests are co-located under `ui/src/**` using `*.test.ts`.
- Shared Go test helpers live in `api/test_helpers_test.go` (test-only file co-located in the package).

## Mock Organization Status
- Go HTTP seams are exercised via small, local stubs (e.g., `stubHTTPClient`) inside the relevant test files.
- Handler tests can stub remote profile behavior through the `RemoteProfileManager` interface.
- No centralized mock package yet; mocks are localized and scoped to the tests that need them.

## Testability Status
- Remote profile handlers depend on a small interface (`RemoteProfileManager`) rather than concrete services.
- Remote profile time-sensitive logic uses an injected clock (`RemoteProfileService.now`) to keep expiry logic deterministic.
- HTTP boundaries use the `HTTPDoer` seam for injected test clients.

## Infrastructure Status
- Database tests use `setupTestDB` which resolves `TEST_DATABASE_URL` or provisions a Postgres container via testcontainers.
- Schema + seed data are ensured for each test run through `ensureSchema` + `seedDefaultData`.

## Issues Found
- Test helpers are currently file-local; consider moving commonly reused stubs into a dedicated `api/internal/testutil` package if reuse grows.

## Priority Improvements
1. Introduce a small `api/internal/testutil` package if cross-test mock reuse increases.
2. Add lightweight table-driven test utilities if validation logic continues to expand.
