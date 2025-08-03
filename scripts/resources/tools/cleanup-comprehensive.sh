#!/usr/bin/env bash
set -euo pipefail

# Comprehensive Resource Cleanup Script
# Handles all remaining non-standard files and directories

RESOURCES_DIR="/home/matthalloran8/Vrooli/scripts/resources"

echo "🧹 Comprehensive Resource Cleanup"
echo "   Cleaning all remaining non-standard files"
echo

# Track changes
MOVED_COUNT=0
REMOVED_COUNT=0

# Function to move files to docs
move_to_docs() {
    local resource_path=$1
    local file=$2
    if [ -f "$resource_path/$file" ]; then
        mkdir -p "$resource_path/docs"
        mv "$resource_path/$file" "$resource_path/docs/"
        echo "   ✓ Moved $file to docs/"
        ((MOVED_COUNT++))
    fi
}

# Function to move scripts to lib
move_to_lib() {
    local resource_path=$1
    local file=$2
    if [ -f "$resource_path/$file" ]; then
        mkdir -p "$resource_path/lib"
        mv "$resource_path/$file" "$resource_path/lib/"
        echo "   ✓ Moved $file to lib/"
        ((MOVED_COUNT++))
    fi
}

echo "📦 AI Resources:"

# Ollama
if [ -d "$RESOURCES_DIR/ai/ollama" ]; then
    echo "  ollama:"
    cd "$RESOURCES_DIR/ai/ollama"
    move_to_docs . "EMBEDDING_MODELS.md"
    # capabilities.yaml can stay for now (will handle separately)
fi

# Automation Resources
echo
echo "📦 Automation Resources:"

# Huginn
if [ -d "$RESOURCES_DIR/automation/huginn" ]; then
    echo "  huginn:"
    cd "$RESOURCES_DIR/automation/huginn"
    move_to_lib . "run-tests.sh"
fi

# N8n
if [ -d "$RESOURCES_DIR/automation/n8n" ]; then
    echo "  n8n:"
    cd "$RESOURCES_DIR/automation/n8n"
    move_to_lib . "setup-api-access.sh"
fi

# Node-RED
if [ -d "$RESOURCES_DIR/automation/node-red" ]; then
    echo "  node-red:"
    cd "$RESOURCES_DIR/automation/node-red"
    move_to_lib . "run-tests.sh"
    # settings.js might be needed at root for Node-RED config
    if [ -f "settings.js" ]; then
        mkdir -p config
        mv settings.js config/
        echo "   ✓ Moved settings.js to config/"
        ((MOVED_COUNT++))
    fi
fi

# Agents
echo
echo "📦 Agent Resources:"

# Claude-code
if [ -d "$RESOURCES_DIR/agents/claude-code" ]; then
    echo "  claude-code:"
    cd "$RESOURCES_DIR/agents/claude-code"
    move_to_lib . "test_helper.bash"
fi

echo
echo "📊 Cleanup Summary:"
echo "   Files moved: $MOVED_COUNT"
echo "   Files removed: $REMOVED_COUNT"
echo
echo "📝 Notes:"
echo "   • Kept docker-related files (.dockerignore, docker-compose.yml) at root"
echo "   • Kept Python package files (setup.py, .env.example) at root"
echo "   • Kept capabilities.yaml files for now (resource discovery)"
echo "   • Moved documentation to docs/"
echo "   • Moved scripts to lib/"