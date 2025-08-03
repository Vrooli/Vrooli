#!/usr/bin/env bash
set -euo pipefail

# Agent-S2 Cleanup Script
# Removes redundant test directories and backups while preserving Python unit tests

AGENT_S2_DIR="/home/matthalloran8/Vrooli/scripts/resources/agents/agent-s2"

echo "🧹 Agent-S2 Cleanup"
echo "   Location: $AGENT_S2_DIR"
echo

# Remove backups directory
if [ -d "$AGENT_S2_DIR/backups" ]; then
    echo "🗑️  Removing backups directory..."
    rm -rf "$AGENT_S2_DIR/backups"
    echo "   ✓ Removed backups/"
fi

# Remove empty testing directories
echo "🗑️  Removing empty testing directories..."
for dir in "$AGENT_S2_DIR/testing" "$AGENT_S2_DIR/docker/compose/testing" "$AGENT_S2_DIR/examples/01-getting-started/testing"; do
    if [ -d "$dir" ]; then
        # Check if directory only contains test-outputs or is empty
        if [ -z "$(find "$dir" -type f -not -path "*/test-outputs/*" 2>/dev/null)" ]; then
            rm -rf "$dir"
            echo "   ✓ Removed empty: $(basename $(dirname "$dir"))/$(basename "$dir")"
        fi
    fi
done

# Keep tests/ directory as it contains actual Python unit tests
echo "📦 Preserving Python unit tests in tests/ directory"
if [ -d "$AGENT_S2_DIR/tests" ]; then
    echo "   Found $(find "$AGENT_S2_DIR/tests" -name "*.py" | wc -l) Python test files"
fi

echo
echo "✅ Agent-S2 cleanup complete!"