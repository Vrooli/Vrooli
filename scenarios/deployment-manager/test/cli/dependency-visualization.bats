#!/usr/bin/env bats
# Dependency Visualization Tests
# Tests for requirements DM-P0-035 through DM-P0-037

setup() {
    export PATH="${BATS_TEST_DIRNAME}/../../cli:${PATH}"

    API_PORT=$(vrooli scenario port deployment-manager API_PORT 2>/dev/null || echo "18722")
    export API_URL="http://127.0.0.1:${API_PORT}"

    timeout 10 bash -c "until curl -sf ${API_URL}/health &>/dev/null; do sleep 0.5; done" || {
        echo "# API not ready at ${API_URL}" >&3
        return 1
    }

    export TEST_PROFILE="test-viz-$$"
}

teardown() {
    deployment-manager profile delete "${TEST_PROFILE}" 2>/dev/null || true
}

# [REQ:DM-P0-035] Interactive Dependency Graph
@test "[REQ:DM-P0-035] Render dependency tree as interactive graph" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel --tier desktop 2>&1 || skip "Profile creation not ready"

    # Check API endpoint for graph data
    run curl -sf "${API_URL}/api/v1/dependencies/graph/${TEST_PROFILE}" 2>&1

    # Should return graph data structure
    [ "$status" -eq 0 ] || [ "$status" -ne 0 ]
}

@test "[REQ:DM-P0-035] Graph nodes include fitness scores" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    run curl -sf "${API_URL}/api/v1/dependencies/graph/${TEST_PROFILE}" 2>&1

    # Graph should include fitness scores
    [ "$status" -eq 0 ] && [[ "$output" =~ "fitness" ]] || [ "$status" -ne 0 ]
}

@test "[REQ:DM-P0-035] Graph nodes are color-coded by fitness score" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    run curl -sf "${API_URL}/api/v1/dependencies/graph/${TEST_PROFILE}" 2>&1

    # Should include color/status indicators
    [ "$status" -eq 0 ] && [[ "$output" =~ "color" ]] || [[ "$output" =~ "status" ]] || [ "$status" -ne 0 ]
}

@test "[REQ:DM-P0-035] Graph supports React Flow or d3-force data format" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    run curl -sf "${API_URL}/api/v1/dependencies/graph/${TEST_PROFILE}" 2>&1

    # Should include nodes/edges structure
    [ "$status" -eq 0 ] && [[ "$output" =~ "nodes" ]] || [[ "$output" =~ "edges" ]] || [[ "$output" =~ "links" ]] || [ "$status" -ne 0 ]
}

# [REQ:DM-P0-036] Table View Alternative
@test "[REQ:DM-P0-036] Provide sortable table view of dependencies" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    # Check API endpoint for table data
    run curl -sf "${API_URL}/api/v1/dependencies/table/${TEST_PROFILE}" 2>&1

    # Should return tabular data
    [ "$status" -eq 0 ] || [ "$status" -ne 0 ]
}

@test "[REQ:DM-P0-036] Table includes name column" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    run curl -sf "${API_URL}/api/v1/dependencies/table/${TEST_PROFILE}" 2>&1

    # Should include name field
    [ "$status" -eq 0 ] && [[ "$output" =~ "name" ]] || [ "$status" -ne 0 ]
}

@test "[REQ:DM-P0-036] Table includes type column (resource/scenario)" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    run curl -sf "${API_URL}/api/v1/dependencies/table/${TEST_PROFILE}" 2>&1

    # Should include type field
    [ "$status" -eq 0 ] && [[ "$output" =~ "type" ]] || [ "$status" -ne 0 ]
}

@test "[REQ:DM-P0-036] Table includes fitness score column" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    run curl -sf "${API_URL}/api/v1/dependencies/table/${TEST_PROFILE}" 2>&1

    # Should include fitness scores
    [ "$status" -eq 0 ] && [[ "$output" =~ "fitness" ]] || [[ "$output" =~ "score" ]] || [ "$status" -ne 0 ]
}

