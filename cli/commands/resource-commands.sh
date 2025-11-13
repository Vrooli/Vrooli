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

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
VROOLI_ROOT="${VROOLI_ROOT:-$APP_ROOT}"
CLI_DIR="${APP_ROOT}/cli/commands"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE:-${APP_ROOT}/scripts/lib/utils/log.sh}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}" 2>/dev/null || true
# shellcheck disable=SC1091
# source "${var_LIB_DIR}/resources/auto-install.sh"  # File doesn't exist, commented out
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/format.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/cli/lib/arg-parser.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/cli/lib/output-formatter.sh"

# Resource paths - use new flat structure under /resources/
RESOURCES_DIR="${var_RESOURCES_DIR}"
RESOURCES_CONFIG="${var_ROOT_DIR}/.vrooli/service.json"
RESOURCE_REGISTRY="${var_ROOT_DIR}/.vrooli/resource-registry"
JQ_RESOURCES_EXPR="$var_JQ_RESOURCES_EXPR"
[[ -z "$JQ_RESOURCES_EXPR" ]] && JQ_RESOURCES_EXPR='(.dependencies.resources // {})'

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
    --json                  Output in JSON format (alias for --format json)
    --format <type>         Output format: text, json
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
            # Clear problematic exported functions that cause issues with set -u
            unset -f sudo_override::main 2>/dev/null || true
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
        # Clear exported functions that might cause issues with set -u
        for func in $(compgen -A function | grep "::"); do 
            unset -f "$func" 2>/dev/null || true
        done
        # Ensure VROOLI_ROOT is preserved for resource commands
        export VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$RESOURCES_DIR/.." && pwd)}"
        exec "$resource_command" "$@"
    fi
    
    # Try direct path to known CLI locations (bulletproof fallback)
    # Resources now use flat structure under /resources/
    local direct_cli_paths=(
        "$RESOURCES_DIR/$resource_name/cli.sh"
        "$RESOURCES_DIR/*/$resource_name/cli.sh"
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
        # VROOLI_ROOT is already set at the script level, no need to redefine
        
        # Simple, reliable execution with explicit environment
        # Note: VROOLI_ROOT is already set and will be inherited by the subshell
        (
            cd "$VROOLI_ROOT"
            # VROOLI_ROOT is already available from parent scope
            export OLLAMA_PORT="${OLLAMA_PORT:-11434}"
            bash "$cli_script" "$@"
        )
        local exit_code=$?
        return $exit_code
    fi
    
    # Fallback to resource CLI approach
    route_to_manage_sh "$resource_name" "$@"
}

# Route to resource CLI
route_to_manage_sh() {
    local resource_name="$1"
    shift
    
    # Find cli.sh for this resource (updated for new structure)
    local manage_script
    # First try the new resource structure
    manage_script="${APP_ROOT}/resources/${resource_name}/cli.sh"
    
    # If not found, try old locations
    if [[ ! -f "$manage_script" ]]; then
        manage_script=$(find "$RESOURCES_DIR" -path "*/${resource_name}/cli.sh" 2>/dev/null | head -1)
    fi
    
    if [[ -z "$manage_script" ]] || [[ ! -f "$manage_script" ]]; then
        log::error "Resource '$resource_name' not found or has no cli.sh"
        echo "Available resources:"
        resource_list_brief
        return 1
    fi
    
    # Execute cli.sh with direct commands (new CLI system)
    local command="${1:-status}"
    shift || true
    
    # Execute cli.sh with error handling (not using exec to capture errors)
    local exit_code=0
    case "$command" in
        inject)
            "$manage_script" inject "$@" || exit_code=$?
            ;;
        validate)
            "$manage_script" validate "$@" || exit_code=$?
            ;;
        status)
            "$manage_script" status "$@" || exit_code=$?
            ;;
        start)
            "$manage_script" start "$@" || exit_code=$?
            ;;
        stop)
            "$manage_script" stop "$@" || exit_code=$?
            ;;
        install)
            "$manage_script" install "$@" || exit_code=$?
            ;;
        *)
            "$manage_script" "$command" "$@" || exit_code=$?
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
        echo "  - Run the CLI script directly for debugging: $manage_script $command"
        return $exit_code
    fi
    
    return 0
}

