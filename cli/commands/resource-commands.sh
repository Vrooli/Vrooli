#!/usr/bin/env bash
################################################################################
# Vrooli CLI - Enhanced Resource Management Commands
# 
# Supports both legacy manage.sh and new resource-<name> CLI commands.
# Handles resource operations including listing, status checking, and command routing.
#
# Usage:
#   vrooli resource <name> <command> [options]  # New style
#   vrooli resource <subcommand> [options]      # Legacy style
#
################################################################################

set -euo pipefail

# Get CLI directory
CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${CLI_DIR}/../../scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE:-${CLI_DIR}/../../scripts/lib/utils/log.sh}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_DIR}/resources/auto-install.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${CLI_DIR}/../../scripts/lib/utils/format.sh"

# Resource paths
RESOURCES_DIR="${var_SCRIPTS_RESOURCES_DIR}"
RESOURCES_CONFIG="${var_ROOT_DIR}/.vrooli/service.json"
RESOURCES_CONFIG_LEGACY="${var_ROOT_DIR}/.vrooli/resources.local.json"
RESOURCE_REGISTRY="${var_ROOT_DIR}/.vrooli/resource-registry"

# Show help for resource commands
show_resource_help() {
    cat << EOF
🚀 Vrooli Resource Commands

USAGE:
    vrooli resource <name> <command> [options]    # Direct resource commands
    vrooli resource <subcommand> [options]        # Management commands

MANAGEMENT COMMANDS:
    list                    List all available resources
    status [name]           Show status of resources (or specific resource)
    install <name>          Install a specific resource (initial setup)
    start <name>            Start a specific resource
    start-all               Start all enabled resources
    stop <name>             Stop a specific resource
    stop-all                Stop all running resources
    enable <name>           Enable resource in configuration
    disable <name>          Disable resource in configuration
    info <name>             Show detailed information about a resource

RESOURCE COMMANDS:
    When a resource has its own CLI installed, you can use:
    vrooli resource <name> <command> [options]
    
    Example:
    vrooli resource n8n inject workflow.json
    vrooli resource ollama status
    vrooli resource postgres backup

OPTIONS:
    --help, -h              Show this help message
    --verbose, -v           Show detailed output

EXAMPLES:
    vrooli resource list                  # List all resources
    vrooli resource status                 # Check status of all resources
    vrooli resource status postgres        # Check PostgreSQL status
    vrooli resource install ollama         # Install Ollama
    vrooli resource start postgres         # Start PostgreSQL
    vrooli resource start-all               # Start all enabled resources
    vrooli resource stop n8n               # Stop n8n resource
    vrooli resource stop-all               # Stop all running resources
    vrooli resource n8n inject file.json   # Use n8n CLI directly
    vrooli resource n8n list-workflows     # n8n-specific command

For more information: https://docs.vrooli.com/cli/resources
EOF
}

# Check if a resource has a registered CLI
has_resource_cli() {
    local resource_name="$1"
    
    # Check registry
    if [[ -f "${RESOURCE_REGISTRY}/${resource_name}.json" ]]; then
        return 0
    fi
    
    # Check if command exists
    if command -v "resource-${resource_name}" >/dev/null 2>&1; then
        return 0
    fi
    
    return 1
}

