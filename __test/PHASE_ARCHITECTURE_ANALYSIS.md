# Phase Testing Architecture Analysis & Solution

## Current State Analysis

### Problem Statement
The test infrastructure has **conflicting phase approaches**:
- Some phases support scoping (--resource, --scenario, --path)
- Other phases test everything regardless of scope
- This creates confusion about what's actually being tested

### Phase Inventory

| Phase | Scoping Support | Coverage | Issue |
|-------|-----------------|----------|-------|
| **static** | ✅ Yes (resource/scenario/path) | All code files | Consistent |
| **structure** | ✅ Yes (resource/scenario/path) | Directory structures | Consistent |
| **integration** | ✅ Yes (resource/scenario/path) | Integration tests | Consistent |
| **unit** | ❌ No | All BATS tests | **Tests everything** |
| **docs** | ❌ No | All markdown files | **Tests everything** |

### The Core Problem

1. **Inconsistent Scoping**: When running `./run-tests.sh --resource=ollama`, the static/structure/integration phases respect the scope, but unit/docs phases test EVERYTHING.

2. **Performance Impact**: Can't efficiently test a single resource because unit/docs phases always run fully.

3. **Developer Confusion**: It's unclear what `--resource=ollama` actually tests.

4. **Mixed Paradigms**:
   - **Location-based**: Resources vs Scenarios (WHERE files are)
   - **Type-based**: Static, Structure, Unit, Integration, Docs (WHAT kind of test)

## Root Cause

The original design tried to mix two organizational principles:
1. **Vertical slicing** (by resource/scenario) - Good for focused development
2. **Horizontal slicing** (by test type) - Good for comprehensive validation

This created a matrix problem where not all combinations make sense.

## Proposed Solution: Universal Scoping

### Principle
**Every phase should respect the same scoping parameters**, creating a consistent mental model.

### Implementation Strategy

#### 1. Add Scoping to Unit Phase
```bash
# Current: Tests all BATS files
find . -name "*.bats"

# Proposed: Respect scoping
if [[ -n "$SCOPE_RESOURCE" ]]; then
    find "resources/$SCOPE_RESOURCE" -name "*.bats"
elif [[ -n "$SCOPE_SCENARIO" ]]; then
    find "scenarios/$SCOPE_SCENARIO" -name "*.bats"
elif [[ -n "$SCOPE_PATH" ]]; then
    find "$SCOPE_PATH" -name "*.bats"
else
    find . -name "*.bats"  # Default to all
fi
```

#### 2. Add Scoping to Docs Phase
```bash
# Current: Tests all markdown
find . -name "*.md"

# Proposed: Respect scoping
if [[ -n "$SCOPE_RESOURCE" ]]; then
    find "resources/$SCOPE_RESOURCE" -name "*.md"
elif [[ -n "$SCOPE_SCENARIO" ]]; then
    find "scenarios/$SCOPE_SCENARIO" -name "*.md"
elif [[ -n "$SCOPE_PATH" ]]; then
    find "$SCOPE_PATH" -name "*.md"
else
    find . -name "*.md"  # Default to all
fi
```

#### 3. Standardize Arguments Across All Phases

Every phase should accept:
- `--resource=NAME` - Test only files related to specific resource
- `--scenario=NAME` - Test only files related to specific scenario
- `--path=PATH` - Test only files in specific path
- `--verbose` - Show detailed output
- `--parallel` - Run in parallel where possible
- `--no-cache` - Skip cache
- `--dry-run` - Show what would run

### Usage Examples (After Fix)

```bash
# Test everything for ollama resource (ALL phases respect this)
./run-tests.sh all --resource=ollama
# ✅ Static: Only ollama shell/python/go files
# ✅ Structure: Only ollama directory structure
# ✅ Integration: Only ollama integration tests
# ✅ Unit: Only ollama BATS tests
# ✅ Docs: Only ollama documentation

# Test only documentation for a scenario
./run-tests.sh docs --scenario=app-generator
# ✅ Only markdown files in scenarios/app-generator/

# Test specific path across all phases
./run-tests.sh all --path=resources/ai
# ✅ All phases test only files in that path
```

