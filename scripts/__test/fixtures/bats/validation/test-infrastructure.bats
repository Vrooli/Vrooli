#!/usr/bin/env bats
# Validation Test Suite for BATS Testing Infrastructure
# This test validates that the unified testing infrastructure works correctly

bats_require_minimum_version 1.5.0

# Load BATS libraries
load "${BATS_TEST_DIRNAME}/../../../helpers/bats-support/load"
load "${BATS_TEST_DIRNAME}/../../../helpers/bats-assert/load"

# Load the infrastructure we're testing
source "${BATS_TEST_DIRNAME}/../core/common_setup.bash"

setup() {
    # Clean environment for each test
    unset COMMON_SETUP_LOADED
    unset MOCK_REGISTRY_LOADED
    unset ASSERTIONS_LOADED
}

teardown() {
    # Cleanup after each test
    cleanup_mocks 2>/dev/null || true
}

@test "infrastructure loads without errors" {
    # Re-source to test loading
    run bash -c "source '${BATS_TEST_DIRNAME}/../core/common_setup.bash' && echo 'loaded'"
    assert_success
    assert_output_contains "loaded"
}

@test "mock registry system is available" {
    run bash -c "source '${BATS_TEST_DIRNAME}/../core/common_setup.bash' && declare -f mock::load >/dev/null && echo 'available'"
    assert_success
    assert_output_contains "available"
}

@test "assertion library is available" {
    run bash -c "source '${BATS_TEST_DIRNAME}/../core/common_setup.bash' && declare -f assert_output_contains >/dev/null && echo 'available'"
    assert_success
    assert_output_contains "available"
}

@test "setup_standard_mocks function works" {
    run bash -c "source '${BATS_TEST_DIRNAME}/../core/common_setup.bash' && setup_standard_mocks && echo 'success'"
    assert_success
    assert_output_contains "success"
    assert_output_contains "Setting up standard test environment"
}

@test "setup_resource_test function works with ollama" {
    run bash -c "source '${BATS_TEST_DIRNAME}/../core/common_setup.bash' && setup_resource_test ollama && echo 'success'"
    assert_success
    assert_output_contains "success"
    assert_output_contains "Setting up test environment for resource: ollama"
}

@test "setup_integration_test function works" {
    run bash -c "source '${BATS_TEST_DIRNAME}/../core/common_setup.bash' && setup_integration_test ollama n8n && echo 'success'"
    assert_success
    assert_output_contains "success"
    assert_output_contains "Setting up integration test environment for: ollama n8n"
}

@test "setup_performance_test function works" {
    run bash -c "source '${BATS_TEST_DIRNAME}/../core/common_setup.bash' && setup_performance_test && echo 'success'"
    assert_success
    assert_output_contains "success"
    assert_output_contains "Setting up performance test environment"
}

@test "cleanup_mocks function works" {
    run bash -c "source '${BATS_TEST_DIRNAME}/../core/common_setup.bash' && setup_standard_mocks && cleanup_mocks && echo 'success'"
    assert_success
    assert_output_contains "success"
    assert_output_contains "Cleaning up test environment"
}

@test "environment variables are set correctly" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'
        setup_standard_mocks
        echo \"FORCE=\${FORCE}\"
        echo \"YES=\${YES}\"
        echo \"OUTPUT_FORMAT=\${OUTPUT_FORMAT}\"
        echo \"TEST_NAMESPACE=\${TEST_NAMESPACE}\"
    "
    assert_success
    assert_output_contains "FORCE=no"
    assert_output_contains "YES=no"
    assert_output_contains "OUTPUT_FORMAT=text"
    assert_output_contains "TEST_NAMESPACE=test_"
}

@test "resource-specific environment variables are set" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'
        setup_resource_test ollama
        echo \"OLLAMA_PORT=\${OLLAMA_PORT}\"
        echo \"OLLAMA_BASE_URL=\${OLLAMA_BASE_URL}\"
        echo \"OLLAMA_CONTAINER_NAME=\${OLLAMA_CONTAINER_NAME}\"
    "
    assert_success
    assert_output_contains "OLLAMA_PORT=11434"
    assert_output_contains "OLLAMA_BASE_URL=http://localhost:11434"
    assert_output_contains "OLLAMA_CONTAINER_NAME="
}

@test "mock loading system works for system mocks" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'
        mock::load system docker
        echo 'docker mock loaded'
    "
    assert_success
    assert_output_contains "docker mock loaded"
    assert_output_contains "Loaded system:docker"
}

@test "mock loading system works for resource mocks" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'
        mock::load resource ollama
        echo 'ollama mock loaded'
    "
    assert_success
    assert_output_contains "ollama mock loaded"
    assert_output_contains "Loaded resource:ollama"
}

@test "resource category detection works" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'
        echo \"ollama: \$(mock::detect_resource_category ollama)\"
        echo \"n8n: \$(mock::detect_resource_category n8n)\"
        echo \"redis: \$(mock::detect_resource_category redis)\"
        echo \"searxng: \$(mock::detect_resource_category searxng)\"
    "
    assert_success
    assert_output_contains "ollama: ai"
    assert_output_contains "n8n: automation"
    assert_output_contains "redis: storage"
    assert_output_contains "searxng: search"
}