# List resources (brief format for errors)
resource_list_brief() {
    # Resources now use flat structure (depth 1)
    find "$RESOURCES_DIR" -mindepth 1 -maxdepth 1 -type d | while read -r dir; do
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
    
    # First, collect all resources from the filesystem
    local filesystem_resources=()
    if [[ -d "$RESOURCES_DIR" ]]; then
        while IFS= read -r resource_dir; do
            [[ -z "$resource_dir" ]] && continue
            local resource_name=$(basename "$resource_dir")
            filesystem_resources+=("$resource_name")
        done < <(find "$RESOURCES_DIR" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | sort)
    fi
    
    # Get all resources from config (if it exists)
    local config_resources=()
    local config_data=""
    if [[ -f "$RESOURCES_CONFIG" ]]; then
        config_data=$(jq -r "${JQ_RESOURCES_EXPR} | to_entries[] | \"\(.key)/\(.value.enabled)\"" "$RESOURCES_CONFIG" 2>/dev/null || echo "")
        
        # Build array of configured resources
        while IFS= read -r resource_line; do
            [[ -z "$resource_line" ]] && continue
            local name="${resource_line%%/*}"
            config_resources+=("$name")
        done <<< "$config_data"
    fi
    
    # Create a combined list of all resources (registered and unregistered), sorted alphabetically
    local all_resource_names=()
    # Add all filesystem resources
    for name in "${filesystem_resources[@]}"; do
        all_resource_names+=("$name")
    done
    # Add any configured resources that aren't in filesystem (shouldn't happen, but be safe)
    for name in "${config_resources[@]}"; do
        local found=false
        for fs_name in "${filesystem_resources[@]}"; do
            if [[ "$fs_name" == "$name" ]]; then
                found=true
                break
            fi
        done
        if [[ "$found" == "false" ]]; then
            all_resource_names+=("$name")
        fi
    done
    
    # Sort the resource names
    IFS=$'\n' sorted_resources=($(sort <<<"${all_resource_names[*]}"))
    unset IFS
    
    # Process each resource
    for name in "${sorted_resources[@]}"; do
        [[ -z "$name" ]] && continue
        
        # Check if resource exists in filesystem
        local resource_dir="$RESOURCES_DIR/$name"
        local exists_in_filesystem="false"
        if [[ -d "$resource_dir" ]]; then
            exists_in_filesystem="true"
        fi
        
        # Check if resource is in config
        local enabled="false"
        local registered="false"
        if [[ -f "$RESOURCES_CONFIG" ]]; then
            local config_entry=$(jq -r --arg name "$name" "${JQ_RESOURCES_EXPR} | .[$name] // null" "$RESOURCES_CONFIG" 2>/dev/null)
            if [[ "$config_entry" != "null" ]]; then
                registered="true"
                enabled=$(jq -r --arg name "$name" "${JQ_RESOURCES_EXPR} | .[$name].enabled // false" "$RESOURCES_CONFIG" 2>/dev/null || echo "false")
            fi
        fi
        
        # Skip if only_running is true and resource is disabled/unregistered
        if [[ "$only_running" == "true" && "$enabled" != "true" ]]; then
            continue
        fi
        
        # Check running status using resource CLI
        local is_running="false"
        if [[ "$exists_in_filesystem" == "true" ]] && has_resource_cli "$name"; then
            local status_output
            # Use timeout to prevent hangs, capture both stdout and stderr
            if status_output=$(timeout 10s resource-"$name" status --format json --fast 2>&1); then
                # Check if we got valid JSON output
                if [[ -n "$status_output" ]] && echo "$status_output" | jq -e '.' >/dev/null 2>&1; then
                    # Parse JSON to get running status
                    local json_running
                    json_running=$(echo "$status_output" | jq -r '.running // false' 2>/dev/null || echo "false")
                    is_running="$json_running"
                fi
            fi
        fi
        
        # Skip if only_running is true and resource is not running
        if [[ "$only_running" == "true" && "$is_running" != "true" ]]; then
            continue
        fi
        
        # Check features
        local has_cli="false"
        local has_script="false"
        local has_capabilities="false"
        
        if [[ "$exists_in_filesystem" == "true" ]]; then
            if has_resource_cli "$name"; then
                has_cli="true"
            fi
            if [[ -f "$resource_dir/manage.sh" ]]; then
                has_script="true"
            fi
            if [[ -f "$resource_dir/capabilities.yaml" ]]; then
                has_capabilities="true"
            fi
        fi
        
        # Output resource data with registration status
        echo "resource:${name}:${enabled}:${is_running}:${has_cli}:${has_script}:${has_capabilities}:${registered}:${exists_in_filesystem}"
    done
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
    resource_dir=$(find "$RESOURCES_DIR" -mindepth 1 -maxdepth 1 -type d -name "$resource_name" 2>/dev/null | head -1)
    
    if [[ -z "$resource_dir" ]]; then
        echo "error:Resource not found: $resource_name"
        return
    fi
    
    # Check if enabled in config (flat structure)
    local enabled="false"
    if [[ -f "$RESOURCES_CONFIG" ]]; then
        enabled=$(jq -r --arg res "$resource_name" "${JQ_RESOURCES_EXPR} | .[$res].enabled // false" "$RESOURCES_CONFIG" 2>/dev/null || echo "false")
    fi
    
    # Check running status using resource CLI only
    local is_running="false"
    local status_message="stopped"
    
    # Use resource CLI for status check with standardized JSON output
    if has_resource_cli "$resource_name"; then
        local status_json
        # Capture output regardless of exit code, as some resources return non-zero even when providing valid JSON
        status_json=$(unset _VAR_SH_SOURCED _LOG_SH_SOURCED _JSON_SH_SOURCED _SYSTEM_COMMANDS_SH_SOURCED; "resource-${resource_name}" status --format json 2>&1 || true)
        
        # Check if we got valid JSON output
        if [[ -n "$status_json" ]] && echo "$status_json" | grep -q '^\s*{'; then
            # Parse JSON using jq for reliable status detection
            if command -v jq >/dev/null 2>&1; then
                local json_running json_healthy
                json_running=$(echo "$status_json" | jq -r '.running // false' 2>/dev/null || echo "false")
                json_healthy=$(echo "$status_json" | jq -r '.healthy // false' 2>/dev/null || echo "false")
                
                if [[ "$json_running" == "true" ]]; then
                    is_running="true"
                    if [[ "$json_healthy" == "true" ]]; then
                        status_message="running"
                    else
                        status_message="degraded"
                    fi
                else
                    status_message="stopped"
                fi
            else
                # Fallback to text parsing if jq is not available
                if echo "$status_json" | grep -q '"running":true'; then
                    is_running="true"
                    if echo "$status_json" | grep -q '"healthy":true'; then
                        status_message="running"
                    else
                        status_message="degraded"
                    fi
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
    
    # Output structured data (no category)
    echo "resource:${resource_name}:${enabled}:${is_running}:${status_message}:${has_cli}:${has_script}:${has_capabilities}:${resource_dir}"
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
    local unregistered_count=0
    
    # Process each resource - now with 8 fields including registered and exists_in_filesystem
    while IFS= read -r line; do
        if [[ "$line" =~ ^resource:(.+):(.+):(.+):(.+):(.+):(.+):(.+):(.+)$ ]]; then
            local name="${BASH_REMATCH[1]}"
            local enabled="${BASH_REMATCH[2]}"
            local is_running="${BASH_REMATCH[3]}"
            local has_cli="${BASH_REMATCH[4]}"
            local has_script="${BASH_REMATCH[5]}"
            local has_capabilities="${BASH_REMATCH[6]}"
            local registered="${BASH_REMATCH[7]}"
            local exists_in_filesystem="${BASH_REMATCH[8]}"
            
            echo "item:${name}:${enabled}:${is_running}:${has_cli}:${has_script}:${has_capabilities}:${registered}:${exists_in_filesystem}"
            
            ((total_count++))
            [[ "$enabled" == "true" ]] && ((enabled_count++))
            [[ "$is_running" == "true" ]] && ((running_count++))
            [[ "$registered" == "false" ]] && ((unregistered_count++))
        fi
    done <<< "$raw_data"
    
    echo "summary:${total_count}:${enabled_count}:${running_count}:${unregistered_count}"
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
    
    if [[ "$raw_data" =~ ^resource:(.+):(.+):(.+):(.+):(.+):(.+):(.+):(.+)$ ]]; then
        local name="${BASH_REMATCH[1]}"
        local enabled="${BASH_REMATCH[2]}"
        local is_running="${BASH_REMATCH[3]}"
        local status_message="${BASH_REMATCH[4]}"
        local has_cli="${BASH_REMATCH[5]}"
        local has_script="${BASH_REMATCH[6]}"
        local has_capabilities="${BASH_REMATCH[7]}"
        local resource_dir="${BASH_REMATCH[8]}"
        
        echo "type:status"
        echo "name:$name"
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
        format::output "$format" "kv" "error" "$error_msg"
        return
    fi
    
    # Extract summary for display
    local summary_line=$(echo "$raw_data" | grep "^summary:")
    local unregistered_count=0
    if [[ "$summary_line" =~ ^summary:([^:]+):([^:]+):([^:]+):([^:]+)$ ]]; then
        unregistered_count="${BASH_REMATCH[4]}"
    fi
    
    # Collect table rows - now with 8 fields
    local table_rows=()
    while IFS= read -r line; do
        if [[ "$line" =~ ^item:(.+):(.+):(.+):(.+):(.+):(.+):(.+):(.+)$ ]]; then
            local name="${BASH_REMATCH[1]}"
            local enabled="${BASH_REMATCH[2]}"
            local running="${BASH_REMATCH[3]}"
            local registered="${BASH_REMATCH[7]}"
            local exists_in_filesystem="${BASH_REMATCH[8]}"
            
            # Mark unregistered resources
            local status_indicator=""
            if [[ "$registered" == "false" ]]; then
                status_indicator="[UNREGISTERED]"
                enabled="N/A"
            fi
            
            # Check for config-only resources (shouldn't normally happen)
            if [[ "$exists_in_filesystem" == "false" ]]; then
                status_indicator="[MISSING]"
            fi
            
            table_rows+=("${name}:${enabled}:${running}:${status_indicator}")
        fi
    done <<< "$raw_data"
    
    # Show warning if there are unregistered resources
    if [[ "$unregistered_count" -gt 0 ]] && [[ "$format" != "json" ]]; then
        echo "⚠️  WARNING: Found $unregistered_count unregistered resource(s) in resources/ folder"
        echo ""
    fi
    
    # Format as table using format.sh with status column
    format::output "$format" "table" "Name" "Enabled" "Running" "Status" -- "${table_rows[@]}"
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
        format::output "$format" "kv" "error" "$error_msg"
        return
    fi
    
    # Extract data
    local name enabled running status path
    name=$(echo "$raw_data" | grep "^name:" | cut -d: -f2)
    enabled=$(echo "$raw_data" | grep "^enabled:" | cut -d: -f2)
    running=$(echo "$raw_data" | grep "^running:" | cut -d: -f2)
    status=$(echo "$raw_data" | grep "^status:" | cut -d: -f2)
    path=$(echo "$raw_data" | grep "^path:" | cut -d: -f2-)
    
    # Format output (JSON format now handled earlier in the chain)
    format::output "$format" "kv" \
        "name" "$name" \
        "enabled" "$enabled" \
        "running" "$running" \
        "status" "$status" \
        "path" "$path"
}