### Benefits

1. **Consistency**: All phases behave the same way
2. **Performance**: Can truly test just one component
3. **Clarity**: Clear mental model for developers
4. **Flexibility**: Mix and match phases with scopes
5. **Parallelization**: Can run different scopes in parallel

## Migration Plan

### Phase 1: Update Unit Phase (Priority: HIGH)
1. Add scoping parameters to test-unit.sh
2. Modify BATS file discovery logic
3. Test with various scopes

### Phase 2: Update Docs Phase (Priority: HIGH)
1. Add scoping parameters to test-docs.sh
2. Modify markdown file discovery logic
3. Ensure link validation respects scope

### Phase 3: Update Run-Tests.sh (Priority: MEDIUM)
1. Pass scoping parameters to all phases
2. Update help documentation
3. Add scope validation

### Phase 4: Documentation (Priority: LOW)
1. Update README with new scoping behavior
2. Add examples for common use cases
3. Document the architecture decision

## Alternative Considered (and Rejected)

### Option: Separate Test Runners
Have `test-resource.sh` and `test-scenario.sh` as separate entry points.
- **Rejected because**: Duplicates logic, harder to maintain

### Option: Remove Phases Entirely
Just have resource-based or scenario-based testing.
- **Rejected because**: Phases provide valuable test-type organization

### Option: Matrix Configuration
Define a matrix of valid phase/scope combinations.
- **Rejected because**: Too complex, hard to remember

## Success Metrics

1. **Consistency**: All 5 phases support the same scoping parameters
2. **Performance**: Testing a single resource takes <30 seconds (not 15 minutes)
3. **Usability**: Developers understand and use scoping effectively
4. **Maintainability**: Scoping logic is DRY (shared helper function)

## Implementation Checklist

- [x] Create shared scoping function in shared/scoping.bash
- [x] Update test-unit.sh to use scoping
- [x] Update test-docs.sh to use scoping
- [x] Update run-tests.sh to pass scope to all phases
- [x] Test all combinations of phase + scope
- [x] Update documentation
- [ ] Remove legacy scoping code (Future cleanup)

## Implementation Results

### ✅ SUCCESS: Universal Scoping Now Live

The solution has been **successfully implemented** and tested. All 5 phases now support consistent scoping:

```bash
# BEFORE: Inconsistent behavior
./run-tests.sh all --resource=ollama
# ❌ Static: Respected scope
# ❌ Docs: Tested ALL files (ignored scope)
# ❌ Unit: Tested ALL files (ignored scope)

# AFTER: Universal consistency  
./run-tests.sh all --resource=ollama
# ✅ Static: "📦 Resource: ollama" 
# ✅ Structure: "📦 Resource: ollama"
# ✅ Integration: "📦 Resource: ollama"  
# ✅ Unit: "🔍 Discovering BATS test files for resource 'ollama'..."
# ✅ Docs: "Finding markdown files for resource 'ollama'..."
```

### Performance Impact
- **Before**: Testing `--resource=ollama` still took 15+ minutes (docs/unit tested everything)
- **After**: Testing `--resource=ollama` takes seconds (all phases respect scope)

### Developer Experience
- **Consistent Mental Model**: All phases work the same way
- **Clear Logging**: Each phase shows what scope it's using
- **Flexible**: Mix any phase with any scope
- **Backwards Compatible**: No scope = test everything (existing behavior)

## Conclusion

**Universal scoping** is the right solution because it:
- Provides a consistent mental model
- Enables efficient focused testing
- Maintains the benefits of phase-based organization
- Is backwards compatible (no scope = test everything)

This will transform the test infrastructure from confusing to intuitive.