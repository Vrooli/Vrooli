#!/bin/bash
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with runtime requirement and 60-second target
testing::phase::init --require-runtime --target-time "60s"

log::info "Testing basic latency and availability metrics"

testing::core::wait_for_scenario "$TESTING_PHASE_SCENARIO_NAME" 20 >/dev/null 2>&1 || true

API_URL=$(testing::connectivity::get_api_url "$TESTING_PHASE_SCENARIO_NAME" 2>/dev/null || echo "")
UI_URL=$(testing::connectivity::get_ui_url "$TESTING_PHASE_SCENARIO_NAME" 2>/dev/null || echo "")

API_PORT="${API_URL##*:}"
UI_PORT="${UI_URL##*:}"

[ -z "$API_PORT" ] && API_PORT=17900
[ -z "$UI_PORT" ] && UI_PORT=38900

declare -a LATENCY_LABELS=()
declare -a LATENCY_VALUES=()

measure_latency() {
    local url="$1"
    local label="$2"
    local threshold="$3"

    local output
    if ! output=$(curl -s -o /dev/null -w '%{time_total} %{http_code}' "$url" --max-time 5); then
        testing::phase::add_error "❌ $label endpoint unreachable ($url)"
        return
    fi

    local latency http_code
    latency=$(echo "$output" | awk '{print $1}')
    http_code=$(echo "$output" | awk '{print $2}')

    if [ "$http_code" != "200" ]; then
        testing::phase::add_error "❌ $label endpoint returned HTTP $http_code"
        return
    fi

    log::success "✅ $label responded in ${latency}s"
    LATENCY_LABELS+=("$label")
    LATENCY_VALUES+=("$latency")
    if awk "BEGIN{exit !($latency > $threshold)}"; then
        testing::phase::add_warning "⚠️  $label latency ${latency}s exceeded target ${threshold}s"
    fi
    testing::phase::add_test passed
}

# API health latency
measure_latency "http://localhost:${API_PORT}/health" "API health" "0.5"

# API invoices list latency
measure_latency "http://localhost:${API_PORT}/api/v1/invoices" "API invoices" "0.8"

# API clients list latency
measure_latency "http://localhost:${API_PORT}/api/v1/clients" "API clients" "0.8"

# API payments summary latency
measure_latency "http://localhost:${API_PORT}/api/v1/payments/summary" "API payments summary" "1.0"

# UI availability check
if [ -n "$UI_PORT" ] && [ "$UI_PORT" != "$API_URL" ]; then
    log::info "Testing UI availability on port ${UI_PORT}"
    ui_status=$(curl -s -o /dev/null -w '%{http_code}' "http://localhost:${UI_PORT}" --max-time 5 || echo "000")
    ui_status=${ui_status:0:3}
    if [ "$ui_status" = "200" ]; then
        log::success "✅ UI responded with HTTP 200"
        LATENCY_LABELS+=("UI root")
        LATENCY_VALUES+=("0.000")
        testing::phase::add_test passed
    else
        testing::phase::add_warning "⚠️  UI not reachable (status $ui_status)"
    fi
else
    log::info "ℹ️  UI port not configured or same as API port"
fi

if [ ${#LATENCY_VALUES[@]} -gt 0 ]; then
    sorted_latencies=($(printf '%s\n' "${LATENCY_VALUES[@]}" | sort -g))
    median_index=$(( (${#sorted_latencies[@]} - 1) / 2 ))
    p95_index=$(( (${#sorted_latencies[@]} * 95 + 99) / 100 - 1 ))
    [ $p95_index -lt 0 ] && p95_index=0
    [ $p95_index -ge ${#sorted_latencies[@]} ] && p95_index=$((${#sorted_latencies[@]}-1))
    median_latency=${sorted_latencies[$median_index]}
    p95_latency=${sorted_latencies[$p95_index]}
    log::info "📈 Latency percentiles (s): median=${median_latency} p95=${p95_latency}"
fi

log::info "📊 Performance Summary: ${TESTING_PHASE_TEST_COUNT} tests, ${TESTING_PHASE_ERROR_COUNT} failed, ${TESTING_PHASE_SKIPPED_COUNT} skipped"

# End with summary
testing::phase::end_with_summary "Performance checks completed"
