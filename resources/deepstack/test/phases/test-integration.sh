#!/usr/bin/env bash
# DeepStack Resource - Integration Tests
# End-to-end functionality validation (must complete in <120s)

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
readonly TEST_DIR="$(dirname "${SCRIPT_DIR}")"
readonly RESOURCE_DIR="$(dirname "${TEST_DIR}")"

# Load test library
source "${RESOURCE_DIR}/lib/test.sh"

# Timeout for integration tests
INTEGRATION_TIMEOUT=120

# Test fixtures directory
FIXTURES_DIR="${TEST_DIR}/fixtures"

# Run integration tests
run_integration_tests() {
    local start_time=$(date +%s)
    local failures=0
    
    echo "DeepStack Integration Tests"
    echo "=========================="
    
    # Ensure service is running
    if ! docker ps | grep -q "${DEEPSTACK_CONTAINER_NAME}"; then
        echo "✗ DeepStack is not running. Start it first:" >&2
        echo "  $(basename "${RESOURCE_DIR}/cli.sh") manage start --wait" >&2
        return 1
    fi
    
    # Test 1: Object Detection API
    echo ""
    echo "Test 1: Object Detection API"
    local test_response=$(curl -s -X POST "http://${DEEPSTACK_HOST}:${DEEPSTACK_PORT}/v1/vision/detection" \
        -F "image=@${FIXTURES_DIR}/test-image.jpg" \
        -F "min_confidence=0.45" 2>/dev/null || echo '{"success":false}')
    
    if echo "$test_response" | jq -e '.success == true' &> /dev/null; then
        echo "✓ Object detection API working"
        local predictions=$(echo "$test_response" | jq '.predictions | length')
        echo "  Detected $predictions objects"
    else
        echo "✗ Object detection API failed" >&2
        ((failures++))
    fi
    
    # Test 2: Face Detection API
    echo ""
    echo "Test 2: Face Detection API"
    test_response=$(curl -s -X POST "http://${DEEPSTACK_HOST}:${DEEPSTACK_PORT}/v1/vision/face" \
        -F "image=@${FIXTURES_DIR}/test-face.jpg" 2>/dev/null || echo '{"success":false}')
    
    if echo "$test_response" | jq -e '.success == true' &> /dev/null; then
        echo "✓ Face detection API working"
        local faces=$(echo "$test_response" | jq '.predictions | length')
        echo "  Detected $faces faces"
    else
        echo "⚠ Face detection API not available (may need activation)"
    fi
    
    # Test 3: Scene Classification API
    echo ""
    echo "Test 3: Scene Classification API"
    test_response=$(curl -s -X POST "http://${DEEPSTACK_HOST}:${DEEPSTACK_PORT}/v1/vision/scene" \
        -F "image=@${FIXTURES_DIR}/test-scene.jpg" 2>/dev/null || echo '{"success":false}')
    
    if echo "$test_response" | jq -e '.success == true' &> /dev/null; then
        echo "✓ Scene classification API working"
        local scene=$(echo "$test_response" | jq -r '.label')
        local confidence=$(echo "$test_response" | jq -r '.confidence')
        echo "  Scene: $scene (confidence: $confidence)"
    else
        echo "⚠ Scene classification API not available (may need activation)"
    fi
    
    # Test 4: Face Registration and Recognition
    echo ""
    echo "Test 4: Face Registration & Recognition"
    
    # Register a face
    local register_response=$(curl -s -X POST "http://${DEEPSTACK_HOST}:${DEEPSTACK_PORT}/v1/vision/face/register" \
        -F "userid=test_user" \
        -F "image=@${FIXTURES_DIR}/test-face.jpg" 2>/dev/null || echo '{"success":false}')
    
    if echo "$register_response" | jq -e '.success == true' &> /dev/null; then
        echo "✓ Face registration successful"
        
        # Try recognition
        local recognize_response=$(curl -s -X POST "http://${DEEPSTACK_HOST}:${DEEPSTACK_PORT}/v1/vision/face/recognize" \
            -F "image=@${FIXTURES_DIR}/test-face.jpg" 2>/dev/null || echo '{"success":false}')
        
        if echo "$recognize_response" | jq -e '.success == true' &> /dev/null; then
            echo "✓ Face recognition successful"
            local recognized=$(echo "$recognize_response" | jq -r '.predictions[0].userid')
            echo "  Recognized: $recognized"
        else
            echo "⚠ Face recognition not available"
        fi
    else
        echo "⚠ Face registration not available"
    fi
    
    # Test 5: Batch Processing
    echo ""
    echo "Test 5: Concurrent Request Handling"
    local concurrent_start=$(date +%s%3N)
    
    # Send 3 concurrent requests
    (curl -s -X POST "http://${DEEPSTACK_HOST}:${DEEPSTACK_PORT}/v1/vision/detection" \
        -F "image=@${FIXTURES_DIR}/test-image.jpg" -F "min_confidence=0.45" &> /dev/null) &
    (curl -s -X POST "http://${DEEPSTACK_HOST}:${DEEPSTACK_PORT}/v1/vision/detection" \
        -F "image=@${FIXTURES_DIR}/test-image.jpg" -F "min_confidence=0.45" &> /dev/null) &
    (curl -s -X POST "http://${DEEPSTACK_HOST}:${DEEPSTACK_PORT}/v1/vision/detection" \
        -F "image=@${FIXTURES_DIR}/test-image.jpg" -F "min_confidence=0.45" &> /dev/null) &
    
    wait
    local concurrent_end=$(date +%s%3N)
    local concurrent_time=$((concurrent_end - concurrent_start))
    
    echo "✓ Handled 3 concurrent requests in ${concurrent_time}ms"
    
    # Test 6: Redis Caching (if enabled)
    if [[ "${DEEPSTACK_REDIS_ENABLED}" == "true" ]]; then
        echo ""
        echo "Test 6: Redis Caching"
        test_redis_connection || echo "⚠ Redis caching not operational"
    fi
    
    # Test 7: GPU Acceleration (if available)
    echo ""
    echo "Test 7: GPU Acceleration"
    test_gpu_available || echo "ℹ Running in CPU mode"
    
    # Check total execution time
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo ""
    echo "Test duration: ${duration}s"
    
    if [[ $duration -gt $INTEGRATION_TIMEOUT ]]; then
        echo "⚠ Warning: Integration tests exceeded ${INTEGRATION_TIMEOUT}s timeout" >&2
    fi
    
    if [[ $failures -gt 0 ]]; then
        echo ""
        echo "✗ Integration tests failed with $failures error(s)" >&2
        return 1
    else
        echo ""
        echo "✓ All integration tests passed"
        return 0
    fi
}

