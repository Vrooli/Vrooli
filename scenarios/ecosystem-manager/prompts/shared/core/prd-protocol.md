# 📋 PRD Protocol: Single Source of Truth

**The PRD is the PRIMARY DELIVERABLE** for generators and the PROGRESS TRACKER for improvers.

{{INCLUDE: ../patterns/prd-essentials.md}}

## Required Sections
1. **Executive Summary** (5 lines): What/Why/Who/Value/Priority
2. **Requirements Checklist** (P0/P1/P2 with checkboxes)  
3. **Technical Specifications** (Architecture/Dependencies/APIs)
4. **Success Metrics** (Completion/Quality/Performance targets)

## Generator Rules (50% of effort)
**MUST**: 5+ P0 requirements, acceptance criteria, technical approach, revenue justification
**Process**: Research → Requirements → Technical → Metrics → PRD.md
**Quality Gate**: All sections filled, revenue > $10K justified

## Improver Rules (Validation & Progress)
**MUST**: Test every ✅, add completion dates, update percentages, document changes
**Validation**: Test → Keep/Uncheck/Note partial → Update history
**Progress Format**: `Date: X% → Y% (Change description)`

## Common Mistakes
**Generators**: Vague requirements, missing criteria, no revenue justification, <5 P0s
**Improvers**: Trusting recent checkmarks are accurate, not updating %, adding unplanned features

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
1. PRD is contract - What's in PRD gets built
2. PRD is truth - Checkmarks must be honest  
3. PRD is progress - Track everything
4. PRD is permanent - Never delete history