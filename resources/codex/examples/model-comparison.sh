#!/usr/bin/env bash
# Model Comparison Example for Codex Resource
# Shows how to use different OpenAI models and compare their outputs

echo "=== OpenAI Model Comparison Demo ==="
echo "Testing code generation with different models"
echo

# The prompt we'll test with each model
PROMPT="Write a Python function that finds the nth Fibonacci number using dynamic programming with memoization"

echo "Prompt: $PROMPT"
echo "=" 
echo

# Test with gpt-4o-mini (default, cheapest)
echo "1. GPT-4o-mini (Best Value - $0.15/1M tokens)"
echo "----------------------------------------"
CODEX_DEFAULT_MODEL="gpt-4o-mini" resource-codex content execute "$PROMPT"
echo

# Test with gpt-4o (more capable)
echo "2. GPT-4o (Flagship - $2.50/1M tokens)"
echo "----------------------------------------"
CODEX_DEFAULT_MODEL="gpt-4o" resource-codex content execute "$PROMPT"
echo

# Test with o1-mini (reasoning model)
echo "3. O1-mini (Reasoning Model)"
echo "----------------------------------------"
CODEX_DEFAULT_MODEL="o1-mini" resource-codex content execute "$PROMPT"
echo

echo "=== Cost Comparison ==="
echo "For this simple request (~500 tokens):"
echo "- gpt-4o-mini: ~$0.00008 (0.008 cents)"
echo "- gpt-4o:      ~$0.00125 (0.125 cents)"
echo "- o1-mini:     ~$0.00300 (0.3 cents)"
echo
echo "gpt-4o-mini is 94% cheaper than gpt-4o for similar quality!"