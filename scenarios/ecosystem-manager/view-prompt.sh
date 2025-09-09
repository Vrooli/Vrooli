#!/bin/bash

# Prompt viewer that uses the existing /tasks/{id}/prompt endpoint
set -e

# Parse arguments
TASK_TYPE="${1:-resource}"
OPERATION="${2:-generator}"
TITLE="${3:-Test Prompt Size Check}"
SHOW_FULL="${4:-preview}"

echo "=== Ecosystem Manager Prompt Viewer ==="
echo "Task Type: $TASK_TYPE"
echo "Operation: $OPERATION" 
echo "Title: $TITLE"
echo "Display: $SHOW_FULL"
echo ""

# Check if API is running
if ! curl -s http://localhost:5020/health > /dev/null 2>&1; then
    echo "❌ Ecosystem Manager API is not running"
    echo "Start it with: vrooli scenario ecosystem-manager develop"
    exit 1
fi

echo "✅ API is running"

# Create a test task
TASK_JSON=$(cat <<EOF
{
    "title": "$TITLE",
    "type": "$TASK_TYPE",
    "operation": "$OPERATION",
    "category": "test",
    "priority": "medium",
    "notes": "Test task for prompt size analysis after optimization"
}
EOF
)

echo "📝 Creating test task..."
RESPONSE=$(curl -s -X POST http://localhost:5020/api/tasks \
    -H "Content-Type: application/json" \
    -d "$TASK_JSON")

TASK_ID=$(echo "$RESPONSE" | jq -r '.id' 2>/dev/null || echo "unknown")

if [ "$TASK_ID" = "unknown" ] || [ "$TASK_ID" = "null" ]; then
    echo "❌ Failed to create task"
    echo "Response: $RESPONSE"
    exit 1
fi

echo "✅ Created test task: $TASK_ID"

# Get the generated prompt
echo ""
echo "🔄 Generating prompt..."
PROMPT_RESPONSE=$(curl -s "http://localhost:5020/api/tasks/$TASK_ID/prompt/assembled")

if echo "$PROMPT_RESPONSE" | jq -e .error > /dev/null 2>&1; then
    echo "❌ Failed to get prompt:"
    echo "$PROMPT_RESPONSE" | jq '.error'
    exit 1
fi

# Extract prompt from response
PROMPT=$(echo "$PROMPT_RESPONSE" | jq -r '.prompt' 2>/dev/null)

if [ "$PROMPT" = "null" ] || [ "$PROMPT" = "" ]; then
    echo "❌ No prompt returned"
    echo "Response: $PROMPT_RESPONSE"
    exit 1
fi

# Analyze prompt size
PROMPT_SIZE=${#PROMPT}
PROMPT_SIZE_KB=$(echo "scale=2; $PROMPT_SIZE / 1024" | bc 2>/dev/null || echo "$(($PROMPT_SIZE / 1024))")
PROMPT_SIZE_MB=$(echo "scale=3; $PROMPT_SIZE_KB / 1024" | bc 2>/dev/null || echo "0")

echo ""
echo "=== PROMPT SIZE ANALYSIS ==="
echo "📏 Total characters: $PROMPT_SIZE"
echo "📊 Size in KB: ${PROMPT_SIZE_KB}"
echo "💾 Size in MB: ${PROMPT_SIZE_MB}"
echo ""

# Show sections info from response
if echo "$PROMPT_RESPONSE" | jq -e '.sections' > /dev/null 2>&1; then
    SECTION_COUNT=$(echo "$PROMPT_RESPONSE" | jq '.sections | length' 2>/dev/null || echo "unknown")
    echo "📑 Total sections: $SECTION_COUNT"
    echo ""
    echo "=== SECTIONS INCLUDED ==="
    echo "$PROMPT_RESPONSE" | jq -r '.sections[]?' 2>/dev/null | nl -v1 -s'. ' || echo "Could not parse sections"
    echo ""
fi

# Display prompt content based on parameter
case "$SHOW_FULL" in
    "full"|"all")
        echo "=== FULL GENERATED PROMPT ==="
        echo "$PROMPT"
        ;;
    "first"|"preview")
        echo "=== PROMPT PREVIEW (first 3000 chars) ==="
        if [ ${#PROMPT} -gt 3000 ]; then
            echo "${PROMPT:0:3000}"
            echo ""
            echo "... [TRUNCATED - ${#PROMPT} total chars] ..."
        else
            echo "$PROMPT"
        fi
        ;;
    "size"|"stats")
        echo "✅ Size analysis complete"
        ;;
    *)
        echo "=== PROMPT SUMMARY ==="
        echo "Use '$0 $TASK_TYPE $OPERATION \"$TITLE\" full' to see full prompt"
        echo "Use '$0 $TASK_TYPE $OPERATION \"$TITLE\" preview' to see preview"
        ;;
esac

echo ""
echo "🧹 Cleaning up test task..."
curl -s -X DELETE "http://localhost:5020/api/tasks/$TASK_ID" > /dev/null 2>&1 || echo "Note: Could not delete test task (task may be in queue)"

echo "✅ Done!"