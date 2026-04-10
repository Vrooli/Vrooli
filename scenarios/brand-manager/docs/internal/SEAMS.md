# Seams — Module Boundaries & Testability Zones

## Module Map

```
┌──────────────┐    ┌─────────────────┐    ┌──────────────────┐
│   main.go    │───▶│   handlers/     │───▶│   repository/    │
│  (wiring)    │    │  brands.go      │    │  interfaces.go   │
└──────────────┘    └─────────────────┘    └────────┬─────────┘
                                                    │
                           ┌────────────────────────┤
                           │                        │
                    ┌──────▼──────┐    ┌────────────▼───────────┐
                    │  domain/    │    │  sqlite_brands.go      │
                    │  types.go   │    │  sqlite_versions.go    │
                    │  (entities) │    │  sqlite_assignments.go │
                    └─────────────┘    └────────────┬───────────┘
                                                    │
                                       ┌────────────▼───────────┐
                                       │  database/             │
                                       │  connection.go         │
                                       │  schema.sql            │
                                       └────────────────────────┘
```

## Seam Definitions

### 1. Handler ↔ Repository (Primary Seam)

| Property | Value |
|----------|-------|
| **Location** | [CODE: api/repository/interfaces.go] |
| **Responsibility boundary** | Handlers own HTTP concerns (validation, status codes, JSON encoding). Repositories own SQL and data marshalling. |
| **Files on each side** | Handlers: `api/handlers/brands.go`. Repositories: `api/repository/sqlite_*.go` |
| **Mock implementations** | `api/repository/mocks/` — in-memory mocks with error injection and `.Seed()` builder |
| **Key pattern** | Interface injection — handlers accept `repository.BrandRepository` etc., never `*SQLiteBrandRepository` |
| **Testing strategy** | Two layers: (1) integration tests with real SQLite via temp file, (2) unit tests with in-memory mocks for isolated handler logic. |

### 2. Repository ↔ Database (Storage Seam)

| Property | Value |
|----------|-------|
| **Location** | [CODE: api/database/connection.go] |
| **Responsibility boundary** | Repositories own query logic. Database package owns connection setup, schema initialization, and DSN resolution. |
| **Files on each side** | Repositories: `api/repository/sqlite_*.go`. Database: `api/database/connection.go`, `api/database/schema.sql` |
| **Key pattern** | Repositories receive `*sql.DB` — they don't create connections |
| **Testing strategy** | Repository tests create isolated temp databases via `database.Connect` with custom path |

### 3. Domain (Shared Types)

| Property | Value |
|----------|-------|
| **Location** | [CODE: api/domain/types.go] |
| **Responsibility boundary** | Pure data types with no behavior or database dependencies |
| **Rule** | Domain types are imported by handlers, repositories, and CLI — never the reverse |
| **Testing strategy** | No tests needed (no behavior). Validated implicitly through handler and repository tests. |

### 4. CLI ↔ API (Network Seam)

| Property | Value |
|----------|-------|
| **Location** | [CODE: cli/app.go] |
| **Responsibility boundary** | CLI owns argument parsing, output formatting, and API base URL resolution. API owns business logic. |
| **Key pattern** | CLI communicates exclusively via HTTP (no direct DB access). Uses `cli-core/cliapp` for standard patterns. |
| **Testing strategy** | CLI integration tests require a running API server. |

### 5. UI ↔ API (Network Seam)

| Property | Value |
|----------|-------|
| **Location** | [CODE: ui/src/lib/api.ts] |
| **Responsibility boundary** | UI owns rendering and user interaction. API owns data and business logic. |
| **Key pattern** | React Query for server state. `@vrooli/api-base` for URL resolution. |
| **Testing strategy** | Currently template — will need mock API or MSW when UI is built out. |

### 6. ID Generation (Handler Seam)

| Property | Value |
|----------|-------|
| **Location** | [CODE: api/handlers/brands.go] — `IDFunc` type and `WithIDFunc()` method |
| **Responsibility boundary** | Handlers need unique IDs for new entities. The `IDFunc` seam decouples ID generation from `uuid.New()`. |
| **Default** | `uuid.New().String()` (set in `handlers.New()`) |
| **Testing strategy** | `WithIDFunc()` injects a deterministic counter (`id-1`, `id-2`, …) for reproducible mock tests. |
| **Why it matters** | Without this seam, handler tests cannot assert on specific IDs, and tests become order-dependent. |

### 7. Contrast Validation (Pure Logic Seam)

