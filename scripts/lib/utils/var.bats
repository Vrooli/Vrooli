#!/usr/bin/env bats
# Tests for var.sh - Variable definitions and path setup

bats_require_minimum_version 1.5.0

# Load test infrastructure
source "${BATS_TEST_DIRNAME}/../../__test/fixtures/setup.bash"

# Load BATS helpers
load "${BATS_TEST_DIRNAME}/../../__test/helpers/bats-support/load"
load "${BATS_TEST_DIRNAME}/../../__test/helpers/bats-assert/load"

setup() {
    vrooli_setup_unit_test
    
    # Source the var utility
    source "${BATS_TEST_DIRNAME}/var.sh"
}

teardown() {
    vrooli_cleanup_test
}

# Test core directory variables
@test "var_LIB_UTILS_DIR is defined and points to correct directory" {
    [[ -n "${var_LIB_UTILS_DIR+x}" ]] || { echo "var_LIB_UTILS_DIR not defined"; return 1; }
    [[ -d "$var_LIB_UTILS_DIR" ]] || { echo "var_LIB_UTILS_DIR directory does not exist: $var_LIB_UTILS_DIR"; return 1; }
    [[ "$var_LIB_UTILS_DIR" == *"/lib/utils" ]] || { echo "var_LIB_UTILS_DIR does not end with /lib/utils: $var_LIB_UTILS_DIR"; return 1; }
}

@test "var_LIB_DIR is defined and points to parent of utils directory" {
    [[ -n "${var_LIB_DIR+x}" ]] || { echo "var_LIB_DIR not defined"; return 1; }
    [[ -d "$var_LIB_DIR" ]] || { echo "var_LIB_DIR directory does not exist: $var_LIB_DIR"; return 1; }
    [[ "$var_LIB_DIR" == *"/lib" ]] || { echo "var_LIB_DIR does not end with /lib: $var_LIB_DIR"; return 1; }
}

@test "var_SCRIPTS_DIR is defined and points to scripts directory" {
    [[ -n "${var_SCRIPTS_DIR+x}" ]] || { echo "var_SCRIPTS_DIR not defined"; return 1; }
    [[ -d "$var_SCRIPTS_DIR" ]] || { echo "var_SCRIPTS_DIR directory does not exist: $var_SCRIPTS_DIR"; return 1; }
    [[ "$var_SCRIPTS_DIR" == *"/scripts" ]] || { echo "var_SCRIPTS_DIR does not end with /scripts: $var_SCRIPTS_DIR"; return 1; }
}

@test "var_ROOT_DIR is defined and points to project root" {
    [[ -n "${var_ROOT_DIR+x}" ]] || { echo "var_ROOT_DIR not defined"; return 1; }
    [[ -d "$var_ROOT_DIR" ]] || { echo "var_ROOT_DIR directory does not exist: $var_ROOT_DIR"; return 1; }
}

# Test Vrooli configuration directory variables
@test "var_VROOLI_CONFIG_DIR is defined" {
    [[ -n "${var_VROOLI_CONFIG_DIR+x}" ]] || { echo "var_VROOLI_CONFIG_DIR not defined"; return 1; }
    [[ "$var_VROOLI_CONFIG_DIR" == *"/.vrooli" ]] || { echo "var_VROOLI_CONFIG_DIR does not end with /.vrooli: $var_VROOLI_CONFIG_DIR"; return 1; }
}

@test "var_SERVICE_JSON_FILE is defined and points to service.json" {
    [[ -n "${var_SERVICE_JSON_FILE+x}" ]] || { echo "var_SERVICE_JSON_FILE not defined"; return 1; }
    [[ "$var_SERVICE_JSON_FILE" == *"/service.json" ]] || { echo "var_SERVICE_JSON_FILE does not end with /service.json: $var_SERVICE_JSON_FILE"; return 1; }
}

