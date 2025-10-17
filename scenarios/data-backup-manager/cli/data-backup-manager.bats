#!/usr/bin/env bats
# Tests for data-backup-manager CLI

setup() {
    # Get API port from running scenario
    export API_PORT=$(vrooli scenario status data-backup-manager --json 2>/dev/null | jq -r '.scenario_data.allocated_ports.API_PORT // .raw_response.data.allocated_ports.API_PORT // "20010"')

    # Path to CLI binary
    CLI="${BATS_TEST_DIRNAME}/data-backup-manager"

    # Check if scenario is running
    if ! curl -sf "http://localhost:${API_PORT}/health" &>/dev/null; then
        skip "data-backup-manager scenario not running"
    fi
}

@test "CLI help command shows usage" {
    run "$CLI" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Data Backup Manager CLI" ]]
}

@test "CLI version command works" {
    run "$CLI" version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "1.0" ]]
}

@test "CLI status command shows system status" {
    run "$CLI" status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "System Status" ]]
}

@test "CLI status --json returns valid JSON" {
    run "$CLI" status --json
    [ "$status" -eq 0 ]

    # Verify JSON is valid
    echo "$output" | jq . >/dev/null
    [ $? -eq 0 ]

    # Check for required fields
    echo "$output" | jq -e '.system_status' >/dev/null
    [ $? -eq 0 ]
}

@test "CLI backup command requires targets" {
    run "$CLI" backup
    [ "$status" -ne 0 ]
    [[ "$output" =~ "target" || "$output" =~ "Usage" ]]
}

@test "CLI backup with invalid type fails" {
    run "$CLI" backup test-data --type invalid_type
    [ "$status" -ne 0 ]
}

@test "CLI list command shows backups" {
    run "$CLI" list
    [ "$status" -eq 0 ]
}

@test "CLI list --json returns valid JSON" {
    run "$CLI" list --json
    [ "$status" -eq 0 ]

    # Verify JSON is valid
    echo "$output" | jq . >/dev/null
    [ $? -eq 0 ]
}

@test "CLI schedule list shows schedules" {
    run "$CLI" schedule list
    [ "$status" -eq 0 ]
}

@test "CLI with invalid command shows error" {
    run "$CLI" invalid_command
    [ "$status" -ne 0 ]
}

@test "CLI status checks API availability" {
    # Temporarily break API connection by using wrong port
    export API_PORT=99999
    run "$CLI" status
    [ "$status" -ne 0 ]
    [[ "$output" =~ "API not available" || "$output" =~ "not available" ]]
}

@test "CLI status --verbose shows detailed output" {
    run "$CLI" status --verbose
    [ "$status" -eq 0 ]
    # Verbose should show more information
    [ ${#output} -gt 100 ]
}

@test "CLI help shows all available commands" {
    run "$CLI" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "status" ]]
    [[ "$output" =~ "backup" ]]
    [[ "$output" =~ "restore" ]]
    [[ "$output" =~ "list" ]]
    [[ "$output" =~ "schedule" ]]
}

@test "CLI status shows resource health" {
    run "$CLI" status
    [ "$status" -eq 0 ]
    # Should mention at least one resource
    [[ "$output" =~ "postgres" || "$output" =~ "PostgreSQL" || "$output" =~ "Storage" ]]
}
