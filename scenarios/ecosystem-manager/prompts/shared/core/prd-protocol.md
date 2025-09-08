# 📋 PRD Protocol: Single Source of Truth

## PRD Purpose & Importance
**The PRD is the PRIMARY DELIVERABLE** for generators (50% effort) and the PROGRESS TRACKER for improvers.

## PRD Structure (ALL Sections MANDATORY)

### 1. Executive Summary (5 lines MAX)
- **What**: One-line description
- **Why**: Problem being solved  
- **Who**: Target users
- **Value**: Revenue/efficiency gain
- **Priority**: P0/P1/P2 designation

### 2. Requirements Checklist
```markdown
## P0 Requirements (Must Have - Day 1)
- [ ] Requirement 1 with specific acceptance criteria
- [ ] Requirement 2 with measurable outcome
- [ ] Requirement 3 with testable behavior

## P1 Requirements (Should Have - Week 1)  
- [ ] Enhancement 1 with clear benefit
- [ ] Enhancement 2 with user impact

## P2 Requirements (Nice to Have - Month 1)
- [ ] Future feature 1
- [ ] Future feature 2
```

### 3. Technical Specifications
- **Architecture**: Component diagram or bullet list
- **Dependencies**: Required resources/services
- **APIs**: Endpoint definitions
- **Data Models**: Schema/structure
- **Performance**: Target metrics (response time, throughput)

### 4. Success Metrics
- **Completion**: X% of P0 done = launched
- **Quality**: Test coverage target
- **Performance**: Response time < Xms
- **Adoption**: Target usage numbers

## Generator PRD Rules (50% of effort)

### MUST Include
✅ **5+ P0 requirements** minimum
✅ **Specific acceptance criteria** for each requirement
✅ **Technical implementation** approach
✅ **Revenue potential** with justification
✅ **Competitive analysis** (what exists elsewhere)

### Creation Process
1. Research findings → Requirements
2. Requirements → Technical approach
3. Technical approach → Implementation plan
4. Implementation plan → Success metrics
5. Everything → Single PRD.md file

### Quality Gates for Generator PRDs
- [ ] All sections present and filled
- [ ] 5+ P0 requirements defined
- [ ] Each requirement has acceptance criteria
- [ ] Technical approach is specific
- [ ] Revenue potential > $10K justified

## Improver PRD Rules (Validation & Progress)

### MUST Do
✅ **Test every ✅** - Uncheck if broken
✅ **Add completion date** when checking boxes
✅ **Update percentages** after changes
✅ **Document why** unchecking items
✅ **Keep history** of changes at bottom

### Validation Process (20% of assessment)
```markdown
For each ✅ in PRD:
1. Test the feature
2. If works: Keep checked, add date
3. If broken: Uncheck, add "BROKEN: [error]"
4. If partial: Note "PARTIAL: [what works]"
```

### Progress Tracking
```markdown
## Progress History
- 2024-01-15: 20% → 35% (Added user auth)
- 2024-01-16: 35% → 30% (Unchecked broken features)
- 2024-01-17: 30% → 45% (Fixed auth, added API)
```

## PRD Templates Location
- **Resources**: `/templates/prd-resource-template.md`
- **Scenarios**: `/templates/prd-scenario-template.md`

Use templates as STARTING POINT, customize for specific needs.

## Common PRD Mistakes to AVOID

### For Generators
❌ **Vague requirements** like "good UX"
❌ **Missing acceptance criteria**
❌ **No revenue justification**
❌ **Copying PRDs without customization**
❌ **Under 5 P0 requirements**

### For Improvers
❌ **Trusting checkmarks without testing**
❌ **Not updating percentages**
❌ **Adding features not in PRD**
❌ **Checking boxes for partial work**
❌ **No documentation of changes**

## PRD Checkbox Rules

### When to Check ✅
- Feature FULLY works as specified
- Tests pass for that feature
- Documentation exists
- No known bugs

### When to Uncheck ☐
- Feature broken/missing
- Tests fail
- Major bugs present
- Spec not met

### Partial Completion
Use notes, not checkmarks:
```markdown
- [ ] User authentication (PARTIAL: login works, logout broken)
```

## Time Allocation for PRDs

### Generators (50% of total effort)
- Research → PRD: 20%
- Writing PRD: 20%
- Technical specs: 10%

### Improvers (Within 20% assessment)
- Validation: 10%
- Updating: 5%
- Progress tracking: 5%

## PRD Success Metrics

### Good Generator PRD
- Reader understands WHAT to build
- Reader understands WHY to build it
- Reader knows HOW to build it
- Reader can TEST if it works

### Good Improver PRD Update
- Checkmarks match reality
- Progress is trackable
- History is preserved
- Next steps are clear

## Golden Rules

1. **PRD is contract** - What's in PRD gets built
2. **PRD is truth** - Checkmarks must be honest
3. **PRD is progress** - Track everything
4. **PRD is permanent** - Never delete history

Remember: **Bad PRD = Bad Product. Good PRD = Good Product.**