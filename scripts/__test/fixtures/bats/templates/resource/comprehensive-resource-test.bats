#!/usr/bin/env bats
# Comprehensive Resource Test Template
# In-depth testing for a specific Vrooli resource with all aspects covered
#
# Copy this template and customize it for your specific resource.
# Example: Complete testing suite for any Vrooli resource

#######################################
# RESOURCE CONFIGURATION
# Customize these variables for your specific resource
#######################################

# Target resource for comprehensive testing
TARGET_RESOURCE="ollama"  # Change this to your resource: ollama, questdb, redis, whisper, n8n, etc.

# Resource-specific configuration
case "$TARGET_RESOURCE" in
    "ollama")
        RESOURCE_PORT="${OLLAMA_PORT:-11434}"
        RESOURCE_BASE_URL="http://localhost:${RESOURCE_PORT}"
        RESOURCE_HEALTH_ENDPOINT="/api/tags"
        RESOURCE_API_ENDPOINTS=("/api/tags" "/api/generate" "/api/pull" "/api/version")
        RESOURCE_CONTAINER_NAME="${TEST_NAMESPACE}_ollama"
        ;;
    "questdb")
        RESOURCE_PORT="${QUESTDB_HTTP_PORT:-9010}"
        RESOURCE_BASE_URL="http://localhost:${RESOURCE_PORT}"
        RESOURCE_HEALTH_ENDPOINT="/status"
        RESOURCE_API_ENDPOINTS=("/status" "/exec" "/imp")
        RESOURCE_CONTAINER_NAME="${TEST_NAMESPACE}_questdb"
        ;;
    "redis")
        RESOURCE_PORT="${REDIS_PORT:-6379}"
        RESOURCE_BASE_URL="redis://localhost:${RESOURCE_PORT}"
        RESOURCE_HEALTH_ENDPOINT="PING"
        RESOURCE_CONTAINER_NAME="${TEST_NAMESPACE}_redis"
        ;;
    "whisper")
        RESOURCE_PORT="${WHISPER_PORT:-8090}"
        RESOURCE_BASE_URL="http://localhost:${RESOURCE_PORT}"
        RESOURCE_HEALTH_ENDPOINT="/health"
        RESOURCE_API_ENDPOINTS=("/health" "/transcribe" "/translate")
        RESOURCE_CONTAINER_NAME="${TEST_NAMESPACE}_whisper"
        ;;
    "n8n")
        RESOURCE_PORT="${N8N_PORT:-5678}"
        RESOURCE_BASE_URL="http://localhost:${RESOURCE_PORT}"
        RESOURCE_HEALTH_ENDPOINT="/healthz"
        RESOURCE_API_ENDPOINTS=("/healthz" "/api/v1/workflows" "/api/v1/executions")
        RESOURCE_CONTAINER_NAME="${TEST_NAMESPACE}_n8n"
        ;;
    *)
        echo "Unsupported resource: $TARGET_RESOURCE" >&2
        exit 1
        ;;
esac

# Test categories to run
TEST_CATEGORIES=(
    "infrastructure"    # Basic setup, connectivity, health
    "api"              # API functionality and endpoints
    "performance"      # Response times and throughput
    "reliability"      # Error handling and recovery
    "security"         # Security aspects and validation
    "integration"      # Integration with other resources
    "data"             # Data handling and persistence
    "monitoring"       # Metrics and observability
)

