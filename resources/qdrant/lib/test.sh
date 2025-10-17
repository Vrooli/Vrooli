#!/usr/bin/env bash
# Qdrant Test Implementation - v2.0 Contract Compliant
# Tests the RESOURCE itself (health, connectivity, functions)
# NOT the business functionality (that's content commands)

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
QDRANT_LIB_DIR="${APP_ROOT}/resources/qdrant/lib"
QDRANT_CONFIG_DIR="${APP_ROOT}/resources/qdrant/config"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${QDRANT_CONFIG_DIR}/defaults.sh"
# Export Qdrant configuration
qdrant::export_config 2>/dev/null || true
# shellcheck disable=SC1091
source "${QDRANT_LIB_DIR}/api-client.sh"
# shellcheck disable=SC1091
source "${QDRANT_LIB_DIR}/search-optimizer.sh"

# Test result tracking
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0

#######################################
# Quick health check (<30s per v2.0)
# Returns: 0 if healthy, 1 if not
#######################################
qdrant::check_basic_health() {
    echo "=== Qdrant Smoke Test ==="
    echo
    
    local start_time
    start_time=$(date +%s)
    
    # Check container running
    echo -n "Checking container status... "
    if docker ps --format "{{.Names}}" | grep -q "^${QDRANT_CONTAINER_NAME}$"; then
        echo "✅ Running"
        ((TESTS_PASSED++))
    else
        echo "❌ Not running"
        ((TESTS_FAILED++))
        return 1
    fi
    
    # Check HTTP health endpoint
    echo -n "Checking HTTP health endpoint... "
    if timeout 5 curl -sf "http://localhost:${QDRANT_PORT}/" &>/dev/null; then
        echo "✅ Responsive"
        ((TESTS_PASSED++))
    else
        echo "❌ Not responding"
        ((TESTS_FAILED++))
        return 1
    fi
    
    # Check API version
    echo -n "Checking API version... "
    local version
    if version=$(timeout 5 curl -sf "http://localhost:${QDRANT_PORT}/" | jq -r '.version' 2>/dev/null); then
        echo "✅ Version: $version"
        ((TESTS_PASSED++))
    else
        echo "❌ Cannot get version"
        ((TESTS_FAILED++))
    fi
    
    # Check collections endpoint
    echo -n "Checking collections API... "
    if timeout 5 curl -sf "http://localhost:${QDRANT_PORT}/collections" &>/dev/null; then
        echo "✅ Accessible"
        ((TESTS_PASSED++))
    else
        echo "❌ Not accessible"
        ((TESTS_FAILED++))
    fi
    
    # Check telemetry endpoint
    echo -n "Checking telemetry endpoint... "
    if timeout 5 curl -sf "http://localhost:${QDRANT_PORT}/telemetry" &>/dev/null; then
        echo "✅ Available"
        ((TESTS_PASSED++))
    else
        echo "❌ Not available"
        ((TESTS_FAILED++))
    fi
    
    local end_time elapsed
    end_time=$(date +%s)
    elapsed=$((end_time - start_time))
    
    echo
    echo "Smoke test completed in ${elapsed}s"
    echo "Passed: $TESTS_PASSED | Failed: $TESTS_FAILED | Skipped: $TESTS_SKIPPED"
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo "✅ Smoke test PASSED"
        return 0
    else
        echo "❌ Smoke test FAILED"
        return 1
    fi
}

#######################################
# Integration tests (<120s per v2.0)
# Returns: 0 if all pass, 1 if any fail
#######################################
qdrant::test_integration() {
    echo "=== Qdrant Integration Tests ==="
    echo
    
    local start_time
    start_time=$(date +%s)
    
    # Test collection operations
    echo "Testing collection operations..."
    if qdrant::test_collection_operations; then
        ((TESTS_PASSED++))
    else
        ((TESTS_FAILED++))
    fi
    
    # Test vector operations
    echo "Testing vector operations..."
    if qdrant::test_vector_operations; then
        ((TESTS_PASSED++))
    else
        ((TESTS_FAILED++))
    fi
    
    # Test search performance
    echo "Testing search performance..."
    if qdrant::test_search_performance; then
        ((TESTS_PASSED++))
    else
        ((TESTS_FAILED++))
    fi
    
    # Test backup operations
    echo "Testing backup operations..."
    if qdrant::test_backup_operations; then
        ((TESTS_PASSED++))
    else
        ((TESTS_FAILED++))
    fi
    
    local end_time elapsed
    end_time=$(date +%s)
    elapsed=$((end_time - start_time))
    
    echo
    echo "Integration tests completed in ${elapsed}s"
    echo "Passed: $TESTS_PASSED | Failed: $TESTS_FAILED | Skipped: $TESTS_SKIPPED"
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo "✅ Integration tests PASSED"
        return 0
    else
        echo "❌ Integration tests FAILED"
        return 1
    fi
}

