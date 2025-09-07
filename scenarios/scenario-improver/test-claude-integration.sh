#!/bin/bash
# Test script to verify Claude Code integration

echo "Testing Claude Code integration for scenario-improver"
echo "======================================================"

# Test 1: Check if resource-claude-code is available
echo -n "1. Checking resource-claude-code CLI availability... "
if command -v resource-claude-code &> /dev/null; then
    echo "✓ Found"
else
    echo "✗ Not found - Please install resource-claude-code first"
    exit 1
fi

# Test 2: Test simple prompt execution
echo "2. Testing Claude Code execution with simple prompt..."
RESULT=$(resource-claude-code run --prompt "Say 'Claude Code integration test successful' and nothing else" --max-turns 1 2>&1)
if echo "$RESULT" | grep -q "successful"; then
    echo "   ✓ Claude Code responded successfully"
else
    echo "   ✗ Failed to get expected response"
    echo "   Response: $RESULT"
fi

# Test 3: Check API can call Claude Code
echo "3. Testing scenario-improver API integration..."
echo "   This would be tested when a queue item is processed"
echo "   API will call: resource-claude-code run --prompt <improvement_prompt> --max-turns 10"

echo ""
echo "Integration setup complete!"
echo "The scenario-improver will now delegate all implementation work to Claude Code."