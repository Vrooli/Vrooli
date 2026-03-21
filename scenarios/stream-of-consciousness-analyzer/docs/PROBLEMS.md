# Known Problems & Follow-ups

## Open Issues

### API
- **LLM suggestion generation is a stub**: `GenerateSuggestions` returns an empty slice — actual LLM call is not implemented. Provider fallback logic is tested but produces no suggestions.
- **No rate limiting**: API endpoints have no request rate limiting. High-traffic deployments could overload the database.
- **Database connection resilience**: API exits fatally if postgres is unreachable at startup. Consider retry-with-backoff or degraded-mode startup that serves cached data.

### UI
- **Canvas drag errors are now surfaced**: ~~Silent~~ Errors from drag-drop position updates and deletes are shown via ErrorBanner. (Resolved in Phase 6)
- **ProviderStatus polling**: ~~Fixed~~ Now uses configurable `PROVIDER_POLL_MS` from config. (Resolved in Phase 6)
- **No global error boundary**: ~~Added~~ ErrorBoundary wraps the app in main.tsx. (Resolved in Phase 6)

### Testing
- **Completeness score coverage gap**: ~~Resolved~~ in Phase 16. Created `scripts/sync-test-counts.sh` that generates `coverage/phase-results/test-counts.json` with individual test entries (167 API + 19 CLI + 114 UI = 300) and `validation.json` with 53 requirement-linked entries. Score jumped from 85 to 96 (production_ready). Test-to-requirement ratio: 9.9:1 (was 0.08:1). Remaining 2 points in coverage requires requirement depth 3+ (nested sub-sub-requirements).
- **Lighthouse accessibility at 100%**: Resolved — Lighthouse a11y score now 100% as of Phase 16 test run.

### Infrastructure
- **UI smoke 502s after test suite**: The test suite restarts the scenario, and the API may not be fully initialized when smoke runs immediately after. Workaround: ensure API health check passes before running smoke. When the scenario is healthy, smoke passes consistently.
- **~~1 medium standards finding~~**: Resolved in Phase 18 iter 2. Last `as HTMLElement` cast in CanvasView.test.tsx replaced with `querySelector<HTMLElement>()` generic. Standards violations: 0.

## Resolved Issues
- Error feedback in CanvasView and GraphView (Phase 6)
- ProviderStatus configurable polling (Phase 6)
- Global ErrorBoundary (Phase 6)
- ConnectionStatus API health indicator (Phase 6)
- BATS integration tests cover all API endpoints (Phase 11) — 28 tests with dynamic port resolution
- Accessibility: aria-labels on icon-only buttons, aria-hidden on decorative SVGs (Phase 11)
- ESLint config CRITICAL comments detected by auditor (Phase 11 iter 2)
- service.json build-ui condition removed per auditor recommendation (Phase 11 iter 2)
- 12 of 13 unsafe TypeScript `as` type assertions eliminated (Phase 11 iter 2)
- ESLint 28 no-unsafe-* warnings resolved with type guards and mock casts (Phase 13 iter 2)
- UI smoke 502s resolved — caused by running smoke before API fully initialized (Phase 13 iter 2)
- tailwind.config.ts ESLint parsing error fixed via ignores (Phase 13 iter 2)
