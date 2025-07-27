# Claude Code BATS Test Fix Plan

## Root Cause Analysis

The test failures are caused by the helper function `setup_claude_code_test_env()` using variables that are not available in the subshell context created by `run bash -c`. Specifically:

1. When `declare -f setup_claude_code_test_env` exports the function, variables like `$CLAUDE_CODE_DIR` are empty
2. This results in paths like `/../../../helpers/utils/log.sh` instead of proper absolute paths
3. The subshell cannot find any source files, so no functions get defined

## Proposed Solutions

### Solution 1: Direct Test Context Sourcing (Recommended)
Follow the n8n pattern where tests call the helper function directly or source files in the test context.

**Pros:**
- Proven pattern from n8n tests
- Simple and straightforward
- No complex subshell handling

**Cons:**
- More verbose tests
- Some code duplication

### Solution 2: Pass Paths as Parameters
Modify helper function to accept paths as parameters instead of relying on global variables.

**Pros:**
- Keeps helper function approach
- More flexible

**Cons:**
- Requires modifying all test calls
- Still uses subshells which can be fragile

### Solution 3: Use BATS Built-in Setup Function
Use BATS `setup()` function to source dependencies once per test file.

**Pros:**
- Standard BATS pattern
- Runs before each test automatically
- No subshells needed

**Cons:**
- Need to handle test-specific mocking separately

## Detailed Implementation Plan (Solution 1 + 3 Hybrid)

### Phase 1: Create Standard BATS Setup Function
```bash
setup() {
    # Set up paths
    export BATS_TEST_DIRNAME="${BATS_TEST_DIRNAME:-$(cd "$(dirname "$BATS_TEST_FILENAME")" && pwd)}"
    export CLAUDE_CODE_DIR="$BATS_TEST_DIRNAME/.."
    export RESOURCES_DIR="$CLAUDE_CODE_DIR/../.."
    export HELPERS_DIR="$RESOURCES_DIR/../helpers"
    export SCRIPT_PATH="$BATS_TEST_DIRNAME/$(basename "$BATS_TEST_FILENAME" .bats).sh"
    
    # Source dependencies in order
    source "$HELPERS_DIR/utils/log.sh" || true
    source "$HELPERS_DIR/utils/system.sh" || true
    source "$HELPERS_DIR/utils/flow.sh" || true
    source "$RESOURCES_DIR/common.sh" || true
    
    # Source config and common modules
    source "$CLAUDE_CODE_DIR/config/defaults.sh"
    source "$CLAUDE_CODE_DIR/lib/common.sh"
    
    # Source the script under test
    source "$SCRIPT_PATH"
}
```

### Phase 2: Update Test Patterns

#### For Simple Function Tests:
```bash
@test "function exists and works" {
    # Function is already loaded by setup()
    claude_code::some_function
    [ $? -eq 0 ]
}
```

#### For Tests Requiring Mocks:
```bash
@test "function with mocked dependencies" {
    # Override functions after setup()
    system::is_command() { return 0; }
    claude() { echo "1.0.0"; }
    
    # Test the function
    result=$(claude_code::get_version)
    [ "$result" = "1.0.0" ]
}
```

#### For Tests Requiring Isolation:
```bash
@test "isolated test" {
    run bash -c "
        # Source paths must be absolute
        source '$HELPERS_DIR/utils/log.sh'
        source '$CLAUDE_CODE_DIR/config/defaults.sh'
        source '$SCRIPT_PATH'
        
        # Mock and test
        system::is_command() { return 1; }
        claude_code::check_node_version
    "
    [ "$status" -eq 1 ]
}
```

### Phase 3: File-by-File Changes

1. **lib/common.bats**
   - Add `setup()` function
   - Remove `setup_claude_code_test_env()`
   - Update tests to use direct calls or absolute paths in subshells

2. **lib/install.bats**
   - Add `setup()` function with `confirm()` mock
   - Update tests to override specific functions after setup

3. **lib/status.bats**
   - Add `setup()` function
   - Simple tests can call functions directly
   - Complex tests use absolute paths

4. **lib/execute.bats**
   - Add `setup()` function
   - Mock `eval` for command testing
   - Use direct function calls

5. **lib/session.bats**
   - Add `setup()` function with `confirm()` mock
   - Test session functions directly

6. **lib/settings.bats**
   - Add `setup()` function with `confirm()` mock
   - Mock `jq` where needed

7. **lib/mcp.bats**
   - Add `setup()` function
   - Complex MCP tests may need absolute paths

### Phase 4: Testing Strategy

1. Fix one test file at a time
2. Run the test file to verify fixes
3. Ensure no regression in functionality
4. Document any special patterns discovered

## Implementation Priority

1. **Start with lib/common.bats** - Simplest, sets the pattern
2. **Then lib/status.bats** - Similar complexity
3. **Then lib/install.bats** - Add mocking patterns
4. **Continue with remaining files** - Apply learned patterns

## Success Criteria

- All tests pass without warnings
- No code duplication where avoidable
- Tests remain readable and maintainable
- Pattern is consistent across all test files
- No changes to actual implementation code (only tests)

## Estimated Effort

- Initial pattern setup: 10 minutes
- Per test file update: 15-20 minutes
- Total time: ~2-3 hours for all files
- Testing and validation: 30 minutes