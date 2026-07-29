# Known Issues & Follow-Up Tasks

This internal ledger tracks unresolved problems, blockers, and technical debt for the secrets-manager scenario.

**Last Updated**: 2025-11-18 22:58 (scenario-improver-20251118-134731-p13)

---

## Current Validation Gates (2026-07-23)

## Work ladder

- Rung: W3
- Evidence: the operator plan requires canonical credential authority and a full scenario-owned suite; `vrooli scenario requirements validate secrets-manager` passes, while the 2026-07-29 Test Genie run `20260729-183036-01656268` initially failed implementation validation for unrouted file persistence, missing baseline headers, and absent BAS workflow coverage.
- Blocker: remaining Security Health lockfile advisories are transitive dependency findings; the direct code and workflow defects are being repaired before the server-owned rerun.
- Measured: 2026-07-29

- Linux Tier 1 Vault acceptance is intentionally fail-closed until the active user has the declared `secret-tool` host prerequisite. `vrooli host install secret-tool --sudo-mode=ask --json` requires an interactive privileged operator on this host; do not bypass it with a raw system package command.
- Secrets Manager's unit policy now has a shared API test utility and valid UI coverage configuration. The UI suite covers API contracts plus manifest-editor, app-shell, journey, resource-panel, data-hook, campaign, tier-readiness, dashboard snapshot, live security/compliance, resource-tree, scenario-selection, deployment failure states, JSON manifest previews, and per-resource workbench behavior. `pnpm test:coverage` meets the enforced global thresholds without lowering them; use that command as the current coverage source of truth and continue expanding behavior tests and production seams as the UI evolves.
- The UI deployment surface is validated statically at interop and PWA L5. Remaining UI Health component-location and governed-primitive findings are advisory refactoring debt; they do not change route or secret-storage behavior.
- The deployment-readiness journey now treats an empty successful response as a retryable readiness error instead of dereferencing it. The regression was caused by asynchronous hook work observing a cleared test mock during teardown; hooks are now cleaned up before mocks reset, and the response guard protects the same runtime failure mode. Evidence: `pnpm test:coverage` and Unit Health run `uh-20260723-045127` (zero blocking findings).
- Database-backed override tests now use the repository-standard `TEST_DATABASE_URL` contract instead of a hard-coded developer credential. Test Genie or CI must provide an isolated database with the Secrets Manager schema; otherwise those integration tests skip with an actionable reason and never target a shared scenario database.

---

## ✅ Recently Resolved (2025-11-18)

1. **UI TypeError on Load** - Fixed HealthResponse interface to match actual API response structure (dependencies.database is object, not string)
2. **Makefile Standards Violations** - Added "Usage:" header to help target per standards requirement
3. **POSTGRES_PASSWORD in Error Logs** - Removed specific env var names from error messages to avoid logging sensitive variable names

---

## 🔴 Critical Blockers

### 1. Deployment Intelligence Still Needs Automation (P0)
**Severity**: 4/5
**Impact**: Deployment-manager relies on manual refreshes when dependency metadata drifts
**Components**: API, Background Jobs

**Description**:
The tier strategy metadata, manifest export endpoint, and deployment-manager handshake now exist (`/api/v1/deployment/secrets` + analyzer auto-refresh). However, reports only refresh when users request a manifest. There is no scheduled job or webhook from `scenario-dependency-analyzer`, so stale `.vrooli/deployment/deployment-report.json` files can linger until someone manually triggers an export.

**Requirements Affected**:
- SEC-DEP-003 & SEC-DEP-004 (deployment-manager handshake + scenario-to-* integration) are functionally complete but lack freshness guarantees.

**Proposed Solution**:
1. Add a cron-style worker (or lifecycle hook) that pings `scenario-dependency-analyzer` when `service.json` changes to keep reports warm.
2. Cache analyzer responses per scenario/tier so repeated manifest requests do not trigger CLI calls.
3. Emit telemetry back to deployment-manager so it can highlight when the last analyzer run occurred.

**Next Steps**:
- [ ] Hook into analyzer CLI or API with a nightly refresh job
- [ ] Persist `analyzer_generated_at` on `deployment_manifests` table
- [ ] Surface “report stale” warnings in the UI journey cards

---

