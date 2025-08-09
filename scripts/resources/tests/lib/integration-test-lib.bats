#!/usr/bin/env bats
# Test suite for integration-test-lib.sh

setup() {
    # Get the directory of the test file
    TEST_DIR="$(cd "$(dirname "$BATS_TEST_FILENAME")" && pwd)"
    
    # Source var.sh
    # shellcheck disable=SC1091
    source "$TEST_DIR/../../../../lib/utils/var.sh"
    
    # Source test helpers and mocks
    # shellcheck disable=SC1091
    source "$var_SCRIPTS_TEST_DIR/fixtures/setup.bash"
    # shellcheck disable=SC1091
    source "$var_SCRIPTS_TEST_DIR/fixtures/assertions.bash"
    # shellcheck disable=SC1091
    source "$var_SCRIPTS_TEST_DIR/fixtures/mocks/http.sh"
    
    # Source the library being tested
    # shellcheck disable=SC1091
    source "$TEST_DIR/integration-test-lib.sh"
    
    # Setup test environment
    export SERVICE_NAME="test-service"
    export BASE_URL="http://localhost:8080"
    export TIMEOUT="30"
    export VERBOSE="false"
    export HEALTH_ENDPOINT="/health"
    
    # Initialize arrays
    REQUIRED_TOOLS=("curl")
    SERVICE_METADATA=()
    TEST_TEMP_FILES=()
    REGISTERED_TESTS=()
    FAILED_TESTS=()
}

teardown() {
    # Cleanup any test artifacts
    integration_test_lib::cleanup_test_files
    
    # Unset environment variables
    unset SERVICE_NAME BASE_URL TIMEOUT VERBOSE HEALTH_ENDPOINT
    unset REQUIRED_TOOLS SERVICE_METADATA TEST_TEMP_FILES
    unset REGISTERED_TESTS FAILED_TESTS
    unset TESTS_PASSED TESTS_FAILED TESTS_SKIPPED
}

@test "integration_test_lib::init_config - sets default values" {
    unset SERVICE_NAME BASE_URL TIMEOUT VERBOSE
    
    integration_test_lib::init_config
    
    [[ "$SERVICE_NAME" == "unknown-service" ]]
    [[ "$BASE_URL" == "http://localhost:8080" ]]
    [[ "$TIMEOUT" == "30" ]]
    [[ "$VERBOSE" == "false" ]]
}

@test "integration_test_lib::log_test_result - tracks test results" {
    TESTS_PASSED=0
    TESTS_FAILED=0
    TESTS_SKIPPED=0
    FAILED_TESTS=()
    
    run integration_test_lib::log_test_result "test1" "PASS"
    assert_success
    [[ $TESTS_PASSED -eq 1 ]]
    
    run integration_test_lib::log_test_result "test2" "FAIL" "error message"
    assert_success
    [[ $TESTS_FAILED -eq 1 ]]
    [[ "${FAILED_TESTS[0]}" == "test2" ]]
    
    run integration_test_lib::log_test_result "test3" "SKIP"
    assert_success
    [[ $TESTS_SKIPPED -eq 1 ]]
}

@test "integration_test_lib::check_service_available - detects service status" {
    # Mock curl to succeed
    function curl() { return 0; }
    export -f curl
    
    run integration_test_lib::check_service_available "http://localhost:8080" 5
    assert_success
    
    # Mock curl to fail
    function curl() { return 1; }
    export -f curl
    
    run integration_test_lib::check_service_available "http://localhost:8080" 5
    assert_failure
    
    unset -f curl
}

@test "integration_test_lib::make_api_request - makes API calls safely" {
    # Mock curl
    function curl() {
        echo '{"status":"ok"}'
        return 0
    }
    export -f curl
    
    run integration_test_lib::make_api_request "/test" "GET" 10
    
    assert_success
    assert_output '{"status":"ok"}'
    
    unset -f curl
}

@test "integration_test_lib::check_required_tools - validates tool availability" {
    REQUIRED_TOOLS=("curl" "jq")
    
    # Mock command -v
    function command() {
        if [[ "$1" == "-v" ]]; then
            case "$2" in
                "curl") return 0 ;;
                "jq") return 1 ;;
            esac
        fi
    }
    export -f command
    
    run integration_test_lib::check_required_tools
    
    assert_failure
    assert_output --partial "jq"
    
    unset -f command
}