#######################################
# Unit tests for library functions (<60s)
# Returns: 0 if all pass, 1 if any fail
#######################################
qdrant::test_unit() {
    echo "=== Qdrant Unit Tests ==="
    echo
    
    local start_time
    start_time=$(date +%s)
    
    # Test API client functions
    echo "Testing API client..."
    if qdrant::test_api_client; then
        ((TESTS_PASSED++))
    else
        ((TESTS_FAILED++))
    fi
    
    # Test search optimizer functions
    echo "Testing search optimizer..."
    if qdrant::test_search_optimizer; then
        ((TESTS_PASSED++))
    else
        ((TESTS_FAILED++))
    fi
    
    # Test mathematical accuracy
    echo "Testing mathematical accuracy..."
    if qdrant::test_mathematical_accuracy; then
        ((TESTS_PASSED++))
    else
        ((TESTS_FAILED++))
    fi
    
    local end_time elapsed
    end_time=$(date +%s)
    elapsed=$((end_time - start_time))
    
    echo
    echo "Unit tests completed in ${elapsed}s"
    echo "Passed: $TESTS_PASSED | Failed: $TESTS_FAILED | Skipped: $TESTS_SKIPPED"
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo "✅ Unit tests PASSED"
        return 0
    else
        echo "❌ Unit tests FAILED"
        return 1
    fi
}

#######################################
# Test collection operations
#######################################
qdrant::test_collection_operations() {
    local test_collection="test-collection-$(date +%s)"
    
    # Create collection
    echo -n "  Creating test collection... "
    local create_payload='{
        "vectors": {
            "size": 128,
            "distance": "Cosine"
        }
    }'
    
    if qdrant::api::put "/collections/${test_collection}" "$create_payload" &>/dev/null; then
        echo "✅"
    else
        echo "❌"
        return 1
    fi
    
    # Check collection exists
    echo -n "  Verifying collection exists... "
    if qdrant::api::get "/collections/${test_collection}" &>/dev/null; then
        echo "✅"
    else
        echo "❌"
        return 1
    fi
    
    # Delete collection
    echo -n "  Deleting test collection... "
    if qdrant::api::delete "/collections/${test_collection}" &>/dev/null; then
        echo "✅"
    else
        echo "❌"
        return 1
    fi
    
    return 0
}

#######################################
# Test vector operations
#######################################
qdrant::test_vector_operations() {
    local test_collection="test-vectors-$(date +%s)"
    
    # Create collection
    echo -n "  Creating vector test collection... "
    local create_payload='{
        "vectors": {
            "size": 4,
            "distance": "Cosine"
        }
    }'
    
    if ! qdrant::api::put "/collections/${test_collection}" "$create_payload" &>/dev/null; then
        echo "❌"
        return 1
    fi
    echo "✅"
    
    # Insert vectors
    echo -n "  Inserting test vectors... "
    local vectors_payload='{
        "points": [
            {"id": 1, "vector": [0.1, 0.2, 0.3, 0.4], "payload": {"name": "test1"}},
            {"id": 2, "vector": [0.5, 0.6, 0.7, 0.8], "payload": {"name": "test2"}},
            {"id": 3, "vector": [0.9, 0.1, 0.2, 0.3], "payload": {"name": "test3"}}
        ]
    }'
    
    if qdrant::api::put "/collections/${test_collection}/points?wait=true" "$vectors_payload" &>/dev/null; then
        echo "✅"
    else
        echo "❌"
        qdrant::api::delete "/collections/${test_collection}" &>/dev/null
        return 1
    fi
    
    # Search vectors
    echo -n "  Searching vectors... "
    local search_payload='{
        "vector": [0.1, 0.2, 0.3, 0.4],
        "limit": 2,
        "with_payload": true
    }'
    
    local search_result
    if search_result=$(qdrant::api::post "/collections/${test_collection}/points/search" "$search_payload" 2>/dev/null); then
        local result_count
        result_count=$(echo "$search_result" | jq '.result | length')
        if [[ $result_count -eq 2 ]]; then
            echo "✅"
        else
            echo "❌ (got $result_count results, expected 2)"
            qdrant::api::delete "/collections/${test_collection}" &>/dev/null
            return 1
        fi
    else
        echo "❌"
        qdrant::api::delete "/collections/${test_collection}" &>/dev/null
        return 1
    fi
    
    # Clean up
    qdrant::api::delete "/collections/${test_collection}" &>/dev/null
    return 0
}

