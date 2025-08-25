#!/usr/bin/env bats
# Test to verify Tier 2 mock integration with BATS

bats_require_minimum_version 1.5.0

# Load BATS helpers first
load "../helpers/bats-support/load"
load "../helpers/bats-assert/load"

# Setup mocks before each test
setup() {
    # Set up environment
    export APP_ROOT="$(builtin cd "${BATS_TEST_DIRNAME}/../.." && builtin pwd)"
    export MOCK_BASE_DIR="${APP_ROOT}/__test/mocks"
    export MOCK_TIER2_DIR="${MOCK_BASE_DIR}/tier2"
    
    # Load mocks directly for each test (BATS runs each test in a subshell)
    source "${MOCK_TIER2_DIR}/redis.sh"
    source "${MOCK_TIER2_DIR}/postgres.sh"
    source "${MOCK_TIER2_DIR}/docker.sh"
    
    # Initialize mock states
    redis_mock_reset
    postgres_mock_reset
    docker_mock_reset
}

teardown() {
    # Clean up mock states
    if declare -F redis_mock_reset >/dev/null 2>&1; then
        redis_mock_reset
    fi
    if declare -F postgres_mock_reset >/dev/null 2>&1; then
        postgres_mock_reset
    fi
    if declare -F docker_mock_reset >/dev/null 2>&1; then
        docker_mock_reset
    fi
}

@test "Redis mock is loaded and functional" {
    # Test that redis-cli command is available
    run redis-cli ping
    assert_success
    assert_output "PONG"
}

@test "Redis mock supports SET/GET operations" {
    # Since BATS runs each command in a subshell, we need to use a different approach
    # We'll run both commands in the same subshell to preserve state
    run bash -c "source '${MOCK_TIER2_DIR}/redis.sh' && redis-cli set 'test_key' 'test_value' && redis-cli get 'test_key'"
    assert_success
    assert_line --index 0 "OK"
    assert_line --index 1 "test_value"
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
    # Run in a single subshell to maintain state
    run bash -c '
        source "${MOCK_TIER2_DIR}/redis.sh"
        redis-cli set "reset_test" "value" >/dev/null
        redis_mock_reset
        redis-cli get "reset_test"
    '
    assert_success
    assert_output "(nil)"
}

@test "Error injection works" {
    # Run error injection test in a single subshell
    run bash -c '
        source "${MOCK_TIER2_DIR}/redis.sh"
        redis_mock_set_error "connection_failed"
        redis-cli ping 2>&1
    '
    assert_failure
    assert_output --partial "Connection refused"
    
    # Test clearing the error
    run bash -c '
        source "${MOCK_TIER2_DIR}/redis.sh"
        redis_mock_set_error ""
        redis-cli ping
    '
    assert_success
    assert_output "PONG"
}