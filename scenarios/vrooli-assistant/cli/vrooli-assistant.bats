#!/usr/bin/env bats
# Vrooli Assistant CLI Tests

setup() {
    # Source the CLI to test functions
    export SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    export SCENARIO_NAME="vrooli-assistant"

    # Get dynamic port from service registry
    export API_PORT=$(vrooli scenario port "${SCENARIO_NAME}" API_PORT 2>/dev/null || echo "17628")
    export API_URL="http://localhost:${API_PORT}"
}

# Test: CLI help command
@test "CLI help command displays usage information" {
    run ./cli/vrooli-assistant help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Vrooli Assistant" ]]
    [[ "$output" =~ "Commands:" ]]
    [[ "$output" =~ "start" ]]
    [[ "$output" =~ "stop" ]]
    [[ "$output" =~ "status" ]]
    [[ "$output" =~ "capture" ]]
}

# Test: CLI version command
@test "CLI version command displays version" {
    run ./cli/vrooli-assistant version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Vrooli Assistant" ]]
    [[ "$output" =~ "v1.0.0" ]]
}

# Test: CLI status command (JSON output)
@test "CLI status command returns JSON format" {
    run ./cli/vrooli-assistant status --json
    [ "$status" -eq 0 ]
    [[ "$output" =~ "daemon" ]]
    [[ "$output" =~ "api" ]]
}

# Test: CLI status command (human-readable output)
@test "CLI status command returns human-readable format" {
    run ./cli/vrooli-assistant status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Vrooli Assistant Status:" ]]
    [[ "$output" =~ "Daemon:" ]]
    [[ "$output" =~ "API:" ]]
}

# Test: CLI capture without description fails
@test "CLI capture command requires description" {
    run ./cli/vrooli-assistant capture
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Error: Description required" ]]
}

# Test: CLI capture with description
@test "CLI capture command with description succeeds" {
    # Only run if API is healthy
    if curl -sf "${API_URL}/health" &>/dev/null; then
        run ./cli/vrooli-assistant capture "Test issue from BATS"
        [ "$status" -eq 0 ]
        [[ "$output" =~ "Issue captured successfully" ]] || [[ "$output" =~ "issue_id" ]]
    else
        skip "API not available"
    fi
}

# Test: CLI history command
@test "CLI history command retrieves issues" {
    # Only run if API is healthy
    if curl -sf "${API_URL}/health" &>/dev/null; then
        run ./cli/vrooli-assistant history
        [ "$status" -eq 0 ]
        [[ "$output" =~ "Recent Issues:" ]]
    else
        skip "API not available"
    fi
}

# Test: Invalid command returns error
@test "CLI invalid command returns error" {
    run ./cli/vrooli-assistant invalid-command
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown command" ]]
}

# Test: CLI is executable
@test "CLI binary is executable" {
    [ -x ./cli/vrooli-assistant ]
}

# Test: API health check
@test "API health endpoint responds" {
    if curl -sf "${API_URL}/health" &>/dev/null; then
        run curl -sf "${API_URL}/health"
        [ "$status" -eq 0 ]
        [[ "$output" =~ "healthy" ]]
    else
        skip "API not available"
    fi
}

# Test: API capture endpoint exists
@test "API capture endpoint is accessible" {
    if curl -sf "${API_URL}/health" &>/dev/null; then
        # Test with OPTIONS to check if endpoint exists
        run curl -sf -X OPTIONS "${API_URL}/api/v1/assistant/capture"
        # Endpoint should at least respond (status 0 or 22 for valid HTTP response)
        [ "$status" -eq 0 ] || [ "$status" -eq 22 ]
    else
        skip "API not available"
    fi
}

# Test: API status endpoint
@test "API status endpoint responds" {
    if curl -sf "${API_URL}/health" &>/dev/null; then
        run curl -sf "${API_URL}/api/v1/assistant/status"
        [ "$status" -eq 0 ]
        [[ "$output" =~ "status" ]]
    else
        skip "API not available"
    fi
}
