# Problems

This document captures ongoing issues, system failures, performance degradations, and recurring problems discovered during PostgreSQL multi-instance operation. It serves as a structured problem registry for autonomous problem detection and resolution.

## Active Issues

<!-- EMBED:ACTIVEPROBLEM:START -->
### Port Allocation Conflicts
**Status:** [active]
**Severity:** [high]
**Frequency:** [frequent]
**Impact:** [system_down]
**Discovered:** 2025-08-28
**Discovered By:** [system-monitor]
**Last Occurrence:** 2025-08-29 11:20

#### Description
Multiple PostgreSQL instances attempting to use the same port during simultaneous client project deployments. Auto-port detection occasionally fails, resulting in instance creation failures and existing instance disruption.

#### Reproduction Steps
1. Create first client instance: `./manage.sh --action create --instance client-a --port 5434`
2. Create second instance without specifying port: `./manage.sh --action create --instance client-b`
3. Second instance attempts to use port 5434
4. Instance creation fails with "port already in use" error

#### Impact Assessment
- **Users Affected:** All clients during parallel development phases
- **Business Impact:** Project deployment delays, client delivery blocked
- **Technical Impact:** Instance creation failure rate ~15%, manual intervention required
- **Urgency Factors:** Blocks client project delivery, affects development team productivity

#### Investigation Status
- **Root Cause:** Port detection algorithm doesn't account for recently created but not yet started instances
- **Workarounds:** Manual port specification, delayed instance creation
- **Related Issues:** Docker networking conflicts, instance state synchronization
- **Attempted Solutions:** Improved port scanning logic (testing in progress)

#### Priority Estimates
```yaml
impact: 8           
urgency: "high"
success_prob: 0.9   
resource_cost: "moderate"
```
<!-- EMBED:ACTIVEPROBLEM:END -->

<!-- EMBED:ACTIVEPROBLEM:START -->
### Database Connection Pool Exhaustion
**Status:** [active]
**Severity:** [high]
**Frequency:** [occasional]
**Impact:** [degraded_performance]
**Discovered:** 2025-08-25
**Discovered By:** [app-debugger]
**Last Occurrence:** 2025-08-29 15:45

#### Description
Client applications exhausting available database connections during high-load periods. PostgreSQL instances hitting max_connections limit (default 100), causing new connection attempts to fail with "too many clients" error.

#### Reproduction Steps
1. Create client instance with default configuration
2. Run client application with high concurrent database access (>100 connections)
3. Observe connection failures and "FATAL: too many clients already" errors
4. Application performance degrades significantly

#### Impact Assessment
- **Users Affected:** Clients with high-traffic applications (20% of deployments)
- **Business Impact:** Client application failures, poor user experience
- **Technical Impact:** Database unavailable during connection exhaustion, cascade failures
- **Urgency Factors:** Affects production client applications, reputation risk

#### Investigation Status
- **Root Cause:** Default PostgreSQL configuration not optimized for production workloads
- **Workarounds:** Application-level connection pooling, periodic connection recycling
- **Related Issues:** Memory allocation for connections, application architecture
- **Attempted Solutions:** Increased max_connections to 300, tuning shared_buffers

#### Priority Estimates
```yaml
impact: 9           
urgency: "high"
success_prob: 0.8   
resource_cost: "moderate"
```
<!-- EMBED:ACTIVEPROBLEM:END -->

<!-- EMBED:ACTIVEPROBLEM:START -->
### Instance Data Corruption During Backup
**Status:** [active]
**Severity:** [critical]
**Frequency:** [rare]
**Impact:** [system_down]
**Discovered:** 2025-08-20
**Discovered By:** [system-monitor]
**Last Occurrence:** 2025-08-27 09:30

#### Description
Intermittent data corruption detected in PostgreSQL instances during backup operations. Backup process occasionally captures inconsistent state, resulting in corrupted backup files that cannot be restored.

#### Reproduction Steps
1. Create client instance with active database operations
2. Initiate backup while database has ongoing transactions
3. Backup appears to complete successfully
4. Attempt to restore backup reveals data corruption errors

#### Impact Assessment
- **Users Affected:** All clients if backup corruption goes undetected
- **Business Impact:** Potential data loss, client trust issues, recovery delays
- **Technical Impact:** Unusable backups, restoration failures, data integrity issues
- **Urgency Factors:** Risk of catastrophic data loss, client SLA violations

#### Investigation Status
- **Root Cause:** Backup process not properly coordinating with active transactions
- **Workarounds:** Stop instance before backup (not practical for production)
- **Related Issues:** pg_dump concurrency handling, transaction isolation
- **Attempted Solutions:** Implementing pg_basebackup with WAL archiving

#### Priority Estimates
```yaml
impact: 10           
urgency: "critical"
success_prob: 0.7   
resource_cost: "heavy"
```
<!-- EMBED:ACTIVEPROBLEM:END -->

## Intermittent Issues

<!-- EMBED:INTERMITTENTPROBLEM:START -->
### Docker Volume Permission Issues
**Pattern:** [environmental]
**Last Seen:** 2025-08-28
**Frequency:** 1-2 times per week
**Tracking Since:** 2025-08-10

