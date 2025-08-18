#!/usr/bin/env bash
################################################################################
# Unified Argument Parser for Vrooli CLI
# 
# Provides consistent argument parsing across all CLI commands.
# Standardizes format handling, common options, and error handling.
#
# Usage:
#   source cli/lib/arg-parser.sh
#   parse_common_args "$@"
#   parse_format_args "$@"
#
################################################################################

set -euo pipefail

# Source dependencies
CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VROOLI_ROOT="$(cd "$CLI_DIR/../.." && pwd)"
source "${VROOLI_ROOT}/scripts/lib/utils/var.sh" 2>/dev/null || true
source "${var_LOG_FILE}" 2>/dev/null || true

################################################################################
# Parse Common Arguments (help, verbose, etc.)
# Args: "$@"
# Returns: Parsed options as space-separated values
################################################################################
parse_common_args() {
    local verbose="false"
    local help_requested="false"
    local remaining_args=()
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --verbose|-v)
                verbose="true"
                shift
                ;;
            --help|-h)
                help_requested="true"
                shift
                ;;
            --)
                # End of options
                shift
                remaining_args+=("$@")
                break
                ;;
            -*)
                # Unknown option - let caller handle it
                remaining_args+=("$1")
                shift
                ;;
            *)
                # Positional argument
                remaining_args+=("$1")
                shift
                ;;
        esac
    done
    
    # Return parsed values (format: verbose:help:remaining_args...)
    echo "${verbose}:${help_requested}:${remaining_args[*]:-}"
}

################################################################################
# Parse Format Arguments (--json, --format)
# Args: "$@"
# Returns: Parsed format as string
################################################################################
parse_format_args() {
    local format="text"
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --json)
                format="json"
                shift
                ;;
            --format)
                format="${2:-text}"
                shift 2
                ;;
            *)
                # Not a format argument
                break
                ;;
        esac
    done
    
    # Validate format
    if [[ "$format" != "text" && "$format" != "json" ]]; then
        log::error "Invalid format: $format. Use 'text' or 'json'"
        return 1
    fi
    
    echo "$format"
}

################################################################################
# Parse Combined Arguments (common + format)
# Args: "$@"
# Returns: Parsed options as space-separated values
################################################################################
parse_combined_args() {
    local verbose="false"
    local help_requested="false"
    local format="text"
    local remaining_args=()
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --verbose|-v)
                verbose="true"
                shift
                ;;
            --help|-h)
                help_requested="true"
                shift
                ;;
            --json)
                format="json"
                shift
                ;;
            --format)
                format="${2:-text}"
                shift 2
                ;;
            --)
                # End of options
                shift
                remaining_args+=("$@")
                break
                ;;
            -*)
                # Unknown option - let caller handle it
                remaining_args+=("$1")
                shift
                ;;
            *)
                # Positional argument
                remaining_args+=("$1")
                shift
                ;;
        esac
    done
    
    # Validate format
    if [[ "$format" != "text" && "$format" != "json" ]]; then
        log::error "Invalid format: $format. Use 'text' or 'json'"
        return 1
    fi
    
    # Return parsed values (format: verbose:help:format:remaining_args...)
    echo "${verbose}:${help_requested}:${format}:${remaining_args[*]:-}"
}

################################################################################
# Extract Values from Parsed Arguments
# Args: parsed_string value_name
# Returns: The requested value
################################################################################
extract_arg() {
    local parsed="$1"
    local value_name="$2"
    
    case "$value_name" in
        verbose)
            echo "$parsed" | cut -d: -f1
            ;;
        help)
            echo "$parsed" | cut -d: -f2
            ;;
        format)
            echo "$parsed" | cut -d: -f3
            ;;
        remaining)
            echo "$parsed" | cut -d: -f4-
            ;;
        *)
            log::error "Unknown value name: $value_name"
            return 1
            ;;
    esac
}

################################################################################
# Convert Remaining Args String Back to Array
# Args: remaining_args_string
# Returns: Array of remaining arguments
################################################################################
args_to_array() {
    local args_string="$1"
    if [[ -n "$args_string" ]]; then
        echo "$args_string" | tr ' ' '\n' | grep -v '^$'
    fi
}

################################################################################
# Standard Error Handler for Unknown Options
# Args: option_name
################################################################################
handle_unknown_option() {
    local option="$1"
    log::error "Unknown option: $option"
    return 1
}

################################################################################
# Standard Error Handler for Missing Required Arguments
# Args: argument_name usage_message
################################################################################
handle_missing_arg() {
    local arg_name="$1"
    local usage="$2"
    log::error "$arg_name is required"
    echo "Usage: $usage"
    return 1
}

# Export functions
export -f parse_common_args
export -f parse_format_args
export -f parse_combined_args
export -f extract_arg
export -f args_to_array
export -f handle_unknown_option
export -f handle_missing_arg 