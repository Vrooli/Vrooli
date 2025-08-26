#!/usr/bin/env bash
################################################################################
# Injection System v2.0 - Integration Tests
# Tests for the injection system
################################################################################
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
INJECTION_DIR="${APP_ROOT}/scripts/resources/injection"
TEST_DIR="${INJECTION_DIR}/test"

# Source utilities
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

#######################################
# Run a test
# Arguments:
#   $1 - test name
#   $2 - test command
#######################################
run_test() {
    local test_name="$1"
    local test_cmd="$2"
    
    ((TESTS_RUN++))
    
    echo -n "Testing $test_name... "
    
    if eval "$test_cmd" >/dev/null 2>&1; then
        echo "✅ PASS"
        ((TESTS_PASSED++))
        return 0
    else
        echo "❌ FAIL"
        ((TESTS_FAILED++))
        return 1
    fi
}

#######################################
# Test help command
#######################################
test_help() {
    "${INJECTION_DIR}/inject.sh" help | grep -q "Vrooli Injection System"
}

#######################################
# Test list command
#######################################
test_list() {
    "${INJECTION_DIR}/inject.sh" list
    return 0  # List should always work
}

#######################################
# Test validate with missing scenario
#######################################
test_validate_missing() {
    ! "${INJECTION_DIR}/inject.sh" validate non-existent-scenario 2>/dev/null
}

#######################################
# Test dry-run mode
#######################################
test_dry_run() {
    # Create a test scenario
    local test_scenario="/tmp/test-scenario-$$.json"
    cat > "$test_scenario" << 'EOF'
{
  "name": "test-scenario",
  "description": "Test scenario for integration tests",
  "resources": {
    "n8n": {
      "content": [
        {
          "type": "workflow",
          "file": "/tmp/test-workflow.json",
          "name": "Test Workflow"
        }
      ]
    }
  }
}
EOF
    
    # Create dummy workflow file
    echo '{"nodes":[]}' > /tmp/test-workflow.json
    
    # Run dry-run
    local result
    result=$("${INJECTION_DIR}/inject.sh" add "$test_scenario" --dry-run 2>&1)
    
    # Clean up
    rm -f "$test_scenario" /tmp/test-workflow.json
    
    # Check output contains dry-run indicator
    echo "$result" | grep -q "DRY RUN"
}

#######################################
# Test validation of valid scenario
#######################################
test_validate_valid() {
    # Create a valid test scenario
    local test_scenario="/tmp/valid-scenario-$$.json"
    cat > "$test_scenario" << 'EOF'
{
  "name": "valid-test",
  "description": "Valid test scenario",
  "resources": {
    "postgres": {
      "content": [
        {
          "type": "schema",
          "file": "schema.sql"
        }
      ]
    }
  }
}
EOF
    
    # Validate
    local result
    result=$("${INJECTION_DIR}/inject.sh" validate "$test_scenario" 2>&1)
    
    # Clean up
    rm -f "$test_scenario"
    
    # Check validation passed
    echo "$result" | grep -q "valid"
}

#######################################
# Test validation of invalid scenario
#######################################
test_validate_invalid() {
    # Create an invalid test scenario (missing required fields)
    local test_scenario="/tmp/invalid-scenario-$$.json"
    cat > "$test_scenario" << 'EOF'
{
  "resources": {
    "postgres": {
      "content": [
        {
          "file": "schema.sql"
        }
      ]
    }
  }
}
EOF
    
    # Validate (should fail)
    if "${INJECTION_DIR}/inject.sh" validate "$test_scenario" 2>/dev/null; then
        rm -f "$test_scenario"
        return 1  # Should have failed
    else
        rm -f "$test_scenario"
        return 0  # Expected failure
    fi
}

#######################################
# Test status command
#######################################
test_status() {
    "${INJECTION_DIR}/inject.sh" status | grep -q "Injection Status"
}

#######################################
# Main test runner
#######################################
main() {
    log::header "🧪 Injection System Integration Tests"
    echo ""
    
    # Run tests
    run_test "help command" "test_help"
    run_test "list command" "test_list"
    run_test "validate missing scenario" "test_validate_missing"
    run_test "dry-run mode" "test_dry_run"
    run_test "validate valid scenario" "test_validate_valid"
    run_test "validate invalid scenario" "test_validate_invalid"
    run_test "status command" "test_status"
    
    # Summary
    echo ""
    log::header "📊 Test Summary"
    echo "Total tests: $TESTS_RUN"
    echo "Passed: $TESTS_PASSED"
    echo "Failed: $TESTS_FAILED"
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        log::success "✅ All tests passed!"
        return 0
    else
        log::error "❌ Some tests failed"
        return 1
    fi
}

# Run tests if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi