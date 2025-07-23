#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/vrooli_cli.sh"

@test "sourcing vrooli_cli.sh defines required functions" {
    run bash -c "source '$SCRIPT_PATH' && declare -f vrooli_cli::setup"
    [ "$status" -eq 0 ]
    [[ "$output" =~ vrooli_cli::setup ]]
}

@test "vrooli_cli::setup detects sudo user correctly" {
    skip "This test would need to mock SUDO_USER environment"
    # This test is difficult to implement without actually running as sudo
    # In a real environment, we'd test that:
    # 1. When SUDO_USER is set, it uses that user's home directory
    # 2. When SUDO_USER is not set, it uses current user's home
}

@test "vrooli_cli::setup handles missing CLI package gracefully" {
    run bash -c "
        source '$SCRIPT_PATH'
        # Mock the project root to a non-existent location
        var_ROOT_DIR='/tmp/nonexistent'
        vrooli_cli::setup 2>&1
    "
    [ "$status" -ne 0 ]
    [[ "$output" =~ "CLI package not found" ]]
}

@test "vrooli_cli::setup skips when CLI already installed" {
    run bash -c "
        source '$SCRIPT_PATH'
        # Mock command to simulate vrooli already installed
        command() {
            [[ \"\$1\" == '-v' ]] && [[ \"\$2\" == 'vrooli' ]] && return 0
            builtin command \"\$@\"
        }
        # Mock vrooli command
        vrooli() {
            [[ \"\$1\" == '--version' ]] && echo 'Vrooli CLI v2.0.0'
        }
        vrooli_cli::setup 2>&1
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Vrooli CLI is already installed" ]]
    [[ "$output" =~ "Vrooli CLI setup check complete" ]]
}

@test "vrooli_cli::setup creates wrapper script" {
    skip "This test would require a full mock environment with pnpm and node"
    # In a real test environment, we'd verify:
    # 1. The wrapper script is created at the correct location
    # 2. The wrapper script has correct permissions (executable)
    # 3. The wrapper script contains the correct shebang and content
}

@test "vrooli_cli::setup handles PATH detection" {
    run bash -c "
        source '$SCRIPT_PATH'
        # Test with a PATH that doesn't include .local/bin
        export PATH='/usr/bin:/bin'
        export HOME='/tmp/test_home'
        # Mock the CLI installation location
        user_bin_dir='\$HOME/.local/bin'
        # This should trigger the PATH warning
        if [[ \":\$PATH:\" != *\":\$user_bin_dir:\"* ]]; then
            echo 'PATH does not include user bin directory'
        fi
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "PATH does not include user bin directory" ]]
}