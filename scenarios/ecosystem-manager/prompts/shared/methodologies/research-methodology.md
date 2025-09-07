# Research Methodology

## Purpose
Research is the foundation of high-quality scenario and resource creation. Proper research prevents duplicated work, identifies proven patterns, and learns from past failures.

## Research Phases (Minimum 40% of Total Effort)

### Phase 1: Local Memory Search (20% minimum)

#### Mandatory Qdrant Searches
Before creating ANYTHING, you MUST search Vrooli's memory:

```bash
# Search for similar work
vrooli resource-qdrant search-all "[your topic] [category]"
vrooli resource-qdrant search "[description keywords]" scenarios
vrooli resource-qdrant search "[description keywords]" resources

# Learn from existing patterns
vrooli resource-qdrant search "[category] pattern" code
vrooli resource-qdrant search "[category] implementation" all
vrooli resource-qdrant search "[category] best practice" docs

# Avoid known problems
vrooli resource-qdrant search "[category] failed" all
vrooli resource-qdrant search "[category] error" docs
vrooli resource-qdrant search "[category] problem" all
vrooli resource-qdrant search "[category] issue" docs

# Find dependencies and integrations
vrooli resource-qdrant search "[category] depends" all
vrooli resource-qdrant search "[category] requires" docs
vrooli resource-qdrant search "[category] integration" scenarios
```

#### Required Research Outputs
- **10+ relevant Qdrant findings** with specific insights
- **5+ reusable patterns** identified from existing work
- **3+ failure patterns** to avoid
- **List of dependencies** and integration points

### Phase 2: External Research (20% minimum)

#### Web Research Requirements
```bash
# Documentation research
- Official documentation for any tools/frameworks
- API documentation for external services
- Best practices guides and tutorials

# Code research
- GitHub repositories with similar functionality
- Stack Overflow solutions for common problems
- Open source implementations for reference

# Business research
- Competitor analysis for similar solutions
- Market validation for the use case
- Revenue potential assessment
```

#### Required External Outputs
- **5+ external references** with URLs and key insights
- **3+ code examples** from reputable sources
- **Security considerations** relevant to the implementation
- **Performance benchmarks** or expectations

### Phase 3: Research Synthesis

#### Pattern Identification
Analyze research to identify:
1. **Common successful patterns** - What works consistently
2. **Anti-patterns to avoid** - What fails consistently
3. **Unique opportunities** - Gaps in existing solutions
4. **Integration possibilities** - How to connect with existing work

#### Knowledge Documentation
Document your research findings:
```markdown
## Research Summary

### Similar Existing Work
- [Name]: [Description] - [Key insights]
- ...

### Patterns to Follow
- [Pattern]: [Why it works] - [Where to apply]
- ...

### Pitfalls to Avoid
- [Problem]: [Why it fails] - [How to prevent]
- ...

### External References
- [Source]: [URL] - [Key takeaways]
- ...
```

## Research Quality Checklist

Before proceeding past research:
- [ ] Searched Qdrant with at least 10 different queries
- [ ] Found and analyzed 10+ relevant memory items
- [ ] Identified 5+ reusable patterns
- [ ] Documented 3+ failure patterns to avoid
- [ ] Gathered 5+ external references
- [ ] Created comprehensive research summary
- [ ] Identified all dependencies and integrations
- [ ] Assessed business value and revenue potential

## Common Research Mistakes

### Insufficient Memory Search
❌ **Bad**: Quick search with 1-2 queries
✅ **Good**: Comprehensive search with 10+ varied queries

### Ignoring Failures
❌ **Bad**: Only looking at successful implementations
✅ **Good**: Actively searching for and learning from failures

### Shallow External Research
❌ **Bad**: Quick Google search
✅ **Good**: Deep dive into documentation, code, and best practices

### No Pattern Analysis
❌ **Bad**: Collecting information without synthesis
✅ **Good**: Identifying patterns, anti-patterns, and opportunities

## Research ROI

Good research provides 10x return:
- **Prevents duplicate work** (saves days)
- **Avoids known failures** (saves iterations)
- **Leverages proven patterns** (improves quality)
- **Identifies integration points** (enables reuse)
- **Validates business value** (ensures revenue)

## Remember

**Research is not optional** - it's the foundation of quality work. The time invested in research pays back exponentially through:
- Fewer failures
- Better implementations
- Faster development
- Higher quality results
- Greater business value

Never skip research to "save time" - it always costs more in the long run.