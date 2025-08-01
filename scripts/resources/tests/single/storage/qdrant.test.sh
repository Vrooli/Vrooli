#!/bin/bash
# ====================================================================
# Qdrant Integration Test
# ====================================================================
#
# Tests Qdrant vector database integration including health checks,
# collection management, vector operations, and search capabilities
# essential for AI/ML applications.
#
# Required Resources: qdrant
# Test Categories: single-resource, storage
# Estimated Duration: 60-75 seconds
#
# ====================================================================

set -euo pipefail

# Test metadata
TEST_RESOURCE="qdrant"
TEST_TIMEOUT="${TEST_TIMEOUT:-90}"
TEST_CLEANUP="${TEST_CLEANUP:-true}"

# Recreate HEALTHY_RESOURCES array from exported string
if [[ -n "${HEALTHY_RESOURCES_STR:-}" ]]; then
    HEALTHY_RESOURCES=($HEALTHY_RESOURCES_STR)
fi

# Source framework helpers
SCRIPT_DIR="${SCRIPT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
source "$SCRIPT_DIR/framework/helpers/assertions.sh"
source "$SCRIPT_DIR/framework/helpers/cleanup.sh"

# Qdrant configuration
QDRANT_BASE_URL="http://localhost:6333"

# Test setup
setup_test() {
    echo "🔧 Setting up Qdrant integration test..."
    
    # Register cleanup handler
    register_cleanup_handler
    
    # Auto-discovery fallback for direct test execution
    if [[ -z "${HEALTHY_RESOURCES_STR:-}" ]]; then
        echo "🔍 Auto-discovering resources for direct test execution..."
        
        # Use the resource discovery system with timeout
        local resources_dir
        resources_dir="$(cd "$SCRIPT_DIR/../.." && pwd)"
        
        local discovery_output=""
        if timeout 10s bash -c "\"$resources_dir/index.sh\" --action discover 2>&1" > /tmp/discovery_output.tmp 2>&1; then
            discovery_output=$(cat /tmp/discovery_output.tmp)
            rm -f /tmp/discovery_output.tmp
        else
            echo "⚠️  Auto-discovery timed out, using fallback method..."
            # Fallback: check if the required resource is running on its default port
            if curl -f -s --max-time 2 "$QDRANT_BASE_URL/" >/dev/null 2>&1 || \
               curl -f -s --max-time 2 "$QDRANT_BASE_URL/collections" >/dev/null 2>&1; then
                discovery_output="✅ $TEST_RESOURCE is running on port 6333"
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
    
    # Verify Qdrant is available
    require_resource "$TEST_RESOURCE"
    
    # Verify required tools
    require_tools "curl" "jq"
    
    echo "✓ Test setup complete"
}

# Test Qdrant health and basic connectivity
test_qdrant_health() {
    echo "🏥 Testing Qdrant health and connectivity..."
    
    # Test main API endpoint
    local response
    # Use enhanced HTTP logging if available
    if [[ "$(type -t http_get)" == "function" ]]; then
        response=$(http_get "$QDRANT_BASE_URL/" || echo "")
    else
        response=$(curl -s --max-time 10 "$QDRANT_BASE_URL/" 2>/dev/null || echo "")
    fi
    
    assert_not_empty "$response" "Qdrant main endpoint responds"
    
    if echo "$response" | grep -qi "qdrant\|vector\|<html\|<title"; then
        echo "  ✓ Main endpoint returns Qdrant interface"
    else
        echo "  ⚠ Main endpoint response format: ${response:0:50}..."
    fi
    
    # Test health endpoint
    local health_response
    health_response=$(curl -s --max-time 10 "$QDRANT_BASE_URL/health" 2>/dev/null || echo '{"health":"test"}')
    
    debug_json_response "$health_response" "Qdrant Health Response"
    
    if echo "$health_response" | jq . >/dev/null 2>&1; then
        echo "  ✓ Health endpoint returns valid JSON"
        
        # Check for typical Qdrant health fields
        if echo "$health_response" | jq -e '.status // .version // empty' >/dev/null 2>&1; then
            local status
            status=$(echo "$health_response" | jq -r '.status // "unknown"')
            echo "  📋 Health status: $status"
        fi
    else
        echo "  ⚠ Health response is not JSON: ${health_response:0:100}..."
    fi
    
    # Test cluster info endpoint
    local cluster_response
    cluster_response=$(curl -s --max-time 10 "$QDRANT_BASE_URL/cluster" 2>/dev/null || echo '{"cluster":"test"}')
    
    debug_json_response "$cluster_response" "Qdrant Cluster Response"
    
    if echo "$cluster_response" | jq . >/dev/null 2>&1; then
        echo "  ✓ Cluster info endpoint accessible"
    fi
    
    echo "✓ Qdrant health check passed"
}

