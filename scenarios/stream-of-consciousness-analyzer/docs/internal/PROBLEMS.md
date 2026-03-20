# Known Problems & Risks

## P1: Clock Skew in Last-Write-Wins Sync
- **Risk:** Client and server clocks may diverge, causing incorrect conflict resolution
- **Mitigation:** Use server-assigned timestamps for all conflict resolution; client timestamps only for display
- **Status:** Design decision made, needs implementation verification

## P2: Ollama Contention Under Load
- **Risk:** Multiple scenarios + ghost node generation + voice refinement could saturate Ollama
- **Mitigation:** Throttled generation (30s interval, max 1 pending), configurable limits
- **Status:** Architectural mitigation designed, needs load testing

## P3: IndexedDB Storage Limits
- **Risk:** Browsers may impose storage limits on IndexedDB, especially with large voice recordings
- **Mitigation:** Store voice recordings as blobs with size tracking; warn user approaching limits; prioritize sync of large items
- **Status:** Needs browser-specific limit research

## P4: Scoring System API Integration Detection
- **Risk:** scenario-completeness-scoring `api_integration` metric shows 0 endpoints despite UI using 14+ API endpoints via `apiFetch` helper
- **Root cause:** The scoring tool likely scans for direct `fetch("/api/v1/...")` calls rather than detecting abstracted API clients
- **Mitigation:** None needed for functionality; scoring improvement is a tooling issue
- **Status:** Documented, not blocking

## P5: Production Lifecycle Missing
- **Risk:** `vrooli scenario status` warns about missing production lifecycle phase
- **Mitigation:** Add production steps to service.json when deploying
- **Status:** Deferred to deployment phase

## P6: Requirements Live Status Not Tracked
- **Risk:** `scenario-completeness-scoring` shows 0/12 requirements passing despite all tests passing
- **Root cause:** The requirements sync system doesn't detect Go test `[REQ:ID]` annotations from `go test` output. The syncer appears to match test output files but doesn't parse Go test function comments.
- **Impact:** Score capped at ~39/100 until requirement pass tracking works
- **Mitigation:** Tooling improvement needed in test-genie requirement syncer
- **Status:** Documented, not fixable within scenario scope

## P7: Medium Standards Violations Remaining
- **Risk:** 3 medium-severity standards violations remain (dangerous TS patterns in selectors.ts, setup step ordering, ESLint per-rule comments)
- **Mitigation:** Address in next iteration; none are HIGH+ so tests pass
- **Status:** Tracked for next phase
