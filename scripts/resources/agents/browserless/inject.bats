#!/usr/bin/env bats
# Tests for Browserless inject.sh script
bats_require_minimum_version 1.5.0

# Load Vrooli test infrastructure
source "${BATS_TEST_DIRNAME}/../../../__test/fixtures/setup.bash"

# Expensive setup operations run once per file
setup_file() {
    # Use Vrooli service test setup
    vrooli_setup_service_test "browserless"
    
    # Set up directories and paths once
    export SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    export MOCK_DIR="${SCRIPT_DIR}/../../../__test/fixtures/mocks"
    
    # Load var.sh for directory variables
    # shellcheck disable=SC1091
    source "$(dirname "$(dirname "$(dirname "${SCRIPT_DIR}")")")/lib/utils/var.sh"
    
    # Load common resources
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"
    
    # Load the browserless mock once
    if [[ -f "$MOCK_DIR/browserless.sh" ]]; then
        # shellcheck disable=SC1091
        source "$MOCK_DIR/browserless.sh"
    fi
    
    # Load inject.sh with all its dependencies
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/inject.sh"
    
    # Export key functions for BATS subshells
    export -f browserless_inject::main
    export -f browserless_inject::usage
    export -f browserless_inject::validate_config
    export -f browserless_inject::inject_data
    export -f browserless_inject::check_status
    export -f browserless_inject::execute_rollback
    export -f browserless_inject::check_accessibility
    export -f browserless_inject::validate_scripts
    export -f browserless_inject::inject_scripts
    export -f browserless_inject::inject_functions
    export -f browserless_inject::inject_contexts
    export -f browserless_inject::apply_configurations
    
    # Export log functions
    log::header() { echo "=== $* ==="; }
    log::info() { echo "[INFO] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    log::success() { echo "[SUCCESS] $*"; }
    log::warning() { echo "[WARNING] $*" >&2; }
    log::debug() { echo "[DEBUG] $*"; }
    export -f log::header log::info log::error log::success log::warning log::debug
    
    # Mock VROOLI_PROJECT_ROOT if not set
    export VROOLI_PROJECT_ROOT="${var_ROOT_DIR}"
}

# Lightweight per-test setup
setup() {
    # Setup standard Vrooli mocks
    vrooli_auto_setup
    
    # Ensure mock log directory exists to prevent warnings
    MOCK_LOG_DIR="${MOCK_RESPONSES_DIR:-${BATS_TEST_DIRNAME}/../../../../data/test-outputs/mock-logs}"
    if [[ ! -d "$MOCK_LOG_DIR" ]]; then
        mkdir -p "$MOCK_LOG_DIR" 2>/dev/null || true
    fi
    
    # Reset mock state to clean slate for each test
    if declare -f mock::browserless::reset >/dev/null 2>&1; then
        mock::browserless::reset
    fi
    
    # Set test-specific environment variables
    export BROWSERLESS_HOST="http://localhost:3001"
    export BROWSERLESS_DATA_DIR="${BATS_TEST_TMPDIR}/browserless"
    export BROWSERLESS_SCRIPTS_DIR="${BROWSERLESS_DATA_DIR}/scripts"
    export BROWSERLESS_FUNCTIONS_DIR="${BROWSERLESS_DATA_DIR}/functions"
    
    # Create test data directories
    mkdir -p "$BROWSERLESS_DATA_DIR" "$BROWSERLESS_SCRIPTS_DIR" "$BROWSERLESS_FUNCTIONS_DIR"
    
    # Configure browserless mock with clean default state
    if declare -f mock::browserless::set_server_status >/dev/null 2>&1; then
        mock::browserless::set_server_status "running"
        mock::browserless::set_health_status "healthy"
    fi
    
    # Create test fixture files
    export TEST_SCRIPT_FILE="${BATS_TEST_TMPDIR}/test-script.js"
    export TEST_FUNCTION_FILE="${BATS_TEST_TMPDIR}/test-function.js"
    
    cat > "$TEST_SCRIPT_FILE" << 'EOF'
// Test Puppeteer script
const puppeteer = require('puppeteer');

module.exports = async (page) => {
    await page.goto('https://example.com');
    return await page.title();
};
EOF
    
    cat > "$TEST_FUNCTION_FILE" << 'EOF'
// Test Browserless function
module.exports = async ({ page, context }) => {
    await page.goto(context.url);
    return { title: await page.title() };
};
EOF
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
    
    # Clean up test data directories
    rm -rf "${BATS_TEST_TMPDIR}/browserless" 2>/dev/null || true
    rm -f "$TEST_SCRIPT_FILE" "$TEST_FUNCTION_FILE" 2>/dev/null || true
}

# ============================================================================
# Test Helper Functions
# ============================================================================

# Create test configuration JSON
create_test_config() {
    local type="$1"
    
    case "$type" in
        "scripts")
            cat << EOF
{
  "scripts": [
    {
      "name": "test_script",
      "file": "$(basename "$TEST_SCRIPT_FILE")",
      "type": "puppeteer"
    }
  ]
}
EOF
            ;;
        "functions")
            cat << EOF
{
  "functions": [
    {
      "name": "test_function",
      "file": "$(basename "$TEST_FUNCTION_FILE")",
      "endpoint": "/function/test_function"
    }
  ]
}
EOF
            ;;
        "contexts")
            cat << EOF
{
  "browser_contexts": [
    {
      "name": "mobile",
      "viewport": {
        "width": 375,
        "height": 667
      },
      "user_agent": "Mozilla/5.0 Mobile"
    }
  ]
}
EOF
            ;;
        "configurations")
            cat << EOF
{
  "configurations": [
    {
      "key": "max_concurrent_sessions",
      "value": 10
    },
    {
      "key": "preboot_chrome",
      "value": true
    }
  ]
}
EOF
            ;;
        "complete")
            # Move test files to VROOLI_PROJECT_ROOT for validation
            cp "$TEST_SCRIPT_FILE" "${VROOLI_PROJECT_ROOT}/test-script.js"
            cp "$TEST_FUNCTION_FILE" "${VROOLI_PROJECT_ROOT}/test-function.js"
            cat << EOF
{
  "scripts": [
    {
      "name": "test_script",
      "file": "test-script.js",
      "type": "puppeteer"
    }
  ],
  "functions": [
    {
      "name": "test_function",
      "file": "test-function.js",
      "endpoint": "/function/test_function"
    }
  ],
  "browser_contexts": [
    {
      "name": "desktop",
      "viewport": {
        "width": 1920,
        "height": 1080
      }
    }
  ],
  "configurations": [
    {
      "key": "timeout",
      "value": 30000
    }
  ]
}
EOF
            ;;
        *)
            echo "{}"
            ;;
    esac
}

