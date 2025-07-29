#!/bin/bash

set -euo pipefail

echo "🧪 Minimal Ollama Test (no framework)"
echo "Resource: ollama"

# Simple health check
echo "🏥 Testing Ollama health..."
if curl -s http://localhost:11434/api/tags > /dev/null; then
    echo "✅ Ollama health check passed"
else
    echo "❌ Ollama health check failed"
    exit 1
fi

# Simple model test
echo "📚 Testing Ollama models..."
models=$(curl -s http://localhost:11434/api/tags | jq -r '.models[].name' | head -1)
if [[ -n "$models" ]]; then
    echo "✅ Found models: $models"
else
    echo "❌ No models found"
    exit 1
fi

echo "✅ Minimal Ollama test passed"
exit 0