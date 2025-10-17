#!/bin/bash
#
# Performance Test Phase for knowledge-observatory
# Integrates with centralized Vrooli testing infrastructure
#

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::log "Running performance tests..."

API_PORT="${API_PORT:-17822}"
UI_PORT="${UI_PORT:-35785}"

# Check if API is running
if ! curl -f -s "http://localhost:$API_PORT/health" >/dev/null 2>&1; then
    testing::phase::warn "API not running, skipping performance tests"
    testing::phase::end_with_summary "API not available for performance testing"
fi

# Response time test - Health endpoint
testing::phase::log "Testing health endpoint response times..."

total_time=0
iterations=10
failed_requests=0

for i in $(seq 1 $iterations); do
    response_time=$(curl -s -w "%{time_total}" -o /dev/null "http://localhost:$API_PORT/health" 2>/dev/null)
    if [ $? -eq 0 ]; then
        total_time=$(echo "$total_time + $response_time" | bc)
    else
        ((failed_requests++))
    fi
done

if [ "$failed_requests" -gt 0 ]; then
    testing::phase::warn "$failed_requests/$iterations health requests failed"
fi

avg_time=$(echo "scale=3; $total_time / ($iterations - $failed_requests)" | bc 2>/dev/null || echo "0")
testing::phase::log "Average health response time: ${avg_time}s"

# Health endpoint should respond in under 2 seconds (it checks multiple dependencies)
if (( $(echo "$avg_time < 2.0" | bc -l) )); then
    testing::phase::success "Health response time within acceptable range (<2s)"
elif (( $(echo "$avg_time < 5.0" | bc -l) )); then
    testing::phase::warn "Health response time slower than ideal: ${avg_time}s (target <2s)"
else
    testing::phase::error "Health response time too slow: ${avg_time}s (target <2s)"
fi

# Response time test - Search endpoint
testing::phase::log "Testing search endpoint response times..."

total_time=0
iterations=5
failed_requests=0

for i in $(seq 1 $iterations); do
    response_time=$(curl -s -w "%{time_total}" -o /dev/null \
        -X POST "http://localhost:$API_PORT/api/v1/knowledge/search" \
        -H "Content-Type: application/json" \
        -d '{"query":"test query","limit":10}' 2>/dev/null)
    if [ $? -eq 0 ]; then
        total_time=$(echo "$total_time + $response_time" | bc)
    else
        ((failed_requests++))
    fi
done

if [ "$failed_requests" -gt 0 ]; then
    testing::phase::warn "$failed_requests/$iterations search requests failed"
fi

avg_time=$(echo "scale=3; $total_time / ($iterations - $failed_requests)" | bc 2>/dev/null || echo "0")
testing::phase::log "Average search response time: ${avg_time}s"

# Search should respond in under 1 second per PRD requirement (500ms target)
if (( $(echo "$avg_time < 1.0" | bc -l) )); then
    testing::phase::success "Search response time within acceptable range (<1s)"
else
    testing::phase::warn "Search response time slower than target: ${avg_time}s (target <1s)"
fi

# Response time test - Graph endpoint
testing::phase::log "Testing graph endpoint response times..."

total_time=0
iterations=5
failed_requests=0

for i in $(seq 1 $iterations); do
    response_time=$(curl -s -w "%{time_total}" -o /dev/null \
        "http://localhost:$API_PORT/api/v1/knowledge/graph" 2>/dev/null)
    if [ $? -eq 0 ]; then
        total_time=$(echo "$total_time + $response_time" | bc)
    else
        ((failed_requests++))
    fi
done

if [ "$failed_requests" -gt 0 ]; then
    testing::phase::warn "$failed_requests/$iterations graph requests failed"
fi

avg_time=$(echo "scale=3; $total_time / ($iterations - $failed_requests)" | bc 2>/dev/null || echo "0")
testing::phase::log "Average graph response time: ${avg_time}s"

# Graph rendering should be under 2 seconds per PRD
if (( $(echo "$avg_time < 2.0" | bc -l) )); then
    testing::phase::success "Graph response time within acceptable range (<2s)"
