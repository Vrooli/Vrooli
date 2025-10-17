#!/usr/bin/env bash
################################################################################
# Gemini Resource Cache Tests
# 
# Redis integration and caching functionality testing
################################################################################

set -euo pipefail

# Determine paths
PHASES_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(cd "$PHASES_DIR/.." && pwd)"
RESOURCE_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="$(cd "$RESOURCE_DIR/../.." && pwd)"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"

# Source resource libraries
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/cache.sh"

log::info "Starting Gemini cache tests..."

# Test 1: Check if Redis is available
log::info "Test 1: Checking Redis availability..."
if gemini::cache::is_available; then
    log::success "✓ Redis is available for caching"
    redis_available=true
else
    log::warn "⚠ Redis not available - skipping cache tests"
    log::info "  To enable caching, start Redis with: vrooli resource redis develop"
    redis_available=false
fi

if [[ "$redis_available" == "true" ]]; then
    # Test 2: Cache key generation
    log::info "Test 2: Testing cache key generation..."
    test_prompt="What is the meaning of life?"
    test_model="gemini-pro"
    test_temp="0.7"
    
    cache_key=$(gemini::cache::generate_key "$test_prompt" "$test_model" "$test_temp")
    
    if [[ "$cache_key" =~ ^gemini:cache:gemini-pro:[a-f0-9]{16}$ ]]; then
        log::success "✓ Cache key generation works: $cache_key"
    else
        log::error "✗ Invalid cache key format: $cache_key"
        exit 1
    fi
    
    # Test 3: Cache set and get
    log::info "Test 3: Testing cache set and get..."
    test_key="gemini:cache:test:$(date +%s)"
    test_value="This is a test response from Gemini"
    
    # Set cache value
    if gemini::cache::set "$test_key" "$test_value" 10; then
        log::success "✓ Cache set successful"
    else
        log::error "✗ Failed to set cache value"
        exit 1
    fi
    
    # Get cache value
    retrieved_value=$(gemini::cache::get "$test_key")
    if [[ "$retrieved_value" == "$test_value" ]]; then
        log::success "✓ Cache get successful - value matches"
    else
        log::error "✗ Cache get failed or value mismatch"
        log::error "  Expected: $test_value"
        log::error "  Got: $retrieved_value"
        exit 1
    fi
    
    # Test 4: Cache deletion
    log::info "Test 4: Testing cache deletion..."
    if gemini::cache::delete "$test_key"; then
        log::success "✓ Cache delete successful"
    else
        log::error "✗ Failed to delete cache entry"
        exit 1
    fi
    
    # Verify deletion
    deleted_value=$(gemini::cache::get "$test_key" 2>/dev/null || echo "")
    if [[ -z "$deleted_value" ]]; then
        log::success "✓ Cache entry properly deleted"
    else
        log::error "✗ Cache entry still exists after deletion"
        exit 1
    fi
    
    # Test 5: Cache TTL expiration
    log::info "Test 5: Testing cache TTL expiration..."
    ttl_key="gemini:cache:ttl:test"
    ttl_value="This should expire"
    
    # Set with 2 second TTL
    gemini::cache::set "$ttl_key" "$ttl_value" 2
    
    # Value should exist immediately
    if [[ -n "$(gemini::cache::get "$ttl_key" 2>/dev/null)" ]]; then
        log::success "✓ Cache value exists before TTL"
    else
        log::error "✗ Cache value missing immediately after set"
        exit 1
    fi
    
    # Wait for expiration
    sleep 3
    
    # Value should be gone
    if [[ -z "$(gemini::cache::get "$ttl_key" 2>/dev/null)" ]]; then
        log::success "✓ Cache value expired correctly"
    else
        log::error "✗ Cache value persisted beyond TTL"
        exit 1
    fi
    
    # Test 6: Cache statistics
    log::info "Test 6: Testing cache statistics..."
    if output=$(gemini::cache::stats 2>&1); then
        if echo "$output" | grep -q "Cache Status: Enabled"; then
            log::success "✓ Cache statistics working"
            echo "$output" | while IFS= read -r line; do
                log::info "  $line"
            done
        else
            log::error "✗ Unexpected cache statistics output"
            exit 1
        fi
    else
        log::error "✗ Failed to get cache statistics"
        exit 1
    fi
    
    # Test 7: Cache clear all
    log::info "Test 7: Testing cache clear all..."
    
    # Add some test entries
    gemini::cache::set "gemini:cache:clear:test1" "value1" 60
    gemini::cache::set "gemini:cache:clear:test2" "value2" 60
    
    # Clear all
    if gemini::cache::clear_all; then
        log::success "✓ Cache clear all executed"
    else
        log::error "✗ Failed to clear cache"
        exit 1
    fi
    
    # Verify entries are gone
    if [[ -z "$(gemini::cache::get "gemini:cache:clear:test1" 2>/dev/null)" ]] && \
       [[ -z "$(gemini::cache::get "gemini:cache:clear:test2" 2>/dev/null)" ]]; then
        log::success "✓ All cache entries cleared"
    else
        log::error "✗ Some cache entries remain after clear"
        exit 1
    fi
fi

log::success "✅ All cache tests passed!"
exit 0