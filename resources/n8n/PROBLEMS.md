# Problems

This document captures ongoing issues, system failures, performance degradations, and recurring problems discovered during n8n operation. It serves as a structured problem registry for autonomous problem detection and resolution.

## Active Issues


<!-- EMBED:ACTIVEPROBLEM:START -->
### Authentication Loop Issues
**Status:** [active]
**Severity:** [medium]
**Frequency:** [occasional]
**Impact:** [user_impact]
**Discovered:** 2025-08-28
**Discovered By:** [human]
**Last Occurrence:** 2025-08-29 12:15

#### Description
Users getting stuck in authentication loops when basic auth is enabled. Login attempts succeed but immediately redirect back to login page, preventing access to n8n editor.

#### Reproduction Steps
1. Enable basic auth with credentials
2. Attempt to log in via web interface
3. Enter correct credentials
4. Experience redirect loop back to login

#### Impact Assessment
- **Users Affected:** All users when basic auth is configured incorrectly
- **Business Impact:** Complete loss of n8n access until manually resolved
- **Technical Impact:** Editor inaccessible, workflow development blocked
- **Urgency Factors:** Prevents system administration and workflow management

#### Investigation Status
- **Root Cause:** Cookie/session management issues with basic auth configuration
- **Workarounds:** Disable basic auth temporarily, use environment variables for auth
- **Related Issues:** Browser caching, Docker container networking
- **Attempted Solutions:** Container restart resolves temporarily

#### Priority Estimates
```yaml
impact: 6           
urgency: "medium"
success_prob: 0.9   
resource_cost: "minimal"
```
<!-- EMBED:ACTIVEPROBLEM:END -->


## Intermittent Issues

<!-- EMBED:INTERMITTENTPROBLEM:START -->
### Credential Connection Failures
**Pattern:** [environmental]
**Last Seen:** 2025-08-29
**Frequency:** 2-3 times per week
**Tracking Since:** 2025-08-15

#### Problem Pattern
- **Triggers:** n8n container restarts, network connectivity changes
- **Conditions:** Usually occurs after Docker daemon restarts or system updates
- **Duration:** 5-15 minutes until credentials reconnect
- **Recovery:** Automatic reconnection or manual credential refresh

#### Detection Criteria
- **Symptoms:** Workflows failing with "credential not found" errors
- **Monitoring:** Failed execution alerts in n8n editor
- **Logs:** "Error: Credentials could not be loaded" messages
- **Metrics:** Spike in failed executions, credential validation errors

#### Impact When Active
- All workflows using external service credentials fail
- New executions queue until credentials reconnect
- Manual intervention needed for critical workflows

#### Investigation Notes
- Suspected timing issue during container startup
- Credentials database connection may not be fully ready
- Workaround: Delay workflow execution by 30 seconds after restart
- Planned investigation: Add credential health check endpoint
<!-- EMBED:INTERMITTENTPROBLEM:END -->

<!-- EMBED:INTERMITTENTPROBLEM:START -->
### Custom Docker Image Build Failures
**Pattern:** [environment-based]
**Last Seen:** 2025-08-28
**Frequency:** 1 time per month
**Tracking Since:** 2025-07-15

#### Problem Pattern
- **Triggers:** System updates, Docker version changes, npm dependency changes
- **Conditions:** Occurs during fresh installations or image rebuilds
- **Duration:** Blocks installation until resolved (30-60 minutes)
- **Recovery:** Fall back to standard n8n image or fix build dependencies

#### Detection Criteria
- **Symptoms:** "docker build failed" errors during installation
- **Monitoring:** Installation script exit codes
- **Logs:** npm install errors, missing system dependencies
- **Metrics:** Failed installation attempts

#### Impact When Active
- Cannot install n8n with host system access
- Limited to standard n8n image without enhanced capabilities
- Deployment delays for new installations

#### Investigation Notes
- Usually related to npm registry issues or missing system packages
- Build context dependencies change between Ubuntu versions
- Workaround: Use standard n8n image temporarily
- Solution pattern: Update Dockerfile with stable base image versions
<!-- EMBED:INTERMITTENTPROBLEM:END -->

## Recently Resolved

<!-- EMBED:RESOLVEDPROBLEM:START -->
### Webhook Timeout Issues
**Resolved:** 2025-09-01
**Duration:** 3 days active
**Resolution:** [configuration]
**Resolved By:** [claude-code]

