#!/usr/bin/env bats
# Test the mock framework itself

# Direct source since we're in the same directory structure
setup() {
    export BATS_TEST_TMPDIR="${BATS_TEST_TMPDIR:-$(mktemp -d)}"
    export MOCK_RESPONSES_DIR="$BATS_TEST_TMPDIR/mock_responses"
    mkdir -p "$MOCK_RESPONSES_DIR"
    
    source "$(dirname "${BATS_TEST_FILENAME}")/mocks/system_mocks.bash"
    source "$(dirname "${BATS_TEST_FILENAME}")/mocks/mock_helpers.bash"
    source "$(dirname "${BATS_TEST_FILENAME}")/mocks/resource_mocks.bash"
}

teardown() {
    [[ -d "$MOCK_RESPONSES_DIR" ]] && rm -rf "$MOCK_RESPONSES_DIR"
}

@test "docker mock responds to basic commands" {
    output=$(docker version)
    [[ "$output" =~ "Docker version" ]]
}

@test "mock::docker::set_container_state configures docker ps correctly" {
    mock::docker::set_container_state "test-container" "running"
    
    output=$(docker ps --filter name=test-container)
    [[ "$output" == "test-container" ]]
}

@test "mock::docker::set_container_state configures docker inspect correctly" {
    mock::docker::set_container_state "test-container" "running"
    
    output=$(docker inspect test-container)
    [[ "$output" =~ "Running.*true" ]]
}

@test "curl mock responds to health endpoints" {
    output=$(curl http://localhost:8080/health)
    [[ "$output" =~ "ok" ]]
}

@test "mock::http::set_endpoint_state configures healthy endpoint" {
    mock::http::set_endpoint_state "http://localhost:9999" "healthy"
    
    output=$(curl http://localhost:9999)
    [[ "$output" =~ "healthy.*true" ]]
}

@test "mock::http::set_endpoint_state configures unhealthy endpoint" {
    mock::http::set_endpoint_state "http://localhost:9999" "unhealthy"
    
    run curl http://localhost:9999
    [ "$status" -eq 1 ]
    [[ "$output" =~ "error" ]]
}

@test "mock::resource::setup configures full resource state" {
    mock::resource::setup "test-resource" "installed_running" "8888"
    
    # Check docker state
    output=$(docker ps --filter name=test-resource)
    [[ "$output" == "test-resource" ]]
    
    # Check HTTP state
    output=$(curl http://localhost:8888)
    [[ "$output" =~ "ok" ]]
}

@test "mock cleanup works correctly" {
    mock::docker::set_container_state "test-container" "running"
    [[ -f "$MOCK_RESPONSES_DIR/container_test-container_state" ]]
    
    mock::cleanup
    [[ ! -f "$MOCK_RESPONSES_DIR/container_test-container_state" ]]
}

@test "ollama-specific mock setup works" {
    mock::ollama::setup "healthy"
    
    output=$(curl http://localhost:11434/api/tags)
    [[ "$output" =~ "llama3.1:8b" ]]
}

@test "qdrant-specific mock setup works" {
    mock::qdrant::setup "healthy"
    
    output=$(curl http://localhost:6333/health)
    [[ "$output" =~ "ok" ]]
    
    output=$(curl http://localhost:6333/collections)
    [[ "$output" =~ "collections" ]]
}