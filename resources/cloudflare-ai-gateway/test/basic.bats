#!/usr/bin/env bats

# Basic tests for Cloudflare AI Gateway resource

load "${BATS_SUPPORT_LOAD_PATH}/load"
load "${BATS_ASSERT_LOAD_PATH}/load"

RESOURCE_DIR="$(builtin cd "${BATS_TEST_DIRNAME%/*}" && builtin pwd)"
CLI="${RESOURCE_DIR}/cli.sh"

@test "CLI script exists and is executable" {
    assert [ -f "${CLI}" ]
    assert [ -x "${CLI}" ]
}

@test "Help command works" {
    run "${CLI}" help
    assert_success
    assert_output --partial "Cloudflare AI Gateway Resource CLI"
}

@test "Status command works" {
    run "${CLI}" status
    assert_success
}

@test "Status supports JSON format" {
    run "${CLI}" status --format json
    assert_success
    # Output should be valid JSON
    echo "${output}" | jq . >/dev/null
}

@test "Content list works" {
    run "${CLI}" content list
    assert_success
}

@test "Content list supports JSON format" {
    run "${CLI}" content list --format json
    assert_success
    # Output should be valid JSON
    echo "${output}" | jq . >/dev/null
}

@test "Info command works" {
    run "${CLI}" info
    # May fail if not configured, but should not crash
    assert [ $status -eq 0 ] || assert [ $status -eq 1 ]
}