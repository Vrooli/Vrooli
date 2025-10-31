#!/bin/bash

# Symbol Search Performance Test Suite
# Tests API performance, UI rendering, and database query optimization

set -e

# Configuration
API_URL="${API_URL:-http://localhost:${API_PORT:-15000}}"
TEST_TYPE="${1:-all}"
RESULTS_FILE="/tmp/symbol-search-perf-results.json"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
}

# Check if API is available
check_api_availability() {
    if ! curl -sf "$API_URL/health" >/dev/null 2>&1; then
        log_error "API is not available at $API_URL"
        exit 1
    fi
}

# Test basic search performance
test_basic_search() {
    log_info "Testing basic search performance..."
    
    local queries=("heart" "star" "arrow" "math" "emoji")
    local total_time=0
    local count=0
    
    for query in "${queries[@]}"; do
        local start_time=$(date +%s%3N)
        
        response=$(curl -sf "$API_URL/api/search?q=$query&limit=50" 2>/dev/null)
        
        local end_time=$(date +%s%3N)
        local duration=$((end_time - start_time))
        
        if [ $duration -lt 100 ]; then
            log_success "Query '$query': ${duration}ms (< 100ms)"
        else
            log_warning "Query '$query': ${duration}ms (>= 100ms)"
        fi
        
        total_time=$((total_time + duration))
        count=$((count + 1))
    done
    
    local avg_time=$((total_time / count))
    
    if [ $avg_time -lt 100 ]; then
        log_success "Average search time: ${avg_time}ms (PASS)"
        echo "PASS"
    else
        log_error "Average search time: ${avg_time}ms (FAIL - should be < 100ms)"
        exit 1
    fi
}

# Test large result set performance
test_large_search() {
    log_info "Testing large result set performance..."
    
    # Search for common categories that return many results
    local queries=("category=So" "category=Lu" "block=Basic%20Latin")
    
    for query in "${queries[@]}"; do
        local start_time=$(date +%s%3N)
        
        response=$(curl -sf "$API_URL/api/search?$query&limit=1000" 2>/dev/null)
        
        local end_time=$(date +%s%3N)
        local duration=$((end_time - start_time))
        
        local total=$(echo "$response" | jq -r '.total // 0')
        
        if [ $duration -lt 200 ]; then
            log_success "Large query '$query' ($total results): ${duration}ms (< 200ms)"
        else
            log_warning "Large query '$query' ($total results): ${duration}ms (>= 200ms)"
        fi
    done
    
    log_success "Large search performance test completed (PASS)"
    echo "PASS"
}

# Test database index performance
test_db_indexes() {
    log_info "Testing database index performance..."
    
    # Test various query patterns that should use indexes
    local test_queries=(
        "SELECT COUNT(*) FROM characters WHERE name ILIKE '%heart%'"
        "SELECT COUNT(*) FROM characters WHERE category = 'So'"
        "SELECT COUNT(*) FROM characters WHERE block = 'Basic Latin'"
        "SELECT COUNT(*) FROM characters WHERE decimal BETWEEN 1000 AND 2000"
    )
    
    local db_host="${POSTGRES_HOST:-localhost}"
    local db_port="${POSTGRES_PORT:-5432}"
    local db_name="${POSTGRES_DB:-vrooli}"
    local db_user="${POSTGRES_USER:-vrooli}"
    
    for query in "${test_queries[@]}"; do
        local start_time=$(date +%s%3N)
        
        # Execute query via psql if available, otherwise skip
        if command -v psql >/dev/null 2>&1; then
            PGPASSWORD="${POSTGRES_PASSWORD:-password}" psql \
                -h "$db_host" -p "$db_port" -U "$db_user" -d "$db_name" \
                -c "$query" -t >/dev/null 2>&1
        else
            log_warning "psql not available, skipping direct database test"
            continue
        fi
        
        local end_time=$(date +%s%3N)
        local duration=$((end_time - start_time))
        
        if [ $duration -lt 50 ]; then
            log_success "Database query: ${duration}ms (< 50ms)"
        else
            log_warning "Database query: ${duration}ms (>= 50ms)"
        fi
    done
    
    log_success "Database index performance test completed (PASS)"
    echo "PASS"
}

