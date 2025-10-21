#!/usr/bin/env bash
set -euo pipefail

# Test: Business Logic Validation
# Tests core business functionality and user workflows

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Get ports from environment or use defaults
API_PORT="${API_PORT:-}"
UI_PORT="${UI_PORT:-}"

# If ports not set, try to get from service.json or fail
if [ -z "${API_PORT}" ]; then
    echo "❌ API_PORT not set and scenario not running"
    echo "ℹ️  Start the scenario first: make run"
    exit 1
fi

echo "💼 Testing SmartNotes business logic..."
echo "   API: http://localhost:${API_PORT}"

# Track failures
FAILURES=0

test_endpoint() {
    local method=$1
    local endpoint=$2
    local desc=$3
    local data=${4:-}

    local url="http://localhost:${API_PORT}${endpoint}"

    if [ -n "${data}" ]; then
        if curl -sf -X "${method}" -H "Content-Type: application/json" -d "${data}" "${url}" > /dev/null; then
            echo "  ✅ ${desc}"
            return 0
        else
            echo "  ❌ ${desc} - ${method} ${endpoint} failed"
            ((FAILURES++))
            return 1
        fi
    else
        if curl -sf -X "${method}" "${url}" > /dev/null; then
            echo "  ✅ ${desc}"
            return 0
        else
            echo "  ❌ ${desc} - ${method} ${endpoint} failed"
            ((FAILURES++))
            return 1
        fi
    fi
}

# Test 1: CRUD Operations
echo "📝 Testing CRUD operations..."

# Create a note
echo "  Creating test note..."
NOTE_DATA='{"title":"Test Note","content":"This is a test note for validation","content_type":"markdown"}'
CREATED_NOTE=$(curl -sf -X POST -H "Content-Type: application/json" -d "${NOTE_DATA}" \
    "http://localhost:${API_PORT}/api/notes" 2>/dev/null || echo "")

if [ -n "${CREATED_NOTE}" ]; then
    NOTE_ID=$(echo "${CREATED_NOTE}" | jq -r '.id' 2>/dev/null || echo "")
    if [ -n "${NOTE_ID}" ] && [ "${NOTE_ID}" != "null" ]; then
        echo "  ✅ Create note (ID: ${NOTE_ID})"
    else
        echo "  ❌ Create note - Invalid response"
        ((FAILURES++))
        NOTE_ID=""
    fi
else
    echo "  ❌ Create note - Request failed"
    ((FAILURES++))
    NOTE_ID=""
fi

# Read the note (if created)
if [ -n "${NOTE_ID}" ]; then
    test_endpoint "GET" "/api/notes/${NOTE_ID}" "Read note by ID"

    # Update the note
    UPDATE_DATA='{"title":"Updated Test Note","content":"Updated content","is_pinned":true}'
    test_endpoint "PUT" "/api/notes/${NOTE_ID}" "Update note" "${UPDATE_DATA}"

    # List all notes
    test_endpoint "GET" "/api/notes" "List all notes"

    # Delete the note
    test_endpoint "DELETE" "/api/notes/${NOTE_ID}" "Delete note"
fi

# Test 2: Folder Operations
echo "📁 Testing folder operations..."
FOLDER_DATA='{"name":"Test Folder","icon":"📂","color":"#6366f1"}'
CREATED_FOLDER=$(curl -sf -X POST -H "Content-Type: application/json" -d "${FOLDER_DATA}" \
    "http://localhost:${API_PORT}/api/folders" 2>/dev/null || echo "")

if [ -n "${CREATED_FOLDER}" ]; then
    FOLDER_ID=$(echo "${CREATED_FOLDER}" | jq -r '.id' 2>/dev/null || echo "")
    if [ -n "${FOLDER_ID}" ] && [ "${FOLDER_ID}" != "null" ]; then
        echo "  ✅ Create folder"
        test_endpoint "GET" "/api/folders" "List folders"
        test_endpoint "DELETE" "/api/folders/${FOLDER_ID}" "Delete folder"
    fi
fi

# Test 3: Tag Operations
echo "🏷️  Testing tag operations..."
TAG_DATA='{"name":"test-tag","color":"#10b981"}'
test_endpoint "POST" "/api/tags" "Create tag" "${TAG_DATA}"
test_endpoint "GET" "/api/tags" "List tags"

# Test 4: Template Operations
echo "📋 Testing template operations..."
test_endpoint "GET" "/api/templates" "List templates"

# Test 5: Search Functionality
echo "🔍 Testing search functionality..."
SEARCH_DATA='{"query":"test","limit":10}'
test_endpoint "POST" "/api/search" "Text search" "${SEARCH_DATA}"

# Test semantic search (if Qdrant is available)
if vrooli resource status qdrant &> /dev/null; then
    SEMANTIC_DATA='{"query":"sample note","limit":5}'
    test_endpoint "POST" "/api/search/semantic" "Semantic search" "${SEMANTIC_DATA}"
fi

# Summary
echo ""
if [ ${FAILURES} -eq 0 ]; then
    echo "✅ Business logic validation passed!"
    exit 0
else
    echo "❌ Business logic validation failed with ${FAILURES} error(s)"
    exit 1
fi
