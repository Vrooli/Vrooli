#!/usr/bin/env bats
# Test to verify Tier 2 mock integration with BATS

bats_require_minimum_version 1.5.0

# Load simple Tier 2 setup (avoiding complex Vrooli infrastructure)
source "${BATS_TEST_DIRNAME}/../fixtures/simple-tier2-setup.bash"

# Load BATS helpers
load "../helpers/bats-support/load"
load "../helpers/bats-assert/load"

setup() {
    # This should load Tier 2 mocks automatically
    vrooli_setup_unit_test
    
    # Explicitly load the mocks we need for these tests
    if declare -F load_test_mock >/dev/null 2>&1; then
        load_test_mock "redis"
        load_test_mock "postgres"
    fi
}

teardown() {
    vrooli_cleanup_test
}

@test "Redis mock is loaded and functional" {
    # Test that redis-cli command is available
    run redis-cli ping
    assert_success
    assert_output "PONG"
}

@test "Redis mock supports SET/GET operations" {
    # Set a value
    run redis-cli set "test_key" "test_value"
    assert_success
    assert_output "OK"
    
    # Get the value
    run redis-cli get "test_key"
    assert_success
    assert_output "test_value"
}

@test "PostgreSQL mock is loaded and functional" {
    # Test basic psql command
    run psql -c "SELECT 1"
    assert_success
}

@test "Docker mock is loaded and functional" {
    # Test docker ps command
    run docker ps
    assert_success
}

@test "Multiple mocks can work together" {
    # Test Redis
    run redis-cli set "multi_test" "works"
    assert_success
    
    # Test PostgreSQL
    run psql -c "CREATE TABLE test (id INT)"
    assert_success
    
    # Test Docker
    run docker ps --format "table"
    assert_success
}

@test "Mock reset functions work" {
    # Set a Redis value
    redis-cli set "reset_test" "value" >/dev/null
    
    # Reset the mock
    if declare -F redis_mock_reset >/dev/null 2>&1; then
        redis_mock_reset
    fi
    
    # Value should be gone
    run redis-cli get "reset_test"
    assert_success
    assert_output "(nil)"
}

@test "Error injection works" {
    # Inject an error in Redis mock
    if declare -F redis_mock_set_error >/dev/null 2>&1; then
        redis_mock_set_error "connection_failed"
        
        run redis-cli ping
        assert_failure
        
        # Clear the error
        redis_mock_set_error ""
        
        run redis-cli ping
        assert_success
        assert_output "PONG"
    else
        skip "Error injection not available"
    fi
}