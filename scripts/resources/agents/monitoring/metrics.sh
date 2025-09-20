#!/usr/bin/env bash
################################################################################
# Agent Metrics Collection
# 
# Lightweight metrics collection for agent monitoring
# Supports counters, gauges, and basic histograms
################################################################################

# Source common utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
if ! declare -F agents::registry::create_temp_file >/dev/null 2>&1; then
    source "${APP_ROOT}/scripts/resources/agents/core/registry.sh"
fi


#######################################
# Initialize metrics for an agent
# Arguments:
#   $1 - Registry file path
#   $2 - Agent ID
# Returns:
#   0 on success, 1 on error
#######################################
agents::metrics::init() {
    local registry_file="$1"
    local agent_id="$2"
    
    [[ -f "$registry_file" ]] || return 1
    
    local lock_fd
    exec {lock_fd}>"${registry_file}.lock" || return 1
    if ! flock -w 5 "$lock_fd"; then
        exec {lock_fd}>&-
        return 1
    fi

    local rc=0
    local temp_file
    temp_file=$(agents::registry::create_temp_file "$registry_file") || rc=1
    local current_time
    current_time=$(date -Iseconds)

    if [[ $rc -eq 0 ]]; then
        if ! jq --arg id "$agent_id" \
                --arg time "$current_time" \
                '
                if .agents[$id] then
                    .agents[$id].metrics = {
                        counters: {
                            requests: 0,
                            errors: 0,
                            restarts: 0
                        },
                        gauges: {
                            cpu_ticks: 0,
                            memory_mb: 0,
                            active_connections: 0
                        },
                        histograms: {
                            request_duration_ms: {
                                count: 0,
                                sum: 0,
                                min: null,
                                max: null,
                                samples: []
                            }
                        },
                        window: {
                            start: $time,
                            last_reset: $time
                        }
                    }
                else . end
                ' "$registry_file" > "$temp_file"; then
            rc=1
        fi
    fi

    if [[ $rc -eq 0 ]]; then
        if ! mv "$temp_file" "$registry_file"; then
            rc=1
        fi
    fi

    [[ -n "$temp_file" && -f "$temp_file" ]] && rm -f "$temp_file"

    flock -u "$lock_fd"
    exec {lock_fd}>&-

    return $rc
}

#######################################
# Increment a counter metric
# Arguments:
#   $1 - Registry file path
#   $2 - Agent ID
#   $3 - Counter name (e.g., "requests", "errors")
#   $4 - Increment value (default: 1)
# Returns:
#   0 on success, 1 on error
#######################################
agents::metrics::increment() {
    local registry_file="$1"
    local agent_id="$2"
    local counter_name="$3"
    local increment="${4:-1}"
    
    [[ -f "$registry_file" ]] || return 1
    
    local lock_fd
    exec {lock_fd}>"${registry_file}.lock" || return 1
    if ! flock -w 5 "$lock_fd"; then
        exec {lock_fd}>&-
        return 1
    fi

    local rc=0
    local temp_file
    temp_file=$(agents::registry::create_temp_file "$registry_file") || rc=1

    if [[ $rc -eq 0 ]]; then
        if ! jq --arg id "$agent_id" \
                --arg counter "$counter_name" \
                --argjson inc "$increment" \
                '
                if .agents[$id].metrics.counters[$counter] then
                    .agents[$id].metrics.counters[$counter] += $inc
                elif .agents[$id].metrics.counters then
                    .agents[$id].metrics.counters[$counter] = $inc
                else . end
                ' "$registry_file" > "$temp_file"; then
            rc=1
        fi
    fi

    if [[ $rc -eq 0 ]]; then
        if ! mv "$temp_file" "$registry_file"; then
            rc=1
        fi
    fi

    [[ -n "$temp_file" && -f "$temp_file" ]] && rm -f "$temp_file"

    flock -u "$lock_fd"
    exec {lock_fd}>&-

    return $rc
}

