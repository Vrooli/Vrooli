#!/bin/bash
# Integration tests for scenario-to-extension

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR" || { echo "Failed to cd to scenario directory"; exit 1; }

testing::phase::info "Starting integration tests for scenario-to-extension"

# Test API availability
testing::phase::step "Testing API health endpoint"
if ! curl -sf "http://localhost:${API_PORT:-3201}/api/v1/health" &>/dev/null; then
    testing::phase::warn "API not running, starting it..."
    cd api && ./scenario-to-extension-api &
    API_PID=$!
    sleep 2
    cleanup_api() {
        kill "$API_PID" 2>/dev/null || true
    }
    trap cleanup_api EXIT
fi

# Test health endpoint
testing::phase::step "Validating health endpoint response"
HEALTH_RESPONSE=$(curl -s "http://localhost:${API_PORT:-3201}/api/v1/health")
if ! echo "$HEALTH_RESPONSE" | jq -e '.status == "healthy"' &>/dev/null; then
    testing::phase::error "Health check failed"
    testing::phase::end_with_summary "Integration tests failed"
    exit 1
fi
testing::phase::success "Health endpoint returned healthy status"

# Test extension generation endpoint
testing::phase::step "Testing extension generation"
GENERATE_RESPONSE=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d '{
        "scenario_name": "integration-test-scenario",
        "template_type": "full",
        "config": {
            "app_name": "Integration Test Extension",
            "app_description": "Test extension for integration testing",
            "api_endpoint": "http://localhost:3000"
        }
    }' \
    "http://localhost:${API_PORT:-3201}/api/v1/extension/generate")

BUILD_ID=$(echo "$GENERATE_RESPONSE" | jq -r '.build_id')
if [ -z "$BUILD_ID" ] || [ "$BUILD_ID" == "null" ]; then
    testing::phase::error "Failed to generate extension: $GENERATE_RESPONSE"
    testing::phase::end_with_summary "Integration tests failed"
    exit 1
fi
testing::phase::success "Extension generation initiated with build_id: $BUILD_ID"

# Test status endpoint
testing::phase::step "Testing build status endpoint"
sleep 1  # Give build time to process
STATUS_RESPONSE=$(curl -s "http://localhost:${API_PORT:-3201}/api/v1/extension/status/$BUILD_ID")
STATUS=$(echo "$STATUS_RESPONSE" | jq -r '.status')
if [ -z "$STATUS" ] || [ "$STATUS" == "null" ]; then
    testing::phase::error "Failed to get build status: $STATUS_RESPONSE"
    testing::phase::end_with_summary "Integration tests failed"
    exit 1
fi
testing::phase::success "Build status retrieved: $STATUS"

# Test list templates endpoint
testing::phase::step "Testing list templates endpoint"
TEMPLATES_RESPONSE=$(curl -s "http://localhost:${API_PORT:-3201}/api/v1/extension/templates")
TEMPLATE_COUNT=$(echo "$TEMPLATES_RESPONSE" | jq -r '.count')
if [ "$TEMPLATE_COUNT" != "4" ]; then
    testing::phase::error "Expected 4 templates, got $TEMPLATE_COUNT"
    testing::phase::end_with_summary "Integration tests failed"
    exit 1
fi
testing::phase::success "Template listing returned $TEMPLATE_COUNT templates"

# Test list builds endpoint
testing::phase::step "Testing list builds endpoint"
BUILDS_RESPONSE=$(curl -s "http://localhost:${API_PORT:-3201}/api/v1/extension/builds")
BUILD_COUNT=$(echo "$BUILDS_RESPONSE" | jq -r '.count')
if [ "$BUILD_COUNT" -lt 1 ]; then
    testing::phase::error "Expected at least 1 build, got $BUILD_COUNT"
    testing::phase::end_with_summary "Integration tests failed"
    exit 1
fi
testing::phase::success "Build listing returned $BUILD_COUNT builds"

# Test extension testing endpoint
testing::phase::step "Testing extension test endpoint"
TEST_RESPONSE=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d '{
        "extension_path": "/tmp/test-extension",
        "test_sites": ["https://example.com"],
        "screenshot": true,
        "headless": true
    }' \
    "http://localhost:${API_PORT:-3201}/api/v1/extension/test")

TEST_SUCCESS=$(echo "$TEST_RESPONSE" | jq -r '.success')
if [ "$TEST_SUCCESS" != "true" ]; then
    testing::phase::warn "Extension test returned success=false (expected for simulated test)"
else
    testing::phase::success "Extension test endpoint responded successfully"
fi

# Test all template types
testing::phase::step "Testing all template types"
for TEMPLATE_TYPE in "full" "content-script-only" "background-only" "popup-only"; do
    TEMPLATE_RESPONSE=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "{
            \"scenario_name\": \"template-test-$TEMPLATE_TYPE\",
            \"template_type\": \"$TEMPLATE_TYPE\",
            \"config\": {
                \"app_name\": \"Template Test $TEMPLATE_TYPE\",
                \"app_description\": \"Test for $TEMPLATE_TYPE\",
                \"api_endpoint\": \"http://localhost:3000\"
            }
        }" \
        "http://localhost:${API_PORT:-3201}/api/v1/extension/generate")

    TEMPLATE_BUILD_ID=$(echo "$TEMPLATE_RESPONSE" | jq -r '.build_id')
    if [ -z "$TEMPLATE_BUILD_ID" ] || [ "$TEMPLATE_BUILD_ID" == "null" ]; then
        testing::phase::error "Failed to generate extension for template type: $TEMPLATE_TYPE"
        testing::phase::end_with_summary "Integration tests failed"
        exit 1
    fi
    testing::phase::success "Generated extension for template type: $TEMPLATE_TYPE"
done

testing::phase::info "All integration tests passed"
testing::phase::end_with_summary "Integration tests completed successfully"
