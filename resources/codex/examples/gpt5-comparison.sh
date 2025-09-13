#!/usr/bin/env bash
# GPT-5 Model Comparison Example for Codex Resource
# Shows the new GPT-5 models released August 2025

echo "=== GPT-5 Model Family Comparison ==="
echo "Released: August 7, 2025"
echo

# The prompt we'll test
PROMPT="Write a Python class for a binary search tree with insert, delete, and search methods"

echo "Prompt: $PROMPT"
echo "========================================="
echo

# Test with gpt-5-nano (new default, cheapest)
echo "1. GPT-5-nano (NEW DEFAULT - $0.05/1M input)"
echo "   Context: 400K tokens | Output: 128K tokens"
echo "----------------------------------------"
CODEX_DEFAULT_MODEL="gpt-5-nano" resource-codex content execute "$PROMPT"
echo

# Test with gpt-5-mini (mid-tier)
echo "2. GPT-5-mini ($0.25/1M input)"
echo "   Context: 400K tokens | Output: 128K tokens"
echo "----------------------------------------"
CODEX_DEFAULT_MODEL="gpt-5-mini" resource-codex content execute "$PROMPT"
echo

# Test with gpt-5 (flagship)
echo "3. GPT-5 Full ($1.25/1M input)"
echo "   Context: 400K tokens | Output: 128K tokens"
echo "----------------------------------------"
CODEX_DEFAULT_MODEL="gpt-5" resource-codex content execute "$PROMPT"
echo

echo "=== Cost & Performance Comparison ==="
echo
echo "GPT-5-nano vs Others (for 1000 token request):"
echo "- GPT-5-nano:  $0.00005 (5/1000 of a cent) ← NEW DEFAULT"
echo "- GPT-5-mini:  $0.00025 (1/4 of a cent) - 5x more"
echo "- GPT-5:       $0.00125 (1.25 cents) - 25x more"
echo "- GPT-4o-mini: $0.00015 (old default) - 3x more"
echo "- GPT-4o:      $0.00250 - 50x more!"
echo
echo "Context Windows:"
echo "- GPT-5 models: 400K input, 128K output (3x larger!)"
echo "- GPT-4o models: 128K input, 16K output"
echo
echo "GPT-5-nano gives you 3x the context at 1/3 the price of GPT-4o-mini!"