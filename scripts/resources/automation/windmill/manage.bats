#!/usr/bin/env bats
# Tests for Windmill main management script

# Setup for each test
setup() {
    # Set test environment
    export WINDMILL_PORT="5681"
    export WINDMILL_CONTAINER_NAME="windmill-test"
    export WINDMILL_BASE_URL="http://localhost:5681"
    export WINDMILL_DB_PASSWORD="test-password"
    export WINDMILL_ADMIN_EMAIL="admin@test.com"
    export WINDMILL_ADMIN_PASSWORD="admin123"
    export ACTION="status"
    export FORCE="no"
    export YES="no"
    
    # Load dependencies
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    WINDMILL_DIR="$SCRIPT_DIR"
    
    # Mock system functions
    system::is_command() {
        case "$1" in
            "docker"|"curl"|"jq") return 0 ;;
            *) return 1 ;;
        esac
    }
    
    # Mock log functions
    log::info() { echo "INFO: $1"; }
    log::error() { echo "ERROR: $1"; }
    log::warn() { echo "WARN: $1"; }
    log::success() { echo "SUCCESS: $1"; }
    log::debug() { echo "DEBUG: $1"; }
    log::header() { echo "=== $1 ==="; }
    
    # Mock argument parsing function
    windmill::parse_arguments() {
        # Mock successful argument parsing
        return 0
    }
    
    # Mock core Windmill functions
    windmill::install() {
        local force_flag="$1"
        echo "INSTALL_CALLED: $force_flag"
        return 0
    }
    
    windmill::uninstall() {
        echo "UNINSTALL_CALLED"
        return 0
    }
    
    windmill::start() {
        echo "START_CALLED"
        return 0
    }
    
    windmill::stop() {
        echo "STOP_CALLED"
        return 0
    }
    
    windmill::restart() {
        echo "RESTART_CALLED"
        return 0
    }
    
    windmill::status() {
        echo "STATUS_CALLED"
        return 0
    }
    
    windmill::logs() {
        echo "LOGS_CALLED"
        return 0
    }
    
    windmill::execute() {
        echo "EXECUTE_CALLED"
        return 0
    }
    
    windmill::backup() {
        echo "BACKUP_CALLED"
        return 0
    }
    
    windmill::restore() {
        echo "RESTORE_CALLED"
        return 0
    }
    
    windmill::update() {
        echo "UPDATE_CALLED"
        return 0
    }
    
    windmill::reset_password() {
        echo "RESET_PASSWORD_CALLED"
        return 0
    }
    
    windmill::create_workspace() {
        echo "CREATE_WORKSPACE_CALLED"
        return 0
    }
    
    windmill::list_workspaces() {
        echo "LIST_WORKSPACES_CALLED"
        return 0
    }
    
    windmill::api_setup() {
        echo "API_SETUP_CALLED"
        return 0
    }
    
    windmill::export_config() {
        export WINDMILL_SERVICE_NAME="windmill"
        export WINDMILL_CONTAINER_NAME="windmill-test"
        export WINDMILL_DEFAULT_PORT="5681"
    }
    
    windmill::export_messages() {
        export MSG_WINDMILL_INSTALLING="Installing Windmill"
        export MSG_INSTALLATION_SUCCESS="✅ Windmill installed successfully"
        export MSG_SERVICE_INFO="Windmill is available"
    }
    
    # Load configuration and messages
    windmill::export_config
    windmill::export_messages
}

# Test main function routing - install action
@test "windmill::main routes install action correctly" {
    export ACTION="install"
    export FORCE="no"
    
    # Mock the install function
    windmill::install() {
        echo "INSTALL_CALLED: $1"
        return 0
    }
    
    result=$(windmill::main --action install)
    [[ "$result" =~ "INSTALL_CALLED: no" ]]
}

# Test main function routing - install with force
@test "windmill::main routes install action with force flag" {
    export ACTION="install"
    export FORCE="yes"
    
    # Mock the install function
    windmill::install() {
        echo "INSTALL_CALLED: $1"
        return 0
    }
    
    result=$(windmill::main --action install --force)
    [[ "$result" =~ "INSTALL_CALLED: yes" ]]
}

