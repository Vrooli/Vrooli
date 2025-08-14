#!/bin/bash

# Test script for Simple Queue Manager n8n workflow
# Tests lightweight async task processing with Redis Lists

set -e

# Configuration
N8N_WEBHOOK_URL="http://localhost:5678/webhook/queue"
QUEUE_NAME="test_image_processing"
TEST_JOB_ID="job_$(date +%s)_test"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to make HTTP request and validate response
make_request() {
    local operation="$1"
    local payload="$2"
    local expected_success="${3:-true}"
    
    log_info "Testing operation: $operation"
    
    response=$(curl -s -X POST "$N8N_WEBHOOK_URL" \
        -H "Content-Type: application/json" \
        -d "$payload")
    
    echo "Response: $response" | jq .
    
    # Validate response structure
    success=$(echo "$response" | jq -r '.success // false')
    
    if [ "$success" = "$expected_success" ]; then
        log_success "$operation operation completed successfully"
        return 0
    else
        log_error "$operation operation failed"
        echo "$response" | jq .
        return 1
    fi
}

# Test functions
test_enqueue_job() {
    log_info "=== Testing Job Enqueue ==="
    
    # Test normal priority job
    payload='{
        "operation": "enqueue",
        "queue_name": "'$QUEUE_NAME'",
        "job": {
            "id": "'$TEST_JOB_ID'",
            "type": "resize_image",
            "payload": {
                "image_url": "https://example.com/test-image.jpg",
                "sizes": [100, 200, 400]
            },
            "priority": 5,
            "delay_seconds": 0,
            "max_retries": 3
        }
    }'
    
    make_request "enqueue" "$payload"
    
    # Test high priority job
    HIGH_PRIORITY_JOB_ID="${TEST_JOB_ID}_high_priority"
    payload='{
        "operation": "enqueue",
        "queue_name": "'$QUEUE_NAME'",
        "job": {
            "id": "'$HIGH_PRIORITY_JOB_ID'",
            "type": "urgent_processing",
            "payload": {
                "urgent": true
            },
            "priority": 1,
            "delay_seconds": 0,
            "max_retries": 5
        }
    }'
    
    make_request "enqueue_priority" "$payload"
    
    # Test delayed job
    DELAYED_JOB_ID="${TEST_JOB_ID}_delayed"
    payload='{
        "operation": "enqueue",
        "queue_name": "'$QUEUE_NAME'",
        "job": {
            "id": "'$DELAYED_JOB_ID'",
            "type": "scheduled_cleanup",
            "payload": {
                "cleanup_type": "temp_files"
            },
            "priority": 5,
            "delay_seconds": 30,
            "max_retries": 2
        }
    }'
    
    make_request "enqueue_delayed" "$payload"
}

test_dequeue_job() {
    log_info "=== Testing Job Dequeue ==="
    
    # Test single job dequeue
    payload='{
        "operation": "dequeue",
        "queue_name": "'$QUEUE_NAME'",
        "timeout_seconds": 10,
        "worker_id": "worker_test_001",
        "max_jobs": 1
    }'
    
    make_request "dequeue" "$payload"
    
    # Test batch dequeue
    payload='{
        "operation": "dequeue",
        "queue_name": "'$QUEUE_NAME'",
        "timeout_seconds": 15,
        "worker_id": "worker_test_batch",
        "max_jobs": 5
    }'
    
    make_request "dequeue_batch" "$payload"
}

test_job_status() {
    log_info "=== Testing Job Status Operations ==="
    
    # Get specific job status
    payload='{
        "operation": "get_status",
        "job_id": "'$TEST_JOB_ID'",
        "include_payload": true
    }'
    
    make_request "get_job_status" "$payload"
    
    # Get queue status
    payload='{
        "operation": "get_status",
        "queue_name": "'$QUEUE_NAME'"
    }'
    
    make_request "get_queue_status" "$payload"
}

test_update_status() {
    log_info "=== Testing Status Updates ==="
    
    # Update job to processing
    payload='{
        "operation": "update_status",
        "job_id": "'$TEST_JOB_ID'",
        "status": "processing",
        "worker_id": "worker_test_001",
        "progress": 25
    }'
    
    make_request "update_processing" "$payload"
    
    # Update job to completed
    payload='{
        "operation": "update_status",
        "job_id": "'$TEST_JOB_ID'",
        "status": "completed",
        "worker_id": "worker_test_001",
        "result": {
            "processed_images": 3,
            "total_size_mb": 2.5,
            "processing_time_ms": 1500
        }
    }'
    
    make_request "update_completed" "$payload"
    
    # Update job to failed
    FAILED_JOB_ID="${TEST_JOB_ID}_failed"
    payload='{
        "operation": "update_status",
        "job_id": "'$FAILED_JOB_ID'",
        "status": "failed",
        "worker_id": "worker_test_001",
        "error": {
            "code": "IMAGE_NOT_FOUND",
            "message": "Source image could not be accessed",
            "details": {
                "url": "https://example.com/missing-image.jpg",
                "http_status": 404
            }
        }
    }'
    
    make_request "update_failed" "$payload"
}

test_queue_statistics() {
    log_info "=== Testing Queue Statistics ==="
    
    # Get basic stats
    payload='{
        "operation": "get_stats",
        "queue_name": "'$QUEUE_NAME'",
        "time_range": "1h",
        "include_details": false
    }'
    
    make_request "get_stats_basic" "$payload"
    
    # Get detailed stats
    payload='{
        "operation": "get_stats",
        "queue_name": "'$QUEUE_NAME'",
        "time_range": "24h",
        "include_details": true
    }'
    
    make_request "get_stats_detailed" "$payload"
    
    # Get global stats
    payload='{
        "operation": "get_stats",
        "time_range": "1h",
        "include_details": false
    }'
    
    make_request "get_stats_global" "$payload"
}

test_retry_mechanisms() {
    log_info "=== Testing Retry Mechanisms ==="
    
    # Retry failed job
    FAILED_JOB_ID="${TEST_JOB_ID}_failed"
    payload='{
        "operation": "retry_job",
        "job_id": "'$FAILED_JOB_ID'",
        "reset_retry_count": false
    }'
    
    make_request "retry_job" "$payload"
    
    # Retry with priority override
    payload='{
        "operation": "retry_job",
        "job_id": "'$FAILED_JOB_ID'",
        "reset_retry_count": true,
        "new_priority": 2
    }'
    
    make_request "retry_job_priority" "$payload"
}

test_dead_letter_queue() {
    log_info "=== Testing Dead Letter Queue ==="
    
    # Move job to DLQ
    DLQ_JOB_ID="${TEST_JOB_ID}_dlq"
    payload='{
        "operation": "move_to_dlq",
        "job_id": "'$DLQ_JOB_ID'",
        "reason": "Exceeded maximum retry attempts"
    }'
    
    make_request "move_to_dlq" "$payload"
}

test_queue_peek() {
    log_info "=== Testing Queue Peek ==="
    
    # Peek at queue contents
    payload='{
        "operation": "peek",
        "queue_name": "'$QUEUE_NAME'",
        "count": 10,
        "offset": 0
    }'
    
    make_request "peek_queue" "$payload"
    
    # Peek with offset
    payload='{
        "operation": "peek",
        "queue_name": "'$QUEUE_NAME'",
        "count": 5,
        "offset": 5
    }'
    
    make_request "peek_queue_offset" "$payload"
}

test_error_conditions() {
    log_info "=== Testing Error Conditions ==="
    
    # Missing operation
    payload='{
        "queue_name": "'$QUEUE_NAME'"
    }'
    
    make_request "missing_operation" "$payload" "false"
    
    # Invalid operation
    payload='{
        "operation": "invalid_operation"
    }'
    
    make_request "invalid_operation" "$payload" "false"
    
    # Missing required fields for enqueue
    payload='{
        "operation": "enqueue",
        "queue_name": "'$QUEUE_NAME'"
    }'
    
    make_request "enqueue_missing_job" "$payload" "false"
    
    # Invalid priority
    payload='{
        "operation": "enqueue",
        "queue_name": "'$QUEUE_NAME'",
        "job": {
            "id": "invalid_priority_job",
            "type": "test",
            "priority": 15
        }
    }'
    
    make_request "enqueue_invalid_priority" "$payload"
    
    # Purge without confirmation
    payload='{
        "operation": "purge",
        "queue_name": "'$QUEUE_NAME'"
    }'
    
    make_request "purge_no_confirm" "$payload" "false"
}

