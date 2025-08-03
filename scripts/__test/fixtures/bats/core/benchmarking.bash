#!/usr/bin/env bash
# Performance Benchmarking System
# Advanced performance measurement and analysis for BATS tests

# Prevent duplicate loading
if [[ "${BENCHMARKING_SYSTEM_LOADED:-}" == "true" ]]; then
    return 0
fi
export BENCHMARKING_SYSTEM_LOADED="true"

# Benchmarking configuration
export BENCHMARK_ENABLED="${BENCHMARK_ENABLED:-false}"
export BENCHMARK_OUTPUT_DIR="${BENCHMARK_OUTPUT_DIR:-${TEST_TMPDIR}/benchmarks}"
export BENCHMARK_PRECISION="${BENCHMARK_PRECISION:-6}"  # Microseconds
export BENCHMARK_WARMUP_ITERATIONS="${BENCHMARK_WARMUP_ITERATIONS:-3}"
export BENCHMARK_DEFAULT_ITERATIONS="${BENCHMARK_DEFAULT_ITERATIONS:-10}"

# Benchmark state storage
declare -A BENCHMARK_TIMERS
declare -A BENCHMARK_RESULTS
declare -A BENCHMARK_METADATA

#######################################
# Initialize benchmarking system
#######################################
benchmark::init() {
    if [[ "$BENCHMARK_ENABLED" != "true" ]]; then
        return 0
    fi
    
    # Create benchmark output directory
    mkdir -p "$BENCHMARK_OUTPUT_DIR"/{timings,summaries,reports}
    
    # Initialize benchmark log
    echo "timestamp,test_name,operation,duration_us,iteration,metadata" > "$BENCHMARK_OUTPUT_DIR/benchmark.csv"
    
    # Reset benchmark state
    BENCHMARK_TIMERS=()
    BENCHMARK_RESULTS=()
    BENCHMARK_METADATA=()
    
    echo "[BENCHMARK] Benchmarking system initialized"
}

#######################################
# Start a benchmark timer
# Arguments: $1 - timer name
#######################################
benchmark::start() {
    if [[ "$BENCHMARK_ENABLED" != "true" ]]; then
        return 0
    fi
    
    local timer_name="$1"
    local start_time=$(date +%s%N)
    
    BENCHMARK_TIMERS["$timer_name"]="$start_time"
}

#######################################
# Stop a benchmark timer and record result
# Arguments: $1 - timer name, $2 - metadata (optional)
#######################################
benchmark::stop() {
    if [[ "$BENCHMARK_ENABLED" != "true" ]]; then
        return 0
    fi
    
    local timer_name="$1"
    local metadata="${2:-}"
    local end_time=$(date +%s%N)
    
    local start_time="${BENCHMARK_TIMERS[$timer_name]:-$end_time}"
    local duration_ns=$((end_time - start_time))
    local duration_us=$((duration_ns / 1000))
    
    # Store result
    BENCHMARK_RESULTS["$timer_name"]="$duration_us"
    BENCHMARK_METADATA["$timer_name"]="$metadata"
    
    # Log to CSV
    echo "$(date -Iseconds),${BATS_TEST_NAME:-unknown},$timer_name,$duration_us,1,$metadata" >> "$BENCHMARK_OUTPUT_DIR/benchmark.csv"
    
    # Clear timer
    unset BENCHMARK_TIMERS["$timer_name"]
    
    echo "$duration_us"  # Return duration for use in tests
}

