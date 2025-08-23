#!/bin/bash
# Example: Basic chat completion with OpenRouter
# This example demonstrates how to use OpenRouter for a simple chat completion

# Get script directory
EXAMPLE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OPENROUTER_DIR="$(dirname "$EXAMPLE_DIR")"

# Source the OpenRouter core functions
source "${OPENROUTER_DIR}/lib/core.sh"

# Initialize OpenRouter (loads API key from Vault or environment)
if ! openrouter::init; then
    echo "Error: Failed to initialize OpenRouter. Please ensure API key is configured."
    echo "To configure:"
    echo "  1. Store in Vault: docker exec vault sh -c 'vault kv put secret/vrooli/openrouter api_key=<YOUR_KEY>'"
    echo "  2. Or set environment: export OPENROUTER_API_KEY=<YOUR_KEY>"
    exit 1
fi

# Example 1: Simple chat completion
echo "Example 1: Simple chat completion"
echo "---------------------------------"

response=$(curl -s -X POST \
    -H "Authorization: Bearer $OPENROUTER_API_KEY" \
    -H "Content-Type: application/json" \
    -d '{
        "model": "openai/gpt-3.5-turbo",
        "messages": [
            {
                "role": "user",
                "content": "What is the capital of France? Answer in one word."
            }
        ],
        "max_tokens": 10
    }' \
    "${OPENROUTER_API_BASE}/chat/completions")

if echo "$response" | jq -e '.choices[0].message.content' > /dev/null 2>&1; then
    answer=$(echo "$response" | jq -r '.choices[0].message.content')
    echo "Q: What is the capital of France?"
    echo "A: $answer"
else
    echo "Error: Failed to get response"
    echo "$response" | jq '.' 2>/dev/null || echo "$response"
fi

echo ""

# Example 2: Multi-turn conversation
echo "Example 2: Multi-turn conversation"
echo "----------------------------------"

conversation_response=$(curl -s -X POST \
    -H "Authorization: Bearer $OPENROUTER_API_KEY" \
    -H "Content-Type: application/json" \
    -d '{
        "model": "openai/gpt-3.5-turbo",
        "messages": [
            {
                "role": "system",
                "content": "You are a helpful assistant that speaks like a pirate."
            },
            {
                "role": "user",
                "content": "Hello! How are you?"
            }
        ],
        "max_tokens": 50
    }' \
    "${OPENROUTER_API_BASE}/chat/completions")

if echo "$conversation_response" | jq -e '.choices[0].message.content' > /dev/null 2>&1; then
    pirate_response=$(echo "$conversation_response" | jq -r '.choices[0].message.content')
    echo "System: You are a helpful assistant that speaks like a pirate."
    echo "User: Hello! How are you?"
    echo "Assistant: $pirate_response"
else
    echo "Error: Failed to get response"
    echo "$conversation_response" | jq '.' 2>/dev/null || echo "$conversation_response"
fi

echo ""

# Example 3: List available models
echo "Example 3: Available models (first 5)"
echo "-------------------------------------"

if command -v openrouter::list_models > /dev/null 2>&1; then
    models=$(openrouter::list_models | head -5)
    if [ -n "$models" ]; then
        echo "$models"
    else
        echo "No models found or API key not valid"
    fi
else
    echo "List models function not available"
fi

echo ""
echo "Note: This example requires a valid OpenRouter API key."
echo "Get your API key at: https://openrouter.ai/keys"