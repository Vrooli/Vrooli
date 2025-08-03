#!/usr/bin/env bats
# Resource Performance Benchmark Test Template
# Comprehensive performance testing for individual and combined resources
#
# Copy this template and customize it for your specific performance requirements.
# Example: Testing resource response times, throughput, and scalability

#######################################
# BENCHMARK CONFIGURATION
# Customize these variables for your performance requirements
#######################################

# Resources to benchmark
BENCHMARK_RESOURCES=("ollama" "questdb" "redis" "whisper")

# Performance thresholds (in milliseconds unless specified)
FAST_THRESHOLD=100          # Operations that should be very fast
NORMAL_THRESHOLD=1000       # Acceptable response time
SLOW_THRESHOLD=5000         # Maximum acceptable time
TIMEOUT_THRESHOLD=30000     # Absolute timeout

# Load testing parameters
LIGHT_LOAD_REQUESTS=10
MEDIUM_LOAD_REQUESTS=50
HEAVY_LOAD_REQUESTS=100
STRESS_LOAD_REQUESTS=500

# Concurrent testing parameters
LOW_CONCURRENCY=2
MEDIUM_CONCURRENCY=10
HIGH_CONCURRENCY=25

# Data size parameters
SMALL_DATA_SIZE=1024        # 1KB
MEDIUM_DATA_SIZE=102400     # 100KB  
LARGE_DATA_SIZE=1048576     # 1MB

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
    # Setup performance test environment
    setup_integration_test "${BENCHMARK_RESOURCES[@]}"
    
    # Configure performance testing environment
    export BENCHMARK_ID="benchmark_$$"
    export BENCHMARK_DATA_DIR="$TEST_TMPDIR/benchmark_data"
    export BENCHMARK_RESULTS_DIR="$TEST_TMPDIR/benchmark_results"
    export BENCHMARK_CONFIG_FILE="$TEST_TMPDIR/benchmark_config.json"
    
    # Create benchmark directories
    mkdir -p "$BENCHMARK_DATA_DIR"/{small,medium,large}
    mkdir -p "$BENCHMARK_RESULTS_DIR"/{response_times,throughput,concurrency}
    
    # Generate test data files of different sizes
    generate_test_data_file "$BENCHMARK_DATA_DIR/small/test_data.txt" "$SMALL_DATA_SIZE"
    generate_test_data_file "$BENCHMARK_DATA_DIR/medium/test_data.txt" "$MEDIUM_DATA_SIZE"
    generate_test_data_file "$BENCHMARK_DATA_DIR/large/test_data.txt" "$LARGE_DATA_SIZE"
    
    # Generate benchmark configuration
    cat > "$BENCHMARK_CONFIG_FILE" << EOF
{
    "benchmark_id": "$BENCHMARK_ID",
    "timestamp": "$(date -Iseconds)",
    "resources": $(printf '%s\n' "${BENCHMARK_RESOURCES[@]}" | jq -R . | jq -s .),
    "thresholds": {
        "fast_ms": $FAST_THRESHOLD,
        "normal_ms": $NORMAL_THRESHOLD,
        "slow_ms": $SLOW_THRESHOLD,
        "timeout_ms": $TIMEOUT_THRESHOLD
    },
    "load_levels": {
        "light": $LIGHT_LOAD_REQUESTS,
        "medium": $MEDIUM_LOAD_REQUESTS,
        "heavy": $HEAVY_LOAD_REQUESTS,
        "stress": $STRESS_LOAD_REQUESTS
    }
}
EOF

    # Setup mock verification for performance tests
    mock::verify::expect_min_calls "http" ".*" "$LIGHT_LOAD_REQUESTS"
    mock::verify::expect_max_calls "docker" ".*" "10"  # Reasonable Docker call limit
}

teardown() {
    # Generate benchmark summary report
    generate_benchmark_summary
    
    # Validate performance expectations
    mock::verify::validate_all
    
    # Archive benchmark results
    if [[ -d "$BENCHMARK_RESULTS_DIR" ]]; then
        tar -czf "${TEST_TMPDIR}/benchmark_results_${BENCHMARK_ID}.tar.gz" -C "$BENCHMARK_RESULTS_DIR" .
        echo "Benchmark results archived: ${TEST_TMPDIR}/benchmark_results_${BENCHMARK_ID}.tar.gz"
    fi
    
    # Clean up benchmark data
    rm -rf "$BENCHMARK_DATA_DIR" "$BENCHMARK_RESULTS_DIR"
    rm -f "$BENCHMARK_CONFIG_FILE"
    
    # Standard teardown
    teardown_test_environment
}

