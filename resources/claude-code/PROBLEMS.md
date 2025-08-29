# Problems

This document captures ongoing issues, system failures, performance degradations, and recurring problems discovered during Claude Code operation. It serves as a structured problem registry for autonomous problem detection and resolution.

## Active Issues

<!-- EMBED:ACTIVEPROBLEM:START -->
### API Rate Limiting During High Automation
**Status:** [active]
**Severity:** [critical]
**Frequency:** [frequent]
**Impact:** [system_down]
**Discovered:** 2025-08-28
**Discovered By:** [swarm-manager]
**Last Occurrence:** 2025-08-29 14:15

#### Description
Claude Code API hitting rate limits during peak automation periods, causing complete halt of automated task execution. Rate limit errors (429 responses) occurring every 2-3 hours during active development cycles, blocking all AI-driven scenarios.

#### Reproduction Steps
1. Execute multiple concurrent tasks via swarm-manager
2. Each task initiates Claude Code sessions with high token usage
3. API rate limit reached within 30-45 minutes
4. All subsequent requests fail with "Rate limit exceeded" errors

#### Impact Assessment
- **Users Affected:** All automated scenarios dependent on Claude Code (100% of swarm-manager tasks)
- **Business Impact:** Complete automation halt, manual intervention required for all tasks
- **Technical Impact:** Task queue backs up, scenario orchestration fails, development productivity drops 90%
- **Urgency Factors:** Blocks entire autonomous operation mode, affects all client project work

#### Investigation Status
- **Root Cause:** Default rate limits insufficient for production automation workloads
- **Workarounds:** Manual task spacing, reduced concurrent executions
- **Related Issues:** Token usage optimization, session management efficiency
- **Attempted Solutions:** API key rotation, request batching (limited improvement)

#### Priority Estimates
```yaml
impact: 10           
urgency: "critical"
success_prob: 0.8   
resource_cost: "moderate"
```
<!-- EMBED:ACTIVEPROBLEM:END -->

<!-- EMBED:ACTIVEPROBLEM:START -->
### Session Authentication Failures
**Status:** [active]
**Severity:** [high]
**Frequency:** [occasional]
**Impact:** [degraded_performance]
**Discovered:** 2025-08-26
**Discovered By:** [system-monitor]
**Last Occurrence:** 2025-08-29 10:30

#### Description
Claude Code sessions randomly failing authentication mid-execution, requiring session restart and losing conversation context. Authentication token appears to expire or become invalid during long-running tasks.

#### Reproduction Steps
1. Start Claude Code session for complex task (>30 minutes duration)
2. Session begins normally with successful authentication
3. After 20-40 minutes, requests start failing with auth errors
4. Session must be terminated and restarted, losing all context

#### Impact Assessment
- **Users Affected:** Long-running tasks and complex multi-step scenarios (40% of tasks)
- **Business Impact:** Task execution failures, lost progress, increased completion time
- **Technical Impact:** Context loss reduces AI effectiveness, requires manual intervention
- **Urgency Factors:** Affects reliability of autonomous task execution

#### Investigation Status
- **Root Cause:** Token expiration handling not properly implemented in session management
- **Workarounds:** Shorter task decomposition, periodic session refresh
- **Related Issues:** Session state persistence, context management
- **Attempted Solutions:** Token refresh logic implementation (testing in progress)

#### Priority Estimates
```yaml
impact: 8           
urgency: "high"
success_prob: 0.9   
resource_cost: "moderate"
```
<!-- EMBED:ACTIVEPROBLEM:END -->

<!-- EMBED:ACTIVEPROBLEM:START -->
### Memory Usage Spikes During Large Context Tasks
**Status:** [active]
**Severity:** [high]
**Frequency:** [frequent]
**Impact:** [degraded_performance]
**Discovered:** 2025-08-24
**Discovered By:** [system-monitor]
**Last Occurrence:** 2025-08-29 16:20

