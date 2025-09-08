# Task Completion Protocol

## Purpose
This protocol ensures every ecosystem-manager task completes properly, updates all systems, and provides clear handoff to the next phase of work.

## Completion Philosophy

**A task is NEVER complete until all systems know about it.**

This means:
- ✅ Original work is done and validated
- ✅ PRD.md reflects new reality  
- ✅ Memory system contains learnings
- ✅ Queue system updated with results
- ✅ Dependent systems notified
- ✅ Human receives clear status report

## The 7-Step Completion Protocol

### Step 1: Final Validation (MANDATORY)
**Before claiming completion, verify everything works:**

```bash
# For Generators - Validate new resource/scenario
curl -f http://localhost:$PORT/health                    # Health check passes
vrooli [resource|scenario] $NAME test                    # All tests pass  
grep -c "✅.*$(date +%Y-%m-%d)" PRD.md                   # PRD updated today
ls lib/ | grep -E "(setup|develop|test|stop|health).sh" # All lifecycle scripts exist

# For Improvers - Verify changes work, no regressions  
curl -f http://localhost:$PORT/health                    # Still healthy
vrooli [resource|scenario] $NAME test                    # Tests still pass
git diff HEAD~1 PRD.md | grep -E "(\[x\]|-.*✅)"       # PRD progress made
# Test key functionality mentioned in task requirements
```

**Validation Failures → STOP**: If any validation fails, task is NOT complete. Fix issues first.

### Step 2: PRD Accuracy Update (MANDATORY)
**Ensure PRD.md reflects exact current state:**

```bash
# For Generators - Mark initial implementation complete
# Update PRD.md:
# - [x] Core P0 requirement implemented ✅ 2025-01-07
# - [x] Health checks working ✅ 2025-01-07  
# - [x] Basic CLI functional ✅ 2025-01-07

# For Improvers - Update specific improvements made  
# Update PRD.md:
# - [x] User authentication fixed ✅ 2025-01-07
#   - Test: curl -H "Authorization: Bearer token" /api/user
#   - Result: Returns 200 with user data
# - Progress: 45% → 52% (authentication + input validation)
```

**PRD Rules:**
- Every checkbox change requires completion date: `✅ YYYY-MM-DD`
- Every new checkmark needs test command that proves it works
- Update progress percentages based on completed requirements
- Add progress history entry at bottom of PRD

### Step 3: Memory System Integration (MANDATORY)
**Update Qdrant with learnings for future agents:**

```bash
# Search for existing knowledge about this target
vrooli resource qdrant search-comprehensive "$TARGET_NAME improvements"

# Create memory update with structured information
cat > /tmp/memory_update.md <<EOF
# $TASK_TYPE: $TARGET_NAME - $(date +%Y-%m-%d)

## What Was Done
$SPECIFIC_CHANGES_MADE

## Key Learnings
$TECHNICAL_INSIGHTS_DISCOVERED

## Patterns That Worked
$SUCCESSFUL_APPROACHES_USED

## Future Recommendations
$SUGGESTIONS_FOR_NEXT_IMPROVERS

## Test Commands
$VALIDATION_COMMANDS_THAT_WORK

## Related Targets
$OTHER_SCENARIOS_RESOURCES_AFFECTED
EOF

# Add to memory system
vrooli resource qdrant add-document --file /tmp/memory_update.md --category "task-completion" --tags "$TARGET_NAME,$TASK_TYPE,$(date +%Y-%m)"
```

**Memory Update Rules:**
- Focus on **actionable insights** not just "what happened"
- Include **specific test commands** that proved success
- Note **patterns** that can be reused by other agents
- Identify **related work** that might benefit from these learnings

### Step 4: Queue Management (MANDATORY)
**Properly close the task in the queue system:**

```bash
# Get current task info
TASK_FILE=$(ls queue/in-progress/*.yaml 2>/dev/null | head -1)
TASK_ID=$(basename "$TASK_FILE" .yaml)

# Update task with completion metadata
cat >> "$TASK_FILE" <<EOF

# COMPLETION METADATA
completed_at: "$(date -Iseconds)"
completion_duration: "$(calculate_duration $START_TIME)"
validation_passed: true
prd_updated: true
memory_updated: true
final_status: "success"

# COMPLETION SUMMARY
changes_made:
  - "$SPECIFIC_CHANGE_1"
  - "$SPECIFIC_CHANGE_2"
tests_passing:
  - "$TEST_COMMAND_1: PASS"
  - "$TEST_COMMAND_2: PASS"
follow_up_needed: false
blocking_issues: none

# HANDOFF INFO
next_recommended_tasks:
  - "$LOGICAL_NEXT_TASK_1"
  - "$LOGICAL_NEXT_TASK_2"
dependencies_updated: []
affected_scenarios: []
EOF

# Move to completed queue with timestamp
mkdir -p queue/completed
mv "$TASK_FILE" "queue/completed/${TASK_ID}-$(date +%Y%m%d-%H%M%S).yaml"

# Update WebSocket clients about completion
echo '{"type":"task_completed","task_id":"'$TASK_ID'","status":"success"}' | \
  curl -X POST http://localhost:30500/api/internal/broadcast -d @-
```