#######################################
# RESPONSE TIME BENCHMARKS
#######################################

@test "performance: ollama response time benchmarks" {
    # Test Ollama API response times under different conditions
    
    echo "Testing Ollama response times..."
    local results_file="$BENCHMARK_RESULTS_DIR/response_times/ollama_response_times.json"
    local test_results=()
    
    # Health check response time
    local health_time=$(benchmark_single_request "GET" "$OLLAMA_BASE_URL/api/tags" "")
    test_results+=("health:$health_time")
    assert_less_than "$health_time" "$FAST_THRESHOLD"
    
    # Model query response time  
    local model_time=$(benchmark_single_request "GET" "$OLLAMA_BASE_URL/api/tags" "")
    test_results+=("models:$model_time")
    assert_less_than "$model_time" "$NORMAL_THRESHOLD"
    
    # Text generation response time
    local generation_data='{"model":"llama3.1:8b","prompt":"Hello","stream":false}'
    local generation_time=$(benchmark_single_request "POST" "$OLLAMA_BASE_URL/api/generate" "$generation_data")
    test_results+=("generation:$generation_time")
    assert_less_than "$generation_time" "$SLOW_THRESHOLD"
    
    # Save results
    printf '%s\n' "${test_results[@]}" | jq -R 'split(":") | {operation: .[0], response_time_ms: (.[1] | tonumber)}' | jq -s '.' > "$results_file"
    
    echo "Ollama benchmarks - Health: ${health_time}ms, Models: ${model_time}ms, Generation: ${generation_time}ms"
}

@test "performance: questdb response time benchmarks" {
    # Test QuestDB query performance across different query types
    
    echo "Testing QuestDB response times..."
    local results_file="$BENCHMARK_RESULTS_DIR/response_times/questdb_response_times.json"
    local test_results=()
    
    # Status check response time
    local status_time=$(benchmark_single_request "GET" "$QUESTDB_BASE_URL/status" "")
    test_results+=("status:$status_time")
    assert_less_than "$status_time" "$FAST_THRESHOLD"
    
    # Simple SELECT response time
    local simple_select_time=$(benchmark_questdb_query "SELECT 1")
    test_results+=("simple_select:$simple_select_time")
    assert_less_than "$simple_select_time" "$FAST_THRESHOLD"
    
    # Table listing response time
    local tables_time=$(benchmark_questdb_query "SELECT table_name FROM tables()")
    test_results+=("list_tables:$tables_time")
    assert_less_than "$tables_time" "$NORMAL_THRESHOLD"
    
    # Complex query response time (if data exists)
    local complex_time=$(benchmark_questdb_query "SELECT COUNT(*) FROM information_schema.tables")
    test_results+=("complex_query:$complex_time")
    assert_less_than "$complex_time" "$NORMAL_THRESHOLD"
    
    # Save results
    printf '%s\n' "${test_results[@]}" | jq -R 'split(":") | {operation: .[0], response_time_ms: (.[1] | tonumber)}' | jq -s '.' > "$results_file"
    
    echo "QuestDB benchmarks - Status: ${status_time}ms, Simple: ${simple_select_time}ms, Complex: ${complex_time}ms"
}

