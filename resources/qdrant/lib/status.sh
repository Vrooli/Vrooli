#!/usr/bin/env bash
# Qdrant Status Management - Standardized Format
# Functions for checking and displaying Qdrant status information

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
QDRANT_STATUS_DIR="${APP_ROOT}/resources/qdrant/lib"

# Source format utilities and config
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/format.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/status-args.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-utils.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/resources/qdrant/config/defaults.sh"
# shellcheck disable=SC1091
source "${QDRANT_STATUS_DIR}/core.sh"
# shellcheck disable=SC1091
source "${QDRANT_STATUS_DIR}/health.sh"

# Ensure configuration is exported
if command -v qdrant::export_config &>/dev/null; then
    qdrant::export_config 2>/dev/null || true
fi

#######################################
# Collect Qdrant status data in format-agnostic structure
# Returns: Key-value pairs ready for formatting
#######################################
qdrant::status::collect_data() {
    local fast_mode="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --fast)
                fast_mode="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    local status_data=()
    
    # Basic status checks
    local installed="false"
    local running="false"
    local healthy="false"
    local container_status="not_found"
    local health_message="Unknown"
    
    if docker::container_exists "$QDRANT_CONTAINER_NAME"; then
        installed="true"
        container_status=$(docker inspect --format='{{.State.Status}}' "$QDRANT_CONTAINER_NAME" 2>/dev/null || echo "unknown")
        
        if docker::is_running "$QDRANT_CONTAINER_NAME"; then
            running="true"
            
            if qdrant::health::is_healthy; then
                healthy="true"
                health_message="Healthy - All systems operational, vector database ready"
            else
                health_message="Unhealthy - Service not responding or API degraded"
            fi
        else
            health_message="Stopped - Container not running"
        fi
    else
        health_message="Not installed - Container does not exist"
    fi
    
    # Basic resource information
    status_data+=("name" "qdrant")
    status_data+=("category" "storage")
    status_data+=("description" "High-performance vector database with full-text search")
    status_data+=("installed" "$installed")
    status_data+=("running" "$running")
    status_data+=("healthy" "$healthy")
    status_data+=("health_message" "$health_message")
    status_data+=("container_name" "$QDRANT_CONTAINER_NAME")
    status_data+=("container_status" "$container_status")
    status_data+=("port" "$QDRANT_PORT")
    
    # Service endpoints
    status_data+=("api_url" "$QDRANT_BASE_URL")
    status_data+=("dashboard_url" "$QDRANT_BASE_URL/dashboard")
    status_data+=("metrics_url" "$QDRANT_BASE_URL/metrics")
    status_data+=("health_url" "$QDRANT_BASE_URL/")
    
    # Configuration details
    status_data+=("image" "$QDRANT_IMAGE")
    status_data+=("data_dir" "$QDRANT_DATA_DIR")
    status_data+=("api_key" "${QDRANT_API_KEY:+configured}")
    
    # Runtime information (only if running and healthy)
    if [[ "$running" == "true" && "$healthy" == "true" ]]; then
        # Get cluster info
        local cluster_info
        cluster_info=$(qdrant::client::get_cluster_info 2>/dev/null)
        
        if [[ -n "$cluster_info" ]]; then
            # Parse cluster info
            local status_json
            status_json=$(echo "$cluster_info" | jq -c '.result' 2>/dev/null)
            
            if [[ -n "$status_json" && "$status_json" != "null" ]]; then
                # Extract version
                local version
                version=$(echo "$status_json" | jq -r '.version // "unknown"' 2>/dev/null)
                status_data+=("version" "${version:-unknown}")
                
                # Extract peer count
                local peer_count
                peer_count=$(echo "$status_json" | jq -r '.peers | length // 0' 2>/dev/null)
                status_data+=("peer_count" "${peer_count:-0}")
                
                # Extract peer id  
                local peer_id
                peer_id=$(echo "$status_json" | jq -r '.peer_id // "unknown"' 2>/dev/null)
                status_data+=("peer_id" "${peer_id:-unknown}")
            fi
        fi
        
        # Get collection count
        local collections_info
        collections_info=$(qdrant::client::get_collections 2>/dev/null)
        
        if [[ -n "$collections_info" ]]; then
            local collection_count
            collection_count=$(echo "$collections_info" | jq -r '.result.collections | length // 0' 2>/dev/null)
            status_data+=("collection_count" "${collection_count:-0}")
            
            # Get total point count across all collections (optimized for performance)
            local total_points=0
            
            # Skip expensive operations in fast mode
            local skip_expensive_ops="$fast_mode"
            
            if [[ "$skip_expensive_ops" == "false" && "${collection_count:-0}" -gt 0 && "${collection_count:-0}" -le 10 ]]; then
                # Only iterate through collections if there are 10 or fewer (to avoid timeout)
                local collections_array
                collections_array=$(echo "$collections_info" | jq -r '.result.collections[].name' 2>/dev/null)
                
                while IFS= read -r collection_name; do
                    [[ -z "$collection_name" ]] && continue
                    local collection_detail
                    collection_detail=$(qdrant::client::get_collection_info "$collection_name" 2>/dev/null)
                    
                    if [[ -n "$collection_detail" ]]; then
                        local points_count
                        points_count=$(echo "$collection_detail" | jq -r '.result.points_count // 0' 2>/dev/null)
                        total_points=$((total_points + ${points_count:-0}))
                    fi
                done <<< "$collections_array"
            elif [[ "${collection_count:-0}" -gt 10 ]]; then
                # Too many collections, skip detailed count for performance
                total_points="10000+"
            fi
            status_data+=("total_points" "$total_points")
        else
            status_data+=("collection_count" "0")
            status_data+=("total_points" "0")
        fi
        
        # Get memory and CPU usage (optimized with smart skipping)
        local stats_output memory_usage cpu_usage
        
        # Reuse skip_expensive_ops from above or use fast mode
        if [[ -z "${skip_expensive_ops:-}" ]]; then
            skip_expensive_ops="$fast_mode"
        fi
        
        if [[ "$skip_expensive_ops" == "true" ]]; then
            memory_usage="N/A"
            cpu_usage="N/A"
        else
            stats_output=$(timeout 2s docker stats "$QDRANT_CONTAINER_NAME" --no-stream --format "{{.MemUsage}}\t{{.CPUPerc}}" 2>/dev/null || echo "N/A\tN/A")
            
            if [[ "$stats_output" != "N/A"*"N/A" ]]; then
                memory_usage=$(echo "$stats_output" | cut -f1 | cut -d' ' -f1)
                cpu_usage=$(echo "$stats_output" | cut -f2)
            else
                memory_usage="N/A"
                cpu_usage="N/A"
            fi
        fi
        
        status_data+=("memory_usage" "${memory_usage:-N/A}")
        status_data+=("cpu_usage" "${cpu_usage:-N/A}")
        
        # Calculate uptime
        local created_time
        created_time=$(docker inspect --format='{{.Created}}' "$QDRANT_CONTAINER_NAME" 2>/dev/null)
        if [[ -n "$created_time" ]]; then
            local created_timestamp
            created_timestamp=$(date -d "$created_time" +%s 2>/dev/null)
            if [[ -n "$created_timestamp" ]]; then
                local current_timestamp
                current_timestamp=$(date +%s)
                local uptime_seconds=$((current_timestamp - created_timestamp))
                local uptime_human
                uptime_human=$(qdrant::status::format_uptime "$uptime_seconds")
                status_data+=("uptime" "$uptime_human")
            fi
        fi
    fi
    
    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}

