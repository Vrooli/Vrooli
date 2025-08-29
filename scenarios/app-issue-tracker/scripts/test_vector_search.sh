#!/usr/bin/env bash

# Test vector search functionality

API_BASE="http://localhost:8090"
QDRANT_URL="http://localhost:6333"

echo "Testing vector search setup..."

# Test 1: Check Qdrant health
echo "1. Checking Qdrant health..."
if curl -s "${QDRANT_URL}/collections" | grep -q "result"; then
    echo "✓ Qdrant is running"
else
    echo "✗ Qdrant is not running"
fi

# Test 2: Check collection exists
echo "2. Checking collection exists..."
if curl -s "${QDRANT_URL}/collections/issue_embeddings" | grep -q "result"; then
    echo "✓ Collection exists"
else
    echo "✗ Collection does not exist"
fi

# Test 3: Count points in collection
echo "3. Checking points in collection..."
POINT_COUNT=$(curl -s "${QDRANT_URL}/collections/issue_embeddings" | grep -o '"points_count":[0-9]*' | grep -o '[0-9]*')
echo "Found $POINT_COUNT points in collection"

# Test 4: Test API search endpoint (if available)
echo "4. Testing API vector search..."
if curl -s "${API_BASE}/api/search/vector?q=login+error&limit=5" | grep -q "success"; then
    echo "✓ Vector search API is working"
else
    echo "⚠ Vector search API not available or not working"
fi

echo "Vector search test completed."
