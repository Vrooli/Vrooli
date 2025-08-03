#!/usr/bin/env bats
# Service Test Template - Test a specific service (40 lines)
# Usage: Replace SERVICE_NAME and add your tests

bats_require_minimum_version 1.5.0

# Load test infrastructure
source "$(dirname "$0")/../setup.bash"

# TODO: Replace with your service name
SERVICE_NAME="ollama"

setup() {
    vrooli_setup_service_test "$SERVICE_NAME"  # Service-specific mocks
}

teardown() {
    vrooli_cleanup_test
}

@test "service environment configured" {
    # Verify service environment variables
    assert_env_set "VROOLI_${SERVICE_NAME^^}_PORT"
    assert_env_set "VROOLI_${SERVICE_NAME^^}_BASE_URL"
}

@test "TODO: test service health" {
    # Example: Test service is responding
    run curl -s "$VROOLI_OLLAMA_BASE_URL/health"
    assert_success
    assert_output_contains "healthy"
}

@test "TODO: test service-specific functionality" {
    # Example: Test unique service features
    run ${SERVICE_NAME} --version
    assert_success
    assert_output_contains "version"
}

# Template: 40 lines | Service-specific testing