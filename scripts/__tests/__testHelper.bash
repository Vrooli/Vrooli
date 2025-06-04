#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Determine the directory containing this helper library file
LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Set up the stub directory and PATH based on test file directory
export BATS_TMPDIR="$(mktemp -d)"
export BATS_MOCK_BINDIR="${BATS_TMPDIR}/bin"
mkdir -p "$BATS_MOCK_BINDIR"
export PATH="$BATS_MOCK_BINDIR:$PATH"

chmod +x "${LIB_DIR}/helpers/bats-support/load.bash"
load "${LIB_DIR}/helpers/bats-support/load.bash"

chmod +x "${LIB_DIR}/helpers/bats-assert/load.bash"
load "${LIB_DIR}/helpers/bats-assert/load.bash"

chmod +x "${LIB_DIR}/__stub.bash"
chmod +x "${LIB_DIR}/__binstub"
load "${LIB_DIR}/__stub.bash"

setup() {
    # Reset any environment variables or settings here if necessary
    : # Noop for now
}

teardown() {
    # Unstub any stubbed commands. Add commands between `in` and `do`
    for cmd in dig curl tput; do
        if [ -x "${BATS_MOCK_BINDIR}/$cmd" ]; then
            unstub "$cmd"
        fi
    done
    # Clean up any temporary files or directories
    rm -rf "$BATS_TMPDIR"
}
