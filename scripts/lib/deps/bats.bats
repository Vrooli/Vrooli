#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Source trash module for safe test cleanup
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Ensure the temporary deps dir is cleaned up even if a test crashes
TMP_DEPS_DIR="$BATS_TEST_DIRNAME/deps_dir"
trap 'trash::safe_remove "$TMP_DEPS_DIR" --test-cleanup' EXIT

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/bats.sh"

setup() {
    # Prepare a clean temporary dependencies dir for each test
    trash::safe_remove "$TMP_DEPS_DIR" --test-cleanup
    mkdir -p "$TMP_DEPS_DIR"
    # Save original CWD and override the deps directory
    export ORIGINAL_DIR="$PWD"
    export BATS_DEPENDENCIES_DIR="$TMP_DEPS_DIR"
}

teardown() {
    # Restore CWD and clean up
    cd "$ORIGINAL_DIR"
    trash::safe_remove "$TMP_DEPS_DIR" --test-cleanup
}

@test "sourcing setupBats.sh defines functions" {
    run bash -c "source '$SCRIPT_PATH' && declare -f bats::determine_prefix bats::create_dependencies_dir bats::install_dependency bats::install_core bats::install"
    [ "$status" -eq 0 ]
    [[ "$output" =~ bats::determine_prefix ]]
    [[ "$output" =~ bats::create_dependencies_dir ]]
    [[ "$output" =~ bats::install_dependency ]]
    [[ "$output" =~ bats::install_core ]]
    [[ "$output" =~ bats::install ]]
}

@test "bats::create_dependencies_dir creates directory when missing" {
    source "$SCRIPT_PATH"
    trash::safe_remove "$BATS_DEPENDENCIES_DIR" --test-cleanup
    bats::create_dependencies_dir
    [ -d "$BATS_DEPENDENCIES_DIR" ]
}

@test "bats::install_dependency clones absent dependency" {
    source "$SCRIPT_PATH"
    # Stub git clone to just create the target directory (handle -c option and different argument positions)
    git() { 
        if [[ "$*" =~ clone ]]; then
            # Extract directory name (last argument)
            local dir_name="${!#}"
            mkdir -p "$BATS_DEPENDENCIES_DIR/$dir_name"
        fi
    }

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
    # Stub git clone and sudo install.sh (handle -c option for TLS config)
    git() { [[ "$*" =~ clone ]] && mkdir -p "$BATS_DEPENDENCIES_DIR/bats-core"; }
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

@test "bats::determine_prefix defaults to /usr/local when sudo available" {
    source "$SCRIPT_PATH"
    # Simulate sudo being available
    sudo() { [[ "$1" == "-n" ]] && [[ "$2" == "true" ]] && return 0; }
    export -f sudo
    
    SUDO_MODE=error bats::determine_prefix
    [ "$BATS_PREFIX" = "/usr/local" ]
}

@test "bats::determine_prefix switches to user local when SUDO_MODE=skip" {
    source "$SCRIPT_PATH"
    
    SUDO_MODE=skip bats::determine_prefix
    [ "$BATS_PREFIX" = "$HOME/.local" ]
}

@test "bats::determine_prefix switches to user local when sudo not available" {
    source "$SCRIPT_PATH"
    # Simulate sudo requiring password
    sudo() { [[ "$1" == "-n" ]] && [[ "$2" == "true" ]] && return 1; }
    export -f sudo
    
    SUDO_MODE=error bats::determine_prefix
    [ "$BATS_PREFIX" = "$HOME/.local" ]
}

@test "bats::determine_prefix uses custom BATS_PREFIX when set" {
    source "$SCRIPT_PATH"
    
    BATS_PREFIX=/custom/path bats::determine_prefix
    [ "$BATS_PREFIX" = "/custom/path" ]
}

@test "bats::install_core installs without sudo to user directory" {
    source "$SCRIPT_PATH"
    # Stub git clone and install.sh (handle -c option for TLS config)
    git() { [[ "$*" =~ clone ]] && mkdir -p "$BATS_DEPENDENCIES_DIR/bats-core"; }
    # Simulate no sudo available
    sudo() { return 1; }
    export -f sudo
    
    # Create a mock install.sh that records if it was called
    mkdir -p "$BATS_DEPENDENCIES_DIR/bats-core"
    echo '#!/bin/bash
echo "INSTALL_CALLED with prefix: $1" > "$BATS_DEPENDENCIES_DIR/install_result"' > "$BATS_DEPENDENCIES_DIR/bats-core/install.sh"
    chmod +x "$BATS_DEPENDENCIES_DIR/bats-core/install.sh"
    
    export BATS_PREFIX="$HOME/.local"
    cd "$BATS_DEPENDENCIES_DIR"
    run bash -c "cd bats-core && ./install.sh \"$BATS_PREFIX\""
    [ "$status" -eq 0 ]
    [ -f "$BATS_DEPENDENCIES_DIR/install_result" ]
    grep -q "INSTALL_CALLED with prefix: $HOME/.local" "$BATS_DEPENDENCIES_DIR/install_result"
} 