# ============================================================================
# Script Loading Tests
# ============================================================================

@test "inject.sh loads without errors" {
    # Should load successfully in setup_file
    [ "$?" -eq 0 ]
}

@test "inject.sh defines required functions" {
    # Functions should be available from setup
    declare -f browserless_inject::main >/dev/null
    declare -f browserless_inject::validate_config >/dev/null
    declare -f browserless_inject::inject_data >/dev/null
    declare -f browserless_inject::check_status >/dev/null
}

# ============================================================================
# Usage and Help Tests
# ============================================================================

@test "browserless_inject::usage displays help text" {
    run browserless_inject::usage
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Browserless Data Injection Adapter" ]]
    [[ "$output" =~ "--validate" ]]
    [[ "$output" =~ "--inject" ]]
    [[ "$output" =~ "--status" ]]
    [[ "$output" =~ "--rollback" ]]
}

@test "inject.sh shows usage when called with --help" {
    run browserless_inject::main --help
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Browserless Data Injection Adapter" ]]
    [[ "$output" =~ "USAGE:" ]]
}

# ============================================================================
# Configuration Validation Tests
# ============================================================================

@test "browserless_inject::validate_config accepts valid script configuration" {
    local config
    config=$(create_test_config "scripts")
    
    # Move test file to expected location for validation
    cp "$TEST_SCRIPT_FILE" "${VROOLI_PROJECT_ROOT}/$(basename "$TEST_SCRIPT_FILE")"
    
    run browserless_inject::validate_config "$config"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "valid" ]]
    
    # Clean up
    rm -f "${VROOLI_PROJECT_ROOT}/$(basename "$TEST_SCRIPT_FILE")"
}

@test "browserless_inject::validate_config accepts valid function configuration" {
    local config
    config=$(create_test_config "functions")
    
    # Move test file to expected location for validation
    cp "$TEST_FUNCTION_FILE" "${VROOLI_PROJECT_ROOT}/$(basename "$TEST_FUNCTION_FILE")"
    
    run browserless_inject::validate_config "$config"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "valid" ]]
    
    # Clean up
    rm -f "${VROOLI_PROJECT_ROOT}/$(basename "$TEST_FUNCTION_FILE")"
}

@test "browserless_inject::validate_config accepts valid context configuration" {
    local config
    config=$(create_test_config "contexts")
    
    run browserless_inject::validate_config "$config"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "valid" ]]
}

@test "browserless_inject::validate_config rejects invalid JSON" {
    run browserless_inject::validate_config '{"invalid": json}'
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Invalid JSON" ]]
}

