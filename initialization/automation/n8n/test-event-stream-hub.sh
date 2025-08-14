#!/bin/bash

# Test script for Event Stream Hub n8n workflow
# Tests all operations: publish, subscribe, ack, get_pending, replay, create_consumer_group, get_stats, set_retention

set -e

# Configuration
N8N_URL="http://localhost:5678"
WEBHOOK_PATH="/webhook/events/publish"
ENDPOINT="$N8N_URL$WEBHOOK_PATH"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counter
TESTS_RUN=0
TESTS_PASSED=0

# Function to print colored output
print_status() {
    local status=$1
    local message=$2
    case $status in
        "PASS")
            echo -e "${GREEN}✓ PASS${NC}: $message"
            ((TESTS_PASSED++))
            ;;
        "FAIL")
            echo -e "${RED}✗ FAIL${NC}: $message"
            ;;
        "INFO")
            echo -e "${BLUE}ℹ INFO${NC}: $message"
            ;;
        "WARN")
            echo -e "${YELLOW}⚠ WARN${NC}: $message"
            ;;
    esac
    ((TESTS_RUN++))
}

# Function to make HTTP request and parse response
make_request() {
    local payload=$1
    local expected_operation=$2
    
    echo -e "\n${BLUE}Request:${NC} $payload"
    
    response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "$ENDPOINT")
    
    echo -e "${BLUE}Response:${NC} $response"
    
    # Check if response is valid JSON
    if ! echo "$response" | jq . >/dev/null 2>&1; then
        print_status "FAIL" "Invalid JSON response"
        return 1
    fi
    
    # Check if operation succeeded
    local success=$(echo "$response" | jq -r '.success // false')
    local operation=$(echo "$response" | jq -r '.operation // "unknown"')
    
    if [ "$success" = "true" ] && [ "$operation" = "$expected_operation" ]; then
        print_status "PASS" "$expected_operation operation successful"
        return 0
    else
        local error_msg=$(echo "$response" | jq -r '.error.message // "Unknown error"')
        print_status "FAIL" "$expected_operation operation failed: $error_msg"
        return 1
    fi
}

# Function to wait for Redis to be ready
wait_for_redis() {
    echo -e "${BLUE}Checking Redis connectivity...${NC}"
    for i in {1..30}; do
        if redis-cli ping >/dev/null 2>&1; then
            print_status "PASS" "Redis is ready"
            return 0
        fi
        echo "Waiting for Redis... ($i/30)"
        sleep 1
    done
    print_status "FAIL" "Redis not available after 30 seconds"
    return 1
}

# Function to check n8n availability
check_n8n() {
    echo -e "${BLUE}Checking n8n availability...${NC}"
    if curl -s -f "$N8N_URL/healthz" >/dev/null 2>&1; then
        print_status "PASS" "n8n is accessible"
        return 0
    else
        print_status "FAIL" "n8n is not accessible at $N8N_URL"
        return 1
    fi
}

# Test 1: Publish Event
test_publish_event() {
    echo -e "\n${YELLOW}=== Test 1: Publish Event ===${NC}"
    
    local payload='{
        "operation": "publish",
        "topic": "app.test.events",
        "event": {
            "type": "test_event",
            "payload": {
                "message": "Hello Event Stream Hub",
                "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ)'",
                "test_id": "test_001"
            },
            "metadata": {
                "source": "test_script",
                "version": "1.0"
            }
        },
        "tags": ["test", "demo", "event-stream-hub"],
        "ttl": 3600
    }'
    
    if make_request "$payload" "publish"; then
        # Extract and store event details for later tests
        event_id=$(echo "$response" | jq -r '.event_id')
        message_id=$(echo "$response" | jq -r '.message_id')
        echo "Published event_id: $event_id, message_id: $message_id"
        
        # Store for later tests
        export TEST_EVENT_ID="$event_id"
        export TEST_MESSAGE_ID="$message_id"
        export TEST_TOPIC="app.test.events"
    fi
}

# Test 2: Create Consumer Group
test_create_consumer_group() {
    echo -e "\n${YELLOW}=== Test 2: Create Consumer Group ===${NC}"
    
    local payload='{
        "operation": "create_consumer_group",
        "topic": "app.test.events",
        "consumer_group": "test_consumers",
        "start_id": "0"
    }'
    
    make_request "$payload" "create_consumer_group"
}

# Test 3: Subscribe to Events
test_subscribe_events() {
    echo -e "\n${YELLOW}=== Test 3: Subscribe to Events ===${NC}"
    
    local payload='{
        "operation": "subscribe",
        "topics": ["app.test.events"],
        "consumer_group": "test_consumers",
        "consumer_name": "test_consumer_1",
        "filters": {
            "type": "test_event",
            "tags": ["test"]
        },
        "max_events": 10,
        "block_time": 1000
    }'
    
    if make_request "$payload" "subscribe"; then
        # Extract message IDs for acknowledgment test
        local message_ids=$(echo "$response" | jq -r '.acknowledgment.message_ids[]' 2>/dev/null || echo "")
        export TEST_SUBSCRIPTION_MESSAGE_IDS="$message_ids"
        
        local event_count=$(echo "$response" | jq -r '.event_count // 0')
        echo "Received $event_count events"
    fi
}

