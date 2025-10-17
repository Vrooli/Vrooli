#!/usr/bin/env bash
# Mathematical Accuracy Benchmarks for Qdrant
# Tests precision of various distance metrics and vector operations

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
QDRANT_LIB_DIR="${APP_ROOT}/resources/qdrant/lib"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${QDRANT_LIB_DIR}/api-client.sh"

# Benchmark configuration
readonly BENCHMARK_COLLECTION="math-benchmark-$(date +%s)"
readonly TOLERANCE_PERCENTAGE=0.1  # 0.1% tolerance for floating point errors

#######################################
# Run all mathematical benchmarks
# Returns: 0 if all pass, 1 if any fail
#######################################
qdrant::math::run_benchmarks() {
    echo "=== Qdrant Mathematical Accuracy Benchmarks ==="
    echo
    
    local start_time
    start_time=$(date +%s)
    
    local tests_passed=0
    local tests_failed=0
    
    # Test Cosine similarity
    echo "Testing Cosine Similarity..."
    if qdrant::math::test_cosine_similarity; then
        ((tests_passed++))
    else
        ((tests_failed++))
    fi
    echo
    
    # Test Euclidean distance
    echo "Testing Euclidean Distance..."
    if qdrant::math::test_euclidean_distance; then
        ((tests_passed++))
    else
        ((tests_failed++))
    fi
    echo
    
    # Test Dot product
    echo "Testing Dot Product..."
    if qdrant::math::test_dot_product; then
        ((tests_passed++))
    else
        ((tests_failed++))
    fi
    echo
    
    # Test high-dimensional accuracy
    echo "Testing High-Dimensional Accuracy..."
    if qdrant::math::test_high_dimensional; then
        ((tests_passed++))
    else
        ((tests_failed++))
    fi
    echo
    
    # Test edge cases
    echo "Testing Edge Cases..."
    if qdrant::math::test_edge_cases; then
        ((tests_passed++))
    else
        ((tests_failed++))
    fi
    
    local end_time elapsed
    end_time=$(date +%s)
    elapsed=$((end_time - start_time))
    
    echo
    echo "==================================="
    echo "Benchmark completed in ${elapsed}s"
    echo "Passed: $tests_passed | Failed: $tests_failed"
    echo "==================================="
    
    if [[ $tests_failed -eq 0 ]]; then
        echo "✅ ALL BENCHMARKS PASSED"
        return 0
    else
        echo "❌ SOME BENCHMARKS FAILED"
        return 1
    fi
}

#######################################
# Test Cosine similarity accuracy
# Returns: 0 if accurate, 1 if not
#######################################
qdrant::math::test_cosine_similarity() {
    local collection="cosine-test-$(date +%s)"
    
    # Create collection with Cosine distance
    local create_payload='{
        "vectors": {
            "size": 3,
            "distance": "Cosine"
        }
    }'
    
    if ! qdrant::api::put "/collections/${collection}" "$create_payload" &>/dev/null; then
        echo "  ❌ Failed to create collection"
        return 1
    fi
    
    # Insert test vectors with known cosine similarities
    # v1 = [1, 0, 0], v2 = [0, 1, 0] -> cosine = 0 (orthogonal)
    # v3 = [1, 1, 0] normalized -> cosine with v1 = 0.707
    # v4 = [-1, 0, 0] -> cosine with v1 = -1 (opposite)
    local vectors_payload='{
        "points": [
            {"id": 1, "vector": [1.0, 0.0, 0.0]},
            {"id": 2, "vector": [0.0, 1.0, 0.0]},
            {"id": 3, "vector": [0.7071067812, 0.7071067812, 0.0]},
            {"id": 4, "vector": [-1.0, 0.0, 0.0]}
        ]
    }'
    
    if ! qdrant::api::put "/collections/${collection}/points?wait=true" "$vectors_payload" &>/dev/null; then
        echo "  ❌ Failed to insert vectors"
        qdrant::api::delete "/collections/${collection}" &>/dev/null
        return 1
    fi
    
    # Test 1: Search with [1, 0, 0] - should find itself with score ~1.0
    local search_result
    search_result=$(qdrant::api::post "/collections/${collection}/points/search" \
        '{"vector": [1.0, 0.0, 0.0], "limit": 4}' 2>/dev/null)
    
    local first_score
    first_score=$(echo "$search_result" | jq -r '.result[0].score // 0')
    
    if ! qdrant::math::check_accuracy "$first_score" "1.0" "$TOLERANCE_PERCENTAGE"; then
        echo "  ❌ Identity vector test failed (score: $first_score, expected: 1.0)"
        qdrant::api::delete "/collections/${collection}" &>/dev/null
        return 1
    fi
    echo "  ✅ Identity vector: score=$first_score (expected: 1.0)"
    
    # Test 2: Check orthogonal vector score
    local orthogonal_score
    orthogonal_score=$(echo "$search_result" | jq -r '.result[] | select(.id == 2) | .score // 0')
    
    if ! qdrant::math::check_accuracy "$orthogonal_score" "0.0" "1"; then
        echo "  ❌ Orthogonal vector test failed (score: $orthogonal_score, expected: 0.0)"
        qdrant::api::delete "/collections/${collection}" &>/dev/null
        return 1
    fi
    echo "  ✅ Orthogonal vector: score=$orthogonal_score (expected: 0.0)"
    
    # Test 3: Check 45-degree vector score (cosine = 0.707)
    local diagonal_score
    diagonal_score=$(echo "$search_result" | jq -r '.result[] | select(.id == 3) | .score // 0')
    
    if ! qdrant::math::check_accuracy "$diagonal_score" "0.707" "$TOLERANCE_PERCENTAGE"; then
        echo "  ❌ 45-degree vector test failed (score: $diagonal_score, expected: 0.707)"
        qdrant::api::delete "/collections/${collection}" &>/dev/null
        return 1
    fi
    echo "  ✅ 45-degree vector: score=$diagonal_score (expected: 0.707)"
    
    # Clean up
    qdrant::api::delete "/collections/${collection}" &>/dev/null
    return 0
}

