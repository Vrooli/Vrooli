#!/usr/bin/env bash
set -euo pipefail

# PostgreSQL Resource Usage Monitor
# Track resource consumption across all PostgreSQL instances

DESCRIPTION="Monitor resource usage for PostgreSQL instances"

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "${SCRIPT_DIR}/config/defaults.sh"

# Configuration
MONITOR_INTERVAL=5
OUTPUT_FORMAT="table"
CONTINUOUS_MODE="no"
SHOW_HISTORY="no"
EXPORT_METRICS="no"
METRICS_FILE=""

#######################################
# Show usage information
#######################################
show_usage() {
    cat << EOF
PostgreSQL Resource Usage Monitor

DESCRIPTION:
    $DESCRIPTION

USAGE:
    $0 [OPTIONS]

OPTIONS:
    -h, --help                          Show this help message
    -i, --interval <seconds>            Monitor interval in seconds (default: 5)
    -f, --format <table|json|csv>       Output format (default: table)
    -c, --continuous                    Run in continuous monitoring mode
    -H, --history                       Show resource usage history
    --export <file>                     Export metrics to file
    --instance <name>                   Monitor specific instance only

OUTPUT FORMATS:
    table       Human-readable table format
    json        JSON format for programmatic use
    csv         CSV format for spreadsheet import

EXAMPLES:
    # One-time resource check
    $0

    # Continuous monitoring with 10-second intervals
    $0 --continuous --interval 10

    # Export metrics to CSV
    $0 --format csv --export metrics.csv

    # Monitor specific instance
    $0 --instance client-name --continuous

    # JSON output for automation
    $0 --format json

NOTES:
    - Requires Docker to be running
    - Monitors CPU, memory, disk, and network usage
    - Includes PostgreSQL-specific metrics
    - Press Ctrl+C to stop continuous monitoring

EOF
}

#######################################
# Parse command line arguments
#######################################
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_usage
                exit 0
                ;;
            -i|--interval)
                MONITOR_INTERVAL="$2"
                shift 2
                ;;
            -f|--format)
                OUTPUT_FORMAT="$2"
                shift 2
                ;;
            -c|--continuous)
                CONTINUOUS_MODE="yes"
                shift
                ;;
            -H|--history)
                SHOW_HISTORY="yes"
                shift
                ;;
            --export)
                EXPORT_METRICS="yes"
                METRICS_FILE="$2"
                shift 2
                ;;
            --instance)
                SPECIFIC_INSTANCE="$2"
                shift 2
                ;;
            *)
                echo "Unknown option: $1" >&2
                show_usage >&2
                exit 1
                ;;
        esac
    done
}

#######################################
# Get container resource usage
#######################################
get_container_stats() {
    local container_name="$1"
    
    if ! docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}\t{{.NetIO}}\t{{.BlockIO}}" "$container_name" 2>/dev/null; then
        echo "$container_name\tN/A\tN/A\tN/A\tN/A\tN/A"
    fi
}

#######################################
# Get PostgreSQL specific metrics
#######################################
get_postgres_metrics() {
    local instance_name="$1"
    local container_name="${POSTGRES_CONTAINER_PREFIX}-${instance_name}"
    
    if ! docker exec "$container_name" psql -U "$POSTGRES_DEFAULT_USER" -d "$POSTGRES_DEFAULT_DB" -t -c "
        SELECT 
            (SELECT count(*) FROM pg_stat_activity WHERE state = 'active') as active_connections,
            (SELECT count(*) FROM pg_stat_activity) as total_connections,
            pg_size_pretty(pg_database_size('$POSTGRES_DEFAULT_DB')) as db_size,
            (SELECT sum(numbackends) FROM pg_stat_database) as backends,
            (SELECT round(avg(query_duration_ms)::numeric, 2) FROM (
                SELECT extract(epoch from (now() - query_start)) * 1000 as query_duration_ms 
                FROM pg_stat_activity 
                WHERE state = 'active' AND query_start IS NOT NULL
            ) q) as avg_query_time_ms;
    " 2>/dev/null | tr -d ' ' | tr '|' '\t'; then
        echo "N/A\tN/A\tN/A\tN/A\tN/A"
    fi
}

#######################################
# Get disk usage for instance
#######################################
get_disk_usage() {
    local instance_name="$1"
    local instance_dir="${POSTGRES_INSTANCES_DIR}/${instance_name}"
    local backup_dir="${POSTGRES_BACKUP_DIR}/${instance_name}"
    
    local data_size="N/A"
    local backup_size="N/A"
    
    if [[ -d "$instance_dir" ]]; then
        data_size=$(du -sh "$instance_dir" 2>/dev/null | cut -f1 || echo "N/A")
    fi
    
    if [[ -d "$backup_dir" ]]; then
        backup_size=$(du -sh "$backup_dir" 2>/dev/null | cut -f1 || echo "N/A")
    fi
    
    echo "$data_size\t$backup_size"
}