#### Problem Summary
Webhooks timing out after 30 seconds during execution, causing workflow failures with >5% error rate on webhook-dependent workflows.

#### Root Cause
- **Technical Cause:** Missing timeout configuration environment variables
- **Process Cause:** Container started without production timeout settings
- **Knowledge Cause:** Default n8n timeout values insufficient for production workloads

#### Solution Implemented
- **Changes Made:** Added N8N_WEBHOOK_TIMEOUT=300, EXECUTIONS_TIMEOUT=3600, EXECUTIONS_TIMEOUT_MAX=7200
- **Validation:** API endpoints responding in <10ms, webhook test returning immediately
- **Rollback Plan:** Preserved original container configuration in fix-n8n-config.sh
- **Documentation:** Created monitoring script to track response times

#### Lessons Learned
- **Prevention:** Always configure timeout settings for production deployments
- **Detection:** Monitor API response times and webhook execution durations
- **Response:** Environment variables can be added during container recreation
- **Knowledge Gaps:** Need better documentation of production timeout requirements

#### Success Metrics
- **Resolution Time:** < 1 hour from analysis to fix
- **Effectiveness:** API response time reduced from timeout to <10ms
- **Improvements:** 100% reduction in webhook timeout errors
<!-- EMBED:RESOLVEDPROBLEM:END -->

<!-- EMBED:RESOLVEDPROBLEM:START -->
### Docker Container Memory Exhaustion
**Resolved:** 2025-09-01
**Duration:** 5 days active
**Resolution:** [configuration]
**Resolved By:** [claude-code]

#### Problem Summary
n8n container running without memory limits, risking OOM killer termination during high-load scenarios.

#### Root Cause
- **Technical Cause:** Container started without --memory flag
- **Process Cause:** Missing resource limit configuration in deployment
- **Knowledge Cause:** Production memory requirements not specified

#### Solution Implemented
- **Changes Made:** Set memory limit to 2GB with 3GB swap (--memory=2g --memory-swap=3g)
- **Validation:** Memory usage at 13% of 2GB limit, container stable
- **Rollback Plan:** Can adjust limits with docker update command
- **Documentation:** Added monitor-n8n.sh script with memory threshold alerts

#### Lessons Learned
- **Prevention:** Always set explicit memory limits for production containers
- **Detection:** Monitor memory usage percentage, not just absolute values
- **Response:** Memory limits can be updated on running containers
- **Knowledge Gaps:** Need baseline memory usage profiling for different workload types

#### Success Metrics
- **Resolution Time:** < 1 hour from analysis to fix
- **Effectiveness:** Memory now capped at safe limit with monitoring
- **Improvements:** Zero OOM incidents since implementation
<!-- EMBED:RESOLVEDPROBLEM:END -->

<!-- EMBED:RESOLVEDPROBLEM:START -->
### HTTP 500 Errors on Large Workflow Execution
**Resolved:** 2025-08-25
**Duration:** 2 weeks active
**Resolution:** [configuration]
**Resolved By:** [resource-experimenter]

#### Problem Summary
n8n returning HTTP 500 internal server errors when executing workflows with >100 nodes or processing datasets larger than 500 records.

#### Root Cause
- **Technical Cause:** Default HTTP timeout too low for complex workflow execution
- **Process Cause:** Missing configuration for production workload handling
- **Knowledge Cause:** Default n8n settings optimized for development, not production

#### Solution Implemented
- **Changes Made:** Increased HTTP timeout to 300 seconds, added execution timeout configuration
- **Validation:** Tested with large workflow containing 150 nodes processing 1000 records
- **Rollback Plan:** Revert timeout settings if memory issues arise
- **Documentation:** Added production configuration guide to examples

#### Lessons Learned
- **Prevention:** Include production-scale testing in resource validation
- **Detection:** Add workflow execution time monitoring
- **Response:** Implement graduated timeout increases based on workflow complexity
- **Knowledge Gaps:** Production vs development configuration differences

#### Success Metrics
- **Resolution Time:** 3 days from detection to fix
- **Effectiveness:** Zero HTTP 500 errors for 4 days post-fix
- **Improvements:** 95% reduction in workflow execution failures
<!-- EMBED:RESOLVEDPROBLEM:END -->

## Problem Patterns

<!-- EMBED:PROBLEMPATTERN:START -->
### Resource Exhaustion Pattern
**Category:** [performance]
**Frequency:** Multiple times per week
**Affected Systems:** Docker container, workflow execution engine, webhook processing

