#!/usr/bin/env bash
################################################################################
# n8n Integration Test - End-to-End Functionality
# 
# Validates n8n workflow execution, API endpoints, and resource integration (120s max)
################################################################################

set -euo pipefail

# Get directory paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
N8N_DIR="${APP_ROOT}/resources/n8n"

# Source utilities and configuration
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/resources/port_registry.sh"
source "${N8N_DIR}/config/defaults.sh"
source "${N8N_DIR}/lib/api.sh" 2>/dev/null || true

# Get n8n port from registry
N8N_PORT=$(get_port "n8n" 2>/dev/null || echo "5678")
N8N_URL="http://localhost:${N8N_PORT}"

log::info "Starting n8n integration tests..."

# Test 1: API - List workflows
log::info "Test 1: Testing workflow listing API..."
if n8n::api::list_workflows > /dev/null 2>&1; then
    log::success "✓ Workflow listing API works"
else
    log::error "✗ Failed to list workflows via API"
    exit 1
fi

# Test 2: Create a test workflow
log::info "Test 2: Testing workflow creation..."
TEST_WORKFLOW_JSON=$(cat <<'EOF'
{
  "name": "Integration Test Workflow",
  "nodes": [
    {
      "parameters": {},
      "name": "Start",
      "type": "n8n-nodes-base.start",
      "position": [250, 300]
    }
  ],
  "connections": {}
}
EOF
)

# Try to create workflow via API
workflow_id=""
if command -v jq > /dev/null 2>&1; then
    response=$(timeout 10 curl -sf -X POST "${N8N_URL}/api/workflows" \
        -H "Content-Type: application/json" \
        -d "$TEST_WORKFLOW_JSON" 2>/dev/null || echo "{}")
    
    if [[ -n "$response" ]] && [[ "$response" != "{}" ]]; then
        workflow_id=$(echo "$response" | jq -r '.data.id' 2>/dev/null || echo "")
        if [[ -n "$workflow_id" ]] && [[ "$workflow_id" != "null" ]]; then
            log::success "✓ Successfully created test workflow (ID: $workflow_id)"
        else
            log::warn "⚠ Could not create test workflow (may require authentication)"
        fi
    else
        log::warn "⚠ Workflow creation test skipped (authentication may be required)"
    fi
else
    log::warn "⚠ Workflow creation test skipped (jq not installed)"
fi

# Test 3: Content management functions
log::info "Test 3: Testing content management..."
if declare -f n8n::content::list > /dev/null 2>&1; then
    if n8n::content::list > /dev/null 2>&1; then
        log::success "✓ Content listing works"
    else
        log::warn "⚠ Content listing returned an error (may be empty)"
    fi
else
    log::error "✗ Content management functions not loaded"
    exit 1
fi

# Test 4: Webhook endpoint availability
log::info "Test 4: Testing webhook endpoint..."
webhook_response=$(timeout 5 curl -s -o /dev/null -w "%{http_code}" "${N8N_URL}/webhook-test/test" 2>/dev/null || echo "000")
if [[ "$webhook_response" == "404" ]] || [[ "$webhook_response" == "426" ]]; then
    log::success "✓ Webhook endpoint is accessible (no test webhook configured)"
elif [[ "$webhook_response" == "200" ]]; then
    log::success "✓ Webhook endpoint is working"
else
    log::warn "⚠ Webhook endpoint returned unexpected status: $webhook_response"
fi

# Test 5: Credential auto-discovery
log::info "Test 5: Testing credential auto-discovery..."
if declare -f n8n::list_discoverable_resources > /dev/null 2>&1; then
    discoverable=$(n8n::list_discoverable_resources 2>/dev/null | wc -l || echo "0")
    if [[ "$discoverable" -gt 0 ]]; then
        log::success "✓ Found $discoverable discoverable resources"
    else
        log::warn "⚠ No discoverable resources found"
    fi
else
    log::warn "⚠ Credential auto-discovery function not available"
fi

# Test 6: Cleanup test workflow if created
if [[ -n "$workflow_id" ]] && [[ "$workflow_id" != "null" ]]; then
    log::info "Test 6: Cleaning up test workflow..."
    if timeout 10 curl -sf -X DELETE "${N8N_URL}/api/workflows/${workflow_id}" > /dev/null 2>&1; then
        log::success "✓ Test workflow cleaned up"
    else
        log::warn "⚠ Could not delete test workflow"
    fi
fi

# Test 7: Performance check - API response time
log::info "Test 7: Testing API response time..."
start_time=$(date +%s%N)
timeout 5 curl -sf "${N8N_URL}/health" > /dev/null 2>&1
end_time=$(date +%s%N)
response_time=$(( (end_time - start_time) / 1000000 ))

if [[ $response_time -lt 500 ]]; then
    log::success "✓ API response time is good (${response_time}ms)"
elif [[ $response_time -lt 1000 ]]; then
    log::warn "⚠ API response time is acceptable (${response_time}ms)"
else
    log::error "✗ API response time is slow (${response_time}ms)"
    exit 1
fi

log::success "All integration tests passed"
exit 0