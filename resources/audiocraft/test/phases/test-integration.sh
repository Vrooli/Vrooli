#!/usr/bin/env bash
################################################################################
# AudioCraft Integration Tests
# End-to-end functionality validation
################################################################################
set -euo pipefail

# Get directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(dirname "$SCRIPT_DIR")"
RESOURCE_DIR="$(dirname "$TEST_DIR")"

# Load libraries
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/test.sh"

echo "🔗 AudioCraft Integration Tests"
echo "=============================="

# Track failures
FAILED=0

# Setup test environment
test::setup

# Test 1: Music generation
echo "Testing music generation..."
OUTPUT_FILE="/tmp/audiocraft_test_music_$(date +%s).wav"
RESPONSE_CODE=$(curl -X POST "http://localhost:${AUDIOCRAFT_PORT}/api/generate/music" \
    -H "Content-Type: application/json" \
    -d '{"prompt": "peaceful piano melody", "duration": 3}' \
    -o "$OUTPUT_FILE" \
    -w "%{http_code}" \
    -s)

if [[ "$RESPONSE_CODE" == "200" ]] && [[ -f "$OUTPUT_FILE" ]] && [[ -s "$OUTPUT_FILE" ]]; then
    echo "✅ Music generation PASSED"
    rm -f "$OUTPUT_FILE"
else
    echo "❌ Music generation FAILED (HTTP $RESPONSE_CODE)"
    FAILED=$((FAILED + 1))
fi

# Test 2: Sound effect generation
echo "Testing sound effect generation..."
OUTPUT_FILE="/tmp/audiocraft_test_sound_$(date +%s).wav"
RESPONSE_CODE=$(curl -X POST "http://localhost:${AUDIOCRAFT_PORT}/api/generate/sound" \
    -H "Content-Type: application/json" \
    -d '{"prompt": "dog barking", "duration": 2}' \
    -o "$OUTPUT_FILE" \
    -w "%{http_code}" \
    -s)

# AudioGen might not be available
if [[ "$RESPONSE_CODE" == "503" ]]; then
    echo "⚠️  Sound generation SKIPPED (AudioGen not loaded)"
elif [[ "$RESPONSE_CODE" == "200" ]] && [[ -f "$OUTPUT_FILE" ]] && [[ -s "$OUTPUT_FILE" ]]; then
    echo "✅ Sound generation PASSED"
    rm -f "$OUTPUT_FILE"
else
    echo "❌ Sound generation FAILED (HTTP $RESPONSE_CODE)"
    FAILED=$((FAILED + 1))
fi

# Test 3: Invalid request handling
echo "Testing error handling..."
RESPONSE=$(curl -X POST "http://localhost:${AUDIOCRAFT_PORT}/api/generate/music" \
    -H "Content-Type: application/json" \
    -d '{"prompt": "", "duration": 999}' \
    -s)

if echo "$RESPONSE" | grep -q "error"; then
    echo "✅ Error handling PASSED"
else
    echo "❌ Error handling FAILED"
    FAILED=$((FAILED + 1))
fi

# Test 4: Model listing
echo "Testing model listing..."
RESPONSE=$(curl -s "http://localhost:${AUDIOCRAFT_PORT}/api/models")

if echo "$RESPONSE" | grep -q "musicgen" && echo "$RESPONSE" | grep -q "variants"; then
    echo "✅ Model listing PASSED"
else
    echo "❌ Model listing FAILED"
    FAILED=$((FAILED + 1))
fi

# Test 5: Content management
echo "Testing content management..."
audiocraft::content::list > /dev/null 2>&1
if [[ $? -eq 0 ]]; then
    echo "✅ Content listing PASSED"
else
    echo "❌ Content listing FAILED"
    FAILED=$((FAILED + 1))
fi

# Cleanup
test::teardown

# Summary
echo "=============================="
if [[ $FAILED -eq 0 ]]; then
    echo "✅ All integration tests passed"
    exit 0
else
    echo "❌ $FAILED integration test(s) failed"
    exit 1
fi