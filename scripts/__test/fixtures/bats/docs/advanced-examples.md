# Advanced BATS Testing Examples

This document provides comprehensive examples for advanced testing scenarios using the Vrooli BATS testing infrastructure.

## Table of Contents

1. [Complex Integration Testing](#complex-integration-testing)
2. [Performance and Load Testing](#performance-and-load-testing)
3. [Error Injection and Recovery Testing](#error-injection-and-recovery-testing)
4. [Mock Verification Patterns](#mock-verification-patterns)
5. [Data Pipeline Testing](#data-pipeline-testing)
6. [Security Testing](#security-testing)
7. [Multi-Resource Workflows](#multi-resource-workflows)
8. [Custom Assertion Development](#custom-assertion-development)

## Complex Integration Testing

### Multi-Service AI Pipeline

```bash
#!/usr/bin/env bats

# Load test infrastructure
source "${VROOLI_TEST_FIXTURES_DIR}/core/common_setup.bash"

setup() {
    # Setup integration environment with multiple AI services
    setup_integration_test "ollama" "whisper" "questdb" "n8n"
    
    # Configure AI pipeline environment
    export AI_PIPELINE_ID="ai_pipeline_$$"
    export AI_PIPELINE_CONFIG="$TEST_TMPDIR/ai_pipeline_config.json"
    
    # Generate pipeline configuration
    cat > "$AI_PIPELINE_CONFIG" << EOF
{
    "pipeline_id": "$AI_PIPELINE_ID",
    "stages": [
        {"name": "audio_ingestion", "service": "whisper"},
        {"name": "text_processing", "service": "ollama"},
        {"name": "data_storage", "service": "questdb"},
        {"name": "workflow_orchestration", "service": "n8n"}
    ]
}
EOF

    # Setup mock response sequences for complex workflows
    mock::http::set_endpoint_sequence "$WHISPER_BASE_URL/transcribe" \
        '{"text":"First transcription","confidence":0.95},{"text":"Second transcription","confidence":0.92},{"text":"Final transcription","confidence":0.98}'
    
    # Setup verification expectations
    mock::verify::expect_sequence "ai_pipeline_flow" \
        "whisper transcribe ollama generate questdb insert n8n execute"
}

@test "ai pipeline: complete audio-to-insights workflow" {
    local pipeline_start=$(date +%s)
    
    # Stage 1: Audio transcription
    local audio_file="$TEST_TMPDIR/test_audio.wav"
    echo "mock audio content" > "$audio_file"
    
    run curl -s -X POST "$WHISPER_BASE_URL/transcribe" \
        -F "audio=@${audio_file}"
    assert_success
    assert_json_field_exists "$output" ".text"
    
    local transcribed_text=$(echo "$output" | jq -r '.text')
    
    # Stage 2: AI text processing
    run curl -s -X POST "$OLLAMA_BASE_URL/api/generate" \
        -H "Content-Type: application/json" \
        -d '{"model":"llama3.1:8b","prompt":"Analyze: '"$transcribed_text"'","stream":false}'
    assert_success
    assert_json_field_exists "$output" ".response"
    
    local ai_analysis=$(echo "$output" | jq -r '.response')
    
    # Stage 3: Store results in time-series database
    run curl -s -G "$QUESTDB_BASE_URL/exec" \
        --data-urlencode "query=INSERT INTO ai_insights VALUES (now(), 'audio_analysis', '$(echo "$ai_analysis" | sed "s/'/''/g")', '$AI_PIPELINE_ID')" \
        --data-urlencode "fmt=json"
    assert_success
    
    # Stage 4: Trigger workflow automation
    run curl -s -X POST "$N8N_BASE_URL/api/v1/workflows/execute" \
        -H "Content-Type: application/json" \
        -d '{"workflowId":"ai_insights_processor","data":{"pipeline_id":"'"$AI_PIPELINE_ID"'","analysis":"'"$ai_analysis"'"}}'
    assert_success
    
    local pipeline_duration=$(($(date +%s) - pipeline_start))
    
    # Validate pipeline performance
    assert_less_than "$pipeline_duration" "60"  # Should complete within 1 minute
    
    echo "AI pipeline completed in ${pipeline_duration}s"
}

@test "ai pipeline: parallel processing with data aggregation" {
    local parallel_tasks=3
    local pids=()
    local results_dir="$TEST_TMPDIR/parallel_results"
    mkdir -p "$results_dir"
    
    # Start parallel AI processing tasks
    for task_id in $(seq 1 $parallel_tasks); do
        (
            local task_start=$(date +%s)
            
            # Each task processes different data
            local task_data="Task $task_id data for parallel processing"
            
            # Parallel Ollama requests
            local ai_result=$(curl -s -X POST "$OLLAMA_BASE_URL/api/generate" \
                -H "Content-Type: application/json" \
                -d '{"model":"llama3.1:8b","prompt":"Process task '"$task_id"': '"$task_data"'","stream":false}' \
                | jq -r '.response')
            
            # Store individual task results
            curl -s -G "$QUESTDB_BASE_URL/exec" \
                --data-urlencode "query=INSERT INTO parallel_tasks VALUES (now(), 'task_$task_id', '$(echo "$ai_result" | sed "s/'/''/g")', '$AI_PIPELINE_ID')" \
                --data-urlencode "fmt=json" > "$results_dir/task_${task_id}_storage.json"
            
            local task_duration=$(($(date +%s) - task_start))
            echo "$task_id:$task_duration" > "$results_dir/task_${task_id}_timing.txt"
            
        ) &
        pids+=($!)
    done
    
    # Wait for all parallel tasks to complete
    for pid in "${pids[@]}"; do
        wait "$pid"
    done
    
    # Aggregate results
    run curl -s -G "$QUESTDB_BASE_URL/exec" \
        --data-urlencode "query=SELECT task_name, COUNT(*) as count FROM parallel_tasks WHERE pipeline_id = '$AI_PIPELINE_ID' GROUP BY task_name" \
        --data-urlencode "fmt=json"
    assert_success
    
    local task_count=$(echo "$output" | jq '.dataset | length')
    assert_equals "$task_count" "$parallel_tasks"
    
    echo "Parallel processing completed: $parallel_tasks tasks"
}
```

## Performance and Load Testing

### Resource Stress Testing with Monitoring

```bash
@test "performance: resource stress testing with real-time monitoring" {
    local stress_duration=120  # 2 minutes
    local monitoring_interval=10  # 10 seconds
    local max_concurrent_requests=50
    
    # Setup monitoring
    local monitoring_file="$TEST_TMPDIR/performance_monitoring.log"
    echo "timestamp,cpu_percent,memory_mb,response_time_ms,active_connections" > "$monitoring_file"
    
    # Start background monitoring
    (
        while [[ -f "$TEST_TMPDIR/stress_test_running" ]]; do
            local timestamp=$(date -Iseconds)
            local cpu_usage=$(docker stats "$OLLAMA_CONTAINER_NAME" --no-stream --format "{{.CPUPerc}}" | sed 's/%//')
            local memory_usage=$(docker stats "$OLLAMA_CONTAINER_NAME" --no-stream --format "{{.MemUsage}}" | cut -d'/' -f1 | sed 's/MiB//')
            
            # Test response time
            local response_start=$(date +%s%N)
            curl -s "$OLLAMA_BASE_URL/api/tags" >/dev/null
            local response_end=$(date +%s%N)
            local response_time=$(( (response_end - response_start) / 1000000 ))
            
            # Count active connections (mock)
            local active_connections=$((RANDOM % 20 + 5))
            
            echo "$timestamp,$cpu_usage,$memory_usage,$response_time,$active_connections" >> "$monitoring_file"
            sleep "$monitoring_interval"
        done
    ) &
    local monitoring_pid=$!
    
    # Create stress test marker
    touch "$TEST_TMPDIR/stress_test_running"
    
    # Execute stress test
    local stress_start=$(date +%s)
    local total_requests=0
    local successful_requests=0
    local failed_requests=0
    local response_times=()
    
    while [[ $(($(date +%s) - stress_start)) -lt $stress_duration ]]; do
        local batch_pids=()
        
        # Send batch of concurrent requests
        for i in $(seq 1 "$max_concurrent_requests"); do
            (
                local request_start=$(date +%s%N)
                local http_code=$(curl -s -o /dev/null -w "%{http_code}" "$OLLAMA_BASE_URL/api/tags")
                local request_end=$(date +%s%N)
                local request_time=$(( (request_end - request_start) / 1000000 ))
                
                echo "$http_code:$request_time" > "$TEST_TMPDIR/request_${i}_$$_result.tmp"
            ) &
            batch_pids+=($!)
        done
        
        # Wait for batch completion
        for pid in "${batch_pids[@]}"; do
            wait "$pid"
        done
        
        # Collect batch results
        for i in $(seq 1 "$max_concurrent_requests"); do
            local result_file="$TEST_TMPDIR/request_${i}_$$_result.tmp"
            if [[ -f "$result_file" ]]; then
                local result=$(cat "$result_file")
                local http_code=$(echo "$result" | cut -d':' -f1)
                local request_time=$(echo "$result" | cut -d':' -f2)
                
                total_requests=$((total_requests + 1))
                response_times+=("$request_time")
                
                if [[ "$http_code" =~ ^(200|201|202)$ ]]; then
                    successful_requests=$((successful_requests + 1))
                else
                    failed_requests=$((failed_requests + 1))
                fi
                
                rm -f "$result_file"
            fi
        done
        
        # Brief pause between batches
        sleep 1
    done
    
    # Stop monitoring
    rm -f "$TEST_TMPDIR/stress_test_running"
    kill "$monitoring_pid" 2>/dev/null || true
    wait "$monitoring_pid" 2>/dev/null || true
    
    # Calculate performance metrics
    local success_rate=$(echo "scale=2; $successful_requests * 100 / $total_requests" | bc)
    local avg_response_time=$(echo "${response_times[@]}" | tr ' ' '\n' | awk '{sum+=$1} END {print int(sum/NR)}')
    local requests_per_second=$(echo "scale=2; $total_requests / $stress_duration" | bc)
    
    # Analyze monitoring data
    local max_cpu=$(tail -n +2 "$monitoring_file" | cut -d',' -f2 | sort -n | tail -1)
    local max_memory=$(tail -n +2 "$monitoring_file" | cut -d',' -f3 | sort -n | tail -1)
    local max_response_time=$(tail -n +2 "$monitoring_file" | cut -d',' -f4 | sort -n | tail -1)
    
    # Generate performance report
    cat > "$TEST_TMPDIR/stress_test_report.json" << EOF
{
    "test_duration_seconds": $stress_duration,
    "total_requests": $total_requests,
    "successful_requests": $successful_requests,
    "failed_requests": $failed_requests,
    "success_rate": $success_rate,
    "requests_per_second": $requests_per_second,
    "avg_response_time_ms": $avg_response_time,
    "resource_usage": {
        "max_cpu_percent": $max_cpu,
        "max_memory_mb": $max_memory,
        "max_response_time_ms": $max_response_time
    }
}
EOF

    # Validate performance under stress
    assert_greater_than "$success_rate" "90"  # At least 90% success under stress
    assert_less_than "$avg_response_time" "2000"  # Average response under 2s
    assert_less_than "$max_cpu" "90"  # CPU usage under 90%
    
    echo "Stress test completed: ${requests_per_second} RPS, ${success_rate}% success, ${avg_response_time}ms avg"
}
```

## Error Injection and Recovery Testing

### Chaos Engineering for Resilience Testing

```bash
@test "resilience: chaos engineering with automatic recovery" {
    # Setup chaos testing environment
    local chaos_scenarios=("network_partition" "resource_exhaustion" "service_crash" "data_corruption")
    local recovery_results=()
    
    for scenario in "${chaos_scenarios[@]}"; do
        echo "Testing chaos scenario: $scenario"
        
        # Record initial state
        local initial_health=$(curl -s "$OLLAMA_BASE_URL/api/tags" | jq -r '.models | length')
        
        case "$scenario" in
            "network_partition")
                # Simulate network issues
                mock::http::set_endpoint_response "$OLLAMA_BASE_URL/api/tags" \
                    '{"error":"Connection refused"}' 0
                
                # Test service behavior during network issues
                run curl -s "$OLLAMA_BASE_URL/api/tags"
                assert_failure
                
                # Restore network
                mock::http::set_endpoint_response "$OLLAMA_BASE_URL/api/tags" \
                    '{"models":[{"name":"llama3.1:8b","size":4900000000}]}' 200
                
                # Verify recovery
                run curl -s "$OLLAMA_BASE_URL/api/tags"
                assert_success
                assert_json_field_exists "$output" ".models"
                
                recovery_results+=("network_partition:recovered")
                ;;
                
            "resource_exhaustion")
                # Simulate resource exhaustion
                mock::docker::set_container_state "$OLLAMA_CONTAINER_NAME" "unhealthy"
                
                # Test service behavior under resource pressure
                run docker inspect "$OLLAMA_CONTAINER_NAME" --format '{{.State.Status}}'
                assert_output "unhealthy"
                
                # Simulate resource recovery
                mock::docker::set_container_state "$OLLAMA_CONTAINER_NAME" "running"
                
                # Verify service recovery
                run docker inspect "$OLLAMA_CONTAINER_NAME" --format '{{.State.Status}}'
                assert_output "running"
                
                recovery_results+=("resource_exhaustion:recovered")
                ;;
                
            "service_crash")
                # Simulate service crash
                mock::docker::set_container_state "$OLLAMA_CONTAINER_NAME" "exited"
                
                # Test crash detection
                run docker inspect "$OLLAMA_CONTAINER_NAME" --format '{{.State.Status}}'
                assert_output "exited"
                
                # Simulate restart
                mock::docker::set_container_state "$OLLAMA_CONTAINER_NAME" "running"
                
                # Verify service restart
                run docker inspect "$OLLAMA_CONTAINER_NAME" --format '{{.State.Status}}'
                assert_output "running"
                
                recovery_results+=("service_crash:recovered")
                ;;
                
            "data_corruption")
                # Simulate data corruption in responses
                mock::http::set_endpoint_response "$OLLAMA_BASE_URL/api/tags" \
                    '{"corrupted":invalid_json}' 200
                
                # Test handling of corrupted data
                run curl -s "$OLLAMA_BASE_URL/api/tags"
                assert_success
                # Response should be invalid JSON
                run echo "$output"
                assert_output_not_contains '"models"'
                
                # Restore valid data
                mock::http::set_endpoint_response "$OLLAMA_BASE_URL/api/tags" \
                    '{"models":[{"name":"llama3.1:8b","size":4900000000}]}' 200
                
                # Verify data recovery
                run curl -s "$OLLAMA_BASE_URL/api/tags"
                assert_success
                assert_json_valid "$output"
                
                recovery_results+=("data_corruption:recovered")
                ;;
        esac
        
        echo "Chaos scenario '$scenario' completed"
    done
    
    # Validate all scenarios recovered successfully
    for result in "${recovery_results[@]}"; do
        local scenario=$(echo "$result" | cut -d':' -f1)
        local status=$(echo "$result" | cut -d':' -f2)
        assert_equals "$status" "recovered"
    done
    
    echo "Chaos engineering completed: ${#chaos_scenarios[@]} scenarios tested"
}
```

## Mock Verification Patterns

### Advanced Mock Call Verification

```bash
@test "mocks: advanced verification patterns" {
    # Setup complex mock verification scenario
    mock::verify::expect_calls "http" "GET.*ollama.*tags" 3
    mock::verify::expect_calls "http" "POST.*ollama.*generate" 2
    mock::verify::expect_sequence "ollama_workflow" "tags generate generate tags"
    
    # Execute operations that should trigger mock calls
    
    # Step 1: Check available models
    run curl -s "$OLLAMA_BASE_URL/api/tags"
    assert_success
    
    # Step 2: Generate text (first call)
    run curl -s -X POST "$OLLAMA_BASE_URL/api/generate" \
        -H "Content-Type: application/json" \
        -d '{"model":"llama3.1:8b","prompt":"First prompt","stream":false}'
    assert_success
    
    # Step 3: Generate text (second call)
    run curl -s -X POST "$OLLAMA_BASE_URL/api/generate" \
        -H "Content-Type: application/json" \
        -d '{"model":"llama3.1:8b","prompt":"Second prompt","stream":false}'
    assert_success
    
    # Step 4: Check models again
    run curl -s "$OLLAMA_BASE_URL/api/tags"
    assert_success
    
    # Step 5: Final model check
    run curl -s "$OLLAMA_BASE_URL/api/tags"
    assert_success
    
    # Verify specific call counts
    mock::verify::assert_called "http" "GET http://localhost:11434/api/tags" 3
    mock::verify::assert_called "http" "POST http://localhost:11434/api/generate" 2
    
    # Verify call patterns
    mock::verify::assert_min_calls "http" "ollama" 5
    mock::verify::assert_max_calls "http" "generate" 2
    
    # Print verification summary
    mock::verify::print_summary
}

@test "mocks: conditional verification based on test outcomes" {
    # Setup conditional verification
    local test_scenario="conditional_verification"
    local scenario_success=true
    
    # Attempt complex operation
    run curl -s -X POST "$OLLAMA_BASE_URL/api/generate" \
        -H "Content-Type: application/json" \
        -d '{"model":"llama3.1:8b","prompt":"Complex operation","stream":false}'
    
    if [[ $status -eq 0 ]]; then
        # If operation succeeded, verify follow-up calls were made
        mock::verify::expect_calls "http" "GET.*tags" 1
        
        # Make the expected follow-up call
        curl -s "$OLLAMA_BASE_URL/api/tags" >/dev/null
        
        mock::verify::assert_called "http" "GET http://localhost:11434/api/tags" 1
    else
        # If operation failed, verify error handling calls
        scenario_success=false
        mock::verify::expect_calls "http" "GET.*health" 1
        
        # Make error handling call
        curl -s "$OLLAMA_BASE_URL/health" >/dev/null
        
        mock::verify::assert_called "http" "GET http://localhost:11434/health" 1
    fi
    
    echo "Conditional verification completed, scenario success: $scenario_success"
}
```

## Data Pipeline Testing

### Real-time Data Processing Pipeline

```bash
@test "data pipeline: real-time streaming with quality validation" {
    # Setup data pipeline components
    local pipeline_id="realtime_pipeline_$$"
    local data_quality_threshold=0.95
    local processing_rate_threshold=100  # messages per second
    
    # Initialize pipeline tables
    curl -s -G "$QUESTDB_BASE_URL/exec" \
        --data-urlencode "query=CREATE TABLE IF NOT EXISTS realtime_events (timestamp TIMESTAMP, event_type STRING, data STRING, quality_score DOUBLE) timestamp(timestamp)" >/dev/null
    
    # Setup Redis streams for data ingestion
    redis-cli -h localhost -p "$REDIS_PORT" DEL "pipeline:$pipeline_id:events" >/dev/null
    
    # Generate streaming data
    local data_generator_pid
    (
        local event_count=0
        while [[ $event_count -lt 1000 ]]; do
            local timestamp=$(date -Iseconds)
            local event_type="sensor_reading"
            local data_value=$((RANDOM % 100))
            local quality_score=$(echo "scale=2; (90 + $RANDOM % 10) / 100" | bc)
            
            # Add to Redis stream
            redis-cli -h localhost -p "$REDIS_PORT" XADD "pipeline:$pipeline_id:events" "*" \
                "timestamp" "$timestamp" \
                "event_type" "$event_type" \
                "data" "$data_value" \
                "quality_score" "$quality_score" >/dev/null
            
            event_count=$((event_count + 1))
            sleep 0.01  # 100 events per second
        done
    ) &
    data_generator_pid=$!
    
    # Process streaming data
    local processor_pid
    (
        local processed_count=0
        local high_quality_count=0
        
        while [[ $processed_count -lt 1000 ]]; do
            # Read from stream
            local stream_data=$(redis-cli -h localhost -p "$REDIS_PORT" XREAD COUNT 10 STREAMS "pipeline:$pipeline_id:events" 0-0 2>/dev/null)
            
            if [[ -n "$stream_data" && "$stream_data" != "(nil)" ]]; then
                # Parse and validate data quality
                local batch_quality=0
                local batch_count=0
                
                # Simulate data processing and quality assessment
                for i in {1..10}; do
                    local quality=$(echo "scale=2; (90 + $RANDOM % 10) / 100" | bc)
                    batch_quality=$(echo "scale=2; $batch_quality + $quality" | bc)
                    batch_count=$((batch_count + 1))
                    
                    if (( $(echo "$quality >= $data_quality_threshold" | bc -l) )); then
                        high_quality_count=$((high_quality_count + 1))
                    fi
                done
                
                local avg_quality=$(echo "scale=2; $batch_quality / $batch_count" | bc)
                
                # Store processed data in QuestDB
                curl -s -G "$QUESTDB_BASE_URL/exec" \
                    --data-urlencode "query=INSERT INTO realtime_events VALUES (now(), 'processed_batch', 'batch_$processed_count', $avg_quality)" >/dev/null
                
                processed_count=$((processed_count + batch_count))
            fi
            
            sleep 0.1
        done
        
        echo "$processed_count:$high_quality_count" > "$TEST_TMPDIR/processing_results.txt"
    ) &
    processor_pid=$!
    
    # Wait for data generation to complete
    wait "$data_generator_pid"
    echo "Data generation completed"
    
    # Wait for processing to complete
    wait "$processor_pid"
    echo "Data processing completed"
    
    # Analyze pipeline performance
    local processing_results=$(cat "$TEST_TMPDIR/processing_results.txt")
    local total_processed=$(echo "$processing_results" | cut -d':' -f1)
    local high_quality_processed=$(echo "$processing_results" | cut -d':' -f2)
    
    # Validate data quality
    local quality_rate=$(echo "scale=2; $high_quality_processed * 100 / $total_processed" | bc)
    
    # Check processing rate
    run redis-cli -h localhost -p "$REDIS_PORT" XLEN "pipeline:$pipeline_id:events"
    assert_success
    local total_events="$output"
    
    # Verify QuestDB storage
    run curl -s -G "$QUESTDB_BASE_URL/exec" \
        --data-urlencode "query=SELECT COUNT(*) as count FROM realtime_events WHERE event_type = 'processed_batch'" \
        --data-urlencode "fmt=json"
    assert_success
    
    local stored_batches=$(echo "$output" | jq -r '.dataset[0][0]')
    
    # Validate pipeline performance
    assert_greater_than "$quality_rate" "$((data_quality_threshold * 100))"
    assert_greater_than "$total_processed" "900"  # Should process most events
    assert_greater_than "$stored_batches" "90"   # Should store most batches
    
    echo "Pipeline completed: $total_processed events processed, ${quality_rate}% high quality, $stored_batches batches stored"
}
```

## Security Testing

### Authentication and Authorization Testing

```bash
@test "security: authentication and authorization validation" {
    # Setup security test environment
    local security_test_id="security_test_$$"
    local unauthorized_endpoints=()
    local protected_endpoints=()
    
    # Test endpoints without authentication
    local test_endpoints=("$OLLAMA_BASE_URL/api/tags" "$QUESTDB_BASE_URL/status" "$N8N_BASE_URL/healthz")
    
    for endpoint in "${test_endpoints[@]}"; do
        echo "Testing endpoint security: $endpoint"
        
        # Test without authentication (should work for health endpoints)
        run curl -s -o /dev/null -w "%{http_code}" "$endpoint"
        local status_code="$output"
        
        if [[ "$status_code" =~ ^(200|201|202)$ ]]; then
            unauthorized_endpoints+=("$endpoint")
        else
            protected_endpoints+=("$endpoint")
        fi
    done
    
    # Test with malicious payloads
    local malicious_payloads=(
        '{"__proto__":{"polluted":true}}'  # Prototype pollution
        '{"eval":"process.exit()"}'        # Code injection attempt
        '{"script":"<script>alert(1)</script>"}' # XSS attempt
        '{"sql":"1; DROP TABLE users; --"}'  # SQL injection attempt
    )
    
    local injection_blocked=0
    local total_injection_tests=0
    
    for payload in "${malicious_payloads[@]}"; do
        total_injection_tests=$((total_injection_tests + 1))
        
        # Test injection on POST endpoints
        run curl -s -X POST "$OLLAMA_BASE_URL/api/generate" \
            -H "Content-Type: application/json" \
            -d "$payload" \
            -w "%{http_code}"
        
        # Check if injection was properly handled/blocked
        if [[ ! "$output" =~ (500|502|503) ]]; then
            injection_blocked=$((injection_blocked + 1))
        fi
    done
    
    # Test data validation and sanitization
    local validation_tests=0
    local validation_passed=0
    
    # Test oversized payloads
    local large_payload=$(head -c 10485760 < /dev/urandom | base64 | tr -d '\n')  # 10MB payload
    validation_tests=$((validation_tests + 1))
    
    run curl -s -X POST "$OLLAMA_BASE_URL/api/generate" \
        -H "Content-Type: application/json" \
        -d '{"model":"llama3.1:8b","prompt":"'"$large_payload"'"}' \
        -w "%{http_code}" \
        --max-time 5
    
    # Should handle large payloads gracefully
    if [[ $status -ne 0 ]] || [[ "$output" =~ (413|400|502|503) ]]; then
        validation_passed=$((validation_passed + 1))
    fi
    
    # Test invalid JSON
    validation_tests=$((validation_tests + 1))
    
    run curl -s -X POST "$OLLAMA_BASE_URL/api/generate" \
        -H "Content-Type: application/json" \
        -d '{invalid json}' \
        -w "%{http_code}"
    
    # Should reject invalid JSON
    if [[ "$output" =~ (400|422) ]]; then
        validation_passed=$((validation_passed + 1))
    fi
    
    # Generate security test report
    cat > "$TEST_TMPDIR/security_report.json" << EOF
{
    "test_id": "$security_test_id",
    "endpoint_security": {
        "unauthorized_access": $(printf '%s\n' "${unauthorized_endpoints[@]}" | jq -R . | jq -s '.'),
        "protected_endpoints": $(printf '%s\n' "${protected_endpoints[@]}" | jq -R . | jq -s '.')
    },
    "injection_protection": {
        "total_tests": $total_injection_tests,
        "blocked_attempts": $injection_blocked,
        "protection_rate": $(echo "scale=2; $injection_blocked * 100 / $total_injection_tests" | bc)
    },
    "input_validation": {
        "total_tests": $validation_tests,
        "validation_passed": $validation_passed,
        "validation_rate": $(echo "scale=2; $validation_passed * 100 / $validation_tests" | bc)
    }
}
EOF

    # Validate security measures
    local injection_protection_rate=$(echo "scale=0; $injection_blocked * 100 / $total_injection_tests" | bc)
    local validation_rate=$(echo "scale=0; $validation_passed * 100 / $validation_tests" | bc)
    
    assert_greater_than "$injection_protection_rate" "80"  # At least 80% injection protection
    assert_greater_than "$validation_rate" "80"           # At least 80% validation success
    
    echo "Security testing completed: ${injection_protection_rate}% injection protection, ${validation_rate}% validation success"
}
```

## Custom Assertion Development

### Building Domain-Specific Assertions

```bash
#!/usr/bin/env bash
# Custom assertions for AI and data processing testing

#######################################
# Assert AI model response quality
# Arguments: $1 - response text, $2 - minimum quality score
#######################################
assert_ai_response_quality() {
    local response="$1"
    local min_quality="${2:-0.7}"
    
    # Simple quality metrics
    local word_count=$(echo "$response" | wc -w)
    local char_count=$(echo "$response" | wc -c)
    local sentence_count=$(echo "$response" | grep -o '\.' | wc -l)
    
    # Quality score calculation (simplified)
    local quality_score=0
    
    # Word count factor (reasonable response length)
    if [[ $word_count -ge 10 && $word_count -le 1000 ]]; then
        quality_score=$(echo "scale=2; $quality_score + 0.3" | bc)
    fi
    
    # Character diversity factor
    local unique_chars=$(echo "$response" | grep -o . | sort -u | wc -l)
    if [[ $unique_chars -ge 20 ]]; then
        quality_score=$(echo "scale=2; $quality_score + 0.3" | bc)
    fi
    
    # Sentence structure factor
    if [[ $sentence_count -ge 1 ]]; then
        quality_score=$(echo "scale=2; $quality_score + 0.4" | bc)
    fi
    
    # Compare with minimum quality
    local quality_check=$(echo "$quality_score >= $min_quality" | bc -l)
    
    if [[ $quality_check -eq 0 ]]; then
        echo "AI response quality below threshold:" >&2
        echo "  Response: $response" >&2
        echo "  Quality score: $quality_score" >&2
        echo "  Minimum required: $min_quality" >&2
        return 1
    fi
    
    return 0
}

#######################################
# Assert data pipeline throughput
# Arguments: $1 - actual throughput, $2 - expected throughput, $3 - tolerance percentage
#######################################
assert_pipeline_throughput() {
    local actual="$1"
    local expected="$2"
    local tolerance="${3:-10}"  # 10% tolerance by default
    
    local tolerance_value=$(echo "scale=2; $expected * $tolerance / 100" | bc)
    local min_threshold=$(echo "scale=2; $expected - $tolerance_value" | bc)
    local max_threshold=$(echo "scale=2; $expected + $tolerance_value" | bc)
    
    local within_range=$(echo "$actual >= $min_threshold && $actual <= $max_threshold" | bc -l)
    
    if [[ $within_range -eq 0 ]]; then
        echo "Pipeline throughput outside acceptable range:" >&2
        echo "  Actual: $actual" >&2
        echo "  Expected: $expected (±${tolerance}%)" >&2
        echo "  Range: $min_threshold - $max_threshold" >&2
        return 1
    fi
    
    return 0
}

#######################################
# Assert time-series data continuity
# Arguments: $1 - data points JSON array, $2 - max gap seconds
#######################################
assert_timeseries_continuity() {
    local data_points="$1"
    local max_gap="${2:-300}"  # 5 minutes default
    
    # Extract timestamps and check for gaps
    local timestamps=($(echo "$data_points" | jq -r '.[].timestamp' | sort))
    local gap_found=false
    local max_gap_found=0
    
    for i in $(seq 1 $((${#timestamps[@]} - 1))); do
        local prev_ts=$(date -d "${timestamps[$((i-1))]}" +%s)
        local curr_ts=$(date -d "${timestamps[$i]}" +%s)
        local gap=$((curr_ts - prev_ts))
        
        if [[ $gap -gt $max_gap ]]; then
            gap_found=true
            if [[ $gap -gt $max_gap_found ]]; then
                max_gap_found=$gap
            fi
        fi
    done
    
    if [[ $gap_found == true ]]; then
        echo "Time-series data continuity gap detected:" >&2
        echo "  Maximum gap found: ${max_gap_found}s" >&2
        echo "  Maximum allowed gap: ${max_gap}s" >&2
        return 1
    fi
    
    return 0
}

#######################################
# Assert resource utilization efficiency
# Arguments: $1 - CPU usage %, $2 - Memory usage %, $3 - efficiency threshold %
#######################################
assert_resource_efficiency() {
    local cpu_usage="$1"
    local memory_usage="$2"
    local efficiency_threshold="${3:-80}"
    
    # Remove % sign if present
    cpu_usage="${cpu_usage%\%}"
    memory_usage="${memory_usage%\%}"
    
    # Calculate efficiency score (simplified)
    local efficiency_score=$(echo "scale=2; (100 - $cpu_usage) * 0.4 + (100 - $memory_usage) * 0.6" | bc)
    
    local meets_threshold=$(echo "$efficiency_score >= $efficiency_threshold" | bc -l)
    
    if [[ $meets_threshold -eq 0 ]]; then
        echo "Resource utilization efficiency below threshold:" >&2
        echo "  CPU usage: ${cpu_usage}%" >&2
        echo "  Memory usage: ${memory_usage}%" >&2
        echo "  Efficiency score: $efficiency_score" >&2
        echo "  Minimum required: $efficiency_threshold" >&2
        return 1
    fi
    
    return 0
}

# Export custom assertions
export -f assert_ai_response_quality
export -f assert_pipeline_throughput
export -f assert_timeseries_continuity
export -f assert_resource_efficiency
```

## Usage Examples for Custom Assertions

```bash
@test "custom assertions: AI response quality validation" {
    # Test AI response quality
    run curl -s -X POST "$OLLAMA_BASE_URL/api/generate" \
        -H "Content-Type: application/json" \
        -d '{"model":"llama3.1:8b","prompt":"Explain quantum computing","stream":false}'
    assert_success
    
    local ai_response=$(echo "$output" | jq -r '.response')
    
    # Use custom assertion to validate response quality
    assert_ai_response_quality "$ai_response" "0.8"
    
    echo "AI response quality validated"
}

@test "custom assertions: data pipeline performance validation" {
    # Measure pipeline throughput
    local start_time=$(date +%s)
    local processed_items=0
    
    # Simulate data processing
    for i in {1..100}; do
        redis-cli -h localhost -p "$REDIS_PORT" LPUSH "test_pipeline" "item_$i" >/dev/null
        processed_items=$((processed_items + 1))
    done
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    local throughput=$(echo "scale=2; $processed_items / $duration" | bc)
    
    # Use custom assertion to validate throughput
    assert_pipeline_throughput "$throughput" "50" "20"  # Expected 50 items/sec with 20% tolerance
    
    echo "Pipeline throughput validated: $throughput items/sec"
}
```

This advanced examples document provides comprehensive patterns for sophisticated testing scenarios. Each example demonstrates real-world testing challenges and solutions using the Vrooli BATS infrastructure.