#### Problem Pattern
- **Triggers:** System reboots, Docker daemon restarts, volume mounting changes
- **Conditions:** Occurs more frequently on different host OS configurations
- **Duration:** 10-30 minutes until permissions corrected
- **Recovery:** Manual permission reset or volume remounting

#### Detection Criteria
- **Symptoms:** PostgreSQL container fails to start with permission denied errors
- **Monitoring:** Container startup failure logs
- **Logs:** "Permission denied" for /var/lib/postgresql/data directory
- **Metrics:** Instance startup failure rate spikes

#### Impact When Active
- Specific client instances cannot start
- Database unavailable until manual intervention
- Development team productivity loss

#### Investigation Notes
- Related to Docker volume ownership mapping
- More prevalent on Ubuntu vs CentOS systems
- Workaround: Reset volume permissions before container start
- Long-term solution: Use named volumes instead of bind mounts
<!-- EMBED:INTERMITTENTPROBLEM:END -->

<!-- EMBED:INTERMITTENTPROBLEM:START -->
### Schema Migration Race Conditions
**Pattern:** [load-based]
**Last Seen:** 2025-08-26
**Frequency:** 3-4 times per month
**Tracking Since:** 2025-07-20

#### Problem Pattern
- **Triggers:** Rapid instance creation, concurrent migration execution
- **Conditions:** Multiple instances created simultaneously with same template
- **Duration:** Migration failures until manual resolution
- **Recovery:** Manual migration re-run or rollback

#### Detection Criteria
- **Symptoms:** Migration scripts fail with "relation already exists" or deadlock errors
- **Monitoring:** Migration success/failure tracking
- **Logs:** PostgreSQL migration error messages
- **Metrics:** Migration failure rate >5% during bulk instance creation

#### Impact When Active
- Client instances created with incomplete schema
- Database structure inconsistencies
- Application functionality breaks

#### Investigation Notes
- Concurrent access to migration state tracking
- Template application race conditions
- Potential solution: Migration locking mechanism
- Workaround: Sequential instance creation during bulk deployments
<!-- EMBED:INTERMITTENTPROBLEM:END -->

## Recently Resolved

<!-- EMBED:RESOLVEDPROBLEM:START -->
### Memory Allocation Errors in Production Template
**Resolved:** 2025-08-22
**Duration:** 1 week active
**Resolution:** [configuration]
**Resolved By:** [system-monitor]

#### Problem Summary
PostgreSQL instances created with production template experiencing out-of-memory errors during high-load operations, particularly with complex queries and large result sets.

#### Root Cause
- **Technical Cause:** Production template allocated too much memory to shared_buffers relative to available system RAM
- **Process Cause:** Template optimization done in isolation without considering multi-instance deployment
- **Knowledge Cause:** Inadequate understanding of PostgreSQL memory allocation in containerized environments

#### Solution Implemented
- **Changes Made:** Reduced shared_buffers from 2GB to 512MB, optimized work_mem settings
- **Validation:** Load testing with 5 concurrent production instances
- **Rollback Plan:** Revert to development template settings if performance degrades
- **Documentation:** Updated template configuration guide with memory allocation formulas

#### Lessons Learned
- **Prevention:** Test template configurations under realistic multi-instance loads
- **Detection:** Implement memory usage monitoring for each instance
- **Response:** Create graduated memory allocation based on instance type and expected load
- **Knowledge Gaps:** Container memory limits vs PostgreSQL configuration alignment

#### Success Metrics
- **Resolution Time:** 2 days from detection to fix
- **Effectiveness:** Zero OOM errors for 7 days post-fix
- **Improvements:** 40% reduction in query execution time
<!-- EMBED:RESOLVEDPROBLEM:END -->

<!-- EMBED:RESOLVEDPROBLEM:START -->
### pgweb GUI Connection Timeouts
**Resolved:** 2025-08-18
**Duration:** 5 days active
**Resolution:** [code_fix]
**Resolved By:** [resource-experimenter]

#### Problem Summary
Web-based database management interface (pgweb) timing out when connecting to client instances, preventing visual database management and troubleshooting.

#### Root Cause
- **Technical Cause:** pgweb container using wrong network configuration for inter-container communication
- **Process Cause:** Network configuration not updated when instance networking was refactored
- **Knowledge Cause:** Docker networking complexity in multi-instance environment

#### Solution Implemented
- **Changes Made:** Updated pgweb container to use Docker internal networking, fixed hostname resolution
- **Validation:** Tested GUI access for all existing client instances
- **Rollback Plan:** Manual connection strings if automated GUI fails
- **Documentation:** Added network troubleshooting section to GUI documentation

#### Lessons Learned
- **Prevention:** Include GUI connectivity in instance creation validation tests
- **Detection:** Add automated GUI health checks to monitoring
- **Response:** Implement network connectivity diagnostics in management script
- **Knowledge Gaps:** Need better Docker networking documentation for multi-instance scenarios

#### Success Metrics
- **Resolution Time:** 1 day from report to fix
- **Effectiveness:** 100% GUI connectivity success for 11 days post-fix
- **Improvements:** GUI load time reduced from 30s to 3s
<!-- EMBED:RESOLVEDPROBLEM:END -->