@test "performance: redis response time benchmarks" {
    # Test Redis operation performance across different command types
    
    echo "Testing Redis response times..."
    local results_file="$BENCHMARK_RESULTS_DIR/response_times/redis_response_times.json"
    local test_results=()
    
    # PING response time
    local ping_time=$(benchmark_redis_command "PING")
    test_results+=("ping:$ping_time")
    assert_less_than "$ping_time" "$FAST_THRESHOLD"
    
    # SET/GET response time
    local set_time=$(benchmark_redis_command "SET test_key test_value")
    test_results+=("set:$set_time")
    assert_less_than "$set_time" "$FAST_THRESHOLD"
    
    local get_time=$(benchmark_redis_command "GET test_key")
    test_results+=("get:$get_time")
    assert_less_than "$get_time" "$FAST_THRESHOLD"
    
    # List operations response time
    local lpush_time=$(benchmark_redis_command "LPUSH test_list item1 item2 item3")
    test_results+=("lpush:$lpush_time")
    assert_less_than "$lpush_time" "$FAST_THRESHOLD"
    
    local lrange_time=$(benchmark_redis_command "LRANGE test_list 0 -1")
    test_results+=("lrange:$lrange_time")
    assert_less_than "$lrange_time" "$FAST_THRESHOLD"
    
    # Save results
    printf '%s\n' "${test_results[@]}" | jq -R 'split(":") | {operation: .[0], response_time_ms: (.[1] | tonumber)}' | jq -s '.' > "$results_file"
    
    echo "Redis benchmarks - PING: ${ping_time}ms, SET: ${set_time}ms, GET: ${get_time}ms"
}

@test "performance: whisper response time benchmarks" {
    # Test Whisper API response times for different audio processing tasks
    
    echo "Testing Whisper response times..."
    local results_file="$BENCHMARK_RESULTS_DIR/response_times/whisper_response_times.json"
    local test_results=()
    
    # Health check response time
    local health_time=$(benchmark_single_request "GET" "$WHISPER_BASE_URL/health" "")
    test_results+=("health:$health_time")
    assert_less_than "$health_time" "$FAST_THRESHOLD"
    
    # Small audio transcription response time
    local small_audio_file="$BENCHMARK_DATA_DIR/small/test_audio.wav"
    echo "mock small audio data" > "$small_audio_file"
    local small_transcription_time=$(benchmark_whisper_transcription "$small_audio_file")
    test_results+=("small_transcription:$small_transcription_time")
    assert_less_than "$small_transcription_time" "$NORMAL_THRESHOLD"
    
    # Medium audio transcription response time
    local medium_audio_file="$BENCHMARK_DATA_DIR/medium/test_audio.wav"
    generate_test_data_file "$medium_audio_file" "$MEDIUM_DATA_SIZE"
    local medium_transcription_time=$(benchmark_whisper_transcription "$medium_audio_file")
    test_results+=("medium_transcription:$medium_transcription_time")
    assert_less_than "$medium_transcription_time" "$SLOW_THRESHOLD"
    
    # Save results
    printf '%s\n' "${test_results[@]}" | jq -R 'split(":") | {operation: .[0], response_time_ms: (.[1] | tonumber)}' | jq -s '.' > "$results_file"
    
    echo "Whisper benchmarks - Health: ${health_time}ms, Small: ${small_transcription_time}ms, Medium: ${medium_transcription_time}ms"
}

#######################################
# THROUGHPUT BENCHMARKS
#######################################

