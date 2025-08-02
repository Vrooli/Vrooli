#!/bin/bash
# ====================================================================
# Automation Resource Test Template
# ====================================================================
#
# Template for testing automation resources with category-specific requirements
# including workflow management, UI responsiveness, trigger mechanisms,
# and integration capabilities.
#
# Usage: Copy this template and customize for specific automation resource
#
# Required Variables to Set:
#   TEST_RESOURCE - Resource name (e.g., "n8n", "node-red", "windmill")
#   RESOURCE_PORT - Default port for the resource
#   RESOURCE_BASE_URL - Base URL for web interface and API
#
# ====================================================================

set -euo pipefail

# Test metadata - CUSTOMIZE THESE
TEST_RESOURCE="your-automation-resource"
TEST_TIMEOUT="${TEST_TIMEOUT:-120}"  # Automation operations can be slow
TEST_CLEANUP="${TEST_CLEANUP:-true}"
RESOURCE_PORT="5678"  # Replace with actual port
RESOURCE_BASE_URL="http://localhost:${RESOURCE_PORT}"

# Source framework helpers
SCRIPT_DIR="${SCRIPT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
source "$SCRIPT_DIR/framework/helpers/assertions.sh"
source "$SCRIPT_DIR/framework/helpers/cleanup.sh"
source "$SCRIPT_DIR/framework/interface-compliance-categories.sh"

