#!/usr/bin/env bats
# Tests for RESOURCE_NAME lib/api.sh API interaction functions
#
# Template Usage:
# 1. Copy this file to RESOURCE_NAME/lib/api.bats
# 2. Replace RESOURCE_NAME with your resource name (e.g., "ollama", "n8n")
# 3. Replace API_ENDPOINTS with your actual API endpoints
# 4. Implement resource-specific API tests
# 5. Remove this header comment block

bats_require_minimum_version 1.5.0

# Setup for each test
setup() {
    # Load shared test infrastructure
    source "$(dirname "${BATS_TEST_FILENAME}")/../../../tests/bats-fixtures/common_setup.bash"
    
    # Setup standard mocks
    setup_standard_mocks
    
    # Set test environment
    export RESOURCE_NAME_PORT="8080"  # Replace with actual port
    export RESOURCE_NAME_BASE_URL="http://localhost:8080"
    export RESOURCE_NAME_API_VERSION="v1"  # Replace with actual API version
    export RESOURCE_NAME_API_KEY=""  # Add if authentication required
    export RESOURCE_NAME_TIMEOUT="30"
    
    # Get resource directory path
    RESOURCE_NAME_DIR="$(dirname "$(dirname "${BATS_TEST_FILENAME}")")"
    
    # Configure healthy API responses by default
    mock::http::set_endpoint_response "http://localhost:8080/health" "200" '{"status":"healthy","version":"1.0.0"}'
    mock::http::set_endpoint_response "http://localhost:8080/api/v1/info" "200" '{"name":"RESOURCE_NAME","version":"1.0.0"}'
    
    # Load configuration and the functions to test
    source "${RESOURCE_NAME_DIR}/config/defaults.sh"
    source "${RESOURCE_NAME_DIR}/config/messages.sh"
    RESOURCE_NAME::export_config
    RESOURCE_NAME::messages::init
    
    # Load the API functions
    source "${RESOURCE_NAME_DIR}/lib/api.sh"
}

teardown() {
    # Clean up test environment
    cleanup_mocks
}

# Test API connectivity

@test "RESOURCE_NAME::api_request should make basic API calls" {
    run RESOURCE_NAME::api_request "GET" "/health"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "healthy" ]]
}

@test "RESOURCE_NAME::api_request should handle different HTTP methods" {
    # Test GET
    run RESOURCE_NAME::api_request "GET" "/health"
    [ "$status" -eq 0 ]
    
    # Test POST
    mock::http::set_endpoint_response "http://localhost:8080/api/v1/create" "201" '{"id":"123","status":"created"}'
    run RESOURCE_NAME::api_request "POST" "/api/v1/create" '{"name":"test"}'
    [ "$status" -eq 0 ]
    [[ "$output" =~ "created" ]]
    
    # Test PUT
    mock::http::set_endpoint_response "http://localhost:8080/api/v1/update/123" "200" '{"id":"123","status":"updated"}'
    run RESOURCE_NAME::api_request "PUT" "/api/v1/update/123" '{"name":"updated"}'
    [ "$status" -eq 0 ]
    [[ "$output" =~ "updated" ]]
    
    # Test DELETE
    mock::http::set_endpoint_response "http://localhost:8080/api/v1/delete/123" "204" ""
    run RESOURCE_NAME::api_request "DELETE" "/api/v1/delete/123"
    [ "$status" -eq 0 ]
}

@test "RESOURCE_NAME::api_request should handle authentication" {
    export RESOURCE_NAME_API_KEY="test-api-key"
    
    # Mock authenticated endpoint
    mock::http::set_endpoint_response "http://localhost:8080/api/v1/secure" "200" '{"authorized":true}'
    
    run RESOURCE_NAME::api_request "GET" "/api/v1/secure"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "authorized" ]]
    
    # Verify auth header was sent (check mock logs)
    grep -q "Authorization: Bearer test-api-key" "$MOCK_RESPONSES_DIR/http_requests.log" || \
    grep -q "X-API-Key: test-api-key" "$MOCK_RESPONSES_DIR/http_requests.log"
}

@test "RESOURCE_NAME::api_request should handle connection failures" {
    # Mock connection failure
    mock::http::set_endpoint_response "http://localhost:8080/api/v1/test" "connection_refused" ""
    
    run RESOURCE_NAME::api_request "GET" "/api/v1/test"
    [ "$status" -ne 0 ]
    [[ "$output" =~ "connection" || "$output" =~ "failed" ]]
}