# List all available resources (completely format-agnostic)
resource_list() {
    # Parse common arguments
    local parsed_args
    if ! parsed_args=$(parse_combined_args "$@"); then
        return 1
    fi
    
    local verbose
    verbose=$(extract_arg "$parsed_args" "verbose")
    local help_requested
    help_requested=$(extract_arg "$parsed_args" "help")
    local format
    format=$(extract_arg "$parsed_args" "format")
    local remaining_args
    remaining_args=$(extract_arg "$parsed_args" "remaining")
    
    # Handle help request
    if [[ "$help_requested" == "true" ]]; then
        show_resource_help
        return 0
    fi
    
    # Parse remaining arguments for resource-specific options
    local include_connection="false"
    local only_running="false"
    
    local args_array
    mapfile -t args_array < <(args_to_array "$remaining_args")
    
    for arg in "${args_array[@]}"; do
        case "$arg" in
            --include-connection-info)
                include_connection="true"
                ;;
            --only-running)
                only_running="true"
                ;;
            *)
                cli::format_error "$format" "Unknown option: $arg"
                return 1
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
resource_status_core() {
    # Parse common arguments
    local parsed_args
    if ! parsed_args=$(parse_combined_args "$@"); then
        return 1
    fi
    
    local verbose
    verbose=$(extract_arg "$parsed_args" "verbose")
    local help_requested
    help_requested=$(extract_arg "$parsed_args" "help")
    local format
    format=$(extract_arg "$parsed_args" "format")
    local remaining_args
    remaining_args=$(extract_arg "$parsed_args" "remaining")
    
    # Handle help request
    if [[ "$help_requested" == "true" ]]; then
        show_resource_help
        return 0
    fi
    
    # Get resource name from remaining arguments
    local args_array
    mapfile -t args_array < <(args_to_array "$remaining_args")
    local resource_name="${args_array[0]:-}"
    
    # Check for unknown arguments
    for arg in "${args_array[@]:1}"; do
        if [[ "$arg" =~ ^- ]]; then
            cli::format_error "$format" "Unknown option: $arg"
            return 1
        fi
    done
    
    if [[ -z "$resource_name" ]]; then
        # Show status of all enabled resources
        local resource_data
        resource_data=$(get_resource_list_data "false" "false")
        
        # Format using resource-specific formatter
        format_resource_overview_data "$format" "$resource_data"
    else
        # Show status of specific resource
        # For JSON format with resource CLI, get JSON directly
        if [[ "$format" == "json" ]] && has_resource_cli "$resource_name"; then
            # Get JSON directly from resource CLI
            local status_json
            status_json=$(unset _VAR_SH_SOURCED _LOG_SH_SOURCED _JSON_SH_SOURCED _SYSTEM_COMMANDS_SH_SOURCED; "resource-${resource_name}" status --format json 2>&1 || true)
            
            # Check if we got valid JSON output
            if [[ -n "$status_json" ]] && echo "$status_json" | grep -q '^\s*{'; then
                echo "$status_json"
                return
            fi
        fi
        
        # Fall back to structured data for text format or resources without CLI
        local status_data
        status_data=$(get_resource_status_data "$resource_name")
        
        # Format using resource-specific formatter
        format_resource_status_data "$format" "$status_data"
    fi
}

