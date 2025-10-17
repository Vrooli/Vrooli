#!/usr/bin/env bats
# Maintenance Orchestrator CLI Tests

CLI_SCRIPT="${BATS_TEST_DIRNAME}/maintenance-orchestrator"
SCENARIO_NAME="maintenance-orchestrator"

setup() {
    # Check if scenario is running before attempting tests
    if ! vrooli scenario port "${SCENARIO_NAME}" API_PORT > /dev/null 2>&1; then
        skip "maintenance-orchestrator is not running (use: vrooli scenario run ${SCENARIO_NAME})"
    fi

    # Get dynamic API port
    API_PORT=$(vrooli scenario port "${SCENARIO_NAME}" API_PORT 2>/dev/null)
    API_URL="http://localhost:${API_PORT}/api/v1"

    # Verify API is actually responsive
    if ! curl -sf "${API_URL}/status" > /dev/null 2>&1; then
        skip "API not responding at ${API_URL}"
    fi
}

################################################################################
# Help and Version Tests
################################################################################

@test "CLI shows help when no arguments provided" {
    run bash "$CLI_SCRIPT"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Maintenance Orchestrator CLI" ]]
    [[ "$output" =~ "Usage:" ]]
    [[ "$output" =~ "Commands:" ]]
}

@test "CLI shows help with help command" {
    run bash "$CLI_SCRIPT" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Maintenance Orchestrator CLI" ]]
    [[ "$output" =~ "Usage:" ]]
}

@test "CLI shows version information" {
    run bash "$CLI_SCRIPT" version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "maintenance-orchestrator version" ]]
    [[ "$output" =~ "API endpoint:" ]]
}

################################################################################
# Status Command Tests
################################################################################

@test "CLI status command works" {
    run bash "$CLI_SCRIPT" status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Maintenance Orchestrator Status" ]]
    [[ "$output" =~ "Health:" ]]
    [[ "$output" =~ "Total:" ]]
    [[ "$output" =~ "Active:" ]]
    [[ "$output" =~ "Inactive:" ]]
}

@test "CLI status command supports --json flag" {
    run bash "$CLI_SCRIPT" status --json
    [ "$status" -eq 0 ]
    # Verify valid JSON output
    echo "$output" | jq . > /dev/null
    # Check for expected JSON fields
    [[ "$output" =~ "\"health\"" ]]
    [[ "$output" =~ "\"totalScenarios\"" ]]
    [[ "$output" =~ "\"activeScenarios\"" ]]
}

@test "CLI status command supports --verbose flag" {
    run bash "$CLI_SCRIPT" status --verbose
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Recent Activity" ]]
}

################################################################################
# List Command Tests
################################################################################

@test "CLI list command shows discovered scenarios" {
    run bash "$CLI_SCRIPT" list
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Discovered Maintenance Scenarios" ]]
    # Should show some scenarios (the system has maintenance scenarios)
}

@test "CLI list command supports --json flag" {
    run bash "$CLI_SCRIPT" list --json
    [ "$status" -eq 0 ]
    # Verify valid JSON output
    echo "$output" | jq . > /dev/null
    [[ "$output" =~ "\"scenarios\"" ]]
}

@test "CLI list command supports --active-only flag" {
    run bash "$CLI_SCRIPT" list --active-only
    [ "$status" -eq 0 ]
    # Output should contain scenario list structure
    [[ "$output" =~ "Discovered Maintenance Scenarios" ]]
}

################################################################################
# Activate Command Tests
################################################################################

@test "CLI activate command requires scenario ID" {
    run bash "$CLI_SCRIPT" activate
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Scenario ID required" ]]
    [[ "$output" =~ "Usage:" ]]
}

@test "CLI activate command can activate a scenario" {
    # First get a valid scenario ID from the list
    local scenario_id=$(bash "$CLI_SCRIPT" list --json | jq -r '.scenarios[0].id' 2>/dev/null)

    if [ -z "$scenario_id" ] || [ "$scenario_id" = "null" ]; then
        skip "No maintenance scenarios available for testing"
    fi

    # Activate the scenario
    run bash "$CLI_SCRIPT" activate "$scenario_id"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Activated" ]]

    # Clean up - deactivate it
    bash "$CLI_SCRIPT" deactivate "$scenario_id" > /dev/null 2>&1 || true
}

@test "CLI activate command handles invalid scenario ID" {
    run bash "$CLI_SCRIPT" activate "nonexistent-scenario-id-12345"
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Failed to activate" ]]
}

################################################################################
# Deactivate Command Tests
################################################################################

@test "CLI deactivate command requires scenario ID" {
    run bash "$CLI_SCRIPT" deactivate
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Scenario ID required" ]]
    [[ "$output" =~ "Usage:" ]]
}