@test "RESOURCE_NAME::api_request should handle timeouts" {
    # Mock slow response
    mock::http::set_endpoint_delay "http://localhost:8080/api/v1/slow" 35
    export RESOURCE_NAME_TIMEOUT="5"
    
    run RESOURCE_NAME::api_request "GET" "/api/v1/slow"
    [ "$status" -ne 0 ]
    [[ "$output" =~ "timeout" ]]
}

# Test API response handling

@test "RESOURCE_NAME::api_request should parse JSON responses" {
    mock::http::set_endpoint_response "http://localhost:8080/api/v1/data" "200" '{"items":[{"id":1,"name":"test"}],"count":1}'
    
    run RESOURCE_NAME::api_request "GET" "/api/v1/data"
    [ "$status" -eq 0 ]
    
    # Should be valid JSON
    echo "$output" | jq . >/dev/null
}

@test "RESOURCE_NAME::api_request should handle invalid JSON" {
    mock::http::set_endpoint_response "http://localhost:8080/api/v1/broken" "200" "invalid json {"
    
    run RESOURCE_NAME::api_request "GET" "/api/v1/broken"
    [ "$status" -ne 0 ]
    [[ "$output" =~ "json" || "$output" =~ "parse" ]]
}

@test "RESOURCE_NAME::api_request should handle HTTP error codes" {
    # Test 400 Bad Request
    mock::http::set_endpoint_response "http://localhost:8080/api/v1/bad" "400" '{"error":"Bad Request","message":"Invalid parameters"}'
    run RESOURCE_NAME::api_request "GET" "/api/v1/bad"
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Bad Request" || "$output" =~ "400" ]]
    
    # Test 401 Unauthorized
    mock::http::set_endpoint_response "http://localhost:8080/api/v1/auth" "401" '{"error":"Unauthorized"}'
    run RESOURCE_NAME::api_request "GET" "/api/v1/auth"
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Unauthorized" || "$output" =~ "401" ]]
    
    # Test 404 Not Found
    mock::http::set_endpoint_response "http://localhost:8080/api/v1/missing" "404" '{"error":"Not Found"}'
    run RESOURCE_NAME::api_request "GET" "/api/v1/missing"
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Not Found" || "$output" =~ "404" ]]
    
    # Test 500 Server Error
    mock::http::set_endpoint_response "http://localhost:8080/api/v1/error" "500" '{"error":"Internal Server Error"}'
    run RESOURCE_NAME::api_request "GET" "/api/v1/error"
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Server Error" || "$output" =~ "500" ]]
}

# Test API helper functions

@test "RESOURCE_NAME::get_api_info should return service information" {
    run RESOURCE_NAME::get_api_info
    [ "$status" -eq 0 ]
    [[ "$output" =~ "RESOURCE_NAME" ]]
    [[ "$output" =~ "version" ]]
}

@test "RESOURCE_NAME::check_api_health should verify API availability" {
    run RESOURCE_NAME::check_api_health
    [ "$status" -eq 0 ]
    [[ "$output" =~ "healthy" ]]
}

@test "RESOURCE_NAME::check_api_health should detect unhealthy API" {
    mock::http::set_endpoint_response "http://localhost:8080/health" "503" '{"status":"unhealthy","errors":["database_down"]}'
    
    run RESOURCE_NAME::check_api_health
    [ "$status" -ne 0 ]
    [[ "$output" =~ "unhealthy" ]]
}

@test "RESOURCE_NAME::get_api_version should return version information" {
    mock::http::set_endpoint_response "http://localhost:8080/version" "200" '{"version":"1.2.3","api_version":"v1"}'
    
    run RESOURCE_NAME::get_api_version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "1.2.3" ]]
}

# Test resource-specific API endpoints (customize based on your resource)

