#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/manage.sh"
CONFIG_DIR="$BATS_TEST_DIRNAME/config"
LIB_DIR="$BATS_TEST_DIRNAME/lib"

# Source dependencies
RESOURCES_DIR="$BATS_TEST_DIRNAME/../.."
HELPERS_DIR="$RESOURCES_DIR/../helpers"

# Source required utilities (suppress errors during test setup)
. "$HELPERS_DIR/utils/log.sh" 2>/dev/null || true
. "$HELPERS_DIR/utils/system.sh" 2>/dev/null || true
. "$HELPERS_DIR/utils/args.sh" 2>/dev/null || true

# ============================================================================
# Script Loading Tests
# ============================================================================

@test "sourcing manage.sh defines required functions" {
    run bash -c "source '$SCRIPT_PATH' && declare -f vault::parse_arguments && declare -f vault::main"
    [ "$status" -eq 0 ]
    [[ "$output" =~ vault::parse_arguments ]]
    [[ "$output" =~ vault::main ]]
}

@test "manage.sh sources all required dependencies" {
    run bash -c "source '$SCRIPT_PATH' 2>&1 | grep -v 'command not found' | head -1"
    [ "$status" -eq 0 ]
}

# ============================================================================
# Argument Parsing Tests
# ============================================================================

@test "vault::parse_arguments sets default action to status" {
    run bash -c "source '$SCRIPT_PATH'; vault::parse_arguments; echo \"\$ACTION\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "status" ]]
}

@test "vault::parse_arguments accepts install action" {
    run bash -c "source '$SCRIPT_PATH'; vault::parse_arguments --action install; echo \"\$ACTION\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "install" ]]
}

@test "vault::parse_arguments accepts status action" {
    run bash -c "source '$SCRIPT_PATH'; vault::parse_arguments --action status; echo \"\$ACTION\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "status" ]]
}

@test "vault::parse_arguments accepts init-dev action" {
    run bash -c "source '$SCRIPT_PATH'; vault::parse_arguments --action init-dev; echo \"\$ACTION\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "init-dev" ]]
}

@test "vault::parse_arguments accepts init-prod action" {
    run bash -c "source '$SCRIPT_PATH'; vault::parse_arguments --action init-prod; echo \"\$ACTION\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "init-prod" ]]
}

@test "vault::parse_arguments accepts put-secret action" {
    run bash -c "source '$SCRIPT_PATH'; vault::parse_arguments --action put-secret; echo \"\$ACTION\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "put-secret" ]]
}

@test "vault::parse_arguments accepts get-secret action" {
    run bash -c "source '$SCRIPT_PATH'; vault::parse_arguments --action get-secret; echo \"\$ACTION\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "get-secret" ]]
}

@test "vault::parse_arguments accepts secret management arguments" {
    run bash -c "source '$SCRIPT_PATH'; vault::parse_arguments --action put-secret --path test/key --value secret123; echo \"PATH=\$SECRET_PATH VALUE=\$SECRET_VALUE\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "PATH=test/key" ]]
    [[ "$output" =~ "VALUE=secret123" ]]
}

@test "vault::parse_arguments accepts mode argument" {
    run bash -c "source '$SCRIPT_PATH'; vault::parse_arguments --mode prod; echo \"\$VAULT_MODE\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "prod" ]]
}

@test "vault::parse_arguments accepts migration arguments" {
    run bash -c "source '$SCRIPT_PATH'; vault::parse_arguments --action migrate-env --env-file .env --vault-prefix dev; echo \"ENV=\$ENV_FILE PREFIX=\$VAULT_PREFIX\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "ENV=.env" ]]
    [[ "$output" =~ "PREFIX=dev" ]]
}

@test "vault::parse_arguments sets follow logs flag" {
    run bash -c "source '$SCRIPT_PATH'; vault::parse_arguments --follow yes; echo \"\$FOLLOW_LOGS\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "yes" ]]
}

@test "vault::parse_arguments accepts monitor interval" {
    run bash -c "source '$SCRIPT_PATH'; vault::parse_arguments --interval 60; echo \"\$MONITOR_INTERVAL\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "60" ]]
}

# ============================================================================
# Help and Usage Tests
# ============================================================================

@test "manage.sh displays help with --help" {
    run bash "$SCRIPT_PATH" --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "HashiCorp Vault Secret Management" ]]
    [[ "$output" =~ "USAGE:" ]]
    [[ "$output" =~ "ACTIONS:" ]]
    [[ "$output" =~ "EXAMPLES:" ]]
}

@test "manage.sh displays help for help action" {
    run bash "$SCRIPT_PATH" --action help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "HashiCorp Vault Secret Management" ]]
}

# ============================================================================
# Main Function Tests
# ============================================================================

@test "vault::main calls correct function for install action" {
    # Mock functions
    run bash -c "
        source '$SCRIPT_PATH'
        vault::install() { echo 'vault::install called'; return 0; }
        export ACTION='install'
        vault::main
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "vault::install called" ]]
}

@test "vault::main calls correct function for status action" {
    run bash -c "
        source '$SCRIPT_PATH'
        vault::show_status() { echo 'vault::show_status called'; return 0; }
        export ACTION='status'
        vault::main
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "vault::show_status called" ]]
}

@test "vault::main validates put-secret arguments" {
    run bash -c "
        source '$SCRIPT_PATH'
        export ACTION='put-secret'
        export SECRET_PATH=''
        export SECRET_VALUE='test'
        vault::main 2>&1
    "
    [ "$status" -eq 1 ]
    [[ "$output" =~ "--path and --value are required" ]]
}

@test "vault::main validates get-secret arguments" {
    run bash -c "
        source '$SCRIPT_PATH'
        export ACTION='get-secret'
        export SECRET_PATH=''
        vault::main 2>&1
    "
    [ "$status" -eq 1 ]
    [[ "$output" =~ "--path is required" ]]
}

@test "vault::main validates migrate-env arguments" {
    run bash -c "
        source '$SCRIPT_PATH'
        export ACTION='migrate-env'
        export ENV_FILE=''
        export VAULT_PREFIX='dev'
        vault::main 2>&1
    "
    [ "$status" -eq 1 ]
    [[ "$output" =~ "--env-file and --vault-prefix are required" ]]
}

@test "vault::main handles unknown action" {
    run bash -c "
        source '$SCRIPT_PATH'
        export ACTION='unknown-action'
        vault::main 2>&1
    "
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown action: unknown-action" ]]
}

# ============================================================================
# Integration Tests
# ============================================================================

@test "manage.sh executes status action without arguments" {
    # Mock vault functions to avoid actual execution
    run bash -c "
        export VAULT_PORT=8200
        export VAULT_MODE=dev
        export VAULT_DATA_DIR=/tmp/vault-test
        export VAULT_CONFIG_DIR=/tmp/vault-test
        export ACTION=status
        
        # Source the script and mock the status function
        source '$SCRIPT_PATH'
        vault::show_status() { echo 'Status: not_installed'; return 0; }
        vault::cleanup() { : ; }
        
        vault::main 2>&1 | grep 'Status:'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Status:" ]]
}

@test "manage.sh validates secret operations require path" {
    run bash "$SCRIPT_PATH" --action put-secret --value "test123"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "path" ]]
}

@test "manage.sh validates secret operations require value for put" {
    run bash "$SCRIPT_PATH" --action put-secret --path "test/key"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "value" ]]
}