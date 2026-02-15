# Deep Research: Fix

## Purpose

Perform root cause analysis for a bug or issue, identify all affected components, assess regression risk, and document a safe remediation approach.

## Input Context

- Item folder at `{{ITEM_FOLDER}}`
- `spec.json` containing item metadata (including error details, reproduction steps)
- Any user-added context files (logs, screenshots, stack traces)
- Access to the Vrooli codebase for investigation

## Output Requirements

**Primary output**: `research/summary.md`
**Supporting files**: Add to `research/` as needed (stack traces, code snippets, test cases)

The summary must include:
1. Executive summary (what broke, root cause, fix complexity)
2. Root cause analysis with evidence
3. Affected components map
4. Regression risk assessment
5. Remediation options
6. Testing requirements
7. Recommended fix approach

## Success Criteria

- [ ] Root cause identified with code references
- [ ] All affected components documented
- [ ] Regression risk quantified
- [ ] Fix approach validated against reproduction steps
- [ ] Testing strategy covers the fix and potential regressions
- [ ] Research enables confident, safe remediation

## Instructions

You are researching a bug or issue for the Swarm Manager. Your goal is to understand the problem deeply enough to fix it safely without introducing regressions.

**Context from spec.json:**
- Kind: {{ITEM_KIND}}
- Title: {{ITEM_TITLE}}
- Description: {{ITEM_DESCRIPTION}}
- Priority: {{ITEM_PRIORITY}}
- Tags: {{ITEM_TAGS}}

### Research Steps

1. **Reproduce and confirm**
   - Verify the issue exists as described
   - Document exact reproduction steps
   - Note any variations in behavior

2. **Trace the root cause**
   - Follow the execution path from symptom to source
   - Identify the exact code location(s) causing the issue
   - Distinguish root cause from symptoms
   - Document with file:line references

3. **Map affected components**
   - What code directly uses the buggy code?
   - What features depend on the affected components?
   - Are there integration points with other scenarios?
   - Create a dependency graph if complex

4. **Assess regression risk**
   - What could break if we change this code?
   - Are there existing tests covering this area?
   - What's the blast radius of a bad fix?

5. **Evaluate fix options**
   - Option A: Minimal fix (smallest change)
   - Option B: Proper fix (may involve refactoring)
   - For each: risk level, effort, confidence

6. **Define testing requirements**
   - What tests must pass before merge?
   - What new tests should be added?
   - Manual testing checklist

7. **Recommend approach**
   - Preferred fix with detailed steps
   - Pre-fix checklist
   - Post-fix verification steps

### Output Format

Write `research/summary.md` with this structure:

```markdown
# Fix Research: {{ITEM_TITLE}}

## Executive Summary
[2-3 sentences: what's broken, root cause, fix complexity (trivial/moderate/complex)]

## Problem Statement

### Reported Issue
[What the user/system reported]

### Reproduction Steps
1. [Step 1]
2. [Step 2]
3. [Expected vs actual behavior]

### Environment
- Scenario: [affected scenario]
- Version/commit: [if relevant]

## Root Cause Analysis

### Investigation Path
[How you traced the issue]

### Root Cause
**Location**: `file/path.go:123`
**Issue**: [Detailed explanation of what's wrong]
**Evidence**: [Code snippet, log output, etc.]

### Contributing Factors
- [Factor 1]: [How it contributes]
- [Factor 2]: [How it contributes]

## Affected Components

### Directly Affected
| Component | File | Impact |
|-----------|------|--------|
| [Name] | `path/to/file` | [What breaks] |

### Indirectly Affected
| Component | Dependency Path | Risk |
|-----------|-----------------|------|
| [Name] | [How connected] | [Risk level] |

### Integration Points
- [Scenario/Resource]: [Integration concern]

## Regression Risk Assessment

### Existing Test Coverage
- Unit tests: [coverage status]
- Integration tests: [coverage status]
- E2E tests: [coverage status]

### Risk Matrix
| Change | Risk Level | Confidence |
|--------|------------|------------|
| [Change 1] | Low/Med/High | [Why] |

### Blast Radius
[What could break in worst case, how to detect]

## Remediation Options

### Option A: Minimal Fix
**Change**: [Specific code change]
**Pros**: Smallest change, lowest risk
**Cons**: [Trade-offs]
**Effort**: [S/M/L]
**Confidence**: [High/Med/Low]

### Option B: Proper Fix
**Change**: [More comprehensive change]
**Pros**: Addresses root cause properly
**Cons**: Larger change, more testing needed
**Effort**: [S/M/L]
**Confidence**: [High/Med/Low]

**Recommended**: [Option] because [reasoning]

## Testing Requirements

### Required Tests (Must Pass)
- [ ] [Existing test 1]
- [ ] [Existing test 2]

### New Tests to Add
- [ ] [Test case for the fix]
- [ ] [Regression test]

### Manual Verification
- [ ] [Manual check 1]
- [ ] [Manual check 2]

## Recommended Approach

### Pre-Fix Checklist
- [ ] [Prerequisite 1]
- [ ] [Prerequisite 2]

### Fix Steps
1. [Step 1 with file:line]
2. [Step 2 with file:line]

### Post-Fix Verification
1. [Verify fix works]
2. [Verify no regressions]
```

## Quality Guidelines

**Good fix research:**
- Root cause proven, not guessed
- Affected components exhaustively mapped
- Multiple fix options considered
- Testing strategy comprehensive
- Clear, actionable fix steps

**Poor fix research:**
- Symptom treated as root cause
- Missing component analysis
- Single fix option without alternatives
- No regression consideration
- Vague "fix the bug" instructions

## Anti-Patterns

- **Don't** assume the first hypothesis is correct - verify
- **Don't** ignore related code - check callers and callees
- **Don't** skip regression analysis - it prevents future bugs
- **Don't** recommend fixes without understanding the full picture
- **Don't** forget to check existing tests - they inform risk
