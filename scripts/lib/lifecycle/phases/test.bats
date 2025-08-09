#!/usr/bin/env bats
################################################################################
# Tests for test.sh - Universal test phase handler
################################################################################

bats_require_minimum_version 1.5.0

# Load test infrastructure
source "${BATS_TEST_DIRNAME}/../../../__test/fixtures/setup.bash"

# Load BATS helpers
load "${BATS_TEST_DIRNAME}/../../../__test/helpers/bats-support/load"
load "${BATS_TEST_DIRNAME}/../../../__test/helpers/bats-assert/load"

# Load required mocks
load "${BATS_TEST_DIRNAME}/../../../__test/fixtures/mocks/commands.bash"
load "${BATS_TEST_DIRNAME}/../../../__test/fixtures/mocks/system.sh"
load "${BATS_TEST_DIRNAME}/../../../__test/fixtures/mocks/filesystem.sh"

setup() {
    vrooli_setup_unit_test
    
    # Reset all mocks
    mock::commands::reset
    mock::system::reset
    mock::filesystem::reset
    
    # Mock phase utilities
    mock::commands::setup_function "phase::init" 0
    mock::commands::setup_function "phase::complete" 0
    mock::commands::setup_function "phase::run_hook" 0
    mock::commands::setup_function "log::info" 0
    mock::commands::setup_function "log::debug" 0
    mock::commands::setup_function "log::header" 0
    mock::commands::setup_function "log::success" 0
    mock::commands::setup_function "log::warning" 0
    mock::commands::setup_function "log::error" 0
    mock::commands::setup_function "flow::is_yes" 1  # Default to "no"
    
    # Set up test environment
    export TEST_TYPE="all"
    export TEST_PATTERN=""
    export COVERAGE="no"
    export BAIL_ON_FAIL="yes"
    export var_ROOT_DIR="${MOCK_ROOT}"
    export var_SCRIPTS_DIR="${MOCK_ROOT}/scripts"
    export var_SCRIPTS_TEST_DIR="${MOCK_ROOT}/scripts/__test"
    export var_APP_LIFECYCLE_DIR="${MOCK_ROOT}/app/lifecycle"
    
    # Create mock package.json
    mock::filesystem::create_file "${MOCK_ROOT}/package.json" '{
        "scripts": {
            "test": "echo running tests",
            "test:unit": "echo running unit tests",
            "test:integration": "echo running integration tests",
            "test:coverage": "echo running coverage"
        }
    }'
    
    # Source test functions for testing
    source "${BATS_TEST_DIRNAME}/test.sh"
}

teardown() {
    vrooli_cleanup_test
}

################################################################################
# Shell Test Execution Tests
################################################################################

@test "test::run_shell skips when bats unavailable" {
    mock::commands::setup_command "command -v bats" "" "1"  # bats not found
    
    run test::run_shell
    
    assert_success
    mock::commands::assert::called "log::warning" "bats is not installed, skipping shell tests"
}

@test "test::run_shell finds and runs bats files" {
    mock::commands::setup_command "command -v bats" "" "0"
    mock::filesystem::create_directory "${var_SCRIPTS_TEST_DIR}"
    mock::filesystem::create_file "${var_SCRIPTS_TEST_DIR}/example.bats" "#!/usr/bin/env bats"
    mock::filesystem::create_file "${var_SCRIPTS_DIR}/manage.bats" "#!/usr/bin/env bats"
    mock::commands::setup_command "find" "${var_SCRIPTS_TEST_DIR} -name \"*.bats\" -print0" "${var_SCRIPTS_TEST_DIR}/example.bats"
    mock::commands::setup_command "find" "${var_SCRIPTS_DIR} -maxdepth 2 -name \"*.bats\" -print0" "${var_SCRIPTS_DIR}/manage.bats"
    mock::commands::setup_command "bats" "${var_SCRIPTS_TEST_DIR}/example.bats" ""
    mock::commands::setup_command "bats" "${var_SCRIPTS_DIR}/manage.bats" ""
    
    run test::run_shell
    
    assert_success
    mock::commands::assert::called "bats" "${var_SCRIPTS_TEST_DIR}/example.bats"
    mock::commands::assert::called "bats" "${var_SCRIPTS_DIR}/manage.bats"
    mock::commands::assert::called "log::success" "✅ All shell tests passed"
}

