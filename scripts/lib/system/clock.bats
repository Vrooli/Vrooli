#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/clock.sh"

@test "sourcing clock.sh defines clock::fix and clock::is_accurate functions" {
    run bash -c "source '$SCRIPT_PATH' && declare -f clock::fix && declare -f clock::is_accurate"
    [ "$status" -eq 0 ]
    [[ "$output" =~ clock::fix ]]
    [[ "$output" =~ clock::is_accurate ]]
}

@test "clock::fix skips sync when clock is already accurate" {
    # Stub clock::is_accurate to return success (clock is accurate)
    run bash -c "source '$SCRIPT_PATH'; clock::is_accurate(){ return 0; }; date(){ echo 'TEST_DATE'; }; clock::fix"
    [ "$status" -eq 0 ]
    # Should see header and success message, but no sync attempts
    echo "$output" | grep -Fq "[HEADER]  Making sure the system clock is accurate"
    echo "$output" | grep -Fq "[SUCCESS] System clock is accurate: TEST_DATE"
    # Should NOT see any sync attempts
    ! echo "$output" | grep -Fq "hwclock"
    ! echo "$output" | grep -Fq "ntpdate"
    ! echo "$output" | grep -Fq "timedatectl"
}

@test "clock::fix attempts sync when clock is inaccurate and sudo available" {
    # Stub clock::is_accurate to fail, flow::can_run_sudo to succeed
    run bash -c "source '$SCRIPT_PATH'; clock::is_accurate(){ return 1; }; flow::can_run_sudo(){ return 0; }; sudo(){ echo \"SUDO_CALLED \$*\"; }; date(){ echo 'TEST_DATE'; }; system::is_command(){ [[ \"\$1\" == \"hwclock\" ]]; }; clock::fix"
    [ "$status" -eq 0 ]
    # Should see warning about adjustment needed
    echo "$output" | grep -Fq "[WARNING] System clock needs adjustment"
    # Should attempt hwclock sync
    echo "$output" | grep -Fq "SUDO_CALLED hwclock -s"
}

@test "clock::fix exits with error when clock inaccurate and no sudo" {
    # Stub clock::is_accurate to fail, flow::can_run_sudo to fail
    run bash -c "source '$SCRIPT_PATH'; clock::is_accurate(){ return 1; }; flow::can_run_sudo(){ return 1; }; ERROR_DEFAULT=1; clock::fix"
    [ "$status" -eq 1 ]
    # Should see error messages and options
    echo "$output" | grep -Fq "[ERROR]   System clock is inaccurate but cannot sync without sudo access"
    echo "$output" | grep -Fq "[WARNING] This may cause SSL/TLS certificate validation errors"
    echo "$output" | grep -Fq "Run setup with sudo: sudo ./scripts/manage.sh setup"
}

@test "clock::is_accurate returns success when time is within tolerance" {
    # Stub curl to return a date header that matches current time
    run bash -c "
        # Define mock functions
        system::is_command(){ [[ \"\$1\" == \"curl\" ]]; }
        curl(){ echo \"Date: \$(command date -R)\"; }
        date(){ if [[ \"\$1\" == \"+%s\" ]]; then command date +%s; else command date \"\$@\"; fi; }
        
        # Export functions to make them available in subshells
        export -f system::is_command curl date
        
        # Source script and run test
        source '$SCRIPT_PATH'
        clock::is_accurate
    "
    [ "$status" -eq 0 ]
    echo "$output" | grep -Fq "[INFO]    System clock is accurate"
}

@test "clock::is_accurate returns failure when time differs significantly" {
    # Stub curl to return a date header 10 minutes in the future
    run bash -c "
        # Define mock functions
        system::is_command(){ [[ \"\$1\" == \"curl\" ]]; }
        curl(){ echo \"Date: \$(command date -R -d '+10 minutes')\"; }
        date(){ command date \"\$@\"; }
        
        # Export functions to make them available in subshells
        export -f system::is_command curl date
        
        # Source script and run test
        source '$SCRIPT_PATH'
        clock::is_accurate
    "
    [ "$status" -eq 1 ]
    echo "$output" | grep -Fq "[WARNING] System clock is off by"
}

@test "clock::is_accurate fallback check when no network available" {
    # Stub curl to not exist
    run bash -c "
        # Source script first to load dependencies
        source '$SCRIPT_PATH'
        
        # Override system::is_command AFTER sourcing
        system::is_command(){ return 1; }
        
        # Export functions to make them available in subshells
        export -f system::is_command
        
        # Run test
        clock::is_accurate
    "
    [ "$status" -eq 0 ]
    echo "$output" | grep -Fq "Cannot verify exact time accuracy (no network time source), but year is reasonable"
} 