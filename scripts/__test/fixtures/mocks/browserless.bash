#!/usr/bin/env bash
# Browserless Resource Mock Implementation
# Provides realistic mock responses for Browserless browser automation service

# Prevent duplicate loading
if [[ "${BROWSERLESS_MOCK_LOADED:-}" == "true" ]]; then
    return 0
fi
export BROWSERLESS_MOCK_LOADED="true"

#######################################
# Setup Browserless mock environment
# Arguments: $1 - state (healthy, unhealthy, installing, stopped)
#######################################
mock::browserless::setup() {
    local state="${1:-healthy}"
    
    # Configure Browserless-specific environment
    export BROWSERLESS_PORT="${BROWSERLESS_PORT:-3000}"
    export BROWSERLESS_BASE_URL="http://localhost:${BROWSERLESS_PORT}"
    export BROWSERLESS_CONTAINER_NAME="${TEST_NAMESPACE}_browserless"
    export BROWSERLESS_TOKEN="${BROWSERLESS_TOKEN:-test-token}"
    
    # Set up Docker mock state
    mock::docker::set_container_state "$BROWSERLESS_CONTAINER_NAME" "$state"
    
    # Configure HTTP endpoints based on state
    case "$state" in
        "healthy")
            mock::browserless::setup_healthy_endpoints
            ;;
        "unhealthy")
            mock::browserless::setup_unhealthy_endpoints
            ;;
        "installing")
            mock::browserless::setup_installing_endpoints
            ;;
        "stopped")
            mock::browserless::setup_stopped_endpoints
            ;;
        *)
            echo "[BROWSERLESS_MOCK] Unknown state: $state" >&2
            return 1
            ;;
    esac
    
    echo "[BROWSERLESS_MOCK] Browserless mock configured with state: $state"
}

#######################################
# Setup healthy Browserless endpoints
#######################################
mock::browserless::setup_healthy_endpoints() {
    # Health endpoint
    mock::http::set_endpoint_response "$BROWSERLESS_BASE_URL/health" \
        '{"status":"ok","version":"2.11.0","chromeVersion":"119.0.6045.105"}'
    
    # Screenshot endpoint
    mock::http::set_endpoint_response "$BROWSERLESS_BASE_URL/screenshot" \
        'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg==' \
        "POST"
    
    # PDF endpoint
    mock::http::set_endpoint_response "$BROWSERLESS_BASE_URL/pdf" \
        '%PDF-1.4\n1 0 obj\n<<\n/Type /Catalog\n/Pages 2 0 R\n>>\nendobj\n2 0 obj\n<<\n/Type /Pages\n/Kids [3 0 R]\n/Count 1\n>>\nendobj\n3 0 obj\n<<\n/Type /Page\n/Parent 2 0 R\n/MediaBox [0 0 612 792]\n>>\nendobj\nxref\n0 4\n0000000000 65535 f\n0000000009 00000 n\n0000000074 00000 n\n0000000120 00000 n\ntrailer\n<<\n/Size 4\n/Root 1 0 R\n>>\nstartxref\n178\n%%EOF' \
        "POST"
    
    # Content endpoint
    mock::http::set_endpoint_response "$BROWSERLESS_BASE_URL/content" \
        '{"content":"<html><head><title>Test Page</title></head><body><h1>Hello World</h1></body></html>"}' \
        "POST"
    
    # Stats endpoint
    mock::http::set_endpoint_response "$BROWSERLESS_BASE_URL/stats" \
        '{
            "date": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",
            "requests": 1000,
            "queued": 0,
            "concurrent": 2,
            "maxConcurrent": 10,
            "maxQueued": 100,
            "cpu": 25.5,
            "memory": 512.3
        }'
    
    # Sessions endpoint
    mock::http::set_endpoint_response "$BROWSERLESS_BASE_URL/sessions" \
        '[
            {
                "id": "session-123",
                "startTime": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",
                "timeout": 30000,
                "userAgent": "Chrome/119.0.6045.105"
            }
        ]'
}

#######################################
# Setup unhealthy Browserless endpoints
#######################################
mock::browserless::setup_unhealthy_endpoints() {
    # Health endpoint returns error
    mock::http::set_endpoint_response "$BROWSERLESS_BASE_URL/health" \
        '{"status":"error","error":"Chrome process crashed"}' \
        "GET" \
        "503"
    
    # Screenshot endpoint returns error
    mock::http::set_endpoint_response "$BROWSERLESS_BASE_URL/screenshot" \
        '{"error":"Service temporarily unavailable"}' \
        "POST" \
        "503"
}

#######################################
# Setup installing Browserless endpoints
#######################################
mock::browserless::setup_installing_endpoints() {
    # Health endpoint returns installing status
    mock::http::set_endpoint_response "$BROWSERLESS_BASE_URL/health" \
        '{"status":"starting","message":"Chrome is starting up"}'
    
    # Other endpoints return not ready
    mock::http::set_endpoint_response "$BROWSERLESS_BASE_URL/screenshot" \
        '{"error":"Browserless is still starting up"}' \
        "POST" \
        "503"
}

#######################################
# Setup stopped Browserless endpoints
#######################################
mock::browserless::setup_stopped_endpoints() {
    # All endpoints fail to connect
    mock::http::set_endpoint_unreachable "$BROWSERLESS_BASE_URL"
}

#######################################
# Mock Browserless-specific operations
#######################################

# Mock page scraping
mock::browserless::scrape_page() {
    local url="$1"
    
    echo '{
        "url": "'$url'",
        "title": "Mock Page Title",
        "content": "<html><body>Mock page content</body></html>",
        "metadata": {
            "description": "Mock page description",
            "keywords": ["mock", "test", "page"]
        }
    }'
}

# Mock function execution
mock::browserless::execute_function() {
    local function_code="$1"
    
    echo '{
        "data": {
            "result": "Function executed successfully",
            "logs": ["Function started", "Function completed"],
            "errors": []
        }
    }'
}

#######################################
# Export mock functions
#######################################
export -f mock::browserless::setup
export -f mock::browserless::setup_healthy_endpoints
export -f mock::browserless::setup_unhealthy_endpoints
export -f mock::browserless::setup_installing_endpoints
export -f mock::browserless::setup_stopped_endpoints
export -f mock::browserless::scrape_page
export -f mock::browserless::execute_function

echo "[BROWSERLESS_MOCK] Browserless mock implementation loaded"