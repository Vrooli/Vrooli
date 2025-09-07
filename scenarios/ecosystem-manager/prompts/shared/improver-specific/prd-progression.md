# PRD Progression for Improvers

## Purpose
PRD progression tracks and advances the implementation of requirements. The PRD is the living contract that guides all work - keeping it accurate is essential.

## PRD Progression Allocation: 10% of Total Effort

### PRD Progression Philosophy

**The PRD is Sacred**
- It defines the permanent capability
- It guides all future work
- It must reflect reality
- It tracks our progress

Never fake progress. Never check boxes without verification.

## PRD Accuracy Standards

### Checkbox Honesty

#### What Justifies a Checkmark ✅
```markdown
A requirement is ONLY complete when:
1. Fully implemented as specified
2. Tests pass consistently  
3. Documentation exists
4. Can be demoed successfully
5. No known issues

Example:
- [ ] User can upload files up to 10MB
Becomes:
- [x] User can upload files up to 10MB
ONLY when:
- Upload endpoint works
- 10MB limit enforced
- Error handling for >10MB
- Success/failure messages
- Tests verify all cases
```

#### Partial Completion Notation
```markdown
Use [~] for partial completion:
- [~] Search with filters
  - ✅ Basic search works
  - ✅ Text search implemented
  - ❌ Date filter not implemented
  - ❌ Category filter not implemented
  - Progress: 50%
```

#### False Completion Correction
```markdown
When finding false completions:
- [x] Email notifications working

Becomes:
- [ ] Email notifications working
  - Note: Was marked complete but testing shows:
  - SMTP not configured
  - Templates missing
  - No error handling
  - Previously checked in error on [date]
```

## Progress Tracking Patterns

### Requirement Lifecycle
```markdown
## Requirement States

1. **Planned** [ ]
   - In PRD but not started
   - No implementation exists

2. **In Progress** [~]
   - Partially implemented
   - Some functionality works
   - List what's done/remaining

3. **Complete** [x]
   - Fully implemented
   - Tests passing
   - Documented

4. **Blocked** [B]
   - Cannot proceed
   - Document blocker
   - Note dependencies

5. **Deferred** [D]
   - Decided to postpone
   - Document reason
   - Note target timeline
```

### Progress Documentation
```markdown
## Implementation Notes Template

### Requirement: [Name]
- Status: [Complete/Partial/Blocked]
- Implemented: [Date or Partial %]
- Location: [Where in code]
- Tests: [Test file/command]
- Verification: `[Command to verify]`
- Notes: [Any important context]

Example:
### Requirement: JWT Authentication
- Status: Complete
- Implemented: 2025-01-07
- Location: api/auth.go:45-120
- Tests: tests/auth_test.go
- Verification: `curl -X POST localhost:3000/api/login -d '{"user":"test","pass":"test"}'`
- Notes: Basic implementation, needs rate limiting
```

## Priority-Driven Progression

### P0 Progression Rules
```markdown
P0 Requirements (Must Have):
- MUST be completed before moving to P1
- Exception: If blocked by external dependency
- Track completion percentage
- Document any deviations

P0 Completion = (Completed P0s / Total P0s) * 100%
Target: 100% before launch
```

### P1 Progression Strategy
```markdown
P1 Requirements (Should Have):
- Complete after all P0s
- Prioritize by value/effort ratio
- OK to defer some to next iteration
- Track as enhancement metrics

P1 Completion = (Completed P1s / Total P1s) * 100%
Target: 80% for good product
```

### P2 Progression Approach
```markdown
P2 Requirements (Nice to Have):
- Only after P0 and critical P1s
- Often become P1s in next version
- Track but don't block on these
- Consider user feedback

P2 Completion = (Completed P2s / Total P2s) * 100%
Target: 30% is excellent
```

## Progression Metrics

### Calculate Real Progress
```python
def calculate_prd_progress(prd):
    # Weight by priority
    p0_weight = 0.6  # 60% of score
    p1_weight = 0.3  # 30% of score
    p2_weight = 0.1  # 10% of score
    
    p0_progress = (p0_completed / p0_total) * p0_weight
    p1_progress = (p1_completed / p1_total) * p1_weight
    p2_progress = (p2_completed / p2_total) * p2_weight
    
    total_progress = p0_progress + p1_progress + p2_progress
    return total_progress * 100  # Percentage

# Example:
# P0: 8/10 complete = 0.8 * 0.6 = 0.48
# P1: 5/8 complete = 0.625 * 0.3 = 0.1875
# P2: 1/5 complete = 0.2 * 0.1 = 0.02
# Total: 68.75% complete
```

### Track Velocity
```markdown
## PRD Velocity Tracking

Week 1: 3 requirements completed (2 P0, 1 P1)
Week 2: 4 requirements completed (2 P0, 2 P1)
Week 3: 2 requirements completed (1 P1, 1 P2)

Average velocity: 3 requirements/week
Estimated completion: 5 weeks remaining
```

## PRD Update Process

### After Each Iteration
```markdown
1. Review what was implemented
2. Test each implementation
3. Update checkboxes accurately
4. Add implementation notes
5. Calculate new progress %
6. Update README with progress
```

### Documentation Requirements
```markdown
## For Each Completed Requirement

Add to PRD:
- [x] Requirement name
  - Implemented: [Date]
  - Version: [Version number]
  - Test: `[Test command]`
  - API: `[Endpoint if applicable]`
  - UI: [Screenshot location if applicable]
```

### Progress Reporting
```markdown
## PRD Progress Report Template

### Current Status
- Overall Progress: X%
- P0 Completion: X% (X/Y requirements)
- P1 Completion: X% (X/Y requirements)  
- P2 Completion: X% (X/Y requirements)

### This Iteration
- Completed: [List requirements]
- Partial Progress: [List requirements]
- Blocked: [List requirements]

### Next Iteration Plan
- Target: [Requirements to complete]
- Estimated Progress: +X%
```

## Common PRD Progression Mistakes

### Premature Completion
❌ **Bad**: Checking off when code is written
✅ **Good**: Checking off when tested and documented

### Vague Progress
❌ **Bad**: "Search is mostly done"
✅ **Good**: "Search: 3/5 features complete (60%)"

### Ignoring Blockers
❌ **Bad**: Leaving blocked items unchecked
✅ **Good**: Marking [B] with explanation

### Progress Inflation
❌ **Bad**: Counting partial as complete
✅ **Good**: Accurate partial notation [~]

## PRD Integrity Verification

### Audit Checklist
```markdown
## PRD Audit

For each checked requirement:
- [ ] Can demo the feature
- [ ] Tests exist and pass
- [ ] Documentation present
- [ ] No known issues
- [ ] Users can actually use it

If any fail: Uncheck and note issue
```

### Regression Prevention
```markdown
## Prevent Progress Regression

Before marking complete:
1. Write test that verifies requirement
2. Add test to CI/CD pipeline
3. Document verification command
4. Add to monitoring dashboard
5. Include in health checks

This ensures once complete, stays complete
```

## Remember for PRD Progression

**Honesty is essential** - False progress hurts everyone

**Partial is OK** - Better to show real state than fake completion

**Tests prove progress** - No test = not complete

**Documentation seals it** - Undocumented features don't exist

**Progress compounds** - Each checkmark enables more

The PRD is the source of truth. Keep it accurate, keep it honest, keep it useful. Your careful tracking enables all future work to build on solid ground.