#!/usr/bin/env bash

# Modular Command Dispatcher
# Provides plugin-style command discovery and execution

set -euo pipefail

# Get the commands directory
COMMANDS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source base command interface
# shellcheck disable=SC1091
source "$COMMANDS_DIR/base.sh"

#######################################
# Command mapping - maps command names to module files
# This provides flexibility for aliases and legacy compatibility
#######################################
declare -A COMMAND_MAP=(
    # Process management
    ["run-loop"]="process/run-loop.sh"
    ["start"]="process/start.sh"
    ["stop"]="process/stop.sh"
    ["force-stop"]="process/stop.sh --force"
    ["status"]="process/status.sh"
    
    # Log management  
    ["logs"]="logging/show.sh"
    ["rotate"]="logging/rotate.sh"
    
    # Data processing
    ["json"]="data/json.sh"
    ["dry-run"]="data/dry-run.sh"
    
    # Health and diagnostics
    ["health"]="health/health.sh"
    ["once"]="health/once.sh"
    ["skip-wait"]="health/skip-wait.sh"
)

#######################################
# Discover available commands
# Returns: List of available command names
#######################################
commands::discover() {
    printf '%s\n' "${!COMMAND_MAP[@]}" | sort
}

#######################################
# Check if a command exists
# Arguments:
#   $1 - Command name
# Returns: 0 if exists, 1 if not
#######################################
commands::exists() {
    local cmd_name="$1"
    [[ -n "${COMMAND_MAP[$cmd_name]:-}" ]]
}

#######################################
# Get module path for a command
# Arguments:
#   $1 - Command name
# Returns: Module file path
#######################################
commands::get_module_path() {
    local cmd_name="$1"
    local module_spec="${COMMAND_MAP[$cmd_name]:-}"
    
    if [[ -z "$module_spec" ]]; then
        echo "ERROR: Unknown command: $cmd_name" >&2
        return 1
    fi
    
    # Extract module path (everything before first space)
    local module_path="${module_spec%% *}"
    echo "$COMMANDS_DIR/$module_path"
}

#######################################
# Get additional arguments for a command
# Arguments:
#   $1 - Command name
# Returns: Additional arguments (if any)
#######################################
commands::get_module_args() {
    local cmd_name="$1"
    local module_spec="${COMMAND_MAP[$cmd_name]:-}"
    
    if [[ -z "$module_spec" ]]; then
        return 0
    fi
    
    # Extract arguments (everything after first space)
    if [[ "$module_spec" =~ \ (.+)$ ]]; then
        echo "${BASH_REMATCH[1]}"
    fi
}

#######################################
# Load and execute a command module
# Arguments:
#   $1 - Command name
#   $@ - Command arguments
# Returns: Exit code from command execution
#######################################
commands::dispatch() {
    local cmd_name="${1:-}"
    shift || true
    
    if [[ -z "$cmd_name" ]]; then
        echo "ERROR: No command specified" >&2
        commands::show_help
        return 1
    fi
    
    # Check if command exists
    if ! commands::exists "$cmd_name"; then
        echo "ERROR: Unknown command: $cmd_name" >&2
        echo ""
        echo "Available commands:"
        commands::discover | sed 's/^/  /'
        return 1
    fi
    
    # Get module path and additional args
    local module_path; module_path=$(commands::get_module_path "$cmd_name")
    local module_args; module_args=$(commands::get_module_args "$cmd_name")
    
    # Prepare arguments array
    local -a final_args=()
    if [[ -n "$module_args" ]]; then
        # shellcheck disable=SC2086
        read -ra module_arg_array <<< $module_args
        final_args+=("${module_arg_array[@]}")
    fi
    final_args+=("$@")
    
    # Load and execute the command module
    if commands::load_module "$module_path" "$cmd_name"; then
        commands::execute "$cmd_name" "${final_args[@]}"
    else
        echo "ERROR: Failed to load command module for: $cmd_name" >&2
        return 1
    fi
}

#######################################
# Show help for a specific command
# Arguments:
#   $1 - Command name
# Returns: 0 on success
#######################################
commands::show_command_help() {
    local cmd_name="$1"
    
    if ! commands::exists "$cmd_name"; then
        echo "ERROR: Unknown command: $cmd_name" >&2
        return 1
    fi
    
    local module_path; module_path=$(commands::get_module_path "$cmd_name")
    
    if commands::load_module "$module_path" "$cmd_name"; then
        commands::help "$cmd_name"
    else
        echo "ERROR: Failed to load command module for: $cmd_name" >&2
        return 1
    fi
}

#######################################
# Show general help and available commands
#######################################
commands::show_help() {
    cat << EOF
Auto Loop Command Dispatcher

Usage: loop_dispatch <command> [arguments...]

Available Commands:

Process Management:
  run-loop [--max N]     Execute main loop in foreground
  start                  Start loop in background
  stop [--force]         Stop loop gracefully or forcefully
  force-stop            Force stop loop immediately
  status                Show loop status and configuration

Log Management:
  logs [-f|--follow]    Show or follow log file
  rotate [--events [N]] [--temp]  Rotate logs and cleanup

Data Processing:
  json <subcommand>     JSON data queries and summaries
  dry-run              Show configuration preview

Health & Diagnostics:
  health               Comprehensive system health check
  once                 Run single iteration synchronously
  skip-wait           Skip current iteration wait

For detailed help on a specific command:
  loop_dispatch <command> --help

Examples:
  loop_dispatch start
  loop_dispatch run-loop --max 5
  loop_dispatch logs --follow
  loop_dispatch rotate --events 10 --temp
EOF
}

#######################################
# Main dispatcher entry point
# Arguments:
#   $1 - Command name (or 'help')
#   $@ - Command arguments
# Returns: Exit code from command execution
#######################################
loop_dispatch_modular() {
    local cmd="${1:-help}"
    shift || true
    
    case "$cmd" in
        help|--help|-h)
            commands::show_help
            return 0
            ;;
        help-*)
            # Handle help-<command> format
            local target_cmd="${cmd#help-}"
            commands::show_command_help "$target_cmd"
            return 0
            ;;
        *)
            # Check for --help flag in arguments
            for arg in "$@"; do
                if [[ "$arg" == "--help" || "$arg" == "-h" ]]; then
                    commands::show_command_help "$cmd"
                    return 0
                fi
            done
            
            # Dispatch to command module
            commands::dispatch "$cmd" "$@"
            ;;
    esac
}

# Export the main dispatcher function
export -f loop_dispatch_modular