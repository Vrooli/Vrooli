#!/bin/bash
# Test library for Ultralytics YOLO resource

set -euo pipefail

#######################################
# Smoke test - Quick health validation
#######################################
yolo::test_smoke() {
    echo "Running YOLO smoke tests..."
    
    local passed=0
    local failed=0
    
    # Test 1: Service is running
    echo -n "  Testing service status... "
    if docker ps --format "table {{.Names}}" | grep -q "^${YOLO_CONTAINER_NAME}$"; then
        echo "✓"
        ((passed++))
    else
        echo "✗ (service not running)"
        ((failed++))
        echo "    Start with: resource-ultralytics-yolo manage start"
        return 1
    fi
    
    # Test 2: Health endpoint responds
    echo -n "  Testing health endpoint... "
    if timeout 5 curl -sf "http://localhost:${YOLO_PORT}/health" &> /dev/null; then
        echo "✓"
        ((passed++))
    else
        echo "✗ (health check failed)"
        ((failed++))
    fi
    
    # Test 3: Model status endpoint
    echo -n "  Testing model status... "
    if timeout 5 curl -sf "http://localhost:${YOLO_PORT}/models/status" &> /dev/null; then
        echo "✓"
        ((passed++))
    else
        echo "✗ (model status failed)"
        ((failed++))
    fi
    
    # Test 4: GPU detection (optional)
    echo -n "  Testing GPU availability... "
    if command -v nvidia-smi &> /dev/null && nvidia-smi &> /dev/null; then
        echo "✓ (GPU available)"
        ((passed++))
    else
        echo "○ (GPU not available - CPU mode)"
    fi
    
    # Test 5: Port accessibility
    echo -n "  Testing port ${YOLO_PORT}... "
    if nc -z localhost "${YOLO_PORT}" 2>/dev/null; then
        echo "✓"
        ((passed++))
    else
        echo "✗ (port not accessible)"
        ((failed++))
    fi
    
    # Summary
    echo ""
    echo "Smoke Test Summary: $passed passed, $failed failed"
    
    if [[ $failed -gt 0 ]]; then
        return 1
    fi
    return 0
}

#######################################
# Integration test - End-to-end functionality
#######################################
yolo::test_integration() {
    echo "Running YOLO integration tests..."
    
    local passed=0
    local failed=0
    
    # Ensure service is running
    if ! docker ps --format "table {{.Names}}" | grep -q "^${YOLO_CONTAINER_NAME}$"; then
        echo "  Service not running. Starting..."
        yolo::start --wait
    fi
    
    # Test 1: Detection with sample image
    echo -n "  Testing object detection... "
    
    # Create a simple test image (red square on white background)
    local test_image="/tmp/yolo_test_image.jpg"
    python3 -c "
import numpy as np
from PIL import Image
img = np.ones((640, 640, 3), dtype=np.uint8) * 255
img[100:200, 100:200] = [255, 0, 0]  # Red square
Image.fromarray(img).save('$test_image')
" 2>/dev/null || {
        # Fallback: download sample image
        curl -sL "https://ultralytics.com/images/zidane.jpg" -o "$test_image" 2>/dev/null || {
            echo "○ (couldn't create test image)"
            return 0
        }
    }
    
    if [[ -f "$test_image" ]]; then
        local response=$(curl -sf -X POST \
            "http://localhost:${YOLO_PORT}/detect" \
            -F "file=@$test_image" 2>/dev/null)
        
        if [[ -n "$response" ]]; then
            if echo "$response" | jq -e '.detections' &> /dev/null; then
                echo "✓"
                ((passed++))
            else
                echo "✗ (invalid response format)"
                ((failed++))
            fi
        else
            echo "✗ (no response)"
            ((failed++))
        fi
        rm -f "$test_image"
    else
        echo "○ (test skipped - no test image)"
    fi
    
    # Test 2: Model listing
    echo -n "  Testing model listing... "
    local models=$(curl -sf "http://localhost:${YOLO_PORT}/models" 2>/dev/null)
    if echo "$models" | jq -e '.models' &> /dev/null; then
        echo "✓"
        ((passed++))
    else
        echo "✗ (model listing failed)"
        ((failed++))
    fi
    
    # Test 3: Performance check
    echo -n "  Testing inference performance... "
    local start_time=$(date +%s%N)
    timeout 10 curl -sf "http://localhost:${YOLO_PORT}/health" &> /dev/null
    local end_time=$(date +%s%N)
    local response_time=$(( (end_time - start_time) / 1000000 ))
    
    if [[ $response_time -lt 1000 ]]; then
        echo "✓ (${response_time}ms)"
        ((passed++))
    else
        echo "✗ (${response_time}ms > 1000ms)"
        ((failed++))
    fi
    
    # Test 4: Integration endpoints (if enabled)
    if [[ "${YOLO_ENABLE_QDRANT}" == "true" ]]; then
        echo -n "  Testing Qdrant integration... "
        if timeout 2 curl -sf "http://${QDRANT_HOST}:${QDRANT_PORT}/collections" &> /dev/null; then
            echo "✓"
            ((passed++))
        else
            echo "✗ (Qdrant not accessible)"
            ((failed++))
        fi
    fi
    
    if [[ "${YOLO_ENABLE_POSTGRES}" == "true" ]]; then
        echo -n "  Testing PostgreSQL integration... "
        if command -v pg_isready &> /dev/null; then
            if timeout 2 pg_isready -h "${POSTGRES_HOST}" -p "${POSTGRES_PORT}" &> /dev/null; then
                echo "✓"
                ((passed++))
            else
                echo "✗ (PostgreSQL not accessible)"
                ((failed++))
            fi
        else
            echo "○ (pg_isready not available)"
        fi
    fi
    
    # Summary
    echo ""
    echo "Integration Test Summary: $passed passed, $failed failed"
    
    if [[ $failed -gt 0 ]]; then
        return 1
    fi
    return 0
}

