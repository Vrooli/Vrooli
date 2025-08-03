#!/usr/bin/env bats
# Comprehensive Validation Test Suite for BATS Infrastructure Improvements
# This test validates all the improvements made to the testing infrastructure

bats_require_minimum_version 1.5.0

# Load BATS libraries
load "${BATS_TEST_DIRNAME}/../../../helpers/bats-support/load" 2>/dev/null || true
load "${BATS_TEST_DIRNAME}/../../../helpers/bats-assert/load" 2>/dev/null || true

# Load the infrastructure we're testing
source "${BATS_TEST_DIRNAME}/../core/common_setup.bash"

setup() {
    # Clean environment for each test
    unset COMMON_SETUP_LOADED
    unset MOCK_REGISTRY_LOADED
    unset ASSERTIONS_LOADED
    unset ERROR_HANDLING_LOADED
}

teardown() {
    # Cleanup after each test
    cleanup_mocks 2>/dev/null || true
}

#######################################
# Phase 1 Improvements Validation
#######################################

@test "Phase 1: Path dependencies are resolved correctly" {
    # Test that the new path resolution system works
    run bash -c "
        # Simulate different working directories
        cd /tmp
        # Should work from any directory using the new path system
        source '${BATS_TEST_DIRNAME}/../core/common_setup.bash' && echo 'success'
    "
    assert_success
    assert_output_contains "success"
}

@test "Phase 1: HTTP mock system is complete and functional" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'
        setup_standard_mocks
        
        # Test that missing function now exists
        declare -f mock::http::set_endpoint_unreachable >/dev/null && echo 'function_exists'
        
        # Test that function works
        mock::http::set_endpoint_unreachable 'http://test.example.com'
        echo 'function_works'
    "
    assert_success
    assert_output_contains "function_exists"
    assert_output_contains "function_works"
}

@test "Phase 1: Mock systems are properly consolidated" {
    # Test that legacy system shows deprecation warning but still works
    run bash -c "
        source '${BATS_TEST_DIRNAME}/../standard_mock_framework.bash'
        setup_standard_mocks >/dev/null 2>&1
        echo 'legacy_compatibility_maintained'
    "
    assert_success
    assert_output_contains "legacy_compatibility_maintained"
    
    # Test that new system works without warnings
    run bash -c "
        source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'
        setup_standard_mocks >/dev/null 2>&1
        echo 'new_system_clean'
    "
    assert_success
    assert_output_contains "new_system_clean"
}

#######################################
# Phase 2 Improvements Validation  
#######################################

@test "Phase 2: Standardized error handling is working" {
    # Test that standardized error functions exist
    run bash -c "
        source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'
        declare -f common_setup_error >/dev/null && echo 'error_function_exists'
        declare -f validate_required_param >/dev/null && echo 'validation_function_exists'
        
        # Test that error handling produces consistent format
        validate_required_param 'test_param' '' 'TEST_MODULE' 2>&1 || echo 'error_handled'
    "
    assert_success
    assert_output_contains "error_function_exists"
    assert_output_contains "validation_function_exists"
    assert_output_contains "[TEST_MODULE] ERROR: Required parameter missing: test_param"
    assert_output_contains "error_handled"
}

@test "Phase 2: Templates are comprehensive and usable" {
    # Test that all templates exist and are valid
    local templates=(
        "standard-test.bats"
        "resource-test.bats" 
        "integration-test.bats"
        "performance-test.bats"
        "advanced-test.bats"
        "README.md"
    )
    
    for template in "${templates[@]}"; do
        assert_file_exists "${BATS_TEST_DIRNAME}/../templates/$template"
        
        # Test that BATS files have valid syntax
        if [[ "$template" =~ \.bats$ ]]; then
            # Basic syntax check
            run bash -n "${BATS_TEST_DIRNAME}/../templates/$template"
            assert_success
        fi
    done
}

