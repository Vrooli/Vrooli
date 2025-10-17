#!/usr/bin/env bash
# SearXNG Resource Smoke Test - Quick health validation
# Tests that SearXNG service is running and responsive
# Max duration: 30 seconds per v2.0 contract

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

# SearXNG Resource Smoke Test
searxng::test::smoke() {
    log::info "Running SearXNG resource smoke test..."
    
    local overall_status=0
    local verbose="${SEARXNG_TEST_VERBOSE:-false}"
    
    # Test 1: Docker container exists
    log::info "1/6 Testing Docker container existence..."
    if searxng::is_installed; then
        log::success "✓ SearXNG container exists"
        if [[ "$verbose" == "true" ]]; then
            local container_image
            container_image=$(docker inspect --format='{{.Config.Image}}' "$SEARXNG_CONTAINER_NAME" 2>/dev/null || echo "unknown")
            log::info "  Image: $container_image"
        fi
    else
        log::error "✗ SearXNG container not found"
        overall_status=1
    fi
    
    # Test 2: Container is running
    log::info "2/6 Testing container status..."
    if searxng::is_running; then
        log::success "✓ SearXNG container is running"
        if [[ "$verbose" == "true" ]]; then
            local container_status
            container_status=$(docker inspect --format='{{.State.Status}}' "$SEARXNG_CONTAINER_NAME" 2>/dev/null || echo "unknown")
            log::info "  Status: $container_status"
        fi
    else
        log::error "✗ SearXNG container is not running"
        overall_status=1
    fi
    
    # Test 3: Port accessibility
    log::info "3/6 Testing port accessibility..."
    if nc -z localhost "${SEARXNG_PORT}" 2>/dev/null; then
        log::success "✓ Port ${SEARXNG_PORT} is accessible"
    else
        log::error "✗ Port ${SEARXNG_PORT} is not accessible"
        overall_status=1
    fi
    
    # Test 4: Web interface accessibility
    log::info "4/6 Testing web interface..."
    local web_response
    if web_response=$(curl -sf --max-time 10 "${SEARXNG_BASE_URL}" 2>/dev/null); then
        if echo "$web_response" | grep -qi "searxng\|search\|<!DOCTYPE html>"; then
            log::success "✓ Web interface accessible"
            if [[ "$verbose" == "true" ]]; then
                local title
                title=$(echo "$web_response" | grep -i "<title>" | sed 's/<[^>]*>//g' | xargs || echo "N/A")
                log::info "  Title: $title"
            fi
        else
            log::warn "⚠ Web interface responding but content unclear"
        fi
    else
        log::error "✗ Web interface not accessible"
        overall_status=1
    fi
    
    # Test 5: Stats endpoint
    log::info "5/6 Testing stats endpoint..."
    local stats_response
    if stats_response=$(curl -sf --max-time 5 "${SEARXNG_BASE_URL}/stats" 2>/dev/null); then
        if echo "$stats_response" | jq . >/dev/null 2>&1; then
            log::success "✓ Stats endpoint responding with JSON"
        elif echo "$stats_response" | grep -qi "searxng\|statistics\|engines"; then
            log::success "✓ Stats endpoint responding with HTML"
        else
            log::warn "⚠ Stats endpoint responding but format unclear"
        fi
    else
        log::error "✗ Stats endpoint not responding"
        overall_status=1
    fi
    
    # Test 6: Basic search API
    log::info "6/6 Testing basic search API..."
    local search_response
    if search_response=$(curl -sf --max-time 10 "${SEARXNG_BASE_URL}/search?q=test&format=json" 2>/dev/null); then
        if echo "$search_response" | jq . >/dev/null 2>&1; then
            local result_count
            result_count=$(echo "$search_response" | jq '.results | length' 2>/dev/null || echo "0")
            if [[ $result_count -gt 0 ]]; then
                log::success "✓ Search API working (results: $result_count)"
            else
                log::warn "⚠ Search API responding but no results returned"
            fi
            
            if [[ "$verbose" == "true" ]]; then
                local query_time
                query_time=$(echo "$search_response" | jq -r '.query // "unknown"' 2>/dev/null)
                log::info "  Query: $query_time"
            fi
        else
            log::warn "⚠ Search API responding but not valid JSON"
        fi
    else
        log::error "✗ Search API not responding"
        overall_status=1
    fi
    
    echo ""
    if [[ $overall_status -eq 0 ]]; then
        log::success "🎉 SearXNG resource smoke test PASSED"
        echo "SearXNG service is healthy and ready for search operations"
    else
        log::error "💥 SearXNG resource smoke test FAILED"
        echo "SearXNG service has issues that need to be resolved"
    fi
    
    return $overall_status
}

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    searxng::test::smoke "$@"
fi