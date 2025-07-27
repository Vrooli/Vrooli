#!/usr/bin/env bash
# Agent S2 Screenshot Demo
# Simple demonstration of screenshot capabilities

API_URL="http://localhost:4113"

echo "==================================="
echo "Agent S2 Screenshot Demo"
echo "==================================="

# Check if Agent S2 is running
echo -n "Checking Agent S2 health... "
if curl -s -f "$API_URL/health" > /dev/null 2>&1; then
    echo "✅ Healthy"
else
    echo "❌ Not running"
    echo "Start Agent S2 with: ./manage.sh --action start"
    exit 1
fi

# Take a screenshot
echo
echo "Taking screenshot..."
RESPONSE=$(curl -s -X POST "$API_URL/screenshot?format=png&quality=95")

if [ $? -eq 0 ]; then
    echo "✅ Screenshot captured successfully"
    
    # Extract image dimensions using jq if available
    if command -v jq > /dev/null 2>&1; then
        WIDTH=$(echo "$RESPONSE" | jq -r '.size.width // "unknown"')
        HEIGHT=$(echo "$RESPONSE" | jq -r '.size.height // "unknown"')
        echo "   Dimensions: ${WIDTH}x${HEIGHT}"
    fi
    
    # Save base64 data to file (if jq is available)
    if command -v jq > /dev/null 2>&1; then
        IMAGE_DATA=$(echo "$RESPONSE" | jq -r '.data // empty' | sed 's/^data:image\/[^;]*;base64,//')
        if [ -n "$IMAGE_DATA" ]; then
            # Ensure output directory exists
            mkdir -p ../test-outputs/screenshots
            echo "$IMAGE_DATA" | base64 -d > ../test-outputs/screenshots/agent-s2-screenshot.png
            echo "   Saved to: ../test-outputs/screenshots/agent-s2-screenshot.png"
        fi
    fi
else
    echo "❌ Screenshot failed"
fi

echo
echo "Other examples:"
echo "  # Take screenshot of specific region"
echo "  curl -X POST '$API_URL/screenshot?format=png&region=100,100,800,600'"
echo
echo "  # Save as JPEG with quality setting"
echo "  curl -X POST '$API_URL/screenshot?format=jpeg&quality=85'"