#!/usr/bin/env bash
################################################################################
# Vrooli CLI - System Status Command
# 
# Provides a comprehensive health check and status overview of the Vrooli system.
# Displays resources, apps, and system diagnostics in a concise format.
#
# Usage:
#   vrooli status [options]
#
################################################################################

set -uo pipefail

# Get CLI directory
CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${CLI_DIR}/../../scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE:-${CLI_DIR}/../../scripts/lib/utils/log.sh}"
# shellcheck disable=SC1091
source "${var_JSON_FILE:-${CLI_DIR}/../../scripts/lib/utils/json.sh}"
# shellcheck disable=SC1091
source "${CLI_DIR}/../../scripts/lib/utils/format.sh"

# Configuration paths
RESOURCES_CONFIG="${var_ROOT_DIR}/.vrooli/service.json"
API_BASE="http://localhost:${VROOLI_API_PORT:-8090}"

# Show help for status command
show_status_help() {
    cat << EOF
🚀 Vrooli Status Command

USAGE:
    vrooli status [options]

OPTIONS:
    --json              Output in JSON format
    --verbose, -v       Show detailed information
    --resources         Show only resource status
    --apps              Show only app status
    --help, -h          Show this help message

EXAMPLES:
    vrooli status                  # Show full system status
    vrooli status --json           # Output as JSON
    vrooli status --resources      # Show only resource status
    vrooli status --apps           # Show only app status
    vrooli status --verbose        # Show detailed status

EOF
}

# Check if API is available
check_api() {
    if ! curl -s -f "${API_BASE}/health" >/dev/null 2>&1; then
        return 1
    fi
    return 0
}


# Get Docker container status
get_docker_status() {
    local container_name="$1"
    
    if docker ps --format "{{.Names}}" 2>/dev/null | grep -q "^${container_name}$"; then
        echo "running"
    else
        echo "stopped"
    fi
}

# Collect resource data (format-agnostic)
collect_resource_data() {
    local verbose="${1:-false}"
    
    # Check if config file exists
    if [[ ! -f "$RESOURCES_CONFIG" ]]; then
        echo "error:No resource configuration found at $RESOURCES_CONFIG"
        return
    fi
    
    # Get enabled resources
    local enabled_count=0
    local running_count=0
    
    # Parse resources from config
    local jq_query='
        .resources | to_entries[] | 
        .value | to_entries[] | 
        select(.value.enabled == true) | 
        "\(.key)"
    '
    
    while IFS= read -r line; do
        if [[ -n "$line" ]]; then
            local name="$line"
            
            # Check if resource is running
            local container_status
            container_status=$(get_docker_status "vrooli-${name}")
            
            if [[ "$container_status" == "running" ]]; then
                ((running_count++))
            fi
            ((enabled_count++))
            
            if [[ "$verbose" == "true" ]]; then
                echo "resource:${name}:${container_status}"
            fi
        fi
    done < <(jq -r "$jq_query" "$RESOURCES_CONFIG" 2>/dev/null || true)
    
    # Always output summary
    echo "enabled:${enabled_count}"
    echo "running:${running_count}"
}

# Get structured resource data (format-agnostic)
get_resource_data() {
    local verbose="${1:-false}"
    
    local raw_data
    raw_data=$(collect_resource_data "$verbose")
    
    # Check for errors
    if echo "$raw_data" | grep -q "^error:"; then
        local error_msg
        error_msg=$(echo "$raw_data" | grep "^error:" | cut -d: -f2-)
        echo "type:error"
        echo "message:$error_msg"
        return
    fi
    
    # Parse collected data
    local enabled_count running_count
    enabled_count=$(echo "$raw_data" | grep "^enabled:" | cut -d: -f2)
    running_count=$(echo "$raw_data" | grep "^running:" | cut -d: -f2)
    
    echo "type:status"
    echo "enabled:${enabled_count}"
    echo "running:${running_count}"
    echo "summary:${running_count}/${enabled_count} running"
    
    if [[ "$verbose" == "true" ]]; then
        echo "details:true"
        while IFS= read -r line; do
            if [[ "$line" =~ ^resource:(.+):(.+)$ ]]; then
                echo "item:${BASH_REMATCH[1]}:${BASH_REMATCH[2]}"
            fi
        done <<< "$raw_data"
    else
        echo "details:false"
    fi
}