#### Description
Claude Code resource consuming excessive system memory (>8GB) during tasks involving large codebases or extensive file analysis. Memory usage continues growing throughout session, eventually causing system performance degradation.

#### Reproduction Steps
1. Execute task involving large codebase analysis (>500 files)
2. Claude Code loads multiple files into context simultaneously
3. Memory usage climbs steadily throughout task execution
4. System becomes unresponsive when memory usage exceeds available RAM

#### Impact Assessment
- **Users Affected:** Tasks involving large codebases or extensive file processing (30% of tasks)
- **Business Impact:** System slowdowns affect all concurrent operations, task failures
- **Technical Impact:** Memory exhaustion causes system instability, requires restarts
- **Urgency Factors:** System-wide impact affects all running resources and scenarios

#### Investigation Status
- **Root Cause:** Context accumulation without proper memory management and cleanup
- **Workarounds:** Smaller batch sizes, periodic session restarts
- **Related Issues:** Context window optimization, garbage collection
- **Attempted Solutions:** Context chunking implementation (partial improvement)

#### Priority Estimates
```yaml
impact: 9           
urgency: "high"
success_prob: 0.7   
resource_cost: "moderate"
```
<!-- EMBED:ACTIVEPROBLEM:END -->

## Intermittent Issues

<!-- EMBED:INTERMITTENTPROBLEM:START -->
### Installation Node.js Version Conflicts
**Pattern:** [environmental]
**Last Seen:** 2025-08-27
**Frequency:** 1-2 times per month
**Tracking Since:** 2025-07-15

#### Problem Pattern
- **Triggers:** System updates, Node.js version changes, npm cache issues
- **Conditions:** More common on systems with multiple Node.js versions or version managers
- **Duration:** 15-45 minutes until correct Node.js version installed
- **Recovery:** Node.js reinstallation or version switching

#### Detection Criteria
- **Symptoms:** "Incompatible Node.js version" errors during installation
- **Monitoring:** Installation failure logs, version compatibility checks
- **Logs:** npm install failures, Node.js version mismatch messages
- **Metrics:** Installation success rate drops below 90%

#### Impact When Active
- Claude Code resource cannot be installed or updated
- New system deployments blocked
- Development environment setup delays

#### Investigation Notes
- Related to Node.js LTS version requirements (18+)
- Conflicts with other tools requiring different Node.js versions
- Workaround: Use Node Version Manager (nvm) for isolated environments
- Long-term solution: Containerized Claude Code deployment
<!-- EMBED:INTERMITTENTPROBLEM:END -->

<!-- EMBED:INTERMITTENTPROBLEM:START -->
### Permission Denied File Access Errors
**Pattern:** [system-dependent]
**Last Seen:** 2025-08-28
**Frequency:** 2-3 times per week
**Tracking Since:** 2025-08-01

#### Problem Pattern
- **Triggers:** File system permission changes, user account changes, sudo configuration
- **Conditions:** More frequent on multi-user systems or after system administration changes
- **Duration:** Variable, until permissions corrected or sudo configured
- **Recovery:** Permission fixing or sudo access configuration

#### Detection Criteria
- **Symptoms:** EACCES errors when accessing files or directories
- **Monitoring:** File operation failure tracking
- **Logs:** Permission denied errors in Claude Code execution logs
- **Metrics:** File access error rate >2% per session

#### Impact When Active
- Claude Code cannot read/write files in project directories
- Task execution fails when file manipulation required
- Repository access and modification blocked

#### Investigation Notes
- Related to Claude Code's need for extensive file system access
- Sudo configuration varies between systems and security policies
- Workaround: Run Claude Code with appropriate user permissions
- Potential solution: Better permission validation during installation
<!-- EMBED:INTERMITTENTPROBLEM:END -->

## Recently Resolved

