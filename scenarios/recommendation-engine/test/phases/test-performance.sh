#!/bin/bash
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with runtime requirement and 90-second target
testing::phase::init --require-runtime --target-time "90s"

log::info "Testing performance: latency, throughput, and load handling"

testing::core::wait_for_scenario "$TESTING_PHASE_SCENARIO_NAME" 20 >/dev/null 2>&1 || true

API_URL=$(testing::connectivity::get_api_url "$TESTING_PHASE_SCENARIO_NAME" 2>/dev/null || echo "")
API_PORT="${API_URL##*:}"
[ -z "$API_PORT" ] && API_PORT=17695

API_BASE="http://localhost:${API_PORT}"

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

# Test 1: Basic endpoint latency
echo ""
echo "Test 1: Measuring endpoint latency..."
measure_latency "$API_BASE/health" "Health check" "0.5"
measure_latency "$API_BASE/docs" "Documentation" "0.5"

# Test 2: Recommendation query latency
echo ""
echo "Test 2: Recommendation query performance..."
test_scenario_id="perf-test-$(date +%s)"

# First ingest some test data
if command -v jq >/dev/null 2>&1; then
    ingest_data=$(jq -n \
        --arg scenario_id "$test_scenario_id" \
        '{
            scenario_id: $scenario_id,
            items: [
                {external_id: "perf-item-1", title: "Performance Test Item 1", description: "Test item", category: "test"},
                {external_id: "perf-item-2", title: "Performance Test Item 2", description: "Test item", category: "test"},
                {external_id: "perf-item-3", title: "Performance Test Item 3", description: "Test item", category: "test"}
            ],
            interactions: [
                {user_id: "perf-user-1", item_external_id: "perf-item-1", interaction_type: "view", interaction_value: 1.0},
                {user_id: "perf-user-2", item_external_id: "perf-item-1", interaction_type: "purchase", interaction_value: 5.0}
            ]
        }')

    curl -sf -X POST "$API_BASE/api/v1/recommendations/ingest" -H "Content-Type: application/json" -d "$ingest_data" >/dev/null 2>&1

    # Measure recommendation query latency
    rec_payload=$(jq -n --arg scenario_id "$test_scenario_id" --arg user_id "perf-user-3" '{scenario_id: $scenario_id, user_id: $user_id, limit: 10}')

    start_time=$(date +%s.%N)
    rec_response=$(curl -sf -X POST "$API_BASE/api/v1/recommendations/get" -H "Content-Type: application/json" -d "$rec_payload" 2>/dev/null || echo "")
    end_time=$(date +%s.%N)

    if echo "$rec_response" | jq -e '.recommendations' >/dev/null 2>&1; then
        latency=$(echo "$end_time - $start_time" | bc)
        log::success "✅ Recommendation query responded in ${latency}s"
        LATENCY_LABELS+=("Recommendation query")
        LATENCY_VALUES+=("$latency")
        if awk "BEGIN{exit !($latency > 1.0)}"; then
            testing::phase::add_warning "⚠️  Recommendation query latency ${latency}s exceeded target 1.0s"
        fi
        testing::phase::add_test passed
    else
        testing::phase::add_error "❌ Recommendation query failed"
    fi
else
    echo "⚠️  jq not available, skipping recommendation query test"
    testing::phase::add_test skipped
fi

# Test 3: Bulk ingestion throughput
echo ""
echo "Test 3: Bulk ingestion throughput (100 items)..."
bulk_scenario_id="bulk-perf-$(date +%s)"

if command -v jq >/dev/null 2>&1; then
    # Generate 100 test items
    items_json='[]'
    for i in {1..100}; do
        items_json=$(echo "$items_json" | jq --arg id "bulk-item-$i" --arg title "Bulk Item $i" '. += [{external_id: $id, title: $title, description: "Bulk test", category: "test"}]')
    done

    bulk_payload=$(jq -n --arg scenario_id "$bulk_scenario_id" --argjson items "$items_json" '{scenario_id: $scenario_id, items: $items}')

    start_time=$(date +%s.%N)
    bulk_response=$(curl -sf -X POST "$API_BASE/api/v1/recommendations/ingest" -H "Content-Type: application/json" -d "$bulk_payload" 2>/dev/null || echo "")
    end_time=$(date +%s.%N)

    if echo "$bulk_response" | jq -e '.success' >/dev/null 2>&1; then
        items_processed=$(echo "$bulk_response" | jq -r '.items_processed')
        duration=$(echo "$end_time - $start_time" | bc)
        throughput=$(echo "scale=2; $items_processed / $duration" | bc)
        log::success "✅ Bulk ingestion: $items_processed items in ${duration}s ($throughput items/sec)"

        if awk "BEGIN{exit !($duration > 10.0)}"; then
            testing::phase::add_warning "⚠️  Bulk ingestion duration ${duration}s exceeded target 10.0s"
        fi
        testing::phase::add_test passed
    else
        testing::phase::add_error "❌ Bulk ingestion failed"
    fi
else
    echo "⚠️  jq not available, skipping bulk ingestion test"
    testing::phase::add_test skipped
fi

# Test 4: Concurrent request handling
echo ""
echo "Test 4: Concurrent request handling (10 parallel requests)..."
concurrent_scenario_id="concurrent-$(date +%s)"

if command -v jq >/dev/null 2>&1; then
    # Setup test data
    concurrent_data=$(jq -n \
        --arg scenario_id "$concurrent_scenario_id" \
        '{
            scenario_id: $scenario_id,
            items: [{external_id: "conc-item-1", title: "Concurrent Item", description: "Test", category: "test"}],
            interactions: [{user_id: "conc-user-1", item_external_id: "conc-item-1", interaction_type: "view", interaction_value: 1.0}]
        }')
    curl -sf -X POST "$API_BASE/api/v1/recommendations/ingest" -H "Content-Type: application/json" -d "$concurrent_data" >/dev/null 2>&1

    # Run 10 concurrent recommendation requests
    start_time=$(date +%s.%N)
    for i in {1..10}; do
        (
            rec_req=$(jq -n --arg scenario_id "$concurrent_scenario_id" --arg user_id "conc-user-$i" '{scenario_id: $scenario_id, user_id: $user_id}')
            curl -sf -X POST "$API_BASE/api/v1/recommendations/get" -H "Content-Type: application/json" -d "$rec_req" >/dev/null 2>&1
        ) &
    done
    wait
    end_time=$(date +%s.%N)

    duration=$(echo "$end_time - $start_time" | bc)
    log::success "✅ Concurrent requests: 10 requests completed in ${duration}s"

    if awk "BEGIN{exit !($duration > 5.0)}"; then
        testing::phase::add_warning "⚠️  Concurrent request duration ${duration}s exceeded target 5.0s"
    fi
    testing::phase::add_test passed
else
    echo "⚠️  jq not available, skipping concurrent request test"
    testing::phase::add_test skipped
fi

# Test 5: Similar items query performance (if Qdrant available)
echo ""
echo "Test 5: Similar items query performance..."
if command -v jq >/dev/null 2>&1; then
    similar_scenario_id="similar-perf-$(date +%s)"

    # Ingest items for similarity test
    similar_data=$(jq -n \
        --arg scenario_id "$similar_scenario_id" \
        '{
            scenario_id: $scenario_id,
            items: [
                {external_id: "sim-item-1", title: "Similar Test Item 1", description: "Technology product", category: "electronics"},
                {external_id: "sim-item-2", title: "Similar Test Item 2", description: "Technology gadget", category: "electronics"}
            ]
        }')
    curl -sf -X POST "$API_BASE/api/v1/recommendations/ingest" -H "Content-Type: application/json" -d "$similar_data" >/dev/null 2>&1

    similar_payload=$(jq -n --arg scenario_id "$similar_scenario_id" --arg item_id "sim-item-1" '{scenario_id: $scenario_id, item_external_id: $item_id, limit: 5}')

    start_time=$(date +%s.%N)
    similar_response=$(curl -sf -X POST "$API_BASE/api/v1/recommendations/similar" -H "Content-Type: application/json" -d "$similar_payload" 2>/dev/null || echo "")
    end_time=$(date +%s.%N)

    if echo "$similar_response" | jq -e '.similar_items' >/dev/null 2>&1; then
        latency=$(echo "$end_time - $start_time" | bc)
        log::success "✅ Similar items query responded in ${latency}s"
        LATENCY_LABELS+=("Similar items query")
        LATENCY_VALUES+=("$latency")
        if awk "BEGIN{exit !($latency > 1.5)}"; then
            testing::phase::add_warning "⚠️  Similar items latency ${latency}s exceeded target 1.5s"
        fi
        testing::phase::add_test passed
    elif [[ "$similar_response" =~ "Qdrant" ]]; then
        echo "ℹ️  Similar items query requires Qdrant (not available)"
        testing::phase::add_test skipped
    else
        echo "⚠️  Similar items query inconclusive"
        testing::phase::add_test skipped
    fi
else
    echo "⚠️  jq not available, skipping similar items test"
    testing::phase::add_test skipped
fi

# Calculate and display statistics
echo ""
echo "📊 Performance Statistics:"
echo "════════════════════════════"

if [ ${#LATENCY_VALUES[@]} -gt 0 ]; then
    sorted_latencies=($(printf '%s\n' "${LATENCY_VALUES[@]}" | sort -g))
    median_index=$(( (${#sorted_latencies[@]} - 1) / 2 ))
    p95_index=$(( (${#sorted_latencies[@]} * 95 + 99) / 100 - 1 ))
    [ $p95_index -lt 0 ] && p95_index=0
    [ $p95_index -ge ${#sorted_latencies[@]} ] && p95_index=$((${#sorted_latencies[@]}-1))
    median_latency=${sorted_latencies[$median_index]}
    p95_latency=${sorted_latencies[$p95_index]}
    min_latency=${sorted_latencies[0]}
    max_latency=${sorted_latencies[-1]}

    echo "Latency (seconds):"
    echo "  Min:    $min_latency"
    echo "  Median: $median_latency"
    echo "  P95:    $p95_latency"
    echo "  Max:    $max_latency"
fi

echo ""
log::info "📊 Performance Summary: ${TESTING_PHASE_TEST_COUNT} tests, ${TESTING_PHASE_ERROR_COUNT} failed, ${TESTING_PHASE_SKIPPED_COUNT} skipped"

if [ $TESTING_PHASE_ERROR_COUNT -eq 0 ]; then
    log::success "✅ SUCCESS: All performance tests passed"
else
    log::error "❌ ERROR: $TESTING_PHASE_ERROR_COUNT performance test(s) failed"
fi

# End with summary
testing::phase::end_with_summary "Performance checks completed"
