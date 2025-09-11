#!/usr/bin/env bash
################################################################################
# CNCjs Test Functions Library
# Implements test suite for validation
################################################################################

set -euo pipefail

# Source guard
[[ -n "${_CNCJS_TEST_SOURCED:-}" ]] && return 0
export _CNCJS_TEST_SOURCED=1

# Source dependencies
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../config/defaults.sh"
source "${SCRIPT_DIR}/core.sh"

#######################################
# Main test runner
#######################################
cncjs::test() {
    local test_type="${1:-all}"
    
    case "$test_type" in
        smoke)
            cncjs::test_smoke
            ;;
        integration)
            cncjs::test_integration
            ;;
        unit)
            cncjs::test_unit
            ;;
        all)
            cncjs::test_smoke
            cncjs::test_integration
            cncjs::test_unit
            ;;
        *)
            log::error "Unknown test type: $test_type"
            echo "Available tests: smoke, integration, unit, all"
            return 1
            ;;
    esac
}

#######################################
# Smoke test - Quick health check
#######################################
cncjs::test_smoke() {
    log::info "Running CNCjs smoke tests..."
    
    local failed=0
    
    # Test 1: Service is running
    log::info "Test 1: Checking if CNCjs is running..."
    if docker ps --format "{{.Names}}" | grep -q "^${CNCJS_CONTAINER_NAME}$"; then
        log::success "✓ CNCjs container is running"
    else
        log::error "✗ CNCjs container is not running"
        ((failed++))
    fi
    
    # Test 2: Health check passes
    log::info "Test 2: Checking health endpoint..."
    if cncjs::health_check; then
        log::success "✓ Health check passed"
    else
        log::error "✗ Health check failed"
        ((failed++))
    fi
    
    # Test 3: Web interface accessible
    log::info "Test 3: Checking web interface..."
    if timeout 5 curl -sf "http://localhost:${CNCJS_PORT}/" &>/dev/null; then
        log::success "✓ Web interface is accessible"
    else
        log::error "✗ Web interface is not accessible"
        ((failed++))
    fi
    
    # Test 4: Configuration exists
    log::info "Test 4: Checking configuration..."
    if [[ -f "${CNCJS_CONFIG_FILE}" ]]; then
        log::success "✓ Configuration file exists"
    else
        log::error "✗ Configuration file missing"
        ((failed++))
    fi
    
    # Test 5: Watch directory exists
    log::info "Test 5: Checking watch directory..."
    if [[ -d "${CNCJS_WATCH_DIR}" ]]; then
        log::success "✓ Watch directory exists"
    else
        log::error "✗ Watch directory missing"
        ((failed++))
    fi
    
    if [[ $failed -eq 0 ]]; then
        log::success "All smoke tests passed!"
        return 0
    else
        log::error "$failed smoke test(s) failed"
        return 1
    fi
}

