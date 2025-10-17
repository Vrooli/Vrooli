#!/bin/bash
# Performance testing phase for symbol-search scenario

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "Running performance tests for symbol-search..."

API_PORT="${API_PORT:-8080}"
API_URL="http://localhost:${API_PORT}"

# Check if API is running
if ! curl -s "${API_URL}/health" > /dev/null 2>&1; then
    echo "⚠️  API not running - skipping performance tests"
    exit 0
fi

# Function to measure response time
measure_response_time() {
    local endpoint="$1"
    local method="${2:-GET}"
    local data="${3:-}"

    local total_time=0
    local iterations=10
    local failed=0

    for i in $(seq 1 $iterations); do
        if [ "$method" = "GET" ]; then
            time_ms=$(curl -s -w "%{time_total}" -o /dev/null "${API_URL}${endpoint}")
        else
            time_ms=$(curl -s -X POST -w "%{time_total}" -o /dev/null \
                -H "Content-Type: application/json" \
                -d "$data" \
                "${API_URL}${endpoint}")
        fi

        # Convert seconds to milliseconds
        time_ms=$(echo "$time_ms * 1000" | bc)
        total_time=$(echo "$total_time + $time_ms" | bc)
    done

    avg_time=$(echo "scale=2; $total_time / $iterations" | bc)
    echo "$avg_time"
}

echo "Testing search endpoint performance..."
search_time=$(measure_response_time "/api/search?q=LATIN&limit=100")
echo "  Average search time: ${search_time}ms"

# Target: < 50ms for search
if (( $(echo "$search_time > 50" | bc -l) )); then
    echo "  ⚠️  Search time exceeds 50ms target"
else
    echo "  ✅ Search performance meets target"
fi

echo "Testing character detail endpoint performance..."
char_time=$(measure_response_time "/api/character/U+0041")
echo "  Average character detail time: ${char_time}ms"

# Target: < 25ms for character detail
if (( $(echo "$char_time > 25" | bc -l) )); then
    echo "  ⚠️  Character detail time exceeds 25ms target"
else
    echo "  ✅ Character detail performance meets target"
fi

echo "Testing categories endpoint performance..."
categories_time=$(measure_response_time "/api/categories")
echo "  Average categories time: ${categories_time}ms"

echo "Testing blocks endpoint performance..."
blocks_time=$(measure_response_time "/api/blocks")
echo "  Average blocks time: ${blocks_time}ms"

echo "Testing bulk range endpoint performance..."
bulk_data='{"ranges":[{"start":"U+0041","end":"U+005A"}]}'
bulk_time=$(measure_response_time "/api/bulk/range" "POST" "$bulk_data")
echo "  Average bulk range time: ${bulk_time}ms"

# Target: < 200ms for bulk operations
if (( $(echo "$bulk_time > 200" | bc -l) )); then
    echo "  ⚠️  Bulk range time exceeds 200ms target"
else
    echo "  ✅ Bulk range performance meets target"
fi

testing::phase::end_with_summary "Performance tests completed"