#######################################
# Benchmark a command or function multiple times
# Arguments: $1 - benchmark name, $2 - iterations, $@ - command to benchmark
#######################################
benchmark::measure() {
    if [[ "$BENCHMARK_ENABLED" != "true" ]]; then
        # Just run the command once if benchmarking is disabled
        shift 2
        "$@" >/dev/null 2>&1
        return $?
    fi
    
    local bench_name="$1"
    local iterations="$2"
    shift 2
    local command=("$@")
    
    local results=()
    local successful_runs=0
    
    echo "[BENCHMARK] Running benchmark '$bench_name' for $iterations iterations"
    
    # Warmup runs
    for i in $(seq 1 "$BENCHMARK_WARMUP_ITERATIONS"); do
        "${command[@]}" >/dev/null 2>&1
    done
    
    # Actual benchmark runs
    for i in $(seq 1 "$iterations"); do
        local start_time=$(date +%s%N)
        
        if "${command[@]}" >/dev/null 2>&1; then
            local end_time=$(date +%s%N)
            local duration_us=$(( (end_time - start_time) / 1000 ))
            
            results+=("$duration_us")
            successful_runs=$((successful_runs + 1))
            
            # Log individual result
            echo "$(date -Iseconds),${BATS_TEST_NAME:-unknown},$bench_name,$duration_us,$i,success" >> "$BENCHMARK_OUTPUT_DIR/benchmark.csv"
        else
            echo "$(date -Iseconds),${BATS_TEST_NAME:-unknown},$bench_name,0,$i,failure" >> "$BENCHMARK_OUTPUT_DIR/benchmark.csv"
        fi
    done
    
    # Calculate statistics
    if [[ $successful_runs -gt 0 ]]; then
        benchmark::calculate_stats "$bench_name" "${results[@]}"
    else
        echo "[BENCHMARK] All runs failed for benchmark '$bench_name'"
        return 1
    fi
}

