# PRD Validation & Accuracy Checking

## Validation Process Overview

PRD validation ensures all requirements meet quality standards before implementation begins. Follow these sequential gates:

## Validation Gates

### Gate 1: Structure Check
Verify PRD contains all mandatory sections per:
**{{INCLUDE: shared/core/prd-protocol.md}}**

### Gate 2: Content Quality
- [ ] **Requirements are measurable** - Each has specific acceptance criteria
- [ ] **Success metrics are testable** - Clear pass/fail validation commands
- [ ] **Technical specs are implementable** - Architecture is realistic
- [ ] **Revenue is justified** - Business value is quantified with evidence

### Gate 3: Accuracy Verification
For each requirement:
```bash
# Test the "WHAT" - Is requirement clear?
Can anyone understand what needs to be built? ✓/❌

# Test the "HOW" - Is approach feasible?  
Are technical specs implementable? ✓/❌

# Test the "WHY" - Is value justified?
Is business case compelling? ✓/❌

# Test the "WHEN" - Is timeline realistic?
Are estimates based on similar work? ✓/❌
```

### Gate 4: Testability Check
Every P0 requirement MUST have:
- **Acceptance criteria** - Definition of done
- **Test command** - How to verify it works  
- **Success threshold** - Measurable outcome

Example:
```markdown
**REQ-P0-001**: User authentication
- Criteria: Users login with email/password
- Test: `curl -X POST /api/auth -d '{"email":"test@test.com","password":"test123"}'`
- Success: Returns JWT token in <1 second
```

## Quality Scoring (0-100 points)

### Quick Assessment
- **90-100**: Excellent - Ready for implementation
- **70-89**: Good - Minor improvements needed  
- **50-69**: Fair - Significant gaps to address
- **<50**: Poor - Major revision required

### Scoring Criteria
- Executive Summary: 15 points
- Requirements Quality: 25 points  
- Technical Feasibility: 20 points
- Business Value: 15 points
- Testability: 15 points
- Documentation: 10 points

## Common Validation Failures

### Vague Success Criteria
❌ **Fails**: "Improve user experience"
✅ **Passes**: "Reduce task completion time by 50% (baseline: 2 minutes)"

### Untestable Requirements  
❌ **Fails**: "System should be secure"
✅ **Passes**: "API validates JWT tokens, returns 401 for invalid tokens: `curl -H 'Authorization: Bearer invalid' /api/user`"

### Unrealistic Targets
❌ **Fails**: "Process 1TB data in 1 second"  
✅ **Passes**: "Process 10GB data in <5 minutes (tested with similar datasets)"

## Validation Commands

### Completeness Check
```bash
# Verify all required sections exist
grep -E "## (Executive Summary|P0 Requirements|Technical|Success Metrics)" PRD.md
```

### Measurability Check  
```bash
# Look for specific metrics vs vague language
grep -E "(<|>|%|ms|MB|users|requests)" PRD.md  # Good - specific
grep -E "(fast|good|better|secure|scalable)" PRD.md  # Bad - vague
```

### Test Coverage Check
```bash  
# Count P0 requirements with test commands
grep -A5 "P0.*:" PRD.md | grep -c "curl\|test\|verify"
```

## Approval Criteria

PRD is **validated and approved** when:
- [ ] All 4 validation gates pass
- [ ] Score ≥70 points  
- [ ] All P0 requirements have test commands
- [ ] Revenue justification includes evidence
- [ ] Technical approach references existing patterns

PRD validation prevents implementation delays caused by unclear requirements. Invest validation time upfront to save development time later.