# Collect app data (format-agnostic)
collect_app_data() {
    local verbose="${1:-false}"
    
    # Check if API is available
    if ! check_api; then
        echo "error:API not available"
        return
    fi
    
    # Get app list from API
    local response
    response=$(curl -s "${API_BASE}/apps" 2>/dev/null || echo '{"success": false}')
    
    if ! echo "$response" | jq -e '.success' >/dev/null 2>&1; then
        echo "error:Failed to get app list"
        return
    fi
    
    local total_apps running_apps
    total_apps=$(echo "$response" | jq '.data | length' 2>/dev/null || echo "0")
    running_apps=$(echo "$response" | jq '[.data[] | select(.runtime_status == "running")] | length' 2>/dev/null || echo "0")
    
    # Output summary
    echo "total:${total_apps}"
    echo "running:${running_apps}"
    
    # Output details if verbose
    if [[ "$verbose" == "true" && "$total_apps" -gt 0 ]]; then
        while IFS= read -r line; do
            IFS=':' read -r name status <<< "$line"
            echo "app:${name}:${status}"
        done < <(echo "$response" | jq -r '.data[] | "\(.name):\(.runtime_status // "stopped")"')
    fi
}

# Get structured app data (format-agnostic)
get_app_data() {
    local verbose="${1:-false}"
    
    local raw_data
    raw_data=$(collect_app_data "$verbose")
    
    # Check for errors
    if echo "$raw_data" | grep -q "^error:"; then
        local error_msg
        error_msg=$(echo "$raw_data" | grep "^error:" | cut -d: -f2-)
        echo "type:error"
        echo "message:$error_msg"
        return
    fi
    
    # Parse collected data
    local total_apps running_apps
    total_apps=$(echo "$raw_data" | grep "^total:" | cut -d: -f2)
    running_apps=$(echo "$raw_data" | grep "^running:" | cut -d: -f2)
    
    echo "type:status"
    echo "total:${total_apps}"
    echo "running:${running_apps}"
    echo "summary:${running_apps}/${total_apps} running"
    
    if [[ "$verbose" == "true" && "$total_apps" -gt 0 ]]; then
        echo "details:true"
        while IFS= read -r line; do
            if [[ "$line" =~ ^app:(.+):(.+)$ ]]; then
                echo "item:${BASH_REMATCH[1]}:${BASH_REMATCH[2]}"
            fi
        done <<< "$raw_data"
    else
        echo "details:false"
    fi
}


# Collect system data (format-agnostic)
collect_system_data() {
    # Get memory usage
    local mem_info
    mem_info=$(free -m | awk 'NR==2{printf "%.1f", $3*100/$2}')
    
    # Get disk usage for Vrooli directory
    local disk_info
    disk_info=$(df -h "${var_ROOT_DIR}" | awk 'NR==2{print $5}' | sed 's/%//')
    
    # Get load average
    local load_avg
    load_avg=$(uptime | awk -F'load average:' '{print $2}' | xargs)
    
    # Check Docker status
    local docker_status="healthy"
    if ! docker info >/dev/null 2>&1; then
        docker_status="unavailable"
    fi
    
    # Output raw data
    echo "memory:${mem_info}"
    echo "disk:${disk_info}"
    echo "load:${load_avg}"
    echo "docker:${docker_status}"
}

