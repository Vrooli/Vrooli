# Task Analysis Prompt

You are analyzing a task for the Swarm Manager orchestration system. Your job is to evaluate the task and provide estimates for priority calculation.

## Task Details
```yaml
{TASK_CONTENT}
```

## Current System State
- Active tasks: {ACTIVE_TASKS}
- Resource usage: {RESOURCE_USAGE}
- Recent failures: {RECENT_FAILURES}

## Your Analysis

Please analyze this task and provide the following estimates:

### 1. Impact (1-10)
How much will completing this task improve the system?
- 1-3: Minor improvement, nice to have
- 4-6: Moderate improvement, valuable
- 7-9: Major improvement, significant value
- 10: Critical improvement, transformative

### 2. Urgency
How time-sensitive is this task?
- `critical`: System broken, immediate fix needed
- `high`: Important issue affecting users/development
- `medium`: Should be done soon but not urgent
- `low`: Can be done whenever convenient

### 3. Success Probability (0.0-1.0)
What's the likelihood this task will complete successfully?
Consider:
- Complexity of the task
- Available resources and tools
- Similar tasks' success rates
- Current system health

### 4. Resource Cost
How resource-intensive will this task be?
- `minimal`: Quick task, low CPU/memory, few API calls
- `moderate`: Standard task, normal resource usage
- `heavy`: Complex task, high resource usage, many API calls

### 5. Dependencies
List any tasks that must be completed before this one:
```yaml
dependencies: []
```

### 6. Blockers
List any issues preventing this task from being executed:
```yaml
blockers: []
```

### 7. Recommended Scenario
Which scenario CLI should handle this task?
- scenario-generator-v1: For creating new scenarios
- resource-experimenter: For testing resource combinations
- app-debugger: For fixing broken applications
- app-issue-tracker: For reported issues
- system-monitor: For system health issues
- task-planner: For complex multi-step tasks
- claude-code: For general development tasks

### 8. Execution Notes
Any special considerations or context for executing this task:
```yaml
notes: ""
```

## Output Format

Return your analysis in this exact YAML format:
```yaml
estimates:
  impact: [1-10]
  urgency: "[critical|high|medium|low]"
  success_probability: [0.0-1.0]
  resource_cost: "[minimal|moderate|heavy]"
dependencies: []
blockers: []
recommended_scenario: "[scenario-name]"
execution_notes: ""
```

Be objective and realistic in your estimates. Consider the current system state and resource availability when making assessments.