#!/usr/bin/env bats
# ====================================================================
# Tests for Test Contract Parsing System
# ====================================================================

# Source trash module for safe test cleanup
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/scripts/resources/tests/framework"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Load test helpers
load "${BATS_TEST_DIRNAME}/../../../__test/helpers/bats-support/load.bash"
load "${BATS_TEST_DIRNAME}/../../../__test/helpers/bats-assert/load.bash"

# Test setup and teardown
setup() {
    # Create temporary test directory
    export TEST_DIR="${BATS_TMPDIR}/contract_parsing_test_$$"
    mkdir -p "$TEST_DIR"
    
    # Create mock contracts directory structure
    export TEST_CONTRACTS_DIR="$TEST_DIR/contracts/v1.0"
    mkdir -p "$TEST_CONTRACTS_DIR"
    
    # Create a minimal core contract for testing
    cat > "$TEST_CONTRACTS_DIR/core.yaml" << 'EOF'
version: "1.0"
contract_type: "core"
description: "Core contract for resource validation"

required_actions:
  install:
    description: "Install the resource"
  start:
    description: "Start the resource"
  stop:
    description: "Stop the resource"
  status:
    description: "Check resource status"
  cleanup:
    description: "Clean up resource"

help_patterns:
  - "--help"
  - "-h"

error_handling:
  - "set -euo pipefail"
  - "trap cleanup EXIT"

required_files:
  - "manage.sh"
  - "config/defaults.sh"
EOF

    # Create a simple AI category contract
    cat > "$TEST_CONTRACTS_DIR/ai.yaml" << 'EOF'
version: "1.0"
contract_type: "category"
extends: "core.yaml"
description: "AI resource validation contract"

additional_actions:
  models:
    description: "Manage AI models"
  chat:
    description: "Chat interface"
EOF

    # Create mock resource directories and manage.sh scripts
    export TEST_RESOURCES_DIR="$TEST_DIR/resources"
    mkdir -p "$TEST_RESOURCES_DIR/ai/ollama"
    mkdir -p "$TEST_RESOURCES_DIR/automation/node-red"
    
    # Create a test manage.sh script
    cat > "$TEST_RESOURCES_DIR/ai/ollama/manage.sh" << 'EOF'
#!/usr/bin/env bash
set -euo pipefail

case "${1:-}" in
    "install")
        echo "Installing ollama"
        ;;
    "start")
        echo "Starting ollama"
        ;;
    "stop")
        echo "Stopping ollama"
        ;;
    "status")
        echo "Checking status"
        ;;
    "cleanup")
        echo "Cleaning up"
        ;;
    "--help"|"-h")
        echo "Usage: $0 {install|start|stop|status|cleanup}"
        ;;
    *)
        echo "Unknown action: ${1:-}"
        exit 1
        ;;
esac
EOF
    chmod +x "$TEST_RESOURCES_DIR/ai/ollama/manage.sh"
    
    # Source the test contract parsing system
    source "${BATS_TEST_DIRNAME}/test-contract-parsing.sh"
    
    # Initialize with test contracts
    contract_parser::init "$TEST_CONTRACTS_DIR"
}

teardown() {
    # Clean up test directory
    if [[ -d "$TEST_DIR" ]]; then
        trash::safe_remove "$TEST_DIR" --test-cleanup
    fi
    
    # Clean up contract parser
    test_contract_parsing::cleanup_tests 2>/dev/null || true
}

@test "test_contract_parsing::test_contract_loading: loads core contract" {
    run test_contract_parsing::test_contract_loading
    
    assert_success
    assert_output --partial "✅ Core contract loaded"
}

@test "test_contract_parsing::test_required_actions: validates required actions" {
    run test_contract_parsing::test_required_actions
    
    assert_success
    assert_output --partial "✅ ai has"
    assert_output --partial "required actions"
}

@test "test_contract_parsing::test_help_patterns: validates help patterns" {
    run test_contract_parsing::test_help_patterns
    
    assert_success
    assert_output --partial "✅ Found"
    assert_output --partial "help patterns"
}

@test "test_contract_parsing::test_real_resources: analyzes real resources" {
    run test_contract_parsing::test_real_resources
    
    # May succeed or have warnings since we're testing with mock resources
    # Just ensure it doesn't crash
    [[ $status -eq 0 ]] || [[ $status -eq 1 ]]
    assert_output --partial "Testing ollama"
}

@test "test_contract_parsing::test_contract_validation: validates contract syntax" {
    run test_contract_parsing::test_contract_validation
    
    assert_success
    assert_output --partial "✅ core.yaml: Syntax validation passed"
}

@test "test_contract_parsing::run_all_tests: executes full test suite" {
    # Run the full test suite but redirect to avoid test output pollution
    run test_contract_parsing::run_all_tests
    
    # Should complete without crashing (may succeed or fail based on test conditions)
    [[ $status -eq 0 ]] || [[ $status -eq 1 ]]
    assert_output --partial "Test Suite Summary:"
}

@test "contract_parser::load_contract: loads core contract correctly" {
    run contract_parser::load_contract "core.yaml"
    
    assert_success
    # Output should be a file path to the loaded/merged contract
    assert_output --regexp ".*core.*yaml"
}

@test "contract_parser::get_required_actions: extracts required actions for core" {
    run contract_parser::get_required_actions "core"
    
    assert_success
    assert_output --partial "install"
    assert_output --partial "start"
    assert_output --partial "stop"
    assert_output --partial "status"
    assert_output --partial "cleanup"
}

@test "contract_parser::get_help_patterns: extracts help patterns" {
    run contract_parser::get_help_patterns "ai"
    
    assert_success
    assert_output --partial "--help"
    assert_output --partial "-h"
}

@test "script_analyzer::extract_script_actions: extracts actions from test script" {
    run script_analyzer::extract_script_actions "$TEST_RESOURCES_DIR/ai/ollama/manage.sh"
    
    assert_success
    assert_output --partial "install"
    assert_output --partial "start"
    assert_output --partial "stop"
    assert_output --partial "status"
    assert_output --partial "cleanup"
}

@test "cleanup function works correctly" {
    # Test that cleanup doesn't crash
    run test_contract_parsing::cleanup_tests
    
    assert_success
    assert_output --partial "cleanup completed"
}