@test "CLI deactivate command can deactivate a scenario" {
    # First activate a scenario, then deactivate it
    local scenario_id=$(bash "$CLI_SCRIPT" list --json | jq -r '.scenarios[0].id' 2>/dev/null)

    if [ -z "$scenario_id" ] || [ "$scenario_id" = "null" ]; then
        skip "No maintenance scenarios available for testing"
    fi

    # Activate first
    bash "$CLI_SCRIPT" activate "$scenario_id" > /dev/null 2>&1 || skip "Could not activate scenario for test"

    # Now test deactivation
    run bash "$CLI_SCRIPT" deactivate "$scenario_id"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Deactivated" ]]
}

@test "CLI deactivate command handles invalid scenario ID" {
    run bash "$CLI_SCRIPT" deactivate "nonexistent-scenario-id-12345"
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Failed to deactivate" ]]
}

################################################################################
# Preset Command Tests
################################################################################

@test "CLI preset command requires subcommand" {
    run bash "$CLI_SCRIPT" preset
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Preset subcommand required" ]]
    [[ "$output" =~ "Usage:" ]]
}

@test "CLI preset list shows available presets" {
    run bash "$CLI_SCRIPT" preset list
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Available Presets" ]]
    # Should show at least one preset (system has default presets)
}

@test "CLI preset apply requires preset ID" {
    # This test needs input redirection, so we test the error case
    run bash "$CLI_SCRIPT" preset apply
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Preset ID required" ]]
}

@test "CLI preset create requires preset name" {
    run bash "$CLI_SCRIPT" preset create
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Preset name required" ]]
}

################################################################################
# Error Handling Tests
################################################################################

@test "CLI handles unknown command gracefully" {
    run bash "$CLI_SCRIPT" unknown-command-xyz
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Unknown command" ]]
}

@test "CLI handles invalid options gracefully" {
    run bash "$CLI_SCRIPT" status --invalid-option
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Unknown option" ]]
}

################################################################################
# Integration Tests
################################################################################

@test "CLI can perform full scenario lifecycle (activate -> deactivate)" {
    # Get first available scenario
    local scenario_id=$(bash "$CLI_SCRIPT" list --json | jq -r '.scenarios[0].id' 2>/dev/null)

    if [ -z "$scenario_id" ] || [ "$scenario_id" = "null" ]; then
        skip "No maintenance scenarios available for testing"
    fi

    # Ensure scenario starts inactive
    bash "$CLI_SCRIPT" deactivate "$scenario_id" > /dev/null 2>&1 || true

    # Activate
    run bash "$CLI_SCRIPT" activate "$scenario_id"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Activated" ]]

    # Verify it's active via status
    run bash "$CLI_SCRIPT" status --json
    [ "$status" -eq 0 ]
    local active_count=$(echo "$output" | jq -r '.activeScenarios')
    [ "$active_count" -gt 0 ]

    # Deactivate
    run bash "$CLI_SCRIPT" deactivate "$scenario_id"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Deactivated" ]]

    # Verify it's inactive
    run bash "$CLI_SCRIPT" status --json
    [ "$status" -eq 0 ]
    local final_active=$(echo "$output" | jq -r '.activeScenarios')
    [ "$final_active" -lt "$active_count" ]
}

@test "CLI status reflects changes immediately" {
    local scenario_id=$(bash "$CLI_SCRIPT" list --json | jq -r '.scenarios[0].id' 2>/dev/null)

    if [ -z "$scenario_id" ] || [ "$scenario_id" = "null" ]; then
        skip "No maintenance scenarios available for testing"
    fi

    # Get initial count
    local initial_active=$(bash "$CLI_SCRIPT" status --json | jq -r '.activeScenarios')

    # Activate scenario
    bash "$CLI_SCRIPT" activate "$scenario_id" > /dev/null 2>&1 || skip "Could not activate scenario"

    # Check count increased
    local new_active=$(bash "$CLI_SCRIPT" status --json | jq -r '.activeScenarios')
    [ "$new_active" -gt "$initial_active" ]

    # Clean up
    bash "$CLI_SCRIPT" deactivate "$scenario_id" > /dev/null 2>&1 || true
}

################################################################################
# Performance Tests
################################################################################

@test "CLI commands respond within reasonable time" {
    local start=$(date +%s%N)
    bash "$CLI_SCRIPT" status > /dev/null 2>&1
    local end=$(date +%s%N)
    local duration=$(( (end - start) / 1000000 )) # Convert to milliseconds

    # Should respond in less than 2 seconds (2000ms)
    [ "$duration" -lt 2000 ]
}

@test "CLI list command handles multiple scenarios efficiently" {
    local start=$(date +%s%N)
    bash "$CLI_SCRIPT" list > /dev/null 2>&1
    local end=$(date +%s%N)
    local duration=$(( (end - start) / 1000000 ))

    # Should respond in less than 2 seconds
    [ "$duration" -lt 2000 ]
}