| Property | Value |
|----------|-------|
| **Location** | [CODE: api/contrast/wcag.go] |
| **Responsibility boundary** | Pure WCAG 2.1 AA contrast calculation — no HTTP, DB, or domain dependencies. Handlers call it as a stateless function. |
| **Files** | `api/contrast/wcag.go` (ParseHex, RelativeLuminance, Ratio, CheckPair, CheckBrandColors) |
| **Testing strategy** | Direct unit tests in `api/contrast/wcag_test.go` — no mocks needed (pure functions). |
| **Why it matters** | Separating contrast logic from handlers keeps it reusable for CLI validation, AI generation rejection, and future scanner plugins. |

### 8. Error Semantics (apierr Package)

| Property | Value |
|----------|-------|
| **Location** | [CODE: api/apierr/apierr.go] |
| **Responsibility boundary** | apierr owns error categories, HTTP status mapping, recovery hints, and JSON serialization. Handlers construct errors via `apierr.Validation()`, `apierr.NotFound()`, `apierr.Internal()`, `apierr.Dependency()` — never raw strings. |
| **Files on each side** | Producers: `api/handlers/brands.go`, `api/handlers/contrast.go`. Consumer: `ui/src/lib/api.ts` (parses `ApiError` JSON). |
| **Key pattern** | Structured error response `{code, message, recovery}` enables UI to classify errors, show recovery hints, and decide whether retry is appropriate. |
| **Testing strategy** | Unit tests in `api/apierr/apierr_test.go` (7 tests). Integration test `TestStructuredErrorResponse` in `api/handlers/brands_mock_test.go` validates the full HTTP→JSON→category flow. |

### 9. Logging Middleware (Signal Capture)

| Property | Value |
|----------|-------|
| **Location** | [CODE: api/main.go] — `statusWriter` + `loggingMiddleware` |
| **Responsibility boundary** | Captures HTTP status codes via `statusWriter` wrapper. Classifies log severity: info (<400), warn (4xx), error (5xx). |
| **Key pattern** | `statusWriter` wraps `http.ResponseWriter` to intercept `WriteHeader` calls, enabling post-request status logging without modifying handlers. |
| **Testing strategy** | Implicitly tested by all handler integration tests (logs are observable in test output). |

### 10. Dry-Run (Mutation Safety Seam)

| Property | Value |
|----------|-------|
| **Location** | [CODE: api/handlers/brands.go] — `isDryRun(r)` helper, `dryRunResponse()` wrapper |
| **Responsibility boundary** | All 5 mutating endpoints (CreateBrand, UpdateBrand, DeleteBrand, CreateAssignment, DeleteAssignment) check `X-Dry-Run: true` header. When set, full validation runs but no mutations are persisted. |
| **Key pattern** | Validation → dry-run check → early return with `200 OK` + `"dry_run": true` marker. Compatible with cli-core's `--dry-run` global flag. |
| **Testing strategy** | 5 dedicated mock tests (`TestDryRun*`) verify no persistence after dry-run for each mutating endpoint. |

### 11. Request ID (Observability Seam)

| Property | Value |
|----------|-------|
| **Location** | [CODE: api/main.go] — `requestIDMiddleware` |
| **Responsibility boundary** | Assigns unique `X-Request-ID` to every request (reuses client-provided ID or generates UUID). Propagated to response headers and structured log output. |
| **Key pattern** | Middleware runs before logging middleware, so all log lines include `req=<id>` for correlation. |
| **Testing strategy** | Implicitly tested via all handler integration tests (response headers observable). |

### 12. Time Generation (Weak Seam — Future Improvement)

| Property | Value |
|----------|-------|
| **Location** | `time.Now()` calls scattered across `api/repository/sqlite_*.go` |
| **Status** | ⚠️ Not yet injectable — repositories call `time.Now()` directly |
| **Recommended fix** | Add `TimeFunc` field to repository structs (same pattern as `IDFunc` in handlers) |
| **Impact** | Cannot test time-dependent behaviour (e.g. ordering by `updated_at`, timestamp assertions) |

## Import Direction Rules

```
main.go → handlers → repository (interfaces only)
                   → domain

sqlite_*.go → repository (interfaces)
            → domain
            → database/sql (stdlib)

database/connection.go → api-core/database
                       → modernc.org/sqlite (driver)

cli/app.go → cli-core/cliapp
           → cli-core/cliutil
```

**Circular import prevention**: Domain types have zero imports from other internal packages. Repository interfaces import only domain. Implementations import interfaces + domain + stdlib.

## Change Axes

These are the primary ways brand-manager is likely to evolve and where changes land today.

### Axis 1: Adding a Brand Facet (e.g. "Imagery", "Accessibility")

