#!/usr/bin/env bats
# Tests for Browserless simplified injection system
bats_require_minimum_version 1.5.0

# Setup paths and source var.sh first
SCRIPT_DIR="$(builtin cd "${BATS_TEST_FILENAME%/*}" && builtin pwd)"
# shellcheck disable=SC1091
source "$(builtin cd "${SCRIPT_DIR%/*/*/*}" && builtin pwd)/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Load Vrooli test infrastructure using var_ variables
# shellcheck disable=SC1091
source "${var_TEST_DIR}/fixtures/setup.bash"

# Expensive setup operations run once per file
setup_file() {
    # Use Vrooli service test setup
    vrooli_setup_service_test "browserless"
    
    # SCRIPT_DIR already set at file level
    export MOCK_DIR="${var_TEST_DIR}/fixtures/mocks"
    
    # Load common resources
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"
    
    # Load the simplified injection system
    # shellcheck disable=SC1091  
    source "${SCRIPT_DIR}/lib/inject.sh"
    
    # Export simplified injection functions for BATS subshells
    export -f browserless::inject
    export -f browserless::validate_injection
    export -f browserless::injection_status
    export -f browserless::cleanup_injection
    export -f browserless::show_injection_info
}

# Setup for each test
setup() {
    # Use Vrooli test directories
    export BROWSERLESS_DATA_DIR="${BATS_TEST_TMPDIR}/browserless-data"
    export BROWSERLESS_CONTAINER_NAME="test-browserless"
    export BROWSERLESS_PORT="3001"
    
    # Create test directories
    mkdir -p "$BROWSERLESS_DATA_DIR"
    
    # Mock docker and http utilities for testing
    docker() {
        case "$1" in
            ps)
                if [[ "$*" == *"test-browserless"* ]]; then
                    echo "test-browserless"
                fi
                ;;
        esac
    }
    
    http::check_endpoint() {
        # Mock successful endpoint check
        return 0
    }
    
    docker::is_running() {
        local container_name="$1"
        [[ "$container_name" == "test-browserless" ]]
    }
    
    export -f docker
    export -f http::check_endpoint  
    export -f docker::is_running
}

# Cleanup after each test
teardown() {
    if [[ -d "$BROWSERLESS_DATA_DIR" ]]; then
        rm -rf "$BROWSERLESS_DATA_DIR"
    fi
}

@test "browserless::inject creates test data directory" {
    run browserless::inject
    
    [ "$status" -eq 0 ]
    [ -d "$BROWSERLESS_DATA_DIR/test-data" ]
}

@test "browserless::inject creates validation scripts" {
    run browserless::inject
    
    [ "$status" -eq 0 ]
    [ -f "$BROWSERLESS_DATA_DIR/test-data/validation.js" ]
    [ -f "$BROWSERLESS_DATA_DIR/test-data/screenshot-test.js" ]
}

@test "browserless::inject validates after creating files" {
    run browserless::inject
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Test data injected and validated successfully"* ]]
}

@test "browserless::inject fails when container not running" {
    # Mock container not running
    docker::is_running() {
        return 1
    }
    export -f docker::is_running
    
    run browserless::inject
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Browserless container not running"* ]]
}

@test "browserless::validate_injection checks test directory exists" {
    run browserless::validate_injection
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Test data directory not found"* ]]
}

@test "browserless::validate_injection checks required files exist" {
    mkdir -p "$BROWSERLESS_DATA_DIR/test-data"
    
    run browserless::validate_injection
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Validation script not found"* ]]
}

@test "browserless::validate_injection passes with complete setup" {
    # Create test directory and files
    mkdir -p "$BROWSERLESS_DATA_DIR/test-data"
    echo "test content" > "$BROWSERLESS_DATA_DIR/test-data/validation.js"
    echo "test content" > "$BROWSERLESS_DATA_DIR/test-data/screenshot-test.js"
    
    run browserless::validate_injection
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"All injection validation checks passed"* ]]
}

@test "browserless::injection_status shows no data when empty" {
    run browserless::injection_status
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"No test data found"* ]]
}

@test "browserless::injection_status shows test files when present" {
    # Create test setup
    mkdir -p "$BROWSERLESS_DATA_DIR/test-data"
    echo "test" > "$BROWSERLESS_DATA_DIR/test-data/validation.js"
    echo "test" > "$BROWSERLESS_DATA_DIR/test-data/screenshot-test.js"
    
    run browserless::injection_status
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Test script files: 2"* ]]
    [[ "$output" == *"validation.js"* ]]
    [[ "$output" == *"screenshot-test.js"* ]]
}

@test "browserless::cleanup_injection removes test data" {
    # Create test data
    mkdir -p "$BROWSERLESS_DATA_DIR/test-data"
    echo "test" > "$BROWSERLESS_DATA_DIR/test-data/validation.js"
    
    run browserless::cleanup_injection
    
    [ "$status" -eq 0 ]
    [ ! -d "$BROWSERLESS_DATA_DIR/test-data" ]
    [[ "$output" == *"Test data cleaned up"* ]]
}

@test "browserless::cleanup_injection handles missing test data gracefully" {
    run browserless::cleanup_injection
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"No test data to clean up"* ]]
}

@test "manage.sh inject action calls browserless::inject" {
    # Test via manage.sh interface
    run "${SCRIPT_DIR}/manage.sh" --action inject --yes yes
    
    [ "$status" -eq 0 ]
    [ -d "$BROWSERLESS_DATA_DIR/test-data" ]
}

@test "manage.sh injection-status action calls browserless::injection_status" {
    # Create some test data first
    mkdir -p "$BROWSERLESS_DATA_DIR/test-data"
    echo "test" > "$BROWSERLESS_DATA_DIR/test-data/validation.js"
    
    run "${SCRIPT_DIR}/manage.sh" --action injection-status --yes yes
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Browserless Injection Status"* ]]
}