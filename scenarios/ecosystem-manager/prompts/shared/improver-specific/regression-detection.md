# Regression Detection & Prevention

## Breaking Changes Are Unacceptable

### 1. Pre-Change Baseline

Before ANY modification:
```bash
# Capture current state
./test.sh > baseline_tests.txt
curl http://localhost:$PORT/health > baseline_health.txt
docker ps > baseline_containers.txt
ps aux | grep $RESOURCE > baseline_processes.txt

# Document working features
echo "Working features:" > baseline_features.txt
vrooli resource $NAME status >> baseline_features.txt
```

### 2. Continuous Regression Monitoring

During development:
```bash
# Run every 5 minutes during changes
watch -n 300 './test.sh --quick'

# Monitor logs for new errors
tail -f logs/*.log | grep -E "ERROR|FATAL|CRITICAL"

# Check resource consumption
htop -p $(pgrep -f $RESOURCE)
```

### 3. Regression Test Suite

Mandatory tests after changes:
```yaml
regression_tests:
  - health_checks:
      - all_endpoints_responding
      - response_times_normal
      - no_new_errors_in_logs
  
  - functionality:
      - previous_features_still_work
      - apis_return_same_format
      - cli_commands_unchanged
      - integrations_intact
  
  - performance:
      - response_time_not_degraded
      - memory_usage_not_increased
      - cpu_usage_reasonable
      - no_resource_leaks
  
  - compatibility:
      - backwards_compatible
      - database_schema_compatible
      - api_contracts_maintained
      - config_format_unchanged
```

### 4. Regression Detection Patterns

Common regression indicators:
```markdown
🔴 CRITICAL REGRESSIONS:
- Health check failing
- Service won't start
- API returns 500 errors
- Database connection lost
- CLI commands broken

🟡 MODERATE REGRESSIONS:
- Performance degraded >20%
- Memory usage increased >30%
- New warning messages
- Flaky test failures
- Documentation mismatch

🟠 MINOR REGRESSIONS:
- Code style violations
- Deprecation warnings
- Suboptimal patterns
- Missing test coverage
```

### 5. Rollback Protocol

If regression detected:
```bash
# Immediate rollback
git stash  # Save WIP
git checkout HEAD~1  # Return to working state

# Or selective revert
git diff HEAD~1 -- affected_file.go | git apply -R

# Restart services
vrooli resource $NAME stop
vrooli resource $NAME setup
vrooli resource $NAME develop
```

### 6. Regression Root Cause Analysis

For each regression:
1. **When**: Exact commit that introduced it
2. **What**: Specific functionality broken
3. **Why**: Root cause (not just symptom)
4. **Impact**: What else might be affected
5. **Fix**: Proper solution (not band-aid)
6. **Prevention**: How to avoid similar issues

### 7. Anti-Regression Checklist

Before marking improvement complete:
- [ ] All previous tests still pass
- [ ] No new errors in logs
- [ ] Performance not degraded
- [ ] Memory usage stable
- [ ] API contracts unchanged
- [ ] CLI commands work
- [ ] Integrations functional
- [ ] Documentation accurate
- [ ] No resource leaks
- [ ] Graceful degradation intact

## Regression Impact Scoring

```
impact = users_affected * feature_criticality * downtime_risk

where:
- users_affected: 1-10 (10 = all users)
- feature_criticality: 1-10 (10 = core feature)
- downtime_risk: 1-10 (10 = service fails)

If impact > 50: BLOCK DEPLOYMENT
If impact > 30: REQUIRE REVIEW
If impact > 10: ADD WARNING
```

## Golden Rule

**If it worked before your change, it MUST work after your change.**

No exceptions. No "planned deprecations" without migration path. No "temporary" breakages.