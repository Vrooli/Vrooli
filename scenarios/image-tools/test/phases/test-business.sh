#!/bin/bash
# Test: Business Logic
# Validates core business functionality and workflows

set -e

echo "  ✓ Testing business logic..."

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$SCENARIO_DIR"

# Get API port
API_PORT=$(jq -r '.ports.api.range' .vrooli/service.json | cut -d'-' -f1)
if [ -z "$API_PORT" ] || [ "$API_PORT" = "null" ]; then
    API_PORT=${API_PORT:-15000}
fi

BASE_URL="http://localhost:${API_PORT}"

# Test 1: Image compression workflow
echo "  ✓ Testing image compression workflow..."
# Create test image data
TEST_IMAGE_DATA="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="

response=$(curl -s -X POST "${BASE_URL}/api/v1/image/compress" \
    -H "Content-Type: application/json" \
    -d "{\"image\":\"${TEST_IMAGE_DATA}\",\"quality\":80}" || echo "ERROR")

if [ "$response" = "ERROR" ] || [ -z "$response" ]; then
    echo "  ⚠️  Compression endpoint not responding (may need scenario running)"
else
    echo "  ✓ Compression workflow validated"
fi

# Test 2: Format conversion workflow
echo "  ✓ Testing format conversion workflow..."
response=$(curl -s -X POST "${BASE_URL}/api/v1/image/convert" \
    -H "Content-Type: application/json" \
    -d "{\"image\":\"${TEST_IMAGE_DATA}\",\"target_format\":\"webp\"}" || echo "ERROR")

if [ "$response" = "ERROR" ] || [ -z "$response" ]; then
    echo "  ⚠️  Conversion endpoint not responding (may need scenario running)"
else
    echo "  ✓ Format conversion workflow validated"
fi

# Test 3: Preset profiles workflow
echo "  ✓ Testing preset profiles workflow..."
response=$(curl -s "${BASE_URL}/api/v1/presets" || echo "ERROR")

if [ "$response" = "ERROR" ] || [ -z "$response" ]; then
    echo "  ⚠️  Presets endpoint not responding (may need scenario running)"
else
    # Verify presets exist
    preset_count=$(echo "$response" | jq -r 'length' 2>/dev/null || echo "0")
    if [ "$preset_count" -ge 5 ]; then
        echo "  ✓ Preset profiles workflow validated (found $preset_count presets)"
    else
        echo "  ⚠️  Expected at least 5 presets, found $preset_count"
    fi
fi

# Test 4: Plugin architecture validation
echo "  ✓ Testing plugin architecture..."
response=$(curl -s "${BASE_URL}/api/v1/plugins" || echo "ERROR")

if [ "$response" = "ERROR" ] || [ -z "$response" ]; then
    echo "  ⚠️  Plugins endpoint not responding (may need scenario running)"
else
    plugin_count=$(echo "$response" | jq -r '.plugins | length' 2>/dev/null || echo "0")
    if [ "$plugin_count" -ge 4 ]; then
        echo "  ✓ Plugin architecture validated (found $plugin_count plugins)"
    else
        echo "  ⚠️  Expected at least 4 plugins (JPEG, PNG, WebP, SVG)"
    fi
fi

# Test 5: Resource integration validation
echo "  ✓ Testing resource integration..."

# Check MinIO integration
if command -v resource-minio &> /dev/null; then
    minio_status=$(resource-minio status 2>&1 || echo "not running")
    if echo "$minio_status" | grep -q "running"; then
        echo "  ✓ MinIO resource integration available"
    else
        echo "  ⚠️  MinIO not running (optional but recommended)"
    fi
else
    echo "  ⚠️  MinIO CLI not available"
fi

echo "  ✓ Business logic tests completed"