@test "RESOURCE_NAME specific API endpoints" {
    # Example for AI resources:
    # @test "RESOURCE_NAME::list_models should return available models"
    # mock::http::set_endpoint_response "http://localhost:8080/api/v1/models" "200" '{"models":["model1","model2"]}'
    # run RESOURCE_NAME::list_models
    # [ "$status" -eq 0 ]
    # [[ "$output" =~ "model1" ]]
    
    # Example for automation resources:
    # @test "RESOURCE_NAME::list_workflows should return workflows"
    # mock::http::set_endpoint_response "http://localhost:8080/api/v1/workflows" "200" '{"workflows":[{"id":"wf1","name":"Test Workflow"}]}'
    # run RESOURCE_NAME::list_workflows
    # [ "$status" -eq 0 ]
    # [[ "$output" =~ "Test Workflow" ]]
    
    # Example for storage resources:
    # @test "RESOURCE_NAME::list_buckets should return storage buckets"
    # mock::http::set_endpoint_response "http://localhost:8080/api/v1/buckets" "200" '{"buckets":["bucket1","bucket2"]}'
    # run RESOURCE_NAME::list_buckets
    # [ "$status" -eq 0 ]
    # [[ "$output" =~ "bucket1" ]]
    
    # Replace this with actual resource-specific tests
    skip "Implement resource-specific API endpoint tests"
}

# Test API data operations

@test "RESOURCE_NAME::create_item should create new items via API" {
    mock::http::set_endpoint_response "http://localhost:8080/api/v1/items" "201" '{"id":"item123","name":"test item","status":"created"}'
    
    run RESOURCE_NAME::create_item "test item"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "item123" ]]
    [[ "$output" =~ "created" ]]
}

@test "RESOURCE_NAME::get_item should retrieve items by ID" {
    mock::http::set_endpoint_response "http://localhost:8080/api/v1/items/item123" "200" '{"id":"item123","name":"test item","status":"active"}'
    
    run RESOURCE_NAME::get_item "item123"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "test item" ]]
    [[ "$output" =~ "active" ]]
}

@test "RESOURCE_NAME::get_item should handle missing items" {
    mock::http::set_endpoint_response "http://localhost:8080/api/v1/items/missing" "404" '{"error":"Item not found"}'
    
    run RESOURCE_NAME::get_item "missing"
    [ "$status" -ne 0 ]
    [[ "$output" =~ "not found" ]]
}

@test "RESOURCE_NAME::update_item should modify existing items" {
    mock::http::set_endpoint_response "http://localhost:8080/api/v1/items/item123" "200" '{"id":"item123","name":"updated item","status":"updated"}'
    
    run RESOURCE_NAME::update_item "item123" "updated item"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "updated" ]]
}

@test "RESOURCE_NAME::delete_item should remove items" {
    mock::http::set_endpoint_response "http://localhost:8080/api/v1/items/item123" "204" ""
    
    run RESOURCE_NAME::delete_item "item123"
    [ "$status" -eq 0 ]
}

@test "RESOURCE_NAME::list_items should return paginated results" {
    mock::http::set_endpoint_response "http://localhost:8080/api/v1/items?page=1&limit=10" "200" '{"items":[{"id":"1"},{"id":"2"}],"pagination":{"page":1,"total":25}}'
    
    run RESOURCE_NAME::list_items 1 10
    [ "$status" -eq 0 ]
    [[ "$output" =~ "pagination" ]]
    [[ "$output" =~ "total.*25" ]]
}

# Test API query operations

@test "RESOURCE_NAME::search should support query parameters" {
    mock::http::set_endpoint_response "http://localhost:8080/api/v1/search?q=test&filter=active" "200" '{"results":[{"id":"1","name":"test item"}],"count":1}'
    
    run RESOURCE_NAME::search "test" "active"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "test item" ]]
    [[ "$output" =~ "count.*1" ]]
}

@test "RESOURCE_NAME::filter_items should apply filters" {
    mock::http::set_endpoint_response "http://localhost:8080/api/v1/items?status=active&type=important" "200" '{"items":[{"id":"1","status":"active","type":"important"}]}'
    
    run RESOURCE_NAME::filter_items "status=active" "type=important"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "active" ]]
    [[ "$output" =~ "important" ]]
}

# Test API streaming and uploads

@test "RESOURCE_NAME::upload_file should handle file uploads" {
    # Create test file
    echo "test content" > "/tmp/test-upload.txt"
    
    mock::http::set_endpoint_response "http://localhost:8080/api/v1/upload" "200" '{"file_id":"file123","status":"uploaded"}'
    
    run RESOURCE_NAME::upload_file "/tmp/test-upload.txt"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "file123" ]]
    [[ "$output" =~ "uploaded" ]]
    
    # Cleanup
    rm -f "/tmp/test-upload.txt"
}

