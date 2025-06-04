#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/internet.sh"

@test "sourcing internet.sh defines internet::check_connection function" {
    run bash -c "source '$SCRIPT_PATH' && declare -f internet::check_connection"
    [ "$status" -eq 0 ]
    [[ "$output" =~ internet::check_connection ]]
}

@test "internet::check_connection prints success when ping succeeds" {
    run bash -c "source '$SCRIPT_PATH'; ping(){ return 0; }; internet::check_connection"
    [ "$status" -eq 0 ]
    [[ "$output" =~ \[HEADER\]\ +Checking\ host\ internet\ access\.\.\. ]]
    [[ "$output" =~ \[SUCCESS\]\ +Host\ internet\ access:\ OK ]]
}

@test "internet::check_connection prints error and exits with correct code when ping fails" {
    run bash -c "export ERROR_NO_INTERNET=5; source '$SCRIPT_PATH'; ping(){ return 1; }; internet::check_connection"
    [ "$status" -eq 5 ]
    [[ "$output" =~ \[HEADER\]\ +Checking\ host\ internet\ access\.\.\. ]]
    [[ "$output" =~ \[ERROR\]\ +Host\ internet\ access:\ FAILED ]]
}