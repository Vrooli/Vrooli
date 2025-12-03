# Resource Unit Testing Guide

This guide covers best practices, common pitfalls, and troubleshooting for testing **Vrooli resources** (PostgreSQL, Redis, Ollama, etc.) - not scenario application code. For scenario unit testing, see [Scenario Unit Testing Guide](scenario-unit-testing.md).

## 🎯 Purpose of Unit Tests

Unit tests validate that individual resource functions exist and behave correctly **without requiring the actual service to be running**. They should:

- ✅ Complete in <60 seconds
- ✅ Test function existence and basic behavior
- ✅ Validate configuration variables
- ✅ Check CLI framework integration
- ❌ NOT require Docker containers or external services

## 📋 Unit Test Structure

All resources should follow this standardized pattern:

```bash
#!/usr/bin/env bash
set -euo pipefail

# Get directory paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
RESOURCE_DIR="${APP_ROOT}/resources/[resource-name]"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"
# ... other lib sources

# Test counters
tests_passed=0
tests_failed=0

# Test functions here...

# Summary
log::info "Unit Test Summary:"
log::success "  Passed: $tests_passed"
if [[ $tests_failed -gt 0 ]]; then
    log::error "  Failed: $tests_failed"
    exit 1
else
    log::success "All unit tests passed!"
    exit 0
fi
```

## ⚠️ Critical Pitfalls & How to Avoid Them

### 1. Arithmetic Expansion with set -e

**❌ DANGEROUS - Will cause script to exit early:**
```bash
set -euo pipefail
((tests_passed++))  # Can exit with code 1 in some bash versions
```

**✅ SAFE - Always use this pattern:**
```bash
tests_passed=$((tests_passed + 1))
tests_failed=$((tests_failed + 1))
```

**Why this happens:** The `((variable++))` syntax can return exit code 1 in certain bash environments, causing immediate script termination when `set -e` is enabled.

### 2. Function Existence Testing

**❌ WRONG - Tests internal bash state:**
```bash
if command -v "function_name" &>/dev/null; then
```

**✅ CORRECT - Tests actual function declarations:**
```bash
if declare -f "function_name" &>/dev/null; then
```

**Why:** `command -v` tests if something is executable in PATH, not if it's a declared function in the current shell session.

### 3. Library Sourcing Order

**❌ WRONG - May cause undefined variable errors:**
```bash
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/config/defaults.sh"  # Variables used by core.sh
```

**✅ CORRECT - Always source config first:**
```bash
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"
```

### 4. Function Name Mismatches

**❌ COMMON MISTAKE - Test and implementation don't match:**
```bash
# Test expects:
assert_function_exists "resource::api::process_file"
# But implementation has:
resource::process_document() { ... }
```

**✅ SOLUTION - Verify actual function names:**
```bash
# Before writing tests, check what functions actually exist:
grep -E "^[a-z_]+::[a-z_:]+\(\)" lib/*.sh
```

### 5. CLI Framework Integration Testing

**❌ WRONG - Tests non-existent array:**
```bash
if [[ -n "${CLI_COMMAND_HANDLERS[test::smoke]:-}" ]]; then
```

**✅ CORRECT - Test framework initialization:**
```bash
if declare -A CLI_COMMAND_HANDLERS &>/dev/null; then
    if [[ -n "${CLI_COMMAND_HANDLERS[test::smoke]:-}" ]]; then
        # Test the handler
    fi
fi
```

## 🧪 Standard Test Categories

Every resource unit test should include these test categories:

### 1. Core Lifecycle Functions
```bash
test_function_exists "resource::install" "Install function"
test_function_exists "resource::uninstall" "Uninstall function" 
test_function_exists "resource::start" "Start function"
test_function_exists "resource::stop" "Stop function"
test_function_exists "resource::restart" "Restart function"
test_function_exists "resource::status" "Status function"
```

### 2. Configuration Variables
```bash
if [[ -n "${RESOURCE_PORT:-}" ]]; then
    log::success "✓ Port configured: $RESOURCE_PORT"
    tests_passed=$((tests_passed + 1))
else
    log::error "✗ Port not configured"
    tests_failed=$((tests_failed + 1))
fi
```

