#!/bin/bash

echo "Testing new API endpoints..."
echo "=================================="

API_PORT="${API_PORT:-30150}"
API_URL="http://localhost:${API_PORT}"

echo "1. Testing /api/logs/scenario-improver endpoint..."
curl -s "${API_URL}/api/logs/scenario-improver" | jq '.' || echo "❌ Logs endpoint failed"
echo ""

echo "2. Testing /api/fixes/scenario-improver endpoint..."
curl -s "${API_URL}/api/fixes/scenario-improver" | jq '.' || echo "❌ Fixes endpoint failed" 
echo ""

echo "3. Testing /api/debug endpoint with errors debug type..."
curl -s -X POST "${API_URL}/api/debug" \
  -H "Content-Type: application/json" \
  -d '{
    "app_name": "scenario-improver",
    "debug_type": "errors",
    "include_errors": true,
    "include_logs": false,
    "include_performance": false
  }' | jq '.' || echo "❌ Debug endpoint failed"
echo ""

echo "4. Testing /api/debug endpoint with performance debug type..."
curl -s -X POST "${API_URL}/api/debug" \
  -H "Content-Type: application/json" \
  -d '{
    "app_name": "scenario-improver", 
    "debug_type": "performance",
    "include_errors": false,
    "include_logs": false,
    "include_performance": true
  }' | jq '.' || echo "❌ Debug performance endpoint failed"
echo ""

echo "Testing complete!"