# Performance thresholds
FAST_RESPONSE_MS=100
NORMAL_RESPONSE_MS=1000
SLOW_RESPONSE_MS=5000
ERROR_RATE_THRESHOLD=5

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
    # Setup resource test environment
    setup_resource_test "$TARGET_RESOURCE"
    
    # Configure comprehensive testing environment
    export RESOURCE_TEST_ID="comprehensive_${TARGET_RESOURCE}_$$"
    export RESOURCE_TEST_DATA_DIR="$TEST_TMPDIR/resource_test_data"
    export RESOURCE_TEST_RESULTS_DIR="$TEST_TMPDIR/resource_test_results"
    export RESOURCE_TEST_CONFIG_FILE="$TEST_TMPDIR/resource_test_config.json"
    
    # Create test directories
    mkdir -p "$RESOURCE_TEST_DATA_DIR"/{input,output,temp,cache}
    mkdir -p "$RESOURCE_TEST_RESULTS_DIR"/{infrastructure,api,performance,reliability,security,integration,data,monitoring}
    
    # Generate resource-specific test data
    generate_resource_test_data
    
    # Generate test configuration
    cat > "$RESOURCE_TEST_CONFIG_FILE" << EOF
{
    "test_id": "$RESOURCE_TEST_ID",
    "timestamp": "$(date -Iseconds)",
    "target_resource": "$TARGET_RESOURCE",
    "resource_config": {
        "port": $RESOURCE_PORT,
        "base_url": "$RESOURCE_BASE_URL",
        "health_endpoint": "$RESOURCE_HEALTH_ENDPOINT",
        "container_name": "$RESOURCE_CONTAINER_NAME"
    },
    "test_categories": $(printf '%s\n' "${TEST_CATEGORIES[@]}" | jq -R . | jq -s '.'),
    "thresholds": {
        "fast_response_ms": $FAST_RESPONSE_MS,
        "normal_response_ms": $NORMAL_RESPONSE_MS,
        "slow_response_ms": $SLOW_RESPONSE_MS,
        "error_rate_threshold": $ERROR_RATE_THRESHOLD
    }
}
EOF

    # Setup mock verification expectations
    mock::verify::expect_calls "docker" "inspect.*${RESOURCE_CONTAINER_NAME}" 5
    mock::verify::expect_calls "docker" "run.*${TARGET_RESOURCE}" 1
    
    if [[ "$TARGET_RESOURCE" != "redis" ]]; then
        mock::verify::expect_min_calls "http" ".*" 10
    fi
    
    echo "Comprehensive testing setup completed for resource: $TARGET_RESOURCE"
}

teardown() {
    # Generate comprehensive test report
    generate_comprehensive_test_report
    
    # Validate test expectations
    mock::verify::validate_all
    
    # Archive test results
    if [[ -d "$RESOURCE_TEST_RESULTS_DIR" ]]; then
        tar -czf "${TEST_TMPDIR}/comprehensive_${TARGET_RESOURCE}_results_${RESOURCE_TEST_ID}.tar.gz" -C "$RESOURCE_TEST_RESULTS_DIR" .
        echo "Comprehensive test results archived"
    fi
    
    # Clean up test data
    rm -rf "$RESOURCE_TEST_DATA_DIR" "$RESOURCE_TEST_RESULTS_DIR"
    rm -f "$RESOURCE_TEST_CONFIG_FILE"
    
    # Standard teardown
    teardown_test_environment
}

#######################################
# INFRASTRUCTURE TESTS
#######################################

@test "infrastructure: container status and health" {
    # Test basic infrastructure requirements
    
    echo "Testing infrastructure for $TARGET_RESOURCE..."
    local results_file="$RESOURCE_TEST_RESULTS_DIR/infrastructure/container_health.json"
    
    # Check container is running
    assert_docker_container_running "$RESOURCE_CONTAINER_NAME"
    
    # Check container resource usage
    local container_stats
    container_stats=$(docker stats "$RESOURCE_CONTAINER_NAME" --no-stream --format "table {{.CPUPerc}}\t{{.MemUsage}}")
    
    # Extract CPU and memory usage
    local cpu_usage=$(echo "$container_stats" | tail -n 1 | awk '{print $1}' | sed 's/%//')
    local memory_usage=$(echo "$container_stats" | tail -n 1 | awk '{print $2}')
    
    # Validate reasonable resource usage
    assert_less_than "${cpu_usage%.*}" "80"  # Less than 80% CPU
    
    # Check container logs for errors
    local recent_logs
    recent_logs=$(docker logs "$RESOURCE_CONTAINER_NAME" --tail 50 2>&1)
    
    # Should not contain critical errors
    run echo "$recent_logs"
    assert_output_not_contains "FATAL"
    assert_output_not_contains "ERROR.*critical"
    
    # Generate infrastructure results
    cat > "$results_file" << EOF
{
    "container_name": "$RESOURCE_CONTAINER_NAME",
    "status": "running",
    "cpu_usage_percent": "$cpu_usage",
    "memory_usage": "$memory_usage",
    "log_analysis": {
        "total_lines": $(echo "$recent_logs" | wc -l),
        "has_fatal_errors": $(echo "$recent_logs" | grep -c "FATAL" || echo "0"),
        "has_critical_errors": $(echo "$recent_logs" | grep -c "ERROR.*critical" || echo "0")
    }
}
EOF

    echo "Infrastructure validation completed - CPU: ${cpu_usage}%, Memory: $memory_usage"
}

