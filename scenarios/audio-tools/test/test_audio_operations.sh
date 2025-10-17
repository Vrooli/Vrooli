#!/bin/bash
# Audio Tools Integration Tests

set -e

API_PORT="${API_PORT:-19588}"
API_URL="http://localhost:${API_PORT}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "Starting Audio Tools Integration Tests..."
echo "API URL: $API_URL"

# Function to check test result
check_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $2"
        return 0
    else
        echo -e "${RED}✗${NC} $2"
        return 1
    fi
}

# Create a test audio file using ffmpeg
create_test_audio() {
    echo "Creating test audio file..."
    ffmpeg -f lavfi -i "sine=frequency=440:duration=5" -y /tmp/test_audio.wav 2>/dev/null
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓${NC} Test audio file created"
    else
        echo -e "${RED}✗${NC} Failed to create test audio file"
        exit 1
    fi
}

# Test 1: Health Check
test_health() {
    echo -e "\n${YELLOW}Test 1: Health Check${NC}"
    RESPONSE=$(curl -sf "${API_URL}/health" 2>/dev/null)
    RESULT=$?
    check_result $RESULT "Health check endpoint"
    
    if [ $RESULT -eq 0 ]; then
        echo "$RESPONSE" | grep -q '"status":"healthy"'
        check_result $? "Health status is healthy"
        
        echo "$RESPONSE" | grep -q '"ffmpeg":"available"'
        check_result $? "FFmpeg is available"
    fi
}

# Test 2: API Status
test_status() {
    echo -e "\n${YELLOW}Test 2: API Status${NC}"
    RESPONSE=$(curl -sf "${API_URL}/api/v1/status" 2>/dev/null)
    RESULT=$?
    check_result $RESULT "Status endpoint"
    
    if [ $RESULT -eq 0 ]; then
        echo "$RESPONSE" | grep -q '"service":"audio-tools"'
        check_result $? "Service identification"
        
        echo "$RESPONSE" | grep -q '"capabilities"'
        check_result $? "Capabilities listed"
    fi
}

# Test 3: Trim Operation
test_trim() {
    echo -e "\n${YELLOW}Test 3: Trim Operation${NC}"
    
    PAYLOAD='{
        "audio_file": "/tmp/test_audio.wav",
        "operations": [{
            "type": "trim",
            "parameters": {
                "start_time": 1.0,
                "end_time": 3.0
            }
        }],
        "output_format": "wav"
    }'
    
    RESPONSE=$(curl -sf -X POST "${API_URL}/api/v1/audio/edit" \
        -H "Content-Type: application/json" \
        -d "$PAYLOAD" 2>/dev/null)
    RESULT=$?
    
    check_result $RESULT "Trim operation request"
    
    if [ $RESULT -eq 0 ]; then
        echo "$RESPONSE" | grep -q '"job_id"'
        check_result $? "Job ID returned"
    fi
}

# Test 4: Volume Adjustment
test_volume() {
    echo -e "\n${YELLOW}Test 4: Volume Adjustment${NC}"
    
    PAYLOAD='{
        "audio_file": "/tmp/test_audio.wav",
        "operations": [{
            "type": "volume",
            "parameters": {
                "volume_factor": 0.5
            }
        }],
        "output_format": "wav"
    }'
    
    RESPONSE=$(curl -sf -X POST "${API_URL}/api/v1/audio/edit" \
        -H "Content-Type: application/json" \
        -d "$PAYLOAD" 2>/dev/null)
    RESULT=$?
    
    check_result $RESULT "Volume adjustment request"
    
    if [ $RESULT -eq 0 ]; then
        echo "$RESPONSE" | grep -q '"job_id"'
        check_result $? "Job ID returned"
    fi
}

# Test 5: Fade Effects
test_fade() {
    echo -e "\n${YELLOW}Test 5: Fade Effects${NC}"
    
    PAYLOAD='{
        "audio_file": "/tmp/test_audio.wav",
        "operations": [{
            "type": "fade",
            "parameters": {
                "fade_in": 1.0,
                "fade_out": 1.0
            }
        }],
        "output_format": "wav"
    }'
    
    RESPONSE=$(curl -sf -X POST "${API_URL}/api/v1/audio/edit" \
        -H "Content-Type: application/json" \
        -d "$PAYLOAD" 2>/dev/null)
    RESULT=$?
    
    check_result $RESULT "Fade effects request"
    
    if [ $RESULT -eq 0 ]; then
        echo "$RESPONSE" | grep -q '"job_id"'
        check_result $? "Job ID returned"
    fi
}

# Test 6: Convert Format
test_convert() {
    echo -e "\n${YELLOW}Test 6: Format Conversion${NC}"
    
    PAYLOAD='{
        "input_file": "/tmp/test_audio.wav",
        "output_format": "mp3",
        "quality": {
            "bitrate": 192,
            "sample_rate": 44100
        }
    }'
    
    RESPONSE=$(curl -sf -X POST "${API_URL}/api/v1/audio/convert" \
        -H "Content-Type: application/json" \
        -d "$PAYLOAD" 2>/dev/null)
    RESULT=$?
    
    check_result $RESULT "Format conversion request"
}

# Test 7: API Documentation
test_docs() {
    echo -e "\n${YELLOW}Test 7: API Documentation${NC}"
    
    RESPONSE=$(curl -sf "${API_URL}/api/docs" 2>/dev/null)
    RESULT=$?
    check_result $RESULT "Documentation endpoint"
    
    if [ $RESULT -eq 0 ]; then
        echo "$RESPONSE" | grep -q '"endpoints"'
        check_result $? "Endpoints documented"
        
        echo "$RESPONSE" | grep -q '"supported_operations"'
        check_result $? "Operations documented"
    fi
}

# Test 8: Error Handling
test_error_handling() {
    echo -e "\n${YELLOW}Test 8: Error Handling${NC}"
    
    # Test with invalid file
    PAYLOAD='{
        "audio_file": "/tmp/nonexistent.wav",
        "operations": [{
            "type": "trim",
            "parameters": {
                "start_time": 1.0,
                "end_time": 3.0
            }
        }]
    }'
    
    RESPONSE=$(curl -s -X POST "${API_URL}/api/v1/audio/edit" \
        -H "Content-Type: application/json" \
        -d "$PAYLOAD" 2>/dev/null)
    
    # We expect this to fail gracefully
    echo "$RESPONSE" | grep -q '"error"'
    check_result $? "Error handling for invalid file"
}

# Main test execution
main() {
    echo "========================================="
    echo "Audio Tools Integration Test Suite"
    echo "========================================="
    
    # Create test audio file
    create_test_audio
    
    # Run tests
    test_health
    test_status
    test_trim
    test_volume
    test_fade
    test_convert
    test_docs
    test_error_handling
    
    echo -e "\n========================================="
    echo "Test Suite Complete"
    echo "========================================="
    
    # Clean up
    rm -f /tmp/test_audio.wav
}

main "$@"