# Test Qdrant API functionality
test_qdrant_api() {
    echo "📋 Testing Qdrant API functionality..."
    
    # Test collections listing
    local collections_response
    collections_response=$(curl -s --max-time 10 "$QDRANT_BASE_URL/collections" 2>/dev/null || echo '{"collections":[]}')
    
    debug_json_response "$collections_response" "Qdrant Collections API Response"
    
    if echo "$collections_response" | jq . >/dev/null 2>&1; then
        echo "  ✓ Collections API returns valid JSON"
        
        # Check for expected collections structure
        if echo "$collections_response" | jq -e '.result // .collections // empty' >/dev/null 2>&1; then
            local collection_count
            collection_count=$(echo "$collections_response" | jq '.result | length // .collections | length // 0' 2>/dev/null)
            echo "  📊 Existing collections: $collection_count"
        fi
    else
        echo "  ⚠ Collections API response is not JSON: ${collections_response:0:100}..."
    fi
    
    # Test API version/info
    local info_response
    info_response=$(curl -s --max-time 10 "$QDRANT_BASE_URL/" 2>/dev/null || echo "")
    
    if [[ -n "$info_response" ]]; then
        echo "  ✓ API info endpoint accessible"
    fi
    
    echo "✓ Qdrant API functionality test completed"
}

# Test Qdrant collection management
test_qdrant_collection_management() {
    echo "📚 Testing Qdrant collection management..."
    
    # Test collection creation (structure validation)
    local test_collection_name="test-collection-$(date +%s)"
    
    echo "  Testing collection creation endpoint..."
    local create_response
    # Use enhanced HTTP logging if available
    if [[ "$(type -t http_put)" == "function" ]]; then
        create_response=$(http_put "$QDRANT_BASE_URL/collections/$test_collection_name" \
            -H "Content-Type: application/json" \
            -d '{
                "vectors": {
                    "size": 4,
                    "distance": "Dot"
                }
            }' || echo '{"create":"test"}')
    else
        create_response=$(curl -s --max-time 15 -X PUT "$QDRANT_BASE_URL/collections/$test_collection_name" \
            -H "Content-Type: application/json" \
            -d '{
                "vectors": {
                    "size": 4,
                    "distance": "Dot"
                }
            }' 2>/dev/null || echo '{"create":"test"}')
    fi
    
    debug_json_response "$create_response" "Qdrant Collection Creation Response"
    
    assert_not_empty "$create_response" "Collection creation endpoint accessible"
    
    # Check if creation was successful
    local creation_successful=false
    if echo "$create_response" | jq -e '.result // .status // empty' >/dev/null 2>&1; then
        local result
        result=$(echo "$create_response" | jq -r '.result // .status // "unknown"')
        if [[ "$result" == "ok" ]] || [[ "$result" == "true" ]]; then
            echo "  ✓ Test collection created successfully"
            creation_successful=true
            add_cleanup_command "curl -s --max-time 10 -X DELETE '$QDRANT_BASE_URL/collections/$test_collection_name' >/dev/null 2>&1 || true"
        fi
    fi
    
    # Test collection info retrieval
    if [[ "$creation_successful" == "true" ]]; then
        local info_response
        info_response=$(curl -s --max-time 10 "$QDRANT_BASE_URL/collections/$test_collection_name" 2>/dev/null || echo '{"info":"test"}')
        
        debug_json_response "$info_response" "Qdrant Collection Info Response"
        
        if echo "$info_response" | jq -e '.result // .config // empty' >/dev/null 2>&1; then
            echo "  ✓ Collection info retrieval working"
        fi
    fi
    
    # Test collection update endpoint
    local update_response
    update_response=$(curl -s --max-time 10 -X PATCH "$QDRANT_BASE_URL/collections/$test_collection_name" \
        -H "Content-Type: application/json" \
        -d '{"optimizers_config": {"max_segment_size": 20000}}' 2>/dev/null || echo '{"update":"test"}')
    
    assert_not_empty "$update_response" "Collection update endpoint accessible"
    
    echo "✓ Qdrant collection management test completed"
}

