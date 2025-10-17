#!/usr/bin/env bash
# Performance Logging for Qdrant Embeddings
# Provides structured logging of performance metrics during embedding operations

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

# Performance logging configuration
PERF_LOG_ENABLED="${EMBEDDING_PERF_LOG_ENABLED:-true}"
PERF_LOG_LEVEL="${EMBEDDING_PERF_LOG_LEVEL:-info}"
PERF_LOG_FORMAT="${EMBEDDING_PERF_LOG_FORMAT:-json}"
PERF_LOG_FILE="${EMBEDDING_PERF_LOG_FILE:-}"

#######################################
# Log structured performance metric
# Arguments:
#   $1 - Operation name (e.g., "embedding_generation", "parallel_processing")
#   $2 - Metric name (e.g., "duration_ms", "items_processed", "success_rate")
#   $3 - Metric value
#   $4 - Additional context (optional JSON string)
# Returns: 0 on success
#######################################
perf::log_metric() {
    if [[ "$PERF_LOG_ENABLED" != "true" ]]; then
        return 0
    fi
    
    local operation="$1"
    local metric="$2"
    local value="$3"
    local context="${4:-{}}"
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    
    # Build structured log entry
    local log_entry
    if [[ "$PERF_LOG_FORMAT" == "json" ]]; then
        log_entry=$(jq -n \
            --arg ts "$timestamp" \
            --arg op "$operation" \
            --arg metric "$metric" \
            --arg value "$value" \
            --argjson context "$context" \
            '{
                timestamp: $ts,
                level: "PERF",
                operation: $op,
                metric: $metric,
                value: $value,
                context: $context
            }')
    else
        log_entry="[$timestamp] PERF $operation.$metric=$value context=$context"
    fi
    
    # Output based on configuration
    if [[ -n "$PERF_LOG_FILE" ]]; then
        echo "$log_entry" >> "$PERF_LOG_FILE"
    fi
    
    # Always log to debug for development
    log::debug "PERF: $operation.$metric=$value"
    
    return 0
}

#######################################
# Log embedding generation performance
# Arguments:
#   $1 - Duration in milliseconds
#   $2 - Items processed count
#   $3 - Model used
#   $4 - Content type (optional)
# Returns: 0 on success
#######################################
perf::log_embedding_generation() {
    local duration_ms="$1"
    local items_count="$2"
    local model="$3"
    local content_type="${4:-unknown}"
    
    local throughput=$(( items_count * 1000 / (duration_ms + 1) ))  # items per second
    local avg_time_per_item=$(( duration_ms / (items_count + 1) ))  # ms per item
    
    local context=$(jq -n \
        --arg model "$model" \
        --arg type "$content_type" \
        --arg count "$items_count" \
        --arg throughput "$throughput" \
        --arg avg_ms "$avg_time_per_item" \
        '{
            model: $model,
            content_type: $type,
            items_processed: ($count | tonumber),
            throughput_per_sec: ($throughput | tonumber),
            avg_ms_per_item: ($avg_ms | tonumber)
        }')
    
    perf::log_metric "embedding_generation" "duration_ms" "$duration_ms" "$context"
    perf::log_metric "embedding_generation" "throughput_per_sec" "$throughput" "$context"
}

#######################################
# Log parallel processing performance
# Arguments:
#   $1 - Total duration in seconds
#   $2 - Number of workers
#   $3 - Number of jobs completed successfully
#   $4 - Number of jobs failed
#   $5 - Content type being processed
# Returns: 0 on success
#######################################
perf::log_parallel_processing() {
    local duration_sec="$1"
    local workers="$2"
    local completed="$3"
    local failed="$4"
    local content_type="$5"
    
    local total_jobs=$((completed + failed))
    local success_rate=0
    if [[ $total_jobs -gt 0 ]]; then
        success_rate=$((completed * 100 / total_jobs))
    fi
    
    local context=$(jq -n \
        --arg workers "$workers" \
        --arg content_type "$content_type" \
        --arg completed "$completed" \
        --arg failed "$failed" \
        --arg total "$total_jobs" \
        '{
            workers: ($workers | tonumber),
            content_type: $content_type,
            jobs_completed: ($completed | tonumber),
            jobs_failed: ($failed | tonumber),
            jobs_total: ($total | tonumber),
            efficiency_ratio: (if ($workers | tonumber) > 0 then (($completed | tonumber) / ($workers | tonumber)) else 0 end)
        }')
    
    perf::log_metric "parallel_processing" "duration_sec" "$duration_sec" "$context"
    perf::log_metric "parallel_processing" "success_rate_pct" "$success_rate" "$context"
}