@test "browserless_inject::validate_config rejects configuration with missing files" {
    local config='{"scripts": [{"name": "missing", "file": "nonexistent.js", "type": "puppeteer"}]}'
    
    run browserless_inject::validate_config "$config"
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not found" ]]
}

@test "browserless_inject::validate_config rejects script with invalid type" {
    local config
    # Move test file to expected location for validation
    cp "$TEST_SCRIPT_FILE" "${VROOLI_PROJECT_ROOT}/$(basename "$TEST_SCRIPT_FILE")"
    
    config='{"scripts": [{"name": "test", "file": "'"$(basename "$TEST_SCRIPT_FILE")"'", "type": "invalid_type"}]}'
    
    run browserless_inject::validate_config "$config"
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Invalid script type" ]]
    
    # Clean up
    rm -f "${VROOLI_PROJECT_ROOT}/$(basename "$TEST_SCRIPT_FILE")"
}

@test "browserless_inject::validate_config rejects empty configuration" {
    run browserless_inject::validate_config '{}'
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "must have" ]]
}

# ============================================================================
# Data Injection Tests
# ============================================================================

@test "browserless_inject::inject_data injects scripts successfully" {
    local config
    config=$(create_test_config "complete")
    
    run browserless_inject::inject_data "$config"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "injection completed" ]]
    
    # Verify script was installed
    [ -f "${BROWSERLESS_SCRIPTS_DIR}/test_script.js" ]
    [ -f "${BROWSERLESS_SCRIPTS_DIR}/test_script.meta.json" ]
    
    # Clean up
    rm -f "${VROOLI_PROJECT_ROOT}/test-script.js" "${VROOLI_PROJECT_ROOT}/test-function.js"
}

@test "browserless_inject::inject_data injects functions successfully" {
    local config
    config=$(create_test_config "complete")
    
    run browserless_inject::inject_data "$config"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "injection completed" ]]
    
    # Verify function was installed
    [ -f "${BROWSERLESS_FUNCTIONS_DIR}/test_function.js" ]
    [ -f "${BROWSERLESS_FUNCTIONS_DIR}/test_function.meta.json" ]
    
    # Clean up
    rm -f "${VROOLI_PROJECT_ROOT}/test-script.js" "${VROOLI_PROJECT_ROOT}/test-function.js"
}

@test "browserless_inject::inject_data creates browser contexts" {
    local config
    config=$(create_test_config "contexts")
    
    run browserless_inject::inject_data "$config"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "injection completed" ]]
    
    # Verify context was created
    [ -f "${BROWSERLESS_DATA_DIR}/contexts/mobile.json" ]
}

@test "browserless_inject::inject_data applies configurations" {
    local config
    config=$(create_test_config "configurations")
    
    run browserless_inject::inject_data "$config"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "injection completed" ]]
    
    # Verify configuration file was created
    [ -f "${BROWSERLESS_DATA_DIR}/config.json" ]
}

# ============================================================================
# Status Check Tests
# ============================================================================

@test "browserless_inject::check_status reports running service" {
    # Set up running service
    if declare -f mock::browserless::scenario::create_running_service >/dev/null 2>&1; then
        mock::browserless::scenario::create_running_service
    fi
    
    local config
    config=$(create_test_config "scripts")
    
    run browserless_inject::check_status "$config"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "running" ]] || [[ "$output" =~ "responding" ]]
}

@test "browserless_inject::check_status detects stopped service" {
    # Set up stopped service
    if declare -f mock::browserless::set_server_status >/dev/null 2>&1; then
        mock::browserless::set_server_status "stopped"
        mock::browserless::set_health_status "unhealthy"
    fi
    
    local config
    config=$(create_test_config "scripts")
    
    run browserless_inject::check_status "$config"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "not running" ]]
}

@test "browserless_inject::check_status counts installed scripts" {
    # Create some test script files
    echo "test script 1" > "${BROWSERLESS_SCRIPTS_DIR}/script1.js"
    echo "test script 2" > "${BROWSERLESS_SCRIPTS_DIR}/script2.js"
    
    local config
    config=$(create_test_config "scripts")
    
    run browserless_inject::check_status "$config"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Found 2 scripts" ]]
}

@test "browserless_inject::check_status counts installed functions" {
    # Create some test function files
    echo "test function 1" > "${BROWSERLESS_FUNCTIONS_DIR}/func1.js"
    echo "test function 2" > "${BROWSERLESS_FUNCTIONS_DIR}/func2.js"
    echo "test function 3" > "${BROWSERLESS_FUNCTIONS_DIR}/func3.js"
    
    local config
    config=$(create_test_config "functions")
    
    run browserless_inject::check_status "$config"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Found 3 functions" ]]
}

# ============================================================================
# Rollback Tests
# ============================================================================

