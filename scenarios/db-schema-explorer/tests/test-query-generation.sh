#!/bin/bash

# Test script for query generation functionality

set -euo pipefail

API_BASE="http://localhost:${API_PORT:-15000}/api/v1"

echo "Testing Database Schema Explorer Query Generation..."

# Test 1: Generate simple SELECT query
echo "Test 1: Simple SELECT query"
response=$(curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "natural_language": "show all users",
    "database_context": "main",
    "include_explanation": true
  }' \
  "${API_BASE}/query/generate")

if echo "$response" | grep -q '"success":true'; then
  echo "✓ Query generation successful"
  echo "$response" | jq -r '.sql' 2>/dev/null || echo "$response"
else
  echo "✗ Query generation failed"
  echo "$response"
  exit 1
fi

# Test 2: Generate complex JOIN query
echo -e "\nTest 2: Complex JOIN query"
response=$(curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "natural_language": "show all projects with their owners",
    "database_context": "main",
    "include_explanation": true
  }' \
  "${API_BASE}/query/generate")

if echo "$response" | grep -q '"success":true'; then
  echo "✓ Complex query generation successful"
  if echo "$response" | grep -qi 'join'; then
    echo "✓ Generated query contains JOIN"
  fi
else
  echo "✗ Complex query generation failed"
fi

# Test 3: Generate aggregation query
echo -e "\nTest 3: Aggregation query"
response=$(curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "natural_language": "count tables with more than 10 columns",
    "database_context": "main",
    "include_explanation": true
  }' \
  "${API_BASE}/query/generate")

if echo "$response" | grep -q '"success":true'; then
  echo "✓ Aggregation query generation successful"
  if echo "$response" | grep -qi 'count'; then
    echo "✓ Generated query contains COUNT"
  fi
else
  echo "✗ Aggregation query generation failed"
fi

echo -e "\n✅ All query generation tests completed!"