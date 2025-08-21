# Prompt Engineering Guide for Auto/ Loops

## 🎯 Overview

The prompts in the auto/ system are **the intelligence layer** - they contain the actual logic, constraints, and guidance that make Claude Code effective over hundreds of iterations. This guide explains how to craft, maintain, and evolve these prompts for maximum effectiveness.

## 🧠 Core Innovation: Context Persistence

### The Problem
Claude Code has no memory between iterations. Each run starts fresh, leading to:
- Repeated mistakes
- Lost progress
- Drift from objectives
- Inconsistent approaches

### The Solution: Do→Report→Summarize→Repeat
```
Iteration N:
├─> Receive summary from N-1
├─> Understand current state
├─> Make improvements
├─> Report what was done
└─> Generate summary for N+1
```

This creates artificial memory across iterations, maintaining context indefinitely.

## 📝 Anatomy of an Effective Loop Prompt

### 1. TL;DR Section (Critical)
```markdown
## 🚀 [Task Name] Loop

### TL;DR — One Iteration = ONE [specific outcome]
1) [Step 1 - Selection/Analysis]
2) [Step 2 - Investigation/Planning]  
3) [Step 3 - Implementation]
4) [Step 4 - Validation]
5) [Step 5 - Reporting]
```
**Purpose**: Immediate clarity on what ONE iteration should accomplish. Prevents scope creep.

### 2. Purpose & Context
```markdown
### 🎯 Purpose & Context
- [Primary objective]
- [How this fits into Vrooli's vision]
- [Why this matters for recursive improvement]
- [Relationship to other loops]
```
**Purpose**: Grounds Claude in the bigger picture while maintaining focus.

### 3. PRD Compliance Section
```markdown
### 📄 PRD Compliance & Progress Tracking
- **Every [resource/scenario] MUST have a PRD.md**
- Use PRD as single source of truth
- Track implementation status directly in PRD
- Priority order: P0 → P1 → P2
```
**Purpose**: Ensures systematic progress tracking across iterations.

### 4. DO / DON'T Lists (Essential)
```markdown
### ✅ DO / ❌ DON'T

✅ DO:
- [Specific positive actions]
- [Safety constraints]
- [Quality requirements]

❌ DON'T:
- [Common mistakes to avoid]
- [Dangerous operations]
- [Scope violations]
```
**Purpose**: Hard constraints that prevent Claude from destructive or wasteful actions.

### 5. Selection Strategy
```markdown
### 🧭 [Resource/Scenario] Selection
- Tools: [Specific selection scripts]
- Priority rubric:
  1) [Highest priority criteria]
  2) [Second priority]
  3) [Third priority]
```
**Purpose**: Ensures intelligent targeting rather than random selection.

### 6. Validation Gates
```markdown
### ✅ Validation Gates
All must pass:
- [Gate 1: Specific measurable criteria]
- [Gate 2: Another specific check]
- [Gate 3: Final validation]

If any gate fails twice:
- Stop attempts
- Document failure
- Move to next target
```
**Purpose**: Prevents infinite loops on impossible tasks.

### 7. Output Format
```markdown
### 📊 Iteration Summary Format
Append ≤10 lines to `/tmp/vrooli-[task-name].md`:
```yaml
iteration: [number]
target: [what was worked on]
status: [success|partial|failed]
changes: [brief list of changes made]
issues: [problems encountered]
next: [recommendation for next iteration]
```
```
**Purpose**: Structured output enables reliable summary generation.

## 🔧 Prompt Engineering Best Practices

### 1. Maintain Iteration Atomicity
**Principle**: Each iteration must be independently valuable
```markdown
❌ BAD: "Continue working on the PostgreSQL resource"
✅ GOOD: "ONE iteration = ONE specific fix to PostgreSQL health checks"
```

### 2. Use Concrete Examples
**Principle**: Show don't tell
```markdown
❌ BAD: "Validate the resource properly"
✅ GOOD: "Validate: vrooli resource postgres status --json | jq '.health'"
```

### 3. Inject Domain Knowledge
**Principle**: Embed expertise Claude might not have
```markdown
Example from resource-improvement:
"Prefer shared workflows in initialization/n8n/ over direct API calls"
→ This teaches architectural best practices
```

### 4. Define Clear Failure Modes
**Principle**: Know when to stop trying
```markdown
"If webhook registration fails twice, switch to Browserless workaround"
→ Prevents infinite attempts at impossible tasks
```

### 5. Maintain Prompt Hygiene
**Principle**: Keep prompts focused and current
- Remove outdated workarounds
- Update based on observed patterns
- Consolidate repeated instructions
- Keep under 2000 tokens if possible

## 📊 Summary Engineering

### Effective Summary Structure
```python
# What makes a good summary for next iteration:
summary = {
    "completed": ["specific achievements"],
    "blocked": ["specific blockers with context"],
    "patterns": ["repeated issues or successes"],
    "recommendations": ["specific next actions"],
    "metrics": {"efficiency": 0.8, "progress": 0.6}
}
```

### Summary Anti-Patterns
```markdown
❌ Vague: "Worked on resources, some progress made"
❌ Too detailed: [500 lines of command output]
❌ No context: "Error occurred"
❌ No direction: "Iteration complete"

✅ Good: "Fixed PostgreSQL health checks (added /health endpoint).
         Blocked on Redis: port conflicts with existing service.
         Next: Use port-registry.sh for Redis port allocation."
```

