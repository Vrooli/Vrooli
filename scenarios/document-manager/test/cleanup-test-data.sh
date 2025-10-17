#!/usr/bin/env bash
# Cleanup test data from Document Manager database

set -euo pipefail

# Get API port dynamically
API_PORT="${API_PORT:-}"
if [ -z "$API_PORT" ]; then
    PID=$(pgrep -f "document-manager-api" | head -1)
    if [ -n "$PID" ]; then
        API_PORT=$(lsof -Pan -p "$PID" -i 2>/dev/null | awk '$1 ~ /document/ {print $9}' | cut -d: -f2 | head -1)
    fi
fi

# Fallback to default port
API_PORT="${API_PORT:-17777}"
API_URL="http://localhost:$API_PORT"

echo "🧹 Cleaning up test data from Document Manager"
echo "API URL: $API_URL"
echo

# Test API connection
if ! curl -sf "$API_URL/health" &>/dev/null; then
    echo "❌ Error: API is not responding at $API_URL"
    echo "Make sure the service is running with 'make run'"
    exit 1
fi

# Get all test applications (Integration Test App)
echo "📊 Finding test applications..."
TEST_APPS=$(curl -s "$API_URL/api/applications" | jq -r '.[] | select(.name == "Integration Test App") | .id')

if [ -z "$TEST_APPS" ]; then
    echo "✨ No test applications found"
else
    APP_COUNT=$(echo "$TEST_APPS" | wc -l)
    echo "Found $APP_COUNT test application(s) to delete"
    echo

    for app_id in $TEST_APPS; do
        echo "  🗑️  Deleting application: $app_id"
        if curl -X DELETE -sf "$API_URL/api/applications?id=$app_id" &>/dev/null; then
            echo "    ✅ Deleted successfully"
        else
            echo "    ⚠️  Failed to delete"
        fi
    done
fi

echo
echo "📊 Current database state:"
TOTAL_APPS=$(curl -s "$API_URL/api/applications" | jq 'length')
TOTAL_AGENTS=$(curl -s "$API_URL/api/agents" | jq 'length')
TOTAL_QUEUE=$(curl -s "$API_URL/api/queue" | jq 'length')

echo "  Applications: $TOTAL_APPS"
echo "  Agents: $TOTAL_AGENTS"
echo "  Queue items: $TOTAL_QUEUE"
echo
echo "✅ Cleanup complete!"
