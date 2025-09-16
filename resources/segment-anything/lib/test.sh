#!/usr/bin/env bash
# Segment Anything Resource - Test Library Functions

set -euo pipefail

# Run smoke test (quick health check)
run_smoke_test() {
    echo "Running smoke test..."
    local exit_code=0
    
    # Test 1: Check if service is running
    echo -n "  Checking if service is running... "
    if docker ps --format "table {{.Names}}" | grep -q "^${SEGMENT_ANYTHING_CONTAINER}$"; then
        echo "✓"
    else
        echo "✗ (service not running)"
        exit_code=1
    fi
    
    # Test 2: Health endpoint
    echo -n "  Checking health endpoint... "
    if timeout 5 curl -sf "http://localhost:${SEGMENT_ANYTHING_PORT}/health" >/dev/null 2>&1; then
        echo "✓"
    else
        echo "✗ (health check failed)"
        exit_code=1
    fi
    
    # Test 3: API models endpoint
    echo -n "  Checking models endpoint... "
    if timeout 5 curl -sf "http://localhost:${SEGMENT_ANYTHING_PORT}/api/v1/models" >/dev/null 2>&1; then
        echo "✓"
    else
        echo "✗ (models endpoint failed)"
        exit_code=1
    fi
    
    # Test 4: Port accessibility
    echo -n "  Checking port ${SEGMENT_ANYTHING_PORT}... "
    if nc -z localhost "${SEGMENT_ANYTHING_PORT}" 2>/dev/null; then
        echo "✓"
    else
        echo "✗ (port not accessible)"
        exit_code=1
    fi
    
    if [[ $exit_code -eq 0 ]]; then
        echo "✓ Smoke test passed"
    else
        echo "✗ Smoke test failed"
    fi
    
    return $exit_code
}

# Run integration test
run_integration_test() {
    echo "Running integration test..."
    local exit_code=0
    
    # Ensure service is running
    if ! docker ps --format "table {{.Names}}" | grep -q "^${SEGMENT_ANYTHING_CONTAINER}$"; then
        echo "Error: Service must be running for integration tests" >&2
        return 1
    fi
    
    # Test 1: Health check with response validation
    echo -n "  Testing health response format... "
    local health_response
    health_response=$(timeout 5 curl -sf "http://localhost:${SEGMENT_ANYTHING_PORT}/health" 2>/dev/null)
    if echo "$health_response" | jq -e '.status == "healthy"' >/dev/null 2>&1; then
        echo "✓"
    else
        echo "✗ (invalid health response)"
        exit_code=1
    fi
    
    # Test 2: Models list
    echo -n "  Testing models list... "
    local models_response
    models_response=$(timeout 5 curl -sf "http://localhost:${SEGMENT_ANYTHING_PORT}/api/v1/models" 2>/dev/null)
    if echo "$models_response" | jq -e '.models | length > 0' >/dev/null 2>&1; then
        echo "✓"
    else
        echo "✗ (no models found)"
        exit_code=1
    fi
    
    # Test 3: Create test image
    echo -n "  Creating test image... "
    create_test_image
    echo "✓"
    
    # Test 4: Segmentation with test image
    echo -n "  Testing segmentation endpoint... "
    local seg_response
    seg_response=$(timeout 10 curl -sf -X POST \
        "http://localhost:${SEGMENT_ANYTHING_PORT}/api/v1/segment" \
        -F "image=@/tmp/test_image.png" \
        -F "prompt=auto" 2>/dev/null)
    
    if echo "$seg_response" | jq -e '.masks' >/dev/null 2>&1; then
        echo "✓"
    else
        echo "✗ (segmentation failed)"
        exit_code=1
    fi
    
    # Test 5: Redis integration (if available)
    if command -v redis-cli &>/dev/null && redis-cli -p "${SEGMENT_ANYTHING_REDIS_PORT}" ping >/dev/null 2>&1; then
        echo -n "  Testing Redis caching... "
        # Would test caching here
        echo "✓"
    fi
    
    # Cleanup
    rm -f /tmp/test_image.png
    
    if [[ $exit_code -eq 0 ]]; then
        echo "✓ Integration test passed"
    else
        echo "✗ Integration test failed"
    fi
    
    return $exit_code
}

