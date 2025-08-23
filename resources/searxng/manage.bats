#!/usr/bin/env bats
# Tests for SearXNG manage.sh script

# Source var.sh to get test paths
source "${BATS_TEST_DIRNAME}/../../../lib/utils/var.sh"

# Load Vrooli test infrastructure
source "${var_SCRIPTS_TEST_DIR}/fixtures/setup.bash"

# Setup for each test
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Set SearXNG-specific test environment
    export SEARXNG_CUSTOM_PORT="9999"
    export SEARCH_QUERY=""
    export SEARCH_FORMAT="json"
    export SEARCH_CATEGORY="general"
    export SEARCH_LANGUAGE="en"
    export FORCE="no"
    export YES="no"
    
    # Load the script without executing main
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    source "${SCRIPT_DIR}/manage.sh" || true
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

# ============================================================================
# Script Loading Tests
# ============================================================================

@test "searxng script loads without errors" {
    # Script loading happens in setup, this verifies it worked
    declare -f searxng::parse_arguments > /dev/null
    [ "$?" -eq 0 ]
}

@test "searxng defines all required functions" {
    declare -f searxng::parse_arguments > /dev/null
    [ "$?" -eq 0 ]
    declare -f searxng::main > /dev/null
    [ "$?" -eq 0 ]
}

@test "manage.sh sources all required dependencies" {
    run bash -c "source '$SCRIPT_PATH' 2>&1 | grep -v 'command not found' | head -1"
    [ "$status" -eq 0 ]
}

@test "manage.sh has correct executable permissions" {
    [ -x "$SCRIPT_PATH" ]
}

@test "manage.sh has proper shebang" {
    run head -n 1 "$SCRIPT_PATH"
    [[ "$output" =~ "#!/usr/bin/env bash" ]]
}

# ============================================================================
# Argument Parsing Tests
# ============================================================================

@test "searxng::parse_arguments sets default action to install" {
    searxng::parse_arguments
    [ "$ACTION" = "install" ]
}

@test "searxng::parse_arguments accepts install action" {
    searxng::parse_arguments --action install
    [ "$ACTION" = "install" ]
}

@test "searxng::parse_arguments accepts status action" {
    searxng::parse_arguments --action status
    [ "$ACTION" = "status" ]
}

@test "searxng::parse_arguments accepts search action" {
    run bash -c "source '$SCRIPT_PATH'; searxng::parse_arguments --action search; echo \"\$ACTION\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "search" ]]
}

@test "searxng::parse_arguments accepts api-test action" {
    run bash -c "source '$SCRIPT_PATH'; searxng::parse_arguments --action api-test; echo \"\$ACTION\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "api-test" ]]
}

@test "searxng::parse_arguments accepts benchmark action" {
    run bash -c "source '$SCRIPT_PATH'; searxng::parse_arguments --action benchmark; echo \"\$ACTION\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "benchmark" ]]
}

@test "searxng::parse_arguments accepts diagnose action" {
    run bash -c "source '$SCRIPT_PATH'; searxng::parse_arguments --action diagnose; echo \"\$ACTION\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "diagnose" ]]
}

@test "searxng::parse_arguments accepts config action" {
    run bash -c "source '$SCRIPT_PATH'; searxng::parse_arguments --action config; echo \"\$ACTION\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "config" ]]
}

@test "searxng::parse_arguments accepts examples action" {
    run bash -c "source '$SCRIPT_PATH'; searxng::parse_arguments --action examples; echo \"\$ACTION\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "examples" ]]
}

@test "searxng::parse_arguments rejects invalid action" {
    run bash -c "source '$SCRIPT_PATH'; searxng::parse_arguments --action invalid; searxng::main"
    [ "$status" -ne 0 ]
}

@test "searxng::parse_arguments sets search query parameter" {
    run bash -c "source '$SCRIPT_PATH'; searxng::parse_arguments --query 'test search'; echo \"\$SEARCH_QUERY\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "test search" ]]
}

@test "searxng::parse_arguments sets search format parameter" {
    run bash -c "source '$SCRIPT_PATH'; searxng::parse_arguments --format xml; echo \"\$SEARCH_FORMAT\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "xml" ]]
}

@test "searxng::parse_arguments sets search category parameter" {
    run bash -c "source '$SCRIPT_PATH'; searxng::parse_arguments --category images; echo \"\$SEARCH_CATEGORY\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "images" ]]
}

@test "searxng::parse_arguments sets force parameter" {
    run bash -c "source '$SCRIPT_PATH'; searxng::parse_arguments --force yes; echo \"\$FORCE\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "yes" ]]
}

@test "searxng::parse_arguments sets count parameter for benchmark" {
    run bash -c "source '$SCRIPT_PATH'; searxng::parse_arguments --count 20; echo \"\$BENCHMARK_COUNT\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "20" ]]
}

# ============================================================================
# Help and Usage Tests
# ============================================================================

