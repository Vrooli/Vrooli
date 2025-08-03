# Migration Guide: Upgrading to the New BATS Testing Infrastructure

This guide helps you migrate existing BATS tests to the new unified testing infrastructure.

## Overview of Changes

The new infrastructure provides:
- ✅ Unified path resolution system
- ✅ Lazy-loaded mock registry
- ✅ Performance optimizations (50-60% faster)
- ✅ Consistent setup patterns
- ✅ 54+ built-in assertions
- ✅ Resource-aware testing

## Quick Migration Checklist

- [ ] Update source paths to use `core/common_setup.bash`
- [ ] Replace legacy setup functions with new equivalents
- [ ] Update assertions to use new assertion library
- [ ] Remove hardcoded paths
- [ ] Test and verify functionality

## Migration Patterns

### 1. Basic Test Migration

**Old Pattern:**
```bash
#!/usr/bin/env bats

# Old way - multiple possible paths
source "${BATS_TEST_DIRNAME}/../standard_mock_framework.bash"
# or
source "$(dirname "$0")/../common_setup.bash"

setup() {
    setup_standard_mock_framework
}

teardown() {
    cleanup_standard_mock_framework
}

@test "example test" {
    run docker ps
    [ "$status" -eq 0 ]
    [[ "$output" =~ "CONTAINER" ]]
}
```

**New Pattern:**
```bash
#!/usr/bin/env bats

bats_require_minimum_version 1.5.0

# New way - consistent path resolution
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${TEST_DIR}/../fixtures/bats/core/common_setup.bash"

setup() {
    setup_standard_mocks  # New function name
}

teardown() {
    cleanup_mocks  # Simplified name
}

@test "example test" {
    run docker ps
    assert_success  # New assertion
    assert_output_contains "CONTAINER"  # Clearer assertion
}
```

### 2. Resource Test Migration

**Old Pattern:**
```bash
# Manually setting up resource environment
export OLLAMA_PORT=11434
export OLLAMA_BASE_URL="http://localhost:$OLLAMA_PORT"

setup() {
    # Manual mock setup
    function docker() {
        if [[ "$1" == "ps" ]]; then
            echo "ollama_container   running"
        fi
    }
    export -f docker
}
```

**New Pattern:**
```bash
setup() {
    setup_resource_test "ollama"  # Automatic configuration
    # All environment variables are set automatically:
    # - OLLAMA_PORT, OLLAMA_BASE_URL
    # - OLLAMA_CONTAINER_NAME
    # - Mock functions configured
}

@test "ollama is healthy" {
    assert_resource_healthy "ollama"  # Built-in assertion
}
```

### 3. Integration Test Migration

**Old Pattern:**
```bash
# Complex manual setup for multiple resources
setup() {
    # Ollama setup
    export OLLAMA_PORT=11434
    function mock_ollama() { ... }
    
    # Whisper setup
    export WHISPER_PORT=9000
    function mock_whisper() { ... }
    
    # Complex mock coordination
}
```

**New Pattern:**
```bash
setup() {
    setup_integration_test "ollama" "whisper"  # Simple!
    # Everything is configured automatically
}

@test "resources work together" {
    assert_resource_healthy "ollama"
    assert_resource_healthy "whisper"
    # Your integration logic here
}
```

## Function Mapping

| Old Function | New Function | Notes |
|--------------|--------------|-------|
| `setup_standard_mock_framework` | `setup_standard_mocks` | Simpler name |
| `cleanup_standard_mock_framework` | `cleanup_mocks` | Unified cleanup |
| `setup_docker_mock` | Automatic with `setup_standard_mocks` | No manual setup needed |
| `setup_http_mock` | Automatic with `setup_standard_mocks` | Included by default |
| `[ "$status" -eq 0 ]` | `assert_success` | Clearer intent |
| `[ "$status" -ne 0 ]` | `assert_failure` | Clearer intent |
| `[[ "$output" =~ "text" ]]` | `assert_output_contains "text"` | Better error messages |

## Assertion Upgrades

### Old Style Assertions
```bash
# Checking exit status
[ "$status" -eq 0 ]
[ "$status" -ne 0 ]

# Checking output
[[ "$output" =~ "expected" ]]
[ -z "$output" ]
[ -n "$output" ]

# Checking files
[ -f "$file" ]
[ -d "$dir" ]

# Checking JSON
echo "$output" | jq -e '.status == "ok"'
```