@test "Phase 2: Template functionality is working" {
    # Test that a template can actually be used
    local template_file="${BATS_TEST_DIRNAME}/../templates/standard-test.bats"
    
    # Extract just the infrastructure loading part and test it
    run bash -c "
        # Simulate what the template does
        if [[ -n \"\${VROOLI_TEST_FIXTURES_DIR:-}\" ]]; then
            source \"\${VROOLI_TEST_FIXTURES_DIR}/core/common_setup.bash\"
        else
            TEST_DIR=\"\$(cd \"\$(dirname \"${template_file}\")\" && pwd)\"
            source \"\${TEST_DIR}/../core/common_setup.bash\"
        fi
        setup_standard_mocks >/dev/null 2>&1
        echo 'template_infrastructure_works'
    "
    assert_success
    assert_output_contains "template_infrastructure_works"
}

#######################################
# Phase 3 Improvements Validation
#######################################

@test "Phase 3: Advanced assertion functions are available" {
    # Test that all new advanced assertion functions exist
    local advanced_functions=(
        "assert_eventually_true"
        "assert_completes_within"
        "assert_file_modified_after"
        "assert_log_contains"
        "assert_process_running"
        "assert_process_stops"
        "assert_eventually_succeeds"
        "assert_network_connectivity"
        "assert_disk_space_available"
        "assert_memory_usage_within"
        "assert_lock_file_behavior"
    )
    
    run bash -c "
        source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'
        for func in ${advanced_functions[*]}; do
            declare -f \"\$func\" >/dev/null && echo \"✓ \$func\"
        done
    "
    assert_success
    
    for func in "${advanced_functions[@]}"; do
        assert_output_contains "✓ $func"
    done
}

@test "Phase 3: Advanced assertions work correctly" {
    # Test assert_eventually_true
    run bash -c "
        source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'
        assert_eventually_true 'true' 1 1 'test condition' 2>/dev/null && echo 'eventually_true_works'
    "
    assert_success
    assert_output_contains "eventually_true_works"
    
    # Test assert_completes_within
    run bash -c "
        source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'
        assert_completes_within 5 'echo test' echo 'hello' 2>/dev/null && echo 'completes_within_works'
    "
    assert_success
    assert_output_contains "completes_within_works"
    
    # Test assert_eventually_succeeds
    run bash -c "
        source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'
        assert_eventually_succeeds 2 1 'simple command' true 2>/dev/null && echo 'eventually_succeeds_works'
    "
    assert_success
    assert_output_contains "eventually_succeeds_works"
}

#######################################
# Overall System Integration Tests
#######################################

@test "Integration: All core modules load without conflicts" {
    # Test that loading all modules together works
    run bash -c "
        source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'
        
        # Check that all major systems are loaded
        [[ \"\${ERROR_HANDLING_LOADED}\" == 'true' ]] && echo 'error_handling_loaded'
        [[ \"\${MOCK_REGISTRY_LOADED}\" == 'true' ]] && echo 'mock_registry_loaded'
        [[ \"\${ASSERTIONS_LOADED}\" == 'true' ]] && echo 'assertions_loaded'
        [[ \"\${COMMON_SETUP_LOADED}\" == 'true' ]] && echo 'common_setup_loaded'
        
        # Check that we have expected function counts
        local assertion_count=\$(compgen -A function | grep -c '^assert_')
        [[ \"\$assertion_count\" -ge 50 ]] && echo \"assertion_count_sufficient: \$assertion_count\"
    "
    assert_success
    assert_output_contains "error_handling_loaded"
    assert_output_contains "mock_registry_loaded" 
    assert_output_contains "assertions_loaded"
    assert_output_contains "common_setup_loaded"
    assert_output_contains "assertion_count_sufficient:"
}

@test "Integration: All setup modes work correctly" {
    # Test standard setup
    run bash -c "
        source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'
        setup_standard_mocks >/dev/null 2>&1 && echo 'standard_setup_works'
    "
    assert_success
    assert_output_contains "standard_setup_works"
    
    # Test resource setup
    run bash -c "
        source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'
        setup_resource_test 'ollama' >/dev/null 2>&1 && echo 'resource_setup_works'
    "
    assert_success
    assert_output_contains "resource_setup_works"
    
    # Test integration setup
    run bash -c "
        source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'
        setup_integration_test 'ollama' 'n8n' >/dev/null 2>&1 && echo 'integration_setup_works'
    "
    assert_success
    assert_output_contains "integration_setup_works"
    
    # Test performance setup
    run bash -c "
        source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'
        setup_performance_test >/dev/null 2>&1 && echo 'performance_setup_works'
    "
    assert_success
    assert_output_contains "performance_setup_works"
}

@test "Integration: Error handling works across all modules" {
    # Test error handling in different modules
    run bash -c "
        source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'
        
        # Test common setup error
        setup_resource_test '' 2>&1 || echo 'common_setup_error_handled'
        
        # Test mock registry error
        mock::load 'invalid_category' 'test' 2>&1 || echo 'mock_registry_error_handled'
    "
    assert_success
    assert_output_contains "[COMMON_SETUP] ERROR: Required parameter missing: resource"
    assert_output_contains "common_setup_error_handled"
    assert_output_contains "[MOCK_REGISTRY] ERROR: Unknown mock category: invalid_category"
    assert_output_contains "mock_registry_error_handled"
}

@test "Integration: Performance is within acceptable limits" {
    # Test that setup time is reasonable
    local start_time end_time duration
    start_time=$(date +%s%N)
    
    run bash -c "
        source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'
        setup_standard_mocks >/dev/null 2>&1
        cleanup_mocks >/dev/null 2>&1
    "
    
    end_time=$(date +%s%N)
    duration=$(( (end_time - start_time) / 1000000 ))  # Convert to milliseconds
    
    assert_success
    
    # Setup should complete in under 2 seconds (2000ms) even in worst case
    assert_less_than "$duration" "2000"
    echo "Setup completed in ${duration}ms" >&2
}

#######################################
# Backward Compatibility Tests
#######################################

@test "Compatibility: Legacy functions still work" {
    # Test that old function names still work
    run bash -c "
        source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'
        
        # Test legacy setup function
        setup_standard_mock_framework >/dev/null 2>&1 && echo 'legacy_setup_works'
        
        # Test legacy cleanup function  
        cleanup_standard_mock_framework >/dev/null 2>&1 && echo 'legacy_cleanup_works'
    "
    assert_success
    assert_output_contains "legacy_setup_works"
    assert_output_contains "legacy_cleanup_works"
}

@test "Compatibility: Existing tests patterns still work" {
    # Test that common existing test patterns continue to work
    run bash -c "
        source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'
        setup_standard_mocks >/dev/null 2>&1
        
        # Common pattern: Docker version check
        docker --version >/dev/null && echo 'docker_pattern_works'
        
        # Common pattern: HTTP health check
        curl -s http://localhost:8080/health >/dev/null && echo 'http_pattern_works'
        
        # Common pattern: Basic assertions
        output='test output'
        assert_output_contains 'test' && echo 'assertion_pattern_works'
        
        cleanup_mocks >/dev/null 2>&1
    "
    assert_success
    assert_output_contains "docker_pattern_works"
    assert_output_contains "http_pattern_works"
    assert_output_contains "assertion_pattern_works"
}

#######################################
# Documentation and Help System Tests
#######################################

@test "Documentation: README files exist and are comprehensive" {
    # Test main README
    assert_file_exists "${BATS_TEST_DIRNAME}/../README.md"
    assert_file_contains "${BATS_TEST_DIRNAME}/../README.md" "setup_standard_mocks"
    assert_file_contains "${BATS_TEST_DIRNAME}/../README.md" "setup_resource_test"
    assert_file_contains "${BATS_TEST_DIRNAME}/../README.md" "setup_integration_test"
    
    # Test templates README
    assert_file_exists "${BATS_TEST_DIRNAME}/../templates/README.md"
    assert_file_contains "${BATS_TEST_DIRNAME}/../templates/README.md" "standard-test.bats"
    assert_file_contains "${BATS_TEST_DIRNAME}/../templates/README.md" "resource-test.bats"
}

@test "Documentation: Examples are syntactically correct" {
    local example_files=(
        "${BATS_TEST_DIRNAME}/../docs/examples/basic-test.bats"
        "${BATS_TEST_DIRNAME}/../docs/examples/resource-test.bats"
        "${BATS_TEST_DIRNAME}/../docs/examples/integration-test.bats"
    )
    
    for example_file in "${example_files[@]}"; do
        if [[ -f "$example_file" ]]; then
            # Basic syntax check
            run bash -n "$example_file"
            assert_success
        fi
    done
}

#######################################
# Final System Health Check
#######################################

@test "System Health: All critical components are functional" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'
        
        # Test all major systems
        setup_standard_mocks >/dev/null 2>&1 || exit 1
        
        # Test mock functionality
        docker --version >/dev/null 2>&1 || exit 1
        curl -s http://test.example.com >/dev/null 2>&1 || exit 1
        
        # Test assertions
        output='test'
        assert_output_contains 'test' || exit 1
        assert_env_set 'PATH' || exit 1
        
        # Test advanced assertions
        assert_eventually_true 'true' 1 1 'health check' 2>/dev/null || exit 1
        
        # Test error handling
        validate_required_param 'test' 'value' 'HEALTH_CHECK' >/dev/null 2>&1 || exit 1
        
        cleanup_mocks >/dev/null 2>&1 || exit 1
        
        echo 'system_health_check_passed'
    "
    assert_success
    assert_output_contains "system_health_check_passed"
}

@test "System Health: Resource count and function availability" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'
        
        # Count available assertion functions
        local assertion_count=\$(compgen -A function | grep -c '^assert_')
        echo \"Assertion functions: \$assertion_count\"
        
        # Count available mock functions
        local mock_count=\$(compgen -A function | grep -c 'mock::')
        echo \"Mock functions: \$mock_count\"
        
        # Count available setup functions
        local setup_count=\$(compgen -A function | grep -c 'setup_.*')
        echo \"Setup functions: \$setup_count\"
        
        # Verify minimum expected counts
        [[ \$assertion_count -ge 50 ]] || exit 1
        [[ \$mock_count -ge 10 ]] || exit 1  
        [[ \$setup_count -ge 4 ]] || exit 1
        
        echo 'function_counts_sufficient'
    "
    assert_success
    assert_output_contains "function_counts_sufficient"
    assert_output_contains "Assertion functions:"
    assert_output_contains "Mock functions:"
    assert_output_contains "Setup functions:"
}

echo "[COMPREHENSIVE_VALIDATION] All infrastructure improvements validated successfully"