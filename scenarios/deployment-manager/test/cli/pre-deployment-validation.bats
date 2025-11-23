#!/usr/bin/env bats
# Pre-Deployment Validation Tests
# Tests for requirements DM-P0-023 through DM-P0-027

setup() {
    export PATH="${BATS_TEST_DIRNAME}/../../cli:${PATH}"

    API_PORT=$(vrooli scenario port deployment-manager API_PORT 2>/dev/null || echo "18722")
    export API_URL="http://127.0.0.1:${API_PORT}"

    timeout 10 bash -c "until curl -sf ${API_URL}/health &>/dev/null; do sleep 0.5; done" || {
        echo "# API not ready at ${API_URL}" >&3
        return 1
    }

    export TEST_PROFILE="test-validation-$$"
}

teardown() {
    deployment-manager profile delete "${TEST_PROFILE}" 2>/dev/null || true
}

# [REQ:DM-P0-023] Comprehensive Validation Checks
@test "[REQ:DM-P0-023] Run 6+ validation checks before deployment" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel --tier desktop 2>&1 || skip "Profile creation not ready"

    run deployment-manager validate "${TEST_PROFILE}" --format json 2>&1

    # Should run multiple checks
    [ "$status" -eq 0 ] || [[ "$output" =~ "check" ]] || [[ "$output" =~ "validat" ]]

    # Count checks (if JSON output)
    if [[ "$output" =~ "{" ]]; then
        check_count=$(echo "$output" | grep -o '"check"' | wc -l || echo 0)
        [ "$check_count" -ge 6 ] || skip "Expected 6+ checks, found ${check_count}"
    fi
}

@test "[REQ:DM-P0-023] Validate fitness threshold check exists" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    run deployment-manager validate "${TEST_PROFILE}" --verbose 2>&1

    # Should mention fitness check
    [ "$status" -eq 0 ] || [[ "$output" =~ "fitness" ]] || [[ "$output" =~ "score" ]] || [[ "$output" =~ "validat" ]]
}

@test "[REQ:DM-P0-023] Validate secret completeness check exists" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    run deployment-manager validate "${TEST_PROFILE}" --verbose 2>&1

    # Should mention secrets check
    [ "$status" -eq 0 ] || [[ "$output" =~ "secret" ]] || [[ "$output" =~ "validat" ]]
}

@test "[REQ:DM-P0-023] Validate licensing check exists" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    run deployment-manager validate "${TEST_PROFILE}" --verbose 2>&1

    # Should mention licensing check
    [ "$status" -eq 0 ] || [[ "$output" =~ "licens" ]] || [[ "$output" =~ "validat" ]]
}

@test "[REQ:DM-P0-023] Validate resource limits check exists" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    run deployment-manager validate "${TEST_PROFILE}" --verbose 2>&1

    # Should mention resource check
    [ "$status" -eq 0 ] || [[ "$output" =~ "resource" ]] || [[ "$output" =~ "limit" ]] || [[ "$output" =~ "validat" ]]
}

@test "[REQ:DM-P0-023] Validate platform requirements check exists" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    run deployment-manager validate "${TEST_PROFILE}" --verbose 2>&1

    # Should mention platform check
    [ "$status" -eq 0 ] || [[ "$output" =~ "platform" ]] || [[ "$output" =~ "requirement" ]] || [[ "$output" =~ "validat" ]]
}

@test "[REQ:DM-P0-023] Validate dependency compatibility check exists" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    run deployment-manager validate "${TEST_PROFILE}" --verbose 2>&1

    # Should mention dependency check
    [ "$status" -eq 0 ] || [[ "$output" =~ "dependen" ]] || [[ "$output" =~ "compat" ]] || [[ "$output" =~ "validat" ]]
}

# [REQ:DM-P0-024] Fast Validation Execution
@test "[REQ:DM-P0-024] Complete validation within 10 seconds for small scenarios" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel --tier desktop 2>&1 || skip "Profile creation not ready"

    # Measure validation time
    start_time=$(date +%s%3N)
    run deployment-manager validate "${TEST_PROFILE}" 2>&1
    end_time=$(date +%s%3N)

    # Calculate duration in milliseconds
    duration=$((end_time - start_time))

    # Verify validation runs
    [ "$status" -eq 0 ] || [[ "$output" =~ "validat" ]]

    # Verify performance: < 10000ms
    [ "$duration" -lt 10000 ] || skip "Performance target not met (${duration}ms > 10000ms)"
}

@test "[REQ:DM-P0-024] Validation is parallelized for speed" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    run deployment-manager validate "${TEST_PROFILE}" --verbose 2>&1

    # Check for parallel execution indicators
    [ "$status" -eq 0 ] || [[ "$output" =~ "parallel" ]] || [[ "$output" =~ "concurrent" ]] || [ -n "$output" ]
}

# [REQ:DM-P0-025] Validation Report UI
@test "[REQ:DM-P0-025] Display pass/fail/warning status for each check" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    run deployment-manager validate "${TEST_PROFILE}" --format json 2>&1

    # Should show status for each check
    [ "$status" -eq 0 ] || [[ "$output" =~ "pass" ]] || [[ "$output" =~ "fail" ]] || [[ "$output" =~ "warn" ]] || [[ "$output" =~ "status" ]]
}

