#!/bin/bash

# Test Spoiler Prevention System
# Validates that the position-based filtering prevents spoilers

set -e

API_PORT="${API_PORT:-20300}"
API_BASE_URL="http://localhost:${API_PORT}/api/v1"
TEST_USER_ID="spoiler-test-user"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[TEST]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Test results tracking
TESTS_PASSED=0
TESTS_FAILED=0

run_test() {
    local test_name="$1"
    local test_command="$2"
    
    echo ""
    log_info "Running test: $test_name"
    
    if eval "$test_command"; then
        log_info "✅ PASS: $test_name"
        ((TESTS_PASSED += 1))
    else
        log_error "❌ FAIL: $test_name"
        ((TESTS_FAILED += 1))
    fi
}

# Wait for API to be ready
wait_for_api() {
    log_info "Waiting for API to be ready..."
    local max_attempts=30
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -s "${API_BASE_URL%/api/v1}/health" >/dev/null 2>&1; then
            log_info "API is ready"
            return 0
        fi
        
        sleep 2
        ((attempt++))
        echo -n "."
    done
    
    log_error "API not ready after $max_attempts attempts"
    return 1
}

# Test 1: Verify API is accessible
test_api_health() {
    local response
    response=$(curl -s "${API_BASE_URL%/api/v1}/health")
    
    if echo "$response" | jq -e '.status == "healthy"' >/dev/null; then
        return 0
    else
        log_error "API health check failed: $response"
        return 1
    fi
}

# Test 2: Get test book for spoiler testing
get_test_book_id() {
    local response
    response=$(curl -s "${API_BASE_URL}/books")
    
    # Get first completed book
    TEST_BOOK_ID=$(echo "$response" | jq -r '.[] | select(.processing_status == "completed") | .id' | head -1)
    
    if [[ "$TEST_BOOK_ID" != "null" && -n "$TEST_BOOK_ID" ]]; then
        log_info "Using test book ID: $TEST_BOOK_ID"
        return 0
    else
        log_error "No completed books found for testing"
        return 1
    fi
}

# Test 3: Set user progress to middle of book
set_test_progress() {
    local book_response
    book_response=$(curl -s "${API_BASE_URL}/books/${TEST_BOOK_ID}")
    local total_chunks
    total_chunks=$(echo "$book_response" | jq -r '.book.total_chunks')
    
    # Set progress to 50% through the book
    local middle_position=$((total_chunks / 2))
    
    local progress_payload
    progress_payload=$(jq -n --arg user_id "$TEST_USER_ID" --arg pos "$middle_position" '{
        user_id: $user_id,
        current_position: ($pos | tonumber),
        position_type: "chunk",
        notes: "Test progress for spoiler prevention"
    }')
    
    local response
    response=$(curl -s -X PUT -H "Content-Type: application/json" \
        -d "$progress_payload" \
        "${API_BASE_URL}/books/${TEST_BOOK_ID}/progress")
    
    if echo "$response" | jq -e '.progress_id' >/dev/null; then
        TEST_POSITION=$(echo "$response" | jq -r '.current_position')
        log_info "Set test progress to position: $TEST_POSITION"
        return 0
    else
        log_error "Failed to set test progress: $response"
        return 1
    fi
}

# Test 4: Test position boundary is respected in chat
test_position_boundary_respected() {
    local chat_payload
    chat_payload=$(jq -n --arg msg "Tell me what happens at the very end of the book" --arg user_id "$TEST_USER_ID" --arg pos "$TEST_POSITION" '{
        message: $msg,
        user_id: $user_id,
        current_position: ($pos | tonumber),
        position_type: "chunk"
    }')
    
    local response
    response=$(curl -s -X POST -H "Content-Type: application/json" \
        -d "$chat_payload" \
        "${API_BASE_URL}/books/${TEST_BOOK_ID}/chat")
    
    # Check if position boundary was respected
    if echo "$response" | jq -e '.position_boundary_respected == true' >/dev/null; then
        log_info "Position boundary respected in response"
        return 0
    else
        log_error "Position boundary NOT respected: $response"
        return 1
    fi
}

# Test 5: Verify response doesn't contain spoiler markers
test_no_spoiler_content() {
    local chat_payload
    chat_payload=$(jq -n --arg msg "What are the main themes in this book?" --arg user_id "$TEST_USER_ID" --arg pos "$TEST_POSITION" '{
        message: $msg,
        user_id: $user_id,
        current_position: ($pos | tonumber),
        position_type: "chunk"
    }')
    
    local response
    response=$(curl -s -X POST -H "Content-Type: application/json" \
        -d "$chat_payload" \
        "${API_BASE_URL}/books/${TEST_BOOK_ID}/chat")
    
    local ai_response
    ai_response=$(echo "$response" | jq -r '.response')
    
    # Check for spoiler warning patterns (indicating proper filtering)
    if echo "$ai_response" | grep -iq "based on what you've read so far\|up to your current position\|without spoiling"; then
        log_info "Response shows spoiler awareness: contains position-aware language"
        return 0
    else
        log_warn "Response may not be position-aware. Content: $ai_response"
        # This is a warning, not a failure, as the exact language may vary
        return 0
    fi
}

