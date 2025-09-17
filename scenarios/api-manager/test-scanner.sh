#!/bin/bash

# Test script for the vulnerability scanner

API_URL="http://localhost:${API_PORT}/api/v1"

echo "Testing API Manager Vulnerability Scanner"
echo "========================================="
echo ""

# Test 1: Quick scan on test-vulnerabilities directory
echo "Test 1: Running quick scan on test files..."
curl -X POST "$API_URL/scenarios/api-manager/scan" \
  -H "Content-Type: application/json" \
  -d '{"type": "quick"}' \
  2>/dev/null | jq '.'

echo ""
echo "Test 2: Running full scan on test files..."
curl -X POST "$API_URL/scenarios/api-manager/scan" \
  -H "Content-Type: application/json" \
  -d '{"type": "full"}' \
  2>/dev/null | jq '.'

echo ""
echo "Test 3: Retrieving vulnerabilities..."
curl "$API_URL/vulnerabilities?scenario=api-manager" \
  2>/dev/null | jq '.'

echo ""
echo "Test complete!"