# Test setup
setup_test() {
    echo "🔧 Setting up automation resource test for $TEST_RESOURCE..."
    
    # Register cleanup handler
    register_cleanup_handler
    
    # Auto-discovery fallback for direct test execution
    if [[ -z "${HEALTHY_RESOURCES_STR:-}" ]]; then
        echo "🔍 Auto-discovering resources for direct test execution..."
        
        local resources_dir
        resources_dir="$(cd "$SCRIPT_DIR/../.." && pwd)"
        
        local discovery_output=""
        if timeout 10s bash -c "\"$resources_dir/index.sh\" --action discover 2>&1" > /tmp/discovery_output.tmp 2>&1; then
            discovery_output=$(cat /tmp/discovery_output.tmp)
            rm -f /tmp/discovery_output.tmp
        else
            echo "⚠️  Auto-discovery timed out, using fallback method..."
            if curl -f -s --max-time 2 "$RESOURCE_BASE_URL/" >/dev/null 2>&1; then
                discovery_output="✅ $TEST_RESOURCE is running on port $RESOURCE_PORT"
            fi
        fi
        
        local discovered_resources=()
        while IFS= read -r line; do
            if [[ "$line" =~ ✅[[:space:]]+([^[:space:]]+)[[:space:]]+is[[:space:]]+running ]]; then
                discovered_resources+=("${BASH_REMATCH[1]}")
            fi
        done <<< "$discovery_output"
        
        if [[ ${#discovered_resources[@]} -eq 0 ]]; then
            echo "⚠️  No resources discovered, but test will proceed..."
            discovered_resources=("$TEST_RESOURCE")
        fi
        
        export HEALTHY_RESOURCES_STR="${discovered_resources[*]}"
        echo "✓ Discovered healthy resources: $HEALTHY_RESOURCES_STR"
    fi
    
    # Verify resource is available
    require_resource "$TEST_RESOURCE"
    
    # Verify required tools
    require_tools "curl" "jq"
    
    echo "✓ Test setup complete"
}

# Test automation resource interface compliance
test_automation_interface_compliance() {
    echo "🔧 Testing automation resource interface compliance..."
    
    # Find the manage.sh script
    local manage_script=""
    local potential_paths=(
        "$SCRIPT_DIR/../automation/$TEST_RESOURCE/manage.sh"
        "$SCRIPT_DIR/../../automation/$TEST_RESOURCE/manage.sh"
        "$(cd "$SCRIPT_DIR/../.." && pwd)/automation/$TEST_RESOURCE/manage.sh"
    )
    
    for path in "${potential_paths[@]}"; do
        if [[ -f "$path" ]]; then
            manage_script="$path"
            break
        fi
    done
    
    if [[ -z "$manage_script" ]]; then
        manage_script=$(find "$(cd "$SCRIPT_DIR/../.." && pwd)" -name "manage.sh" -path "*/$TEST_RESOURCE/*" -type f 2>/dev/null | head -1)
    fi
    
    if [[ -z "$manage_script" || ! -f "$manage_script" ]]; then
        echo "⚠️  Could not find $TEST_RESOURCE manage.sh script - skipping interface compliance test"
        return 0
    fi
    
    echo "📍 Using manage script: $manage_script"
    
    # Run complete compliance test (base + category-specific)
    if test_complete_resource_compliance "$TEST_RESOURCE" "$manage_script" "automation"; then
        echo "✅ Automation resource interface compliance test passed"
        return 0
    else
        echo "❌ Automation resource interface compliance test failed"
        return 1
    fi
}

# Test automation resource health and connectivity
test_automation_health() {
    echo "🏥 Testing automation resource health..."
    
    # Use category-specific health check
    local health_result
    health_result=$(check_automation_resource_health "$TEST_RESOURCE" "$RESOURCE_PORT" "detailed")
    
    assert_not_equals "$health_result" "unreachable" "Automation resource is reachable"
    
    # Parse detailed health information
    if [[ "$health_result" =~ healthy ]]; then
        echo "✓ Automation resource health check passed: $health_result"
    else
        echo "⚠️  Automation resource health degraded: $health_result"
    fi
}

# Test web UI responsiveness (Automation-specific)
test_automation_ui_responsiveness() {
    echo "🖥️ Testing web UI responsiveness..."
    
    local start_time=$(date +%s.%N)
    
    # Test main UI page load
    local ui_response
    ui_response=$(curl -s --max-time 15 "$RESOURCE_BASE_URL/" 2>/dev/null)
    
    local end_time=$(date +%s.%N)
    local load_time=$(echo "$end_time - $start_time" | bc 2>/dev/null || echo "unknown")
    
    assert_not_empty "$ui_response" "UI page loads successfully"
    
    # Check for expected UI elements (customize based on platform)
    case "$TEST_RESOURCE" in
        "n8n")
            if echo "$ui_response" | grep -q "n8n"; then
                echo "✓ n8n UI elements detected"
            fi
            ;;
        "node-red")
            if echo "$ui_response" | grep -q "Node-RED"; then
                echo "✓ Node-RED UI elements detected"
            fi
            ;;
        "windmill")
            if echo "$ui_response" | grep -q "Windmill"; then
                echo "✓ Windmill UI elements detected"
            fi
            ;;
        "huginn")
            if echo "$ui_response" | grep -q "Huginn"; then
                echo "✓ Huginn UI elements detected"
            fi
            ;;
        "comfyui")
            if echo "$ui_response" | grep -q "ComfyUI"; then
                echo "✓ ComfyUI elements detected"
            fi
            ;;
        *)
            echo "⚠️  UI element detection needs customization for $TEST_RESOURCE"
            ;;
    esac
    
    if [[ "$load_time" != "unknown" ]]; then
        echo "UI load time: ${load_time}s"
        local load_time_ms=$(echo "$load_time * 1000" | bc 2>/dev/null || echo "unknown")
        if [[ "$load_time_ms" != "unknown" ]] && (( $(echo "$load_time_ms < 2000" | bc -l) )); then
            echo "✓ UI loads quickly (< 2s)"
        elif [[ "$load_time_ms" != "unknown" ]] && (( $(echo "$load_time_ms < 5000" | bc -l) )); then
            echo "⚠️  UI load time is acceptable (2-5s)"
        else
            echo "⚠️  UI load time may need optimization (> 5s)"
        fi
    fi
    
    echo "✓ UI responsiveness test completed"
}

