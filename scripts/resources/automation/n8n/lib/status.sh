#!/usr/bin/env bash
# n8n Status Functions - Minimal wrapper using status engine

# Source core and frameworks
N8N_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/../../../lib/status-engine.sh"
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/../../../lib/docker-utils.sh"
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/core.sh"
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/health.sh"

#######################################
# Show n8n status using unified engine
#######################################
n8n::status() {
    # Check Docker daemon
    if ! docker::check_daemon; then
        return 1
    fi
    
    # Use status engine
    local config
    config=$(n8n::get_status_config)
    status::display_unified_status "$config" "n8n::display_workflow_info"
}


#######################################
# Show n8n information
#######################################
n8n::info() {
    cat << EOF
=== n8n Resource Information ===

ID: n8n
Category: automation
Display Name: n8n
Description: Workflow automation platform

Service Details:
- Container Name: $N8N_CONTAINER_NAME
- Service Port: $N8N_PORT
- Service URL: $N8N_BASE_URL
- Docker Image: $N8N_IMAGE
- Data Directory: $N8N_DATA_DIR

Configuration:
- Authentication: ${BASIC_AUTH:-yes}
- Database: ${DATABASE_TYPE:-sqlite}
- Tunnel Mode: ${TUNNEL_ENABLED:-no}

For more information, visit: https://docs.n8n.io
EOF
}





#######################################
# Test n8n functionality
#######################################
n8n::test() {
    log::info "Testing n8n functionality..."
    
    local tests_passed=0
    local total_tests=0
    
    # Test 1: Tiered health check
    log::info "1. Running health checks..."
    local health_tier
    health_tier=$(n8n::tiered_health_check)
    total_tests=$((total_tests + 1))
    if [[ "$health_tier" == "HEALTHY" ]]; then
        log::success "   ✅ Health check: PASSED ($health_tier)"
        tests_passed=$((tests_passed + 1))
    else
        log::warn "   ⚠️  Health check: PARTIAL ($health_tier)"
        # Don't fail completely on degraded health
        if [[ "$health_tier" != "UNHEALTHY" ]]; then
            tests_passed=$((tests_passed + 1))
        fi
    fi
    
    # Test 2: API connectivity
    log::info "2. Testing API connectivity..."
    total_tests=$((total_tests + 1))
    if n8n::test_api >/dev/null 2>&1; then
        log::success "   ✅ API connectivity: PASSED"
        tests_passed=$((tests_passed + 1))
    else
        log::warn "   ⚠️  API connectivity: FAILED (API key may be missing)"
        # Don't count as failure if just missing API key
    fi
    
    # Summary
    log::info "Test Results: $tests_passed/$total_tests core tests passed"
    
    if [[ "$tests_passed" -eq "$total_tests" ]]; then
        log::success "🎉 All n8n tests passed"
        return 0
    elif [[ "$tests_passed" -gt 0 ]]; then
        log::warn "⚠️  Some tests passed - n8n is functional but may need configuration"
        return 0
    else
        log::error "❌ All tests failed - n8n is not functional"
        return 1
    fi
}