# Process: Fix

## Purpose

Apply a well-researched fix to the codebase, following the remediation plan from research. Ensure the fix is correct, doesn't introduce regressions, and is properly verified.

## Input Context

**Required reading:** `prompt-manager skill read swarm-manager-processing-guidance` — shared processing workflow and decision hierarchy.

## Output Requirements

- Apply fix to affected codebase files
- Add/update tests covering the fix
- Write `notes.md` in item folder with:
  - What was changed and why
  - Files modified with specific changes
  - Verification evidence
  - Any follow-up needed

## Success Criteria

- [ ] Bug no longer reproduces
- [ ] Root cause addressed (not just symptoms)
- [ ] Existing tests pass
- [ ] New test covers the fix
- [ ] No regressions in related functionality
- [ ] Completion summary documents the fix

## Instructions

You are applying a fix for a bug in the Vrooli codebase. Your goal is to eliminate the bug safely without introducing regressions.

**Context from spec.json:**
- Kind: {{ITEM_KIND}}
- Title: {{ITEM_TITLE}}
- Description: {{ITEM_DESCRIPTION}}
- Priority: {{ITEM_PRIORITY}}
- Tags: {{ITEM_TAGS}}

### Processing Steps

1. **Review the research**
   - Read `research/summary.md` thoroughly
   - Understand the root cause
   - Review the recommended fix approach
   - Note any risks identified

2. **Verify current state**
   - Confirm the bug still reproduces
   - Identify the exact code locations to modify
   - Check existing test coverage

3. **Plan the fix**
   - List specific changes to make
   - Identify files to modify
   - Plan new tests to add

4. **Apply the fix**
   - Make minimal, focused changes
   - Address root cause, not symptoms
   - Follow existing code patterns
   - Add comments if logic isn't obvious

5. **Add test coverage**
   - Test that reproduces the bug (should fail before fix)
   - Test that verifies the fix (should pass after)
   - Regression tests for related functionality

6. **Verify the fix**
   - Run existing tests
   - Run new tests
   - Manually verify bug no longer reproduces
   - Check for regressions

7. **Document the fix**
   - Write completion summary
   - Note all changes made
   - Document verification evidence

### Fix Implementation Guidelines

#### Minimal Changes
```go
// BAD: Refactoring while fixing
func processItem(item Item) error {
    // Completely rewritten function
    // Risk: introduces new bugs
}

// GOOD: Targeted fix
func processItem(item Item) error {
    // ... existing code ...

    // Fix: Check for nil before access
    if item.Data == nil {
        return ErrNilData
    }

    // ... rest of existing code ...
}
```

#### Test for the Fix
```go
func TestProcessItem_NilData(t *testing.T) {
    // This test would fail before the fix
    item := Item{Data: nil}
    err := processItem(item)

    if err != ErrNilData {
        t.Errorf("expected ErrNilData, got %v", err)
    }
}
```

#### Completion Summary Template
```markdown
# Fix Applied: {{ITEM_TITLE}}

## Root Cause
[Brief restatement from research]

## Changes Made

### File: `path/to/file.go`
**Lines**: 123-130
**Change**: Added nil check before accessing item.Data
**Reason**: Prevents panic when Data is uninitialized

### File: `path/to/file_test.go`
**Added**: TestProcessItem_NilData
**Purpose**: Verifies fix handles nil Data correctly

## Verification

### Automated Tests
- [x] All existing tests pass
- [x] New test TestProcessItem_NilData passes
- [x] Integration tests pass

### Manual Verification
- [x] Reproduction steps no longer trigger bug
- [x] Related functionality still works

## Regression Check
- [x] Feature A still works
- [x] Feature B still works
- [x] No new warnings in logs

## Follow-up
- Consider adding more edge case tests
- Related code in file2.go might have same issue (separate ticket)
```

### Fix Quality Standards

- **Correctness**: Fix addresses root cause
- **Safety**: No new bugs introduced
- **Minimality**: Smallest change that fixes the issue
- **Testability**: Fix is covered by tests
- **Clarity**: Changes are understandable

## Quality Guidelines

**Good fix:**
- Addresses root cause from research
- Minimal, focused changes
- Comprehensive test coverage
- Clear documentation of what changed
- No regressions

**Poor fix:**
- Fixes symptoms, not cause
- Over-engineers or refactors
- No new tests
- No verification evidence
- Leaves related issues

## Anti-Patterns

- **Don't** ignore the research - it contains the analysis
- **Don't** refactor while fixing - separate concerns
- **Don't** skip tests - they prove the fix works
- **Don't** fix more than the bug - scope creep
- **Don't** leave the fix undocumented

## Edge Cases

### If Research is Missing
- STOP: Do not apply fix without understanding root cause
- Request deep research mode first

### If Bug No Longer Reproduces
- Investigate: Was it fixed elsewhere?
- Document: Note in summary that bug couldn't be reproduced
- Close: May already be resolved

### If Fix Has High Regression Risk
- Consider: Is the minimal fix approach safer?
- Test: Add extra regression tests
- Document: Note risks in summary
