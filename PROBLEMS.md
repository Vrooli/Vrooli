# Problems - Vrooli System-Wide Issues

This document captures cross-cutting issues, system-wide failures, and problems that affect multiple resources or scenarios. For resource-specific issues, see the PROBLEMS.md file in each resource/scenario directory.

## Active Issues

<!-- EMBED:ACTIVEPROBLEM:START -->
### Concurrent Claude Code Sessions Causing Rate Limits
**Status:** [active]
**Severity:** [critical]
**Frequency:** [frequent]
**Impact:** [degraded_performance]
**Discovered:** 2025-01-28
**Discovered By:** [swarm-manager]
**Last Occurrence:** 2025-01-28 18:30

#### Description
When multiple scenarios run Claude Code sessions concurrently, the system hits API rate limits within 30-45 minutes, causing all AI-driven automation to halt. This affects the entire swarm-manager orchestration capability.

#### Reproduction Steps
1. Start swarm-manager with 3+ active tasks
2. Each task initiates a Claude Code session
3. Monitor API usage for 30 minutes
4. Observe rate limit errors (HTTP 429) blocking all requests

#### Impact Assessment
- **Users Affected:** All automated scenarios using Claude Code (90% of automation)
- **Business Impact:** Complete automation freeze, manual intervention required
- **Technical Impact:** Task backlog accumulation, scenario failures cascade
- **Urgency Factors:** Blocks primary value proposition of autonomous operation

#### Investigation Status
- **Root Cause:** No request queuing or rate limit management across scenarios
- **Workarounds:** Manually space out task execution, reduce concurrent tasks to 2
- **Related Issues:** Session pooling not implemented, token usage not optimized
- **Attempted Solutions:** Reduced concurrent tasks in config (partial mitigation)

#### Priority Estimates
```yaml
impact: 9
urgency: "critical"
success_prob: 0.8
resource_cost: "moderate"
```

#### Evidence
```yaml
error_messages:
  - "Rate limit exceeded. Please try again later."
  - "HTTP 429: Too Many Requests"
metrics:
  api_calls_per_hour: "150+"
  concurrent_sessions: "5-8"
  failure_rate: "40%"
logs:
  - "2025-01-28 18:30:00 ERROR: Claude API rate limit hit"
  - "2025-01-28 18:30:01 ERROR: All 5 active tasks blocked"
```

#### Proposed Solution
Implement a global Claude Code session manager that queues requests and enforces rate limits across all scenarios.