### New Style Assertions
```bash
# Exit status assertions
assert_success
assert_failure

# Output assertions
assert_output_contains "expected"
assert_output_empty
assert_output_not_empty
assert_output_matches "regex.*pattern"

# File assertions
assert_file_exists "$file"
assert_dir_exists "$dir"
assert_file_contains "$file" "content"

# JSON assertions
assert_json_valid "$output"
assert_json_field_equals "$output" ".status" "ok"
```

## Path Resolution

### Old Approach (Problematic)
```bash
# Different tests used different approaches
source "${BATS_TEST_DIRNAME}/../fixtures/something.bash"
source "$(dirname "$0")/../lib/helpers.bash"
source "$PWD/test/fixtures/mocks.bash"
```

### New Approach (Consistent)
```bash
# Always use the path resolver
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${TEST_DIR}/path/to/fixtures/bats/core/common_setup.bash"

# Or use helper functions after loading common_setup
vrooli_source_fixture "mocks/system/docker.bash"
vrooli_source_fixture "core/assertions.bash"
```

## Environment Variables

### Automatically Set Variables
When you use `setup_resource_test` or `setup_integration_test`, these are set automatically:

```bash
# Test isolation
TEST_NAMESPACE="test_$$_${RANDOM}"
TEST_PORT_BASE=$((8000 + (RANDOM % 1000)))

# Temporary directories (uses /dev/shm for speed)
BATS_TEST_TMPDIR="/dev/shm/vrooli-tests-$$"
MOCK_RESPONSES_DIR="$BATS_TEST_TMPDIR/mock_responses"

# Resource-specific (example for Ollama)
OLLAMA_PORT=11434
OLLAMA_BASE_URL="http://localhost:11434"
OLLAMA_CONTAINER_NAME="${TEST_NAMESPACE}_ollama"
```

## Performance Improvements

The new infrastructure is significantly faster:

1. **Lazy Loading**: Mocks are only loaded when needed
2. **In-Memory Operations**: Uses `/dev/shm` when available
3. **Optimized Assertions**: Faster pattern matching
4. **Smart Caching**: Reuses mock configurations

## Common Migration Issues

### Issue 1: Path Not Found
**Error:** `cannot source: No such file or directory`

**Solution:** Update your source path to use the new structure:
```bash
# Find your test file's location
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Source relative to that
source "${TEST_DIR}/../path/to/fixtures/bats/core/common_setup.bash"
```

### Issue 2: Missing Functions
**Error:** `command not found: setup_standard_mock_framework`

**Solution:** The function was renamed. Use the function mapping table above.

### Issue 3: Assertion Failures
**Error:** `[: too many arguments` or similar

**Solution:** Switch to new assertion functions which handle edge cases better.

### Issue 4: Mock Not Working
**Error:** Mock commands not intercepting real commands

**Solution:** Ensure you're calling the appropriate setup function:
- `setup_standard_mocks` for basic mocking
- `setup_resource_test "name"` for resource-specific mocking
- `setup_integration_test "res1" "res2"` for multiple resources

## Testing Your Migration

After migrating, verify your tests:

```bash
# Run a single test file
bats your-test.bats

# Run with verbose output for debugging
bats -v your-test.bats

# Run with TAP output for CI
bats -t your-test.bats
```

## Backward Compatibility

The following files provide backward compatibility but show deprecation warnings:
- `standard_mock_framework.bash` → redirects to `core/common_setup.bash`
- `common_setup.bash` (at root) → redirects to `core/common_setup.bash`

These will be removed in a future version. Please migrate your tests.

## Getting Help

- Check the [Setup Guide](setup-guide.md) for detailed setup instructions
- See [Assertions Reference](assertions.md) for all available assertions
- Review [Troubleshooting](troubleshooting.md) for common issues
- Look at [Examples](examples/) for working test patterns

## Migration Script

For large codebases, you can use this script to help identify tests that need migration:

```bash
#!/bin/bash
# Find tests using old patterns
echo "Tests potentially needing migration:"
grep -r "standard_mock_framework\|cleanup_standard_mock_framework" --include="*.bats" .
grep -r '\[ "\$status" -eq\|\[ "\$status" -ne' --include="*.bats" .
echo "Consider updating these to use the new infrastructure"
```

## Summary

The new infrastructure makes tests:
- **Faster**: 50-60% performance improvement
- **Cleaner**: Less boilerplate, clearer assertions
- **More Reliable**: Consistent path resolution, better isolation
- **Easier to Write**: Resource-aware setup, 54+ assertions

Start with migrating a simple test to get familiar with the patterns, then tackle more complex tests.