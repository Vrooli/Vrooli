#!/bin/bash
# AI Model Orchestra Controller - Performance Phase Tests
set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_ROOT="$(cd "$TEST_DIR/../.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() { echo -e "ℹ️  $1"; }
log_success() { echo -e "${GREEN}✅ $1${NC}"; }
log_error() { echo -e "${RED}❌ $1${NC}"; }
log_warn() { echo -e "${YELLOW}⚠️  $1${NC}"; }

echo "🚀 Running performance tests for AI Model Orchestra Controller..."

# Test API response times
test_api_response_times() {
    log_info "Testing API response times..."
    
    local api_port
    api_port=$(vrooli scenario port ai-model-orchestra-controller API_PORT 2>/dev/null || echo "")
    
    if [ -z "$api_port" ]; then
        log_error "Scenario not running - cannot test performance"
        return 1
    fi
    
    local base_url="http://localhost:${api_port}/api/v1"
    
    # Test health endpoint response time (should be very fast)
    log_info "Testing health endpoint response time..."
    local health_url="${base_url}/health"
    local health_time
    health_time=$(timeout 10 curl -w "%{time_total}" -o /dev/null -s "$health_url" 2>/dev/null || echo "999")
    
    # Convert to milliseconds for easier comparison
    local health_ms=$(echo "$health_time * 1000" | bc 2>/dev/null || echo "999")
    
    if (( $(echo "$health_ms < 500" | bc -l 2>/dev/null || echo "0") )); then
        log_success "Health endpoint response time: ${health_ms}ms (excellent)"
    elif (( $(echo "$health_ms < 1000" | bc -l 2>/dev/null || echo "0") )); then
        log_success "Health endpoint response time: ${health_ms}ms (good)"
    else
        log_warn "Health endpoint response time: ${health_ms}ms (may need optimization)"
    fi
    
    # Test models endpoint response time
    log_info "Testing models endpoint response time..."
    local models_url="${base_url}/models"
    local models_time
    models_time=$(timeout 30 curl -w "%{time_total}" -o /dev/null -s "$models_url" 2>/dev/null || echo "999")
    local models_ms=$(echo "$models_time * 1000" | bc 2>/dev/null || echo "999")
    
    if (( $(echo "$models_ms < 2000" | bc -l 2>/dev/null || echo "0") )); then
        log_success "Models endpoint response time: ${models_ms}ms (good)"
    elif (( $(echo "$models_ms < 5000" | bc -l 2>/dev/null || echo "0") )); then
        log_success "Models endpoint response time: ${models_ms}ms (acceptable)"
    else
        log_warn "Models endpoint response time: ${models_ms}ms (may need caching)"
    fi
    
    # Test metrics endpoint response time
    log_info "Testing metrics endpoint response time..."
    local metrics_url="${base_url}/metrics"
    local metrics_time
    metrics_time=$(timeout 15 curl -w "%{time_total}" -o /dev/null -s "$metrics_url" 2>/dev/null || echo "999")
    local metrics_ms=$(echo "$metrics_time * 1000" | bc 2>/dev/null || echo "999")
    
    if (( $(echo "$metrics_ms < 1000" | bc -l 2>/dev/null || echo "0") )); then
        log_success "Metrics endpoint response time: ${metrics_ms}ms (good)"
    elif (( $(echo "$metrics_ms < 3000" | bc -l 2>/dev/null || echo "0") )); then
        log_success "Metrics endpoint response time: ${metrics_ms}ms (acceptable)"
    else
        log_warn "Metrics endpoint response time: ${metrics_ms}ms (consider optimization)"
    fi
    
    return 0
}

