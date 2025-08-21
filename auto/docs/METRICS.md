# Auto/ System Metrics Guide

## 🎯 Overview

The auto/ system collects comprehensive metrics to track progress, identify issues, and optimize performance. This guide explains what metrics are collected, how to interpret them, and how to use them for decision-making.

## 📊 Metric Categories

### 1. Efficiency Metrics
**Purpose**: Measure how effectively the loop uses resources

| Metric | Description | Good Range | Warning Signs |
|--------|-------------|------------|---------------|
| `success_rate` | % of iterations producing improvements | >70% | <50% indicates prompt issues |
| `avg_duration` | Average iteration time (seconds) | 300-900s | >1800s suggests hanging operations |
| `iterations_per_hour` | Throughput rate | 4-12 | <2 indicates problems |
| `improvement_velocity` | Meaningful changes per iteration | >0.7 | <0.3 suggests inefficiency |
| `resource_utilization` | CPU/Memory usage | 20-60% | >80% risks system impact |

### 2. Progress Metrics
**Purpose**: Track advancement toward goals

| Metric | Description | Target | Measurement |
|--------|-------------|--------|-------------|
| `prd_completion` | % of PRD requirements met | 100% | Checkbox counting in PRD.md |
| `resources_healthy` | % of resources running properly | >90% | Status checks |
| `scenarios_valid` | % of scenarios passing validation | >95% | Conversion + runtime tests |
| `issues_resolved` | Cumulative fixes applied | Increasing | Event log analysis |
| `capabilities_added` | New features implemented | Steady growth | PRD P0/P1 tracking |

### 3. Quality Metrics
**Purpose**: Ensure improvements don't cause regressions

| Metric | Description | Acceptable | Critical Action |
|--------|-------------|------------|-----------------|
| `rollback_rate` | % of changes reverted | <5% | >10% stop and review |
| `error_frequency` | Errors per iteration | <2 | >5 investigate immediately |
| `validation_pass_rate` | % passing all gates | >80% | <60% prompt adjustment needed |
| `drift_coefficient` | Deviation from objectives | <0.2 | >0.5 prompt reinforcement |
| `regression_count` | Previously working features broken | 0 | Any requires immediate fix |

### 4. Operational Metrics
**Purpose**: Monitor system health and stability

| Metric | Description | Normal | Alert Threshold |
|--------|-------------|--------|-----------------|
| `uptime_hours` | Continuous loop runtime | Any | N/A (informational) |
| `memory_growth` | MB increase per hour | <10 | >50 indicates leak |
| `log_size_mb` | Disk usage for logs | <100 | >500 rotation needed |
| `worker_timeouts` | Iterations hitting timeout | <5% | >15% reduce timeout |
| `restart_count` | Loop restarts required | <1/day | >3/day stability issue |

## 📈 Reading Metrics Files

### summary.json Structure
```json
{
  "task": "resource-improvement",
  "total_iterations": 145,
  "successful_iterations": 112,
  "failed_iterations": 33,
  "success_rate": 0.77,
  "average_duration": 485.3,
  "last_updated": "2024-01-15T10:30:00Z",
  "progress": {
    "resources_improved": 12,
    "prd_completion_avg": 0.82,
    "validation_passes": 108
  },
  "quality": {
    "error_rate": 0.23,
    "rollback_count": 4,
    "regression_count": 1
  },
  "operational": {
    "uptime_hours": 72.5,
    "memory_usage_mb": 245,
    "disk_usage_mb": 87
  }
}
```

### events.ndjson Format
```json
{"timestamp":"2024-01-15T10:00:00Z","event":"iteration_start","iteration":145,"task":"resource-improvement"}
{"timestamp":"2024-01-15T10:08:23Z","event":"iteration_end","iteration":145,"duration":503,"exit_code":0,"changes":3}
{"timestamp":"2024-01-15T10:08:24Z","event":"summary_generated","size":487,"model":"llama3.2:3b"}
```

### Accessing Metrics

