#!/bin/bash

################################################################################
# Analytics and Monitoring for Judge0
# 
# Provides execution analytics, performance tracking, and monitoring insights
################################################################################

# Initialize analytics storage
judge0::analytics::init() {
    local analytics_dir="${JUDGE0_ANALYTICS_DIR:-$HOME/.vrooli/resources/judge0/analytics}"
    
    if [[ ! -d "$analytics_dir" ]]; then
        mkdir -p "$analytics_dir"
        echo "[INFO] Analytics directory created: $analytics_dir"
    fi
    
    # Initialize daily stats file
    local today
    today=$(date +%Y-%m-%d)
    local stats_file="$analytics_dir/stats_$today.json"
    
    if [[ ! -f "$stats_file" ]]; then
        echo '{"submissions": [], "summary": {}}' > "$stats_file"
    fi
    
    export JUDGE0_ANALYTICS_DIR="$analytics_dir"
    export JUDGE0_STATS_FILE="$stats_file"
}

# Record submission
judge0::analytics::record_submission() {
    local token="$1"
    local language_id="$2"
    local status="$3"
    local execution_time="$4"
    local memory_used="$5"
    
    judge0::analytics::init
    
    local timestamp
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    
    local entry
    entry=$(jq -n \
        --arg tk "$token" \
        --arg ts "$timestamp" \
        --arg lang "$language_id" \
        --arg st "$status" \
        --arg time "${execution_time:-0}" \
        --arg mem "${memory_used:-0}" \
        '{
            token: $tk,
            timestamp: $ts,
            language_id: ($lang | tonumber),
            status: $st,
            execution_time: ($time | tonumber),
            memory_used: ($mem | tonumber)
        }')
    
    # Append to stats file
    local updated
    updated=$(jq ".submissions += [$entry]" "$JUDGE0_STATS_FILE")
    echo "$updated" > "$JUDGE0_STATS_FILE"
    
    echo "[INFO] Recorded submission: $token"
}

# Get execution statistics
judge0::analytics::get_stats() {
    local period="${1:-today}"
    local analytics_dir="${JUDGE0_ANALYTICS_DIR:-$HOME/.vrooli/resources/judge0/analytics}"
    
    case "$period" in
        today)
            local today
            today=$(date +%Y-%m-%d)
            local stats_file="$analytics_dir/stats_$today.json"
            
            if [[ -f "$stats_file" ]]; then
                judge0::analytics::calculate_summary "$stats_file"
            else
                echo '{"error": "No data for today"}'
            fi
            ;;
        week)
            judge0::analytics::aggregate_period 7
            ;;
        month)
            judge0::analytics::aggregate_period 30
            ;;
        *)
            echo "[ERROR] Invalid period. Use: today, week, or month" >&2
            return 1
            ;;
    esac
}

# Calculate summary statistics
judge0::analytics::calculate_summary() {
    local stats_file="$1"
    
    if [[ ! -f "$stats_file" ]]; then
        echo '{"error": "Stats file not found"}'
        return 1
    fi
    
    # Calculate statistics using jq
    jq '{
        total_submissions: .submissions | length,
        languages: (.submissions | group_by(.language_id) | map({
            language_id: .[0].language_id,
            count: length,
            avg_execution_time: (map(.execution_time) | add / length),
            avg_memory: (map(.memory_used) | add / length)
        })),
        status_distribution: (.submissions | group_by(.status) | map({
            status: .[0].status,
            count: length
        })),
        performance: {
            avg_execution_time: (.submissions | map(.execution_time) | add / length),
            max_execution_time: (.submissions | map(.execution_time) | max),
            min_execution_time: (.submissions | map(.execution_time) | min),
            avg_memory: (.submissions | map(.memory_used) | add / length),
            max_memory: (.submissions | map(.memory_used) | max)
        },
        time_range: {
            first: (.submissions | first | .timestamp),
            last: (.submissions | last | .timestamp)
        }
    }' "$stats_file"
}

# Aggregate statistics over a period
judge0::analytics::aggregate_period() {
    local days="$1"
    local analytics_dir="${JUDGE0_ANALYTICS_DIR:-$HOME/.vrooli/resources/judge0/analytics}"
    
    local all_submissions='[]'
    local i
    
    for ((i=0; i<days; i++)); do
        local date
        date=$(date -d "$i days ago" +%Y-%m-%d)
        local stats_file="$analytics_dir/stats_$date.json"
        
        if [[ -f "$stats_file" ]]; then
            local day_submissions
            day_submissions=$(jq '.submissions' "$stats_file")
            all_submissions=$(echo "$all_submissions" "$day_submissions" | jq -s 'add')
        fi
    done
    
    # Create temporary file with aggregated data
    local temp_file="/tmp/judge0_aggregate_$$.json"
    echo "{\"submissions\": $all_submissions}" > "$temp_file"
    
    # Calculate summary
    judge0::analytics::calculate_summary "$temp_file"
    
    # Cleanup
    rm -f "$temp_file"
}

