#!/usr/bin/env bats

# BTCPay Server Integration Tests

setup() {
    load '../../../__test/helpers/bats-support/load'
    load '../../../__test/helpers/bats-assert/load'
    
    export BTCPAY_CLI="${BATS_TEST_DIRNAME}/../cli.sh"
}

@test "BTCPay CLI exists and is executable" {
    [ -x "${BTCPAY_CLI}" ]
}

@test "BTCPay CLI shows help" {
    run "${BTCPAY_CLI}" help
    assert_success
    assert_output --partial "BTCPay Server Resource Management CLI"
}

@test "BTCPay status command works" {
    run "${BTCPAY_CLI}" status
    assert_success
    assert_output --partial "BTCPay Server Status Report"
}

@test "BTCPay status JSON format works" {
    run "${BTCPAY_CLI}" status --format json
    assert_success
    # Verify JSON structure
    echo "$output" | jq -e '.name == "btcpay"'
    echo "$output" | jq -e '.category == "execution"'
}

@test "BTCPay install check" {
    run "${BTCPAY_CLI}" status --format json
    assert_success
    local installed=$(echo "$output" | jq -r '.installed')
    # Just verify the field exists
    [[ "$installed" == "true" ]] || [[ "$installed" == "false" ]]
}