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
    run bash -c "source '$SCRIPT_PATH'; timeout(){ shift; \"\$@\"; }; ping(){ [[ \"\$1\" == \"-4\" ]] && return 0 || return 1; }; internet::check_connection"
    [ "$status" -eq 0 ]
    [[ "$output" =~ \[HEADER\]\ +Checking\ host\ internet\ access\.\.\. ]]
    [[ "$output" =~ \[SUCCESS\]\ +Host\ internet\ access\ to\ google\.com:\ OK ]]
}

@test "internet::check_connection prints error and exits with correct code when ping fails" {
    run bash -c "export ERROR_NO_INTERNET=5; source '$SCRIPT_PATH'; timeout(){ shift; \"\$@\"; }; ping(){ return 1; }; internet::check_connection"
    [ "$status" -eq 5 ]
    [[ "$output" =~ \[HEADER\]\ +Checking\ host\ internet\ access\.\.\. ]]
    [[ "$output" =~ \[ERROR\]\ +Host\ internet\ access\ to\ google\.com:\ FAILED ]]
}

@test "internet::check_connection uses IPv4 flag with ping" {
    # Test that ping is called with -4 flag by checking if it succeeds only with -4
    run bash -c "
        source '$SCRIPT_PATH'
        # Mock timeout to pass through arguments
        timeout() { shift; \"\$@\"; }
        # Mock ping that only succeeds if -4 is present
        ping() {
            # Only succeed if -4 flag is present
            for arg in \"\$@\"; do
                [[ \"\$arg\" == '-4' ]] && return 0
            done
            return 1
        }
        internet::check_connection
    "
    # If the test passes (status 0), it means ping was called with -4
    [ "$status" -eq 0 ]
    [[ "$output" =~ \[SUCCESS\] ]]
}