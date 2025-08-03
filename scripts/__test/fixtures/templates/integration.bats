#!/usr/bin/env bats
# Integration Test Template - Multiple services working together (50 lines)
# Usage: Replace SERVICES array and add integration scenarios

bats_require_minimum_version 1.5.0

# Load test infrastructure
source "$(dirname "$0")/../setup.bash"

# TODO: Replace with your services
SERVICES=("ollama" "postgres")

setup() {
    vrooli_setup_integration_test "${SERVICES[@]}"  # Multi-service mocks
}

teardown() {
    vrooli_cleanup_test
}

@test "all services configured" {
    # Verify each service has proper environment
    for service in "${SERVICES[@]}"; do
        assert_env_set "VROOLI_${service^^}_PORT"
    done
}

@test "TODO: test service A to B flow" {
    # Example: Data flows from postgres to ollama
    
    # Step 1: Store data in service A
    run psql -c "INSERT INTO test VALUES('data');"
    assert_success
    
    # Step 2: Process with service B
    run ollama generate "Analyze: data"
    assert_success
    
    # Step 3: Verify integration
    assert_output_contains "analysis"
}

@test "TODO: test end-to-end workflow" {
    # Example: Complete workflow across all services
    run curl -X POST "$VROOLI_OLLAMA_BASE_URL/workflow"
    assert_success
    assert_json_valid "$output"
}

# Template: 50 lines | Multi-service integration testing