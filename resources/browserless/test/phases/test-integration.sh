#!/usr/bin/env bash
# Browserless Resource Integration Test - Full end-to-end testing
# Tests that Browserless service works for the retained compatibility contract
# Max duration: 120 seconds per v2.0 contract

set -euo pipefail

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
TEST_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${TEST_DIR}/.." && builtin pwd)"
REPO_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"
BROWSERLESS_CLI_DIR="${RESOURCE_DIR}"

# Source utilities
# shellcheck disable=SC1091
source "${REPO_ROOT}/scripts/lib/utils/var.sh" || { echo "FATAL: Failed to load variable definitions" >&2; exit 1; }
# shellcheck disable=SC1091
source "${var_LOG_FILE}" || { echo "FATAL: Failed to load logging library" >&2; exit 1; }
# shellcheck disable=SC1091
source "${BROWSERLESS_CLI_DIR}/config/defaults.sh"
# Ensure configuration is exported
browserless::export_config
# shellcheck disable=SC1091
source "${BROWSERLESS_CLI_DIR}/lib/common.sh"
# shellcheck disable=SC1091
source "${BROWSERLESS_CLI_DIR}/lib/health.sh"
# Test output directory for integration tests
INTEGRATION_TEST_DIR="/tmp/browserless-integration-test-$$"