#######################################
# Show Qdrant status using standardized format
# Args: [--format json|text] [--verbose]
#######################################
qdrant::status() {
    status::run_standard "qdrant" "qdrant::status::collect_data" "qdrant::status::display_text" "$@"
}

#######################################
# Display status in text format
#######################################
qdrant::status::display_text() {
    local -A data
    
    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        if [[ $value_idx -le $# ]]; then
            local value="${!value_idx}"
            data["$key"]="$value"
        fi
    done
    
    # Header
    log::header "📊 Qdrant Status"
    echo
    
    # Basic status
    log::info "📊 Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
        echo
        log::info "💡 Installation Required:"
        log::info "   To install Qdrant, run: resource-qdrant install"
        return
    fi
    
    if [[ "${data[running]:-false}" == "true" ]]; then
        log::success "   ✅ Running: Yes"
    else
        log::warn "   ⚠️  Running: No"
    fi
    
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        log::success "   ✅ Health: Healthy"
    else
        log::warn "   ⚠️  Health: ${data[health_message]:-Unknown}"
    fi
    echo
    
    # Container information
    log::info "🐳 Container Info:"
    log::info "   📦 Name: ${data[container_name]:-unknown}"
    log::info "   📊 Status: ${data[container_status]:-unknown}"
    log::info "   🖼️  Image: ${data[image]:-unknown}"
    if [[ "${data[running]:-false}" == "true" ]]; then
        log::info "   💾 Memory: ${data[memory_usage]:-N/A}"
        log::info "   🔥 CPU: ${data[cpu_usage]:-N/A}"
        log::info "   ⏱️  Uptime: ${data[uptime]:-N/A}"
    fi
    echo
    
    # Service endpoints
    log::info "🌐 Service Endpoints:"
    log::info "   🔌 API: ${data[api_url]:-unknown}"
    log::info "   📊 Dashboard: ${data[dashboard_url]:-unknown}"
    log::info "   📈 Metrics: ${data[metrics_url]:-unknown}"
    log::info "   🏥 Health: ${data[health_url]:-unknown}"
    echo
    
    # Configuration
    log::info "⚙️  Configuration:"
    log::info "   📶 Port: ${data[port]:-unknown}"
    log::info "   📁 Data Directory: ${data[data_dir]:-unknown}"
    if [[ -n "${data[api_key]:-}" ]]; then
        log::info "   🔐 API Key: Configured"
    else
        log::info "   🔓 API Key: Not configured"
    fi
    echo
    
    # Runtime information (only if healthy)
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        log::info "📈 Runtime Information:"
        log::info "   🔖 Version: ${data[version]:-unknown}"
        log::info "   👥 Peer ID: ${data[peer_id]:-unknown}"
        log::info "   🔗 Peer Count: ${data[peer_count]:-0}"
        log::info "   📂 Collections: ${data[collection_count]:-0}"
        log::info "   📍 Total Points: ${data[total_points]:-0}"
    fi
}

#######################################
# Format uptime seconds to human readable
# Arguments:
#   $1 - uptime in seconds
# Returns: Human readable uptime
#######################################
qdrant::status::format_uptime() {
    local seconds=$1
    local days=$((seconds / 86400))
    local hours=$(((seconds % 86400) / 3600))
    local minutes=$(((seconds % 3600) / 60))
    local secs=$((seconds % 60))
    
    if [[ $days -gt 0 ]]; then
        echo "${days}d ${hours}h ${minutes}m ${secs}s"
    elif [[ $hours -gt 0 ]]; then
        echo "${hours}h ${minutes}m ${secs}s"
    elif [[ $minutes -gt 0 ]]; then
        echo "${minutes}m ${secs}s"
    else
        echo "${secs}s"
    fi
}