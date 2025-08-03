#!/usr/bin/env bats
# Load Testing Template
# Comprehensive load testing for system stability and performance under stress
#
# Copy this template and customize it for your specific load testing requirements.
# Example: Testing system behavior under sustained load and traffic spikes

#######################################
# LOAD TEST CONFIGURATION
# Customize these variables for your load testing requirements
#######################################

# Target resources for load testing
LOAD_TEST_RESOURCES=("questdb" "redis" "ollama" "n8n")

# Load test scenarios
LOAD_SCENARIOS=(
    "baseline"          # Normal operational load
    "sustained"         # Extended high load
    "spike"            # Sudden traffic spikes
    "gradual_increase" # Gradually increasing load
    "mixed_workload"   # Different types of operations
)

# Load parameters
BASELINE_RPS=10        # Requests per second for baseline
SUSTAINED_RPS=50       # Requests per second for sustained load
SPIKE_RPS=200          # Requests per second during spikes
MAX_RPS=500            # Maximum requests per second

# Test duration parameters (in seconds)
BASELINE_DURATION=60   # 1 minute baseline
SUSTAINED_DURATION=300 # 5 minutes sustained load
SPIKE_DURATION=30      # 30 second spikes
GRADUAL_DURATION=240   # 4 minutes gradual increase

# Concurrent user simulation
MIN_USERS=5
NORMAL_USERS=25
HIGH_USERS=100
STRESS_USERS=250

# Performance thresholds under load
LOAD_FAST_THRESHOLD=200      # 200ms under load
LOAD_NORMAL_THRESHOLD=1000   # 1s under load
LOAD_SLOW_THRESHOLD=5000     # 5s maximum under load
ERROR_RATE_THRESHOLD=5       # 5% maximum error rate

#######################################
# SETUP AND TEARDOWN
#######################################

# Load the test infrastructure
if [[ -n "${VROOLI_TEST_FIXTURES_DIR:-}" ]]; then
    source "${VROOLI_TEST_FIXTURES_DIR}/core/common_setup.bash"
else
    TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    source "${TEST_DIR}/../../core/common_setup.bash"
fi

setup() {
    # Setup load testing environment
    setup_integration_test "${LOAD_TEST_RESOURCES[@]}"
    
    # Configure load testing environment
    export LOAD_TEST_ID="loadtest_$$"
    export LOAD_TEST_DATA_DIR="$TEST_TMPDIR/load_test_data"
    export LOAD_TEST_RESULTS_DIR="$TEST_TMPDIR/load_test_results"
    export LOAD_TEST_CONFIG_FILE="$TEST_TMPDIR/load_test_config.json"
    export LOAD_TEST_METRICS_FILE="$TEST_TMPDIR/load_metrics.log"
    
    # Create load testing directories
    mkdir -p "$LOAD_TEST_DATA_DIR"/{payloads,responses,errors}
    mkdir -p "$LOAD_TEST_RESULTS_DIR"/{baseline,sustained,spike,gradual,mixed}
    
    # Generate test payloads for different scenarios
    generate_load_test_payloads
    
    # Generate load test configuration
    cat > "$LOAD_TEST_CONFIG_FILE" << EOF
{
    "load_test_id": "$LOAD_TEST_ID",
    "timestamp": "$(date -Iseconds)",
    "resources": $(printf '%s\n' "${LOAD_TEST_RESOURCES[@]}" | jq -R . | jq -s .),
    "scenarios": $(printf '%s\n' "${LOAD_SCENARIOS[@]}" | jq -R . | jq -s .),
    "parameters": {
        "baseline_rps": $BASELINE_RPS,
        "sustained_rps": $SUSTAINED_RPS,
        "spike_rps": $SPIKE_RPS,
        "max_rps": $MAX_RPS,
        "durations": {
            "baseline_seconds": $BASELINE_DURATION,
            "sustained_seconds": $SUSTAINED_DURATION,
            "spike_seconds": $SPIKE_DURATION,
            "gradual_seconds": $GRADUAL_DURATION
        }
    },
    "thresholds": {
        "fast_ms": $LOAD_FAST_THRESHOLD,
        "normal_ms": $LOAD_NORMAL_THRESHOLD,
        "slow_ms": $LOAD_SLOW_THRESHOLD,
        "max_error_rate": $ERROR_RATE_THRESHOLD
    }
}
EOF

    # Initialize metrics logging
    echo "timestamp,scenario,rps,response_time_ms,success,error_type" > "$LOAD_TEST_METRICS_FILE"
    
    # Setup load test data in databases
    setup_load_test_data
    
    # Setup mock verification for load tests
    mock::verify::expect_min_calls "http" ".*" "$((BASELINE_RPS * BASELINE_DURATION / 10))"  # Adjusted for realistic expectations
    mock::verify::expect_min_calls "command" "redis-cli.*" "$((BASELINE_RPS * BASELINE_DURATION / 20))"
}