### Step 5: Dependency Notification (IF APPLICABLE)
**Notify other systems that depend on this work:**

```bash
# Check if other scenarios/resources depend on this target
DEPENDENTS=$(grep -r "$TARGET_NAME" scenarios/*/PRD.md resources/*/README.md 2>/dev/null | grep -v "^$TARGET_NAME" | cut -d: -f1 | sort -u)

if [ -n "$DEPENDENTS" ]; then
    echo "⚠️ DEPENDENT SYSTEMS DETECTED:"
    echo "$DEPENDENTS"
    echo "Consider notifying teams or creating follow-up tasks to update these systems."
    
    # Create follow-up tasks for dependent systems (if major changes)
    if [ "$BREAKING_CHANGES" = "true" ]; then
        for dep in $DEPENDENTS; do
            DEP_NAME=$(basename $(dirname "$dep"))
            echo "Creating compatibility check task for $DEP_NAME..."
            cp queue/templates/compatibility-check.yaml "queue/pending/200-check-${DEP_NAME}-compatibility-$(date +%s).yaml"
        done
    fi
fi
```

### Step 6: Human Status Report (MANDATORY)
**Provide clear, actionable status to human user:**

```bash
echo "
🎉 TASK COMPLETED SUCCESSFULLY
=============================

📋 Task: $TASK_TITLE
🎯 Target: $TARGET_NAME  
⏱️ Duration: $COMPLETION_DURATION
✅ Status: All validations passed

🔧 CHANGES MADE:
$(echo "$CHANGES_MADE" | sed 's/^/  • /')

🧪 TESTS PASSING:
$(echo "$TESTS_PASSING" | sed 's/^/  ✓ /')

📊 PRD PROGRESS:
  • Previous: $OLD_PROGRESS%
  • Current: $NEW_PROGRESS%
  • Change: +$PROGRESS_DELTA%

🧠 MEMORY UPDATED:
  • Added learnings to Qdrant
  • $MEMORY_INSIGHTS_COUNT insights captured
  • Available for future agents

🔄 QUEUE STATUS:
  • This task: Moved to completed
  • Pending tasks: $(ls queue/pending/*.yaml 2>/dev/null | wc -l)
  • Next recommended: $NEXT_RECOMMENDED_TASK

$DEPENDENCY_NOTIFICATIONS

Ready for next task or human review.
"
```

### Step 7: Next Action Decision (CONTEXT DEPENDENT)
**Determine what to do next based on context:**

```bash
# Decision tree for next actions:

if [ "$MAINTENANCE_STATE" = "inactive" ]; then
    echo "🛑 MAINTENANCE INACTIVE - Waiting for activation"
    echo "Task completed but not selecting next task until maintenance state is 'active'"
    exit 0
    
elif [ "$AUTO_CONTINUE" = "true" ] && [ "$PENDING_COUNT" -gt 0 ] && [ "$ERROR_COUNT" -eq 0 ]; then
    echo "🔄 AUTO-SELECTING NEXT TASK"
    echo "Conditions met for automatic continuation..."
    # Run task selection logic
    NEXT_TASK=$(ecosystem-manager select-next-task)
    echo "Selected: $NEXT_TASK"
    echo "Starting in 5 seconds... (Ctrl+C to abort)"
    sleep 5
    ecosystem-manager process-task "$NEXT_TASK"
    
elif [ "$PENDING_COUNT" -eq 0 ]; then
    echo "✨ ALL TASKS COMPLETE - No pending work"
    echo "Ecosystem manager entering idle state."
    echo "Add new tasks or wait for automated task generation."
    
elif [ "$HUMAN_APPROVAL_REQUIRED" = "true" ]; then
    echo "👤 HUMAN APPROVAL REQUIRED"
    echo "Task completed successfully, but waiting for human review before continuing."
    echo "Next recommended task: $NEXT_RECOMMENDED_TASK"
    
else
    echo "⏸️ TASK COMPLETE - Awaiting Instructions"
    echo "Task finished successfully. Use 'ecosystem-manager select-next-task' to continue."
    echo "Or review completed work and provide new tasks."
fi
```

