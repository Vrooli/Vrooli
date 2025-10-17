#!/usr/bin/env bash
################################################################################
# PostGIS Unit Tests
# Test individual library functions (<60s)
################################################################################

set -euo pipefail

# Determine script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_ROOT="$(cd "$SCRIPT_DIR/../../../../" && pwd)"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC2154  # var_LOG_FILE is set by var.sh
source "${var_LOG_FILE}"

# Test utility functions
test::suite() { echo -e "\n╔════════════════════════════════════╗\n║ $* ║\n╚════════════════════════════════════════╝"; }
test::start() { echo -n "  Testing $*... "; }
test::pass() { echo -e "✅ $*"; }
test::fail() { echo -e "❌ $*"; }
test::success() { echo -e "\n✅ $*"; }
test::error() { echo -e "\n❌ $*"; }
log::success() { test::success "$@"; }
log::error() { test::error "$@"; }

# Source PostGIS libraries
source "${APP_ROOT}/resources/postgis/lib/core.sh"
source "${APP_ROOT}/resources/postgis/lib/common.sh"
source "${APP_ROOT}/resources/postgis/config/defaults.sh"

# Test functions
test_config_loading() {
    test::start "Configuration loading"
    
    # Check required variables are set
    if [[ -n "${POSTGIS_PG_DATABASE}" ]] && \
       [[ -n "${POSTGIS_PG_USER}" ]] && \
       [[ -n "${POSTGIS_DEFAULT_SRID}" ]]; then
        test::pass "Configuration loaded successfully"
    else
        test::fail "Configuration variables not set"
        return 1
    fi
}

test_directory_initialization() {
    test::start "Directory initialization"
    
    # Test directory creation function
    postgis_init_dirs
    
    if [[ -d "${POSTGIS_DATA_DIR}" ]] && \
       [[ -d "${POSTGIS_IMPORT_DIR}" ]] && \
       [[ -d "${POSTGIS_EXPORT_DIR}" ]] && \
       [[ -d "${POSTGIS_SQL_DIR}" ]]; then
        test::pass "Directories initialized successfully"
    else
        test::fail "Directory initialization failed"
        return 1
    fi
}

test_extension_list() {
    test::start "Extension list generation"
    
    # Test with default settings
    if [[ "${POSTGIS_EXTENSIONS}" == *"postgis"* ]] && \
       [[ "${POSTGIS_EXTENSIONS}" == *"postgis_raster"* ]]; then
        test::pass "Default extensions configured"
    else
        test::fail "Extension list incorrect"
        return 1
    fi
}

test_port_configuration() {
    test::start "Port configuration"
    
    # Check port is in valid range
    local port="${POSTGIS_STANDALONE_PORT:-5434}"
    
    if [[ $port -ge 1024 ]] && [[ $port -le 65535 ]]; then
        test::pass "Port $port is in valid range"
    else
        test::fail "Port $port is invalid"
        return 1
    fi
}

test_schema_validation() {
    test::start "Schema validation"
    
    local schema_file="${APP_ROOT}/resources/postgis/config/schema.json"
    
    if [[ -f "$schema_file" ]]; then
        # Basic JSON validation
        if python3 -m json.tool "$schema_file" > /dev/null 2>&1; then
            test::pass "Schema JSON is valid"
        else
            test::fail "Schema JSON is invalid"
            return 1
        fi
    else
        test::fail "Schema file not found"
        return 1
    fi
}

test_runtime_config() {
    test::start "Runtime configuration"
    
    local runtime_file="${APP_ROOT}/resources/postgis/config/runtime.json"
    
    if [[ -f "$runtime_file" ]]; then
        # Check required fields
        local startup_order
        startup_order=$(jq -r '.startup_order' "$runtime_file" 2>/dev/null)
        local dependencies
        dependencies=$(jq -r '.dependencies[]' "$runtime_file" 2>/dev/null)
        
        if [[ -n "$startup_order" ]] && [[ "$dependencies" == *"postgres"* ]]; then
            test::pass "Runtime config valid (order: $startup_order)"
        else
            test::fail "Runtime config missing required fields"
            return 1
        fi
    else
        test::fail "Runtime config not found"
        return 1
    fi
}

test_cli_structure() {
    test::start "CLI command structure"
    
    # Check CLI script exists and is executable
    local cli_script="${APP_ROOT}/resources/postgis/cli.sh"
    
    if [[ -f "$cli_script" ]] && [[ -x "$cli_script" ]]; then
        # Check for required command handlers
        if grep -q "manage::install" "$cli_script" && \
           grep -q "test::smoke" "$cli_script" && \
           grep -q "content::add" "$cli_script"; then
            test::pass "CLI structure correct"
        else
            test::fail "CLI missing required handlers"
            return 1
        fi
    else
        test::fail "CLI script not found or not executable"
        return 1
    fi
}

test_health_check_logic() {
    test::start "Health check logic"
    
    # Test the health check function exists
    if type -t postgis::status::check &>/dev/null; then
        test::pass "Health check function defined"
    else
        test::fail "Health check function not found"
        return 1
    fi
}

# Main test execution
main() {
    test::suite "PostGIS Unit Tests"
    
    local failed=0
    
    # Run tests
    test_config_loading || ((failed++))
    test_directory_initialization || ((failed++))
    test_extension_list || ((failed++))
    test_port_configuration || ((failed++))
    test_schema_validation || ((failed++))
    test_runtime_config || ((failed++))
    test_cli_structure || ((failed++))
    test_health_check_logic || ((failed++))
    
    # Summary
    echo
    if [[ $failed -eq 0 ]]; then
        test::success "All unit tests passed!"
        return 0
    else
        test::error "$failed unit tests failed"
        return 1
    fi
}

# Run tests
main "$@"