#!/bin/bash

# Simple Performance Metrics for Huginn
# Basic performance tracking without complex queries

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCRIPT_DIR}/../../.." && builtin pwd)}"

# Source required dependencies
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${SCRIPT_DIR}/common.sh"

# Get basic performance metrics
get_basic_metrics() {
    # Get basic counts
    local agent_count=$(docker exec huginn bash -c 'cd /app && timeout 5 bundle exec rails runner -e production "puts Agent.count"' 2>/dev/null)
    local event_count=$(docker exec huginn bash -c 'cd /app && timeout 5 bundle exec rails runner -e production "puts Event.count"' 2>/dev/null)
    local job_count=$(docker exec huginn bash -c 'cd /app && timeout 5 bundle exec rails runner -e production "puts DelayedJob.count"' 2>/dev/null)
    local failed_jobs=$(docker exec huginn bash -c 'cd /app && timeout 5 bundle exec rails runner -e production "puts DelayedJob.where(\"last_error IS NOT NULL\").count"' 2>/dev/null)
    
    # Get recent event counts
    local hour_events=$(docker exec huginn bash -c 'cd /app && timeout 5 bundle exec rails runner -e production "puts Event.where(\"created_at > ?\", 1.hour.ago).count"' 2>/dev/null)
    local day_events=$(docker exec huginn bash -c 'cd /app && timeout 5 bundle exec rails runner -e production "puts Event.where(\"created_at > ?\", 24.hours.ago).count"' 2>/dev/null)
    
    # Output as JSON
    cat <<EOF
{
  "overview": {
    "total_agents": ${agent_count:-0},
    "total_events": ${event_count:-0},
    "total_jobs": ${job_count:-0},
    "failed_jobs": ${failed_jobs:-0},
    "database_size_mb": 0
  },
  "performance": {
    "events_last_hour": ${hour_events:-0},
    "events_last_24h": ${day_events:-0}
  }
}
EOF
}

# Show performance summary
show_performance_summary() {
    log::info "Fetching performance metrics..."
    
    local metrics
    metrics=$(get_basic_metrics 2>/dev/null)
    
    if [[ -n "$metrics" ]]; then
        echo
        echo "═══════════════════════════════════════════════════"
        echo "           HUGINN PERFORMANCE SUMMARY"
        echo "═══════════════════════════════════════════════════"
        echo
        
        # Parse and display metrics
        echo "$metrics" | jq -r '
          "📊 SYSTEM OVERVIEW",
          "  Total Agents: \(.overview.total_agents)",
          "  Total Events: \(.overview.total_events)",
          "  Queue Size: \(.overview.total_jobs)",
          "  Failed Jobs: \(.overview.failed_jobs)",
          "",
          "⚡ ACTIVITY",
          "  Events (Last Hour): \(.performance.events_last_hour)",
          "  Events (Last 24h): \(.performance.events_last_24h)"
        ' 2>/dev/null || echo "Error parsing metrics"
        
        echo
        echo "═══════════════════════════════════════════════════"
    else
        log::error "Unable to fetch metrics. Is Huginn running?"
    fi
}

# Export metrics to file
export_metrics() {
    local output_file="${1:-huginn-metrics-$(date +%Y%m%d-%H%M%S).json}"
    
    log::info "Exporting metrics to ${output_file}..."
    
    local metrics
    metrics=$(get_basic_metrics 2>/dev/null)
    
    if [[ -n "$metrics" ]]; then
        echo "$metrics" > "$output_file"
        log::success "Metrics exported to ${output_file}"
    else
        log::error "Failed to export metrics"
        return 1
    fi
}

# Main function
main() {
    local command="${1:-summary}"
    
    case "$command" in
        summary)
            show_performance_summary
            ;;
        json)
            get_basic_metrics | jq .
            ;;
        export)
            export_metrics "${2:-}"
            ;;
        *)
            log::error "Unknown command: $command"
            echo "Available commands: summary, json, export"
            return 1
            ;;
    esac
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi