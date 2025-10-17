# PROBLEMS.md Template

This template defines the standard format for reporting problems in Vrooli resources and scenarios. Copy this template to create a PROBLEMS.md file in your resource or scenario directory.

## Purpose

The PROBLEMS.md file serves as a structured problem registry that:
- Captures ongoing issues, system failures, and performance degradations
- Provides semantic markers for automated problem discovery
- Enables the swarm-manager to automatically create tasks for problem resolution
- Maintains a historical record of issues and their resolutions

## File Structure

```markdown
# Problems

This document captures ongoing issues, system failures, performance degradations, and recurring problems discovered during [RESOURCE/SCENARIO NAME] operation. It serves as a structured problem registry for autonomous problem detection and resolution.

## Active Issues

[Place active problems here using the template below]

## Resolved Issues

[Move resolved problems here with resolution details]

## Known Limitations

[Document inherent limitations that aren't fixable]
```

## Problem Entry Template

Copy and paste this template for each problem. The embedded markers are REQUIRED for automated scanning.

```markdown
<!-- EMBED:ACTIVEPROBLEM:START -->
### [Problem Title - Clear, Specific, Searchable]
**Status:** [active|investigating|resolved|won't-fix]
**Severity:** [critical|high|medium|low]
**Frequency:** [constant|frequent|occasional|rare]
**Impact:** [system_down|degraded_performance|user_impact|cosmetic]
**Discovered:** YYYY-MM-DD
**Discovered By:** [system-monitor|swarm-manager|human|scenario-name]
**Last Occurrence:** YYYY-MM-DD HH:MM

#### Description
[2-3 sentences describing the problem clearly. Include what's happening, when it happens, and what the expected behavior should be.]

#### Reproduction Steps
1. [First step to reproduce]
2. [Second step]
3. [Observable result]
4. [Expected result]

#### Impact Assessment
- **Users Affected:** [Who/what is impacted - be specific]
- **Business Impact:** [How this affects operations/productivity]
- **Technical Impact:** [System/performance implications]
- **Urgency Factors:** [Why this needs attention now/later]

#### Investigation Status
- **Root Cause:** [Known/Suspected/Unknown - provide details]
- **Workarounds:** [Temporary solutions users can apply]
- **Related Issues:** [Links to related problems or dependencies]
- **Attempted Solutions:** [What has been tried and results]

#### Priority Estimates
```yaml
impact: [1-10]           # How severely does this affect the system? (10=complete failure)
urgency: "[critical|high|medium|low]"  # How quickly must this be resolved?
success_prob: [0.0-1.0]  # Likelihood of successful resolution (1.0=certain)
resource_cost: "[minimal|moderate|heavy]"  # Expected effort to fix
```

#### Evidence
```yaml
error_messages: 
  - "Exact error message text"
  - "Another error message"
metrics:
  error_rate: "5%"
  response_time: "45s"
  affected_requests: "150/hour"
logs:
  - "2025-01-15 10:30:00 ERROR: Connection timeout after 30s"
  - "2025-01-15 10:30:01 WARN: Retry attempt 1 of 3"
```

#### Proposed Solution
[If known, describe the fix approach. This helps the swarm-manager create appropriate tasks.]

#### Tags
[#performance] [#reliability] [#integration] [#security] [#ux]

<!-- EMBED:ACTIVEPROBLEM:END -->
```

## Severity Guidelines

- **Critical**: System completely unusable, data loss risk, security breach
- **High**: Major functionality broken, significant user impact, frequent failures
- **Medium**: Feature degraded, workaround available, occasional failures
- **Low**: Minor issues, cosmetic problems, rare occurrences

## Frequency Definitions

- **Constant**: Always present, 100% reproducible
- **Frequent**: Multiple times per day, >50% reproducible
- **Occasional**: Weekly occurrences, 10-50% reproducible
- **Rare**: Monthly or less frequent, <10% reproducible

## Impact Categories

- **system_down**: Complete service unavailability
- **degraded_performance**: Slow response, reduced capacity
- **user_impact**: UX degradation without system failure
- **cosmetic**: Visual/formatting issues only

## Priority Calculation

The swarm-manager calculates priority using:
```
Priority = (Impact × Urgency × Success_Probability) / (Resource_Cost × Cooldown_Factor)
```

## Best Practices

1. **Be Specific**: Use exact error messages, timestamps, and metrics
2. **Update Regularly**: Keep status and last occurrence current
3. **Include Evidence**: Add logs, metrics, and error messages
4. **Link Related Issues**: Reference dependencies and related problems
5. **Document Workarounds**: Help users continue working while fixes are developed
6. **Move Resolved Issues**: Keep active issues visible, archive resolved ones

## Integration with Swarm-Manager

The swarm-manager automatically:
1. Scans for PROBLEMS.md files every 5 minutes
2. Parses problems using the embedded markers
3. Creates tasks for high/critical problems (in YOLO mode)
4. Tracks problem resolution status
5. Adjusts task priorities based on problem severity

## Example Scenarios

### Example 1: Performance Issue
```markdown
<!-- EMBED:ACTIVEPROBLEM:START -->
### Database Query Timeout on Large Datasets
**Status:** [active]
**Severity:** [high]
**Frequency:** [frequent]
**Impact:** [degraded_performance]
**Discovered:** 2025-01-15
**Discovered By:** [system-monitor]
**Last Occurrence:** 2025-01-15 14:30

#### Description
Database queries timeout when processing datasets over 10GB. Query execution exceeds the 30-second timeout limit, causing request failures.

#### Priority Estimates
```yaml
impact: 8
urgency: "high"
success_prob: 0.8
resource_cost: "moderate"
```
<!-- EMBED:ACTIVEPROBLEM:END -->
```

### Example 2: Integration Failure
```markdown
<!-- EMBED:ACTIVEPROBLEM:START -->
### OAuth Token Refresh Failing with External API
**Status:** [investigating]
**Severity:** [critical]
**Frequency:** [constant]
**Impact:** [system_down]
**Discovered:** 2025-01-15
**Discovered By:** [human]
**Last Occurrence:** 2025-01-15 16:45

#### Description
OAuth token refresh with ExternalAPI v2 failing with 401 errors. All authenticated requests fail after initial token expiry.

#### Priority Estimates
```yaml
impact: 10
urgency: "critical"
success_prob: 0.7
resource_cost: "moderate"
```
<!-- EMBED:ACTIVEPROBLEM:END -->
```

## Automation Triggers

Problems with these characteristics trigger immediate action:
- Severity: critical AND Status: active → Immediate task creation
- Impact: system_down → Alert sent to monitoring
- Frequency: constant AND Severity: high → Escalation to human review

## See Also

- [Swarm-Manager README](/scenarios/swarm-manager/README.md)
- [V2.0 Resource Contract](/scripts/resources/contracts/v2.0/universal.yaml)
- [Troubleshooting Guide](./TROUBLESHOOTING.md)