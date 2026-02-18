# vrooli-autoheal Unit Testing Architecture

## Last Updated
2026-02-18

## Test Organization Status
- [x] Go tests co-located with source files (`*_test.go`)
- [x] TypeScript tests co-located with source files (`*.test.ts` / `*.test.tsx`)
- [x] Consistent naming conventions for scenario-owned tests
- [x] Test utilities package exists (`api/internal/testutil`, `ui/src/test-utils`)

## Mock Organization Status
- [~] Centralized mock packages partially in place (`api/internal/checks/mocks.go`, `api/internal/checks/registry_mocks_test.go`, `api/internal/checks/system/mocks_test.go`, `api/internal/handlers/mocks_test.go`, `api/internal/bootstrap/mocks_test.go`, `api/internal/healing/mocks_test.go`, `ui/src/test-utils/mocks/checkMetadataContext.ts`)
- [~] Mock factory/builder patterns partially in place (UI API payload factories now centralized in `ui/src/test-utils/factories/api.ts`; Go fixtures still ad-hoc in several suites)
- [~] No inline mock definitions in test files (major inline API mocks removed from checks/system and checks/registry; several test-local doubles still remain in infra/watchdog tests)

## Testability Status
- [x] Dependency injection patterns used for key boundaries (checks execution, handlers, healing strategies)
- [x] Interfaces defined for external dependencies in API internals
- [~] Time/environment/filesystem abstractions are partial (filesystem abstractions present; direct env/time usage still exists in some paths)

## Infrastructure Status
- [ ] Testcontainers configured for database tests (currently sqlmock-only for persistence tests)
- [x] Test setup files configured (`ui/vite.config.ts`, `ui/src/test-setup.ts`, `ui/src/test-utils/setup.ts`)
- [x] Scenario test commands are runnable via Go test and Vitest

## Issues Found
1. `scenarios/vrooli-autoheal/api/internal/handlers/handlers_test.go` - repeated manual status/decode assertions without consistent helper usage throughout the full suite.
2. `scenarios/vrooli-autoheal/api/internal/persistence/store_test.go` - DB behavior validated with sqlmock only; no testcontainers-backed path for real Postgres semantics.
3. `scenarios/vrooli-autoheal/api/internal/checks/autoheal_integration_test.go` - test-local doubles are still inline instead of package-level fixture/mock builders.
4. `scenarios/vrooli-autoheal/api/internal/checks/infra/certificate_test.go` - file/FS doubles are test-local and not reusable across infra tests.

## Improvements Applied In This Pass
1. Added API test helper package for reusable HTTP assertions and JSON decoding: `scenarios/vrooli-autoheal/api/internal/testutil/http.go`.
2. Adopted API helper usage in handler tests to establish pattern: `scenarios/vrooli-autoheal/api/internal/handlers/handlers_test.go`.
3. Added shared UI render/test infrastructure: `scenarios/vrooli-autoheal/ui/src/test-utils/renderWithProviders.tsx` and `scenarios/vrooli-autoheal/ui/src/test-utils/index.ts`.
4. Added centralized UI metadata-context mock helper: `scenarios/vrooli-autoheal/ui/src/test-utils/mocks/checkMetadataContext.ts`.
5. Added global Vitest setup for deterministic cleanup and baseline DOM shim: `scenarios/vrooli-autoheal/ui/src/test-utils/setup.ts`, wired via `scenarios/vrooli-autoheal/ui/src/test-setup.ts`.
6. Refactored existing test files to consume shared test utilities (no new test cases): `App.test.tsx`, `CheckDetailModal.test.tsx`, `TrendsPage.test.tsx`, `SystemProtection.test.tsx`, `CheckTrendGrid.test.tsx`.
7. Extracted `internal/handlers` inline test doubles to centralized package-level test mock file: `scenarios/vrooli-autoheal/api/internal/handlers/mocks_test.go`.
8. Extracted `internal/bootstrap` full-stack and factory test doubles to centralized package-level test mock file: `scenarios/vrooli-autoheal/api/internal/bootstrap/mocks_test.go`.
9. Extracted `internal/healing` registry test doubles to centralized package-level test mock file: `scenarios/vrooli-autoheal/api/internal/healing/mocks_test.go`.
10. Verified no behavioral regression in refactored domains via targeted Go test execution: `go test ./internal/handlers ./internal/bootstrap ./internal/healing`.
11. Extracted `internal/checks` registry suite inline doubles into centralized package-level test mock file: `scenarios/vrooli-autoheal/api/internal/checks/registry_mocks_test.go`.
12. Extracted `internal/checks/system` suite inline doubles into centralized package-level test mock file: `scenarios/vrooli-autoheal/api/internal/checks/system/mocks_test.go`.
13. Verified refactored check suites compile and pass targeted executions: `go test ./internal/checks ./internal/checks/system -run 'TestPlatformFiltering|TestSwapCheck_WithMockReader|TestPortCheck_WithMockReader'`.
14. Added centralized UI API test-data factory module: `scenarios/vrooli-autoheal/ui/src/test-utils/factories/api.ts`.
15. Re-exported factories from shared UI test-utils entrypoint: `scenarios/vrooli-autoheal/ui/src/test-utils/index.ts`, `scenarios/vrooli-autoheal/ui/src/test-utils/factories/index.ts`.
16. Refactored existing UI suites to consume shared factories (no new test cases): `scenarios/vrooli-autoheal/ui/src/App.test.tsx`, `scenarios/vrooli-autoheal/ui/src/shared/components/CheckDetailModal.test.tsx`, `scenarios/vrooli-autoheal/ui/src/surfaces/trends/TrendsPage.test.tsx`, `scenarios/vrooli-autoheal/ui/src/surfaces/trends/components/CheckTrendGrid.test.tsx`, `scenarios/vrooli-autoheal/ui/src/surfaces/dashboard/components/SystemProtection.test.tsx`.
17. Verified refactored UI suites via targeted Vitest run: `pnpm vitest run src/App.test.tsx src/shared/components/CheckDetailModal.test.tsx src/surfaces/trends/TrendsPage.test.tsx src/surfaces/trends/components/CheckTrendGrid.test.tsx src/surfaces/dashboard/components/SystemProtection.test.tsx`.

## Priority Improvements
1. Introduce optional Postgres testcontainers suite for persistence behavior not covered by sqlmock (disabled by default in fast unit path).
2. Add shared Go test fixture builders for common `checks.Result` / timeline payloads under `api/internal/testutil` to reduce ad-hoc struct literals.
3. Move remaining Go test-local doubles from integration/infra suites into package-level `mocks_test.go` files.
