#!/usr/bin/env bash
# ====================================================================
# JUnit XML Reporter - Layer 1 Validation CI/CD Integration
# ====================================================================
#
# Generates JUnit XML output for seamless integration with CI/CD pipelines,
# test frameworks, and development tools that support JUnit format.
#
# Usage:
#   source junit-reporter.sh
#   junit_reporter::init "test_suite_name"
#   junit_reporter::report_resource_result "resource_name" "status" "details" "duration_ms"
#   junit_reporter::report_finalize > output.xml
#
# Output Format:
#   Standard JUnit XML compatible with:
#   - GitHub Actions
#   - Jenkins
#   - GitLab CI
#   - Azure DevOps
#   - Most IDEs and test runners
#
# ====================================================================

set -euo pipefail

_HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${_HERE}/../../../../lib/utils/var.sh"

# JUnit XML components
JUNIT_TEST_SUITE_NAME=""
JUNIT_TEST_CASES=()
JUNIT_TOTAL_TESTS=0
JUNIT_TOTAL_FAILURES=0
JUNIT_TOTAL_ERRORS=0
JUNIT_TOTAL_SKIPPED=0
JUNIT_TOTAL_TIME=0
JUNIT_SUITE_TIMESTAMP=""

#######################################
# Initialize JUnit reporter
# Arguments: $1 - test suite name (optional, defaults to "ResourceValidation")
# Returns: 0 on success
#######################################
junit_reporter::init() {
    local suite_name="${1:-ResourceValidation}"
    
    JUNIT_TEST_SUITE_NAME="$suite_name"
    JUNIT_TEST_CASES=()
    JUNIT_TOTAL_TESTS=0
    JUNIT_TOTAL_FAILURES=0
    JUNIT_TOTAL_ERRORS=0
    JUNIT_TOTAL_SKIPPED=0
    JUNIT_TOTAL_TIME=0
    JUNIT_SUITE_TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%S")
    
    return 0
}

#######################################
# Escape XML special characters
# Arguments: $1 - text to escape
# Returns: 0, outputs escaped text
#######################################
junit_reporter::xml_escape() {
    local text="$1"
    
    # Escape XML special characters using sed for reliable escaping
    text=$(echo "$text" | sed -e 's/&/\&amp;/g' \
                              -e 's/</\&lt;/g' \
                              -e 's/>/\&gt;/g' \
                              -e 's/"/\&quot;/g' \
                              -e "s/'/\&#39;/g")
    
    # Remove or replace problematic control characters
    text=$(echo "$text" | tr -d '\000-\010\013\014\016-\037' | tr '\011\012\015' '   ')
    
    echo "$text"
}

#######################################
# Generate JUnit test case XML
# Arguments: $1 - resource name, $2 - status, $3 - details, $4 - duration_ms, $5 - cached
# Returns: 0, outputs XML test case element
#######################################
junit_reporter::generate_test_case() {
    local resource_name="$1"
    local status="$2"
    local details="$3"
    local duration_ms="$4"
    local cached="${5:-false}"
    
    # Convert duration to seconds (JUnit expects seconds with decimal)
    local duration_seconds
    if command -v bc >/dev/null 2>&1; then
        duration_seconds=$(echo "scale=3; $duration_ms / 1000" | bc 2>/dev/null)
        # bc might output .150 instead of 0.150, so fix that
        if [[ "$duration_seconds" =~ ^\. ]]; then
            duration_seconds="0$duration_seconds"
        fi
    else
        # Fallback if bc is not available
        duration_seconds="$(( duration_ms / 1000 )).$(( (duration_ms % 1000) / 10 ))"
    fi
    
    # Escape XML content
    local escaped_resource_name
    escaped_resource_name=$(junit_reporter::xml_escape "$resource_name")
    local escaped_details
    escaped_details=$(junit_reporter::xml_escape "$details")
    
    # Create test case classname (resource category + resource name)
    local classname="vrooli.resources.$resource_name"
    local testname="syntax_validation"
    
    # Add cache information to test name if applicable
    if [[ "$cached" == "true" ]]; then
        testname="syntax_validation_cached"
    fi
    
    echo "    <testcase"
    echo "        name=\"$testname\""
    echo "        classname=\"$classname\""
    echo "        time=\"$duration_seconds\">"
    
    case "$status" in
        "passed")
            # Successful test case - no additional content needed
            if [[ "$cached" == "true" ]]; then
                echo "        <system-out><![CDATA[Result retrieved from cache]]></system-out>"
            fi
            ;;
        "failed")
            # Failed test case with failure details
            echo "        <failure"
            echo "            message=\"Resource validation failed\""
            echo "            type=\"ValidationFailure\"><![CDATA["
            echo "$escaped_details"
            echo "        ]]></failure>"
            
            # Add system output for additional context
            echo "        <system-err><![CDATA["
            echo "Resource: $escaped_resource_name"
            echo "Status: Failed"
            echo "Duration: ${duration_ms}ms"
            if [[ "$cached" == "true" ]]; then
                echo "Source: Cache hit"
            else
                echo "Source: Fresh validation"
            fi
            echo "        ]]></system-err>"
            ;;
        "skipped")
            # Skipped test case
            echo "        <skipped message=\"Resource validation skipped\"><![CDATA["
            echo "$escaped_details"
            echo "        ]]></skipped>"
            ;;
        "error")
            # Error in test execution
            echo "        <error"
            echo "            message=\"Validation error occurred\""
            echo "            type=\"ValidationError\"><![CDATA["
            echo "$escaped_details"
            echo "        ]]></error>"
            ;;
    esac
    
    # Add properties for additional metadata
    echo "        <properties>"
    echo "            <property name=\"resource.name\" value=\"$escaped_resource_name\"/>"
    echo "            <property name=\"validation.layer\" value=\"1\"/>"
    echo "            <property name=\"validation.type\" value=\"syntax\"/>"
    echo "            <property name=\"result.cached\" value=\"$cached\"/>"
    echo "            <property name=\"duration.ms\" value=\"$duration_ms\"/>"
    echo "        </properties>"
    
    echo "    </testcase>"
}

#######################################
# Report individual resource validation result
# Arguments: $1 - resource name, $2 - status, $3 - details, $4 - duration_ms, $5 - cached (optional)
# Returns: 0
#######################################
junit_reporter::report_resource_result() {
    local resource_name="$1"
    local status="$2"
    local details="$3"
    local duration_ms="$4"
    local cached="${5:-false}"
    
    # Generate test case XML and store it
    local test_case_xml
    test_case_xml=$(junit_reporter::generate_test_case "$resource_name" "$status" "$details" "$duration_ms" "$cached")
    JUNIT_TEST_CASES+=("$test_case_xml")
    
    # Update totals
    JUNIT_TOTAL_TESTS=$((JUNIT_TOTAL_TESTS + 1))
    JUNIT_TOTAL_TIME=$((JUNIT_TOTAL_TIME + duration_ms))
    
    case "$status" in
        "failed")
            JUNIT_TOTAL_FAILURES=$((JUNIT_TOTAL_FAILURES + 1))
            ;;
        "error")
            JUNIT_TOTAL_ERRORS=$((JUNIT_TOTAL_ERRORS + 1))
            ;;
        "skipped")
            JUNIT_TOTAL_SKIPPED=$((JUNIT_TOTAL_SKIPPED + 1))
            ;;
    esac
    
    return 0
}

#######################################
# Add system properties to the test suite
# Arguments: $1 - cache stats JSON (optional)
# Returns: 0, outputs XML properties
#######################################
junit_reporter::generate_system_properties() {
    local cache_stats_json="${1:-}"
    
    echo "    <properties>"
    echo "        <property name=\"validation.framework\" value=\"Vrooli Layer 1 Syntax Validation\"/>"
    echo "        <property name=\"validation.version\" value=\"1.0\"/>"
    echo "        <property name=\"timestamp\" value=\"$JUNIT_SUITE_TIMESTAMP\"/>"
    echo "        <property name=\"hostname\" value=\"$(hostname 2>/dev/null || echo 'unknown')\"/>"
    echo "        <property name=\"os.name\" value=\"$(uname -s 2>/dev/null || echo 'unknown')\"/>"
    echo "        <property name=\"os.version\" value=\"$(uname -r 2>/dev/null || echo 'unknown')\"/>"
    
    # Add cache statistics if provided
    if [[ -n "$cache_stats_json" ]]; then
        local cache_hits cache_misses hit_rate total_entries
        cache_hits=$(echo "$cache_stats_json" | grep -o '"cache_hits":[[:space:]]*[0-9]*' | cut -d: -f2 | tr -d ' ' 2>/dev/null || echo "0")
        cache_misses=$(echo "$cache_stats_json" | grep -o '"cache_misses":[[:space:]]*[0-9]*' | cut -d: -f2 | tr -d ' ' 2>/dev/null || echo "0")
        hit_rate=$(echo "$cache_stats_json" | grep -o '"hit_rate_percent":[[:space:]]*[0-9]*' | cut -d: -f2 | tr -d ' ' 2>/dev/null || echo "0")
        total_entries=$(echo "$cache_stats_json" | grep -o '"total_entries":[[:space:]]*[0-9]*' | cut -d: -f2 | tr -d ' ' 2>/dev/null || echo "0")
        
        echo "        <property name=\"cache.hits\" value=\"$cache_hits\"/>"
        echo "        <property name=\"cache.misses\" value=\"$cache_misses\"/>"
        echo "        <property name=\"cache.hit_rate\" value=\"$hit_rate\"/>"
        echo "        <property name=\"cache.total_entries\" value=\"$total_entries\"/>"
    fi
    
    echo "    </properties>"
}

#######################################
# Generate system-out content with summary information
# Arguments: None
# Returns: 0, outputs XML system-out content
#######################################
junit_reporter::generate_system_out() {
    echo "    <system-out><![CDATA["
    echo "Vrooli Resource Interface Validation"
    echo "Generated: $(date)"
    echo ""
    echo "Summary:"
    echo "  Total resources: $JUNIT_TOTAL_TESTS"
    echo "  Passed: $((JUNIT_TOTAL_TESTS - JUNIT_TOTAL_FAILURES - JUNIT_TOTAL_ERRORS - JUNIT_TOTAL_SKIPPED))"
    echo "  Failed: $JUNIT_TOTAL_FAILURES"
    echo "  Errors: $JUNIT_TOTAL_ERRORS"
    echo "  Skipped: $JUNIT_TOTAL_SKIPPED"
    echo "  Total time: ${JUNIT_TOTAL_TIME}ms"
    if [[ $JUNIT_TOTAL_TESTS -gt 0 ]]; then
        echo "  Average time: $((JUNIT_TOTAL_TIME / JUNIT_TOTAL_TESTS))ms per resource"
    fi
    echo ""
    echo "For detailed information, see the individual test cases above."
    echo "    ]]></system-out>"
}

#######################################
# Finalize and output complete JUnit XML
# Arguments: $1 - cache stats JSON (optional)
# Returns: 0, outputs complete JUnit XML to stdout
#######################################
junit_reporter::report_finalize() {
    local cache_stats_json="${1:-}"
    
    # Calculate total time in seconds
    local total_time_seconds
    if command -v bc >/dev/null 2>&1; then
        total_time_seconds=$(echo "scale=3; $JUNIT_TOTAL_TIME / 1000" | bc 2>/dev/null)
        # bc might output .150 instead of 0.150, so fix that
        if [[ "$total_time_seconds" =~ ^\. ]]; then
            total_time_seconds="0$total_time_seconds"
        fi
    else
        # Fallback if bc is not available
        total_time_seconds="$(( JUNIT_TOTAL_TIME / 1000 )).$(( (JUNIT_TOTAL_TIME % 1000) / 10 ))"
    fi
    
    # Output XML header
    echo '<?xml version="1.0" encoding="UTF-8"?>'
    echo '<testsuites>'
    
    # Test suite opening tag with summary statistics
    echo "    <testsuite"
    echo "        name=\"$(junit_reporter::xml_escape "$JUNIT_TEST_SUITE_NAME")\""
    echo "        tests=\"$JUNIT_TOTAL_TESTS\""
    echo "        failures=\"$JUNIT_TOTAL_FAILURES\""
    echo "        errors=\"$JUNIT_TOTAL_ERRORS\""
    echo "        skipped=\"$JUNIT_TOTAL_SKIPPED\""
    echo "        time=\"$total_time_seconds\""
    echo "        timestamp=\"$JUNIT_SUITE_TIMESTAMP\">"
    
    # System properties
    junit_reporter::generate_system_properties "$cache_stats_json"
    
    # Output all test cases
    for test_case in "${JUNIT_TEST_CASES[@]}"; do
        echo "$test_case"
    done
    
    # System output
    junit_reporter::generate_system_out
    
    # Close test suite and test suites
    echo "    </testsuite>"
    echo "</testsuites>"
    
    return 0
}

#######################################
# Create a test case for a batch summary
# Arguments: $1 - total, $2 - passed, $3 - failed, $4 - duration_ms
# Returns: 0
#######################################
junit_reporter::report_batch_summary() {
    local total="$1"
    local passed="$2"
    local failed="$3"
    local duration_ms="$4"
    
    local status="passed"
    local details="Batch validation completed: $passed passed, $failed failed out of $total resources"
    
    if [[ $failed -gt 0 ]]; then
        status="failed"
        details="Batch validation failed: $failed out of $total resources failed validation"
    fi
    
    junit_reporter::report_resource_result "batch_summary" "$status" "$details" "$duration_ms" "false"
    
    return 0
}

#######################################
# Validate JUnit XML output (basic validation)
# Arguments: $1 - XML content (via stdin if not provided)
# Returns: 0 if valid, 1 if invalid
#######################################
junit_reporter::validate_xml() {
    local xml_content="${1:-}"
    
    if [[ -z "$xml_content" ]]; then
        xml_content=$(cat)
    fi
    
    # Basic validation checks
    if ! echo "$xml_content" | grep -q '<?xml version="1.0"'; then
        echo "ERROR: Missing XML declaration" >&2
        return 1
    fi
    
    if ! echo "$xml_content" | grep -q '<testsuites>'; then
        echo "ERROR: Missing testsuites root element" >&2
        return 1
    fi
    
    if ! echo "$xml_content" | grep -q '</testsuites>'; then
        echo "ERROR: Missing closing testsuites tag" >&2
        return 1
    fi
    
    # Check for balanced tags
    local testsuite_open testsuite_close
    testsuite_open=$(echo "$xml_content" | grep -c '<testsuite' || echo 0)
    testsuite_close=$(echo "$xml_content" | grep -c '</testsuite>' || echo 0)
    
    if [[ $testsuite_open -ne $testsuite_close ]]; then
        echo "ERROR: Unbalanced testsuite tags (open: $testsuite_open, close: $testsuite_close)" >&2
        return 1
    fi
    
    return 0
}

# Export functions for use in other scripts
export -f junit_reporter::init junit_reporter::xml_escape junit_reporter::generate_test_case
export -f junit_reporter::report_resource_result junit_reporter::generate_system_properties
export -f junit_reporter::generate_system_out junit_reporter::report_finalize junit_reporter::report_batch_summary
export -f junit_reporter::validate_xml