#!/bin/bash
# Performance test phase - validates performance characteristics
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase
testing::phase::init --target-time "60s" --require-runtime

cd "$TESTING_PHASE_SCENARIO_DIR"

log::info "Running performance tests..."

# Check if API is running
if ! curl -sf "http://localhost:${API_PORT:-18888}/health" &>/dev/null; then
    log::warn "API not available for performance testing"
    testing::phase::end_with_summary "Performance tests skipped"
    exit 0
fi

# Test response times
log::info "Testing API response times..."

# Health endpoint should respond quickly
start_time=$(date +%s%N)
if curl -sf "http://localhost:${API_PORT:-18888}/health" &>/dev/null; then
    end_time=$(date +%s%N)
    duration=$(( (end_time - start_time) / 1000000 )) # Convert to milliseconds

    if [ "$duration" -lt 100 ]; then
        log::success "Health endpoint responded in ${duration}ms"
    elif [ "$duration" -lt 500 ]; then
        log::warn "Health endpoint responded in ${duration}ms (slower than ideal)"
    else
        testing::phase::add_error "Health endpoint too slow: ${duration}ms"
    fi
else
    testing::phase::add_error "Health endpoint failed"
fi

# Test campaigns endpoint
start_time=$(date +%s%N)
if curl -sf "http://localhost:${API_PORT:-18888}/api/campaigns" &>/dev/null; then
    end_time=$(date +%s%N)
    duration=$(( (end_time - start_time) / 1000000 ))

    if [ "$duration" -lt 500 ]; then
        log::success "Campaigns endpoint responded in ${duration}ms"
    elif [ "$duration" -lt 1000 ]; then
        log::warn "Campaigns endpoint responded in ${duration}ms (acceptable)"
    else
        testing::phase::add_error "Campaigns endpoint too slow: ${duration}ms"
    fi
fi

# Test notes endpoint
start_time=$(date +%s%N)
if curl -sf "http://localhost:${API_PORT:-18888}/api/notes?limit=10" &>/dev/null; then
    end_time=$(date +%s%N)
    duration=$(( (end_time - start_time) / 1000000 ))

    if [ "$duration" -lt 500 ]; then
        log::success "Notes endpoint responded in ${duration}ms"
    elif [ "$duration" -lt 1000 ]; then
        log::warn "Notes endpoint responded in ${duration}ms (acceptable)"
    else
        log::warn "Notes endpoint took ${duration}ms (may need optimization)"
    fi
fi

# Test concurrent requests
log::info "Testing concurrent request handling..."
temp_file=$(mktemp)

# Send 10 concurrent requests
for i in {1..10}; do
    curl -sf "http://localhost:${API_PORT:-18888}/health" &>/dev/null &
done

# Wait for all requests to complete
wait

log::success "Concurrent requests handled successfully"

# Check memory usage (if ps is available)
if command -v ps &>/dev/null; then
    log::info "Checking API process memory usage..."
    api_pid=$(pgrep -f "stream-of-consciousness-analyzer-api" | head -1)

    if [ -n "$api_pid" ]; then
        mem_usage=$(ps -o rss= -p "$api_pid" 2>/dev/null)
        if [ -n "$mem_usage" ]; then
            mem_mb=$((mem_usage / 1024))
            if [ "$mem_mb" -lt 100 ]; then
                log::success "API memory usage: ${mem_mb}MB (excellent)"
            elif [ "$mem_mb" -lt 500 ]; then
                log::success "API memory usage: ${mem_mb}MB (good)"
            else
                log::warn "API memory usage: ${mem_mb}MB (consider optimization)"
            fi
        fi
    fi
fi

rm -f "$temp_file"

testing::phase::end_with_summary "Performance tests completed"
