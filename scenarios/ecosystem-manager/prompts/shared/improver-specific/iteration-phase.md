# Iteration Phase for Improvers

## Purpose
Iteration is where prioritized improvements become reality. The key is making focused, incremental changes that advance the PRD while maintaining system stability.

## Iteration Philosophy

**Incremental and Irreversible Progress**
- Each iteration moves forward
- Never break working features
- Small steps compound
- Progress is permanent

Think of it like climbing a mountain - each step up is secure before taking the next.

## Iteration Execution Process

### Step 1: Pre-Iteration Setup

#### Create Safety Checkpoint
```bash
# Record current state
{{STANDARD_HEALTH_CHECK}} > /tmp/before-health.txt
# OR
./test.sh > /tmp/before-test.txt

# Note working features
echo "Working before changes:" > /tmp/iteration-log.txt
echo "- Feature A works" >> /tmp/iteration-log.txt
echo "- Feature B works" >> /tmp/iteration-log.txt

# Document rollback plan
echo "If changes fail:" >> /tmp/iteration-log.txt
echo "1. Revert file X to previous state" >> /tmp/iteration-log.txt
echo "2. Restart service" >> /tmp/iteration-log.txt
```

#### Define Success Criteria
```markdown
## This Iteration's Goals
Primary: Implement API authentication (P0)
Success Criteria:
- [ ] /api/login endpoint responds
- [ ] Valid credentials return JWT token
- [ ] Invalid credentials return 401
- [ ] Protected endpoints require token
- [ ] Existing endpoints still work
```

### Step 2: Implementation

#### The Golden Rule: One Change at a Time
```bash
# BAD: Multiple changes at once
- Add authentication
- Refactor database
- Update UI
- Fix logging
Result: Something breaks, unclear what

# GOOD: Single focused change
- Add authentication
- Test thoroughly
- Commit
- Then move to next
Result: Clear progress, easy debugging
```

#### Implementation Pattern
```python
# 1. Add without breaking
def enhanced_function(params):
    # Keep existing behavior
    result = original_logic(params)
    
    # Add new capability
    if new_feature_enabled:
        result = enhance_result(result)
    
    return result

# 2. Test the addition
# 3. Enable gradually
# 4. Remove old code only when safe
```

#### Common Iteration Patterns

**Common Iteration Patterns:**
```pseudo
AUTHENTICATION:
  add_middleware(auth_check) → protected_routes
  preserve_existing(public_routes) → no_auth_required
  test(protected + public) → both_work
  
VALIDATION:
  add_validator(input_rules) → error_list  
  apply_conditionally(strict_mode) → fail_or_continue
  maintain_fallback(existing_logic) → no_regression

HEALTH_CHECKS:
  <!-- health-checks.md already included in base sections -->
```

### Step 3: Validation

#### Test the Specific Change
```bash
# Test new functionality
echo "Testing new feature..."
curl -X POST localhost:3000/api/login -d '{"user":"test","pass":"test"}'
# Expected: JWT token

# Verify old functionality still works
echo "Testing existing features..."
{{STANDARD_HEALTH_CHECK}}/api
# Expected: {"healthy": true}

# Run existing tests
echo "Running test suite..."
./tests/test.sh
# Expected: All pass
```

#### Verify No Regressions
```markdown
## Regression Check
- [ ] All previously working features still work
- [ ] Performance hasn't degraded
- [ ] No new errors in logs
- [ ] Dependencies still connect
- [ ] Resource usage acceptable
```

#### Security Validation (MANDATORY)
**Every iteration must pass security validation** - See base security requirements.

Run Phase 3 (Validation Security) checks after every change.

### Step 4: PRD Update

#### Update PRD Accurately
```markdown
## Before
- [ ] User authentication with JWT

## After  
- [x] User authentication with JWT
  - Implemented: 2025-01-07
  - Endpoints: /api/login, /api/refresh
  - Test: `curl -X POST localhost:3000/api/login -d '{"user":"test","pass":"test"}'`
  - Note: Basic implementation, needs rate limiting in future
```

#### Document Partial Progress
```markdown
## Partial Implementation Example
- [~] Search functionality
  - ✅ Basic search works
  - ✅ Returns results
  - ❌ No pagination yet
  - ❌ No filters yet
  - Test: `curl localhost:3000/api/search?q=test`
```


## Iteration Anti-Patterns

### The Big Bang
❌ **Bad**: Rewrite everything at once
✅ **Good**: Incremental replacement

### The Perfectionist
❌ **Bad**: Trying to fix everything in one iteration
✅ **Good**: One improvement, done well

### The Cowboy
❌ **Bad**: Making changes without tests
✅ **Good**: Test before, during, and after

### The Scope Creeper
❌ **Bad**: "While I'm here, let me also..."
✅ **Good**: Stick to planned improvement

## Iteration Success Metrics

Track per iteration:
- **Change Success**: Did it work first try?
- **Time Accuracy**: Actual vs estimated time
- **No Regressions**: Nothing broken?
- **PRD Progress**: Requirements advanced?
- **Value Delivered**: User benefit achieved?

## Iteration Completion Checklist

Before marking iteration complete:
- [ ] Primary goal achieved
- [ ] All tests passing
- [ ] No regressions introduced
- [ ] PRD updated accurately
- [ ] Documentation current
- [ ] Changes are reversible if needed
- [ ] Memory (Qdrant) will be updated

## Recovery from Failed Iterations

If iteration fails:
1. **Stop immediately** - Don't dig deeper
2. **Assess damage** - What's broken?
3. **Rollback if needed** - Restore working state
4. **Document failure** - What went wrong?
5. **Update approach** - How to succeed next time
6. **Create smaller iteration** - Reduce scope

## Remember for Iterations

**Small steps win** - Consistent small progress beats sporadic large attempts

**Test obsessively** - If it's not tested, assume it's broken

**Preserve working code** - Never break what works

**Document reality** - PRD should reflect actual state

**Learn from each iteration** - Failures teach valuable lessons

Each iteration is a permanent step forward. Make it count. Make it solid. Make it irreversible progress toward the goal.

## When Iteration Work Is Complete

After completing all iteration changes, follow the comprehensive completion protocol:

<!-- task-completion-protocol.md already included in base sections -->

Proper task completion ensures your iteration work becomes permanent knowledge that enhances all future agents.