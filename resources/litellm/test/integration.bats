#!/usr/bin/env bats
# LiteLLM Integration Tests

# Load test helpers
load "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/helpers/common.sh"
load "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/helpers/docker.sh"

# Test configuration
LITELLM_CLI="${BATS_TEST_DIRNAME}/../cli.sh"
LITELLM_CONTAINER="vrooli-litellm"
LITELLM_PORT="11435"
LITELLM_URL="http://localhost:${LITELLM_PORT}"

setup() {
    # Ensure we have a clean test environment
    export var_ROOT_DIR="${BATS_TEST_DIRNAME}/../../../../.."
    mkdir -p "${var_ROOT_DIR}/data/test-results"
}

teardown() {
    # Clean up any test artifacts
    true
}

@test "CLI exists and is executable" {
    [ -x "$LITELLM_CLI" ]
}

@test "CLI shows help" {
    run "$LITELLM_CLI" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "LiteLLM Resource CLI" ]]
}

@test "status command works" {
    run "$LITELLM_CLI" status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Status:" ]]
}

@test "status command supports JSON format" {
    run "$LITELLM_CLI" status --format json
    [ "$status" -eq 0 ]
    # Should produce valid JSON
    echo "$output" | jq . >/dev/null
}

@test "can install LiteLLM" {
    # Skip if already installed and running
    if docker ps | grep -q "$LITELLM_CONTAINER"; then
        skip "LiteLLM already running"
    fi
    
    run timeout 300 "$LITELLM_CLI" install --verbose
    [ "$status" -eq 0 ]
    [[ "$output" =~ "installation successful" || "$output" =~ "already running" ]]
}

@test "container is running after install" {
    run docker ps --format "{{.Names}}"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "$LITELLM_CONTAINER" ]]
}

@test "service responds to health check" {
    # Wait for service to be ready
    sleep 10
    
    run "$LITELLM_CLI" test --timeout 30
    [ "$status" -eq 0 ]
    [[ "$output" =~ "accessible" ]]
}

@test "can list models" {
    run timeout 30 "$LITELLM_CLI" list-models
    # Should succeed even if no API keys configured (may return empty list)
    [ "$status" -eq 0 ]
}

@test "proxy status endpoint works" {
    run timeout 15 "$LITELLM_CLI" proxy-status
    [ "$status" -eq 0 ]
    # Should return JSON or error message
}

@test "logs command works" {
    run "$LITELLM_CLI" logs 10
    [ "$status" -eq 0 ]
}

@test "stats command works" {
    run "$LITELLM_CLI" stats
    [ "$status" -eq 0 ]
}

@test "content management works" {
    # Test adding content
    run "$LITELLM_CLI" content add --type example --name test --data "test content"
    [ "$status" -eq 0 ]
    
    # Test listing content
    run "$LITELLM_CLI" content list --type example
    [ "$status" -eq 0 ]
    [[ "$output" =~ "test" ]]
    
    # Test getting content
    run "$LITELLM_CLI" content get --type example --name test
    [ "$status" -eq 0 ]
    [[ "$output" =~ "test content" ]]
    
    # Test removing content
    run "$LITELLM_CLI" content remove --type example --name test
    [ "$status" -eq 0 ]
}

@test "validate command passes" {
    run "$LITELLM_CLI" validate --verbose
    [ "$status" -eq 0 ]
    [[ "$output" =~ "validation successful" ]]
}

@test "can stop LiteLLM service" {
    run "$LITELLM_CLI" stop --verbose
    [ "$status" -eq 0 ]
}

@test "can start LiteLLM service" {
    run timeout 60 "$LITELLM_CLI" start --verbose
    [ "$status" -eq 0 ]
    
    # Verify it's running
    sleep 5
    run docker ps --format "{{.Names}}"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "$LITELLM_CONTAINER" ]]
}

@test "service restarts successfully" {
    run timeout 90 "$LITELLM_CLI" restart --verbose
    [ "$status" -eq 0 ]
    
    # Verify it's running after restart
    sleep 5
    run "$LITELLM_CLI" test --timeout 15
    [ "$status" -eq 0 ]
}

@test "configuration files exist" {
    [ -f "${var_ROOT_DIR}/data/litellm/config/config.yaml" ]
    [ -f "${var_ROOT_DIR}/data/litellm/config/.env" ]
}

@test "port is accessible" {
    # Test if the port is accessible
    run timeout 10 curl -s "http://localhost:${LITELLM_PORT}/health"
    # Should not fail with connection refused
    [[ ! "$output" =~ "Connection refused" ]]
}

@test "can backup configuration" {
    run "$LITELLM_CLI" backup --verbose
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Backup created successfully" ]]
}