# Test lib subdirectory variables
@test "lib subdirectory variables are defined correctly" {
    # Test that all lib subdirectories are defined
    [[ -n "${var_LIB_DEPLOY_DIR+x}" ]] || { echo "var_LIB_DEPLOY_DIR not defined"; return 1; }
    [[ -n "${var_LIB_DEPS_DIR+x}" ]] || { echo "var_LIB_DEPS_DIR not defined"; return 1; }
    [[ -n "${var_LIB_LIFECYCLE_DIR+x}" ]] || { echo "var_LIB_LIFECYCLE_DIR not defined"; return 1; }
    [[ -n "${var_LIB_NETWORK_DIR+x}" ]] || { echo "var_LIB_NETWORK_DIR not defined"; return 1; }
    [[ -n "${var_LIB_SERVICE_DIR+x}" ]] || { echo "var_LIB_SERVICE_DIR not defined"; return 1; }
    [[ -n "${var_LIB_SYSTEM_DIR+x}" ]] || { echo "var_LIB_SYSTEM_DIR not defined"; return 1; }
    
    # Test that they point to correct subdirectories
    [[ "$var_LIB_DEPLOY_DIR" == *"/lib/deploy" ]] || { echo "var_LIB_DEPLOY_DIR incorrect: $var_LIB_DEPLOY_DIR"; return 1; }
    [[ "$var_LIB_DEPS_DIR" == *"/lib/deps" ]] || { echo "var_LIB_DEPS_DIR incorrect: $var_LIB_DEPS_DIR"; return 1; }
    [[ "$var_LIB_LIFECYCLE_DIR" == *"/lib/lifecycle" ]] || { echo "var_LIB_LIFECYCLE_DIR incorrect: $var_LIB_LIFECYCLE_DIR"; return 1; }
    [[ "$var_LIB_NETWORK_DIR" == *"/lib/network" ]] || { echo "var_LIB_NETWORK_DIR incorrect: $var_LIB_NETWORK_DIR"; return 1; }
    [[ "$var_LIB_SERVICE_DIR" == *"/lib/service" ]] || { echo "var_LIB_SERVICE_DIR incorrect: $var_LIB_SERVICE_DIR"; return 1; }
    [[ "$var_LIB_SYSTEM_DIR" == *"/lib/system" ]] || { echo "var_LIB_SYSTEM_DIR incorrect: $var_LIB_SYSTEM_DIR"; return 1; }
}

# Test scripts subdirectory variables
@test "scripts subdirectory variables are defined correctly" {
    [[ -n "${var_TEST_DIR+x}" ]] || { echo "var_TEST_DIR not defined"; return 1; }
    [[ -n "${var_SCRIPTS_RESOURCES_DIR+x}" ]] || { echo "var_SCRIPTS_RESOURCES_DIR not defined"; return 1; }
    [[ -n "${var_SCRIPTS_SCENARIOS_DIR+x}" ]] || { echo "var_SCRIPTS_SCENARIOS_DIR not defined"; return 1; }
    [[ -n "${var_SCRIPTS_SCENARIOS_INJECTION_DIR+x}" ]] || { echo "var_SCRIPTS_SCENARIOS_INJECTION_DIR not defined"; return 1; }
    
    [[ "$var_TEST_DIR" == *"/__test" ]] || { echo "var_TEST_DIR incorrect: $var_TEST_DIR"; return 1; }
    [[ "$var_SCRIPTS_RESOURCES_DIR" == *"/scripts/resources" ]] || { echo "var_SCRIPTS_RESOURCES_DIR incorrect: $var_SCRIPTS_RESOURCES_DIR"; return 1; }
    [[ "$var_SCRIPTS_SCENARIOS_DIR" == *"/scripts/scenarios" ]] || { echo "var_SCRIPTS_SCENARIOS_DIR incorrect: $var_SCRIPTS_SCENARIOS_DIR"; return 1; }
}

