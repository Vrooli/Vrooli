# Agent Manager - Failure Topography & Graceful Degradation

This document maps the failure landscape of agent-manager, describing how the system fails and how it recovers. Understanding failure modes is essential for building reliable agent orchestration.

## Design Philosophy

Agent-manager is designed for **accident prevention, not adversary security**. The failure handling focuses on:

1. **Safe Defaults**: Fail closed, not open. When uncertain, require sandbox and approval.
2. **Observable Failures**: Every failure is logged with correlation IDs and structured context.
3. **Recoverable State**: Operations are designed to leave the system in a consistent state on failure.
4. **Clear Guidance**: Errors tell users what happened and what to do next.

---

## Critical Flows & Failure Maps

### 1. Run Creation Flow

```
CreateRun Request
       │
       ▼
┌──────────────┐     ┌─────────────────────────────┐
│  Get Task    │────▶│ NotFoundError (Task)        │
└──────────────┘     │ Recovery: fix_input         │
       │             │ Retryable: No               │
       ▼             └─────────────────────────────┘
┌──────────────┐     ┌─────────────────────────────┐
│ Get Profile  │────▶│ NotFoundError (Profile)     │
└──────────────┘     │ Recovery: fix_input         │
       │             │ Retryable: No               │
       ▼             └─────────────────────────────┘
┌──────────────┐     ┌─────────────────────────────┐
│ Evaluate     │────▶│ PolicyViolationError        │
│ Policy       │     │ Recovery: use_alternative   │
└──────────────┘     │ or escalate (if overrideable)│
       │             │ Retryable: No               │
       ▼             └─────────────────────────────┘
┌──────────────┐     ┌─────────────────────────────┐
│ Check        │────▶│ CapacityExceededError       │
│ Capacity     │     │ Recovery: retry_backoff     │
└──────────────┘     │ Retryable: Yes              │
       │             └─────────────────────────────┘
       ▼             ┌─────────────────────────────┐
┌──────────────┐     │ ScopeConflictError          │
│ Check Scope  │────▶│ Recovery: wait              │
│ Locks        │     │ Retryable: Yes              │
└──────────────┘     │ Includes: wait estimate     │
       │             └─────────────────────────────┘
       ▼             ┌─────────────────────────────┐
┌──────────────┐     │ DatabaseError               │
│ Persist Run  │────▶│ Recovery: retry_backoff     │
└──────────────┘     │ Retryable: If transient     │
       │             └─────────────────────────────┘
       ▼
   SUCCESS
 (Run created, execution starts async)
```

