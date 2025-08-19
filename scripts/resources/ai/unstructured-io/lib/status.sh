#!/usr/bin/env bash
# Unstructured.io Status Management - Standardized Format
# Functions for checking and displaying Unstructured.io status information

# Source format utilities and config
UNSTRUCTURED_IO_STATUS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${UNSTRUCTURED_IO_STATUS_DIR}/../../../../lib/utils/format.sh"
# shellcheck disable=SC1091
source "${UNSTRUCTURED_IO_STATUS_DIR}/../config/defaults.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${UNSTRUCTURED_IO_STATUS_DIR}/../config/messages.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${UNSTRUCTURED_IO_STATUS_DIR}/common.sh" 2>/dev/null || true

# Ensure configuration is exported
if command -v unstructured_io::export_config &>/dev/null; then
    unstructured_io::export_config 2>/dev/null || true
fi
if command -v unstructured_io::messages::init &>/dev/null; then
    unstructured_io::messages::init 2>/dev/null || true
fi

#######################################
# Collect Unstructured.io status data in format-agnostic structure
# Returns: Key-value pairs ready for formatting
#######################################
unstructured_io::status::collect_data() {
    local status_data=()
    
    # Basic status checks
    local installed="false"
    local running="false"
    local healthy="false"
    local container_status="not_found"
    local health_message="Unknown"
    
    if unstructured_io::container_exists; then
        installed="true"
        container_status=$(docker inspect --format='{{.State.Status}}' "${UNSTRUCTURED_IO_CONTAINER_NAME}" 2>/dev/null || echo "unknown")
        
        if unstructured_io::container_running; then
            running="true"
            
            if unstructured_io::check_health; then
                healthy="true"
                health_message="Healthy - API responding and ready for document processing"
            else
                health_message="Unhealthy - Service not responding or degraded"
            fi
        else
            health_message="Stopped - Container not running"
        fi
    else
        health_message="Not installed - Container does not exist"
    fi
    
    # Basic resource information
    status_data+=("name" "unstructured-io")
    status_data+=("category" "ai")
    status_data+=("description" "Document processing and text extraction service")
    status_data+=("installed" "$installed")
    status_data+=("running" "$running")
    status_data+=("healthy" "$healthy")
    status_data+=("health_message" "$health_message")
    status_data+=("container_name" "$UNSTRUCTURED_IO_CONTAINER_NAME")
    status_data+=("container_status" "$container_status")
    status_data+=("port" "$UNSTRUCTURED_IO_PORT")
    
    # Service endpoints
    status_data+=("base_url" "$UNSTRUCTURED_IO_BASE_URL")
    status_data+=("health_endpoint" "${UNSTRUCTURED_IO_BASE_URL}${UNSTRUCTURED_IO_HEALTH_ENDPOINT}")
    status_data+=("process_endpoint" "${UNSTRUCTURED_IO_BASE_URL}${UNSTRUCTURED_IO_PROCESS_ENDPOINT}")
    status_data+=("batch_endpoint" "${UNSTRUCTURED_IO_BASE_URL}${UNSTRUCTURED_IO_BATCH_ENDPOINT}")
    
    # Configuration details
    status_data+=("image" "$UNSTRUCTURED_IO_IMAGE")
    status_data+=("memory_limit" "$UNSTRUCTURED_IO_MEMORY_LIMIT")
    status_data+=("cpu_limit" "$UNSTRUCTURED_IO_CPU_LIMIT")
    status_data+=("default_strategy" "$UNSTRUCTURED_IO_DEFAULT_STRATEGY")
    status_data+=("max_file_size" "$UNSTRUCTURED_IO_MAX_FILE_SIZE")
    status_data+=("timeout_seconds" "$UNSTRUCTURED_IO_TIMEOUT_SECONDS")
    status_data+=("max_concurrent" "$UNSTRUCTURED_IO_MAX_CONCURRENT_REQUESTS")
    
    # Supported formats count
    local format_count=${#UNSTRUCTURED_IO_SUPPORTED_FORMATS[@]}
    status_data+=("supported_formats_count" "$format_count")
    
    # Runtime information (only if running and healthy)
    if [[ "$running" == "true" && "$healthy" == "true" ]]; then
        # Get container resource usage with fast timeout (ultra-optimized)
        local container_stats cpu_usage memory_usage
        
        # Auto-detect if we should skip expensive docker stats
        # Skip stats if: explicitly requested, in parallel mode, or format is JSON (likely automated)
        local skip_stats="false"
        if [[ "${UNSTRUCTURED_IO_SKIP_STATS:-false}" == "true" ]] || \
           [[ "${PARALLEL_STATUS_CHECK:-false}" == "true" ]] || \
           [[ "${format:-}" == "json" && "${verbose:-}" != "true" ]]; then
            skip_stats="true"
        fi
        
        if [[ "$skip_stats" == "true" ]]; then
            cpu_usage="N/A"
            memory_usage="N/A"
        else
            container_stats=$(timeout 2s docker stats --no-stream --format "{{.CPUPerc}}|{{.MemUsage}}" "$UNSTRUCTURED_IO_CONTAINER_NAME" 2>/dev/null || echo "N/A|N/A")
            
            if [[ "$container_stats" != "N/A|N/A" ]]; then
                cpu_usage=$(echo "$container_stats" | cut -d'|' -f1)
                memory_usage=$(echo "$container_stats" | cut -d'|' -f2)
            else
                cpu_usage="N/A"
                memory_usage="N/A"
            fi
        fi
        
        status_data+=("cpu_usage" "$cpu_usage")
        status_data+=("memory_usage" "$memory_usage")
        
        # Get container uptime
        local started_at
        started_at=$(docker inspect --format='{{.State.StartedAt}}' "$UNSTRUCTURED_IO_CONTAINER_NAME" 2>/dev/null)
        if [[ -n "$started_at" ]]; then
            status_data+=("started_at" "$started_at")
        fi
        
        # Get image creation date
        local image_created
        image_created=$(docker inspect --format='{{.Created}}' "$UNSTRUCTURED_IO_IMAGE" 2>/dev/null)
        if [[ -n "$image_created" ]]; then
            status_data+=("image_created" "$image_created")
        fi
    fi
    
    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}

#######################################
# Show Unstructured.io status using standardized format
# Args: [--format json|text] [--verbose]
#######################################
unstructured_io::status::show() {
    local format="text"
    local verbose="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --format)
                format="$2"
                shift 2
                ;;
            --json)
                format="json"
                shift
                ;;
            --verbose|-v)
                verbose="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Collect status data
    local data_string
    data_string=$(unstructured_io::status::collect_data 2>/dev/null)
    
    if [[ -z "$data_string" ]]; then
        # Fallback if data collection fails
        if [[ "$format" == "json" ]]; then
            echo '{"error": "Failed to collect status data"}'
        else
            log::error "Failed to collect Unstructured.io status data"
        fi
        return 1
    fi
    
    # Convert string to array
    local data_array
    mapfile -t data_array <<< "$data_string"
    
    # Output based on format
    if [[ "$format" == "json" ]]; then
        format::output "json" "kv" "${data_array[@]}"
    else
        # Text format with standardized structure
        unstructured_io::status::display_text "${data_array[@]}"
    fi
    
    # Return appropriate exit code
    local healthy="false"
    local running="false"
    for ((i=0; i<${#data_array[@]}; i+=2)); do
        case "${data_array[i]}" in
            "healthy") healthy="${data_array[i+1]}" ;;
            "running") running="${data_array[i+1]}" ;;
        esac
    done
    
    if [[ "$healthy" == "true" ]]; then
        return 0
    elif [[ "$running" == "true" ]]; then
        return 1
    else
        return 2
    fi
}

