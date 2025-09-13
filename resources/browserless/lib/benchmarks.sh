#!/usr/bin/env bash
#
# Browserless Performance Benchmarks
#
# This module provides performance benchmarking for browser operations
# to track performance over time and identify bottlenecks.
#

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
BROWSERLESS_LIB_DIR="${APP_ROOT}/resources/browserless/lib"

# Source dependencies
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh" || { echo "FATAL: Failed to load variable definitions" >&2; exit 1; }
# shellcheck disable=SC1091
source "${var_LOG_FILE}" || { echo "FATAL: Failed to load logging library" >&2; exit 1; }
# shellcheck disable=SC1091
source "${BROWSERLESS_LIB_DIR}/common.sh"
# shellcheck disable=SC1091
source "${BROWSERLESS_LIB_DIR}/browser-ops.sh"
# shellcheck disable=SC1091
source "${BROWSERLESS_LIB_DIR}/workflow-ops.sh"

# Benchmark configuration
BENCHMARK_ITERATIONS="${BROWSERLESS_BENCHMARK_ITERATIONS:-10}"
BENCHMARK_WARMUP="${BROWSERLESS_BENCHMARK_WARMUP:-2}"
BENCHMARK_OUTPUT_DIR="${BROWSERLESS_DATA_DIR}/benchmarks"
BENCHMARK_TIMEOUT="${BROWSERLESS_BENCHMARK_TIMEOUT:-30}"

#######################################
# Initialize benchmarking
#######################################
benchmark::init() {
    mkdir -p "$BENCHMARK_OUTPUT_DIR"
    
    # Check browserless is running
    if ! is_running; then
        log::error "Browserless is not running. Start it first."
        return 1
    fi
}

#######################################
# Measure operation time
# Arguments:
#   $1 - Operation name
#   $2 - Function to execute
#   $@ - Additional arguments for function
# Returns:
#   Execution time in milliseconds
#######################################
benchmark::measure() {
    local operation="${1:?Operation name required}"
    local function="${2:?Function required}"
    shift 2
    
    local start_time=$(date +%s%3N)
    
    # Execute function with timeout
    if timeout "$BENCHMARK_TIMEOUT" "$function" "$@" >/dev/null 2>&1; then
        local end_time=$(date +%s%3N)
        local duration=$((end_time - start_time))
        echo "$duration"
    else
        echo "-1"  # Error or timeout
    fi
}