# Test 4: Acknowledge Messages
test_acknowledge_messages() {
    echo -e "\n${YELLOW}=== Test 4: Acknowledge Messages ===${NC}"
    
    # Only run if we have message IDs from subscription
    if [ -n "$TEST_SUBSCRIPTION_MESSAGE_IDS" ]; then
        local payload='{
            "operation": "ack",
            "topic": "app.test.events",
            "consumer_group": "test_consumers",
            "message_ids": ["'$TEST_SUBSCRIPTION_MESSAGE_IDS'"]
        }'
        
        make_request "$payload" "ack"
    else
        print_status "WARN" "Skipping ACK test - no message IDs available"
    fi
}

# Test 5: Get Pending Messages
test_get_pending() {
    echo -e "\n${YELLOW}=== Test 5: Get Pending Messages ===${NC}"
    
    local payload='{
        "operation": "get_pending",
        "topic": "app.test.events",
        "consumer_group": "test_consumers",
        "consumer_name": "test_consumer_1",
        "max_events": 100
    }'
    
    if make_request "$payload" "get_pending"; then
        local pending_count=$(echo "$response" | jq -r '.pending_count // 0')
        echo "Found $pending_count pending messages"
    fi
}

# Test 6: Replay Events
test_replay_events() {
    echo -e "\n${YELLOW}=== Test 6: Replay Events ===${NC}"
    
    # Replay events from 1 hour ago
    local from_timestamp=$(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%S.%3NZ)
    
    local payload='{
        "operation": "replay",
        "topic": "app.test.events",
        "from_timestamp": "'$from_timestamp'",
        "max_events": 100
    }'
    
    if make_request "$payload" "replay"; then
        local event_count=$(echo "$response" | jq -r '.event_count // 0')
        echo "Replayed $event_count events"
    fi
}

# Test 7: Get Stream Statistics
test_get_stats() {
    echo -e "\n${YELLOW}=== Test 7: Get Stream Statistics ===${NC}"
    
    local payload='{
        "operation": "get_stats",
        "topic": "app.test.events"
    }'
    
    if make_request "$payload" "get_stats"; then
        local total_messages=$(echo "$response" | jq -r '.summary.total_messages // 0')
        local consumer_groups=$(echo "$response" | jq -r '.summary.total_consumer_groups // 0')
        echo "Stream has $total_messages messages and $consumer_groups consumer groups"
    fi
}

# Test 8: Set Retention Policy
test_set_retention() {
    echo -e "\n${YELLOW}=== Test 8: Set Retention Policy ===${NC}"
    
    local payload='{
        "operation": "set_retention",
        "topic": "app.test.events",
        "max_length": 1000,
        "approximate": true
    }'
    
    if make_request "$payload" "set_retention"; then
        local messages_removed=$(echo "$response" | jq -r '.retention_results.messages_removed // 0')
        echo "Retention policy applied, $messages_removed messages removed"
    fi
}

# Test 9: Publish Multiple Events (Batch Test)
test_publish_multiple() {
    echo -e "\n${YELLOW}=== Test 9: Publish Multiple Events ===${NC}"
    
    for i in {1..5}; do
        local payload='{
            "operation": "publish",
            "topic": "app.test.batch",
            "event": {
                "type": "batch_test_event",
                "payload": {
                    "batch_id": "batch_001",
                    "sequence": '$i',
                    "total": 5,
                    "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ)'"
                }
            },
            "tags": ["batch", "test"],
            "ttl": 1800
        }'
        
        echo -e "\n  Publishing event $i/5..."
        if make_request "$payload" "publish"; then
            echo "  ✓ Event $i published successfully"
        else
            echo "  ✗ Event $i failed to publish"
        fi
    done
}

# Test 10: Subscribe with Filters
test_subscribe_with_filters() {
    echo -e "\n${YELLOW}=== Test 10: Subscribe with Filters ===${NC}"
    
    # Create consumer group for batch topic
    local create_group_payload='{
        "operation": "create_consumer_group",
        "topic": "app.test.batch",
        "consumer_group": "batch_consumers",
        "start_id": "0"
    }'
    
    echo "Creating consumer group for batch topic..."
    make_request "$create_group_payload" "create_consumer_group"
    
    # Subscribe with type filter
    local payload='{
        "operation": "subscribe",
        "topics": ["app.test.batch"],
        "consumer_group": "batch_consumers",
        "consumer_name": "batch_consumer_1",
        "filters": {
            "type": "batch_test_event",
            "tags": ["batch"]
        },
        "max_events": 10,
        "block_time": 2000
    }'
    
    if make_request "$payload" "subscribe"; then
        local event_count=$(echo "$response" | jq -r '.event_count // 0')
        local filtered_count=$(echo "$response" | jq -r '.filtered_count // 0')
        echo "Received $event_count events (filtered out $filtered_count)"
    fi
}