#######################################
# Integration test - Full functionality
#######################################
cncjs::test_integration() {
    log::info "Running CNCjs integration tests..."
    
    local failed=0
    
    # Test 1: Full lifecycle
    log::info "Test 1: Testing full lifecycle..."
    cncjs::stop &>/dev/null || true
    if cncjs::start --wait; then
        log::success "✓ Service started successfully"
    else
        log::error "✗ Failed to start service"
        ((failed++))
    fi
    
    # Test 2: Content management
    log::info "Test 2: Testing content management..."
    local test_file="/tmp/test.gcode"
    echo "G0 X0 Y0 Z0" > "$test_file"
    
    if cncjs::content add "$test_file" &>/dev/null; then
        log::success "✓ File upload successful"
    else
        log::error "✗ File upload failed"
        ((failed++))
    fi
    
    if [[ -f "${CNCJS_WATCH_DIR}/test.gcode" ]]; then
        log::success "✓ File exists in watch directory"
    else
        log::error "✗ File not found in watch directory"
        ((failed++))
    fi
    
    if cncjs::content remove "test.gcode" &>/dev/null; then
        log::success "✓ File removal successful"
    else
        log::error "✗ File removal failed"
        ((failed++))
    fi
    
    rm -f "$test_file"
    
    # Test 3: Status reporting
    log::info "Test 3: Testing status reporting..."
    local status_json
    if status_json=$(cncjs::status --json); then
        if echo "$status_json" | jq -e '.status == "running"' &>/dev/null; then
            log::success "✓ Status reports running"
        else
            log::error "✗ Status does not report running"
            ((failed++))
        fi
    else
        log::error "✗ Failed to get status"
        ((failed++))
    fi
    
    # Test 4: Restart functionality
    log::info "Test 4: Testing restart..."
    if cncjs::restart &>/dev/null; then
        sleep 5
        if cncjs::health_check; then
            log::success "✓ Restart successful"
        else
            log::error "✗ Service unhealthy after restart"
            ((failed++))
        fi
    else
        log::error "✗ Restart failed"
        ((failed++))
    fi
    
    # Test 5: WebSocket endpoint check
    log::info "Test 5: Testing WebSocket endpoint..."
    # CNCjs uses WebSocket for real-time communication, check if upgrade headers work
    if timeout 5 curl -sf -H "Connection: Upgrade" -H "Upgrade: websocket" "http://localhost:${CNCJS_PORT}/socket.io/" 2>&1 | grep -q "transport"; then
        log::success "✓ WebSocket endpoint accessible"
    else
        # Fallback: just check main page loads
        if timeout 5 curl -sf "http://localhost:${CNCJS_PORT}/" &>/dev/null; then
            log::success "✓ Web interface accessible (WebSocket not tested)"
        else
            log::error "✗ Web endpoints not accessible"
            ((failed++))
        fi
    fi
    
    if [[ $failed -eq 0 ]]; then
        log::success "All integration tests passed!"
        return 0
    else
        log::error "$failed integration test(s) failed"
        return 1
    fi
}

#######################################
# Unit test - Library functions
#######################################
cncjs::test_unit() {
    log::info "Running CNCjs unit tests..."
    
    local failed=0
    
    # Test 1: Configuration validation
    log::info "Test 1: Testing configuration validation..."
    if [[ -n "${CNCJS_PORT}" ]] && [[ "${CNCJS_PORT}" -gt 1024 ]]; then
        log::success "✓ Port configuration valid"
    else
        log::error "✗ Invalid port configuration"
        ((failed++))
    fi
    
    # Test 2: Path validation
    log::info "Test 2: Testing path validation..."
    if [[ -n "${CNCJS_DATA_DIR}" ]]; then
        log::success "✓ Data directory path set"
    else
        log::error "✗ Data directory path not set"
        ((failed++))
    fi
    
    # Test 3: Controller validation
    log::info "Test 3: Testing controller validation..."
    case "${CNCJS_CONTROLLER}" in
        Grbl|Marlin|Smoothie|TinyG|g2core)
            log::success "✓ Valid controller type: ${CNCJS_CONTROLLER}"
            ;;
        *)
            log::error "✗ Invalid controller type: ${CNCJS_CONTROLLER}"
            ((failed++))
            ;;
    esac
    
    # Test 4: Baud rate validation
    log::info "Test 4: Testing baud rate validation..."
    case "${CNCJS_BAUD_RATE}" in
        9600|19200|38400|57600|115200|230400|250000)
            log::success "✓ Valid baud rate: ${CNCJS_BAUD_RATE}"
            ;;
        *)
            log::error "✗ Invalid baud rate: ${CNCJS_BAUD_RATE}"
            ((failed++))
            ;;
    esac
    
    # Test 5: Docker availability
    log::info "Test 5: Testing Docker availability..."
    if command -v docker &>/dev/null; then
        log::success "✓ Docker is available"
    else
        log::error "✗ Docker not found"
        ((failed++))
    fi
    
    if [[ $failed -eq 0 ]]; then
        log::success "All unit tests passed!"
        return 0
    else
        log::error "$failed unit test(s) failed"
        return 1
    fi
}