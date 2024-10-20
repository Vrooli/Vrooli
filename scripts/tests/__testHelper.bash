#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

HERE="$BATS_TEST_DIRNAME"

# Set up the stub directory and PATH
export BATS_TMPDIR="${HERE}/tmp" # WARNING: We use `rm -rf "$BATS_TMPDIR"` in teardown. Make sure it's safe to delete!
export BATS_MOCK_BINDIR="${BATS_TMPDIR}/bin"
mkdir -p "$BATS_MOCK_BINDIR"
export PATH="$BATS_MOCK_BINDIR:$PATH"

chmod +x "${HERE}/helpers/bats-support/load.bash"
load "${HERE}/helpers/bats-support/load.bash"

chmod +x "${HERE}/helpers/bats-assert/load.bash"
load "${HERE}/helpers/bats-assert/load.bash"

chmod +x "${HERE}/__stub.bash"
chmod +x "${HERE}/__binstub"
load "${HERE}/__stub.bash"

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
