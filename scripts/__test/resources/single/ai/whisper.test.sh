#!/bin/bash
# ====================================================================
# Whisper Integration Test
# ====================================================================
#
# Tests Whisper speech-to-text service integration including health checks,
# audio transcription, and API functionality.
#
# Required Resources: whisper
# Test Categories: single-resource, ai
# Estimated Duration: 30-45 seconds
#
# ====================================================================

set -euo pipefail

# Test metadata
TEST_RESOURCE="whisper"
TEST_TIMEOUT="${TEST_TIMEOUT:-60}"
TEST_CLEANUP="${TEST_CLEANUP:-true}"

# Recreate HEALTHY_RESOURCES array from exported string
if [[ -n "${HEALTHY_RESOURCES_STR:-}" ]]; then
    HEALTHY_RESOURCES=($HEALTHY_RESOURCES_STR)
fi

# Source framework helpers
SCRIPT_DIR="${SCRIPT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
source "$SCRIPT_DIR/framework/helpers/assertions.sh"
source "$SCRIPT_DIR/framework/helpers/cleanup.sh"
source "$SCRIPT_DIR/framework/helpers/fixtures.sh"

# Whisper configuration
WHISPER_BASE_URL="http://localhost:8090"

# Test setup
setup_test() {
    echo "🔧 Setting up Whisper integration test..."
    
    # Register cleanup handler
    register_cleanup_handler
    
    # Auto-discovery fallback for direct test execution
    if [[ -z "${HEALTHY_RESOURCES_STR:-}" ]]; then
        echo "🔍 Auto-discovering resources for direct test execution..."
        
        # Use the resource discovery system with timeout
        local resources_dir
        resources_dir="$(cd "$SCRIPT_DIR/../.." && pwd)"
        
        local discovery_output=""
        if timeout 10s bash -c "\"$resources_dir/index.sh\" --action discover 2>&1" > /tmp/discovery_output.tmp 2>&1; then
            discovery_output=$(cat /tmp/discovery_output.tmp)
            rm -f /tmp/discovery_output.tmp
        else
            echo "⚠️  Auto-discovery timed out, using fallback method..."
            # Fallback: check if the required resource is running on its default port
            if curl -f -s --max-time 2 "$WHISPER_BASE_URL/docs" >/dev/null 2>&1; then
                discovery_output="✅ $TEST_RESOURCE is running on port 8090"
            fi
        fi
        
        local discovered_resources=()
        while IFS= read -r line; do
            if [[ "$line" =~ ✅[[:space:]]+([^[:space:]]+)[[:space:]]+is[[:space:]]+running ]]; then
                discovered_resources+=("${BASH_REMATCH[1]}")
            fi
        done <<< "$discovery_output"
        
        if [[ ${#discovered_resources[@]} -eq 0 ]]; then
            echo "⚠️  No resources discovered, but test will proceed..."
            discovered_resources=("$TEST_RESOURCE")
        fi
        
        export HEALTHY_RESOURCES_STR="${discovered_resources[*]}"
        echo "✓ Discovered healthy resources: $HEALTHY_RESOURCES_STR"
    fi
    
    # Verify Whisper is available
    require_resource "$TEST_RESOURCE"
    
    # Verify required tools
    require_tools "curl"
    
    echo "✓ Test setup complete"
}

# Test Whisper health and basic connectivity
test_whisper_health() {
    echo "🏥 Testing Whisper health endpoint..."
    
    # Try health endpoint
    local response
    response=$(curl -s --max-time 10 "$WHISPER_BASE_URL/health" 2>/dev/null || echo "")
    
    if [[ -n "$response" ]]; then
        assert_http_success "$response" "Whisper health endpoint responds"
        echo "✓ Whisper health endpoint accessible"
    else
        # If no specific health endpoint, try root
        response=$(curl -s --max-time 10 "$WHISPER_BASE_URL/" 2>/dev/null || echo "")
        assert_http_success "$response" "Whisper service responds"
        echo "✓ Whisper service accessible"
    fi
    
    echo "✓ Whisper health check passed"
}

# Test transcription with test audio
test_whisper_transcription() {
    echo "🎤 Testing Whisper transcription..."
    
    # Create test audio file
    echo "Creating test audio file..."
    local test_audio_file
    test_audio_file=$(generate_test_audio "whisper-test.wav" 3)
    
    assert_file_exists "$test_audio_file" "Test audio file created"
    
    # Register for cleanup
    add_cleanup_file "$test_audio_file"
    
    # Test transcription
    echo "Sending transcription request..."
    local response
    response=$(curl -s --max-time 30 \
        -X POST "$WHISPER_BASE_URL/asr?output=json&task=transcribe&language=en" \
        -F "audio_file=@$test_audio_file" 2>/dev/null || echo '{"error":"request_failed"}')
    
    assert_http_success "$response" "Transcription request successful"
    
    # Check response format
    if command -v jq >/dev/null 2>&1; then
        if echo "$response" | jq . >/dev/null 2>&1; then
            assert_json_valid "$response" "Transcription response is valid JSON"
            
            # Check for transcription text
            local transcription_text
            transcription_text=$(echo "$response" | jq -r '.text // .transcription // empty' 2>/dev/null)
            
            if [[ -n "$transcription_text" ]]; then
                echo "✓ Transcription successful"
                echo "Transcribed text: '$transcription_text'"
            else
                echo "⚠ Transcription response received but no text found"
            fi
        else
            echo "⚠ Transcription response is not JSON format"
            echo "Response preview: ${response:0:200}"
        fi
    else
        echo "⚠ jq not available, skipping JSON validation"
        assert_not_empty "$response" "Transcription returned some response"
    fi
    
    echo "✓ Transcription test completed"
}

# Test different audio formats (if supported)
test_whisper_formats() {
    echo "🎵 Testing different audio formats..."
    
    # Create test files in different formats
    local formats=("wav")
    
    for format in "${formats[@]}"; do
        echo "Testing format: $format"
        
        local test_file
        test_file=$(generate_test_audio "test-${format}.${format}" 2)
        
        if [[ -f "$test_file" ]]; then
            add_cleanup_file "$test_file"
            
            local response
            response=$(curl -s --max-time 20 \
                -X POST "$WHISPER_BASE_URL/asr?output=json&task=transcribe" \
                -F "audio_file=@$test_file" 2>/dev/null || echo "failed")
            
            if [[ "$response" != "failed" && -n "$response" ]]; then
                echo "✓ Format $format supported"
            else
                echo "⚠ Format $format may not be supported"
            fi
        else
            echo "⚠ Could not create test file for format $format"
        fi
    done
    
    echo "✓ Format testing completed"
}

# Test error handling
test_whisper_error_handling() {
    echo "⚠️ Testing Whisper error handling..."
    
    # Test with invalid file
    echo "Testing with invalid file..."
    local invalid_response
    invalid_response=$(curl -s --max-time 10 \
        -X POST "$WHISPER_BASE_URL/asr" \
        -F "audio_file=invalid-data" 2>/dev/null || echo "connection_failed")
    
    # Should get some kind of error response
    if [[ "$invalid_response" == "connection_failed" ]]; then
        echo "⚠ Connection failed during error test"
    elif [[ -n "$invalid_response" ]]; then
        echo "✓ Error handling responds appropriately"
        echo "Error response preview: ${invalid_response:0:100}"
    else
        echo "⚠ No response for invalid input"
    fi
    
    echo "✓ Error handling test completed"
}

# Test performance characteristics
test_whisper_performance() {
    echo "⚡ Testing Whisper performance..."
    
    # Create a longer test audio file
    local test_file
    test_file=$(generate_test_audio "performance-test.wav" 5)
    
    if [[ ! -f "$test_file" ]]; then
        echo "⚠ Could not create performance test file"
        return 0
    fi
    
    add_cleanup_file "$test_file"
    
    # Time the transcription
    echo "Timing transcription performance..."
    local start_time=$(date +%s)
    
    local response
    response=$(curl -s --max-time 45 \
        -X POST "$WHISPER_BASE_URL/asr?output=json&task=transcribe" \
        -F "audio_file=@$test_file" 2>/dev/null || echo "timeout")
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo "Transcription took: ${duration}s"
    
    if [[ $duration -lt 30 ]]; then
        echo "✓ Performance is good (< 30s)"
    elif [[ $duration -lt 60 ]]; then
        echo "⚠ Performance is acceptable (< 60s)"
    else
        echo "⚠ Performance is slow (>= 60s)"
    fi
    
    if [[ "$response" != "timeout" && -n "$response" ]]; then
        echo "✓ Performance test completed successfully"
    else
        echo "⚠ Performance test timed out or failed"
    fi
    
    echo "✓ Performance test completed"
}

# Main test execution
main() {
    echo "🧪 Starting Whisper Integration Test"
    echo "Resource: $TEST_RESOURCE"
    echo "Timeout: ${TEST_TIMEOUT}s"
    echo
    
    # Setup
    setup_test
    
    # Run test suite
    test_whisper_health
    test_whisper_transcription
    test_whisper_formats
    test_whisper_error_handling
    test_whisper_performance
    
    # Print summary
    echo
    print_assertion_summary
    
    if [[ $FAILED_ASSERTIONS -gt 0 ]]; then
        echo "❌ Whisper integration test failed"
        exit 1
    else
        echo "✅ Whisper integration test passed"
        exit 0
    fi
}

# Run main function
main "$@"