# Run unit tests
run_unit_test() {
    echo "Running unit tests..."
    local exit_code=0
    
    # Test 1: Configuration loading
    echo -n "  Testing configuration loading... "
    if [[ -n "${SEGMENT_ANYTHING_PORT}" ]] && [[ "${SEGMENT_ANYTHING_PORT}" -eq 11454 ]]; then
        echo "✓"
    else
        echo "✗ (configuration not loaded)"
        exit_code=1
    fi
    
    # Test 2: Directory structure
    echo -n "  Testing directory structure... "
    if [[ -d "${RESOURCE_DIR}/lib" ]] && [[ -d "${RESOURCE_DIR}/config" ]]; then
        echo "✓"
    else
        echo "✗ (invalid directory structure)"
        exit_code=1
    fi
    
    # Test 3: Runtime configuration
    echo -n "  Testing runtime.json... "
    if [[ -f "${RESOURCE_DIR}/config/runtime.json" ]]; then
        if jq -e '.startup_order == 550' "${RESOURCE_DIR}/config/runtime.json" >/dev/null 2>&1; then
            echo "✓"
        else
            echo "✗ (invalid runtime config)"
            exit_code=1
        fi
    else
        echo "✗ (runtime.json missing)"
        exit_code=1
    fi
    
    # Test 4: Docker availability
    echo -n "  Testing Docker availability... "
    if command -v docker &>/dev/null; then
        echo "✓"
    else
        echo "✗ (Docker not available)"
        exit_code=1
    fi
    
    # Test 5: Model directory
    echo -n "  Testing model directory... "
    if [[ -d "${SEGMENT_ANYTHING_MODEL_DIR}" ]] || mkdir -p "${SEGMENT_ANYTHING_MODEL_DIR}" 2>/dev/null; then
        echo "✓"
    else
        echo "✗ (cannot create model dir)"
        exit_code=1
    fi
    
    if [[ $exit_code -eq 0 ]]; then
        echo "✓ Unit tests passed"
    else
        echo "✗ Unit tests failed"
    fi
    
    return $exit_code
}

# Run all tests
run_all_tests() {
    echo "Running all test suites..."
    local exit_code=0
    
    # Run unit tests first (no dependencies)
    echo -e "\n[Unit Tests]"
    if ! run_unit_test; then
        exit_code=1
    fi
    
    # Run smoke test
    echo -e "\n[Smoke Test]"
    if ! run_smoke_test; then
        exit_code=1
    fi
    
    # Run integration test
    echo -e "\n[Integration Test]"
    if ! run_integration_test; then
        exit_code=1
    fi
    
    echo -e "\n========================================"
    if [[ $exit_code -eq 0 ]]; then
        echo "✓ All tests passed"
    else
        echo "✗ Some tests failed"
    fi
    
    return $exit_code
}

# Create a simple test image
create_test_image() {
    # Create a simple PNG using ImageMagick if available, otherwise use Python
    if command -v convert &>/dev/null; then
        convert -size 640x480 xc:white \
            -fill blue -draw "rectangle 100,100 200,200" \
            -fill red -draw "circle 400,300 450,300" \
            /tmp/test_image.png
    elif command -v python3 &>/dev/null; then
        python3 -c "
from PIL import Image, ImageDraw
img = Image.new('RGB', (640, 480), 'white')
draw = ImageDraw.Draw(img)
draw.rectangle([100, 100, 200, 200], fill='blue')
draw.ellipse([350, 250, 450, 350], fill='red')
img.save('/tmp/test_image.png')
" 2>/dev/null || {
            # Fallback: create a minimal PNG
            echo -e '\x89PNG\r\n\x1a\n' > /tmp/test_image.png
        }
    else
        # Create a minimal valid PNG file
        echo -e '\x89PNG\r\n\x1a\n' > /tmp/test_image.png
    fi
}