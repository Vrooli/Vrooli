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
- **Completeness score coverage gap**: Test-to-requirement ratio is 0.25 (target: 2:1). The scoring system parses phase-results summary strings for "N passed" counts — only the unit phase contributes "3 passed" (one per language: Go, Node, Shell). BATS integration tests (28 tests) and individual Go/TS test functions (95+/90) are not reflected in the scoring metric. Root cause: test-genie's integration phase produces a descriptive summary instead of an "N passed" summary. The "good" threshold for tests is 25 — unreachable without changes to how test-genie reports integration results.
- **Lighthouse accessibility at 92%**: Aria-labels added to all icon-only buttons in Phase 11 — rerun Lighthouse to verify improvement. Remaining issues may include color contrast (slate-400 on slate-950) and SVG edge visualization lacking aria description.

### Infrastructure
- **UI smoke 502s after test suite**: The test suite restarts the scenario, and the API may not be fully initialized when smoke runs immediately after. Workaround: ensure API health check passes before running smoke. When the scenario is healthy, smoke passes consistently.
- **1 medium standards finding**: type-safety — 2 remaining `as` casts in api.ts (`undefined as T` for 204 void return, `res.json() as Promise<T>` for generic JSON parse) and 1 `@ts-expect-error` in selectors.ts registry (unavoidable generic type inference limitation). These are all well-documented and justified.

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
