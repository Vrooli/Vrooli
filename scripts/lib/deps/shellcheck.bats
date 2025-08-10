#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Source trash module for safe test cleanup
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/shellcheck.sh"

setup() {
    # Create a temporary directory for isolated testing
    TMP_DIR=$(mktemp -d)
    export HOME="$TMP_DIR/home"
    mkdir -p "$HOME/.local/bin"
    export PATH="$HOME/.local/bin:$PATH"
    
    # Save original commands
    ORIGINAL_SHELLCHECK=$(command -v shellcheck 2>/dev/null || echo "")
}

teardown() {
    trash::safe_remove "$TMP_DIR" --test-cleanup
    # Restore PATH if modified
    if [[ ":$PATH:" == *":$HOME/.local/bin:"* ]]; then
        PATH=$(echo "$PATH" | sed "s|$HOME/.local/bin:||g")
    fi
}

@test "sourcing shellcheck.sh defines functions" {
    run bash -c "source '$SCRIPT_PATH' && declare -f shellcheck::install"
    [ "$status" -eq 0 ]
    [[ "$output" =~ shellcheck::install ]]
}

@test "shellcheck::install skips when already installed" {
    source "$SCRIPT_PATH"
    
    # Mock system::is_command to return true (shellcheck already installed)
    system::is_command() { [[ "$1" == "shellcheck" ]] && return 0; }
    export -f system::is_command
    
    run shellcheck::install
    [ "$status" -eq 0 ]
    [[ "$output" =~ "already installed" ]]
}

@test "shellcheck::install attempts package manager first" {
    source "$SCRIPT_PATH"
    
    # Mock system::is_command to return false (not installed)
    system::is_command() { [[ "$1" == "shellcheck" ]] && return 1; }
    
    # Mock system::install_pkg to succeed
    system::install_pkg() { [[ "$1" == "shellcheck" ]] && return 0; }
    
    export -f system::is_command system::install_pkg
    
    run shellcheck::install
    [ "$status" -eq 0 ]
    [[ "$output" =~ "installed via package manager" ]]
}

@test "shellcheck::install falls back to GitHub when package manager fails" {
    source "$SCRIPT_PATH"
    
    # Mock system::is_command to return false (not installed)
    system::is_command() { [[ "$1" == "shellcheck" ]] && return 1; }
    
    # Mock system::install_pkg to fail
    system::install_pkg() { return 1; }
    
    # Mock curl and tar for successful download
    curl() {
        if [[ "$*" =~ shellcheck.*tar\.xz ]]; then
            echo "mock shellcheck binary" > "$TMP_DIR/shellcheck"
            return 0
        fi
        return 1
    }
    
    tar() {
        if [[ "$*" =~ "-xJ" ]]; then
            local extract_dir=""
            local in_c_flag=false
            for arg in "$@"; do
                if [[ "$in_c_flag" == "true" ]]; then
                    extract_dir="$arg"
                    break
                elif [[ "$arg" == "-C" ]]; then
                    in_c_flag=true
                fi
            done
            if [[ -n "$extract_dir" ]]; then
                mkdir -p "$extract_dir/shellcheck-v0.10.0"
                echo "#!/bin/bash" > "$extract_dir/shellcheck-v0.10.0/shellcheck"
                chmod +x "$extract_dir/shellcheck-v0.10.0/shellcheck"
            fi
            return 0
        fi
        return 1
    }
    
    # Mock mv and chmod for user installation
    mv() { 
        if [[ "$2" =~ \.local/bin/shellcheck$ ]]; then
            touch "$2"
            return 0
        fi
        command mv "$@"
    }
    
    chmod() {
        if [[ "$1" == "+x" && "$2" =~ \.local/bin/shellcheck$ ]]; then
            return 0
        fi
        command chmod "$@"
    }
    
    # Mock sudo to fail (force local installation)
    sudo() { return 1; }
    
    export -f system::is_command system::install_pkg curl tar mv chmod sudo
    export SUDO_MODE=error
    
    run shellcheck::install
    [ "$status" -eq 0 ]
    [[ "$output" =~ "switching to user-local" ]]
    [[ "$output" =~ "installed to" ]]
}

@test "shellcheck::install uses SUDO_MODE=skip for user installation" {
    source "$SCRIPT_PATH"
    
    # Mock system::is_command to return false (not installed)
    system::is_command() { [[ "$1" == "shellcheck" ]] && return 1; }
    
    # Mock system::install_pkg to fail
    system::install_pkg() { return 1; }
    
    # Mock curl to fail all versions to test error handling
    curl() { return 1; }
    
    # Mock mktemp to avoid actual temp dir creation issues
    mktemp() { 
        [[ "$1" == "-d" ]] && echo "$TMP_DIR/mock_temp" && mkdir -p "$TMP_DIR/mock_temp"
    }
    
    # Mock rm to avoid errors
    rm() { 
        [[ "$1" == "-rf" ]] && return 0 || command rm "$@"
    }
    
    export -f system::is_command system::install_pkg curl mktemp rm
    export SUDO_MODE=skip
    
    run shellcheck::install
    [ "$status" -ne 0 ]
    [[ "$output" =~ "SUDO_MODE=skip" ]]
    [[ "$output" =~ "switching to user-local" ]]
    [[ "$output" =~ "All fallback ShellCheck installations failed" ]]
}

@test "shellcheck::install handles unsupported architecture" {
    source "$SCRIPT_PATH"
    
    # Mock system::is_command to return false (not installed)
    system::is_command() { [[ "$1" == "shellcheck" ]] && return 1; }
    
    # Mock system::install_pkg to fail
    system::install_pkg() { return 1; }
    
    # Mock uname to return unsupported architecture
    uname() {
        if [[ "$1" == "-m" ]]; then
            echo "unsupported_arch"
        else
            command uname "$@"
        fi
    }
    
    export -f system::is_command system::install_pkg uname
    export SUDO_MODE=skip
    
    run shellcheck::install
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Unsupported architecture" ]]
}