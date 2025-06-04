#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Ensure the temporary deps dir is cleaned up even if a test crashes
TMP_DEPS_DIR="$BATS_TEST_DIRNAME/deps_dir"
trap 'rm -rf "$TMP_DEPS_DIR"' EXIT

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/bats.sh"

setup() {
    # Prepare a clean temporary dependencies dir for each test
    rm -rf "$TMP_DEPS_DIR"
    mkdir -p "$TMP_DEPS_DIR"
    # Save original CWD and override the deps directory
    export ORIGINAL_DIR="$PWD"
    export BATS_DEPENDENCIES_DIR="$TMP_DEPS_DIR"
}

teardown() {
    # Restore CWD and clean up
    cd "$ORIGINAL_DIR"
    rm -rf "$TMP_DEPS_DIR"
}

@test "sourcing setupBats.sh defines functions" {
    run bash -c "source '$SCRIPT_PATH' && declare -f bats::create_dependencies_dir bats::install_dependency bats::install_core bats::install"
    [ "$status" -eq 0 ]
    [[ "$output" =~ bats::create_dependencies_dir ]]
    [[ "$output" =~ bats::install_dependency ]]
    [[ "$output" =~ bats::install_core ]]
    [[ "$output" =~ bats::install ]]
}

@test "bats::create_dependencies_dir creates directory when missing" {
    source "$SCRIPT_PATH"
    rm -rf "$BATS_DEPENDENCIES_DIR"
    bats::create_dependencies_dir
    [ -d "$BATS_DEPENDENCIES_DIR" ]
}

@test "bats::install_dependency clones absent dependency" {
    source "$SCRIPT_PATH"
    # Stub git clone to just create the target directory
    git() { [[ "$1" == "clone" ]] && mkdir -p "$BATS_DEPENDENCIES_DIR/$3"; }

    run bats::install_dependency "https://example.com/foo.git" "foo"
    [ "$status" -eq 0 ]
    [ -d "$BATS_DEPENDENCIES_DIR/foo" ]
    [[ "$output" =~ installed\ successfully ]]
}

@test "bats::install_dependency skips when already present" {
    source "$SCRIPT_PATH"
    mkdir -p "$BATS_DEPENDENCIES_DIR/bar"

    run bats::install_dependency "any" "bar"
    [ "$status" -eq 0 ]
    [[ "$output" =~ already\ installed ]]
}

@test "bats::install_core clones and installs when missing" {
    source "$SCRIPT_PATH"
    # Stub git clone and sudo install.sh
    git() { [[ "$1" == "clone" ]] && mkdir -p "$BATS_DEPENDENCIES_DIR/bats-core"; }
    sudo() { :; }

    run bats::install_core
    [ "$status" -eq 0 ]
    [ -d "$BATS_DEPENDENCIES_DIR/bats-core" ]
    [[ "$output" =~ installed\ successfully ]]
}

@test "bats::install_core skips when already installed" {
    source "$SCRIPT_PATH"
    mkdir -p "$BATS_DEPENDENCIES_DIR/bats-core"

    run bats::install_core
    [ "$status" -eq 0 ]
    [[ "$output" =~ already\ installed ]]
} 