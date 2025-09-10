#!/bin/bash
# Apache Kafka Resource - Test Library Functions
# v2.0 Contract Implementation

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Source configuration
source "$SCRIPT_DIR/config/defaults.sh"
source "$SCRIPT_DIR/lib/core.sh"

# Run smoke tests (quick health check - <30s)
test_smoke() {
    echo "=== Kafka Smoke Test ==="
    local start_time=$(date +%s)
    local test_passed=0
    
    # Test 1: Check if container is running
    echo -n "1. Checking container status... "
    if docker ps --format '{{.Names}}' | grep -q "^${KAFKA_CONTAINER_NAME}$"; then
        echo "PASS"
    else
        echo "FAIL (container not running)"
        test_passed=1
    fi
    
    # Test 2: Check broker health
    echo -n "2. Checking broker health... "
    if check_health; then
        echo "PASS"
    else
        echo "FAIL (broker not responding)"
        test_passed=1
    fi
    
    # Test 3: Quick topic creation test
    echo -n "3. Testing topic creation... "
    local test_topic="smoke-test-$(date +%s)"
    if docker exec "$KAFKA_CONTAINER_NAME" \
        /opt/kafka/bin/kafka-topics.sh \
        --create \
        --topic "$test_topic" \
        --partitions 1 \
        --replication-factor 1 \
        --bootstrap-server "localhost:${KAFKA_PORT}" >/dev/null 2>&1; then
        echo "PASS"
        # Clean up test topic
        docker exec "$KAFKA_CONTAINER_NAME" \
            /opt/kafka/bin/kafka-topics.sh \
            --delete \
            --topic "$test_topic" \
            --bootstrap-server "localhost:${KAFKA_PORT}" >/dev/null 2>&1
    else
        echo "FAIL (could not create topic)"
        test_passed=1
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo ""
    echo "Smoke test completed in ${duration}s"
    
    if [ $test_passed -eq 0 ]; then
        echo "Result: ALL TESTS PASSED"
        return 0
    else
        echo "Result: SOME TESTS FAILED"
        return 1
    fi
}

# Run integration tests (full functionality - <120s)
test_integration() {
    echo "=== Kafka Integration Test ==="
    local start_time=$(date +%s)
    local test_passed=0
    
    # Test 1: Create test topic
    echo -n "1. Creating test topic... "
    local test_topic="integration-test-$(date +%s)"
    if add_topic "$test_topic" --partitions 3; then
        echo "PASS"
    else
        echo "FAIL"
        test_passed=1
    fi
    
    # Test 2: List topics
    echo -n "2. Listing topics... "
    if list_topics | grep -q "$test_topic"; then
        echo "PASS"
    else
        echo "FAIL"
        test_passed=1
    fi
    
    # Test 3: Produce message
    echo -n "3. Producing test message... "
    if echo "Test message $(date)" | docker exec -i "$KAFKA_CONTAINER_NAME" \
        /opt/kafka/bin/kafka-console-producer.sh \
        --topic "$test_topic" \
        --bootstrap-server "localhost:${KAFKA_PORT}" >/dev/null 2>&1; then
        echo "PASS"
    else
        echo "FAIL"
        test_passed=1
    fi
    
    # Test 4: Consume message
    echo -n "4. Consuming test message... "
    if timeout 10 docker exec "$KAFKA_CONTAINER_NAME" \
        /opt/kafka/bin/kafka-console-consumer.sh \
        --topic "$test_topic" \
        --from-beginning \
        --max-messages 1 \
        --bootstrap-server "localhost:${KAFKA_PORT}" >/dev/null 2>&1; then
        echo "PASS"
    else
        echo "FAIL"
        test_passed=1
    fi
    
    # Test 5: Describe topic
    echo -n "5. Describing topic... "
    if describe_topic "$test_topic" | grep -q "PartitionCount: 3"; then
        echo "PASS"
    else
        echo "FAIL"
        test_passed=1
    fi
    
    # Test 6: Delete topic
    echo -n "6. Deleting test topic... "
    if remove_topic "$test_topic" >/dev/null 2>&1; then
        echo "PASS"
    else
        echo "FAIL"
        test_passed=1
    fi
    
    # Test 7: Performance test
    echo -n "7. Performance test (1000 messages)... "
    local perf_topic="perf-test-$(date +%s)"
    add_topic "$perf_topic" --partitions 3 >/dev/null 2>&1
    
    # Produce 1000 messages
    local perf_start=$(date +%s%N)
    for i in {1..1000}; do
        echo "Message $i"
    done | docker exec -i "$KAFKA_CONTAINER_NAME" \
        /opt/kafka/bin/kafka-console-producer.sh \
        --topic "$perf_topic" \
        --bootstrap-server "localhost:${KAFKA_PORT}" >/dev/null 2>&1
    
    local perf_end=$(date +%s%N)
    local perf_duration=$(( (perf_end - perf_start) / 1000000 ))
    
    if [ $perf_duration -lt 10000 ]; then  # Less than 10 seconds
        echo "PASS (${perf_duration}ms)"
    else
        echo "FAIL (too slow: ${perf_duration}ms)"
        test_passed=1
    fi
    
    # Clean up
    remove_topic "$perf_topic" >/dev/null 2>&1
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo ""
    echo "Integration test completed in ${duration}s"
    
    if [ $test_passed -eq 0 ]; then
        echo "Result: ALL TESTS PASSED"
        return 0
    else
        echo "Result: SOME TESTS FAILED"
        return 1
    fi
}