teardown() {
    # Generate load test summary report
    generate_load_test_summary
    
    # Validate load test expectations
    mock::verify::validate_all
    
    # Archive load test results
    if [[ -d "$LOAD_TEST_RESULTS_DIR" ]]; then
        tar -czf "${TEST_TMPDIR}/load_test_results_${LOAD_TEST_ID}.tar.gz" -C "$LOAD_TEST_RESULTS_DIR" .
        echo "Load test results archived: ${TEST_TMPDIR}/load_test_results_${LOAD_TEST_ID}.tar.gz"
    fi
    
    # Clean up load test data
    rm -rf "$LOAD_TEST_DATA_DIR" "$LOAD_TEST_RESULTS_DIR"
    rm -f "$LOAD_TEST_CONFIG_FILE" "$LOAD_TEST_METRICS_FILE"
    
    # Standard teardown
    teardown_test_environment
}

#######################################
# BASELINE LOAD TESTS
#######################################

@test "load: baseline performance measurement" {
    # Establish baseline performance metrics under normal load
    
    echo "Running baseline load test (${BASELINE_RPS} RPS for ${BASELINE_DURATION}s)..."
    local results_file="$LOAD_TEST_RESULTS_DIR/baseline/baseline_results.json"
    
    local test_start_time=$(date +%s)
    local total_requests=0
    local successful_requests=0
    local total_response_time=0
    local errors=()
    
    # Run baseline load for specified duration
    while [[ $(($(date +%s) - test_start_time)) -lt $BASELINE_DURATION ]]; do
        local batch_start=$(date +%s%N)
        
        # Execute batch of requests at target RPS
        for i in $(seq 1 "$BASELINE_RPS"); do
            local request_start=$(date +%s%N)
            local success=true
            local error_type=""
            
            # Rotate between different operations
            case $((i % 4)) in
                0)
                    if ! baseline_test_questdb_operation; then
                        success=false
                        error_type="questdb_error"
                    fi
                    ;;
                1)
                    if ! baseline_test_redis_operation; then
                        success=false
                        error_type="redis_error"
                    fi
                    ;;
                2)
                    if ! baseline_test_ollama_operation; then
                        success=false
                        error_type="ollama_error"
                    fi
                    ;;
                3)
                    if ! baseline_test_n8n_operation; then
                        success=false
                        error_type="n8n_error"
                    fi
                    ;;
            esac
            
            local request_end=$(date +%s%N)
            local response_time=$(( (request_end - request_start) / 1000000 ))
            
            total_requests=$((total_requests + 1))
            total_response_time=$((total_response_time + response_time))
            
            if [[ "$success" == "true" ]]; then
                successful_requests=$((successful_requests + 1))
            else
                errors+=("$error_type")
            fi
            
            # Log metrics
            echo "$(date -Iseconds),baseline,$BASELINE_RPS,$response_time,$success,$error_type" >> "$LOAD_TEST_METRICS_FILE"
        done
        
        # Wait for next second to maintain RPS
        local batch_end=$(date +%s%N)
        local batch_duration=$(( (batch_end - batch_start) / 1000000 ))
        local sleep_time=$((1000 - batch_duration))
        
        if [[ $sleep_time -gt 0 ]]; then
            sleep "0.$(printf "%03d" $sleep_time)"
        fi
    done
    
    # Calculate baseline metrics
    local avg_response_time=$((total_response_time / total_requests))
    local success_rate=$(echo "scale=2; $successful_requests * 100 / $total_requests" | bc)
    local error_rate=$(echo "scale=2; (100 - $success_rate)" | bc)
    local actual_rps=$(echo "scale=2; $total_requests / $BASELINE_DURATION" | bc)
    
    # Generate baseline results
    cat > "$results_file" << EOF
{
    "scenario": "baseline",
    "duration_seconds": $BASELINE_DURATION,
    "target_rps": $BASELINE_RPS,
    "actual_rps": $actual_rps,
    "total_requests": $total_requests,
    "successful_requests": $successful_requests,
    "avg_response_time_ms": $avg_response_time,
    "success_rate": $success_rate,
    "error_rate": $error_rate,
    "errors": $(printf '%s\n' "${errors[@]}" | sort | uniq -c | jq -R 'split(" ") | {count: .[0], type: .[1]}' | jq -s '.')
}
EOF

    # Validate baseline performance
    assert_less_than "$avg_response_time" "$LOAD_NORMAL_THRESHOLD"
    assert_greater_than "$success_rate" "$((100 - ERROR_RATE_THRESHOLD))"
    assert_greater_than "$actual_rps" "$((BASELINE_RPS * 8 / 10))"  # Within 80% of target
    
    echo "Baseline completed: ${actual_rps} RPS, ${avg_response_time}ms avg, ${success_rate}% success"
}