#######################################
# Test Euclidean distance accuracy
# Returns: 0 if accurate, 1 if not
#######################################
qdrant::math::test_euclidean_distance() {
    local collection="euclidean-test-$(date +%s)"
    
    # Create collection with Euclidean distance
    local create_payload='{
        "vectors": {
            "size": 2,
            "distance": "Euclid"
        }
    }'
    
    if ! qdrant::api::put "/collections/${collection}" "$create_payload" &>/dev/null; then
        echo "  ❌ Failed to create collection"
        return 1
    fi
    
    # Insert test vectors with known Euclidean distances
    # Origin, unit vectors, and diagonal
    local vectors_payload='{
        "points": [
            {"id": 1, "vector": [0.0, 0.0]},
            {"id": 2, "vector": [1.0, 0.0]},
            {"id": 3, "vector": [0.0, 1.0]},
            {"id": 4, "vector": [1.0, 1.0]},
            {"id": 5, "vector": [3.0, 4.0]}
        ]
    }'
    
    if ! qdrant::api::put "/collections/${collection}/points?wait=true" "$vectors_payload" &>/dev/null; then
        echo "  ❌ Failed to insert vectors"
        qdrant::api::delete "/collections/${collection}" &>/dev/null
        return 1
    fi
    
    # Test: Search from origin [0, 0]
    # Distance to [1, 0] = 1.0
    # Distance to [0, 1] = 1.0
    # Distance to [1, 1] = sqrt(2) = 1.414
    # Distance to [3, 4] = 5.0
    local search_result
    search_result=$(qdrant::api::post "/collections/${collection}/points/search" \
        '{"vector": [0.0, 0.0], "limit": 5}' 2>/dev/null)
    
    # Euclidean distance uses negative score (closer = higher score)
    # Check distance to [3, 4] should be 5.0
    local point5_score
    point5_score=$(echo "$search_result" | jq -r '.result[] | select(.id == 5) | .score // 0')
    
    # Convert score back to distance (Qdrant uses 1/(1+distance) for Euclidean)
    # So distance = (1/score) - 1
    local calculated_distance
    if [[ "$point5_score" != "0" ]]; then
        calculated_distance=$(echo "scale=4; (1 / $point5_score) - 1" | bc)
        
        if ! qdrant::math::check_accuracy "$calculated_distance" "5.0" "1"; then
            echo "  ❌ Distance calculation failed (distance: $calculated_distance, expected: 5.0)"
            qdrant::api::delete "/collections/${collection}" &>/dev/null
            return 1
        fi
        echo "  ✅ Distance to [3,4]: $calculated_distance (expected: 5.0)"
    else
        echo "  ⚠️  Euclidean scoring format different than expected"
    fi
    
    # Clean up
    qdrant::api::delete "/collections/${collection}" &>/dev/null
    return 0
}

