#!/usr/bin/env bats
################################################################################
# Time Tools CLI Test Suite
# Tests the time-tools CLI wrapper functionality
################################################################################

# Test setup
setup() {
    # Ensure time-tools CLI is available
    if ! command -v time-tools >/dev/null 2>&1; then
        skip "time-tools CLI not installed. Run: cd cli && ./install.sh"
    fi
    
    # Check if time-tools service is running
    if ! vrooli scenario port time-tools TIME_TOOLS_PORT >/dev/null 2>&1; then
        skip "time-tools service not running. Start with: vrooli scenario run time-tools"
    fi
}

################################################################################
# Basic CLI Tests
################################################################################

@test "CLI shows help message" {
    run time-tools help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Time Tools CLI" ]]
    [[ "$output" =~ "convert" ]]
    [[ "$output" =~ "duration" ]]
}

@test "CLI shows version information" {
    run time-tools version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "time-tools CLI" ]]
    [[ "$output" =~ "API:" ]]
}

@test "CLI shows status" {
    run time-tools status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Time Tools API Status" ]]
}

################################################################################
# Timezone Conversion Tests
################################################################################

@test "CLI converts timezone from UTC to Eastern" {
    run time-tools convert '2024-01-15T10:00:00Z' UTC 'America/New_York'
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Timezone Conversion" ]]
    [[ "$output" =~ "Original:" ]]
    [[ "$output" =~ "Converted:" ]]
    [[ "$output" =~ "America/New_York" ]]
}

@test "CLI converts timezone from Eastern to Pacific" {
    run time-tools convert '2024-01-15T15:00:00' 'America/New_York' 'America/Los_Angeles'
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Timezone Conversion" ]]
}

@test "CLI handles invalid timezone gracefully" {
    run time-tools convert '2024-01-15T10:00:00Z' 'Invalid/Timezone' UTC
    [ "$status" -ne 0 ]
}

@test "CLI handles missing timezone conversion arguments" {
    run time-tools convert
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Usage:" ]]
}

################################################################################
# Duration Calculation Tests
################################################################################

@test "CLI calculates basic duration" {
    run time-tools duration '2024-01-15T09:00:00Z' '2024-01-15T17:00:00Z'
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Duration Calculation" ]]
    [[ "$output" =~ "Total Hours:" ]]
    [[ "$output" =~ "8.00" ]]
}

@test "CLI calculates duration with business hours only" {
    run time-tools duration '2024-01-15T09:00:00Z' '2024-01-17T17:00:00Z' --business-hours-only
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Duration Calculation" ]]
    [[ "$output" =~ "Business Hours:" ]]
}

@test "CLI calculates duration excluding weekends" {
    run time-tools duration '2024-01-15T09:00:00Z' '2024-01-22T17:00:00Z' --exclude-weekends
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Duration Calculation" ]]
    [[ "$output" =~ "Business Days:" ]]
}

@test "CLI handles missing duration arguments" {
    run time-tools duration
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Usage:" ]]
}

################################################################################
# Time Formatting Tests
################################################################################

@test "CLI formats time in human readable format" {
    run time-tools format '2024-01-15T10:00:00Z' human
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Formatted Time" ]]
    [[ "$output" =~ "Result:" ]]
}

@test "CLI formats time in ISO format" {
    run time-tools format '2024-01-15T10:00:00Z' iso8601
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Formatted Time" ]]
}

@test "CLI formats time with timezone" {
    run time-tools format '2024-01-15T10:00:00Z' human --timezone 'America/New_York'
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Formatted Time" ]]
    [[ "$output" =~ "Timezone: America/New_York" ]]
}

@test "CLI handles missing format arguments" {
    run time-tools format
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Usage:" ]]
}

################################################################################
# Current Time Tests
################################################################################

@test "CLI shows current time in UTC" {
    run time-tools now UTC
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Current Time" ]]
    [[ "$output" =~ "Timezone: UTC" ]]
}

@test "CLI shows current time in Eastern timezone" {
    run time-tools now 'America/New_York'
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Current Time" ]]
    [[ "$output" =~ "Timezone: America/New_York" ]]
}

@test "CLI shows current time with default UTC" {
    run time-tools now
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Current Time" ]]
    [[ "$output" =~ "Timezone: UTC" ]]
}

