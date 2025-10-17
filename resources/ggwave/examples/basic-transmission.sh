#!/bin/bash
# GGWave Basic Transmission Example
# Demonstrates encoding and decoding data through sound

set -euo pipefail

# Configuration
GGWAVE_PORT="${GGWAVE_PORT:-8196}"
API_BASE="http://localhost:${GGWAVE_PORT}"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "========================================="
echo "GGWave Basic Transmission Example"
echo "========================================="
echo ""

# Check if service is running
echo -n "Checking GGWave service... "
if curl -sf "${API_BASE}/health" &>/dev/null; then
    echo -e "${GREEN}✓ Running${NC}"
else
    echo "✗ Not running"
    echo "Please start the service: vrooli resource ggwave manage start"
    exit 1
fi

echo ""
echo -e "${BLUE}1. Encoding data to audio signal${NC}"
echo "----------------------------------------"

# Encode some data
TEST_DATA="Hello from GGWave!"
echo "Data to transmit: ${TEST_DATA}"
echo ""

ENCODE_RESPONSE=$(curl -sf -X POST "${API_BASE}/api/encode" \
    -H "Content-Type: application/json" \
    -d "{\"data\": \"${TEST_DATA}\", \"mode\": \"normal\"}")

if [[ -n "$ENCODE_RESPONSE" ]]; then
    echo "Encoding successful!"
    echo "Response:"
    echo "$ENCODE_RESPONSE" | python3 -m json.tool
    
    # Extract the encoded audio
    ENCODED_AUDIO=$(echo "$ENCODE_RESPONSE" | python3 -c "import json, sys; print(json.load(sys.stdin)['audio'])")
else
    echo "Encoding failed!"
    exit 1
fi

echo ""
echo -e "${BLUE}2. Simulating audio transmission${NC}"
echo "----------------------------------------"
echo "[In real usage, the audio would play through speakers]"
echo "[and be captured by a microphone on another device]"
sleep 1

echo ""
echo -e "${BLUE}3. Decoding audio back to data${NC}"
echo "----------------------------------------"

DECODE_RESPONSE=$(curl -sf -X POST "${API_BASE}/api/decode" \
    -H "Content-Type: application/json" \
    -d "{\"audio\": \"${ENCODED_AUDIO}\", \"mode\": \"auto\"}")

if [[ -n "$DECODE_RESPONSE" ]]; then
    echo "Decoding successful!"
    echo "Response:"
    echo "$DECODE_RESPONSE" | python3 -m json.tool
    
    # Extract the decoded data
    DECODED_DATA=$(echo "$DECODE_RESPONSE" | python3 -c "import json, sys; print(json.load(sys.stdin)['data'])")
    
    echo ""
    echo -e "${GREEN}Transmission complete!${NC}"
    echo "Original data: ${TEST_DATA}"
    echo "Decoded data:  ${DECODED_DATA}"
    
    if [[ "$TEST_DATA" == "$DECODED_DATA" ]]; then
        echo -e "${GREEN}✓ Data matches!${NC}"
    else
        echo "✗ Data mismatch!"
    fi
else
    echo "Decoding failed!"
    exit 1
fi

echo ""
echo "========================================="
echo "Example completed successfully!"
echo "========================================="