### 2. Guided UX Needs Deeper Action Hooks (P0)
**Severity**: 4/5
**Impact**: Journeys exist but still dead-end without provisioning/remediation automation
**Components**: UI, CLI

**Description**:
The orientation hub, journey cards, and multi-step flows now ship in the React app. Operators can start "Configure Secrets", "Fix Vulnerabilities", and "Prep Deployment" experiences. However, the flows still rely on copy/pasting CLI commands. There are no embedded wizards for uploading secrets, no "file issue" buttons, and no one-click remediation triggers.

**Requirements Affected**:
- SEC-UX-003 / SEC-UX-004 are functionally present but need action hooks to meet the "autopilot" bar the PRD calls for.

**Proposed Solution**:
1. Integrate the new provisioning endpoint into the Configure Secrets journey (inline forms + success toasts).
2. Add CTA buttons that open `app-issue-tracker` tasks or spawn claude-code remediation agents directly from vulnerability cards.
3. Expand the deployment journey so the exported manifest can be sent straight to `scenario-to-desktop/mobile/cloud` rather than just dumping JSON.

**Next Steps**:
- [ ] Wire the provisioning modal to `/api/v1/secrets/provision`
- [ ] Add "Create issue" + "Auto-fix" actions to vulnerability cards
- [ ] Integrate manifest export CTA with scenario-to-desktop CLI

---

## 🟠 High Priority Issues

### 3. Main.go Monolithic Structure
**Severity**: 4/5
**Impact**: Maintainability, testing, readability
**Components**: API

**Description**:
`api/main.go` is 2204 lines with all types, handlers, services, and bootstrap logic mixed together. This makes it hard to:
- Isolate and test individual components
- Onboard new developers
- Track changes in version control (large diff surfaces)

**Proposed Solution**:
Refactor into:
- `models.go`: All struct definitions
- `handlers.go`: HTTP handlers only
- `vault_service.go`: Vault CLI integration and fallback logic
- `scanner_service.go`: Security scanning and vulnerability detection
- `compliance_service.go`: Compliance aggregation logic
- `db.go`: Database queries and connection management
- `config.go`: Environment variable parsing and config struct
- `main.go`: Server bootstrap (routes, middleware, startup)

**Next Steps**:
- [ ] Create branch for refactoring
- [ ] Move models to separate file
- [ ] Extract service layers
- [ ] Verify all tests still pass
- [ ] Update imports across test files

---

### 4. Requirements Traceability Evidence Refresh
**Severity**: 1/5
**Impact**: Requirement completion is pending a fresh comprehensive Test Genie snapshot.
**Components**: Requirements, Tests

**Description**:
The imported registry module maps all 12 PRD operational targets to focused requirements with resolvable validation references. `vrooli scenario requirements validate secrets-manager` passes and reports zero unproven claims. The current requirements snapshot predates the latest server-owned suite, so no requirement status or PRD checkbox may be promoted until Test Genie records live evidence.

**Evidence**:
- `requirements/01-operational-targets/module.json`
- `vrooli scenario requirements validate secrets-manager`
- `swarm-manager` record `rec-961a96c0f5a31dc6`

**Next Steps**:
- [x] Normalize exact PRD target references and move authoritative claims into an imported module.
- [x] Tag the existing API, CLI, and UI tests that exercise the registered claims.
- [ ] Let the comprehensive Test Genie suite update `coverage/requirements-sync/latest.json`.
- [ ] Promote requirement statuses and PRD checkboxes only if the updated snapshot proves them.

---

### 5. Security Scanner Errors
**Severity**: 3/5
**Impact**: Scan results may be incomplete
**Components**: API (scanner_service)

**Description**:
Scenario status reports "⚠️ Scan errors: 11" in API logs. Root cause unknown - could be:
- File permission issues
- Pattern regex compilation failures
- Context timeouts on large files
- Unsupported file encodings

**Proposed Solution**:
1. Add detailed error logging to `scanComponentsForVulnerabilities` function
2. Wrap pattern matching in error handlers
3. Add file size/encoding checks before attempting to scan
4. Return partial results with error annotations

**Next Steps**:
- [ ] Review scanner logs for specific error messages
- [ ] Add error telemetry to scanner functions
- [ ] Test scanner on edge cases (binary files, large files, empty files)
- [ ] Update scanner tests to cover error paths

