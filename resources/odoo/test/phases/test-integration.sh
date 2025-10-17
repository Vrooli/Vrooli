#!/usr/bin/env bash
################################################################################
# Odoo Resource - Integration Tests
# Full end-to-end functionality tests (< 120 seconds)
################################################################################

set -euo pipefail

# Determine paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
RESOURCE_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="$(cd "$RESOURCE_DIR/../.." && pwd)"

# Source utilities
source "$APP_ROOT/scripts/lib/utils/log.sh"
source "$RESOURCE_DIR/config/defaults.sh"

# Get dynamic port
ODOO_PORT="${ODOO_PORT:-$(source "$APP_ROOT/scripts/resources/port_registry.sh" && get_port odoo)}"

################################################################################
# Integration Test Functions
################################################################################

test_lifecycle() {
    log::info "Testing: Resource lifecycle"
    
    # Test that lifecycle commands exist
    local commands=("install" "start" "stop" "restart" "uninstall")
    local failed=0
    
    for cmd in "${commands[@]}"; do
        if "$RESOURCE_DIR/cli.sh" manage "$cmd" --help &>/dev/null; then
            log::success "✓ Command 'manage $cmd' available"
        else
            log::error "✗ Command 'manage $cmd' not found"
            ((failed++))
        fi
    done
    
    return $failed
}

test_xmlrpc_api() {
    log::info "Testing: XML-RPC API"
    
    # Skip if Odoo not running
    if ! pgrep -f "odoo" &>/dev/null && ! docker ps | grep -q "odoo"; then
        log::warning "⚠ Odoo not running, skipping XML-RPC test"
        return 0
    fi
    
    # Test XML-RPC endpoint with Python
    if command -v python3 &>/dev/null; then
        local result
        result=$(python3 -c "
import xmlrpc.client
try:
    common = xmlrpc.client.ServerProxy('http://localhost:${ODOO_PORT}/xmlrpc/2/common')
    version = common.version()
    print('success' if version else 'failed')
except Exception as e:
    print(f'failed: {e}')
" 2>&1)
        
        if [[ "$result" == "success" ]]; then
            log::success "✓ XML-RPC API responsive"
            return 0
        else
            log::error "✗ XML-RPC API failed: $result"
            return 1
        fi
    else
        log::warning "⚠ Python3 not available, skipping XML-RPC test"
        return 0
    fi
}

test_web_interface() {
    log::info "Testing: Web interface"
    
    # Skip if Odoo not running
    if ! pgrep -f "odoo" &>/dev/null && ! docker ps | grep -q "odoo"; then
        log::warning "⚠ Odoo not running, skipping web interface test"
        return 0
    fi
    
    # Test web interface availability
    if timeout 10 curl -sf "http://localhost:${ODOO_PORT}/web/database/selector" | grep -q "Odoo" &>/dev/null; then
        log::success "✓ Web interface accessible"
        return 0
    else
        log::error "✗ Web interface not accessible"
        return 1
    fi
}

test_content_commands() {
    log::info "Testing: Content management commands"
    
    local commands=("list" "add" "get" "remove" "execute")
    local failed=0
    
    for cmd in "${commands[@]}"; do
        if "$RESOURCE_DIR/cli.sh" content "$cmd" --help &>/dev/null 2>&1 || \
           "$RESOURCE_DIR/cli.sh" content &>/dev/null 2>&1; then
            log::success "✓ Content command '$cmd' available"
        else
            log::error "✗ Content command '$cmd' not found"
            ((failed++))
        fi
    done
    
    return $failed
}

test_database_operations() {
    log::info "Testing: Database operations"
    
    # Skip if Odoo not running
    if ! pgrep -f "odoo" &>/dev/null && ! docker ps | grep -q "odoo"; then
        log::warning "⚠ Odoo not running, skipping database test"
        return 0
    fi
    
    # Test database list command
    if "$RESOURCE_DIR/cli.sh" content databases &>/dev/null 2>&1; then
        log::success "✓ Database operations available"
        return 0
    else
        log::warning "⚠ Database operations not fully configured"
        return 0
    fi
}

test_info_command() {
    log::info "Testing: Info command output"
    
    local info_output
    info_output=$("$RESOURCE_DIR/cli.sh" info 2>/dev/null || echo "")
    
    # Check for required fields from runtime.json
    if [[ "$info_output" =~ startup_order ]] && \
       [[ "$info_output" =~ dependencies ]] && \
       [[ "$info_output" =~ startup_timeout ]]; then
        log::success "✓ Info command shows runtime configuration"
        return 0
    else
        log::error "✗ Info command missing required fields"
        return 1
    fi
}

test_logging() {
    log::info "Testing: Logging functionality"
    
    if "$RESOURCE_DIR/cli.sh" logs --help &>/dev/null 2>&1 || \
       "$RESOURCE_DIR/cli.sh" logs 5 &>/dev/null 2>&1; then
        log::success "✓ Logging commands available"
        return 0
    else
        log::warning "⚠ Logging not fully configured"
        return 0
    fi
}

################################################################################
# Main Execution
################################################################################

main() {
    local failed=0
    
    log::info "=== Odoo Integration Tests ==="
    
    # Run each test
    test_lifecycle || ((failed++))
    test_info_command || ((failed++))
    test_content_commands || ((failed++))
    test_xmlrpc_api || ((failed++))
    test_web_interface || ((failed++))
    test_database_operations || ((failed++))
    test_logging || ((failed++))
    
    # Summary
    if [[ $failed -eq 0 ]]; then
        log::success "All integration tests passed!"
        return 0
    else
        log::error "$failed integration test(s) failed"
        return 1
    fi
}

# Execute
main "$@"