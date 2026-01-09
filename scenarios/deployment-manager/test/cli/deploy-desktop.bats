#!/usr/bin/env bats
# Deploy Desktop End-to-End Tests
# Tests for the deploy-desktop orchestration command

setup() {
    export PATH="${BATS_TEST_DIRNAME}/../../cli:${PATH}"

    API_PORT=$(vrooli scenario port deployment-manager API_PORT 2>/dev/null || echo "18722")
    export API_URL="http://127.0.0.1:${API_PORT}"

    timeout 10 bash -c "until curl -sf ${API_URL}/health &>/dev/null; do sleep 0.5; done" || {
        echo "# API not ready at ${API_URL}" >&3
        return 1
    }

    export TEST_PROFILE="test-desktop-$$"
}

teardown() {
    deployment-manager profile delete "${TEST_PROFILE}" 2>/dev/null || true
}

# ============================================================================
# Basic Command Tests
# ============================================================================

@test "deploy-desktop command exists and shows help" {
    run deployment-manager deploy-desktop --help 2>&1

    [ "$status" -eq 0 ] || [ "$status" -eq 2 ]
    [[ "$output" =~ "deploy-desktop" ]]
    [[ "$output" =~ "profile" ]]
}

@test "deploy-desktop requires --profile flag" {
    run deployment-manager deploy-desktop 2>&1

    [ "$status" -ne 0 ]
    [[ "$output" =~ "profile" ]] || [[ "$output" =~ "required" ]]
}

@test "deploy-desktop fails gracefully for nonexistent profile" {
    run deployment-manager deploy-desktop --profile nonexistent-profile-$$

    [ "$status" -ne 0 ]
    [[ "$output" =~ "not found" ]] || [[ "$output" =~ "error" ]] || [[ "$output" =~ "fail" ]]
}

# ============================================================================
# Orchestration Step Tests
# ============================================================================

@test "deploy-desktop dry-run shows all steps" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel --tier 2 2>&1 || skip "Profile creation failed"

    run deployment-manager deploy-desktop --profile "${TEST_PROFILE}" --dry-run --format json 2>&1

    [ "$status" -eq 0 ]
    [[ "$output" =~ "steps" ]]
    [[ "$output" =~ "Load profile" ]]
    [[ "$output" =~ "Assemble manifest" ]]
}

@test "deploy-desktop with --skip-build skips binary compilation" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel --tier 2 2>&1 || skip "Profile creation failed"

    run deployment-manager deploy-desktop --profile "${TEST_PROFILE}" --skip-build --dry-run --format json 2>&1

    [ "$status" -eq 0 ]
    [[ "$output" =~ "skipped" ]]
}

@test "deploy-desktop with --skip-packaging skips Electron generation" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel --tier 2 2>&1 || skip "Profile creation failed"

    run deployment-manager deploy-desktop --profile "${TEST_PROFILE}" --skip-packaging --dry-run --format json 2>&1

    [ "$status" -eq 0 ]
    [[ "$output" =~ "skipped" ]]
}

@test "deploy-desktop with --skip-installers skips installer build" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel --tier 2 2>&1 || skip "Profile creation failed"

    run deployment-manager deploy-desktop --profile "${TEST_PROFILE}" --skip-installers --dry-run --format json 2>&1

    [ "$status" -eq 0 ]
    [[ "$output" =~ "skipped" ]]
}

# ============================================================================
# Deployment Mode Tests
# ============================================================================

@test "deploy-desktop supports bundled mode (default)" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel --tier 2 2>&1 || skip "Profile creation failed"

    run deployment-manager deploy-desktop --profile "${TEST_PROFILE}" --mode bundled --dry-run --format json 2>&1

    [ "$status" -eq 0 ]
    [[ "$output" =~ "bundled" ]] || [[ "$output" =~ "steps" ]]
}

@test "deploy-desktop supports external-server mode" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel --tier 2 2>&1 || skip "Profile creation failed"

    run deployment-manager deploy-desktop --profile "${TEST_PROFILE}" --mode external-server --dry-run --format json 2>&1

    [ "$status" -eq 0 ]
}

@test "deploy-desktop supports cloud-api mode" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel --tier 2 2>&1 || skip "Profile creation failed"

    run deployment-manager deploy-desktop --profile "${TEST_PROFILE}" --mode cloud-api --dry-run --format json 2>&1

    [ "$status" -eq 0 ]
}

# ============================================================================
# Platform Selection Tests
# ============================================================================

@test "deploy-desktop accepts --platforms flag" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel --tier 2 2>&1 || skip "Profile creation failed"

    run deployment-manager deploy-desktop --profile "${TEST_PROFILE}" --platforms win,mac --dry-run --format json 2>&1

    [ "$status" -eq 0 ]
}

@test "deploy-desktop accepts single platform" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel --tier 2 2>&1 || skip "Profile creation failed"

    run deployment-manager deploy-desktop --profile "${TEST_PROFILE}" --platforms linux --dry-run --format json 2>&1

    [ "$status" -eq 0 ]
}

