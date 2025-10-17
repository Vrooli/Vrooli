#!/usr/bin/env bash
# CLI Command Framework for Vrooli Resources
# Eliminates duplicate CLI code and provides consistent command interface

# Source guard to prevent multiple sourcing
# Clear the source guard if functions are missing (handles function cleanup scenarios)
if ! declare -F cli::init >/dev/null 2>&1; then
    unset _CLI_COMMAND_FRAMEWORK_SOURCED 2>/dev/null || true
fi
[[ -n "${_CLI_COMMAND_FRAMEWORK_SOURCED:-}" ]] && return 0
_CLI_COMMAND_FRAMEWORK_SOURCED=1

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPTS_RESOURCES_LIB_DIR="${APP_ROOT}/scripts/resources/lib"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/log.sh"

# Framework state
declare -gA CLI_COMMANDS=()
declare -gA CLI_COMMAND_DESCRIPTIONS=()  
declare -gA CLI_COMMAND_HANDLERS=()
declare -gA CLI_COMMAND_FLAGS=()
declare -g CLI_RESOURCE_NAME=""
declare -g CLI_RESOURCE_DESCRIPTION=""
declare -g CLI_DRY_RUN="${DRY_RUN:-false}"

#######################################
# Initialize CLI framework for a resource
# Args: $1 - resource_name, $2 - description (optional)
#######################################
cli::init() {
    local resource_name="$1"
    local description="${2:-$resource_name resource management}"
    
    CLI_RESOURCE_NAME="$resource_name"
    CLI_RESOURCE_DESCRIPTION="$description"
    
    # Register core commands that every resource should have
    cli::register_command "help" "Show this help message" "cli::_handle_help"
    cli::register_command "status" "Show resource status" "cli::_handle_status"
    cli::register_command "start" "Start the resource" "cli::_handle_start" "modifies-system"
    cli::register_command "stop" "Stop the resource" "cli::_handle_stop" "modifies-system"
    cli::register_command "restart" "Restart the resource" "cli::_handle_restart" "modifies-system"
    cli::register_command "install" "Install the resource" "cli::_handle_install" "modifies-system"
    cli::register_command "validate" "Validate resource configuration" "cli::_handle_validate"
    cli::register_command "content" "Manage resource content (add/list/get/remove/execute)" "cli::_handle_content" "modifies-system"
    
    log::debug "CLI framework initialized for: $resource_name"
}
# Don't export - sourced directly in CLI scripts

#######################################
# Register a command with the framework
# Args: $1 - command_name, $2 - description, $3 - handler_function, $4 - flags (optional)
#######################################
cli::register_command() {
    local cmd="$1"
    local desc="$2"
    local handler="$3"
    local flags="${4:-}"
    
    # Use explicit key format to avoid bash associative array issues with set -u
    CLI_COMMANDS[$cmd]=1
    CLI_COMMAND_DESCRIPTIONS[$cmd]="$desc"
    CLI_COMMAND_HANDLERS[$cmd]="$handler"
    CLI_COMMAND_FLAGS[$cmd]="$flags"
    
    log::debug "Registered command: $cmd -> $handler"
}
# Don't export - sourced directly in CLI scripts