# Run unit tests (library functions - <60s)
test_unit() {
    echo "=== Kafka Unit Test ==="
    local start_time=$(date +%s)
    local test_passed=0
    
    # Test 1: Configuration validation
    echo -n "1. Testing configuration validation... "
    if validate_config; then
        echo "PASS"
    else
        echo "FAIL"
        test_passed=1
    fi
    
    # Test 2: Port availability check
    echo -n "2. Testing port availability check... "
    # Temporarily use a known used port
    local original_port="$KAFKA_PORT"
    export KAFKA_PORT="22"  # SSH port, likely in use
    if ! validate_config 2>/dev/null; then
        echo "PASS (correctly detected port conflict)"
    else
        echo "FAIL (did not detect port conflict)"
        test_passed=1
    fi
    export KAFKA_PORT="$original_port"
    
    # Test 3: Docker availability
    echo -n "3. Testing Docker availability check... "
    if command -v docker &> /dev/null; then
        echo "PASS"
    else
        echo "FAIL"
        test_passed=1
    fi
    
    # Test 4: Network creation function
    echo -n "4. Testing network creation... "
    if docker network ls | grep -q "$KAFKA_NETWORK"; then
        echo "PASS (network exists)"
    else
        echo "FAIL (network missing)"
        test_passed=1
    fi
    
    # Test 5: Volume creation function
    echo -n "5. Testing volume creation... "
    if docker volume ls | grep -q "$KAFKA_VOLUME_NAME"; then
        echo "PASS (volume exists)"
    else
        echo "FAIL (volume missing)"
        test_passed=1
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo ""
    echo "Unit test completed in ${duration}s"
    
    if [ $test_passed -eq 0 ]; then
        echo "Result: ALL TESTS PASSED"
        return 0
    else
        echo "Result: SOME TESTS FAILED"
        return 1
    fi
}

# Run all tests
test_all() {
    echo "=== Running All Kafka Tests ==="
    local overall_result=0
    
    # Run unit tests
    echo ""
    echo "[1/3] Running unit tests..."
    if ! test_unit; then
        overall_result=1
    fi
    
    # Run smoke tests
    echo ""
    echo "[2/3] Running smoke tests..."
    if ! test_smoke; then
        overall_result=1
    fi
    
    # Run integration tests
    echo ""
    echo "[3/3] Running integration tests..."
    if ! test_integration; then
        overall_result=1
    fi
    
    echo ""
    echo "==============================="
    if [ $overall_result -eq 0 ]; then
        echo "OVERALL RESULT: ALL TESTS PASSED"
    else
        echo "OVERALL RESULT: SOME TESTS FAILED"
    fi
    
    return $overall_result
}