# Route to resource-specific CLI
route_to_resource_cli() {
    local resource_name="$1"
    shift
    
    # Try registered command first
    if [[ -f "${RESOURCE_REGISTRY}/${resource_name}.json" ]]; then
        local cli_command
        cli_command=$(jq -r '.command' "${RESOURCE_REGISTRY}/${resource_name}.json" 2>/dev/null)
        if [[ -n "$cli_command" ]] && command -v "$cli_command" >/dev/null 2>&1; then
            # Clear source guards before exec to allow proper sourcing
            unset _VAR_SH_SOURCED _LOG_SH_SOURCED _JSON_SH_SOURCED _SYSTEM_COMMANDS_SH_SOURCED
            # Ensure VROOLI_ROOT is preserved for registered commands
            export VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$RESOURCES_DIR/.." && pwd)}"
            exec "$cli_command" "$@"
        fi
    fi
    
    # Try standard resource-<name> pattern
    local resource_command="resource-${resource_name}"
    if command -v "$resource_command" >/dev/null 2>&1; then
        # Clear source guards before exec to allow proper sourcing
        unset _VAR_SH_SOURCED _LOG_SH_SOURCED _JSON_SH_SOURCED _SYSTEM_COMMANDS_SH_SOURCED
        # Ensure VROOLI_ROOT is preserved for resource commands
        export VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$RESOURCES_DIR/.." && pwd)}"
        exec "$resource_command" "$@"
    fi
    
    # Try direct path to known CLI locations (bulletproof fallback)
    local direct_cli_paths=(
        "$RESOURCES_DIR/ai/$resource_name/cli.sh"
        "$RESOURCES_DIR/storage/$resource_name/cli.sh" 
        "$RESOURCES_DIR/automation/$resource_name/cli.sh"
        "$RESOURCES_DIR/*/$resource_name/cli.sh"
        "$RESOURCES_DIR/$resource_name/cli.sh"
    )
    
    local cli_script=""
    for path_pattern in "${direct_cli_paths[@]}"; do
        if [[ "$path_pattern" == *"*"* ]]; then
            # Use find for wildcard patterns
            cli_script=$(find "$RESOURCES_DIR" -name "cli.sh" -path "*/${resource_name}/*" 2>/dev/null | head -1)
        else
            # Direct path check
            cli_script="$path_pattern"
        fi
        
        if [[ -n "$cli_script" ]] && [[ -f "$cli_script" ]]; then
            break
        fi
        cli_script=""  # Reset for next iteration
    done
    
    if [[ -n "$cli_script" ]] && [[ -f "$cli_script" ]]; then
        # Clear source guards before calling CLI to allow proper sourcing  
        unset _VAR_SH_SOURCED _LOG_SH_SOURCED _JSON_SH_SOURCED _SYSTEM_COMMANDS_SH_SOURCED
        
        # Set up proper environment and working directory
        local vrooli_root="${VROOLI_ROOT:-$(cd "$RESOURCES_DIR/.." && pwd)}"
        
        # Simple, reliable execution with explicit environment
        (
            cd "$vrooli_root"
            export VROOLI_ROOT="$vrooli_root"
            export OLLAMA_PORT="${OLLAMA_PORT:-11434}"
            bash "$cli_script" "$@"
        )
        local exit_code=$?
        return $exit_code
    fi
    
    # Fallback to legacy manage.sh approach
    route_to_manage_sh "$resource_name" "$@"
}

# Route to legacy manage.sh
route_to_manage_sh() {
    local resource_name="$1"
    shift
    
    # Find manage.sh for this resource
    local manage_script
    manage_script=$(find "$RESOURCES_DIR" -path "*/${resource_name}/manage.sh" 2>/dev/null | head -1)
    
    if [[ -z "$manage_script" ]] || [[ ! -f "$manage_script" ]]; then
        log::error "Resource '$resource_name' not found or has no manage.sh"
        echo "Available resources:"
        resource_list_brief
        return 1
    fi
    
    # Convert new-style commands to legacy manage.sh arguments
    local command="${1:-status}"
    shift || true
    
    # Execute manage.sh with error handling (not using exec to capture errors)
    local exit_code=0
    case "$command" in
        inject)
            "$manage_script" --action inject "$@" || exit_code=$?
            ;;
        validate)
            "$manage_script" --action validate-injection "$@" || exit_code=$?
            ;;
        status)
            "$manage_script" --action status "$@" || exit_code=$?
            ;;
        start)
            "$manage_script" --action start "$@" || exit_code=$?
            ;;
        stop)
            "$manage_script" --action stop "$@" || exit_code=$?
            ;;
        install)
            "$manage_script" --action install "$@" || exit_code=$?
            ;;
        *)
            "$manage_script" --action "$command" "$@" || exit_code=$?
            ;;
    esac
    
    # Handle errors with helpful messages
    if [[ $exit_code -ne 0 ]]; then
        log::error "Failed to execute $command for resource '$resource_name' (exit code: $exit_code)"
        echo ""
        echo "Troubleshooting tips:"
        echo "  - Check if the resource is properly installed: vrooli resource status $resource_name"
        echo "  - Try reinstalling the resource: vrooli resource install $resource_name"
        echo "  - Check logs for more details"
        echo "  - Run the manage script directly for debugging: $manage_script --action $command"
        return $exit_code
    fi
    
    return 0
}

# List resources (brief format for errors)
resource_list_brief() {
    find "$RESOURCES_DIR" -mindepth 2 -maxdepth 2 -type d | while read -r dir; do
        basename "$dir"
    done | sort | tr '\n' ' '
    echo ""
}