get_system_data() {
    local raw_data
    raw_data=$(collect_system_data)
    
    # Parse collected data
    local mem_info disk_info load_avg docker_status
    mem_info=$(echo "$raw_data" | grep "^memory:" | cut -d: -f2)
    disk_info=$(echo "$raw_data" | grep "^disk:" | cut -d: -f2)
    load_avg=$(echo "$raw_data" | grep "^load:" | cut -d: -f2-)
    docker_status=$(echo "$raw_data" | grep "^docker:" | cut -d: -f2)
    
    echo "type:system"
    echo "memory:${mem_info}"
    echo "disk:${disk_info}"
    echo "load:${load_avg}"
    echo "docker:${docker_status}"
}

################################################################################
# Status-Specific Formatters (using generic format.sh functions)
################################################################################

# Format system data for display
format_system_data() {
    local format="$1"
    local raw_data="$2"
    
    local memory disk load docker
    memory=$(echo "$raw_data" | grep "^memory:" | cut -d: -f2)
    disk=$(echo "$raw_data" | grep "^disk:" | cut -d: -f2)
    load=$(echo "$raw_data" | grep "^load:" | cut -d: -f2-)
    docker=$(echo "$raw_data" | grep "^docker:" | cut -d: -f2)
    
    if [[ "$format" == "json" ]]; then
        format::key_value json \
            memory_usage "${memory}%" \
            disk_usage "${disk}%" \
            load_average "${load}" \
            docker "${docker}"
    else
        local docker_icon="✅"
        [[ "$docker" == "unavailable" ]] && docker_icon="❌"
        
        format::table text "Metric" "Value" "Status" -- \
            "Memory:${memory}% used:$([ -n "${memory}" ] && [ ${memory%.*} -gt 80 ] && echo '⚠️' || echo '✅')" \
            "Disk:${disk}% used:$([ -n "${disk}" ] && [ ${disk%.*} -gt 80 ] && echo '⚠️' || echo '✅')" \
            "Load:${load}:📊" \
            "Docker:${docker}:${docker_icon}"
    fi
}

