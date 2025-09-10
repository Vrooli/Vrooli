# Standard Memory Search Patterns

## Flexible Search Strategy

Before starting work, search for existing knowledge using the most appropriate method:

### Option A: Qdrant Search (If Available)
```bash
# Try with 10-second timeout
timeout 10 vrooli resource qdrant search "[exact-name] [category]"
timeout 10 vrooli resource qdrant search "[core-functionality]"
timeout 10 vrooli resource qdrant search "[category] implementation"
```
**Note**: If Qdrant is slow or unavailable, skip to Option B

### Option B: File Search (Always Available)
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