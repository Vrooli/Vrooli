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

### Flexible Search Strategy
Use the most appropriate method based on availability:

#### Option A: Qdrant Search (If Available)
```bash
# Try with 10-second timeout
timeout 10 vrooli resource qdrant search "[exact-name] [category]"
timeout 10 vrooli resource qdrant search "[core-functionality]"
timeout 10 vrooli resource qdrant search "[category] implementation"
```
**Note**: If Qdrant is slow or unavailable, skip to Option B

#### Option B: File Search (Always Available)
```bash
# 1. Exact match search
rg -i "exact-name" /home/matthalloran8/Vrooli --type md

# 2. Functional equivalent search
rg -i "core-functionality|similar-feature" /home/matthalloran8/Vrooli/scenarios

# 3. Component reuse search
rg -i "component-name.*implementation" /home/matthalloran8/Vrooli --type go --type js

# 4. Failure analysis search
rg -i "(failed|error|issue).*category-name" /home/matthalloran8/Vrooli --type md

# 5. Pattern mining search
find /home/matthalloran8/Vrooli/scenarios -name "*template*" -o -name "*example*"
```

### Required Outputs
- 5+ search findings with insights (Qdrant or file search)
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
# Full duplicate check (flexible approach)
(timeout 10 vrooli resource qdrant search "[name]" 2>/dev/null || rg -i "[name]" /home/matthalloran8/Vrooli) && \
find /home/matthalloran8/Vrooli -name "*[name]*"

# Pattern mining (try Qdrant first, fallback to file search)
timeout 10 vrooli resource qdrant search "template [category]" code 2>/dev/null || \
find /home/matthalloran8/Vrooli/*/templates/*

# Failure check (flexible)
timeout 10 vrooli resource qdrant search "[name] broken issue" docs 2>/dev/null || \
rg -i "(failed|error|issue).*[name]" /home/matthalloran8/Vrooli --type md
```

## Remember
- Research saves 10x development time
- Patterns > Original code
- Learn from failures
- Document everything