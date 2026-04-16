#!/usr/bin/env bash

set -euo pipefail

echo "Example 1: Simple chat completion"
echo "---------------------------------"
resource-openrouter generate "What is the capital of France? Answer in one word."

echo ""
echo "Example 2: Multi-turn style prompt"
echo "----------------------------------"
cat <<'EOF' | resource-openrouter generate --model openai/gpt-4o-mini
You are a helpful assistant that speaks like a pirate.
User: Hello! How are you?
EOF

echo ""
echo "Example 3: Available models (first 5)"
echo "-------------------------------------"
resource-openrouter list-models | head -5

echo ""
echo "If credentials are not configured, run:"
echo "  resource-openrouter configure --api-key <key>"
echo "or export OPENROUTER_API_KEY in your shell."
