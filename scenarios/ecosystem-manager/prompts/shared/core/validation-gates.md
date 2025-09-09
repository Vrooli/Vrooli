# ✅ Validation Gates System (Condensed)

## The Five MANDATORY Gates

Every change MUST pass ALL gates. No exceptions.

### Gate 1: Functional ⚙️
**Verify**: Component starts, responds, stops
```bash
./manage.sh setup && ./manage.sh develop  # Must start
{{TIMEOUT_HEALTH_CHECK}}  # Must respond 200
./manage.sh stop                          # Must stop < 10 sec
```
**FAIL = STOP**: Don't proceed if basics don't work

### Gate 2: Integration 🔗  
**Verify**: Works with dependencies
```bash
# Test each declared resource connection
# Test API compatibility (no breaking changes)
# Test shared APIs still respond
# For UI scenarios: Take and READ screenshots to verify visual integration
```
**FAIL = STOP**: Broken integrations break ecosystem

### Gate 3: Documentation 📚
**Verify**: Knowledge captured
- README.md has Overview, Usage, Troubleshooting sections
- PRD.md shows progress (checked boxes)
- Code has inline comments for complex logic
**FAIL = STOP**: Undocumented code is tech debt

### Gate 4: Testing 🧪
**Verify**: Tests pass
```bash
./test.sh  # All tests must pass
```
- Existing tests still pass (no regressions)
- New features have tests
- Error handling works
**FAIL = STOP**: Untested code is broken code

### Gate 5: Memory 🧠
**Verify**: Qdrant updated
```bash
# New patterns documented in markdown
# Qdrant embeddings refreshed
vrooli resource qdrant embeddings refresh --path .
```
**FAIL = WARNING**: Continue but mark for update

## Generator-Specific Gates

### PRD Gate (PRIMARY - 50% effort)
- [ ] All PRD sections filled
- [ ] 5+ P0 requirements defined
- [ ] Revenue potential documented
- [ ] Technical specs complete

### Uniqueness Gate
- [ ] Zero duplicates found in search
- [ ] Adds new value to ecosystem
- [ ] Not reimplementing existing functionality

### Scaffolding Gate
- [ ] Directory structure created
- [ ] v2.0 contract files present
- [ ] Health check responds
- [ ] CLI commands work

## Improver-Specific Gates

### PRD Accuracy Gate (20% effort)
- [ ] Each checkbox verified against reality
- [ ] False completions unchecked
- [ ] Progress percentages updated
- [ ] New completions checked

### No-Regression Gate
- [ ] All previously working features still work
- [ ] Performance not degraded
- [ ] No new security issues
- [ ] Dependencies not broken
- [ ] **UI not visually broken (verified via screenshots)**

### Progress Gate  
- [ ] At least 1 PRD checkbox advanced
- [ ] Measurable improvement achieved
- [ ] Change is incremental and focused

## Gate Execution Order

```
1. Functional → 2. Integration → 3. Documentation → 4. Testing → 5. Memory
```

**Golden Rule**: If a gate fails, STOP. Fix it before proceeding.

## Quick Gate Checks

### For Generators
```bash
# 1-minute validation
[ -f PRD.md ] && grep -q "\[x\]" PRD.md  # Has content?
[ -f README.md ]                          # Has docs?
./manage.sh develop && {{STANDARD_HEALTH_CHECK}}  # Works?
```

### For Improvers  
```bash
# 1-minute validation
git diff PRD.md | grep "^\+.*\[x\]"  # Advanced checkboxes?
./test.sh                             # Tests pass?
git diff --stat                      # Focused changes?
```

## Gate Failure Protocol

1. **STOP immediately** - Don't compound problems
2. **Document failure** - What gate? What error?
3. **Fix root cause** - Not symptoms
4. **Restart from Gate 1** - Ensure all gates still pass

## Time Allocation
- Gates 1-2: 5% (should be quick)
- Gate 3: 5% (documentation)  
- Gate 4: 5% (run tests)
- Gate 5: 5% (update memory)
- **Reserve 20% total for validation**

## Non-Negotiable Rules

✅ **ALL gates must pass** - No "mostly working" code
✅ **Order matters** - Don't skip gates
✅ **Document failures** - Help the next person
✅ **Fix immediately** - Don't accumulate tech debt

❌ **Never skip gates** "just this once"
❌ **Never merge failing code** even partially
❌ **Never disable tests** to make them pass
❌ **Never fake documentation** to pass gates

Remember: **Gates protect ecosystem quality.** Respect them.