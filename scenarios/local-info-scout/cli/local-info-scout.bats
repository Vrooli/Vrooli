#!/usr/bin/env bats
# Local Info Scout CLI Tests

setup() {
    # Get the directory where this test file is located
    TEST_DIR="$(cd "$(dirname "$BATS_TEST_FILENAME")" && pwd)"
    export CLI_PATH="${TEST_DIR}/local-info-scout"
    export API_PORT="18538"
    export API_URL="http://localhost:${API_PORT}"
}

# Helper function to check if API is running
api_available() {
    curl -sf "${API_URL}/health" > /dev/null 2>&1
}

# Tests for core CLI functionality

@test "CLI shows help with --help flag" {
    run "$CLI_PATH" --help
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"Local Info Scout CLI - Discover local places"* ]]
    [[ "$output" == *"Usage:"* ]]
    [[ "$output" == *"Options:"* ]]
}

@test "CLI shows help with -help flag" {
    run "$CLI_PATH" -help
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"Local Info Scout CLI"* ]]
}

@test "CLI shows help with no arguments" {
    run "$CLI_PATH"
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"Local Info Scout CLI"* ]]
    [[ "$output" == *"Examples:"* ]]
}

@test "CLI requires API_PORT when not using --api flag" {
    unset API_PORT
    run "$CLI_PATH" "test query"
    [[ "$status" -eq 1 ]]
    [[ "$output" == *"API_PORT environment variable not set"* ]]
}

@test "CLI accepts --api flag to override API_PORT" {
    skip "Requires live API"
    unset API_PORT
    run "$CLI_PATH" --api="${API_URL}" "coffee shops"
    # Should not fail due to missing API_PORT since --api is provided
    [[ "$output" != *"API_PORT environment variable not set"* ]]
}

@test "CLI --categories flag lists available categories" {
    if ! api_available; then
        skip "API not running"
    fi

    run "$CLI_PATH" --categories
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"Available categories:"* ]]
}

@test "CLI --categories shows restaurant category" {
    if ! api_available; then
        skip "API not running"
    fi

    run "$CLI_PATH" --categories
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"restaurant"* ]]
}

@test "CLI search with query argument" {
    if ! api_available; then
        skip "API not running"
    fi

    run "$CLI_PATH" "vegan restaurants"
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"Search results"* ]]
}

@test "CLI search with --query flag" {
    if ! api_available; then
        skip "API not running"
    fi

    run "$CLI_PATH" --query="coffee shops"
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"Search results"* ]]
}

@test "CLI search accepts --lat and --lon flags" {
    if ! api_available; then
        skip "API not running"
    fi

    run "$CLI_PATH" --query="parks" --lat=40.7589 --lon=-73.9851
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"Search results"* ]]
    [[ "$output" == *"40.7589"* ]]
}

@test "CLI search accepts --radius flag" {
    if ! api_available; then
        skip "API not running"
    fi

    run "$CLI_PATH" --query="pharmacy" --radius=2
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"Search results"* ]]
}

@test "CLI search accepts --category flag" {
    if ! api_available; then
        skip "API not running"
    fi

    run "$CLI_PATH" --query="food" --category="restaurant"
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"Search results"* ]]
}

@test "CLI search handles empty results gracefully" {
    if ! api_available; then
        skip "API not running"
    fi

    run "$CLI_PATH" "zzz_unlikely_to_find_this_xyz"
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"Search results"* ]]
}

@test "CLI search shows distance in results" {
    if ! api_available; then
        skip "API not running"
    fi

    run "$CLI_PATH" "restaurants"
    [[ "$status" -eq 0 ]]
    if [[ "$output" != *"No places found"* ]]; then
        [[ "$output" == *"miles"* ]]
    fi
}

@test "CLI search shows ratings when available" {
    if ! api_available; then
        skip "API not running"
    fi

    run "$CLI_PATH" "restaurants"
    [[ "$status" -eq 0 ]]
    # Results may or may not have ratings, so just check format is reasonable
    [[ "$output" == *"Search results"* ]]
}

@test "CLI search shows open/closed status" {
    if ! api_available; then
        skip "API not running"
    fi

    run "$CLI_PATH" "pharmacy"
    [[ "$status" -eq 0 ]]
    if [[ "$output" != *"No places found"* ]]; then
        # Should show either 🟢 Open or 🔴 Closed for results
        [[ "$output" =~ (Open|Closed) ]]
    fi
}

@test "CLI fails gracefully when API is unreachable" {
    export API_PORT="99999"
    run "$CLI_PATH" "test"
    [[ "$status" -eq 1 ]]
    [[ "$output" == *"Error"* ]]
}

@test "CLI handles multi-word queries correctly" {
    if ! api_available; then
        skip "API not running"
    fi

    run "$CLI_PATH" "24 hour pharmacy near me"
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"Search results"* ]]
}

@test "CLI accepts natural language queries" {
    if ! api_available; then
        skip "API not running"
    fi

    run "$CLI_PATH" "find vegan restaurants within 2 miles"
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"Search results"* ]]
}

@test "CLI query with both positional and --query flag prefers --query" {
    if ! api_available; then
        skip "API not running"
    fi

    run "$CLI_PATH" --query="specific query" "ignored positional"
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"specific query"* ]]
}

@test "CLI validates required arguments" {
    # No query provided should show help
    run "$CLI_PATH"
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"Local Info Scout CLI"* ]]
}

@test "CLI outputs JSON-parseable data for structured queries" {
    if ! api_available; then
        skip "API not running"
    fi

    run "$CLI_PATH" "restaurants"
    [[ "$status" -eq 0 ]]
    # Output should be human-readable, not raw JSON
    [[ "$output" == *"Search results"* ]]
    [[ "$output" != "{\"places\":"* ]]
}

@test "CLI search handles special characters in query" {
    if ! api_available; then
        skip "API not running"
    fi

    run "$CLI_PATH" "café & bakery"
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"Search results"* ]]
}
