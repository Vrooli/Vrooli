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

# List all available resources (detailed)
resource_list() {
    local format="${1:-pretty}"           # pretty or json
    local include_connection="${2:-false}" # include connection info for credentials
    local only_running="${3:-false}"      # only show running resources
    
    # Handle format parameter passed as --format json
    while [[ $# -gt 0 ]]; do
        case $1 in
            --format)
                format="$2"
                shift 2
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
    
    if [[ "$format" == "json" ]]; then
        resource_list_json "$include_connection" "$only_running"
        return $?
    fi
    
    # Check if terminal supports colors (use ANSI codes directly for better compatibility)
    local use_colors=false
    if [[ -t 1 ]]; then
        # Check if terminal supports colors using multiple methods
        if [[ -n "${COLORTERM:-}" ]] || [[ "${TERM:-}" == *"color"* ]] || [[ "${TERM:-}" == "xterm"* ]]; then
            use_colors=true
        elif command -v tput >/dev/null 2>&1; then
            local colors=$(tput colors 2>/dev/null || echo 0)
            [[ $colors -ge 8 ]] && use_colors=true
        fi
    fi
    
    # Use ANSI escape codes for better compatibility
    local CYAN=''
    local GREEN=''
    local YELLOW=''
    local BLUE=''
    local GRAY=''
    local BOLD=''
    local DIM=''
    local NC=''
    
    if [[ "$use_colors" == "true" ]]; then
        # ANSI color codes
        CYAN=$'\033[36m'
        GREEN=$'\033[32m'
        YELLOW=$'\033[33m'
        BLUE=$'\033[34m'
        GRAY=$'\033[90m'
        BOLD=$'\033[1m'
        DIM=$'\033[2m'
        NC=$'\033[0m'
    fi
    
    echo ""
    echo "${CYAN}╔════════════════════════════════════════════════════════════════════╗${NC}"
    echo "${CYAN}║${NC}                    ${BOLD}📦 VROOLI RESOURCE MANAGER${NC}                     ${CYAN}║${NC}"
    echo "${CYAN}╚════════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    
    # Configuration info in a box
    echo "${DIM}┌─ Configuration ─────────────────────────────────────────────────────┐${NC}"
    if [[ -f "$RESOURCES_CONFIG" ]]; then
        echo "${DIM}│${NC} Config: ${BLUE}${RESOURCES_CONFIG}${NC}"
    else
        echo "${DIM}│${NC} Config: ${GRAY}No configuration found${NC}"
    fi
    echo "${DIM}│${NC} Registry: ${BLUE}${RESOURCE_REGISTRY}${NC}"
    echo "${DIM}└──────────────────────────────────────────────────────────────────────┘${NC}"
    echo ""
    
    # Get list of resource directories by category
    local categories=(
        "storage:💾 Databases & Storage"
        "ai:🤖 AI Models & Inference"
        "automation:⚙️  Workflow Automation"
        "agents:🌐 AI Agents & Browsers"
        "search:🔍 Search Engines"
        "monitoring:📊 Monitoring & Analytics"
        "communication:💬 Chat & Communication"
    )
    
    # Track if any resources found
    local resources_found=false
    
    for category_info in "${categories[@]}"; do
        local category="${category_info%%:*}"
        local description="${category_info#*:}"
        local category_dir="${RESOURCES_DIR}/${category}"
        
        if [[ ! -d "$category_dir" ]]; then
            continue
        fi
        
        # Check if category has any resources
        local has_resources=false
        for resource_dir in "$category_dir"/*; do
            if [[ -d "$resource_dir" ]]; then
                has_resources=true
                break
            fi
        done
        
        if [[ "$has_resources" == "false" ]]; then
            continue
        fi
        
        resources_found=true
        
        echo "${BOLD}${description}${NC}"
        echo "${DIM}├────────────────────────────────────────────────────────────────────${NC}"
        
        # Header row
        echo "${DIM}│${NC} Resource             Status     Running         Features"
        echo "${DIM}├────────────────────────────────────────────────────────────────────${NC}"
        
        for resource_dir in "$category_dir"/*; do
            [[ ! -d "$resource_dir" ]] && continue
            
            local resource_name
            resource_name=$(basename "$resource_dir")
            local running_status=""
            local enabled_status=""
            local features=""
            
            # Check if resource is enabled in config
            if [[ -f "$RESOURCES_CONFIG" ]]; then
                local is_enabled
                # Use more robust JQ query with proper error handling
                is_enabled=$(jq -r --arg cat "$category" --arg res "$resource_name" '.resources[$cat][$res].enabled // false' "$RESOURCES_CONFIG" 2>/dev/null || echo "false")
                if [[ "$is_enabled" == "true" ]]; then
                    enabled_status="✓ Enabled"
                    enabled_color="${GREEN}"
                else
                    enabled_status="○ Disabled"
                    enabled_color="${GRAY}"
                fi
            else
                enabled_status="○ Unknown"
                enabled_color="${GRAY}"
            fi
            
            # Check features
            local feature_list=""
            if has_resource_cli "$resource_name"; then
                feature_list="${feature_list}[CLI] "
            fi
            if [[ -f "$resource_dir/manage.sh" ]]; then
                feature_list="${feature_list}[Script] "
            fi
            if [[ -f "$resource_dir/capabilities.yaml" ]]; then
                feature_list="${feature_list}[Cap] "
            fi
            
            # Check if running (simplified version to debug early exit)
            local running_status="○ Available"
            local running_color="${GRAY}"
            
            # Very simple status check - just show if manage.sh exists
            if [[ -f "$resource_dir/manage.sh" ]]; then
                running_status="○ Available"
                running_color="${GRAY}"
            else
                running_status="○ Unknown"
                running_color="${GRAY}"
            fi
            
            # Format resource name with color based on status
            local name_color=""
            if [[ "$is_enabled" == "true" ]]; then
                name_color="${BOLD}"
            else
                name_color="${DIM}"
            fi
            
            # Print the row with proper formatting
            printf "${DIM}│${NC} ${name_color}%-20s${NC} ${enabled_color}%-10s${NC} ${running_color}%-15s${NC} ${GRAY}%-20s${NC}\n" \
                "$resource_name" "$enabled_status" "$running_status" "$feature_list"
        done
        
        echo "${DIM}└────────────────────────────────────────────────────────────────────${NC}"
        echo ""
    done
    
    if [[ "$resources_found" == "false" ]]; then
        echo "${YELLOW}No resources found in ${RESOURCES_DIR}${NC}"
        echo ""
    fi
    
    # Show legend
    echo "${DIM}┌─ Legend ────────────────────────────────────────────────────────────┐${NC}"
    echo "${DIM}│${NC} Status:  ${GREEN}✓ Enabled${NC}  ${GRAY}○ Disabled${NC}"
    echo "${DIM}│${NC} Running: ${GREEN}● Running${NC}  ${YELLOW}○ Stopped${NC}  ${GRAY}○ Not installed${NC}"
    echo "${DIM}│${NC} Features: ${BLUE}[CLI]${NC} Has CLI  ${GRAY}[Script]${NC} Has manage.sh  ${GRAY}[Cap]${NC} Has capabilities"
    echo "${DIM}└──────────────────────────────────────────────────────────────────────┘${NC}"
    echo ""
    
    # Show quick commands
    echo "${DIM}Quick Commands:${NC}"
    echo "  ${CYAN}vrooli resource start <name>${NC}     Start a resource"
    echo "  ${CYAN}vrooli resource stop <name>${NC}      Stop a resource"
    echo "  ${CYAN}vrooli resource status <name>${NC}    Check resource status"
    echo "  ${CYAN}vrooli resource install <name>${NC}   Install a resource"
    echo ""
}

#######################################
# Get resource status in JSON format
# Args: $1 - resource name, $2 - resource directory, $3 - category, $4 - include_connection, $5 - docker_status_cache
# Returns: JSON object with status and connection info
#######################################
get_resource_status_json() {
    local resource_name="$1"
    local resource_dir="$2"
    local category="$3"
    local include_connection="${4:-false}"
    local docker_status_cache="${5:-{}}"
    
    # Initialize status info
    local is_running="false"
    local status_message="unknown"
    local connection_info="{}"
    
    # Quick check using Docker cache first (much faster)
    # Try multiple container name patterns
    local docker_running="not_running"
    local container_patterns=("$resource_name" "vrooli-$resource_name" "vrooli-${resource_name}-main" "${resource_name}-container")
    
    for pattern in "${container_patterns[@]}"; do
        local check_result
        check_result=$(echo "$docker_status_cache" | jq -r --arg name "$pattern" '.[$name] // "not_running"' 2>/dev/null || echo "not_running")
        if [[ "$check_result" == "running" ]]; then
            docker_running="running"
            break
        fi
    done
    
    if [[ "$docker_running" == "running" ]]; then
        # Resource is running according to Docker - use that
        is_running="true"
        status_message="running"
    else
        # Resource not in Docker or not running - try detailed status check
        local status_output=""
        local status_exit_code=1
        
        if has_resource_cli "$resource_name"; then
            # Try resource CLI - check if it supports --format json
            if command -v "resource-${resource_name}" >/dev/null 2>&1; then
                # Try with --format json first (future-proof)
                if status_output=$(unset _VAR_SH_SOURCED _LOG_SH_SOURCED _JSON_SH_SOURCED _SYSTEM_COMMANDS_SH_SOURCED; "resource-${resource_name}" status --format json 2>/dev/null) && \
                   echo "$status_output" | jq empty 2>/dev/null; then
                    # Resource supports JSON format and returned valid JSON
                    echo "$status_output"
                    return 0
                else
                    # Fall back to text parsing (but limit to fast check only)
                    status_output=$(unset _VAR_SH_SOURCED _LOG_SH_SOURCED _JSON_SH_SOURCED _SYSTEM_COMMANDS_SH_SOURCED; "resource-${resource_name}" status 2>/dev/null | head -5) || status_output=""
                    status_exit_code=$?
                fi
            fi
        else
            # Try manage.sh (with timeout to prevent hanging)
            local manage_script="$resource_dir/manage.sh"
            if [[ -f "$manage_script" ]]; then
                # Try with --format json first (future-proof)
                if status_output=$(timeout 3 "$manage_script" --action status --format json 2>/dev/null) && \
                   echo "$status_output" | jq empty 2>/dev/null; then
                    # Resource supports JSON format and returned valid JSON
                    echo "$status_output"
                    return 0
                else
                    # Fall back to quick text parsing (with timeout)
                    status_output=$(timeout 3 "$manage_script" --action status 2>/dev/null | head -5) || status_output=""
                    status_exit_code=$?
                fi
            fi
        fi
        
        # Parse text output for running status
        if [[ -n "$status_output" ]] && [[ $status_exit_code -eq 0 ]]; then
            if echo "$status_output" | grep -Eq "✅.*(Running|Health|Healthy)|Status:.*running|Container.*running" && \
               ! echo "$status_output" | grep -q "not running\|is not running\|stopped"; then
                is_running="true"
                status_message="running"
            else
                is_running="false"
                status_message="stopped"
            fi
        else
            # Default to not running
            is_running="false"
            status_message="not_running"
        fi
    fi
    
    # Get connection info if requested and resource is running
    if [[ "$include_connection" == "true" ]] && [[ "$is_running" == "true" ]]; then
        connection_info=$(extract_connection_info "$resource_name" "$resource_dir" "$category" 2>/dev/null || echo "{}")
    fi
    
    # Output JSON
    jq -n \
        --arg name "$resource_name" \
        --arg category "$category" \
        --argjson running "$is_running" \
        --arg status "$status_message" \
        --argjson connection "$connection_info" \
        '{
            name: $name,
            category: $category,
            running: $running,
            status: $status,
            connection: $connection
        }'
}

#######################################
# Extract connection info from resource (simplified version of auto-credentials logic)
#######################################
extract_connection_info() {
    local resource_name="$1"
    local resource_dir="$2"
    local category="$3"
    
    local defaults_file="$resource_dir/config/defaults.sh"
    [[ ! -f "$defaults_file" ]] && echo "{}" && return
    
    # Extract configuration in subshell to avoid variable pollution
    local config_data
    config_data=$(
        # Redirect all output to avoid pollution
        exec 3>&1 4>&2 >/dev/null 2>&1
        
        # shellcheck disable=SC1090
        source "$defaults_file" || exit 1
        
        # Call export function if it exists (suppress all output)
        if declare -f "${resource_name}::export_config" >/dev/null 2>&1; then
            "${resource_name}::export_config" >/dev/null 2>&1 || true
        fi
        
        # Restore output for jq
        exec 1>&3 2>&4
        
        # Extract common configuration patterns
        local port host user password database
        local resource_upper=$(echo "$resource_name" | tr '[:lower:]' '[:upper:]' | tr '-' '_')
        
        # Use printenv instead of eval for security
        port=$(printenv "${resource_upper}_PORT" || printenv "${resource_upper}_DEFAULT_PORT" || echo "")
        host=$(printenv "${resource_upper}_HOST" || echo "localhost")
        user=$(printenv "${resource_upper}_USER" || printenv "${resource_upper}_DEFAULT_USER" || printenv "${resource_upper}_ROOT_USER" || echo "")
        password=$(printenv "${resource_upper}_PASSWORD" || printenv "${resource_upper}_ROOT_PASSWORD" || echo "")
        database=$(printenv "${resource_upper}_DATABASE" || printenv "${resource_upper}_DEFAULT_DB" || echo "")
        
        # Skip if no port detected (likely not a network service)
        [[ -z "$port" ]] && echo "{}" && exit 0
        
        # Detect Docker networking adjustments
        local docker_host="$host"
        local container_name="${resource_name}"
        if command -v docker >/dev/null 2>&1 && docker ps --format "{{.Names}}" | grep -q "^n8n$" 2>/dev/null; then
            case "$host" in
                localhost|127.0.0.1)
                    # Check if target resource is also in Docker
                    if docker ps --format "{{.Names}}" | grep -q "^${container_name}$" 2>/dev/null; then
                        docker_host="$container_name"  # Container-to-container
                    else
                        # For Linux, try host.docker.internal, fall back to gateway
                        if [[ "$(uname)" == "Linux" ]]; then
                            # Try to get Docker gateway IP
                            local gateway_ip
                            gateway_ip=$(docker network inspect bridge --format='{{range .IPAM.Config}}{{.Gateway}}{{end}}' 2>/dev/null || echo "")
                            if [[ -n "$gateway_ip" ]]; then
                                docker_host="$gateway_ip"
                            else
                                docker_host="172.17.0.1"  # Common Docker bridge gateway
                            fi
                        else
                            docker_host="host.docker.internal"  # Works on Mac/Windows
                        fi
                    fi
                    ;;
            esac
        fi
        
        # Output as JSON (simplified to avoid shell escaping issues)
        jq -n \
            --arg host "$docker_host" \
            --arg original_host "$host" \
            --arg port "${port:-}" \
            --arg user "${user:-}" \
            --arg password "${password:-}" \
            --arg database "${database:-}" \
            --arg container "$container_name" \
            '{
                host: $host,
                original_host: $original_host,
                port: ($port | if . == "" then null else (tonumber? // .) end),
                user: ($user | if . == "" then null else . end),
                password: ($password | if . == "" then null else . end),
                database: ($database | if . == "" then null else . end),
                container_name: $container
            }'
    ) 2>/dev/null
    
    # Return the config data or empty JSON if extraction failed
    if [[ -n "$config_data" ]]; then
        echo "$config_data"
    else
        echo "{}"
    fi
}

#######################################
# Batch check Docker container status for all resources
# Returns: associative array of container_name -> status
#######################################
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

#######################################
# List resources in JSON format
#######################################
resource_list_json() {
    local include_connection="${1:-false}"
    local only_running="${2:-false}"
    
    # Check which config file exists
    local config_file=""
    if [[ -f "$RESOURCES_CONFIG" ]]; then
        config_file="$RESOURCES_CONFIG"
    elif [[ -f "$RESOURCES_CONFIG_LEGACY" ]]; then
        config_file="$RESOURCES_CONFIG_LEGACY"
    else
        echo "[]"
        return 0
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
        echo "[]"
        return 0
    fi
    
    local resource_json_array=()
    
    # Process each resource  
    while IFS= read -r resource_line; do
        [[ -z "$resource_line" ]] && continue
        
        local category="${resource_line%%/*}"
        local rest="${resource_line#*/}"
        local name="${rest%%/*}"
        local enabled="${rest##*/}"
        
        # Skip disabled resources unless we want all
        if [[ "$only_running" == "false" ]] || [[ "$enabled" == "true" ]]; then
            local resource_dir="$RESOURCES_DIR/$category/$name"
            
            if [[ -d "$resource_dir" ]]; then
                # Pre-filter for --only-running using Docker cache (much faster)
                if [[ "$only_running" == "true" ]]; then
                    local container_patterns=("$name" "vrooli-$name" "vrooli-${name}-main" "${name}-container")
                    local quick_running_check="false"
                    
                    for pattern in "${container_patterns[@]}"; do
                        local check_result
                        check_result=$(echo "$docker_status_cache" | jq -r --arg name "$pattern" '.[$name] // "not_running"' 2>/dev/null || echo "not_running")
                        if [[ "$check_result" == "running" ]]; then
                            quick_running_check="true"
                            break
                        fi
                    done
                    
                    # Skip expensive status check if not running in Docker
                    if [[ "$quick_running_check" == "false" ]]; then
                        continue
                    fi
                fi
                
                # Get status for this resource (with Docker cache for performance)
                local resource_status_json
                resource_status_json=$(get_resource_status_json "$name" "$resource_dir" "$category" "$include_connection" "$docker_status_cache")
                
                # Add enabled field
                if [[ -n "$resource_status_json" && "$resource_status_json" != "{}" ]]; then
                    local enabled_value
                    [[ "$enabled" == "true" ]] && enabled_value="true" || enabled_value="false"
                    resource_status_json=$(echo "$resource_status_json" | jq --argjson enabled "$enabled_value" '. + {enabled: $enabled}')
                else
                    # Create minimal JSON for resources without status
                    local enabled_value
                    [[ "$enabled" == "true" ]] && enabled_value="true" || enabled_value="false"
                    resource_status_json=$(jq -n \
                        --arg name "$name" \
                        --arg category "$category" \
                        --argjson running false \
                        --arg status "unknown" \
                        --argjson enabled "$enabled_value" \
                        '{
                            name: $name,
                            category: $category,
                            running: $running,
                            status: $status,
                            enabled: $enabled,
                            connection: {}
                        }')
                fi
                
                # Add to results (already filtered by running status if needed)
                resource_json_array+=("$resource_status_json")
            fi
        fi
    done <<< "$all_resources"
    
    # Output JSON array
    if [[ ${#resource_json_array[@]} -gt 0 ]]; then
        printf '%s\n' "${resource_json_array[@]}" | jq -s '.'
    else
        echo "[]"
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