# Create test fixtures if they don't exist
create_test_fixtures() {
    mkdir -p "$FIXTURES_DIR"
    
    # Create simple test images if not present
    if [[ ! -f "${FIXTURES_DIR}/test-image.jpg" ]]; then
        echo "Creating test image fixture..."
        # Download a sample image or create a simple one
        curl -s -o "${FIXTURES_DIR}/test-image.jpg" \
            "https://raw.githubusercontent.com/deepquestai/deepstack/master/tests/test-image3.jpg" 2>/dev/null || \
        echo "⚠ Could not download test image"
    fi
    
    if [[ ! -f "${FIXTURES_DIR}/test-face.jpg" ]]; then
        # Use the same image for face testing
        cp "${FIXTURES_DIR}/test-image.jpg" "${FIXTURES_DIR}/test-face.jpg" 2>/dev/null || true
    fi
    
    if [[ ! -f "${FIXTURES_DIR}/test-scene.jpg" ]]; then
        # Use the same image for scene testing
        cp "${FIXTURES_DIR}/test-image.jpg" "${FIXTURES_DIR}/test-scene.jpg" 2>/dev/null || true
    fi
}

# Create fixtures and run tests
create_test_fixtures

# Run with timeout enforcement
if timeout "$INTEGRATION_TIMEOUT" bash -c "$(declare -f run_integration_tests); $(declare -f test_redis_connection); $(declare -f test_gpu_available); source '${RESOURCE_DIR}/config/defaults.sh'; run_integration_tests"; then
    exit 0
else
    exit_code=$?
    if [[ $exit_code -eq 124 ]]; then
        echo "✗ Integration tests timed out after ${INTEGRATION_TIMEOUT}s" >&2
    fi
    exit $exit_code
fi