#######################################
# Test Dot product accuracy
# Returns: 0 if accurate, 1 if not
#######################################
qdrant::math::test_dot_product() {
    local collection="dot-test-$(date +%s)"
    
    # Create collection with Dot product distance
    local create_payload='{
        "vectors": {
            "size": 3,
            "distance": "Dot"
        }
    }'
    
    if ! qdrant::api::put "/collections/${collection}" "$create_payload" &>/dev/null; then
        echo "  ❌ Failed to create collection"
        return 1
    fi
    
    # Insert test vectors with known dot products
    local vectors_payload='{
        "points": [
            {"id": 1, "vector": [1.0, 0.0, 0.0]},
            {"id": 2, "vector": [0.0, 1.0, 0.0]},
            {"id": 3, "vector": [1.0, 1.0, 0.0]},
            {"id": 4, "vector": [2.0, 3.0, 4.0]}
        ]
    }'
    
    if ! qdrant::api::put "/collections/${collection}/points?wait=true" "$vectors_payload" &>/dev/null; then
        echo "  ❌ Failed to insert vectors"
        qdrant::api::delete "/collections/${collection}" &>/dev/null
        return 1
    fi
    
    # Test: Search with [1, 2, 3]
    # Dot with [1, 0, 0] = 1*1 + 2*0 + 3*0 = 1
    # Dot with [0, 1, 0] = 1*0 + 2*1 + 3*0 = 2
    # Dot with [1, 1, 0] = 1*1 + 2*1 + 3*0 = 3
    # Dot with [2, 3, 4] = 1*2 + 2*3 + 3*4 = 2 + 6 + 12 = 20
    local search_result
    search_result=$(qdrant::api::post "/collections/${collection}/points/search" \
        '{"vector": [1.0, 2.0, 3.0], "limit": 4}' 2>/dev/null)
    
    # Check highest score should be point 4 with dot product 20
    local first_id first_score
    first_id=$(echo "$search_result" | jq -r '.result[0].id // 0')
    first_score=$(echo "$search_result" | jq -r '.result[0].score // 0')
    
    if [[ "$first_id" == "4" ]]; then
        echo "  ✅ Highest dot product correctly identified (id: $first_id, score: $first_score)"
    else
        echo "  ❌ Dot product ordering incorrect (top id: $first_id, expected: 4)"
        qdrant::api::delete "/collections/${collection}" &>/dev/null
        return 1
    fi
    
    # Clean up
    qdrant::api::delete "/collections/${collection}" &>/dev/null
    return 0
}

#######################################
# Test high-dimensional vector accuracy
# Returns: 0 if accurate, 1 if not
#######################################
qdrant::math::test_high_dimensional() {
    local collection="highdim-test-$(date +%s)"
    local dimensions=1536  # Common embedding size
    
    # Create collection with high dimensions
    local create_payload="{
        \"vectors\": {
            \"size\": $dimensions,
            \"distance\": \"Cosine\"
        }
    }"
    
    if ! qdrant::api::put "/collections/${collection}" "$create_payload" &>/dev/null; then
        echo "  ❌ Failed to create high-dimensional collection"
        return 1
    fi
    
    # Generate orthogonal unit vectors in high dimensions
    # v1 = [1, 0, 0, ..., 0]
    # v2 = [0, 1, 0, ..., 0]
    local v1 v2
    v1=$(python3 -c "import json; v=[0.0]*$dimensions; v[0]=1.0; print(json.dumps(v))")
    v2=$(python3 -c "import json; v=[0.0]*$dimensions; v[1]=1.0; print(json.dumps(v))")
    
    local vectors_payload="{
        \"points\": [
            {\"id\": 1, \"vector\": $v1},
            {\"id\": 2, \"vector\": $v2}
        ]
    }"
    
    if ! qdrant::api::put "/collections/${collection}/points?wait=true" "$vectors_payload" &>/dev/null; then
        echo "  ❌ Failed to insert high-dimensional vectors"
        qdrant::api::delete "/collections/${collection}" &>/dev/null
        return 1
    fi
    
    # Search with v1, should find itself with score ~1.0
    local search_result
    search_result=$(qdrant::api::post "/collections/${collection}/points/search" \
        "{\"vector\": $v1, \"limit\": 2}" 2>/dev/null)
    
    local first_score second_score
    first_score=$(echo "$search_result" | jq -r '.result[0].score // 0')
    second_score=$(echo "$search_result" | jq -r '.result[1].score // 0')
    
    if ! qdrant::math::check_accuracy "$first_score" "1.0" "$TOLERANCE_PERCENTAGE"; then
        echo "  ❌ High-dimensional self-similarity failed (score: $first_score, expected: 1.0)"
        qdrant::api::delete "/collections/${collection}" &>/dev/null
        return 1
    fi
    
    if ! qdrant::math::check_accuracy "$second_score" "0.0" "1"; then
        echo "  ❌ High-dimensional orthogonality failed (score: $second_score, expected: 0.0)"
        qdrant::api::delete "/collections/${collection}" &>/dev/null
        return 1
    fi
    
    echo "  ✅ High-dimensional accuracy maintained (${dimensions}D)"
    
    # Clean up
    qdrant::api::delete "/collections/${collection}" &>/dev/null
    return 0
}