---

### 10. Unit Test Failures (Health & Security Scan)
**Severity**: 3/5
**Impact**: Unit test suite failing (2 of 6 phases failed)
**Components**: API, Tests

**Description**:
1. **Health Handler Test**: Expects status "healthy" but gets "degraded". Test environment doesn't mock Postgres/Vault connections properly.
2. **Security Scan Handler Test**: Times out after 30s. Scanner walks entire filesystem slowly, needs optimization or scoped test fixtures.

**Test Output**:
```
--- FAIL: TestHealthHandler/Success (0.00s)
    main_test.go:48: Expected status 'healthy', got degraded
panic: test timed out after 30s
```

**Proposed Solution**:
1. For health tests: Mock DB and Vault connections or accept "degraded" as valid when dependencies unavailable
2. For scan tests: Create small test fixtures in `api/testdata/` instead of scanning entire codebase
3. Add timeout configuration per test

**Next Steps**:
- [ ] Mock database connection for unit tests (use testcontainers or in-memory sqlite)
- [ ] Create api/testdata/ with minimal scan fixtures
- [ ] Reduce TestSecurityScanHandler timeout or optimize scanner
- [ ] Verify tests pass in CI environment

---

### 11. Business Test API Endpoint Timeout (UPDATED 2025-11-18 14:52)
**Severity**: 3/5
**Impact**: Integration test suite incomplete
**Components**: API, Scanner, Tests

**Description**:
Business phase test fails on `/api/v1/security/vulnerabilities` endpoint due to response timeout. Endpoint takes 20+ seconds to respond because it performs a full filesystem scan (23,693 files) on every request. Test framework likely has 10-15 second timeout.

**Root Cause**:
- `scanComponentsForVulnerabilities()` walks entire Vrooli repository on each API call
- No caching mechanism for scan results
- No background/async scanning with result storage

**Proposed Solution** (choose one):
1. **Add caching layer**: Store scan results in memory/Redis with TTL, only re-scan on demand
2. **Background scanning**: Run scan on startup/schedule, store in Postgres, API reads from DB
3. **Increase business test timeout**: Modify test framework to allow 30-60s for scan endpoints
4. **Scope scanning**: Accept component/path filters to scan only requested directories

**Next Steps**:
- [x] Confirmed endpoint works but takes 20+ seconds (2025-11-18 14:52)
- [ ] Implement caching or background scanning
- [ ] Add scan status/progress endpoint for long-running scans
- [ ] Update business test timeout configuration

---

## 🟡 Medium Priority Issues

### 6. Test Infrastructure Incomplete
**Severity**: 3/5
**Impact**: Cannot run full phased testing suite
**Components**: Test

**Description**:
Missing files required for comprehensive testing:
- `.vrooli/endpoints.json`: API contract definitions for integration tests
- `.vrooli/lighthouse.json`: UI performance budgets
- Test lifecycle step doesn't invoke `test/run-tests.sh` properly

**Proposed Solution**:
1. Create `.vrooli/endpoints.json` with critical endpoint contracts:
   ```json
   {
     "endpoints": [
       {"path": "/api/v1/health", "method": "GET", "schema": "health-api"},
       {"path": "/api/v1/vault/secrets/status", "method": "GET", "response_required_fields": ["total_resources", "missing_secrets"]}
     ]
   }
   ```
2. Create `.vrooli/lighthouse.json` with performance budgets
3. Verify `test/run-tests.sh` is executable and covers structure → unit → integration phases

**Next Steps**:
- [ ] Generate endpoints.json from current API routes
- [ ] Define Lighthouse budgets based on current performance baseline
- [ ] Ensure test lifecycle step correctly invokes run-tests.sh
- [ ] Add requirement coverage reporting to test output

---

### 7. UI Bundle Dependency on API_PORT Environment Variable
**Severity**: 2/5
**Impact**: Build fails if API_PORT not set during setup
**Components**: Lifecycle, UI

**Description**:
`build-ui` step in lifecycle requires `API_PORT` to be available at build time to bake into Vite bundle. If port changes later, bundle is stale.

**Proposed Solution**:
- Use runtime config injection instead of build-time baking
- Create `/config.js` endpoint that serves `window.__CONFIG__ = {API_BASE_URL}`
- UI loads config before React mounts
- Allows port changes without rebuild