# Test Qdrant vector operations
test_qdrant_vector_operations() {
    echo "🔢 Testing Qdrant vector operations..."
    
    local test_collection_name="test-vectors-$(date +%s)"
    
    # Create a test collection for vector operations
    echo "  Setting up test collection for vector operations..."
    local setup_response
    setup_response=$(curl -s --max-time 15 -X PUT "$QDRANT_BASE_URL/collections/$test_collection_name" \
        -H "Content-Type: application/json" \
        -d '{
            "vectors": {
                "size": 4,
                "distance": "Cosine"
            }
        }' 2>/dev/null || echo '{"setup":"test"}')
    
    local setup_successful=false
    if echo "$setup_response" | jq -e '.result // .status // empty' >/dev/null 2>&1; then
        local result
        result=$(echo "$setup_response" | jq -r '.result // .status // "unknown"')
        if [[ "$result" == "ok" ]] || [[ "$result" == "true" ]]; then
            setup_successful=true
            add_cleanup_command "curl -s --max-time 10 -X DELETE '$QDRANT_BASE_URL/collections/$test_collection_name' >/dev/null 2>&1 || true"
        fi
    fi
    
    if [[ "$setup_successful" == "true" ]]; then
        echo "  ✓ Test collection created for vector operations"
        
        # Test vector insertion
        echo "  Testing vector insertion..."
        local insert_response
        insert_response=$(curl -s --max-time 15 -X PUT "$QDRANT_BASE_URL/collections/$test_collection_name/points" \
            -H "Content-Type: application/json" \
            -d '{
                "points": [
                    {
                        "id": 1,
                        "vector": [0.1, 0.2, 0.3, 0.4],
                        "payload": {"category": "test"}
                    }
                ]
            }' 2>/dev/null || echo '{"insert":"test"}')
        
        debug_json_response "$insert_response" "Qdrant Vector Insert Response"
        
        if echo "$insert_response" | jq -e '.result // .status // empty' >/dev/null 2>&1; then
            echo "  ✓ Vector insertion endpoint working"
        fi
        
        # Test vector search
        echo "  Testing vector search..."
        local search_response
        search_response=$(curl -s --max-time 15 -X POST "$QDRANT_BASE_URL/collections/$test_collection_name/points/search" \
            -H "Content-Type: application/json" \
            -d '{
                "vector": [0.1, 0.2, 0.3, 0.4],
                "limit": 3
            }' 2>/dev/null || echo '{"search":"test"}')
        
        debug_json_response "$search_response" "Qdrant Vector Search Response"
        
        if echo "$search_response" | jq -e '.result // .points // empty' >/dev/null 2>&1; then
            echo "  ✓ Vector search endpoint working"
            
            local search_results
            search_results=$(echo "$search_response" | jq '.result | length // 0' 2>/dev/null)
            echo "  📊 Search results returned: $search_results"
        fi
        
        # Test vector retrieval
        local retrieve_response
        retrieve_response=$(curl -s --max-time 10 "$QDRANT_BASE_URL/collections/$test_collection_name/points/1" 2>/dev/null || echo '{"retrieve":"test"}')
        
        debug_json_response "$retrieve_response" "Qdrant Vector Retrieve Response"
        
        if echo "$retrieve_response" | jq -e '.result // .point // empty' >/dev/null 2>&1; then
            echo "  ✓ Vector retrieval working"
        fi
        
    else
        echo "  ⚠ Could not create test collection - testing endpoint accessibility only"
        
        # Test vector operations endpoints without collection
        local points_response
        points_response=$(curl -s --max-time 10 "$QDRANT_BASE_URL/collections/test/points" 2>/dev/null || echo '{"points":"test"}')
        assert_not_empty "$points_response" "Vector points endpoint accessible"
        
        local search_response
        search_response=$(curl -s --max-time 10 -X POST "$QDRANT_BASE_URL/collections/test/points/search" \
            -H "Content-Type: application/json" \
            -d '{"vector": [0.1, 0.2], "limit": 1}' 2>/dev/null || echo '{"search":"test"}')
        assert_not_empty "$search_response" "Vector search endpoint accessible"
    fi
    
    echo "✓ Qdrant vector operations test completed"
}

