# Implementation Plan

This document provides a structured implementation plan for scenario-completeness-scoring, designed for use with ecosystem-manager's auto-steer system.

## Overview

**Goal**: Replace the JS-based completeness scoring (`scripts/scenarios/lib/completeness.js`) with a configurable Go API that provides resilience through circuit breakers and enables UI-based configuration.

**Reference Implementation**: `scripts/scenarios/lib/completeness.js` (~550 lines)

---

## Phase 1: Core Scoring API (P0)

**Priority**: Critical - Must complete before other phases
**Estimated Complexity**: Medium-High
**Dependencies**: None

### Tasks

#### 1.1 Port Scoring Algorithm to Go
- [ ] Create `api/pkg/scoring/calculator.go`
- [ ] Implement `CalculateQualityScore()` - requirement/target/test pass rates
- [ ] Implement `CalculateCoverageScore()` - test coverage ratio, requirement depth
- [ ] Implement `CalculateQuantityScore()` - counts vs thresholds
- [ ] Implement `CalculateUIScore()` - template detection, components, API integration
- [ ] Implement `CalculateCompletenessScore()` - aggregate all dimensions
- [ ] Port classification logic (`classifyScore()`)

**Reference**: `scripts/scenarios/lib/completeness.js` lines 47-338

#### 1.2 Create Metric Collectors
- [ ] Create `api/pkg/collectors/interface.go` - Collector interface
- [ ] Create `api/pkg/collectors/requirements.go` - Load requirements from index.json
- [ ] Create `api/pkg/collectors/targets.go` - Load operational targets
- [ ] Create `api/pkg/collectors/tests.go` - Load test results
- [ ] Create `api/pkg/collectors/ui.go` - Analyze UI metrics (file count, LOC, etc.)

**Reference**: `scripts/scenarios/lib/completeness-data.js`

#### 1.3 Implement Score API Endpoints
- [ ] `GET /api/scores` - List all scenario scores
- [ ] `GET /api/scores/{scenario}` - Get detailed score breakdown
- [ ] `POST /api/scores/{scenario}/calculate` - Force recalculation
- [ ] Add proper error handling and JSON responses

#### 1.4 Add Unit Tests
- [ ] Test each scoring dimension independently
- [ ] Test weight calculations
- [ ] Test classification thresholds
- [ ] Tag tests with `[REQ:SCS-CORE-*]`

### Acceptance Criteria
- Score output matches JS implementation for same input
- All 4 dimensions calculated correctly
- API returns proper JSON with breakdown

---

## Phase 2: Configuration System (P0)

**Priority**: Critical
**Estimated Complexity**: Medium
**Dependencies**: Phase 1.1 (scoring types)

### Tasks

#### 2.1 Define Configuration Types
- [ ] Create `api/pkg/config/types.go`
- [ ] Define `ScoringConfig` struct with component toggles
- [ ] Define `ComponentConfig` for each dimension (Quality, Coverage, Quantity, UI)
- [ ] Define `PenaltyConfig` for penalty toggles
- [ ] Define `CircuitBreakerConfig` for resilience settings

```go
type ScoringConfig struct {
    Components      ComponentConfig      `json:"components"`
    Penalties       PenaltyConfig        `json:"penalties"`
    CircuitBreaker  CircuitBreakerConfig `json:"circuit_breaker"`
}
```

#### 2.2 Implement Config Loading
- [ ] Create `api/pkg/config/loader.go`
- [ ] Load global config from `~/.vrooli/scoring-config.json`
- [ ] Load per-scenario overrides from `scenarios/{name}/.vrooli/scoring-config.json`
- [ ] Merge configs with proper precedence (scenario > global > defaults)

#### 2.3 Implement Config API Endpoints
- [ ] `GET /api/config` - Get global config
- [ ] `PUT /api/config` - Update global config
- [ ] `GET /api/config/scenarios/{scenario}` - Get scenario override
- [ ] `PUT /api/config/scenarios/{scenario}` - Set scenario override

#### 2.4 Implement Presets
- [ ] Create `api/pkg/config/presets.go`
- [ ] Define "default" preset (all enabled)
- [ ] Define "skip-e2e" preset (disable e2e/lighthouse collectors)
- [ ] Define "code-quality-only" preset (disable UI dimension)
- [ ] `GET /api/config/presets` - List presets
- [ ] `POST /api/config/presets/{name}/apply` - Apply preset

