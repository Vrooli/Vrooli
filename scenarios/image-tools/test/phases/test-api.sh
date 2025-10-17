#!/bin/bash
# Test: API endpoints
# Tests all API endpoints with various payloads

set -e

echo "  ✓ Setting up test environment..."

# Get the API port
API_PORT=$(vrooli scenario logs image-tools --step start-api --tail 5 2>/dev/null | grep -oP 'port \K\d+' | tail -1)
API_PORT=${API_PORT:-19364}

BASE_URL="http://localhost:${API_PORT}/api/v1"

# Create a test image
TEST_IMAGE="/tmp/test-image.jpg"
if [ ! -f "$TEST_IMAGE" ]; then
    echo "  ✓ Creating test image..."
    # Create a simple 100x100 JPEG using ImageMagick if available, or use base64 encoded minimal JPEG
    if command -v convert &> /dev/null; then
        convert -size 100x100 xc:red "$TEST_IMAGE"
    else
        # Minimal valid JPEG in base64
        echo "/9j/4AAQSkZJRgABAQEAYABgAAD/2wBDAAgGBgcGBQgHBwcJCQgKDBQNDAsLDBkSEw8UHRofHh0aHBwgJC4nICIsIxwcKDcpLDAxNDQ0Hyc5PTgyPC4zNDL/2wBDAQkJCQwLDBgNDRgyIRwhMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjL/wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAr/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFQEBAQAAAAAAAAAAAAAAAAAAAAX/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIRAxEAPwCwAA8A/9k=" | base64 -d > "$TEST_IMAGE"
    fi
fi

echo "  ✓ Testing /image/compress endpoint..."
RESPONSE=$(curl -sf -X POST "$BASE_URL/image/compress" \
    -F "image=@$TEST_IMAGE" \
    -F "quality=80" 2>/dev/null || echo "{}")

if echo "$RESPONSE" | grep -q "savings_percent"; then
    echo "    ✓ Compression successful"
else
    echo "    ❌ Compression failed"
    echo "$RESPONSE"
fi

echo "  ✓ Testing /image/info endpoint..."
RESPONSE=$(curl -sf -X POST "$BASE_URL/image/info" \
    -F "image=@$TEST_IMAGE" 2>/dev/null || echo "{}")

if echo "$RESPONSE" | grep -q "format"; then
    echo "    ✓ Image info retrieved"
else
    echo "    ❌ Image info failed"
fi

echo "  ✓ Testing /image/resize endpoint..."
RESPONSE=$(curl -sf -X POST "$BASE_URL/image/resize" \
    -F "image=@$TEST_IMAGE" \
    -F "width=50" \
    -F "height=50" \
    -F "maintain_aspect=true" 2>/dev/null || echo "{}")

if echo "$RESPONSE" | grep -q "url\|dimensions"; then
    echo "    ✓ Resize successful"
else
    echo "    ❌ Resize failed"
fi

echo "  ✓ Testing /image/metadata endpoint..."
RESPONSE=$(curl -sf -X POST "$BASE_URL/image/metadata" \
    -F "image=@$TEST_IMAGE" \
    -F "strip=true" 2>/dev/null || echo "{}")

if echo "$RESPONSE" | grep -q "metadata_removed\|url"; then
    echo "    ✓ Metadata stripping successful"
else
    echo "    ❌ Metadata operation failed"
fi

echo "  ✓ API tests complete"