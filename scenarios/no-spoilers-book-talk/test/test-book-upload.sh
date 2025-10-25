#!/bin/bash

# Test Book Upload Functionality
# Validates book upload, processing, and basic functionality

set -e

API_PORT="${API_PORT:-20300}"
API_BASE_URL="http://localhost:${API_PORT}/api/v1"
TEST_USER_ID="upload-test-user"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[TEST]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Create test book content
create_test_book() {
    local test_file="$1"
    
    cat > "$test_file" << 'EOF'
# Test Book: The Adventures of Code

## Chapter 1: The Beginning

It was the best of times, it was the worst of times for software development. 
Our hero, Alice the Developer, sat at her computer contemplating the mysteries of clean code.

"I wonder," she mused, "what makes a program truly elegant?"

The cursor blinked expectantly on the empty screen, as if waiting for her to discover the answer.

## Chapter 2: The Journey Starts

Alice decided to embark on a journey to understand the fundamentals of programming. 
She opened her favorite text editor and began to type.

The first principle she learned was simplicity. "Write code that tells a story," her mentor had said.

As she wrote her first function, Alice realized that every line of code was like a sentence in a book. 
It should have purpose, clarity, and contribute to the overall narrative.

## Chapter 3: The Plot Thickens

Days turned into weeks as Alice dove deeper into the world of software architecture. 
She discovered patterns, learned about algorithms, and began to see the beauty in well-structured code.

But then came the challenge - a complex project that would test everything she had learned.
The deadline was tight, the requirements were unclear, and the stakeholders kept changing their minds.

This is where our story really begins...

## Chapter 4: The Resolution

[This chapter would contain spoilers about how Alice solves the problem]

Alice finally cracked the code (literally and figuratively). 
Through persistence, collaboration, and applying the principles she had learned, 
she delivered a solution that exceeded everyone's expectations.

The project was not just successful - it became a template for future endeavors.
Alice realized that good software is not just about the code, but about solving real problems for real people.

## Epilogue: The Continuing Journey

Alice's journey as a developer was just beginning. 
With each project, each bug fixed, and each feature implemented, 
she grew not just as a programmer, but as a problem solver.

And somewhere in a distant server, her code continues to run, 
serving users and making their lives a little bit easier.

The End.

---

Total word count: approximately 350 words
Perfect for testing the book upload and chunking system!
EOF

    log_info "Created test book: $test_file"
}

# Test 1: Upload book file
test_book_upload() {
    local test_file="/tmp/test-book-upload.txt"
    create_test_book "$test_file"
    
    log_info "Uploading test book..."
    
    local response
    response=$(curl -s -X POST \
        -F "file=@$test_file" \
        -F "title=The Adventures of Code" \
        -F "author=Test Author" \
        -F "user_id=$TEST_USER_ID" \
        "${API_BASE_URL}/books/upload")
    
    # Extract book ID
    UPLOADED_BOOK_ID=$(echo "$response" | jq -r '.book_id')
    
    if [[ "$UPLOADED_BOOK_ID" != "null" && -n "$UPLOADED_BOOK_ID" ]]; then
        log_info "Book uploaded successfully: $UPLOADED_BOOK_ID"
        rm -f "$test_file"
        return 0
    else
        log_error "Book upload failed: $response"
        rm -f "$test_file"
        return 1
    fi
}

# Test 2: Wait for processing to complete
test_processing_completion() {
    log_info "Waiting for book processing to complete..."
    
    local max_attempts=30
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        local response
        response=$(curl -s "${API_BASE_URL}/books/${UPLOADED_BOOK_ID}")
        local status
        status=$(echo "$response" | jq -r '.book.processing_status')
        
        case "$status" in
            "completed")
                log_info "Book processing completed"
                return 0
                ;;
            "processing")
                log_info "Still processing... (attempt $attempt/$max_attempts)"
                ;;
            "failed")
                log_error "Book processing failed"
                return 1
                ;;
            "pending")
                log_info "Processing pending... (attempt $attempt/$max_attempts)"
                ;;
        esac
        
        sleep 2
        ((attempt++))
    done
    
    log_error "Book processing did not complete within expected time"
    return 1
}

# Test 3: Verify book appears in list
test_book_in_list() {
    log_info "Checking if book appears in user's book list..."
    
    local response
    response=$(curl -s "${API_BASE_URL}/books?user_id=$TEST_USER_ID")
    
    local found_book
    found_book=$(echo "$response" | jq -r --arg book_id "$UPLOADED_BOOK_ID" '.[] | select(.id == $book_id) | .id')
    
    if [[ "$found_book" == "$UPLOADED_BOOK_ID" ]]; then
        log_info "Book found in user's book list"
        return 0
    else
        log_error "Book not found in user's book list"
        return 1
    fi
}