@test "load: sustained high load test" {
    # Test system stability under sustained high load
    
    echo "Running sustained load test (${SUSTAINED_RPS} RPS for ${SUSTAINED_DURATION}s)..."
    local results_file="$LOAD_TEST_RESULTS_DIR/sustained/sustained_results.json"
    
    local test_start_time=$(date +%s)
    local total_requests=0
    local successful_requests=0
    local response_times=()
    local errors=()
    local performance_degraded=false
    
    # Track performance degradation over time
    local checkpoint_interval=60  # Check every minute
    local last_checkpoint=$test_start_time
    local checkpoint_metrics=()
    
    while [[ $(($(date +%s) - test_start_time)) -lt $SUSTAINED_DURATION ]]; do
        local batch_start=$(date +%s%N)
        local batch_successful=0
        local batch_total=0
        local batch_response_time=0
        
        # Execute batch of requests at sustained RPS
        for i in $(seq 1 "$SUSTAINED_RPS"); do
            local request_start=$(date +%s%N)
            local success=true
            local error_type=""
            
            # Use mixed workload for sustained testing
            if ! sustained_mixed_operation "$((i % 6))"; then
                success=false
                error_type="mixed_operation_error"
            fi
            
            local request_end=$(date +%s%N)
            local response_time=$(( (request_end - request_start) / 1000000 ))
            
            total_requests=$((total_requests + 1))
            batch_total=$((batch_total + 1))
            batch_response_time=$((batch_response_time + response_time))
            response_times+=("$response_time")
            
            if [[ "$success" == "true" ]]; then
                successful_requests=$((successful_requests + 1))
                batch_successful=$((batch_successful + 1))
            else
                errors+=("$error_type")
            fi
            
            # Check for performance degradation
            if [[ $response_time -gt $LOAD_SLOW_THRESHOLD ]]; then
                performance_degraded=true
            fi
        done
        
        # Record checkpoint metrics
        local current_time=$(date +%s)
        if [[ $((current_time - last_checkpoint)) -ge $checkpoint_interval ]]; then
            local checkpoint_avg=$((batch_response_time / batch_total))
            local checkpoint_success_rate=$(echo "scale=2; $batch_successful * 100 / $batch_total" | bc)
            checkpoint_metrics+=("$(date -Iseconds):${checkpoint_avg}ms:${checkpoint_success_rate}%")
            last_checkpoint=$current_time
        fi
        
        # Maintain RPS timing
        local batch_end=$(date +%s%N)
        local batch_duration=$(( (batch_end - batch_start) / 1000000 ))
        local sleep_time=$((1000 - batch_duration))
        
        if [[ $sleep_time -gt 0 ]]; then
            sleep "0.$(printf "%03d" $sleep_time)"
        fi
    done
    
    # Calculate sustained load metrics
    local avg_response_time=$(echo "${response_times[@]}" | tr ' ' '\n' | awk '{sum+=$1} END {print int(sum/NR)}')
    local success_rate=$(echo "scale=2; $successful_requests * 100 / $total_requests" | bc)
    local actual_rps=$(echo "scale=2; $total_requests / $SUSTAINED_DURATION" | bc)
    
    # Calculate percentiles
    local sorted_times=($(printf '%s\n' "${response_times[@]}" | sort -n))
    local p95_index=$(( ${#sorted_times[@]} * 95 / 100 ))
    local p99_index=$(( ${#sorted_times[@]} * 99 / 100 ))
    local p95_time="${sorted_times[$p95_index]}"
    local p99_time="${sorted_times[$p99_index]}"
    
    # Generate sustained load results
    cat > "$results_file" << EOF
{
    "scenario": "sustained",
    "duration_seconds": $SUSTAINED_DURATION,
    "target_rps": $SUSTAINED_RPS,
    "actual_rps": $actual_rps,
    "total_requests": $total_requests,
    "successful_requests": $successful_requests,
    "avg_response_time_ms": $avg_response_time,
    "p95_response_time_ms": $p95_time,
    "p99_response_time_ms": $p99_time,
    "success_rate": $success_rate,
    "performance_degraded": $performance_degraded,
    "checkpoint_metrics": $(printf '%s\n' "${checkpoint_metrics[@]}" | jq -R . | jq -s '.')
}
EOF

    # Validate sustained load performance
    assert_less_than "$avg_response_time" "$LOAD_SLOW_THRESHOLD"
    assert_greater_than "$success_rate" "$((100 - ERROR_RATE_THRESHOLD * 2))"  # Allow higher error rate under sustained load
    assert_equals "$performance_degraded" "false"
    
    echo "Sustained load completed: ${actual_rps} RPS, ${avg_response_time}ms avg, ${success_rate}% success"
    echo "Performance percentiles: P95=${p95_time}ms, P99=${p99_time}ms"
}

#######################################
# SPIKE LOAD TESTS
#######################################

@test "load: traffic spike handling" {
    # Test system response to sudden traffic spikes
    
    echo "Running traffic spike test (${SPIKE_RPS} RPS spikes)..."
    local results_file="$LOAD_TEST_RESULTS_DIR/spike/spike_results.json"
    
    local spike_results=()
    local num_spikes=3
    local spike_interval=60  # 1 minute between spikes
    
    for spike_num in $(seq 1 $num_spikes); do
        echo "Executing spike $spike_num of $num_spikes..."
        
        # Baseline period before spike
        local baseline_start=$(date +%s)
        local baseline_requests=0
        local baseline_successful=0
        
        for i in $(seq 1 $((BASELINE_RPS * 10))); do  # 10 seconds of baseline
            if baseline_test_redis_operation; then
                baseline_successful=$((baseline_successful + 1))
            fi
            baseline_requests=$((baseline_requests + 1))
            sleep 0.1
        done
        
        # Sudden spike
        local spike_start=$(date +%s%N)
        local spike_requests=0
        local spike_successful=0
        local spike_response_times=()
        
        for i in $(seq 1 $((SPIKE_RPS * SPIKE_DURATION))); do
            local request_start=$(date +%s%N)
            local success=true
            
            # Spike operations (higher load)
            if ! spike_test_operation "$((i % 3))"; then
                success=false
            fi
            
            local request_end=$(date +%s%N)
            local response_time=$(( (request_end - request_start) / 1000000 ))
            
            spike_requests=$((spike_requests + 1))
            spike_response_times+=("$response_time")
            
            if [[ "$success" == "true" ]]; then
                spike_successful=$((spike_successful + 1))
            fi
        done
        
        local spike_end=$(date +%s%N)
        local spike_duration_ms=$(( (spike_end - spike_start) / 1000000 ))
        local spike_avg_response=$(echo "${spike_response_times[@]}" | tr ' ' '\n' | awk '{sum+=$1} END {print int(sum/NR)}')
        local spike_success_rate=$(echo "scale=2; $spike_successful * 100 / $spike_requests" | bc)
        
        spike_results+=("spike_${spike_num}:${spike_avg_response}:${spike_success_rate}")
        
        # Recovery period
        sleep $spike_interval
    done
    
    # Analyze spike performance
    local avg_spike_response=0
    local avg_spike_success=0
    local spike_count=0
    
    for result in "${spike_results[@]}"; do
        local response_time=$(echo "$result" | cut -d':' -f2)
        local success_rate=$(echo "$result" | cut -d':' -f3)
        avg_spike_response=$((avg_spike_response + response_time))
        avg_spike_success=$(echo "scale=2; $avg_spike_success + $success_rate" | bc)
        spike_count=$((spike_count + 1))
    done
    
    avg_spike_response=$((avg_spike_response / spike_count))
    avg_spike_success=$(echo "scale=2; $avg_spike_success / $spike_count" | bc)
    
    # Generate spike test results
    cat > "$results_file" << EOF
{
    "scenario": "traffic_spike",
    "num_spikes": $num_spikes,
    "spike_rps": $SPIKE_RPS,
    "spike_duration_seconds": $SPIKE_DURATION,
    "avg_spike_response_time_ms": $avg_spike_response,
    "avg_spike_success_rate": $avg_spike_success,
    "spike_details": $(printf '%s\n' "${spike_results[@]}" | jq -R 'split(":") | {spike: .[0], response_time_ms: (.[1] | tonumber), success_rate: (.[2] | tonumber)}' | jq -s '.')
}
EOF

    # Validate spike handling
    assert_less_than "$avg_spike_response" "$((LOAD_SLOW_THRESHOLD * 2))"  # Allow degraded performance during spikes
    assert_greater_than "$avg_spike_success" "$((100 - ERROR_RATE_THRESHOLD * 3))"  # Allow higher error rate during spikes
    
    echo "Spike testing completed: ${avg_spike_response}ms avg response, ${avg_spike_success}% avg success"
}

@test "load: gradual load increase test" {
    # Test system behavior under gradually increasing load
    
    echo "Running gradual load increase test (${MIN_USERS} to ${HIGH_USERS} users over ${GRADUAL_DURATION}s)..."
    local results_file="$LOAD_TEST_RESULTS_DIR/gradual/gradual_results.json"
    
    local gradual_results=()
    local step_duration=$((GRADUAL_DURATION / 10))  # 10 steps
    local user_increment=$(( (HIGH_USERS - MIN_USERS) / 10 ))
    
    for step in $(seq 1 10); do
        local current_users=$((MIN_USERS + (step - 1) * user_increment))
        local step_start=$(date +%s)
        local step_requests=0
        local step_successful=0
        local step_response_times=()
        
        echo "Step $step: $current_users concurrent users"
        
        # Simulate concurrent users
        local pids=()
        for user in $(seq 1 "$current_users"); do
            (
                local user_requests=$((step_duration / 2))  # Each user makes requests every 2 seconds
                local user_successful=0
                
                for req in $(seq 1 "$user_requests"); do
                    local request_start=$(date +%s%N)
                    
                    if gradual_test_operation "$((user % 4))"; then
                        user_successful=$((user_successful + 1))
                    fi
                    
                    local request_end=$(date +%s%N)
                    local response_time=$(( (request_end - request_start) / 1000000 ))
                    
                    echo "$response_time" > "$LOAD_TEST_DATA_DIR/responses/step_${step}_user_${user}_req_${req}.tmp"
                    
                    sleep 2
                done
                
                echo "$user_successful" > "$LOAD_TEST_DATA_DIR/responses/step_${step}_user_${user}_success.tmp"
            ) &
            pids+=($!)
        done
        
        # Wait for step completion
        sleep "$step_duration"
        
        # Kill any remaining processes
        for pid in "${pids[@]}"; do
            kill "$pid" 2>/dev/null || true
        done
        wait 2>/dev/null || true
        
        # Collect step results
        for user in $(seq 1 "$current_users"); do
            local user_success_file="$LOAD_TEST_DATA_DIR/responses/step_${step}_user_${user}_success.tmp"
            if [[ -f "$user_success_file" ]]; then
                local user_successful=$(cat "$user_success_file")
                step_successful=$((step_successful + user_successful))
                rm -f "$user_success_file"
            fi
            
            for req_file in "$LOAD_TEST_DATA_DIR/responses/step_${step}_user_${user}_req_"*.tmp; do
                if [[ -f "$req_file" ]]; then
                    local response_time=$(cat "$req_file")
                    step_response_times+=("$response_time")
                    step_requests=$((step_requests + 1))
                    rm -f "$req_file"
                fi
            done
        done
        
        # Calculate step metrics
        local step_avg_response=$(echo "${step_response_times[@]}" | tr ' ' '\n' | awk '{sum+=$1} END {print int(sum/NR)}' 2>/dev/null || echo "0")
        local step_success_rate=$(echo "scale=2; $step_successful * 100 / $step_requests" | bc 2>/dev/null || echo "0")
        
        gradual_results+=("step_${step}:${current_users}:${step_avg_response}:${step_success_rate}")
        
        echo "Step $step completed: $current_users users, ${step_avg_response}ms avg, ${step_success_rate}% success"
    done
    
    # Generate gradual load results
    cat > "$results_file" << EOF
{
    "scenario": "gradual_increase",
    "duration_seconds": $GRADUAL_DURATION,
    "min_users": $MIN_USERS,
    "max_users": $HIGH_USERS,
    "steps": $(printf '%s\n' "${gradual_results[@]}" | jq -R 'split(":") | {step: .[0], users: (.[1] | tonumber), avg_response_ms: (.[2] | tonumber), success_rate: (.[3] | tonumber)}' | jq -s '.')
}
EOF

    # Validate gradual load handling
    local final_step_data="${gradual_results[-1]}"
    local final_response=$(echo "$final_step_data" | cut -d':' -f3)
    local final_success=$(echo "$final_step_data" | cut -d':' -f4)
    
    assert_less_than "$final_response" "$LOAD_SLOW_THRESHOLD"
    assert_greater_than "$final_success" "$((100 - ERROR_RATE_THRESHOLD * 2))"
    
    echo "Gradual load increase completed successfully"
}

#######################################
# MIXED WORKLOAD TESTS
#######################################

@test "load: mixed workload stress test" {
    # Test system with mixed types of operations under load
    
    echo "Running mixed workload test..."
    local results_file="$LOAD_TEST_RESULTS_DIR/mixed/mixed_workload_results.json"
    
    local workload_types=("read_heavy" "write_heavy" "compute_heavy" "mixed_operations")
    local workload_results=()
    
    for workload in "${workload_types[@]}"; do
        echo "Testing workload: $workload"
        
        local workload_start=$(date +%s)
        local workload_requests=0
        local workload_successful=0
        local workload_response_times=()
        local test_duration=60  # 1 minute per workload type
        
        while [[ $(($(date +%s) - workload_start)) -lt $test_duration ]]; do
            local batch_start=$(date +%s%N)
            
            for i in $(seq 1 20); do  # 20 RPS for mixed workload
                local request_start=$(date +%s%N)
                local success=true
                
                case "$workload" in
                    "read_heavy")
                        if ! mixed_workload_read_operation "$((i % 3))"; then
                            success=false
                        fi
                        ;;
                    "write_heavy")
                        if ! mixed_workload_write_operation "$((i % 3))"; then
                            success=false
                        fi
                        ;;
                    "compute_heavy")
                        if ! mixed_workload_compute_operation "$((i % 2))"; then
                            success=false
                        fi
                        ;;
                    "mixed_operations")
                        if ! mixed_workload_random_operation "$((i % 6))"; then
                            success=false
                        fi
                        ;;
                esac
                
                local request_end=$(date +%s%N)
                local response_time=$(( (request_end - request_start) / 1000000 ))
                
                workload_requests=$((workload_requests + 1))
                workload_response_times+=("$response_time")
                
                if [[ "$success" == "true" ]]; then
                    workload_successful=$((workload_successful + 1))
                fi
            done
            
            # Maintain timing
            local batch_end=$(date +%s%N)
            local batch_duration=$(( (batch_end - batch_start) / 1000000 ))
            local sleep_time=$((1000 - batch_duration))
            
            if [[ $sleep_time -gt 0 ]]; then
                sleep "0.$(printf "%03d" $sleep_time)"
            fi
        done
        
        # Calculate workload metrics
        local workload_avg_response=$(echo "${workload_response_times[@]}" | tr ' ' '\n' | awk '{sum+=$1} END {print int(sum/NR)}')
        local workload_success_rate=$(echo "scale=2; $workload_successful * 100 / $workload_requests" | bc)
        
        workload_results+=("${workload}:${workload_avg_response}:${workload_success_rate}")
        
        echo "Workload $workload completed: ${workload_avg_response}ms avg, ${workload_success_rate}% success"
    done
    
    # Generate mixed workload results
    cat > "$results_file" << EOF
{
    "scenario": "mixed_workload",
    "workload_types": $(printf '%s\n' "${workload_types[@]}" | jq -R . | jq -s '.'),
    "workload_results": $(printf '%s\n' "${workload_results[@]}" | jq -R 'split(":") | {workload: .[0], avg_response_ms: (.[1] | tonumber), success_rate: (.[2] | tonumber)}' | jq -s '.')
}
EOF

    # Validate mixed workload performance
    for result in "${workload_results[@]}"; do
        local response_time=$(echo "$result" | cut -d':' -f2)
        local success_rate=$(echo "$result" | cut -d':' -f3)
        
        assert_less_than "$response_time" "$LOAD_SLOW_THRESHOLD"
        assert_greater_than "$success_rate" "$((100 - ERROR_RATE_THRESHOLD * 2))"
    done
    
    echo "Mixed workload testing completed successfully"
}

#######################################
# LOAD TEST HELPER FUNCTIONS
#######################################

# Helper: Generate load test payloads
generate_load_test_payloads() {
    # Generate different payload sizes for testing
    echo '{"operation":"get","key":"test_key"}' > "$LOAD_TEST_DATA_DIR/payloads/small_payload.json"
    
    # Medium payload
    local medium_data=$(head -c 1024 < /dev/urandom | base64 | tr -d '\n')
    echo '{"operation":"set","key":"medium_key","data":"'"$medium_data"'"}' > "$LOAD_TEST_DATA_DIR/payloads/medium_payload.json"
    
    # Large payload
    local large_data=$(head -c 10240 < /dev/urandom | base64 | tr -d '\n')
    echo '{"operation":"bulk_insert","data":"'"$large_data"'"}' > "$LOAD_TEST_DATA_DIR/payloads/large_payload.json"
}

# Helper: Setup load test data
setup_load_test_data() {
    # Pre-populate databases with test data for load testing
    
    # Setup Redis test data
    for i in $(seq 1 100); do
        redis-cli -h localhost -p "$REDIS_PORT" SET "load_test_key_$i" "value_$i" >/dev/null
    done
    
    # Setup QuestDB test tables
    curl -s -G "$QUESTDB_BASE_URL/exec" \
        --data-urlencode "query=CREATE TABLE IF NOT EXISTS load_test_metrics (timestamp TIMESTAMP, metric_name STRING, value DOUBLE) timestamp(timestamp)" >/dev/null
    
    # Insert some initial data
    for i in $(seq 1 50); do
        curl -s -G "$QUESTDB_BASE_URL/exec" \
            --data-urlencode "query=INSERT INTO load_test_metrics VALUES (now(), 'test_metric_$i', $i)" >/dev/null
    done
}

# Helper: Baseline test operations
baseline_test_questdb_operation() {
    curl -s -G "$QUESTDB_BASE_URL/exec" \
        --data-urlencode "query=SELECT COUNT(*) FROM load_test_metrics" \
        --data-urlencode "fmt=json" >/dev/null
}

baseline_test_redis_operation() {
    redis-cli -h localhost -p "$REDIS_PORT" GET "load_test_key_$((RANDOM % 100 + 1))" >/dev/null
}

baseline_test_ollama_operation() {
    curl -s "$OLLAMA_BASE_URL/api/tags" >/dev/null
}

baseline_test_n8n_operation() {
    curl -s "$N8N_BASE_URL/healthz" >/dev/null
}

# Helper: Sustained load mixed operations
sustained_mixed_operation() {
    local operation_type="$1"
    
    case "$operation_type" in
        0|1)
            baseline_test_redis_operation
            ;;
        2|3)
            baseline_test_questdb_operation
            ;;
        4)
            baseline_test_ollama_operation
            ;;
        5)
            baseline_test_n8n_operation
            ;;
    esac
}

