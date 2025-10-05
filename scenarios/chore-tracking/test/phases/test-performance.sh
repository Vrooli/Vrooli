#!/bin/bash
# Performance test phase
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with 180-second target
testing::phase::init --target-time "180s"

cd "$TESTING_PHASE_SCENARIO_DIR"

log::info "Running performance tests for chore-tracking"

# Check if scenario is running
if ! vrooli scenario status chore-tracking | grep -q "running"; then
    log::warn "Scenario not running, starting it..."
    vrooli scenario start chore-tracking
    sleep 5
fi

# Performance test: Response time for health check
log::info "Testing health endpoint response time..."
HEALTH_TIME=$(curl -w "%{time_total}" -o /dev/null -s http://localhost:${API_PORT:-8080}/health)
log::info "Health check response time: ${HEALTH_TIME}s"

if (( $(echo "$HEALTH_TIME < 1.0" | bc -l) )); then
    log::success "Health check response time acceptable"
else
    testing::phase::add_error "Health check response time too slow: ${HEALTH_TIME}s"
fi

# Performance test: Chore listing
log::info "Testing chore listing response time..."
CHORES_TIME=$(curl -w "%{time_total}" -o /dev/null -s http://localhost:${API_PORT:-8080}/api/chores)
log::info "Chore listing response time: ${CHORES_TIME}s"

if (( $(echo "$CHORES_TIME < 2.0" | bc -l) )); then
    log::success "Chore listing response time acceptable"
else
    testing::phase::add_error "Chore listing response time too slow: ${CHORES_TIME}s"
fi

# Performance test: User listing
log::info "Testing user listing response time..."
USERS_TIME=$(curl -w "%{time_total}" -o /dev/null -s http://localhost:${API_PORT:-8080}/api/users)
log::info "User listing response time: ${USERS_TIME}s"

if (( $(echo "$USERS_TIME < 2.0" | bc -l) )); then
    log::success "User listing response time acceptable"
else
    testing::phase::add_error "User listing response time too slow: ${USERS_TIME}s"
fi

# Concurrent request test
log::info "Testing concurrent request handling..."
CONCURRENT_REQUESTS=10
START_TIME=$(date +%s.%N)

for i in $(seq 1 $CONCURRENT_REQUESTS); do
    curl -s http://localhost:${API_PORT:-8080}/health > /dev/null &
done

wait

END_TIME=$(date +%s.%N)
TOTAL_TIME=$(echo "$END_TIME - $START_TIME" | bc)
log::info "Concurrent requests completed in: ${TOTAL_TIME}s"

if (( $(echo "$TOTAL_TIME < 5.0" | bc -l) )); then
    log::success "Concurrent request handling acceptable"
else
    testing::phase::add_error "Concurrent request handling too slow: ${TOTAL_TIME}s"
fi

# End with summary
testing::phase::end_with_summary "Performance tests completed"
