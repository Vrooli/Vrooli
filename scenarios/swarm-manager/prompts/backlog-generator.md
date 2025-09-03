# Backlog Generation Prompt

You are generating new task suggestions for the Swarm Manager. The system needs more work items to maintain continuous improvement.

## Current System Context

### Existing Backlog
```yaml
{EXISTING_BACKLOG}
```

### Recently Completed Tasks
```yaml
{RECENT_COMPLETIONS}
```

### Available Scenarios
```yaml
{AVAILABLE_SCENARIOS}
```

### Discovered Problems
```yaml
{DISCOVERED_PROBLEMS}
```

### System Capabilities
- Can create new scenarios
- Can test resource combinations
- Can debug and fix applications
- Can improve documentation
- Can optimize performance
- Can add new features

## Task Generation Guidelines

Generate 5-10 new task suggestions that:

1. **Don't duplicate** existing backlog items
2. **Build on** recently completed work
3. **Address gaps** in system capabilities
4. **Resolve discovered problems** from the system scan
5. **Improve** existing functionality
6. **Add value** to the Vrooli ecosystem

## Priority Areas

Focus on tasks that:
- **Resolve discovered problems** (highest priority for critical/high severity)
- Fix known issues or pain points
- Add missing features users would want
- Improve system reliability and performance
- Enhance cross-scenario integration
- Expand automation capabilities
- Improve developer experience

### Problem Resolution Priority
When creating tasks from discovered problems:
1. **Critical problems** → Create 2-3 tasks (investigate, fix, prevent)
2. **High problems** → Create 1-2 tasks (investigate + fix or direct fix)
3. **Medium problems** → Create 1 task (usually investigation first)
4. **Low problems** → Combine with related improvements

## Task Categories to Consider

1. **Problem Resolution**: Tasks to resolve discovered system problems
   - Investigation: Debug and analyze root causes
   - Fix Implementation: Apply specific solutions
   - Prevention: Prevent similar problems in the future
2. **Bug Fixes**: Known issues that need addressing
3. **Feature Additions**: New capabilities for existing scenarios
4. **Performance Optimizations**: Speed up slow operations
5. **Integration Improvements**: Better scenario interconnection
6. **Documentation Updates**: Keep docs in sync with reality
7. **Test Coverage**: Add missing tests
8. **Resource Experiments**: Try new resource combinations
9. **New Scenarios**: Generate valuable new applications

## Output Format

For each suggested task, provide:
```yaml
- title: "Clear, actionable task title"
  description: |
    Detailed description of what needs to be done and why
  type: [problem-resolution|bug-fix|feature|optimization|integration|documentation|testing|experiment|scenario]
  target: "<scenario or resource name>"
  related_problem_id: "<problem-id if this task addresses a discovered problem>"
  estimated_impact: [1-10]
  estimated_complexity: [low|medium|high]
  priority_estimates:
    impact: [1-10]
    urgency: [critical|high|medium|low]
    success_prob: [0.0-1.0]
    resource_cost: [minimal|moderate|heavy]
  rationale: |
    Why this task would be valuable
```

## Examples of Good Tasks

```yaml
- title: "Investigate N8N webhook timeout issues"
  description: |
    Debug and analyze root cause of webhook timeouts >30s causing >5% error rate.
    Check connection pooling, resource limits, and network configurations.
  type: problem-resolution
  target: app-debugger
  related_problem_id: "prob-1735518000"
  estimated_impact: 8
  estimated_complexity: medium
  priority_estimates:
    impact: 8
    urgency: "high"
    success_prob: 0.8
    resource_cost: "moderate"
  rationale: |
    High-frequency problem causing significant workflow failures.
    Must investigate before implementing fix to avoid wrong solution.

- title: "Add retry logic to n8n webhook handler"
  description: |
    N8n webhooks occasionally fail on first attempt. Add exponential
    backoff retry logic to improve reliability.
  type: bug-fix
  target: n8n
  estimated_impact: 7
  estimated_complexity: medium
  priority_estimates:
    impact: 7
    urgency: "medium"
    success_prob: 0.9
    resource_cost: "moderate"
  rationale: |
    Webhook reliability is critical for scenario automation.
    This would prevent false failures and improve system stability.

- title: "Create invoice-generator scenario"
  description: |
    Build a SaaS application for generating and managing invoices
    with PDF export and email capabilities.
  type: scenario
  target: scenario-generator-v1
  estimated_impact: 8
  estimated_complexity: high
  rationale: |
    Invoice generation is a common business need worth $10-50K.
    Would demonstrate Vrooli's business application capabilities.
```

Generate diverse, valuable tasks that will meaningfully improve Vrooli. **Prioritize resolving discovered problems first**, then focus on creative but practical improvements.

## Problem Resolution Strategy

When discovered problems are present:
1. **Always create tasks** for critical and high severity problems
2. **Group related problems** into comprehensive resolution tasks
3. **Consider the full resolution lifecycle**: investigate → fix → prevent
4. **Match problems to appropriate scenarios** based on problem type:
   - Core infrastructure failures → core-debugger (HIGHEST PRIORITY)
   - CLI/orchestrator issues → core-debugger
   - Application failures → app-debugger
   - Performance issues → system-monitor
   - Integration problems → resource-experimenter
   - Missing capabilities → scenario-generator-v1

Focus on making the system more reliable, performant, and autonomous through intelligent problem resolution.