# Test concurrent request handling
test_concurrent_requests() {
    log_info "Testing concurrent request handling..."
    
    local api_port
    api_port=$(vrooli scenario port ai-model-orchestra-controller API_PORT 2>/dev/null || echo "")
    
    if [ -z "$api_port" ]; then
        log_error "Cannot test concurrent requests - scenario not running"
        return 1
    fi
    
    local base_url="http://localhost:${api_port}/api/v1"
    local health_url="${base_url}/health"
    
    # Test with increasing concurrent load
    local concurrent_levels=(5 10 20)
    
    for level in "${concurrent_levels[@]}"; do
        log_info "Testing with $level concurrent requests..."
        
        local temp_dir="/tmp/perf-test-$$-${level}"
        mkdir -p "$temp_dir"
        
        local start_time=$(date +%s.%N)
        local pids=()
        
        # Launch concurrent requests
        for ((i=1; i<=level; i++)); do
            (
                timeout 20 curl -s "$health_url" > "$temp_dir/response_${i}.json" 2>/dev/null
                echo $? > "$temp_dir/exit_${i}.txt"
            ) &
            pids+=($!)
        done
        
        # Wait for all requests to complete
        local completed=0
        local failed=0
        
        for i in $(seq 1 "$level"); do
            wait ${pids[$((i-1))]} 2>/dev/null || true
            
            if [ -f "$temp_dir/exit_${i}.txt" ] && [ "$(cat "$temp_dir/exit_${i}.txt")" = "0" ]; then
                ((completed++))
            else
                ((failed++))
            fi
        done
        
        local end_time=$(date +%s.%N)
        local duration=$(echo "$end_time - $start_time" | bc 2>/dev/null || echo "unknown")
        
        # Cleanup
        rm -rf "$temp_dir"
        
        # Analyze results
        local success_rate=$(echo "scale=1; $completed * 100 / $level" | bc 2>/dev/null || echo "0")
        
        if (( $(echo "$success_rate >= 95" | bc -l 2>/dev/null || echo "0") )); then
            log_success "Level $level: ${completed}/${level} requests completed (${success_rate}%, ${duration}s)"
        elif (( $(echo "$success_rate >= 80" | bc -l 2>/dev/null || echo "0") )); then
            log_warn "Level $level: ${completed}/${level} requests completed (${success_rate}%, ${duration}s) - some degradation"
        else
            log_error "Level $level: ${completed}/${level} requests completed (${success_rate}%, ${duration}s) - significant failures"
        fi
    done
    
    return 0
}