# Test common file variables
@test "common file variables are defined correctly" {
    # Test that file variables are defined
    [[ -n "${var_EXIT_CODES_FILE+x}" ]] || { echo "var_EXIT_CODES_FILE not defined"; return 1; }
    [[ -n "${var_FLOW_FILE+x}" ]] || { echo "var_FLOW_FILE not defined"; return 1; }
    [[ -n "${var_LOG_FILE+x}" ]] || { echo "var_LOG_FILE not defined"; return 1; }
    [[ -n "${var_LIFECYCLE_ENGINE_FILE+x}" ]] || { echo "var_LIFECYCLE_ENGINE_FILE not defined"; return 1; }
    
    # Test that they point to correct files
    [[ "$var_EXIT_CODES_FILE" == *"/exit_codes.sh" ]] || { echo "var_EXIT_CODES_FILE incorrect: $var_EXIT_CODES_FILE"; return 1; }
    [[ "$var_FLOW_FILE" == *"/flow.sh" ]] || { echo "var_FLOW_FILE incorrect: $var_FLOW_FILE"; return 1; }
    [[ "$var_LOG_FILE" == *"/log.sh" ]] || { echo "var_LOG_FILE incorrect: $var_LOG_FILE"; return 1; }
    [[ "$var_LIFECYCLE_ENGINE_FILE" == *"/engine.sh" ]] || { echo "var_LIFECYCLE_ENGINE_FILE incorrect: $var_LIFECYCLE_ENGINE_FILE"; return 1; }
}

# Test VROOLI_CONTEXT detection
@test "VROOLI_CONTEXT is set correctly" {
    [[ -n "${VROOLI_CONTEXT+x}" ]] || { echo "VROOLI_CONTEXT not defined"; return 1; }
    [[ "$VROOLI_CONTEXT" == "monorepo" || "$VROOLI_CONTEXT" == "standalone" ]] || { echo "VROOLI_CONTEXT has invalid value: $VROOLI_CONTEXT"; return 1; }
}

# Test monorepo-specific variables (if in monorepo context)
@test "monorepo variables are defined correctly if in monorepo context" {
    if [[ "$VROOLI_CONTEXT" == "monorepo" ]]; then
        [[ -n "${var_PACKAGES_DIR+x}" ]] || { echo "var_PACKAGES_DIR not defined in monorepo context"; return 1; }
        [[ -n "${var_BACKUPS_DIR+x}" ]] || { echo "var_BACKUPS_DIR not defined in monorepo context"; return 1; }
        [[ -n "${var_DATA_DIR+x}" ]] || { echo "var_DATA_DIR not defined in monorepo context"; return 1; }
        [[ -n "${var_DEST_DIR+x}" ]] || { echo "var_DEST_DIR not defined in monorepo context"; return 1; }
        
        [[ "$var_PACKAGES_DIR" == *"/packages" ]] || { echo "var_PACKAGES_DIR incorrect: $var_PACKAGES_DIR"; return 1; }
        [[ "$var_DEST_DIR" == *"/dist" ]] || { echo "var_DEST_DIR incorrect: $var_DEST_DIR"; return 1; }
    else
        skip "Not in monorepo context"
    fi
}

# Test environment file variables (if in monorepo context)
@test "environment file variables are defined correctly if in monorepo context" {
    if [[ "$VROOLI_CONTEXT" == "monorepo" ]]; then
        [[ -n "${var_ENV_DEV_FILE+x}" ]] || { echo "var_ENV_DEV_FILE not defined"; return 1; }
        [[ -n "${var_ENV_PROD_FILE+x}" ]] || { echo "var_ENV_PROD_FILE not defined"; return 1; }
        [[ -n "${var_ENV_EXAMPLE_FILE+x}" ]] || { echo "var_ENV_EXAMPLE_FILE not defined"; return 1; }
        
        [[ "$var_ENV_DEV_FILE" == *"/.env-dev" ]] || { echo "var_ENV_DEV_FILE incorrect: $var_ENV_DEV_FILE"; return 1; }
        [[ "$var_ENV_PROD_FILE" == *"/.env-prod" ]] || { echo "var_ENV_PROD_FILE incorrect: $var_ENV_PROD_FILE"; return 1; }
        [[ "$var_ENV_EXAMPLE_FILE" == *"/.env-example" ]] || { echo "var_ENV_EXAMPLE_FILE incorrect: $var_ENV_EXAMPLE_FILE"; return 1; }
    else
        skip "Not in monorepo context"
    fi
}

