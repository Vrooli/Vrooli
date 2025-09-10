#!/usr/bin/env bash
# DeepStack Resource - Test Library Functions

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
readonly RESOURCE_DIR="$(dirname "${SCRIPT_DIR}")"

# Load configuration
source "${RESOURCE_DIR}/config/defaults.sh"

# Test helper functions
test_command_exists() {
    local cmd="$1"
    if command -v "$cmd" &> /dev/null; then
        echo "✓ Command exists: $cmd"
        return 0
    else
        echo "✗ Command not found: $cmd" >&2
        return 1
    fi
}

test_port_available() {
    local port="$1"
    if nc -z localhost "$port" 2>/dev/null; then
        echo "✗ Port $port is already in use" >&2
        return 1
    else
        echo "✓ Port $port is available"
        return 0
    fi
}

test_docker_running() {
    if docker info &> /dev/null; then
        echo "✓ Docker is running"
        return 0
    else
        echo "✗ Docker is not running" >&2
        return 1
    fi
}

test_container_exists() {
    local container="$1"
    if docker ps -a | grep -q "$container"; then
        echo "✓ Container exists: $container"
        return 0
    else
        echo "✗ Container not found: $container" >&2
        return 1
    fi
}

test_container_running() {
    local container="$1"
    if docker ps | grep -q "$container"; then
        echo "✓ Container running: $container"
        return 0
    else
        echo "✗ Container not running: $container" >&2
        return 1
    fi
}

test_api_endpoint() {
    local endpoint="$1"
    local expected_code="${2:-200}"
    
    local response_code=$(curl -s -o /dev/null -w "%{http_code}" "$endpoint" 2>/dev/null || echo "000")
    
    if [[ "$response_code" == "$expected_code" ]]; then
        echo "✓ API endpoint responding: $endpoint (${response_code})"
        return 0
    else
        echo "✗ API endpoint failed: $endpoint (got ${response_code}, expected ${expected_code})" >&2
        return 1
    fi
}

test_health_check() {
    local url="http://${DEEPSTACK_HOST}:${DEEPSTACK_PORT}/v1/vision/detection"
    
    if timeout "${DEEPSTACK_HEALTH_TIMEOUT}" curl -sf "$url" &> /dev/null; then
        echo "✓ Health check passed"
        return 0
    else
        echo "✗ Health check failed" >&2
        return 1
    fi
}

test_gpu_available() {
    if command -v nvidia-smi &> /dev/null && nvidia-smi &> /dev/null; then
        echo "✓ GPU available"
        local gpu_info=$(nvidia-smi --query-gpu=name,memory.total --format=csv,noheader | head -1)
        echo "  GPU: $gpu_info"
        return 0
    else
        echo "ℹ No GPU detected (CPU mode will be used)"
        return 1
    fi
}

test_redis_connection() {
    if [[ "${DEEPSTACK_REDIS_ENABLED}" != "true" ]]; then
        echo "ℹ Redis caching disabled"
        return 0
    fi
    
    if timeout 2 redis-cli -h "${DEEPSTACK_REDIS_HOST}" -p "${DEEPSTACK_REDIS_PORT}" ping &> /dev/null; then
        echo "✓ Redis connection successful"
        return 0
    else
        echo "⚠ Redis not available (caching disabled)" >&2
        return 1
    fi
}

test_directory_writable() {
    local dir="$1"
    local test_file="${dir}/.write_test_$$"
    
    if touch "$test_file" 2>/dev/null; then
        rm -f "$test_file"
        echo "✓ Directory writable: $dir"
        return 0
    else
        echo "✗ Directory not writable: $dir" >&2
        return 1
    fi
}

test_image_detection() {
    local test_image="${1:-}"
    
    if [[ -z "$test_image" ]] || [[ ! -f "$test_image" ]]; then
        echo "⚠ No test image provided, skipping detection test"
        return 2
    fi
    
    echo "Testing object detection with: $test_image"
    
    local response=$(curl -s -X POST "http://${DEEPSTACK_HOST}:${DEEPSTACK_PORT}/v1/vision/detection" \
        -F "image=@${test_image}" \
        -F "min_confidence=${DEEPSTACK_CONFIDENCE_THRESHOLD}" 2>/dev/null)
    
    if echo "$response" | jq -e '.success == true' &> /dev/null; then
        local count=$(echo "$response" | jq '.predictions | length')
        echo "✓ Detection successful: Found $count objects"
        return 0
    else
        echo "✗ Detection failed" >&2
        echo "Response: $response" >&2
        return 1
    fi
}

test_face_detection() {
    local test_image="${1:-}"
    
    if [[ -z "$test_image" ]] || [[ ! -f "$test_image" ]]; then
        echo "⚠ No test image provided, skipping face detection test"
        return 2
    fi
    
    echo "Testing face detection with: $test_image"
    
    local response=$(curl -s -X POST "http://${DEEPSTACK_HOST}:${DEEPSTACK_PORT}/v1/vision/face" \
        -F "image=@${test_image}" 2>/dev/null)
    
    if echo "$response" | jq -e '.success == true' &> /dev/null; then
        local count=$(echo "$response" | jq '.predictions | length')
        echo "✓ Face detection successful: Found $count faces"
        return 0
    else
        echo "✗ Face detection failed" >&2
        echo "Response: $response" >&2
        return 1
    fi
}

# Run basic validation tests
run_validation_tests() {
    local failures=0
    
    echo "Running DeepStack validation tests..."
    echo "===================================="
    
    # Check prerequisites
    test_command_exists "docker" || ((failures++))
    test_command_exists "curl" || ((failures++))
    test_command_exists "jq" || ((failures++))
    test_docker_running || ((failures++))
    
    # Check directories
    test_directory_writable "${DEEPSTACK_DATA_DIR}" || ((failures++))
    test_directory_writable "${DEEPSTACK_MODEL_DIR}" || ((failures++))
    test_directory_writable "${DEEPSTACK_TEMP_DIR}" || ((failures++))
    
    # Check optional components
    test_gpu_available || true
    test_redis_connection || true
    
    if [[ $failures -gt 0 ]]; then
        echo ""
        echo "✗ Validation failed with $failures errors" >&2
        return 1
    else
        echo ""
        echo "✓ All validation tests passed"
        return 0
    fi
}

# Run service tests
run_service_tests() {
    local failures=0
    
    echo "Running DeepStack service tests..."
    echo "=================================="
    
    # Check if service is running
    test_container_running "${DEEPSTACK_CONTAINER_NAME}" || ((failures++))
    
    # Check API endpoints
    test_api_endpoint "http://${DEEPSTACK_HOST}:${DEEPSTACK_PORT}/v1/vision/detection" 200 || ((failures++))
    test_health_check || ((failures++))
    
    # Check port binding
    if nc -z localhost "${DEEPSTACK_PORT}" 2>/dev/null; then
        echo "✓ Port ${DEEPSTACK_PORT} is bound"
    else
        echo "✗ Port ${DEEPSTACK_PORT} is not accessible" >&2
        ((failures++))
    fi
    
    if [[ $failures -gt 0 ]]; then
        echo ""
        echo "✗ Service tests failed with $failures errors" >&2
        return 1
    else
        echo ""
        echo "✓ All service tests passed"
        return 0
    fi
}

# Export functions for use by test scripts
export -f test_command_exists
export -f test_port_available
export -f test_docker_running
export -f test_container_exists
export -f test_container_running
export -f test_api_endpoint
export -f test_health_check
export -f test_gpu_available
export -f test_redis_connection
export -f test_directory_writable
export -f test_image_detection
export -f test_face_detection
export -f run_validation_tests
export -f run_service_tests