# Helper: Spike test operations (more intensive)
spike_test_operation() {
    local operation_type="$1"
    
    case "$operation_type" in
        0)
            # Intensive Redis operation
            redis-cli -h localhost -p "$REDIS_PORT" LPUSH "spike_list" "item_$(date +%s%N)" >/dev/null
            ;;
        1)
            # Complex QuestDB query
            curl -s -G "$QUESTDB_BASE_URL/exec" \
                --data-urlencode "query=SELECT metric_name, AVG(value) FROM load_test_metrics GROUP BY metric_name" \
                --data-urlencode "fmt=json" >/dev/null
            ;;
        2)
            # Ollama generation request
            curl -s -X POST "$OLLAMA_BASE_URL/api/generate" \
                -H "Content-Type: application/json" \
                -d '{"model":"llama3.1:8b","prompt":"Test prompt","stream":false}' >/dev/null
            ;;
    esac
}

# Helper: Gradual load test operation
gradual_test_operation() {
    local operation_type="$1"
    
    case "$operation_type" in
        0)
            redis-cli -h localhost -p "$REDIS_PORT" PING >/dev/null
            ;;
        1)
            curl -s -G "$QUESTDB_BASE_URL/exec" \
                --data-urlencode "query=SELECT 1" >/dev/null
            ;;
        2)
            curl -s "$OLLAMA_BASE_URL/api/tags" >/dev/null
            ;;
        3)
            curl -s "$N8N_BASE_URL/healthz" >/dev/null
            ;;
    esac
}