#######################################
# Calculate and store benchmark statistics
# Arguments: $1 - benchmark name, $@ - duration values in microseconds
#######################################
benchmark::calculate_stats() {
    local bench_name="$1"
    shift
    local durations=("$@")
    
    if [[ ${#durations[@]} -eq 0 ]]; then
        return 1
    fi
    
    # Sort durations for percentile calculations
    local sorted_durations=($(printf '%s\n' "${durations[@]}" | sort -n))
    local count=${#sorted_durations[@]}
    
    # Calculate basic statistics
    local sum=0
    local min="${sorted_durations[0]}"
    local max="${sorted_durations[-1]}"
    
    for duration in "${durations[@]}"; do
        sum=$((sum + duration))
    done
    
    local mean=$((sum / count))
    
    # Calculate percentiles
    local p50_index=$(( count * 50 / 100 ))
    local p95_index=$(( count * 95 / 100 ))
    local p99_index=$(( count * 99 / 100 ))
    
    local p50="${sorted_durations[$p50_index]}"
    local p95="${sorted_durations[$p95_index]}"
    local p99="${sorted_durations[$p99_index]}"
    
    # Calculate standard deviation
    local variance_sum=0
    for duration in "${durations[@]}"; do
        local diff=$((duration - mean))
        variance_sum=$((variance_sum + diff * diff))
    done
    local variance=$((variance_sum / count))
    local stddev=$(echo "scale=2; sqrt($variance)" | bc -l 2>/dev/null || echo "0")
    
    # Store statistics
    local stats_file="$BENCHMARK_OUTPUT_DIR/summaries/${bench_name}_stats.json"
    cat > "$stats_file" << EOF
{
    "benchmark_name": "$bench_name",
    "test_name": "${BATS_TEST_NAME:-unknown}",
    "timestamp": "$(date -Iseconds)",
    "iterations": $count,
    "statistics": {
        "min_us": $min,
        "max_us": $max,
        "mean_us": $mean,
        "median_us": $p50,
        "p95_us": $p95,
        "p99_us": $p99,
        "stddev_us": $stddev,
        "range_us": $((max - min))
    },
    "raw_data": $(printf '%s\n' "${durations[@]}" | jq -R 'tonumber' | jq -s '.')
}
EOF

    echo "[BENCHMARK] Statistics calculated for '$bench_name': mean=${mean}μs, p95=${p95}μs"
}

#######################################
# Compare two benchmark results
# Arguments: $1 - baseline benchmark name, $2 - current benchmark name, $3 - tolerance percentage
#######################################
benchmark::compare() {
    local baseline_name="$1"
    local current_name="$2"
    local tolerance="${3:-10}"  # 10% default tolerance
    
    local baseline_file="$BENCHMARK_OUTPUT_DIR/summaries/${baseline_name}_stats.json"
    local current_file="$BENCHMARK_OUTPUT_DIR/summaries/${current_name}_stats.json"
    
    if [[ ! -f "$baseline_file" || ! -f "$current_file" ]]; then
        echo "[BENCHMARK] Cannot compare: missing benchmark files"
        return 1
    fi
    
    local baseline_mean=$(jq -r '.statistics.mean_us' "$baseline_file")
    local current_mean=$(jq -r '.statistics.mean_us' "$current_file")
    
    local difference=$((current_mean - baseline_mean))
    local percentage_change=$(echo "scale=2; $difference * 100 / $baseline_mean" | bc -l)
    
    # Generate comparison report
    local comparison_file="$BENCHMARK_OUTPUT_DIR/reports/comparison_${baseline_name}_vs_${current_name}.json"
    cat > "$comparison_file" << EOF
{
    "comparison": {
        "baseline": "$baseline_name",
        "current": "$current_name",
        "timestamp": "$(date -Iseconds)"
    },
    "results": {
        "baseline_mean_us": $baseline_mean,
        "current_mean_us": $current_mean,
        "difference_us": $difference,
        "percentage_change": $percentage_change,
        "tolerance_percent": $tolerance
    },
    "verdict": "$(benchmark::determine_verdict "$percentage_change" "$tolerance")"
}
EOF

    # Determine if change is within tolerance
    local abs_change=$(echo "scale=2; if ($percentage_change < 0) -$percentage_change else $percentage_change" | bc -l)
    local within_tolerance=$(echo "$abs_change <= $tolerance" | bc -l)
    
    if [[ $within_tolerance -eq 1 ]]; then
        echo "[BENCHMARK] Performance within tolerance: ${percentage_change}% change"
        return 0
    else
        echo "[BENCHMARK] Performance outside tolerance: ${percentage_change}% change (limit: ±${tolerance}%)"
        return 1
    fi
}

#######################################
# Determine performance verdict based on percentage change
# Arguments: $1 - percentage change, $2 - tolerance
#######################################
benchmark::determine_verdict() {
    local percentage_change="$1"
    local tolerance="$2"
    
    local abs_change=$(echo "scale=2; if ($percentage_change < 0) -$percentage_change else $percentage_change" | bc -l)
    local within_tolerance=$(echo "$abs_change <= $tolerance" | bc -l)
    
    if [[ $within_tolerance -eq 1 ]]; then
        echo "PASS"
    elif [[ $(echo "$percentage_change > 0" | bc -l) -eq 1 ]]; then
        echo "REGRESSION"
    else
        echo "IMPROVEMENT"
    fi
}

#######################################
# Generate comprehensive benchmark report
#######################################
benchmark::generate_report() {
    if [[ "$BENCHMARK_ENABLED" != "true" ]]; then
        return 0
    fi
    
    local report_file="$BENCHMARK_OUTPUT_DIR/reports/benchmark_report_$(date +%s).json"
    
    # Collect all benchmark summaries
    local summaries=()
    for summary_file in "$BENCHMARK_OUTPUT_DIR/summaries"/*.json; do
        if [[ -f "$summary_file" ]]; then
            summaries+=("$(basename "$summary_file" .json)")
        fi
    done
    
    # Calculate overall statistics
    local total_benchmarks=${#summaries[@]}
    local total_iterations=0
    local avg_mean_us=0
    
    if [[ $total_benchmarks -gt 0 ]]; then
        for summary in "${summaries[@]}"; do
            local summary_file="$BENCHMARK_OUTPUT_DIR/summaries/${summary}.json"
            local iterations=$(jq -r '.iterations' "$summary_file")
            local mean_us=$(jq -r '.statistics.mean_us' "$summary_file")
            
            total_iterations=$((total_iterations + iterations))
            avg_mean_us=$((avg_mean_us + mean_us))
        done
        
        avg_mean_us=$((avg_mean_us / total_benchmarks))
    fi
    
    # Generate comprehensive report
    cat > "$report_file" << EOF
{
    "report": {
        "generated_at": "$(date -Iseconds)",
        "test_session": "${BATS_TEST_NAME:-unknown}",
        "benchmark_system_version": "1.0.0"
    },
    "summary": {
        "total_benchmarks": $total_benchmarks,
        "total_iterations": $total_iterations,
        "average_mean_duration_us": $avg_mean_us
    },
    "benchmarks": $(
        echo "["
        local first=true
        for summary in "${summaries[@]}"; do
            if [[ $first == true ]]; then
                first=false
            else
                echo ","
            fi
            cat "$BENCHMARK_OUTPUT_DIR/summaries/${summary}.json"
        done
        echo "]"
    ),
    "configuration": {
        "warmup_iterations": $BENCHMARK_WARMUP_ITERATIONS,
        "default_iterations": $BENCHMARK_DEFAULT_ITERATIONS,
        "precision": "$BENCHMARK_PRECISION microseconds"
    }
}
EOF

    echo "[BENCHMARK] Comprehensive report generated: $report_file"
}

#######################################
# Benchmark assertion helpers
#######################################

# Assert that operation completes within time limit
benchmark::assert_time_limit() {
    local operation="$1"
    local time_limit_us="$2"
    shift 2
    local command=("$@")
    
    benchmark::start "$operation"
    "${command[@]}"
    local actual_time=$(benchmark::stop "$operation")
    
    if [[ $actual_time -gt $time_limit_us ]]; then
        echo "Operation '$operation' exceeded time limit:" >&2
        echo "  Actual: ${actual_time}μs" >&2
        echo "  Limit: ${time_limit_us}μs" >&2
        return 1
    fi
    
    return 0
}

# Assert that operation performance is within tolerance of baseline
benchmark::assert_performance_regression() {
    local operation="$1"
    local baseline_us="$2"
    local tolerance_percent="${3:-20}"
    shift 3
    local command=("$@")
    
    benchmark::start "$operation"
    "${command[@]}"
    local actual_time=$(benchmark::stop "$operation")
    
    local difference=$((actual_time - baseline_us))
    local percentage_change=$(echo "scale=2; $difference * 100 / $baseline_us" | bc -l)
    local abs_change=$(echo "scale=2; if ($percentage_change < 0) -$percentage_change else $percentage_change" | bc -l)
    
    local within_tolerance=$(echo "$abs_change <= $tolerance_percent" | bc -l)
    
    if [[ $within_tolerance -eq 0 ]]; then
        echo "Performance regression detected for '$operation':" >&2
        echo "  Baseline: ${baseline_us}μs" >&2
        echo "  Actual: ${actual_time}μs" >&2
        echo "  Change: ${percentage_change}%" >&2
        echo "  Tolerance: ±${tolerance_percent}%" >&2
        return 1
    fi
    
    return 0
}

# Assert that operation throughput meets minimum requirement
benchmark::assert_throughput() {
    local operation="$1"
    local min_ops_per_second="$2"
    local iterations="${3:-$BENCHMARK_DEFAULT_ITERATIONS}"
    shift 3
    local command=("$@")
    
    local start_time=$(date +%s%N)
    
    for i in $(seq 1 "$iterations"); do
        "${command[@]}" >/dev/null 2>&1
    done
    
    local end_time=$(date +%s%N)
    local total_duration_us=$(( (end_time - start_time) / 1000 ))
    local ops_per_second=$(echo "scale=2; $iterations * 1000000 / $total_duration_us" | bc -l)
    
    local meets_requirement=$(echo "$ops_per_second >= $min_ops_per_second" | bc -l)
    
    if [[ $meets_requirement -eq 0 ]]; then
        echo "Throughput requirement not met for '$operation':" >&2
        echo "  Actual: ${ops_per_second} ops/sec" >&2
        echo "  Required: ${min_ops_per_second} ops/sec" >&2
        return 1
    fi
    
    echo "[BENCHMARK] Throughput requirement met: ${ops_per_second} ops/sec"
    return 0
}

#######################################
# Cleanup benchmarking resources
#######################################
benchmark::cleanup() {
    if [[ "$BENCHMARK_ENABLED" == "true" ]]; then
        benchmark::generate_report
    fi
    
    # Clear benchmark state
    BENCHMARK_TIMERS=()
    BENCHMARK_RESULTS=()
    BENCHMARK_METADATA=()
}

#######################################
# Export benchmarking functions
#######################################
export -f benchmark::init
export -f benchmark::start
export -f benchmark::stop
export -f benchmark::measure
export -f benchmark::calculate_stats
export -f benchmark::compare
export -f benchmark::generate_report
export -f benchmark::assert_time_limit
export -f benchmark::assert_performance_regression
export -f benchmark::assert_throughput
export -f benchmark::cleanup

echo "[BENCHMARK] Performance benchmarking system loaded"