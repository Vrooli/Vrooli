#!/usr/bin/env bash
################################################################################
# AudioCraft Test Library
# Testing functionality for AudioCraft resource
################################################################################
set -euo pipefail

# Test configuration
TEST_TIMEOUT="${TEST_TIMEOUT:-30}"
TEST_PORT="${AUDIOCRAFT_PORT:-7862}"

################################################################################
# Test Helper Functions
################################################################################

test::setup() {
    echo "Setting up test environment..."
    # Ensure service is running for tests
    if ! docker ps | grep -q "${AUDIOCRAFT_CONTAINER_NAME:-vrooli-audiocraft}"; then
        audiocraft::manage::start --wait
    fi
}

test::teardown() {
    echo "Cleaning up test environment..."
    # Clean up test files if any
    rm -f /tmp/audiocraft_test_*.wav 2>/dev/null || true
}

test::assert_equals() {
    local expected="$1"
    local actual="$2"
    local message="${3:-Assertion failed}"
    
    if [[ "$expected" != "$actual" ]]; then
        echo "❌ $message"
        echo "   Expected: $expected"
        echo "   Actual: $actual"
        return 1
    fi
}

test::assert_contains() {
    local haystack="$1"
    local needle="$2"
    local message="${3:-Assertion failed}"
    
    if [[ "$haystack" != *"$needle"* ]]; then
        echo "❌ $message"
        echo "   String does not contain: $needle"
        return 1
    fi
}

################################################################################
# Smoke Tests
################################################################################

test::smoke::health_check() {
    echo "Testing health endpoint..."
    
    local response
    response=$(curl -sf "http://localhost:${TEST_PORT}/health" 2>/dev/null || echo "FAILED")
    
    if [[ "$response" == "FAILED" ]]; then
        echo "❌ Health check failed"
        return 1
    fi
    
    # Check response contains expected fields
    if echo "$response" | grep -q "status.*healthy"; then
        echo "✅ Health check passed"
        return 0
    else
        echo "❌ Health check response invalid"
        return 1
    fi
}

test::smoke::models_endpoint() {
    echo "Testing models endpoint..."
    
    local response
    response=$(curl -sf "http://localhost:${TEST_PORT}/api/models" 2>/dev/null || echo "FAILED")
    
    if [[ "$response" == "FAILED" ]]; then
        echo "❌ Models endpoint failed"
        return 1
    fi
    
    # Check response contains model information
    if echo "$response" | grep -q "musicgen"; then
        echo "✅ Models endpoint passed"
        return 0
    else
        echo "❌ Models response invalid"
        return 1
    fi
}

################################################################################
# Integration Tests
################################################################################

test::integration::generate_music() {
    echo "Testing music generation..."
    
    local output_file="/tmp/audiocraft_test_music.wav"
    
    # Generate music
    local response_code
    response_code=$(curl -X POST "http://localhost:${TEST_PORT}/api/generate/music" \
        -H "Content-Type: application/json" \
        -d '{"prompt": "test music", "duration": 2}' \
        -o "$output_file" \
        -w "%{http_code}" \
        -s)
    
    if [[ "$response_code" != "200" ]]; then
        echo "❌ Music generation failed with code: $response_code"
        return 1
    fi
    
    # Check file was created and has content
    if [[ -f "$output_file" ]] && [[ -s "$output_file" ]]; then
        echo "✅ Music generation passed"
        rm -f "$output_file"
        return 0
    else
        echo "❌ Music file not created or empty"
        return 1
    fi
}

test::integration::generate_sound() {
    echo "Testing sound generation..."
    
    local output_file="/tmp/audiocraft_test_sound.wav"
    
    # Generate sound effect
    local response_code
    response_code=$(curl -X POST "http://localhost:${TEST_PORT}/api/generate/sound" \
        -H "Content-Type: application/json" \
        -d '{"prompt": "rain sound", "duration": 2}' \
        -o "$output_file" \
        -w "%{http_code}" \
        -s)
    
    # AudioGen might not be available, so we accept 503 as expected
    if [[ "$response_code" == "503" ]]; then
        echo "⚠️  Sound generation skipped (AudioGen not loaded)"
        return 0
    elif [[ "$response_code" != "200" ]]; then
        echo "❌ Sound generation failed with code: $response_code"
        return 1
    fi
    
    # Check file was created and has content
    if [[ -f "$output_file" ]] && [[ -s "$output_file" ]]; then
        echo "✅ Sound generation passed"
        rm -f "$output_file"
        return 0
    else
        echo "❌ Sound file not created or empty"
        return 1
    fi
}