#######################################
# Run benchmark suite for an operation
# Arguments:
#   $1 - Operation name
#   $2 - Function to benchmark
#   $@ - Additional arguments
# Returns:
#   JSON with benchmark results
#######################################
benchmark::run_suite() {
    local operation="${1:?Operation name required}"
    local function="${2:?Function required}"
    shift 2
    
    log::info "Benchmarking: $operation"
    
    local times=()
    local errors=0
    
    # Warmup runs
    log::debug "Warming up ($BENCHMARK_WARMUP iterations)..."
    for ((i=1; i<=BENCHMARK_WARMUP; i++)); do
        benchmark::measure "$operation" "$function" "$@" >/dev/null
    done
    
    # Actual benchmark runs
    log::debug "Running benchmark ($BENCHMARK_ITERATIONS iterations)..."
    for ((i=1; i<=BENCHMARK_ITERATIONS; i++)); do
        local time
        time=$(benchmark::measure "$operation" "$function" "$@")
        
        if [[ $time -eq -1 ]]; then
            ((errors++))
        else
            times+=("$time")
        fi
        
        # Progress indicator
        echo -n "." >&2
    done
    echo "" >&2
    
    # Calculate statistics
    if [[ ${#times[@]} -eq 0 ]]; then
        log::error "All benchmark runs failed"
        return 1
    fi
    
    # Sort times array
    IFS=$'\n' sorted_times=($(sort -n <<<"${times[*]}"))
    unset IFS
    
    local min="${sorted_times[0]}"
    local max="${sorted_times[-1]}"
    local count="${#times[@]}"
    
    # Calculate average
    local sum=0
    for t in "${times[@]}"; do
        sum=$((sum + t))
    done
    local avg=$((sum / count))
    
    # Calculate median
    local median
    if [[ $((count % 2)) -eq 0 ]]; then
        local mid1="${sorted_times[$((count/2 - 1))]}"
        local mid2="${sorted_times[$((count/2))]}"
        median=$(( (mid1 + mid2) / 2 ))
    else
        median="${sorted_times[$((count/2))]}"
    fi
    
    # Calculate 95th percentile
    local p95_index=$(( (count * 95) / 100 ))
    local p95="${sorted_times[$p95_index]}"
    
    # Build results JSON
    cat <<EOF
{
    "operation": "$operation",
    "iterations": $BENCHMARK_ITERATIONS,
    "successful": $count,
    "errors": $errors,
    "statistics": {
        "min": $min,
        "max": $max,
        "avg": $avg,
        "median": $median,
        "p95": $p95
    },
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
}

#######################################
# Benchmark navigation operations
#######################################
benchmark::navigation() {
    log::header "🚀 Navigation Benchmarks"
    
    local results=()
    
    # Simple navigation
    local result
    result=$(benchmark::run_suite "simple_navigation" browser::navigate "https://example.com")
    results+=("$result")
    
    # Navigation with wait
    result=$(benchmark::run_suite "navigation_with_wait" browser::navigate "https://httpbin.org/delay/1" "default" "networkidle2")
    results+=("$result")
    
    # Complex page navigation
    result=$(benchmark::run_suite "complex_navigation" browser::navigate "https://news.ycombinator.com")
    results+=("$result")
    
    # Output results
    echo "["
    local first=true
    for r in "${results[@]}"; do
        if [[ "$first" == "true" ]]; then
            first=false
        else
            echo ","
        fi
        echo "$r"
    done
    echo "]"
}

#######################################
# Benchmark screenshot operations
#######################################
benchmark::screenshots() {
    log::header "📸 Screenshot Benchmarks"
    
    local results=()
    local temp_file="/tmp/benchmark_screenshot.png"
    
    # Simple screenshot
    local result
    result=$(benchmark::run_suite "simple_screenshot" browser::screenshot "$temp_file")
    results+=("$result")
    
    # Full page screenshot
    result=$(benchmark::run_suite "fullpage_screenshot" browser::screenshot "$temp_file" "default" "true")
    results+=("$result")
    
    # Cleanup
    rm -f "$temp_file"
    
    # Output results
    echo "["
    local first=true
    for r in "${results[@]}"; do
        if [[ "$first" == "true" ]]; then
            first=false
        else
            echo ","
        fi
        echo "$r"
    done
    echo "]"
}

#######################################
# Benchmark extraction operations
#######################################
benchmark::extraction() {
    log::header "📊 Extraction Benchmarks"
    
    local results=()
    
    # Navigate to test page first
    browser::navigate "https://example.com" >/dev/null 2>&1
    
    # Text extraction
    local result
    result=$(benchmark::run_suite "text_extraction" browser::get_text "h1")
    results+=("$result")
    
    # Element existence check
    result=$(benchmark::run_suite "element_exists" browser::element_exists "div")
    results+=("$result")
    
    # Attribute extraction
    result=$(benchmark::run_suite "attribute_extraction" browser::get_attribute "a" "href")
    results+=("$result")
    
    # Output results
    echo "["
    local first=true
    for r in "${results[@]}"; do
        if [[ "$first" == "true" ]]; then
            first=false
        else
            echo ","
        fi
        echo "$r"
    done
    echo "]"
}

#######################################
# Benchmark workflow operations
#######################################
benchmark::workflows() {
    log::header "🔄 Workflow Benchmarks"
    
    local results=()
    
    # Simple workflow navigation
    local result
    result=$(benchmark::run_suite "workflow_navigation" workflow::navigate "https://example.com")
    results+=("$result")
    
    # Text extraction workflow
    result=$(benchmark::run_suite "workflow_extract_text" workflow::extract_text "h1")
    results+=("$result")
    
    # Screenshot workflow
    local temp_file="/tmp/workflow_screenshot.png"
    result=$(benchmark::run_suite "workflow_screenshot" workflow::screenshot "$temp_file")
    rm -f "$temp_file"
    results+=("$result")
    
    # Output results
    echo "["
    local first=true
    for r in "${results[@]}"; do
        if [[ "$first" == "true" ]]; then
            first=false
        else
            echo ","
        fi
        echo "$r"
    done
    echo "]"
}

#######################################
# Run all benchmarks
#######################################
benchmark::run_all() {
    log::header "🏁 Running All Benchmarks"
    
    benchmark::init || return 1
    
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local output_file="${BENCHMARK_OUTPUT_DIR}/benchmark_${timestamp}.json"
    
    {
        echo "{"
        echo '  "navigation": '
        benchmark::navigation | sed 's/^/    /'
        echo ','
        echo '  "screenshots": '
        benchmark::screenshots | sed 's/^/    /'
        echo ','
        echo '  "extraction": '
        benchmark::extraction | sed 's/^/    /'
        echo ','
        echo '  "workflows": '
        benchmark::workflows | sed 's/^/    /'
        echo ""
        echo "}"
    } > "$output_file"
    
    log::success "Benchmarks saved to: $output_file"
    
    # Show summary
    benchmark::show_summary "$output_file"
}

#######################################
# Show benchmark summary
# Arguments:
#   $1 - Benchmark file path
#######################################
benchmark::show_summary() {
    local file="${1:?File path required}"
    
    if [[ ! -f "$file" ]]; then
        log::error "Benchmark file not found: $file"
        return 1
    fi
    
    log::header "📈 Benchmark Summary"
    
    # Parse and display key metrics
    echo "Navigation Performance:"
    jq -r '.navigation[] | "  \(.operation): avg=\(.statistics.avg)ms, p95=\(.statistics.p95)ms"' "$file"
    
    echo ""
    echo "Screenshot Performance:"
    jq -r '.screenshots[] | "  \(.operation): avg=\(.statistics.avg)ms, p95=\(.statistics.p95)ms"' "$file"
    
    echo ""
    echo "Extraction Performance:"
    jq -r '.extraction[] | "  \(.operation): avg=\(.statistics.avg)ms, p95=\(.statistics.p95)ms"' "$file"
    
    echo ""
    echo "Workflow Performance:"
    jq -r '.workflows[] | "  \(.operation): avg=\(.statistics.avg)ms, p95=\(.statistics.p95)ms"' "$file"
}

#######################################
# Compare benchmarks
# Arguments:
#   $1 - First benchmark file
#   $2 - Second benchmark file
#######################################
benchmark::compare() {
    local file1="${1:?First file required}"
    local file2="${2:?Second file required}"
    
    if [[ ! -f "$file1" ]] || [[ ! -f "$file2" ]]; then
        log::error "Both benchmark files must exist"
        return 1
    fi
    
    log::header "📊 Benchmark Comparison"
    
    # Compare navigation
    echo "Navigation Changes:"
    for op in $(jq -r '.navigation[].operation' "$file1"); do
        local avg1=$(jq -r ".navigation[] | select(.operation==\"$op\") | .statistics.avg" "$file1")
        local avg2=$(jq -r ".navigation[] | select(.operation==\"$op\") | .statistics.avg" "$file2")
        
        if [[ -n "$avg1" ]] && [[ -n "$avg2" ]]; then
            local change=$(( (avg2 - avg1) * 100 / avg1 ))
            echo "  $op: ${avg1}ms → ${avg2}ms (${change}%)"
        fi
    done
    
    # Similar comparisons for other categories...
}

# Export functions
export -f benchmark::init
export -f benchmark::measure
export -f benchmark::run_suite
export -f benchmark::navigation
export -f benchmark::screenshots
export -f benchmark::extraction
export -f benchmark::workflows
export -f benchmark::run_all
export -f benchmark::show_summary
export -f benchmark::compare