#### Tags
[#performance] [#reliability] [#integration] [#critical]

<!-- EMBED:ACTIVEPROBLEM:END -->

<!-- EMBED:ACTIVEPROBLEM:START -->
### Resource Discovery Inconsistencies
**Status:** [active]
**Severity:** [medium]
**Frequency:** [occasional]
**Impact:** [user_impact]
**Discovered:** 2025-01-26
**Discovered By:** [human]
**Last Occurrence:** 2025-01-28 14:15

#### Description
The `vrooli resource` command sometimes fails to discover installed resources, showing different results between invocations. Resources appear and disappear from the list randomly.

#### Reproduction Steps
1. Run `vrooli resource list`
2. Note the resources shown
3. Wait 5 seconds and run again
4. Observe different resource list

#### Impact Assessment
- **Users Affected:** All users managing resources via CLI
- **Business Impact:** Confusion about system state, unreliable resource management
- **Technical Impact:** Resource commands may fail unexpectedly
- **Urgency Factors:** Affects basic resource management operations

#### Investigation Status
- **Root Cause:** Suspected race condition in resource discovery script
- **Workarounds:** Run command multiple times until desired resource appears
- **Related Issues:** Docker container state detection issues
- **Attempted Solutions:** Added retry logic (didn't fully resolve)

#### Priority Estimates
```yaml
impact: 5
urgency: "medium"
success_prob: 0.9
resource_cost: "minimal"
```

#### Evidence
```yaml
error_messages:
  - "Resource not found: postgres"
  - "No such resource: n8n"
metrics:
  inconsistency_rate: "15%"
  affected_commands: "list, status, start, stop"
```

#### Proposed Solution
Refactor resource discovery to use consistent Docker API queries with proper state locking.

#### Tags
[#reliability] [#ux] [#cli]

<!-- EMBED:ACTIVEPROBLEM:END -->

<!-- EMBED:ACTIVEPROBLEM:START -->
### Inter-Resource Communication Failures
**Status:** [active]
**Severity:** [high]
**Frequency:** [occasional]
**Impact:** [degraded_performance]
**Discovered:** 2025-01-25
**Discovered By:** [system-monitor]
**Last Occurrence:** 2025-01-28 16:00

#### Description
Resources fail to communicate with each other intermittently due to Docker network issues. N8N can't reach Postgres, Qdrant can't be accessed by scenarios, etc.

#### Reproduction Steps
1. Start all core resources (postgres, redis, n8n, qdrant)
2. Run integration tests
3. Observe random connection failures between services
4. Docker network inspect shows containers on different networks

#### Impact Assessment
- **Users Affected:** All integration scenarios and workflows
- **Business Impact:** Workflows fail unpredictably, data consistency issues
- **Technical Impact:** Integration tests fail, scenarios can't access required services
- **Urgency Factors:** Undermines system reliability

#### Investigation Status
- **Root Cause:** Docker network configuration not consistent across resources
- **Workarounds:** Manually restart Docker daemon, recreate networks
- **Related Issues:** Container DNS resolution problems
- **Attempted Solutions:** Added network creation to resource scripts (partial fix)

#### Priority Estimates
```yaml
impact: 7
urgency: "high"
success_prob: 0.7
resource_cost: "moderate"
```

#### Evidence
```yaml
error_messages:
  - "Connection refused: postgres:5432"
  - "getaddrinfo ENOTFOUND n8n"
  - "Network vrooli_default not found"
metrics:
  failure_rate: "8%"
  mttr: "15 minutes"
logs:
  - "2025-01-28 16:00:00 ERROR: Failed to connect to postgres from n8n"
  - "2025-01-28 16:00:05 INFO: Network recreation resolved issue"
```

#### Proposed Solution
Create a unified Docker network management system that ensures all resources join the same network with consistent DNS.

#### Tags
[#infrastructure] [#reliability] [#integration]

<!-- EMBED:ACTIVEPROBLEM:END -->

## Resolved Issues

<!-- Move resolved problems here with resolution details -->

## Known Limitations

### Scenario Concurrency Limits
- Maximum 5 concurrent scenarios due to system resource constraints
- Claude Code API limited to 100 calls per hour
- Docker Desktop memory limited to 8GB by default

### Platform-Specific Issues
- **macOS**: File watching may miss changes in Docker volumes
- **Linux**: SELinux may block container communications
- **Windows**: WSL2 networking can cause intermittent failures

## Integration Points

<!-- EMBED:PROBLEMINTEGRATION:START -->
### Swarm-Manager Integration
This global problems file is automatically scanned by swarm-manager every 5 minutes to identify system-wide issues that need attention. Problems marked as critical with active status trigger immediate task creation.

### Monitoring Integration
System-monitor scenario checks this file for critical issues and can trigger alerts or automated remediation based on problem severity.
<!-- EMBED:PROBLEMINTEGRATION:END -->

## Problem Reporting Guidelines

For system-wide issues:
1. Add to this file if the problem affects multiple resources/scenarios
2. Add to specific resource/scenario PROBLEMS.md if localized
3. Use the template from PROBLEMS_TEMPLATE.md
4. Update status when investigating or resolved
5. Move to Resolved section when fixed

## See Also

- [PROBLEMS_TEMPLATE.md](./PROBLEMS_TEMPLATE.md) - Template for creating problem entries
- [Swarm-Manager README](./scenarios/swarm-manager/README.md) - Autonomous task orchestration
- Individual resource PROBLEMS.md files in `/resources/*/PROBLEMS.md`
- Individual scenario PROBLEMS.md files in `/scenarios/*/PROBLEMS.md`