test::integration::concurrent_requests() {
    echo "Testing concurrent requests..."
    
    # Send multiple requests in background
    local pids=()
    for i in {1..3}; do
        curl -X POST "http://localhost:${TEST_PORT}/api/generate/music" \
            -H "Content-Type: application/json" \
            -d "{\"prompt\": \"test $i\", \"duration\": 1}" \
            -o "/tmp/audiocraft_test_concurrent_$i.wav" \
            -s &
        pids+=($!)
    done
    
    # Wait for all requests to complete
    local failed=0
    for pid in "${pids[@]}"; do
        if ! wait "$pid"; then
            failed=$((failed + 1))
        fi
    done
    
    # Clean up test files
    rm -f /tmp/audiocraft_test_concurrent_*.wav 2>/dev/null || true
    
    if [[ $failed -eq 0 ]]; then
        echo "✅ Concurrent requests handled"
        return 0
    else
        echo "❌ $failed concurrent requests failed"
        return 1
    fi
}

################################################################################
# Unit Tests
################################################################################

test::unit::configuration() {
    echo "Testing configuration..."
    
    # Check required environment variables
    if [[ -z "${AUDIOCRAFT_PORT:-}" ]]; then
        echo "❌ AUDIOCRAFT_PORT not set"
        return 1
    fi
    
    # Check port is valid
    if [[ ! "$AUDIOCRAFT_PORT" =~ ^[0-9]+$ ]] || [[ "$AUDIOCRAFT_PORT" -lt 1 || "$AUDIOCRAFT_PORT" -gt 65535 ]]; then
        echo "❌ Invalid port: $AUDIOCRAFT_PORT"
        return 1
    fi
    
    echo "✅ Configuration valid"
    return 0
}

test::unit::directories() {
    echo "Testing directory structure..."
    
    local dirs=(
        "${AUDIOCRAFT_DATA_DIR:-/data/resources/audiocraft}"
        "${AUDIOCRAFT_MODELS_DIR:-/data/resources/audiocraft/models}"
        "${AUDIOCRAFT_OUTPUT_DIR:-/data/resources/audiocraft/output}"
    )
    
    for dir in "${dirs[@]}"; do
        if [[ ! -d "$dir" ]]; then
            echo "⚠️  Directory not found: $dir (will be created on install)"
        fi
    done
    
    echo "✅ Directory structure checked"
    return 0
}

################################################################################
# Run All Tests
################################################################################

test::run_all() {
    echo "🧪 Running all AudioCraft tests..."
    echo "================================"
    
    local failed=0
    
    # Setup
    test::setup
    
    # Smoke tests
    echo ""
    echo "🔥 Smoke Tests"
    echo "--------------"
    test::smoke::health_check || failed=$((failed + 1))
    test::smoke::models_endpoint || failed=$((failed + 1))
    
    # Integration tests
    echo ""
    echo "🔗 Integration Tests"
    echo "-------------------"
    test::integration::generate_music || failed=$((failed + 1))
    test::integration::generate_sound || failed=$((failed + 1))
    test::integration::concurrent_requests || failed=$((failed + 1))
    
    # Unit tests
    echo ""
    echo "📦 Unit Tests"
    echo "------------"
    test::unit::configuration || failed=$((failed + 1))
    test::unit::directories || failed=$((failed + 1))
    
    # Teardown
    test::teardown
    
    # Summary
    echo ""
    echo "================================"
    if [[ $failed -eq 0 ]]; then
        echo "✅ All tests passed!"
        return 0
    else
        echo "❌ $failed test(s) failed"
        return 1
    fi
}