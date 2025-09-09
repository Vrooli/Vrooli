# Research Methodology

## Core Principle
**Research prevents duplication.** 30% of generator effort, 20% of improver effort.

## Research Checklist (MUST complete ALL)
- [ ] Search for duplicates (5 Qdrant searches)
- [ ] Find reusable patterns/templates
- [ ] Validate market need
- [ ] Check technical feasibility
- [ ] Learn from past failures

## Phase 1: Memory Search (50% of time)

### Required Searches
```bash
# 1. Exact duplicates
vrooli resource qdrant search "[exact name]"
grep -r "[name]" /home/matthalloran8/Vrooli

# 2. Similar functionality
vrooli resource qdrant search "[category] [function]"

# 3. Past failures
vrooli resource qdrant search "[name] issue problem broken"

# 4. Best practices
vrooli resource qdrant search "[category] pattern template"

# 5. Integration points
vrooli resource qdrant search "[dependency] integration"
```

### Required Outputs
- 5 Qdrant findings with insights
- 3+ reusable patterns identified
- 3+ failure patterns to avoid
- List of dependencies

## Phase 2: External Research (50% of time)

### Web Research Targets
```yaml
Technical:
  - Official docs for tools/frameworks
  - GitHub similar implementations  
  - Stack Overflow solutions
  - Security considerations

Business:
  - Competitor analysis
  - Market validation
  - Revenue models
  - User pain points

Standards:
  - Industry standards/RFCs
  - API design patterns
  - Authentication methods
  - Data formats
```

### Required Outputs
- 5+ external references with URLs
- 3+ code examples
- Security considerations
- Performance expectations

## Phase 3: Synthesis

### Go/No-Go Decision
**STOP if:**
- >80% overlap with existing
- No unique value
- Required resources unavailable

**PROCEED if:**
- Unique value confirmed
- Template selected
- Feasibility validated

### Document in PRD
```markdown
## Research Findings
- **Similar Work**: [list 2-3 most similar]
- **Template Selected**: [pattern to copy]
- **Unique Value**: [1 sentence differentiator]
- **External References**: [5+ URLs]
- **Security Notes**: [considerations]
```

## Quick Research Commands
```bash
# Full duplicate check
vrooli resource qdrant search "[name]" && \
grep -r "[name]" /home/matthalloran8/Vrooli && \
find /home/matthalloran8/Vrooli -name "*[name]*"

# Pattern mining
vrooli resource qdrant search "template [category]" code
find /home/matthalloran8/Vrooli/*/templates/*

# Failure check
vrooli resource qdrant search "[name] broken issue" docs
```

## Remember
- Research saves 10x development time
- Patterns > Original code
- Learn from failures
- Document everything