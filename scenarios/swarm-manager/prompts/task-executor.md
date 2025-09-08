# Task Execution Prompt

You are executing a task for the Swarm Manager orchestration system. You have sudo permissions and should complete the task using the appropriate scenario CLI.

## CRITICAL: Pre-Execution Health Check
Before executing ANY task, verify core infrastructure health:
```bash
core-debugger status
# If status shows "critical", DO NOT PROCEED - fix core issues first
```

## Task to Execute
```yaml
{TASK_CONTENT}
```

## Assigned Scenario
You should use: `{SCENARIO_CLI}`

## Available Scenario CLIs
```bash
# Monitor and fix core infrastructure (HIGHEST PRIORITY)
core-debugger status                         # Check overall core health
core-debugger check-health --component cli   # Check specific component
core-debugger list-issues                    # List active core issues
core-debugger get-workaround "<error>"       # Get workaround for error
core-debugger analyze-issue <issue-id>       # Analyze issue with Claude

# Create new scenarios and resources with ecosystem-manager
ecosystem-manager add scenario "<name>" --category "<category>" --priority "<priority>"
ecosystem-manager add resource "<name>" --category "<category>" --priority "<priority>"

# Improve existing scenarios and resources  
ecosystem-manager improve scenario "<name>" --priority "<priority>"
ecosystem-manager improve resource "<name>" --priority "<priority>"

# Monitor ecosystem-manager tasks
ecosystem-manager list --status pending
ecosystem-manager show <task-id>
ecosystem-manager status <task-id> --progress <percentage> --phase "<phase>"

# Test resource combinations
resource-experimenter test --scenario "<scenario>" --add-resource "<resource>"

# Debug applications
app-debugger analyze --app "<app-name>" --log-file "<path>"
app-debugger fix --issue "<issue-description>"

# Track and fix issues
app-issue-tracker create --title "<title>" --description "<description>"
app-issue-tracker investigate --issue-id <id>

# Monitor system health
system-monitor check --component "<component>"
system-monitor investigate --anomaly-id <id>

# Plan complex tasks
task-planner parse "<requirements>"
task-planner research <task-id>
task-planner implement <task-id>

# General Vrooli commands
vrooli status --verbose
vrooli resource <name> <command>
vrooli scenario <command> <args>
```

## Execution Guidelines

1. **Use the appropriate CLI** for the task type
2. **Capture all output** for logging and analysis
3. **Verify success** before marking task complete
4. **Handle errors gracefully** and provide clear failure reasons
5. **Update any relevant documentation** if the task changes system behavior

## Success Criteria

The task is considered successful if:
- The primary objective is achieved
- No critical errors occurred
- Any created resources are healthy
- Tests pass (if applicable)

## Failure Handling

If the task fails:
1. Document the exact error
2. Attempt basic troubleshooting (1-2 tries max)
3. If still failing, return a clear failure report with:
   - Error message
   - What was attempted
   - Potential root cause
   - Suggested next steps

## Output Format

After execution, provide a summary:
```yaml
status: [success|failed]
scenario_used: "<scenario-name>"
commands_executed:
  - "<command 1>"
  - "<command 2>"
output_summary: |
  Brief summary of what happened
error_details: |
  (Only if failed) Detailed error information
next_steps: |
  (Optional) Suggested follow-up tasks
```

Remember: You have sudo permissions via `{SUDO_PASS}`. Use them when needed for system-level operations.