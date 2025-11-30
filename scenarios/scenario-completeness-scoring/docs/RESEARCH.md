# Research Notes

## Background

This scenario was created to address limitations in the existing completeness scoring system that caused issues during automated agent loops in ecosystem-manager.

### Problem Statement

The legacy JavaScript-based scoring (formerly in `scripts/scenarios/lib/completeness.js`, now archived):
1. Has no configurability - all components are hardcoded
2. Fails entirely if any collector fails (no graceful degradation)
3. Can't disable broken infrastructure (e.g., browser-automation-studio)
4. Has no circuit breaker to prevent repeated failures
5. Can't be deployed as part of a desktop app bundle

### Solution

Create a Go-based scenario with:
- Configurable component toggles
- Circuit breaker pattern for resilience
- Graceful degradation (partial scores)
- Score history and trend tracking
- What-if analysis
- UI dashboard for configuration

## Uniqueness Check

Searched for similar functionality in the repo:

```bash
rg -l 'completeness' scenarios/
```

-**Related files found:**
- `scripts/scenarios/lib/completeness.js` - Archived JS implementation (replaced by this scenario)
- `scripts/scenarios/lib/completeness-config.json` - Archived configuration data (superseded by the Go config loader)
- `scenarios/ecosystem-manager/api/pkg/autosteer/metrics*.go` - Duplicate metrics code in ecosystem-manager

**Conclusion**: No existing scenario provides this functionality. The JS implementation and ecosystem-manager's internal metrics code have been deprecated in favor of this scenario.

## Related Scenarios/Resources

| Name | Relationship |
|------|-------------|
| ecosystem-manager | Primary consumer - will call this API for autosteer metrics |
| browser-automation-studio | Optional dependency - provides e2e test metrics when available |
| tidiness-manager | Optional dependency - provides code quality metrics |
| scenario-auditor | Related - provides structural validation (different concern) |

## External References

### Circuit Breaker Pattern
- Martin Fowler's article: https://martinfowler.com/bliki/CircuitBreaker.html
- Netflix Hystrix (now deprecated but pattern is well-documented)
- Go implementation patterns: sony/gobreaker, afex/hystrix-go

### Scoring Systems
- Code quality metrics: SonarQube, CodeClimate
- Test coverage: Istanbul, Go coverage tool
- Complexity metrics: Cyclomatic complexity, cognitive complexity

## Technical Decisions

### Why Go?
1. Consistent with ecosystem-manager and other scenarios
2. Easy cross-compilation for desktop deployment
3. Good concurrency for parallel collector execution
4. Strong typing for configuration management

### Why SQLite?
1. Self-contained - no external database dependency
2. Portable - single file for deployment
3. Sufficient for score history (not high-write workload)
4. Easy backup/restore

### Why React?
1. Consistent with other Vrooli scenario UIs
2. react-vite template provides good starting point
3. Rich ecosystem for charts (sparklines, progress bars)

## Implementation Notes

### Scoring Algorithm (from JS)

The existing scoring uses weighted dimensions:
- Quality (50%): req_pass_rate(20) + target_pass_rate(15) + test_pass_rate(15)
- Coverage (15%): test_coverage_ratio(8) + depth_score(7)
- Quantity (10%): req_count(4) + target_count(3) + test_count(3)
- UI (25%): template(10) + components(5) + api(6) + routing(1.5) + loc(2.5)

Plus penalty deductions for validation quality issues.

### Circuit Breaker States
1. **Closed**: Normal operation, requests pass through
2. **Open**: Failures exceeded threshold, requests fail fast
3. **Half-Open**: After timeout, allow one request to test recovery

For collectors, we use a simplified model:
- Track consecutive failure count
- When count >= threshold, mark as "tripped" (open)
- Periodically attempt retry (half-open)
- On success, reset counter (closed)
