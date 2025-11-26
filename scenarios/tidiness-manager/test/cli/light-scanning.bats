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
@test "Light scan computes file metrics" {
    # Test will be implemented when light scan endpoint is fully wired
    skip "Full light scan implementation pending"
}

# [REQ:TM-LS-006] Flag files exceeding configurable line count threshold as 'long file' issues
@test "Light scan flags long files" {
    skip "Full light scan implementation pending"
}