@test "infrastructure: network connectivity and ports" {
    # Test network connectivity and port accessibility
    
    echo "Testing network connectivity for $TARGET_RESOURCE..."
    local results_file="$RESOURCE_TEST_RESULTS_DIR/infrastructure/network_connectivity.json"
    
    # Test port accessibility
    assert_port_open "localhost" "$RESOURCE_PORT"
    
    # Test basic connectivity based on resource type
    case "$TARGET_RESOURCE" in
        "redis")
            # Test Redis connection
            run redis-cli -h localhost -p "$RESOURCE_PORT" ping
            assert_success
            assert_output "PONG"
            ;;
        *)
            # Test HTTP connectivity
            run curl -s -o /dev/null -w "%{http_code}" "$RESOURCE_BASE_URL$RESOURCE_HEALTH_ENDPOINT"
            assert_success
            assert_output_matches "^(200|201|202)$"
            ;;
    esac
    
    # Test connection timing
    local connection_start=$(date +%s%N)
    case "$TARGET_RESOURCE" in
        "redis")
            redis-cli -h localhost -p "$RESOURCE_PORT" ping >/dev/null
            ;;
        *)
            curl -s "$RESOURCE_BASE_URL$RESOURCE_HEALTH_ENDPOINT" >/dev/null
            ;;
    esac
    local connection_end=$(date +%s%N)
    local connection_time=$(( (connection_end - connection_start) / 1000000 ))
    
    # Connection should be fast
    assert_less_than "$connection_time" "$FAST_RESPONSE_MS"
    
    # Generate network connectivity results
    cat > "$results_file" << EOF
{
    "port": $RESOURCE_PORT,
    "port_accessible": true,
    "connection_time_ms": $connection_time,
    "health_endpoint": "$RESOURCE_HEALTH_ENDPOINT",
    "connectivity_test": "passed"
}
EOF

    echo "Network connectivity validated - Port: $RESOURCE_PORT, Connection time: ${connection_time}ms"
}

@test "infrastructure: resource dependencies" {
    # Test resource dependencies and prerequisites
    
    echo "Testing dependencies for $TARGET_RESOURCE..."
    local results_file="$RESOURCE_TEST_RESULTS_DIR/infrastructure/dependencies.json"
    
    local dependency_checks=()
    
    # Check Docker dependency
    assert_command_exists "docker"
    dependency_checks+=("docker:available")
    
    # Check resource-specific dependencies
    case "$TARGET_RESOURCE" in
        "ollama")
            # Check if required models are available (mocked)
            run curl -s "$OLLAMA_BASE_URL/api/tags"
            assert_success
            assert_json_valid "$output"
            dependency_checks+=("models:available")
            ;;
        "questdb")
            # Check if database is accessible
            run curl -s "$QUESTDB_BASE_URL/status"
            assert_success
            assert_json_field_equals "$output" ".status" "OK"
            dependency_checks+=("database:accessible")
            ;;
        "redis")
            # Check Redis server info
            run redis-cli -h localhost -p "$RESOURCE_PORT" info server
            assert_success
            assert_output_contains "redis_version"
            dependency_checks+=("server_info:available")
            ;;
        "whisper")
            # Check Whisper model availability
            run curl -s "$WHISPER_BASE_URL/health"
            assert_success
            assert_json_field_exists "$output" ".status"
            dependency_checks+=("model:loaded")
            ;;
        "n8n")
            # Check N8N workflow engine
            run curl -s "$N8N_BASE_URL/healthz"
            assert_success
            assert_json_field_equals "$output" ".status" "ok"
            dependency_checks+=("workflow_engine:ready")
            ;;
    esac
    
    # Check network dependencies
    if command -v curl >/dev/null 2>&1; then
        dependency_checks+=("curl:available")
    fi
    
    if command -v jq >/dev/null 2>&1; then
        dependency_checks+=("jq:available")
    fi
    
    # Generate dependency results
    cat > "$results_file" << EOF
{
    "resource": "$TARGET_RESOURCE",
    "dependency_checks": $(printf '%s\n' "${dependency_checks[@]}" | jq -R 'split(":") | {name: .[0], status: .[1]}' | jq -s '.'),
    "all_dependencies_met": true
}
EOF

    echo "Dependencies validated: ${#dependency_checks[@]} checks passed"
}