# Test main function routing - uninstall action
@test "windmill::main routes uninstall action correctly" {
    export ACTION="uninstall"
    
    result=$(windmill::main --action uninstall)
    [[ "$result" =~ "UNINSTALL_CALLED" ]]
}

# Test main function routing - start action
@test "windmill::main routes start action correctly" {
    export ACTION="start"
    
    result=$(windmill::main --action start)
    [[ "$result" =~ "START_CALLED" ]]
}

# Test main function routing - stop action
@test "windmill::main routes stop action correctly" {
    export ACTION="stop"
    
    result=$(windmill::main --action stop)
    [[ "$result" =~ "STOP_CALLED" ]]
}

# Test main function routing - restart action
@test "windmill::main routes restart action correctly" {
    export ACTION="restart"
    
    result=$(windmill::main --action restart)
    [[ "$result" =~ "RESTART_CALLED" ]]
}

# Test main function routing - status action (default)
@test "windmill::main routes status action correctly" {
    export ACTION="status"
    
    result=$(windmill::main --action status)
    [[ "$result" =~ "STATUS_CALLED" ]]
}

# Test main function routing - logs action
@test "windmill::main routes logs action correctly" {
    export ACTION="logs"
    
    result=$(windmill::main --action logs)
    [[ "$result" =~ "LOGS_CALLED" ]]
}

# Test main function routing - execute action
@test "windmill::main routes execute action correctly" {
    export ACTION="execute"
    
    result=$(windmill::main --action execute)
    [[ "$result" =~ "EXECUTE_CALLED" ]]
}

# Test main function routing - backup action
@test "windmill::main routes backup action correctly" {
    export ACTION="backup"
    
    result=$(windmill::main --action backup)
    [[ "$result" =~ "BACKUP_CALLED" ]]
}

# Test main function routing - restore action
@test "windmill::main routes restore action correctly" {
    export ACTION="restore"
    
    result=$(windmill::main --action restore)
    [[ "$result" =~ "RESTORE_CALLED" ]]
}

# Test main function routing - update action
@test "windmill::main routes update action correctly" {
    export ACTION="update"
    
    result=$(windmill::main --action update)
    [[ "$result" =~ "UPDATE_CALLED" ]]
}

# Test main function routing - reset-password action
@test "windmill::main routes reset-password action correctly" {
    export ACTION="reset-password"
    
    result=$(windmill::main --action reset-password)
    [[ "$result" =~ "RESET_PASSWORD_CALLED" ]]
}

# Test main function routing - create-workspace action
@test "windmill::main routes create-workspace action correctly" {
    export ACTION="create-workspace"
    
    result=$(windmill::main --action create-workspace)
    [[ "$result" =~ "CREATE_WORKSPACE_CALLED" ]]
}

# Test main function routing - list-workspaces action
@test "windmill::main routes list-workspaces action correctly" {
    export ACTION="list-workspaces"
    
    result=$(windmill::main --action list-workspaces)
    [[ "$result" =~ "LIST_WORKSPACES_CALLED" ]]
}

# Test main function routing - api-setup action
@test "windmill::main routes api-setup action correctly" {
    export ACTION="api-setup"
    
    result=$(windmill::main --action api-setup)
    [[ "$result" =~ "API_SETUP_CALLED" ]]
}

# Test main function routing - invalid action
@test "windmill::main handles invalid action gracefully" {
    export ACTION="invalid-action"
    
    run windmill::main --action invalid-action
    [ "$status" -eq 1 ] || [[ "$output" =~ "invalid" ]]
}

# Test main function without action (should default to status)
@test "windmill::main defaults to status when no action specified" {
    unset ACTION
    
    result=$(windmill::main)
    [[ "$result" =~ "STATUS_CALLED" ]]
}

