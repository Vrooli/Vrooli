# Problems

This is a test PROBLEMS.md file to validate the problem scanning functionality.

## Active Issues

<!-- EMBED:ACTIVEPROBLEM:START -->
### N8N Webhook Timeout Issues
**Status:** [active]
**Severity:** [high]
**Frequency:** [frequent]
**Impact:** [degraded_performance]
**Discovered:** 2025-01-28
**Discovered By:** [system-monitor]
**Last Occurrence:** 2025-01-28 10:30

#### Description
Webhooks timing out after 30 seconds, causing workflow failures. Error rate has increased to >5% in the last 24 hours.

#### Reproduction Steps
1. Send webhook to N8N endpoint
2. Observe timeout after 30 seconds
3. Check N8N logs for connection errors

#### Impact Assessment
- **Users Affected:** All webhook-dependent workflows
- **Business Impact:** Automated processes failing
- **Technical Impact:** 5% error rate on critical workflows
- **Urgency Factors:** Blocking automation workflows

#### Investigation Status
- **Root Cause:** Unknown - requires investigation
- **Workarounds:** Manual retrigger of failed workflows
- **Related Issues:** None identified
- **Attempted Solutions:** None yet

#### Priority Estimates
```yaml
impact: 8           
urgency: "high"
success_prob: 0.8   
resource_cost: "moderate"
```
<!-- EMBED:ACTIVEPROBLEM:END -->

<!-- EMBED:ACTIVEPROBLEM:START -->
### Claude Code Rate Limiting Issues
**Status:** [active]
**Severity:** [critical]
**Frequency:** [occasional]
**Impact:** [system_down]
**Discovered:** 2025-01-28
**Discovered By:** [swarm-manager]
**Last Occurrence:** 2025-01-28 09:15

#### Description
Claude Code API calls hitting rate limits during peak usage, causing complete system halt for task execution.

#### Reproduction Steps
1. Execute multiple concurrent tasks
2. Observe 429 rate limit errors
3. System stops processing new tasks

#### Impact Assessment
- **Users Affected:** All automated task processing
- **Business Impact:** Complete system shutdown
- **Technical Impact:** No task execution possible
- **Urgency Factors:** Critical system functionality blocked

#### Investigation Status
- **Root Cause:** Suspected rate limit configuration issue
- **Workarounds:** Wait for rate limit reset
- **Related Issues:** None
- **Attempted Solutions:** None yet

#### Priority Estimates
```yaml
impact: 10           
urgency: "critical"
success_prob: 0.9   
resource_cost: "minimal"
```
<!-- EMBED:ACTIVEPROBLEM:END -->

<!-- EMBED:ACTIVEPROBLEM:START -->
### UI Styling Issues on Mobile
**Status:** [active]
**Severity:** [low]
**Frequency:** [constant]
**Impact:** [cosmetic]
**Discovered:** 2025-01-27
**Discovered By:** [human]
**Last Occurrence:** 2025-01-28 10:00

#### Description
Swarm Manager UI components not properly responsive on mobile devices. Some buttons are cut off and text overlaps.

#### Priority Estimates
```yaml
impact: 3           
urgency: "low"
success_prob: 0.95   
resource_cost: "minimal"
```
<!-- EMBED:ACTIVEPROBLEM:END -->

## Integration Points

<!-- EMBED:PROBLEMINTEGRATION:START -->
### Swarm-Manager Integration
This problems file is automatically scanned by swarm-manager for task generation and priority calculation.
<!-- EMBED:PROBLEMINTEGRATION:END -->