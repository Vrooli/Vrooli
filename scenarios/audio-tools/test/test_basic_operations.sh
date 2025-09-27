#!/bin/bash

# Test basic audio operations for audio-tools scenario

API_URL="http://localhost:${API_PORT:-19603}/api"
TEST_DIR="/tmp/audio-tools-test-$$"
mkdir -p "$TEST_DIR"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

test_count=0
pass_count=0
fail_count=0

# Test helper functions
test_start() {
    echo -n "Testing $1... "
    ((test_count++))
}

test_pass() {
    echo -e "${GREEN}PASS${NC}"
    ((pass_count++))
}

test_fail() {
    echo -e "${RED}FAIL${NC}: $1"
    ((fail_count++))
}

# Create test audio file
create_test_file() {
    ffmpeg -f lavfi -i "sine=frequency=440:duration=5" -c:a pcm_s16le "$TEST_DIR/test.wav" -y &>/dev/null
}

# Test health endpoint
test_health() {
    test_start "Health endpoint"
    response=$(curl -sf "$API_URL/health")
    if [ $? -eq 0 ] && echo "$response" | grep -q "healthy"; then
        test_pass
    else
        test_fail "Health check failed"
    fi
}

# Test trim operation
test_trim() {
    test_start "Trim operation"
    response=$(curl -sf -X POST "$API_URL/v1/audio/edit" \
        -H "Content-Type: application/json" \
        -d "{
            \"audio_file\": \"$TEST_DIR/test.wav\",
            \"operations\": [
                {
                    \"type\": \"trim\",
                    \"parameters\": {
                        \"start_time\": 1.0,
                        \"end_time\": 3.0
                    }
                }
            ],
            \"output_format\": \"wav\"
        }")
    
    if [ $? -eq 0 ] && echo "$response" | grep -q "output_files"; then
        test_pass
    else
        test_fail "Trim operation failed"
    fi
}

# Test format conversion
test_convert() {
    test_start "Format conversion"
    response=$(curl -sf -X POST "$API_URL/v1/audio/convert" \
        -F "audio=@$TEST_DIR/test.wav" \
        -F "format=mp3" \
        -F "bitrate=192k")
    
    if [ $? -eq 0 ] && echo "$response" | grep -q "file_path"; then
        test_pass
    else
        test_fail "Conversion failed"
    fi
}

# Test metadata extraction
test_metadata() {
    test_start "Metadata extraction"
    response=$(curl -sf -X POST "$API_URL/v1/audio/metadata" \
        -F "audio=@$TEST_DIR/test.wav")
    
    if [ $? -eq 0 ] && echo "$response" | grep -q "duration"; then
        test_pass
    else
        test_fail "Metadata extraction failed"
    fi
}

# Test volume adjustment
test_volume() {
    test_start "Volume adjustment"
    response=$(curl -sf -X POST "$API_URL/v1/audio/edit" \
        -H "Content-Type: application/json" \
        -d "{
            \"audio_file\": \"$TEST_DIR/test.wav\",
            \"operations\": [
                {
                    \"type\": \"volume\",
                    \"parameters\": {
                        \"volume_factor\": 0.5
                    }
                }
            ],
            \"output_format\": \"wav\"
        }")
    
    if [ $? -eq 0 ] && echo "$response" | grep -q "output_files"; then
        test_pass
    else
        test_fail "Volume adjustment failed"
    fi
}

# Test normalize operation
test_normalize() {
    test_start "Normalize operation"
    response=$(curl -sf -X POST "$API_URL/v1/audio/edit" \
        -H "Content-Type: application/json" \
        -d "{
            \"audio_file\": \"$TEST_DIR/test.wav\",
            \"operations\": [
                {
                    \"type\": \"normalize\",
                    \"parameters\": {
                        \"target_level\": -16.0
                    }
                }
            ],
            \"output_format\": \"wav\"
        }")
    
    if [ $? -eq 0 ] && echo "$response" | grep -q "output_files"; then
        test_pass
    else
        test_fail "Normalize operation failed"
    fi
}

# Main test execution
main() {
    echo "=== Audio Tools Basic Operations Test ==="
    echo "API URL: $API_URL"
    echo
    
    # Create test file
    echo "Creating test audio file..."
    create_test_file
    
    # Run tests
    test_health
    test_trim
    test_convert
    test_metadata
    test_volume
    test_normalize
    
    # Cleanup
    rm -rf "$TEST_DIR"
    
    # Report results
    echo
    echo "=== Test Results ==="
    echo "Total tests: $test_count"
    echo -e "Passed: ${GREEN}$pass_count${NC}"
    echo -e "Failed: ${RED}$fail_count${NC}"
    
    if [ $fail_count -eq 0 ]; then
        echo -e "${GREEN}All tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}Some tests failed${NC}"
        exit 1
    fi
}

main "$@"