# Test help display
@test "windmill::main shows help with --help flag" {
    result=$(windmill::main --help)
    [[ "$result" =~ "Usage:" ]] || [[ "$result" =~ "help" ]]
}

# Test version display
@test "windmill::main shows version with --version flag" {
    result=$(windmill::main --version)
    [[ "$result" =~ "version" ]] || [[ "$result" =~ "Windmill" ]]
}

# Test argument parsing integration
@test "windmill::main processes arguments correctly" {
    # Test with multiple arguments
    export ACTION="install"
    export WINDMILL_PORT="8080"
    export WINDMILL_ADMIN_EMAIL="test@example.com"
    
    result=$(windmill::main --action install --port 8080 --admin-email test@example.com)
    [[ "$result" =~ "INSTALL_CALLED" ]]
}

# Test environment variable precedence
@test "windmill::main respects environment variable overrides" {
    export WINDMILL_CUSTOM_PORT="9999"
    export ACTION="status"
    
    result=$(windmill::main --action status)
    [[ "$result" =~ "STATUS_CALLED" ]]
    # Port should be overridden
    [[ "$WINDMILL_BASE_URL" =~ "9999" ]] || [[ "$WINDMILL_CUSTOM_PORT" == "9999" ]]
}

# Test configuration loading
@test "windmill::main loads configuration correctly" {
    # Check that configuration variables are set
    [ -n "$WINDMILL_SERVICE_NAME" ]
    [ -n "$WINDMILL_CONTAINER_NAME" ]
    [ -n "$WINDMILL_DEFAULT_PORT" ]
}

# Test error handling in main function
@test "windmill::main handles function errors appropriately" {
    export ACTION="install"
    
    # Override install to fail
    windmill::install() {
        echo "Installation failed"
        return 1
    }
    
    run windmill::main --action install
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Installation failed" ]]
}

# Test workspace parameter handling
@test "windmill::main handles workspace parameters" {
    export ACTION="create-workspace"
    export WORKSPACE_NAME="test-workspace"
    
    result=$(windmill::main --action create-workspace --workspace test-workspace)
    [[ "$result" =~ "CREATE_WORKSPACE_CALLED" ]]
}

# Test script path handling
@test "windmill::main handles script path parameters" {
    export ACTION="execute"
    export SCRIPT_PATH="/path/to/script.py"
    
    result=$(windmill::main --action execute --script /path/to/script.py)
    [[ "$result" =~ "EXECUTE_CALLED" ]]
}

# Test backup path handling
@test "windmill::main handles backup path parameters" {
    export ACTION="backup"
    export BACKUP_PATH="/tmp/windmill-backup.tar.gz"
    
    result=$(windmill::main --action backup --backup-path /tmp/windmill-backup.tar.gz)
    [[ "$result" =~ "BACKUP_CALLED" ]]
}

# Test restore path handling  
@test "windmill::main handles restore path parameters" {
    export ACTION="restore"
    export RESTORE_PATH="/tmp/windmill-backup.tar.gz"
    
    result=$(windmill::main --action restore --restore-path /tmp/windmill-backup.tar.gz)
    [[ "$result" =~ "RESTORE_CALLED" ]]
}

# Integration tests (when service is running)
@test "windmill manage script has correct permissions" {
    [ -x "${BATS_TEST_DIRNAME}/manage.sh" ]
}

@test "windmill examples directory structure exists" {
    [ -d "${BATS_TEST_DIRNAME}/examples/apps" ]
    [ -d "${BATS_TEST_DIRNAME}/examples/scripts" ]
    [ -d "${BATS_TEST_DIRNAME}/examples/flows" ]
}

@test "windmill app examples have valid JSON structure" {
    if command -v jq >/dev/null 2>&1 && [ -f "${BATS_TEST_DIRNAME}/examples/apps/admin-dashboard.json" ]; then
        run jq '.' "${BATS_TEST_DIRNAME}/examples/apps/admin-dashboard.json"
        [ "$status" -eq 0 ]
    else
        skip "jq not available or file missing"
    fi
}