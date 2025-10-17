#!/usr/bin/env bats
# Test suite for enhanced-integration-test-lib.sh

setup() {
    # Get the directory of the test file
    TEST_DIR="$(builtin cd "${BATS_TEST_FILENAME%/*}" && builtin pwd)"
    
    # Source var.sh
    # shellcheck disable=SC1091
    source "$TEST_DIR/../../../../lib/utils/var.sh"
    
    # Source test helpers and mocks
    # shellcheck disable=SC1091
    source "$var_TEST_DIR/fixtures/setup.bash"
    # shellcheck disable=SC1091
    source "$var_TEST_DIR/fixtures/assertions.bash"
    
    # Source the library being tested
    # shellcheck disable=SC1091
    source "$TEST_DIR/enhanced-integration-test-lib.sh"
    
    # Setup test environment
    export SERVICE_NAME="test-service"
    export BASE_URL="http://localhost:8080"
    export TIMEOUT="30"
    export VERBOSE="false"
    export HEALTH_ENDPOINT="/health"
    export INTEGRATION_MODE="basic"
    export FIXTURE_TESTING="disabled"
    
    # Initialize test tracking variables
    TESTS_PASSED=0
    TESTS_FAILED=0
    TESTS_SKIPPED=0
    FAILED_TESTS=()
    REGISTERED_TESTS=()
}

teardown() {
    # Cleanup any test artifacts
    unset SERVICE_NAME BASE_URL TIMEOUT VERBOSE HEALTH_ENDPOINT
    unset INTEGRATION_MODE FIXTURE_TESTING
    unset TESTS_PASSED TESTS_FAILED TESTS_SKIPPED
    unset FAILED_TESTS REGISTERED_TESTS
}

@test "enhanced_test_lib::test_with_fixture - skips when fixtures disabled" {
    FIXTURE_TESTING="disabled"
    
    run enhanced_test_lib::test_with_fixture "test" "category" "fixture" "test_function"
    
    # Should return exit code 2 for skip
    [ "$status" -eq 2 ]
}

@test "enhanced_test_lib::enhanced_service_health_check - detects service down" {
    # Mock curl to fail
    function curl() { return 1; }
    export -f curl
    
    run enhanced_test_lib::enhanced_service_health_check
    
    assert_failure
    unset -f curl
}

@test "enhanced_test_lib::test_api_content_types - checks multiple content types" {
    # Mock successful API request
    function integration_test_lib::make_api_request() {
        echo '{"status":"ok"}'
        return 0
    }
    export -f integration_test_lib::make_api_request
    
    run enhanced_test_lib::test_api_content_types "/api/test"
    
    assert_success
    unset -f integration_test_lib::make_api_request
}

@test "enhanced_test_lib::test_service_load - handles concurrent requests" {
    # Mock API request function
    function integration_test_lib::make_api_request() { return 0; }
    export -f integration_test_lib::make_api_request
    
    run enhanced_test_lib::test_service_load "/health" 2 4
    
    assert_success
    unset -f integration_test_lib::make_api_request
}

@test "enhanced_test_lib::test_service_configuration - skips when no config endpoint" {
    # Mock API request to fail for all config endpoints
    function integration_test_lib::make_api_request() { return 1; }
    export -f integration_test_lib::make_api_request
    
    run enhanced_test_lib::test_service_configuration
    
    # Should return exit code 2 for skip
    [ "$status" -eq 2 ]
    unset -f integration_test_lib::make_api_request
}

@test "enhanced_test_lib::register_enhanced_ai_tests - adds tests to registry" {
    REGISTERED_TESTS=()
    
    enhanced_test_lib::register_enhanced_ai_tests
    
    [[ ${#REGISTERED_TESTS[@]} -gt 0 ]]
}

@test "enhanced_test_lib::test_ai_model_availability - checks model endpoints" {
    # Mock successful model list response
    function integration_test_lib::make_api_request() {
        if [[ "$1" == "/api/tags" ]]; then
            echo '{"models": ["model1", "model2"]}'
            return 0
        fi
        return 1
    }
    export -f integration_test_lib::make_api_request
    
    run enhanced_test_lib::test_ai_model_availability
    
    assert_success
    unset -f integration_test_lib::make_api_request
}

@test "enhanced_test_lib::generate_fixture_report - generates report with fixtures" {
    PROCESSED_FIXTURES=("test/fixture1" "test/fixture2")
    FIXTURE_RESULTS=("PASS:test/fixture1" "FAIL:test/fixture2")
    VERBOSE="false"
    
    run enhanced_test_lib::generate_fixture_report
    
    assert_success
    assert_output --partial "Total fixtures processed: 2"
}

@test "enhanced_test_lib::discover_resource_fixtures - returns correct fixtures for AI" {
    run enhanced_test_lib::discover_resource_fixtures "ollama" "ai"
    
    assert_success
    assert_output --partial "documents/llm-prompt.json"
}

@test "enhanced_test_lib::rotate_fixtures - rotates fixture selection" {
    FIXTURES_AVAILABLE="true"
    
    # Mock fixture_get_all function
    function fixture_get_all() {
        echo "fixture1"
        echo "fixture2"
        echo "fixture3"
    }
    export -f fixture_get_all
    
    # Mock shuf command
    function shuf() { cat; }
    export -f shuf
    
    run enhanced_test_lib::rotate_fixtures "test-category" 2
    
    assert_success
    [[ $(echo "$output" | wc -l) -le 2 ]]
    
    unset -f fixture_get_all shuf
}