# Test directory hierarchy consistency
@test "directory hierarchy is consistent" {
    # Test that child directories are children of parent directories
    [[ "$var_LIB_UTILS_DIR" == "$var_LIB_DIR"* ]] || { echo "var_LIB_UTILS_DIR is not under var_LIB_DIR"; return 1; }
    [[ "$var_LIB_DIR" == "$var_SCRIPTS_DIR"* ]] || { echo "var_LIB_DIR is not under var_SCRIPTS_DIR"; return 1; }
    [[ "$var_SCRIPTS_DIR" == "$var_ROOT_DIR"* ]] || { echo "var_SCRIPTS_DIR is not under var_ROOT_DIR"; return 1; }
    [[ "$var_VROOLI_CONFIG_DIR" == "$var_ROOT_DIR"* ]] || { echo "var_VROOLI_CONFIG_DIR is not under var_ROOT_DIR"; return 1; }
}

# Test path absoluteness
@test "all paths are absolute" {
    local vars=(
        "var_LIB_UTILS_DIR" "var_LIB_DIR" "var_SCRIPTS_DIR" "var_ROOT_DIR"
        "var_VROOLI_CONFIG_DIR" "var_SERVICE_JSON_FILE" "var_EXIT_CODES_FILE"
        "var_FLOW_FILE" "var_LOG_FILE"
    )
    
    for var_name in "${vars[@]}"; do
        local var_value
        eval "var_value=\$$var_name"
        [[ "$var_value" == /* ]] || { echo "$var_name is not an absolute path: $var_value"; return 1; }
    done
}

# Test that variables are exported
@test "key variables are exported" {
    # Test that important variables are available to subprocesses
    run bash -c "echo \$var_ROOT_DIR"
    [[ -n "$output" ]] || { echo "var_ROOT_DIR not exported"; return 1; }
    
    run bash -c "echo \$VROOLI_CONTEXT"
    [[ -n "$output" ]] || { echo "VROOLI_CONTEXT not exported"; return 1; }
}

# Integration test
@test "all utility file variables point to existing files" {
    # Test files that should exist in the utils directory
    [[ -f "$var_EXIT_CODES_FILE" ]] || { echo "Exit codes file does not exist: $var_EXIT_CODES_FILE"; return 1; }
    [[ -f "$var_FLOW_FILE" ]] || { echo "Flow file does not exist: $var_FLOW_FILE"; return 1; }
    [[ -f "$var_LOG_FILE" ]] || { echo "Log file does not exist: $var_LOG_FILE"; return 1; }
}

# Test variable naming conventions
@test "variables follow naming convention" {
    # All var_ prefixed variables should be uppercase after the prefix
    local var_vars
    var_vars=$(compgen -v | grep '^var_' || true)
    
    if [[ -n "$var_vars" ]]; then
        while IFS= read -r var; do
            # Check that everything after var_ is uppercase
            local suffix="${var#var_}"
            [[ "$suffix" == "${suffix^^}" ]] || { echo "Variable $var does not follow uppercase naming convention"; return 1; }
        done <<< "$var_vars"
    fi
}

# Edge case: test sourcing multiple times
@test "var.sh can be sourced multiple times without errors" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/var.sh'
        source '${BATS_TEST_DIRNAME}/var.sh'
        echo 'success'
    "
    assert_success
    assert_output "success"
}