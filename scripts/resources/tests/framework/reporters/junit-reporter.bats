#!/usr/bin/env bats
# ====================================================================
# Tests for JUnit XML Reporter - Layer 1 Validation CI/CD Integration
# ====================================================================

# Load test helpers
load "${BATS_TEST_DIRNAME}/../../../../__test/helpers/bats-support/load.bash"
load "${BATS_TEST_DIRNAME}/../../../../__test/helpers/bats-assert/load.bash"

# Define location for this test file
REPORTERS_DIR="$BATS_TEST_DIRNAME"

# shellcheck disable=SC1091
source "${REPORTERS_DIR}/../../../../lib/utils/var.sh"

# Path to the script under test
SCRIPT_PATH="${REPORTERS_DIR}/junit-reporter.sh"

# Test setup
setup() {
    # Source the script
    # shellcheck disable=SC1091
    source "$SCRIPT_PATH"
    
    # Initialize reporter for each test
    junit_reporter::init "TestSuite"
}

# Test teardown
teardown() {
    # Clean up any test artifacts
    unset JUNIT_TEST_SUITE_NAME JUNIT_TEST_CASES JUNIT_TOTAL_TESTS
    unset JUNIT_TOTAL_FAILURES JUNIT_TOTAL_ERRORS JUNIT_TOTAL_SKIPPED
    unset JUNIT_TOTAL_TIME JUNIT_SUITE_TIMESTAMP
}

@test "junit_reporter::init initializes variables correctly" {
    run bash -c "
        source '$SCRIPT_PATH'
        junit_reporter::init 'CustomSuite'
        echo \"Suite: \$JUNIT_TEST_SUITE_NAME\"
        echo \"Tests: \$JUNIT_TOTAL_TESTS\"
        echo \"Failures: \$JUNIT_TOTAL_FAILURES\"
        echo \"Errors: \$JUNIT_TOTAL_ERRORS\"
        echo \"Skipped: \$JUNIT_TOTAL_SKIPPED\"
        echo \"Time: \$JUNIT_TOTAL_TIME\"
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Suite: CustomSuite" ]]
    [[ "$output" =~ "Tests: 0" ]]
    [[ "$output" =~ "Failures: 0" ]]
    [[ "$output" =~ "Errors: 0" ]]
    [[ "$output" =~ "Skipped: 0" ]]
    [[ "$output" =~ "Time: 0" ]]
}

@test "junit_reporter::xml_escape handles special characters" {
    run bash -c "
        source '$SCRIPT_PATH'
        junit_reporter::xml_escape '<test>&\"value\"</test>' | tr -d ' '
    "
    [ "$status" -eq 0 ]
    [[ "$output" == "&lt;test&gt;&amp;&quot;value&quot;&lt;/test&gt;" ]]
}

@test "junit_reporter::xml_escape handles apostrophes" {
    run bash -c "
        source '$SCRIPT_PATH'
        junit_reporter::xml_escape \"It's a test\" | tr -d ' '
    "
    [ "$status" -eq 0 ]
    [[ "$output" == "It&#39;satest" ]]
}

@test "junit_reporter::report_resource_result updates counters for passed test" {
    run bash -c "
        source '$SCRIPT_PATH'
        junit_reporter::init
        junit_reporter::report_resource_result 'test_resource' 'passed' 'All good' 100
        echo \"Total tests: \$JUNIT_TOTAL_TESTS\"
        echo \"Total time: \$JUNIT_TOTAL_TIME\"
        echo \"Failures: \$JUNIT_TOTAL_FAILURES\"
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Total tests: 1" ]]
    [[ "$output" =~ "Total time: 100" ]]
    [[ "$output" =~ "Failures: 0" ]]
}

@test "junit_reporter::report_resource_result updates counters for failed test" {
    run bash -c "
        source '$SCRIPT_PATH'
        junit_reporter::init
        junit_reporter::report_resource_result 'test_resource' 'failed' 'Something wrong' 200
        echo \"Total tests: \$JUNIT_TOTAL_TESTS\"
        echo \"Failures: \$JUNIT_TOTAL_FAILURES\"
        echo \"Total time: \$JUNIT_TOTAL_TIME\"
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Total tests: 1" ]]
    [[ "$output" =~ "Failures: 1" ]]
    [[ "$output" =~ "Total time: 200" ]]
}

@test "junit_reporter::report_resource_result handles error status" {
    run bash -c "
        source '$SCRIPT_PATH'
        junit_reporter::init
        junit_reporter::report_resource_result 'test_resource' 'error' 'Critical error' 300
        echo \"Errors: \$JUNIT_TOTAL_ERRORS\"
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Errors: 1" ]]
}

