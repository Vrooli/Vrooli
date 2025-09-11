#!/usr/bin/env bash
################################################################################
# Judge0 Performance Benchmark Library
# 
# Comprehensive benchmarking for all supported languages
################################################################################

set -euo pipefail

# Define log function if not available
if ! declare -f log &>/dev/null; then
    log() {
        echo "$@"
    }
fi

# Benchmark test programs for different languages
declare -A BENCHMARK_PROGRAMS=(
    ["92"]='# Python
import time
start = time.time()
result = sum(range(1000000))
print(f"Result: {result}")
print(f"Time: {time.time() - start:.3f}s")'
    
    ["93"]='// JavaScript
const start = Date.now();
let result = 0;
for (let i = 0; i < 1000000; i++) {
    result += i;
}
console.log(`Result: ${result}`);
console.log(`Time: ${(Date.now() - start) / 1000}s`);'
    
    ["95"]='// Go
package main
import (
    "fmt"
    "time"
)
func main() {
    start := time.Now()
    result := 0
    for i := 0; i < 1000000; i++ {
        result += i
    }
    fmt.Printf("Result: %d\n", result)
    fmt.Printf("Time: %v\n", time.Since(start))
}'
    
    ["73"]='// Rust
use std::time::Instant;
fn main() {
    let start = Instant::now();
    let mut result: i64 = 0;
    for i in 0..1000000 {
        result += i;
    }
    println!("Result: {}", result);
    println!("Time: {:?}", start.elapsed());
}'
    
    ["91"]='// Java
public class Main {
    public static void main(String[] args) {
        long start = System.currentTimeMillis();
        long result = 0;
        for (int i = 0; i < 1000000; i++) {
            result += i;
        }
        System.out.println("Result: " + result);
        System.out.println("Time: " + (System.currentTimeMillis() - start) / 1000.0 + "s");
    }
}'
)

# Language names for reporting
declare -A LANGUAGE_NAMES=(
    ["92"]="Python 3"
    ["93"]="JavaScript (Node.js)"
    ["95"]="Go"
    ["73"]="Rust"
    ["91"]="Java"
    ["105"]="C++"
    ["72"]="Ruby"
    ["68"]="PHP"
    ["51"]="C#"
    ["94"]="TypeScript"
)

# Run comprehensive benchmark
judge0::benchmark::run_all() {
    local api_url="http://localhost:${JUDGE0_PORT:-2358}"
    local output_file="${1:-/tmp/judge0_benchmark_$(date +%Y%m%d_%H%M%S).json}"
    
    log "Starting comprehensive Judge0 benchmark..."
    log "Results will be saved to: $output_file"
    
    # Initialize results
    local results='{"benchmark_date":"'$(date -Iseconds)'","languages":{},"summary":{}}'
    
    # Get available languages
    local languages=$(timeout 5 curl -sf "${api_url}/languages" 2>/dev/null || echo "[]")
    
    # Benchmark each language with test program
    for lang_id in "${!BENCHMARK_PROGRAMS[@]}"; do
        local lang_name="${LANGUAGE_NAMES[$lang_id]:-Language $lang_id}"
        
        log "Benchmarking $lang_name (ID: $lang_id)..."
        
        local benchmark_result=$(judge0::benchmark::single_language "$lang_id" "${BENCHMARK_PROGRAMS[$lang_id]}")
        
        if [[ -n "$benchmark_result" ]]; then
            results=$(echo "$results" | jq ".languages.\"$lang_id\" = $benchmark_result")
            log "  ✅ $lang_name benchmark complete"
        else
            log "  ❌ $lang_name benchmark failed"
        fi
    done
    
    # Run concurrent submission benchmark
    log "Running concurrent submission benchmark..."
    local concurrent_result=$(judge0::benchmark::concurrent_submissions)
    results=$(echo "$results" | jq ".concurrent = $concurrent_result")
    
    # Calculate summary statistics
    results=$(judge0::benchmark::calculate_summary "$results")
    
    # Save results
    echo "$results" | jq '.' > "$output_file"
    
    # Display summary
    judge0::benchmark::display_summary "$results"
    
    log "Benchmark complete. Full results saved to: $output_file"
}

# Benchmark a single language
judge0::benchmark::single_language() {
    local lang_id="$1"
    local source_code="$2"
    local api_url="http://localhost:${JUDGE0_PORT:-2358}"
    
    local iterations=5
    local total_time=0
    local successful=0
    local times=()
    
    for i in $(seq 1 $iterations); do
        local start_time=$(date +%s%N)
        
        local result=$(timeout 30 curl -sf -X POST "${api_url}/submissions?wait=true" \
            -H "Content-Type: application/json" \
            -d "$(jq -n --arg code "$source_code" --arg lang "$lang_id" \
                '{source_code: $code, language_id: ($lang | tonumber)}')" 2>/dev/null || echo "FAILED")
        
        local end_time=$(date +%s%N)
        local exec_time=$(( (end_time - start_time) / 1000000 ))
        
        if [[ "$result" != "FAILED" ]]; then
            local status_id=$(echo "$result" | jq -r '.status.id' 2>/dev/null || echo "0")
            local cpu_time=$(echo "$result" | jq -r '.time // "0"' 2>/dev/null || echo "0")
            local memory=$(echo "$result" | jq -r '.memory // "0"' 2>/dev/null || echo "0")
            
            if [[ $status_id -eq 3 ]]; then
                times+=("$exec_time")
                total_time=$((total_time + exec_time))
                successful=$((successful + 1))
            fi
        fi
    done
    
    if [[ $successful -gt 0 ]]; then
        local avg_time=$((total_time / successful))
        
        # Calculate min/max
        local min_time=$(printf '%s\n' "${times[@]}" | sort -n | head -1)
        local max_time=$(printf '%s\n' "${times[@]}" | sort -n | tail -1)
        
        # Return JSON result
        jq -n \
            --arg name "${LANGUAGE_NAMES[$lang_id]:-Language $lang_id}" \
            --arg avg "$avg_time" \
            --arg min "$min_time" \
            --arg max "$max_time" \
            --arg success "$successful" \
            --arg total "$iterations" \
            '{
                name: $name,
                avg_time_ms: ($avg | tonumber),
                min_time_ms: ($min | tonumber),
                max_time_ms: ($max | tonumber),
                success_rate: (($success | tonumber) / ($total | tonumber) * 100),
                iterations: ($total | tonumber)
            }'
    else
        echo ""
    fi
}