<!-- EMBED:RESOLVEDPROBLEM:START -->
### npm Global Package Installation Failures
**Resolved:** 2025-08-23
**Duration:** 10 days active
**Resolution:** [configuration]
**Resolved By:** [resource-experimenter]

#### Problem Summary
Claude Code installation failing due to npm permission issues when installing global packages, particularly on systems with restricted npm global directories.

#### Root Cause
- **Technical Cause:** Default npm configuration attempting to install to system directories without sufficient permissions
- **Process Cause:** Installation script didn't handle npm permission configuration
- **Knowledge Cause:** Insufficient documentation of npm permission requirements

#### Solution Implemented
- **Changes Made:** Updated installation script to configure npm prefix to user directory
- **Validation:** Tested installation on fresh Ubuntu and CentOS systems
- **Rollback Plan:** Manual npm configuration if automated setup fails
- **Documentation:** Added npm permission troubleshooting section

#### Lessons Learned
- **Prevention:** Include npm configuration validation in installation process
- **Detection:** Add npm permission checks to pre-installation validation
- **Response:** Provide clear error messages with resolution steps
- **Knowledge Gaps:** Better understanding of npm security models across distributions

#### Success Metrics
- **Resolution Time:** 2 days from detection to fix
- **Effectiveness:** Installation success rate improved from 70% to 95%
- **Improvements:** Zero npm permission errors for 6 days post-fix
<!-- EMBED:RESOLVEDPROBLEM:END -->

<!-- EMBED:RESOLVEDPROBLEM:START -->
### Context Window Exceeded Errors
**Resolved:** 2025-08-19
**Duration:** 2 weeks active
**Resolution:** [code_fix]
**Resolved By:** [task-planner]

#### Problem Summary
Claude Code sessions failing when processing large files or complex tasks due to context window limitations, resulting in truncated responses and incomplete task execution.

#### Root Cause
- **Technical Cause:** No context management for large inputs, entire context sent in single API call
- **Process Cause:** Task decomposition not implemented for large context scenarios
- **Knowledge Cause:** Insufficient understanding of Claude API context limits

#### Solution Implemented
- **Changes Made:** Implemented intelligent context chunking and sliding window approach
- **Validation:** Tested with large codebase analysis tasks (>200 files)
- **Rollback Plan:** Fallback to smaller context windows if chunking fails
- **Documentation:** Added context management best practices guide

#### Lessons Learned
- **Prevention:** Implement context size validation before API calls
- **Detection:** Monitor context usage patterns and API response truncation
- **Response:** Automatic context optimization based on task requirements
- **Knowledge Gaps:** Need better understanding of optimal context chunking strategies

#### Success Metrics
- **Resolution Time:** 5 days from detection to implementation
- **Effectiveness:** Large file analysis success rate increased from 30% to 85%
- **Improvements:** Context-related failures reduced by 90%
<!-- EMBED:RESOLVEDPROBLEM:END -->

## Problem Patterns

<!-- EMBED:PROBLEMPATTERN:START -->
### Rate Limiting and Resource Management
**Category:** [performance]
**Frequency:** Multiple times daily
**Affected Systems:** API calls, session management, concurrent task execution

#### Pattern Description
- **Common Symptoms:** 429 rate limit errors, API timeouts, session throttling, execution delays
- **Typical Triggers:** High automation load, concurrent scenario execution, token-heavy tasks
- **Evolution:** Starts with occasional delays, escalates to complete service unavailability
- **Cascade Effects:** Task queue backup, scenario failures, automation system shutdown

#### Detection Strategy
- **Early Warnings:** API response time >5s, rate limit warning headers, token usage >80% of limit
- **Monitoring Approach:** API call success rates, token consumption tracking
- **Alert Configuration:** Rate limit warnings, API failure rate >5%
- **Automated Checks:** Token usage monitoring, API health endpoint validation