# Helper: Mixed workload operations
mixed_workload_read_operation() {
    local operation_type="$1"
    
    case "$operation_type" in
        0)
            redis-cli -h localhost -p "$REDIS_PORT" GET "load_test_key_$((RANDOM % 100 + 1))" >/dev/null
            ;;
        1)
            curl -s -G "$QUESTDB_BASE_URL/exec" \
                --data-urlencode "query=SELECT * FROM load_test_metrics LIMIT 10" >/dev/null
            ;;
        2)
            curl -s "$OLLAMA_BASE_URL/api/tags" >/dev/null
            ;;
    esac
}

mixed_workload_write_operation() {
    local operation_type="$1"
    
    case "$operation_type" in
        0)
            redis-cli -h localhost -p "$REDIS_PORT" SET "mixed_key_$(date +%s)" "value_$(date +%s)" >/dev/null
            ;;
        1)
            curl -s -G "$QUESTDB_BASE_URL/exec" \
                --data-urlencode "query=INSERT INTO load_test_metrics VALUES (now(), 'mixed_metric', $((RANDOM % 100)))" >/dev/null
            ;;
        2)
            curl -s -X POST "$N8N_BASE_URL/api/v1/workflows/execute" \
                -H "Content-Type: application/json" \
                -d '{"workflowId":"test","data":{}}' >/dev/null
            ;;
    esac
}