@test "searxng::usage displays help information" {
    run bash -c "source '$SCRIPT_PATH'; searxng::usage"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Install and manage SearXNG metasearch engine" ]]
    [[ "$output" =~ "Examples:" ]]
    [[ "$output" =~ "Configuration:" ]]
}

@test "manage.sh shows help with --help flag" {
    run "$SCRIPT_PATH" --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Install and manage SearXNG metasearch engine" ]]
}

@test "manage.sh shows help with -h flag" {
    run "$SCRIPT_PATH" -h
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Install and manage SearXNG metasearch engine" ]]
}

# ============================================================================
# Action Routing Tests
# ============================================================================

@test "searxng::main routes install action correctly" {
    # Mock the install function to avoid actual execution
    run bash -c "
        source '$SCRIPT_PATH'
        searxng::install() { echo 'install called'; return 0; }
        export ACTION='install'
        searxng::main
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "install called" ]]
}

@test "searxng::main routes status action correctly" {
    # Mock the status function to avoid actual execution
    run bash -c "
        source '$SCRIPT_PATH'
        searxng::show_status() { echo 'status called'; return 0; }
        export ACTION='status'
        searxng::main
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "status called" ]]
}

@test "searxng::main routes search action correctly" {
    # Mock the search function to avoid actual execution
    run bash -c "
        source '$SCRIPT_PATH'
        searxng::search() { echo 'search called'; return 0; }
        searxng::interactive_search() { echo 'interactive search called'; return 0; }
        export ACTION='search'
        export SEARCH_QUERY='test'
        export SEARCH_FORMAT='json'
        export SEARCH_CATEGORY='general'
        export SEARCH_LANGUAGE='en'
        export SEARCH_PAGENO='1'
        export SEARCH_SAFESEARCH='1'
        export SEARCH_TIME_RANGE=''
        export OUTPUT_FORMAT='json'
        export RESULT_LIMIT=''
        export SAVE_FILE=''
        export APPEND_FILE=''
        searxng::main
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "search called" ]]
}

@test "searxng::main routes search action to interactive when no query" {
    # Mock the search function to avoid actual execution
    run bash -c "
        source '$SCRIPT_PATH'
        searxng::search() { echo 'search called'; return 0; }
        searxng::interactive_search() { echo 'interactive search called'; return 0; }
        export ACTION='search'
        export SEARCH_QUERY=''
        searxng::main
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "interactive search called" ]]
}

@test "searxng::main handles unknown action" {
    run bash -c "
        source '$SCRIPT_PATH'
        export ACTION='unknown'
        searxng::main
    "
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown action" ]]
}

# ============================================================================
# File Structure Tests
# ============================================================================

@test "all required configuration files exist" {
    [ -f "$CONFIG_DIR/defaults.sh" ]
    [ -f "$CONFIG_DIR/messages.sh" ]
    [ -f "$CONFIG_DIR/settings.yml.template" ]
}

@test "all required library files exist" {
    [ -f "$LIB_DIR/common.sh" ]
    [ -f "$LIB_DIR/docker.sh" ]
    [ -f "$LIB_DIR/install.sh" ]
    [ -f "$LIB_DIR/status.sh" ]
    [ -f "$LIB_DIR/config.sh" ]
    [ -f "$LIB_DIR/api.sh" ]
}

@test "docker compose file exists" {
    [ -f "$BATS_TEST_DIRNAME/docker/docker-compose.yml" ]
}

# ============================================================================
# Signal Handling Tests
# ============================================================================

@test "manage.sh sets up signal handlers" {
    run bash -c "source '$SCRIPT_PATH'; trap -p EXIT"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "searxng::cleanup" ]]
}

@test "searxng::cleanup function exists" {
    run bash -c "source '$SCRIPT_PATH'; declare -f searxng::cleanup"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "searxng::cleanup" ]]
}

# ============================================================================
# Environment Variable Tests
# ============================================================================

@test "script exports all required environment variables" {
    run bash -c "
        source '$SCRIPT_PATH'
        searxng::parse_arguments --action install --query 'test' --format json --category general
        echo \"ACTION=\$ACTION\"
        echo \"SEARCH_QUERY=\$SEARCH_QUERY\"
        echo \"SEARCH_FORMAT=\$SEARCH_FORMAT\"
        echo \"SEARCH_CATEGORY=\$SEARCH_CATEGORY\"
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "ACTION=install" ]]
    [[ "$output" =~ "SEARCH_QUERY=test" ]]
    [[ "$output" =~ "SEARCH_FORMAT=json" ]]
    [[ "$output" =~ "SEARCH_CATEGORY=general" ]]
}

# ============================================================================
# Integration Tests (Mock-based)
# ============================================================================

@test "script can be executed directly and runs successfully" {
    # Test that the script can be executed with help flag
    run bash -c "'$SCRIPT_PATH' --help"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage:" ]]
}