#######################################
# Log memory usage during processing
# Arguments:
#   $1 - Memory usage percentage
#   $2 - Operation context
# Returns: 0 on success
#######################################
perf::log_memory_usage() {
    local memory_pct="$1"
    local operation_context="$2"
    
    local context=$(jq -n \
        --arg ctx "$operation_context" \
        '{
            operation: $ctx,
            threshold_warning: 85,
            threshold_critical: 95
        }')
    
    perf::log_metric "memory_usage" "memory_pct" "$memory_pct" "$context"
    
    # Warn if memory usage is high
    if [[ $memory_pct -gt 85 ]]; then
        log::warn "High memory usage detected: ${memory_pct}% during $operation_context"
    fi
}

#######################################
# Log extractor performance
# Arguments:
#   $1 - Extractor name (e.g., "code", "docs", "scenarios")
#   $2 - Files processed count
#   $3 - Processing duration in seconds
#   $4 - Errors encountered count
# Returns: 0 on success
#######################################
perf::log_extractor_performance() {
    local extractor="$1"
    local files_processed="$2"
    local duration_sec="$3"
    local errors="$4"
    
    local files_per_sec=0
    if [[ $duration_sec -gt 0 ]]; then
        files_per_sec=$((files_processed / duration_sec))
    fi
    
    local error_rate=0
    if [[ $files_processed -gt 0 ]]; then
        error_rate=$((errors * 100 / files_processed))
    fi
    
    local context=$(jq -n \
        --arg extractor "$extractor" \
        --arg files "$files_processed" \
        --arg errors "$errors" \
        --arg rate "$files_per_sec" \
        '{
            extractor_type: $extractor,
            files_processed: ($files | tonumber),
            errors_count: ($errors | tonumber),
            files_per_sec: ($rate | tonumber),
            error_rate_pct: (if ($files | tonumber) > 0 then (($errors | tonumber) * 100 / ($files | tonumber)) else 0 end)
        }')
    
    perf::log_metric "extractor_performance" "duration_sec" "$duration_sec" "$context"
    perf::log_metric "extractor_performance" "files_per_sec" "$files_per_sec" "$context"
}

#######################################
# Start a performance timer
# Arguments:
#   $1 - Timer name
# Returns: Sets global variable PERF_TIMER_<name> with start time
#######################################
perf::timer_start() {
    local timer_name="$1"
    local var_name="PERF_TIMER_${timer_name^^}"
    local start_time=$(date +%s%3N)  # milliseconds since epoch
    
    # Use declare to create global variable
    declare -g "$var_name"="$start_time"
}

#######################################
# Stop a performance timer and log result
# Arguments:
#   $1 - Timer name
#   $2 - Operation context (optional)
# Returns: Duration in milliseconds
#######################################
perf::timer_stop() {
    local timer_name="$1"
    local context="${2:-{}}"
    local var_name="PERF_TIMER_${timer_name^^}"
    local end_time=$(date +%s%3N)
    
    # Get start time from global variable
    local start_time="${!var_name:-0}"
    local duration_ms=$((end_time - start_time))
    
    if [[ $start_time -eq 0 ]]; then
        log::warn "Timer '$timer_name' was not started"
        return 1
    fi
    
    perf::log_metric "timer" "${timer_name}_duration_ms" "$duration_ms" "$context"
    
    # Clean up the timer variable
    unset "$var_name"
    
    echo "$duration_ms"
}

