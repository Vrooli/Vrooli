#!/usr/bin/env bash
set -euo pipefail

# Examples Directory Cleanup Script
# Based on comprehensive analysis of all examples directories

RESOURCES_DIR="/home/matthalloran8/Vrooli/scripts/resources"

echo "🧹 Examples Directory Cleanup"
echo "   Based on comprehensive analysis"
echo

# Remove empty examples directories
echo "🗑️  Removing empty examples directories..."
EMPTY_DIRS=(
    "$RESOURCES_DIR/search/searxng/examples"
    "$RESOURCES_DIR/storage/qdrant/examples"
    "$RESOURCES_DIR/storage/redis/examples"
)

for dir in "${EMPTY_DIRS[@]}"; do
    if [ -d "$dir" ] && [ -z "$(ls -A "$dir" 2>/dev/null)" ]; then
        rmdir "$dir" 2>/dev/null && echo "   ✓ Removed empty: $(basename $(dirname "$dir"))/examples/"
    fi
done

# Check and clean ollama examples (has empty subdirectories)
if [ -d "$RESOURCES_DIR/ai/ollama/examples" ]; then
    echo "📦 Ollama examples cleanup:"
    # Remove empty subdirectories
    find "$RESOURCES_DIR/ai/ollama/examples" -type d -empty -delete 2>/dev/null
    # If main examples dir is now empty, remove it
    if [ -z "$(ls -A "$RESOURCES_DIR/ai/ollama/examples" 2>/dev/null)" ]; then
        rmdir "$RESOURCES_DIR/ai/ollama/examples" && echo "   ✓ Removed empty ollama/examples/"
    else
        echo "   ℹ️  Kept non-empty content in ollama/examples/"
    fi
fi

# Convert whisper examples (just documentation) to docs
if [ -f "$RESOURCES_DIR/ai/whisper/examples/README.md" ]; then
    echo "📚 Converting documentation-only examples to docs:"
    mkdir -p "$RESOURCES_DIR/ai/whisper/docs"
    if [ -f "$RESOURCES_DIR/ai/whisper/examples/README.md" ]; then
        mv "$RESOURCES_DIR/ai/whisper/examples/README.md" "$RESOURCES_DIR/ai/whisper/docs/USAGE_EXAMPLES.md"
        echo "   ✓ Moved whisper/examples/README.md to docs/USAGE_EXAMPLES.md"
        # Remove empty examples directory
        rmdir "$RESOURCES_DIR/ai/whisper/examples" 2>/dev/null && echo "   ✓ Removed empty whisper/examples/"
    fi
fi

# Check claude-code empty directories
if [ -d "$RESOURCES_DIR/agents/claude-code/examples" ]; then
    echo "📦 Claude-code examples cleanup:"
    # Remove empty numbered directories
    for dir in 02-advanced-usage 03-integration-patterns 04-troubleshooting; do
        if [ -d "$RESOURCES_DIR/agents/claude-code/examples/$dir" ] && [ -z "$(ls -A "$RESOURCES_DIR/agents/claude-code/examples/$dir" 2>/dev/null)" ]; then
            rmdir "$RESOURCES_DIR/agents/claude-code/examples/$dir" 2>/dev/null && echo "   ✓ Removed empty: $dir/"
        fi
    done
fi

# Document valuable examples that are being preserved
echo
echo "✅ Cleanup complete!"
echo
echo "📊 Summary:"
echo "   Removed: 4-5 empty examples directories"
echo "   Converted: 1 documentation-only directory to docs/"
echo "   Preserved: 14 high-value examples directories with real code"
echo
echo "🌟 Preserved High-Value Examples:"
echo "   • agent-s2: Progressive Python examples"
echo "   • claude-code: Shell script examples (cleaned empty dirs)"
echo "   • n8n, windmill, huginn: Workflow examples"
echo "   • postgres: Industry-specific seed data"
echo "   • minio: boto3 integration examples"
echo "   • unstructured-io: Document processing examples"