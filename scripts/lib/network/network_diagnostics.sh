#!/usr/bin/env bash
################################################################################
# Simplified Network Diagnostics
# 
# Basic network connectivity checks for setup phase
# Consolidated from the modular diagnostics system
################################################################################

set -euo pipefail

# Source dependencies
NETWORK_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "$NETWORK_DIR/../utils/var.sh"
source "$NETWORK_DIR/../utils/log.sh"

################################################################################
# Main Diagnostic Function
################################################################################

#######################################
# Run basic network diagnostics
# Returns:
#   0 if all critical tests pass, 1 otherwise
#######################################
network_diagnostics::run() {
    log::info "Running network diagnostics..."
    
    local has_issues=false
    
    # Check DNS resolution
    log::debug "Testing DNS resolution..."
    if ! host google.com >/dev/null 2>&1 && ! nslookup google.com >/dev/null 2>&1; then
        log::warning "DNS resolution failed - you may have connectivity issues"
        has_issues=true
    else
        log::debug "✓ DNS resolution working"
    fi
    
    # Check HTTP connectivity
    log::debug "Testing HTTP connectivity..."
    if command -v curl >/dev/null 2>&1; then
        if ! curl -sS --connect-timeout 5 -o /dev/null https://www.google.com 2>/dev/null; then
            log::warning "HTTP connectivity test failed"
            has_issues=true
        else
            log::debug "✓ HTTP connectivity working"
        fi
    elif command -v wget >/dev/null 2>&1; then
        if ! wget -q --timeout=5 -O /dev/null https://www.google.com 2>/dev/null; then
            log::warning "HTTP connectivity test failed"
            has_issues=true
        else
            log::debug "✓ HTTP connectivity working"
        fi
    else
        log::debug "No HTTP client available for connectivity test"
    fi
    
    # Check Docker Hub connectivity (important for setup)
    log::debug "Testing Docker Hub connectivity..."
    if command -v curl >/dev/null 2>&1; then
        if ! curl -sS --connect-timeout 5 -o /dev/null https://hub.docker.com 2>/dev/null; then
            log::warning "Cannot reach Docker Hub - Docker pulls may fail"
            has_issues=true
        else
            log::debug "✓ Docker Hub reachable"
        fi
    fi
    
    # Check GitHub connectivity (for git operations)
    log::debug "Testing GitHub connectivity..."
    if command -v curl >/dev/null 2>&1; then
        if ! curl -sS --connect-timeout 5 -o /dev/null https://api.github.com 2>/dev/null; then
            log::warning "Cannot reach GitHub - git operations may fail"
            has_issues=true
        else
            log::debug "✓ GitHub reachable"
        fi
    fi
    
    # Summary
    if [[ "$has_issues" == "true" ]]; then
        log::warning "Some network tests failed - setup may encounter issues"
        return 1
    else
        log::success "✓ Network connectivity verified"
        return 0
    fi
}

# Export function
export -f network_diagnostics::run