#######################################
# Test search performance
#######################################
qdrant::test_search_performance() {
    # Use an existing collection if available
    local collections_json
    if ! collections_json=$(qdrant::api::get "/collections" 2>/dev/null); then
        echo "  ⚠️  No collections available for performance test"
        return 0
    fi
    
    local first_collection
    first_collection=$(echo "$collections_json" | jq -r '.result.collections[0].name // ""')
    
    if [[ -z "$first_collection" ]]; then
        echo "  ⚠️  No collections available for performance test"
        return 0
    fi
    
    # Get collection info
    local collection_info
    if ! collection_info=$(qdrant::api::get "/collections/${first_collection}" 2>/dev/null); then
        echo "  ⚠️  Cannot get collection info"
        return 0
    fi
    
    local vector_size vector_count
    vector_size=$(echo "$collection_info" | jq -r '.result.config.params.vectors.size // 0')
    vector_count=$(echo "$collection_info" | jq -r '.result.vectors_count // 0')
    
    if [[ $vector_size -eq 0 ]] || [[ $vector_count -eq 0 ]]; then
        echo "  ⚠️  Collection empty or invalid"
        return 0
    fi
    
    echo -n "  Testing search on $first_collection ($vector_count vectors)... "
    
    # Generate random test vector
    local test_vector
    test_vector=$(python3 -c "import random, json; print(json.dumps([random.random() for _ in range($vector_size)]))" 2>/dev/null)
    
    # Measure search time
    local search_start search_end latency_ms
    search_start=$(date +%s%N)
    
    if timeout 2 qdrant::api::post "/collections/${first_collection}/points/search" \
        "{\"vector\": $test_vector, \"limit\": 10}" &>/dev/null; then
        search_end=$(date +%s%N)
        latency_ms=$(((search_end - search_start) / 1000000))
        
        if [[ $latency_ms -lt 50 ]]; then
            echo "✅ Excellent ($latency_ms ms)"
        elif [[ $latency_ms -lt 200 ]]; then
            echo "✅ Good ($latency_ms ms)"
        elif [[ $latency_ms -lt 1000 ]]; then
            echo "⚠️  Degraded ($latency_ms ms)"
        else
            echo "❌ Critical ($latency_ms ms)"
            return 1
        fi
    else
        echo "❌ Timeout"
        return 1
    fi
    
    return 0
}

#######################################
# Test backup operations
#######################################
qdrant::test_backup_operations() {
    # Create a test collection for backup
    local test_collection="test-backup-$(date +%s)"
    
    echo -n "  Creating collection for backup test... "
    local create_payload='{
        "vectors": {
            "size": 4,
            "distance": "Cosine"
        }
    }'
    
    if ! qdrant::api::put "/collections/${test_collection}" "$create_payload" &>/dev/null; then
        echo "❌"
        return 1
    fi
    echo "✅"
    
    # Create snapshot
    echo -n "  Creating snapshot... "
    if qdrant::api::post "/collections/${test_collection}/snapshots" "" &>/dev/null; then
        echo "✅"
    else
        echo "❌"
        qdrant::api::delete "/collections/${test_collection}" &>/dev/null
        return 1
    fi
    
    # List snapshots
    echo -n "  Listing snapshots... "
    local snapshots
    if snapshots=$(qdrant::api::get "/collections/${test_collection}/snapshots" 2>/dev/null); then
        local snapshot_count
        snapshot_count=$(echo "$snapshots" | jq '.result | length')
        if [[ $snapshot_count -gt 0 ]]; then
            echo "✅ ($snapshot_count snapshots)"
        else
            echo "❌ No snapshots found"
            qdrant::api::delete "/collections/${test_collection}" &>/dev/null
            return 1
        fi
    else
        echo "❌"
        qdrant::api::delete "/collections/${test_collection}" &>/dev/null
        return 1
    fi
    
    # Clean up
    qdrant::api::delete "/collections/${test_collection}" &>/dev/null
    return 0
}

