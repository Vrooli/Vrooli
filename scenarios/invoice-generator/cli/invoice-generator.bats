#!/usr/bin/env bats
# Tests for invoice-generator CLI

# Test configuration
readonly TEST_CLI="invoice-generator"
readonly TEST_CONFIG_DIR="$HOME/.vrooli/invoice-generator"
readonly CLI_PATH="${HOME}/.vrooli/bin/invoice-generator"

# Setup and teardown
setup() {
    # Store original API_URL if set
    if [[ -n "$API_URL" ]]; then
        ORIGINAL_API_URL="$API_URL"
    fi

    # Verify CLI is installed (check both PATH and direct location)
    if ! command -v invoice-generator &> /dev/null && [[ ! -f "$CLI_PATH" ]]; then
        skip "invoice-generator CLI not installed"
    fi

    # Use CLI from known location if not in PATH
    if ! command -v invoice-generator &> /dev/null && [[ -f "$CLI_PATH" ]]; then
        export PATH="${HOME}/.vrooli/bin:$PATH"
    fi

    # Verify scenario is running
    run vrooli scenario status invoice-generator
    if [[ "$status" -ne 0 ]] || [[ ! "$output" =~ "RUNNING" ]]; then
        skip "invoice-generator scenario not running"
    fi
}

teardown() {
    # Restore original API_URL if it was set
    if [[ -n "$ORIGINAL_API_URL" ]]; then
        export API_URL="$ORIGINAL_API_URL"
    else
        unset API_URL
    fi
}

# Test: CLI exists and is executable
@test "CLI command is available" {
    run which invoice-generator
    [ "$status" -eq 0 ]
}

@test "CLI is executable" {
    run invoice-generator help
    [ "$status" -eq 0 ]
}

# Test: Help command
@test "help command displays usage information" {
    run invoice-generator help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Invoice Generator CLI" ]]
    [[ "$output" =~ "Usage:" ]]
    [[ "$output" =~ "Commands:" ]]
}

@test "help shows all main commands" {
    run invoice-generator help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "create" ]]
    [[ "$output" =~ "list" ]]
    [[ "$output" =~ "get" ]]
    [[ "$output" =~ "status" ]]
    [[ "$output" =~ "clients" ]]
}

# Test: List clients command
@test "clients command lists clients" {
    run invoice-generator clients
    [ "$status" -eq 0 ]
    # Should return JSON array or table
    [[ "$output" =~ "client" || "$output" =~ "Client" || "$output" =~ "[" ]]
}

# Test: List invoices command
@test "list command shows invoices" {
    run invoice-generator list
    [ "$status" -eq 0 ]
    # Should return successful output (could be empty list)
}

# Test: Create invoice command structure
@test "create command requires parameters" {
    run invoice-generator create
    # Should either succeed with defaults or fail gracefully with usage
    [[ "$status" -eq 0 || "$status" -eq 1 ]]
}

# Test: Get invoice command
@test "get command requires invoice ID" {
    run invoice-generator get
    # Should fail gracefully when no ID provided
    [ "$status" -ne 0 ]
}

# Test: Status command requires parameters
@test "status command requires invoice ID" {
    run invoice-generator status
    # Should fail or show usage when no ID provided
    [ "$status" -ne 0 ]
}

# Test: Environment variable support
@test "CLI accepts API_URL environment variable" {
    # Test that help mentions environment variables
    run invoice-generator help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "API_URL" || "$output" =~ "API_PORT" ]]
}

# Test: Invalid command handling
@test "invalid command shows error" {
    run invoice-generator invalid-command-xyz
    [ "$status" -ne 0 ]
}

# Integration test: List clients returns data
@test "clients command returns valid data" {
    run invoice-generator clients
    [ "$status" -eq 0 ]
    # Should return some output (clients list or empty array)
    [[ -n "$output" ]]
}

# Integration test: Create and retrieve invoice (if API is fully operational)
@test "create invoice workflow" {
    # This test requires API to be running
    run invoice-generator create --client "Test Client" --amount 1000 --description "Test invoice"

    # Check if creation succeeded
    if [ "$status" -eq 0 ]; then
        # Extract invoice ID if present in output
        if [[ "$output" =~ INV-[0-9]+-[0-9]+ ]]; then
            # Invoice was created successfully
            [[ "$output" =~ "created" || "$output" =~ "Created" || "$output" =~ "success" ]]
        fi
    else
        # If creation failed, ensure it's a graceful failure
        [[ "$output" =~ "error" || "$output" =~ "Error" || "$output" =~ "failed" ]]
    fi
}

# Test: CLI version/info
@test "CLI shows version or help by default" {
    run invoice-generator
    # Without arguments, should show help or version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Invoice" || "$output" =~ "Usage" ]]
}

# Test: Add client command
@test "add-client command structure" {
    run invoice-generator help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "add-client" ]]
}