# Backward-compatible wrapper: delegate to format-agnostic core
resource_status() {
    resource_status_core "$@"
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
    
    # Try to route to resource CLI or manage.sh with improved error handling
    local exit_code=0
    if has_resource_cli "$resource_name"; then
        # Try v2.0 manage start pattern first (most likely to work)
        local resource_command="resource-${resource_name}"
        if command -v "$resource_command" >/dev/null 2>&1; then
            # Clear source guards before exec to allow proper sourcing
            if (unset _VAR_SH_SOURCED _LOG_SH_SOURCED _JSON_SH_SOURCED _SYSTEM_COMMANDS_SH_SOURCED; 
                export VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$RESOURCES_DIR/.." && pwd)}";
                "$resource_command" manage start >/dev/null 2>&1); then
                exit_code=0
            elif (unset _VAR_SH_SOURCED _LOG_SH_SOURCED _JSON_SH_SOURCED _SYSTEM_COMMANDS_SH_SOURCED;
                  export VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$RESOURCES_DIR/.." && pwd)}";
                  "$resource_command" start >/dev/null 2>&1); then
                exit_code=0
            else
                exit_code=1
            fi
        else
            route_to_resource_cli "$resource_name" start >/dev/null 2>&1 || exit_code=$?
        fi
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
	
	local output_format="text"
	for arg in "$@"; do
		if [[ "$arg" == "--json" ]]; then output_format="json"; fi
	done
	
	if [[ "$output_format" == "text" ]]; then
		log::header "Starting All Enabled Resources"
	fi
	
	# Check if config file exists
	if [[ ! -f "$RESOURCES_CONFIG" ]]; then
		log::warning "No resource configuration found at $RESOURCES_CONFIG"
		return 0
	fi
    
    # Parse enabled resources from config
    local enabled_resources
    enabled_resources=$(jq -r "${JQ_RESOURCES_EXPR} | to_entries[] | select(.value.enabled == true) | .key" "$RESOURCES_CONFIG" 2>/dev/null)
    
    if [[ -z "$enabled_resources" ]]; then
        log::info "No resources are enabled"
        return 0
    fi
    
    local started_count=0
    local failed_count=0
    
    while IFS= read -r name; do
        
        log::info "Starting: $name"
        
        # Check if already running
        if command -v docker >/dev/null 2>&1 && docker ps --format '{{.Names}}' 2>/dev/null | grep -q "^${name}$" || false; then
            log::info "Already running: $name"
            ((started_count++))
            continue
        fi
        
        # Try v2.0 CLI patterns (try working pattern first)
        if has_resource_cli "$name"; then
            # First try full v2.0 manage start command (most likely to work)
            if (unset _VAR_SH_SOURCED _LOG_SH_SOURCED _JSON_SH_SOURCED _SYSTEM_COMMANDS_SH_SOURCED; 
                export VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$RESOURCES_DIR/.." && pwd)}";
                "resource-${name}" manage start --force >/dev/null 2>&1); then
                ((started_count++))
                log::success "✅ Started: $name"
                
                # Update registry
                if command -v resource_registry::register >/dev/null 2>&1; then
                    resource_registry::register "$name" "running"
                fi
                continue
            fi
            
            # Then try direct start command as fallback
            if (unset _VAR_SH_SOURCED _LOG_SH_SOURCED _JSON_SH_SOURCED _SYSTEM_COMMANDS_SH_SOURCED; 
                export VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$RESOURCES_DIR/.." && pwd)}";
                "resource-${name}" start >/dev/null 2>&1); then
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
		echo "Usage: vrooli resource stop-all [options]"
		echo ""
		echo "Stop all currently running resources."
		echo ""
		echo "Options:"
		echo "  --force         Force stop (SIGKILL instead of SIGTERM)"
		echo "  --dry-run       Show what would be stopped without stopping"
		echo "  --verbose       Detailed output"
		echo "  --json          Output in JSON format"
		echo ""
		echo "This will attempt to gracefully stop all resources that are running."
		return 0
	fi
	
	# Use new unified stop manager if available
	local stop_manager="${VROOLI_ROOT}/scripts/lib/lifecycle/stop-manager.sh"
	
	if [[ -f "$stop_manager" ]]; then
		log::info "Using unified stop system for resources..."
		source "$stop_manager"
		
		# Parse flags for stop manager
		local stop_args=()
		local output_format="text"
		
		for arg in "$@"; do
			case "$arg" in
				--force)
					export FORCE_STOP=true
					;;
				--verbose|-v)
					export VERBOSE=true
					;;
				--dry-run|--check)
					export DRY_RUN=true
					;;
				--json)
					output_format="json"
					# TODO: Implement JSON output in stop-manager
					log::warning "JSON output not yet implemented in stop manager"
					;;
				*)
					stop_args+=("$arg")
					;;
			esac
		done
		
		if [[ "$output_format" == "text" ]]; then
			log::header "Stopping All Resources"
		fi
		
		stop::main resources "${stop_args[@]}"
		return $?
	fi
	
	# Fallback to legacy implementation
	log::warning "Stop manager not available, using legacy method"
	
	local output_format="text"
	for arg in "$@"; do
		if [[ "$arg" == "--json" ]]; then output_format="json"; fi
	done
	
	if [[ "$output_format" == "text" ]]; then
		log::header "Stopping All Resources"
	fi
	
	# Use the auto-install module if available
	if command -v resource_auto::stop_all >/dev/null 2>&1; then
		resource_auto::stop_all
	else
		# Simplified fallback - just try to stop all Docker containers
		log::info "Stopping resources via Docker..."
		
		if command -v docker >/dev/null 2>&1; then
			local containers
			containers=$(docker ps -q)
			if [[ -n "$containers" ]]; then
				log::info "Stopping $(echo "$containers" | wc -l) containers..."
				# Note: docker stop already returns 0 even if container doesn't exist, so || true is actually safe here
				docker stop "$containers" 2>/dev/null || true
				log::success "Resources stopped"
			else
				log::info "No resources running"
			fi
		else
			log::error "Docker not available, cannot stop resources"
			return 1
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
                    resource_status_core "$@"
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
            if find "$RESOURCES_DIR" -mindepth 1 -maxdepth 1 -type d -name "$resource_name" 2>/dev/null | grep -q . || false; then
                route_to_resource_cli "$resource_name" "$@"
            else
                log::error "Unknown command or resource: $command"
                
                # Try to provide fuzzy suggestions for resource names
                if [[ -f "${VROOLI_ROOT}/scripts/lib/utils/fuzzy-match.sh" ]]; then
                    source "${VROOLI_ROOT}/scripts/lib/utils/fuzzy-match.sh"
                    
                    # Get available resource names
                    local available_resources=()
                    while IFS= read -r -d '' resource_path; do
                        local resource_name_from_path
                        resource_name_from_path=$(basename "$resource_path")
                        available_resources+=("$resource_name_from_path")
                    done < <(find "$RESOURCES_DIR" -mindepth 1 -maxdepth 1 -type d -print0 2>/dev/null)
                    
                    if [[ ${#available_resources[@]} -gt 0 ]]; then
                        local suggestions
                        mapfile -t suggestions < <(fuzzy::find_suggestions "$command" 3 0.3 "${available_resources[@]}")
                        
                        if fuzzy::format_suggestions "$command" "${suggestions[@]}"; then
                            echo
                        fi
                    fi
                fi
                
                echo "Run 'vrooli resource --help' for usage information"
                exit 1
            fi
            ;;
    esac
}

# Execute main function with all arguments
main "$@"