#######################################
# API FUNCTIONALITY TESTS
#######################################

@test "api: endpoint availability and basic functionality" {
    # Test all API endpoints for basic functionality
    
    echo "Testing API endpoints for $TARGET_RESOURCE..."
    local results_file="$RESOURCE_TEST_RESULTS_DIR/api/endpoint_functionality.json"
    
    local endpoint_results=()
    
    case "$TARGET_RESOURCE" in
        "redis")
            # Test Redis commands
            local redis_commands=("PING" "INFO" "SET test_key test_value" "GET test_key" "DEL test_key")
            for cmd in "${redis_commands[@]}"; do
                local cmd_start=$(date +%s%N)
                run redis-cli -h localhost -p "$RESOURCE_PORT" $cmd
                local cmd_end=$(date +%s%N)
                local cmd_time=$(( (cmd_end - cmd_start) / 1000000 ))
                
                if [[ $status -eq 0 ]]; then
                    endpoint_results+=("${cmd// /_}:success:${cmd_time}")
                else
                    endpoint_results+=("${cmd// /_}:failure:${cmd_time}")
                fi
            done
            ;;
        *)
            # Test HTTP endpoints
            for endpoint in "${RESOURCE_API_ENDPOINTS[@]}"; do
                local endpoint_start=$(date +%s%N)
                run curl -s -o /dev/null -w "%{http_code}" "$RESOURCE_BASE_URL$endpoint"
                local endpoint_end=$(date +%s%N)
                local endpoint_time=$(( (endpoint_end - endpoint_start) / 1000000 ))
                
                if [[ "$output" =~ ^(200|201|202)$ ]]; then
                    endpoint_results+=("${endpoint//\//_}:success:${endpoint_time}")
                else
                    endpoint_results+=("${endpoint//\//_}:failure:${endpoint_time}")
                fi
            done
            ;;
    esac
    
    # Validate all endpoints are working
    local successful_endpoints=0
    local total_endpoints=${#endpoint_results[@]}
    
    for result in "${endpoint_results[@]}"; do
        local status=$(echo "$result" | cut -d':' -f2)
        if [[ "$status" == "success" ]]; then
            successful_endpoints=$((successful_endpoints + 1))
        fi
    done
    
    local success_rate=$(echo "scale=2; $successful_endpoints * 100 / $total_endpoints" | bc)
    
    # Generate API results
    cat > "$results_file" << EOF
{
    "resource": "$TARGET_RESOURCE",
    "total_endpoints": $total_endpoints,
    "successful_endpoints": $successful_endpoints,
    "success_rate": $success_rate,
    "endpoint_details": $(printf '%s\n' "${endpoint_results[@]}" | jq -R 'split(":") | {endpoint: .[0], status: .[1], response_time_ms: (.[2] | tonumber)}' | jq -s '.')
}
EOF

    # Validate API functionality
    assert_greater_than "$success_rate" "90"  # At least 90% of endpoints should work
    
    echo "API functionality validated: ${successful_endpoints}/${total_endpoints} endpoints working (${success_rate}%)"
}

