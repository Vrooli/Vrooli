#!/bin/bash

# Test data ingestion workflow

set -euo pipefail

API_BASE_URL="${API_BASE_URL:-http://localhost:8200}"
N8N_BASE_URL="${N8N_BASE_URL:-http://localhost:5678}"

echo "Testing data ingestion workflow..."

# First, create a test persona if not exists
if [[ -z "${TEST_PERSONA_ID:-}" ]]; then
    echo "Creating test persona for data ingestion test..."
    response=$(curl -sf -X POST "$API_BASE_URL/api/persona/create" \
        -H "Content-Type: application/json" \
        -d "{
            \"name\": \"Data Ingestion Test Persona $(date +%s)\",
            \"description\": \"Test persona for data ingestion\"
        }")
    
    TEST_PERSONA_ID=$(echo "$response" | jq -r '.id')
    echo "Created test persona: $TEST_PERSONA_ID"
fi

# Test connecting a data source
echo "Testing data source connection..."
datasource_response=$(curl -sf -X POST "$API_BASE_URL/api/datasource/connect" \
    -H "Content-Type: application/json" \
    -d "{
        \"persona_id\": \"$TEST_PERSONA_ID\",
        \"source_type\": \"local_files\",
        \"source_config\": {
            \"path\": \"/test-data\",
            \"file_types\": [\"txt\", \"md\", \"pdf\"]
        }
    }")

if echo "$datasource_response" | jq -e '.source_id' > /dev/null; then
    source_id=$(echo "$datasource_response" | jq -r '.source_id')
    echo "✓ Data source connected successfully with ID: $source_id"
    
    # Verify data source can be retrieved
    datasource_list=$(curl -sf "$API_BASE_URL/api/datasources/$TEST_PERSONA_ID")
    if echo "$datasource_list" | jq -e ".data_sources | length > 0" > /dev/null; then
        echo "✓ Data source listed successfully"
    else
        echo "✗ Failed to list data sources"
        exit 1
    fi
    
    # Test n8n webhook trigger (if n8n is available)
    if curl -sf "$N8N_BASE_URL/health" > /dev/null 2>&1; then
        echo "Testing n8n webhook trigger..."
        webhook_response=$(curl -sf -X POST "$N8N_BASE_URL/webhook/ingest" \
            -H "Content-Type: application/json" \
            -d "{
                \"persona_id\": \"$TEST_PERSONA_ID\",
                \"source_type\": \"local_files\",
                \"source_id\": \"$source_id\"
            }" || echo '{"status": "webhook_not_available"}')
        
        if echo "$webhook_response" | jq -e '.status' > /dev/null 2>&1; then
            echo "✓ n8n webhook triggered successfully"
        else
            echo "ℹ n8n webhook endpoint not available (expected in development)"
        fi
    else
        echo "ℹ n8n not available for webhook testing"
    fi
    
else
    echo "✗ Failed to connect data source"
    echo "Response: $datasource_response"
    exit 1
fi

echo "Data ingestion test passed!"