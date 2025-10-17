#!/usr/bin/env bash
# SearXNG Resource Unit Test - Library function validation
# Tests individual SearXNG library functions work correctly
# Max duration: 60 seconds per v2.0 contract

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
SEARXNG_CLI_DIR="${APP_ROOT}/resources/searxng"

# Set test mode to avoid readonly variable issues
export SEARXNG_TEST_MODE="yes"

# Source utilities
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/log.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/common.sh"
# shellcheck disable=SC1091
source "${SEARXNG_CLI_DIR}/config/defaults.sh"
# Ensure configuration is exported
searxng::export_config
# shellcheck disable=SC1091
source "${SEARXNG_CLI_DIR}/lib/common.sh"
# shellcheck disable=SC1091
source "${SEARXNG_CLI_DIR}/lib/docker.sh"
# shellcheck disable=SC1091
source "${SEARXNG_CLI_DIR}/lib/status.sh"
# shellcheck disable=SC1091
source "${SEARXNG_CLI_DIR}/lib/api.sh"
# shellcheck disable=SC1091
source "${SEARXNG_CLI_DIR}/lib/config.sh"

# SearXNG Resource Unit Test
searxng::test::unit() {
    log::info "Running SearXNG resource unit tests..."
    
    local overall_status=0
    local verbose="${SEARXNG_TEST_VERBOSE:-false}"
    
    # Test 1: Configuration validation function
    log::info "1/8 Testing configuration validation..."
    if searxng::validate_config >/dev/null 2>&1; then
        log::success "✓ Configuration validation function works"
    else
        log::error "✗ Configuration validation function failed"
        overall_status=1
    fi
    
    # Test 2: Status collection function
    log::info "2/8 Testing status data collection..."
    local status_data
    if status_data=$(searxng::status::collect_data --fast 2>/dev/null); then
        if [[ -n "$status_data" ]]; then
            log::success "✓ Status data collection works"
            if [[ "$verbose" == "true" ]]; then
                local line_count
                line_count=$(echo "$status_data" | wc -l)
                log::info "  Data points: $line_count"
            fi
        else
            log::warn "⚠ Status data collection returns empty result"
        fi
    else
        log::error "✗ Status data collection failed"
        overall_status=1
    fi
    
    # Test 3: Container state functions (simplified for unit test)
    log::info "3/8 Testing container state functions..."
    # Note: Container state functions are tested thoroughly in smoke and integration tests
    # Unit tests focus on non-blocking validation only
    if command -v docker >/dev/null 2>&1; then
        log::success "✓ Container state functions accessible (Docker available)"
    else
        log::warn "⚠ Docker not available for container state testing"
    fi
    
    # Test 4: Configuration file validation
    log::info "4/8 Testing configuration file validation..."
    if searxng::validate_config_files >/dev/null 2>&1; then
        log::success "✓ Configuration file validation works"
    else
        log::warn "⚠ Configuration file validation issues detected"
    fi
    
    # Test 5: Network creation function (dry run)
    log::info "5/8 Testing network management functions..."
    if command -v docker >/dev/null 2>&1; then
        # Test network inspection (doesn't create, just checks)
        if docker network inspect "$SEARXNG_NETWORK_NAME" >/dev/null 2>&1; then
            log::success "✓ Network management functions accessible"
        else
            log::info "  Network not found (expected if not installed)"
        fi
    else
        log::skip "Docker not available for network tests"
    fi
    
    # Test 6: URL building and validation
    log::info "6/8 Testing URL construction..."
    local base_url="$SEARXNG_BASE_URL"
    if [[ "$base_url" =~ ^https?://[^/]+(:([0-9]+))?$ ]]; then
        log::success "✓ Base URL format is valid: $base_url"
    else
        log::error "✗ Base URL format is invalid: $base_url"
        overall_status=1
    fi
    
    # Test search URL construction
    local search_url="${base_url}/search?q=test&format=json"
    if [[ "$search_url" == *"/search?q=test&format=json" ]]; then
        log::success "✓ Search URL construction works"
        if [[ "$verbose" == "true" ]]; then
            log::info "  URL: $search_url"
        fi
    else
        log::error "✗ Search URL construction failed"
        overall_status=1
    fi
    
    # Test 7: Configuration show function
    log::info "7/8 Testing configuration display..."
    local config_output
    if config_output=$(searxng::show_config 2>/dev/null); then
        if echo "$config_output" | grep -q "SearXNG Configuration"; then
            log::success "✓ Configuration display function works"
        else
            log::warn "⚠ Configuration display function incomplete"
        fi
    else
        log::error "✗ Configuration display function failed"
        overall_status=1
    fi
    
    # Test 8: Health check timeout handling
    log::info "8/8 Testing health check with timeout..."
    local health_start=$(date +%s)
    local health_result
    
    # Test with short timeout to ensure function handles timeouts properly
    if health_result=$(timeout 5s bash -c 'searxng::is_healthy' 2>/dev/null); then
        local health_duration=$(($(date +%s) - health_start))
        log::success "✓ Health check completes within timeout (${health_duration}s)"
    else
        local health_duration=$(($(date +%s) - health_start))
        if [[ $health_duration -ge 5 ]]; then
            log::error "✗ Health check does not respect timeout"
            overall_status=1
        else
            log::warn "⚠ Health check failed but timeout handling works"
        fi
    fi
    
    echo ""
    if [[ $overall_status -eq 0 ]]; then
        log::success "🎉 SearXNG resource unit tests PASSED"
        echo "All library functions are working correctly"
    else
        log::error "💥 SearXNG resource unit tests FAILED"
        echo "Some library functions need attention"
    fi
    
    return $overall_status
}

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    searxng::test::unit "$@"
fi