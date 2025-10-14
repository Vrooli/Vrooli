#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::log "Running funnel-builder performance tests..."

# Get API port dynamically
API_PORT=$(vrooli scenario port funnel-builder API_PORT 2>/dev/null || echo "16133")
API_BASE_URL="http://localhost:$API_PORT"

testing::phase::log "Using API at $API_BASE_URL"

# Test concurrent request handling
testing::phase::log "Testing concurrent request handling..."

if curl -f -s "$API_BASE_URL/health" >/dev/null 2>&1; then
    testing::phase::log "Running concurrent health check test..."

    # Use GNU parallel if available, otherwise use basic loop
    if command -v parallel &>/dev/null; then
        testing::phase::log "Sending 100 concurrent requests..."
        seq 100 | parallel -j 10 "curl -s -w '\n%{http_code}\n' $API_BASE_URL/health" > /tmp/concurrent-test.log 2>&1

        # Count successful responses
        success_count=$(grep "^200$" /tmp/concurrent-test.log | wc -l)
        testing::phase::log "Successful responses: $success_count/100"

        if [ "$success_count" -ge 95 ]; then
            testing::phase::success "Concurrent request test passed (≥95% success rate)"
        else
            testing::phase::warn "Concurrent request test had lower success rate: $success_count%"
        fi
    else
        testing::phase::warn "GNU parallel not available, skipping concurrent request test"
    fi
else
    testing::phase::warn "API not running, skipping live concurrent tests"
fi

# Memory usage test (if running)
testing::phase::log "Checking memory usage..."

if pgrep -f "funnel-builder-api" >/dev/null; then
    pid=$(pgrep -f "funnel-builder-api" | head -1)
    if command -v ps &>/dev/null; then
        mem_usage=$(ps -p "$pid" -o rss= | awk '{print $1/1024}')
        testing::phase::log "Current memory usage: ${mem_usage}MB"

        # Warn if memory usage is very high (>512MB per PRD target)
        if (( $(echo "$mem_usage > 512" | bc -l 2>/dev/null || echo "0") )); then
            testing::phase::warn "High memory usage detected: ${mem_usage}MB (target: <512MB)"
        else
            testing::phase::success "Memory usage within normal range"
        fi
    fi
else
    testing::phase::warn "funnel-builder-api not running, skipping memory test"
fi

# CPU usage test
testing::phase::log "Checking CPU usage..."

if pgrep -f "funnel-builder-api" >/dev/null; then
    pid=$(pgrep -f "funnel-builder-api" | head -1)
    if command -v ps &>/dev/null; then
        cpu_usage=$(ps -p "$pid" -o %cpu= 2>/dev/null || echo "0.0")
        testing::phase::log "Current CPU usage: ${cpu_usage}%"

        # PRD target: <10% CPU
        if (( $(echo "$cpu_usage > 10" | bc -l 2>/dev/null || echo "0") )); then
            testing::phase::warn "High CPU usage detected: ${cpu_usage}% (target: <10%)"
        else
            testing::phase::success "CPU usage within target range"
        fi
    fi
else
    testing::phase::warn "funnel-builder-api not running, skipping CPU test"
fi

# Response time test
testing::phase::log "Testing response times..."

if curl -f -s "$API_BASE_URL/health" >/dev/null 2>&1; then
    total_time=0
    iterations=10

    for i in $(seq 1 $iterations); do
        response_time=$(curl -s -w "%{time_total}" -o /dev/null "$API_BASE_URL/health")
        total_time=$(echo "$total_time + $response_time" | bc -l 2>/dev/null || echo "$total_time")
    done

    avg_time=$(echo "scale=3; $total_time / $iterations" | bc -l 2>/dev/null || echo "0")
    testing::phase::log "Average response time: ${avg_time}s"

    # PRD target: <200ms for step transitions
    if (( $(echo "$avg_time < 0.2" | bc -l 2>/dev/null || echo "0") )); then
        testing::phase::success "Response time within target range (<200ms)"
    else
        testing::phase::warn "Response time slower than target: ${avg_time}s (target: <200ms)"
    fi
else
    testing::phase::warn "API not running, skipping response time test"
fi

# Funnel execution performance test
testing::phase::log "Testing funnel execution performance..."

if curl -f -s "$API_BASE_URL/health" >/dev/null 2>&1; then
    # Create a test funnel
    funnel_response=$(curl -s -X POST "$API_BASE_URL/api/v1/funnels" \
        -H "Content-Type: application/json" \
        -d '{"name":"Performance Test Funnel","steps":[]}')

    if echo "$funnel_response" | grep -q '"slug"'; then
        funnel_slug=$(echo "$funnel_response" | jq -r '.slug' 2>/dev/null)

        # Test funnel retrieval time
        start_time=$(date +%s.%N)
        curl -s "$API_BASE_URL/api/v1/funnels" >/dev/null
        end_time=$(date +%s.%N)

        retrieval_time=$(echo "$end_time - $start_time" | bc -l)
        testing::phase::log "Funnel list retrieval time: ${retrieval_time}s"

        if (( $(echo "$retrieval_time < 0.5" | bc -l 2>/dev/null || echo "0") )); then
            testing::phase::success "Funnel retrieval within acceptable time"
        else
            testing::phase::warn "Funnel retrieval slower than expected: ${retrieval_time}s"
        fi
    fi
else
    testing::phase::warn "API not running, skipping funnel execution test"
fi

# Database query performance (if accessible)
testing::phase::log "Testing database query performance..."

if command -v psql &>/dev/null; then
    if psql -h localhost -U vrooli -d vrooli -c "SELECT COUNT(*) FROM funnel_builder.funnels" &>/dev/null; then
        start_time=$(date +%s.%N)
        psql -h localhost -U vrooli -d vrooli -c "SELECT COUNT(*) FROM funnel_builder.funnels" >/dev/null 2>&1
        end_time=$(date +%s.%N)

        query_time=$(echo "$end_time - $start_time" | bc -l)
        testing::phase::log "Database query time: ${query_time}s"

        if (( $(echo "$query_time < 0.1" | bc -l 2>/dev/null || echo "0") )); then
            testing::phase::success "Database query performance good"
        else
            testing::phase::warn "Database query slower than expected: ${query_time}s"
        fi
    else
        testing::phase::warn "Database not accessible, skipping query performance test"
    fi
else
    testing::phase::warn "psql not available, skipping database performance test"
fi

# UI build performance test
testing::phase::log "Testing UI build performance..."

if [ -d "ui" ]; then
    cd ui

    if [ -f "package.json" ]; then
        start_time=$(date +%s.%N)
        if npm run build >/dev/null 2>&1; then
            end_time=$(date +%s.%N)
            build_time=$(echo "$end_time - $start_time" | bc -l)
            testing::phase::log "UI build time: ${build_time}s"

            # UI builds can be slow, allow up to 60s
            if (( $(echo "$build_time < 60" | bc -l 2>/dev/null || echo "0") )); then
                testing::phase::success "UI build completed in acceptable time"
            else
                testing::phase::warn "UI build slower than expected: ${build_time}s"
            fi
        else
            testing::phase::warn "UI build failed, cannot measure performance"
        fi
    fi

    cd ..
fi

testing::phase::end_with_summary "Performance tests completed"