#### 2.5 Integrate Config with Scoring
- [ ] Modify `CalculateCompletenessScore()` to accept config
- [ ] Skip disabled components
- [ ] Implement weight redistribution when components disabled

### Acceptance Criteria
- Config persists across restarts
- Per-scenario overrides work correctly
- Disabling components redistributes weights
- Presets apply correctly

---

## Phase 3: Circuit Breaker (P0)

**Priority**: Critical - Key differentiator from JS version
**Estimated Complexity**: Medium
**Dependencies**: Phase 2 (config for thresholds)

### Tasks

#### 3.1 Implement Circuit Breaker State Machine
- [ ] Create `api/pkg/circuitbreaker/breaker.go`
- [ ] Define states: Closed, Open, HalfOpen
- [ ] Track failure counts per collector
- [ ] Implement state transitions

```go
type CircuitBreaker struct {
    Name           string
    State          BreakerState
    FailureCount   int
    LastFailure    time.Time
    LastSuccess    time.Time
    Threshold      int
    RetryInterval  time.Duration
}
```

#### 3.2 Integrate with Collectors
- [ ] Wrap each collector with circuit breaker
- [ ] On failure: increment count, check threshold
- [ ] On threshold exceeded: trip breaker, log warning
- [ ] On success: reset counter

#### 3.3 Implement Periodic Retry
- [ ] Create background goroutine for retry checks
- [ ] For tripped breakers past retry interval: attempt half-open
- [ ] On retry success: close breaker
- [ ] On retry failure: keep open, reset timer

#### 3.4 Implement Circuit Breaker API
- [ ] `GET /api/health/circuit-breaker` - Get all breaker states
- [ ] `POST /api/health/circuit-breaker/reset` - Reset all breakers
- [ ] `POST /api/health/circuit-breaker/{collector}/reset` - Reset specific

#### 3.5 Add Tests
- [ ] Test state transitions
- [ ] Test threshold behavior
- [ ] Test retry logic
- [ ] Tag tests with `[REQ:SCS-CB-*]`

### Acceptance Criteria
- Breakers trip after N consecutive failures
- Tripped breakers skip collection (fast fail)
- Periodic retry recovers from transient failures
- Reset API works correctly

---

## Phase 4: Health Monitoring (P0)

**Priority**: Critical
**Estimated Complexity**: Low
**Dependencies**: Phase 3 (circuit breaker states)

### Tasks

#### 4.1 Define Health Status Types
- [ ] Create `api/pkg/health/types.go`
- [ ] Define collector status: OK, Degraded, Failed
- [ ] Define overall health status

#### 4.2 Implement Health Tracking
- [ ] Create `api/pkg/health/tracker.go`
- [ ] Track last success/failure time per collector
- [ ] Compute status based on recent history
- [ ] Integrate circuit breaker state

#### 4.3 Implement Health API
- [ ] `GET /health` - Overall API health (for lifecycle)
- [ ] `GET /api/health/collectors` - Per-collector status
- [ ] `POST /api/health/collectors/{name}/test` - Test specific collector

### Acceptance Criteria
- Health endpoint returns collector status
- Status reflects circuit breaker state
- Lifecycle health checks pass

---

## Phase 5: Score History & Trends (P1)

**Priority**: Important
**Estimated Complexity**: Medium
**Dependencies**: Phase 1 (scoring)

### Tasks

#### 5.1 Set Up SQLite Database
- [ ] Create `api/pkg/history/db.go`
- [ ] Define schema for score_snapshots table
- [ ] Implement migrations
- [ ] Create `data/scores.db` on first run

```sql
CREATE TABLE score_snapshots (
    id INTEGER PRIMARY KEY,
    scenario TEXT NOT NULL,
    score INTEGER NOT NULL,
    classification TEXT NOT NULL,
    breakdown JSON NOT NULL,
    config_snapshot JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### 5.2 Implement History Storage
- [ ] Create `api/pkg/history/repository.go`
- [ ] Save snapshot after each calculation
- [ ] Query history by scenario
- [ ] Limit/pagination support

#### 5.3 Implement Trend Detection
- [ ] Create `api/pkg/history/trends.go`
- [ ] Calculate score delta from previous
- [ ] Detect stalls (unchanged across N snapshots)
- [ ] Classify trend: improving, declining, stalled, stable

#### 5.4 Implement History API
- [ ] `GET /api/scores/{scenario}/history` - Get history with trends
- [ ] Add `?limit=N` query parameter
- [ ] Include trend indicator in response

### Acceptance Criteria
- Scores persist across restarts
- History shows score progression
- Trends accurately reflect changes

---

## Phase 6: Analysis Features (P1)

**Priority**: Important
**Estimated Complexity**: Medium
**Dependencies**: Phase 1, Phase 5

### Tasks

#### 6.1 Implement What-If Analysis
- [ ] Create `api/pkg/analysis/whatif.go`
- [ ] Accept hypothetical metric changes
- [ ] Calculate projected score
- [ ] Return delta and new classification

```go
type WhatIfRequest struct {
    Changes []MetricChange `json:"changes"`
}

