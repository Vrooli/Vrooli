# shellcheck shell=bash

# Exit codes
export ERROR_MOCK_DIR_CREATION=65 # Failed to create mock binary directory
export ERROR_LINK_FAIL=66         # Failed to link binstub
export ERROR_WRITE_PLAN_FAIL=67   # Failed to write stub plan
export ERROR_TRANSFORM_FAIL=68    # Failed to transform program name to uppercase
export ERROR_CLEANUP_FAIL=69      # Failed to clean up stub files

# Set a temporary directory for mock binaries specific to BATS (Bash Automated Testing System)
export BATS_MOCK_TMPDIR="${BATS_TMPDIR}"
export BATS_MOCK_BINDIR="${BATS_MOCK_TMPDIR}/bin"

# Prepend the mock binary directory to the PATH to prioritize mock binaries over actual commands
PATH="$BATS_MOCK_BINDIR:$PATH"

# stub function: Creates a mock command that outputs specified responses in tests
stub() {
    # Capture the program name and remove it from the argument list
    local program="$1"
    shift

    # Check if the mock binary directory exists and is writable, create if necessary
    if [ ! -d "${BATS_MOCK_BINDIR}" ] || [ ! -w "${BATS_MOCK_BINDIR}" ]; then
        mkdir -p "${BATS_MOCK_BINDIR}" || {
            echo "Error: Failed to create or write to mock binary directory at ${BATS_MOCK_BINDIR}" >&2
            return $ERROR_MOCK_DIR_CREATION
        }
    fi

    # Define the path to the binstub script and ensure it is executable
    BINSTUB_PATH="${BASH_SOURCE[0]%/*}/__binstub"
    chmod +x "$BINSTUB_PATH"

    # Attempt to link the binstub script to the mock program name
    if [ -f "${BATS_MOCK_BINDIR}/${program}" ] && [ ! -w "${BATS_MOCK_BINDIR}/${program}" ]; then
        echo "Error: Existing link is not writable: ${BATS_MOCK_BINDIR}/${program}" >&2
        return $ERROR_LINK_FAIL
    fi
    ln -sf "$BINSTUB_PATH" "${BATS_MOCK_BINDIR}/${program}" || {
        echo "Error: Failed to link __binstub for ${program}" >&2
        return $ERROR_LINK_FAIL
    }

    # Write the mock output plan (sequence of responses) to a file to be used during tests
    if [ ! -w "${BATS_MOCK_TMPDIR}" ]; then
        echo "Error: Cannot write to temporary directory ${BATS_MOCK_TMPDIR}" >&2
        return $ERROR_WRITE_PLAN_FAIL
    fi
    printf "%s\n" "$@" >"${BATS_MOCK_TMPDIR}/${program}-stub-plan" || {
        echo "Error: Failed to write stub plan for ${program}" >&2
        return $ERROR_WRITE_PLAN_FAIL
    }
}

# stub_repeated function: A variant of stub for repeated commands, adding an environment variable for tracking
stub_repeated() {
    local program="$1"
    # Convert program name to uppercase with underscores for environment variable prefix
    local prefix
    prefix="$(echo "$program" | tr 'a-z-' 'A-Z_')" || {
        echo "Error: Failed to transform program name to uppercase" >&2
        return $ERROR_TRANSFORM_FAIL
    }

    # Set environment variable to prevent index incrementation (simulate same output for repeated calls)
    export "${prefix}_STUB_NOINDEX"=1

    # Call stub with the program and additional arguments
    stub "$@" || {
        echo "Error: Failed to set up repeated stub for ${program}" >&2
        return $?
    }
}

# unstub function: Removes the mock command and cleans up temporary files
unstub() {
    local program="$1"
    # Define prefix for environment variable tracking
    local prefix
    prefix="$(echo "$program" | tr 'a-z-' 'A-Z_')" || {
        echo "Error: Failed to transform program name to uppercase" >&2
        return $ERROR_TRANSFORM_FAIL
    }
    # Define path to the mock binary for the program
    local path="${BATS_MOCK_BINDIR}/${program}"

    # Set an environment variable to indicate the mock has ended
    export "${prefix}_STUB_END"=1

    # Initialize a status variable to track command exit status
    local STATUS=0
    if [ -x "$path" ]; then
        # If the mock binary exists, execute it, capturing its exit status
        "$path" || STATUS="$?"
    fi

    # Remove the mock binary and any temporary files related to this program
    rm -f "$path" "${BATS_MOCK_TMPDIR}/${program}-stub-plan" "${BATS_MOCK_TMPDIR}/${program}-stub-run" || {
        echo "Error: Failed to clean up files for ${program}" >&2
        return $ERROR_CLEANUP_FAIL
    }

    # Return the exit status of the mock command or zero if it wasn't executed
    return "$STATUS"
}