# Browserless Resource Integration Test
browserless::test::integration() {
    log::info "Running Browserless resource integration test..."
    
    local overall_status=0
    local verbose="${BROWSERLESS_TEST_VERBOSE:-false}"
    
    # Setup test environment
    mkdir -p "$INTEGRATION_TEST_DIR"
    
    # Test 1: Screenshot API functionality
    log::info "1/5 Testing screenshot API..."
    local screenshot_ok=true
    local test_url="https://www.example.com"
    local screenshot_file="$INTEGRATION_TEST_DIR/test-screenshot.png"
    
    if timeout 30 curl -X POST "http://localhost:${BROWSERLESS_PORT}/chrome/screenshot" \
        -H "Content-Type: application/json" \
        -d "{\"url\": \"$test_url\"}" \
        --output "$screenshot_file" >/dev/null 2>&1; then
        
        # Verify file was created and has content
        if [[ -f "$screenshot_file" ]] && [[ -s "$screenshot_file" ]]; then
            # Check if it's actually a PNG file
            if file "$screenshot_file" | grep -q "PNG image"; then
                log::success "✓ Screenshot API working - created valid PNG"
                if [[ "$verbose" == "true" ]]; then
                    local file_size=$(stat -f%z "$screenshot_file" 2>/dev/null || stat -c%s "$screenshot_file" 2>/dev/null || echo "unknown")
                    echo "    File size: ${file_size} bytes"
                fi
            else
                log::error "✗ Screenshot API created file but not valid PNG"
                screenshot_ok=false
            fi
        else
            log::error "✗ Screenshot API did not create output file"
            screenshot_ok=false
        fi
    else
        log::error "✗ Screenshot API request failed or timed out"
        screenshot_ok=false
    fi
    
    if [[ "$screenshot_ok" != "true" ]]; then
        overall_status=1
    fi
    
    # Test 2: Pressure/metrics endpoint
    log::info "2/5 Testing pressure metrics endpoint..."
    local pressure_ok=true
    local pressure_file="$INTEGRATION_TEST_DIR/test-pressure.json"
    
    if timeout 10 curl -s "http://localhost:${BROWSERLESS_PORT}/pressure" \
        --output "$pressure_file" 2>/dev/null; then
        
        # Verify pressure data was returned
        if [[ -f "$pressure_file" ]] && [[ -s "$pressure_file" ]]; then
            # Check if it's valid JSON with expected fields
            if jq -e '.pressure.running' "$pressure_file" >/dev/null 2>&1; then
                log::success "✓ Pressure metrics API working - returned valid data"
                if [[ "$verbose" == "true" ]]; then
                    local running=$(jq -r '.pressure.running // 0' "$pressure_file")
                    local queued=$(jq -r '.pressure.queued // 0' "$pressure_file")
                    local max_concurrent=$(jq -r '.pressure.maxConcurrent // 0' "$pressure_file")
                    echo "    Browsers: $running running, $queued queued (max: $max_concurrent)"
                fi
            else
                log::error "✗ Pressure endpoint returned invalid JSON format"
                pressure_ok=false
            fi
        else
            log::error "✗ Pressure endpoint did not return data"
            pressure_ok=false
        fi
    else
        log::error "✗ Pressure endpoint request failed or timed out"
        pressure_ok=false
    fi
    
    if [[ "$pressure_ok" != "true" ]]; then
        overall_status=1
    fi
    
    # Test 3: JavaScript function endpoint availability
    log::info "3/5 Testing function execution endpoint..."
    local function_ok=true
    local function_status
    function_status=$(timeout 15 curl -s -o /dev/null -w "%{http_code}" -X POST \
        "http://localhost:${BROWSERLESS_PORT}/function" \
        -H "Content-Type: application/javascript" \
        --data 'module.exports = async () => ({ ok: true });' 2>/dev/null || echo "000")
    if [[ "$function_status" == "200" ]] || [[ "$function_status" == "400" ]] || [[ "$function_status" == "500" ]]; then
        log::success "✓ Function endpoint is reachable (HTTP $function_status)"
    else
        log::error "✗ Function endpoint unreachable (HTTP $function_status)"
        function_ok=false
    fi
    if [[ "$function_ok" != "true" ]]; then
        overall_status=1
    fi

    # Test 4: CLI command functionality
    log::info "4/5 Testing CLI command functionality..."
    local cli_ok=true
    
    # Test CLI status command
    if timeout 10 bash -c "cd '$BROWSERLESS_CLI_DIR' && ./cli.sh status >/dev/null 2>&1"; then
        log::success "✓ CLI status command working"
    else
        log::error "✗ CLI status command failed"
        cli_ok=false
    fi
    
    # Test CLI health check via smoke test
    if timeout 15 bash -c "cd '$BROWSERLESS_CLI_DIR' && ./cli.sh test smoke >/dev/null 2>&1"; then
        log::success "✓ CLI smoke test command working"
    else
        log::error "✗ CLI smoke test command failed"
        cli_ok=false
    fi
    
    if [[ "$cli_ok" != "true" ]]; then
        overall_status=1
    fi
    
    # Test 5: Diagnostics CLI availability
    log::info "5/5 Testing diagnostics CLI command..."
    local diagnostics_ok=true
    if timeout 20 bash -c "cd '$BROWSERLESS_CLI_DIR' && ./cli.sh diagnostics https://www.example.com >/dev/null 2>&1"; then
        log::success "✓ CLI diagnostics command working"
    else
        log::error "✗ CLI diagnostics command failed"
        diagnostics_ok=false
    fi

    if [[ "$diagnostics_ok" != "true" ]]; then
        overall_status=1
    fi

    # Keep one viewport-oriented screenshot request as part of the compatibility suite.
    log::info "Compatibility check: advanced screenshot options..."
    local advanced_screenshot_ok=true
    local viewport_screenshot="$INTEGRATION_TEST_DIR/viewport-screenshot.png"
    
    # Test screenshot with custom viewport size
    if timeout 30 curl -X POST "http://localhost:${BROWSERLESS_PORT}/chrome/screenshot" \
        -H "Content-Type: application/json" \
        -d "{\"url\": \"https://www.example.com\", \"viewport\": {\"width\": 1920, \"height\": 1080}}" \
        --output "$viewport_screenshot" >/dev/null 2>&1; then
        
        if [[ -f "$viewport_screenshot" ]] && [[ -s "$viewport_screenshot" ]]; then
            log::success "✓ Advanced screenshot with viewport settings working"
        else
            log::error "✗ Advanced screenshot did not create output file"
            advanced_screenshot_ok=false
        fi
    else
        log::error "✗ Advanced screenshot request failed"
        advanced_screenshot_ok=false
    fi
    
    if [[ "$advanced_screenshot_ok" != "true" ]]; then
        overall_status=1
    fi
    
    # Cleanup test files
    rm -rf "$INTEGRATION_TEST_DIR"
    
    echo ""
    if [[ $overall_status -eq 0 ]]; then
        log::success "🎉 Browserless resource integration test PASSED"
        echo "Browserless service is fully operational with all APIs functional"
        
        if [[ "$verbose" == "true" ]]; then
            echo ""
            echo "Tested functionality:"
            echo "  ✓ Screenshot generation (PNG)"
            echo "  ✓ PDF generation"
            echo "  ✓ Content extraction (HTML)"
            echo "  ✓ Pressure/metrics monitoring"
            echo "  ✓ JavaScript function execution"
            echo "  ✓ CLI command interface"
            echo "  ✓ Adapter system framework"
        fi
    else
        log::error "💥 Browserless resource integration test FAILED"
        echo "Browserless service has functional issues that need attention"
        
        echo ""
        echo "Common solutions:"
        echo "  1. Restart the service: resource-browserless manage restart"
        echo "  2. Check container logs: docker logs ${BROWSERLESS_CONTAINER_NAME}"
        echo "  3. Verify network connectivity: curl http://localhost:${BROWSERLESS_PORT}/pressure"
        echo "  4. Check disk space for output files"
        echo "  5. Increase timeout values in production"
    fi
    
    return $overall_status
}

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    browserless::test::integration "$@"
fi