@test "performance: light load throughput test" {
    # Test system throughput under light load conditions
    
    echo "Testing throughput under light load ($LIGHT_LOAD_REQUESTS requests)..."
    local results_file="$BENCHMARK_RESULTS_DIR/throughput/light_load_results.json"
    
    # Test each resource under light load
    local start_time=$(date +%s%N)
    
    # Ollama throughput
    local ollama_success=0
    for i in $(seq 1 "$LIGHT_LOAD_REQUESTS"); do
        if benchmark_single_request "GET" "$OLLAMA_BASE_URL/api/tags" "" >/dev/null; then
            ollama_success=$((ollama_success + 1))
        fi
    done
    
    # QuestDB throughput
    local questdb_success=0
    for i in $(seq 1 "$LIGHT_LOAD_REQUESTS"); do
        if benchmark_questdb_query "SELECT 1" >/dev/null; then
            questdb_success=$((questdb_success + 1))
        fi
    done
    
    # Redis throughput
    local redis_success=0
    for i in $(seq 1 "$LIGHT_LOAD_REQUESTS"); do
        if benchmark_redis_command "PING" >/dev/null; then
            redis_success=$((redis_success + 1))
        fi
    done
    
    local end_time=$(date +%s%N)
    local total_duration_ms=$(( (end_time - start_time) / 1000000 ))
    local total_requests=$((LIGHT_LOAD_REQUESTS * 3))
    local requests_per_second=$(( (total_requests * 1000) / total_duration_ms ))
    
    # Generate results
    cat > "$results_file" << EOF
{
    "test_type": "light_load",
    "total_requests": $total_requests,
    "duration_ms": $total_duration_ms,
    "requests_per_second": $requests_per_second,
    "results": {
        "ollama": {"requests": $LIGHT_LOAD_REQUESTS, "successful": $ollama_success, "success_rate": $(echo "scale=2; $ollama_success * 100 / $LIGHT_LOAD_REQUESTS" | bc)},
        "questdb": {"requests": $LIGHT_LOAD_REQUESTS, "successful": $questdb_success, "success_rate": $(echo "scale=2; $questdb_success * 100 / $LIGHT_LOAD_REQUESTS" | bc)},
        "redis": {"requests": $LIGHT_LOAD_REQUESTS, "successful": $redis_success, "success_rate": $(echo "scale=2; $redis_success * 100 / $LIGHT_LOAD_REQUESTS" | bc)}
    }
}
EOF

    # Validate throughput expectations
    assert_greater_than "$requests_per_second" "10"  # At least 10 RPS under light load
    assert_equals "$ollama_success" "$LIGHT_LOAD_REQUESTS"
    assert_equals "$questdb_success" "$LIGHT_LOAD_REQUESTS"
    assert_equals "$redis_success" "$LIGHT_LOAD_REQUESTS"
    
    echo "Light load throughput: $requests_per_second RPS, duration: ${total_duration_ms}ms"
}

@test "performance: medium load throughput test" {
    # Test system throughput under medium load conditions
    
    echo "Testing throughput under medium load ($MEDIUM_LOAD_REQUESTS requests)..."
    local results_file="$BENCHMARK_RESULTS_DIR/throughput/medium_load_results.json"
    
    local start_time=$(date +%s%N)
    local total_success=0
    local total_requests=$((MEDIUM_LOAD_REQUESTS * 2))  # Test 2 resources for medium load
    
    # Mixed workload: QuestDB and Redis
    for i in $(seq 1 "$MEDIUM_LOAD_REQUESTS"); do
        # Alternate between QuestDB and Redis
        if (( i % 2 == 0 )); then
            if benchmark_questdb_query "SELECT $i" >/dev/null; then
                total_success=$((total_success + 1))
            fi
        else
            if benchmark_redis_command "SET benchmark_key_$i value_$i" >/dev/null; then
                total_success=$((total_success + 1))
            fi
        fi
    done
    
    local end_time=$(date +%s%N)
    local total_duration_ms=$(( (end_time - start_time) / 1000000 ))
    local requests_per_second=$(( (total_requests * 1000) / total_duration_ms ))
    local success_rate=$(echo "scale=2; $total_success * 100 / $total_requests" | bc)
    
    # Generate results
    cat > "$results_file" << EOF
{
    "test_type": "medium_load",
    "total_requests": $total_requests,
    "successful_requests": $total_success,
    "duration_ms": $total_duration_ms,
    "requests_per_second": $requests_per_second,
    "success_rate": $success_rate
}
EOF

    # Validate medium load performance
    assert_greater_than "$requests_per_second" "5"   # At least 5 RPS under medium load
    assert_greater_than "$success_rate" "95"         # At least 95% success rate
    
    echo "Medium load throughput: $requests_per_second RPS, success rate: ${success_rate}%"
}

#######################################
# CONCURRENCY BENCHMARKS
#######################################