else
    testing::phase::warn "Graph response time slower than target: ${avg_time}s (target <2s)"
fi

# Concurrent request handling test
testing::phase::log "Testing concurrent request handling..."

if command -v parallel &>/dev/null; then
    testing::phase::log "Sending 50 concurrent health check requests..."
    seq 50 | parallel -j 10 "curl -s -w '\n%{http_code}\n' http://localhost:$API_PORT/health" > /tmp/concurrent-ko-test.log 2>&1

    # Count successful responses
    success_count=$(grep -c "^200$" /tmp/concurrent-ko-test.log 2>/dev/null || echo "0")
    testing::phase::log "Successful responses: $success_count/50"

    if [ "$success_count" -ge 45 ]; then
        testing::phase::success "Concurrent request test passed (≥90% success rate)"
    elif [ "$success_count" -ge 35 ]; then
        testing::phase::warn "Concurrent request test had lower success rate: $success_count/50"
    else
        testing::phase::error "Concurrent request test failed: only $success_count/50 succeeded"
    fi

    rm -f /tmp/concurrent-ko-test.log
else
    testing::phase::warn "GNU parallel not available, skipping concurrent request test"
fi

# Memory usage test
testing::phase::log "Checking memory usage..."

if pgrep -f "knowledge-observatory-api" >/dev/null; then
    pid=$(pgrep -f "knowledge-observatory-api" | head -1)
    if command -v ps &>/dev/null; then
        mem_usage=$(ps -p "$pid" -o rss= 2>/dev/null | awk '{print $1/1024}')
        testing::phase::log "Current memory usage: ${mem_usage}MB"

        # Warn if memory usage exceeds 512MB per PRD requirement
        if (( $(echo "$mem_usage > 512" | bc -l 2>/dev/null || echo 0) )); then
            testing::phase::warn "Memory usage exceeds target: ${mem_usage}MB (target <512MB)"
        else
            testing::phase::success "Memory usage within target range (<512MB)"
        fi
    fi
else
    testing::phase::warn "knowledge-observatory-api not running, skipping memory test"
fi

# CPU usage test
testing::phase::log "Checking CPU usage..."

if pgrep -f "knowledge-observatory-api" >/dev/null; then
    pid=$(pgrep -f "knowledge-observatory-api" | head -1)
    if command -v ps &>/dev/null; then
        # Sample CPU usage a few times
        cpu_samples=0
        cpu_total=0
        for i in $(seq 1 3); do
            cpu=$(ps -p "$pid" -o %cpu= 2>/dev/null || echo "0.0")
            cpu_total=$(echo "$cpu_total + $cpu" | bc 2>/dev/null || echo "0")
            ((cpu_samples++))
            sleep 1
        done

        cpu_avg=$(echo "scale=1; $cpu_total / $cpu_samples" | bc 2>/dev/null || echo "0.0")
        testing::phase::log "Average CPU usage: ${cpu_avg}%"

        # Warn if CPU usage exceeds 10% at rest per PRD
        if (( $(echo "$cpu_avg > 10" | bc -l 2>/dev/null || echo 0) )); then
            testing::phase::warn "CPU usage higher than target: ${cpu_avg}% (target <10%)"
        else
            testing::phase::success "CPU usage within target range (<10%)"
        fi
    fi
else
    testing::phase::warn "knowledge-observatory-api not running, skipping CPU test"
fi

# UI load time test
testing::phase::log "Testing UI load time..."

ui_load_time=$(curl -s -w "%{time_total}" -o /dev/null "http://localhost:$UI_PORT/" 2>/dev/null)
if [ $? -eq 0 ]; then
    testing::phase::log "UI load time: ${ui_load_time}s"

    if (( $(echo "$ui_load_time < 1.0" | bc -l) )); then
        testing::phase::success "UI loads quickly (<1s)"
    else
        testing::phase::warn "UI load time: ${ui_load_time}s"
    fi
else
    testing::phase::warn "UI not accessible on port $UI_PORT"
fi

testing::phase::end_with_summary "Performance tests completed"
