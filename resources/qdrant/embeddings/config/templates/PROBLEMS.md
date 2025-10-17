# Problems

This document captures ongoing issues, system failures, performance degradations, and recurring problems discovered during operation. It serves as a structured problem registry for autonomous problem detection and resolution.

## Active Issues

<!-- EMBED:ACTIVEPROBLEM:START -->
### Problem Title
**Status:** [active|investigating|blocked]
**Severity:** [critical|high|medium|low]
**Frequency:** [constant|frequent|occasional|rare]
**Impact:** [system_down|degraded_performance|user_impact|cosmetic]
**Discovered:** YYYY-MM-DD
**Discovered By:** [human|swarm-manager|scenario_name|resource_name]
**Last Occurrence:** YYYY-MM-DD HH:MM

#### Description
Clear description of the problem, including:
- What's happening vs what should happen
- When the problem occurs
- Which components are affected
- Error messages or symptoms

#### Reproduction Steps
1. Step to reproduce the issue
2. Expected vs actual behavior
3. Environmental conditions

#### Impact Assessment
- **Users Affected:** Number or percentage
- **Business Impact:** Revenue, productivity, reputation
- **Technical Impact:** Performance degradation, cascade effects
- **Urgency Factors:** Why this needs immediate attention

#### Investigation Status
- **Root Cause:** Known or suspected cause
- **Workarounds:** Temporary solutions available
- **Related Issues:** Links to similar problems
- **Attempted Solutions:** What has been tried

#### Priority Estimates
```yaml
impact: 8           # 1-10 scale
urgency: "critical" # critical|high|medium|low  
success_prob: 0.6   # 0-1 likelihood of resolution
resource_cost: "moderate" # minimal|moderate|heavy
```
<!-- EMBED:ACTIVEPROBLEM:END -->

## Intermittent Issues

<!-- EMBED:INTERMITTENTPROBLEM:START -->
### Sporadic Problem Title
**Pattern:** [time_based|load_based|random|environmental]
**Last Seen:** YYYY-MM-DD
**Frequency:** X times per [day|week|month]
**Tracking Since:** YYYY-MM-DD

#### Problem Pattern
- **Triggers:** What seems to cause the issue
- **Conditions:** Environment when it occurs
- **Duration:** How long it lasts
- **Recovery:** How it resolves

#### Detection Criteria
- **Symptoms:** How to recognize the problem
- **Monitoring:** What alerts fire
- **Logs:** Key log messages to watch for
- **Metrics:** Performance indicators

#### Impact When Active
- Service degradation description
- User experience impact
- Business consequences

#### Investigation Notes
- Hypotheses about root cause
- Data collected during occurrences
- Correlation with other events
- Planned investigation steps
<!-- EMBED:INTERMITTENTPROBLEM:END -->

## Recently Resolved

<!-- EMBED:RESOLVEDPROBLEM:START -->
### Fixed Problem Title
**Resolved:** YYYY-MM-DD
**Duration:** Days/weeks active
**Resolution:** [code_fix|configuration|infrastructure|process]
**Resolved By:** [human|swarm-manager|scenario_name]

#### Problem Summary
Brief description of what was wrong and impact.

#### Root Cause
What actually caused the problem:
- **Technical Cause:** Code bug, config error, resource limit
- **Process Cause:** Missing monitoring, inadequate testing
- **Knowledge Cause:** Misunderstanding, missing documentation

#### Solution Implemented
- **Changes Made:** Specific fixes applied
- **Validation:** How fix was verified
- **Rollback Plan:** How to undo if needed
- **Documentation:** Updates made to prevent recurrence

#### Lessons Learned
- **Prevention:** How to avoid similar issues
- **Detection:** How to catch it earlier next time
- **Response:** How to resolve faster
- **Knowledge Gaps:** What we didn't know

#### Success Metrics
- **Resolution Time:** How quickly was it fixed
- **Effectiveness:** Has it stayed fixed
- **Improvements:** Performance gains from fix
<!-- EMBED:RESOLVEDPROBLEM:END -->

## Problem Patterns

<!-- EMBED:PROBLEMPATTERN:START -->
### Pattern Name
**Category:** [performance|reliability|security|usability|integration]
**Frequency:** How often this pattern appears
**Affected Systems:** Which components show this pattern

#### Pattern Description
- **Common Symptoms:** What this pattern looks like
- **Typical Triggers:** What usually causes it
- **Evolution:** How the problem develops over time
- **Cascade Effects:** What other issues it leads to

#### Detection Strategy
- **Early Warnings:** Indicators before failure
- **Monitoring Approach:** How to watch for this pattern
- **Alert Configuration:** When to notify humans/systems
- **Automated Checks:** Scripts or tests to run