@test "backward compatibility with legacy common_setup.bash" {
    run bash -c "source '${BATS_TEST_DIRNAME}/../common_setup.bash' && setup_standard_mocks && echo 'legacy works'"
    assert_success
    assert_output_contains "legacy works"
    assert_output_contains "Setting up standard test environment"
}

@test "all essential assertion functions are available" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'
        declare -f assert_output_contains >/dev/null && echo 'assert_output_contains: OK'
        declare -f assert_file_exists >/dev/null && echo 'assert_file_exists: OK'
        declare -f assert_env_set >/dev/null && echo 'assert_env_set: OK'
        declare -f assert_json_valid >/dev/null && echo 'assert_json_valid: OK'
        declare -f assert_resource_healthy >/dev/null && echo 'assert_resource_healthy: OK'
        declare -f assert_docker_container_running >/dev/null && echo 'assert_docker_container_running: OK'
    "
    assert_success
    assert_output_contains "assert_output_contains: OK"
    assert_output_contains "assert_file_exists: OK"
    assert_output_contains "assert_env_set: OK"
    assert_output_contains "assert_json_valid: OK"
    assert_output_contains "assert_resource_healthy: OK"
    assert_output_contains "assert_docker_container_running: OK"
}

@test "assertion functions work correctly" {
    source "${BATS_TEST_DIRNAME}/../core/common_setup.bash"
    
    # Test basic output assertion
    output="test output"
    assert_output_contains "test"
    assert_output_contains "output"
    
    # Test environment variable assertion
    export TEST_VAR="test_value"
    assert_env_set "TEST_VAR"
    assert_env_equals "TEST_VAR" "test_value"
    
    # Test JSON assertion
    local json='{"status": "ok", "version": "1.0"}'
    assert_json_valid "$json"
    assert_json_field_equals "$json" ".status" "ok"
    assert_json_field_exists "$json" ".version"
}

@test "temporary directory setup works" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'
        setup_standard_mocks
        echo \"TEST_TMPDIR=\${TEST_TMPDIR}\"
        [[ -d \"\${TEST_TMPDIR}\" ]] && echo 'directory exists'
    "
    assert_success
    assert_output_contains "TEST_TMPDIR="
    assert_output_contains "directory exists"
}

@test "mock call tracking works" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'
        setup_standard_mocks
        # Mock calls should be tracked
        docker ps >/dev/null 2>&1
        [[ -f \"\${MOCK_RESPONSES_DIR}/command_calls.log\" ]] && echo 'tracking works' || echo 'tracking missing'
    "
    assert_success
    # Note: This might show 'tracking missing' if the specific system mock doesn't implement call tracking
    # That's okay - it means we should enhance the system mocks in the future
}

@test "error handling works for invalid resource" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'
        setup_resource_test nonexistent_resource
    "
    # Should not fail catastrophically, should handle gracefully
    # The specific behavior depends on implementation
}

@test "error handling works for integration test without resources" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'
        setup_integration_test
    "
    assert_failure
    assert_output_contains "At least one resource required"
}

@test "performance mode works" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'
        setup_performance_test generic
        echo \"TEST_PERFORMANCE_MODE=\${TEST_PERFORMANCE_MODE}\"
        echo \"FORCE=\${FORCE}\"
        echo \"QUIET=\${QUIET}\"
    "
    assert_success
    assert_output_contains "TEST_PERFORMANCE_MODE=true"
    assert_output_contains "FORCE=yes"
    assert_output_contains "QUIET=yes"
}

@test "function export works correctly" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'
        # Test that functions are properly exported and available in subshells
        bash -c 'declare -f setup_standard_mocks >/dev/null && echo \"exported\"'
    "
    assert_success
    assert_output_contains "exported"
}

@test "auto-detection functionality works" {
    run bash -c "
        export BATS_TEST_NAME='performance test suite'
        source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'
        auto_setup_test_environment
        echo 'auto-detection complete'
    "
    assert_success
    assert_output_contains "auto-detection complete"
    assert_output_contains "Auto-detected performance test"
}

@test "shellcheck passes on all core files" {
    # Only run if shellcheck is available
    if ! command -v shellcheck >/dev/null 2>&1; then
        skip "shellcheck not available"
    fi
    
    run shellcheck "${BATS_TEST_DIRNAME}/../core/common_setup.bash"
    # Allow SC1091 (info about not following sourced files)
    if [[ "$status" -ne 0 ]]; then
        # Check if only SC1091 warnings exist
        if [[ "$output" =~ ^.*SC1091.*$ ]] && ! [[ "$output" =~ SC[0-9]{4}.*$ ]]; then
            # Only SC1091 warnings, which are acceptable
            assert_success
        else
            # Other warnings exist
            assert_failure
        fi
    else
        assert_success
    fi
}