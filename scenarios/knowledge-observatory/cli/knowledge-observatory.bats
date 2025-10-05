#!/usr/bin/env bats

# Knowledge Observatory CLI Tests

setup() {
    # Discover the actual API port from the running scenario
    # The lifecycle system sets API_PORT in the environment
    if [ -z "$API_PORT" ]; then
        # Try to discover from the running process if not in environment
        API_PORT=$(ps aux | grep knowledge-observatory-api | grep -v grep | head -n1 | xargs -I{} sh -c 'cat /proc/$(echo {} | awk "{print \$2}")/environ 2>/dev/null | tr "\0" "\n" | grep "^API_PORT=" | cut -d= -f2' || echo "20270")
    fi
    export KNOWLEDGE_OBSERVATORY_API_URL="http://localhost:${API_PORT}"
    CLI="./knowledge-observatory"
}

@test "CLI exists and is executable" {
    [ -f "$CLI" ]
    [ -x "$CLI" ]
}

@test "CLI shows help" {
    run $CLI help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Knowledge Observatory CLI" ]]
    [[ "$output" =~ "Commands:" ]]
    [[ "$output" =~ "search" ]]
}

@test "CLI shows version" {
    run $CLI version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "1.0.0" ]]
}

@test "CLI status command works" {
    run $CLI status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Knowledge Observatory Status" ]]
}

@test "CLI status with JSON flag" {
    run $CLI status --json
    [ "$status" -eq 0 ]
    # Should output valid JSON
    echo "$output" | grep -q '"status"'
}

@test "CLI search requires query" {
    run $CLI search
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Search query is required" ]]
}

@test "CLI search with query" {
    skip "Requires API to be running"
    run $CLI search "test query"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Search Results" ]]
}

@test "CLI search with options" {
    skip "Requires API to be running"
    run $CLI search "test" --limit 5 --collection vrooli_knowledge
    [ "$status" -eq 0 ]
}

@test "CLI health command" {
    skip "Requires API to be running"
    run $CLI health
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Knowledge Observatory Status" ]]
}

@test "CLI graph command" {
    skip "Requires API to be running"
    run $CLI graph
    [ "$status" -eq 0 ]
}

@test "CLI graph with center concept" {
    skip "Requires API to be running"
    run $CLI graph --center "scenario-generator" --depth 2
    [ "$status" -eq 0 ]
}

@test "CLI metrics command" {
    skip "Requires API to be running"
    run $CLI metrics
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Knowledge Quality Metrics" ]]
}

@test "CLI metrics with collection filter" {
    skip "Requires API to be running"
    run $CLI metrics --collection vrooli_knowledge
    [ "$status" -eq 0 ]
}

@test "CLI handles unknown command" {
    run $CLI nonexistent
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown command" ]]
}

@test "CLI search help" {
    run $CLI search --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage:" ]]
    [[ "$output" =~ "--collection" ]]
}

@test "CLI health help" {
    run $CLI health --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage:" ]]
    [[ "$output" =~ "--watch" ]]
}

@test "CLI graph help" {
    run $CLI graph --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage:" ]]
    [[ "$output" =~ "--depth" ]]
}

@test "CLI metrics help" {
    run $CLI metrics --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage:" ]]
    [[ "$output" =~ "--time-range" ]]
}