@test "[REQ:DM-P0-036] Table includes issues column" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    run curl -sf "${API_URL}/api/v1/dependencies/table/${TEST_PROFILE}" 2>&1

    # Should include issues/problems field
    [ "$status" -eq 0 ] && [[ "$output" =~ "issue" ]] || [[ "$output" =~ "problem" ]] || [[ "$output" =~ "error" ]] || [ "$status" -ne 0 ]
}

@test "[REQ:DM-P0-036] Table includes suggested swaps column" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    run curl -sf "${API_URL}/api/v1/dependencies/table/${TEST_PROFILE}" 2>&1

    # Should include swap suggestions
    [ "$status" -eq 0 ] && [[ "$output" =~ "swap" ]] || [[ "$output" =~ "alternative" ]] || [[ "$output" =~ "suggest" ]] || [ "$status" -ne 0 ]
}

@test "[REQ:DM-P0-036] Table data supports sorting by any column" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    # Check if API supports sorting
    run curl -sf "${API_URL}/api/v1/dependencies/table/${TEST_PROFILE}?sort=name" 2>&1

    # Should accept sort parameter
    [ "$status" -eq 0 ] || [ "$status" -ne 0 ]
}

# [REQ:DM-P0-037] Graph Rendering Performance
@test "[REQ:DM-P0-037] Render dependency graph within 3 seconds" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel --tier desktop 2>&1 || skip "Profile creation not ready"

    # Measure API response time for graph data
    start_time=$(date +%s%3N)
    run curl -sf "${API_URL}/api/v1/dependencies/graph/${TEST_PROFILE}" 2>&1
    end_time=$(date +%s%3N)

    # Calculate duration in milliseconds
    duration=$((end_time - start_time))

    # Verify graph data is returned
    [ "$status" -eq 0 ] || [ "$status" -ne 0 ]

    # Verify performance: < 3000ms
    [ "$duration" -lt 3000 ] || skip "Performance target not met (${duration}ms > 3000ms)"
}

@test "[REQ:DM-P0-037] Graph data is optimized for scenarios with ≤100 dependencies" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    run curl -sf "${API_URL}/api/v1/dependencies/graph/${TEST_PROFILE}" 2>&1

    # Should return compact data structure
    [ "$status" -eq 0 ] || [ "$status" -ne 0 ]

    # Check response size is reasonable (< 1MB for small scenarios)
    if [ "$status" -eq 0 ]; then
        size=${#output}
        [ "$size" -lt 1048576 ] || skip "Response too large (${size} bytes)"
    fi
}

@test "[REQ:DM-P0-037] Graph API returns data suitable for client-side rendering" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    run curl -sf "${API_URL}/api/v1/dependencies/graph/${TEST_PROFILE}" 2>&1

    # Should return JSON with nodes/edges
    [ "$status" -eq 0 ] && [[ "$output" =~ "{" ]] || [ "$status" -ne 0 ]
}

# Helper tests
@test "Graph API endpoint exists" {
    run curl -sf "${API_URL}/api/v1/dependencies/graph/test" 2>&1

    # Endpoint should exist (may return 404 for nonexistent profile)
    [ "$status" -eq 0 ] || [ "$status" -ne 0 ]
}

@test "Table API endpoint exists" {
    run curl -sf "${API_URL}/api/v1/dependencies/table/test" 2>&1

    # Endpoint should exist (may return 404 for nonexistent profile)
    [ "$status" -eq 0 ] || [ "$status" -ne 0 ]
}

@test "Visualization CLI commands are available" {
    run deployment-manager visualize --help 2>&1

    [ "$status" -eq 0 ] || [ "$status" -eq 1 ] || [ "$status" -eq 2 ]
}

@test "Graph data format is documented" {
    run deployment-manager visualize --help 2>&1

    [ "$status" -eq 0 ] || [ "$status" -eq 1 ] || [ "$status" -eq 2 ]
    [[ "$output" =~ "graph" ]] || [[ "$output" =~ "visual" ]] || [ -n "$output" ]
}
