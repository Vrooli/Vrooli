#!/bin/bash
# Integration tests for SmartNotes

set -e

echo "🔗 Running SmartNotes integration tests..."

# Test configuration
API_URL="http://localhost:${API_PORT:-17009}"
UI_URL="http://localhost:${UI_PORT:-36529}"

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
    echo "✅ Folder created"
else
    echo "❌ Failed to create folder"
    exit 1
fi

# Test tags functionality
echo "Testing tags..."
tag_response=$(api_call POST /api/tags '{"name":"test-tag","color":"#10b981"}')
if echo "$tag_response" | grep -q "test-tag"; then
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
    echo "✅ Template created"
else
    echo "❌ Failed to create template"
    exit 1
fi

# Test search functionality
echo "Testing search..."
api_call POST /api/notes '{"title":"Searchable Note","content":"This note contains unique keywords for testing"}'
sleep 1
search_response=$(api_call POST /api/notes/search '{"query":"unique keywords"}')
if echo "$search_response" | grep -q "Searchable Note"; then
    echo "✅ Search functionality works"
else
    echo "❌ Search functionality failed"
    exit 1
fi

echo "✅ All integration tests passed!"