@test "api: data format validation and error handling" {
    # Test API data formats and error handling
    
    echo "Testing API data formats and error handling for $TARGET_RESOURCE..."
    local results_file="$RESOURCE_TEST_RESULTS_DIR/api/data_validation.json"
    
    local validation_results=()
    
    case "$TARGET_RESOURCE" in
        "ollama")
            # Test valid request
            run curl -s -X POST "$OLLAMA_BASE_URL/api/generate" \
                -H "Content-Type: application/json" \
                -d '{"model":"llama3.1:8b","prompt":"Hello","stream":false}'
            assert_success
            assert_json_valid "$output"
            assert_json_field_exists "$output" ".response"
            validation_results+=("valid_generation:success")
            
            # Test invalid request
            run curl -s -X POST "$OLLAMA_BASE_URL/api/generate" \
                -H "Content-Type: application/json" \
                -d '{"invalid":"request"}'
            # Should handle gracefully
            validation_results+=("invalid_request:handled")
            ;;
        "questdb")
            # Test valid query
            run curl -s -G "$QUESTDB_BASE_URL/exec" \
                --data-urlencode "query=SELECT 1 as test" \
                --data-urlencode "fmt=json"
            assert_success
            assert_json_valid "$output"
            assert_json_field_exists "$output" ".dataset"
            validation_results+=("valid_query:success")
            
            # Test invalid query
            run curl -s -G "$QUESTDB_BASE_URL/exec" \
                --data-urlencode "query=INVALID SQL SYNTAX" \
                --data-urlencode "fmt=json"
            # Should return error response
            validation_results+=("invalid_query:error_handled")
            ;;
        "redis")
            # Test valid commands
            run redis-cli -h localhost -p "$RESOURCE_PORT" SET api_test_key "test_value"
            assert_success
            assert_output "OK"
            validation_results+=("valid_set:success")
            
            run redis-cli -h localhost -p "$RESOURCE_PORT" GET api_test_key
            assert_success
            assert_output "test_value"
            validation_results+=("valid_get:success")
            
            # Test invalid command
            run redis-cli -h localhost -p "$RESOURCE_PORT" INVALID_COMMAND
            assert_failure
            validation_results+=("invalid_command:error_handled")
            ;;
        "whisper")
            # Test health endpoint
            run curl -s "$WHISPER_BASE_URL/health"
            assert_success
            assert_json_valid "$output"
            validation_results+=("health_check:success")
            
            # Test transcription with mock data
            local test_audio_file="$RESOURCE_TEST_DATA_DIR/input/test_audio.wav"
            echo "mock audio data" > "$test_audio_file"
            
            run curl -s -X POST "$WHISPER_BASE_URL/transcribe" \
                -F "audio=@${test_audio_file}"
            assert_success
            assert_json_valid "$output"
            validation_results+=("transcription:success")
            ;;
        "n8n")
            # Test health endpoint
            run curl -s "$N8N_BASE_URL/healthz"
            assert_success
            assert_json_valid "$output"
            assert_json_field_equals "$output" ".status" "ok"
            validation_results+=("health_check:success")
            
            # Test workflow endpoint
            run curl -s "$N8N_BASE_URL/api/v1/workflows"
            assert_success
            assert_json_valid "$output"
            validation_results+=("workflows_list:success")
            ;;
    esac
    
    # Generate validation results
    cat > "$results_file" << EOF
{
    "resource": "$TARGET_RESOURCE",
    "validation_tests": $(printf '%s\n' "${validation_results[@]}" | jq -R 'split(":") | {test: .[0], result: .[1]}' | jq -s '.'),
    "total_tests": ${#validation_results[@]}
}
EOF

    echo "API data validation completed: ${#validation_results[@]} tests executed"
}

#######################################
# PERFORMANCE TESTS
#######################################

