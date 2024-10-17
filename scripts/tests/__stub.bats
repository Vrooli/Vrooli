#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

HERE="$BATS_TEST_DIRNAME"

# Set up the stub directory and PATH
export BATS_TMPDIR="${HERE}/tmp" # WARNING: We use `rm -rf "$BATS_TMPDIR"` in teardown. Make sure it's safe to delete!
export BATS_MOCK_BINDIR="${BATS_TMPDIR}/bin"
mkdir -p "$BATS_MOCK_BINDIR"
export PATH="$BATS_MOCK_BINDIR:$PATH"

chmod +x "${HERE}/__stub.bash"
chmod +x "${HERE}/__binstub"
load "${HERE}/__stub.bash"

# Common exit codes
E_COMMAND_NOT_FOUND=127
# Exit codes (copied from script we're testing)
E_MOCK_DIR_CREATION=65
E_LINK_FAIL=66
E_WRITE_PLAN_FAIL=67
E_TRANSFORM_FAIL=68
E_CLEANUP_FAIL=69

E_NO_PLAN=74        # No stub plan file found
E_MISSING_TMPDIR=75 # BATS_MOCK_TMPDIR not set
E_OUTPUT_FAIL=76    # Failed to output stub plan
E_LOGGING_FAIL=77   # Failed to log the stub call

setup() {
    # Reset any environment variables or settings here if necessary
    : # Noop for now
}

teardown() {
    # Clean up any temporary files or directories
    rm -rf "$BATS_TMPDIR"
}

@test "Check environment and path settings" {
    # First, stub echo to create the executable link
    stub "echo" "Test"
    echo "BATS_TMPDIR: $BATS_TMPDIR"
    echo "BATS_MOCK_BINDIR: $BATS_MOCK_BINDIR"
    echo "PATH: $PATH"
    [ -d "$BATS_MOCK_BINDIR" ] || false
    [ -x "${BATS_MOCK_BINDIR}/echo" ] || false # Checks if the link exists and is executable
}

@test "Check if mock command is linked and outputs correctly for a nonexistent command" {
    stub "myecho" "Hello, world!"
    [ -L "${BATS_MOCK_BINDIR}/myecho" ] && [ -x "${BATS_MOCK_BINDIR}/myecho" ] || false # Ensure link exists and is executable

    # Act and Assert
    run myecho
    [ "$status" -eq 0 ]
    [ "$output" == "Hello, world!" ]
}

@test "Check if mock command is linked and outputs correctly for an existing command" {
    stub "ls" "Hello, world!"
    [ -L "${BATS_MOCK_BINDIR}/ls" ] && [ -x "${BATS_MOCK_BINDIR}/ls" ] || false # Ensure link exists and is executable

    # Act and Assert
    run ls "test"
    [ "$status" -eq 0 ]
    [ "$output" == "Hello, world!" ]
}

@test "stub creates a mock command with specified output" {
    # Arrange
    stub "ls" "Hello, world!"

    # Act and Assert
    run ls
    [ "$status" -eq 0 ]
    [ "$output" == "Hello, world!" ]
}

@test "stub_repeated creates a mock command with consistent repeated output" {
    # Arrange
    stub_repeated "ls" "Repeat this message"

    # Act and Assert
    run ls
    [ "$status" -eq 0 ]
    [ "$output" == "Repeat this message" ]

    run ls
    [ "$status" -eq 0 ]
    [ "$output" == "Repeat this message" ]
}

@test "unstub removes the mock command and cleans up" {
    # Arrange
    stub "skrrt" "Temporary message"
    run skrrt
    [ "$status" -eq 0 ]
    [ "$output" == "Temporary message" ]

    # Act: remove the stub
    unstub "skrrt"

    # Assert: skrrt command should now return an error (not found)
    run -$E_COMMAND_NOT_FOUND skrrt
    [ "$status" -eq $E_COMMAND_NOT_FOUND ]
}

@test "stub without plan file outputs an error" {
    # Arrange
    stub "nonexistent_command"

    # Remove the plan file manually to simulate the error condition
    rm "${BATS_MOCK_TMPDIR}/nonexistent_command-stub-plan"

    # Act: Attempt to run the stub without a plan file
    run nonexistent_command

    # Assert: Expect specific error message since there’s no plan for this stub
    [ "$status" -eq $E_NO_PLAN ]
    [[ "$output" == *"No stub plan found for nonexistent_command"* ]]
}
