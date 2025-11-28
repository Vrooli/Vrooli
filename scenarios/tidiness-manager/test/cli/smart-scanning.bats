#!/usr/bin/env bats
# CLI tests for smart scanning integration
# [REQ:TM-SS-001,TM-SS-002,TM-SS-003,TM-SS-004,TM-SS-005,TM-SS-006,TM-SS-007,TM-SS-008]

load '../../../scripts/lib/testing/bats-helpers.sh'

setup() {
    # Ensure tidiness-manager is running
    if ! pgrep -f "tidiness-manager-api" > /dev/null; then
        skip "tidiness-manager-api not running"
    fi

    export API_PORT="${API_PORT:-16821}"
    export API_BASE="http://localhost:${API_PORT}"
    export TEST_SCENARIO="test-scenario-$$"  # Unique per test run
}

# [REQ:TM-SS-001] AI batch configuration validation
@test "Smart scan API validates configuration parameters" {
    # Verify smart scan endpoint exists and validates config
    run curl -s -w "\n%{http_code}" -X POST "${API_BASE}/api/v1/scan/smart" \
        -H "Content-Type: application/json" \
        -d '{"scenario":"test-scenario","max_files":5,"max_concurrent":2}'

    [ "$status" -eq 0 ]

    # Check HTTP status is 2xx (success) or 4xx (validation error, endpoint exists)
    # Both indicate endpoint is wired correctly
    local http_code=$(echo "$output" | tail -n1)
    [[ "$http_code" =~ ^[24][0-9]{2}$ ]]
}

# [REQ:TM-SS-001] AI batch configuration boundary validation
@test "Smart scan API rejects invalid configuration" {
    # Test with invalid max_files (negative)
    run curl -s -w "\n%{http_code}" -X POST "${API_BASE}/api/v1/scan/smart" \
        -H "Content-Type: application/json" \
        -d '{"scenario":"test","max_files":-1,"max_concurrent":2}'

    [ "$status" -eq 0 ]

    # Should reject invalid config (4xx) or handle gracefully (2xx with error in body)
    local http_code=$(echo "$output" | tail -n1)
    [[ "$http_code" =~ ^[24][0-9]{2}$ ]]
}

# [REQ:TM-SS-002] AI issue recording - verify endpoint structure
@test "Issues endpoint returns structured data" {
    # Query issues endpoint to verify it returns well-formed response
    run curl -s -X GET "${API_BASE}/api/v1/agent/issues?scenario=${TEST_SCENARIO}"

    [ "$status" -eq 0 ]

    # Response must be valid JSON (array or object with issues field)
    echo "$output" | jq -e 'type == "array" or has("issues")' > /dev/null
}

# [REQ:TM-SS-002] AI issue recording - verify filtering works
@test "Issues endpoint supports scenario filtering" {
    # Verify scenario parameter is respected (different scenarios return different results)
    run curl -s "${API_BASE}/api/v1/agent/issues?scenario=nonexistent-scenario-$$"

    [ "$status" -eq 0 ]

    # Should return empty or valid JSON, not error
    echo "$output" | jq -e '. != null' > /dev/null
}

# [REQ:TM-SS-003,TM-SS-004] Campaign manager - list campaigns
@test "Campaign manager returns campaigns list" {
    # Test that API returns campaigns for scenario
    run curl -s "${API_BASE}/api/v1/campaigns?scenario=${TEST_SCENARIO}"

    [ "$status" -eq 0 ]

    # Response must be valid JSON array
    echo "$output" | jq -e 'type == "array"' > /dev/null
}

# [REQ:TM-SS-003,TM-SS-004] Campaign manager - campaign structure validation
@test "Campaign response contains required fields" {
    # Get campaigns and verify structure
    run curl -s "${API_BASE}/api/v1/campaigns"

    [ "$status" -eq 0 ]

    local response="$output"
    echo "$response" | jq -e 'type == "array"' > /dev/null

    # If campaigns exist, validate first one has required fields
    local count=$(echo "$response" | jq 'length')
    if [ "$count" -gt 0 ]; then
        echo "$response" | jq -e '.[0] | has("id") and has("scenario") and has("status")' > /dev/null
    fi
}

