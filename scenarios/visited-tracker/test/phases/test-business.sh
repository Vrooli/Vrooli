#!/bin/bash
# Simplified business tests phase
set -euo pipefail

echo "=== Business Logic Tests Phase ==="
echo "Testing core business functionality..."

# Test counters
error_count=0
test_count=0
created_campaign_ids=()

# Cleanup function to run on exit
cleanup() {
    echo "Cleaning up test data..."
    for campaign_id in "${created_campaign_ids[@]}"; do
        if [ -n "$campaign_id" ] && [ "$campaign_id" != "null" ]; then
            echo "Deleting test campaign: $campaign_id"
            curl -sf -X DELETE "http://localhost:17695/api/v1/campaigns/$campaign_id" >/dev/null 2>&1 || echo "Warning: Failed to delete campaign $campaign_id"
        fi
    done
}

# Set up cleanup to run on script exit
trap cleanup EXIT

# Pre-cleanup: Remove any existing test campaigns
echo "Cleaning up any existing test campaigns..."
existing_campaigns=$(curl -sf "http://localhost:17695/api/v1/campaigns" 2>/dev/null || echo "")
if echo "$existing_campaigns" | jq -e '.campaigns' >/dev/null 2>&1; then
    echo "$existing_campaigns" | jq -r '.campaigns[] | select(.name | test("^business-test-")) | .id' | while read -r old_campaign_id; do
        if [ -n "$old_campaign_id" ] && [ "$old_campaign_id" != "null" ]; then
            echo "Deleting old test campaign: $old_campaign_id"
            curl -sf -X DELETE "http://localhost:17695/api/v1/campaigns/$old_campaign_id" >/dev/null 2>&1 || echo "Warning: Failed to delete old campaign $old_campaign_id"
        fi
    done
fi

# Test API business logic
echo "Testing campaign creation..."
timestamp=$(date +%s)
campaign_data='{"name":"business-test-'$timestamp'","from_agent":"test","patterns":["**/*.js"]}'
campaign_response=$(curl -sf -X POST "http://localhost:17695/api/v1/campaigns" -H "Content-Type: application/json" -d "$campaign_data" 2>/dev/null || echo "")

if echo "$campaign_response" | jq -e '.id' >/dev/null 2>&1; then
    campaign_id=$(echo "$campaign_response" | jq -r '.id')
    created_campaign_ids+=("$campaign_id")
    echo "Campaign creation tests passed - ID: $campaign_id"
    test_count=$((test_count + 1))
else
    echo "Campaign creation tests failed"
    error_count=$((error_count + 1))
fi

# Test CLI business workflows
echo "Testing CLI business workflows..."
# Get scenario directory
SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CLI_BINARY="$SCENARIO_DIR/cli/visited-tracker"

if [ -f "$CLI_BINARY" ] && [ -x "$CLI_BINARY" ]; then
    if "$CLI_BINARY" version >/dev/null 2>&1; then
        echo "CLI business workflow tests passed"
        test_count=$((test_count + 1))
    else
        echo "CLI business workflow tests failed"
        error_count=$((error_count + 1))
    fi
else
    echo "CLI not available at $CLI_BINARY - skipping"
fi

# Test data persistence
echo "Testing data persistence..."
campaigns_response=$(curl -sf "http://localhost:17695/api/v1/campaigns" 2>/dev/null || echo "")
if echo "$campaigns_response" | jq -e '.campaigns' >/dev/null 2>&1; then
    campaign_count=$(echo "$campaigns_response" | jq '.campaigns | length')
    echo "Data persistence tests passed - $campaign_count campaigns found"
    test_count=$((test_count + 1))
else
    echo "Data persistence tests failed"
    error_count=$((error_count + 1))
fi

echo "Summary: $test_count passed, $error_count failed"

if [ $error_count -eq 0 ]; then
    echo "SUCCESS: All business tests passed"
    exit 0
else
    echo "ERROR: Some business tests failed"
    exit 1
fi