**Next Steps**:
- [ ] Implement `/config.js` API endpoint
- [ ] Update UI to load config at runtime
- [ ] Remove VITE_API_BASE_URL from build step
- [ ] Test that port changes work without rebuild

---

## 🟢 Low Priority / Future

### 8. Historical Telemetry Not Used in UI
**Severity**: 1/5
**Impact**: Missing compliance trend insights
**Components**: UI, Database

**Description**:
Database has `secret_validations` and `secret_scans` tables for historical data, but UI only shows current snapshot. No trend charts or delta analysis.

**Proposed Solution**:
- Add `/api/v1/analytics/trends` endpoint with date range filtering
- Build chart components using a lightweight library (e.g., recharts)
- Display 30-day rolling compliance score on homepage

**Next Steps** (when P0/P1 complete):
- [ ] Design trend visualization mockups
- [ ] Implement analytics aggregation query
- [ ] Add chart components to dashboard
- [ ] Wire up to React Query for real-time updates

---

### 9. WCAG AA Accessibility Unverified
**Severity**: 2/5
**Impact**: May exclude users with disabilities
**Components**: UI

**Description**:
Dark chrome theme looks good but hasn't been tested with accessibility tools. Contrast ratios, keyboard navigation, and screen reader compatibility unknown.

**Proposed Solution**:
- Run axe-core accessibility audits on all pages
- Test with keyboard-only navigation
- Verify color contrast ratios meet WCAG AA (4.5:1 for normal text)
- Add aria-labels to interactive elements

**Next Steps** (when UX flows implemented):
- [ ] Install axe DevTools extension
- [ ] Audit all UI pages
- [ ] Fix identified violations
- [ ] Document accessibility compliance in README

---

### 10. Ambient Clock and Environment Seams Require an API-Wide Injection Migration
**Severity**: 2/5
**Impact**: API tests cannot substitute all wall-clock and environment behavior deterministically.
**Components**: API, Testing Architecture

**Description**:
Unit Health run `uh-20260723-043923` reported `MISSING_INJECTABLE_SEAM` because production API code directly called `time.Now()` and `os.Getenv()` / `os.LookupEnv()`. The deployment manifest path has a real `ManifestClock` seam, and the validation path now has a real `envx.Reader` seam: `SecretValidator` receives `envx.Reader` through `NewSecretValidatorWithEnv`, production uses `envx.OS`, and tests use `mocks.FakeEnv`. This is not a detector-only declaration.

The API-wide migration remains incomplete. Direct environment reads still exist in `security_handlers.go`, `security_scan.go`, `receipt_signing_handlers.go`, `watchlist.go`, `vault_handlers.go`, `postgres_schema.go`, `http_utils.go`, `desktop_storage.go`, and `logger.go`; composition-root reads in `main.go` are intentionally retained there. Pattern strings in scanners are not process reads.

**Proposed Solution**:
- Migrate the remaining direct environment reads to narrow injected readers, starting with security and storage boundaries.
- Continue wiring clock dependencies at the composition root or server dependency boundary while preserving observable behavior.
- Keep the seam registry current and add a structural registry-drift test before considering this debt closed.

**Evidence**:
- `unit-health validate scenario secrets-manager --execution --json`
- `docs/internal/SEAMS.md`
- `docs/concepts/ARCHITECTURE.md`
- `go test ./... -run 'TestManifestBuilder_Build_UsesInjectedClock|TestValidateEnvironmentVariable'` in `scenarios/secrets-manager/api`

---

## 📝 Notes

### Cross-Scenario Dependencies
- **deployment-manager**: Waiting for SEC-DEP-003 handshake API
- **scenario-to-desktop/mobile/cloud**: Waiting for SEC-DEP-004 SDK helpers

### Technical Decisions Pending
1. Should tier strategies be mutable or version-controlled?
2. How to handle secret rotation in deployment manifests?
3. Should remediation flows integrate with claude-code agent spawning?

### Out of Scope (Per Task Boundaries)
- Modifying vault resource implementation (use CLI only)
- Changing postgres resource schema (extend via migrations only)
- Cross-scenario API contract changes (additive only)

---

**Last Updated**: 2025-11-18
**Maintainer**: scenario-improver agents