### 3. Content Management (if applicable)
```bash
test_function_exists "resource::content::add" "Content add function"
test_function_exists "resource::content::list" "Content list function"
test_function_exists "resource::content::get" "Content get function" 
test_function_exists "resource::content::remove" "Content remove function"
```

### 4. Health Check Functions
```bash
test_function_exists "resource::check_health" "Health check function"
test_function_exists "resource::container_exists" "Container exists check"
test_function_exists "resource::container_running" "Container running check"
```

## 🔧 Test Helper Functions

Use these standardized helper functions:

```bash
# Test function existence
test_function_exists() {
    local func_name="$1"
    local description="${2:-Function $func_name}"
    
    if declare -f "$func_name" &>/dev/null; then
        log::success "✓ $description exists"
        tests_passed=$((tests_passed + 1))
        return 0
    else
        log::error "✗ $description missing"
        tests_failed=$((tests_failed + 1))
        return 1
    fi
}

# Test function execution (without side effects)
test_function_runs() {
    local func_name="$1"
    local description="${2:-Function $func_name}"
    local expected_exit="${3:-0}"
    
    # Only test if function exists
    if ! declare -f "$func_name" &>/dev/null; then
        log::error "✗ $description does not exist"
        tests_failed=$((tests_failed + 1))
        return 1
    fi
    
    # Test execution with timeout
    if timeout 5 bash -c "$func_name" &>/dev/null; then
        local actual_exit=$?
        if [[ $actual_exit -eq $expected_exit ]]; then
            log::success "✓ $description runs successfully"
            tests_passed=$((tests_passed + 1))
            return 0
        else
            log::error "✗ $description returned exit code $actual_exit (expected $expected_exit)"
            tests_failed=$((tests_failed + 1))
            return 1
        fi
    else
        log::error "✗ $description failed to execute or timed out"
        tests_failed=$((tests_failed + 1))
        return 1
    fi
}
```

## 🚨 Troubleshooting Common Failures

### Unit Test Exits Immediately After First Test

**Symptoms:**
```
[INFO] Test 1: Core lifecycle functions...
[SUCCESS] ✓ Install function exists
[ERROR] unit tests failed
```

**Cause:** Arithmetic expansion issue with `((variable++))`

**Solution:** 
1. Search for all instances: `grep -n '((' test/phases/test-unit.sh`
2. Replace all with safe form: `variable=$((variable + 1))`

### Function Not Found Errors

**Symptoms:**
```
[ERROR] ✗ Content add function missing
```

**Diagnosis Steps:**
1. Check if function actually exists: `grep -r "content::add" lib/`
2. Verify function name matches test expectation exactly
3. Ensure proper library sourcing order
4. Check if function is in a different namespace

### CLI Framework Integration Failures

**Symptoms:**
```
[ERROR] ✗ Install handler not registered
```

**Diagnosis Steps:**
1. Verify CLI framework is properly initialized
2. Check if handlers are registered after cli::init
3. Ensure CLI_COMMAND_HANDLERS array exists
4. Verify handler registration syntax

### Configuration Variable Missing

**Symptoms:**
```
[ERROR] ✗ Port not configured
```

**Diagnosis Steps:**
1. Check if defaults.sh is sourced before testing
2. Verify variable name matches between config and test
3. Ensure proper export statements in config
4. Check for typos in variable names

## ✅ Unit Test Checklist

Before submitting a resource, ensure your unit test:

- [ ] Uses safe arithmetic expansion: `var=$((var + 1))`
- [ ] Tests function existence with `declare -f`
- [ ] Sources configuration before libraries
- [ ] Has consistent function names between test and implementation
- [ ] Includes all required test categories
- [ ] Uses standardized helper functions
- [ ] Completes in <60 seconds
- [ ] Has proper error handling and exit codes
- [ ] Includes summary with pass/fail counts
- [ ] Does not require external services to be running

## See Also

### Related Guides
- [Scenario Unit Testing](scenario-unit-testing.md) - Testing scenario application code
- [CLI Testing](cli-testing.md) - BATS testing for CLIs
- [Phased Testing](phased-testing.md) - How resource tests fit into phases

### Reference
- [Test Runners](../reference/test-runners.md) - Language-specific test runners
- [Phase Catalog](../reference/phase-catalog.md) - Phase definitions
- [Presets](../reference/presets.md) - Test preset configurations

### Concepts
- [Architecture](../concepts/architecture.md) - Go orchestrator design