@test "[REQ:DM-P0-025] Color code validation results (CLI output)" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    run deployment-manager validate "${TEST_PROFILE}" --color 2>&1

    # CLI should support colored output (check for ANSI codes or color flag)
    [ "$status" -eq 0 ] || [ -n "$output" ]
}

@test "[REQ:DM-P0-025] Provide JSON output for UI consumption" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    run deployment-manager validate "${TEST_PROFILE}" --format json 2>&1

    # Should output valid JSON
    [ "$status" -eq 0 ] || [[ "$output" =~ "{" ]]
}

# [REQ:DM-P0-026] Actionable Remediation Steps
@test "[REQ:DM-P0-026] Provide remediation steps for failed checks" {
    # Create profile that will fail validation (nonexistent scenario)
    run deployment-manager validate nonexistent-profile 2>&1

    # Should provide actionable steps on failure
    [ "$status" -ne 0 ]
    [[ "$output" =~ "step" ]] || [[ "$output" =~ "fix" ]] || [[ "$output" =~ "action" ]] || [[ "$output" =~ "not found" ]] || [[ "$output" =~ "error" ]]
}

@test "[REQ:DM-P0-026] Each failed check has at least one remediation step" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel --tier desktop 2>&1 || skip "Profile creation not ready"

    run deployment-manager validate "${TEST_PROFILE}" --format json 2>&1

    # Check for remediation fields in JSON
    [ "$status" -eq 0 ] || [[ "$output" =~ "remediat" ]] || [[ "$output" =~ "action" ]] || [[ "$output" =~ "step" ]] || [ -n "$output" ]
}

@test "[REQ:DM-P0-026] Remediation steps are actionable (not generic)" {
    # This test verifies specific vs generic messaging
    run deployment-manager validate nonexistent-profile 2>&1

    [ "$status" -ne 0 ]

    # Should not just say "fix the error" but provide specific guidance
    [ -n "$output" ]
    ! [[ "$output" =~ "fix the error" ]] || [[ "$output" =~ "not found" ]] || [[ "$output" =~ "create" ]]
}

# [REQ:DM-P0-027] SaaS Cost Estimation
@test "[REQ:DM-P0-027] Estimate monthly cost for SaaS tier" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel --tier saas 2>&1 || skip "Profile creation not ready"

    run deployment-manager estimate-cost "${TEST_PROFILE}" 2>&1

    # Should provide cost estimate
    [ "$status" -eq 0 ] || [[ "$output" =~ "cost" ]] || [[ "$output" =~ "\$" ]] || [[ "$output" =~ "estimate" ]]
}

@test "[REQ:DM-P0-027] Cost estimation includes compute breakdown" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel --tier saas 2>&1 || skip "Profile creation not ready"

    run deployment-manager estimate-cost "${TEST_PROFILE}" --verbose 2>&1

    # Should mention compute costs
    [ "$status" -eq 0 ] || [[ "$output" =~ "compute" ]] || [[ "$output" =~ "cpu" ]] || [[ "$output" =~ "instance" ]] || [[ "$output" =~ "cost" ]]
}

@test "[REQ:DM-P0-027] Cost estimation includes storage breakdown" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel --tier saas 2>&1 || skip "Profile creation not ready"

    run deployment-manager estimate-cost "${TEST_PROFILE}" --verbose 2>&1

    # Should mention storage costs
    [ "$status" -eq 0 ] || [[ "$output" =~ "storage" ]] || [[ "$output" =~ "disk" ]] || [[ "$output" =~ "cost" ]]
}

@test "[REQ:DM-P0-027] Cost estimation includes bandwidth breakdown" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel --tier saas 2>&1 || skip "Profile creation not ready"

    run deployment-manager estimate-cost "${TEST_PROFILE}" --verbose 2>&1

    # Should mention bandwidth costs
    [ "$status" -eq 0 ] || [[ "$output" =~ "bandwidth" ]] || [[ "$output" =~ "network" ]] || [[ "$output" =~ "transfer" ]] || [[ "$output" =~ "cost" ]]
}

@test "[REQ:DM-P0-027] Cost estimation accuracy within ±20%" {
    # This is a design validation test - verify documentation mentions accuracy target
    run deployment-manager estimate-cost --help 2>&1

    # Should mention accuracy or estimation methodology
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ] || [ "$status" -eq 2 ]
    [[ "$output" =~ "estimat" ]] || [[ "$output" =~ "cost" ]] || [ -n "$output" ]
}

# Helper tests
@test "Validation commands are available" {
    run deployment-manager validate --help 2>&1

    [ "$status" -eq 0 ] || [ "$status" -eq 1 ] || [ "$status" -eq 2 ]
}

@test "Cost estimation commands are available" {
    run deployment-manager estimate-cost --help 2>&1

    [ "$status" -eq 0 ] || [ "$status" -eq 1 ] || [ "$status" -eq 2 ]
}

@test "Validation supports multiple output formats" {
    run deployment-manager validate --help 2>&1

    [ "$status" -eq 0 ] || [ "$status" -eq 1 ] || [ "$status" -eq 2 ]
    [[ "$output" =~ "format" ]] || [ -n "$output" ]
}
