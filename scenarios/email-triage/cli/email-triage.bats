#!/usr/bin/env bats
# Email Triage CLI Test Suite
# Tests all CLI commands and functionality

setup() {
    # Load the CLI script path
    CLI_SCRIPT="${BATS_TEST_DIRNAME}/email-triage"

    # Ensure the CLI is executable
    chmod +x "$CLI_SCRIPT"

    # Detect the API port from scenario status
    if command -v vrooli &> /dev/null; then
        export API_PORT=$(vrooli scenario status email-triage 2>/dev/null | grep "API_PORT:" | awk '{print $2}')
    fi

    # Fallback to default port if detection fails
    export API_PORT="${API_PORT:-19525}"
    export EMAIL_TRIAGE_API_URL="http://localhost:${API_PORT}"
}

@test "CLI executable exists and is executable" {
    [ -x "$CLI_SCRIPT" ]
}

@test "CLI version command works" {
    run "$CLI_SCRIPT" --version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Email Triage CLI version" ]]
}

@test "CLI version command with --json flag" {
    run "$CLI_SCRIPT" --version --json
    [ "$status" -eq 0 ]
    [[ "$output" =~ "\"version\"" ]]
    [[ "$output" =~ "\"api_url\"" ]]
}

@test "CLI help command works" {
    run "$CLI_SCRIPT" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Email Triage CLI" ]]
    [[ "$output" =~ "COMMANDS:" ]]
    [[ "$output" =~ "account" ]]
    [[ "$output" =~ "rule" ]]
    [[ "$output" =~ "search" ]]
}

@test "CLI status command works" {
    run "$CLI_SCRIPT" status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Email Triage API is" ]]
}

@test "CLI status command with --json flag" {
    run "$CLI_SCRIPT" status --json
    [ "$status" -eq 0 ]
    [[ "$output" =~ "\"status\"" ]]
}

@test "CLI detects correct API port automatically" {
    run "$CLI_SCRIPT" --version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "http://localhost:${API_PORT}" ]]
}

@test "CLI account list command works" {
    run "$CLI_SCRIPT" account list
    [ "$status" -eq 0 ]
    # Should not crash, even if no accounts exist
}

@test "CLI rule list command works" {
    run "$CLI_SCRIPT" rule list
    [ "$status" -eq 0 ]
    # Should not crash, even if no rules exist
}

@test "CLI search command requires query" {
    run "$CLI_SCRIPT" search
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Usage" ]] || [[ "$output" =~ "query" ]]
}

@test "CLI handles invalid command gracefully" {
    run "$CLI_SCRIPT" nonexistent-command
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Unknown command" ]] || [[ "$output" =~ "help" ]]
}

@test "CLI config directory is created" {
    # Run a command to trigger config initialization
    run "$CLI_SCRIPT" status
    [ "$status" -eq 0 ]
    [ -d "${HOME}/.vrooli/email-triage" ]
}

@test "CLI config file is created with correct structure" {
    # Run a command to trigger config initialization
    run "$CLI_SCRIPT" status
    [ "$status" -eq 0 ]
    [ -f "${HOME}/.vrooli/email-triage/config.yaml" ]

    # Check config file contains expected keys
    grep -q "api_url:" "${HOME}/.vrooli/email-triage/config.yaml"
}

@test "CLI respects EMAIL_TRIAGE_API_URL environment variable" {
    export EMAIL_TRIAGE_API_URL="http://custom-host:9999"
    run "$CLI_SCRIPT" --version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "http://custom-host:9999" ]]
}

@test "CLI account add command requires email and password" {
    run "$CLI_SCRIPT" account add
    [ "$status" -ne 0 ]
}

@test "CLI rule create command requires description" {
    run "$CLI_SCRIPT" rule create
    [ "$status" -ne 0 ]
}

@test "CLI sync command works" {
    run "$CLI_SCRIPT" sync
    [ "$status" -eq 0 ]
    # May show warning if no accounts, but should not crash
}

@test "CLI handles network errors gracefully when API is unreachable" {
    export EMAIL_TRIAGE_API_URL="http://localhost:99999"
    run "$CLI_SCRIPT" status
    [ "$status" -ne 0 ]
    # Should show an error message, not crash
}