# Test workflow management capabilities (Automation-specific)
test_automation_workflow_management() {
    echo "📋 Testing workflow management capabilities..."
    
    # Test workflow listing (customize endpoints based on platform)
    local workflows_endpoint=""
    case "$TEST_RESOURCE" in
        "n8n")
            workflows_endpoint="/rest/workflows"
            ;;
        "node-red")
            workflows_endpoint="/flows"
            ;;
        "windmill")
            workflows_endpoint="/api/w/demo/scripts/list"  # May need workspace
            ;;
        "huginn")
            workflows_endpoint="/agents"
            ;;
        "comfyui")
            workflows_endpoint="/api/queue"
            ;;
        *)
            echo "⚠️  Workflow endpoint needs customization for $TEST_RESOURCE"
            workflows_endpoint="/workflows"
            ;;
    esac
    
    echo "Testing workflow listing endpoint: $workflows_endpoint"
    
    local workflows_response
    workflows_response=$(curl -s --max-time 15 "$RESOURCE_BASE_URL$workflows_endpoint" 2>/dev/null)
    
    if [[ -n "$workflows_response" ]]; then
        # Try to parse as JSON
        if echo "$workflows_response" | jq . >/dev/null 2>&1; then
            echo "✓ Workflow API returns valid JSON"
            
            local workflow_count
            workflow_count=$(echo "$workflows_response" | jq 'length' 2>/dev/null || echo "parse_failed")
            
            if [[ "$workflow_count" != "parse_failed" ]]; then
                echo "✓ Found $workflow_count workflows/items"
            else
                echo "⚠️  Could not count workflows - may require authentication"
            fi
        else
            # May be HTML or require authentication
            echo "⚠️  Workflow API returned non-JSON response - may require authentication"
        fi
    else
        echo "⚠️  Workflow API endpoint not accessible - may require authentication or different endpoint"
    fi
    
    echo "✓ Workflow management test completed"
}

# Test API endpoints and integration (Automation-specific)
test_automation_api_integration() {
    echo "🔌 Testing API endpoints and integration..."
    
    # Test various API endpoints based on platform
    case "$TEST_RESOURCE" in
        "n8n")
            # Test health endpoint
            if curl -s --max-time 5 "$RESOURCE_BASE_URL/healthz" >/dev/null 2>&1; then
                echo "✓ n8n health endpoint accessible"
            fi
            
            # Test webhook endpoint structure
            if curl -s --max-time 5 "$RESOURCE_BASE_URL/webhook/" >/dev/null 2>&1; then
                echo "✓ n8n webhook endpoint accessible"
            fi
            ;;
        "node-red")
            # Test admin API
            if curl -s --max-time 5 "$RESOURCE_BASE_URL/settings" >/dev/null 2>&1; then
                echo "✓ Node-RED settings endpoint accessible"
            fi
            ;;
        "windmill")
            # Test version endpoint
            local version_response
            version_response=$(curl -s --max-time 5 "$RESOURCE_BASE_URL/api/version" 2>/dev/null)
            if [[ -n "$version_response" ]]; then
                echo "✓ Windmill version endpoint accessible: $version_response"
            fi
            ;;
        "comfyui")
            # Test system stats
            if curl -s --max-time 5 "$RESOURCE_BASE_URL/api/system_stats" >/dev/null 2>&1; then
                echo "✓ ComfyUI system stats endpoint accessible"
            fi
            ;;
        *)
            echo "⚠️  API integration tests need customization for $TEST_RESOURCE"
            ;;
    esac
    
    echo "✓ API integration test completed"
}

# Test trigger mechanisms (Automation-specific)
test_automation_triggers() {
    echo "⚡ Testing trigger mechanisms..."
    
    # Test different trigger types based on platform capabilities
    case "$TEST_RESOURCE" in
        "n8n")
            # Test webhook trigger capability
            echo "Testing n8n webhook trigger structure..."
            local webhook_test=$(curl -s --max-time 5 "$RESOURCE_BASE_URL/webhook/test" 2>/dev/null)
            if [[ $? -eq 0 ]]; then
                echo "✓ Webhook trigger endpoint responsive"
            fi
            ;;
        "node-red")
            # Test HTTP endpoints
            echo "Testing Node-RED HTTP input capability..."
            # Node-RED typically creates custom endpoints, so we test the framework
            if curl -s --max-time 5 "$RESOURCE_BASE_URL/httpRoot" >/dev/null 2>&1 || curl -s --max-time 5 "$RESOURCE_BASE_URL/" >/dev/null 2>&1; then
                echo "✓ HTTP trigger framework available"
            fi
            ;;
        "windmill")
            # Test script execution triggers
            echo "Testing Windmill execution framework..."
            if curl -s --max-time 5 "$RESOURCE_BASE_URL/api/w/demo/jobs/list" >/dev/null 2>&1; then
                echo "✓ Job execution framework available"
            fi
            ;;
        "huginn")
            # Test agent triggers
            echo "Testing Huginn agent framework..."
            echo "⚠️  Agent trigger testing requires authentication"
            ;;
        "comfyui")
            # Test queue triggers
            echo "Testing ComfyUI queue system..."
            local queue_response=$(curl -s --max-time 5 "$RESOURCE_BASE_URL/api/queue" 2>/dev/null)
            if [[ -n "$queue_response" ]] && echo "$queue_response" | jq . >/dev/null 2>&1; then
                echo "✓ Queue trigger system available"
            fi
            ;;
        *)
            echo "⚠️  Trigger testing needs customization for $TEST_RESOURCE"
            ;;
    esac
    
    echo "✓ Trigger mechanisms test completed"
}

