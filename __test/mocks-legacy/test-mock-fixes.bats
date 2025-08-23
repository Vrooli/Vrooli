#!/usr/bin/env bats

# Simple test to verify mock infrastructure fixes

setup() {
    export TEST_DIR="$(cd "$BATS_TEST_DIRNAME" && pwd)"
    
    # Load mocks
    source "$TEST_DIR/system.sh"
    source "$TEST_DIR/commands.bash"
    
    # Enable strict mode to test for unbound variables
    set -euo pipefail
    
    # Initialize mock state
    mock::system::reset
    
    # Set up error codes
    export ERROR_USAGE=64
}

teardown() {
    # Reset state
    mock::system::reset || true
}

@test "system mock: can set and get process state without unbound errors" {
    # This should not trigger unbound variable errors
    mock::system::set_process_state "12345" "test_process" "running" "1" "/usr/bin/test" "testuser" "0.1" "0.2"
    
    # Verify the process was set
    run mock::system::get::process_name "12345"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "test_process" ]]
}

@test "system mock: handles missing array keys without errors" {
    # Access a non-existent process should not cause unbound variable error
    run mock::system::get::process_name "99999"
    [ "$status" -eq 0 ]
    # Output should be empty but no error
    [ -z "$output" ]
}

@test "systemctl mock: can set and get service state without unbound errors" {
    # This should not trigger unbound variable errors
    mock::systemctl::set_service_state "test_service" "active" "enabled" "running"
    
    # Verify the service was set
    run mock::systemctl::get::service_state "test_service"
    [ "$status" -eq 0 ]
    [[ "$output" == "active" ]]
}

@test "systemctl mock: handles missing services without errors" {
    # Access a non-existent service should not cause unbound variable error
    run mock::systemctl::get::service_state "nonexistent_service"
    [ "$status" -eq 0 ]
    # Output should be empty but no error
    [ -z "$output" ]
}

@test "commands mock: exports are correct" {
    # Test that the functions are exported correctly
    run bash -c "type -t command"
    [ "$status" -eq 0 ]
    [[ "$output" == "function" ]]
    
    run bash -c "type -t which"
    [ "$status" -eq 0 ]
    [[ "$output" == "function" ]]
    
    # jq should NOT be exported (we removed it)
    run bash -c "type -t jq"
    [ "$status" -ne 0 ] || [[ "$output" != "function" ]]
}

@test "system mock: process pattern cache updates correctly" {
    # Add a process
    mock::system::set_process_state "10001" "nginx" "running" "1" "/usr/sbin/nginx" "www-data" "1.0" "2.0"
    
    # Pattern cache should have been updated
    # This tests the fixed unbound variable issue in _update_pattern_cache
    run pgrep nginx
    [ "$status" -eq 0 ]
    [[ "$output" == "10001" ]]
}

@test "system mock: kill command handles PIDs correctly" {
    # Add a process
    mock::system::set_process_state "20001" "test_app" "running" "1" "/usr/bin/test_app" "testuser" "0.5" "1.0"
    
    # Kill should work without unbound variable errors
    run kill 20001
    [ "$status" -eq 0 ]
    
    # Process should be gone
    run mock::system::assert::process_not_exists "20001"
    [ "$status" -eq 0 ]
}

@test "systemctl mock: is-active works with undefined services" {
    # This should return a default state without errors
    run systemctl is-active undefined_service
    # Should either succeed with "active" or fail gracefully
    # But should NOT cause unbound variable error
    [ "$status" -eq 0 ] || [ "$status" -eq 3 ]
}