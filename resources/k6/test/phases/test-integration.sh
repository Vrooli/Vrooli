#!/usr/bin/env bash
# K6 Resource Integration Test - Full functionality validation
# Tests end-to-end K6 functionality including script execution and result handling

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
source "${K6_CLI_DIR}/lib/content.sh"
# shellcheck disable=SC1091
source "${K6_CLI_DIR}/lib/core.sh"

# K6 Resource Integration Test
k6::test::integration() {
    log::info "Running K6 resource integration test..."
    
    local overall_status=0
    local test_script="/tmp/k6-integration-test.js"
    local test_script_name="integration-test.js"
    
    # Create test script
    cat > "$test_script" << 'EOF'
// K6 Integration Test Script
import http from 'k6/http';
import { check } from 'k6';

export const options = {
    vus: 1,
    duration: '5s',
    thresholds: {
        http_req_duration: ['p(95)<1000'],
    },
};

export default function () {
    // Test against httpbin.org (reliable test endpoint)
    const response = http.get('https://httpbin.org/get');
    check(response, {
        'status is 200': (r) => r.status === 200,
        'response has json': (r) => r.headers['Content-Type'].includes('application/json'),
    });
}
EOF

    # Test 1: Content addition
    log::info "1/5 Testing content addition..."
    if k6::content::add --file "$test_script" --name "$test_script_name" >/dev/null 2>&1; then
        log::success "✓ Can add test scripts to K6"
    else
        log::error "✗ Cannot add test scripts to K6"
        overall_status=1
    fi
    
    # Test 2: Content listing
    log::info "2/5 Testing content listing..."
    if k6::content::list | grep -q "$test_script_name"; then
        log::success "✓ Can list K6 content"
    else
        log::error "✗ Cannot list K6 content"
        overall_status=1
    fi
    
    # Test 3: Content retrieval
    log::info "3/5 Testing content retrieval..."
    if k6::content::get --name "$test_script_name" >/dev/null 2>&1; then
        log::success "✓ Can retrieve K6 content"
    else
        log::error "✗ Cannot retrieve K6 content"
        overall_status=1
    fi
    
    # Test 4: Script execution (performance test)
    log::info "4/5 Testing script execution..."
    if k6::content::execute --name "$test_script_name" >/dev/null 2>&1; then
        log::success "✓ Can execute K6 performance tests"
    else
        log::error "✗ Cannot execute K6 performance tests"
        overall_status=1
    fi
    
    # Test 5: Results handling
    log::info "5/5 Testing results handling..."
    if k6::content::show_results 1 | grep -q "Recent K6 performance test results"; then
        log::success "✓ Can handle test results"
    else
        log::error "✗ Cannot handle test results"
        overall_status=1
    fi
    
    # Cleanup
    log::info "Cleaning up test artifacts..."
    k6::content::remove --name "$test_script_name" >/dev/null 2>&1 || true
    rm -f "$test_script"
    
    echo ""
    if [[ $overall_status -eq 0 ]]; then
        log::success "🎉 K6 resource integration test PASSED"
        echo "K6 service fully functional - content management and performance testing work perfectly"
    else
        log::error "💥 K6 resource integration test FAILED"  
        echo "K6 service has functional issues that need to be resolved"
    fi
    
    return $overall_status
}

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    k6::test::integration "$@"
fi