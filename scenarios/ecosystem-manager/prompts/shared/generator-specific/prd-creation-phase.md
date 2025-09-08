# PRD Creation Phase - PRIMARY DELIVERABLE

## Purpose
The PRD is the **PRIMARY DELIVERABLE** for generators. A comprehensive PRD enables future improvements and defines the permanent capability being added to Vrooli.

## PRD Structure Requirements

### Executive Summary (5 lines MAX)
- **What**: One-line description
- **Why**: Problem being solved  
- **Who**: Target users/scenarios
- **Value**: Problem solved and efficiency gained
- **Priority**: P0/P1/P2 designation

### Requirements Checklist
```markdown
## P0 Requirements (Must Have - Day 1)
- [ ] Requirement with specific acceptance criteria
- [ ] Requirement with measurable outcome
- [ ] Requirement with testable behavior
[MINIMUM 5 P0 requirements required]

## P1 Requirements (Should Have - Week 1)
- [ ] Enhancement with clear benefit
- [ ] Enhancement with user impact

## P2 Requirements (Nice to Have - Month 1)
- [ ] Future feature possibilities
```

### Technical Specifications
- **Architecture**: Component diagram or detailed bullet list
- **Dependencies**: Required resources/services
- **APIs**: Endpoint definitions with request/response
- **Data Models**: Schema/structure
- **Performance**: Specific metrics (response time < Xms)

### Success Metrics
- **Completion**: % of P0 done = launched
- **Quality**: Test coverage target
- **Performance**: Response time targets
- **Value**: Clear user benefit achieved

## Generator PRD Creation Process

### Step 1: Synthesize Research
Transform your research findings into requirements:
- Each Qdrant finding → Potential requirement
- Each external reference → Technical approach
- Each pattern identified → Implementation strategy

### Step 2: Define Requirements
Create specific, measurable requirements:
- **P0**: Core functionality that MUST work day 1
- **P1**: Enhancements that add significant value
- **P2**: Future possibilities worth documenting

Each requirement needs:
- Clear acceptance criteria
- Testable behavior
- Measurable outcome

### Step 3: Technical Design
Document HOW to build it:
- Architecture approach
- Resource dependencies
- API design
- Database schema
- Performance targets

### Step 4: Success Metrics
Define what success looks like:
- Completion criteria
- Quality metrics
- Performance benchmarks
- Value validation

## Quality Gates for Generator PRDs

**Quality Checkpoints:**
- [ ] 5+ P0 requirements with acceptance criteria
- [ ] Technical approach is specific and implementable
- [ ] Clear value proposition with justification
- [ ] All 4 sections complete (Summary, Requirements, Technical, Success)
- [ ] Research findings incorporated into requirements

## Common PRD Creation Mistakes

❌ **Vague requirements** like "good UX" or "fast performance"
✅ **Specific requirements** like "Response time < 200ms for all API calls"

❌ **Missing acceptance criteria** that can't be tested
✅ **Clear criteria** like "User can create invoice in < 3 clicks"

❌ **No value justification** or vague benefits
✅ **Solid justification** like "Saves 2 hours per week for 100 users"

❌ **Copy-pasting from other PRDs** without customization
✅ **Tailored requirements** based on specific research

## PRD Output Location

Save the PRD as `PRD.md` in the scenario/resource root directory.

## Remember

**The PRD is your PRIMARY deliverable.** Future improvers will use it as their guide. Make it comprehensive, accurate, and actionable. Every hour spent on a good PRD saves 10 hours of future confusion.

For complete PRD rules and additional examples, see: `shared/core/prd-protocol.md`