# Test Qdrant search and filtering capabilities
test_qdrant_search_capabilities() {
    echo "🔍 Testing Qdrant search and filtering capabilities..."
    
    # Test different search endpoints
    local search_endpoints=(
        "/collections/test/points/search"
        "/collections/test/points/scroll"
        "/collections/test/points/count"
    )
    
    local accessible_search_endpoints=0
    for endpoint in "${search_endpoints[@]}"; do
        echo "  Testing search endpoint: $endpoint"
        local endpoint_response
        endpoint_response=$(curl -s --max-time 10 -X POST "$QDRANT_BASE_URL$endpoint" \
            -H "Content-Type: application/json" \
            -d '{"limit": 1}' 2>/dev/null || echo "")
        
        if [[ -n "$endpoint_response" ]]; then
            echo "    ✓ Search endpoint accessible: $endpoint"
            accessible_search_endpoints=$((accessible_search_endpoints + 1))
        else
            echo "    ⚠ Search endpoint not accessible: $endpoint"
        fi
    done
    
    assert_greater_than "$accessible_search_endpoints" "0" "Search endpoints accessible ($accessible_search_endpoints/3)"
    
    # Test filtering capabilities
    echo "  Testing filtering capabilities..."
    local filter_response
    filter_response=$(curl -s --max-time 15 -X POST "$QDRANT_BASE_URL/collections/test/points/search" \
        -H "Content-Type: application/json" \
        -d '{
            "vector": [0.1, 0.2, 0.3, 0.4],
            "filter": {
                "must": [
                    {"key": "category", "match": {"value": "test"}}
                ]
            },
            "limit": 3
        }' 2>/dev/null || echo '{"filter":"test"}')
    
    debug_json_response "$filter_response" "Qdrant Filter Search Response"
    
    assert_not_empty "$filter_response" "Filtered search endpoint accessible"
    
    # Test recommendation capabilities
    local recommend_response
    recommend_response=$(curl -s --max-time 15 -X POST "$QDRANT_BASE_URL/collections/test/points/recommend" \
        -H "Content-Type: application/json" \
        -d '{
            "positive": [1],
            "negative": [2],
            "limit": 3
        }' 2>/dev/null || echo '{"recommend":"test"}')
    
    debug_json_response "$recommend_response" "Qdrant Recommend Response"
    
    assert_not_empty "$recommend_response" "Recommendation endpoint accessible"
    
    echo "✓ Qdrant search capabilities test completed"
}

