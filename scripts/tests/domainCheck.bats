#!/usr/bin/env bats
bats_require_minimum_version 1.5.0
load '__testHelper.bash'

SCRIPT_PATH="$BATS_TEST_DIRNAME/../domainCheck.sh"
. "$SCRIPT_PATH"

@test "Script exits with usage when no arguments are provided" {
    run "$SCRIPT_PATH"
    assert_equal "$status" $ERROR_USAGE
    expected_usage="Usage: "
    assert_output --partial "$expected_usage"
}

@test "Script exits with usage when incorrect number of arguments are provided" {
    run "$SCRIPT_PATH" "5.6.7.8" "http://example.com" "extra_arg"
    assert_equal "$status" $ERROR_USAGE
    expected_usage="Usage: "
    assert_output --partial "$expected_usage"
}

@test "Script outputs 'remote' when current IPv4 matches SITE_IP and domain IP is valid" {
    stub_repeated dig "5.6.7.8"
    stub_repeated curl "5.6.7.8"
    run "$SCRIPT_PATH" "5.6.7.8" "http://example.com"
    assert_equal "$status" 0
    last_line=$(echo "$output" | tail -n 1)
    assert_equal "$last_line" "remote"
}

@test "Script outputs 'remote' when current IPv6 matches SITE_IP and domain IP is valid" {
    stub_repeated dig "2001:db8::1234"
    stub_repeated curl "2001:db8::1234"
    run "$SCRIPT_PATH" "2001:db8::1234" "http://example.com"
    assert_equal "$status" 0
    last_line=$(echo "$output" | tail -n 1)
    assert_equal "$last_line" "remote"
}

@test "Script exits with error when SITE_IP does not match domain IP" {
    stub_repeated dig "1.2.3.4"
    stub_repeated curl "5.6.7.8"
    run "$SCRIPT_PATH" "5.6.7.8" "http://example.com"
    assert_equal "$status" $ERROR_INVALID_SITE_IP
    last_line=$(echo "$output" | tail -n 1)
    refute [[ "$last_line" == "remote" ]]
    refute [[ "$last_line" == "local" ]]
}

@test "Script outputs 'local' when current IPv4 does not match SITE_IP" {
    stub_repeated dig "5.6.7.8"
    stub_repeated curl "1.2.3.4"
    run "$SCRIPT_PATH" "5.6.7.8" "http://example.com"
    assert_equal "$status" 0
    last_line=$(echo "$output" | tail -n 1)
    assert_equal "$last_line" "local"
}

@test "Script outputs 'local' when current IPv6 does not match SITE_IP" {
    stub_repeated dig "2001:db8::1234"
    stub_repeated curl "2001:db8::5678"
    run "$SCRIPT_PATH" "2001:db8::1234" "http://example.com"
    assert_equal "$status" 0
    last_line=$(echo "$output" | tail -n 1)
    assert_equal "$last_line" "local"
}

@test "Script handles case when domain cannot be resolved" {
    stub_repeated dig ""
    stub_repeated curl "1.2.3.4"
    # run "$SCRIPT_PATH" "5.6.7.8" "http://nonexistentdomain.com"
    run "$SCRIPT_PATH" "5.6.7.8" "http://nonexistentdomain.com"
    assert_equal "$status" $ERROR_DOMAIN_RESOLVE
    last_line=$(echo "$output" | tail -n 1)
    refute [[ "$last_line" == "remote" ]]
    refute [[ "$last_line" == "local" ]]
}

@test "Script handles case when current IP cannot be retrieved" {
    stub_repeated dig "5.6.7.8"
    stub_repeated curl ""
    run "$SCRIPT_PATH" "5.6.7.8" "http://example.com"
    assert_equal "$status" $ERROR_CURRENT_IP_FAIL
    last_line=$(echo "$output" | tail -n 1)
    refute [[ "$last_line" == "remote" ]]
    refute [[ "$last_line" == "local" ]]
}

@test "Script handles invalid SERVER_URL" {
    run "$SCRIPT_PATH" "5.6.7.8" "invalid_url"
    assert_equal "$status" $ERROR_USAGE
    last_line=$(echo "$output" | tail -n 1)
    refute [[ "$last_line" == "remote" ]]
    refute [[ "$last_line" == "local" ]]
}

@test "Script handles invalid SITE_IP" {
    run "$SCRIPT_PATH" "invalid_ip" "http://example.com"
    assert_equal "$status" $ERROR_INVALID_SITE_IP

    run "$SCRIPT_PATH" "2001:db8::xxxx" "http://example.com"
    assert_equal "$status" $ERROR_INVALID_SITE_IP
}
