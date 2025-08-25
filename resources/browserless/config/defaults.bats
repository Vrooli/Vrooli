#!/usr/bin/env bats
# Tests for Browserless defaults.sh configuration
bats_require_minimum_version 1.5.0

# Setup paths and source var.sh first
SCRIPT_DIR="$(builtin cd "${BATS_TEST_FILENAME%/*}" && builtin pwd)"
# shellcheck disable=SC1091
source "$(builtin cd "${SCRIPT_DIR%/*/*/*/*/*}" && builtin pwd)/lib/utils/var.sh"

# Load Vrooli test infrastructure using var_ variables
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/fixtures/setup.bash"

# Expensive setup operations run once per file
setup_file() {
    # Use Vrooli service test setup
    vrooli_setup_service_test "browserless"
    
    # Load the defaults once per file
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/defaults.sh"
    
    # Export the setup_file directory for use in setup()
    export SETUP_FILE_SCRIPT_DIR="$SCRIPT_DIR"
}

# Lightweight per-test setup
setup() {
    # Use the directory from setup_file
    SCRIPT_DIR="${SETUP_FILE_SCRIPT_DIR}"
    
    # Set test environment (lightweight per-test)
    export BROWSERLESS_CUSTOM_PORT="9999"
    export MAX_BROWSERS="3"
    export TIMEOUT="15000"
    export HEADLESS="no"
    
    # Mock resources function (lightweight)
    resources::get_default_port() {
        echo "4110"
    }
    export -f resources::get_default_port
    
    # Re-source defaults to ensure functions are available in test scope
    source "${SCRIPT_DIR}/defaults.sh"
}

# Test configuration export
@test "browserless::export_config sets all required variables" {
    browserless::export_config
    
    # Test basic configuration
    [ "$BROWSERLESS_PORT" = "9999" ]  # Custom port should override default
    [ "$BROWSERLESS_BASE_URL" = "http://localhost:9999" ]
    [ "$BROWSERLESS_CONTAINER_NAME" = "browserless" ]
    [ "$BROWSERLESS_IMAGE" = "ghcr.io/browserless/chrome:latest" ]
    
    # Test browser configuration
    [ "$BROWSERLESS_MAX_BROWSERS" = "3" ]
    [ "$BROWSERLESS_TIMEOUT" = "15000" ]
    [ "$BROWSERLESS_HEADLESS" = "no" ]
    
    # Test network configuration
    [ "$BROWSERLESS_NETWORK_NAME" = "browserless-network" ]
    
    # Test health check configuration
    [ "$BROWSERLESS_HEALTH_CHECK_INTERVAL" = "5" ]
    [ "$BROWSERLESS_HEALTH_CHECK_MAX_ATTEMPTS" = "12" ]
    [ "$BROWSERLESS_API_TIMEOUT" = "10" ]
}

# Test default port fallback
@test "browserless::export_config uses default port when no custom port set" {
    unset BROWSERLESS_CUSTOM_PORT
    
    browserless::export_config
    
    [ "$BROWSERLESS_PORT" = "4110" ]
    [ "$BROWSERLESS_BASE_URL" = "http://localhost:4110" ]
}

# Test data directory configuration
@test "browserless::export_config sets data directory correctly" {
    browserless::export_config
    
    [[ "$BROWSERLESS_DATA_DIR" == *".browserless" ]]
}

# Test Docker configuration
@test "browserless::export_config sets Docker options correctly" {
    browserless::export_config
    
    [ "$BROWSERLESS_DOCKER_SHM_SIZE" = "2gb" ]
    [ "$BROWSERLESS_DOCKER_CAPS" = "SYS_ADMIN" ]
    [ "$BROWSERLESS_DOCKER_SECCOMP" = "unconfined" ]
}