#!/usr/bin/env bats

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/lib/api.sh"
SEARXNG_DIR="$BATS_TEST_DIRNAME"

# Simplified setup function
setup_test_env() {
    export SEARXNG_TEST_MODE=yes
    echo "Test environment setup"
}

@test "simple setup test" {
    setup_test_env
    [ "$SEARXNG_TEST_MODE" = "yes" ]
}

@test "source api.sh" {
    # Just try to source the api.sh file
    source "$SCRIPT_PATH"
    [ $? -eq 0 ]
}