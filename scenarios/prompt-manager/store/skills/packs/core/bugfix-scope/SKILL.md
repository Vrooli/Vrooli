---
name: "bugfix-scope"
description: "Minimal targeted changes with regression tests"
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["scope"]
  tags: ["scope","bugfix","constraints"]
  icon: "bug"
  status: "active"
  revision: 19
  createdAt: "2026-02-04T12:00:00Z"
  updatedAt: "2026-02-04T13:13:54Z"
  requires:
    scenarios: []
    commands: []
  origin:
    kind: "authored"
---
# Bugfix Scope

Session constraints for bug fixes. These boundaries ensure fixes are targeted and don't introduce new issues.

## Session Boundaries

### ALLOWED
- Fixing the specific reported bug
- Adding a regression test for the bug
- Minimal code changes to resolve the issue
- Updating related tests if they were incorrect
- Adding defensive checks directly related to the bug
- Refactoring code directly related to the bug to increase testability (e.g. testing seams)

### NOT ALLOWED
- Refactoring unrelated code
- Adding new features
- "While I'm here" improvements
- Changing code style in unaffected areas
- Modifying unrelated tests
- Architectural changes

## Quality Requirements

1. **Minimal Diff**: Changes should be limited to fixing the issue and related test quality/coverage
2. **Regression Test**: Add a test that would have caught this bug
3. **Root Cause**: Fix the actual cause, not just symptoms
4. **Isolation**: Don't mix bug fixes with other changes

## Debugging Process

1. **Reproduce**: Verify the bug can be reproduced
2. **Isolate**: Identify the exact code causing the issue
3. **Fix**: Make the minimal change to resolve it
4. **Test**: Add regression test before applying fix
5. **Verify**: Confirm the fix resolves the issue

## Verification Checklist

Before completing any bugfix task:
- [ ] Bug is reproduced and understood
- [ ] Regression test added (fails before fix, passes after)
- [ ] Fix is minimal and targeted
- [ ] All existing tests pass
- [ ] No new changes made outside the target scope (changes may exist in other parts of the project due to parallel agents - leave these be)
