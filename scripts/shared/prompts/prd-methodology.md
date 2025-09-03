# 📄 PRD-Driven Development

## Core Principle

EVERY scenario and resource in Vrooli MUST have a Product Requirements Document (PRD.md) that serves as the single source of truth. The PRD defines what permanent capability is being added to Vrooli and how it generates value.

## PRD Structure

### Location
- Scenarios: `scenarios/[scenario-name]/PRD.md`
- Resources: `resources/[resource-name]/PRD.md`
- Template: `scripts/scenarios/templates/full/PRD.md`

### Required Sections

```markdown
# PRD: [Name]

## Executive Summary
- What problem does this solve?
- Who benefits from this?
- What is the business value?

## Success Metrics
- Quantifiable success criteria
- Performance targets
- Quality benchmarks

## Requirements

### P0 - Must Have (Launch Blockers)
- [ ] Critical functionality
- [ ] Core features
- [ ] Essential integrations

### P1 - Should Have (Important)
- [ ] Important enhancements
- [ ] Quality improvements
- [ ] Extended functionality

### P2 - Nice to Have (Future)
- [ ] Optional features
- [ ] Optimizations
- [ ] Advanced capabilities

## Technical Specifications
- Architecture overview
- Resource requirements
- Integration points
- API contracts

## Validation Criteria
- How to verify each requirement
- Test scenarios
- Acceptance criteria
```

## Working with PRDs

### Before Starting Work

1. **Read the PRD First**
```bash
# Check if PRD exists
if [ -f PRD.md ]; then
    cat PRD.md
else
    echo "ERROR: No PRD found - cannot proceed"
    exit 1
fi
```

2. **Identify Uncompleted Requirements**
```markdown
# Look for unchecked boxes
- [ ] This requirement needs work
- [x] This requirement is complete
```

3. **Prioritize by Level**
- Complete ALL P0 requirements first
- Then P1 requirements
- P2 only after P0 and P1 are done

### During Implementation

1. **Track Progress**
```markdown
# Update PRD as you complete requirements
- [x] ~~Implement user authentication~~ ✅ 2025-01-03
- [x] ~~Add rate limiting~~ ✅ 2025-01-03
- [ ] Add OAuth integration (in progress)
```

2. **Document Deviations**
```markdown
## Implementation Notes
- Requirement X was modified because [reason]
- Added additional requirement Y for [purpose]
- Deferred requirement Z due to [blocker]
```

3. **Update Completion Percentage**
```markdown
## Completion Status
- P0: 100% (5/5 requirements)
- P1: 60% (3/5 requirements)
- P2: 0% (0/4 requirements)
- Overall: 57% (8/14 requirements)
```

### After Completing Work

1. **Verify PRD Compliance**
```bash
# Check all P0 requirements are complete
grep -c "\- \[ \].*P0" PRD.md | grep -q "^0$" || echo "P0 requirements incomplete!"
```

2. **Update Success Metrics**
```markdown
## Success Metrics - Actual Results
- ✅ Response time: 145ms (target: <200ms)
- ✅ Uptime: 99.9% (target: 99.5%)
- ⚠️ User adoption: 67% (target: 80%)
```

3. **Document Lessons Learned**
```markdown
## Post-Implementation Review
- What worked well
- What could be improved
- Recommendations for similar projects
```

## PRD Compliance Scoring

### Calculation Method
```javascript
function calculatePRDCompliance() {
    const p0Complete = countCompleted('P0');
    const p0Total = countTotal('P0');
    const p1Complete = countCompleted('P1');
    const p1Total = countTotal('P1');
    const p2Complete = countCompleted('P2');
    const p2Total = countTotal('P2');
    
    // P0 requirements are mandatory
    if (p0Complete < p0Total) {
        return {
            score: 0,
            status: 'BLOCKED',
            message: `P0 requirements incomplete: ${p0Complete}/${p0Total}`
        };
    }
    
    // Calculate weighted score
    const p1Score = (p1Complete / p1Total) * 30;
    const p2Score = (p2Complete / p2Total) * 20;
    
    return {
        score: 50 + p1Score + p2Score, // 50 points for P0, 30 for P1, 20 for P2
        status: 'IN_PROGRESS',
        details: {
            P0: `${p0Complete}/${p0Total}`,
            P1: `${p1Complete}/${p1Total}`,
            P2: `${p2Complete}/${p2Total}`
        }
    };
}
```

### Status Levels
- **BLOCKED**: P0 requirements incomplete (0-49%)
- **IN_PROGRESS**: P0 complete, P1/P2 in progress (50-94%)
- **COMPLETE**: All P0 and P1 complete (95-99%)
- **PERFECT**: All requirements complete (100%)

## PRD Templates by Type

### Business Application PRD
Focus on:
- Revenue generation potential
- User workflows
- Business metrics
- ROI calculations

### Development Tool PRD
Focus on:
- Developer experience
- Integration capabilities
- Performance benchmarks
- API completeness

### AI/ML Scenario PRD
Focus on:
- Model performance metrics
- Training requirements
- Inference speed
- Accuracy targets

### Infrastructure Resource PRD
Focus on:
- Reliability metrics
- Scalability limits
- Resource consumption
- Health monitoring

## Common PRD Patterns

### Good PRD Requirements
✅ **Specific**: "Response time under 200ms for 95% of requests"
✅ **Measurable**: "Support 1000 concurrent users"
✅ **Achievable**: "Integrate with existing PostgreSQL database"
✅ **Relevant**: "Enable PDF export for invoice generation"
✅ **Time-bound**: "Complete authentication by sprint 2"

### Poor PRD Requirements
❌ **Vague**: "Make it fast"
❌ **Unmeasurable**: "Good user experience"
❌ **Unrealistic**: "100% uptime forever"
❌ **Irrelevant**: "Add feature nobody asked for"
❌ **Open-ended**: "Improve performance"

## PRD Enforcement

### Automated Checks
```bash
# Pre-commit hook example
check_prd_compliance() {
    if [ ! -f PRD.md ]; then
        echo "ERROR: PRD.md required"
        return 1
    fi
    
    # Check for P0 completion
    if grep -q "\- \[ \].*P0" PRD.md; then
        echo "WARNING: P0 requirements incomplete"
    fi
    
    # Check for completion tracking
    if ! grep -q "Completion Status" PRD.md; then
        echo "ERROR: PRD must track completion status"
        return 1
    fi
}
```

### Review Criteria
Before accepting any work:
1. PRD exists and is complete
2. All P0 requirements are checked
3. Completion percentage is calculated
4. Success metrics are measured
5. Validation criteria are verified

## Integration with Other Systems

### Scenario Improver
- Reads PRDs to identify gaps
- Updates completion percentages
- Prioritizes based on requirement levels

### Resource Generator
- Creates PRDs for new resources
- Ensures P0 requirements for v2.0 compliance
- Tracks health monitoring requirements

### Auto Migration
- Extracts requirements from auto/ prompts
- Converts to PRD format
- Maintains requirement tracking

## Remember

- **No PRD = No Development**: Never start without requirements
- **P0 First**: These are non-negotiable
- **Track Everything**: Update PRD as you work
- **Measure Success**: Verify against success metrics
- **Document Deviations**: Explain any changes

The PRD is your contract with the future. It ensures every piece of work adds permanent, measurable value to Vrooli.