@test "performance: response time benchmarks" {
    # Test response times for all operations
    
    echo "Running performance benchmarks for $TARGET_RESOURCE..."
    local results_file="$RESOURCE_TEST_RESULTS_DIR/performance/response_times.json"
    
    local performance_results=()
    local iterations=10
    
    case "$TARGET_RESOURCE" in
        "redis")
            # Benchmark Redis operations
            local operations=("PING" "SET perf_key perf_value" "GET perf_key" "INFO server")
            for operation in "${operations[@]}"; do
                local total_time=0
                local successful_ops=0
                
                for i in $(seq 1 $iterations); do
                    local op_start=$(date +%s%N)
                    if redis-cli -h localhost -p "$RESOURCE_PORT" $operation >/dev/null 2>&1; then
                        local op_end=$(date +%s%N)
                        local op_time=$(( (op_end - op_start) / 1000000 ))
                        total_time=$((total_time + op_time))
                        successful_ops=$((successful_ops + 1))
                    fi
                done
                
                if [[ $successful_ops -gt 0 ]]; then
                    local avg_time=$((total_time / successful_ops))
                    performance_results+=("${operation// /_}:${avg_time}:${successful_ops}")
                fi
            done
            ;;
        *)
            # Benchmark HTTP operations
            for endpoint in "${RESOURCE_API_ENDPOINTS[@]}"; do
                local total_time=0
                local successful_ops=0
                
                for i in $(seq 1 $iterations); do
                    local op_start=$(date +%s%N)
                    local status_code=$(curl -s -o /dev/null -w "%{http_code}" "$RESOURCE_BASE_URL$endpoint" 2>/dev/null)
                    local op_end=$(date +%s%N)
                    
                    if [[ "$status_code" =~ ^(200|201|202)$ ]]; then
                        local op_time=$(( (op_end - op_start) / 1000000 ))
                        total_time=$((total_time + op_time))
                        successful_ops=$((successful_ops + 1))
                    fi
                done
                
                if [[ $successful_ops -gt 0 ]]; then
                    local avg_time=$((total_time / successful_ops))
                    performance_results+=("${endpoint//\//_}:${avg_time}:${successful_ops}")
                fi
            done
            ;;
    esac
    
    # Validate performance thresholds
    for result in "${performance_results[@]}"; do
        local avg_time=$(echo "$result" | cut -d':' -f2)
        local operation=$(echo "$result" | cut -d':' -f1)
        
        # Most operations should be fast
        if [[ "$operation" =~ (ping|health|status) ]]; then
            assert_less_than "$avg_time" "$FAST_RESPONSE_MS"
        else
            assert_less_than "$avg_time" "$NORMAL_RESPONSE_MS"
        fi
    done
    
    # Generate performance results
    cat > "$results_file" << EOF
{
    "resource": "$TARGET_RESOURCE",
    "iterations_per_test": $iterations,
    "performance_results": $(printf '%s\n' "${performance_results[@]}" | jq -R 'split(":") | {operation: .[0], avg_response_time_ms: (.[1] | tonumber), successful_iterations: (.[2] | tonumber)}' | jq -s '.'),
    "thresholds": {
        "fast_ms": $FAST_RESPONSE_MS,
        "normal_ms": $NORMAL_RESPONSE_MS,
        "slow_ms": $SLOW_RESPONSE_MS
    }
}
EOF

    echo "Performance benchmarks completed: ${#performance_results[@]} operations tested"
}

@test "performance: load and stress testing" {
    # Test resource behavior under load
    
    echo "Running load testing for $TARGET_RESOURCE..."
    local results_file="$RESOURCE_TEST_RESULTS_DIR/performance/load_testing.json"
    
    local load_scenarios=("light" "medium" "heavy")
    local load_results=()
    
    for scenario in "${load_scenarios[@]}"; do
        local concurrent_requests=0
        local test_duration=0
        
        case "$scenario" in
            "light")
                concurrent_requests=5
                test_duration=30
                ;;
            "medium")
                concurrent_requests=15
                test_duration=30
                ;;
            "heavy")
                concurrent_requests=30
                test_duration=30
                ;;
        esac
        
        echo "Running $scenario load test: $concurrent_requests concurrent requests for ${test_duration}s"
        
        local scenario_start=$(date +%s)
        local total_requests=0
        local successful_requests=0
        local pids=()
        
        # Start concurrent workers
        for worker in $(seq 1 "$concurrent_requests"); do
            (
                local worker_requests=0
                local worker_successful=0
                
                while [[ $(($(date +%s) - scenario_start)) -lt $test_duration ]]; do
                    case "$TARGET_RESOURCE" in
                        "redis")
                            if redis-cli -h localhost -p "$RESOURCE_PORT" SET "load_key_${worker}_${worker_requests}" "value_${worker_requests}" >/dev/null 2>&1; then
                                worker_successful=$((worker_successful + 1))
                            fi
                            ;;
                        *)
                            if curl -s "$RESOURCE_BASE_URL$RESOURCE_HEALTH_ENDPOINT" >/dev/null 2>&1; then
                                worker_successful=$((worker_successful + 1))
                            fi
                            ;;
                    esac
                    
                    worker_requests=$((worker_requests + 1))
                    sleep 1
                done
                
                echo "$worker_requests:$worker_successful" > "$RESOURCE_TEST_DATA_DIR/temp/worker_${scenario}_${worker}.result"
            ) &
            pids+=($!)
        done
        
        # Wait for test completion
        sleep "$test_duration"
        
        # Kill workers and collect results
        for pid in "${pids[@]}"; do
            kill "$pid" 2>/dev/null || true
        done
        wait 2>/dev/null || true
        
        # Collect worker results
        for worker in $(seq 1 "$concurrent_requests"); do
            local result_file="$RESOURCE_TEST_DATA_DIR/temp/worker_${scenario}_${worker}.result"
            if [[ -f "$result_file" ]]; then
                local worker_data=$(cat "$result_file")
                local worker_total=$(echo "$worker_data" | cut -d':' -f1)
                local worker_success=$(echo "$worker_data" | cut -d':' -f2)
                
                total_requests=$((total_requests + worker_total))
                successful_requests=$((successful_requests + worker_success))
                
                rm -f "$result_file"
            fi
        done
        
        local success_rate=$(echo "scale=2; $successful_requests * 100 / $total_requests" | bc)
        local requests_per_second=$(echo "scale=2; $total_requests / $test_duration" | bc)
        
        load_results+=("${scenario}:${requests_per_second}:${success_rate}")
        
        echo "$scenario load completed: ${requests_per_second} RPS, ${success_rate}% success rate"
    done
    
    # Generate load testing results
    cat > "$results_file" << EOF
{
    "resource": "$TARGET_RESOURCE",
    "load_scenarios": $(printf '%s\n' "${load_results[@]}" | jq -R 'split(":") | {scenario: .[0], requests_per_second: (.[1] | tonumber), success_rate: (.[2] | tonumber)}' | jq -s '.')
}
EOF

    # Validate load testing results
    for result in "${load_results[@]}"; do
        local success_rate=$(echo "$result" | cut -d':' -f3)
        assert_greater_than "$success_rate" "$((100 - ERROR_RATE_THRESHOLD * 2))"  # Allow higher error rate under load
    done

    echo "Load testing completed: ${#load_scenarios[@]} scenarios tested"
}