# Monitor performance trends
judge0::analytics::monitor_trends() {
    local threshold_time="${1:-5.0}"  # Default 5 second threshold
    local threshold_memory="${2:-512000}"  # Default 500MB threshold
    
    judge0::analytics::init
    
    echo "[INFO] Monitoring performance trends..."
    echo "[INFO] Thresholds - Time: ${threshold_time}s, Memory: ${threshold_memory} bytes"
    
    # Check today's stats
    local slow_submissions
    slow_submissions=$(jq --arg t "$threshold_time" \
        '[.submissions[] | select(.execution_time > ($t | tonumber))]' \
        "$JUDGE0_STATS_FILE")
    
    local memory_heavy
    memory_heavy=$(jq --arg m "$threshold_memory" \
        '[.submissions[] | select(.memory_used > ($m | tonumber))]' \
        "$JUDGE0_STATS_FILE")
    
    local slow_count
    slow_count=$(echo "$slow_submissions" | jq 'length')
    
    local heavy_count
    heavy_count=$(echo "$memory_heavy" | jq 'length')
    
    echo "[INFO] Slow submissions (>${threshold_time}s): $slow_count"
    echo "[INFO] Memory-heavy submissions (>${threshold_memory} bytes): $heavy_count"
    
    # Return detailed report
    jq -n \
        --argjson slow "$slow_submissions" \
        --argjson heavy "$memory_heavy" \
        --arg st "$threshold_time" \
        --arg sm "$threshold_memory" \
        '{
            thresholds: {
                time: ($st | tonumber),
                memory: ($sm | tonumber)
            },
            slow_submissions: $slow,
            memory_heavy_submissions: $heavy,
            alerts: {
                slow_count: ($slow | length),
                heavy_count: ($heavy | length)
            }
        }'
}

# Real-time monitoring dashboard
judge0::analytics::dashboard() {
    echo "[INFO] Judge0 Analytics Dashboard"
    echo "================================="
    
    # Get current stats
    local stats
    stats=$(judge0::analytics::get_stats today)
    
    echo ""
    echo "📊 Today's Statistics:"
    echo "$stats" | jq -r '
        "  Total Submissions: \(.total_submissions)",
        "  Average Execution Time: \(.performance.avg_execution_time // 0)s",
        "  Max Execution Time: \(.performance.max_execution_time // 0)s",
        "  Average Memory: \(.performance.avg_memory // 0) bytes"'
    
    echo ""
    echo "📈 Language Distribution:"
    echo "$stats" | jq -r '.languages[] | 
        "  Language \(.language_id): \(.count) submissions (avg: \(.avg_execution_time)s)"'
    
    echo ""
    echo "📋 Status Distribution:"
    echo "$stats" | jq -r '.status_distribution[] | 
        "  \(.status): \(.count)"'
    
    # Check for performance issues
    local trends
    trends=$(judge0::analytics::monitor_trends)
    
    local slow_count
    slow_count=$(echo "$trends" | jq '.alerts.slow_count')
    
    if [[ "$slow_count" -gt 0 ]]; then
        echo ""
        echo "⚠️  Performance Alerts:"
        echo "  $slow_count slow submissions detected"
    fi
}

# Export analytics data
judge0::analytics::export() {
    local format="${1:-json}"
    local output_file="${2:-judge0_analytics_export.json}"
    
    judge0::analytics::init
    
    case "$format" in
        json)
            # Aggregate all data
            local all_data
            all_data=$(judge0::analytics::aggregate_period 30)
            echo "$all_data" > "$output_file"
            echo "[SUCCESS] Analytics exported to: $output_file"
            ;;
        csv)
            # Convert to CSV
            local csv_file="${output_file%.json}.csv"
            echo "timestamp,token,language_id,status,execution_time,memory_used" > "$csv_file"
            
            jq -r '.submissions[] | 
                [.timestamp, .token, .language_id, .status, .execution_time, .memory_used] | 
                @csv' "$JUDGE0_STATS_FILE" >> "$csv_file"
            
            echo "[SUCCESS] Analytics exported to: $csv_file"
            ;;
        *)
            echo "[ERROR] Unsupported format. Use: json or csv" >&2
            return 1
            ;;
    esac
}

# Export functions
export -f judge0::analytics::init
export -f judge0::analytics::record_submission
export -f judge0::analytics::get_stats
export -f judge0::analytics::calculate_summary
export -f judge0::analytics::aggregate_period
export -f judge0::analytics::monitor_trends
export -f judge0::analytics::dashboard
export -f judge0::analytics::export