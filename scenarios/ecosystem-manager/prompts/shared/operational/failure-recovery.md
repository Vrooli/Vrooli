# Failure Recovery

## Purpose
Failures are learning opportunities. Proper failure recovery ensures progress continues, knowledge is preserved, and patterns don't repeat.

## Failure Classification

### Severity Levels
```markdown
## Failure Severity

Level 1: Trivial (Continue)
- Cosmetic issues
- Non-critical warnings
- Minor test failures
- Documentation typos

Level 2: Minor (Fix and Continue)
- Single feature broken
- Performance degradation <20%
- Non-blocking errors
- Partial functionality loss

Level 3: Major (Stop and Fix)
- Core feature broken
- Multiple test failures
- API breakage
- Data integrity risk

Level 4: Critical (Rollback)
- System won't start
- Data corruption
- Security vulnerability exposed
- Dependencies broken

Level 5: Catastrophic (Emergency)
- Production system down
- Data loss occurring
- Security breach active
- Revenue impact immediate
```

## Failure Response Protocol

### Immediate Response
```bash
# When failure detected
handle_failure() {
    local severity=$1
    local description=$2
    
    case $severity in
        1|2)  # Trivial/Minor - Document and continue
            log_failure "$description"
            add_to_backlog "$description"
            continue_with_workaround
            ;;
            
        3)  # Major - Stop current work
            stop_current_work
            document_failure "$description"
            create_fix_queue_item
            switch_to_different_task
            ;;
            
        4)  # Critical - Rollback
            immediate_rollback
            alert_team "$description"
            document_rollback_steps
            create_urgent_fix_item
            ;;
            
        5)  # Catastrophic - Emergency mode
            emergency_shutdown
            alert_all_systems
            initiate_disaster_recovery
            ;;
    esac
}
```

### Failure Documentation
```markdown
## Failure Report Template

### Failure Summary
- **Timestamp**: 2025-01-07T12:34:56Z
- **Component**: [scenario/resource name]
- **Severity**: [1-5]
- **Impact**: [Who/what affected]

### What Happened
[Describe the failure clearly]

### Root Cause
[If known, otherwise "Under investigation"]

### Attempted Fixes
1. [What was tried]
   - Result: [Did it work?]
2. [What else was tried]
   - Result: [Did it work?]

### Current State
- System status: [Running/Degraded/Down]
- Workaround: [If any]
- Data impact: [None/Some/Significant]

### Resolution Plan
1. [Immediate action]
2. [Short-term fix]
3. [Long-term solution]

### Lessons Learned
- [What we learned]
- [How to prevent recurrence]

### Memory System Update
```bash
# Commands to update Qdrant with failure pattern
vrooli resource qdrant add-failure "[component] [failure type]" "details"
```
```

## Recovery Strategies

### Strategy 1: Rollback and Retry
```bash
rollback_and_retry() {
    local checkpoint=$1
    
    # Restore to last known good state
    git diff > /tmp/failed-changes.patch
    git checkout -- .
    
    # Analyze what went wrong
    analyze_failure /tmp/failed-changes.patch
    
    # Adjust approach
    create_smaller_change
    add_more_tests
    
    # Retry with lessons learned
    implement_with_safeguards
}
```

### Strategy 2: Workaround and Continue
```python
def workaround_and_continue(failure):
    """
    Implement temporary workaround to maintain progress
    """
    # Document the workaround
    workaround = {
        'original_approach': failure['attempted'],
        'workaround': determine_workaround(failure),
        'technical_debt': True,
        'fix_priority': 'high',
        'estimated_removal': '2 weeks'
    }
    
    # Implement workaround
    implement_workaround(workaround)
    
    # Create follow-up task
    create_queue_item(
        title=f"Remove workaround for {failure['component']}",
        priority='high',
        type='technical-debt'
    )
    
    # Continue with main work
    return continue_progress()
```

### Strategy 3: Decompose and Simplify
```python
def decompose_failed_task(failed_item):
    """
    Break failed complex task into smaller pieces
    """
    # Analyze why it failed
    complexity_score = analyze_complexity(failed_item)
    
    if complexity_score > 8:
        # Too complex - break it down
        subtasks = []
        
        # Create smaller, focused tasks
        for requirement in failed_item['requirements']:
            subtask = {
                'title': f"Part of {failed_item['title']}: {requirement}",
                'requirements': [requirement],
                'estimated_hours': failed_item['estimated_hours'] / len(failed_item['requirements']),
                'parent': failed_item['id']
            }
            subtasks.append(subtask)
        
        # Add to queue with staggered priorities
        for i, task in enumerate(subtasks):
            task['priority'] = failed_item['priority'] + i * 10
            add_to_queue(task)
    
    return subtasks
```

## Failure Patterns and Prevention

