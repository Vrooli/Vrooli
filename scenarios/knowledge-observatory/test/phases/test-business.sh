#!/bin/bash
# Business workflow validation for knowledge-observatory
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/connectivity.sh"

testing::phase::init --target-time "180s" --require-runtime

SCENARIO_NAME="$TESTING_PHASE_SCENARIO_NAME"
API_URL=$(testing::connectivity::get_api_url "$SCENARIO_NAME" || true)

if [ -z "$API_URL" ]; then
  testing::phase::add_error "Unable to discover API URL"
  testing::phase::end_with_summary "Business tests skipped (API unavailable)"
fi

if ! command -v jq >/dev/null 2>&1; then
  testing::phase::add_warning "jq not available; skipping payload structure assertions"
fi

# Health summary should include dependency statuses
health_payload=$(curl -sf "$API_URL/health" || true)
if [ -n "$health_payload" ]; then
  testing::phase::add_test passed
  log::success "Health payload retrieved"
  if command -v jq >/dev/null 2>&1; then
    if ! echo "$health_payload" | jq -e '.dependencies' >/dev/null 2>&1; then
      testing::phase::add_warning "Health response missing dependency block"
    fi
  fi
else
  testing::phase::add_test failed
  testing::phase::end_with_summary "Health endpoint unavailable"
fi

# Metrics API should report collection statistics
testing::phase::check "Metrics endpoint" curl -sf "$API_URL/api/v1/knowledge/metrics"

# Semantic search should return structured data
search_payload=$(curl -sf -X POST "$API_URL/api/v1/knowledge/search" \
  -H 'Content-Type: application/json' \
  -d '{"query":"vector health report","limit":5}')
if [ -n "$search_payload" ]; then
  testing::phase::add_test passed
  if command -v jq >/dev/null 2>&1; then
    if ! echo "$search_payload" | jq -e '.results' >/dev/null 2>&1; then
      testing::phase::add_warning "Search payload missing results array"
    fi
  fi
else
  testing::phase::add_test failed
  testing::phase::add_error "Search endpoint returned empty response"
fi

# Knowledge graph summary should expose nodes & edges
if command -v jq >/dev/null 2>&1; then
  testing::phase::check "Graph payload includes nodes" bash -c "curl -sf '$API_URL/api/v1/knowledge/graph?max_nodes=15' | jq -e '.nodes' >/dev/null"
  testing::phase::check "Graph payload includes edges" bash -c "curl -sf '$API_URL/api/v1/knowledge/graph?max_nodes=15' | jq -e '.edges' >/dev/null"
else
  testing::phase::check "Graph endpoint responds" curl -sf "$API_URL/api/v1/knowledge/graph?max_nodes=15"
fi

# CORS configuration should avoid wildcards
cors_headers=$(curl -sI -X OPTIONS "$API_URL/api/v1/knowledge/search" \
  -H 'Origin: http://localhost:3000' \
  -H 'Access-Control-Request-Method: POST' || true)
if echo "$cors_headers" | grep -qi "Access-Control-Allow-Origin"; then
  if echo "$cors_headers" | grep -q "Access-Control-Allow-Origin: \*"; then
    testing::phase::add_error "CORS header uses wildcard; tighten origin list"
  else
    testing::phase::add_test passed
    log::success "CORS configuration present without wildcard"
  fi
else
  testing::phase::add_warning "CORS headers missing from preflight response"
fi

# Business metric expectations: ensure seeded collections available via metrics endpoint
if [ -n "$health_payload" ] && command -v jq >/dev/null 2>&1; then
  collection_status=$(echo "$health_payload" | jq -r '.dependencies.postgres.status // empty' 2>/dev/null || true)
  if [ -z "$collection_status" ]; then
    testing::phase::add_warning "Health payload missing postgres dependency details"
  fi
fi


testing::phase::end_with_summary "Business validation completed"