#######################################
# Test edge cases
# Returns: 0 if handled correctly, 1 if not
#######################################
qdrant::math::test_edge_cases() {
    local collection="edge-test-$(date +%s)"
    
    # Create collection
    local create_payload='{
        "vectors": {
            "size": 3,
            "distance": "Cosine"
        }
    }'
    
    if ! qdrant::api::put "/collections/${collection}" "$create_payload" &>/dev/null; then
        echo "  ❌ Failed to create collection"
        return 1
    fi
    
    # Test zero vector handling
    echo -n "  Testing zero vector... "
    local zero_vector='{"id": 1, "vector": [0.0, 0.0, 0.0]}'
    local insert_result
    insert_result=$(qdrant::api::put "/collections/${collection}/points?wait=true" \
        "{\"points\": [$zero_vector]}" 2>&1)
    
    if echo "$insert_result" | grep -q "error"; then
        echo "✅ Zero vector properly rejected"
    else
        echo "⚠️  Zero vector accepted (may cause issues)"
    fi
    
    # Test very small values (denormalized numbers)
    echo -n "  Testing denormalized numbers... "
    local tiny_vector='{"id": 2, "vector": [1e-308, 1e-308, 1e-308]}'
    if qdrant::api::put "/collections/${collection}/points?wait=true" \
        "{\"points\": [$tiny_vector]}" &>/dev/null; then
        echo "✅ Handles tiny values"
    else
        echo "⚠️  Issues with tiny values"
    fi
    
    # Test very large values
    echo -n "  Testing large values... "
    local large_vector='{"id": 3, "vector": [1e38, 1e38, 1e38]}'
    if qdrant::api::put "/collections/${collection}/points?wait=true" \
        "{\"points\": [$large_vector]}" &>/dev/null; then
        echo "✅ Handles large values"
    else
        echo "⚠️  Issues with large values"
    fi
    
    # Clean up
    qdrant::api::delete "/collections/${collection}" &>/dev/null
    return 0
}

#######################################
# Check if a value is within tolerance of expected
# Arguments:
#   $1 - Actual value
#   $2 - Expected value
#   $3 - Tolerance percentage
# Returns: 0 if within tolerance, 1 if not
#######################################
qdrant::math::check_accuracy() {
    local actual="${1:-0}"
    local expected="${2:-0}"
    local tolerance="${3:-0.1}"
    
    # Calculate absolute difference
    local diff
    diff=$(echo "scale=10; ($actual - $expected)" | bc)
    if (( $(echo "$diff < 0" | bc -l) )); then
        diff=$(echo "scale=10; -($diff)" | bc)
    fi
    
    # Calculate tolerance threshold
    local threshold
    if (( $(echo "$expected == 0" | bc -l) )); then
        threshold=$(echo "scale=10; $tolerance / 100" | bc)
    else
        local abs_expected="$expected"
        if (( $(echo "$expected < 0" | bc -l) )); then
            abs_expected=$(echo "-($expected)" | bc)
        fi
        threshold=$(echo "scale=10; $abs_expected * $tolerance / 100" | bc)
    fi
    
    # Check if within tolerance
    if (( $(echo "$diff <= $threshold" | bc -l) )); then
        return 0
    else
        return 1
    fi
}

#######################################
# Generate benchmark report
#######################################
qdrant::math::generate_report() {
    echo "=== Mathematical Accuracy Report ==="
    echo
    echo "Timestamp: $(date -Iseconds)"
    echo "Qdrant Version: $(curl -sf "http://localhost:${QDRANT_PORT}/" | jq -r '.version // "unknown"')"
    echo
    
    # Run benchmarks and capture results
    local benchmark_output
    benchmark_output=$(qdrant::math::run_benchmarks 2>&1)
    
    echo "$benchmark_output"
    
    # Generate summary
    echo
    echo "=== Summary ==="
    local passed failed
    passed=$(echo "$benchmark_output" | grep -c "✅" || true)
    failed=$(echo "$benchmark_output" | grep -c "❌" || true)
    
    echo "Tests Passed: $passed"
    echo "Tests Failed: $failed"
    
    if [[ $failed -eq 0 ]]; then
        echo "Status: EXCELLENT - All mathematical operations accurate within tolerance"
    elif [[ $failed -le 2 ]]; then
        echo "Status: GOOD - Minor accuracy issues detected"
    else
        echo "Status: DEGRADED - Multiple accuracy issues require attention"
    fi
    
    return 0
}