#######################################
# Display status in text format
#######################################
unstructured_io::status::display_text() {
    local -A data
    
    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        local value="${!value_idx}"
        data["$key"]="$value"
    done
    
    # Header
    log::header "📄 Unstructured.io Status"
    echo
    
    # Basic status
    log::info "📊 Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
        echo
        log::info "💡 Installation Required:"
        log::info "   To install Unstructured.io, run: ./manage.sh --action install"
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
    
    if [[ "${data[running]:-false}" == "true" && -n "${data[cpu_usage]:-}" ]]; then
        log::info "   💻 CPU Usage: ${data[cpu_usage]:-N/A}"
        log::info "   💾 Memory Usage: ${data[memory_usage]:-N/A}"
    fi
    echo
    
    # Service endpoints
    log::info "🌐 Service Endpoints:"
    log::info "   🔗 Base URL: ${data[base_url]:-unknown}"
    log::info "   🏥 Health Check: ${data[health_endpoint]:-unknown}"
    log::info "   📄 Process Endpoint: ${data[process_endpoint]:-unknown}"
    log::info "   📦 Batch Endpoint: ${data[batch_endpoint]:-unknown}"
    echo
    
    # Configuration
    log::info "⚙️  Configuration:"
    log::info "   📶 Port: ${data[port]:-unknown}"
    log::info "   💾 Memory Limit: ${data[memory_limit]:-unknown}"
    log::info "   🖥️  CPU Limit: ${data[cpu_limit]:-unknown}"
    log::info "   🎯 Default Strategy: ${data[default_strategy]:-unknown}"
    log::info "   📏 Max File Size: ${data[max_file_size]:-unknown}"
    log::info "   ⏱️  Timeout: ${data[timeout_seconds]:-unknown}s"
    log::info "   🔀 Max Concurrent: ${data[max_concurrent]:-unknown}"
    log::info "   📋 Supported Formats: ${data[supported_formats_count]:-0}"
    
    # Runtime information (only if healthy)
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        echo
        log::info "📈 Runtime Information:"
        if [[ -n "${data[started_at]:-}" ]]; then
            local started_date
            started_date=$(date -d "${data[started_at]}" "+%Y-%m-%d %H:%M:%S" 2>/dev/null || echo "${data[started_at]}")
            log::info "   🚀 Started: $started_date"
        fi
        if [[ -n "${data[image_created]:-}" ]]; then
            local image_date
            image_date=$(date -d "${data[image_created]}" "+%Y-%m-%d %H:%M:%S" 2>/dev/null || echo "${data[image_created]}")
            log::info "   📅 Image Created: $image_date"
        fi
        
        echo
        log::info "💡 Usage Examples:"
        log::info "   Process file: curl -X POST '${data[process_endpoint]:-}' -F 'files=@document.pdf'"
        log::info "   Health check: curl '${data[health_endpoint]:-}'"
    fi
}

