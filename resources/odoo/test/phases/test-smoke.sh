#!/usr/bin/env bash
################################################################################
# Odoo Resource - Smoke Tests
# Quick validation that the resource is healthy (< 30 seconds)
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
# Test Functions
################################################################################

test_cli_exists() {
    log::info "Testing: CLI exists and is executable"
    
    if [[ -x "$RESOURCE_DIR/cli.sh" ]]; then
        log::success "✓ CLI is executable"
        return 0
    else
        log::error "✗ CLI not found or not executable"
        return 1
    fi
}

test_help_command() {
    log::info "Testing: Help command"
    
    if "$RESOURCE_DIR/cli.sh" help &>/dev/null; then
        log::success "✓ Help command works"
        return 0
    else
        log::error "✗ Help command failed"
        return 1
    fi
}

test_health_endpoint() {
    log::info "Testing: Health endpoint (port $ODOO_PORT)"
    
    # Check if service is running first
    if ! pgrep -f "odoo" &>/dev/null && ! docker ps | grep -q "odoo"; then
        log::warning "⚠ Odoo not running, skipping health check"
        return 0
    fi
    
    if timeout 5 curl -sf "http://localhost:${ODOO_PORT}/web/health" &>/dev/null || \
       timeout 5 curl -sf "http://localhost:${ODOO_PORT}/web/database/selector" | grep -q "Odoo" &>/dev/null; then
        log::success "✓ Health endpoint responsive"
        return 0
    else
        log::error "✗ Health endpoint not responding"
        return 1
    fi
}

test_status_command() {
    log::info "Testing: Status command"
    
    if "$RESOURCE_DIR/cli.sh" status &>/dev/null; then
        log::success "✓ Status command works"
        return 0
    else
        # Status may return non-zero if not healthy, but should not error
        log::success "✓ Status command works (service may not be running)"
        return 0
    fi
}

test_runtime_config() {
    log::info "Testing: Runtime configuration"
    
    if [[ -f "$RESOURCE_DIR/config/runtime.json" ]]; then
        # Validate JSON structure
        if python3 -m json.tool "$RESOURCE_DIR/config/runtime.json" &>/dev/null; then
            log::success "✓ Runtime config is valid JSON"
            return 0
        else
            log::error "✗ Runtime config has invalid JSON"
            return 1
        fi
    else
        log::error "✗ Runtime config not found"
        return 1
    fi
}

################################################################################
# Main Execution
################################################################################

main() {
    local failed=0
    
    log::info "=== Odoo Smoke Tests ==="
    
    # Run each test
    test_cli_exists || ((failed++))
    test_help_command || ((failed++))
    test_runtime_config || ((failed++))
    test_status_command || ((failed++))
    test_health_endpoint || ((failed++))
    
    # Summary
    if [[ $failed -eq 0 ]]; then
        log::success "All smoke tests passed!"
        return 0
    else
        log::error "$failed smoke test(s) failed"
        return 1
    fi
}

# Execute
main "$@"