#######################################
# Main command dispatcher - call this from resource CLI
# Args: $@ - command line arguments
#######################################
cli::dispatch() {
    local cmd
    
    # Handle help flags first, before processing other arguments
    if [[ $# -gt 0 && ("${1:-}" == "--help" || "${1:-}" == "-h") ]]; then
        cmd="help"
    elif [[ $# -gt 0 ]]; then
        cmd="$1"
        shift
        
        # Check for multi-word commands (e.g., "test all", "content add")
        if [[ $# -gt 0 ]] && [[ "${1:-}" != "--"* ]]; then
            local potential_multi_cmd="$cmd $1"
            if [[ -n "${CLI_COMMANDS[$potential_multi_cmd]:-}" ]]; then
                cmd="$potential_multi_cmd"
                shift
            fi
        fi
    else
        cmd="help"
    fi
    
    # Handle global flags
    while [[ $# -gt 0 ]]; do
        case "${1:-}" in
            --dry-run)
                export CLI_DRY_RUN="true"
                export DRY_RUN="true"
                shift
                ;;
            --help|-h)
                cmd="help"
                break
                ;;
            *)
                # Arguments are already correct, just break to continue
                break
                ;;
        esac
    done
    
    # Check if command exists with suggestions
    if [[ -z "${CLI_COMMANDS[$cmd]:-}" ]]; then
        log::error "Unknown command: $cmd"
        
        # Try to provide suggestions
        if [[ -f "${VROOLI_ROOT:-.}/scripts/lib/utils/fuzzy-match.sh" ]]; then
            source "${VROOLI_ROOT:-.}/scripts/lib/utils/fuzzy-match.sh"
            
            local available_commands=("${!CLI_COMMANDS[@]}")
            local suggestions
            mapfile -t suggestions < <(fuzzy::find_suggestions "$cmd" 3 0.3 "${available_commands[@]}")
            
            if fuzzy::format_suggestions "$cmd" "${suggestions[@]}"; then
                echo
            fi
        fi
        
        cli::_handle_help
        return 1
    fi
    
    local handler="${CLI_COMMAND_HANDLERS[$cmd]}"
    local flags="${CLI_COMMAND_FLAGS[$cmd]:-}"
    
    # Check dry-run for system-modifying commands
    if [[ "$flags" == *"modifies-system"* ]] && [[ "$CLI_DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would execute: $cmd"
        if command -v "${handler}_dry_run" &>/dev/null; then
            "${handler}_dry_run" "$@"
        else
            log::info "[DRY RUN] Command: $cmd with args: $*"
        fi
        return 0
    fi
    
    # Execute the command handler
    if command -v "$handler" &>/dev/null; then
        log::debug "Executing handler: $handler"
        "$handler" "$@"
    else
        log::error "Handler function not found: $handler"
        return 1
    fi
}
# Don't export - sourced directly in CLI scripts

#######################################
# Generate and display help output
#######################################
cli::_handle_help() {
    echo "Usage: resource-${CLI_RESOURCE_NAME} <command> [options]"
    echo
    echo "$CLI_RESOURCE_DESCRIPTION"
    echo
    echo "Commands:"
    
    # Separate single-word and multi-word commands
    local single_commands=()
    local multi_commands=()
    
    for cmd in "${!CLI_COMMANDS[@]}"; do
        if [[ "$cmd" == *" "* ]]; then
            multi_commands+=("$cmd")
        else
            single_commands+=("$cmd")
        fi
    done
    
    # Sort commands
    IFS=$'\n' single_commands=($(sort <<<"${single_commands[*]}"))
    IFS=$'\n' multi_commands=($(sort <<<"${multi_commands[*]}"))
    
    # Display single-word commands first
    for cmd in "${single_commands[@]}"; do
        local desc="${CLI_COMMAND_DESCRIPTIONS[$cmd]:-No description available}"
        local flags="${CLI_COMMAND_FLAGS[$cmd]:-}"
        
        printf "  %-20s %s" "$cmd" "$desc"
        if [[ "$flags" == *"modifies-system"* ]]; then
            printf " [modifies system]"
        fi
        echo
    done
    
    # Display multi-word commands grouped by prefix
    if [[ ${#multi_commands[@]} -gt 0 ]]; then
        local last_prefix=""
        for cmd in "${multi_commands[@]}"; do
            local prefix="${cmd%% *}"
            local subcommand="${cmd#* }"
            local desc="${CLI_COMMAND_DESCRIPTIONS[$cmd]:-No description available}"
            local flags="${CLI_COMMAND_FLAGS[$cmd]:-}"
            
            if [[ "$prefix" != "$last_prefix" ]] && [[ -n "$last_prefix" ]]; then
                echo  # Add spacing between groups
            fi
            
            printf "  %-20s %s" "$cmd" "$desc"
            if [[ "$flags" == *"modifies-system"* ]]; then
                printf " [modifies system]"
            fi
            echo
            
            last_prefix="$prefix"
        done
    fi
    
    echo
    echo "Global Options:"
    echo "  --dry-run    Show what would be done without making changes"
    echo "  --help, -h   Show this help message"
    echo
    echo "Examples:"
    echo "  resource-${CLI_RESOURCE_NAME} status"
    echo "  resource-${CLI_RESOURCE_NAME} start --dry-run"
    echo "  resource-${CLI_RESOURCE_NAME} install"
    if command -v "${CLI_RESOURCE_NAME}::content" &>/dev/null || command -v "${CLI_RESOURCE_NAME}::content::add" &>/dev/null; then
        echo "  resource-${CLI_RESOURCE_NAME} content add --file data.json"
        echo "  resource-${CLI_RESOURCE_NAME} content list"
    fi
}

#######################################
# Core command handlers - these delegate to resource-specific functions
#######################################

cli::_handle_status() {
    if command -v "resource_cli::status" &>/dev/null; then
        resource_cli::status "$@"
    elif command -v "${CLI_RESOURCE_NAME}::status" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::status" "$@"
    else
        # Fallback to manage.sh if available
        cli::_fallback_to_manage "status" "$@"
    fi
}

cli::_handle_start() {
    if command -v "resource_cli::start" &>/dev/null; then
        resource_cli::start "$@"
    elif command -v "${CLI_RESOURCE_NAME}::start" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::start" "$@"
    else
        # Fallback to manage.sh if available
        cli::_fallback_to_manage "start" "$@"
    fi
}

cli::_handle_stop() {
    if command -v "resource_cli::stop" &>/dev/null; then
        resource_cli::stop "$@"
    elif command -v "${CLI_RESOURCE_NAME}::stop" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::stop" "$@"
    else
        # Fallback to manage.sh if available
        cli::_fallback_to_manage "stop" "$@"
    fi
}

cli::_handle_restart() {
    log::info "Restarting ${CLI_RESOURCE_NAME}..."
    cli::_handle_stop "$@" && sleep 2 && cli::_handle_start "$@"
}

cli::_handle_install() {
    if command -v "resource_cli::install" &>/dev/null; then
        resource_cli::install "$@"
    elif command -v "${CLI_RESOURCE_NAME}::install" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::install" "$@"
    else
        # Fallback to manage.sh if available
        cli::_fallback_to_manage "install" "$@"
    fi
}

cli::_handle_validate() {
    if command -v "resource_cli::validate" &>/dev/null; then
        resource_cli::validate "$@"
    elif command -v "${CLI_RESOURCE_NAME}::validate" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::validate" "$@"
    else
        log::error "No validate handler found for ${CLI_RESOURCE_NAME}"
        return 1
    fi
}

#######################################
# Handle content management subcommands
# Args: $1 - subcommand (add/list/get/remove/execute), $@ - remaining args
#######################################
cli::_handle_content() {
    local subcommand="${1:-list}"
    shift || true
    
    # Map common aliases to standard subcommands
    case "$subcommand" in
        create|new)
            subcommand="add"
            ;;
        ls|show)
            subcommand="list"
            ;;
        rm|delete|del)
            subcommand="remove"
            ;;
        run|exec)
            subcommand="execute"
            ;;
        help|--help|-h)
            echo "Usage: resource-${CLI_RESOURCE_NAME} content <subcommand> [options]"
            echo ""
            echo "Subcommands:"
            echo "  add       Add content to the resource"
            echo "  list      List stored content"
            echo "  get       Get specific content"
            echo "  remove    Remove content"
            echo "  execute   Execute/run content (if applicable)"
            echo ""
            echo "Examples:"
            echo "  resource-${CLI_RESOURCE_NAME} content add --file data.json"
            echo "  resource-${CLI_RESOURCE_NAME} content list --type all"
            echo "  resource-${CLI_RESOURCE_NAME} content get --name my-config"
            echo "  resource-${CLI_RESOURCE_NAME} content remove --id config-123"
            return 0
            ;;
    esac
    
    # Check if resource has content management implementation
    local handler="${CLI_RESOURCE_NAME}::content::${subcommand}"
    
    # Try resource-specific handler first
    if command -v "$handler" &>/dev/null; then
        log::debug "Executing content handler: $handler"
        "$handler" "$@"
    elif command -v "${CLI_RESOURCE_NAME}::content" &>/dev/null; then
        # Fallback to single content handler with subcommand as first arg
        log::debug "Executing unified content handler: ${CLI_RESOURCE_NAME}::content"
        "${CLI_RESOURCE_NAME}::content" "$subcommand" "$@"
    else
        log::info "Content management not implemented for ${CLI_RESOURCE_NAME}"
        echo "This resource does not support content management."
        echo "Content management is typically used for resources that store data/configurations."
        return 0
    fi
}

#######################################
# Fallback to cli.sh for standard operations
# Args: $1 - action (start, stop, status, install), $@ - additional args
#######################################
cli::_fallback_to_manage() {
    local action="$1"
    shift || true
    
    # CRITICAL: Prevent infinite recursion - check if we're already in a CLI context
    if [[ "${CLI_RECURSION_GUARD:-}" == "active" ]]; then
        log::error "Cannot find ${action} implementation for ${CLI_RESOURCE_NAME}"
        log::error "Please implement ${CLI_RESOURCE_NAME}::${action} function"
        return 1
    fi
    
    # Set recursion guard
    export CLI_RECURSION_GUARD="active"
    
    # Find cli.sh for this resource
    local manage_script=""
    
    # Check if we're running from a symlink (installed CLI)
    if [[ -L "${BASH_SOURCE[0]}" ]]; then
        local real_path="$(readlink -f "${BASH_SOURCE[0]}")"
        local resource_dir="$(dirname "$real_path")"
        if [[ -f "$resource_dir/cli.sh" ]]; then
            manage_script="$resource_dir/cli.sh"
        fi
    fi
    
    # Try common locations if not found
    if [[ -z "$manage_script" ]]; then
        local possible_locations=(
            "${APP_ROOT}/resources/${CLI_RESOURCE_NAME}/cli.sh"
            "${var_SCRIPTS_RESOURCES_DIR}/${CLI_RESOURCE_NAME}/cli.sh"
            "${var_SCRIPTS_RESOURCES_DIR}/*/${CLI_RESOURCE_NAME}/cli.sh"
        )
        
        for pattern in "${possible_locations[@]}"; do
            for file in $pattern; do
                if [[ -f "$file" ]]; then
                    manage_script="$file"
                    break 2
                fi
            done
        done
    fi
    
    if [[ -z "$manage_script" ]]; then
        log::error "No cli.sh found for ${CLI_RESOURCE_NAME}"
        unset CLI_RECURSION_GUARD
        return 1
    fi
    
    # Execute cli.sh with the action
    log::debug "Delegating to cli.sh: $manage_script $action"
    local result=0
    "$manage_script" "$action" --yes yes "$@" || result=$?
    
    # Clear recursion guard
    unset CLI_RECURSION_GUARD
    return $result
}

#######################################
# Utility functions for resource CLIs
#######################################

# Check if resource is running (generic implementation)
cli::is_running() {
    local port="${1:-}"
    if [[ -z "$port" ]]; then
        log::error "Port required for running check"
        return 1
    fi
    
    if command -v "resources::is_service_running" &>/dev/null; then
        resources::is_service_running "$port"
    else
        # Fallback check
        if command -v "lsof" &>/dev/null; then
            lsof -i ":$port" >/dev/null 2>&1
        elif command -v "netstat" &>/dev/null; then
            netstat -tln | grep -q ":$port "
        else
            return 1
        fi
    fi
}
# Don't export - sourced directly in CLI scripts

# Get resource configuration value
cli::get_config() {
    local key="$1"
    local default="${2:-}"
    
    # Try environment variable first
    local env_var="${CLI_RESOURCE_NAME^^}_${key^^}"
    if [[ -n "${!env_var:-}" ]]; then
        echo "${!env_var}"
        return 0
    fi
    
    # Fallback to default
    echo "$default"
}
# Don't export - sourced directly in CLI scripts

#######################################
# Advanced command registration helpers
#######################################

# Register a command with custom validation
cli::register_command_with_validation() {
    local cmd="$1"
    local desc="$2" 
    local handler="$3"
    local validator="$4"
    local flags="${5:-}"
    
    cli::register_command "$cmd" "$desc" "cli::_validate_and_execute" "$flags"
    
    # Store the real handler and validator
    CLI_COMMAND_HANDLERS["${cmd}_real"]="$handler"
    CLI_COMMAND_HANDLERS["${cmd}_validator"]="$validator"
}

cli::_validate_and_execute() {
    local cmd="${CLI_RESOURCE_NAME}_current_command"
    local real_handler="${CLI_COMMAND_HANDLERS[${cmd}_real]}"
    local validator="${CLI_COMMAND_HANDLERS[${cmd}_validator]}"
    
    # Run validator first
    if command -v "$validator" &>/dev/null; then
        if ! "$validator" "$@"; then
            log::error "Validation failed for command"
            return 1
        fi
    fi
    
    # Run real handler
    "$real_handler" "$@"
}

# Batch register multiple simple commands
cli::register_simple_commands() {
    local -n cmd_map=$1  # Pass associative array by reference
    
    for cmd in "${!cmd_map[@]}"; do
        local desc="${cmd_map[${cmd}]}"
        cli::register_command "$cmd" "$desc" "cli::_simple_delegator_${cmd}"
    done
}

# Create a simple delegator function dynamically
cli::_create_simple_delegator() {
    local cmd="$1"
    eval "cli::_simple_delegator_${cmd}() { 
        if command -v \"resource_cli::${cmd}\" &>/dev/null; then
            resource_cli::${cmd} \"\$@\"
        elif command -v \"${CLI_RESOURCE_NAME}::${cmd}\" &>/dev/null; then
            \"${CLI_RESOURCE_NAME}::${cmd}\" \"\$@\"
        else
            log::error \"No ${cmd} handler found for ${CLI_RESOURCE_NAME}\"
            return 1
        fi
    }"
}

#######################################
# Export functions for use by resource CLIs
#######################################
# Note: Exporting functions can cause issues with bash associative arrays
# when set -u is enabled in sourced scripts. Only export essential functions
# Don't export - sourced directly in CLI scripts
# Don't export - sourced directly in CLI scripts
# Don't export - sourced directly in CLI scripts
# Don't export - sourced directly in CLI scripts
# Don't export - sourced directly in CLI scripts
