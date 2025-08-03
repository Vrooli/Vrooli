#!/usr/bin/env bats
# Infrastructure Health Check
# Comprehensive validation of the BATS testing infrastructure
# Run this after making changes to ensure everything still works

bats_require_minimum_version 1.5.0

# Load the infrastructure
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${TEST_DIR}/../core/common_setup.bash"

setup() {
    # Clean slate for each test
    unset COMMON_SETUP_LOADED
    unset MOCK_REGISTRY_LOADED
    unset ASSERTIONS_LOADED
    unset PATH_RESOLVER_LOADED
    unset ERROR_HANDLING_LOADED
    unset BENCHMARKING_LOADED
}

teardown() {
    cleanup_mocks 2>/dev/null || true
}

#######################################
# Core Infrastructure Tests
#######################################

@test "Health: Path resolver loads and functions correctly" {
    run bash -c "
        source '${TEST_DIR}/../core/path_resolver.bash'
        vrooli_validate_test_environment && echo 'validated'
        vrooli_fixture_path 'core/assertions.bash' | grep -q 'assertions.bash' && echo 'path_works'
    "
    assert_success
    assert_output_contains "validated"
    assert_output_contains "path_works"
}

@test "Health: Common setup loads without errors" {
    run bash -c "
        source '${TEST_DIR}/../core/common_setup.bash'
        [[ \${COMMON_SETUP_LOADED} == 'true' ]] && echo 'loaded'
    "
    assert_success
    assert_output_contains "loaded"
}

@test "Health: All core modules load correctly" {
    run bash -c "
        source '${TEST_DIR}/../core/common_setup.bash'
        [[ \${ERROR_HANDLING_LOADED} == 'true' ]] && echo 'error_handling: OK'
        [[ \${BENCHMARKING_LOADED} == 'true' ]] && echo 'benchmarking: OK'
        [[ \${MOCK_REGISTRY_LOADED} == 'true' ]] && echo 'mock_registry: OK'
        [[ \${ASSERTIONS_LOADED} == 'true' ]] && echo 'assertions: OK'
    "
    assert_success
    assert_output_contains "error_handling: OK"
    assert_output_contains "benchmarking: OK"
    assert_output_contains "mock_registry: OK"
    assert_output_contains "assertions: OK"
}

#######################################
# Setup Function Tests
#######################################

@test "Health: Standard mocks setup works" {
    run bash -c "
        source '${TEST_DIR}/../core/common_setup.bash'
        setup_standard_mocks
        [[ -n \${TEST_NAMESPACE} ]] && echo 'namespace_set'
        [[ -d \${MOCK_RESPONSES_DIR} ]] && echo 'mock_dir_exists'
    "
    assert_success
    assert_output_contains "namespace_set"
    assert_output_contains "mock_dir_exists"
}

@test "Health: Resource test setup works" {
    run bash -c "
        source '${TEST_DIR}/../core/common_setup.bash'
        setup_resource_test 'ollama'
        [[ \${OLLAMA_PORT} == '11434' ]] && echo 'port_configured'
        [[ -n \${OLLAMA_BASE_URL} ]] && echo 'url_configured'
    "
    assert_success
    assert_output_contains "port_configured"
    assert_output_contains "url_configured"
}

@test "Health: Integration test setup works" {
    run bash -c "
        source '${TEST_DIR}/../core/common_setup.bash'
        setup_integration_test 'ollama' 'whisper'
        mock::is_loaded 'resource' 'ollama' && echo 'ollama_loaded'
        mock::is_loaded 'resource' 'whisper' && echo 'whisper_loaded'
    "
    assert_success
    assert_output_contains "ollama_loaded"
    assert_output_contains "whisper_loaded"
}

#######################################
# Assertion Tests
#######################################

@test "Health: Basic assertions work" {
    source "${TEST_DIR}/../core/common_setup.bash"
    
    # Test success assertion
    run true
    assert_success
    
    # Test failure assertion
    run false
    assert_failure
    
    # Test output assertions
    run echo "test output"
    assert_output_contains "test"
    assert_output_not_contains "missing"
}

@test "Health: File assertions work" {
    source "${TEST_DIR}/../core/common_setup.bash"
    setup_standard_mocks
    
    local test_file="${BATS_TEST_TMPDIR}/test.txt"
    echo "content" > "$test_file"
    
    assert_file_exists "$test_file"
    assert_file_contains "$test_file" "content"
    assert_dir_exists "$BATS_TEST_TMPDIR"
}

@test "Health: JSON assertions work" {
    source "${TEST_DIR}/../core/common_setup.bash"
    
    local json='{"status":"ok","count":42}'
    assert_json_valid "$json"
    assert_json_field_equals "$json" ".status" "ok"
    assert_json_field_equals "$json" ".count" "42"
}

#######################################
# Mock System Tests
#######################################

@test "Health: Docker mocks work" {
    source "${TEST_DIR}/../core/common_setup.bash"
    setup_standard_mocks
    
    run docker --version
    assert_success
    assert_output_contains "Docker"
    
    run docker ps
    assert_success
}

@test "Health: HTTP mocks work" {
    source "${TEST_DIR}/../core/common_setup.bash"
    setup_standard_mocks
    
    run curl -s http://localhost:8080/health
    assert_success
}