#######################################
# Generate performance summary report
# Returns: JSON summary of performance metrics
#######################################
perf::generate_summary() {
    if [[ -z "$PERF_LOG_FILE" ]] || [[ ! -f "$PERF_LOG_FILE" ]]; then
        echo '{"error": "No performance log file available"}'
        return 1
    fi
    
    # Parse recent performance logs and generate summary
    local recent_logs=$(tail -100 "$PERF_LOG_FILE" | grep -E '"level":"PERF"' || echo "")
    
    if [[ -z "$recent_logs" ]]; then
        echo '{"error": "No recent performance data found"}'
        return 1
    fi
    
    echo "$recent_logs" | jq -s '
        {
            summary: {
                total_entries: length,
                operations: [.[].operation] | unique,
                avg_embedding_duration: ([.[] | select(.operation == "embedding_generation" and .metric == "duration_ms") | .value | tonumber] | add / length),
                avg_throughput: ([.[] | select(.operation == "embedding_generation" and .metric == "throughput_per_sec") | .value | tonumber] | add / length),
                parallel_efficiency: ([.[] | select(.operation == "parallel_processing" and .metric == "success_rate_pct") | .value | tonumber] | add / length)
            },
            recent_metrics: .[-10:]
        }'
}

#######################################
# Analyze parallel processing efficiency
# Arguments:
#   $1 - Number of workers used
#   $2 - Total processing time (seconds)
#   $3 - Items processed
#   $4 - Sequential baseline time (optional, estimated if not provided)
# Returns: JSON with efficiency analysis
#######################################
perf::analyze_parallel_efficiency() {
    local workers="$1"
    local parallel_time="$2"
    local items_processed="$3"
    local sequential_baseline="${4:-}"
    
    # Estimate sequential time if not provided (assuming linear scaling)
    if [[ -z "$sequential_baseline" ]]; then
        sequential_baseline=$((parallel_time * workers))
    fi
    
    # Calculate efficiency metrics
    local theoretical_speedup=$workers
    local actual_speedup=1
    if [[ $parallel_time -gt 0 ]]; then
        actual_speedup=$(echo "$sequential_baseline $parallel_time" | awk '{printf "%.2f", $1/$2}')
    fi
    
    local efficiency=$(echo "$actual_speedup $theoretical_speedup" | awk '{printf "%.2f", ($1/$2)*100}')
    local throughput=0
    if [[ $parallel_time -gt 0 ]]; then
        throughput=$(echo "$items_processed $parallel_time" | awk '{printf "%.2f", $1/$2}')
    fi
    
    # Determine efficiency rating
    local rating="poor"
    if (( $(echo "$efficiency >= 80" | bc -l 2>/dev/null || echo 0) )); then
        rating="excellent"
    elif (( $(echo "$efficiency >= 60" | bc -l 2>/dev/null || echo 0) )); then
        rating="good" 
    elif (( $(echo "$efficiency >= 40" | bc -l 2>/dev/null || echo 0) )); then
        rating="fair"
    fi
    
    # Generate efficiency analysis
    jq -n \
        --arg workers "$workers" \
        --arg parallel_time "$parallel_time" \
        --arg sequential_time "$sequential_baseline" \
        --arg items "$items_processed" \
        --arg actual_speedup "$actual_speedup" \
        --arg theoretical_speedup "$theoretical_speedup" \
        --arg efficiency "$efficiency" \
        --arg throughput "$throughput" \
        --arg rating "$rating" \
        '{
            workers: ($workers | tonumber),
            parallel_time_sec: ($parallel_time | tonumber),
            estimated_sequential_time_sec: ($sequential_time | tonumber),
            items_processed: ($items | tonumber),
            speedup: {
                actual: ($actual_speedup | tonumber),
                theoretical: ($theoretical_speedup | tonumber),
                efficiency_percent: ($efficiency | tonumber)
            },
            throughput_per_sec: ($throughput | tonumber),
            rating: $rating,
            recommendations: (
                if ($efficiency | tonumber) < 40 then
                    ["Consider reducing workers due to overhead", "Check for I/O bottlenecks", "Optimize task distribution"]
                elif ($efficiency | tonumber) < 60 then
                    ["Monitor memory usage", "Consider task size optimization"]
                else
                    ["Current configuration is efficient", "Consider increasing workers if more resources available"]
                end
            )
        }'
}