@test "browserless_inject::execute_rollback with no actions" {
    run browserless_inject::execute_rollback
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "No.*rollback actions" ]]
}

@test "rollback removes injected scripts" {
    local config
    config=$(create_test_config "complete")
    
    # First inject data
    browserless_inject::inject_data "$config"
    
    # Verify files exist
    [ -f "${BROWSERLESS_SCRIPTS_DIR}/test_script.js" ]
    [ -f "${BROWSERLESS_SCRIPTS_DIR}/test_script.meta.json" ]
    
    # Execute rollback
    run browserless_inject::execute_rollback
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "rollback" ]]
    
    # Verify files were removed
    [ ! -f "${BROWSERLESS_SCRIPTS_DIR}/test_script.js" ]
    [ ! -f "${BROWSERLESS_SCRIPTS_DIR}/test_script.meta.json" ]
    
    # Clean up
    rm -f "${VROOLI_PROJECT_ROOT}/test-script.js" "${VROOLI_PROJECT_ROOT}/test-function.js"
}

# ============================================================================
# Main Function Tests
# ============================================================================

@test "browserless_inject::main validates configuration" {
    local config
    config=$(create_test_config "scripts")
    
    # Move test file to expected location for validation
    cp "$TEST_SCRIPT_FILE" "${VROOLI_PROJECT_ROOT}/$(basename "$TEST_SCRIPT_FILE")"
    
    run browserless_inject::main --validate "$config"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "valid" ]]
    
    # Clean up
    rm -f "${VROOLI_PROJECT_ROOT}/$(basename "$TEST_SCRIPT_FILE")"
}

@test "browserless_inject::main injects data" {
    local config
    config=$(create_test_config "complete")
    
    run browserless_inject::main --inject "$config"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "injection completed" ]]
    
    # Clean up
    rm -f "${VROOLI_PROJECT_ROOT}/test-script.js" "${VROOLI_PROJECT_ROOT}/test-function.js"
}

@test "browserless_inject::main checks status" {
    local config
    config=$(create_test_config "scripts")
    
    run browserless_inject::main --status "$config"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "status" ]] || [[ "$output" =~ "Found" ]]
}

@test "browserless_inject::main executes rollback" {
    run browserless_inject::main --rollback ""
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "rollback" ]]
}

@test "browserless_inject::main rejects unknown action" {
    run browserless_inject::main --unknown-action ""
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown action" ]]
}

@test "browserless_inject::main requires configuration for most actions" {
    run browserless_inject::main --validate
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Configuration JSON required" ]]
}

# ============================================================================
# Integration Tests with Mock
# ============================================================================

@test "accessibility check with running service" {
    # Set up running service that responds to pressure endpoint
    if declare -f mock::browserless::scenario::create_running_service >/dev/null 2>&1; then
        mock::browserless::scenario::create_running_service
    fi
    
    run browserless_inject::check_accessibility
    
    [ "$status" -eq 0 ]
}

@test "accessibility check with stopped service" {
    # Set up stopped service
    if declare -f mock::browserless::set_server_status >/dev/null 2>&1; then
        mock::browserless::set_server_status "stopped"
    fi
    
    run browserless_inject::check_accessibility
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not accessible" ]]
}

@test "injection works without running service" {
    # Set up stopped service (file-based injection should still work)
    if declare -f mock::browserless::set_server_status >/dev/null 2>&1; then
        mock::browserless::set_server_status "stopped"
    fi
    
    local config
    config=$(create_test_config "complete")
    
    run browserless_inject::inject_data "$config"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "injection completed" ]]
    
    # Verify files were created despite service being stopped
    [ -f "${BROWSERLESS_SCRIPTS_DIR}/test_script.js" ]
    [ -f "${BROWSERLESS_FUNCTIONS_DIR}/test_function.js" ]
    
    # Clean up
    rm -f "${VROOLI_PROJECT_ROOT}/test-script.js" "${VROOLI_PROJECT_ROOT}/test-function.js"
}

@test "complete injection and status workflow" {
    # Set up running service
    if declare -f mock::browserless::scenario::create_running_service >/dev/null 2>&1; then
        mock::browserless::scenario::create_running_service
    fi
    
    local config
    config=$(create_test_config "complete")
    
    # Validate configuration
    run browserless_inject::main --validate "$config"
    [ "$status" -eq 0 ]
    
    # Inject data
    run browserless_inject::main --inject "$config"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "injection completed" ]]
    
    # Check status
    run browserless_inject::main --status "$config"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Found" ]]
    
    # Execute rollback
    run browserless_inject::main --rollback "$config"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "rollback" ]]
    
    # Clean up
    rm -f "${VROOLI_PROJECT_ROOT}/test-script.js" "${VROOLI_PROJECT_ROOT}/test-function.js"
}