```bash
# Quick summary view
cat auto/data/resource-improvement/summary.json | jq

# Success rate over time
jq '.success_rate' auto/data/*/summary.json

# Recent events
tail -20 auto/data/resource-improvement/events.ndjson | jq

# Error analysis
jq 'select(.event == "error")' auto/data/*/events.ndjson

# Duration trends
jq '.duration' auto/data/resource-improvement/events.ndjson | \
  awk '{sum+=$1; count++} END {print "Avg:", sum/count}'
```

## 📉 Metric Interpretation Patterns

### Pattern 1: Declining Success Rate
```
Iterations 1-50: 85% success
Iterations 51-100: 70% success  
Iterations 101-150: 45% success
```
**Diagnosis**: Prompt staleness or increasing complexity
**Action**: Review recent failures, update prompt guidance

### Pattern 2: Increasing Duration
```
Week 1: 5 min average
Week 2: 10 min average
Week 3: 20 min average
```
**Diagnosis**: Growing complexity or inefficient operations
**Action**: Add timeouts, optimize selection strategy

### Pattern 3: Validation Failures
```
Convert: ✓ Pass
Start: ✓ Pass
Verify: ✗ Fail (repeated)
```
**Diagnosis**: Runtime issues not caught in static validation
**Action**: Strengthen verification steps, add integration tests

### Pattern 4: Resource Starvation
```
CPU: 95%
Memory: 90%
Iterations/hour: 1
```
**Diagnosis**: System overload
**Action**: Reduce concurrent workers, add resource limits

## 🎯 Key Performance Indicators (KPIs)

### Daily KPIs
```bash
# Generate daily report
auto/task-manager.sh --task resource-improvement json summary | jq '{
  date: now | strftime("%Y-%m-%d"),
  success_rate: .success_rate,
  iterations_today: .total_iterations,
  improvements: .progress.resources_improved,
  health: (.success_rate > 0.7 and .quality.error_rate < 0.3)
}'
```

### Weekly KPIs
```bash
# Weekly progress check
for task in resource-improvement scenario-improvement; do
  echo "=== $task ==="
  auto/task-manager.sh --task $task json recent 168 | jq '{
    task: "'$task'",
    weekly_iterations: length,
    weekly_success_rate: (map(select(.exit_code == 0)) | length) / length,
    avg_duration: (map(.duration) | add / length),
    prd_progress: .[-1].prd_completion // "unknown"
  }'
done
```

### Monthly KPIs
```bash
# Monthly executive summary
{
  echo "# Monthly Auto/ System Report"
  echo "Generated: $(date)"
  echo ""
  echo "## Resource Maturity"
  vrooli resource status --format json | jq -r '
    map(select(.Enabled)) |
    "Total: \(length), Healthy: \(map(select(.Running)) | length)"
  '
  echo ""
  echo "## Scenario Reliability"
  # Count validated scenarios
  echo ""
  echo "## System Efficiency"
  jq -s 'add | {
    total_iterations: (map(.total_iterations) | add),
    avg_success_rate: (map(.success_rate) | add / length),
    total_improvements: (map(.progress.resources_improved // 0) | add)
  }' auto/data/*/summary.json
} > monthly_report.md
```

## 🔍 Anomaly Detection

### Red Flags Requiring Immediate Action

1. **Sudden Success Rate Drop**
   ```bash
   # Alert if success rate drops >20% in 10 iterations
   tail -10 events.ndjson | jq '
     select(.exit_code != null) | 
     {success: (.exit_code == 0)} 
   ' | jq -s 'map(select(.success)) | length / 10'
   ```

2. **Memory Leak Detection**
   ```bash
   # Check for steady memory growth
   ps aux | grep loop.sh | awk '{print $6}' # RSS in KB
   # Run hourly and compare
   ```

3. **Infinite Loop Detection**
   ```bash
   # Same target repeatedly failing
   tail -50 loop.log | grep "Target:" | uniq -c | \
     awk '$1 > 5 {print "WARNING: Stuck on", $2}'
   ```

4. **Resource Exhaustion**
   ```bash
   # Disk space monitoring
   df -h auto/data/ | awk 'NR==2 {
     use=int($5);
     if (use > 80) print "CRITICAL: Disk usage at", $5
   }'
   ```

## 📊 Metrics-Driven Optimization

### Using Metrics to Improve Prompts

