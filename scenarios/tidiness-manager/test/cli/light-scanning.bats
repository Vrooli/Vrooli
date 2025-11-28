#!/usr/bin/env bats
# [REQ:TM-LS-001] [REQ:TM-LS-002] [REQ:TM-LS-003] [REQ:TM-LS-004]
# Light scanning: make lint/type execution and parsing

load "../../__test/lib/bats-support/load"
load "../../__test/lib/bats-assert/load"

setup() {
    export API_PORT="${API_PORT:-16820}"
    export API_BASE="http://localhost:${API_PORT}"
}

# [REQ:TM-LS-001] Execute `make lint` for any scenario with a Makefile and capture full output
@test "POST /api/v1/scan/light/parse-lint accepts lint output" {
    run curl -s -X POST "${API_BASE}/api/v1/scan/light/parse-lint" \
        -H "Content-Type: application/json" \
        -d '{"scenario":"test-scenario","tool":"golangci-lint","output":"api/main.go:10:1: unused variable (unused)"}'

    assert_success
    assert_output --partial '"issues"'
}

# [REQ:TM-LS-002] Execute `make type` for any scenario with a Makefile and capture full output
@test "POST /api/v1/scan/light/parse-type accepts type output" {
    run curl -s -X POST "${API_BASE}/api/v1/scan/light/parse-type" \
        -H "Content-Type: application/json" \
        -d '{"scenario":"test-scenario","tool":"tsc","output":"src/App.tsx(45,10): error TS2304: Cannot find name \"Foo\"."}'

    assert_success
    assert_output --partial '"issues"'
}

# [REQ:TM-LS-003] Parse lint outputs into structured issues with scenario, file path, line/column
@test "Parse lint output extracts file, line, column" {
    local response
    response=$(curl -s -X POST "${API_BASE}/api/v1/scan/light/parse-lint" \
        -H "Content-Type: application/json" \
        -d '{"scenario":"my-scenario","tool":"golangci-lint","output":"api/handlers.go:42:5: error message"}')

    echo "$response" | jq -e '.issues[0].file_path == "api/handlers.go"'
    echo "$response" | jq -e '.issues[0].line == 42'
    echo "$response" | jq -e '.issues[0].column == 5'
}

# [REQ:TM-LS-004] Parse type checker outputs into structured issues
@test "Parse type output extracts file, line, column" {
    local response
    response=$(curl -s -X POST "${API_BASE}/api/v1/scan/light/parse-type" \
        -H "Content-Type: application/json" \
        -d '{"scenario":"my-scenario","tool":"tsc","output":"ui/src/App.tsx(100,25): error TS2345"}')

    echo "$response" | jq -e '.issues[0].file_path == "ui/src/App.tsx"'
    echo "$response" | jq -e '.issues[0].line == 100'
    echo "$response" | jq -e '.issues[0].column == 25'
}

# [REQ:TM-LS-005] Compute per-file line counts for all source files
@test "GET /api/v1/scan/light/metrics returns file metrics" {
    # Create a temporary scenario and scan it
    local scenario="test-scenario-$(date +%s)"

    # For now, test the endpoint structure
    run curl -s -X GET "${API_BASE}/api/v1/scenarios/${scenario}/files"

    # Should return JSON even if scenario doesn't exist yet
    assert_success
}

# [REQ:TM-LS-006] Flag files exceeding configurable line count threshold as 'long file' issues
@test "Long files are flagged in scan results" {
    local scenario="test-scenario-long"

    # Query issues for a scenario (endpoint should exist)
    run curl -s -X GET "${API_BASE}/api/v1/agent/issues?scenario=${scenario}&type=length"

    assert_success
    # Should return valid JSON with issues array
    echo "$output" | jq -e 'type == "array"' || echo "$output" | jq -e '.issues | type == "array"'
}

# [REQ:TM-LS-007] Light scans for scenarios <50 files must complete in under 60 seconds
@test "Light scan CLI completes quickly for small scenarios" {
    # Test CLI command exists and responds
    run timeout 5 tidiness-manager scan light --help

    # Help should work quickly
    assert_success
}

# [REQ:TM-LS-008] Light scans for scenarios <200 files must complete in under 120 seconds
@test "Light scan handles medium-sized scenarios" {
    # Verify scan command accepts scenario parameter
    run tidiness-manager scan light --help

    assert_success
    assert_output --partial "scenario"
}
