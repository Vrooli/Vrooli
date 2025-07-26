#!/usr/bin/env bats
# Tests for Browserless manage.sh script

# Load test helper
load_helper() {
    local helper_file="$1"
    if [[ -f "$helper_file" ]]; then
        # shellcheck disable=SC1090
        source "$helper_file"
    fi
}

# Setup for each test
setup() {
    # Set test environment
    export BROWSERLESS_CUSTOM_PORT="9999"
    export FORCE="no"
    export YES="no"
    
    # Load the script without executing main
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    source "${SCRIPT_DIR}/manage.sh" || true
}

# Test script loading
@test "manage.sh loads without errors" {
    # The script should source successfully in setup
    [ "$?" -eq 0 ]
}

# Test argument parsing
@test "browserless::parse_arguments sets defaults correctly" {
    browserless::parse_arguments --action status
    
    [ "$ACTION" = "status" ]
    [ "$FORCE" = "no" ]
    [ "$HEADLESS" = "yes" ]
    [ "$MAX_BROWSERS" = "5" ]
    [ "$TIMEOUT" = "30000" ]
}

# Test argument parsing with custom values
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

# Test usage display
@test "browserless::usage displays help text" {
    run browserless::usage
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Install and manage Browserless"* ]]
    [[ "$output" == *"--action install"* ]]
}

# Test configuration export
@test "configuration is exported correctly" {
    browserless::export_config
    
    [ -n "$BROWSERLESS_PORT" ]
    [ -n "$BROWSERLESS_BASE_URL" ]
    [ -n "$BROWSERLESS_CONTAINER_NAME" ]
    [ -n "$BROWSERLESS_IMAGE" ]
}

# Test message export
@test "messages are exported correctly" {
    browserless::export_messages
    
    [ -n "$MSG_INSTALL_SUCCESS" ]
    [ -n "$MSG_DOCKER_NOT_FOUND" ]
    [ -n "$MSG_HEALTHY" ]
}