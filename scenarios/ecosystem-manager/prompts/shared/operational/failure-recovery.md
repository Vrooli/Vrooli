# Failure Recovery

## Purpose
Document failures clearly and know when to stop making changes. Failures teach us - capture the lessons.

## Failure Classification

### Severity Levels

**Level 1: Trivial (Continue)**
- Cosmetic issues
- Non-critical warnings  
- Minor test failures
- Documentation typos

**Level 2: Minor (Fix and Continue)**  
- Single feature broken
- Performance degradation <20%
- Non-blocking errors
- Partial functionality loss

**Level 3: Major (Stop and Fix)**
- Core feature broken
- Multiple test failures
- API breakage
- Data integrity risk

**Level 4: Critical (Complete Task)**
- System won't start
- Data corruption
- Security vulnerability exposed
- Dependencies broken

**Level 5: Catastrophic (Complete Task)**
- Production system down
- Data loss occurring
- Security breach active

## Recovery Approach

### For Levels 1-3: Try to Fix
Attempt to resolve through editing:
- Make targeted fixes
- Test incremental changes
- Document what you try

### For Levels 4-5: Stop and Complete Task
❌ **DO NOT use git to undo files**
❌ **DO NOT attempt complex rollbacks**

✅ **If you cannot edit your way back to a working state:**
1. Stop making further changes
2. Document the failure clearly
3. Follow the Task Completion Protocol section

## Documenting Failures in Your Response

Include this failure documentation format in your response:

```
### Failure Summary
- **Component**: [resource/scenario name]
- **Severity Level**: [1-5] 
- **What Happened**: [Clear description]
- **Root Cause**: [If identified]

### What Was Attempted
1. [First fix attempt] - Result: [Failed/Partial/Success]
2. [Second fix attempt] - Result: [Failed/Partial/Success]
3. [etc.]

### Current State
- **Status**: [Working/Degraded/Broken]
- **Key Issues**: [List main problems]
- **Workaround**: [If any temporary fix was applied]

### Lessons Learned
- [What this failure teaches us]
- [How to prevent similar issues]
```

## When to Complete the Task

If you reach a point where:
- Multiple fix attempts have failed
- System is in worse state than before
- You're unsure what changes are safe
- Severity is Level 4 or 5

**Then refer to the Task Completion Protocol section** for instructions on how to properly finish and document your work.

## Key Principles

1. **Document everything** - Future agents learn from your experience
2. **No git rollbacks** - Edit your way forward or stop
3. **Know when to stop** - Don't make things worse
4. **Capture lessons** - Every failure teaches something valuable