@test "test::run_shell handles test failures" {
    mock::commands::setup_command "command -v bats" "" "0"
    mock::filesystem::create_file "${var_SCRIPTS_DIR}/failing.bats" "#!/usr/bin/env bats"
    mock::commands::setup_command "find" "${var_SCRIPTS_TEST_DIR} -name \"*.bats\" -print0" ""
    mock::commands::setup_command "find" "${var_SCRIPTS_DIR} -maxdepth 2 -name \"*.bats\" -print0" "${var_SCRIPTS_DIR}/failing.bats"
    mock::commands::setup_command "bats" "${var_SCRIPTS_DIR}/failing.bats" "" "1"  # Test failure
    
    run test::run_shell
    
    assert_failure
    assert_equal "$status" 1
    mock::commands::assert::called "log::error" "❌ Failed: failing.bats"
    mock::commands::assert::called "log::error" "1 test file(s) failed"
}

################################################################################
# Unit Test Execution Tests
################################################################################

@test "test::run_unit skips when no package.json" {
    rm "${MOCK_ROOT}/package.json"
    
    run test::run_unit
    
    assert_success
    mock::commands::assert::called "log::info" "No package.json found, skipping unit tests"
}

@test "test::run_unit runs pnpm test:unit when available" {
    mock::commands::setup_command "command -v jq" "" "0"
    mock::commands::setup_command "jq" "-e '.scripts[\"test:unit\"]' '${MOCK_ROOT}/package.json'" ""
    mock::commands::setup_command "command -v pnpm" "" "0"
    mock::commands::setup_command "pnpm" "test:unit" ""
    
    run test::run_unit
    
    assert_success
    mock::commands::assert::called "pnpm" "test:unit"
    mock::commands::assert::called "log::success" "✅ Unit tests passed"
}

@test "test::run_unit falls back to npm test" {
    mock::commands::setup_command "command -v jq" "" "0"
    mock::commands::setup_command "jq" "-e '.scripts[\"test:unit\"]' '${MOCK_ROOT}/package.json'" "1"  # Not found
    mock::commands::setup_command "jq" "-e '.scripts[\"test\"]' '${MOCK_ROOT}/package.json'" ""
    mock::commands::setup_command "command -v pnpm" "" "1"  # pnpm not available
    mock::commands::setup_command "command -v npm" "" "0"
    mock::commands::setup_command "npm" "test" ""
    
    run test::run_unit
    
    assert_success
    mock::commands::assert::called "npm" "test"
}

@test "test::run_unit adds pattern when provided" {
    mock::commands::setup_command "command -v jq" "" "0"
    mock::commands::setup_command "jq" "-e '.scripts[\"test:unit\"]' '${MOCK_ROOT}/package.json'" ""
    mock::commands::setup_command "command -v pnpm" "" "0"
    mock::commands::setup_command "pnpm" "test:unit -- MyTest" ""
    
    run test::run_unit "MyTest"
    
    assert_success
    mock::commands::assert::called "pnpm" "test:unit -- MyTest"
}

@test "test::run_unit handles test failure" {
    mock::commands::setup_command "command -v jq" "" "0"
    mock::commands::setup_command "jq" "-e '.scripts[\"test:unit\"]' '${MOCK_ROOT}/package.json'" ""
    mock::commands::setup_command "command -v pnpm" "" "0"
    mock::commands::setup_command "pnpm" "test:unit" "" "1"  # Test failure
    
    run test::run_unit
    
    assert_failure
    assert_equal "$status" 1
    mock::commands::assert::called "log::error" "Unit tests failed"
}

################################################################################
# Integration Test Execution Tests
################################################################################

@test "test::run_integration runs when configured" {
    mock::commands::setup_command "command -v jq" "" "0"
    mock::commands::setup_command "jq" "-e '.scripts[\"test:integration\"]' '${MOCK_ROOT}/package.json'" ""
    mock::commands::setup_command "command -v pnpm" "" "0"
    mock::commands::setup_command "pnpm" "test:integration" ""
    
    run test::run_integration
    
    assert_success
    mock::commands::assert::called "pnpm" "test:integration"
    mock::commands::assert::called "log::success" "✅ Integration tests passed"
}

@test "test::run_integration skips when not configured" {
    mock::commands::setup_command "command -v jq" "" "0"
    mock::commands::setup_command "jq" "-e '.scripts[\"test:integration\"]' '${MOCK_ROOT}/package.json'" "1"  # Not found
    
    run test::run_integration
    
    assert_success
    mock::commands::assert::called "log::info" "No integration tests configured"
}

