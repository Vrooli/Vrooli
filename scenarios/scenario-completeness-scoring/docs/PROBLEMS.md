# Known Problems and Deferred Ideas

## Open Issues

### High Priority
1. **UI Smoke Test Fails via ecosystem-manager Embedding** (Cross-scenario)
   - **Issue**: The `vrooli scenario ui-smoke scenario-completeness-scoring` test routes through ecosystem-manager's UI (port 36110) which embeds scenarios in iframes. Ecosystem-manager's `ThemeContext.tsx` uses `localStorage` directly without a try/catch guard, causing `SecurityError: Failed to read the 'localStorage' property from 'Window': Access is denied for this document` when run in Browserless's sandboxed context.
   - **Impact**: UI smoke tests fail even though scenario-completeness-scoring's own code is fixed.
   - **Fix Required**: ecosystem-manager's `ui/src/contexts/ThemeContext.tsx` needs the same localStorage availability check pattern applied in this scenario's `useRecentScenarios.ts`.
   - **Workaround**: Test the scenario UI directly at its allocated port (e.g., `http://localhost:39141`) rather than through ecosystem-manager embedding.
   - **Date Identified**: 2025-11-29

### Medium Priority
1. **JS to Go Migration Complexity**: The existing `scripts/scenarios/lib/completeness.js` has ~550 lines of logic to port. Need to ensure parity during migration.

2. **Circuit Breaker Threshold Tuning**: Default threshold of 3 failures may need adjustment based on real-world collector behavior. Consider making this configurable per-collector.

### Low Priority
1. **UI Template Code**: The react-vite template includes placeholder UI that should be replaced with actual dashboard components.

## Deferred Ideas

### P2 Features (Future Consideration)
- **OT-P2-005 Webhook Notifications**: Alert on score changes or collector failures
- **OT-P2-007 CLI Enhancement**: Update `vrooli scenario completeness` to call this API
- **OT-P2-008 Score Badges**: Embeddable badges for README files
- **OT-P2-009 Custom Collectors**: Plugin system for new scoring dimensions
- **OT-P2-010 Anomaly Detection**: Alert on unexpected score changes

### Technical Debt to Address Later
1. Consider shared Go library for direct import by ecosystem-manager (avoid HTTP overhead)
2. Evaluate if SQLite is sufficient for high-volume history storage
3. Plan migration path from existing JS scoring to this API

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Browser-automation-studio dependency | High - e2e metrics unavailable | Circuit breaker auto-disables; weight redistribution |
| Score parity with JS version | Medium - different scores during migration | Run parallel validation before deprecating JS |
| SQLite performance at scale | Low - many scenarios with long history | Consider PostgreSQL option for large deployments |

## Resolved Issues

1. **localStorage SecurityError in useRecentScenarios** (2025-11-29)
   - **Issue**: The `useRecentScenarios` hook accessed `localStorage` directly without checking availability, causing `SecurityError` in sandboxed iframe contexts (Browserless, incognito mode, etc.).
   - **Fix**: Added `isLocalStorageAvailable()` helper function that tests localStorage access with try/catch before use. The hook now gracefully degrades when localStorage is unavailable - recent scenarios tracking simply won't persist across sessions but the UI continues to function.
   - **Files Changed**: `ui/src/hooks/useRecentScenarios.ts`

2. **Failure Topography & Graceful Degradation** (2025-11-29)
   - **Issue**: Collectors lacked circuit breaker integration, errors returned generic messages without actionable guidance, and UI didn't communicate partial/degraded states to users.
   - **Fix**: Implemented comprehensive failure handling improvements:
     - Created `pkg/errors/` package with structured error types (`CollectorError`, `ScoringError`, `PartialResult`, `APIError`) categorizing failures by severity, recovery info, and next steps
     - Integrated circuit breakers with collectors via `NewMetricsCollectorWithCircuitBreaker()`
     - Added `CollectWithPartialResults()` returning confidence scores (0-1) and missing collector info
     - API responses now include `degradation` and `partial_result` fields
     - UI shows degradation/partial data banners with expandable details
   - **Files Changed**: New `api/pkg/errors/types.go`, `api/pkg/errors/types_test.go`, updated `api/pkg/collectors/interface.go`, `api/pkg/handlers/scores.go`, `api/main.go`, `ui/src/lib/api.ts`, `ui/src/pages/Dashboard.tsx`, `ui/src/pages/ScenarioDetail.tsx`