@test "performance: low concurrency stress test" {
    # Test system behavior under low concurrent load
    
    echo "Testing low concurrency ($LOW_CONCURRENCY concurrent requests)..."
    local results_file="$BENCHMARK_RESULTS_DIR/concurrency/low_concurrency_results.json"
    
    local start_time=$(date +%s%N)
    local pids=()
    local temp_results=()
    
    # Start concurrent requests
    for i in $(seq 1 "$LOW_CONCURRENCY"); do
        (
            local worker_success=0
            local worker_requests=10
            
            for j in $(seq 1 "$worker_requests"); do
                if benchmark_redis_command "SET concurrent_${i}_${j} value_${j}" >/dev/null; then
                    worker_success=$((worker_success + 1))
                fi
            done
            
            echo "$worker_success" > "$BENCHMARK_RESULTS_DIR/worker_${i}_results.tmp"
        ) &
        pids+=($!)
    done
    
    # Wait for all workers to complete
    for pid in "${pids[@]}"; do
        wait "$pid"
    done
    
    local end_time=$(date +%s%N)
    local total_duration_ms=$(( (end_time - start_time) / 1000000 ))
    
    # Collect results
    local total_success=0
    for i in $(seq 1 "$LOW_CONCURRENCY"); do
        local worker_result=$(cat "$BENCHMARK_RESULTS_DIR/worker_${i}_results.tmp")
        total_success=$((total_success + worker_result))
        rm -f "$BENCHMARK_RESULTS_DIR/worker_${i}_results.tmp"
    done
    
    local total_requests=$((LOW_CONCURRENCY * 10))
    local success_rate=$(echo "scale=2; $total_success * 100 / $total_requests" | bc)
    
    # Generate results
    cat > "$results_file" << EOF
{
    "test_type": "low_concurrency",
    "concurrent_workers": $LOW_CONCURRENCY,
    "total_requests": $total_requests,
    "successful_requests": $total_success,
    "duration_ms": $total_duration_ms,
    "success_rate": $success_rate
}
EOF

    # Validate concurrency performance
    assert_greater_than "$success_rate" "90"  # At least 90% success under low concurrency
    assert_less_than "$total_duration_ms" "10000"  # Should complete within 10 seconds
    
    echo "Low concurrency: $LOW_CONCURRENCY workers, ${success_rate}% success rate, ${total_duration_ms}ms duration"
}

@test "performance: medium concurrency stress test" {
    # Test system behavior under medium concurrent load
    
    echo "Testing medium concurrency ($MEDIUM_CONCURRENCY concurrent requests)..."
    local results_file="$BENCHMARK_RESULTS_DIR/concurrency/medium_concurrency_results.json"
    
    local start_time=$(date +%s%N)
    local pids=()
    
    # Start concurrent mixed workload
    for i in $(seq 1 "$MEDIUM_CONCURRENCY"); do
        (
            local worker_success=0
            local worker_requests=5
            
            for j in $(seq 1 "$worker_requests"); do
                # Mix different resource types
                case $((j % 3)) in
                    0)
                        if benchmark_redis_command "LPUSH concurrent_list_$i item_$j" >/dev/null; then
                            worker_success=$((worker_success + 1))
                        fi
                        ;;
                    1)
                        if benchmark_questdb_query "SELECT $j as test_value" >/dev/null; then
                            worker_success=$((worker_success + 1))
                        fi
                        ;;
                    2)
                        if benchmark_single_request "GET" "$OLLAMA_BASE_URL/api/tags" "" >/dev/null; then
                            worker_success=$((worker_success + 1))
                        fi
                        ;;
                esac
            done
            
            echo "$worker_success" > "$BENCHMARK_RESULTS_DIR/medium_worker_${i}_results.tmp"
        ) &
        pids+=($!)
    done
    
    # Wait for all workers to complete
    for pid in "${pids[@]}"; do
        wait "$pid"
    done
    
    local end_time=$(date +%s%N)
    local total_duration_ms=$(( (end_time - start_time) / 1000000 ))
    
    # Collect results
    local total_success=0
    for i in $(seq 1 "$MEDIUM_CONCURRENCY"); do
        local worker_result=$(cat "$BENCHMARK_RESULTS_DIR/medium_worker_${i}_results.tmp")
        total_success=$((total_success + worker_result))
        rm -f "$BENCHMARK_RESULTS_DIR/medium_worker_${i}_results.tmp"
    done
    
    local total_requests=$((MEDIUM_CONCURRENCY * 5))
    local success_rate=$(echo "scale=2; $total_success * 100 / $total_requests" | bc)
    
    # Generate results
    cat > "$results_file" << EOF
{
    "test_type": "medium_concurrency",
    "concurrent_workers": $MEDIUM_CONCURRENCY,
    "total_requests": $total_requests,
    "successful_requests": $total_success,
    "duration_ms": $total_duration_ms,
    "success_rate": $success_rate
}
EOF

    # Validate medium concurrency performance
    assert_greater_than "$success_rate" "80"  # At least 80% success under medium concurrency
    assert_less_than "$total_duration_ms" "15000"  # Should complete within 15 seconds
    
    echo "Medium concurrency: $MEDIUM_CONCURRENCY workers, ${success_rate}% success rate, ${total_duration_ms}ms duration"
}

