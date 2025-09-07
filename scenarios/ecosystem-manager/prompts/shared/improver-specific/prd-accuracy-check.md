# PRD Accuracy Verification Protocol

## Trust But Verify Everything

CRITICAL: PRD checkboxes often lie. Always verify actual implementation status.

### 1. Checkbox Verification Process

For each checked item in PRD.md:
```bash
# Step 1: Locate claimed implementation
grep -r "feature_name" . --include="*.go" --include="*.js" --include="*.ts"

# Step 2: Test the feature
curl -X POST http://localhost:PORT/api/feature
vrooli resource <name> <command>

# Step 3: Check for error handling
# Try edge cases, invalid inputs, missing params

# Step 4: Verify documentation
grep -i "feature" README.md
```

### 2. Common False Positives

Watch for these frequently incorrect checkmarks:
- ✅ "Health checks implemented" → Often missing timeout handling
- ✅ "CLI commands complete" → Often missing help text or validation
- ✅ "Error handling robust" → Often just try/catch with no recovery
- ✅ "Tests comprehensive" → Often just smoke tests
- ✅ "Documentation complete" → Often missing examples

### 3. Implementation Depth Assessment

Rate each feature's actual completion:
- **0%**: Not started (uncheck immediately)
- **25%**: Stub/placeholder exists
- **50%**: Basic implementation, missing edge cases
- **75%**: Works normally, fails on edge cases
- **100%**: Production-ready with all cases handled

### 4. Testing Checklist

For each claimed feature:
```markdown
- [ ] Feature file exists
- [ ] Feature is imported/used
- [ ] Happy path works
- [ ] Error cases handled
- [ ] Edge cases covered
- [ ] Performance acceptable
- [ ] Documentation accurate
- [ ] Tests exist and pass
```

### 5. PRD Correction Protocol

When inaccuracies found:
1. **Uncheck** false completions immediately
2. **Add comment** explaining what's actually missing
3. **Create queue item** for completion
4. **Update percentage** complete accurately
5. **Document blockers** if any exist

## Red Flags for False Completions

Suspect false completion if:
- No tests for the feature
- No documentation mentions
- No error handling visible
- Implementation is just TODO comment
- Feature fails on first try
- Code is commented out
- Function exists but returns hardcoded value

## Accuracy Standards

- **P0 Requirements**: Must be 100% accurate
- **P1 Requirements**: Must be ≥90% accurate
- **P2 Requirements**: Must be ≥80% accurate
- **Documentation**: Must match implementation exactly

## Required PRD Updates

After assessment, update PRD with:
```markdown
## Implementation Status Audit
*Last verified: [DATE]*

### Verified Complete
- [✅] Feature A (tested: [test description])

### Partially Complete
- [⚠️] Feature B (75% - missing: [specific gaps])

### Incorrectly Marked Complete
- [❌→⬜] Feature C (reason: [why it's not actually done])

### Newly Discovered Requirements
- [ ] Feature D (found during testing)
```