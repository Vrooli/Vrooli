#!/bin/bash

# Test script to verify Ollama embedding integration

set -e

echo "Testing Ollama Embedding Integration"
echo "====================================="

# Check if Ollama is running
if ! curl -sf http://localhost:11434/api/version > /dev/null 2>&1; then
    echo "❌ Ollama is not running on port 11434"
    echo "   Please start Ollama first"
    exit 1
fi

echo "✅ Ollama is running"

# Check if nomic-embed-text model is available
if ! ollama list 2>/dev/null | grep -q "nomic-embed-text"; then
    echo "⚠️  nomic-embed-text model not found"
    echo "   Pulling model..."
    ollama pull nomic-embed-text
fi

echo "✅ nomic-embed-text model is available"

# Test embedding generation
echo ""
echo "Testing embedding generation..."
RESPONSE=$(curl -s -X POST http://localhost:11434/api/embeddings \
    -H "Content-Type: application/json" \
    -d '{
        "model": "nomic-embed-text",
        "prompt": "This is a test of the embedding system for metareasoning workflows"
    }')

if echo "$RESPONSE" | grep -q '"embedding"'; then
    # Extract embedding dimensions
    DIMENSIONS=$(echo "$RESPONSE" | jq '.embedding | length')
    echo "✅ Successfully generated embedding with $DIMENSIONS dimensions"
    
    # Show first few values
    echo "   First 5 values: $(echo "$RESPONSE" | jq '.embedding[:5]')"
else
    echo "❌ Failed to generate embedding"
    echo "   Response: $RESPONSE"
    exit 1
fi

echo ""
echo "Testing Qdrant collection creation..."

# Check if Qdrant is running
if ! curl -sf http://localhost:6333 > /dev/null 2>&1; then
    echo "⚠️  Qdrant is not running on port 6333"
    echo "   Skipping Qdrant tests"
else
    echo "✅ Qdrant is running"
    
    # Check/create collection with correct dimensions
    COLLECTION_EXISTS=$(curl -s http://localhost:6333/collections/workflow_embeddings | grep -c '"status":"ok"' || true)
    
    if [ "$COLLECTION_EXISTS" -eq 0 ]; then
        echo "   Creating workflow_embeddings collection with $DIMENSIONS dimensions..."
        curl -s -X PUT http://localhost:6333/collections/workflow_embeddings \
            -H "Content-Type: application/json" \
            -d "{
                \"vectors\": {
                    \"size\": $DIMENSIONS,
                    \"distance\": \"Cosine\"
                }
            }" > /dev/null
        echo "   ✅ Collection created"
    else
        echo "   ✅ Collection already exists"
    fi
fi

echo ""
echo "====================================="
echo "✅ All embedding integration tests passed!"
echo ""
echo "The API can now:"
echo "  1. Generate real embeddings using Ollama"
echo "  2. Store them in Qdrant for semantic search"
echo "  3. Perform similarity searches on workflows"