test_queue_purge() {
    log_info "=== Testing Queue Purge (DESTRUCTIVE) ==="
    log_warning "This will delete queue data!"
    
    # Purge specific status
    payload='{
        "operation": "purge",
        "queue_name": "'$QUEUE_NAME'",
        "status_filter": "dead",
        "confirm": true
    }'
    
    make_request "purge_dead_jobs" "$payload"
    
    # Full queue purge (commented out for safety)
    # payload='{
    #     "operation": "purge",
    #     "queue_name": "'$QUEUE_NAME'",
    #     "confirm": true
    # }'
    # make_request "purge_all" "$payload"
}

# Performance test
test_performance() {
    log_info "=== Testing Performance ==="
    
    log_info "Enqueueing 10 jobs concurrently..."
    
    for i in {1..10}; do
        {
            job_id="perf_job_${i}_$(date +%s)"
            payload='{
                "operation": "enqueue",
                "queue_name": "'$QUEUE_NAME'",
                "job": {
                    "id": "'$job_id'",
                    "type": "performance_test",
                    "payload": {"test_number": '$i'},
                    "priority": '$((i % 10 + 1))'
                }
            }'
            
            curl -s -X POST "$N8N_WEBHOOK_URL" \
                -H "Content-Type: application/json" \
                -d "$payload" > /dev/null
            
            echo "Job $i enqueued"
        } &
    done
    
    wait
    log_success "All performance test jobs enqueued"
    
    # Check queue stats after performance test
    payload='{
        "operation": "get_stats",
        "queue_name": "'$QUEUE_NAME'",
        "time_range": "5m"
    }'
    
    make_request "performance_stats" "$payload"
}

# Main test execution
main() {
    log_info "Starting Simple Queue Manager Tests"
    log_info "Target URL: $N8N_WEBHOOK_URL"
    log_info "Test Queue: $QUEUE_NAME"
    log_info "Test Job ID: $TEST_JOB_ID"
    echo
    
    # Check if n8n is running
    if ! curl -s "$N8N_WEBHOOK_URL" -o /dev/null; then
        log_error "Cannot connect to n8n webhook at $N8N_WEBHOOK_URL"
        log_error "Please ensure n8n is running and the Simple Queue Manager workflow is active"
        exit 1
    fi
    
    # Run tests
    test_enqueue_job
    echo
    
    test_dequeue_job
    echo
    
    test_job_status
    echo
    
    test_update_status
    echo
    
    test_queue_statistics
    echo
    
    test_retry_mechanisms
    echo
    
    test_dead_letter_queue
    echo
    
    test_queue_peek
    echo
    
    test_error_conditions
    echo
    
    test_performance
    echo
    
    # Only run destructive tests if explicitly requested
    if [ "$1" = "--include-destructive" ]; then
        test_queue_purge
        echo
    else
        log_warning "Skipping destructive tests. Use --include-destructive to run purge tests."
    fi
    
    log_success "All Simple Queue Manager tests completed!"
    
    # Final queue status
    log_info "Final queue status:"
    payload='{
        "operation": "get_stats",
        "queue_name": "'$QUEUE_NAME'",
        "time_range": "1h",
        "include_details": true
    }'
    
    make_request "final_stats" "$payload"
}

# Check dependencies
check_dependencies() {
    if ! command -v curl &> /dev/null; then
        log_error "curl is required but not installed"
        exit 1
    fi
    
    if ! command -v jq &> /dev/null; then
        log_error "jq is required but not installed"
        exit 1
    fi
}

# Script entry point
if [ "${BASH_SOURCE[0]}" == "${0}" ]; then
    check_dependencies
    main "$@"
fi