## Completion Validation Checklist

Before running this protocol, ensure:

- [ ] **Primary task objective achieved** - What was requested is working
- [ ] **No regressions introduced** - Previous functionality still works  
- [ ] **Security requirements met** - All security validations pass
- [ ] **Documentation reflects reality** - READMEs and PRDs are accurate
- [ ] **Tests are comprehensive** - Edge cases and error conditions covered
- [ ] **Performance is acceptable** - No significant degradation introduced

## Error Recovery in Completion Protocol

If any step in the completion protocol fails:

### Step Failure Recovery
```bash
# If Step 1 (Validation) fails:
echo "❌ VALIDATION FAILED - Task cannot be marked complete"
echo "Issues found: $VALIDATION_ERRORS"
echo "Keeping task in in-progress state for fixes"
echo "Run completion protocol again after fixes"
exit 1

# If Step 3 (Memory) fails:
echo "⚠️ MEMORY UPDATE FAILED - Task work complete but not recorded"
echo "Continuing completion but flagging for manual memory update"
MEMORY_UPDATE_FAILED=true

# If Step 4 (Queue) fails:
echo "⚠️ QUEUE UPDATE FAILED - Task complete but not properly closed"
echo "Manual intervention required to move task file"
echo "Work is complete and validated - safe to continue"

# If Step 6 (Reporting) fails:
echo "⚠️ REPORTING FAILED - Task complete but status not sent"
echo "Task successfully completed: $TASK_ID"
echo "Manual status check required"
```

### Recovery Commands
```bash
# Retry just the memory update
ecosystem-manager retry-memory-update $TASK_ID

# Retry just the queue management  
ecosystem-manager retry-queue-update $TASK_ID

# Full completion protocol retry
ecosystem-manager retry-completion $TASK_ID

# Manual override (human verification required)
ecosystem-manager force-complete $TASK_ID --reason "manual_override"
```

## Coordination with Multiple Agents

### Agent Collision Prevention
```bash
# Before starting completion, acquire lock
LOCK_FILE="queue/.completion-lock-$TASK_ID"
if ! (set -C; echo $$ > "$LOCK_FILE") 2>/dev/null; then
    echo "⚠️ Another agent is completing this task"
    echo "Waiting for completion to finish..."
    while [ -f "$LOCK_FILE" ]; do sleep 1; done
    echo "Completion finished by other agent"
    exit 0
fi

# Ensure lock is released even on error
trap 'rm -f "$LOCK_FILE"' EXIT

# ... run completion protocol ...

# Lock automatically released by trap
```

### Agent Coordination Messages
```bash
# Broadcast completion start
echo '{"type":"completion_started","task_id":"'$TASK_ID'","agent_id":"'$AGENT_ID'"}' | \
  curl -X POST http://localhost:30500/api/internal/broadcast -d @-

# Broadcast completion finished
echo '{"type":"completion_finished","task_id":"'$TASK_ID'","success":true}' | \
  curl -X POST http://localhost:30500/api/internal/broadcast -d @-
```

## Integration with Ecosystem Manager API

The completion protocol integrates with the ecosystem-manager API:

```bash
# API calls made during completion:
PUT /api/tasks/$TASK_ID/status {"status": "completing"}
PUT /api/tasks/$TASK_ID/progress {"progress": 100, "phase": "validation"}  
POST /api/tasks/$TASK_ID/completion {"validation_results": {...}}
PUT /api/tasks/$TASK_ID/status {"status": "completed"}

# WebSocket broadcasts sent:
{"type": "task_completing", "task_id": "$TASK_ID"}
{"type": "validation_passed", "task_id": "$TASK_ID"}  
{"type": "memory_updated", "task_id": "$TASK_ID"}
{"type": "task_completed", "task_id": "$TASK_ID", "success": true}
```

## Success Metrics for Completion Protocol

Track completion quality:

- **Completion Success Rate**: % of tasks that complete without errors
- **Validation Accuracy**: % of completed tasks that actually work as expected  
- **Memory Quality**: % of memory updates that help future agents
- **Handoff Clarity**: % of status reports that clearly communicate results
- **Next Task Relevance**: % of recommended next tasks that are actually useful

## Remember: Quality Over Speed

**A properly completed task is worth 10 rushed completions.**

Take time to:
- Validate thoroughly  
- Update memory with insights
- Communicate clearly
- Set up future agents for success

The completion protocol is what turns individual task work into permanent ecosystem improvements. Every step matters.