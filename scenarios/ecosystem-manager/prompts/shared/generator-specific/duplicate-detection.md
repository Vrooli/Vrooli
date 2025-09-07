# Duplicate Detection Protocol

## Preventing Redundant Creation

CRITICAL: Never create duplicates. Follow this detection protocol:

### 1. Direct Duplication Check
```bash
# Check exact name matches
ls /home/matthalloran8/Vrooli/resources/ | grep -i "resource_name"
ls /home/matthalloran8/Vrooli/scenarios/ | grep -i "scenario_name"

# Check service.json for similar purposes
find . -name "service.json" -exec grep -l "similar_keyword" {} \;
```

### 2. Functional Overlap Analysis

Check if existing components already provide:
- **Same core functionality** (abort if >80% overlap)
- **Similar API endpoints** (abort if >70% overlap)  
- **Identical integrations** (abort if >60% overlap)
- **Same problem solution** (abort if exists)

### 3. Naming Collision Prevention

Verify uniqueness of:
- Resource/scenario name
- Port allocations
- CLI commands
- Environment variables
- Database names
- Queue names

### 4. Capability Matrix Check

Create capability comparison:
```
| Feature | Existing Resource | Proposed Resource | Unique? |
|---------|------------------|-------------------|---------|
| Core    | ✓                | ✓                 | ❌      |
| API     | ✓                | ✓                 | ❌      |
| New     | ❌               | ✓                 | ✅      |
```

### 5. Consolidation Opportunities

Instead of creating new, consider:
- **Extending** existing resource with new features
- **Composing** multiple resources together
- **Configuring** existing resource differently
- **Templating** existing resource for new use case

## Abort Conditions

STOP generation if:
1. Exact functional duplicate exists
2. >80% capability overlap with existing resource
3. Same problem already solved by another component
4. Simple configuration change achieves same result

## When Overlap is Acceptable

Proceed only if:
1. Significantly different implementation approach (>50% different)
2. Performance improvement of >2x
3. Major simplification of complexity
4. Different technology stack for specific requirements
5. Specialized variant for specific use case

## Documentation Requirement

In PRD.md, mandatory section:
```markdown
## Differentiation from Existing Resources

### Similar Resources Analyzed
- [Resource A]: 30% overlap, different because...
- [Resource B]: 20% overlap, unique features...

### Why This Isn't a Duplicate
[Clear justification for new resource]

### Integration vs. Creation Decision
[Why extending existing wasn't sufficient]
```