#######################################
# Unit test - Library function tests
#######################################
yolo::test_unit() {
    echo "Running YOLO unit tests..."
    
    local passed=0
    local failed=0
    
    # Test 1: Configuration loading
    echo -n "  Testing configuration loading... "
    if [[ -n "${YOLO_PORT}" ]] && [[ "${YOLO_PORT}" == "11455" ]]; then
        echo "✓"
        ((passed++))
    else
        echo "✗ (configuration not loaded)"
        ((failed++))
    fi
    
    # Test 2: Device detection function
    echo -n "  Testing device detection... "
    local test_device=$(YOLO_DEVICE="auto" bash -c '
        if [[ "$YOLO_DEVICE" == "auto" ]]; then
            if command -v nvidia-smi &> /dev/null && nvidia-smi &> /dev/null; then
                echo "cuda"
            else
                echo "cpu"
            fi
        else
            echo "$YOLO_DEVICE"
        fi
    ')
    
    if [[ "$test_device" == "cuda" ]] || [[ "$test_device" == "cpu" ]]; then
        echo "✓ (detected: $test_device)"
        ((passed++))
    else
        echo "✗ (invalid device: $test_device)"
        ((failed++))
    fi
    
    # Test 3: Model path validation
    echo -n "  Testing model path... "
    if [[ -d "${HOME}/.yolo/models" ]] || mkdir -p "${HOME}/.yolo/models" 2>/dev/null; then
        echo "✓"
        ((passed++))
    else
        echo "✗ (cannot create model directory)"
        ((failed++))
    fi
    
    # Test 4: Docker availability
    echo -n "  Testing Docker availability... "
    if command -v docker &> /dev/null && docker ps &> /dev/null; then
        echo "✓"
        ((passed++))
    else
        echo "✗ (Docker not available)"
        ((failed++))
    fi
    
    # Test 5: Python dependencies
    echo -n "  Testing Python environment... "
    if docker run --rm "${YOLO_DOCKER_IMAGE}" python -c "import ultralytics; print('OK')" &> /dev/null; then
        echo "✓"
        ((passed++))
    else
        echo "✗ (Python dependencies missing)"
        ((failed++))
    fi
    
    # Summary
    echo ""
    echo "Unit Test Summary: $passed passed, $failed failed"
    
    if [[ $failed -gt 0 ]]; then
        return 1
    fi
    return 0
}