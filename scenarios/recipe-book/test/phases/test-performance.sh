#!/bin/bash
set -euo pipefail

# Recipe Book Performance Test Runner

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::log "Starting Recipe Book performance tests"

# Performance benchmark: Recipe creation
testing::phase::log "Benchmarking recipe creation..."
START_TIME=$(date +%s%N)
ITERATIONS=50
SUCCESS_COUNT=0

for i in $(seq 1 $ITERATIONS); do
    if curl -sf -X POST "http://localhost:${API_PORT:-3250}/api/v1/recipes" \
        -H "Content-Type: application/json" \
        -d "{
            \"title\": \"Perf Test Recipe $i\",
            \"ingredients\": [{\"name\": \"test\", \"amount\": 1, \"unit\": \"cup\"}],
            \"instructions\": [\"Step 1\"],
            \"prep_time\": 10,
            \"cook_time\": 20,
            \"servings\": 4,
            \"created_by\": \"perf-test-user\",
            \"visibility\": \"private\"
        }" > /dev/null 2>&1; then
        SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
    fi
done

END_TIME=$(date +%s%N)
DURATION=$(( (END_TIME - START_TIME) / 1000000 ))
AVG_TIME=$(( DURATION / ITERATIONS ))

testing::phase::log "Created $SUCCESS_COUNT/$ITERATIONS recipes in ${DURATION}ms (avg: ${AVG_TIME}ms per recipe)"

if [ $AVG_TIME -lt 500 ]; then
    testing::phase::log "✓ Recipe creation performance: Excellent (< 500ms)"
elif [ $AVG_TIME -lt 1000 ]; then
    testing::phase::log "⚠ Recipe creation performance: Acceptable (< 1000ms)"
else
    testing::phase::log "✗ Recipe creation performance: Slow (> 1000ms)"
fi

# Performance benchmark: Recipe listing
testing::phase::log "Benchmarking recipe listing..."
START_TIME=$(date +%s%N)

for i in $(seq 1 20); do
    curl -sf "http://localhost:${API_PORT:-3250}/api/v1/recipes?limit=20" > /dev/null 2>&1 || true
done

END_TIME=$(date +%s%N)
DURATION=$(( (END_TIME - START_TIME) / 1000000 ))
AVG_TIME=$(( DURATION / 20 ))

testing::phase::log "Listed recipes 20 times in ${DURATION}ms (avg: ${AVG_TIME}ms per request)"

if [ $AVG_TIME -lt 200 ]; then
    testing::phase::log "✓ Recipe listing performance: Excellent (< 200ms)"
elif [ $AVG_TIME -lt 500 ]; then
    testing::phase::log "⚠ Recipe listing performance: Acceptable (< 500ms)"
else
    testing::phase::log "✗ Recipe listing performance: Slow (> 500ms)"
fi

# Cleanup performance test data
testing::phase::log "Cleaning up performance test data..."
if command -v psql &> /dev/null && [ -n "${POSTGRES_DB:-}" ]; then
    PGPASSWORD="${POSTGRES_PASSWORD:-postgres}" psql -h "${POSTGRES_HOST:-localhost}" \
        -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-recipe_book}" \
        -c "DELETE FROM recipes WHERE created_by = 'perf-test-user';" > /dev/null 2>&1 || true
fi

testing::phase::end_with_summary "Recipe Book performance tests completed"