#######################################
# Legacy compatibility function
#######################################
unstructured_io::status() {
    # Parse arguments for backwards compatibility
    local verbose="yes"
    local format="text"
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            "no")
                verbose="no"
                shift
                ;;
            --json)
                format="json"
                shift
                ;;
            --format)
                format="$2"
                shift 2
                ;;
            --verbose|-v)
                verbose="yes"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Call the new standardized function
    if [[ "$verbose" == "no" && "$format" == "text" ]]; then
        # Silent mode for backwards compatibility
        unstructured_io::status::show --format text >/dev/null 2>&1
        return $?
    else
        unstructured_io::status::show --format "$format" ${verbose:+--verbose}
    fi
}

#######################################
# Check API health
#######################################
unstructured_io::check_health() {
    local response
    response=$(curl -s -w "\n%{http_code}" "${UNSTRUCTURED_IO_BASE_URL}${UNSTRUCTURED_IO_HEALTH_ENDPOINT}" 2>/dev/null)
    
    local http_code=$(echo "$response" | tail -n1)
    local body=$(echo "$response" | head -n-1)
    
    if [ "$http_code" = "200" ]; then
        return 0
    else
        return 1
    fi
}

#######################################
# Display service information
#######################################
unstructured_io::info() {
    echo "📄 Unstructured.io Service Information"
    echo "======================================"
    echo
    echo "Service URL: $UNSTRUCTURED_IO_BASE_URL"
    echo "Container: $UNSTRUCTURED_IO_CONTAINER_NAME"
    echo "Docker Image: $UNSTRUCTURED_IO_IMAGE"
    echo
    echo "$MSG_CONFIG_STRATEGY"
    echo "$MSG_CONFIG_LANGUAGES"
    echo "$MSG_CONFIG_MEMORY"
    echo "$MSG_CONFIG_CPU"
    echo "$MSG_CONFIG_TIMEOUT"
    echo
    echo "$MSG_API_ENDPOINT_INFO"
    echo "$MSG_API_PROCESS"
    echo "$MSG_API_BATCH"
    echo "$MSG_API_HEALTH"
    echo "$MSG_API_METRICS"
    echo
    echo "Supported Document Formats:"
    echo "=========================="
    local count=0
    for format in "${UNSTRUCTURED_IO_SUPPORTED_FORMATS[@]}"; do
        printf "%-8s" ".$format"
        ((count++))
        if [ $((count % 8)) -eq 0 ]; then
            echo
        fi
    done
    [ $((count % 8)) -ne 0 ] && echo
    echo
    
    # Show current status
    unstructured_io::status::show
}

#######################################
# View container logs
#######################################
unstructured_io::logs() {
    local follow="${1:-no}"
    
    if ! unstructured_io::container_exists; then
        echo "$MSG_UNSTRUCTURED_IO_NOT_FOUND"
        return 1
    fi
    
    echo "📋 Unstructured.io Container Logs"
    echo "================================="
    
    if [ "$follow" = "yes" ]; then
        docker logs -f "$UNSTRUCTURED_IO_CONTAINER_NAME"
    else
        docker logs --tail 50 "$UNSTRUCTURED_IO_CONTAINER_NAME"
    fi
}

#######################################
# Get processing metrics
#######################################
unstructured_io::metrics() {
    if ! unstructured_io::status "no"; then
        echo "❌ Unstructured.io is not running"
        return 1
    fi
    
    echo "📊 Processing Metrics"
    echo "===================="
    
    local response
    response=$(curl -s "${UNSTRUCTURED_IO_BASE_URL}${UNSTRUCTURED_IO_METRICS_ENDPOINT}" 2>/dev/null)
    
    if [ $? -eq 0 ]; then
        echo "$response" | grep -E "^(# HELP|# TYPE|unstructured_)" | head -20
    else
        echo "❌ Failed to retrieve metrics"
        return 1
    fi
}

#######################################
# Get container resource usage
#######################################
unstructured_io::resource_usage() {
    if ! unstructured_io::container_running; then
        echo "Container is not running"
        return 1
    fi
    
    echo "💻 Resource Usage"
    echo "================"
    
    docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}" \
        "$UNSTRUCTURED_IO_CONTAINER_NAME"
}

#######################################
# Export functions for subshell availability
#######################################
export -f unstructured_io::status
export -f unstructured_io::status::show
export -f unstructured_io::status::collect_data
export -f unstructured_io::status::display_text
export -f unstructured_io::check_health
export -f unstructured_io::resource_usage