# Test 11: Error Handling
test_error_handling() {
    echo -e "\n${YELLOW}=== Test 11: Error Handling ===${NC}"
    
    # Test missing operation
    echo -e "\n  Testing missing operation..."
    local payload1='{"topic": "test"}'
    
    response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$payload1" \
        "$ENDPOINT")
    
    local success=$(echo "$response" | jq -r '.success // true')
    if [ "$success" = "false" ]; then
        print_status "PASS" "Error handling for missing operation"
    else
        print_status "FAIL" "Should have failed for missing operation"
    fi
    
    # Test invalid operation
    echo -e "\n  Testing invalid operation..."
    local payload2='{"operation": "invalid_op"}'
    
    response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$payload2" \
        "$ENDPOINT")
    
    local success=$(echo "$response" | jq -r '.success // true')
    if [ "$success" = "false" ]; then
        print_status "PASS" "Error handling for invalid operation"
    else
        print_status "FAIL" "Should have failed for invalid operation"
    fi
    
    # Test publish without topic
    echo -e "\n  Testing publish without topic..."
    local payload3='{
        "operation": "publish",
        "event": {"type": "test"}
    }'
    
    response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$payload3" \
        "$ENDPOINT")
    
    local success=$(echo "$response" | jq -r '.success // true')
    if [ "$success" = "false" ]; then
        print_status "PASS" "Error handling for publish without topic"
    else
        print_status "FAIL" "Should have failed for publish without topic"
    fi
}

# Test 12: Performance Test
test_performance() {
    echo -e "\n${YELLOW}=== Test 12: Performance Test ===${NC}"
    
    echo "Publishing 20 events rapidly..."
    local start_time=$(date +%s%3N)
    
    for i in {1..20}; do
        local payload='{
            "operation": "publish",
            "topic": "app.test.performance",
            "event": {
                "type": "performance_test",
                "payload": {
                    "sequence": '$i',
                    "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ)'"
                }
            },
            "tags": ["performance", "test"]
        }'
        
        curl -s -X POST \
            -H "Content-Type: application/json" \
            -d "$payload" \
            "$ENDPOINT" > /dev/null &
    done
    
    # Wait for all background processes
    wait
    
    local end_time=$(date +%s%3N)
    local duration=$((end_time - start_time))
    local throughput=$((20000 / duration))
    
    echo "Performance: 20 events in ${duration}ms (~${throughput} events/sec)"
    
    if [ $duration -lt 10000 ]; then  # Less than 10 seconds
        print_status "PASS" "Performance test completed in acceptable time"
    else
        print_status "WARN" "Performance test took longer than expected"
    fi
}

# Main test execution
main() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Event Stream Hub n8n Workflow Test${NC}"
    echo -e "${BLUE}========================================${NC}"
    
    # Prerequisites
    if ! command -v jq &> /dev/null; then
        echo -e "${RED}Error: jq is required but not installed${NC}"
        exit 1
    fi
    
    if ! command -v curl &> /dev/null; then
        echo -e "${RED}Error: curl is required but not installed${NC}"
        exit 1
    fi
    
    # Check services
    check_n8n || exit 1
    wait_for_redis || exit 1
    
    # Run tests
    test_publish_event
    test_create_consumer_group
    test_subscribe_events
    test_acknowledge_messages
    test_get_pending
    test_replay_events
    test_get_stats
    test_set_retention
    test_publish_multiple
    test_subscribe_with_filters
    test_error_handling
    test_performance
    
    # Summary
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}Test Summary${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo -e "Tests Run: $TESTS_RUN"
    echo -e "Tests Passed: $TESTS_PASSED"
    echo -e "Tests Failed: $((TESTS_RUN - TESTS_PASSED))"
    
    if [ $TESTS_PASSED -eq $TESTS_RUN ]; then
        echo -e "${GREEN}All tests passed! ✓${NC}"
        exit 0
    else
        echo -e "${RED}Some tests failed! ✗${NC}"
        exit 1
    fi
}

# Cleanup function
cleanup() {
    echo -e "\n${YELLOW}Cleaning up test data...${NC}"
    
    # Clean up test streams (optional - Redis will auto-expire based on TTL)
    redis-cli DEL "stream:app.test.events" "stream:app.test.batch" "stream:app.test.performance" >/dev/null 2>&1 || true
    
    echo "Cleanup completed"
}

# Trap cleanup on script exit
trap cleanup EXIT

# Run main function
main "$@"