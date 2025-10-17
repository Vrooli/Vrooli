#!/usr/bin/env bats
# Comprehensive CLI tests for visited-tracker
# Covers all commands, options, error cases, and integration scenarios

################################################################################
# Test Setup and Utilities
################################################################################

setup() {
    export API_PORT="${API_PORT:-17695}"
    export API_BASE_URL="http://localhost:${API_PORT}"
    export TEST_CAMPAIGN_ID=""
    export VISITED_TRACKER_TEST_ROOT="$(mktemp -d /tmp/visited-tracker-cli-XXXXXX)"
    export TEST_FILE_DIR="${VISITED_TRACKER_TEST_ROOT}/files"
    
    # Ensure CLI is available
    if ! command -v visited-tracker >/dev/null 2>&1; then
        skip "visited-tracker CLI not installed"
    fi
    
    # Create test files for scenarios that need them
    mkdir -p "$TEST_FILE_DIR"
    echo "// Test file content" > "$TEST_FILE_DIR/test1.js"
    echo "/* Another test file */" > "$TEST_FILE_DIR/test2.js"
    echo "console.log('test');" > "$TEST_FILE_DIR/test3.js"

    # Create a disposable campaign when service is running
    if service_running; then
        if ! command -v jq >/dev/null 2>&1; then
            skip "jq is required for CLI tests"
        fi

        local payload
        payload=$(jq -n --arg name "cli-bats-$(date +%s)" --arg from_agent "cli-tests" --argjson patterns '["**/*.js"]' '{name:$name, from_agent:$from_agent, patterns:$patterns}')
        local response
        response=$(curl -sf -X POST "$API_BASE_URL/api/v1/campaigns" -H "Content-Type: application/json" -d "$payload" 2>/dev/null || echo "")
        if echo "$response" | jq -e '.id' >/dev/null 2>&1; then
            TEST_CAMPAIGN_ID=$(echo "$response" | jq -r '.id')
            export TEST_CAMPAIGN_ID
            export VISITED_TRACKER_CAMPAIGN_ID="$TEST_CAMPAIGN_ID"
        else
            skip "Unable to create test campaign via API"
        fi
    fi
}

teardown() {
    if [ -n "${VISITED_TRACKER_TEST_ROOT:-}" ]; then
        rm -rf "${VISITED_TRACKER_TEST_ROOT}" 2>/dev/null || true
        unset VISITED_TRACKER_TEST_ROOT
        unset TEST_FILE_DIR
    fi

    if [ -n "${TEST_CAMPAIGN_ID:-}" ]; then
        curl -sf -X DELETE "$API_BASE_URL/api/v1/campaigns/$TEST_CAMPAIGN_ID" >/dev/null 2>&1 || true
        unset VISITED_TRACKER_CAMPAIGN_ID
    fi
}

# Helper function to check if service is running
service_running() {
    timeout 3 curl -sf "http://localhost:${API_PORT}/health" >/dev/null 2>&1
}

# Helper function to validate JSON output
is_valid_json() {
    echo "$1" | jq . >/dev/null 2>&1
}

################################################################################
# Basic Command Tests
################################################################################

@test "CLI help command works" {
    run visited-tracker help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Visited Tracker CLI" ]]
    [[ "$output" =~ "Usage:" ]]
    [[ "$output" =~ "Commands:" ]]
}

@test "CLI help shows all main commands" {
    run visited-tracker help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "visit" ]]
    [[ "$output" =~ "sync" ]]
    [[ "$output" =~ "least-visited" ]]
    [[ "$output" =~ "most-stale" ]]
    [[ "$output" =~ "coverage" ]]
    [[ "$output" =~ "import" ]]
    [[ "$output" =~ "export" ]]
    [[ "$output" =~ "status" ]]
    [[ "$output" =~ "version" ]]
}

@test "CLI help shows examples section" {
    run visited-tracker help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Examples:" ]]
    [[ "$output" =~ "--context security" ]]
    [[ "$output" =~ "--limit 10" ]]
}

@test "CLI version command works" {
    run visited-tracker version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "visited-tracker CLI v" ]]
    [[ "$output" =~ "API:" ]]
}

@test "CLI version shows correct format" {
    run visited-tracker version
    [ "$status" -eq 0 ]
    [[ "$output" =~ v[0-9]+\.[0-9]+\.[0-9]+ ]]
}

@test "CLI version works even if service is down" {
    run visited-tracker version
    [ "$status" -eq 0 ]
    # Should still show CLI version even if API is not running
    [[ "$output" =~ "visited-tracker CLI" ]]
}

################################################################################
# Error Handling Tests
################################################################################