#######################################
# Format output as table
#######################################
format_table_output() {
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    echo "PostgreSQL Resource Usage Monitor - $timestamp"
    echo "================================================================"
    printf "%-20s %-8s %-12s %-8s %-15s %-15s %-8s %-8s %-12s %-8s %-8s %-12s\n" \
        "Instance" "Status" "CPU%" "Memory%" "Memory" "Network I/O" "Block I/O" "Active" "Total" "DB Size" "Data" "Backups" "Avg Query"
    printf "%-20s %-8s %-12s %-8s %-15s %-15s %-8s %-8s %-12s %-8s %-8s %-12s\n" \
        "--------" "------" "----" "-------" "------" "-----------" "---------" "------" "-----" "-------" "----" "-------" "---------"
    
    # Get list of instances
    local instances
    if [[ -n "${SPECIFIC_INSTANCE:-}" ]]; then
        instances=("$SPECIFIC_INSTANCE")
    else
        mapfile -t instances < <(find "${POSTGRES_INSTANCES_DIR}" -maxdepth 1 -type d ! -path "${POSTGRES_INSTANCES_DIR}" -exec basename {} \; | sort)
    fi
    
    for instance in "${instances[@]}"; do
        local container_name="${POSTGRES_CONTAINER_PREFIX}-${instance}"
        local status="stopped"
        
        # Check if container is running
        if docker ps --format "table {{.Names}}" | grep -q "^${container_name}$"; then
            status="running"
        fi
        
        if [[ "$status" == "running" ]]; then
            # Get container stats
            local stats_line
            stats_line=$(get_container_stats "$container_name" | tail -1)
            local cpu_perc=$(echo "$stats_line" | cut -f2)
            local mem_usage=$(echo "$stats_line" | cut -f3)
            local mem_perc=$(echo "$stats_line" | cut -f4)
            local net_io=$(echo "$stats_line" | cut -f5)
            local block_io=$(echo "$stats_line" | cut -f6)
            
            # Get PostgreSQL metrics
            local pg_metrics
            pg_metrics=$(get_postgres_metrics "$instance")
            local active_conn=$(echo "$pg_metrics" | cut -f1)
            local total_conn=$(echo "$pg_metrics" | cut -f2)
            local db_size=$(echo "$pg_metrics" | cut -f3)
            local avg_query_time=$(echo "$pg_metrics" | cut -f5)
            
            # Get disk usage
            local disk_usage
            disk_usage=$(get_disk_usage "$instance")
            local data_size=$(echo "$disk_usage" | cut -f1)
            local backup_size=$(echo "$disk_usage" | cut -f2)
            
            printf "%-20s %-8s %-12s %-8s %-15s %-15s %-8s %-8s %-12s %-8s %-8s %-12s\n" \
                "$instance" "$status" "$cpu_perc" "$mem_perc" "$mem_usage" "$net_io" "$block_io" \
                "$active_conn" "$total_conn" "$db_size" "$data_size" "$backup_size" "${avg_query_time}ms"
        else
            # Instance is stopped
            local disk_usage
            disk_usage=$(get_disk_usage "$instance")
            local data_size=$(echo "$disk_usage" | cut -f1)
            local backup_size=$(echo "$disk_usage" | cut -f2)
            
            printf "%-20s %-8s %-12s %-8s %-15s %-15s %-8s %-8s %-12s %-8s %-8s %-12s\n" \
                "$instance" "$status" "N/A" "N/A" "N/A" "N/A" "N/A" "N/A" "N/A" "N/A" "$data_size" "$backup_size" "N/A"
        fi
    done
    
    echo ""
}

