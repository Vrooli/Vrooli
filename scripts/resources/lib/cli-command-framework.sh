#!/usr/bin/env bash
# CLI Command Framework for Vrooli Resources
# Eliminates duplicate CLI code and provides consistent command interface

# Source guard to prevent multiple sourcing
[[ -n "${_CLI_COMMAND_FRAMEWORK_SOURCED:-}" ]] && return 0
export _CLI_COMMAND_FRAMEWORK_SOURCED=1

# Source required utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../lib/utils/log.sh" 2>/dev/null || true

# Framework state
declare -A CLI_COMMANDS=()
declare -A CLI_COMMAND_DESCRIPTIONS=()  
declare -A CLI_COMMAND_HANDLERS=()
declare -A CLI_COMMAND_FLAGS=()
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
    
    log::debug "CLI framework initialized for: $resource_name"
}

#######################################
# Register a command with the framework
# Args: $1 - command_name, $2 - description, $3 - handler_function, $4 - flags (optional)
#######################################
cli::register_command() {
    local cmd="$1"
    local desc="$2"
    local handler="$3"
    local flags="${4:-}"
    
    CLI_COMMANDS["$cmd"]=1
    CLI_COMMAND_DESCRIPTIONS["$cmd"]="$desc"
    CLI_COMMAND_HANDLERS["$cmd"]="$handler"
    CLI_COMMAND_FLAGS["$cmd"]="$flags"
    
    log::debug "Registered command: $cmd -> $handler"
}

#######################################
# Main command dispatcher - call this from resource CLI
# Args: $@ - command line arguments
#######################################
cli::dispatch() {
    local cmd="${1:-help}"
    shift || true
    
    # Handle global flags
    while [[ $# -gt 0 ]]; do
        case "$1" in
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
                # Put the argument back for the command handler
                set -- "$1" "$@"
                break
                ;;
        esac
    done
    
    # Check if command exists
    if [[ -z "${CLI_COMMANDS[$cmd]:-}" ]]; then
        log::error "Unknown command: $cmd"
        echo
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

#######################################
# Generate and display help output
#######################################
cli::_handle_help() {
    echo "Usage: resource-${CLI_RESOURCE_NAME} <command> [options]"
    echo
    echo "$CLI_RESOURCE_DESCRIPTION"
    echo
    echo "Commands:"
    
    # Sort commands for consistent display
    local commands=($(printf '%s\n' "${!CLI_COMMANDS[@]}" | sort))
    
    for cmd in "${commands[@]}"; do
        local desc="${CLI_COMMAND_DESCRIPTIONS[$cmd]}"
        local flags="${CLI_COMMAND_FLAGS[$cmd]:-}"
        
        printf "  %-12s %s" "$cmd" "$desc"
        if [[ "$flags" == *"modifies-system"* ]]; then
            printf " [modifies system]"
        fi
        echo
    done
    
    echo
    echo "Global Options:"
    echo "  --dry-run    Show what would be done without making changes"
    echo "  --help, -h   Show this help message"
    echo
    echo "Examples:"
    echo "  resource-${CLI_RESOURCE_NAME} status"
    echo "  resource-${CLI_RESOURCE_NAME} start --dry-run"
    echo "  resource-${CLI_RESOURCE_NAME} install"
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
        log::error "No status handler found for ${CLI_RESOURCE_NAME}"
        return 1
    fi
}

cli::_handle_start() {
    if command -v "resource_cli::start" &>/dev/null; then
        resource_cli::start "$@"
    elif command -v "${CLI_RESOURCE_NAME}::start" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::start" "$@"
    else
        log::error "No start handler found for ${CLI_RESOURCE_NAME}"
        return 1
    fi
}

cli::_handle_stop() {
    if command -v "resource_cli::stop" &>/dev/null; then
        resource_cli::stop "$@"
    elif command -v "${CLI_RESOURCE_NAME}::stop" &>/dev/null; then
        "${CLI_RESOURCE_NAME}::stop" "$@"
    else
        log::error "No stop handler found for ${CLI_RESOURCE_NAME}"
        return 1
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
        log::error "No install handler found for ${CLI_RESOURCE_NAME}"
        return 1
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
        local desc="${cmd_map[$cmd]}"
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
export -f cli::init
export -f cli::register_command
export -f cli::dispatch
export -f cli::is_running
export -f cli::get_config
export -f cli::register_command_with_validation
export -f cli::register_simple_commands