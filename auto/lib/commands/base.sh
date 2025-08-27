#!/usr/bin/env bash

# Base Command Interface for Auto Loop System
# Provides common utilities and interface contract for command modules

set -euo pipefail

# Command interface contract:
# Each command module must implement:
# - cmd_execute() - main execution function
# - cmd_help() - help text for the command
# Optional:
# - cmd_validate() - input validation
# - cmd_init() - initialization logic

#######################################
# Validate command module interface
# Arguments:
#   $1 - Command name for error reporting
# Returns: 0 if valid, 1 if invalid
#######################################
commands::validate_interface() {
    local cmd_name="${1:-unknown}"
    
    # Check required functions exist
    if ! declare -F cmd_execute >/dev/null 2>&1; then
        echo "ERROR: Command '$cmd_name' missing required function: cmd_execute" >&2
        return 1
    fi
    
    if ! declare -F cmd_help >/dev/null 2>&1; then
        echo "ERROR: Command '$cmd_name' missing required function: cmd_help" >&2
        return 1
    fi
    
    return 0
}

#######################################
# Load and validate a command module
# Arguments:
#   $1 - Path to command module file
#   $2 - Command name for validation
# Returns: 0 if loaded successfully, 1 on failure
#######################################
commands::load_module() {
    local module_path="$1"
    local cmd_name="$2"
    
    if [[ ! -f "$module_path" ]]; then
        echo "ERROR: Command module not found: $module_path" >&2
        return 1
    fi
    
    # Source the module
    # shellcheck disable=SC1090
    if ! source "$module_path"; then
        echo "ERROR: Failed to source command module: $module_path" >&2
        return 1
    fi
    
    # Validate interface
    if ! commands::validate_interface "$cmd_name"; then
        return 1
    fi
    
    # Call optional initialization
    if declare -F cmd_init >/dev/null 2>&1; then
        if ! cmd_init; then
            echo "ERROR: Command '$cmd_name' initialization failed" >&2
            return 1
        fi
    fi
    
    return 0
}

#######################################
# Execute a command with validation
# Arguments:
#   $1 - Command name
#   $@ - Command arguments
# Returns: Exit code from command execution
#######################################
commands::execute() {
    local cmd_name="$1"
    shift || true
    
    # Validate input if function exists
    if declare -F cmd_validate >/dev/null 2>&1; then
        if ! cmd_validate "$@"; then
            echo "ERROR: Command '$cmd_name' validation failed" >&2
            return 1
        fi
    fi
    
    # Execute the command
    cmd_execute "$@"
}

#######################################
# Get help for a command
# Arguments:
#   $1 - Command name
# Returns: 0 on success
#######################################
commands::help() {
    local cmd_name="${1:-unknown}"
    
    if declare -F cmd_help >/dev/null 2>&1; then
        cmd_help
    else
        echo "No help available for command: $cmd_name"
    fi
}

#######################################
# Common utilities for command modules
#######################################

# Check if a PID is running
commands::pid_running() {
    local pid="$1"
    kill -0 "$pid" 2>/dev/null
}

# Get current loop status
commands::get_loop_status() {
    if [[ -f "$PID_FILE" ]]; then
        local pid; pid=$(cat "$PID_FILE")
        if commands::pid_running "$pid"; then
            echo "RUNNING"
            return 0
        else
            echo "STOPPED"
            return 1
        fi
    else
        echo "STOPPED"
        return 1
    fi
}

# Safe file operations with error handling
commands::safe_remove() {
    local file="$1"
    if [[ -f "$file" ]]; then
        rm -f "$file"
    fi
}

commands::safe_read() {
    local file="$1"
    if [[ -f "$file" ]]; then
        cat "$file"
    else
        echo ""
    fi
}

# Export utilities for command modules
export -f commands::validate_interface
export -f commands::load_module
export -f commands::execute
export -f commands::help
export -f commands::pid_running
export -f commands::get_loop_status
export -f commands::safe_remove
export -f commands::safe_read