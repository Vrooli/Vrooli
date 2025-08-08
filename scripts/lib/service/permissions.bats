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
    # Create temporary test directories
    local test_scripts_dir="/tmp/test_scripts_$$"
    local test_postgres_dir="/tmp/test_postgres_$$"
    
    mkdir -p "$test_scripts_dir" "$test_postgres_dir"
    
    # Run the test with properly set variables
    run bash -c "
        # Set required variables BEFORE sourcing the script
        export var_SCRIPTS_DIR='$test_scripts_dir'
        export var_POSTGRES_ENTRYPOINT_DIR='$test_postgres_dir'
        
        # Source the script (which will source its dependencies)
        source '$SCRIPT_PATH'
        
        # Run the function
        permissions::make_scripts_executable
    "
    
    # Clean up
    rm -rf "$test_scripts_dir" "$test_postgres_dir"
    
    # Check that the function succeeded
    [ "$status" -eq 0 ]
    
    # Verify output contains expected messages
    [[ "$output" =~ "Making scripts" ]]
} 