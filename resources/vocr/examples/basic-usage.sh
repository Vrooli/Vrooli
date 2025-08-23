#\!/usr/bin/env bash
# VOCR Basic Usage Examples

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}VOCR Basic Usage Examples${NC}"
echo "========================="
echo ""

# 1. Check health
echo -e "${YELLOW}1. Checking VOCR health:${NC}"
curl -s http://localhost:9420/health | jq '.status'
echo ""

# 2. Capture a screenshot (mock)
echo -e "${YELLOW}2. Capturing screenshot:${NC}"
response=$(curl -s -X POST http://localhost:9420/capture \
    -H "Content-Type: application/json" \
    -d '{"region": "100,100,400,300"}')
echo "$response" | jq '.' 2>/dev/null || echo "$response"
echo ""

# 3. OCR text extraction (mock)
echo -e "${YELLOW}3. OCR text extraction:${NC}"
# Create a test image if possible
if command -v convert &>/dev/null; then
    convert -size 300x100 xc:white -font Helvetica -pointsize 24 \
            -draw "text 20,50 'Vrooli VOCR Test'" /tmp/vocr-test.png 2>/dev/null
    
    response=$(curl -s -X POST http://localhost:9420/ocr \
        -H "Content-Type: application/json" \
        -d '{"image": "/tmp/vocr-test.png"}')
    echo "$response" | jq '.' 2>/dev/null || echo "$response"
else
    echo "ImageMagick not installed, skipping OCR example"
fi
echo ""

# 4. Vision AI query (not yet implemented)
echo -e "${YELLOW}4. Vision AI query:${NC}"
response=$(curl -s -X POST http://localhost:9420/ask \
    -H "Content-Type: application/json" \
    -d '{"question": "What is on the screen?"}')
echo "$response" | jq '.' 2>/dev/null || echo "$response"
echo ""

echo -e "${GREEN}Example complete\!${NC}"
echo "To use VOCR in your workflows:"
echo "  - Health check: GET http://localhost:9420/health"
echo "  - Capture: POST http://localhost:9420/capture"
echo "  - OCR: POST http://localhost:9420/ocr"
echo "  - Vision: POST http://localhost:9420/ask"