@test "CLI handles unknown command gracefully" {
    run visited-tracker nonexistent-command
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Error: Unknown command" ]]
    [[ "$output" =~ "Run 'visited-tracker help'" ]]
}

@test "CLI handles empty command gracefully" {
    run visited-tracker ""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Visited Tracker CLI" ]]
}

################################################################################
# Visit Command Tests
################################################################################

@test "CLI visit command requires file arguments" {
    run visited-tracker visit
    [ "$status" -eq 1 ]
    [[ "$output" =~ "At least one file required" ]]
    [[ "$output" =~ "Usage: visited-tracker visit" ]]
}

@test "CLI visit command with service down shows error" {
    if service_running; then
        skip "Service is running - cannot test service down scenario"
    fi
    
    run visited-tracker visit "${TEST_FILE_DIR}/test1.js"
    [ "$status" -ne 0 ]
    [[ "$output" =~ "not running" ]] || [[ "$output" =~ "Error" ]]
}

@test "CLI visit command accepts context option" {
    run visited-tracker visit --context security
    # Should fail due to no file args, but should recognize the context option
    [ "$status" -eq 1 ]
    [[ "$output" =~ "At least one file required" ]]
}

@test "CLI visit command accepts agent option" {
    run visited-tracker visit --agent claude-code
    # Should fail due to no file args, but should recognize the agent option
    [ "$status" -eq 1 ]
    [[ "$output" =~ "At least one file required" ]]
}

@test "CLI visit command accepts conversation option" {
    run visited-tracker visit --conversation test-conv-123
    # Should fail due to no file args, but should recognize the conversation option
    [ "$status" -eq 1 ]
    [[ "$output" =~ "At least one file required" ]]
}

################################################################################
# Sync Command Tests
################################################################################

@test "CLI sync command accepts patterns option" {
    if ! service_running; then
        skip "Service not running"
    fi
    
    run visited-tracker sync --patterns "*.js"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Structure synchronized" ]]
}

@test "CLI sync command accepts remove-deleted option" {
    if ! service_running; then
        skip "Service not running"
    fi
    
    run visited-tracker sync --remove-deleted
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Structure synchronized" ]]
}

@test "CLI sync command accepts structure option" {
    if ! service_running; then
        skip "Service not running"
    fi
    
    run visited-tracker sync --structure "flat"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Structure synchronized" ]]
}

@test "CLI sync command with json output" {
    if ! service_running; then
        skip "Service not running"
    fi
    
    run visited-tracker sync --json
    [ "$status" -eq 0 ]
    is_valid_json "$output"
}

################################################################################
# Least-Visited Command Tests
################################################################################

@test "CLI least-visited works with limit option" {
    if ! service_running; then
        skip "Service not running"
    fi
    
    run visited-tracker least-visited --limit 5
    [ "$status" -eq 0 ]
}

@test "CLI least-visited works with context filter" {
    if ! service_running; then
        skip "Service not running"
    fi
    
    run visited-tracker least-visited --context security --limit 1
    [ "$status" -eq 0 ]
}

@test "CLI least-visited works with patterns filter" {
    if ! service_running; then
        skip "Service not running"
    fi
    
    run visited-tracker least-visited --patterns "*.js" --limit 1
    [ "$status" -eq 0 ]
}

@test "CLI least-visited json output is valid" {
    if ! service_running; then
        skip "Service not running"
    fi
    
    run visited-tracker least-visited --limit 1 --json
    [ "$status" -eq 0 ]
    is_valid_json "$output"
    [[ "$output" =~ "files" ]]
}

@test "CLI least-visited no-unvisited option" {
    if ! service_running; then
        skip "Service not running"
    fi
    
    run visited-tracker least-visited --no-unvisited --limit 1
    [ "$status" -eq 0 ]
}

################################################################################
# Most-Stale Command Tests
################################################################################

@test "CLI most-stale works with limit option" {
    if ! service_running; then
        skip "Service not running"
    fi
    
    run visited-tracker most-stale --limit 3
    [ "$status" -eq 0 ]
}

@test "CLI most-stale works with threshold option" {
    if ! service_running; then
        skip "Service not running"
    fi
    
    run visited-tracker most-stale --threshold 1.0 --limit 5
    [ "$status" -eq 0 ]
}

@test "CLI most-stale json output is valid" {
    if ! service_running; then
        skip "Service not running"
    fi
    
    run visited-tracker most-stale --limit 1 --json
    [ "$status" -eq 0 ]
    is_valid_json "$output"
    [[ "$output" =~ "files" ]]
}