| Step | File(s) | Localized? |
|------|---------|-----------|
| 1. Define struct | `domain/types.go` | Yes |
| 2. Add DB column | `database/schema.sql` + migration | Yes |
| 3. Marshal/unmarshal in repo | `repository/sqlite_brands.go` (Create, Update, scan, `hasContent`) | Moderate — 4 touch points in one file |
| 4. Include in partial-update | `domain/types.go` `ApplyPartialUpdate()` | Yes — single method |
| 5. Add to version snapshot | Automatic (brand is JSON-marshalled whole) | Free |
| 6. UI form fields | `BrandFormPage.tsx` (form state + render) | Yes |
| 7. UI detail display | `BrandDetailPage.tsx` (new section) | Yes |
| 8. API types | `ui/src/lib/api.ts` (add interface) | Yes |

**Cost**: ~8 edits across 5-6 files. Repository scan helpers are the densest area — consider a facet registry if >2 facets are added in one cycle.

### Axis 2: WCAG Contrast Pairings

**Before Phase 7**: Pairings were hard-coded inside `CheckBrandColors`. Adding one pairing meant editing a function body.

**After Phase 7**: Pairings are defined in `contrast.StandardPairings` (a slice of `StandardPairing{Foreground, Background}`). Adding a new pairing is a single append — no logic changes needed.

**Cost**: 1 line change.

### Axis 3: API Error Categories

Adding a new error category (e.g. `rate_limited`):
1. Add `Code` constant in `apierr/apierr.go`
2. Add constructor function
3. Add case to `StatusCode()` switch
4. UI `ApiError.code` union type in `api.ts`

**Cost**: 4 edits in 2 files. Well-localized.

### Axis 4: New API Endpoint

1. Add handler method on `Handlers` struct
2. Register route in `RegisterRoutes()`
3. Add UI API function in `api.ts`
4. Add tests (mock + integration)

**Cost**: Straightforward and additive. No existing code needs to change.

## Decision Points

Key places where the system chooses between alternatives.

| Decision | Location | What it decides | How to find it |
|----------|----------|----------------|----------------|
| Partial-update field merge | `domain.Brand.ApplyPartialUpdate()` | Which fields from the request overwrite the existing brand (non-zero → overwrite, zero → keep) |
| Facet hydration | `repository.hasContent()` | Whether a JSON column from SQLite produces a non-nil pointer or nil (signals "not set" to API consumers) |
| Error classification | `apierr.Validation/NotFound/Internal/Dependency` constructors in handlers | Which HTTP status and recovery hint the client receives |
| Retry eligibility | `ui/src/lib/api.ts` `ApiRequestError.isRetryable` | Whether the UI shows a retry button (internal + dependency = retryable) |
| Version snapshot degradation | `handlers.CreateBrand/UpdateBrand` | If version creation fails, log warning and continue (brand is usable without version history) |
| Contrast pass/fail | `contrast.Checker.CheckPair()` | Whether a color pairing meets AA thresholds (configurable via `Config.ContrastAANormal/Large`) |
| Pagination clamping | `config.Config.ClampLimit()` | Applies default and max limits to caller-provided limit values |
| Scenario status shape | `domain.ScenarioStatusUnassigned/FromAssignment` | Whether to return "no brand" or "has brand" response for a scenario |
| WCAG pairing selection | `contrast.StandardPairings` | Which foreground/background role combinations are checked for brand contrast |
| Dry-run bypass | `handlers.isDryRun(r)` in each mutating handler | Whether to validate-only or validate-and-persist; returns 200+dry_run:true vs normal response |
| Request ID propagation | `main.go` `requestIDMiddleware` | Whether to reuse client X-Request-ID or generate a new UUID for correlation |

## Safe vs Dangerous Changes

| Change | Safety | Notes |
|--------|--------|-------|
| Add field to domain type | Safe | JSON tags handle backwards compatibility |
| Add repository method | Safe | Add to interface + implementation |
| Change SQL schema | Dangerous | Requires migration logic or fresh DB |
| Change handler response shape | Dangerous | CLI and UI depend on response format |
| Change repository interface | Moderate | Must update all implementations and tests |
| Add new handler | Safe | Register in `RegisterRoutes`, add tests |
| Add WCAG pairing | Safe | Append to `contrast.StandardPairings` |
| Add error category | Safe | Add to `apierr` package, update UI type |
| Add CLI command | Safe | Add to `cli/app.go`, register in `registerCommands()` |
| Add dry-run to new endpoint | Safe | Check `isDryRun(r)` after validation, return `dryRunResponse()` |