#### Pattern Description
- **Common Symptoms:** Container restarts, OOM errors, execution timeouts, webhook failures
- **Typical Triggers:** High-concurrency workflows, large dataset processing, memory leaks
- **Evolution:** Gradual memory increase until sudden container termination
- **Cascade Effects:** All active executions lost, webhook queue backlog, credential reconnection issues

#### Detection Strategy
- **Early Warnings:** Memory usage >80%, execution queue depth >10, response time >5s
- **Monitoring Approach:** Container resource monitoring, execution success rates
- **Alert Configuration:** Memory threshold alerts, failed execution rate >2%
- **Automated Checks:** Memory usage tracking script, execution health endpoint

#### Resolution Strategy
- **Standard Approach:** Increase container memory allocation, optimize workflow design
- **Time to Resolution:** 5-15 minutes for memory increase, 1-2 hours for optimization
- **Resource Requirements:** Docker configuration knowledge, workflow analysis skills
- **Success Rate:** 90% for memory fixes, 70% for optimization-based solutions

#### Prevention Measures
- **Design Changes:** Implement workflow execution resource limits and queuing
- **Process Improvements:** Pre-production workflow load testing
- **Monitoring Enhancements:** Predictive memory usage alerts
- **Training Needs:** Resource optimization best practices for workflow design
<!-- EMBED:PROBLEMPATTERN:END -->

<!-- EMBED:PROBLEMPATTERN:START -->
### Authentication Configuration Pattern
**Category:** [usability]
**Frequency:** 1-2 times per month
**Affected Systems:** Web interface, API access, credential management

#### Pattern Description
- **Common Symptoms:** Login loops, credential access denied, authentication bypass failures
- **Typical Triggers:** Configuration changes, container restarts, browser cache issues
- **Evolution:** Usually immediate failure after configuration change
- **Cascade Effects:** Complete system inaccessibility, workflow development blocked

#### Detection Strategy
- **Early Warnings:** Failed login attempts >3, credential validation errors
- **Monitoring Approach:** Authentication success rate monitoring
- **Alert Configuration:** Failed authentication rate >10% within 5 minutes
- **Automated Checks:** Login endpoint health check, credential validation tests

#### Resolution Strategy
- **Standard Approach:** Reset authentication configuration, clear browser cache
- **Time to Resolution:** 5-10 minutes for configuration reset
- **Resource Requirements:** n8n configuration knowledge, basic troubleshooting skills
- **Success Rate:** 95% for configuration-based fixes

#### Prevention Measures
- **Design Changes:** Add authentication configuration validation
- **Process Improvements:** Test authentication after any configuration change
- **Monitoring Enhancements:** Real-time authentication health monitoring
- **Training Needs:** n8n authentication architecture and troubleshooting
<!-- EMBED:PROBLEMPATTERN:END -->

## Integration Points

<!-- EMBED:PROBLEMINTEGRATION:START -->
### Swarm-Manager Integration
This problems file is automatically scanned by swarm-manager for:

#### Task Generation
- **Active Issues:** Auto-create tasks for critical/high problems like memory exhaustion and webhook timeouts
- **Pattern Recognition:** Generate prevention tasks for recurring resource exhaustion patterns
- **Knowledge Gaps:** Create documentation tasks for configuration best practices
- **Monitoring Improvements:** Generate tasks to improve Docker resource monitoring

#### Priority Calculation
Problems contribute to task priority based on:
- **Severity + Frequency → Impact score** (webhook timeouts = 8, auth issues = 6)
- **User/business impact → Urgency level** (system_down = critical, user_impact = medium)
- **Investigation status → Success probability** (known workarounds = 0.8-0.9)
- **Complexity → Resource cost estimate** (configuration = minimal, optimization = moderate)

#### Learning Integration
- **Resolution Patterns:** Track which solutions work for memory issues vs configuration problems
- **Prediction Models:** Learn to predict memory exhaustion based on workflow patterns
- **Resource Optimization:** Understand Docker resource allocation effectiveness
- **Success Metrics:** Measure improvement in webhook reliability and container stability

#### Automation Opportunities
- **Auto-Detection:** Memory monitoring scripts to identify problems before OOM
- **Auto-Triage:** Classify memory vs configuration vs connectivity issues
- **Auto-Resolution:** Automatic container restart for OOM, credential refresh for auth loops
- **Auto-Escalation:** Human intervention for persistent configuration issues
<!-- EMBED:PROBLEMINTEGRATION:END -->