# Test 4: Test invalid file upload
test_invalid_file_upload() {
    log_info "Testing invalid file upload..."
    
    # Create invalid file
    local invalid_file="/tmp/invalid-file.xyz"
    echo "This is not a valid book file" > "$invalid_file"
    
    local response
    response=$(curl -s -X POST \
        -F "file=@$invalid_file" \
        -F "user_id=$TEST_USER_ID" \
        "${API_BASE_URL}/books/upload")
    
    # Should get an error for unsupported file type
    if echo "$response" | jq -e '.error' >/dev/null; then
        log_info "Invalid file upload properly rejected"
        rm -f "$invalid_file"
        return 0
    else
        log_error "Invalid file upload was accepted (should have been rejected)"
        rm -f "$invalid_file"
        return 1
    fi
}

# Test 5: Test upload without file
test_upload_without_file() {
    log_info "Testing upload without file..."
    
    local response
    response=$(curl -s -X POST \
        -F "user_id=$TEST_USER_ID" \
        "${API_BASE_URL}/books/upload")
    
    if echo "$response" | jq -e '.error' >/dev/null; then
        log_info "Upload without file properly rejected"
        return 0
    else
        log_error "Upload without file was accepted (should have been rejected)"
        return 1
    fi
}

# Test 6: Test book details retrieval
test_book_details() {
    log_info "Testing book details retrieval..."
    
    local response
    response=$(curl -s "${API_BASE_URL}/books/${UPLOADED_BOOK_ID}?user_id=$TEST_USER_ID")
    
    local book_title
    book_title=$(echo "$response" | jq -r '.book.title')
    
    if [[ "$book_title" == "The Adventures of Code" ]]; then
        log_info "Book details retrieved successfully"
        return 0
    else
        log_error "Book details retrieval failed or title mismatch"
        return 1
    fi
}

# Test 7: Set initial progress
test_initial_progress() {
    log_info "Testing initial progress setting..."
    
    local progress_payload
    progress_payload=$(jq -n --arg user_id "$TEST_USER_ID" '{
        user_id: $user_id,
        current_position: 1,
        position_type: "chunk",
        notes: "Started reading the test book"
    }')
    
    local response
    response=$(curl -s -X PUT -H "Content-Type: application/json" \
        -d "$progress_payload" \
        "${API_BASE_URL}/books/${UPLOADED_BOOK_ID}/progress")
    
    if echo "$response" | jq -e '.progress_id' >/dev/null; then
        log_info "Initial progress set successfully"
        return 0
    else
        log_error "Failed to set initial progress: $response"
        return 1
    fi
}

# Test 8: Basic chat functionality
test_basic_chat() {
    log_info "Testing basic chat functionality..."
    
    local chat_payload
    chat_payload=$(jq -n --arg msg "What is this book about?" --arg user_id "$TEST_USER_ID" '{
        message: $msg,
        user_id: $user_id,
        current_position: 1,
        position_type: "chunk"
    }')
    
    local response
    response=$(curl -s -X POST -H "Content-Type: application/json" \
        -d "$chat_payload" \
        "${API_BASE_URL}/books/${UPLOADED_BOOK_ID}/chat")
    
    if echo "$response" | jq -e '.response' >/dev/null; then
        log_info "Basic chat functionality works"
        return 0
    else
        log_error "Chat functionality failed: $response"
        return 1
    fi
}

# Cleanup function
cleanup() {
    if [[ -n "$UPLOADED_BOOK_ID" ]]; then
        log_info "Cleaning up test data..."
        # Note: In a real implementation, we might want to add a delete endpoint
        # For now, just log the cleanup intent
        log_info "Test book ID for manual cleanup: $UPLOADED_BOOK_ID"
    fi
}

# Test execution
main() {
    echo "📚 No Spoilers Book Talk - Book Upload Tests"
    echo "============================================="
    
    # Set up cleanup trap
    trap cleanup EXIT
    
    local tests_passed=0
    local tests_failed=0
    
    run_test() {
        local test_name="$1"
        local test_command="$2"
        
        echo ""
        log_info "Running test: $test_name"
        
        if eval "$test_command"; then
            log_info "✅ PASS: $test_name"
            ((tests_passed += 1))
        else
            log_error "❌ FAIL: $test_name"
            ((tests_failed += 1))
        fi
    }
    
    # Check API availability
    if ! curl -s "${API_BASE_URL%/api/v1}/health" >/dev/null 2>&1; then
        log_error "API not available - cannot run tests"
        exit 1
    fi
    
    # Run tests
    run_test "Book Upload" "test_book_upload"
    run_test "Processing Completion" "test_processing_completion"
    run_test "Book in List" "test_book_in_list"
    run_test "Invalid File Upload" "test_invalid_file_upload"
    run_test "Upload Without File" "test_upload_without_file"
    run_test "Book Details Retrieval" "test_book_details"
    run_test "Initial Progress Setting" "test_initial_progress"
    run_test "Basic Chat Functionality" "test_basic_chat"
    
    # Results
    echo ""
    echo "============================================="
    echo "📊 Test Results Summary"
    echo "============================================="
    echo "✅ Tests Passed: $tests_passed"
    echo "❌ Tests Failed: $tests_failed"
    
    if [[ $tests_failed -eq 0 ]]; then
        log_info "🎉 All book upload tests passed!"
        exit 0
    else
        log_error "💥 Some tests failed"
        exit 1
    fi
}

# Run the tests
main "$@"
