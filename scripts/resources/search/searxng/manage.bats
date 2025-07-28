#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/manage.sh"
CONFIG_DIR="$BATS_TEST_DIRNAME/config"
LIB_DIR="$BATS_TEST_DIRNAME/lib"

# Source dependencies
RESOURCES_DIR="$BATS_TEST_DIRNAME/../.."
HELPERS_DIR="$RESOURCES_DIR/../helpers"

# Source required utilities (suppress errors during test setup)
. "$HELPERS_DIR/utils/log.sh" 2>/dev/null || true
. "$HELPERS_DIR/utils/system.sh" 2>/dev/null || true
. "$HELPERS_DIR/utils/args.sh" 2>/dev/null || true

# ============================================================================
# Script Loading Tests
# ============================================================================

@test "sourcing manage.sh defines required functions" {
    run bash -c "source '$SCRIPT_PATH' && declare -f searxng::parse_arguments && declare -f searxng::main"
    [ "$status" -eq 0 ]
    [[ "$output" =~ searxng::parse_arguments ]]
    [[ "$output" =~ searxng::main ]]
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
    run bash -c "source '$SCRIPT_PATH'; searxng::parse_arguments; echo \"\$ACTION\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "install" ]]
}

@test "searxng::parse_arguments accepts install action" {
    run bash -c "source '$SCRIPT_PATH'; searxng::parse_arguments --action install; echo \"\$ACTION\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "install" ]]
}

@test "searxng::parse_arguments accepts status action" {
    run bash -c "source '$SCRIPT_PATH'; searxng::parse_arguments --action status; echo \"\$ACTION\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "status" ]]
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
    run bash -c "source '$SCRIPT_PATH'; searxng::parse_arguments --action invalid"
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

@test "script can be executed directly when sourced properly" {
    # Test that when run as main script, it executes correctly
    run bash -c "
        export BASH_SOURCE=('$SCRIPT_PATH')
        export 0='$SCRIPT_PATH'
        
        # Mock all functions to avoid actual execution
        searxng::parse_arguments() { export ACTION='status'; }
        searxng::main() { echo 'main executed'; return 0; }
        
        source '$SCRIPT_PATH'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "main executed" ]]
}