#######################################
# SCALABILITY BENCHMARKS
#######################################

@test "performance: data size scalability test" {
    # Test how performance scales with different data sizes
    
    echo "Testing data size scalability..."
    local results_file="$BENCHMARK_RESULTS_DIR/scalability/data_size_results.json"
    local size_results=()
    
    # Test small data
    local small_start=$(date +%s%N)
    benchmark_questdb_query "SELECT '$(head -c $SMALL_DATA_SIZE < /dev/urandom | base64 | tr -d '\n')' as data" >/dev/null
    local small_duration=$(( ($(date +%s%N) - small_start) / 1000000 ))
    size_results+=("small:$small_duration")
    
    # Test medium data
    local medium_start=$(date +%s%N)
    benchmark_questdb_query "SELECT '$(head -c 1024 < /dev/urandom | base64 | tr -d '\n')' as data" >/dev/null  # Reduced for practicality
    local medium_duration=$(( ($(date +%s%N) - medium_start) / 1000000 ))
    size_results+=("medium:$medium_duration")
    
    # Test Redis with different data sizes
    local redis_small_start=$(date +%s%N)
    local small_data=$(head -c 512 < /dev/urandom | base64 | tr -d '\n')
    benchmark_redis_command "SET size_test_small \"$small_data\"" >/dev/null
    local redis_small_duration=$(( ($(date +%s%N) - redis_small_start) / 1000000 ))
    size_results+=("redis_small:$redis_small_duration")
    
    # Generate scalability analysis
    cat > "$results_file" << EOF
{
    "test_type": "data_size_scalability",
    "results": {
        "questdb_small_ms": $small_duration,
        "questdb_medium_ms": $medium_duration,
        "redis_small_ms": $redis_small_duration,
        "scalability_factor": $(echo "scale=2; $medium_duration / $small_duration" | bc)
    }
}
EOF

    # Validate scalability
    assert_less_than "$small_duration" "$NORMAL_THRESHOLD"
    assert_less_than "$medium_duration" "$SLOW_THRESHOLD"
    
    echo "Data size scalability - Small: ${small_duration}ms, Medium: ${medium_duration}ms, Redis: ${redis_small_duration}ms"
}

@test "performance: resource interaction scalability" {
    # Test performance when multiple resources interact
    
    echo "Testing resource interaction scalability..."
    local results_file="$BENCHMARK_RESULTS_DIR/scalability/interaction_results.json"
    
    # Single resource operations
    local single_start=$(date +%s%N)
    benchmark_redis_command "SET interaction_test single_value" >/dev/null
    local single_duration=$(( ($(date +%s%N) - single_start) / 1000000 ))
    
    # Dual resource operations
    local dual_start=$(date +%s%N)
    benchmark_redis_command "SET interaction_test dual_value" >/dev/null
    benchmark_questdb_query "SELECT 'dual_test' as operation" >/dev/null
    local dual_duration=$(( ($(date +%s%N) - dual_start) / 1000000 ))
    
    # Triple resource operations
    local triple_start=$(date +%s%N)
    benchmark_redis_command "SET interaction_test triple_value" >/dev/null
    benchmark_questdb_query "SELECT 'triple_test' as operation" >/dev/null
    benchmark_single_request "GET" "$OLLAMA_BASE_URL/api/tags" "" >/dev/null
    local triple_duration=$(( ($(date +%s%N) - triple_start) / 1000000 ))
    
    # Generate interaction analysis
    cat > "$results_file" << EOF
{
    "test_type": "resource_interaction_scalability",
    "results": {
        "single_resource_ms": $single_duration,
        "dual_resource_ms": $dual_duration,
        "triple_resource_ms": $triple_duration,
        "dual_overhead_factor": $(echo "scale=2; $dual_duration / $single_duration" | bc),
        "triple_overhead_factor": $(echo "scale=2; $triple_duration / $single_duration" | bc)
    }
}
EOF

    # Validate interaction performance
    assert_less_than "$single_duration" "$FAST_THRESHOLD"
    assert_less_than "$dual_duration" "$NORMAL_THRESHOLD"
    assert_less_than "$triple_duration" "$SLOW_THRESHOLD"
    
    echo "Interaction scalability - Single: ${single_duration}ms, Dual: ${dual_duration}ms, Triple: ${triple_duration}ms"
}

