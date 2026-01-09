# Requirements Registry

This directory maps PRD operational targets to technical requirements for the scenario-completeness-scoring scenario.

## Module Structure

| Folder | Priority | Description |
|--------|----------|-------------|
| `01-core-scoring/` | P0 | Core score calculation (4 dimensions, weight redistribution, graceful degradation) |
| `02-configuration/` | P0 | Configuration management (toggles, per-scenario overrides, presets) |
| `03-circuit-breaker/` | P0 | Resilience pattern (auto-disable, failure tracking, periodic retry) |
| `04-health-monitoring/` | P0 | Collector health status and diagnostics |
| `05-history-trends/` | P1 | Score history storage and trend analysis |
| `06-analysis/` | P1 | What-if analysis and recommendations |
| `07-ui-dashboard/` | P1 | Visual interface for score management |

## Requirement ID Pattern

All requirements use the prefix `SCS-` followed by module abbreviation:
- `SCS-CORE-*` - Core scoring
- `SCS-CFG-*` - Configuration
- `SCS-CB-*` - Circuit breaker
- `SCS-HEALTH-*` - Health monitoring
- `SCS-HIST-*` - History/trends
- `SCS-ANALYSIS-*` - Analysis
- `SCS-UI-*` - UI dashboard

## Test Tagging

Tag tests with `[REQ:ID]` to enable auto-sync:

```go
// [REQ:SCS-CORE-001] Calculate completeness scores
func TestCalculateScore(t *testing.T) { ... }
```

```typescript
// [REQ:SCS-UI-001] Dashboard overview
describe('Dashboard', () => { ... })
```

## Lifecycle

1. PRD operational targets (OT-P0-*, OT-P1-*, OT-P2-*) map to modules here
2. `requirements/index.json` imports each module
3. Tests update requirement status when they run
4. Coverage summaries in `coverage/phase-results/`

## Related Documentation

- [PRD.md](../PRD.md) - Operational targets
- [Testing Guide](../../docs/testing/guides/requirement-tracking-quick-start.md) - Schema details
