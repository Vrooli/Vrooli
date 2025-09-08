# 🚀 Qdrant Memory Patterns (Condensed)

## MANDATORY: Search Protocol with Fallback

### Primary: Qdrant Search (Try First)
```bash
# Execute these 5 searches BEFORE any work
vrooli resource qdrant search-all "[name] [category]"
vrooli resource qdrant search-all "[functionality] implementation"  
vrooli resource qdrant search-all "[category] patterns"
vrooli resource qdrant search "similar [description]" scenarios
vrooli resource qdrant search "[technology] [component]" code
```

### Fallback: File Search (If Qdrant Fails)
**If Qdrant returns errors, is unavailable, or seems broken, use standard file search:**
```bash
# Fallback to grep/ripgrep when Qdrant is down
rg -i "[name]" /home/matthalloran8/Vrooli --type md
rg -i "[functionality]" /home/matthalloran8/Vrooli/scenarios
find /home/matthalloran8/Vrooli -name "*[category]*" -type f
grep -r "[description]" /home/matthalloran8/Vrooli/resources 2>/dev/null
```

**Signs Qdrant is broken:**
- Connection refused/timeout errors
- Returns 0 results for obvious terms
- Extremely slow responses (>10 seconds)  
- Inconsistent results for same query
- Error messages about embeddings/collections

## Generator-Specific Searches (30% of effort)

### Duplicate Detection (MANDATORY)
```bash
# Must find 0 duplicates to proceed
vrooli resource qdrant search-all "[exact name]"
vrooli resource qdrant search-all "identical [functionality]"
# FALLBACK: rg -i "[exact name]" /home/matthalloran8/Vrooli
```

### Pattern Mining
```bash
# Find templates and successful patterns
vrooli resource qdrant search "template [category]" code
vrooli resource qdrant search "scaffold [type]" resources
# FALLBACK: find /home/matthalloran8/Vrooli -path "*/templates/*" -name "*.md"
```

## Improver-Specific Searches (20% of effort)

### Current State Audit
```bash
# Understand what exists
vrooli resource qdrant search-all "[target name]"
vrooli resource qdrant search "[target name] issue" docs
# FALLBACK: rg -A5 -B5 "{{TARGET_NAME}}" /home/matthalloran8/Vrooli
```

### Enhancement Patterns
```bash
# Find proven improvements
vrooli resource qdrant search "improved [component]" code
vrooli resource qdrant search-all "fixed [issue type]"
# FALLBACK: grep -r "improve\|enhance\|fix" --include="*.md" /home/matthalloran8/Vrooli
```

## Score Interpretation
- **0.9-1.0**: Near duplicate (stop - already exists)
- **0.7-0.9**: Strong match (use as template)
- **0.5-0.7**: Related (review for patterns)
- **<0.5**: Different domain (usually ignore)

## Knowledge Contribution Format

### After Creating (Generators)
```markdown
## Qdrant Entry: {{NAME}}
**Category**: {{CATEGORY}}
**Created**: {{DATE}}
**Unique Value**: {{WHAT_THIS_ADDS}}
**Reusable Patterns**: {{LIST_PATTERNS}}
```

### After Improving (Improvers)
```markdown
## Improvement: {{TARGET_NAME}}
**Fixed**: {{ISSUES_RESOLVED}}
**Enhanced**: {{FEATURES_ADDED}}
**PRD Progress**: {{CHECKBOXES_ADVANCED}}
**Lessons**: {{WHAT_WE_LEARNED}}
```

## Best Practices

### DO's
✅ Search Qdrant first, fallback to grep if needed
✅ Use multiple search variations
✅ Document patterns immediately
✅ Check for duplicates BEFORE creating

### DON'Ts
❌ Skip search when Qdrant fails (use fallback)
❌ Trust single search results
❌ Create without searching
❌ Ignore file search capabilities

## Quick Troubleshooting

**Qdrant returns nothing**: 
1. Try synonyms
2. Switch to file search immediately
3. Use `rg` or `grep` as fallback

**Qdrant is slow/broken**:
1. Don't wait - switch to file search
2. Report issue but continue work
3. Use: `rg -i "pattern" /home/matthalloran8/Vrooli`

**Can't tell if exists**:
1. Search both Qdrant AND files
2. If either finds matches, investigate
3. When in doubt, it probably exists

## Search Time Limits
- Qdrant searches: MAX 30 seconds total
- If slow/failing: Switch to file search immediately  
- File searches: No limit (usually faster)
- Total research phase: 30% of task time

Remember: **File search is ALWAYS available as backup.** Don't let Qdrant issues block progress.