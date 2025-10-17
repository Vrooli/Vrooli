#!/usr/bin/env bash
# Browserless Resource Smoke Test - Quick health validation
# Tests that Browserless service is running and responsive
# Max duration: 30 seconds per v2.0 contract

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
BROWSERLESS_CLI_DIR="${APP_ROOT}/resources/browserless"

# Source utilities
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh" || { echo "FATAL: Failed to load variable definitions" >&2; exit 1; }
# shellcheck disable=SC1091
source "${var_LOG_FILE}" || { echo "FATAL: Failed to load logging library" >&2; exit 1; }
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/common.sh"
# shellcheck disable=SC1091
source "${BROWSERLESS_CLI_DIR}/config/defaults.sh"
# Ensure configuration is exported
browserless::export_config
# shellcheck disable=SC1091
source "${BROWSERLESS_CLI_DIR}/lib/common.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/docker-utils.sh"
# shellcheck disable=SC1091
source "${BROWSERLESS_CLI_DIR}/lib/health.sh"

# Browserless Resource Smoke Test
browserless::test::smoke() {
    log::info "Running Browserless resource smoke test..."
    
    local overall_status=0
    local verbose="${BROWSERLESS_TEST_VERBOSE:-false}"
    
    # Test 1: Docker container health
    log::info "1/5 Testing Docker container status..."
    if docker ps --format "{{.Names}}" 2>/dev/null | grep -q "^${BROWSERLESS_CONTAINER_NAME}$"; then
        log::success "✓ Browserless container is running"
    else
        log::error "✗ Browserless container is not running"
        overall_status=1
    fi
    
    # Test 2: Port accessibility
    log::info "2/5 Testing port accessibility..."
    if timeout 5 bash -c "echo >/dev/tcp/localhost/${BROWSERLESS_PORT}" 2>/dev/null; then
        log::success "✓ Port ${BROWSERLESS_PORT} is accessible"
    else
        log::error "✗ Port ${BROWSERLESS_PORT} is not accessible"
        overall_status=1
    fi
    
    # Test 3: Health endpoint response
    log::info "3/5 Testing health endpoint..."
    if timeout 10 curl -s "http://localhost:${BROWSERLESS_PORT}/pressure" >/dev/null 2>&1; then
        log::success "✓ Health endpoint responding"
    else
        log::error "✗ Health endpoint not responding"
        overall_status=1
    fi
    
    # Test 4: Basic health check function
    log::info "4/5 Testing health check function..."
    if browserless::is_healthy; then
        log::success "✓ Health check function reports healthy"
    else
        log::error "✗ Health check function reports unhealthy"
        overall_status=1
    fi
    
    # Test 5: Container resource usage check
    log::info "5/5 Testing container resource usage..."
    local container_stats
    if container_stats=$(docker stats --no-stream --format "table {{.CPUPerc}}\t{{.MemUsage}}" "${BROWSERLESS_CONTAINER_NAME}" 2>/dev/null); then
        log::success "✓ Container resource monitoring accessible"
        if [[ "$verbose" == "true" ]]; then
            echo "    Resources: $(echo "$container_stats" | tail -1)"
        fi
    else
        log::error "✗ Cannot access container resource information"
        overall_status=1
    fi
    
    echo ""
    if [[ $overall_status -eq 0 ]]; then
        log::success "🎉 Browserless resource smoke test PASSED"
        echo "Browserless service is healthy and ready for browser automation"
        
        # Show basic service info if verbose
        if [[ "$verbose" == "true" ]]; then
            echo ""
            echo "Service Details:"
            echo "  Container: ${BROWSERLESS_CONTAINER_NAME}"
            echo "  Port: ${BROWSERLESS_PORT}"
            echo "  Base URL: ${BROWSERLESS_BASE_URL}"
            echo "  Image: ${BROWSERLESS_IMAGE}"
        fi
    else
        log::error "💥 Browserless resource smoke test FAILED"
        echo "Browserless service has issues that need to be resolved"
        
        # Provide troubleshooting hints
        echo ""
        echo "Troubleshooting hints:"
        echo "  1. Check if Docker is running: docker info"
        echo "  2. Check container logs: docker logs ${BROWSERLESS_CONTAINER_NAME}"
        echo "  3. Restart service: resource-browserless manage restart"
        echo "  4. Check port conflicts: sudo lsof -i :${BROWSERLESS_PORT}"
    fi
    
    return $overall_status
}

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    browserless::test::smoke "$@"
fi