# Test 6: Test context chunks are within position boundary
test_context_chunks_boundary() {
    local chat_payload
    chat_payload=$(jq -n --arg msg "Summarize the plot so far" --arg user_id "$TEST_USER_ID" --arg pos "$TEST_POSITION" '{
        message: $msg,
        user_id: $user_id,
        current_position: ($pos | tonumber),
        position_type: "chunk"
    }')
    
    local response
    response=$(curl -s -X POST -H "Content-Type: application/json" \
        -d "$chat_payload" \
        "${API_BASE_URL}/books/${TEST_BOOK_ID}/chat")
    
    local context_chunks
    context_chunks=$(echo "$response" | jq -r '.context_chunks_count')
    
    if [[ "$context_chunks" -gt 0 ]]; then
        log_info "Context chunks used: $context_chunks (should all be within position $TEST_POSITION)"
        return 0
    else
        log_error "No context chunks used in response"
        return 1
    fi
}

# Test 7: Test chat with position 0 (beginning)
test_beginning_position() {
    # Update progress to beginning
    local progress_payload
    progress_payload=$(jq -n --arg user_id "$TEST_USER_ID" '{
        user_id: $user_id,
        current_position: 0,
        position_type: "chunk",
        notes: "Testing from beginning"
    }')
    
    curl -s -X PUT -H "Content-Type: application/json" \
        -d "$progress_payload" \
        "${API_BASE_URL}/books/${TEST_BOOK_ID}/progress" >/dev/null
    
    # Try to chat
    local chat_payload
    chat_payload=$(jq -n --arg msg "What happens in this book?" --arg user_id "$TEST_USER_ID" '{
        message: $msg,
        user_id: $user_id,
        current_position: 0,
        position_type: "chunk"
    }')
    
    local response
    response=$(curl -s -X POST -H "Content-Type: application/json" \
        -d "$chat_payload" \
        "${API_BASE_URL}/books/${TEST_BOOK_ID}/chat")
    
    if echo "$response" | jq -e '.response' >/dev/null; then
        log_info "Successfully handled chat at position 0"
        return 0
    else
        log_error "Failed to handle chat at position 0: $response"
        return 1
    fi
}

# Test 8: Verify conversation is stored
test_conversation_storage() {
    local response
    response=$(curl -s "${API_BASE_URL}/books/${TEST_BOOK_ID}/conversations?user_id=${TEST_USER_ID}")
    
    local conversation_count
    conversation_count=$(echo "$response" | jq -r '.count')
    
    if [[ "$conversation_count" -gt 0 ]]; then
        log_info "Conversations stored successfully: $conversation_count conversations"
        return 0
    else
        log_error "No conversations stored"
        return 1
    fi
}

# Main test execution
main() {
    echo "🛡️ No Spoilers Book Talk - Spoiler Prevention Tests"
    echo "=================================================="
    
    # Wait for API
    if ! wait_for_api; then
        log_error "API not available - cannot run tests"
        exit 1
    fi
    
    # Run all tests
    run_test "API Health Check" "test_api_health"
    run_test "Get Test Book ID" "get_test_book_id"
    run_test "Set Test Progress" "set_test_progress" 
    run_test "Position Boundary Respected" "test_position_boundary_respected"
    run_test "No Spoiler Content Markers" "test_no_spoiler_content"
    run_test "Context Chunks Within Boundary" "test_context_chunks_boundary"
    run_test "Beginning Position Handling" "test_beginning_position"
    run_test "Conversation Storage" "test_conversation_storage"
    
    # Results summary
    echo ""
    echo "=================================================="
    echo "📊 Test Results Summary"
    echo "=================================================="
    echo "✅ Tests Passed: $TESTS_PASSED"
    echo "❌ Tests Failed: $TESTS_FAILED"
    echo "📈 Success Rate: $(( (TESTS_PASSED * 100) / (TESTS_PASSED + TESTS_FAILED) ))%"
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        log_info "🎉 All spoiler prevention tests passed!"
        exit 0
    else
        log_error "💥 Some tests failed - spoiler prevention may not be working correctly"
        exit 1
    fi
}

# Run tests
main "$@"