@test "integration_test_lib::test_file_upload - handles file uploads" {
    # Create test file
    local test_file="${BATS_TMPDIR}/test.txt"
    echo "test content" > "$test_file"
    
    # Mock curl
    function curl() {
        echo "200"
        return 0
    }
    export -f curl
    
    run integration_test_lib::test_file_upload "/upload" "$test_file"
    
    assert_success
    
    # Test with non-existent file
    run integration_test_lib::test_file_upload "/upload" "/nonexistent.txt"
    
    assert_failure
    
    rm -f "$test_file"
    unset -f curl
}

@test "integration_test_lib::poll_async_job - polls job status" {
    local call_count=0
    
    # Mock API request that succeeds after 2 attempts
    function integration_test_lib::make_api_request() {
        ((call_count++))
        if [[ $call_count -eq 2 ]]; then
            echo '{"status":"completed"}'
            return 0
        fi
        echo '{"status":"pending"}'
        return 0
    }
    export -f integration_test_lib::make_api_request
    
    # Mock sleep
    function sleep() { :; }
    export -f sleep
    
    run integration_test_lib::poll_async_job "/jobs" "123" 5 1
    
    assert_success
    
    unset -f integration_test_lib::make_api_request sleep
}

@test "integration_test_lib::wait_for_health - waits for service health" {
    local call_count=0
    
    # Mock service that becomes available after 2 attempts
    function integration_test_lib::check_service_available() {
        ((call_count++))
        [[ $call_count -ge 2 ]]
    }
    export -f integration_test_lib::check_service_available
    
    # Mock sleep
    function sleep() { :; }
    export -f sleep
    
    run integration_test_lib::wait_for_health 3 1
    
    assert_success
    
    unset -f integration_test_lib::check_service_available sleep
}

@test "integration_test_lib::create_test_file - creates temporary files" {
    TEST_TEMP_FILES=()
    
    local test_file
    test_file=$(integration_test_lib::create_test_file "test content" ".json")
    
    [[ -f "$test_file" ]]
    [[ $(cat "$test_file") == "test content" ]]
    [[ ${#TEST_TEMP_FILES[@]} -eq 1 ]]
    
    # Cleanup
    rm -f "$test_file"
}

@test "integration_test_lib::cleanup_test_files - removes temporary files" {
    TEST_TEMP_FILES=()
    
    # Create test files
    local file1="${BATS_TMPDIR}/test1.txt"
    local file2="${BATS_TMPDIR}/test2.txt"
    echo "test" > "$file1"
    echo "test" > "$file2"
    
    TEST_TEMP_FILES=("$file1" "$file2")
    
    integration_test_lib::cleanup_test_files
    
    [[ ! -f "$file1" ]]
    [[ ! -f "$file2" ]]
}

@test "integration_test_lib::validate_json_field - validates JSON fields" {
    local json='{"name":"test","value":42,"nested":{"key":"data"}}'
    
    run integration_test_lib::validate_json_field "$json" ".name" "test"
    assert_success
    
    run integration_test_lib::validate_json_field "$json" ".value" "42"
    assert_success
    
    run integration_test_lib::validate_json_field "$json" ".nested.key" "data"
    assert_success
    
    run integration_test_lib::validate_json_field "$json" ".missing" ""
    assert_failure
}

@test "integration_test_lib::register_tests - registers test functions" {
    REGISTERED_TESTS=()
    
    integration_test_lib::register_tests "test1" "test2" "test3"
    
    [[ ${#REGISTERED_TESTS[@]} -eq 3 ]]
    [[ "${REGISTERED_TESTS[0]}" == "test1" ]]
    [[ "${REGISTERED_TESTS[2]}" == "test3" ]]
}

@test "integration_test_lib::test_service_availability - checks service availability" {
    # Mock successful service check
    function integration_test_lib::check_service_available() { return 0; }
    export -f integration_test_lib::check_service_available
    
    TESTS_PASSED=0
    TESTS_FAILED=0
    
    run integration_test_lib::test_service_availability
    
    assert_success
    [[ $TESTS_PASSED -eq 1 ]]
    
    unset -f integration_test_lib::check_service_available
}

@test "integration_test_lib::register_standard_interface_tests - adds standard tests" {
    REGISTERED_TESTS=()
    
    integration_test_lib::register_standard_interface_tests
    
    [[ ${#REGISTERED_TESTS[@]} -gt 0 ]]
    # Should include standard tests
    [[ " ${REGISTERED_TESTS[*]} " =~ " test_manage_script_exists " ]]
}