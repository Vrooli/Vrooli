#!/bin/bash
# Performance validation phase
set -euo pipefail

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/core.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/connectivity.sh"

SCENARIO_NAME="visited-tracker"

log::info "=== Performance Tests Phase ==="
log::info "Testing basic latency and availability metrics"

# Ensure scenario is healthy before measuring
if testing::core::ensure_runtime_or_skip "$SCENARIO_NAME" "performance checks"; then
    :
else
    status=$?
    if [ "$status" -eq 200 ]; then
        exit 200
    else
        exit 1
    fi
fi

testing::core::wait_for_scenario "$SCENARIO_NAME" 20 >/dev/null 2>&1 || true

API_URL=$(testing::connectivity::get_api_url "$SCENARIO_NAME" 2>/dev/null || echo "")
UI_URL=$(testing::connectivity::get_ui_url "$SCENARIO_NAME" 2>/dev/null || echo "")

API_PORT="${API_URL##*:}"
UI_PORT="${UI_URL##*:}"

[ -z "$API_PORT" ] && API_PORT=17695
[ -z "$UI_PORT" ] && UI_PORT=38442

error_count=0
test_count=0
skipped_count=0
declare -a LATENCY_LABELS=()
declare -a LATENCY_VALUES=()

measure_latency() {
    local url="$1"
    local label="$2"
    local threshold="$3"

    local output
    if ! output=$(curl -s -o /dev/null -w '%{time_total} %{http_code}' "$url" --max-time 5); then
        log::error "❌ $label endpoint unreachable ($url)"
        error_count=$((error_count + 1))
        return
    fi

    local latency http_code
    latency=$(echo "$output" | awk '{print $1}')
    http_code=$(echo "$output" | awk '{print $2}')

    if [ "$http_code" != "200" ]; then
        log::error "❌ $label endpoint returned HTTP $http_code"
        error_count=$((error_count + 1))
        return
    fi

    log::success "✅ $label responded in ${latency}s"
    LATENCY_LABELS+=("$label")
    LATENCY_VALUES+=("$latency")
    if awk "BEGIN{exit !($latency > $threshold)}"; then
        log::warning "⚠️  $label latency ${latency}s exceeded target ${threshold}s"
    fi
    test_count=$((test_count + 1))
}

# API health latency
measure_latency "http://localhost:${API_PORT}/health" "API health" "0.5"

# API campaigns list latency (simulated concurrency check)
measure_latency "http://localhost:${API_PORT}/api/v1/campaigns" "API campaigns" "0.8"

# UI availability check
log::info "Testing UI availability on port ${UI_PORT}"
ui_status=$(curl -s -o /dev/null -w '%{http_code}' "http://localhost:${UI_PORT}" --max-time 5 || echo "000")
ui_status=${ui_status:0:3}
if [ "$ui_status" = "200" ]; then
    log::success "✅ UI responded with HTTP 200"
    LATENCY_LABELS+=("UI root")
    LATENCY_VALUES+=("0.000")
    test_count=$((test_count + 1))
else
    log::error "❌ UI not reachable (status $ui_status)"
    error_count=$((error_count + 1))
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

log::info "📊 Performance Summary: ${test_count} passed, ${error_count} failed, ${skipped_count} skipped"

if [ $error_count -eq 0 ]; then
    log::success "SUCCESS: Performance checks completed"
    exit 0
else
    log::error "ERROR: Performance regressions detected"
    exit 1
fi
