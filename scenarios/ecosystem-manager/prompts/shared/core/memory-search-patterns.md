# Standard Memory Search Patterns

## Required Search Sequence (5 commands max)

Before starting ANY work, run these exact searches in order:

### 1. Exact Match Search
```bash
vrooli resource qdrant search-all "[exact-name] [category]"
```
**Purpose**: Find identical or near-identical existing work

### 2. Functional Equivalent Search
```bash
vrooli resource qdrant search-all "[core-functionality]"
```
**Purpose**: Find different implementations of same capability

### 3. Component Reuse Search
```bash
vrooli resource qdrant search "[major-component] implementation" code
```
**Purpose**: Find existing code patterns to reuse

### 4. Failure Analysis Search
```bash
vrooli resource qdrant search-all "[category] failed"
```
**Purpose**: Learn what's been tried and didn't work

### 5. Pattern Mining Search
```bash
vrooli resource qdrant search-all "[category] template"
```
**Purpose**: Find proven templates and structures

## Search Result Analysis

### Document Findings Like This:
```markdown
## Memory Search Results
1. **Exact matches found**: [List with similarity %]
2. **Similar implementations**: [What exists, how they differ]
3. **Reusable components**: [Code/patterns to copy]  
4. **Known failures**: [What to avoid]
5. **Best templates**: [What to start from]

## Decision Based on Search:
- **Proceed?** [Yes/No - if similar exists >80%, consider improving instead]
- **Template selected**: [Which base to copy from]
- **Components to reuse**: [Specific pieces to leverage]
```

## Search Quality Gates

**Stop and reconsider if:**
- Found >80% similar existing work
- No reusable patterns found (search deeper)
- Multiple failures on same approach

**Proceed confidently if:**
- Unique value proposition confirmed
- 3+ reusable patterns identified  
- Template selected for base structure

## Common Search Mistakes

❌ **Don't**: Search once and assume nothing exists
✅ **Do**: Run all 5 searches systematically

❌ **Don't**: Ignore failure results  
✅ **Do**: Learn why previous attempts failed

❌ **Don't**: Start from scratch when templates exist
✅ **Do**: Copy and modify existing templates