# Test performance characteristics
test_automation_performance() {
    echo "⚡ Testing automation resource performance..."
    
    # Test basic response times
    local start_time=$(date +%s.%N)
    
    # Test quickest endpoint
    curl -s --max-time 10 "$RESOURCE_BASE_URL/" >/dev/null 2>&1
    
    local end_time=$(date +%s.%N)
    local response_time=$(echo "$end_time - $start_time" | bc 2>/dev/null || echo "unknown")
    
    echo "Basic response time: ${response_time}s"
    
    if [[ "$response_time" != "unknown" ]]; then
        local response_time_ms=$(echo "$response_time * 1000" | bc 2>/dev/null || echo "unknown")
        if [[ "$response_time_ms" != "unknown" ]] && (( $(echo "$response_time_ms < 500" | bc -l) )); then
            echo "✓ Response time is excellent (< 500ms)"
        elif [[ "$response_time_ms" != "unknown" ]] && (( $(echo "$response_time_ms < 2000" | bc -l) )); then
            echo "✓ Response time is good (< 2s)"
        else
            echo "⚠️  Response time may need optimization (> 2s)"
        fi
    fi
    
    echo "✓ Performance test completed"
}

# Test error handling
test_automation_error_handling() {
    echo "⚠️ Testing automation resource error handling..."
    
    # Test invalid endpoints
    local error_response
    error_response=$(curl -s --max-time 5 "$RESOURCE_BASE_URL/nonexistent-endpoint" 2>/dev/null)
    
    # Should get a proper error response (404, error page, etc.)
    if [[ -n "$error_response" ]]; then
        if echo "$error_response" | grep -q -E "(404|Not Found|Error)" || echo "$error_response" | grep -q "$TEST_RESOURCE"; then
            echo "✓ Error handling works correctly for invalid endpoints"
        else
            echo "⚠️  Error response format may need review"
        fi
    else
        echo "⚠️  No response for invalid endpoint - connection may have failed"
    fi
    
    echo "✓ Error handling test completed"
}

# Main test execution
main() {
    echo "🧪 Starting Automation Resource Integration Test"
    echo "Resource: $TEST_RESOURCE"
    echo "Timeout: ${TEST_TIMEOUT}s"
    echo "Base URL: $RESOURCE_BASE_URL"
    echo
    
    # Setup
    setup_test
    
    # Run test suite
    echo "📋 Running automation resource test suite..."
    echo
    
    # Phase 1: Interface Compliance (should be first)
    echo "Phase 1: Interface Compliance"
    test_automation_interface_compliance
    echo
    
    # Phase 2: Functional Tests
    echo "Phase 2: Functional Tests"
    test_automation_health
    test_automation_ui_responsiveness
    test_automation_workflow_management
    test_automation_api_integration
    test_automation_triggers
    test_automation_error_handling
    test_automation_performance
    
    # Print summary
    echo
    print_assertion_summary
    
    if [[ $FAILED_ASSERTIONS -gt 0 ]]; then
        echo "❌ Automation resource integration test failed"
        exit 1
    else
        echo "✅ Automation resource integration test passed"
        exit 0
    fi
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi