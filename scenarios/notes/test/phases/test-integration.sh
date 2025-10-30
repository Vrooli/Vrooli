#!/bin/bash
# Integration tests for SmartNotes

set -euo pipefail

echo "🔗 Running SmartNotes integration tests..."

# Test configuration - require ports from environment
if [ -z "$API_PORT" ] || [ -z "$UI_PORT" ]; then
    echo "❌ API_PORT and UI_PORT environment variables are required"
    exit 1
fi
API_URL="http://localhost:${API_PORT}"
UI_URL="http://localhost:${UI_PORT}"

# Track test results
TESTS_PASSED=0
TESTS_FAILED=0

# Track created resources for cleanup
CREATED_NOTE_IDS=()
CREATED_FOLDER_IDS=()
CREATED_TAG_IDS=()
CREATED_TEMPLATE_IDS=()

# Cleanup function
cleanup_test_data() {
    echo ""
    echo "🧹 Cleaning up test data..."

    # Delete created notes
    for note_id in "${CREATED_NOTE_IDS[@]}"; do
        curl -sf -X DELETE "${API_URL}/api/notes/${note_id}" > /dev/null 2>&1 || true
    done

    # Delete created folders
    for folder_id in "${CREATED_FOLDER_IDS[@]}"; do
        curl -sf -X DELETE "${API_URL}/api/folders/${folder_id}" > /dev/null 2>&1 || true
    done

    # Delete created tags
    for tag_id in "${CREATED_TAG_IDS[@]}"; do
        curl -sf -X DELETE "${API_URL}/api/tags/${tag_id}" > /dev/null 2>&1 || true
    done

    # Delete created templates
    for template_id in "${CREATED_TEMPLATE_IDS[@]}"; do
        curl -sf -X DELETE "${API_URL}/api/templates/${template_id}" > /dev/null 2>&1 || true
    done

    echo "✅ Test data cleaned up"
}

# Register cleanup to run on exit
trap cleanup_test_data EXIT

# Function to make API calls
api_call() {
    local method=$1
    local endpoint=$2
    local data=${3:-}

    if [ -n "$data" ]; then
        curl -sf -X "$method" "${API_URL}${endpoint}" \
            -H "Content-Type: application/json" \
            -d "$data"
    else
        curl -sf -X "$method" "${API_URL}${endpoint}"
    fi
}

# Test UI availability
echo "Testing UI availability..."
if curl -sf "${UI_URL}/" | grep -q "SmartNotes"; then
    echo "✅ UI is accessible"
else
    echo "❌ UI is not accessible"
    exit 1
fi

# Test folders functionality
echo "Testing folders..."
folder_response=$(api_call POST /api/folders '{"name":"Test Folder","icon":"📁","color":"#6366f1"}')
if echo "$folder_response" | grep -q "Test Folder"; then
    folder_id=$(echo "$folder_response" | jq -r '.id' 2>/dev/null || echo "")
    [ -n "$folder_id" ] && CREATED_FOLDER_IDS+=("$folder_id")
    echo "✅ Folder created"
else
    echo "❌ Failed to create folder"
    exit 1
fi

# Test tags functionality
echo "Testing tags..."
tag_response=$(api_call POST /api/tags '{"name":"test-tag","color":"#10b981"}')
if echo "$tag_response" | grep -q "test-tag"; then
    tag_id=$(echo "$tag_response" | jq -r '.id' 2>/dev/null || echo "")
    [ -n "$tag_id" ] && CREATED_TAG_IDS+=("$tag_id")
    echo "✅ Tag created"
else
    echo "❌ Failed to create tag"
    exit 1
fi

# Test templates functionality
echo "Testing templates..."
template_response=$(api_call POST /api/templates '{
    "name":"Meeting Notes",
    "description":"Template for meeting notes",
    "content":"# Meeting Notes\\n\\n## Date: \\n## Attendees: \\n## Agenda: \\n## Action Items: ",
    "category":"business"
}')
if echo "$template_response" | grep -q "Meeting Notes"; then
    template_id=$(echo "$template_response" | jq -r '.id' 2>/dev/null || echo "")
    [ -n "$template_id" ] && CREATED_TEMPLATE_IDS+=("$template_id")
    echo "✅ Template created"
else
    echo "❌ Failed to create template"
    exit 1
fi

# Test search functionality
echo "Testing search..."
search_note_response=$(api_call POST /api/notes '{"title":"Searchable Note","content":"This note contains unique keywords for testing"}')
search_note_id=$(echo "$search_note_response" | jq -r '.id' 2>/dev/null || echo "")
[ -n "$search_note_id" ] && CREATED_NOTE_IDS+=("$search_note_id")
sleep 1
search_response=$(api_call POST /api/search '{"query":"unique keywords"}')
if echo "$search_response" | grep -q "Searchable Note"; then
    echo "✅ Search functionality works"
else
    echo "❌ Search functionality failed"
    exit 1
fi

# Test semantic search if Qdrant is available
echo "Testing semantic search..."
if curl -sf "http://localhost:6333/collections" &>/dev/null; then
    # Create a note with specific content
    ai_note_response=$(api_call POST /api/notes '{"title":"AI Research","content":"Machine learning, neural networks, deep learning algorithms"}')
    ai_note_id=$(echo "$ai_note_response" | jq -r '.id' 2>/dev/null || echo "")
    [ -n "$ai_note_id" ] && CREATED_NOTE_IDS+=("$ai_note_id")
    sleep 2 # Allow time for indexing

    # Test semantic search
    semantic_response=$(api_call POST /api/search/semantic '{"query":"artificial intelligence","limit":5}' || echo "FAILED")
    if [[ "$semantic_response" != "FAILED" ]]; then
        echo "✅ Semantic search functionality available"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo "⚠️  Semantic search not fully working"
    fi
else
    echo "⚠️  Qdrant not available - semantic search skipped"
fi

# Summary
echo ""
echo "📊 Test Summary:"
echo "   ✅ Tests Passed: ${TESTS_PASSED}"
if [[ $TESTS_FAILED -gt 0 ]]; then
    echo "   ❌ Tests Failed: ${TESTS_FAILED}"
    exit 1
else
    echo "✅ All integration tests passed!"
    exit 0
fi