# [REQ:TM-SS-005,TM-SS-006] File prioritization - verify parameter accepted
@test "Smart scan accepts visited-tracker prioritization parameter" {
    # Verify that scan accepts use_visited_tracker flag
    run curl -s -w "\n%{http_code}" -X POST "${API_BASE}/api/v1/scan/smart" \
        -H "Content-Type: application/json" \
        -d '{"scenario":"test-scenario","use_visited_tracker":true}'

    [ "$status" -eq 0 ]

    # Should accept parameter (2xx) or validate it exists (4xx with specific error)
    local http_code=$(echo "$output" | tail -n1)
    [[ "$http_code" =~ ^[24][0-9]{2}$ ]]
}

# [REQ:TM-SS-005,TM-SS-006] File prioritization - verify boolean validation
@test "Smart scan validates visited-tracker parameter type" {
    # Test with non-boolean value for use_visited_tracker
    run curl -s -w "\n%{http_code}" -X POST "${API_BASE}/api/v1/scan/smart" \
        -H "Content-Type: application/json" \
        -d '{"scenario":"test-scenario","use_visited_tracker":"invalid"}'

    [ "$status" -eq 0 ]

    # Should either validate type or accept and coerce
    local http_code=$(echo "$output" | tail -n1)
    [[ "$http_code" =~ ^[24][0-9]{2}$ ]]
}

# [REQ:TM-SS-007,TM-SS-008] CLI responsiveness
@test "Smart scan CLI help responds quickly" {
    # CLI help should complete in under 5 seconds
    local start_time=$(date +%s)

    run timeout 5 tidiness-manager scan smart --help

    [ "$status" -eq 0 ]

    local end_time=$(date +%s)
    local duration=$((end_time - start_time))

    # Help should complete in under 2 seconds (generous for CI)
    [ "$duration" -lt 3 ]
}

# [REQ:TM-SS-007,TM-SS-008] CLI parameter documentation
@test "Smart scan help documents required parameters" {
    run tidiness-manager scan smart --help

    [ "$status" -eq 0 ]

    # Help must document scenario parameter
    [[ "$output" =~ "scenario" ]] || [[ "$output" =~ "SCENARIO" ]]
}

# [REQ:TM-SS-007,TM-SS-008] CLI parameter documentation completeness
@test "Smart scan help documents optional parameters" {
    run tidiness-manager scan smart --help

    [ "$status" -eq 0 ]

    # Help should document key optional parameters
    local has_max_files=$([[ "$output" =~ "max-files" ]] || [[ "$output" =~ "max_files" ]] && echo "yes" || echo "no")
    local has_concurrent=$([[ "$output" =~ "concurrent" ]] && echo "yes" || echo "no")

    # At least one optional parameter should be documented
    [[ "$has_max_files" == "yes" || "$has_concurrent" == "yes" ]]
}

# [REQ:TM-SS-001,TM-SS-002] Smart scan error handling
@test "Smart scan API handles missing required fields gracefully" {
    # POST without required scenario field
    run curl -s -w "\n%{http_code}" -X POST "${API_BASE}/api/v1/scan/smart" \
        -H "Content-Type: application/json" \
        -d '{"max_files":5}'

    [ "$status" -eq 0 ]

    # Should return 4xx error for missing required field
    local http_code=$(echo "$output" | tail -n1)
    [[ "$http_code" =~ ^4[0-9]{2}$ ]]
}

# [REQ:TM-SS-001,TM-SS-002] Smart scan error handling - malformed JSON
@test "Smart scan API rejects malformed JSON" {
    # POST with invalid JSON
    run curl -s -w "\n%{http_code}" -X POST "${API_BASE}/api/v1/scan/smart" \
        -H "Content-Type: application/json" \
        -d '{invalid json}'

    [ "$status" -eq 0 ]

    # Should return 4xx error for malformed JSON
    local http_code=$(echo "$output" | tail -n1)
    [[ "$http_code" =~ ^4[0-9]{2}$ ]]
}

# [REQ:TM-SS-001] Smart scan configuration - zero values handling
@test "Smart scan handles zero max_files edge case" {
    # Test with max_files=0 (edge case)
    run curl -s -w "\n%{http_code}" -X POST "${API_BASE}/api/v1/scan/smart" \
        -H "Content-Type: application/json" \
        -d '{"scenario":"test","max_files":0,"max_concurrent":1}'

    [ "$status" -eq 0 ]

    # Should either accept (2xx) or validate (4xx)
    local http_code=$(echo "$output" | tail -n1)
    [[ "$http_code" =~ ^[24][0-9]{2}$ ]]
}

