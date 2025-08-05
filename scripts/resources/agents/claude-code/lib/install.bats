#!/usr/bin/env bats

# BATS setup function - runs before each test
setup() {
    # Set up paths
    export BATS_TEST_DIRNAME="${BATS_TEST_DIRNAME:-$(cd "$(dirname "$BATS_TEST_FILENAME")" && pwd)}"
    export CLAUDE_CODE_DIR="$BATS_TEST_DIRNAME/.."
    export RESOURCES_DIR="$CLAUDE_CODE_DIR/../.."
    export HELPERS_DIR="$RESOURCES_DIR/../helpers"
    export SCRIPT_PATH="$BATS_TEST_DIRNAME/install.sh"
    
    # Source dependencies in order
    source "$HELPERS_DIR/utils/log.sh" 2>/dev/null || true
    source "$HELPERS_DIR/utils/system.sh" 2>/dev/null || true
    source "$HELPERS_DIR/utils/flow.sh" 2>/dev/null || true
    source "$RESOURCES_DIR/common.sh" 2>/dev/null || true
    
    # Source config and messages
    source "$CLAUDE_CODE_DIR/config/defaults.sh"
    source "$CLAUDE_CODE_DIR/config/messages.sh"
    
    # Source common functions
    source "$CLAUDE_CODE_DIR/lib/common.sh"
    
    # Source the script under test
    source "$SCRIPT_PATH"
    
    # Default mocks
    confirm() { return 0; }  # Always confirm
    resources::update_config() { return 0; }
    resources::remove_from_config() { return 0; }
}

# ============================================================================
# Function Definition Tests
# ============================================================================

@test "install.sh defines required functions" {
    # Functions are already loaded by setup()
    declare -f claude_code::install
    declare -f claude_code::uninstall
}
# ============================================================================
# Install Tests
# ============================================================================
@test "claude_code::install checks if already installed" {
    # Override to simulate already installed
    claude_code::is_installed() { return 0; }
    claude_code::get_version() { echo '1.0.0'; }
    FORCE=no
    run claude_code::install
    [ "$status" -eq 0 ]
    [[ "$output" =~ "already installed" ]]
}

@test "claude_code::install proceeds with force flag" {
    # Override to simulate various conditions
    claude_code::is_installed() { return 0; }
    claude_code::check_node_version() { return 0; }
    system::is_command() { return 0; }
    node() { echo 'v20.0.0'; }
    npm() { 
        case "$1" in
            --version) echo '9.0.0' ;;
            install) return 0 ;;
        esac
    }
    FORCE=yes
    
    run claude_code::install
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Installing Claude Code" ]]
}

@test "claude_code::install fails without node" {
    # Override to simulate no node
    claude_code::is_installed() { return 1; }
    claude_code::check_node_version() { return 1; }
    
    run claude_code::install
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Node.js" ]]
    [[ "$output" =~ "required" ]]
}

@test "claude_code::install fails without npm" {
    # Override to simulate npm missing
    claude_code::is_installed() { return 1; }
    claude_code::check_node_version() { return 0; }
    node() { echo 'v20.0.0'; }
    system::is_command() { 
        [[ "$1" == "npm" ]] && return 1 || return 0
    }
    
    run claude_code::install
    [ "$status" -eq 1 ]
    [[ "$output" =~ "npm is required" ]]
}

@test "claude_code::install shows next steps on success" {
    # Track install state
    local install_called=0
    claude_code::is_installed() { 
        # First call returns false (not installed), second returns true (after install)
        if [[ $install_called -eq 0 ]]; then
            install_called=1
            return 1
        else
            return 0
        fi
    }
    claude_code::get_version() { echo '1.0.0'; }
    claude_code::check_node_version() { return 0; }
    system::is_command() { return 0; }
    node() { echo 'v20.0.0'; }
    npm() { 
        case "$1" in
            --version) echo '9.0.0' ;;
            install) return 0 ;;
        esac
    }
    claude_code::install_next_steps() { echo 'Next steps shown'; }
    
    run claude_code::install
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Next steps shown" ]]
}

# ============================================================================
# Uninstall Tests
# ============================================================================

@test "claude_code::uninstall handles not installed case" {
    # Override to simulate not installed
    claude_code::is_installed() { return 1; }
    
    run claude_code::uninstall
    [ "$status" -eq 0 ]
    [[ "$output" =~ "not installed" ]]
}

@test "claude_code::uninstall cancels on no confirmation" {
    # Override to simulate installed and no confirmation
    claude_code::is_installed() { return 0; }
    confirm() { return 1; }
    
    run claude_code::uninstall
    [ "$status" -eq 0 ]
    [[ "$output" =~ "cancelled" ]]
}

@test "claude_code::uninstall proceeds with confirmation" {
    # Override to simulate successful uninstall
    claude_code::is_installed() { return 0; }
    confirm() { return 0; }
    npm() { 
        [[ "$1" == "uninstall" ]] && return 0
    }
    
    run claude_code::uninstall
    [ "$status" -eq 0 ]
    [[ "$output" =~ "removed successfully" ]]
}

@test "claude_code::uninstall handles npm failure" {
    # Override to simulate npm failure
    claude_code::is_installed() { return 0; }
    confirm() { return 0; }
    npm() { return 1; }
    
    run claude_code::uninstall
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Failed to uninstall" ]]
}