#### Resolution Strategy
- **Standard Approach:** Typical solution pattern
- **Time to Resolution:** Expected fix timeline
- **Resource Requirements:** Skills/tools needed
- **Success Rate:** How often standard approach works

#### Prevention Measures
- **Design Changes:** How to architect against this
- **Process Improvements:** Operational changes needed
- **Monitoring Enhancements:** Better detection systems
- **Training Needs:** Knowledge team should have
<!-- EMBED:PROBLEMPATTERN:END -->

## Historical Analysis

<!-- EMBED:PROBLEMANALYSIS:START -->
### Analysis Period: [timeframe]
**Problem Count:** Active: X, Resolved: Y, Total: Z
**Resolution Rate:** X% resolved within SLA
**Top Categories:** Most common problem types

#### Trending Issues
- **Increasing:** Problems getting worse/more frequent
- **Decreasing:** Problems being resolved/prevented
- **Stable:** Consistent problem areas
- **New:** Recently discovered issue types

#### System Health Indicators
- **MTTR (Mean Time to Resolution):** Average fix time
- **MTBF (Mean Time Between Failures):** Reliability measure
- **Problem Density:** Issues per component/time
- **Resolution Effectiveness:** How often fixes stick

#### Recommendations
1. **Immediate Actions:** Critical problems to address
2. **Process Improvements:** How to prevent similar issues
3. **Monitoring Gaps:** Where we need better detection
4. **Knowledge Building:** Documentation/training needed
<!-- EMBED:PROBLEMANALYSIS:END -->

## Integration Points

<!-- EMBED:PROBLEMINTEGRATION:START -->
### Swarm-Manager Integration
This problems file is automatically scanned by swarm-manager for:

#### Task Generation
- **Active Issues:** Auto-create tasks for critical/high problems
- **Pattern Recognition:** Generate prevention tasks for recurring patterns
- **Knowledge Gaps:** Create documentation/investigation tasks
- **Monitoring Improvements:** Generate tasks to improve detection

#### Priority Calculation
Problems contribute to task priority based on:
- **Severity + Frequency → Impact score**
- **User/business impact → Urgency level**
- **Investigation status → Success probability**
- **Complexity → Resource cost estimate**

#### Learning Integration
- **Resolution Patterns:** Track which solutions work
- **Prediction Models:** Learn to predict problem occurrence
- **Resource Optimization:** Understand investigation costs
- **Success Metrics:** Measure improvement over time

#### Automation Opportunities
- **Auto-Detection:** Scripts to identify new problems
- **Auto-Triage:** AI classification of severity/impact
- **Auto-Resolution:** Simple fixes applied automatically
- **Auto-Escalation:** When to involve humans
<!-- EMBED:PROBLEMINTEGRATION:END -->

## Problem Report Template

```yaml
# Quick problem report format for CLI/API
title: "Brief problem description"
severity: critical|high|medium|low
frequency: constant|frequent|occasional|rare
impact: system_down|degraded_performance|user_impact|cosmetic
discovered_by: swarm-manager
occurred_at: "2024-01-15T10:30:00Z"
affected_components:
  - resource: n8n
  - scenario: webhook-processor
symptoms:
  - "Webhook timeouts increasing"
  - "Error rate >5% in last hour"
evidence:
  - log_snippet: "Connection timeout after 30s"
  - metric: "response_time_p95: 45s"
related_issues: []
priority_estimates:
  impact: 9
  urgency: "critical"
  success_prob: 0.8
  resource_cost: "moderate"
```

## Monitoring Integration

### Problem Detection Alerts
```yaml
# Example alert configurations
webhook_failure_rate:
  condition: "error_rate > 0.05"
  duration: "5m"
  severity: "high"
  action: "create_problem_report"

resource_exhaustion:
  condition: "cpu_usage > 0.9 OR memory_usage > 0.9"
  duration: "10m"
  severity: "critical"
  action: "create_problem_report"

service_unavailable:
  condition: "health_check_failed"
  duration: "1m"
  severity: "critical"
  action: "create_problem_report"
```

### Automated Problem Reports
The system can automatically create problem entries when:
- Alert thresholds are exceeded
- Error rates spike above baseline
- Performance metrics degrade significantly
- Health checks fail consistently
- User complaint patterns detected

---

**Usage Notes:**
- Update problems in real-time as status changes
- Use embedding markers for semantic search
- Include priority estimates for swarm-manager integration  
- Link related problems for pattern recognition
- Maintain historical data for trend analysis

**Integration:** This document is automatically scanned by swarm-manager every 15 minutes to generate tasks for problem resolution and prevention.