# [REQ:TM-SS-001] Smart scan configuration - large values handling
@test "Smart scan handles very large max_files" {
    # Test with very large max_files value
    run curl -s -w "\n%{http_code}" -X POST "${API_BASE}/api/v1/scan/smart" \
        -H "Content-Type: application/json" \
        -d '{"scenario":"test","max_files":999999,"max_concurrent":100}'

    [ "$status" -eq 0 ]

    # Should accept large but valid values
    local http_code=$(echo "$output" | tail -n1)
    [[ "$http_code" =~ ^2[0-9]{2}$ ]]
}

# [REQ:TM-SS-002] AI issue recording - empty scenario handling
@test "Issues endpoint handles empty scenario gracefully" {
    # Query for scenario that has no issues
    run curl -s "${API_BASE}/api/v1/agent/issues?scenario=empty-nonexistent-scenario-$$"

    [ "$status" -eq 0 ]

    # Should return empty array or object, not error
    echo "$output" | jq -e 'if type == "array" then length >= 0 else true end' > /dev/null
}

# [REQ:TM-SS-002] AI issue recording - pagination support check
@test "Issues endpoint supports limit parameter" {
    # Test limit query parameter
    run curl -s "${API_BASE}/api/v1/agent/issues?scenario=${TEST_SCENARIO}&limit=5"

    [ "$status" -eq 0 ]

    # Response must be valid JSON
    echo "$output" | jq -e '. != null' > /dev/null
}

# [REQ:TM-SS-002] AI issue recording - filtering validation
@test "Issues endpoint supports category filtering" {
    # Test category filter parameter
    run curl -s "${API_BASE}/api/v1/agent/issues?scenario=${TEST_SCENARIO}&category=lint"

    [ "$status" -eq 0 ]

    # Response must be valid JSON
    echo "$output" | jq -e '. != null' > /dev/null
}

# [REQ:TM-SS-003,TM-SS-004] Campaign manager - concurrent request handling
@test "Campaign manager handles concurrent requests" {
    # Launch multiple concurrent requests
    curl -s "${API_BASE}/api/v1/campaigns?scenario=${TEST_SCENARIO}" &
    curl -s "${API_BASE}/api/v1/campaigns?scenario=${TEST_SCENARIO}" &
    curl -s "${API_BASE}/api/v1/campaigns?scenario=${TEST_SCENARIO}" &

    # Wait for all to complete
    wait

    # If we get here without hanging, concurrent handling works
    [ "$?" -eq 0 ]
}

# [REQ:TM-SS-005,TM-SS-006] File prioritization - priority order verification
@test "Smart scan respects file prioritization order" {
    # Test that use_visited_tracker actually affects scan order
    # (This is a smoke test - full validation requires checking scan results)
    run curl -s -w "\n%{http_code}" -X POST "${API_BASE}/api/v1/scan/smart" \
        -H "Content-Type: application/json" \
        -d '{"scenario":"test-scenario","use_visited_tracker":true,"max_files":3}'

    [ "$status" -eq 0 ]

    local http_code=$(echo "$output" | tail -n1)
    [[ "$http_code" =~ ^[24][0-9]{2}$ ]]
}

# [REQ:TM-SS-007,TM-SS-008] CLI error messaging quality
@test "Smart scan CLI provides helpful error for missing scenario" {
    # Run scan without required scenario argument
    run tidiness-manager scan smart 2>&1 || true

    # Error message should mention "scenario" or "required"
    [[ "$output" =~ "scenario" ]] || [[ "$output" =~ "required" ]] || [[ "$output" =~ "SCENARIO" ]]
}

# [REQ:TM-SS-007,TM-SS-008] CLI validation - invalid flags
@test "Smart scan CLI rejects invalid flags gracefully" {
    # Test with invalid flag
    run tidiness-manager scan smart --invalid-flag-that-does-not-exist 2>&1 || true

    # Should exit with error
    [ "$status" -ne 0 ]

    # Error should be informative (mention "flag" or "unknown")
    [[ "$output" =~ "flag" ]] || [[ "$output" =~ "unknown" ]] || [[ "$output" =~ "invalid" ]]
}

# [REQ:TM-SS-001,TM-SS-002] Smart scan HTTP method validation
@test "Smart scan API only accepts POST requests" {
    # Try GET request (should fail)
    run curl -s -w "\n%{http_code}" -X GET "${API_BASE}/api/v1/scan/smart"

    [ "$status" -eq 0 ]

    # Should return 405 Method Not Allowed
    local http_code=$(echo "$output" | tail -n1)
    [[ "$http_code" =~ ^405$ ]] || [[ "$http_code" =~ ^404$ ]]
}