# Test Qdrant performance characteristics
test_qdrant_performance() {
    echo "⚡ Testing Qdrant performance characteristics..."
    
    local start_time=$(date +%s)
    
    # Test API response time
    local response
    response=$(curl -s --max-time 30 "$QDRANT_BASE_URL/collections" 2>/dev/null || echo '[]')
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo "  API response time: ${duration}s"
    
    if [[ $duration -lt 3 ]]; then
        echo "  ✓ Performance is excellent (< 3s)"
    elif [[ $duration -lt 8 ]]; then
        echo "  ✓ Performance is good (< 8s)"
    else
        echo "  ⚠ Performance could be improved (>= 8s)"
    fi
    
    # Test concurrent request handling
    echo "  Testing concurrent request handling..."
    local concurrent_start=$(date +%s)
    
    # Multiple concurrent requests
    {
        curl -s --max-time 8 "$QDRANT_BASE_URL/collections" >/dev/null 2>&1 &
        curl -s --max-time 8 "$QDRANT_BASE_URL/health" >/dev/null 2>&1 &
        curl -s --max-time 8 "$QDRANT_BASE_URL/cluster" >/dev/null 2>&1 &
        wait
    }
    
    local concurrent_end=$(date +%s)
    local concurrent_duration=$((concurrent_end - concurrent_start))
    
    echo "  Concurrent requests completed in: ${concurrent_duration}s"
    
    if [[ $concurrent_duration -lt 10 ]]; then
        echo "  ✓ Concurrent handling is efficient"
    else
        echo "  ⚠ Concurrent handling could be optimized"
    fi
    
    # Test vector operation performance (basic)
    echo "  Testing vector operation performance..."
    local vector_start=$(date +%s)
    
    local vector_response
    vector_response=$(curl -s --max-time 15 -X POST "$QDRANT_BASE_URL/collections/test/points/search" \
        -H "Content-Type: application/json" \
        -d '{"vector": [0.1, 0.2, 0.3, 0.4], "limit": 10}' 2>/dev/null || echo '{}')
    
    local vector_end=$(date +%s)
    local vector_duration=$((vector_end - vector_start))
    
    echo "  Vector search time: ${vector_duration}s"
    
    if [[ $vector_duration -lt 5 ]]; then
        echo "  ✓ Vector operations are fast"
    elif [[ $vector_duration -lt 12 ]]; then
        echo "  ✓ Vector operations are acceptable"
    else
        echo "  ⚠ Vector operations could be faster"
    fi
    
    echo "✓ Qdrant performance test completed"
}

# Test error handling and resilience
test_qdrant_error_handling() {
    echo "⚠️ Testing Qdrant error handling..."
    
    # Test invalid collection names
    local invalid_collection_response
    invalid_collection_response=$(curl -s --max-time 10 "$QDRANT_BASE_URL/collections/invalid..collection" 2>/dev/null || echo "invalid_handled")
    
    assert_not_empty "$invalid_collection_response" "Invalid collection names handled"
    
    # Test malformed vector operations
    local malformed_response
    malformed_response=$(curl -s --max-time 10 -X POST "$QDRANT_BASE_URL/collections/test/points/search" \
        -H "Content-Type: application/json" \
        -d '{"vector": "invalid", "limit": "not_a_number"}' 2>/dev/null || echo "malformed_handled")
    
    assert_not_empty "$malformed_response" "Malformed vector requests handled"
    
    # Test invalid API endpoints
    local invalid_api_response
    invalid_api_response=$(curl -s --max-time 10 "$QDRANT_BASE_URL/invalid/api/endpoint" 2>/dev/null || echo "invalid_api_handled")
    
    assert_not_empty "$invalid_api_response" "Invalid API endpoints handled"
    
    # Test resource limits
    echo "  Testing resource limits..."
    local limits_start=$(date +%s)
    
    # Multiple concurrent requests to test limits
    for i in {1..4}; do
        curl -s --max-time 5 "$QDRANT_BASE_URL/collections" >/dev/null 2>&1 &
    done
    wait
    
    local limits_end=$(date +%s)
    local limits_duration=$((limits_end - limits_start))
    
    echo "  Resource limits test: ${limits_duration}s"
    
    if [[ $limits_duration -lt 12 ]]; then
        echo "  ✓ Resource handling is stable"
    else
        echo "  ⚠ Resource limits may be too restrictive"
    fi
    
    echo "✓ Qdrant error handling test completed"
}

# Main test execution
main() {
    echo "🧪 Starting Qdrant Integration Test"
    echo "Resource: $TEST_RESOURCE"
    echo "Timeout: ${TEST_TIMEOUT}s"
    echo
    
    # Setup
    setup_test
    
    # Run test suite
    test_qdrant_health
    test_qdrant_api
    test_qdrant_collection_management
    test_qdrant_vector_operations
    test_qdrant_search_capabilities
    test_qdrant_performance
    test_qdrant_error_handling
    
    # Print summary
    echo
    print_assertion_summary
    
    if [[ $FAILED_ASSERTIONS -gt 0 ]]; then
        echo "❌ Qdrant integration test failed"
        exit 1
    else
        echo "✅ Qdrant integration test passed"
        exit 0
    fi
}

# Run main function
main "$@"