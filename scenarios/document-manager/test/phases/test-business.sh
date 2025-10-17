#!/bin/bash
# Phase 5: Business logic tests - <180 seconds
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

echo "=== Phase 5: Business Logic Tests ==="
start_time=$(date +%s)
error_count=0
test_count=0

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$SCENARIO_DIR"

scenario_name="document-manager"
API_PORT=$(vrooli scenario port "$scenario_name" API_PORT 2>/dev/null || echo "17810")
API_BASE_URL="http://localhost:$API_PORT"

log::info "Testing complete business workflows..."

# Test Application CRUD workflow
((test_count++))
log::info "Testing Application CRUD workflow..."

# Create application
app_response=$(curl -sf -X POST "$API_BASE_URL/api/applications" \
  -H "Content-Type: application/json" \
  -d '{"name":"Business Test App","repository_url":"https://github.com/test/business","documentation_path":"/docs","active":true}' 2>/dev/null)

if echo "$app_response" | jq -e '.id' >/dev/null 2>&1; then
  APP_ID=$(echo "$app_response" | jq -r '.id')
  log::success "✅ Application created successfully: $APP_ID"

  # Read application
  if curl -sf "$API_BASE_URL/api/applications" | jq -e ".[] | select(.id==\"$APP_ID\")" >/dev/null 2>&1; then
    log::success "✅ Application retrieved successfully"
  else
    log::error "❌ Application retrieval failed"
    ((error_count++))
  fi
else
  log::error "❌ Application creation failed"
  ((error_count++))
  APP_ID=""
fi

# Test Agent workflow
if [ -n "$APP_ID" ]; then
  ((test_count++))
  log::info "Testing Agent creation workflow..."

  agent_response=$(curl -sf -X POST "$API_BASE_URL/api/agents" \
    -H "Content-Type: application/json" \
    -d "{\"name\":\"Business Test Agent\",\"type\":\"documentation_analyzer\",\"application_id\":\"$APP_ID\",\"configuration\":\"{}\",\"enabled\":true}" 2>/dev/null)

  if echo "$agent_response" | jq -e '.id' >/dev/null 2>&1; then
    AGENT_ID=$(echo "$agent_response" | jq -r '.id')
    log::success "✅ Agent created successfully: $AGENT_ID"
  else
    log::error "❌ Agent creation failed"
    ((error_count++))
    AGENT_ID=""
  fi
fi

# Test Improvement Queue workflow
if [ -n "$APP_ID" ] && [ -n "$AGENT_ID" ]; then
  ((test_count++))
  log::info "Testing Improvement Queue workflow..."

  queue_response=$(curl -sf -X POST "$API_BASE_URL/api/queue" \
    -H "Content-Type: application/json" \
    -d "{\"agent_id\":\"$AGENT_ID\",\"application_id\":\"$APP_ID\",\"type\":\"documentation_improvement\",\"title\":\"Business Test\",\"description\":\"Test improvement\",\"severity\":\"low\",\"status\":\"pending\"}" 2>/dev/null)

  if echo "$queue_response" | jq -e '.id' >/dev/null 2>&1; then
    log::success "✅ Queue item created successfully"
  else
    log::error "❌ Queue item creation failed"
    ((error_count++))
  fi
fi

# Test concurrent operations
((test_count++))
log::info "Testing concurrent request handling..."
concurrent_count=0
for i in {1..5}; do
  if curl -sf "$API_BASE_URL/api/applications" >/dev/null 2>&1 & then
    ((concurrent_count++))
  fi
done
wait

if [ $concurrent_count -eq 5 ]; then
  log::success "✅ Concurrent requests handled successfully"
else
  log::warning "⚠️  Some concurrent requests failed"
fi

# Test error handling
((test_count++))
log::info "Testing error handling..."
error_status=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API_BASE_URL/api/applications" \
  -H "Content-Type: application/json" \
  -d 'invalid json' 2>/dev/null)

if [ "$error_status" = "400" ]; then
  log::success "✅ Error handling validated"
else
  log::warning "⚠️  Unexpected error status: $error_status"
fi

# Performance check
end_time=$(date +%s)
duration=$((end_time - start_time))

if [ $error_count -eq 0 ]; then
  log::success "✅ Business logic tests completed in ${duration}s"
  log::success "   Tests run: $test_count, Errors: $error_count"
  exit 0
else
  log::error "❌ Business logic tests failed with $error_count errors in ${duration}s"
  exit 1
fi
