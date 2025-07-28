# SearXNG Bats Test Analysis Report

## Executive Summary

**Test Results:** 11 Passed / 13 Failed (out of 24 tests)

The primary issue is a fundamental misunderstanding of function signatures in the test suite. Most failures stem from incorrect function invocation patterns where tests attempt to pipe data to functions that expect positional parameters.

## Detailed Analysis of Failures

### 1. **Function Signature Mismatch** (Root Cause for 7 failures)

**Affected Tests:**
- `searxng::format_output handles json format`
- `searxng::format_output handles title-only format`
- `searxng::format_output handles csv format`
- `searxng::format_output handles markdown format`
- `searxng::format_output applies limit`
- `searxng::search handles time range`
- `searxng::lucky returns first result URL`

**Issue:** The tests use piping pattern:
```bash
echo '$MOCK_API_RESPONSE' | searxng::format_output json
```

**Actual Function Signature:**
```bash
searxng::format_output() {
    local raw_result="$1"      # Expects JSON as first parameter
    local output_format="${2:-json}"  # Format as second parameter
    local limit="${3:-}"       # Limit as third parameter
}
```

**Diagnosis:** 
- The function receives "json" as `$1` (raw_result) instead of the JSON data
- No data is piped to the function; it's all positional parameters
- This explains why output is just "json" - it's echoing the first parameter

### 2. **Missing Echo/Log Output** (5 failures)

**Affected Tests:**
- `searxng::headlines fetches general headlines`
- `searxng::batch_search_queries processes multiple queries`
- `searxng::batch_search_file processes queries from file`
- `searxng::test_api performs comprehensive test`
- `functions handle missing jq gracefully`

**Issue:** Tests expect specific log output that functions don't produce:
```bash
[[ "$output" =~ "Fetching latest headlines" ]]  # Function doesn't echo this
[[ "$output" =~ "Processing 2 queries" ]]       # Function doesn't echo this
```

**Diagnosis:** 
- The actual functions may not output these exact messages
- Log messages might be sent to stderr instead of stdout
- Functions might succeed but not produce expected output

### 3. **Exit Code Expectations** (1 failure)

**Affected Test:** `functions handle missing jq gracefully`

**Issue:** Test expects exit code 1, but function might return different code
```bash
[ "$status" -eq 1 ]  # Expecting specific exit code
```

**Diagnosis:**
- Function might return a different error code (e.g., 127 for command not found)
- Error handling might not be consistent across functions

## Root Causes Summary

1. **Incorrect Function Invocation Pattern** (58% of failures)
   - Tests use pipe pattern for functions expecting positional parameters
   - Fundamental misunderstanding of API design

2. **Output Expectation Mismatch** (33% of failures)
   - Tests expect specific log messages that functions don't produce
   - Possible stdout/stderr confusion

3. **Exit Code Assumptions** (8% of failures)
   - Hardcoded exit code expectations that don't match implementation

## Recommendations

### Immediate Fixes Required

1. **Fix Function Invocations in Tests**
   ```bash
   # Current (incorrect):
   echo '$MOCK_API_RESPONSE' | searxng::format_output json
   
   # Should be:
   searxng::format_output "$MOCK_API_RESPONSE" "json"
   ```

2. **Update Output Expectations**
   - Review actual function output and update test assertions
   - Consider capturing both stdout and stderr
   - Use more flexible pattern matching

3. **Standardize Exit Codes**
   - Document expected exit codes for each error condition
   - Update tests to match actual implementation

### Test Suite Improvements

1. **Add Function Signature Documentation**
   ```bash
   # Document each function's expected parameters
   # @param $1 - raw JSON result
   # @param $2 - output format
   # @param $3 - limit (optional)
   ```

2. **Create Integration Tests**
   - Test actual API calls against real SearXNG instance
   - Separate unit tests from integration tests

3. **Improve Mock Setup**
   - Make mocks more closely match real function behavior
   - Add debug output to understand test failures

### Code Quality Recommendations

1. **Consistent Error Handling**
   - Standardize exit codes across all functions
   - Use consistent error message format

2. **Better Logging**
   - Add debug mode for verbose output
   - Ensure important messages go to appropriate stream

3. **API Documentation**
   - Create comprehensive API documentation
   - Include examples of correct usage

## Critical Functions Needing Attention

1. **searxng::format_output** - Complete signature mismatch in tests
2. **searxng::headlines** - Missing expected output
3. **searxng::batch_search_queries** - Output doesn't match expectations
4. **searxng::lucky** - Invocation pattern incorrect

## Next Steps Priority

1. **High Priority**: Fix all format_output test invocations
2. **Medium Priority**: Update output expectations for action functions
3. **Low Priority**: Standardize error codes and improve documentation

The test suite needs significant refactoring to match the actual API implementation. The core functionality appears sound, but the tests don't properly exercise it.