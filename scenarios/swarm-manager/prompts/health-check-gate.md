# Core Health Check Gate

Before executing any task, check core Vrooli infrastructure health to prevent cascading failures.

## Pre-Execution Health Check

Run this check before dispatching tasks:

```bash
# Check overall core health
core-debugger status --json

# If critical issues exist, do not proceed
if [ "$(core-debugger status --json | jq -r '.status')" == "critical" ]; then
    echo "CRITICAL: Core infrastructure unhealthy. Resolving core issues first..."
    core-debugger list-issues --severity critical
    exit 1
fi

# Check specific component if task targets it
core-debugger check-health --component <component>
```

## Health Status Interpretation

### Status: `healthy` ✅
- All core components operational
- Safe to proceed with task execution

### Status: `degraded` ⚠️
- Some components experiencing issues
- Check if workarounds are available
- Proceed with caution, log warnings

### Status: `critical` ❌
- Core infrastructure failures detected
- DO NOT proceed with normal tasks
- Priority: Fix core issues immediately

## Workaround Application

If core issues exist but have known workarounds:

```bash
# Get workaround for specific error
workaround=$(core-debugger get-workaround "<error-message>")

# Apply workaround if available
if [ -n "$workaround" ]; then
    echo "Applying workaround: $workaround"
    # Execute workaround commands
    eval "$workaround"
fi
```

## Priority Override Rules

When core issues are detected:

1. **Stop all non-critical work**
2. **Create high-priority core-fix task**:
   ```yaml
   title: "Fix core infrastructure: <component>"
   type: core-infrastructure
   scenario: core-debugger
   priority: 10000  # Maximum priority
   ```
3. **Route to core-debugger immediately**
4. **Resume normal work only after resolution**

## Component Dependencies

Check these components based on task type:

| Task Type | Required Components |
|-----------|-------------------|
| scenario-* | cli, orchestrator |
| resource-* | cli, resource-manager |
| setup-* | cli, setup |
| any | cli |

## Health Check Integration Points

### 1. Task Dispatcher (Before Execution)
```bash
# In task dispatcher
check_core_health() {
    local health=$(core-debugger check-health --json)
    local status=$(echo "$health" | jq -r '.status')
    
    if [ "$status" == "critical" ]; then
        create_core_fix_task
        return 1
    elif [ "$status" == "degraded" ]; then
        log_warning "Core degraded, proceeding with caution"
    fi
    return 0
}
```

### 2. Periodic Health Monitoring
```bash
# Run every 60 seconds
while true; do
    core-debugger check-health --json > /tmp/core-health.json
    process_health_status
    sleep 60
done
```

### 3. Error Recovery
```bash
# On any task failure
on_task_failure() {
    # Check if failure was due to core issue
    if core-debugger check-health --component "$failed_component" | grep -q "unhealthy"; then
        # Create core-fix task with details
        core-debugger report-issue \
            --component "$failed_component" \
            --description "$error_message" \
            --severity high
    fi
}
```

## Automatic Core Issue Resolution Flow

1. **Detection**: Health check fails
2. **Analysis**: `core-debugger analyze-issue <issue-id>`
3. **Workaround Check**: Look for known fixes
4. **Apply Fix**: Execute workaround or create fix task
5. **Verify**: Re-run health check
6. **Resume**: Continue normal operations

## Critical Components Priority

Order of importance when multiple components fail:

1. **CLI** - Nothing works without this
2. **Setup Scripts** - Can't initialize anything
3. **Orchestrator** - Can't run scenarios
4. **Resource Manager** - Can't manage resources
5. **API** - Can still work via CLI

## Example Health Check Output

```json
{
  "status": "degraded",
  "components": [
    {
      "name": "cli",
      "status": "healthy",
      "response_time_ms": 50
    },
    {
      "name": "orchestrator",
      "status": "unhealthy",
      "error": "timeout",
      "workaround_available": true
    }
  ],
  "active_issues": 2,
  "recommendation": "Apply workaround for orchestrator timeout"
}
```

## Integration with Swarm Manager

The swarm-manager should:
1. Load this health check configuration from `config/settings.yaml`
2. Run checks based on `health_check_before_dispatch.enabled`
3. Block on critical if `health_check_before_dispatch.block_on_critical`
4. Apply priority modifiers from `priority_modifiers.core_infrastructure_issue`

Remember: **Core health is the foundation - without it, nothing else works!**