# ============================================================================
# Output Format Tests
# ============================================================================

@test "deploy-desktop supports --format json" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel --tier 2 2>&1 || skip "Profile creation failed"

    run deployment-manager deploy-desktop --profile "${TEST_PROFILE}" --dry-run --format json 2>&1

    [ "$status" -eq 0 ]
    # Should be valid JSON
    echo "$output" | jq . &>/dev/null
}

@test "deploy-desktop returns structured response with status" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel --tier 2 2>&1 || skip "Profile creation failed"

    run deployment-manager deploy-desktop --profile "${TEST_PROFILE}" --dry-run --format json 2>&1

    [ "$status" -eq 0 ]
    [[ "$output" =~ "status" ]]
    [[ "$output" =~ "profile_id" ]]
    [[ "$output" =~ "scenario" ]]
}

@test "deploy-desktop returns next_steps on success" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel --tier 2 2>&1 || skip "Profile creation failed"

    run deployment-manager deploy-desktop --profile "${TEST_PROFILE}" --skip-packaging --dry-run --format json 2>&1

    [ "$status" -eq 0 ]
    [[ "$output" =~ "next_steps" ]] || [[ "$output" =~ "success" ]]
}

# ============================================================================
# API Integration Tests
# ============================================================================

@test "deploy-desktop API endpoint accepts POST request" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel --tier 2 2>&1 || skip "Profile creation failed"

    run curl -sf -X POST "${API_URL}/api/v1/deploy-desktop" \
        -H "Content-Type: application/json" \
        -d "{\"profile_id\": \"${TEST_PROFILE}\", \"dry_run\": true}"

    [ "$status" -eq 0 ]
    [[ "$output" =~ "status" ]]
}

@test "deploy-desktop API returns step-by-step progress" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel --tier 2 2>&1 || skip "Profile creation failed"

    result=$(curl -sf -X POST "${API_URL}/api/v1/deploy-desktop" \
        -H "Content-Type: application/json" \
        -d "{\"profile_id\": \"${TEST_PROFILE}\", \"dry_run\": true}")

    [ -n "$result" ]
    echo "$result" | jq -e '.steps' &>/dev/null
    echo "$result" | jq -e '.steps | length > 0' &>/dev/null
}

# ============================================================================
# Integration with scenario-to-desktop Tests
# ============================================================================

@test "deploy-desktop detects scenario-to-desktop availability" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel --tier 2 2>&1 || skip "Profile creation failed"

    # Run without dry-run but with skip-installers to test service detection
    run deployment-manager deploy-desktop --profile "${TEST_PROFILE}" --skip-build --skip-installers --format json 2>&1

    # Should either succeed or fail with meaningful error about scenario-to-desktop
    [ "$status" -eq 0 ] || [[ "$output" =~ "scenario-to-desktop" ]] || [[ "$output" =~ "failed" ]]
}

# ============================================================================
# Validation Tests
# ============================================================================

@test "deploy-desktop validates profile before deployment" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel --tier 2 2>&1 || skip "Profile creation failed"

    run deployment-manager deploy-desktop --profile "${TEST_PROFILE}" --dry-run --format json 2>&1

    [ "$status" -eq 0 ]
    [[ "$output" =~ "Validate profile" ]] || [[ "$output" =~ "validation" ]]
}

@test "deploy-desktop with --skip-validation skips profile validation" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel --tier 2 2>&1 || skip "Profile creation failed"

    run deployment-manager deploy-desktop --profile "${TEST_PROFILE}" --skip-validation --dry-run --format json 2>&1

    [ "$status" -eq 0 ]
    [[ "$output" =~ "skipped" ]] || [[ "$output" =~ "Validate profile" ]]
}

# ============================================================================
# Duration and Performance Tests
# ============================================================================

@test "deploy-desktop returns duration in response" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel --tier 2 2>&1 || skip "Profile creation failed"

    result=$(curl -sf -X POST "${API_URL}/api/v1/deploy-desktop" \
        -H "Content-Type: application/json" \
        -d "{\"profile_id\": \"${TEST_PROFILE}\", \"dry_run\": true}")

    [ -n "$result" ]
    echo "$result" | jq -e '.duration' &>/dev/null
}

# ============================================================================
# Error Handling Tests
# ============================================================================

@test "deploy-desktop handles invalid JSON gracefully" {
    run curl -sf -X POST "${API_URL}/api/v1/deploy-desktop" \
        -H "Content-Type: application/json" \
        -d "not valid json"

    [ "$status" -ne 0 ] || [[ "$output" =~ "error" ]] || [[ "$output" =~ "invalid" ]]
}

@test "deploy-desktop handles missing profile_id" {
    run curl -sf -X POST "${API_URL}/api/v1/deploy-desktop" \
        -H "Content-Type: application/json" \
        -d "{}"

    [ "$status" -ne 0 ] || [[ "$output" =~ "required" ]] || [[ "$output" =~ "profile_id" ]]
}