################################################################################
# Coverage Command Tests
################################################################################

@test "CLI coverage command basic functionality" {
    if ! service_running; then
        skip "Service not running"
    fi
    
    run visited-tracker coverage
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Coverage Report" ]]
}

@test "CLI coverage with patterns filter" {
    if ! service_running; then
        skip "Service not running"
    fi
    
    run visited-tracker coverage --patterns "*.js"
    [ "$status" -eq 0 ]
}

@test "CLI coverage with group-by option" {
    if ! service_running; then
        skip "Service not running"
    fi
    
    run visited-tracker coverage --group-by directory
    [ "$status" -eq 0 ]
}

@test "CLI coverage json output is valid" {
    if ! service_running; then
        skip "Service not running"
    fi
    
    run visited-tracker coverage --json
    [ "$status" -eq 0 ]
    is_valid_json "$output"
    [[ "$output" =~ "coverage_percentage" ]]
}

################################################################################
# Status Command Tests
################################################################################

@test "CLI status command basic functionality" {
    if ! service_running; then
        skip "Service not running"
    fi

    run visited-tracker status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Visited Tracker Status" ]]
}

@test "CLI status shows service state" {
    if ! service_running; then
        skip "Service not running"
    fi

    run visited-tracker status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Status: " ]]
}

################################################################################
# Import/Export Command Tests
################################################################################

@test "CLI import command requires file argument" {
    run visited-tracker import
    [ "$status" -ne 0 ]
}

@test "CLI import command with non-existent file" {
    run visited-tracker import /nonexistent/file.json
    [ "$status" -eq 1 ]
    [[ "$output" =~ "File not found" ]]
}

@test "CLI export command requires file argument" {
    run visited-tracker export
    [ "$status" -ne 0 ]
}

@test "CLI export command accepts format option" {
    if ! service_running; then
        skip "Service not running"
    fi
    
    run visited-tracker export "${VISITED_TRACKER_TEST_ROOT}/export.json" --format json
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Export completed" ]]
    [ -f "${VISITED_TRACKER_TEST_ROOT}/export.json" ]
}

@test "CLI export command with include-history option" {
    if ! service_running; then
        skip "Service not running"
    fi
    
    run visited-tracker export "${VISITED_TRACKER_TEST_ROOT}/export.json" --include-history
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Export completed" ]]
    [ -f "${VISITED_TRACKER_TEST_ROOT}/export.json" ]
}

################################################################################
# Integration Tests with Service Running
################################################################################

@test "CLI full workflow: sync, visit, least-visited" {
    if ! service_running; then
        skip "Service not running"
    fi
    
    # Sync some files
    run visited-tracker sync --patterns "*.js"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Structure synchronized" ]]
    
    # Visit a file
    run visited-tracker visit "${TEST_FILE_DIR}/test1.js" --context security
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Visit recorded successfully" ]]
    
    # Check least-visited
    run visited-tracker least-visited --limit 5 --json
    [ "$status" -eq 0 ]
    is_valid_json "$output"
}

@test "CLI integration: coverage after sync" {
    if ! service_running; then
        skip "Service not running"
    fi
    
    # Sync files first
    run visited-tracker sync --patterns "*.js"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Structure synchronized" ]]
    
    # Get coverage
    run visited-tracker coverage --json
    [ "$status" -eq 0 ]
    is_valid_json "$output"
    [[ "$output" =~ "coverage_percentage" ]]
}

################################################################################
# Option Validation Tests
################################################################################

@test "CLI visit command reports error when context provided without files" {
    run visited-tracker visit --context invalid-context
    [ "$status" -eq 1 ]
    [[ "$output" =~ "At least one file required" ]]
}

@test "CLI validates numeric options" {
    run visited-tracker least-visited --limit abc
    [ "$status" -eq 1 ]
    [[ "$output" =~ "must be a positive integer" ]]
}

@test "CLI validates threshold values" {
    run visited-tracker most-stale --threshold not-a-number
    [ "$status" -eq 1 ]
    [[ "$output" =~ "must be numeric" ]]
}

################################################################################
# Alternative Command Forms Tests
################################################################################

@test "CLI accepts --version flag" {
    run visited-tracker --version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "visited-tracker CLI" ]]
}

@test "CLI accepts -v flag" {
    run visited-tracker -v
    [ "$status" -eq 0 ]
    [[ "$output" =~ "visited-tracker CLI" ]]
}

@test "CLI accepts --help flag" {
    run visited-tracker --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Visited Tracker CLI" ]]
}

@test "CLI accepts -h flag" {
    run visited-tracker -h
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Visited Tracker CLI" ]]
}
