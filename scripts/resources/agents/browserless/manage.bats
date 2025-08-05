#!/usr/bin/env bats
# Tests for Browserless manage.sh script

# Load Vrooli test infrastructure
source "$(dirname "${BATS_TEST_FILENAME}")/../../../../__test/fixtures/setup.bash"

# Expensive setup operations run once per file
setup_file() {
    # Use Vrooli service test setup
    vrooli_setup_service_test "browserless"
    
    # Load Browserless specific configuration once per file
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    BROWSERLESS_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Load configuration and manage script once
    source "${BROWSERLESS_DIR}/config/defaults.sh"
    source "${BROWSERLESS_DIR}/config/messages.sh"
    source "${SCRIPT_DIR}/manage.sh"
}

# Lightweight per-test setup
setup() {
    # Setup standard Vrooli mocks
    vrooli_auto_setup
    
    # Set test environment variables (lightweight per-test)
    export BROWSERLESS_CUSTOM_PORT="9999"
    export BROWSERLESS_CONTAINER_NAME="browserless-test"
    export BROWSERLESS_BASE_URL="http://localhost:9999"
    export FORCE="no"
    export YES="no"
    export OUTPUT_FORMAT="text"
    export QUIET="no"
    export HEADLESS="yes"
    export MAX_BROWSERS="5"
    export TIMEOUT="30000"
    
    # Export config functions
    browserless::export_config
    browserless::export_messages
    
    # Mock log functions
    log::header() { echo "=== $* ==="; }
    log::info() { echo "[INFO] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    log::success() { echo "[SUCCESS] $*"; }
    log::warning() { echo "[WARNING] $*" >&2; }
    export -f log::header log::info log::error log::success log::warning
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

# ============================================================================
# Script Loading Tests  
# ============================================================================

@test "manage.sh loads without errors" {
    # Should load successfully in setup_file
    [ "$?" -eq 0 ]
}

@test "manage.sh defines required functions" {
    # Functions should be available from setup_file
    declare -f browserless::parse_arguments >/dev/null
    declare -f browserless::main >/dev/null
    declare -f browserless::install >/dev/null
    declare -f browserless::status >/dev/null
}

# ============================================================================
# Argument Parsing Tests
# ============================================================================

@test "browserless::parse_arguments sets defaults correctly" {
    browserless::parse_arguments --action status
    
    [ "$ACTION" = "status" ]
    [ "$FORCE" = "no" ]
    [ "$HEADLESS" = "yes" ]
    [ "$MAX_BROWSERS" = "5" ]
    [ "$TIMEOUT" = "30000" ]
}

@test "browserless::parse_arguments handles custom values" {
    browserless::parse_arguments \
        --action install \
        --force yes \
        --max-browsers 10 \
        --headless no \
        --timeout 60000
    
    [ "$ACTION" = "install" ]
    [ "$FORCE" = "yes" ]
    [ "$MAX_BROWSERS" = "10" ]
    [ "$HEADLESS" = "no" ]
    [ "$TIMEOUT" = "60000" ]
}

# ============================================================================
# Help and Usage Tests
# ============================================================================

@test "browserless::usage displays help text" {
    run browserless::usage
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Install and manage Browserless" ]]
    [[ "$output" =~ "--action install" ]]
}

# ============================================================================
# Configuration Tests
# ============================================================================

@test "browserless::export_config exports variables correctly" {
    browserless::export_config
    
    [ -n "$BROWSERLESS_PORT" ]
    [ -n "$BROWSERLESS_BASE_URL" ]
    [ -n "$BROWSERLESS_CONTAINER_NAME" ]
    [ -n "$BROWSERLESS_IMAGE" ]
}

@test "browserless::export_messages exports variables correctly" {
    browserless::export_messages
    
    [ -n "$MSG_INSTALL_SUCCESS" ]
    [ -n "$MSG_DOCKER_NOT_FOUND" ]
    [ -n "$MSG_HEALTHY" ]
}