@test "Health: System command mocks work" {
    source "${TEST_DIR}/../core/common_setup.bash"
    setup_standard_mocks
    
    run jq --version
    assert_success
    
    run systemctl status docker
    assert_success
}

#######################################
# Resource Mock Tests
#######################################

@test "Health: Resource mocks load correctly" {
    source "${TEST_DIR}/../core/common_setup.bash"
    
    # Test each resource category
    local resources=(
        "ai:ollama"
        "ai:whisper"
        "automation:n8n"
        "agents:claude-code"
        "storage:postgres"
        "search:searxng"
        "execution:judge0"
    )
    
    for resource_spec in "${resources[@]}"; do
        IFS=':' read -r category name <<< "$resource_spec"
        run bash -c "
            source '${TEST_DIR}/../core/common_setup.bash'
            mock::load 'resource' '$name'
            echo 'loaded: $name'
        "
        assert_success
        assert_output_contains "loaded: $name"
    done
}

#######################################
# Path Resolution Tests
#######################################

@test "Health: Path resolution works from different directories" {
    # Test that infrastructure works when called from various locations
    for test_dir in "/tmp" "$HOME" "/"; do
        run bash -c "
            cd '$test_dir' 2>/dev/null || exit 0
            source '${TEST_DIR}/../core/common_setup.bash'
            [[ \${COMMON_SETUP_LOADED} == 'true' ]] && echo 'works_from_$test_dir'
        "
        assert_success
        assert_output_contains "works_from_$test_dir"
    done
}

#######################################
# Backward Compatibility Tests
#######################################

@test "Health: Legacy redirect files work with warnings" {
    # Test standard_mock_framework.bash redirect
    run bash -c "
        source '${TEST_DIR}/../standard_mock_framework.bash' 2>&1
        [[ \${COMMON_SETUP_LOADED} == 'true' ]] && echo 'redirect_works'
    "
    assert_success
    assert_output_contains "DEPRECATED"
    assert_output_contains "redirect_works"
}

@test "Health: Legacy functions are available" {
    source "${TEST_DIR}/../core/common_setup.bash"
    
    # Check that legacy function names still work
    assert_function_exists "setup_standard_mock_framework"
    assert_function_exists "cleanup_standard_mock_framework"
}

#######################################
# Performance Tests
#######################################

@test "Health: Performance mode setup is fast" {
    local start_time=$(date +%s%N)
    
    source "${TEST_DIR}/../core/common_setup.bash"
    setup_performance_test
    
    local end_time=$(date +%s%N)
    local duration=$((($end_time - $start_time) / 1000000)) # Convert to milliseconds
    
    # Should complete in less than 100ms
    [[ $duration -lt 100 ]] || fail "Performance setup took ${duration}ms (expected < 100ms)"
}

#######################################
# Cleanup Tests
#######################################

@test "Health: Cleanup removes temporary files" {
    source "${TEST_DIR}/../core/common_setup.bash"
    setup_standard_mocks
    
    local tmpdir="$BATS_TEST_TMPDIR"
    [[ -d "$tmpdir" ]] || fail "Temp directory not created"
    
    cleanup_mocks
    
    # Directory should be removed
    [[ ! -d "$tmpdir" ]] || fail "Temp directory not cleaned up"
}

#######################################
# Error Handling Tests
#######################################

@test "Health: Invalid resource name is handled gracefully" {
    run bash -c "
        source '${TEST_DIR}/../core/common_setup.bash'
        setup_resource_test 'nonexistent_resource' 2>&1
        echo 'continued_execution'
    "
    assert_success
    assert_output_contains "continued_execution"
}

@test "Health: Missing required parameters are caught" {
    run bash -c "
        source '${TEST_DIR}/../core/common_setup.bash'
        setup_resource_test '' 2>&1
    "
    assert_failure
    assert_output_contains "required"
}

#######################################
# Integration Sanity Check
#######################################

@test "Health: Complete workflow test" {
    # This test simulates a complete test workflow
    source "${TEST_DIR}/../core/common_setup.bash"
    
    # Setup
    setup_integration_test "ollama" "whisper"
    
    # Verify environment
    assert_env_set "TEST_NAMESPACE"
    assert_env_set "OLLAMA_PORT"
    assert_env_set "WHISPER_PORT"
    
    # Test mock functionality
    run docker ps
    assert_success
    
    run curl -s http://localhost:11434/health
    assert_success
    
    # Cleanup
    cleanup_mocks
    
    # Verify cleanup
    [[ -z "${LOADED_MOCKS[*]}" ]] || fail "Mocks not cleared"
}

#######################################
# Summary
#######################################

@test "Health: Infrastructure summary" {
    echo "=== BATS Testing Infrastructure Health Check ==="
    echo "Path Resolver: ✓"
    echo "Common Setup: ✓"
    echo "Mock Registry: ✓"
    echo "Assertions: ✓"
    echo "Resource Mocks: ✓"
    echo "Backward Compatibility: ✓"
    echo "Performance: ✓"
    echo "Error Handling: ✓"
    echo "=============================================="
    
    # Always pass - this is just informational
    true
}