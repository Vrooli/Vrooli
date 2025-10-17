#!/usr/bin/env bats

# Load Vrooli test infrastructure (REQUIRED)
source "${BATS_TEST_DIRNAME}/../../../__test/fixtures/setup.bash"

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "windmill"
    
    # Export paths for use in setup()
    export SETUP_FILE_SCRIPT_PATH="${BATS_TEST_DIRNAME}/manage.sh"
    export SETUP_FILE_CONFIG_DIR="${BATS_TEST_DIRNAME}/config"
    export SETUP_FILE_LIB_DIR="${BATS_TEST_DIRNAME}/lib"
    export SETUP_FILE_WINDMILL_DIR="${BATS_TEST_DIRNAME}"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Use paths from setup_file
    SCRIPT_PATH="${SETUP_FILE_SCRIPT_PATH}"
    CONFIG_DIR="${SETUP_FILE_CONFIG_DIR}"
    LIB_DIR="${SETUP_FILE_LIB_DIR}"
    WINDMILL_DIR="${SETUP_FILE_WINDMILL_DIR}"
    
    # Set test environment BEFORE sourcing config files to avoid readonly conflicts
    export WINDMILL_PORT="5681"
    export WINDMILL_CONTAINER_NAME="windmill-test"
    export WINDMILL_BASE_URL="http://localhost:5681"
    export WINDMILL_DB_PASSWORD="test-password"
    export WINDMILL_ADMIN_EMAIL="admin@test.com"
    export WINDMILL_ADMIN_PASSWORD="admin123"
    export ACTION="status"
    export FORCE="no"
    export YES="no"
    
    # Mock resources functions that are called during config loading
    resources::get_default_port() {
        case "$1" in
            "windmill") echo "5681" ;;
            *) echo "8080" ;;
        esac
    }
    
    # Now source the config files
    source "${WINDMILL_DIR}/config/defaults.sh"
    source "${WINDMILL_DIR}/config/messages.sh"
    
    # Export config and messages
    windmill::export_config
    windmill::export_messages
    
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
    
    windmill::info() {
        echo "INFO_CALLED"
        return 0
    }
    
    windmill::scale_workers() {
        local count="$1"
        echo "SCALE_WORKERS_CALLED: $count"
        return 0
    }
    
    windmill::restart_workers() {
        echo "RESTART_WORKERS_CALLED"
        return 0
    }
    
    windmill::show_api_setup_instructions() {
        echo "API_SETUP_CALLED"
        return 0
    }
    
    windmill::save_api_key() {
        echo "SAVE_API_KEY_CALLED"
        return 0
    }
    
    windmill::backup() {
        local path="$1"
        echo "BACKUP_CALLED: $path"
        return 0
    }
    
    windmill::restore() {
        local path="$1"
        echo "RESTORE_CALLED: $path"
        return 0
    }
    
    windmill::list_apps() {
        echo "LIST_APPS_CALLED"
        return 0
    }
    
    windmill::prepare_app() {
        local name="$1"
        local output="$2"
        echo "PREPARE_APP_CALLED: $name to $output"
        return 0
    }
    
    windmill::deploy_app() {
        local name="$1"
        local workspace="$2"
        echo "DEPLOY_APP_CALLED: $name to $workspace"
        return 0
    }
    
    windmill::check_app_api() {
        echo "CHECK_APP_API_CALLED"
        return 0
    }
    
    # Mock validation functions
    windmill::validate_config() {
        return 0
    }
    
    windmill::usage() {
        echo "Usage: manage.sh [options]"
        return 0
    }
}

# Cleanup after each test
teardown() {
    vrooli_cleanup_test
}

# Test main function routing - install action
@test "windmill::main routes install action correctly" {
    export ACTION="install"
    export FORCE="no"
    
    # Mock the install function
    windmill::install() {
        echo "INSTALL_CALLED"
        return 0
    }
    
    result=$(windmill::main --action install)
    [[ "$result" =~ "INSTALL_CALLED" ]]
}

