# SearXNG Resource - Known Problems and Solutions

## Resolved Issues

### 0. Recent Fixes (2025-09-14)

#### Benchmark Command Parameter Parsing
**Problem**: The benchmark command failed when using --iterations parameter.
**Symptoms**: `searxng::benchmark: line 748: iterations: unbound variable`
**Root Cause**: CLI framework consumed parameters before they reached the benchmark function.
**Solution**: Created a wrapper function `searxng::benchmark_cli()` that properly parses CLI arguments and passes them to the actual benchmark function.

#### Advanced Search Time Range Parameter
**Problem**: The advanced-search command didn't recognize --time-range parameter.
**Symptoms**: `[ERROR] Unknown option: --time-range`
**Root Cause**: Only --time was accepted, not the more intuitive --time-range.
**Solution**: Updated parameter parsing to accept both --time and --time-range for better user experience.

#### Redis Readonly Variable Error
**Problem**: Enabling/disabling Redis threw readonly variable error.
**Symptoms**: `/lib/docker.sh: line 296: SEARXNG_ENABLE_REDIS: readonly variable`
**Root Cause**: Attempting to export a variable that was declared as readonly in defaults.sh.
**Solution**: Removed the export statements and just logged the status changes instead.

## Previously Resolved Issues

### 1. Benchmark Loop Execution Failure
**Problem**: The benchmark command would hang when executed due to bash C-style for loop incompatibility with `set -euo pipefail`.
**Symptoms**: Command would print "Running benchmark..." and then hang indefinitely.
**Root Cause**: The syntax `for ((i=0; i<num_queries; i++))` doesn't work reliably with strict error handling enabled.
**Solution**: Replaced C-style for loop with while loop and used safe arithmetic operations.
```bash
# Before (problematic)
for ((i=0; i<num_queries; i++)); do

# After (working)
local i=0
while [[ $i -lt $num_queries ]]; do
    # loop body
    ((i++)) || true
done
```

### 2. Integration Test JSON Parsing Timeouts
**Problem**: Integration tests would hang when parsing JSON responses with jq.
**Symptoms**: Test would hang at "Testing JSON search API with multiple queries..." and never complete.
**Root Cause**: 
1. Large JSON responses could cause jq to hang
2. Arithmetic operations with `((var++))` would fail with strict error handling
3. Passing JSON through bash -c with single quotes could fail with JSON containing single quotes

**Solution**: 
1. Added timeout to jq commands to prevent hanging
2. Replaced all `((var++))` with `var=$((var + 1))` 
3. Removed unnecessary bash -c wrapper for jq operations

### 3. Arithmetic Operations Failing Silently
**Problem**: Increment operations like `((successful_queries++))` would cause script to exit when the value was 0.
**Symptoms**: Tests would exit unexpectedly without error messages.
**Root Cause**: With `set -e`, arithmetic operations that evaluate to 0 are treated as failures.
**Solution**: Used explicit arithmetic assignment or added `|| true` to ignore the exit code.
```bash
# Before
((successful_queries++))

# After (option 1)
successful_queries=$((successful_queries + 1))

# After (option 2)
((successful_queries++)) || true
```

## Best Practices Learned

1. **Always use timeouts with external commands**: Especially for JSON parsing operations
2. **Avoid C-style for loops in bash with strict error handling**: Use while loops instead
3. **Be careful with arithmetic in bash**: Increment operations can fail unexpectedly
4. **Test with verbose output**: Helps identify where scripts hang
5. **Use simple fallback parsing**: When jq fails or times out, have grep-based alternatives

## Testing Commands

To verify all issues are resolved:
```bash
# Test benchmark functionality
vrooli resource searxng content benchmark 5

# Test integration tests
vrooli resource searxng test integration

# Run full test suite
vrooli resource searxng test all
```

All tests should complete without hanging and show passing results.