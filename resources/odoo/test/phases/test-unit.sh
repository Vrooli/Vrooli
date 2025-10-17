#!/usr/bin/env bash
################################################################################
# Odoo Resource - Unit Tests
# Test library functions in isolation (< 60 seconds)
################################################################################

set -euo pipefail

# Determine paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
RESOURCE_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="$(cd "$RESOURCE_DIR/../.." && pwd)"

# Source utilities and libraries
source "$APP_ROOT/scripts/lib/utils/log.sh"
source "$RESOURCE_DIR/config/defaults.sh"

# Source Odoo libraries for testing
for lib in common install status docker content; do
    lib_file="$RESOURCE_DIR/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        source "$lib_file"
    fi
done

################################################################################
# Unit Test Functions
################################################################################

test_config_defaults() {
    log::info "Testing: Configuration defaults"
    
    if [[ -n "${ODOO_VERSION:-}" ]] && [[ -n "${ODOO_CONTAINER_NAME:-}" ]]; then
        log::success "✓ Configuration variables defined"
        return 0
    else
        log::error "✗ Configuration variables missing"
        return 1
    fi
}

test_port_allocation() {
    log::info "Testing: Port allocation"
    
    # Test port registry integration
    local port
    port=$(source "$APP_ROOT/scripts/resources/port_registry.sh" && get_port odoo)
    
    if [[ "$port" =~ ^[0-9]+$ ]] && [[ "$port" -gt 1024 ]]; then
        log::success "✓ Port allocation works: $port"
        return 0
    else
        log::error "✗ Port allocation failed"
        return 1
    fi
}

test_docker_functions() {
    log::info "Testing: Docker utility functions"
    
    # Check if docker functions are defined
    if type -t odoo_is_installed &>/dev/null && \
       type -t odoo_is_running &>/dev/null; then
        log::success "✓ Docker functions defined"
        return 0
    else
        log::error "✗ Docker functions not found"
        return 1
    fi
}

test_python_examples() {
    log::info "Testing: Python example scripts"
    
    if ! command -v python3 &>/dev/null; then
        log::warning "⚠ Python3 not available, skipping"
        return 0
    fi
    
    local failed=0
    
    # Test syntax of example scripts
    for example in "$RESOURCE_DIR/examples"/*.py; do
        if [[ -f "$example" ]]; then
            if python3 -m py_compile "$example" &>/dev/null; then
                log::success "✓ Valid syntax: $(basename "$example")"
            else
                log::error "✗ Invalid syntax: $(basename "$example")"
                ((failed++))
            fi
        fi
    done
    
    return $failed
}

test_data_directory() {
    log::info "Testing: Data directory configuration"
    
    # Check if data directory path is configured
    if [[ -n "${ODOO_DATA_DIR:-}" ]]; then
        log::success "✓ Data directory configured: $ODOO_DATA_DIR"
        # Try to create it to test permissions
        if mkdir -p "$ODOO_DATA_DIR" 2>/dev/null; then
            log::success "✓ Data directory writable"
            return 0
        else
            log::warning "⚠ Data directory not writable (may need sudo)"
            return 0
        fi
    else
        log::error "✗ Data directory not configured"
        return 1
    fi
}

test_cli_framework() {
    log::info "Testing: CLI framework integration"
    
    # Test that CLI sources v2 framework
    if grep -q "cli-command-framework-v2.sh" "$RESOURCE_DIR/cli.sh"; then
        log::success "✓ Uses v2.0 CLI framework"
        return 0
    else
        log::error "✗ Not using v2.0 CLI framework"
        return 1
    fi
}

################################################################################
# Main Execution
################################################################################

main() {
    local failed=0
    
    log::info "=== Odoo Unit Tests ==="
    
    # Run each test
    test_config_defaults || ((failed++))
    test_port_allocation || ((failed++))
    test_docker_functions || ((failed++))
    test_python_examples || ((failed++))
    test_data_directory || ((failed++))
    test_cli_framework || ((failed++))
    
    # Summary
    if [[ $failed -eq 0 ]]; then
        log::success "All unit tests passed!"
        return 0
    else
        log::error "$failed unit test(s) failed"
        return 1
    fi
}

# Execute
main "$@"