@test "junit_reporter::report_resource_result handles skipped status" {
    run bash -c "
        source '$SCRIPT_PATH'
        junit_reporter::init
        junit_reporter::report_resource_result 'test_resource' 'skipped' 'Not applicable' 0
        echo \"Skipped: \$JUNIT_TOTAL_SKIPPED\"
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Skipped: 1" ]]
}

@test "junit_reporter::generate_test_case creates valid XML for passed test" {
    run bash -c "
        source '$SCRIPT_PATH'
        junit_reporter::generate_test_case 'my_resource' 'passed' 'Success' 150 'false'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "<testcase" ]]
    [[ "$output" =~ "name=\"syntax_validation\"" ]]
    [[ "$output" =~ "classname=\"vrooli.resources.my_resource\"" ]]
    [[ "$output" =~ "time=\"0.150\"" ]]
    [[ "$output" =~ "</testcase>" ]]
}

@test "junit_reporter::generate_test_case creates valid XML for failed test" {
    run bash -c "
        source '$SCRIPT_PATH'
        junit_reporter::generate_test_case 'my_resource' 'failed' 'Test failed' 250 'false'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "<failure" ]]
    [[ "$output" =~ "message=\"Resource validation failed\"" ]]
    [[ "$output" =~ "type=\"ValidationFailure\"" ]]
    [[ "$output" =~ "</failure>" ]]
}

@test "junit_reporter::generate_test_case handles cached results" {
    run bash -c "
        source '$SCRIPT_PATH'
        junit_reporter::generate_test_case 'my_resource' 'passed' 'Success' 50 'true'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "name=\"syntax_validation_cached\"" ]]
    [[ "$output" =~ "Result retrieved from cache" ]]
    [[ "$output" =~ "<property name=\"result.cached\" value=\"true\"/>" ]]
}

@test "junit_reporter::report_batch_summary creates summary test case" {
    run bash -c "
        source '$SCRIPT_PATH'
        junit_reporter::init
        junit_reporter::report_batch_summary 10 8 2 1500
        echo \"Total tests: \$JUNIT_TOTAL_TESTS\"
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Total tests: 1" ]]
}

@test "junit_reporter::validate_xml detects missing XML declaration" {
    run bash -c "
        source '$SCRIPT_PATH'
        echo '<testsuites></testsuites>' | junit_reporter::validate_xml
    "
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Missing XML declaration" ]]
}

@test "junit_reporter::validate_xml detects missing testsuites element" {
    run bash -c "
        source '$SCRIPT_PATH'
        echo '<?xml version=\"1.0\"?><testsuite></testsuite>' | junit_reporter::validate_xml
    "
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Missing testsuites root element" ]]
}

@test "junit_reporter::validate_xml accepts valid XML" {
    run bash -c "
        source '$SCRIPT_PATH'
        echo '<?xml version=\"1.0\"?><testsuites><testsuite></testsuite></testsuites>' | junit_reporter::validate_xml
    "
    [ "$status" -eq 0 ]
}

@test "junit_reporter::report_finalize generates complete valid XML" {
    run bash -c "
        source '$SCRIPT_PATH'
        junit_reporter::init 'TestSuite'
        junit_reporter::report_resource_result 'resource1' 'passed' 'OK' 100
        junit_reporter::report_resource_result 'resource2' 'failed' 'Error' 200
        output=\$(junit_reporter::report_finalize)
        echo \"\$output\" | head -1
    "
    [ "$status" -eq 0 ]
    [[ "$output" == '<?xml version="1.0" encoding="UTF-8"?>' ]]
}

@test "junit_reporter::report_finalize includes test statistics" {
    run bash -c "
        source '$SCRIPT_PATH'
        junit_reporter::init
        junit_reporter::report_resource_result 'r1' 'passed' 'OK' 100
        junit_reporter::report_resource_result 'r2' 'failed' 'Error' 200
        junit_reporter::report_resource_result 'r3' 'skipped' 'Skip' 0
        output=\$(junit_reporter::report_finalize)
        echo \"\$output\" | grep 'tests=\"3\"'
        echo \"\$output\" | grep 'failures=\"1\"'
        echo \"\$output\" | grep 'skipped=\"1\"'
    "
    [ "$status" -eq 0 ]
}

@test "junit_reporter functions are exported" {
    run bash -c "
        source '$SCRIPT_PATH'
        declare -F | grep 'junit_reporter::' | wc -l
    "
    [ "$status" -eq 0 ]
    # Should have at least 9 exported functions
    [ "$output" -ge 9 ]
}