1. **Identify Problem Areas**
   ```bash
   # Find most common errors
   jq 'select(.event == "error") | .details.message' events.ndjson | \
     sort | uniq -c | sort -rn | head -5
   ```

2. **Track Prompt Changes**
   ```markdown
   <!-- In prompt file -->
   <!-- METRICS_NOTE: v1.2 - Added timeout guidance after 15% timeout rate -->
   ```

3. **A/B Testing Prompts**
   ```bash
   # Run with different prompts
   PROMPT_PATH=/tmp/prompt_a.md ./task-manager.sh --task resource-improvement run-loop --max 10
   # Compare success rates
   ```

### Using Metrics for Resource Allocation

```bash
# Determine optimal worker count based on TCP connections
jq '.tcp_connections' events.ndjson | awk '
  $1 > 15 {high++} 
  END {
    if (high > 10) print "Reduce MAX_CONCURRENT_WORKERS"
    else print "Can increase MAX_CONCURRENT_WORKERS"
  }'
```

## 🎨 Visualization Commands

### Progress Over Time
```bash
# Generate CSV for graphing
echo "iteration,duration,success" > metrics.csv
jq -r '
  select(.event == "iteration_end") | 
  [.iteration, .duration, (.exit_code == 0)] | 
  @csv
' events.ndjson >> metrics.csv
```

### Success Rate Trend
```bash
# Rolling success rate (last 20 iterations)
jq -s '
  [range(20; length)] as $indices |
  $indices | map(. as $i | 
    ($data[($i-20):$i] | 
    map(select(.exit_code == 0)) | length) / 20
  )
' events.ndjson
```

### Resource Health Dashboard
```bash
# Simple text dashboard
watch -n 60 'clear
echo "=== AUTO/ SYSTEM DASHBOARD ==="
echo "Time: $(date)"
echo ""
echo "=== Active Loops ==="
ps aux | grep -E "loop.sh|task-manager" | grep -v grep
echo ""
echo "=== Recent Activity ==="
tail -5 auto/data/*/loop.log | grep -E "SUCCESS|FAILED|ERROR"
echo ""
echo "=== Metrics ==="
jq -r "\"Success Rate: \\(.success_rate)\"" auto/data/*/summary.json
'
```

## 🔮 Predictive Metrics

### Early Warning Indicators

1. **Degradation Prediction**
   - Success rate declining >5% per 10 iterations
   - Duration increasing >10% per day
   - Error rate climbing steadily

2. **Failure Prediction**
   - Same error appearing 3+ times
   - Validation gates failing repeatedly
   - Worker timeouts increasing

3. **Optimization Opportunities**
   - Consistent fast iterations (<3 min)
   - High success rate (>90%)
   - Low resource usage (<30%)

### Metric-Based Decisions

| If Metrics Show... | Then Consider... |
|-------------------|------------------|
| Success rate >80% for 50+ iterations | Increase complexity/scope |
| Duration <5 min consistently | Reduce interval between iterations |
| Memory stable for 24h | Add concurrent workers |
| All resources healthy | Start scenario-improvement |
| PRD completion >95% | Begin self-play testing |

## 📚 Best Practices

### 1. Regular Monitoring
```bash
# Add to crontab
0 * * * * /path/to/auto/generate_hourly_report.sh
```

### 2. Metric Archival
```bash
# Weekly archive
tar -czf "metrics_$(date +%Y%m%d).tar.gz" auto/data/*/events.ndjson
```

### 3. Alert Thresholds
```bash
# Simple alerting
if [[ $(jq '.success_rate' summary.json) < 0.5 ]]; then
  echo "ALERT: Success rate below 50%" | mail -s "Auto/ Alert" admin@example.com
fi
```

### 4. Continuous Improvement
- Review metrics weekly
- Update prompts based on patterns
- Adjust parameters for optimization
- Document metric-driven changes

## 🎬 Conclusion

Metrics are the compass that guides the auto/ system toward its goal of bootstrapping Vrooli's autonomous intelligence. By understanding and acting on these measurements, we ensure steady progress toward a self-improving, self-sustaining system that no longer needs the scaffolding that built it.