#######################################
# Test API client functions
#######################################
qdrant::test_api_client() {
    echo -n "  Testing GET request... "
    if qdrant::api::get "/" &>/dev/null; then
        echo "✅"
    else
        echo "❌"
        return 1
    fi
    
    echo -n "  Testing telemetry endpoint... "
    local telemetry
    if telemetry=$(qdrant::api::get "/telemetry" 2>/dev/null); then
        if echo "$telemetry" | jq -e '.result' &>/dev/null; then
            echo "✅"
        else
            echo "❌ Invalid response"
            return 1
        fi
    else
        echo "❌"
        return 1
    fi
    
    return 0
}

#######################################
# Test search optimizer functions
#######################################
qdrant::test_search_optimizer() {
    echo -n "  Testing performance metrics... "
    if qdrant::search::performance_metrics &>/dev/null; then
        echo "✅"
    else
        echo "❌"
        return 1
    fi
    
    echo -n "  Testing cache cleanup... "
    if qdrant::search::clean_cache; then
        echo "✅"
    else
        echo "❌"
        return 1
    fi
    
    return 0
}

#######################################
# Test mathematical accuracy of distance functions
#######################################
qdrant::test_mathematical_accuracy() {
    local test_collection="test-math-accuracy-$(date +%s)"
    
    echo -n "  Creating test collection for accuracy... "
    local create_payload='{
        "vectors": {
            "size": 3,
            "distance": "Cosine"
        }
    }'
    
    if ! qdrant::api::put "/collections/${test_collection}" "$create_payload" &>/dev/null; then
        echo "❌"
        return 1
    fi
    echo "✅"
    
    # Insert known vectors with calculated cosine similarity
    echo -n "  Testing cosine similarity accuracy... "
    local vectors_payload='{
        "points": [
            {"id": 1, "vector": [1.0, 0.0, 0.0]},
            {"id": 2, "vector": [0.0, 1.0, 0.0]},
            {"id": 3, "vector": [0.707107, 0.707107, 0.0]},
            {"id": 4, "vector": [1.0, 0.0, 0.0]}
        ]
    }'
    
    if ! qdrant::api::put "/collections/${test_collection}/points?wait=true" "$vectors_payload" &>/dev/null; then
        echo "❌ Failed to insert vectors"
        qdrant::api::delete "/collections/${test_collection}" &>/dev/null
        return 1
    fi
    
    # Search with vector [1, 0, 0] - should match id:1 and id:4 with score ~1.0
    local search_payload='{
        "vector": [1.0, 0.0, 0.0],
        "limit": 4,
        "with_payload": false,
        "score_threshold": 0.0
    }'
    
    local search_result
    if search_result=$(qdrant::api::post "/collections/${test_collection}/points/search" "$search_payload" 2>/dev/null); then
        # Check if first result has score close to 1.0 (perfect match)
        local first_score
        first_score=$(echo "$search_result" | jq -r '.result[0].score // 0')
        
        # Cosine similarity should be 1.0 for identical vectors
        if (( $(echo "$first_score > 0.999" | bc -l) )); then
            echo "✅ (accuracy within 0.1%)"
        else
            echo "❌ (score $first_score, expected ~1.0)"
            qdrant::api::delete "/collections/${test_collection}" &>/dev/null
            return 1
        fi
    else
        echo "❌ Search failed"
        qdrant::api::delete "/collections/${test_collection}" &>/dev/null
        return 1
    fi
    
    # Clean up
    qdrant::api::delete "/collections/${test_collection}" &>/dev/null
    return 0
}

#######################################
# Run all tests
#######################################
qdrant::test_all() {
    echo "=== Running All Qdrant Tests ==="
    echo
    
    local overall_start
    overall_start=$(date +%s)
    
    # Reset counters
    TESTS_PASSED=0
    TESTS_FAILED=0
    TESTS_SKIPPED=0
    
    # Run test suites
    qdrant::check_basic_health
    echo
    qdrant::test_integration
    echo
    qdrant::test_unit
    
    local overall_end overall_elapsed
    overall_end=$(date +%s)
    overall_elapsed=$((overall_end - overall_start))
    
    echo
    echo "==================================="
    echo "All tests completed in ${overall_elapsed}s"
    echo "Total Passed: $TESTS_PASSED"
    echo "Total Failed: $TESTS_FAILED"
    echo "Total Skipped: $TESTS_SKIPPED"
    echo "==================================="
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo "✅ ALL TESTS PASSED"
        return 0
    else
        echo "❌ SOME TESTS FAILED"
        return 1
    fi
}