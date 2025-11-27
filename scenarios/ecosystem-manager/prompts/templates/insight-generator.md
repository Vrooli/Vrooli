# Execution Analysis & Insight Generation

You are analyzing execution history to identify patterns, diagnose issues, and suggest specific improvements to prompts, code, and configuration.

## Task Context

**Task ID**: {{TASK_ID}}
**Title**: {{TITLE}}
**Analysis Target**: {{TARGET}}
**Your Mission**: Examine execution failures and patterns to generate actionable improvement suggestions

## Analysis Data

**Execution Window**: Last {{EXECUTION_COUNT}} executions ({{STATUS_FILTER}})
**Success Rate**: {{SUCCESS_RATE}}%
**Average Duration**: {{AVG_DURATION}}
**Most Common Exit**: {{MOST_COMMON_EXIT_REASON}}

### Recent Executions

{{EXECUTIONS_SUMMARY}}

## Your Analysis Framework

### 1. Pattern Identification

Identify **recurring patterns** in failed/stalled executions:

**Pattern Types to Look For:**
- **failure_mode**: Specific errors that repeat (build failures, test failures, runtime errors)
- **timeout**: Where agents get stuck (specific phases, commands, waiting for responses)
- **rate_limit**: Token usage patterns causing API rate limits
- **stuck_state**: Agent loops, indecision, or circular reasoning

**For Each Pattern:**
- **Frequency**: Count how many executions show this (e.g., "6 out of 10 recent executions")
- **Severity**: Assess impact (critical/high/medium/low)
  - critical: Blocks all progress
  - high: Blocks most attempts
  - medium: Occasional blocker
  - low: Minor inconvenience
- **Evidence**: Extract specific log excerpts that prove the pattern
- **Examples**: List execution IDs that demonstrate this pattern

### 2. Root Cause Analysis

For each identified pattern, determine the **underlying cause**:

**Prompt Issues:**
- Unclear or conflicting instructions
- Missing context needed for decisions
- Assumptions that don't match reality
- Over-constraining or under-constraining guidance

**Timeout Issues:**
- Insufficient time for legitimate work
- Phase timeout too short for actual command execution
- No buffer for agent reasoning and iteration

**Code Issues:**
- Bugs preventing successful execution
- Missing error handling
- Inefficient implementations causing slow execution
- Dependencies or environment issues

**Auto Steer Issues:**
- Phase configuration problems
- Quality gate thresholds too strict/loose
- Poor phase transitions
- Steering mode mismatch with actual work needed

### 3. Suggestion Generation

For each root cause, propose **specific, actionable changes**:

**Requirements for Good Suggestions:**
- **Specific**: Exact file paths, exact changes, no vague advice
- **Actionable**: Can be applied immediately without further research
- **Measurable**: Clear success criteria
- **Evidence-based**: Justified by the data you analyzed

**Suggestion Types:**
- **prompt**: Changes to prompt templates (add context, clarify instructions, remove conflicts)
- **timeout**: Adjust timeout values in task config or Auto Steer profiles
- **code**: Fix bugs, improve error handling, optimize performance
- **autosteer_profile**: Modify phase configuration, quality gates, or steering modes

**For Each Suggestion:**
```json
{
  "type": "timeout|prompt|code|autosteer_profile",
  "priority": "critical|high|medium|low",
  "title": "Brief, specific title",
  "description": "Detailed explanation of what to change and why",
  "changes": [
    {
      "file": "relative/path/from/scenario/root",
      "type": "edit|create|config_update",
      "description": "What this change does",

      // For type: "edit"
      "before": "exact text to find",
      "after": "exact replacement text",

      // For type: "create"
      "content": "full file content",

      // For type: "config_update"
      "config_path": "phases[2].timeout_minutes",
      "config_value": 90
    }
  ],
  "impact": {
    "success_rate_improvement": "+30-40%",
    "time_reduction": "-10m avg",  // optional
    "confidence": "high|medium|low",
    "rationale": "Why you expect this impact, based on the data"
  }
}
```

