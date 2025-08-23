#!/usr/bin/env bash
# Test script for image generation endpoint

set -euo pipefail

# Configuration
API_BASE="${1:-http://localhost:${SERVICE_PORT:-8090}}"
TEST_CAMPAIGN_ID="test-campaign-123"
TEST_BRAND_ID="test-brand-456"

echo "🧪 Testing Image Generation Pipeline API endpoints..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test health endpoint
echo -e "${BLUE}Testing health endpoint...${NC}"
if response=$(curl -sf "$API_BASE/health"); then
    echo -e "${GREEN}✅ Health endpoint working${NC}"
    echo "$response" | jq '.' 2>/dev/null || echo "$response"
else
    echo -e "${RED}❌ Health endpoint failed${NC}"
    exit 1
fi

# Create test brand
echo -e "${BLUE}Creating test brand...${NC}"
brand_data=$(cat << EOF
{
    "name": "Test Brand API",
    "description": "Brand created by API test",
    "colors": ["#FF0000", "#00FF00", "#0000FF"],
    "fonts": ["Arial", "Helvetica", "Times"]
}
EOF
)

if brand_response=$(curl -sf -X POST \
    -H "Content-Type: application/json" \
    -d "$brand_data" \
    "$API_BASE/api/brands"); then
    echo -e "${GREEN}✅ Brand creation successful${NC}"
    brand_id=$(echo "$brand_response" | jq -r '.id' 2>/dev/null || echo "$TEST_BRAND_ID")
    echo "Brand ID: $brand_id"
else
    echo -e "${RED}❌ Brand creation failed${NC}"
    exit 1
fi

# Create test campaign
echo -e "${BLUE}Creating test campaign...${NC}"
campaign_data=$(cat << EOF
{
    "name": "Test Campaign API",
    "brand_id": "$brand_id",
    "description": "Campaign created by API test",
    "status": "active"
}
EOF
)

if campaign_response=$(curl -sf -X POST \
    -H "Content-Type: application/json" \
    -d "$campaign_data" \
    "$API_BASE/api/campaigns"); then
    echo -e "${GREEN}✅ Campaign creation successful${NC}"
    campaign_id=$(echo "$campaign_response" | jq -r '.id' 2>/dev/null || echo "$TEST_CAMPAIGN_ID")
    echo "Campaign ID: $campaign_id"
else
    echo -e "${RED}❌ Campaign creation failed${NC}"
    exit 1
fi

# Test image generation
echo -e "${BLUE}Testing image generation endpoint...${NC}"
generation_data=$(cat << EOF
{
    "campaign_id": "$campaign_id",
    "prompt": "A beautiful sunset over mountains, professional photography style",
    "style": "photorealistic",
    "dimensions": "1024x1024",
    "metadata": {
        "test": true,
        "source": "api_test"
    }
}
EOF
)

if generation_response=$(curl -sf -X POST \
    -H "Content-Type: application/json" \
    -d "$generation_data" \
    "$API_BASE/api/generate"); then
    echo -e "${GREEN}✅ Image generation endpoint working${NC}"
    generation_id=$(echo "$generation_response" | jq -r '.id' 2>/dev/null || echo "unknown")
    echo "Generation ID: $generation_id"
    echo "$generation_response" | jq '.' 2>/dev/null || echo "$generation_response"
else
    echo -e "${RED}❌ Image generation endpoint failed${NC}"
    exit 1
fi

# Test generations list
echo -e "${BLUE}Testing generations list endpoint...${NC}"
if generations_response=$(curl -sf "$API_BASE/api/generations?campaign_id=$campaign_id"); then
    echo -e "${GREEN}✅ Generations list endpoint working${NC}"
    echo "$generations_response" | jq '.' 2>/dev/null || echo "$generations_response"
else
    echo -e "${RED}❌ Generations list endpoint failed${NC}"
    exit 1
fi

# Test voice brief endpoint (with dummy data)
echo -e "${BLUE}Testing voice brief endpoint...${NC}"
voice_data=$(cat << EOF
{
    "audio_data": "$(echo 'dummy audio data' | base64 -w 0)",
    "format": "wav"
}
EOF
)

# Note: This will likely fail because we don't have Whisper running, but we test the endpoint structure
if curl -sf -X POST \
    -H "Content-Type: application/json" \
    -d "$voice_data" \
    "$API_BASE/api/voice-brief" >/dev/null 2>&1; then
    echo -e "${GREEN}✅ Voice brief endpoint accessible${NC}"
else
    echo -e "${YELLOW}⚠️  Voice brief endpoint not accessible (Whisper service may not be running)${NC}"
fi

echo -e "${GREEN}🎉 All accessible API endpoint tests completed successfully!${NC}"