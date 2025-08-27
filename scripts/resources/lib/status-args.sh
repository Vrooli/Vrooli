#!/usr/bin/env bash
# Standardized Status Argument Parser
# Provides consistent argument parsing for all resource status commands

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
STATUS_ARGS_DIR="${APP_ROOT}/scripts/resources/lib"
source "${APP_ROOT}/scripts/lib/utils/format.sh"

#######################################
# Parse standard status command arguments
# Sets the following variables in the calling scope:
#   STATUS_FORMAT - Output format (text/json)
#   STATUS_VERBOSE - Verbose mode (true/false)
#   STATUS_FAST - Fast mode flag (true/false)
# Usage:
#   status::parse_args "$@"
#   # Variables are now set in current scope
# Returns:
#   0 on success
#   Sets variables in calling scope
#######################################
status::parse_args() {
    # Initialize defaults
    STATUS_FORMAT="text"
    STATUS_VERBOSE="false"
    STATUS_FAST="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --format)
                if [[ $# -gt 1 ]]; then
                    STATUS_FORMAT="$2"
                    shift 2
                else
                    shift
                fi
                ;;
            --json)
                STATUS_FORMAT="json"
                shift
                ;;
            --verbose|-v)
                STATUS_VERBOSE="true"
                shift
                ;;
            --fast)
                STATUS_FAST="true"
                shift
                ;;
            --help|-h)
                status::show_help
                return 1
                ;;
            *)
                # Unknown argument, skip it
                shift
                ;;
        esac
    done
    
    # Export for use in subshells if needed
    export STATUS_FORMAT
    export STATUS_VERBOSE
    export STATUS_FAST
    
    return 0
}

#######################################
# Parse arguments and return as associative array declaration
# For resources that prefer associative arrays
# Usage:
#   eval "$(status::parse_args_map "$@")"
#   echo "${status_args[format]}"
# Returns:
#   Outputs bash declare statement for associative array
#######################################
status::parse_args_map() {
    local format="text"
    local verbose="false"
    local fast="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --format)
                if [[ $# -gt 1 ]]; then
                    format="$2"
                    shift 2
                else
                    shift
                fi
                ;;
            --json)
                format="json"
                shift
                ;;
            --verbose|-v)
                verbose="true"
                shift
                ;;
            --fast)
                fast="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Output associative array declaration
    echo "declare -A status_args=("
    echo "  [format]=\"$format\""
    echo "  [verbose]=\"$verbose\""
    echo "  [fast]=\"$fast\""
    echo ")"
}

#######################################
# Show standard help message for status commands
#######################################
status::show_help() {
    cat <<EOF
Usage: resource-<name> status [OPTIONS]

Options:
  --format <type>  Output format (text, json) [default: text]
  --json          Shorthand for --format json
  --verbose, -v   Show detailed information
  --fast          Skip expensive operations for faster response
  --help, -h      Show this help message

Examples:
  resource-<name> status
  resource-<name> status --json
  resource-<name> status --verbose
  resource-<name> status --fast
  resource-<name> status --format json --verbose

EOF
}

#######################################
# Build standard collect_data arguments based on parsed flags
# Usage:
#   local collect_args=$(status::build_collect_args)
#   data=$(resource::status::collect_data $collect_args)
# Returns:
#   String of arguments to pass to collect_data
#######################################
status::build_collect_args() {
    local args=""
    
    # Check if variables are set, if not parse from current args
    if [[ -z "${STATUS_FAST:-}" ]]; then
        status::parse_args "$@"
    fi
    
    if [[ "${STATUS_FAST}" == "true" ]]; then
        args="--fast"
    fi
    
    echo "$args"
}

#######################################
# Standard status function wrapper
# Provides complete status implementation with argument parsing
# Usage:
#   status::run_standard "resource_name" "collect_function" "display_function" "$@"
# Example:
#   status::run_standard "vault" "vault::status::collect_data" "vault::status::display_text" "$@"
#######################################
status::run_standard() {
    local resource_name="$1"
    local collect_function="$2"
    local display_function="$3"
    shift 3
    
    # Parse arguments
    if ! status::parse_args "$@"; then
        return 1
    fi
    
    # Build collect arguments
    local collect_args
    collect_args=$(status::build_collect_args)
    
    # Collect status data
    local data_string
    data_string=$($collect_function $collect_args 2>/dev/null)
    
    if [[ -z "$data_string" ]]; then
        # Fallback if data collection fails
        if [[ "$STATUS_FORMAT" == "json" ]]; then
            echo "{\"error\": \"Failed to collect status data for $resource_name\"}"
        else
            echo "Failed to collect $resource_name status data"
        fi
        return 1
    fi
    
    # Convert string to array
    local data_array
    mapfile -t data_array <<< "$data_string"
    
    # Output based on format
    if [[ "$STATUS_FORMAT" == "json" ]]; then
        # Assume format::output is available
        if command -v format::output &>/dev/null; then
            format::output "json" "kv" "${data_array[@]}"
        else
            # Basic JSON fallback
            echo "{"
            local first=true
            for ((i=0; i<${#data_array[@]}; i+=2)); do
                [[ "$first" == "false" ]] && echo ","
                echo -n "  \"${data_array[i]}\": \"${data_array[i+1]}\""
                first=false
            done
            echo ""
            echo "}"
        fi
    else
        # Text format with display function
        if command -v "$display_function" &>/dev/null; then
            $display_function "${data_array[@]}"
        else
            # Basic text fallback
            echo "$resource_name Status:"
            for ((i=0; i<${#data_array[@]}; i+=2)); do
                echo "  ${data_array[i]}: ${data_array[i+1]}"
            done
        fi
    fi
    
    # Return appropriate exit code based on health
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

# Export functions for use by other scripts
export -f status::parse_args
export -f status::parse_args_map
export -f status::show_help
export -f status::build_collect_args
export -f status::run_standard