#!/usr/bin/env bash
set -euo pipefail

# Cleanup script for remaining test directories across resources

RESOURCES_DIR="/home/matthalloran8/Vrooli/scripts/resources"

echo "🧹 Cleaning up remaining test directories"
echo

# Windmill - remove fixtures and test_fixtures
if [ -d "$RESOURCES_DIR/automation/windmill/fixtures" ] || [ -d "$RESOURCES_DIR/automation/windmill/test_fixtures" ]; then
    echo "📦 Windmill:"
    rm -rf "$RESOURCES_DIR/automation/windmill/fixtures" 2>/dev/null && echo "   ✓ Removed fixtures/"
    rm -rf "$RESOURCES_DIR/automation/windmill/test_fixtures" 2>/dev/null && echo "   ✓ Removed test_fixtures/"
fi

# Node-RED - remove test_fixtures
if [ -d "$RESOURCES_DIR/automation/node-red/test_fixtures" ]; then
    echo "📦 Node-RED:"
    rm -rf "$RESOURCES_DIR/automation/node-red/test_fixtures" && echo "   ✓ Removed test_fixtures/"
fi

# Huginn - remove test-fixtures and test_fixtures
if [ -d "$RESOURCES_DIR/automation/huginn/test-fixtures" ] || [ -d "$RESOURCES_DIR/automation/huginn/test_fixtures" ]; then
    echo "📦 Huginn:"
    rm -rf "$RESOURCES_DIR/automation/huginn/test-fixtures" 2>/dev/null && echo "   ✓ Removed test-fixtures/"
    rm -rf "$RESOURCES_DIR/automation/huginn/test_fixtures" 2>/dev/null && echo "   ✓ Removed test_fixtures/"
fi

# Whisper - check if tests directory has unique content
if [ -d "$RESOURCES_DIR/ai/whisper/tests" ]; then
    echo "📦 Whisper:"
    # Check for audio files that might be unique
    if [ -n "$(find "$RESOURCES_DIR/ai/whisper/tests" -name "*.wav" -o -name "*.mp3" 2>/dev/null)" ]; then
        echo "   ⚠️  Found audio files in tests/ - preserving for manual review"
    else
        rm -rf "$RESOURCES_DIR/ai/whisper/tests" && echo "   ✓ Removed tests/"
    fi
fi

# ComfyUI - remove test directory
if [ -d "$RESOURCES_DIR/ai/comfyui/test" ]; then
    echo "📦 ComfyUI:"
    rm -rf "$RESOURCES_DIR/ai/comfyui/test" && echo "   ✓ Removed test/"
fi

# Claude-code - check sandbox/test-files
if [ -d "$RESOURCES_DIR/agents/claude-code/sandbox/test-files" ]; then
    echo "📦 Claude-code:"
    # This might contain example sandbox files, check if it's just examples
    if [ -f "$RESOURCES_DIR/agents/claude-code/sandbox/test-files/example.js" ]; then
        echo "   ℹ️  Keeping sandbox/test-files/ (contains sandbox examples)"
    fi
fi

echo
echo "✅ Test directory cleanup complete!"
echo
echo "📊 Summary:"
echo "   - Removed redundant test/fixture directories"
echo "   - Preserved Python unit tests in agent-s2"
echo "   - Moved unique PDFs to centralized fixtures"
echo "   - Kept sandbox examples in claude-code"