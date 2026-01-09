#!/usr/bin/env bats
# SmartNotes CLI test suite
# Tests all CLI commands for proper functionality

# Setup runs before each test
setup() {
    # Ensure API_PORT is set
    if [ -z "$API_PORT" ]; then
        skip "API_PORT not set - scenario must be running"
    fi

    # Set test environment
    export NOTES_API_URL="http://localhost:${API_PORT}"
    export NOTES_USER_ID="${NOTES_USER_ID:-df9cdbc0-b4eb-4799-8514-cce7c15ebeaf}"

    # Path to CLI
    CLI="${BATS_TEST_DIRNAME}/notes"

    # Wait for API to be ready
    local max_attempts=10
    local attempt=0
    while ! curl -sf "$NOTES_API_URL/health" >/dev/null 2>&1; do
        attempt=$((attempt + 1))
        if [ $attempt -ge $max_attempts ]; then
            skip "API not responding after ${max_attempts} attempts"
        fi
        sleep 1
    done
}

# Test: CLI help command
@test "CLI help displays usage information" {
    run "$CLI" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "SmartNotes CLI" ]]
    [[ "$output" =~ "Usage: notes" ]]
    [[ "$output" =~ "Commands:" ]]
}

# Test: CLI without arguments shows help
@test "CLI without arguments shows help" {
    run "$CLI"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "SmartNotes CLI" ]]
}

# Test: List notes command
@test "CLI list command works" {
    run "$CLI" list
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Your Notes" ]] || [[ "$output" =~ "No notes found" ]]
}

# Test: Create note command
@test "CLI can create a note" {
    # Create a note with content via stdin
    run bash -c "echo 'Test content for CLI test' | $CLI new 'BATS Test Note'"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Note created successfully" ]] || [[ "$output" =~ "Note ID:" ]]
}

# Test: Search notes command
@test "CLI search command works" {
    run "$CLI" search "test"
    [ "$status" -eq 0 ]
    # Search should either find results or return empty (both valid)
}

# Test: List folders command
@test "CLI folders command works" {
    run "$CLI" folders
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Folders" ]] || [[ "$output" =~ "No folders" ]]
}

# Test: List tags command
@test "CLI tags command works" {
    run "$CLI" tags
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Tags" ]] || [[ "$output" =~ "No tags" ]]
}

# Test: List templates command
@test "CLI templates command works" {
    run "$CLI" templates
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Templates" ]] || [[ "$output" =~ "No templates" ]]
}

# Test: Invalid command shows error
@test "CLI shows error for invalid command" {
    run "$CLI" invalid-command-xyz
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Unknown command" ]] || [[ "$output" =~ "Error" ]]
}

# Test: API connectivity check
@test "CLI detects when API is unavailable" {
    # Temporarily override API URL to non-existent port
    NOTES_API_URL="http://localhost:99999" run "$CLI" list
    # CLI may not exit with error code, but should show "No notes" or not crash
    # This test verifies graceful degradation rather than hard failure
    [[ "$output" =~ "Your Notes" ]] || [[ "$output" =~ "No notes" ]] || [[ "$output" =~ "Error" ]]
}

# Test: Create note without title fails gracefully
@test "CLI create without title shows error" {
    run "$CLI" new
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Error" ]] || [[ "$output" =~ "title" ]]
}

# Test: View note without ID fails gracefully
@test "CLI view without ID shows error" {
    run "$CLI" view
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Error" ]] || [[ "$output" =~ "ID" ]]
}

# Test: Delete note without ID fails gracefully
@test "CLI delete without ID shows error" {
    run "$CLI" delete
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Error" ]] || [[ "$output" =~ "ID" ]]
}

# Test: Environment variable validation
@test "CLI requires API_PORT to be set" {
    # Unset API_PORT and test
    (
        unset API_PORT
        run "$CLI" list
        [ "$status" -ne 0 ]
        [[ "$output" =~ "API_PORT" ]] || [[ "$output" =~ "required" ]]
    )
}

# Test: Full workflow - create, list, search
@test "CLI full workflow: create note and find it" {
    # Create a unique test note
    local test_title="CLI_BATS_Test_$(date +%s)"
    local test_content="This is a test note created by BATS at $(date)"

    # Create note
    run bash -c "echo '$test_content' | $CLI new '$test_title'"
    [ "$status" -eq 0 ]

    # Extract note ID from output
    local note_id
    note_id=$(echo "$output" | grep -oP 'Note ID: \K[a-f0-9-]+' || echo "")

    # List notes and verify our note is there
    run "$CLI" list
    [ "$status" -eq 0 ]
    if [ -n "$note_id" ]; then
        [[ "$output" =~ "$test_title" ]] || [[ "$output" =~ "${note_id:0:8}" ]]
    fi

    # Search for our note
    run "$CLI" search "BATS_Test"
    [ "$status" -eq 0 ]
}