#### Resolution Strategy
- **Standard Approach:** Request spacing, token optimization, API key rotation
- **Time to Resolution:** 15-30 minutes for spacing fixes, 1-2 hours for optimization
- **Resource Requirements:** API management knowledge, Claude Code architecture understanding
- **Success Rate:** 85% for spacing solutions, 70% for long-term optimization

#### Prevention Measures
- **Design Changes:** Implement intelligent request queuing, predictive token usage
- **Process Improvements:** Better task planning to optimize API usage
- **Monitoring Enhancements:** Real-time token usage tracking, predictive rate limiting
- **Training Needs:** Claude API best practices, resource optimization strategies
<!-- EMBED:PROBLEMPATTERN:END -->

<!-- EMBED:PROBLEMPATTERN:START -->
### Installation and Environment Configuration
**Category:** [reliability]
**Frequency:** 2-3 times per month
**Affected Systems:** Node.js environment, npm packages, file system permissions

#### Pattern Description
- **Common Symptoms:** Installation failures, version conflicts, permission errors, dependency issues
- **Typical Triggers:** System updates, environment changes, multi-user configurations
- **Evolution:** Usually immediate failure during installation or first execution attempt
- **Cascade Effects:** Complete Claude Code unavailability, development environment blocking

#### Detection Strategy
- **Early Warnings:** Installation script errors, version compatibility warnings
- **Monitoring Approach:** Installation success rate tracking, environment validation
- **Alert Configuration:** Installation failure alerts, environment misconfiguration detection
- **Automated Checks:** Pre-installation environment validation, dependency verification

#### Resolution Strategy
- **Standard Approach:** Environment reset, dependency reinstallation, permission correction
- **Time to Resolution:** 30-60 minutes for environment fixes, 2+ hours for complex conflicts
- **Resource Requirements:** System administration knowledge, Node.js expertise
- **Success Rate:** 90% for permission fixes, 60% for complex environment issues

#### Prevention Measures
- **Design Changes:** Containerized deployment, environment isolation
- **Process Improvements:** Comprehensive pre-installation validation
- **Monitoring Enhancements:** Continuous environment health monitoring
- **Training Needs:** Node.js environment management, system administration basics
<!-- EMBED:PROBLEMPATTERN:END -->

## Integration Points

<!-- EMBED:PROBLEMINTEGRATION:START -->
### Swarm-Manager Integration
This problems file is automatically scanned by swarm-manager for:

#### Task Generation
- **Active Issues:** Auto-create tasks for critical rate limiting and authentication problems
- **Pattern Recognition:** Generate prevention tasks for environment configuration patterns
- **Knowledge Gaps:** Create documentation tasks for API optimization and installation best practices
- **Monitoring Improvements:** Generate tasks to improve rate limit monitoring and session management

#### Priority Calculation
Problems contribute to task priority based on:
- **Severity + Frequency → Impact score** (rate limiting = 10, auth failures = 8, memory issues = 9)
- **User/business impact → Urgency level** (system_down = critical, degraded_performance = high)
- **Investigation status → Success probability** (config issues = 0.9, optimization = 0.7)
- **Complexity → Resource cost estimate** (installation = moderate, rate limiting = moderate)

#### Learning Integration
- **Resolution Patterns:** Track which solutions work for rate limiting vs authentication vs environment issues
- **Prediction Models:** Learn to predict rate limit problems based on task queue depth and token usage
- **Resource Optimization:** Understand optimal API usage patterns and session management
- **Success Metrics:** Measure improvement in API success rates and installation reliability

#### Automation Opportunities
- **Auto-Detection:** Token usage monitoring, API health checks, environment validation
- **Auto-Triage:** Classify rate limiting vs authentication vs environment vs context issues
- **Auto-Resolution:** API request spacing, session refresh, permission fixing
- **Auto-Escalation:** Human intervention for complex environment conflicts or persistent API issues
<!-- EMBED:PROBLEMINTEGRATION:END -->