#######################################
# Format output as JSON
#######################################
format_json_output() {
    local timestamp=$(date -Iseconds)
    
    echo "{"
    echo "  \"timestamp\": \"$timestamp\","
    echo "  \"instances\": ["
    
    # Get list of instances
    local instances
    if [[ -n "${SPECIFIC_INSTANCE:-}" ]]; then
        instances=("$SPECIFIC_INSTANCE")
    else
        mapfile -t instances < <(find "${POSTGRES_INSTANCES_DIR}" -maxdepth 1 -type d ! -path "${POSTGRES_INSTANCES_DIR}" -exec basename {} \; | sort)
    fi
    
    local first=true
    for instance in "${instances[@]}"; do
        if [[ "$first" != true ]]; then
            echo ","
        fi
        first=false
        
        local container_name="${POSTGRES_CONTAINER_PREFIX}-${instance}"
        local status="stopped"
        
        # Check if container is running
        if docker ps --format "table {{.Names}}" | grep -q "^${container_name}$"; then
            status="running"
        fi
        
        echo "    {"
        echo "      \"name\": \"$instance\","
        echo "      \"status\": \"$status\","
        
        if [[ "$status" == "running" ]]; then
            # Get container stats
            local stats_line
            stats_line=$(get_container_stats "$container_name" | tail -1)
            local cpu_perc=$(echo "$stats_line" | cut -f2 | tr -d '%')
            local mem_usage=$(echo "$stats_line" | cut -f3)
            local mem_perc=$(echo "$stats_line" | cut -f4 | tr -d '%')
            local net_io=$(echo "$stats_line" | cut -f5)
            local block_io=$(echo "$stats_line" | cut -f6)
            
            # Get PostgreSQL metrics
            local pg_metrics
            pg_metrics=$(get_postgres_metrics "$instance")
            local active_conn=$(echo "$pg_metrics" | cut -f1)
            local total_conn=$(echo "$pg_metrics" | cut -f2)
            local db_size=$(echo "$pg_metrics" | cut -f3)
            local avg_query_time=$(echo "$pg_metrics" | cut -f5)
            
            echo "      \"cpu_percent\": \"$cpu_perc\","
            echo "      \"memory_usage\": \"$mem_usage\","
            echo "      \"memory_percent\": \"$mem_perc\","
            echo "      \"network_io\": \"$net_io\","
            echo "      \"block_io\": \"$block_io\","
            echo "      \"active_connections\": \"$active_conn\","
            echo "      \"total_connections\": \"$total_conn\","
            echo "      \"database_size\": \"$db_size\","
            echo "      \"avg_query_time_ms\": \"$avg_query_time\""
        else
            echo "      \"cpu_percent\": null,"
            echo "      \"memory_usage\": null,"
            echo "      \"memory_percent\": null,"
            echo "      \"network_io\": null,"
            echo "      \"block_io\": null,"
            echo "      \"active_connections\": null,"
            echo "      \"total_connections\": null,"
            echo "      \"database_size\": null,"
            echo "      \"avg_query_time_ms\": null"
        fi
        
        # Get disk usage
        local disk_usage
        disk_usage=$(get_disk_usage "$instance")
        local data_size=$(echo "$disk_usage" | cut -f1)
        local backup_size=$(echo "$disk_usage" | cut -f2)
        
        echo "      \"data_size\": \"$data_size\","
        echo "      \"backup_size\": \"$backup_size\""
        echo -n "    }"
    done
    
    echo ""
    echo "  ]"
    echo "}"
}

#######################################
# Format output as CSV
#######################################
format_csv_output() {
    local timestamp=$(date -Iseconds)
    
    # CSV header
    echo "timestamp,instance,status,cpu_percent,memory_usage,memory_percent,network_io,block_io,active_connections,total_connections,database_size,data_size,backup_size,avg_query_time_ms"
    
    # Get list of instances
    local instances
    if [[ -n "${SPECIFIC_INSTANCE:-}" ]]; then
        instances=("$SPECIFIC_INSTANCE")
    else
        mapfile -t instances < <(find "${POSTGRES_INSTANCES_DIR}" -maxdepth 1 -type d ! -path "${POSTGRES_INSTANCES_DIR}" -exec basename {} \; | sort)
    fi
    
    for instance in "${instances[@]}"; do
        local container_name="${POSTGRES_CONTAINER_PREFIX}-${instance}"
        local status="stopped"
        
        # Check if container is running
        if docker ps --format "table {{.Names}}" | grep -q "^${container_name}$"; then
            status="running"
        fi
        
        if [[ "$status" == "running" ]]; then
            # Get container stats
            local stats_line
            stats_line=$(get_container_stats "$container_name" | tail -1)
            local cpu_perc=$(echo "$stats_line" | cut -f2 | tr -d '%')
            local mem_usage=$(echo "$stats_line" | cut -f3 | tr -d ' ')
            local mem_perc=$(echo "$stats_line" | cut -f4 | tr -d '%')
            local net_io=$(echo "$stats_line" | cut -f5 | tr -d ' ')
            local block_io=$(echo "$stats_line" | cut -f6 | tr -d ' ')
            
            # Get PostgreSQL metrics
            local pg_metrics
            pg_metrics=$(get_postgres_metrics "$instance")
            local active_conn=$(echo "$pg_metrics" | cut -f1)
            local total_conn=$(echo "$pg_metrics" | cut -f2)
            local db_size=$(echo "$pg_metrics" | cut -f3)
            local avg_query_time=$(echo "$pg_metrics" | cut -f5)
            
            # Get disk usage
            local disk_usage
            disk_usage=$(get_disk_usage "$instance")
            local data_size=$(echo "$disk_usage" | cut -f1)
            local backup_size=$(echo "$disk_usage" | cut -f2)
            
            echo "$timestamp,$instance,$status,$cpu_perc,\"$mem_usage\",$mem_perc,\"$net_io\",\"$block_io\",$active_conn,$total_conn,\"$db_size\",\"$data_size\",\"$backup_size\",$avg_query_time"
        else
            # Instance is stopped
            local disk_usage
            disk_usage=$(get_disk_usage "$instance")
            local data_size=$(echo "$disk_usage" | cut -f1)
            local backup_size=$(echo "$disk_usage" | cut -f2)
            
            echo "$timestamp,$instance,$status,,,,,,,,,\"$data_size\",\"$backup_size\","
        fi
    done
}

