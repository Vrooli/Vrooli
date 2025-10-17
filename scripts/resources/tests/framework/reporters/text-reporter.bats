#!/usr/bin/env bats
# ====================================================================
# Tests for Text Reporter - Layer 1 Validation Human-Readable Output
# ====================================================================

# Load test helpers
load "${BATS_TEST_DIRNAME}/../../../../__test/helpers/bats-support/load.bash"
load "${BATS_TEST_DIRNAME}/../../../../__test/helpers/bats-assert/load.bash"

# Define location for this test file
REPORTERS_DIR="$BATS_TEST_DIRNAME"

# shellcheck disable=SC1091
source "${REPORTERS_DIR}/../../../../lib/utils/var.sh"

# Path to the script under test
SCRIPT_PATH="${REPORTERS_DIR}/text-reporter.sh"

# Test setup
setup() {
    # Source the script
    # shellcheck disable=SC1091
    source "$SCRIPT_PATH"
    
    # Initialize reporter for each test
    text_reporter::init
}

# Test teardown
teardown() {
    # Clean up any test artifacts
    unset TEXT_REPORTER_INITIALIZED TEXT_REPORTER_RESOURCE_COUNT 
    unset TEXT_REPORTER_FAILED_RESOURCES
}

@test "text_reporter::init initializes variables correctly" {
    run bash -c "
        source '$SCRIPT_PATH'
        text_reporter::init
        echo \"Initialized: \$TEXT_REPORTER_INITIALIZED\"
        echo \"Count: \$TEXT_REPORTER_RESOURCE_COUNT\"
        echo \"Failed count: \${#TEXT_REPORTER_FAILED_RESOURCES[@]}\"
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Initialized: true" ]]
    [[ "$output" =~ "Count: 0" ]]
    [[ "$output" =~ "Failed count: 0" ]]
}

@test "text_reporter::report_header handles major style" {
    run bash -c "
        source '$SCRIPT_PATH'
        text_reporter::report_header 'Test Header' 'major' | grep -c '='
    "
    [ "$status" -eq 0 ]
    # Should have separator lines with equals signs
    [ "$output" -ge 2 ]
}

@test "text_reporter::report_header handles minor style" {
    run bash -c "
        source '$SCRIPT_PATH'
        text_reporter::report_header 'Minor Header' 'minor'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "--- Minor Header ---" ]]
}

@test "text_reporter::report_header handles subsection style" {
    run bash -c "
        source '$SCRIPT_PATH'
        text_reporter::report_header 'Subsection' 'subsection'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Subsection" ]]
}

@test "text_reporter::report_resource_result handles passed status" {
    run bash -c "
        source '$SCRIPT_PATH'
        text_reporter::init
        text_reporter::report_resource_result 'test_resource' 'passed' 'All validations passed' 150 'false'
        echo \"Count: \$TEXT_REPORTER_RESOURCE_COUNT\"
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "test_resource" ]]
    [[ "$output" =~ "150ms" ]]
    [[ "$output" =~ "Count: 1" ]]
}

@test "text_reporter::report_resource_result handles failed status" {
    run bash -c "
        source '$SCRIPT_PATH'
        text_reporter::init
        text_reporter::report_resource_result 'failed_resource' 'failed' 'Missing required action: install' 200 'false'
        echo \"Failed resources: \${TEXT_REPORTER_FAILED_RESOURCES[@]}\"
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "failed_resource" ]]
    [[ "$output" =~ "Issues found:" ]]
    [[ "$output" =~ "Failed resources: failed_resource" ]]
}

@test "text_reporter::report_resource_result handles warning status" {
    run bash -c "
        source '$SCRIPT_PATH'
        text_reporter::init
        text_reporter::report_resource_result 'warn_resource' 'warning' 'Minor issue detected' 100 'false'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "warn_resource" ]]
    [[ "$output" =~ "Minor issue detected" ]]
}

@test "text_reporter::report_resource_result formats duration correctly for ms" {
    run bash -c "
        source '$SCRIPT_PATH'
        text_reporter::init
        text_reporter::report_resource_result 'fast_resource' 'passed' '' 850 'false'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "850ms" ]]
}

@test "text_reporter::report_resource_result formats duration correctly for seconds" {
    run bash -c "
        source '$SCRIPT_PATH'
        text_reporter::init
        text_reporter::report_resource_result 'slow_resource' 'passed' '' 2500 'false'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "2.5s" ]]
}

@test "text_reporter::report_resource_result shows cache indicator" {
    run bash -c "
        source '$SCRIPT_PATH'
        text_reporter::init
        # Override cache mark for testing
        export TEXT_REPORTER_CACHE_MARK='[CACHE]'
        text_reporter::report_resource_result 'cached_resource' 'passed' '' 50 'true'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "[CACHE]" ]]
}

@test "text_reporter::report_error_details provides fix for missing install action" {
    run bash -c "
        source '$SCRIPT_PATH'
        text_reporter::report_error_details 'test' 'Missing required action: install'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Missing required action: install" ]]
    [[ "$output" =~ "\"install\") install_resource;;" ]]
}

@test "text_reporter::report_error_details provides fix for missing help pattern" {
    run bash -c "
        source '$SCRIPT_PATH'
        text_reporter::report_error_details 'test' 'Missing help pattern'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Help patterns not implemented" ]]
    [[ "$output" =~ "--help, -h, --version" ]]
}

@test "text_reporter::report_error_details provides fix for missing error handling" {
    run bash -c "
        source '$SCRIPT_PATH'
        text_reporter::report_error_details 'test' 'Missing error handling'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Error handling patterns missing" ]]
    [[ "$output" =~ "set -euo pipefail" ]]
    [[ "$output" =~ "trap cleanup EXIT" ]]
}

@test "text_reporter::report_summary shows correct statistics" {
    run bash -c "
        source '$SCRIPT_PATH'
        text_reporter::init
        TEXT_REPORTER_FAILED_RESOURCES=('res1' 'res2')
        text_reporter::report_summary 10 7 3 5
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Resources Validated: 10" ]]
    [[ "$output" =~ "Passed: 7" ]]
    [[ "$output" =~ "Failed: 3" ]]
    [[ "$output" =~ "Duration: 5s" ]]
    [[ "$output" =~ "Success Rate: 70%" ]]
}

@test "text_reporter::report_summary calculates average time" {
    run bash -c "
        source '$SCRIPT_PATH'
        text_reporter::init
        text_reporter::report_summary 4 4 0 8
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Average: 2000ms per resource" ]]
}

@test "text_reporter::report_recommendations provides useful suggestions" {
    run bash -c "
        source '$SCRIPT_PATH'
        text_reporter::report_recommendations
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Quick Fixes:" ]]
    [[ "$output" =~ "Common Patterns:" ]]
    [[ "$output" =~ "Documentation:" ]]
    [[ "$output" =~ "validate-interfaces.sh" ]]
}

@test "text_reporter::report_cache_stats parses JSON correctly" {
    run bash -c "
        source '$SCRIPT_PATH'
        json='{\"cache_hits\": 5, \"cache_misses\": 2, \"hit_rate_percent\": 71, \"total_entries\": 7, \"cache_size_kb\": 15}'
        text_reporter::report_cache_stats \"\$json\"
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Cache Hits: 5" ]]
    [[ "$output" =~ "Cache Misses: 2" ]]
    [[ "$output" =~ "Hit Rate: 71%" ]]
    [[ "$output" =~ "Cached Entries: 7" ]]
    [[ "$output" =~ "Cache Size: 15KB" ]]
}

@test "text_reporter::report_cache_stats shows excellent performance message" {
    run bash -c "
        source '$SCRIPT_PATH'
        # Override trophy mark for testing
        export TEXT_REPORTER_TROPHY_MARK='[TROPHY]'
        json='{\"cache_hits\": 90, \"cache_misses\": 10, \"hit_rate_percent\": 90}'
        text_reporter::report_cache_stats \"\$json\"
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "[TROPHY] Excellent cache performance!" ]]
}

@test "text_reporter::report_completion shows success message when no failures" {
    run bash -c "
        source '$SCRIPT_PATH'
        # Override trophy mark for testing
        export TEXT_REPORTER_TROPHY_MARK='[TROPHY]'
        export TEXT_REPORTER_ROCKET_MARK='[ROCKET]'
        text_reporter::report_completion 0
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "[TROPHY] All Resources Pass Validation!" ]]
    [[ "$output" =~ "[ROCKET]" ]]
    [[ "$output" =~ "Congratulations!" ]]
}

@test "text_reporter::report_completion shows failure message when failures exist" {
    run bash -c "
        source '$SCRIPT_PATH'
        # Override warning mark for testing
        export TEXT_REPORTER_WARNING_MARK='[WARN]'
        export TEXT_REPORTER_CROSS_MARK='[FAIL]'
        text_reporter::report_completion 3
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "[WARN] Validation Issues Found" ]]
    [[ "$output" =~ "[FAIL]" ]]
    [[ "$output" =~ "3 resource(s)" ]]
}

@test "text_reporter::report_progress shows progress bar" {
    run bash -c "
        source '$SCRIPT_PATH'
        text_reporter::report_progress 5 10
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "50% (5/10)" ]]
}

@test "text_reporter functions are exported" {
    run bash -c "
        source '$SCRIPT_PATH'
        declare -F | grep 'text_reporter::' | wc -l
    "
    [ "$status" -eq 0 ]
    # Should have at least 9 exported functions
    [ "$output" -ge 9 ]
}

@test "text_reporter handles non-TTY environment" {
    run bash -c "
        # Simulate non-TTY
        exec < /dev/null
        source '$SCRIPT_PATH'
        # Colors should be empty in non-TTY
        [[ -z \"\$TEXT_REPORTER_RED\" ]] && echo 'No colors'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "No colors" ]]
}

@test "text_reporter handles non-UTF8 environment" {
    run bash -c "
        export LANG=C
        export LC_ALL=C
        source '$SCRIPT_PATH'
        # Should use ASCII fallbacks
        [[ \"\$TEXT_REPORTER_CHECK_MARK\" == '[PASS]' ]] && echo 'ASCII mode'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "ASCII mode" ]]
}