# Format resource/app data for display
format_component_data() {
    local format="$1"
    local component="$2"  # resources or apps
    local raw_data="$3"
    
    # Check for errors first
    if echo "$raw_data" | grep -q "^type:error"; then
        local error_msg
        error_msg=$(echo "$raw_data" | grep "^message:" | cut -d: -f2-)
        if [[ "$format" == "json" ]]; then
            echo "{\"error\": \"$error_msg\"}"
        else
            echo "${component^}: $error_msg"
        fi
        return
    fi
    
    local summary
    summary=$(echo "$raw_data" | grep "^summary:" | cut -d: -f2-)
    
    if [[ "$format" == "json" ]]; then
        # Build JSON based on component type
        if [[ "$component" == "resources" ]]; then
            local enabled running
            enabled=$(echo "$raw_data" | grep "^enabled:" | cut -d: -f2)
            running=$(echo "$raw_data" | grep "^running:" | cut -d: -f2)
            
            if echo "$raw_data" | grep -q "^details:true"; then
                local details_json="["
                local first=true
                while IFS= read -r line; do
                    if [[ "$line" =~ ^item:(.+):(.+)$ ]]; then
                        [[ "$first" == "true" ]] && first=false || details_json="${details_json},"
                        details_json="${details_json}{\"name\":\"${BASH_REMATCH[1]}\",\"status\":\"${BASH_REMATCH[2]}\"}"
                    fi
                done <<< "$raw_data"
                details_json="${details_json}]"
                format::json_object "enabled" "$enabled" "running" "$running" "details" "$details_json"
            else
                format::key_value json enabled "$enabled" running "$running"
            fi
        else  # apps
            local total running
            total=$(echo "$raw_data" | grep "^total:" | cut -d: -f2)
            running=$(echo "$raw_data" | grep "^running:" | cut -d: -f2)
            
            if echo "$raw_data" | grep -q "^details:true"; then
                local details_json="["
                local first=true
                while IFS= read -r line; do
                    if [[ "$line" =~ ^item:(.+):(.+)$ ]]; then
                        [[ "$first" == "true" ]] && first=false || details_json="${details_json},"
                        details_json="${details_json}{\"name\":\"${BASH_REMATCH[1]}\",\"status\":\"${BASH_REMATCH[2]}\"}"
                    fi
                done <<< "$raw_data"
                details_json="${details_json}]"
                format::json_object "total" "$total" "running" "$running" "details" "$details_json"
            else
                format::key_value json total "$total" running "$running"
            fi
        fi
    else
        # Text format
        echo "${component^}: $summary"
        
        if echo "$raw_data" | grep -q "^details:true"; then
            echo ""
            local table_rows=()
            while IFS= read -r line; do
                if [[ "$line" =~ ^item:(.+):(.+)$ ]]; then
                    local name="${BASH_REMATCH[1]}"
                    local status="${BASH_REMATCH[2]}"
                    local status_icon="🔴"
                    [[ "$status" == "running" ]] && status_icon="🟢"
                    table_rows+=("${name}:${status}:${status_icon}")
                fi
            done <<< "$raw_data"
            
            if [[ ${#table_rows[@]} -gt 0 ]]; then
                local header_name="Resource"
                [[ "$component" == "apps" ]] && header_name="App"
                format::table text "$header_name" "Status" "" -- "${table_rows[@]}"
            fi
        fi
    fi
}


# Main status display
show_status() {
    local output_format="text"
    local verbose="false"
    local show_resources="true"
    local show_apps="true"
    local show_system="true"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --json)
                output_format="json"
                shift
                ;;
            --verbose|-v)
                verbose="true"
                shift
                ;;
            --resources)
                show_apps="false"
                show_system="false"
                shift
                ;;
            --apps)
                show_resources="false"
                show_system="false"
                shift
                ;;
            --help|-h)
                show_status_help
                return 0
                ;;
            *)
                log::error "Unknown option: $1"
                show_status_help
                return 1
                ;;
        esac
    done
    
    if [[ "$output_format" == "json" ]]; then
        # JSON output using format library
        local components=()
        
        if [[ "$show_system" == "true" ]]; then
            components+=("system" "$(format_system_data json "$(get_system_data)")")
        fi
        
        if [[ "$show_resources" == "true" ]]; then
            components+=("resources" "$(format_component_data json resources "$(get_resource_data "$verbose")")")
        fi
        
        if [[ "$show_apps" == "true" ]]; then
            components+=("apps" "$(format_component_data json apps "$(get_app_data "$verbose")")")
        fi
        
        local json_output
        json_output=$(format::json_object "${components[@]}")
        echo "$json_output" | jq '.' 2>/dev/null || echo "$json_output"
    else
        # Text output
        echo ""
        echo "🚀 Vrooli System Status"
        echo "=================================================="
        
        if [[ "$show_system" == "true" ]]; then
            echo ""
            echo "📊 System Metrics"
            echo "------------------"
            format_system_data text "$(get_system_data)" || true
            echo ""
        fi
        
        if [[ "$show_resources" == "true" ]]; then
            echo "🔧 Resources"
            echo "-------------"
            format_component_data text resources "$(get_resource_data "$verbose")" || true
            echo ""
        fi
        
        if [[ "$show_apps" == "true" ]]; then
            echo "📦 Applications"
            echo "----------------"
            format_component_data text apps "$(get_app_data "$verbose")" || true
            echo ""
        fi
        
        # Quick health check
        local health_status="✅ Healthy"
        
        # Check if Docker is available
        if ! docker info >/dev/null 2>&1; then
            health_status="⚠️  Docker unavailable"
        fi
        
        # Check if API is available
        if [[ "$show_apps" == "true" ]] && ! check_api; then
            health_status="⚠️  API offline"
        fi
        
        echo "Overall: ${health_status}"
        echo ""
        
        if [[ "$verbose" == "false" ]]; then
            echo "Use 'vrooli status --verbose' for detailed information"
        fi
    fi
}

# Main entry point
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    show_status "$@"
fi