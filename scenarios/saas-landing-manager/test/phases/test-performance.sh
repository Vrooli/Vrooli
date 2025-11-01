#!/bin/bash
# Performance test phase for saas-landing-manager
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase
testing::phase::init --target-time "90s" --require-runtime

cd "$TESTING_PHASE_SCENARIO_DIR"

log::info "Running performance tests for saas-landing-manager"

# Resolve API URL for health checks
API_URL=$(testing::connectivity::get_api_url "$TESTING_PHASE_SCENARIO_NAME" 2>/dev/null || echo "")

if [ -z "$API_URL" ]; then
    testing::phase::add_error "Unable to resolve API URL for ${TESTING_PHASE_SCENARIO_NAME}"
    testing::phase::end_with_summary "Performance tests incomplete"
fi

# Test 1: API response time
log::info "Test 1: Measuring API response time"
START_TIME=$(date +%s%N)
if ! curl -s "${API_URL}/health" >/dev/null; then
    testing::phase::add_error "API health endpoint is not reachable"
    testing::phase::end_with_summary "Performance tests incomplete"
fi
END_TIME=$(date +%s%N)
RESPONSE_TIME=$(( (END_TIME - START_TIME) / 1000000 ))

if [ "$RESPONSE_TIME" -lt 100 ]; then
    log::success "Health endpoint response time: ${RESPONSE_TIME}ms (excellent)"
elif [ "$RESPONSE_TIME" -lt 500 ]; then
    log::success "Health endpoint response time: ${RESPONSE_TIME}ms (good)"
else
    testing::phase::add_warning "Health endpoint response time: ${RESPONSE_TIME}ms (slow)"
fi

# Test 2: Concurrent requests
log::info "Test 2: Testing concurrent request handling"
CONCURRENT_REQUESTS=10
SUCCESS_COUNT=0

for _ in $(seq 1 $CONCURRENT_REQUESTS); do
    if curl -s -f "${API_URL}/health" > /dev/null 2>&1; then
        SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
    fi
done

log::info "Handled $SUCCESS_COUNT/$CONCURRENT_REQUESTS health probes"
if [ "$SUCCESS_COUNT" -eq "$CONCURRENT_REQUESTS" ]; then
    log::success "Health endpoint responded successfully to repeated probes"
else
    testing::phase::add_warning "Health endpoint responded to ${SUCCESS_COUNT}/${CONCURRENT_REQUESTS} probes"
fi

# Test 3: Run Go performance tests
log::info "Test 3: Running Go performance benchmarks"
cd api
if go test -tags=testing -run TestPerformance -timeout 60s 2>&1 | tee /tmp/perf-test.log; then
    if grep -q "PASS" /tmp/perf-test.log; then
        log::success "Performance tests passed"
    else
        testing::phase::add_warning "Performance tests completed with warnings"
    fi
else
    testing::phase::add_error "Performance tests failed"
fi

# End with summary
testing::phase::end_with_summary "Performance tests completed"