#######################################
# Set a gauge metric
# Arguments:
#   $1 - Registry file path
#   $2 - Agent ID
#   $3 - Gauge name (e.g., "cpu_percent", "memory_mb")
#   $4 - Value
# Returns:
#   0 on success, 1 on error
#######################################
agents::metrics::gauge() {
    local registry_file="$1"
    local agent_id="$2"
    local gauge_name="$3"
    local value="$4"
    
    [[ -f "$registry_file" ]] || return 1
    
    local lock_fd
    exec {lock_fd}>"${registry_file}.lock" || return 1
    if ! flock -w 5 "$lock_fd"; then
        exec {lock_fd}>&-
        return 1
    fi

    local temp_file
    local rc=0
    temp_file=$(agents::registry::create_temp_file "$registry_file") || rc=1

    if [[ $rc -eq 0 ]]; then
        if ! jq --arg id "$agent_id" \
                --arg gauge "$gauge_name" \
                --argjson val "$value" \
                '
                if .agents[$id].metrics.gauges then
                    .agents[$id].metrics.gauges[$gauge] = $val
                elif .agents[$id].metrics then
                    .agents[$id].metrics.gauges = {($gauge): $val}
                else . end
                ' "$registry_file" > "$temp_file"; then
            rc=1
        fi
    fi

    if [[ $rc -eq 0 ]]; then
        if ! mv "$temp_file" "$registry_file"; then
            rc=1
        fi
    fi

    [[ -n "$temp_file" && -f "$temp_file" ]] && rm -f "$temp_file"

    flock -u "$lock_fd"
    exec {lock_fd}>&-

    return $rc
}

#######################################
# Record a histogram sample
# Arguments:
#   $1 - Registry file path
#   $2 - Agent ID
#   $3 - Histogram name (e.g., "request_duration_ms")
#   $4 - Sample value
# Returns:
#   0 on success, 1 on error
#######################################
agents::metrics::histogram() {
    local registry_file="$1"
    local agent_id="$2"
    local histogram_name="$3"
    local sample_value="$4"
    
    [[ -f "$registry_file" ]] || return 1
    
    local lock_fd
    exec {lock_fd}>"${registry_file}.lock" || return 1
    if ! flock -w 5 "$lock_fd"; then
        exec {lock_fd}>&-
        return 1
    fi

    local temp_file
    local rc=0
    temp_file=$(agents::registry::create_temp_file "$registry_file") || rc=1

    if [[ $rc -eq 0 ]]; then
        if ! jq --arg id "$agent_id" \
                --arg hist "$histogram_name" \
                --argjson sample "$sample_value" \
                '
                if .agents[$id].metrics.histograms[$hist] then
                    .agents[$id].metrics.histograms[$hist].count += 1 |
                    .agents[$id].metrics.histograms[$hist].sum += $sample |
                    .agents[$id].metrics.histograms[$hist].samples = (
                        (.agents[$id].metrics.histograms[$hist].samples + [$sample]) | .[-100:]
                    ) |
                    .agents[$id].metrics.histograms[$hist].min = (
                        if .agents[$id].metrics.histograms[$hist].min == null then
                            $sample
                        else
                            [.agents[$id].metrics.histograms[$hist].min, $sample] | min
                        end
                    ) |
                    .agents[$id].metrics.histograms[$hist].max = (
                        if .agents[$id].metrics.histograms[$hist].max == null then
                            $sample
                        else
                            [.agents[$id].metrics.histograms[$hist].max, $sample] | max
                        end
                    ) |
                    .agents[$id].metrics.histograms[$hist].avg = (
                        .agents[$id].metrics.histograms[$hist].sum / .agents[$id].metrics.histograms[$hist].count
                    )
                elif .agents[$id].metrics.histograms then
                    .agents[$id].metrics.histograms[$hist] = {
                        count: 1,
                        sum: $sample,
                        min: $sample,
                        max: $sample,
                        samples: [$sample],
                        avg: $sample
                    }
                elif .agents[$id].metrics then
                    .agents[$id].metrics.histograms = {
                        ($hist): {
                            count: 1,
                            sum: $sample,
                            min: $sample,
                            max: $sample,
                            samples: [$sample],
                            avg: $sample
                        }
                    }
                else . end
                ' "$registry_file" > "$temp_file"; then
            rc=1
        fi
    fi

    if [[ $rc -eq 0 ]]; then
        if ! mv "$temp_file" "$registry_file"; then
            rc=1
        fi
    fi

    [[ -n "$temp_file" && -f "$temp_file" ]] && rm -f "$temp_file"

    flock -u "$lock_fd"
    exec {lock_fd}>&-

    return $rc
}