type MetricChange struct {
    Component string  `json:"component"` // e.g., "quality.test_pass_rate"
    NewValue  float64 `json:"new_value"`
}
```

#### 6.2 Implement Recommendations
- [ ] Create `api/pkg/analysis/recommendations.go`
- [ ] Analyze current breakdown
- [ ] Identify highest-impact improvements
- [ ] Estimate point gain per recommendation
- [ ] Sort by impact

#### 6.3 Implement Bulk Refresh
- [ ] `POST /api/scores/refresh-all` - Recalculate all scenarios
- [ ] Return summary of changes

#### 6.4 Implement API Endpoints
- [ ] `POST /api/scores/{scenario}/what-if`
- [ ] `GET /api/recommendations/{scenario}`
- [ ] `GET /api/trends` - Cross-scenario trends

### Acceptance Criteria
- What-if accurately predicts score changes
- Recommendations are actionable and prioritized
- Bulk refresh works for all scenarios

---

## Phase 7: UI Dashboard (P1)

**Priority**: Important
**Estimated Complexity**: High
**Dependencies**: All API phases

### Tasks

#### 7.1 Create Dashboard Page
- [ ] Replace template `ui/src/App.tsx`
- [ ] Create `ui/src/pages/Dashboard.tsx`
- [ ] List all scenarios with scores
- [ ] Color-code by classification
- [ ] Show trend indicators

#### 7.2 Create Scenario Detail Page
- [ ] Create `ui/src/pages/ScenarioDetail.tsx`
- [ ] Show score breakdown with progress bars
- [ ] Show recommendations list
- [ ] Show history chart (sparkline)

#### 7.3 Create Configuration Panel
- [ ] Create `ui/src/pages/Configuration.tsx`
- [ ] Toggle switches for each component
- [ ] Health status indicators
- [ ] Preset selector
- [ ] Save/reset buttons

#### 7.4 Create Shared Components
- [ ] `ScoreBar.tsx` - Progress bar for dimension
- [ ] `TrendIndicator.tsx` - Up/down/stable arrow
- [ ] `HealthBadge.tsx` - OK/Degraded/Failed badge
- [ ] `Sparkline.tsx` - Mini history chart

#### 7.5 Add API Integration
- [ ] Create `ui/src/lib/api.ts` - API client
- [ ] Fetch scores on load
- [ ] Real-time refresh capability

### Acceptance Criteria
- Dashboard shows all scenarios
- Detail view matches mockups in README
- Configuration changes persist
- UI is responsive (no overflow issues)

---

## Testing Strategy

### Unit Tests (Each Phase)
- Test individual functions in isolation
- Mock external dependencies
- Tag with `[REQ:SCS-*]`

### Integration Tests (After Phase 4)
- Test API endpoints end-to-end
- Test config persistence
- Test circuit breaker behavior

### E2E Tests (After Phase 7)
- Test full user workflows via UI
- Test CLI commands if implemented

---

## Migration Plan

After all phases complete:

1. **Parallel Validation**
   - Run both JS and Go scoring on same scenarios
   - Compare outputs, fix discrepancies

2. **Update Ecosystem-Manager**
   - Replace `pkg/autosteer/metrics*.go` with API calls
   - Update autosteer to use this scenario's API

3. **Update CLI**
   - Modify `vrooli scenario completeness` to call API
   - Fall back to JS if API unavailable

4. **Deprecate JS**
   - Remove `scripts/scenarios/lib/completeness*.js`
   - Update documentation

---

## Notes for Agents

1. **Start with Phase 1** - Core scoring must work before anything else
2. **Reference the JS** - The existing implementation is well-documented
3. **Test early** - Add tests as you implement, don't batch them
4. **Check parity** - Compare Go output with JS output frequently
5. **Use the PRD** - UX mockups show expected behavior
6. **Tag requirements** - Use `[REQ:SCS-*]` in test comments