################################################################################
# Format-Agnostic Data Collection Functions
################################################################################

# Collect raw resource data from configuration and filesystem
collect_resource_list_data() {
    local include_connection="${1:-false}"
    local only_running="${2:-false}"
    
    # Check which config file exists
    local config_file=""
    if [[ -f "$RESOURCES_CONFIG" ]]; then
        config_file="$RESOURCES_CONFIG"
    elif [[ -f "$RESOURCES_CONFIG_LEGACY" ]]; then
        config_file="$RESOURCES_CONFIG_LEGACY"
    else
        echo "error:No resource configuration found"
        return
    fi
    
    # Batch check Docker container status for performance
    local docker_status_cache
    docker_status_cache=$(batch_check_docker_status)
    
    # Get all resources from config
    local all_resources
    all_resources=$(jq -r '
        .resources | to_entries[] as $category | 
        $category.value | to_entries[] | 
        "\($category.key)/\(.key)/\(.value.enabled)"
    ' "$config_file" 2>/dev/null)
    
    if [[ -z "$all_resources" ]]; then
        echo "error:No resources found in configuration"
        return
    fi
    
    # Process each resource  
    while IFS= read -r resource_line; do
        [[ -z "$resource_line" ]] && continue
        
        local category="${resource_line%%/*}"
        local rest="${resource_line#*/}"
        local name="${rest%%/*}"
        local enabled="${rest##*/}"
        
        # Skip disabled resources if only_running is true
        if [[ "$only_running" == "true" && "$enabled" != "true" ]]; then
            continue
        fi
        
        local resource_dir="$RESOURCES_DIR/$category/$name"
        
        if [[ -d "$resource_dir" ]]; then
            # Check running status using Docker cache
            local container_patterns=("$name" "vrooli-$name" "vrooli-${name}-main" "${name}-container")
            local is_running="false"
            
            for pattern in "${container_patterns[@]}"; do
                local check_result
                check_result=$(echo "$docker_status_cache" | jq -r --arg name "$pattern" '.[$name] // "not_running"' 2>/dev/null || echo "not_running")
                if [[ "$check_result" == "running" ]]; then
                    is_running="true"
                    break
                fi
            done
            
            # Skip if only_running is true and resource is not running
            if [[ "$only_running" == "true" && "$is_running" != "true" ]]; then
                continue
            fi
            
            # Check features
            local has_cli="false"
            local has_script="false"
            local has_capabilities="false"
            
            if has_resource_cli "$name"; then
                has_cli="true"
            fi
            if [[ -f "$resource_dir/manage.sh" ]]; then
                has_script="true"
            fi
            if [[ -f "$resource_dir/capabilities.yaml" ]]; then
                has_capabilities="true"
            fi
            
            # Output resource data
            echo "resource:${name}:${category}:${enabled}:${is_running}:${has_cli}:${has_script}:${has_capabilities}"
        fi
    done <<< "$all_resources"
}

# Collect raw status data for a single resource
collect_resource_status_data() {
    local resource_name="$1"
    
    if [[ -z "$resource_name" ]]; then
        echo "error:Resource name required"
        return
    fi
    
    # Find resource in filesystem
    local resource_dir
    resource_dir=$(find "$RESOURCES_DIR" -mindepth 2 -maxdepth 2 -type d -name "$resource_name" 2>/dev/null | head -1)
    
    if [[ -z "$resource_dir" ]]; then
        echo "error:Resource not found: $resource_name"
        return
    fi
    
    local category
    category=$(basename "$(dirname "$resource_dir")")
    
    # Check if enabled in config
    local enabled="false"
    if [[ -f "$RESOURCES_CONFIG" ]]; then
        enabled=$(jq -r --arg cat "$category" --arg res "$resource_name" '.resources[$cat][$res].enabled // false' "$RESOURCES_CONFIG" 2>/dev/null || echo "false")
    fi
    
    # Check running status
    local is_running="false"
    local status_message="stopped"
    
    # Quick Docker check
    if docker ps --format "{{.Names}}" 2>/dev/null | grep -q "^${resource_name}$"; then
        is_running="true"
        status_message="running"
    else
        # Try resource CLI for detailed status
        if has_resource_cli "$resource_name"; then
            local status_output
            if status_output=$(unset _VAR_SH_SOURCED _LOG_SH_SOURCED _JSON_SH_SOURCED _SYSTEM_COMMANDS_SH_SOURCED; "resource-${resource_name}" status 2>/dev/null | head -5); then
                if echo "$status_output" | grep -Eq "✅.*(Running|Health|Healthy)|Status:.*running|Container.*running" && \
                   ! echo "$status_output" | grep -q "not running\|is not running\|stopped"; then
                    is_running="true"
                    status_message="running"
                else
                    status_message="stopped"
                fi
            fi
        fi
    fi
    
    # Check features
    local has_cli="false"
    local has_script="false"
    local has_capabilities="false"
    
    if has_resource_cli "$resource_name"; then
        has_cli="true"
    fi
    if [[ -f "$resource_dir/manage.sh" ]]; then
        has_script="true"
    fi
    if [[ -f "$resource_dir/capabilities.yaml" ]]; then
        has_capabilities="true"
    fi
    
    # Output structured data
    echo "resource:${resource_name}:${category}:${enabled}:${is_running}:${status_message}:${has_cli}:${has_script}:${has_capabilities}:${resource_dir}"
}

################################################################################
# Structured Data Functions (format-agnostic)
################################################################################

# Get structured resource list data
get_resource_list_data() {
    local include_connection="${1:-false}"
    local only_running="${2:-false}"
    
    local raw_data
    raw_data=$(collect_resource_list_data "$include_connection" "$only_running")
    
    # Check for errors
    if echo "$raw_data" | grep -q "^error:"; then
        local error_msg
        error_msg=$(echo "$raw_data" | grep "^error:" | cut -d: -f2-)
        echo "type:error"
        echo "message:$error_msg"
        return
    fi
    
    echo "type:list"
    
    local total_count=0
    local enabled_count=0
    local running_count=0
    
    # Process each resource
    while IFS= read -r line; do
        if [[ "$line" =~ ^resource:(.+):(.+):(.+):(.+):(.+):(.+):(.+)$ ]]; then
            local name="${BASH_REMATCH[1]}"
            local category="${BASH_REMATCH[2]}"
            local enabled="${BASH_REMATCH[3]}"
            local is_running="${BASH_REMATCH[4]}"
            local has_cli="${BASH_REMATCH[5]}"
            local has_script="${BASH_REMATCH[6]}"
            local has_capabilities="${BASH_REMATCH[7]}"
            
            echo "item:${name}:${category}:${enabled}:${is_running}:${has_cli}:${has_script}:${has_capabilities}"
            
            ((total_count++))
            [[ "$enabled" == "true" ]] && ((enabled_count++))
            [[ "$is_running" == "true" ]] && ((running_count++))
        fi
    done <<< "$raw_data"
    
    echo "summary:${total_count}:${enabled_count}:${running_count}"
}

# Get structured status data for a single resource
get_resource_status_data() {
    local resource_name="$1"
    
    local raw_data
    raw_data=$(collect_resource_status_data "$resource_name")
    
    # Check for errors
    if echo "$raw_data" | grep -q "^error:"; then
        local error_msg
        error_msg=$(echo "$raw_data" | grep "^error:" | cut -d: -f2-)
        echo "type:error"
        echo "message:$error_msg"
        return
    fi
    
    if [[ "$raw_data" =~ ^resource:(.+):(.+):(.+):(.+):(.+):(.+):(.+):(.+):(.+)$ ]]; then
        local name="${BASH_REMATCH[1]}"
        local category="${BASH_REMATCH[2]}"
        local enabled="${BASH_REMATCH[3]}"
        local is_running="${BASH_REMATCH[4]}"
        local status_message="${BASH_REMATCH[5]}"
        local has_cli="${BASH_REMATCH[6]}"
        local has_script="${BASH_REMATCH[7]}"
        local has_capabilities="${BASH_REMATCH[8]}"
        local resource_dir="${BASH_REMATCH[9]}"
        
        echo "type:status"
        echo "name:$name"
        echo "category:$category"
        echo "enabled:$enabled"
        echo "running:$is_running"
        echo "status:$status_message"
        echo "has_cli:$has_cli"
        echo "has_script:$has_script"
        echo "has_capabilities:$has_capabilities"
        echo "path:$resource_dir"
    else
        echo "type:error"
        echo "message:Invalid resource data format"
    fi
}

################################################################################
# Resource-Specific Formatters (using generic format.sh functions)
################################################################################

# Format resource list data for display
format_resource_list_data() {
    local format="$1"
    local raw_data="$2"
    
    # Check for errors
    if echo "$raw_data" | grep -q "^type:error"; then
        local error_msg
        error_msg=$(echo "$raw_data" | grep "^message:" | cut -d: -f2-)
        format::key_value "$format" "error" "$error_msg"
        return
    fi
    
    # Collect table rows
    local table_rows=()
    while IFS= read -r line; do
        if [[ "$line" =~ ^item:(.+):(.+):(.+):(.+):(.+):(.+):(.+)$ ]]; then
            local name="${BASH_REMATCH[1]}"
            local category="${BASH_REMATCH[2]}"
            local enabled="${BASH_REMATCH[3]}"
            local running="${BASH_REMATCH[4]}"
            table_rows+=("${name}:${category}:${enabled}:${running}")
        fi
    done <<< "$raw_data"
    
    # Format as table
    format::table "$format" "Name" "Category" "Enabled" "Running" -- "${table_rows[@]}"
}

# Format resource overview data (for resource status with no args)
format_resource_overview_data() {
    local format="$1"
    local raw_data="$2"
    
    format_resource_list_data "$format" "$raw_data"
}

# Format single resource status data for display  
format_resource_status_data() {
    local format="$1"
    local raw_data="$2"
    
    # Check for errors
    if echo "$raw_data" | grep -q "^type:error"; then
        local error_msg
        error_msg=$(echo "$raw_data" | grep "^message:" | cut -d: -f2-)
        format::key_value "$format" "error" "$error_msg"
        return
    fi
    
    # Extract data
    local name category enabled running status path
    name=$(echo "$raw_data" | grep "^name:" | cut -d: -f2)
    category=$(echo "$raw_data" | grep "^category:" | cut -d: -f2)
    enabled=$(echo "$raw_data" | grep "^enabled:" | cut -d: -f2)
    running=$(echo "$raw_data" | grep "^running:" | cut -d: -f2)
    status=$(echo "$raw_data" | grep "^status:" | cut -d: -f2)
    path=$(echo "$raw_data" | grep "^path:" | cut -d: -f2-)
    
    format::key_value "$format" \
        "name" "$name" \
        "category" "$category" \
        "enabled" "$enabled" \
        "running" "$running" \
        "status" "$status" \
        "path" "$path"
}


# List all available resources (completely format-agnostic)
resource_list() {
    local format="text"
    local include_connection="false"
    local only_running="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --format)
                format="$2"
                shift 2
                ;;
            --json)
                format="json"
                shift
                ;;
            --include-connection-info)
                include_connection="true"
                shift
                ;;
            --only-running)
                only_running="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Get structured data (format-agnostic)
    local resource_data
    resource_data=$(get_resource_list_data "$include_connection" "$only_running")
    
    # Format and display using resource-specific formatter
    format_resource_list_data "$format" "$resource_data"
}


