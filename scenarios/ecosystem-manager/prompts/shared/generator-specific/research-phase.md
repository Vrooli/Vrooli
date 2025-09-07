# Generator Research Phase

## Purpose
Generators create NEW scenarios and resources. Research ensures we're not duplicating work and that we're building on the best available patterns.

## 📋 Required Research Checklist
{{INCLUDE: generator-specific/research-checklist.md}}

## Research Allocation: 40% of Total Effort

### Deep Qdrant Memory Search (Required)

Before generating ANYTHING new, exhaustively search existing knowledge:

```bash
# MANDATORY SEARCHES - Run ALL of these

# 1. Direct similarity search
vrooli resource-qdrant search-all "[exact name] [category]"
vrooli resource-qdrant search "[full description]" all
vrooli resource-qdrant search "similar to [core functionality]" all

# 2. Component search
vrooli resource-qdrant search "[each major component]" all
vrooli resource-qdrant search "[each technology mentioned]" resources
vrooli resource-qdrant search "[each integration point]" scenarios

# 3. Pattern mining
vrooli resource-qdrant search "[category] template" all
vrooli resource-qdrant search "[category] scaffold" all
vrooli resource-qdrant search "[category] boilerplate" code
vrooli resource-qdrant search "[category] structure" docs

# 4. Failure analysis
vrooli resource-qdrant search "[category] failed" all
vrooli resource-qdrant search "[category] deprecated" all
vrooli resource-qdrant search "[category] replaced" all
vrooli resource-qdrant search "[category] problem" docs

# 5. Business context
vrooli resource-qdrant search "[use case] revenue" all
vrooli resource-qdrant search "[use case] customer" docs
vrooli resource-qdrant search "[use case] value" all
```

### Research Output Requirements

#### Similarity Analysis
```markdown
## Existing Similar Work
1. [Name]: [Similarity %] - [Key differences]
   - What it does well: [...]
   - What it lacks: [...]
   - How we'll improve: [...]
```

#### Pattern Extraction
```markdown
## Reusable Patterns Found
1. [Pattern Name]: [Source]
   - Use case: [When to apply]
   - Implementation: [How to apply]
   - Adaptation needed: [Modifications required]
```

#### Innovation Opportunities
```markdown
## Gaps and Opportunities
1. [Gap Identified]: [Evidence]
   - Current solutions: [What exists]
   - Missing capability: [What's needed]
   - Our approach: [How we'll fill the gap]
```

### External Research Requirements

#### Industry Research
- **5+ competitor solutions** analyzed
- **3+ open source references** studied
- **10+ documentation sources** reviewed
- **Security best practices** identified
- **Performance benchmarks** gathered

#### Market Validation
```markdown
## Market Research
- Target users: [Who needs this]
- Problem severity: [How painful is the problem]
- Existing solutions: [What they use now]
- Value proposition: [Why ours is better]
- Revenue potential: [Realistic estimate]
```

### Research Synthesis for Generators

#### Decision Framework
Based on research, answer:
1. **Should this be created?** (Or does equivalent exist?)
2. **What makes this unique?** (Why new vs. improving existing?)
3. **What patterns can we reuse?** (Don't reinvent wheels)
4. **What failures must we avoid?** (Learn from history)
5. **What's the revenue potential?** ($10K? $50K? More?)

#### Template Selection
Based on research, identify:
```markdown
## Template/Reference Selection
- Base template: [Which template to start from]
- Reference implementations: [2-3 to study]
- Components to reuse: [List existing pieces]
- New components needed: [What must be created]
```

## Generator-Specific Research Checklist

- [ ] Searched for exact name match
- [ ] Searched for functional equivalents  
- [ ] Identified ALL similar existing work
- [ ] Extracted 5+ reusable patterns
- [ ] Found 3+ templates or references
- [ ] Analyzed 5+ external solutions
- [ ] Validated market need
- [ ] Estimated revenue potential
- [ ] Documented innovation opportunity
- [ ] Created template selection plan

## Research Quality Gates for Generators

### Gate 1: Uniqueness Validation
**STOP if existing solution provides 80%+ of requirements**
- Consider improving existing instead
- Document why new is needed if proceeding

### Gate 2: Pattern Identification  
**Must identify 3+ reusable patterns**
- If none found, research deeper
- Patterns reduce implementation time by 50%

### Gate 3: Revenue Validation
**Must demonstrate $10K+ potential value**
- If lower, reconsider priority
- Document value proposition clearly

## Common Generator Research Mistakes

### Insufficient Uniqueness Check
❌ **Bad**: Quick search, assume it's new
✅ **Good**: Exhaustive search, prove it's needed

### Ignoring Existing Components
❌ **Bad**: Building everything from scratch
✅ **Good**: Maximizing reuse of existing work

### No Market Validation
❌ **Bad**: Building without confirming need
✅ **Good**: Validating demand before building

## Remember for Generators

**Research prevents duplicate work** - The #1 generator sin is recreating what exists

**Patterns accelerate development** - Found patterns cut development time in half

**Market validation ensures value** - No point building what nobody needs

**Your research becomes permanent** - Document thoroughly for future generators

The goal is to create something NEW and VALUABLE, not to recreate what exists. Research ensures you're adding unique value to Vrooli.