#!/usr/bin/env bats
# CLI tests for agent API integration
# [REQ:TM-API-001,TM-API-002,TM-API-003,TM-API-005]

load '../../../scripts/lib/testing/bats-helpers.sh'

setup() {
    # Ensure tidiness-manager is running
    if ! pgrep -f "tidiness-manager-api" > /dev/null; then
        skip "tidiness-manager-api not running"
    fi
}

# [REQ:TM-API-001] Scenario summary endpoint
@test "CLI: tidiness-manager issues --scenario picker-wheel returns summary" {
    run tidiness-manager issues --scenario picker-wheel --format json

    [ "$status" -eq 0 ]
    echo "$output" | jq -e '.scenario == "picker-wheel"'
    echo "$output" | jq -e '.total_issues >= 0'
    echo "$output" | jq -e '.by_category'
    echo "$output" | jq -e '.by_severity'
}

# [REQ:TM-API-002] File-scoped issue query
@test "CLI: tidiness-manager issues --file returns file-specific issues" {
    run tidiness-manager issues --scenario picker-wheel --file "src/main.tsx" --format json

    [ "$status" -eq 0 ]
    echo "$output" | jq -e '.file == "src/main.tsx"'
    echo "$output" | jq -e '.issues'
}

# [REQ:TM-API-003] Category-scoped issue query
@test "CLI: tidiness-manager issues --category returns category-filtered issues" {
    run tidiness-manager issues --scenario picker-wheel --category lint --format json

    [ "$status" -eq 0 ]
    echo "$output" | jq -e '.category == "lint"'
    echo "$output" | jq -e '.issues'
}

# [REQ:TM-API-005] CLI wrapper supports multiple output formats
@test "CLI: tidiness-manager issues supports --format json" {
    run tidiness-manager issues --scenario picker-wheel --format json

    [ "$status" -eq 0 ]
    echo "$output" | jq -e '.scenario'
}

@test "CLI: tidiness-manager issues supports --format table (human-readable)" {
    run tidiness-manager issues --scenario picker-wheel --format table

    [ "$status" -eq 0 ]
    # Table format should contain column headers
    [[ "$output" =~ "SCENARIO" ]] || [[ "$output" =~ "ISSUES" ]]
}

# [REQ:TM-API-005] CLI supports --limit for top N issues
@test "CLI: tidiness-manager issues --limit returns limited results" {
    run tidiness-manager issues --scenario picker-wheel --limit 5 --format json

    [ "$status" -eq 0 ]
    count=$(echo "$output" | jq '.issues | length')
    [ "$count" -le 5 ]
}

# [REQ:TM-API-005] CLI supports --severity filter
@test "CLI: tidiness-manager issues --severity filters by severity" {
    run tidiness-manager issues --scenario picker-wheel --severity critical --format json

    [ "$status" -eq 0 ]
    # All returned issues should be critical
    echo "$output" | jq -e '.issues[] | select(.severity != "critical") | length == 0'
}

# [REQ:TM-API-001] CLI handles missing scenario gracefully
@test "CLI: tidiness-manager issues with invalid scenario returns error" {
    run tidiness-manager issues --scenario nonexistent-scenario-xyz

    [ "$status" -ne 0 ]
    [[ "$output" =~ "not found" ]] || [[ "$output" =~ "error" ]]
}

# [REQ:TM-API-005] CLI provides helpful error messages
@test "CLI: tidiness-manager issues without --scenario shows usage" {
    run tidiness-manager issues

    [ "$status" -ne 0 ]
    [[ "$output" =~ "scenario" ]] || [[ "$output" =~ "required" ]]
}

# [REQ:TM-API-006] CLI can query issue storage
@test "CLI: tidiness-manager issues queries postgres-backed storage" {
    # This tests that CLI can retrieve stored issues
    run tidiness-manager issues --scenario tidiness-manager --format json

    [ "$status" -eq 0 ]
    echo "$output" | jq -e '.total_issues >= 0'
}
