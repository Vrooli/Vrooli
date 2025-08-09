#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Source var.sh first to get all the var_ variables
# BATS_TEST_DIRNAME is set by bats to the directory containing this test file
source "${BATS_TEST_DIRNAME}/../utils/var.sh"

# Load test infrastructure using var_ variables
source "${var_SCRIPTS_TEST_DIR}/fixtures/setup.bash"

# Load BATS helpers using var_ variables
load "${var_SCRIPTS_TEST_DIR}/helpers/bats-support/load"
load "${var_SCRIPTS_TEST_DIR}/helpers/bats-assert/load"

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/permissions.sh"

# Note: We can't easily mock log functions because the script sources log.sh
# which overrides any mocks we define. The tests will use the real log functions.

# Setup and teardown for test environment
setup() {
    # Create test directory structure
    export TEST_DIR="${BATS_TEST_TMPDIR}/permissions_test_$$"
    export TEST_SCRIPTS_DIR="${TEST_DIR}/scripts"
    export TEST_POSTGRES_DIR="${TEST_DIR}/postgres"
    
    mkdir -p "$TEST_SCRIPTS_DIR/subdir"
    mkdir -p "$TEST_POSTGRES_DIR"
    
    # Create test files with different extensions
    touch "$TEST_SCRIPTS_DIR/script1.sh"
    touch "$TEST_SCRIPTS_DIR/script2.sh"
    touch "$TEST_SCRIPTS_DIR/test1.bats"
    touch "$TEST_SCRIPTS_DIR/subdir/script3.sh"
    touch "$TEST_SCRIPTS_DIR/subdir/test2.bats"
    touch "$TEST_SCRIPTS_DIR/readme.txt"  # Should be ignored
    touch "$TEST_SCRIPTS_DIR/config.json" # Should be ignored
    
    # Create postgres entrypoint scripts
    touch "$TEST_POSTGRES_DIR/init.sh"
    touch "$TEST_POSTGRES_DIR/setup.bats"
    
    # Make files non-executable initially
    chmod 644 "$TEST_SCRIPTS_DIR"/*.sh "$TEST_SCRIPTS_DIR"/*.bats 2>/dev/null || true
    chmod 644 "$TEST_SCRIPTS_DIR/subdir"/*.sh "$TEST_SCRIPTS_DIR/subdir"/*.bats 2>/dev/null || true
    chmod 644 "$TEST_POSTGRES_DIR"/*.sh "$TEST_POSTGRES_DIR"/*.bats 2>/dev/null || true
}

teardown() {
    # Clean up test directory
    rm -rf "$TEST_DIR"
}

# Test that the script can be sourced successfully
@test "sourcing permissions.sh defines required functions" {
    run bash -c "source '$SCRIPT_PATH' && declare -f permissions::make_files_in_dir_executable && declare -f permissions::make_scripts_executable"
    assert_success
    assert_output --partial "permissions::make_files_in_dir_executable"
    assert_output --partial "permissions::make_scripts_executable"
}

# Test making files in a directory executable
@test "permissions::make_files_in_dir_executable makes .sh and .bats files executable" {
    # Verify files are not executable initially
    [ ! -x "$TEST_SCRIPTS_DIR/script1.sh" ]
    [ ! -x "$TEST_SCRIPTS_DIR/test1.bats" ]
    
    # Run the function
    run bash -c "
        source '$SCRIPT_PATH'
        permissions::make_files_in_dir_executable '$TEST_SCRIPTS_DIR'
    "
    
    assert_success
    
    # Verify .sh and .bats files are now executable
    [ -x "$TEST_SCRIPTS_DIR/script1.sh" ]
    [ -x "$TEST_SCRIPTS_DIR/script2.sh" ]
    [ -x "$TEST_SCRIPTS_DIR/test1.bats" ]
    [ -x "$TEST_SCRIPTS_DIR/subdir/script3.sh" ]
    [ -x "$TEST_SCRIPTS_DIR/subdir/test2.bats" ]
    
    # Verify other files remain non-executable
    [ ! -x "$TEST_SCRIPTS_DIR/readme.txt" ]
    [ ! -x "$TEST_SCRIPTS_DIR/config.json" ]
    
    # Verify output contains success message
    assert_output --partial "Made 5 script(s) in $TEST_SCRIPTS_DIR executable"
}

# Test handling of non-existent directory
@test "permissions::make_files_in_dir_executable fails for non-existent directory" {
    run bash -c "
        source '$SCRIPT_PATH'
        permissions::make_files_in_dir_executable '/non/existent/directory'
    "
    
    assert_failure
    assert_output --partial "Directory not found: /non/existent/directory"
}

# Test empty directory handling
@test "permissions::make_files_in_dir_executable handles empty directory" {
    local empty_dir="${TEST_DIR}/empty"
    mkdir -p "$empty_dir"
    
    run bash -c "
        source '$SCRIPT_PATH'
        permissions::make_files_in_dir_executable '$empty_dir'
    "
    
    assert_success
    assert_output --partial "No scripts found in $empty_dir"
}

# Test directory with no matching files
@test "permissions::make_files_in_dir_executable handles directory with no .sh or .bats files" {
    local no_scripts_dir="${TEST_DIR}/no_scripts"
    mkdir -p "$no_scripts_dir"
    touch "$no_scripts_dir/readme.txt"
    touch "$no_scripts_dir/config.json"
    touch "$no_scripts_dir/data.csv"
    
    run bash -c "
        source '$SCRIPT_PATH'
        permissions::make_files_in_dir_executable '$no_scripts_dir'
    "
    
    assert_success
    assert_output --partial "No scripts found in $no_scripts_dir"
}

# Test permissions::make_scripts_executable with all directories
@test "permissions::make_scripts_executable processes both scripts and postgres directories" {
    # Verify files are not executable initially
    [ ! -x "$TEST_SCRIPTS_DIR/script1.sh" ]
    [ ! -x "$TEST_POSTGRES_DIR/init.sh" ]
    
    run bash -c "
        source '$SCRIPT_PATH'
        export var_SCRIPTS_DIR='$TEST_SCRIPTS_DIR'
        export var_POSTGRES_ENTRYPOINT_DIR='$TEST_POSTGRES_DIR'
        permissions::make_scripts_executable
    "
    
    assert_success
    
    # Verify both directories were processed
    assert_output --partial "Making scripts in $TEST_SCRIPTS_DIR executable"
    assert_output --partial "Making scripts in $TEST_POSTGRES_DIR executable"
    
    # Verify files are now executable
    [ -x "$TEST_SCRIPTS_DIR/script1.sh" ]
    [ -x "$TEST_SCRIPTS_DIR/test1.bats" ]
    [ -x "$TEST_POSTGRES_DIR/init.sh" ]
    [ -x "$TEST_POSTGRES_DIR/setup.bats" ]
}

# Test permissions::make_scripts_executable without postgres directory
@test "permissions::make_scripts_executable works without postgres directory" {
    run bash -c "
        source '$SCRIPT_PATH'
        export var_SCRIPTS_DIR='$TEST_SCRIPTS_DIR'
        unset var_POSTGRES_ENTRYPOINT_DIR
        permissions::make_scripts_executable
    "
    
    assert_success
    
    # Verify only scripts directory was processed
    assert_output --partial "Making scripts in $TEST_SCRIPTS_DIR executable"
    refute_output --partial "postgres"
    
    # Verify scripts files are executable
    [ -x "$TEST_SCRIPTS_DIR/script1.sh" ]
    [ -x "$TEST_SCRIPTS_DIR/test1.bats" ]
}

# Test permissions::make_scripts_executable with non-existent postgres directory
@test "permissions::make_scripts_executable skips non-existent postgres directory" {
    run bash -c "
        source '$SCRIPT_PATH'
        export var_SCRIPTS_DIR='$TEST_SCRIPTS_DIR'
        export var_POSTGRES_ENTRYPOINT_DIR='/non/existent/postgres'
        permissions::make_scripts_executable
    "
    
    assert_success
    
    # Verify only scripts directory was processed
    assert_output --partial "Making scripts in $TEST_SCRIPTS_DIR executable"
    refute_output --partial "/non/existent/postgres"
}

# Test that var_SCRIPTS_DIR is required
@test "permissions::make_scripts_executable fails without var_SCRIPTS_DIR" {
    run bash -c "
        source '$SCRIPT_PATH'
        unset var_SCRIPTS_DIR
        permissions::make_scripts_executable
    "
    
    assert_failure
    assert_output --partial "var_SCRIPTS_DIR must be set"
}

# Test handling of files with spaces in names
@test "permissions::make_files_in_dir_executable handles files with spaces" {
    local space_dir="${TEST_DIR}/with spaces"
    mkdir -p "$space_dir"
    touch "$space_dir/my script.sh"
    touch "$space_dir/test file.bats"
    chmod 644 "$space_dir"/*.sh "$space_dir"/*.bats
    
    run bash -c "
        source '$SCRIPT_PATH'
        permissions::make_files_in_dir_executable '$space_dir'
    "
    
    assert_success
    [ -x "$space_dir/my script.sh" ]
    [ -x "$space_dir/test file.bats" ]
    assert_output --partial "Made 2 script(s)"
}

# Test running script directly
@test "running permissions.sh directly executes permissions::make_scripts_executable" {
    # Note: When the script runs directly, it sources var.sh which sets var_SCRIPTS_DIR 
    # to the actual scripts directory, not our test directory. We can't easily override this
    # without modifying the script itself. So we test that:
    # 1. The script runs without error
    # 2. It outputs the expected header message (with the real scripts dir)
    
    run bash "$SCRIPT_PATH"
    
    assert_success
    
    # Verify the function was called (it will process the real scripts directory)
    assert_output --partial "Making scripts in"
    assert_output --partial "executable"
    
    # Since we can't control which directories it processes when run directly,
    # we just verify it completes successfully. The actual functionality is
    # tested in the other test cases where we can control the variables.
}