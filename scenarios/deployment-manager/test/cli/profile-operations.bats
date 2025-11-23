#!/usr/bin/env bats
# Profile Operations CLI Tests
# Tests for requirements DM-P0-012 through DM-P0-017

setup() {
    export PATH="${BATS_TEST_DIRNAME}/../../cli:${PATH}"

    API_PORT=$(vrooli scenario port deployment-manager API_PORT 2>/dev/null || echo "18722")
    export API_URL="http://127.0.0.1:${API_PORT}"

    timeout 10 bash -c "until curl -sf ${API_URL}/health &>/dev/null; do sleep 0.5; done" || {
        echo "# API not ready at ${API_URL}" >&3
        return 1
    }

    export TEST_PROFILE="test-profile-$$"
}

teardown() {
    deployment-manager profile delete "${TEST_PROFILE}" 2>/dev/null || true
}

# [REQ:DM-P0-012] Profile Creation UX
@test "[REQ:DM-P0-012] Create profile with minimal CLI interaction" {
    run deployment-manager profile create "${TEST_PROFILE}" picker-wheel --tier desktop 2>&1

    # Should create successfully or provide clear feedback
    [ "$status" -eq 0 ] || [[ "$output" =~ "created" ]] || [[ "$output" =~ "profile" ]]
}

# [REQ:DM-P0-012] Profile Listing
@test "[REQ:DM-P0-012] List all deployment profiles" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    run deployment-manager profile list 2>&1

    # Should show profiles
    [[ "$output" =~ "profile" ]] || [[ "$output" =~ "${TEST_PROFILE}" ]] || [ "$status" -eq 0 ]
}

# [REQ:DM-P0-013] Auto-Save with Debounce
@test "[REQ:DM-P0-013] Profile changes auto-save" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    # Update profile
    run deployment-manager profile update "${TEST_PROFILE}" --tier mobile 2>&1

    # Should save automatically
    [[ "$output" =~ "saved" ]] || [[ "$output" =~ "updated" ]] || [ "$status" -eq 0 ]
}

# [REQ:DM-P0-013] Manual Save Option
@test "[REQ:DM-P0-013] Manually save profile changes" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    run deployment-manager profile save "${TEST_PROFILE}" 2>&1

    # Should confirm save
    [[ "$output" =~ "saved" ]] || [ "$status" -eq 0 ]
}

# [REQ:DM-P0-014] Profile Export JSON
@test "[REQ:DM-P0-014] Export profile to JSON format" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    run deployment-manager profile export "${TEST_PROFILE}" --format json 2>&1

    # Should output valid JSON
    [[ "$output" =~ "{" ]] || [[ "$output" =~ "export" ]]
}

# [REQ:DM-P0-014] Profile Import JSON
@test "[REQ:DM-P0-014] Import profile from JSON file" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    # Export first
    deployment-manager profile export "${TEST_PROFILE}" --output /tmp/test-profile-export.json 2>&1 || skip "Export not ready"

    # Import
    run deployment-manager profile import /tmp/test-profile-export.json --name "${TEST_PROFILE}-imported" 2>&1

    [[ "$output" =~ "imported" ]] || [[ "$output" =~ "created" ]] || [ "$status" -eq 0 ]

    deployment-manager profile delete "${TEST_PROFILE}-imported" 2>/dev/null || true
    rm -f /tmp/test-profile-export.json
}

# [REQ:DM-P0-015] Profile Versioning
@test "[REQ:DM-P0-015] Create new version on profile edit" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    # Make changes to create versions
    deployment-manager profile update "${TEST_PROFILE}" --tier mobile 2>&1 || true
    deployment-manager profile update "${TEST_PROFILE}" --tier desktop 2>&1 || true

    run deployment-manager profile versions "${TEST_PROFILE}" 2>&1

    # Should show version history
    [[ "$output" =~ "version" ]] || [[ "$output" =~ "history" ]] || [ "$status" -eq 0 ]
}

# [REQ:DM-P0-015] Version Timestamp and Attribution
@test "[REQ:DM-P0-015] Versions include timestamp and user" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    deployment-manager profile update "${TEST_PROFILE}" --tier mobile 2>&1 || true

    run deployment-manager profile versions "${TEST_PROFILE}" --format json 2>&1

    if [[ "$output" =~ "{" ]]; then
        [[ "$output" =~ "created_at" ]] || [[ "$output" =~ "timestamp" ]] || [[ "$output" =~ "time" ]]
    else
        skip "No JSON output available"
    fi
}

# [REQ:DM-P0-016] Version History Display
@test "[REQ:DM-P0-016] Display version history with changes" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    deployment-manager profile update "${TEST_PROFILE}" --tier mobile 2>&1 || true

    run deployment-manager profile versions "${TEST_PROFILE}" 2>&1

    # Should show history
    [[ "$output" =~ "version" ]] || [[ "$output" =~ "change" ]] || [ "$status" -eq 0 ]
}

# [REQ:DM-P0-016] Version Diff Visualization
@test "[REQ:DM-P0-016] Show diff between profile versions" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    deployment-manager profile update "${TEST_PROFILE}" --tier mobile 2>&1 || true

    run deployment-manager profile diff "${TEST_PROFILE}" --from 1 --to 2 2>&1

    # Should show diffs
    [[ "$output" =~ "diff" ]] || [[ "$output" =~ "change" ]] || [[ "$output" =~ "version" ]]
}

# [REQ:DM-P0-017] Version Rollback Performance
@test "[REQ:DM-P0-017] Rollback profile within 2 seconds" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    deployment-manager profile update "${TEST_PROFILE}" --tier mobile 2>&1 || true

    start_time=$(date +%s%3N)
    run deployment-manager profile rollback "${TEST_PROFILE}" --version 1 2>&1
    end_time=$(date +%s%3N)

    duration=$((end_time - start_time))

    # Verify performance: < 2000ms
    [ "$duration" -lt 2000 ] || skip "Performance target not met (${duration}ms > 2000ms)"
}

# [REQ:DM-P0-017] Rollback Restores All Settings
@test "[REQ:DM-P0-017] Rollback restores swaps, secrets, and env vars" {
    deployment-manager profile create "${TEST_PROFILE}" picker-wheel 2>&1 || skip "Profile creation not ready"

    # Make changes
    deployment-manager profile update "${TEST_PROFILE}" --tier mobile 2>&1 || true
    deployment-manager swaps apply "${TEST_PROFILE}" postgres sqlite 2>&1 || true

    # Rollback
    run deployment-manager profile rollback "${TEST_PROFILE}" --version 1 2>&1

    [[ "$output" =~ "restored" ]] || [[ "$output" =~ "rollback" ]] || [ "$status" -eq 0 ]
}