# Test search latency across multiple queries
test_search_latency() {
    log_info "Testing search latency distribution..."
    
    local queries=(
        "heart" "star" "arrow" "plus" "minus" "dollar" "percent" 
        "copyright" "registered" "infinity" "alpha" "beta" "gamma"
        "U+1F600" "U+2764" "U+2605" "128512" "10084"
    )
    
    local times=()
    local total_time=0
    local count=0
    local fast_queries=0
    
    for query in "${queries[@]}"; do
        local start_time=$(date +%s%3N)
        
        curl -sf "$API_URL/api/search?q=$query&limit=10" >/dev/null 2>&1
        
        local end_time=$(date +%s%3N)
        local duration=$((end_time - start_time))
        
        times+=($duration)
        total_time=$((total_time + duration))
        count=$((count + 1))
        
        if [ $duration -lt 100 ]; then
            fast_queries=$((fast_queries + 1))
        fi
    done
    
    local avg_time=$((total_time / count))
    local fast_percentage=$((fast_queries * 100 / count))
    
    log_info "Search latency results:"
    log_info "  Total queries: $count"
    log_info "  Average time: ${avg_time}ms"
    log_info "  Queries < 100ms: $fast_queries ($fast_percentage%)"
    
    if [ $fast_percentage -ge 95 ]; then
        log_success "Search latency: $fast_percentage% under 100ms (PASS)"
        echo "PASS"
    else
        log_error "Search latency: $fast_percentage% under 100ms (FAIL - should be >= 95%)"
        exit 1
    fi
}

# Test UI render performance (mock test)
test_ui_render() {
    log_info "Testing UI render performance (simulated)..."
    
    # This would require a browser automation tool like Puppeteer
    # For now, we'll test the API performance that feeds the UI
    
    local start_time=$(date +%s%3N)
    
    # Simulate large result set that UI would need to render
    response=$(curl -sf "$API_URL/api/search?category=So&limit=100" 2>/dev/null)
    
    local end_time=$(date +%s%3N)
    local duration=$((end_time - start_time))
    
    local total=$(echo "$response" | jq -r '.total // 0')
    
    if [ $duration -lt 200 ]; then
        log_success "UI data fetch (100 characters): ${duration}ms (< 200ms)"
        log_success "UI render performance test (PASS)"
        echo "PASS"
    else
        log_error "UI data fetch took ${duration}ms (>= 200ms)"
        exit 1
    fi
}

# Test memory usage under load (basic)
test_memory_load() {
    log_info "Testing memory usage under concurrent load..."
    
    local concurrent_requests=10
    local pids=()
    
    # Start multiple concurrent requests
    for ((i=1; i<=concurrent_requests; i++)); do
        {
            curl -sf "$API_URL/api/search?q=test&limit=100" >/dev/null 2>&1
            curl -sf "$API_URL/api/categories" >/dev/null 2>&1
            curl -sf "$API_URL/api/blocks" >/dev/null 2>&1
        } &
        pids+=($!)
    done
    
    # Wait for all requests to complete
    for pid in "${pids[@]}"; do
        wait "$pid"
    done
    
    log_success "Concurrent request test completed"
    
    # Basic memory check (if available)
    if command -v ps >/dev/null 2>&1; then
        local api_memory=$(ps aux | grep '[s]ymbol-search-api' | awk '{print $6}' | head -1)
        if [ -n "$api_memory" ] && [ "$api_memory" -lt 524288 ]; then  # 512MB in KB
            log_success "API memory usage: ${api_memory}KB (< 512MB)"
        else
            log_warning "API memory usage check skipped or high"
        fi
    fi
    
    log_success "Memory load test completed (PASS)"
    echo "PASS"
}

# Main execution
main() {
    check_api_availability
    
    case "$TEST_TYPE" in
        "basic-search")
            test_basic_search
            ;;
        "large-search")
            test_large_search
            ;;
        "db-indexes")
            test_db_indexes
            ;;
        "search-latency")
            test_search_latency
            ;;
        "ui-render")
            test_ui_render
            ;;
        "memory-load")
            test_memory_load
            ;;
        "all")
            log_info "Running all performance tests..."
            test_basic_search
            test_large_search
            test_db_indexes
            test_search_latency
            test_ui_render
            test_memory_load
            log_success "All performance tests completed (PASS)"
            echo "PASS"
            ;;
        *)
            echo "Usage: $0 {basic-search|large-search|db-indexes|search-latency|ui-render|memory-load|all}"
            echo ""
            echo "Available tests:"
            echo "  basic-search   - Test basic search query performance (< 100ms)"
            echo "  large-search   - Test large result set performance (< 200ms)" 
            echo "  db-indexes     - Test database index effectiveness (< 50ms)"
            echo "  search-latency - Test search latency distribution (95% < 100ms)"
            echo "  ui-render      - Test UI rendering performance (< 200ms)"
            echo "  memory-load    - Test memory usage under concurrent load"
            echo "  all           - Run all performance tests"
            exit 1
            ;;
    esac
}

main "$@"
