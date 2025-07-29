#!/bin/bash

# Simple test to verify ollama functionality without the framework
echo "=== Simple Ollama Test ==="

# Test ollama health
echo "Testing Ollama health..."
if curl -s http://localhost:11434/api/tags > /dev/null; then
    echo "✅ Ollama is accessible"
else
    echo "❌ Ollama is not accessible"
    exit 1
fi

# Test basic generation
echo "Testing Ollama generation..."
response=$(curl -s -X POST http://localhost:11434/api/generate \
    -H "Content-Type: application/json" \
    -d '{"model": "llama2:7b", "prompt": "Hello", "stream": false}' \
    --max-time 30)

if [[ -n "$response" ]] && echo "$response" | grep -q "response"; then
    echo "✅ Ollama generation works"
    echo "Response excerpt: $(echo "$response" | jq -r '.response' | head -c 50)..."
else
    echo "❌ Ollama generation failed"
    echo "Response: $response"
    exit 1
fi

echo "✅ All tests passed!"