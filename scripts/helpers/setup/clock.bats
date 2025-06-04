#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/clock.sh"

@test "sourcing time.sh defines clock::fix function" {
    run bash -c "source '$SCRIPT_PATH' && declare -f clock::fix"
    [ "$status" -eq 0 ]
    [[ "$output" =~ clock::fix ]]
}

@test "clock::fix prints header and info with stubbed date" {
    # Stub sudo to noop and date to return a fixed timestamp
    run bash -c "source '$SCRIPT_PATH'; sudo(){ :; }; date(){ echo 'TEST_DATE'; }; clock::fix"
    [ "$status" -eq 0 ]
    # Verify header and info output (warning may precede these)
    echo "$output" | grep -Fq "[HEADER]  Making sure the system clock is accurate"
    echo "$output" | grep -Fq "[INFO]    System clock is now: TEST_DATE"
}

@test "clock::fix runs hwclock when sudo is available" {
    # Stub flow::can_run_sudo to succeed, stub sudo to capture invocation, stub date and system::is_command
    run bash -c "source '$SCRIPT_PATH'; flow::can_run_sudo(){ return 0; }; sudo(){ echo \"SUDO_CALLED \$*\"; }; date(){ echo 'TEST_DATE'; }; system::is_command(){ return 0; }; clock::fix"
    [ "$status" -eq 0 ]
    # Verify header, sudo invocation, and info output
    echo "$output" | grep -Fq "[HEADER]  Making sure the system clock is accurate"
    echo "$output" | grep -Fq "SUDO_CALLED hwclock -s"
    echo "$output" | grep -Fq "[INFO]    System clock is now: TEST_DATE"
} 