# Test memory usage patterns
test_memory_usage() {
    log_info "Testing memory usage patterns..."
    
    # Check if scenario process is running and get its PID
    local scenario_pid
    scenario_pid=$(pgrep -f "ai-model-orchestra-controller-api" 2>/dev/null || echo "")
    
    if [ -z "$scenario_pid" ]; then
        log_warn "Scenario process not found - cannot test memory usage"
        return 0
    fi
    
    # Monitor memory usage over time
    log_info "Monitoring memory usage for 30 seconds..."
    local memory_samples=()
    local max_memory=0
    local min_memory=999999
    
    for i in {1..10}; do
        if [ -d "/proc/$scenario_pid" ]; then
            local memory_kb
            memory_kb=$(ps -p "$scenario_pid" -o rss= 2>/dev/null | tr -d ' ' || echo "0")
            
            if [ "$memory_kb" -gt 0 ]; then
                memory_samples+=("$memory_kb")
                
                if [ "$memory_kb" -gt "$max_memory" ]; then
                    max_memory="$memory_kb"
                fi
                
                if [ "$memory_kb" -lt "$min_memory" ]; then
                    min_memory="$memory_kb"
                fi
            fi
        fi
        
        sleep 3
    done
    
    if [ ${#memory_samples[@]} -gt 0 ]; then
        local avg_memory=0
        for mem in "${memory_samples[@]}"; do
            avg_memory=$((avg_memory + mem))
        done
        avg_memory=$((avg_memory / ${#memory_samples[@]}))
        
        local max_memory_mb=$((max_memory / 1024))
        local avg_memory_mb=$((avg_memory / 1024))
        local min_memory_mb=$((min_memory / 1024))
        
        log_success "Memory usage - Min: ${min_memory_mb}MB, Avg: ${avg_memory_mb}MB, Max: ${max_memory_mb}MB"
        
        # Alert if memory usage is concerning
        if [ "$max_memory_mb" -gt 500 ]; then
            log_warn "High memory usage detected (${max_memory_mb}MB) - consider optimization"
        elif [ "$max_memory_mb" -gt 200 ]; then
            log_info "Moderate memory usage (${max_memory_mb}MB) - monitor in production"
        else
            log_success "Low memory footprint (${max_memory_mb}MB) - efficient"
        fi
    else
        log_warn "Could not collect memory usage samples"
    fi
    
    return 0
}

# Test database query performance
test_database_performance() {
    log_info "Testing database query performance..."
    
    # Check if PostgreSQL is available
    local postgres_port
    postgres_port=$(vrooli resource port postgres RESOURCE_PORTS_POSTGRES 2>/dev/null || echo "5432")
    
    if ! timeout 5 pg_isready -h localhost -p "$postgres_port" >/dev/null 2>&1; then
        log_warn "PostgreSQL not available - skipping database performance tests"
        return 0
    fi
    
    # Test basic connection time
    log_info "Testing database connection time..."
    local connect_start=$(date +%s.%N)
    
    if timeout 10 psql -h localhost -p "$postgres_port" -U postgres -d postgres -c "SELECT 1;" >/dev/null 2>&1; then
        local connect_end=$(date +%s.%N)
        local connect_time=$(echo "($connect_end - $connect_start) * 1000" | bc 2>/dev/null || echo "unknown")
        
        if (( $(echo "$connect_time < 100" | bc -l 2>/dev/null || echo "0") )); then
            log_success "Database connection time: ${connect_time}ms (excellent)"
        elif (( $(echo "$connect_time < 500" | bc -l 2>/dev/null || echo "0") )); then
            log_success "Database connection time: ${connect_time}ms (good)"
        else
            log_warn "Database connection time: ${connect_time}ms (may need connection pooling)"
        fi
    else
        log_warn "Database connection test failed"
    fi
    
    # Test through API if available
    local api_port
    api_port=$(vrooli scenario port ai-model-orchestra-controller API_PORT 2>/dev/null || echo "")
    
    if [ -n "$api_port" ]; then
        local base_url="http://localhost:${api_port}/api/v1"
        
        # Test endpoint that likely uses database
        local db_endpoint_url="${base_url}/models"
        local db_query_start=$(date +%s.%N)
        
        if timeout 20 curl -s "$db_endpoint_url" >/dev/null 2>&1; then
            local db_query_end=$(date +%s.%N)
            local db_query_time=$(echo "($db_query_end - $db_query_start) * 1000" | bc 2>/dev/null || echo "unknown")
            
            if (( $(echo "$db_query_time < 1000" | bc -l 2>/dev/null || echo "0") )); then
                log_success "Database query through API: ${db_query_time}ms (good)"
            elif (( $(echo "$db_query_time < 3000" | bc -l 2>/dev/null || echo "0") )); then
                log_success "Database query through API: ${db_query_time}ms (acceptable)"
            else
                log_warn "Database query through API: ${db_query_time}ms (consider indexing/caching)"
            fi
        else
            log_warn "Database query through API test failed"
        fi
    fi
    
    return 0
}

# Test AI model response performance
test_ai_model_performance() {
    log_info "Testing AI model response performance..."
    
    # Check if Ollama is available
    local ollama_port
    ollama_port=$(vrooli resource port ollama RESOURCE_PORTS_OLLAMA 2>/dev/null || echo "11434")
    
    if ! timeout 10 curl -s "http://localhost:${ollama_port}/api/tags" >/dev/null 2>&1; then
        log_warn "Ollama not available - skipping AI model performance tests"
        return 0
    fi
    
    local api_port
    api_port=$(vrooli scenario port ai-model-orchestra-controller API_PORT 2>/dev/null || echo "")
    
    if [ -z "$api_port" ]; then
        log_warn "Orchestrator API not available - testing Ollama directly"
        
        # Test direct Ollama performance
        local test_prompt='{"model":"llama2","prompt":"What is 2+2?","stream":false}'
        local ollama_start=$(date +%s.%N)
        
        if timeout 60 curl -s -X POST \
            -H "Content-Type: application/json" \
            -d "$test_prompt" \
            "http://localhost:${ollama_port}/api/generate" >/dev/null 2>&1; then
            
            local ollama_end=$(date +%s.%N)
            local ollama_time=$(echo "($ollama_end - $ollama_start) * 1000" | bc 2>/dev/null || echo "unknown")
            
            log_success "Direct Ollama response time: ${ollama_time}ms"
        else
            log_warn "Direct Ollama test failed (may need model download)"
        fi
        
        return 0
    fi
    
    # Test through orchestrator
    local base_url="http://localhost:${api_port}/api/v1"
    local route_url="${base_url}/route"
    
    # Test simple routing performance
    log_info "Testing AI routing performance..."
    local routing_request='{"prompt":"What is artificial intelligence?","task":"general","max_tokens":50}'
    local routing_start=$(date +%s.%N)
    
    if timeout 90 curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$routing_request" \
        "$route_url" >/dev/null 2>&1; then
        
        local routing_end=$(date +%s.%N)
        local routing_time=$(echo "($routing_end - $routing_start) * 1000" | bc 2>/dev/null || echo "unknown")
        
        if (( $(echo "$routing_time < 10000" | bc -l 2>/dev/null || echo "0") )); then
            log_success "AI routing response time: ${routing_time}ms (good)"
        elif (( $(echo "$routing_time < 30000" | bc -l 2>/dev/null || echo "0") )); then
            log_success "AI routing response time: ${routing_time}ms (acceptable)"
        else
            log_warn "AI routing response time: ${routing_time}ms (consider optimization)"
        fi
    else
        log_warn "AI routing performance test failed (may need model setup)"
    fi
    
    return 0
}

# Test system resource utilization
test_system_resources() {
    log_info "Testing system resource utilization..."
    
    # Test CPU usage during load
    log_info "Monitoring CPU usage during API load..."
    
    local api_port
    api_port=$(vrooli scenario port ai-model-orchestra-controller API_PORT 2>/dev/null || echo "")
    
    if [ -n "$api_port" ]; then
        local base_url="http://localhost:${api_port}/api/v1"
        local health_url="${base_url}/health"
        
        # Generate load and monitor CPU
        local cpu_samples=()
        local load_pids=()
        
        # Start background load
        for i in {1..5}; do
            (
                for j in {1..10}; do
                    timeout 5 curl -s "$health_url" >/dev/null 2>&1 || true
                    sleep 0.1
                done
            ) &
            load_pids+=($!)
        done
        
        # Monitor CPU usage during load
        for i in {1..10}; do
            local cpu_usage
            cpu_usage=$(top -bn1 | grep "Cpu(s)" | awk '{print $2}' | awk -F'%' '{print $1}' 2>/dev/null || echo "0")
            
            if [[ "$cpu_usage" =~ ^[0-9]+\.?[0-9]*$ ]]; then
                cpu_samples+=("$cpu_usage")
            fi
            
            sleep 1
        done
        
        # Wait for load generators to complete
        for pid in "${load_pids[@]}"; do
            wait "$pid" 2>/dev/null || true
        done
        
        # Calculate average CPU usage
        if [ ${#cpu_samples[@]} -gt 0 ]; then
            local total_cpu=0
            local max_cpu=0
            
            for cpu in "${cpu_samples[@]}"; do
                total_cpu=$(echo "$total_cpu + $cpu" | bc 2>/dev/null || echo "$total_cpu")
                if (( $(echo "$cpu > $max_cpu" | bc -l 2>/dev/null || echo "0") )); then
                    max_cpu="$cpu"
                fi
            done
            
            local avg_cpu=$(echo "scale=1; $total_cpu / ${#cpu_samples[@]}" | bc 2>/dev/null || echo "unknown")
            
            log_success "CPU usage during load - Average: ${avg_cpu}%, Peak: ${max_cpu}%"
            
            if (( $(echo "$max_cpu > 80" | bc -l 2>/dev/null || echo "0") )); then
                log_warn "High CPU usage detected - consider optimization or scaling"
            elif (( $(echo "$max_cpu > 50" | bc -l 2>/dev/null || echo "0") )); then
                log_info "Moderate CPU usage - acceptable for AI workloads"
            else
                log_success "Low CPU usage - system has capacity for more load"
            fi
        else
            log_warn "Could not collect CPU usage samples"
        fi
    fi
    
    # Test disk I/O if applicable
    if [ -d "$SCENARIO_ROOT/logs" ] || [ -d "$SCENARIO_ROOT/data" ]; then
        log_info "Checking disk I/O patterns..."
        
        # Simple disk usage check
        local disk_usage
        disk_usage=$(df "$SCENARIO_ROOT" | tail -1 | awk '{print $5}' | sed 's/%//' 2>/dev/null || echo "unknown")
        
        if [[ "$disk_usage" =~ ^[0-9]+$ ]]; then
            if [ "$disk_usage" -lt 80 ]; then
                log_success "Disk usage: ${disk_usage}% (healthy)"
            elif [ "$disk_usage" -lt 95 ]; then
                log_warn "Disk usage: ${disk_usage}% (monitor closely)"
            else
                log_error "Disk usage: ${disk_usage}% (critical - cleanup needed)"
            fi
        fi
    fi
    
    return 0
}

# Test performance under different load scenarios
test_load_scenarios() {
    log_info "Testing performance under different load scenarios..."
    
    local api_port
    api_port=$(vrooli scenario port ai-model-orchestra-controller API_PORT 2>/dev/null || echo "")
    
    if [ -z "$api_port" ]; then
        log_warn "Cannot test load scenarios - API not available"
        return 0
    fi
    
    local base_url="http://localhost:${api_port}/api/v1"
    
    # Scenario 1: Burst load (many requests quickly)
    log_info "Testing burst load scenario...")
    local burst_start=$(date +%s.%N)
    local burst_pids=()
    
    for i in {1..15}; do
        timeout 15 curl -s "${base_url}/health" >/dev/null 2>&1 &
        burst_pids+=($!)
    done
    
    local burst_completed=0
    for pid in "${burst_pids[@]}"; do
        if wait "$pid" 2>/dev/null; then
            ((burst_completed++))
        fi
    done
    
    local burst_end=$(date +%s.%N)
    local burst_time=$(echo "$burst_end - $burst_start" | bc 2>/dev/null || echo "unknown")
    local burst_rate=$(echo "scale=1; $burst_completed / $burst_time" | bc 2>/dev/null || echo "unknown")
    
    log_success "Burst load: ${burst_completed}/15 requests completed in ${burst_time}s (${burst_rate} req/s)"
    
    # Scenario 2: Sustained load (steady requests over time)
    log_info "Testing sustained load scenario..."
    local sustained_completed=0
    local sustained_start=$(date +%s)
    
    for i in {1..20}; do
        if timeout 10 curl -s "${base_url}/health" >/dev/null 2>&1; then
            ((sustained_completed++))
        fi
        sleep 0.5  # Sustained pace
    done
    
    local sustained_end=$(date +%s)
    local sustained_time=$((sustained_end - sustained_start))
    local sustained_rate=$(echo "scale=1; $sustained_completed / $sustained_time" | bc 2>/dev/null || echo "unknown")
    
    log_success "Sustained load: ${sustained_completed}/20 requests completed in ${sustained_time}s (${sustained_rate} req/s)"
    
    return 0
}

# Run all performance tests
echo "Starting performance validation tests..."

# Check if bc (calculator) is available for math operations
if ! command -v bc >/dev/null 2>&1; then
    log_warn "bc (calculator) not available - some timing calculations may be inaccurate"
fi

# Execute all tests (most are non-critical but provide valuable insights)
test_api_response_times || exit 1
test_concurrent_requests || exit 1  
test_memory_usage # Non-critical
test_database_performance # Non-critical
test_ai_model_performance # Non-critical but important
test_system_resources # Non-critical
test_load_scenarios # Non-critical

echo ""
log_success "All performance tests completed!"
echo "✅ Performance phase completed successfully"