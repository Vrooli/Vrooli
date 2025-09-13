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
        ((TESTS_PASSED++))
        return 0
    else
        echo -e "${RED}✗${NC}"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Create test image
create_test_image() {
    # Create a simple test image using ImageMagick convert or base64
    local test_image="/tmp/test_nsfw_image.jpg"
    
    # Create a minimal JPEG using base64 (1x1 pixel white image)
    echo "/9j/4AAQSkZJRgABAQEAYABgAAD/2wBDAAgGBgcGBQgHBwcJCQgKDBQNDAsLDBkSEw8UHRofHh0aHBwgJC4nICIsIxwcKDcpLDAxNDQ0Hyc5PTgyPC4zNDL/2wBDAQkJCQwLDBgNDRgyIRwhMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjL/wAARCAABAAEDASIAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAv/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFQEBAQAAAAAAAAAAAAAAAAAAAAX/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIRAxEAPwCwAA8A/9k=" | base64 -d > "$test_image"
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
    
    # Test 9: Service restart
    echo -e "${YELLOW}Testing service restart...${NC}"
    if ${CLI} manage restart --wait > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Service restart successful${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗ Service restart failed${NC}"
        ((TESTS_FAILED++))
    fi
    
    # Test 10: Health after restart
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