#######################################
# HELPER FUNCTIONS
#######################################

# Helper: Generate resource-specific test data
generate_resource_test_data() {
    case "$TARGET_RESOURCE" in
        "whisper")
            # Generate mock audio files
            echo "mock audio data for small file" > "$RESOURCE_TEST_DATA_DIR/input/small_audio.wav"
            head -c 10240 < /dev/urandom | base64 > "$RESOURCE_TEST_DATA_DIR/input/medium_audio.wav"
            ;;
        "questdb")
            # Generate test CSV data
            cat > "$RESOURCE_TEST_DATA_DIR/input/test_data.csv" << 'EOF'
timestamp,metric_name,value,source
2024-01-01T00:00:00.000Z,cpu_usage,75.5,server1
2024-01-01T00:01:00.000Z,memory_usage,60.2,server1
2024-01-01T00:02:00.000Z,disk_usage,45.8,server1
EOF
            ;;
        "ollama")
            # Generate test prompts
            cat > "$RESOURCE_TEST_DATA_DIR/input/test_prompts.json" << 'EOF'
[
    {"prompt": "Hello, world!", "model": "llama3.1:8b"},
    {"prompt": "What is 2+2?", "model": "llama3.1:8b"},
    {"prompt": "Explain AI briefly", "model": "llama3.1:8b"}
]
EOF
            ;;
        "n8n")
            # Generate test workflow
            cat > "$RESOURCE_TEST_DATA_DIR/input/test_workflow.json" << 'EOF'
{
    "name": "Test Workflow",
    "nodes": [
        {
            "name": "Start",
            "type": "n8n-nodes-base.start"
        }
    ],
    "connections": {}
}
EOF
            ;;
    esac
}

# Helper: Generate comprehensive test report
generate_comprehensive_test_report() {
    local report_file="$RESOURCE_TEST_RESULTS_DIR/comprehensive_test_report.json"
    
    # Collect all result files
    local result_files=($(find "$RESOURCE_TEST_RESULTS_DIR" -name "*.json" -not -name "comprehensive_test_report.json" 2>/dev/null || true))
    
    # Generate summary
    cat > "$report_file" << EOF
{
    "test_id": "$RESOURCE_TEST_ID",
    "timestamp": "$(date -Iseconds)",
    "resource": "$TARGET_RESOURCE",
    "test_summary": {
        "total_test_files": ${#result_files[@]},
        "test_categories": $(printf '%s\n' "${TEST_CATEGORIES[@]}" | jq -R . | jq -s '.'),
        "resource_config": {
            "port": $RESOURCE_PORT,
            "base_url": "$RESOURCE_BASE_URL",
            "container_name": "$RESOURCE_CONTAINER_NAME"
        }
    },
    "test_files": $(printf '%s\n' "${result_files[@]}" | jq -R . | jq -s '.')
}
EOF

    echo "Comprehensive test report generated: $report_file"
}