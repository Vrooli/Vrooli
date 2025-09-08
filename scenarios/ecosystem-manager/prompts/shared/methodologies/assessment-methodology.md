# Assessment Methodology

## Purpose
Assessment is critical for improvers to understand the TRUE state of a resource or scenario before making changes. Many PRDs have incorrectly checked items, and code often doesn't match documentation.

## Assessment Phases (Minimum 30% of Total Effort)

### Phase 1: PRD Accuracy Verification (10%)

#### Read and Validate PRD
Never trust PRD checkboxes without verification:

```bash
# For each checked requirement in PRD:
1. Locate the implementation
2. Test the functionality
3. Verify it actually works
4. Uncheck if not truly complete

# For each unchecked requirement:
1. Confirm it's actually missing
2. Check if partially implemented
3. Note any blockers or dependencies
```

#### PRD Validation Checklist
- [ ] Every P0 requirement tested and verified
- [ ] Every P1 requirement checked for accuracy
- [ ] False completions unchecked with notes
- [ ] Actual completion percentage calculated
- [ ] Missing dependencies documented

### Phase 2: Current State Analysis (10%)

#### Code Assessment
```bash
# Check implementation quality
- Review directory structure against standards
- Verify all required files exist
- Check for TODO/FIXME/HACK comments
- Identify incomplete implementations
- Find hardcoded values that should be configurable

# For resources: Check v2.0 contract compliance
- Health check implementation
- Lifecycle hooks (setup, develop, test, stop)
- CLI integration completeness
- Service.json validity
- Port configuration

# For scenarios: Check structure compliance
- API implementation
- CLI commands
- UI functionality
- Test coverage
- Documentation completeness
```

#### Functionality Testing
```bash
# Start the component
vrooli scenario run [name] OR vrooli resource [name] start

# Test core functionality
- API endpoints (if applicable)
- CLI commands
- UI interactions
- Integration points

# Document what works and what doesn't
- Working features: [list]
- Broken features: [list]
- Partial implementations: [list]
```

#### Visual UI Assessment (MANDATORY for Scenarios with UI)
```bash
# Take screenshots of ALL major UI views
resource-browserless screenshot http://localhost:[PORT]/ --output /tmp/assessment-home.png
resource-browserless screenshot http://localhost:[PORT]/dashboard --output /tmp/assessment-dashboard.png
resource-browserless screenshot http://localhost:[PORT]/settings --output /tmp/assessment-settings.png

# READ and visually inspect EACH screenshot
Read /tmp/assessment-home.png      # Check homepage layout
Read /tmp/assessment-dashboard.png # Check dashboard rendering
Read /tmp/assessment-settings.png  # Check settings page

# Document visual state in assessment report:
## UI Visual Assessment
- [ ] Homepage renders correctly
- [ ] Dashboard displays all elements
- [ ] Forms are properly laid out
- [ ] Navigation menu is visible and functional
- [ ] No overlapping or misaligned elements
- [ ] Colors and themes apply correctly
- [ ] Images and icons load properly
- [ ] Text is readable (no truncation/overflow)
- [ ] Responsive design works (if applicable)
- [ ] Error states display correctly

# If UI has issues, uncheck related PRD items:
- [ ] "Beautiful UI" - BROKEN: Navigation overlaps content
- [ ] "Responsive design" - PARTIAL: Only works on desktop
```

### Phase 3: Dependency and Integration Assessment (5%)

#### Check Dependencies
```bash
# Identify all dependencies
- Required resources
- External services
- Shared workflows
- Configuration requirements

# Verify dependency health
- Are dependencies running?
- Are versions compatible?
- Are integrations working?
```

#### Cross-Impact Analysis
```bash
# Find affected components
vrooli resource qdrant search-all "[component] depends"
vrooli resource qdrant search "[component] uses" scenarios

# Assess potential impacts
- What breaks if this changes?
- What depends on current behavior?
- What improvements would benefit others?
```

### Phase 4: Memory Search for Context (5%)

#### Historical Context
```bash
# Search for past work
vrooli resource qdrant search-all "[component] improvement"
vrooli resource qdrant search-all "[component] fix"
vrooli resource qdrant search-all "[component] issue"

# Learn from history
- Previous improvement attempts
- Known issues and workarounds
- Successful patterns
- Failed approaches
```

## Assessment Output Requirements

### Assessment Report Structure
```markdown
## Assessment Report: [Component Name]

### PRD Accuracy
- Claimed completion: X%
- Actual completion: Y%
- False positives: [list of incorrectly checked items]
- Missing implementations: [list]

### Current State
- Working features: [list]
- Broken features: [list]
- Partial implementations: [list]
- Code quality issues: [list]

### Dependencies
- Required resources: [list with status]
- Integration points: [list with health]
- Configuration issues: [list]

### Historical Context
- Previous improvements: [summary]
- Known issues: [list]
- Attempted fixes: [what worked/failed]

### Improvement Opportunities
1. [High priority]: [Description] - [Impact]
2. [Medium priority]: [Description] - [Impact]
3. [Low priority]: [Description] - [Impact]
```

## Assessment Quality Checklist

Before proceeding to implementation:
- [ ] PRD accuracy verified with actual testing
- [ ] All false completions unchecked
- [ ] Current functionality tested and documented
- [ ] Dependencies identified and checked
- [ ] Cross-scenario impacts assessed
- [ ] Historical context researched
- [ ] Improvement opportunities prioritized
- [ ] Assessment report created

## Common Assessment Mistakes

### Trusting Documentation
❌ **Bad**: Assuming PRD checkboxes are accurate
✅ **Good**: Testing every claimed feature

### Shallow Testing
❌ **Bad**: Just checking if it starts
✅ **Good**: Testing all functionality thoroughly

### Ignoring Dependencies
❌ **Bad**: Not checking if dependencies work
✅ **Good**: Verifying all integration points

### No Historical Research
❌ **Bad**: Starting fresh without context
✅ **Good**: Learning from past attempts

## Assessment ROI

Proper assessment prevents:
- **Fixing already-working features** (waste)
- **Breaking dependencies** (regression)
- **Repeating failed approaches** (inefficiency)
- **Missing high-impact improvements** (opportunity cost)

## Key Principles

1. **Trust but Verify**: Never assume documentation is accurate
2. **Test Everything**: Actually run and test functionality
3. **Document Reality**: Record what you find, not what's claimed
4. **Learn from History**: Past attempts provide valuable context
5. **Prioritize Impact**: Focus on improvements that matter most

## Remember

**Assessment reveals the truth** about the current state. Without accurate assessment:
- You might "fix" things that aren't broken
- You might break things that work
- You might miss the most important improvements
- You might repeat past failures

Invest time in assessment - it guides everything that follows.