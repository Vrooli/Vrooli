# Generator Research Phase: Finding Gaps & Opportunities

## Purpose
**GENERATORS CREATE NEW** - Research ensures we're filling real gaps, not duplicating.

## Research Phase

### Primary Mission: Prove Need for NEW
1. **Find the gap** - What's missing from ecosystem?
2. **Explore solutions** - What could fill this gap?
3. **Design approach** - How will we create it?

### Essential Duplicate Check
```bash
# Search for exact name matches
rg -i "^#.*{{EXACT_NAME}}" /home/matthalloran8/Vrooli --type md

# Search for similar functionality 
grep -r "{{CORE_FUNCTIONALITY}}" /home/matthalloran8/Vrooli/scenarios
```
**If duplicates found: Consider improving the existing implementation instead.**

### Pattern Mining (Find What to Copy/Adapt)
```bash
# Find templates to start from
vrooli resource qdrant search "template [category]" code
find /home/matthalloran8/Vrooli -path "*/templates/*{{CATEGORY}}*"

# Find successful patterns to reuse
vrooli resource qdrant search "[category] best practice" docs
rg "pattern|template|scaffold" --type md | grep -i {{CATEGORY}}
```

### Market/Need Validation
Research EXTERNAL sources to confirm need:
- Similar products/services that exist elsewhere
- Problems users currently face without this
- How users solve this problem today (painfully)
- Why Vrooli's version would be better

## Generator Research Output Template
```markdown
## Research Summary
**Gap Found**: [What's missing]
**Duplicates**: NONE (confirmed via X searches)
**Templates Found**: [List 2-3 to copy from]
**Patterns to Reuse**: [List 3-5 patterns]
**External Solutions**: [List 3+ competitors analyzed]
**Unique Value**: [Why ours is needed]
**Revenue Potential**: [Realistic estimate]
```

## Research Quality Gates

### Gate 1: Uniqueness Check
- [ ] No exact matches found
- [ ] No functional duplicates found
- [ ] Clear gap identified
- [ ] Not recreating existing work

### Gate 2: Foundation Ready
- [ ] Templates identified to start from
- [ ] Patterns found to reuse
- [ ] Base scaffold selected
- [ ] Copy/adapt strategy clear

### Gate 3: Value Validation
- [ ] Real problem identified
- [ ] External validation completed
- [ ] Unique solution designed
- [ ] Practical value confirmed

## What Generators DON'T Research
❌ **DON'T research how to fix existing** - That's improver work
❌ **DON'T research current bugs** - We're creating new
❌ **DON'T research incremental improvements** - We're seeding foundations
❌ **DON'T research PRD validation** - We're creating PRDs from scratch

## Generator Research Mindset
Think: **"What doesn't exist that should?"**
NOT: "What exists that needs fixing?"

Think: **"What templates can I combine innovatively?"**
NOT: "How do I fix this existing component?"

Think: **"What new value can I create?"**
NOT: "How do I enhance current value?"

## Time Boxing
- Duplicate detection: 5 minutes MAX
- Pattern mining: 10 minutes MAX  
- External research: 10 minutes MAX
- Documentation: 5 minutes MAX
- **TOTAL: 30% of task time**

Remember: **Generators SEED new ground.** Research proves the ground is empty and fertile.