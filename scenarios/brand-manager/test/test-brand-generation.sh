#!/bin/bash
# Test brand generation workflow
set -euo pipefail

# Source shared resource utilities
SCRIPT_DIR="$(cd "${BASH_SOURCE[0]%/*}/.." && pwd)"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/scripts/lib/resource-utils.sh"

# Test configuration
API_PORT="${SERVICE_PORT:-$(get_resource_port "brand-manager")}"
API_HOST=$(get_resource_hostname "brand-manager")
API_BASE="${BRAND_MANAGER_API_BASE:-http://$API_HOST:$API_PORT}"
TEST_BRAND_NAME="TestBrand-$(date +%s)"
TEST_INDUSTRY="technology"

echo "🧪 Testing brand generation workflow..."
echo "📡 API Base: $API_BASE"
echo "🏷️ Test Brand: $TEST_BRAND_NAME"

# Test API health
echo "🩺 Checking API health..."
if ! curl -sf "$API_BASE/health" >/dev/null; then
    echo "❌ API health check failed" >&2
    exit 1
fi
echo "✅ API is healthy"

# Test brand creation
echo "🎨 Creating test brand..."
create_response=$(curl -sf -X POST "$API_BASE/api/brands" \
    -H "Content-Type: application/json" \
    -d '{
        "brand_name": "'"$TEST_BRAND_NAME"'",
        "industry": "'"$TEST_INDUSTRY"'",
        "template": "modern-tech",
        "logo_style": "minimalist",
        "color_scheme": "primary"
    }' 2>/dev/null)

if [[ $? -ne 0 ]]; then
    echo "❌ Brand creation failed" >&2
    exit 1
fi

echo "✅ Brand creation request submitted"
echo "📄 Response: $create_response"

# Test listing brands
echo "📋 Testing brand listing..."
if ! curl -sf "$API_BASE/api/brands?limit=5" >/dev/null; then
    echo "❌ Brand listing failed" >&2
    exit 1
fi
echo "✅ Brand listing works"

# Test services endpoint
echo "🔗 Testing services endpoint..."
if ! curl -sf "$API_BASE/api/services" >/dev/null; then
    echo "❌ Services endpoint failed" >&2
    exit 1
fi
echo "✅ Services endpoint works"

echo "🎉 Brand generation workflow test completed successfully!"