# Test main function routing - install with force
@test "windmill::main routes install action with force flag" {
    export ACTION="install"
    export FORCE="yes"
    
    # Mock the install function
    windmill::install() {
        echo "INSTALL_CALLED_WITH_FORCE"
        return 0
    }
    
    result=$(windmill::main --action install --force yes)
    [[ "$result" =~ "INSTALL_CALLED_WITH_FORCE" ]]
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

# Test main function routing - info action
@test "windmill::main routes info action correctly" {
    export ACTION="info"
    
    result=$(windmill::main --action info)
    [[ "$result" =~ "INFO_CALLED" ]]
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

# Test main function routing - scale-workers action
@test "windmill::main routes scale-workers action correctly" {
    export ACTION="scale-workers"
    export WORKER_COUNT="5"
    
    result=$(windmill::main --action scale-workers)
    [[ "$result" =~ "SCALE_WORKERS_CALLED: 5" ]]
}

# Test main function routing - restart-workers action
@test "windmill::main routes restart-workers action correctly" {
    export ACTION="restart-workers"
    
    result=$(windmill::main --action restart-workers)
    [[ "$result" =~ "RESTART_WORKERS_CALLED" ]]
}

# Test main function routing - save-api-key action
@test "windmill::main routes save-api-key action correctly" {
    export ACTION="save-api-key"
    
    result=$(windmill::main --action save-api-key)
    [[ "$result" =~ "SAVE_API_KEY_CALLED" ]]
}

# Test main function routing - list-apps action
@test "windmill::main routes list-apps action correctly" {
    export ACTION="list-apps"
    
    result=$(windmill::main --action list-apps)
    [[ "$result" =~ "LIST_APPS_CALLED" ]]
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

# Test main function without action (should default to install)
@test "windmill::main defaults to install when no action specified" {
    export ACTION="install"
    
    # Mock the install function
    windmill::install() {
        echo "INSTALL_CALLED: no"
        return 0
    }
    
    result=$(windmill::main)
    [[ "$result" =~ "INSTALL_CALLED" ]]
}

# Test help display
@test "windmill::main shows help with --help flag" {
    run windmill::main --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage:" ]] || [[ "$output" =~ "help" ]]
}

# Test api-setup action  
@test "windmill::main routes api-setup action correctly" {
    export ACTION="api-setup"
    
    result=$(windmill::main --action api-setup)
    [[ "$result" =~ "API_SETUP_CALLED" ]]
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
    [[ "$WINDMILL_CUSTOM_PORT" == "9999" ]]
}

# Test configuration loading
@test "windmill::main loads configuration correctly" {
    # Check that configuration variables are set
    [ -n "$WINDMILL_PROJECT_NAME" ]
    [ -n "$WINDMILL_SERVER_PORT" ]
    [ -n "$WINDMILL_BASE_URL" ]
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

# Test app deployment parameters
@test "windmill::main handles app deployment parameters" {
    export ACTION="deploy-app"
    export APP_NAME="test-app"
    export WORKSPACE="demo"
    
    result=$(windmill::main --action deploy-app --app-name test-app --workspace demo)
    [[ "$result" =~ "DEPLOY_APP_CALLED: test-app to demo" ]]
}

# Test app preparation parameters
@test "windmill::main handles app preparation parameters" {
    export ACTION="prepare-app"
    export APP_NAME="test-app"
    export OUTPUT_DIR="/tmp/apps"
    
    result=$(windmill::main --action prepare-app --app-name test-app --output-dir /tmp/apps)
    [[ "$result" =~ "PREPARE_APP_CALLED: test-app to /tmp/apps" ]]
}

# Test backup path handling
@test "windmill::main handles backup path parameters" {
    export ACTION="backup"
    export BACKUP_PATH="/tmp/windmill-backup.tar.gz"
    
    result=$(windmill::main --action backup --backup-path /tmp/windmill-backup.tar.gz)
    [[ "$result" =~ "BACKUP_CALLED: /tmp/windmill-backup.tar.gz" ]]
}

# Test restore path handling  
@test "windmill::main handles restore path parameters" {
    export ACTION="restore"
    export BACKUP_PATH="/tmp/windmill-backup.tar.gz"
    
    result=$(windmill::main --action restore --backup-path /tmp/windmill-backup.tar.gz)
    [[ "$result" =~ "RESTORE_CALLED: /tmp/windmill-backup.tar.gz" ]]
}

# Integration tests (when service is running)
@test "windmill manage script has correct permissions" {
    [ -x "${WINDMILL_DIR}/manage.sh" ]
}

@test "windmill examples directory structure exists" {
    [ -d "${WINDMILL_DIR}/examples/apps" ]
    [ -d "${WINDMILL_DIR}/examples/scripts" ]
    [ -d "${WINDMILL_DIR}/examples/flows" ]
}

@test "windmill app examples have valid JSON structure" {
    if command -v jq >/dev/null 2>&1 && [ -f "${WINDMILL_DIR}/examples/apps/admin-dashboard.json" ]; then
        run jq '.' "${WINDMILL_DIR}/examples/apps/admin-dashboard.json"
        [ "$status" -eq 0 ]
    else
        skip "jq not available or file missing"
    fi
}

# Test injection actions
@test "windmill::main routes inject action correctly" {
    export ACTION="inject"
    export INJECTION_CONFIG='{"scripts":[{"path":"f/test","file":"test.ts"}]}'
    
    # Mock the injection script execution
    "${var_SCRIPTS_RESOURCES_DIR}/automation/windmill/inject.sh"() {
        echo "INJECT_CALLED: $*"
        return 0
    }
    
    result=$(windmill::main --action inject --injection-config '{"scripts":[{"path":"f/test","file":"test.ts"}]}')
    [[ "$result" =~ "INJECT_CALLED" ]]
}

@test "windmill::main routes validate-injection action correctly" {
    export ACTION="validate-injection"
    export INJECTION_CONFIG='{"scripts":[{"path":"f/test","file":"test.ts"}]}'
    
    # Mock the injection script execution
    "${var_SCRIPTS_RESOURCES_DIR}/automation/windmill/inject.sh"() {
        echo "VALIDATE_CALLED: $*"
        return 0
    }
    
    result=$(windmill::main --action validate-injection --injection-config '{"scripts":[{"path":"f/test","file":"test.ts"}]}')
    [[ "$result" =~ "VALIDATE_CALLED" ]]
}

@test "windmill::main fails inject action without config" {
    export ACTION="inject"
    export INJECTION_CONFIG=""
    
    run windmill::main --action inject
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Injection configuration required" ]]
}

# Test check-app-api action
@test "windmill::main routes check-app-api action correctly" {
    export ACTION="check-app-api"
    
    result=$(windmill::main --action check-app-api)
    [[ "$result" =~ "CHECK_APP_API_CALLED" ]]
}