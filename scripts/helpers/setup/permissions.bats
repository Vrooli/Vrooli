#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/permissions.sh"

@test "sourcing permissions.sh defines permissions::make_scripts_executable function" {
    run bash -c "source '$SCRIPT_PATH' && declare -f permissions::make_scripts_executable"
    [ "$status" -eq 0 ]
    [[ "$output" =~ permissions::make_scripts_executable ]]
}

@test "permissions::make_scripts_executable prints header and success messages" {
    run bash -c "source '$SCRIPT_PATH'; find(){ return 0; }; permissions::make_scripts_executable"
    [ "$status" -eq 0 ]
} 