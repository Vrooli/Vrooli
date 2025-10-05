#!/bin/bash
# Performance test phase
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase
testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR"

log::info "Running performance tests..."

API_PORT="${API_PORT:-17961}"

# Helper function to measure response time
measure_response_time() {
    local url=$1
    local max_time=$2
    local description=$3

    local start=$(date +%s%N)
    local response=$(curl -sf "$url" 2>/dev/null)
    local end=$(date +%s%N)

    local duration=$(( (end - start) / 1000000 )) # Convert to milliseconds

    if [ -z "$response" ]; then
        testing::phase::add_warning "$description: No response (API may not be running)"
        return 1
    fi

    if [ $duration -gt $max_time ]; then
        testing::phase::add_error "$description: Too slow (${duration}ms > ${max_time}ms)"
        return 1
    else
        log::success "$description: ${duration}ms (target: <${max_time}ms)"
        return 0
    fi
}

# Test 1: Health check should be fast
log::info "Test: Health check response time"
measure_response_time \
    "http://localhost:${API_PORT}/api/v1/health" \
    100 \
    "Health check"

# Test 2: Registry response time
log::info "Test: Registry response time"
measure_response_time \
    "http://localhost:${API_PORT}/api/v1/mcp/registry" \
    50 \
    "Registry query"

# Test 3: Endpoints listing response time
log::info "Test: Endpoints listing response time"
measure_response_time \
    "http://localhost:${API_PORT}/api/v1/mcp/endpoints" \
    200 \
    "Endpoints scan"

# Test 4: Concurrent requests handling
log::info "Test: Concurrent request handling"
CONCURRENT_COUNT=10
SUCCESS_COUNT=0

for i in $(seq 1 $CONCURRENT_COUNT); do
    curl -sf "http://localhost:${API_PORT}/api/v1/health" &>/dev/null &
done

wait

for i in $(seq 1 $CONCURRENT_COUNT); do
    if curl -sf "http://localhost:${API_PORT}/api/v1/health" | jq -e '.status == "healthy"' &>/dev/null; then
        ((SUCCESS_COUNT++))
    fi
done

if [ $SUCCESS_COUNT -eq $CONCURRENT_COUNT ]; then
    log::success "Handled $CONCURRENT_COUNT concurrent requests successfully"
else
    testing::phase::add_warning "Only $SUCCESS_COUNT/$CONCURRENT_COUNT concurrent requests succeeded"
fi

# Test 5: Memory usage check (if possible)
log::info "Test: Memory usage check"
if command -v ps &>/dev/null; then
    if pgrep -f "scenario-to-mcp-api" &>/dev/null; then
        MEM_USAGE=$(ps aux | grep "scenario-to-mcp-api" | grep -v grep | awk '{print $6}')
        if [ -n "$MEM_USAGE" ]; then
            MEM_MB=$((MEM_USAGE / 1024))
            log::info "API memory usage: ${MEM_MB}MB"

            if [ $MEM_MB -lt 100 ]; then
                log::success "Memory usage is acceptable"
            elif [ $MEM_MB -lt 200 ]; then
                testing::phase::add_warning "Memory usage is moderate: ${MEM_MB}MB"
            else
                testing::phase::add_error "Memory usage is high: ${MEM_MB}MB"
            fi
        fi
    else
        testing::phase::add_warning "API process not found for memory check"
    fi
fi

testing::phase::end_with_summary "Performance tests completed"