#######################################
# Compare multiple parallel processing runs
# Arguments:
#   None (reads from performance log file)
# Returns: JSON with comparative efficiency analysis
#######################################
perf::compare_parallel_runs() {
    if [[ -z "$PERF_LOG_FILE" ]] || [[ ! -f "$PERF_LOG_FILE" ]]; then
        echo '{"error": "No performance log file available"}'
        return 1
    fi
    
    # Extract parallel processing metrics
    local parallel_runs=$(grep -E '"operation":"parallel_processing"' "$PERF_LOG_FILE" | tail -10)
    
    if [[ -z "$parallel_runs" ]]; then
        echo '{"error": "No parallel processing data found"}'
        return 1
    fi
    
    echo "$parallel_runs" | jq -s '
        group_by(.context.workers) | 
        map({
            workers: .[0].context.workers,
            runs: length,
            avg_duration: ([.[].value] | add / length),
            avg_success_rate: ([.[] | select(.metric == "success_rate_pct") | .value] | add / length // 0),
            avg_efficiency: ([.[] | select(.context.efficiency_ratio != null) | .context.efficiency_ratio] | add / length // 0),
            total_items: ([.[].context.jobs_completed] | add),
            recommendations: (
                if length >= 3 and (([.[].value] | add / length) < (.[0].value * 0.9)) then
                    "Performance degrading - investigate system load"
                elif .[0].context.workers > 8 and (([.[] | select(.metric == "success_rate_pct") | .value] | add / length) < 90) then
                    "High worker count causing failures - consider reduction"
                else
                    "Performance stable"
                end
            )
        }) |
        {
            comparison_summary: .,
            optimal_workers: (map(select(.avg_success_rate > 90 and .avg_efficiency > 0.5)) | max_by(.avg_efficiency) | .workers // "insufficient_data"),
            analysis: {
                total_runs_analyzed: (map(.runs) | add),
                worker_configs_tested: length,
                best_throughput: (map(.total_items / .avg_duration) | max),
                efficiency_trend: (if length > 1 then (if .[1].avg_efficiency > .[0].avg_efficiency then "improving" else "declining" end) else "single_point" end)
            }
        }'
}

#######################################
# Generate parallel processing efficiency report
# Returns: Comprehensive efficiency report
#######################################
perf::generate_efficiency_report() {
    echo "=== Parallel Processing Efficiency Report ==="
    echo
    
    # Current configuration analysis
    echo "🔧 Current Configuration:"
    echo "  Max Workers: ${EMBEDDING_MAX_WORKERS:-16}"
    echo "  Memory Limit: ${EMBEDDING_MEMORY_LIMIT:-80}%"
    echo "  Batch Size: ${EMBEDDING_BATCH_SIZE:-50}"
    echo
    
    # Recent efficiency analysis
    local comparison=$(perf::compare_parallel_runs 2>/dev/null)
    if [[ "$comparison" != *"error"* ]]; then
        echo "📊 Efficiency Analysis:"
        echo "$comparison" | jq -r '
            "  Optimal Workers: " + (.optimal_workers | tostring) + "\n" +
            "  Total Runs Analyzed: " + (.analysis.total_runs_analyzed | tostring) + "\n" +
            "  Configs Tested: " + (.analysis.worker_configs_tested | tostring) + "\n" +
            "  Best Throughput: " + (.analysis.best_throughput | tostring) + " items/sec\n" +
            "  Efficiency Trend: " + .analysis.efficiency_trend
        '
        
        echo
        echo "📈 Performance by Worker Count:"
        echo "$comparison" | jq -r '.comparison_summary[] | 
            "  " + (.workers | tostring) + " workers: " + 
            (.avg_duration | tostring) + "s avg, " + 
            (.avg_success_rate | tostring) + "% success, " +
            "recommendation: " + .recommendations'
    else
        echo "⚠️  Insufficient data for efficiency analysis"
        echo "   Run multiple embedding refreshes to generate comparison data"
    fi
    
    echo
    echo "💡 General Recommendations:"
    echo "  • Monitor memory usage during parallel processing"  
    echo "  • Test different worker counts to find optimal configuration"
    echo "  • Consider I/O limitations when increasing parallelism"
    echo "  • Use sequential mode for debugging or resource-constrained environments"
}

# Export all functions
export -f perf::log_metric
export -f perf::log_embedding_generation
export -f perf::log_parallel_processing
export -f perf::log_memory_usage
export -f perf::log_extractor_performance
export -f perf::timer_start
export -f perf::timer_stop
export -f perf::generate_summary
export -f perf::analyze_parallel_efficiency
export -f perf::compare_parallel_runs
export -f perf::generate_efficiency_report