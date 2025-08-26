#!/usr/bin/env bash
# K6 Resource Smoke Test - Quick health validation
# Tests that K6 service is running and responsive

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
K6_CLI_DIR="${APP_ROOT}/resources/k6"

# Source utilities
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/log.sh"
# shellcheck disable=SC1091
source "${K6_CLI_DIR}/config/defaults.sh"
# shellcheck disable=SC1091  
source "${K6_CLI_DIR}/lib/docker.sh"
# shellcheck disable=SC1091
source "${K6_CLI_DIR}/lib/core.sh"

# K6 Resource Smoke Test
k6::test::smoke() {
    log::info "Running K6 resource smoke test..."
    
    local overall_status=0
    
    # Test 1: Docker container health
    log::info "1/4 Testing Docker container status..."
    if k6::docker::status >/dev/null 2>&1; then
        log::success "✓ K6 container is running"
    else
        log::error "✗ K6 container is not running"
        overall_status=1
    fi
    
    # Test 2: K6 CLI availability
    log::info "2/4 Testing K6 CLI availability..."
    if docker exec vrooli-k6 k6 --version >/dev/null 2>&1; then
        local version=$(docker exec vrooli-k6 k6 --version 2>/dev/null | head -1)
        log::success "✓ K6 CLI accessible: $version"
    else
        log::error "✗ K6 CLI not accessible in container"
        overall_status=1
    fi
    
    # Test 3: Directories accessible
    log::info "3/4 Testing directory mounts..."
    local dirs_ok=true
    for dir in scripts data results; do
        if docker exec vrooli-k6 test -d "/$dir" 2>/dev/null; then
            log::success "✓ /$dir directory mounted"
        else
            log::error "✗ /$dir directory not accessible"
            dirs_ok=false
        fi
    done
    if [[ "$dirs_ok" != "true" ]]; then
        overall_status=1
    fi
    
    # Test 4: Basic K6 execution capability
    log::info "4/4 Testing K6 execution capability..."
    if docker exec vrooli-k6 sh -c 'echo "export default function() { console.log(\"smoke test\"); }" | k6 run -' >/dev/null 2>&1; then
        log::success "✓ K6 can execute JavaScript tests"
    else
        log::error "✗ K6 cannot execute basic JavaScript"
        overall_status=1
    fi
    
    echo ""
    if [[ $overall_status -eq 0 ]]; then
        log::success "🎉 K6 resource smoke test PASSED"
        echo "K6 service is healthy and ready for performance testing"
    else
        log::error "💥 K6 resource smoke test FAILED"
        echo "K6 service has issues that need to be resolved"
    fi
    
    return $overall_status
}

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    k6::test::smoke "$@"
fi