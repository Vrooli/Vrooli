#!/usr/bin/env bash
# Test script for the new AI task functionality

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

echo "Testing Agent-S2 AI task functionality..."
echo

# Test 1: Check if manage.sh accepts the new action
echo "1. Testing --action ai-task with help..."
"$SCRIPT_DIR/manage.sh" --help | grep -q "ai-task" && echo "✓ ai-task action is registered" || echo "✗ ai-task action not found"

# Test 2: Test error handling when no task is provided
echo
echo "2. Testing error handling (no task provided)..."
"$SCRIPT_DIR/manage.sh" --action ai-task 2>&1 | grep -q "Task description is required" && echo "✓ Proper error handling" || echo "✗ Error handling issue"

# Test 3: Show example command (won't execute without running container)
echo
echo "3. Example command that would execute if Agent-S2 is running:"
echo '   ./manage.sh --action ai-task --task "take a screenshot"'
echo
echo "To fully test, ensure Agent-S2 is installed and running:"
echo "   ./manage.sh --action install"
echo "   ./manage.sh --action start"
echo "   ./manage.sh --action ai-task --task \"go to google\""

echo
echo "Test script completed!"