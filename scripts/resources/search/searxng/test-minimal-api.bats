#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/lib/api.sh"
SEARXNG_DIR="$BATS_TEST_DIRNAME"

# Minimal setup
setup_searxng_api_test_env() {
    export SEARXNG_TEST_MODE=yes
    export SEARXNG_PORT="8100"
    export SEARXNG_BASE_URL="http://localhost:8100"
    export SEARXNG_DATA_DIR="$HOME/.searxng"
    export MOCK_SEARXNG_HEALTHY="yes"
}

@test "sourcing api.sh defines required functions" {
    setup_searxng_api_test_env
    
    # Source the api.sh file
    source "$SCRIPT_PATH"
    
    # Check one function exists
    declare -F searxng::search >/dev/null 2>&1
}