################################################################################
# Coverage Test Execution Tests
################################################################################

@test "test::run_coverage runs when configured" {
    mock::commands::setup_command "command -v jq" "" "0"
    mock::commands::setup_command "jq" "-e '.scripts[\"test:coverage\"]' '${MOCK_ROOT}/package.json'" ""
    mock::commands::setup_command "command -v pnpm" "" "0"
    mock::commands::setup_command "pnpm" "test:coverage" ""
    mock::filesystem::create_directory "${MOCK_ROOT}/coverage"
    
    run test::run_coverage
    
    assert_success
    mock::commands::assert::called "pnpm" "test:coverage"
    mock::commands::assert::called "log::success" "✅ Coverage tests passed"
    mock::commands::assert::called "log::info" "Coverage report available at: ${MOCK_ROOT}/coverage"
}

@test "test::run_coverage skips when not configured" {
    mock::commands::setup_command "command -v jq" "" "0"
    mock::commands::setup_command "jq" "-e '.scripts[\"test:coverage\"]' '${MOCK_ROOT}/package.json'" "1"  # Not found
    
    run test::run_coverage
    
    assert_success
    mock::commands::assert::called "log::info" "No coverage configuration found"
}

################################################################################
# Main Test Function Tests
################################################################################

@test "test::universal::main initializes phase correctly" {
    mock::commands::setup_command "command -v bats" "" "1"  # Skip shell tests
    mock::commands::setup_function "flow::is_yes" 1 1 1 1  # Skip all optional tests
    
    run test::universal::main
    
    assert_success
    mock::commands::assert::called "phase::init" "Test"
    mock::commands::assert::called "phase::complete"
}

@test "test::universal::main runs all test types by default" {
    export TEST_TYPE="all"
    mock::commands::setup_command "command -v bats" "" "0"
    mock::filesystem::create_file "${var_SCRIPTS_DIR}/test.bats" "#!/usr/bin/env bats"
    mock::commands::setup_command "find" "${var_SCRIPTS_TEST_DIR} -name \"*.bats\" -print0" ""
    mock::commands::setup_command "find" "${var_SCRIPTS_DIR} -maxdepth 2 -name \"*.bats\" -print0" "${var_SCRIPTS_DIR}/test.bats"
    mock::commands::setup_command "bats" "${var_SCRIPTS_DIR}/test.bats" ""
    mock::commands::setup_command "command -v jq" "" "0"
    mock::commands::setup_command "jq" "-e '.scripts[\"test:unit\"]' '${MOCK_ROOT}/package.json'" ""
    mock::commands::setup_command "jq" "-e '.scripts[\"test:integration\"]' '${MOCK_ROOT}/package.json'" ""
    mock::commands::setup_command "command -v pnpm" "" "0"
    mock::commands::setup_command "pnpm" "test:unit" ""
    mock::commands::setup_command "pnpm" "test:integration" ""
    
    run test::universal::main
    
    assert_success
    mock::commands::assert::called "bats" "${var_SCRIPTS_DIR}/test.bats"
    mock::commands::assert::called "pnpm" "test:unit"
    mock::commands::assert::called "pnpm" "test:integration"
}

@test "test::universal::main runs only shell tests when specified" {
    export TEST_TYPE="shell"
    mock::commands::setup_command "command -v bats" "" "0"
    mock::filesystem::create_file "${var_SCRIPTS_DIR}/test.bats" "#!/usr/bin/env bats"
    mock::commands::setup_command "find" "${var_SCRIPTS_TEST_DIR} -name \"*.bats\" -print0" ""
    mock::commands::setup_command "find" "${var_SCRIPTS_DIR} -maxdepth 2 -name \"*.bats\" -print0" "${var_SCRIPTS_DIR}/test.bats"
    mock::commands::setup_command "bats" "${var_SCRIPTS_DIR}/test.bats" ""
    
    run test::universal::main
    
    assert_success
    mock::commands::assert::called "bats" "${var_SCRIPTS_DIR}/test.bats"
    # Should not call unit or integration tests
    mock::commands::assert::not_called "pnpm" "test:unit"
    mock::commands::assert::not_called "pnpm" "test:integration"
}