#######################################
# BENCHMARK HELPER FUNCTIONS
#######################################

# Helper: Benchmark a single HTTP request
benchmark_single_request() {
    local method="$1"
    local url="$2"
    local data="$3"
    
    local start_time=$(date +%s%N)
    
    if [[ "$method" == "GET" ]]; then
        curl -s "$url" >/dev/null
    elif [[ "$method" == "POST" ]]; then
        curl -s -X POST "$url" -H "Content-Type: application/json" -d "$data" >/dev/null
    fi
    
    local end_time=$(date +%s%N)
    echo $(( (end_time - start_time) / 1000000 ))  # Convert to milliseconds
}

# Helper: Benchmark QuestDB query
benchmark_questdb_query() {
    local query="$1"
    
    local start_time=$(date +%s%N)
    curl -s -G "$QUESTDB_BASE_URL/exec" \
        --data-urlencode "query=$query" \
        --data-urlencode "fmt=json" >/dev/null
    local end_time=$(date +%s%N)
    
    echo $(( (end_time - start_time) / 1000000 ))
}

# Helper: Benchmark Redis command
benchmark_redis_command() {
    local command="$1"
    
    local start_time=$(date +%s%N)
    redis-cli -h localhost -p "$REDIS_PORT" $command >/dev/null
    local end_time=$(date +%s%N)
    
    echo $(( (end_time - start_time) / 1000000 ))
}

# Helper: Benchmark Whisper transcription
benchmark_whisper_transcription() {
    local audio_file="$1"
    
    local start_time=$(date +%s%N)
    curl -s -X POST "$WHISPER_BASE_URL/transcribe" \
        -F "audio=@${audio_file}" >/dev/null
    local end_time=$(date +%s%N)
    
    echo $(( (end_time - start_time) / 1000000 ))
}

# Helper: Generate test data file of specified size
generate_test_data_file() {
    local file_path="$1"
    local size_bytes="$2"
    
    head -c "$size_bytes" < /dev/urandom | base64 > "$file_path"
}

# Helper: Generate benchmark summary report
generate_benchmark_summary() {
    local summary_file="$BENCHMARK_RESULTS_DIR/benchmark_summary.json"
    
    # Collect all result files
    local response_time_files=($(find "$BENCHMARK_RESULTS_DIR/response_times" -name "*.json" 2>/dev/null || true))
    local throughput_files=($(find "$BENCHMARK_RESULTS_DIR/throughput" -name "*.json" 2>/dev/null || true))
    local concurrency_files=($(find "$BENCHMARK_RESULTS_DIR/concurrency" -name "*.json" 2>/dev/null || true))
    local scalability_files=($(find "$BENCHMARK_RESULTS_DIR/scalability" -name "*.json" 2>/dev/null || true))
    
    # Generate summary
    cat > "$summary_file" << EOF
{
    "benchmark_id": "$BENCHMARK_ID",
    "timestamp": "$(date -Iseconds)",
    "summary": {
        "response_time_tests": ${#response_time_files[@]},
        "throughput_tests": ${#throughput_files[@]},
        "concurrency_tests": ${#concurrency_files[@]},
        "scalability_tests": ${#scalability_files[@]}
    },
    "thresholds": {
        "fast_ms": $FAST_THRESHOLD,
        "normal_ms": $NORMAL_THRESHOLD,
        "slow_ms": $SLOW_THRESHOLD
    }
}
EOF

    echo "Benchmark summary generated: $summary_file"
}

# Helper: Assert performance threshold
assert_performance_threshold() {
    local actual_time="$1"
    local threshold="$2"
    local operation="$3"
    
    if [[ "$actual_time" -gt "$threshold" ]]; then
        echo "Performance threshold exceeded for $operation:" >&2
        echo "  Actual: ${actual_time}ms" >&2
        echo "  Threshold: ${threshold}ms" >&2
        return 1
    fi
    
    return 0
}