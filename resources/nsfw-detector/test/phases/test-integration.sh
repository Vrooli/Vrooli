#!/usr/bin/env bash
# Integration tests for NSFW Detector resource
# End-to-end functionality testing - must complete in <120s

set -euo pipefail

# Test configuration
readonly TIMEOUT=10
readonly PORT="${NSFW_DETECTOR_PORT:-11451}"
readonly BASE_URL="http://localhost:${PORT}"
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
readonly CLI="${SCRIPT_DIR}/cli.sh"

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly NC='\033[0m'

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Test helper function
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    echo -n "Testing $test_name... "
    
    if eval "$test_command" > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo -e "${RED}✗${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# Create test image
create_test_image() {
    # Create a simple test image using base64
    local test_image="/tmp/test_nsfw_image.jpg"
    
    # Create a minimal valid JPEG (1x1 red pixel)
    printf '\xff\xd8\xff\xe0\x00\x10\x4a\x46\x49\x46\x00\x01\x01\x00\x00\x01\x00\x01\x00\x00' > "$test_image"
    printf '\xff\xdb\x00\x43\x00\x03\x02\x02\x02\x02\x02\x03\x02\x02\x02\x03\x03\x03\x03\x04' >> "$test_image"
    printf '\x06\x04\x04\x04\x04\x04\x08\x06\x06\x05\x06\x09\x08\x0a\x0a\x09\x08\x09\x09\x0a' >> "$test_image"
    printf '\x0c\x0f\x0c\x0a\x0b\x0e\x0b\x09\x09\x0d\x11\x0d\x0e\x0f\x10\x10\x11\x10\x0a\x0c' >> "$test_image"
    printf '\x12\x13\x12\x10\x13\x0f\x10\x10\x10\xff\xc0\x00\x0b\x08\x00\x01\x00\x01\x01\x01' >> "$test_image"
    printf '\x11\x00\xff\xc4\x00\x14\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00' >> "$test_image"
    printf '\x00\x00\x00\x09\xff\xc4\x00\x14\x10\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00' >> "$test_image"
    printf '\x00\x00\x00\x00\x00\x00\xff\xda\x00\x08\x01\x01\x00\x00\x3f\x00\x2a\x1f\xff\xd9' >> "$test_image"
    
    echo "$test_image"
}

# Main integration tests
main() {
    echo "Running NSFW Detector integration tests..."
    echo "=========================================="
    
    # Ensure service is running
    if ! timeout $TIMEOUT curl -sf "${BASE_URL}/health" > /dev/null 2>&1; then
        echo -e "${RED}Service is not running. Start it first.${NC}"
        exit 1
    fi
    
    # Test 1: CLI help command
    run_test "CLI help command" \
        "${CLI} help | grep -q 'NSFW Detector'"
    
    # Test 2: CLI info command
    run_test "CLI info command" \
        "${CLI} info --json | jq -e '.port'"
    
    # Test 3: CLI status command
    run_test "CLI status command" \
        "${CLI} status | grep -q 'Status:'"
    
    # Test 4: Content list command
    run_test "content list models" \
        "${CLI} content list | grep -q 'nsfwjs'"
    
    # Test 5: Classification endpoint with test image
    local test_img=$(create_test_image)
    run_test "image classification" \
        "curl -sf -X POST ${BASE_URL}/classify -F 'image=@${test_img}' | jq -e '.safe'"
    
    # Test 6: Models endpoint returns array
    run_test "models endpoint array" \
        "timeout $TIMEOUT curl -sf ${BASE_URL}/models | jq -e '. | type == \"array\"'"
    
    # Test 7: Health check includes models
    run_test "health includes models" \
        "timeout $TIMEOUT curl -sf ${BASE_URL}/health | jq -e '.models | length > 0'"
    
    # Test 8: CLI content execute (basic)
    run_test "CLI content execute" \
        "${CLI} content execute --file ${test_img} | jq -e '.safe'"
    
    # Test 9: Configuration update
    run_test "configuration update" \
        "timeout $TIMEOUT curl -X POST -H 'Content-Type: application/json' \
            -d '{\"thresholds\": {\"adult\": 0.5}}' \
            ${BASE_URL}/config | jq -e '.thresholds.adult' | grep -q '0.5'"
    
    # Test 10: Model loading
    run_test "model loading" \
        "timeout $TIMEOUT curl -X POST -H 'Content-Type: application/json' \
            -d '{\"model\": \"nsfwjs\"}' \
            ${BASE_URL}/models/load | jq -e '.message'"
    
    # Test 11: Model unloading
    run_test "model unloading" \
        "timeout $TIMEOUT curl -X POST -H 'Content-Type: application/json' \
            -d '{\"model\": \"nsfwjs\"}' \
            ${BASE_URL}/models/unload | jq -e '.message'"
    
    # Test 12: Metrics tracking
    run_test "metrics tracking" \
        "timeout $TIMEOUT curl -sf ${BASE_URL}/metrics | jq -e '.requests_processed,.average_latency_ms'"
    
    # Test 13: Service restart
    echo -e "${YELLOW}Testing service restart...${NC}"
    if ${CLI} manage restart --wait > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Service restart successful${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "${RED}✗ Service restart failed${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
    
    # Test 14: Health after restart
    sleep 2
    run_test "health after restart" \
        "timeout $TIMEOUT curl -sf ${BASE_URL}/health"
    
    # Cleanup
    rm -f "$test_img"
    
    # Summary
    echo "=========================================="
    echo "Integration Test Results:"
    echo "  Passed: $TESTS_PASSED"
    echo "  Failed: $TESTS_FAILED"
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo -e "${GREEN}All integration tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}Some integration tests failed${NC}"
        exit 1
    fi
}

# Run tests
main "$@"