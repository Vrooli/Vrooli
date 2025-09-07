#!/bin/bash
# Health check script for Resource Generator

# Check for required environment variable
if [ -z "$API_PORT" ]; then
    echo "Error: API_PORT environment variable is not set"
    echo "Usage: API_PORT=<port> $0"
    exit 1
fi

API_HOST="${API_HOST:-localhost}"
API_URL="http://${API_HOST}:${API_PORT}/health"

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "Checking Resource Generator health at ${API_HOST}:${API_PORT}..."

# Check API health
if curl -sf "$API_URL" > /dev/null 2>&1; then
    HEALTH_DATA=$(curl -s "$API_URL")
    echo -e "${GREEN}✅ API is healthy${NC}"
    
    # Parse and display key metrics
    if command -v jq >/dev/null 2>&1; then
        STATUS=$(echo "$HEALTH_DATA" | jq -r '.data.status')
        PENDING=$(echo "$HEALTH_DATA" | jq -r '.data.pending_items')
        PROCESSING=$(echo "$HEALTH_DATA" | jq -r '.data.processing_active')
        
        echo "  Status: $STATUS"
        echo "  Pending items: $PENDING"
        echo "  Processing active: $PROCESSING"
    else
        echo "$HEALTH_DATA"
    fi
    exit 0
else
    echo -e "${RED}❌ API is not responding at ${API_HOST}:${API_PORT}${NC}"
    echo "Please ensure the Resource Generator is running:"
    echo "  API_PORT=<port> vrooli scenario resource-generator start"
    exit 1
fi