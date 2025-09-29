#!/bin/bash
# Unit tests for SmartNotes

set -e

echo "🧪 Running SmartNotes unit tests..."

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

# Test health endpoint
echo "Testing health endpoint..."
health_response=$(api_call GET /health)
if echo "$health_response" | grep -q "healthy"; then
    echo "✅ Health check passed"
else
    echo "❌ Health check failed"
    exit 1
fi

# Test CRUD operations
echo "Testing CRUD operations..."

# Create a note
echo "Creating a note..."
create_response=$(api_call POST /api/notes '{"title":"Unit Test Note","content":"This is a test note"}')
note_id=$(echo "$create_response" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)

if [ -n "$note_id" ]; then
    echo "✅ Note created with ID: $note_id"
else
    echo "❌ Failed to create note"
    exit 1
fi

# Read the note
echo "Reading the note..."
read_response=$(api_call GET "/api/notes/$note_id")
if echo "$read_response" | grep -q "Unit Test Note"; then
    echo "✅ Note retrieved successfully"
else
    echo "❌ Failed to retrieve note"
    exit 1
fi

# Update the note
echo "Updating the note..."
update_response=$(api_call PUT "/api/notes/$note_id" '{"title":"Updated Test Note","content":"Updated content"}')
if echo "$update_response" | grep -q "Updated Test Note"; then
    echo "✅ Note updated successfully"
else
    echo "❌ Failed to update note"
    exit 1
fi

# List notes
echo "Listing notes..."
list_response=$(api_call GET /api/notes)
if echo "$list_response" | grep -q "$note_id"; then
    echo "✅ Note appears in list"
else
    echo "❌ Note not found in list"
    exit 1
fi

# Delete the note
echo "Deleting the note..."
api_call DELETE "/api/notes/$note_id"
list_after_delete=$(api_call GET /api/notes)
if echo "$list_after_delete" | grep -q "$note_id"; then
    echo "❌ Note still exists after delete"
    exit 1
else
    echo "✅ Note deleted successfully"
fi

echo "✅ All unit tests passed!"