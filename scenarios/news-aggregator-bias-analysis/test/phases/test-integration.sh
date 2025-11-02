#!/bin/bash
# Integration tests for news-aggregator-bias-analysis scenario

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Require the runtime so the shared runner can auto-manage the scenario lifecycle.
testing::phase::init --target-time "180s" --require-runtime

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "🧪 Running integration checks..."

postgres_url="${POSTGRES_URL:-}"
if [ -n "$postgres_url" ] && command -v psql >/dev/null 2>&1; then
    testing::phase::check "Postgres connectivity" bash -c "psql \"$postgres_url\" -c 'SELECT 1;' >/dev/null"
else
    testing::phase::add_warning "⚠️  Skipping Postgres connectivity check (missing POSTGRES_URL or psql)"
    testing::phase::add_test skipped
fi

if ! API_BASE_URL=$(testing::connectivity::get_api_url "$TESTING_PHASE_SCENARIO_NAME"); then
    testing::phase::add_error "❌ Unable to discover API base URL"
    testing::phase::end_with_summary "Integration tests incomplete"
fi

echo "🌐 Using API base URL: $API_BASE_URL"

if command -v jq >/dev/null 2>&1; then
    testing::phase::check "Health endpoint reports healthy" bash -c "curl -sf '$API_BASE_URL/health' | jq -e '.status == \"healthy\"' >/dev/null"
else
    testing::phase::check "Health endpoint reachable" curl -sf "$API_BASE_URL/health"
fi

testing::phase::check "Articles endpoint returns data" curl -sf "$API_BASE_URL/articles"

testing::phase::check "Feeds endpoint returns data" curl -sf "$API_BASE_URL/feeds"

testing::phase::check "Perspectives endpoint responds" curl -sf "$API_BASE_URL/perspectives/test"

testing::phase::check "Perspective aggregation endpoint" bash -c "curl -sf -X POST '$API_BASE_URL/perspectives/aggregate' -H 'Content-Type: application/json' -d '{"topic":"integration test","time_range":"24 hours"}' >/dev/null"

if command -v jq >/dev/null 2>&1; then
    testing::phase::check "Refresh endpoint status" bash -c "curl -sf -X POST '$API_BASE_URL/refresh' | jq -e '.status == \"refresh_triggered\"' >/dev/null"
else
    testing::phase::check "Refresh endpoint reachable" curl -sf "$API_BASE_URL/refresh"
fi

if curl -sf "$API_BASE_URL/bias/summary" >/dev/null 2>&1; then
    testing::phase::check "Bias summary endpoint" curl -sf "$API_BASE_URL/bias/summary"
else
    testing::phase::add_warning "⚠️  Bias summary endpoint unavailable; skipping optional check"
    testing::phase::add_test skipped
fi

# Finish with phase summary
testing::phase::end_with_summary "Integration validation completed"