## Problem Patterns

<!-- EMBED:PROBLEMPATTERN:START -->
### Multi-Instance Resource Conflicts
**Category:** [integration]
**Frequency:** Multiple times per week
**Affected Systems:** Port allocation, Docker networking, volume management

#### Pattern Description
- **Common Symptoms:** Port conflicts, volume permission issues, network connectivity failures
- **Typical Triggers:** Rapid instance creation, concurrent deployments, resource cleanup failures
- **Evolution:** Initial single instance works fine, problems emerge with 2+ instances
- **Cascade Effects:** Instance creation failures, existing instance disruption, development delays

#### Detection Strategy
- **Early Warnings:** Port scan conflicts, Docker volume errors, network connectivity timeouts
- **Monitoring Approach:** Instance health tracking, resource utilization monitoring
- **Alert Configuration:** Instance creation failure rate >10%, port conflict alerts
- **Automated Checks:** Pre-creation resource availability validation

#### Resolution Strategy
- **Standard Approach:** Resource conflict detection, automated port assignment, volume cleanup
- **Time to Resolution:** 10-15 minutes for port conflicts, 30+ minutes for volume issues
- **Resource Requirements:** Docker administration knowledge, networking troubleshooting skills
- **Success Rate:** 90% for port conflicts, 70% for volume/networking issues

#### Prevention Measures
- **Design Changes:** Implement resource reservation system, improve conflict detection
- **Process Improvements:** Sequential instance creation during bulk operations
- **Monitoring Enhancements:** Real-time resource utilization tracking
- **Training Needs:** Docker multi-instance best practices, networking troubleshooting
<!-- EMBED:PROBLEMPATTERN:END -->

<!-- EMBED:PROBLEMPATTERN:START -->
### Data Integrity and Backup Pattern
**Category:** [reliability]
**Frequency:** 1-2 times per month
**Affected Systems:** Backup operations, data restoration, migration processes

#### Pattern Description
- **Common Symptoms:** Backup corruption, restoration failures, data consistency errors
- **Typical Triggers:** High transaction load during backup, concurrent operations, migration race conditions
- **Evolution:** Problems often not detected until backup restoration is attempted
- **Cascade Effects:** Data loss risk, client delivery delays, trust issues

#### Detection Strategy
- **Early Warnings:** Backup verification failures, transaction log inconsistencies
- **Monitoring Approach:** Automated backup testing, data integrity checks
- **Alert Configuration:** Backup failure alerts, restore test failures
- **Automated Checks:** Daily backup verification, weekly restore testing

#### Resolution Strategy
- **Standard Approach:** Stop-start backup process, point-in-time recovery, manual data verification
- **Time to Resolution:** 30-60 minutes for backup fixes, 2-4 hours for data recovery
- **Resource Requirements:** PostgreSQL administration expertise, backup/recovery knowledge
- **Success Rate:** 80% for backup fixes, 60% for complete data recovery

#### Prevention Measures
- **Design Changes:** Implement online backup with WAL archiving, transaction coordination
- **Process Improvements:** Scheduled backup windows, pre-backup consistency checks
- **Monitoring Enhancements:** Continuous data integrity monitoring, backup quality metrics
- **Training Needs:** PostgreSQL backup strategies, disaster recovery procedures
<!-- EMBED:PROBLEMPATTERN:END -->

## Integration Points

<!-- EMBED:PROBLEMINTEGRATION:START -->
### Swarm-Manager Integration
This problems file is automatically scanned by swarm-manager for:

#### Task Generation
- **Active Issues:** Auto-create tasks for critical data corruption issues and connection pool problems
- **Pattern Recognition:** Generate prevention tasks for multi-instance resource conflict patterns
- **Knowledge Gaps:** Create documentation tasks for Docker multi-instance best practices
- **Monitoring Improvements:** Generate tasks to improve backup verification and instance health monitoring

#### Priority Calculation
Problems contribute to task priority based on:
- **Severity + Frequency → Impact score** (data corruption = 10, port conflicts = 8)
- **User/business impact → Urgency level** (system_down = critical, degraded_performance = high)
- **Investigation status → Success probability** (backup issues = 0.7, config issues = 0.9)
- **Complexity → Resource cost estimate** (data integrity = heavy, port conflicts = moderate)

#### Learning Integration
- **Resolution Patterns:** Track which solutions work for resource conflicts vs data integrity issues
- **Prediction Models:** Learn to predict port conflicts based on instance creation patterns
- **Resource Optimization:** Understand optimal resource allocation for multi-instance deployments
- **Success Metrics:** Measure improvement in instance creation success rate and backup reliability

#### Automation Opportunities
- **Auto-Detection:** Port scanning and resource availability checks before instance creation
- **Auto-Triage:** Classify resource conflicts vs data integrity vs configuration issues
- **Auto-Resolution:** Automatic port assignment, volume permission fixing
- **Auto-Escalation:** Human intervention for data corruption or complex backup issues
<!-- EMBED:PROBLEMINTEGRATION:END -->