@test "test::universal::main aborts on failure when bail enabled" {
    export TEST_TYPE="all"
    export BAIL_ON_FAIL="yes"
    mock::commands::setup_command "command -v bats" "" "0"
    mock::filesystem::create_file "${var_SCRIPTS_DIR}/failing.bats" "#!/usr/bin/env bats"
    mock::commands::setup_command "find" "${var_SCRIPTS_TEST_DIR} -name \"*.bats\" -print0" ""
    mock::commands::setup_command "find" "${var_SCRIPTS_DIR} -maxdepth 2 -name \"*.bats\" -print0" "${var_SCRIPTS_DIR}/failing.bats"
    mock::commands::setup_command "bats" "${var_SCRIPTS_DIR}/failing.bats" "" "1"  # Test failure
    mock::commands::setup_function "flow::is_yes" 0  # Bail on fail enabled
    
    run test::universal::main
    
    assert_failure
    assert_equal "$status" 1
    mock::commands::assert::called "log::error" "Shell tests failed, stopping"
}

@test "test::universal::main runs coverage when requested" {
    export COVERAGE="yes"
    mock::commands::setup_command "command -v bats" "" "1"  # Skip shell tests
    mock::commands::setup_command "command -v jq" "" "0"
    mock::commands::setup_command "jq" "-e '.scripts[\"test:coverage\"]' '${MOCK_ROOT}/package.json'" ""
    mock::commands::setup_command "command -v pnpm" "" "0"
    mock::commands::setup_command "pnpm" "test:coverage" ""
    mock::commands::setup_function "flow::is_yes" 1 0  # Coverage enabled
    
    run test::universal::main
    
    assert_success
    mock::commands::assert::called "pnpm" "test:coverage"
}

@test "test::universal::main runs app-specific tests when available" {
    local app_test="${var_APP_LIFECYCLE_DIR}/test.sh"
    mock::filesystem::create_file "$app_test" '#!/bin/bash\necho "app-specific tests"'
    chmod +x "$app_test"
    mock::commands::setup_command "bash" "$app_test" ""
    mock::commands::setup_command "command -v bats" "" "1"  # Skip other tests
    
    run test::universal::main
    
    assert_success
    mock::commands::assert::called "bash" "$app_test"
}

@test "test::universal::main runs hooks correctly" {
    mock::commands::setup_command "command -v bats" "" "1"  # Skip shell tests
    
    run test::universal::main
    
    assert_success
    mock::commands::assert::called "phase::run_hook" "preTest"
    mock::commands::assert::called "phase::run_hook" "postTest"
}

@test "test::universal::main reports failed test suites" {
    export TEST_TYPE="all"
    export BAIL_ON_FAIL="no"
    mock::commands::setup_command "command -v bats" "" "0"
    mock::filesystem::create_file "${var_SCRIPTS_DIR}/failing.bats" "#!/usr/bin/env bats"
    mock::commands::setup_command "find" "${var_SCRIPTS_TEST_DIR} -name \"*.bats\" -print0" ""
    mock::commands::setup_command "find" "${var_SCRIPTS_DIR} -maxdepth 2 -name \"*.bats\" -print0" "${var_SCRIPTS_DIR}/failing.bats"
    mock::commands::setup_command "bats" "${var_SCRIPTS_DIR}/failing.bats" "" "1"  # Shell tests fail
    mock::commands::setup_command "command -v jq" "" "0"
    mock::commands::setup_command "jq" "-e '.scripts[\"test:unit\"]' '${MOCK_ROOT}/package.json'" ""
    mock::commands::setup_command "command -v pnpm" "" "0"
    mock::commands::setup_command "pnpm" "test:unit" "" "1"  # Unit tests fail
    mock::commands::setup_function "flow::is_yes" 1  # Bail disabled
    
    run test::universal::main
    
    assert_failure
    assert_equal "$status" 1
    mock::commands::assert::called "log::error" "The following test suites failed:"
    mock::commands::assert::called "log::error" "  - shell"
    mock::commands::assert::called "log::error" "  - unit"
}

################################################################################
# Entry Point Tests
################################################################################

@test "test script requires lifecycle engine invocation" {
    unset LIFECYCLE_PHASE
    mock::commands::setup_function "log::error" 0
    mock::commands::setup_function "log::info" 0
    
    run bash -c 'BASH_SOURCE=("test_script"); 0="test_script"; source "${BATS_TEST_DIRNAME}/test.sh"'
    
    assert_failure
    assert_equal "$status" 1
}