# Show resource status (completely format-agnostic)
resource_status() {
    local resource_name="${1:-}"
    local format="text"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --format)
                format="$2"
                shift 2
                ;;
            --json)
                format="json"
                shift
                ;;
            *)
                resource_name="$1"
                shift
                ;;
        esac
    done
    
    if [[ -z "$resource_name" ]]; then
        # Show status of all enabled resources
        local resource_data
        resource_data=$(get_resource_list_data "false" "false")
        
        # Format using resource-specific formatter
        format_resource_overview_data "$format" "$resource_data"
    else
        # Show status of specific resource
        local status_data
        status_data=$(get_resource_status_data "$resource_name")
        
        # Format using resource-specific formatter
        format_resource_status_data "$format" "$status_data"
    fi
}

# Batch check Docker container status for all resources
batch_check_docker_status() {
    # Get all running containers in one call
    local running_containers=""
    if command -v docker >/dev/null 2>&1; then
        running_containers=$(docker ps --format '{{.Names}}' 2>/dev/null || true)
    fi
    
    # Create a lookup for running containers
    declare -A container_status
    while IFS= read -r container_name; do
        [[ -n "$container_name" ]] && container_status["$container_name"]="running"
    done <<< "$running_containers"
    
    # Output as JSON for easy parsing using jq for safety
    if [[ ${#container_status[@]} -gt 0 ]]; then
        # Build JSON object with jq to ensure proper escaping
        local json_pairs=()
        for container in "${!container_status[@]}"; do
            json_pairs+=("\"$container\":\"${container_status[$container]}\"")
        done
        echo "{$(IFS=,; echo "${json_pairs[*]}")}"
    else
        echo "{}"
    fi
}


# Show resource status
resource_status() {
    local resource_name="${1:-}"
    
    if [[ -z "$resource_name" ]]; then
        # Show status of all enabled resources
        log::header "Resource Status Overview"
        
        # Check which config file exists
        local config_file=""
        if [[ -f "$RESOURCES_CONFIG" ]]; then
            config_file="$RESOURCES_CONFIG"
        elif [[ -f "$RESOURCES_CONFIG_LEGACY" ]]; then
            config_file="$RESOURCES_CONFIG_LEGACY"
        else
            log::warning "No resource configuration found"
            return 0
        fi
        
        # Parse enabled resources from config
        local enabled_resources
        enabled_resources=$(jq -r '
            .resources | to_entries[] as $category | 
            $category.value | to_entries[] | 
            select(.value.enabled == true) | 
            "\($category.key)/\(.key)"
        ' "$config_file" 2>/dev/null)
        
        if [[ -z "$enabled_resources" ]]; then
            log::info "No resources are enabled"
            return 0
        fi
        
        while IFS= read -r resource_path; do
            local category="${resource_path%%/*}"
            local name="${resource_path#*/}"
            
            echo ""
            echo "📦 $name ($category)"
            echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
            
            # Try to use resource CLI if available
            if has_resource_cli "$name"; then
                if command -v "resource-${name}" >/dev/null 2>&1; then
                    # Clear source guards to allow resource CLI to source its dependencies
                    (unset _VAR_SH_SOURCED _LOG_SH_SOURCED _JSON_SH_SOURCED _SYSTEM_COMMANDS_SH_SOURCED; "resource-${name}" status 2>/dev/null) || echo "Status check failed"
                fi
            else
                # Fallback to basic status
                if command -v docker >/dev/null 2>&1 && docker ps --format '{{.Names}}' 2>/dev/null | grep -q "^${name}$" || false; then
                    echo "Status: 🟢 Running (Docker)"
                else
                    echo "Status: ⭕ Not running"
                fi
            fi
        done <<< "$enabled_resources"
    else
        # Show status of specific resource
        if has_resource_cli "$resource_name"; then
            route_to_resource_cli "$resource_name" status
        else
            # Basic status check
            echo "📦 Resource: $resource_name"
            echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
            
            if command -v docker >/dev/null 2>&1 && docker ps --format '{{.Names}}' 2>/dev/null | grep -q "^${resource_name}$" || false; then
                echo "Status: 🟢 Running (Docker)"
                docker ps --filter "name=^${resource_name}$" --format "table {{.Status}}\t{{.Ports}}" | tail -n 1
            else
                echo "Status: ⭕ Not running"
                
                # Check if container exists but stopped
                if command -v docker >/dev/null 2>&1 && docker ps -a --format '{{.Names}}' 2>/dev/null | grep -q "^${resource_name}$" || false; then
                    echo "Container exists but is stopped"
                fi
            fi
        fi
    fi
}

# Install a resource
resource_install() {
    local resource_name="${1:-}"
    
    if [[ -z "$resource_name" ]]; then
        log::error "Resource name required"
        echo "Usage: vrooli resource install <name>"
        return 1
    fi
    
    # Try resource CLI first
    if has_resource_cli "$resource_name"; then
        route_to_resource_cli "$resource_name" install
    else
        route_to_manage_sh "$resource_name" install
    fi
}

# Start a specific resource
resource_start() {
    local resource_name="${1:-}"
    
    # Handle help flag
    if [[ "$resource_name" == "--help" ]] || [[ "$resource_name" == "-h" ]]; then
        echo "Usage: vrooli resource start <name>"
        echo ""
        echo "Start a specific resource that has been installed."
        echo ""
        echo "Examples:"
        echo "  vrooli resource start postgres     # Start PostgreSQL"
        echo "  vrooli resource start n8n          # Start n8n"
        return 0
    fi
    
    if [[ -z "$resource_name" ]]; then
        log::error "Resource name required"
        echo "Usage: vrooli resource start <name>"
        return 1
    fi
    
    log::info "Starting resource: $resource_name"
    
    # Try to route to resource CLI or manage.sh with error handling
    local exit_code=0
    if has_resource_cli "$resource_name"; then
        route_to_resource_cli "$resource_name" start || exit_code=$?
    else
        route_to_manage_sh "$resource_name" start || exit_code=$?
    fi
    
    # Only update registry if start was successful
    if [[ $exit_code -eq 0 ]]; then
        if command -v resource_registry::register >/dev/null 2>&1; then
            resource_registry::register "$resource_name" "running"
        fi
        log::success "Successfully started resource: $resource_name"
    else
        log::error "Failed to start resource: $resource_name"
        return $exit_code
    fi
    
    return 0
}

# Start all enabled resources
resource_start_all() {
    # Handle help flag
    if [[ "${1:-}" == "--help" ]] || [[ "${1:-}" == "-h" ]]; then
        echo "Usage: vrooli resource start-all"
        echo ""
        echo "Start all resources that are marked as enabled in the configuration."
        echo ""
        echo "This will start resources from service.json that have 'enabled: true'."
        return 0
    fi
    
    log::header "Starting All Enabled Resources"
    
    # Check which config file exists
    local config_file=""
    if [[ -f "$RESOURCES_CONFIG" ]]; then
        config_file="$RESOURCES_CONFIG"
    elif [[ -f "$RESOURCES_CONFIG_LEGACY" ]]; then
        config_file="$RESOURCES_CONFIG_LEGACY"
    else
        log::warning "No resource configuration found"
        return 0
    fi
    
    # Parse enabled resources from config
    local enabled_resources
    enabled_resources=$(jq -r '
        .resources | to_entries[] as $category | 
        $category.value | to_entries[] | 
        select(.value.enabled == true) | 
        "\($category.key)/\(.key)"
    ' "$config_file" 2>/dev/null)
    
    if [[ -z "$enabled_resources" ]]; then
        log::info "No resources are enabled"
        return 0
    fi
    
    local started_count=0
    local failed_count=0
    
    while IFS= read -r resource_path; do
        local category="${resource_path%%/*}"
        local name="${resource_path#*/}"
        
        log::info "Starting: $name"
        
        # Check if already running
        if command -v docker >/dev/null 2>&1 && docker ps --format '{{.Names}}' 2>/dev/null | grep -q "^${name}$" || false; then
            log::info "Already running: $name"
            ((started_count++))
            continue
        fi
        
        # Try CLI first
        if has_resource_cli "$name"; then
            if (unset _VAR_SH_SOURCED _LOG_SH_SOURCED _JSON_SH_SOURCED _SYSTEM_COMMANDS_SH_SOURCED; "resource-${name}" start 2>/dev/null); then
                ((started_count++))
                log::success "✅ Started: $name"
                
                # Update registry
                if command -v resource_registry::register >/dev/null 2>&1; then
                    resource_registry::register "$name" "running"
                fi
                continue
            fi
        fi
        
        # Try manage.sh
        local resource_dir="${RESOURCES_DIR}/${category}/${name}"
        if [[ -f "$resource_dir/manage.sh" ]]; then
            if "$resource_dir/manage.sh" --action start --yes yes 2>/dev/null; then
                ((started_count++))
                log::success "✅ Started: $name"
                
                # Update registry
                if command -v resource_registry::register >/dev/null 2>&1; then
                    resource_registry::register "$name" "running"
                fi
                continue
            fi
        fi
        
        log::warning "Could not start: $name"
        ((failed_count++))
    done <<< "$enabled_resources"
    
    if [[ $started_count -gt 0 ]]; then
        log::success "✅ Started $started_count resource(s)"
    fi
    
    if [[ $failed_count -gt 0 ]]; then
        log::warning "Failed to start $failed_count resource(s)"
    fi
}

# Stop a specific resource
resource_stop() {
    local resource_name="${1:-}"
    
    # Handle help flag
    if [[ "$resource_name" == "--help" ]] || [[ "$resource_name" == "-h" ]]; then
        echo "Usage: vrooli resource stop <name>"
        echo ""
        echo "Stop a running resource."
        echo ""
        echo "Examples:"
        echo "  vrooli resource stop postgres     # Stop PostgreSQL"
        echo "  vrooli resource stop n8n          # Stop n8n"
        return 0
    fi
    
    if [[ -z "$resource_name" ]]; then
        log::error "Resource name required"
        echo "Usage: vrooli resource stop <name>"
        return 1
    fi
    
    log::info "Stopping resource: $resource_name"
    
    # Try to route to resource CLI or manage.sh
    if has_resource_cli "$resource_name"; then
        route_to_resource_cli "$resource_name" stop
    else
        route_to_manage_sh "$resource_name" stop
    fi
    
    # Update registry if available
    if command -v resource_registry::register >/dev/null 2>&1; then
        resource_registry::register "$resource_name" "stopped"
    fi
}

# Stop all running resources
resource_stop_all() {
    # Handle help flag
    if [[ "${1:-}" == "--help" ]] || [[ "${1:-}" == "-h" ]]; then
        echo "Usage: vrooli resource stop-all"
        echo ""
        echo "Stop all currently running resources."
        echo ""
        echo "This will attempt to gracefully stop all resources that are running."
        return 0
    fi
    
    log::header "Stopping All Resources"
    
    # Use the auto-install module if available
    if command -v resource_auto::stop_all >/dev/null 2>&1; then
        resource_auto::stop_all
    else
        # Fallback to Docker-based approach
        log::info "Stopping Docker containers..."
        
        local stopped_count=0
        
        # Find all resource directories
        find "$RESOURCES_DIR" -mindepth 2 -maxdepth 2 -type d | while read -r resource_dir; do
            local resource_name
            resource_name=$(basename "$resource_dir")
            
            # Check if running in Docker
            if command -v docker >/dev/null 2>&1 && docker ps --format '{{.Names}}' 2>/dev/null | grep -q "^${resource_name}$" || false; then
                log::info "Stopping: $resource_name"
                
                # Try CLI first
                if has_resource_cli "$resource_name"; then
                    if (unset _VAR_SH_SOURCED _LOG_SH_SOURCED _JSON_SH_SOURCED _SYSTEM_COMMANDS_SH_SOURCED; "resource-${resource_name}" stop 2>/dev/null); then
                        ((stopped_count++))
                        log::success "✅ Stopped: $resource_name"
                        continue
                    fi
                fi
                
                # Try manage.sh
                if [[ -f "$resource_dir/manage.sh" ]]; then
                    if "$resource_dir/manage.sh" --action stop --yes yes 2>/dev/null; then
                        ((stopped_count++))
                        log::success "✅ Stopped: $resource_name"
                        continue
                    fi
                fi
                
                # Fallback to Docker stop
                if docker stop "$resource_name" 2>/dev/null; then
                    ((stopped_count++))
                    log::success "✅ Stopped: $resource_name (Docker)"
                else
                    log::warning "Could not stop: $resource_name"
                fi
            fi
        done
        
        if [[ $stopped_count -gt 0 ]]; then
            log::success "✅ Stopped $stopped_count resource(s)"
        else
            log::info "No running resources found"
        fi
    fi
}

# Main command router
main() {
    local command="${1:-help}"
    
    # Check if first argument might be a resource name
    # (if it doesn't match known subcommands)
    case "$command" in
        list|status|install|start|start-all|stop|stop-all|enable|disable|info|help|--help|-h)
            # These are management commands
            shift || true
            case "$command" in
                list)
                    resource_list "$@"
                    ;;
                status)
                    resource_status "$@"
                    ;;
                install)
                    resource_install "$@"
                    ;;
                start)
                    resource_start "$@"
                    ;;
                start-all)
                    resource_start_all "$@"
                    ;;
                stop)
                    resource_stop "$@"
                    ;;
                stop-all)
                    resource_stop_all "$@"
                    ;;
                enable|disable)
                    log::error "Not implemented yet: $command"
                    ;;
                info)
                    log::error "Not implemented yet: $command"
                    ;;
                help|--help|-h)
                    show_resource_help
                    ;;
            esac
            ;;
        *)
            # Assume it's a resource name and route to its CLI
            local resource_name="$command"
            shift || true
            
            # Check if this looks like a valid resource
            if find "$RESOURCES_DIR" -mindepth 2 -maxdepth 2 -type d -name "$resource_name" 2>/dev/null | grep -q . || false; then
                route_to_resource_cli "$resource_name" "$@"
            else
                log::error "Unknown command or resource: $command"
                echo ""
                echo "Run 'vrooli resource --help' for usage information"
                exit 1
            fi
            ;;
    esac
}

# Execute main function with all arguments
main "$@"