### Common Failure Patterns
```markdown
## Pattern 1: Dependency Not Running
**Symptom**: Connection refused errors
**Cause**: Required resource not started
**Prevention**:
- Check dependencies before starting
- Add to setup validation
- Document in README

## Pattern 2: Port Conflicts
**Symptom**: Address already in use
**Cause**: Port allocation collision
**Prevention**:
- Use port registry
- Check availability before binding
- Configure fallback ports

## Pattern 3: Configuration Mismatch
**Symptom**: Unexpected behavior, wrong endpoints
**Cause**: Hardcoded values, environment variables not set
**Prevention**:
- Centralize configuration
- Validate on startup
- Provide defaults

## Pattern 4: API Breaking Changes
**Symptom**: 404s, schema validation errors
**Cause**: Incompatible updates
**Prevention**:
- Version APIs
- Deprecation warnings
- Backward compatibility

## Pattern 5: Test False Positives
**Symptom**: Tests pass but feature broken
**Cause**: Tests not testing actual functionality
**Prevention**:
- Test real scenarios
- Integration tests
- Manual verification
```

### Prevention Checklist
```markdown
## Before Making Changes
- [ ] Dependencies running and healthy
- [ ] Current tests passing
- [ ] Backup/checkpoint created
- [ ] Rollback plan documented
- [ ] Impact radius understood

## During Implementation
- [ ] Incremental changes
- [ ] Test after each change
- [ ] Monitor logs for warnings
- [ ] Check resource usage
- [ ] Validate assumptions

## After Changes
- [ ] All tests pass
- [ ] Manual verification done
- [ ] No performance regression
- [ ] Documentation updated
- [ ] Qdrant memory updated
```

## Learning from Failures

### Failure Analysis
```python
def analyze_failure(failure_report):
    """
    Extract learnings from failure
    """
    analysis = {
        'pattern': identify_pattern(failure_report),
        'root_cause': find_root_cause(failure_report),
        'prevention': determine_prevention(failure_report),
        'similar_risks': find_similar_risks(failure_report)
    }
    
    # Update memory system
    update_qdrant_memory(
        f"failure pattern {analysis['pattern']}",
        analysis
    )
    
    # Check for systemic issues
    if count_similar_failures(analysis['pattern']) > 3:
        create_systemic_fix_task(analysis['pattern'])
    
    return analysis
```

### Post-Mortem Process
```markdown
## Post-Mortem for Significant Failures

### When to Conduct
- Severity 4 or 5 failures
- Repeated failures (3+ times)
- Customer-impacting issues
- Data loss or corruption

### Post-Mortem Template
1. **Timeline**: What happened when
2. **Impact**: Who/what was affected
3. **Root Cause**: Why it happened
4. **Contributing Factors**: What made it worse
5. **What Went Well**: What prevented worse outcome
6. **Action Items**: Specific preventive measures
7. **Lessons Learned**: Knowledge to share

### Follow-Up Actions
- [ ] Update documentation
- [ ] Add prevention tests
- [ ] Update memory system
- [ ] Share with team
- [ ] Create prevention tasks
```

## Recovery Tools

### Diagnostic Commands
```bash
# Check system health
diagnose_failure() {
    echo "=== System Diagnosis ==="
    
    # Check services
    echo "Service Status:"
    for service in postgres redis qdrant; do
        vrooli resource $service health 2>&1 || echo "  $service: DOWN"
    done
    
    # Check ports
    echo -e "\nPort Status:"
    netstat -tlnp 2>/dev/null | grep LISTEN
    
    # Check disk space
    echo -e "\nDisk Space:"
    df -h | grep -E "^/dev/"
    
    # Check memory
    echo -e "\nMemory:"
    free -h
    
    # Recent errors
    echo -e "\nRecent Errors:"
    journalctl -p err -n 20 --no-pager 2>/dev/null || \
        grep -i error /var/log/syslog | tail -20
}
```

### Recovery Commands
```bash
# Clean recovery
clean_recovery() {
    # Stop everything
    vrooli stop --all
    
    # Clean temporary files
    rm -rf /tmp/vrooli-*
    
    # Reset state
    for dir in queue/in-progress queue/failed; do
        mkdir -p queue/pending
        mv $dir/*.yaml queue/pending/ 2>/dev/null || true
    done
    
    # Restart core services
    vrooli resource postgres start
    vrooli resource redis start
    vrooli resource qdrant start
    
    # Verify health
    sleep 5
    diagnose_failure
}
```

## Remember for Failure Recovery

**Fail fast** - Detect and respond quickly

**Document everything** - Future agents need to learn

**Small rollbacks** - Undo minimum necessary

**Learn patterns** - Same failure twice is preventable

**Update memory** - Make failure knowledge permanent

Failures are teachers. Embrace them, learn from them, and use them to make the system stronger. Every failure handled well prevents countless future failures.