**Failure Handling Strategy:**
- Fast-fail on validation/policy errors (don't create partial state)
- Return wait estimates for capacity errors
- Log all failures with request ID for correlation

---

### 2. Run Execution Flow (Async)

```
Execute Run (Background)
       │
       ▼
┌──────────────┐     ┌─────────────────────────────┐
│ Update to    │────▶│ DatabaseError               │
│ "starting"   │     │ Graceful: Mark run as failed│
└──────────────┘     │ State: Run → failed         │
       │             └─────────────────────────────┘
       ▼
┌──────────────┐     ┌─────────────────────────────┐
│ Setup        │────▶│ SandboxError (create)       │
│ Workspace    │     │ Graceful: Mark run as failed│
│              │     │ with sandbox_fail outcome   │
│ ├─sandboxed  │     │ Recovery hint: Check sandbox│
│ └─in_place   │     │ service availability        │
└──────────────┘     └─────────────────────────────┘
       │
       ▼             ┌─────────────────────────────┐
┌──────────────┐     │ RunnerError (unavailable)   │
│ Acquire      │────▶│ Graceful: Mark run as failed│
│ Runner       │     │ Suggest alternative runner  │
└──────────────┘     │ if available                │
       │             └─────────────────────────────┘
       ▼             ┌─────────────────────────────┐
┌──────────────┐     │ RunnerError (execution)     │
│ Execute      │────▶│ Graceful: Capture error,    │
│ Agent        │     │ preserve partial work in    │
│              │     │ sandbox for inspection      │
└──────────────┘     └─────────────────────────────┘
       │             ┌─────────────────────────────┐
       │             │ RunnerError (timeout)       │
       ├────────────▶│ Graceful: Stop runner,      │
       │             │ save partial state          │
       │             └─────────────────────────────┘
       ▼
┌──────────────┐     ┌─────────────────────────────┐
│ Handle       │────▶│ Classify outcome and        │
│ Result       │     │ transition to appropriate   │
└──────────────┘     │ terminal state              │
       │             └─────────────────────────────┘
       ▼
   SUCCESS → needs_review
   FAILURE → failed
   CANCEL  → cancelled
```

**Graceful Degradation Principles:**
1. **Preserve Partial Work**: On runner failure, sandbox state is preserved for inspection
2. **Clean Termination**: Timeouts trigger graceful stop, not hard kill
3. **Clear Classification**: Every outcome maps to a specific RunOutcome for debugging
4. **Event Capture**: All events captured up to failure point

---

### 3. Approval Flow

```
ApproveRun Request
       │
       ▼
┌──────────────┐     ┌─────────────────────────────┐
│ Get Run      │────▶│ NotFoundError (Run)         │
└──────────────┘     │ Recovery: fix_input         │
       │             └─────────────────────────────┘
       ▼             ┌─────────────────────────────┐
┌──────────────┐     │ StateError (not in          │
│ Check        │────▶│ needs_review state)         │
│ Approvable   │     │ Recovery: none (if terminal)│
└──────────────┘     │ or wait (if in progress)    │
       │             └─────────────────────────────┘
       ▼             ┌─────────────────────────────┐
┌──────────────┐     │ SandboxError (approve)      │
│ Apply via    │────▶│ Graceful: Approval failed   │
│ Sandbox      │     │ but state unchanged         │
│              │     │ Recovery: retry_immediate   │
└──────────────┘     └─────────────────────────────┘
       │             ┌─────────────────────────────┐
       │             │ Conflict on Apply           │
       ├────────────▶│ (merge conflict in files)   │
       │             │ Recovery: resolve manually  │
       │             │ or use Force=true           │
       │             └─────────────────────────────┘
       ▼
┌──────────────┐     ┌─────────────────────────────┐
│ Mark Run     │────▶│ DatabaseError               │
│ Approved     │     │ Note: Approval succeeded    │
└──────────────┘     │ but state update failed     │
       │             │ Edge case: idempotent retry │
       ▼             └─────────────────────────────┘
   SUCCESS
 (Changes applied, commit created)
```

**Edge Case Handling:**
- If sandbox approval succeeds but state update fails, the run state is inconsistent
- Mitigation: Changes are applied, so subsequent approval attempts should detect "already approved"
- Future improvement: Use database transactions with sandbox as compensating action

---

## Failure Categories

### Category 1: Input Errors (Client Fixable)
These require the client to correct their request.

| Error Type | Code | Recovery | Example |
|------------|------|----------|---------|
| NotFoundError | NOT_FOUND_* | fix_input | Task ID doesn't exist |
| ValidationError | VALIDATION_* | fix_input | Invalid runner type |
| StateError (terminal) | STATE_TERMINAL | none | Task already completed |

**API Response Pattern:**
```json
{
  "code": "NOT_FOUND_TASK",
  "message": "Task not found: abc-123",
  "userMessage": "The requested Task could not be found. Please verify the ID is correct.",
  "recovery": "fix_input",
  "retryable": false
}
```

---

### Category 2: Policy Errors (Admin/Config Fixable)
These require policy changes or admin override.

| Error Type | Code | Recovery | Example |
|------------|------|----------|---------|
| PolicyViolationError | POLICY_* | escalate/use_alternative | Sandbox required |
| StateError (non-terminal) | STATE_TRANSITION | wait | Task is still running |

**API Response Pattern:**
```json
{
  "code": "POLICY_SANDBOX_REQUIRED",
  "message": "Policy violation [security]: sandbox_required - In-place execution not allowed",
  "userMessage": "Policy 'security' prevents this action. An administrator may override.",
  "recovery": "escalate",
  "retryable": false,
  "details": {
    "policy_name": "security",
    "overrideable": true
  }
}
```

---

### Category 3: Capacity Errors (Transient, Retryable)
These resolve when resources become available.

| Error Type | Code | Recovery | Example |
|------------|------|----------|---------|
| CapacityExceededError | CAPACITY_* | retry_backoff | Max concurrent runs reached |
| ScopeConflictError | CAPACITY_SCOPE | wait | Another run using this path |

**API Response Pattern:**
```json
{
  "code": "CAPACITY_SCOPE_LOCKED",
  "message": "scope path src/ conflicts with 1 existing scope(s)",
  "userMessage": "Another run is using this scope. Estimated wait: 5m30s",
  "recovery": "wait",
  "retryable": true,
  "details": {
    "conflicts": [{"run_id": "xyz-789", "scope_path": "src/"}],
    "wait_estimate": "5m30s"
  }
}
```

**HTTP Headers:**
```
X-Retryable: true
Retry-After: 330
```

---

### Category 4: Infrastructure Errors (System-Level)
These indicate infrastructure problems.

| Error Type | Code | Recovery | Example |
|------------|------|----------|---------|
| RunnerError | RUNNER_* | retry_backoff/use_alternative | Runner timeout |
| SandboxError | SANDBOX_* | retry_backoff/escalate | Sandbox creation failed |
| DatabaseError | DATABASE_* | retry_backoff/escalate | Connection lost |

**API Response Pattern:**
```json
{
  "code": "RUNNER_UNAVAILABLE",
  "message": "runner claude-code error during execute: connection timeout",
  "userMessage": "Runner claude-code is unavailable. You can try using codex instead.",
  "recovery": "use_alternative",
  "retryable": true,
  "details": {
    "runner_type": "claude-code",
    "alternative": "codex",
    "is_transient": true
  }
}
```

---

## Graceful Degradation Strategies

### 1. Runner Failover
When a runner is unavailable:
1. **Check alternatives**: If profile allows multiple runners, suggest alternative
2. **Capture availability**: Include health status in error details
3. **Preserve request**: Don't create a failed run; let client retry with different runner

### 2. Sandbox Unavailability
When sandbox service is down:
1. **Block new runs**: Return 503 Service Unavailable with retry hint
2. **Preserve existing**: Running sandboxes continue; approval deferred
3. **Health endpoint**: Report sandbox status for monitoring

### 3. Database Connectivity Issues
When database is unavailable:
1. **Read-only mode**: Cache recent data for read operations
2. **Queue writes**: Buffer state updates for short outages (future)
3. **Clear status**: Health endpoint reports database unavailable

### 4. Partial Execution Failures
When a run fails mid-execution:
1. **Preserve sandbox**: Don't delete sandbox on failure; allow inspection
2. **Capture events**: Flush all buffered events before marking failed
3. **Record outcome**: Classify failure type (exit_error, timeout, exception)

---

## Observability Requirements

### Structured Logging
All errors include:
```json
{
  "level": "error",
  "error_code": "RUNNER_TIMEOUT",
  "error_category": "RUNNER",
  "request_id": "req-abc-123",
  "run_id": "run-xyz-789",
  "task_id": "task-def-456",
  "runner_type": "claude-code",
  "recovery": "retry_backoff",
  "retryable": true,
  "message": "Runner execution timed out after 30m",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### Metrics to Track
| Metric | Type | Labels |
|--------|------|--------|
| `agent_manager_errors_total` | Counter | code, category |
| `agent_manager_run_failures_total` | Counter | outcome, runner_type |
| `agent_manager_recovery_attempts` | Counter | action, success |
| `agent_manager_scope_lock_wait_seconds` | Histogram | |

### Alerting Thresholds
| Condition | Severity | Action |
|-----------|----------|--------|
| Error rate > 10% | Warning | Investigate |
| Runner unavailable > 5min | Critical | Page on-call |
| Sandbox unavailable > 2min | Critical | Page on-call |
| Database unavailable > 30s | Critical | Page on-call |

---

## State Recovery Patterns

### Orphaned Runs
Runs stuck in non-terminal states after system restart:
1. **Detection**: Scan for runs in `running` state with no active executor
2. **Recovery**: Mark as `failed` with "system_restart" reason
3. **Notification**: Log and emit event for visibility

### Expired Scope Locks
Locks held past TTL:
1. **Detection**: Background job checks lock expiry
2. **Recovery**: Release lock, allowing new runs
3. **Safety**: Log orphaned lock for investigation

### Partial Approvals
Approvals that succeeded but state update failed:
1. **Detection**: Sandbox marked approved but run state is `needs_review`
2. **Recovery**: Re-attempt state update on next read
3. **Idempotency**: Approval operations are idempotent

---

## Testing Failure Paths

### Unit Tests Required
- [ ] Each error type returns correct Code(), Recovery(), Retryable()
- [ ] State machine rejects invalid transitions with proper StateError
- [ ] Decision helpers return accurate reasons

### Integration Tests Required
- [ ] Runner timeout correctly marks run as failed with timeout outcome
- [ ] Sandbox creation failure prevents run from starting
- [ ] Scope lock conflict returns proper wait estimate
- [ ] Database failure during run creation rolls back cleanly

### Chaos Testing (Future)
- [ ] Kill runner mid-execution: verify partial events captured
- [ ] Simulate network partition: verify retry logic
- [ ] Overload system: verify backpressure and capacity errors

---

## Related Documentation

- [errors.go](../api/internal/domain/errors.go) - Error type definitions
- [decisions.go](../api/internal/domain/decisions.go) - State machine logic
- [run_executor.go](../api/internal/orchestration/run_executor.go) - Execution flow
- [SEAMS.md](./SEAMS.md) - Architectural boundaries
