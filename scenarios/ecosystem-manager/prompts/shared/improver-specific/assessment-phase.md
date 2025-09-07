# Assessment Phase for Improvers

## Purpose
Improvers enhance EXISTING scenarios and resources. Assessment reveals the TRUE current state, not what documentation claims. This phase is critical for making targeted, valuable improvements.

## Assessment Allocation: 30% of Total Effort

### Assessment Philosophy

**Trust but Verify Everything**
- Documentation lies
- PRDs have false checkmarks  
- Code doesn't match claims
- Tests may not test correctly

Your job is to discover the TRUTH.

## Comprehensive Assessment Process

### Step 1: PRD Reality Check (10%)

#### Verify Every Checkmark
```bash
# For each checked requirement in PRD.md:
1. Find the implementation
   - Search codebase for feature
   - Locate specific functions
   - Identify test coverage

2. Test the functionality
   - Run the actual commands
   - Verify outputs match expectations
   - Check error handling

3. Update PRD accuracy
   - Uncheck false completions
   - Add notes about partial implementations
   - Document blockers

# Example verification:
# PRD says: ✅ User authentication implemented
# Test: curl -X POST localhost:3000/api/login -d '{"user":"test","pass":"test"}'
# Result: 404 Not Found
# Action: Uncheck requirement, note "Endpoint doesn't exist"
```

#### Calculate Real Progress
```markdown
## PRD Accuracy Report
- Claimed completion: 75% (15/20 requirements)
- Verified completion: 35% (7/20 requirements)  
- False positives: 8 requirements
- Partial implementations: 3 requirements
- Blocked items: 2 requirements
```

### Step 2: Functional Assessment (10%)

#### Start and Test Everything
```bash
# Start the component
vrooli scenario [name] run
# OR
vrooli resource [name] start

# Test each major feature
- API endpoints: Test with curl/httpie
- CLI commands: Run each command
- UI features: Screenshot and verify
- Integrations: Check connections

# Document results
✅ Working:
- Health check responds
- Basic API endpoint works
- CLI help command works

❌ Broken:
- Authentication returns 404
- Database connection fails
- UI shows blank page

⚠️ Partial:
- Search works but no pagination
- Upload works but no validation
```

#### Check Resource Dependencies
```bash
# For resources
- Check v2.0 contract compliance
- Test health check with timeout
- Verify lifecycle hooks work
- Validate CLI integration

# For scenarios
- Verify all dependencies running
- Test resource connections
- Check shared workflow access
- Validate initialization
```

### Step 3: Code Quality Assessment (5%)

#### Identify Technical Debt
```bash
# Search for problems
grep -r "TODO\|FIXME\|HACK\|XXX" .
grep -r "hardcoded\|temporary\|workaround" .

# Check for anti-patterns
- Hardcoded values that should be config
- Missing error handling
- No input validation
- Commented out code
- Duplicate code blocks
- Missing tests
```

#### Review Structure
```markdown
## Structure Assessment
- [ ] Follows template structure
- [ ] Consistent naming conventions
- [ ] Proper separation of concerns
- [ ] Clear module boundaries
- [ ] Appropriate abstractions
```

### Step 4: Historical Context Research (5%)

#### Learn from Past Attempts
```bash
# Search Qdrant for history
vrooli resource-qdrant search "[component] improvement" all
vrooli resource-qdrant search "[component] fix" all
vrooli resource-qdrant search "[component] failed" all
vrooli resource-qdrant search "[component] issue" all

# Extract insights
- What's been tried before?
- What worked/failed?
- Are there known issues?
- What patterns emerged?
```

#### Check Cross-Dependencies
```bash
# Find what depends on this
vrooli resource-qdrant search "[component] used by" all
vrooli resource-qdrant search "[component] depends" scenarios

# Assess impact radius
- Which scenarios would break if this changes?
- What relies on current behavior?
- Where are the integration points?
```

## Assessment Output Template

### Required Assessment Report
```markdown
# Assessment Report: [Component Name]

## Executive Summary
- Overall health: [Good/Fair/Poor]
- PRD accuracy: [X]% claimed vs [Y]% actual
- Immediate issues: [Count]
- Improvement opportunities: [Count]

## PRD Validation
### False Completions
1. [Requirement]: [Why it's not actually done]
2. [Requirement]: [Why it's not actually done]

### Partial Implementations
1. [Requirement]: [What works vs what doesn't]

### Blocked Items
1. [Requirement]: [What's blocking it]

## Functional Status
### Working Features
- [Feature]: [Evidence it works]

### Broken Features
- [Feature]: [How it fails]

### Missing Features
- [Feature]: [Not implemented at all]

## Code Quality Issues
### Critical Issues
- [Issue]: [Impact and location]

### Technical Debt
- [Debt item]: [Cost to fix]

### Improvement Opportunities
- [Opportunity]: [Benefit if implemented]

## Historical Context
### Previous Attempts
- [Date]: [What was tried] - [Result]

### Known Issues
- [Issue]: [Workaround if any]

## Cross-Dependencies
### Used By
- [Scenario/Resource]: [How it's used]

### Depends On
- [Resource]: [What it needs]

## Prioritized Improvements
1. [Critical]: [Description] - [Why urgent]
2. [High]: [Description] - [Impact]
3. [Medium]: [Description] - [Value]
4. [Low]: [Description] - [Nice to have]

## Risks
- [Risk]: [Probability and impact]

## Recommendations
### Immediate Actions
1. [Action]: [Expected outcome]

### Next Iteration
1. [Improvement]: [Value delivered]
```

## Assessment Quality Checklist

- [ ] Every PRD checkmark verified with actual test
- [ ] All false completions unchecked with notes
- [ ] Functional testing performed on all features
- [ ] Code quality issues documented
- [ ] Historical context researched
- [ ] Dependencies mapped
- [ ] Improvements prioritized by value
- [ ] Assessment report created

## Common Assessment Pitfalls

### Trusting Documentation
❌ **Bad**: "PRD says it's done, must be done"
✅ **Good**: "PRD says it's done, let me verify... nope, doesn't work"

### Shallow Testing
❌ **Bad**: "Service starts, must be working"
✅ **Good**: "Service starts, but API returns 404, auth is broken, and UI is blank"

### Ignoring History
❌ **Bad**: "Let me try this fresh approach"
✅ **Good**: "This was tried 3 times before and failed because X, let me try Y instead"

### Missing Dependencies
❌ **Bad**: "I'll just fix this in isolation"
✅ **Good**: "5 scenarios depend on this behavior, I need to maintain compatibility"

## Assessment Success Metrics

Good assessment results in:
- **No surprises** during implementation
- **Targeted improvements** that add real value
- **No regressions** from changes
- **Faster implementation** due to clarity
- **Higher success rate** from learning

## Remember for Assessment

**Reality beats documentation** - Test everything yourself

**Details matter** - Small issues cascade into big problems

**History repeats** - Learn from past attempts

**Dependencies constrain** - Know what you can't break

**Time invested pays off** - Good assessment prevents failures

Assessment is your map of the territory. Without accurate assessment, you're working blind. Invest the time to understand the TRUE state - it guides everything that follows.