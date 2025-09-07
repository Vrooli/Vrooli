#!/bin/bash
# Test script to verify Claude Code integration

echo "Testing Claude Code integration for resource-improver"
echo "====================================================="

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

# Test 3: Test resource improvement prompt
echo "3. Testing resource improvement analysis prompt..."
ANALYSIS_PROMPT="Analyze the health check reliability of a hypothetical 'postgres' resource. Just respond with 'Analysis complete: postgres health check evaluated' and nothing else."
ANALYSIS_RESULT=$(resource-claude-code run --prompt "$ANALYSIS_PROMPT" --max-turns 1 2>&1)
if echo "$ANALYSIS_RESULT" | grep -q "Analysis complete"; then
    echo "   ✓ Resource analysis prompt works"
else
    echo "   ✗ Resource analysis prompt failed"
    echo "   Response: $ANALYSIS_RESULT"
fi

# Test 4: Check API can call Claude Code
echo "4. Testing resource-improver API integration..."
echo "   This would be tested when a queue item is processed"
echo "   API will call: resource-claude-code run --prompt <improvement_prompt> --max-turns 10"

echo ""
echo "Integration setup complete!"
echo "The resource-improver will now delegate all implementation work to Claude Code."