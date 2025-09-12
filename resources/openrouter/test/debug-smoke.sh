#!/usr/bin/env bash

echo "Starting test..."

# Test: Check if CLI is accessible
if command -v resource-openrouter &>/dev/null; then
    echo "✓ CLI accessibility"
else
    echo "✗ CLI accessibility"
fi

# Test: Check help command
if resource-openrouter help >/dev/null 2>&1; then
    echo "✓ Help command"
else
    echo "✗ Help command"
fi

# Test: Check status command
if resource-openrouter status >/dev/null 2>&1; then
    echo "✓ Status command"
else
    echo "✗ Status command"
fi

echo "Test complete!"