# Benchmark concurrent submissions
judge0::benchmark::concurrent_submissions() {
    local api_url="http://localhost:${JUDGE0_PORT:-2358}"
    local concurrent_count=10
    
    log "  Submitting $concurrent_count concurrent tasks..."
    
    local start_time=$(date +%s%N)
    
    # Submit concurrent jobs
    local pids=()
    for i in $(seq 1 $concurrent_count); do
        (
            timeout 30 curl -sf -X POST "${api_url}/submissions?wait=true" \
                -H "Content-Type: application/json" \
                -d "{\"source_code\": \"print($i)\", \"language_id\": 92}" &>/dev/null
        ) &
        pids+=($!)
    done
    
    # Wait for all to complete
    local completed=0
    for pid in "${pids[@]}"; do
        if wait $pid; then
            completed=$((completed + 1))
        fi
    done
    
    local end_time=$(date +%s%N)
    local total_time=$(( (end_time - start_time) / 1000000 ))
    
    # Return JSON result
    jq -n \
        --arg total "$concurrent_count" \
        --arg completed "$completed" \
        --arg time "$total_time" \
        '{
            submissions: ($total | tonumber),
            completed: ($completed | tonumber),
            total_time_ms: ($time | tonumber),
            avg_time_ms: (($time | tonumber) / ($total | tonumber)),
            throughput_per_sec: (($completed | tonumber) * 1000 / ($time | tonumber))
        }'
}

# Calculate summary statistics
judge0::benchmark::calculate_summary() {
    local results="$1"
    
    # Calculate averages across all languages
    local avg_time=$(echo "$results" | jq '[.languages[].avg_time_ms] | add / length')
    local min_time=$(echo "$results" | jq '[.languages[].min_time_ms] | min')
    local max_time=$(echo "$results" | jq '[.languages[].max_time_ms] | max')
    local lang_count=$(echo "$results" | jq '.languages | length')
    
    # Add summary
    echo "$results" | jq \
        --arg avg "$avg_time" \
        --arg min "$min_time" \
        --arg max "$max_time" \
        --arg count "$lang_count" \
        '.summary = {
            languages_tested: ($count | tonumber),
            avg_execution_ms: ($avg | tonumber),
            min_execution_ms: ($min | tonumber),
            max_execution_ms: ($max | tonumber),
            concurrent_throughput: .concurrent.throughput_per_sec
        }'
}

# Display benchmark summary
judge0::benchmark::display_summary() {
    local results="$1"
    
    echo
    echo "═══════════════════════════════════════════════════════"
    echo "                Judge0 Benchmark Results                "
    echo "═══════════════════════════════════════════════════════"
    echo
    
    # Language results
    echo "Language Performance (1M iterations sum):"
    echo "─────────────────────────────────────────────────────"
    echo "$results" | jq -r '.languages | to_entries[] | 
        "  \(.value.name): \(.value.avg_time_ms)ms avg (\(.value.success_rate)% success)"'
    echo
    
    # Concurrent performance
    echo "Concurrent Execution:"
    echo "─────────────────────────────────────────────────────"
    echo "$results" | jq -r '.concurrent | 
        "  Submissions: \(.submissions)\n  Completed: \(.completed)\n  Total Time: \(.total_time_ms)ms\n  Throughput: \(.throughput_per_sec | round) req/s"'
    echo
    
    # Summary
    echo "Overall Summary:"
    echo "─────────────────────────────────────────────────────"
    echo "$results" | jq -r '.summary | 
        "  Languages Tested: \(.languages_tested)\n  Avg Execution: \(.avg_execution_ms | round)ms\n  Min Execution: \(.min_execution_ms)ms\n  Max Execution: \(.max_execution_ms)ms"'
    echo "═══════════════════════════════════════════════════════"
}

# Quick performance test
judge0::benchmark::quick() {
    local api_url="http://localhost:${JUDGE0_PORT:-2358}"
    
    log "Running quick performance test..."
    
    # Test Python execution
    local start_time=$(date +%s%N)
    local result=$(timeout 10 curl -sf -X POST "${api_url}/submissions?wait=true" \
        -H "Content-Type: application/json" \
        -d '{"source_code": "print(sum(range(100000)))", "language_id": 92}' 2>/dev/null || echo "FAILED")
    local end_time=$(date +%s%N)
    
    if [[ "$result" != "FAILED" ]]; then
        local exec_time=$(( (end_time - start_time) / 1000000 ))
        local status_id=$(echo "$result" | jq -r '.status.id' 2>/dev/null || echo "0")
        
        if [[ $status_id -eq 3 ]]; then
            log "✅ Quick test passed: ${exec_time}ms"
            
            if [[ $exec_time -lt 2000 ]]; then
                log "  Performance: Excellent (<2s)"
            elif [[ $exec_time -lt 5000 ]]; then
                log "  Performance: Good (<5s)"
            else
                log "  Performance: Needs optimization (>${exec_time}ms)"
            fi
        else
            log "❌ Quick test failed: status $status_id"
        fi
    else
        log "❌ Quick test failed: no response"
    fi
}