################################################################################
# Scheduling Tests
################################################################################

@test "CLI finds optimal meeting times" {
    run time-tools schedule --participants alice,bob --duration 60 --earliest 2024-06-15 --latest 2024-06-22
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Optimal Meeting Times" ]]
    [[ "$output" =~ "Participants: alice,bob" ]]
    [[ "$output" =~ "Duration: 60 minutes" ]]
}

@test "CLI finds optimal meeting times with business hours only" {
    run time-tools schedule --participants alice,bob,charlie --duration 30 --earliest 2024-06-15 --latest 2024-06-16 --business-hours-only
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Optimal Meeting Times" ]]
}

@test "CLI handles missing participants for scheduling" {
    run time-tools schedule --duration 60 --earliest 2024-06-15 --latest 2024-06-22
    [ "$status" -ne 0 ]
    [[ "$output" =~ "--participants required" ]]
}

################################################################################
# Conflict Detection Tests
################################################################################

@test "CLI checks for conflicts" {
    run time-tools conflicts '2024-01-15T14:00:00Z' '2024-01-15T15:00:00Z' --organizer alice
    [ "$status" -eq 0 ]
    # Should show either conflicts detected or no conflicts
    [[ "$output" =~ "Conflict" ]]
}

@test "CLI handles missing conflict detection arguments" {
    run time-tools conflicts
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Usage:" ]]
}

################################################################################
# Event Management Tests
################################################################################

@test "CLI lists events" {
    run time-tools event list
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Events" ]]
}

@test "CLI lists events with filters" {
    run time-tools event list --organizer test@example.com --status confirmed
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Events" ]]
}

################################################################################
# JSON Output Tests
################################################################################

@test "CLI returns JSON output when requested" {
    run time-tools --json convert '2024-01-15T10:00:00Z' UTC 'America/New_York'
    [ "$status" -eq 0 ]
    # Should be valid JSON
    echo "$output" | jq . >/dev/null
}

@test "CLI returns JSON for status check" {
    run time-tools --json status
    [ "$status" -eq 0 ]
    # Should be valid JSON with expected fields
    echo "$output" | jq -e '.status and .version' >/dev/null
}

@test "CLI returns JSON for duration calculation" {
    run time-tools --json duration '2024-01-15T09:00:00Z' '2024-01-15T17:00:00Z'
    [ "$status" -eq 0 ]
    echo "$output" | jq -e '.total_hours' >/dev/null
}

################################################################################
# Error Handling Tests
################################################################################

@test "CLI handles unknown command gracefully" {
    run time-tools unknown-command
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Unknown command" ]]
    [[ "$output" =~ "help" ]]
}

@test "CLI handles API unavailable scenario" {
    # This test might be skipped if API is running
    skip "Requires API to be stopped for testing"
}

@test "CLI validates date formats" {
    run time-tools convert 'invalid-date' UTC 'America/New_York'
    [ "$status" -ne 0 ]
}

################################################################################
# Integration Tests
################################################################################

@test "CLI workflow: convert timezone then calculate duration" {
    # Convert timezone
    run time-tools convert '2024-01-15T09:00:00' 'America/New_York' UTC
    [ "$status" -eq 0 ]
    
    # Calculate duration
    run time-tools duration '2024-01-15T14:00:00Z' '2024-01-15T22:00:00Z'
    [ "$status" -eq 0 ]
    [[ "$output" =~ "8.00" ]]
}

@test "CLI workflow: check current time and format it" {
    # Get current time
    run time-tools now UTC
    [ "$status" -eq 0 ]
    
    # Format a specific time
    run time-tools format '2024-01-15T10:00:00Z' human
    [ "$status" -eq 0 ]
}

################################################################################
# Performance Tests
################################################################################

@test "CLI responds quickly to simple commands" {
    # Test that basic commands complete within reasonable time
    start_time=$(date +%s)
    run time-tools status
    end_time=$(date +%s)
    duration=$((end_time - start_time))
    
    [ "$status" -eq 0 ]
    [ "$duration" -le 10 ]  # Should complete within 10 seconds
}

################################################################################
# Cleanup
################################################################################

teardown() {
    # Clean up any temporary files or state if needed
    true
}