# Iteration Phase for Improvers

## Purpose
Iteration is where prioritized improvements become reality. The key is making focused, incremental changes that advance the PRD while maintaining system stability.

## Iteration Allocation: 40% of Total Effort

### Iteration Philosophy

**Incremental and Irreversible Progress**
- Each iteration moves forward
- Never break working features
- Small steps compound
- Progress is permanent

Think of it like climbing a mountain - each step up is secure before taking the next.

## Iteration Execution Process

### Step 1: Pre-Iteration Setup (5%)

#### Create Safety Checkpoint
```bash
# Record current state
vrooli scenario [name] health > /tmp/before-health.txt
# OR
vrooli resource [name] test > /tmp/before-test.txt

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

### Step 2: Implementation (30%)

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

**Adding Authentication**
```javascript
// Step 1: Add auth middleware
const authMiddleware = (req, res, next) => {
    // New auth logic
    if (requiresAuth(req.path)) {
        validateToken(req, res, next);
    } else {
        next(); // Don't break existing public routes
    }
};

// Step 2: Apply selectively
app.use('/api/protected', authMiddleware);
// Public routes remain unchanged

// Step 3: Test both paths work
```

**Improving Health Checks**

For comprehensive health check implementations, patterns, and best practices:
{{INCLUDE: resource-specific/health-checks.md}}

**Adding Validation**
```python
# Step 1: Add validation function
def validate_input(data):
    errors = []
    if not data.get('required_field'):
        errors.append('required_field missing')
    if len(data.get('name', '')) > 100:
        errors.append('name too long')
    return errors

# Step 2: Apply with fallback
def process_request(data):
    # New validation
    errors = validate_input(data)
    if errors and strict_mode:
        return {'error': errors}, 400
    
    # Existing logic continues
    return original_process(data)
```

### Step 3: Validation (5%)

#### Test the Specific Change
```bash
# Test new functionality
echo "Testing new feature..."
curl -X POST localhost:3000/api/login -d '{"user":"test","pass":"test"}'
# Expected: JWT token

# Verify old functionality still works
echo "Testing existing features..."
curl localhost:3000/api/health
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

### Step 4: PRD Update (5%)

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

## Iteration Patterns by Type

### Bug Fix Iterations
```bash
# 1. Reproduce the bug
# 2. Write failing test
# 3. Fix the bug
# 4. Verify test passes
# 5. Check no new issues
```

### Feature Addition Iterations
```bash
# 1. Add feature flag/config
# 2. Implement behind flag
# 3. Test with flag on
# 4. Test with flag off
# 5. Enable when stable
```

### Performance Iterations
```bash
# 1. Measure baseline
# 2. Make optimization
# 3. Measure improvement
# 4. Verify functionality unchanged
# 5. Document gains
```

### Refactoring Iterations
```bash
# 1. Add comprehensive tests
# 2. Refactor small section
# 3. Run all tests
# 4. Repeat for next section
# 5. Remove old code when safe
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