#######################################
# Get metrics summary for an agent
# Arguments:
#   $1 - Registry file path
#   $2 - Agent ID
#   $3 - Output format (json|text)
# Returns:
#   0 on success, 1 on error
# Outputs:
#   Metrics summary to stdout
#######################################
agents::metrics::get_summary() {
    local registry_file="$1"
    local agent_id="$2"
    local output_format="${3:-text}"
    
    [[ -f "$registry_file" ]] || return 1
    
    # Get metrics data
    local metrics_data
    metrics_data=$(jq --arg id "$agent_id" '.agents[$id].metrics // {}' "$registry_file" 2>/dev/null)
    
    if [[ -z "$metrics_data" || "$metrics_data" == "{}" ]]; then
        if [[ "$output_format" == "json" ]]; then
            echo '{}'
        else
            echo "No metrics available for agent: $agent_id"
        fi
        return 0
    fi
    
    if [[ "$output_format" == "json" ]]; then
        echo "$metrics_data"
    else
        # Format as text
        echo "=== Metrics for $agent_id ==="
        echo ""
        echo "Counters:"
        echo "$metrics_data" | jq -r '.counters | to_entries[] | "  \(.key): \(.value)"'
        echo ""
        echo "Gauges:"
        echo "$metrics_data" | jq -r '.gauges | to_entries[] | "  \(.key): \(.value)"'
        echo ""
        echo "Histograms:"
        echo "$metrics_data" | jq -r '
            .histograms | to_entries[] | 
            "  \(.key):\n    count: \(.value.count)\n    avg: \(.value.avg // "N/A")\n    min: \(.value.min // "N/A")\n    max: \(.value.max // "N/A")"
        '
    fi
    
    return 0
}

#######################################
# Calculate percentiles from histogram
# Arguments:
#   $1 - Registry file path
#   $2 - Agent ID
#   $3 - Histogram name
#   $4 - Percentile (e.g., 50, 95, 99)
# Returns:
#   0 on success, 1 on error
# Outputs:
#   Percentile value to stdout
#######################################
agents::metrics::percentile() {
    local registry_file="$1"
    local agent_id="$2"
    local histogram_name="$3"
    local percentile="$4"
    
    [[ -f "$registry_file" ]] || return 1
    
    # Get sorted samples
    local samples
    samples=$(jq -r --arg id "$agent_id" \
                    --arg hist "$histogram_name" \
                    '.agents[$id].metrics.histograms[$hist].samples | sort | .[]' \
                    "$registry_file" 2>/dev/null)
    
    if [[ -z "$samples" ]]; then
        echo "0"
        return 1
    fi
    
    # Calculate percentile index
    local count
    count=$(echo "$samples" | wc -l)
    local index=$(( (count * percentile + 99) / 100 ))
    
    # Get value at percentile
    echo "$samples" | sed -n "${index}p"
    return 0
}

#######################################
# Reset metrics window
# Arguments:
#   $1 - Registry file path
#   $2 - Agent ID
# Returns:
#   0 on success, 1 on error
#######################################
agents::metrics::reset_window() {
    local registry_file="$1"
    local agent_id="$2"
    
    [[ -f "$registry_file" ]] || return 1
    
    local temp_file="${registry_file}.tmp.$$"
    local current_time
    current_time=$(date -Iseconds)
    
    # Reset metrics window (keep structure but reset values)
    if ! jq --arg id "$agent_id" \
            --arg time "$current_time" \
            '
            if .agents[$id].metrics then
                .agents[$id].metrics.counters = (.agents[$id].metrics.counters | map_values(0)) |
                .agents[$id].metrics.histograms = (
                    .agents[$id].metrics.histograms | map_values({
                        count: 0,
                        sum: 0,
                        min: null,
                        max: null,
                        samples: []
                    })
                ) |
                .agents[$id].metrics.window.last_reset = $time
            else . end
            ' "$registry_file" > "$temp_file"; then
        rm -f "$temp_file"
        return 1
    fi
    
    mv "$temp_file" "$registry_file" || rm -f "$temp_file"
    return 0
}

# Export functions for use by resources
export -f agents::metrics::init
export -f agents::metrics::increment
export -f agents::metrics::gauge
export -f agents::metrics::histogram
export -f agents::metrics::get_summary
export -f agents::metrics::percentile
export -f agents::metrics::reset_window
