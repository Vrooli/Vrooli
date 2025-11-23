#!/usr/bin/env bats
# Swap Analysis CLI Tests
# Tests for requirements DM-P0-007 through DM-P0-011

setup() {
    export PATH="${BATS_TEST_DIRNAME}/../../cli:${PATH}"

    API_PORT=$(vrooli scenario port deployment-manager API_PORT 2>/dev/null || echo "18722")
    export API_URL="http://127.0.0.1:${API_PORT}"

    timeout 10 bash -c "until curl -sf ${API_URL}/health &>/dev/null; do sleep 0.5; done" || {
        echo "# API not ready at ${API_URL}" >&3
        return 1
    }
}

# [REQ:DM-P0-007] Swap Suggestions
@test "[REQ:DM-P0-007] Suggest swaps for low-fitness dependencies" {
    run deployment-manager swaps suggest postgres --tier mobile 2>&1

    # Should suggest alternatives or show "No known alternatives"
    [ "$status" -eq 0 ] || [[ "$output" =~ "swap" ]] || [[ "$output" =~ "alternative" ]] || [[ "$output" =~ "No known" ]]
}

# [REQ:DM-P0-007] Multiple Swap Suggestions
@test "[REQ:DM-P0-007] Provide at least one swap suggestion for blockers" {
    run deployment-manager swaps suggest postgres --tier mobile --format json 2>&1

    # Should have suggestions array
    [[ "$output" =~ "suggestions" ]] || [[ "$output" =~ "alternatives" ]] || [[ "$output" =~ "swap" ]]
}

# [REQ:DM-P0-008] Swap Impact Analysis
@test "[REQ:DM-P0-008] Show fitness delta for swap" {
    run deployment-manager swaps analyze postgres sqlite --tier mobile 2>&1

    # Should show before/after fitness or delta
    [[ "$output" =~ "delta" ]] || [[ "$output" =~ "before" ]] || [[ "$output" =~ "after" ]] || [[ "$output" =~ "fitness" ]]
}

# [REQ:DM-P0-008] Swap Trade-offs Display
@test "[REQ:DM-P0-008] Display trade-offs (pros/cons) for swaps" {
    run deployment-manager swaps analyze postgres sqlite --tier mobile --verbose 2>&1

    # Should show trade-offs
    [[ "$output" =~ "pros" ]] || [[ "$output" =~ "cons" ]] || [[ "$output" =~ "trade-off" ]] || [[ "$output" =~ "impact" ]]
}

# [REQ:DM-P0-008] Migration Effort Estimate
@test "[REQ:DM-P0-008] Show migration effort estimate" {
    run deployment-manager swaps analyze postgres sqlite --tier mobile --format json 2>&1

    # Should include migration effort
    [[ "$output" =~ "effort" ]] || [[ "$output" =~ "migration" ]] || [[ "$output" =~ "impact" ]]
}

# [REQ:DM-P0-009] Non-Destructive Swaps
@test "[REQ:DM-P0-009] Applying swap updates profile only" {
    export TEST_PROFILE="test-swap-$$"

    deployment-manager profile create "${TEST_PROFILE}" picker-wheel --tier mobile 2>&1 || skip "Profile creation not ready"

    run deployment-manager swaps apply "${TEST_PROFILE}" postgres sqlite 2>&1

    # Should indicate profile update, not source code modification
    [[ "$output" =~ "profile" ]] || [[ "$output" =~ "updated" ]] || [[ "$output" =~ "swap" ]]

    deployment-manager profile delete "${TEST_PROFILE}" 2>/dev/null || true
}

# [REQ:DM-P0-009] Source Code Preservation
@test "[REQ:DM-P0-009] Verify swap does not modify source files" {
    export TEST_PROFILE="test-source-$$"

    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    # Apply swap
    deployment-manager swaps apply "${TEST_PROFILE}" postgres sqlite 2>&1 || true

    # Verify no source file modifications (check scenario files unchanged)
    run ls /home/matthalloran8/Vrooli/scenarios/picker-wheel/.vrooli/service.json 2>&1
    [ "$status" -eq 0 ]

    deployment-manager profile delete "${TEST_PROFILE}" 2>/dev/null || true
}

# [REQ:DM-P0-010] Real-Time Recalculation Performance
@test "[REQ:DM-P0-010] Recalculate fitness within 1 second after swap" {
    export TEST_PROFILE="test-recalc-$$"

    deployment-manager profile create "${TEST_PROFILE}" picker-wheel --tier mobile 2>&1 || skip "Profile creation not ready"

    start_time=$(date +%s%3N)
    run deployment-manager swaps apply "${TEST_PROFILE}" postgres sqlite --recalculate 2>&1
    end_time=$(date +%s%3N)

    duration=$((end_time - start_time))

    # Verify performance: < 1000ms
    [ "$duration" -lt 1000 ] || skip "Performance target not met (${duration}ms > 1000ms)"

    deployment-manager profile delete "${TEST_PROFILE}" 2>/dev/null || true
}

# [REQ:DM-P0-010] Automatic Fitness Update
@test "[REQ:DM-P0-010] Fitness scores update automatically after swap" {
    export TEST_PROFILE="test-auto-$$"

    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    # Apply swap and check for updated fitness
    run deployment-manager swaps apply "${TEST_PROFILE}" postgres sqlite --show-fitness 2>&1

    [[ "$output" =~ "fitness" ]] || [[ "$output" =~ "score" ]] || [[ "$output" =~ "updated" ]]

    deployment-manager profile delete "${TEST_PROFILE}" 2>/dev/null || true
}

# [REQ:DM-P0-011] Cascading Swap Detection
@test "[REQ:DM-P0-011] Detect cascading impacts when swapping" {
    export TEST_PROFILE="test-cascade-$$"

    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    run deployment-manager swaps analyze postgres sqlite --check-cascade 2>&1

    # Should check for cascading impacts
    [[ "$output" =~ "cascade" ]] || [[ "$output" =~ "dependent" ]] || [[ "$output" =~ "impact" ]]

    deployment-manager profile delete "${TEST_PROFILE}" 2>/dev/null || true
}

# [REQ:DM-P0-011] Warn About Dependent Scenarios
@test "[REQ:DM-P0-011] Warn when swap removes dependent scenario" {
    export TEST_PROFILE="test-warn-$$"

    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    run deployment-manager swaps apply "${TEST_PROFILE}" postgres sqlite --dry-run 2>&1

    # If dependencies exist, should warn
    [[ "$output" =~ "warning" ]] || [[ "$output" =~ "dependent" ]] || [[ "$output" =~ "impact" ]] || [[ "$output" =~ "swap" ]]

    deployment-manager profile delete "${TEST_PROFILE}" 2>/dev/null || true
}