mixed_workload_compute_operation() {
    local operation_type="$1"
    
    case "$operation_type" in
        0)
            curl -s -G "$QUESTDB_BASE_URL/exec" \
                --data-urlencode "query=SELECT metric_name, COUNT(*), AVG(value), MAX(value) FROM load_test_metrics GROUP BY metric_name" >/dev/null
            ;;
        1)
            curl -s -X POST "$OLLAMA_BASE_URL/api/generate" \
                -H "Content-Type: application/json" \
                -d '{"model":"llama3.1:8b","prompt":"Calculate: 123 * 456","stream":false}' >/dev/null
            ;;
    esac
}

mixed_workload_random_operation() {
    local operation_type="$1"
    
    case "$operation_type" in
        0|1)
            mixed_workload_read_operation "$((operation_type % 3))"
            ;;
        2|3)
            mixed_workload_write_operation "$((operation_type % 3))"
            ;;
        4|5)
            mixed_workload_compute_operation "$((operation_type % 2))"
            ;;
    esac
}

# Helper: Generate load test summary
generate_load_test_summary() {
    local summary_file="$LOAD_TEST_RESULTS_DIR/load_test_summary.json"
    
    # Collect all result files
    local baseline_files=($(find "$LOAD_TEST_RESULTS_DIR/baseline" -name "*.json" 2>/dev/null || true))
    local sustained_files=($(find "$LOAD_TEST_RESULTS_DIR/sustained" -name "*.json" 2>/dev/null || true))
    local spike_files=($(find "$LOAD_TEST_RESULTS_DIR/spike" -name "*.json" 2>/dev/null || true))
    local gradual_files=($(find "$LOAD_TEST_RESULTS_DIR/gradual" -name "*.json" 2>/dev/null || true))
    local mixed_files=($(find "$LOAD_TEST_RESULTS_DIR/mixed" -name "*.json" 2>/dev/null || true))
    
    # Generate summary
    cat > "$summary_file" << EOF
{
    "load_test_id": "$LOAD_TEST_ID",
    "timestamp": "$(date -Iseconds)",
    "summary": {
        "baseline_tests": ${#baseline_files[@]},
        "sustained_tests": ${#sustained_files[@]},
        "spike_tests": ${#spike_files[@]},
        "gradual_tests": ${#gradual_files[@]},
        "mixed_workload_tests": ${#mixed_files[@]}
    },
    "thresholds": {
        "fast_ms": $LOAD_FAST_THRESHOLD,
        "normal_ms": $LOAD_NORMAL_THRESHOLD,
        "slow_ms": $LOAD_SLOW_THRESHOLD,
        "max_error_rate": $ERROR_RATE_THRESHOLD
    }
}
EOF

    echo "Load test summary generated: $summary_file"
}