### 4. Priority Assessment

**Prioritize suggestions by:**
1. **Impact**: How many executions would this fix?
2. **Confidence**: How certain are you this will work?
3. **Risk**: How likely to cause regressions?
4. **Effort**: How complex is the change?

**Priority Guidelines:**
- **critical**: Affects >75% of executions, high confidence fix
- **high**: Affects >50% of executions, medium-high confidence
- **medium**: Affects >25% of executions, or nice-to-have improvements
- **low**: Optimization opportunities, affects <25%

## Output Format

Respond with a **valid JSON object** (not markdown, just the JSON):

```json
{
  "patterns": [
    {
      "id": "0",
      "type": "timeout|failure_mode|rate_limit|stuck_state",
      "frequency": 6,
      "severity": "critical|high|medium|low",
      "description": "Clear description of the pattern",
      "examples": ["execution-id-1", "execution-id-2"],
      "evidence": [
        "Exact log excerpt 1",
        "Exact log excerpt 2"
      ]
    }
  ],
  "suggestions": [
    {
      "id": "0",
      "pattern_id": "0",
      "type": "timeout|prompt|code|autosteer_profile",
      "priority": "critical|high|medium|low",
      "title": "Specific, actionable title",
      "description": "Detailed explanation with justification",
      "changes": [
        {
          "file": "path/to/file",
          "type": "edit|create|config_update",
          "description": "What this does",
          "before": "...",
          "after": "..."
        }
      ],
      "impact": {
        "success_rate_improvement": "+X-Y%",
        "confidence": "high|medium|low",
        "rationale": "Data-driven justification"
      }
    }
  ]
}
```

## Critical Guidelines

1. **Be Specific**: No vague suggestions like "improve prompt clarity"
   - Bad: "Make the prompt clearer"
   - Good: "Add explicit instruction in prompts/phases/testing.md line 15: 'Run tests with --timeout 300 flag to allow for slow integration tests'"

2. **Provide Evidence**: Quote actual log excerpts
   - Don't say "tests often fail"
   - Say "Tests timeout in 6/10 executions with message: 'ERROR: go test timed out after 60m'"

3. **Estimate Impact Using Data**: Base estimates on execution data
   - If 6/10 failed due to timeout: "+50-60% success rate improvement"
   - If 2/10 had this issue: "+15-20% success rate improvement"

4. **Consider Interactions**: Note if suggestions conflict
   - Example: If increasing timeout and improving test speed both solve the same issue, note they're alternatives

5. **Prioritize Ruthlessly**: Only suggest changes with measurable benefit
   - Must have clear evidence in the execution data
   - Must have realistic path to implementation
   - Must estimate concrete impact

6. **Output Valid JSON**: No markdown fences, no explanatory text, just the JSON object

## Special Cases

**No Failures Found:**
```json
{
  "patterns": [],
  "suggestions": [
    {
      "id": "0",
      "pattern_id": null,
      "type": "code",
      "priority": "low",
      "title": "Consider optimization opportunities",
      "description": "All executions succeeded. Focus on performance or code quality improvements.",
      "changes": [],
      "impact": {
        "success_rate_improvement": "0%",
        "time_reduction": "potential",
        "confidence": "low",
        "rationale": "No failures to address, only optimization opportunities"
      }
    }
  ]
}
```

**All Successful But Slow:**
Focus on timeout reduction, code optimization, or process improvements.

**Parse Errors Expected:**
If your JSON is malformed, the system will ask you to retry. Double-check your JSON syntax.

## Execution Details Follow

The following sections contain the actual execution data for analysis:

{{EXECUTION_DETAILS}}

---

**Remember**: Your goal is to provide actionable, evidence-based suggestions that will measurably improve execution success rates. Be specific, be precise, and justify every suggestion with data.
