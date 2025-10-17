#!/usr/bin/env bash
################################################################################
# Judge0 Test Library - v2.0 Contract Compliant
# 
# Test implementation functions for Judge0 resource
################################################################################

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"

# Export test functions
judge0::test::smoke() {
    log "Running Judge0 smoke tests..."
    "${RESOURCE_DIR}/test/run-tests.sh" smoke
}

judge0::test::integration() {
    log "Running Judge0 integration tests..."
    "${RESOURCE_DIR}/test/run-tests.sh" integration
}

judge0::test::unit() {
    log "Running Judge0 unit tests..."
    "${RESOURCE_DIR}/test/run-tests.sh" unit
}

judge0::test::all() {
    log "Running all Judge0 tests..."
    "${RESOURCE_DIR}/test/run-tests.sh" all
}

# Health check with enhanced diagnostics
judge0::test::health() {
    local api_url="http://localhost:${JUDGE0_PORT:-2358}"
    
    log "Testing Judge0 health..."
    
    # Check API endpoint
    if timeout 5 curl -sf "${api_url}/system_info" &>/dev/null; then
        log "✅ API endpoint is responding"
        
        # Check response time
        local start_time=$(date +%s%N)
        timeout 1 curl -sf "${api_url}/system_info" &>/dev/null
        local end_time=$(date +%s%N)
        local response_time=$(( (end_time - start_time) / 1000000 ))
        
        if [[ $response_time -lt 500 ]]; then
            log "✅ Response time: ${response_time}ms (good)"
        else
            log "⚠️  Response time: ${response_time}ms (slow)"
        fi
        
        # Check workers
        local workers=$(docker ps --filter "name=judge0-workers" --format "{{.Names}}" | wc -l)
        log "ℹ️  Active workers: $workers"
        
        # Check database
        if docker ps --filter "name=judge0-server-db" --format "{{.Names}}" | grep -q judge0; then
            log "✅ Database is running"
        else
            log "❌ Database is not running"
            return 1
        fi
        
        # Check Redis
        if docker ps --filter "name=judge0-server-redis" --format "{{.Names}}" | grep -q judge0; then
            log "✅ Redis cache is running"
        else
            log "❌ Redis cache is not running"
            return 1
        fi
        
        return 0
    else
        log "❌ API endpoint not responding"
        return 1
    fi
}

# Performance benchmark function
judge0::test::benchmark() {
    local api_url="http://localhost:${JUDGE0_PORT:-2358}"
    
    log "Running Judge0 performance benchmarks..."
    
    # Benchmark languages
    local languages=("92:Python" "93:JavaScript" "95:Go")
    
    for lang_spec in "${languages[@]}"; do
        IFS=':' read -r lang_id lang_name <<< "$lang_spec"
        
        log "Benchmarking $lang_name (ID: $lang_id)..."
        
        # Simple execution benchmark
        local start_time=$(date +%s%N)
        local result=$(timeout 10 curl -sf -X POST "${api_url}/submissions?wait=true" \
            -H "Content-Type: application/json" \
            -d "{
                \"source_code\": \"print(42)\",
                \"language_id\": $lang_id
            }" 2>/dev/null || echo "FAILED")
        local end_time=$(date +%s%N)
        
        if [[ "$result" != "FAILED" ]]; then
            local exec_time=$(( (end_time - start_time) / 1000000 ))
            log "  $lang_name execution: ${exec_time}ms"
        else
            log "  $lang_name: Failed to execute"
        fi
    done
    
    # Concurrent execution benchmark
    log "Testing concurrent execution (5 submissions)..."
    local concurrent_start=$(date +%s%N)
    
    for i in {1..5}; do
        timeout 10 curl -sf -X POST "${api_url}/submissions?wait=false" \
            -H "Content-Type: application/json" \
            -d "{\"source_code\": \"print($i)\", \"language_id\": 92}" &>/dev/null &
    done
    wait
    
    local concurrent_end=$(date +%s%N)
    local concurrent_time=$(( (concurrent_end - concurrent_start) / 1000000 ))
    log "  Concurrent execution time: ${concurrent_time}ms"
    
    # Memory usage check
    local memory_usage=$(docker stats --no-stream --format "{{.MemUsage}}" vrooli-judge0-server 2>/dev/null | head -1)
    log "  Server memory usage: $memory_usage"
}

# Validation function for PRD requirements
judge0::test::validate_requirements() {
    log "Validating PRD requirements..."
    
    local api_url="http://localhost:${JUDGE0_PORT:-2358}"
    local passed=0
    local failed=0
    
    # P0: Secure Code Execution
    if timeout 10 curl -sf -X POST "${api_url}/submissions?wait=true" \
        -H "Content-Type: application/json" \
        -d '{"source_code": "print(1)", "language_id": 92}' &>/dev/null; then
        log "✅ P0: Secure code execution working"
        ((passed++))
    else
        log "❌ P0: Secure code execution failed"
        ((failed++))
    fi
    
    # P0: Multi-Language Support
    local languages=$(timeout 5 curl -sf "${api_url}/languages" 2>/dev/null || echo "[]")
    local lang_count=$(echo "$languages" | python3 -c "import sys, json; print(len(json.load(sys.stdin)))" 2>/dev/null || echo "0")
    if [[ $lang_count -gt 20 ]]; then
        log "✅ P0: Multi-language support ($lang_count languages)"
        ((passed++))
    else
        log "❌ P0: Insufficient languages ($lang_count/20+)"
        ((failed++))
    fi
    
    # P0: Health Monitoring
    if judge0::test::health &>/dev/null; then
        log "✅ P0: Health monitoring functional"
        ((passed++))
    else
        log "❌ P0: Health monitoring failed"
        ((failed++))
    fi
    
    # P0: Performance Limits
    local limit_test=$(timeout 10 curl -sf -X POST "${api_url}/submissions?wait=true" \
        -H "Content-Type: application/json" \
        -d '{
            "source_code": "while True: pass",
            "language_id": 92,
            "cpu_time_limit": 1
        }' 2>/dev/null || echo "FAILED")
    
    if [[ "$limit_test" != "FAILED" ]]; then
        local status_id=$(echo "$limit_test" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', {}).get('id', 0))" 2>/dev/null || echo "0")
        if [[ $status_id -eq 5 ]]; then
            log "✅ P0: Performance limits enforced"
            ((passed++))
        else
            log "❌ P0: Performance limits not working"
            ((failed++))
        fi
    else
        log "❌ P0: Performance limit test failed"
        ((failed++))
    fi
    
    log "Requirements validation: $passed passed, $failed failed"
    return $failed
}