#######################################
# Export metrics to file
#######################################
export_metrics() {
    local output
    
    case "$OUTPUT_FORMAT" in
        "json")
            output=$(format_json_output)
            ;;
        "csv")
            output=$(format_csv_output)
            ;;
        *)
            echo "❌ Error: Export only supports json and csv formats" >&2
            exit 1
            ;;
    esac
    
    if [[ -n "$METRICS_FILE" ]]; then
        echo "$output" > "$METRICS_FILE"
        echo "✅ Metrics exported to: $METRICS_FILE"
    else
        echo "$output"
    fi
}

#######################################
# Run continuous monitoring
#######################################
run_continuous_monitoring() {
    echo "🔄 Starting continuous monitoring (Press Ctrl+C to stop)"
    echo "Interval: ${MONITOR_INTERVAL}s | Format: $OUTPUT_FORMAT"
    echo ""
    
    # Trap Ctrl+C
    trap 'echo -e "\n👋 Monitoring stopped"; exit 0' INT
    
    while true; do
        case "$OUTPUT_FORMAT" in
            "table")
                clear
                format_table_output
                ;;
            "json")
                format_json_output
                echo ""
                ;;
            "csv")
                format_csv_output
                echo ""
                ;;
        esac
        
        sleep "$MONITOR_INTERVAL"
    done
}

#######################################
# Show resource usage summary
#######################################
show_summary() {
    echo "📊 PostgreSQL Resource Usage Summary"
    echo "==================================="
    
    # Count instances
    local total_instances running_instances stopped_instances
    total_instances=$(find "${POSTGRES_INSTANCES_DIR}" -maxdepth 1 -type d ! -path "${POSTGRES_INSTANCES_DIR}" | wc -l)
    running_instances=$(docker ps --format "table {{.Names}}" | grep -c "^${POSTGRES_CONTAINER_PREFIX}-" || echo "0")
    stopped_instances=$((total_instances - running_instances))
    
    echo "Total Instances:   $total_instances"
    echo "Running:           $running_instances"
    echo "Stopped:           $stopped_instances"
    echo ""
    
    # Total disk usage
    local total_data_size total_backup_size
    total_data_size=$(du -sh "${POSTGRES_INSTANCES_DIR}" 2>/dev/null | cut -f1 || echo "N/A")
    total_backup_size=$(du -sh "${POSTGRES_BACKUP_DIR}" 2>/dev/null | cut -f1 || echo "N/A")
    
    echo "Total Data Size:   $total_data_size"
    echo "Total Backup Size: $total_backup_size"
    echo ""
}

#######################################
# Main execution
#######################################
main() {
    parse_args "$@"
    
    # Validate output format
    case "$OUTPUT_FORMAT" in
        "table"|"json"|"csv") ;;
        *)
            echo "❌ Error: Invalid output format: $OUTPUT_FORMAT" >&2
            echo "Valid formats: table, json, csv" >&2
            exit 1
            ;;
    esac
    
    # Check if Docker is running
    if ! docker info >/dev/null 2>&1; then
        echo "❌ Error: Docker is not running" >&2
        exit 1
    fi
    
    # Handle different modes
    if [[ "$EXPORT_METRICS" == "yes" ]]; then
        export_metrics
    elif [[ "$CONTINUOUS_MODE" == "yes" ]]; then
        run_continuous_monitoring
    else
        # One-time monitoring
        if [[ "$OUTPUT_FORMAT" == "table" ]]; then
            show_summary
        fi
        
        case "$OUTPUT_FORMAT" in
            "table")
                format_table_output
                ;;
            "json")
                format_json_output
                ;;
            "csv")
                format_csv_output
                ;;
        esac
    fi
}

# Run main function with all arguments
main "$@"