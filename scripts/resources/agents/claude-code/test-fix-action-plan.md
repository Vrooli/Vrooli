# BATS Test Fix Action Plan - Detailed Implementation Guide

## Overview
This plan will address the remaining 13 test failures across 4 modules to achieve 100% test coverage (117/117 tests passing).

## Priority-Based Fix Strategy

### PHASE 1: High Priority - execute.bats (7 failures)
**Target**: Fix command building and batch operation tests
**Impact**: Resolves 54% of remaining failures

#### Step 1.1: Add Mock Log Functions to execute.bats Setup
```bash
# Add to execute.bats setup() function:
log::header() { echo "HEADER: $*"; }
log::info() { echo "INFO: $*"; }
log::success() { echo "SUCCESS: $*"; }
log::warn() { echo "WARN: $*"; }
log::error() { echo "ERROR: $*"; }
```

#### Step 1.2: Fix Command Building Tests (Tests 40-43)
**Issue**: Tests expect clean command output but get log messages
**Solution**: Filter log output or update test assertions

```bash
# Current failing pattern:
output=$(claude_code::run 2>&1)
[[ "$output" =~ "Command: claude --prompt \"Test prompt\" --max-turns 5" ]]

# Fixed pattern:
output=$(claude_code::run 2>&1)
[[ "$output" =~ "Command: claude --prompt \"Test prompt\" --max-turns 5" ]]
# OR filter out log messages:
command_line=$(echo "$output" | grep "Command:")
[[ "$command_line" =~ "claude --prompt \"Test prompt\" --max-turns 5" ]]
```

#### Step 1.3: Fix Batch Operation Tests (Tests 46-48)
**Issues**:
- Test 46: Stream-json format not properly tested
- Test 47: Output file creation logic
- Test 48: Unbound variable error on line 171

**Solutions**:
```bash
# Test 46: Fix OUTPUT_FORMAT check
output=$(
    claude_code::is_installed() { return 0; }
    PROMPT='Test'
    OUTPUT_FORMAT='stream-json'
    # ... rest of test
)

# Test 47: Mock file operations
output=$(
    claude_code::is_installed() { return 0; }
    OUTPUT_FILE='test.json'
    # Mock file creation
    touch() { echo "Creating file: $*"; }
    # ... rest of test
)

# Test 48: Fix unbound variable on line 171
# Need to examine actual line 171 in execute.bats and ensure all variables are defined
```

### PHASE 2: Medium Priority - mcp.bats (3 failures)
**Target**: Fix JSON output and registration status tests
**Impact**: Resolves 23% of remaining failures

#### Step 2.1: Fix Registration Status Tests (Tests 73-74)
**Issue**: Output pattern expectations don't match actual function behavior
**Solution**: Update test assertions to match actual MCP function output

```bash
# Current failing pattern:
[[ "$output" =~ '"scopes":["user"]' ]]

# Need to examine actual mcp::get_registration_status output and update accordingly
# Likely need to mock the curl/API calls properly
```

#### Step 2.2: Fix JSON Output Test (Test 78)
**Issue**: JSON format expectation mismatch
**Solution**: 
1. Check actual `claude_code::mcp_status` output format
2. Update test to match real output or fix function to match test

### PHASE 3: Low Priority - session.bats (2 failures)
**Target**: Fix session resume command building
**Impact**: Resolves 15% of remaining failures

#### Step 3.1: Add Mock Log Functions to session.bats
Similar to execute.bats, add log function mocks to setup()

#### Step 3.2: Fix Session Resume Tests (Tests 87-88)
**Issue**: Command building patterns similar to execute.bats
**Solution**: Apply same patterns as execute.bats fixes

### PHASE 4: Low Priority - status.bats (1 failure)
**Target**: Fix status display test
**Impact**: Resolves 8% of remaining failures

#### Step 4.1: Fix Status Display Test (Test 113)
**Issue**: Output pattern doesn't match function behavior
**Solution**: Examine actual output and update assertion

```bash
# Current failing pattern:
[[ "$output" =~ "not installed" ]]

# Need to check actual claude_code::status output when not installed
# May need to filter log headers or update assertion pattern
```

## Implementation Scripts

### Script 1: fix-execute-tests.sh
```bash
#!/bin/bash
# Fix execute.bats tests

# Update setup function to include log mocks
# Fix command building test assertions
# Resolve unbound variable issues
# Update batch operation tests
```

### Script 2: fix-mcp-tests.sh
```bash
#!/bin/bash
# Fix mcp.bats tests

# Examine and fix registration status output patterns
# Fix JSON output format expectations
# Update mock functions for API calls
```

### Script 3: fix-session-tests.sh
```bash
#!/bin/bash
# Fix session.bats tests

# Add log function mocks
# Fix session resume command building patterns
```

### Script 4: fix-status-tests.sh
```bash
#!/bin/bash
# Fix status.bats tests

# Update status display assertion patterns
```

## Testing Strategy

### Incremental Testing Approach
1. Fix one module at a time
2. Run tests after each module fix
3. Verify no regressions in working modules
4. Document any unexpected issues

### Validation Commands
```bash
# Test individual modules:
bats lib/execute.bats    # After Phase 1
bats lib/mcp.bats        # After Phase 2
bats lib/session.bats    # After Phase 3
bats lib/status.bats     # After Phase 4

# Full test suite validation:
bats manage.bats lib/*.bats

# Expected final result:
# 117/117 tests passing (100%)
```

## Risk Mitigation

### Backup Strategy
- Create backup of all .bats files before making changes
- Keep detailed log of changes for rollback if needed

### Regression Prevention
- Run full test suite after each phase
- Verify working modules remain at 100%

### Documentation Updates
- Update final test report with 100% results
- Document any implementation insights discovered

## Success Criteria

### Phase Completion Criteria
- **Phase 1**: execute.bats shows 12/12 tests passing
- **Phase 2**: mcp.bats shows 21/21 tests passing  
- **Phase 3**: session.bats shows 16/16 tests passing
- **Phase 4**: status.bats shows 6/6 tests passing

### Final Success Criteria
- **All tests passing**: 117/117 (100%)
- **No regressions**: All previously working modules remain 100%
- **Clean test output**: No errors or warnings during test execution
- **Documentation complete**: Updated final test report

## Estimated Timeline

### Development Time
- **Phase 1 (execute.bats)**: 2-3 hours
- **Phase 2 (mcp.bats)**: 1-2 hours  
- **Phase 3 (session.bats)**: 1 hour
- **Phase 4 (status.bats)**: 30 minutes
- **Final validation**: 30 minutes

### Total Estimated Time: 5-7 hours

## Next Steps
1. **Review and approve this plan**
2. **Begin Phase 1 implementation**
3. **Execute phases sequentially**
4. **Validate and document results**

This plan provides a systematic approach to achieving 100% test coverage while minimizing risk and ensuring all fixes are properly validated.