@test "RESOURCE_NAME::download_file should handle file downloads" {
    # Mock file download
    mock::http::set_endpoint_response "http://localhost:8080/api/v1/files/file123" "200" "file content here"
    
    run RESOURCE_NAME::download_file "file123" "/tmp/downloaded.txt"
    [ "$status" -eq 0 ]
    [ -f "/tmp/downloaded.txt" ]
    
    # Verify content
    grep -q "file content here" "/tmp/downloaded.txt"
    
    # Cleanup
    rm -f "/tmp/downloaded.txt"
}

@test "RESOURCE_NAME::stream_data should handle streaming responses" {
    # Mock streaming endpoint
    mock::http::set_endpoint_stream "http://localhost:8080/api/v1/stream" "data chunk 1\ndata chunk 2\ndata chunk 3"
    
    run RESOURCE_NAME::stream_data
    [ "$status" -eq 0 ]
    [[ "$output" =~ "chunk 1" ]]
    [[ "$output" =~ "chunk 2" ]]
    [[ "$output" =~ "chunk 3" ]]
}

# Test API rate limiting and retries

@test "RESOURCE_NAME::api_request should handle rate limiting" {
    # Mock rate limit response
    mock::http::set_endpoint_response "http://localhost:8080/api/v1/limited" "429" '{"error":"Rate limit exceeded","retry_after":2}'
    
    run RESOURCE_NAME::api_request "GET" "/api/v1/limited"
    [ "$status" -ne 0 ]
    [[ "$output" =~ "rate limit" || "$output" =~ "429" ]]
}

@test "RESOURCE_NAME::api_request should retry failed requests" {
    # Mock sequence: fail, fail, success
    mock::http::set_endpoint_sequence "http://localhost:8080/api/v1/retry" "503,503,200" "Service Unavailable,Service Unavailable,{\"status\":\"success\"}"
    
    run RESOURCE_NAME::api_request "GET" "/api/v1/retry" "" 3
    [ "$status" -eq 0 ]
    [[ "$output" =~ "success" ]]
}

@test "RESOURCE_NAME::api_request should respect retry limits" {
    # Mock persistent failure
    mock::http::set_endpoint_response "http://localhost:8080/api/v1/persistent_fail" "503" "Service Unavailable"
    
    run RESOURCE_NAME::api_request "GET" "/api/v1/persistent_fail" "" 2
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Service Unavailable" ]]
}

# Test API validation

@test "RESOURCE_NAME::validate_api_response should check response format" {
    # Valid response
    run RESOURCE_NAME::validate_api_response '{"status":"success","data":{"id":1}}'
    [ "$status" -eq 0 ]
    
    # Invalid response
    run RESOURCE_NAME::validate_api_response 'invalid json'
    [ "$status" -ne 0 ]
    
    # Missing required fields
    run RESOURCE_NAME::validate_api_response '{"data":{}}'  # Missing status
    [ "$status" -ne 0 ]
}

@test "RESOURCE_NAME::check_api_compatibility should verify API version" {
    # Compatible version
    mock::http::set_endpoint_response "http://localhost:8080/version" "200" '{"api_version":"v1"}'
    run RESOURCE_NAME::check_api_compatibility
    [ "$status" -eq 0 ]
    
    # Incompatible version
    mock::http::set_endpoint_response "http://localhost:8080/version" "200" '{"api_version":"v2"}'
    run RESOURCE_NAME::check_api_compatibility
    [ "$status" -ne 0 ]
    [[ "$output" =~ "incompatible" || "$output" =~ "version" ]]
}

# Add more resource-specific API tests here:

# Example templates for different resource types:

# For AI resources:
# @test "RESOURCE_NAME::generate should handle AI inference requests" { ... }
# @test "RESOURCE_NAME::train_model should handle training requests" { ... }

# For automation resources:
# @test "RESOURCE_NAME::execute_workflow should run workflows" { ... }
# @test "RESOURCE_NAME::schedule_task should create scheduled tasks" { ... }

# For storage resources:
# @test "RESOURCE_NAME::create_bucket should create storage containers" { ... }
# @test "RESOURCE_NAME::set_permissions should handle access control" { ... }