## 🔄 Prompt Evolution Patterns

### When to Update Prompts

1. **Repeated Failures** (>3 iterations hitting same issue)
   - Add specific guidance for the issue
   - Example: "If you see 'port already in use', check port-registry.sh"

2. **Discovered Best Practices**
   - Codify patterns that work
   - Example: "Always run 'vrooli resource status' before starting work"

3. **Context Drift** (Claude forgetting objectives)
   - Strengthen TL;DR section
   - Add more specific validation gates

4. **New Capabilities Available**
   - Update tool lists
   - Add new resource dependencies

### How to Update Prompts

1. **Review Recent Logs**
   ```bash
   tail -100 auto/data/[task]/loop.log | grep -A5 "ERROR\|FAILED"
   ```

2. **Identify Patterns**
   - What mistakes repeat?
   - What works consistently?
   - Where does Claude get confused?

3. **Make Minimal Changes**
   - Add specific guidance for identified issues
   - Don't rewrite entire sections
   - Maintain prompt structure

4. **Test Changes**
   ```bash
   # Run single iteration to test
   auto/task-manager.sh --task [task] once
   ```

## 🎯 Advanced Techniques

### 1. Conditional Instructions
```markdown
If resource is storage type:
  - Validate with data persistence test
If resource is automation type:
  - Validate with workflow execution
```

### 2. Progressive Difficulty
```markdown
Iteration 1-10: Focus on P0 requirements only
Iteration 11-20: Include P1 requirements
Iteration 21+: Include P2 if all P0/P1 complete
```

### 3. Cross-Loop Intelligence
```markdown
Before starting, check related loop summaries:
- auto/data/resource-improvement/summary.txt
- auto/data/scenario-improvement/summary.txt
Look for patterns that might affect this task.
```

### 4. Failure Recovery Patterns
```markdown
If previous iteration failed:
1. DO NOT retry exact same approach
2. Check logs for root cause
3. Try alternative approach from this list:
   - [Alternative 1]
   - [Alternative 2]
```

## 📋 Prompt Templates

### Minimal Task Prompt Template
```markdown
## 🚀 [Task Name] Loop

### TL;DR — One Iteration = ONE [outcome]
[5 numbered steps]

### 🎯 Purpose
[Why this task matters]

### ✅ DO / ❌ DON'T
✅ [Key positive actions]
❌ [Key restrictions]

### Validation
- [Success criteria]

### Output
Append to `/tmp/vrooli-[task].md`:
[Structured format]
```

### Selection-Based Task Template
```markdown
[Minimal template above, plus:]

### 🧭 Selection Strategy
Tools: [selection scripts]
Priority: [ordered criteria]

### PRD Tracking
- Check PRD.md exists
- Update completion checkboxes
- Track progress metrics
```

## 🚨 Common Pitfalls

### 1. Overloading Single Iteration
**Problem**: Trying to do too much in one iteration
**Solution**: Enforce "ONE iteration = ONE specific outcome"

### 2. Vague Instructions
**Problem**: "Improve the resource"
**Solution**: "Fix health check endpoint to return JSON status"

### 3. Missing Failure Modes
**Problem**: Claude retries failing operations infinitely
**Solution**: Add explicit "try twice then stop" rules

### 4. Context Explosion
**Problem**: Summaries grow unbounded
**Solution**: Limit summary to key points, rotate old context

### 5. Prompt Drift
**Problem**: Prompts become stale as system evolves
**Solution**: Weekly prompt review and updates

## 🎉 Success Metrics for Prompts

A well-engineered prompt exhibits:
1. **High Success Rate**: >70% iterations produce value
2. **Low Drift**: Stays on objective for 50+ iterations
3. **Good Recovery**: Handles failures gracefully
4. **Clear Progress**: Measurable advancement toward goals
5. **Efficient Summaries**: Context maintained in <500 tokens

## 🔮 Future Directions

### Prompt Self-Improvement
Research into prompts that can identify their own weaknesses and suggest improvements.

### Prompt Composition
Building prompts from modular components that can be mixed and matched.

### Prompt Learning
Using successful iteration patterns to automatically refine prompts.

### Prompt Metrics
Quantitative scoring of prompt effectiveness for A/B testing.

## 📚 Examples

### Excellent Prompt Section
From `resource-improvement-loop.md`:
```markdown
✅ Keep changes to existing resources minimal
✅ Use `vrooli resource <name> <cmd>` and `resource-<name>` CLIs
✅ Redact secrets; use environment variables
✅ Use timeouts for potentially hanging operations
```
**Why it works**: Specific, actionable, safety-focused

### Excellent Validation Gate
From `scenario-improvement-loop.md`:
```markdown
3) Validate (all gates must pass)
   - Convert: `vrooli scenario convert <name> --force`
   - Start: `vrooli app start <name>`
   - Verify: API/CLI outputs match expectations
```
**Why it works**: Concrete commands, clear success criteria

## 🎬 Conclusion

The prompts ARE the intelligence of the auto/ system. They encode:
- Domain knowledge
- Best practices  
- Safety constraints
- Progress tracking
- Recovery strategies

Master prompt engineering, and you master the ability to build self-improving systems with AI.