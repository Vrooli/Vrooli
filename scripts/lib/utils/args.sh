#!/usr/bin/env bash
################################################################################
# Argument Parsing Utilities
# 
# Provides functions for parsing command-line arguments
################################################################################

# Source var.sh with relative path first
# shellcheck disable=SC1091
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/var.sh"

#######################################
# Parse arguments into associative array
# Globals:
#   ARGS - associative array of parsed arguments
# Arguments:
#   $@ - command line arguments to parse
#######################################
args::parse() {
    declare -gA ARGS
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --*=*)
                # Handle --key=value format
                local key="${1%%=*}"
                key="${key#--}"
                local value="${1#*=}"
                ARGS["$key"]="$value"
                shift
                ;;
            --*)
                # Handle --key value format
                local key="${1#--}"
                if [[ $# -gt 1 && ! "$2" =~ ^- ]]; then
                    ARGS["$key"]="$2"
                    shift 2
                else
                    ARGS["$key"]="true"
                    shift
                fi
                ;;
            -*)
                # Handle short options
                local key="${1#-}"
                if [[ $# -gt 1 && ! "$2" =~ ^- ]]; then
                    ARGS["$key"]="$2"
                    shift 2
                else
                    ARGS["$key"]="true"
                    shift
                fi
                ;;
            *)
                # Positional argument
                shift
                ;;
        esac
    done
}

#######################################
# Get argument value with default
# Arguments:
#   $1 - argument key
#   $2 - default value (optional)
# Returns:
#   Argument value or default
#######################################
args::get() {
    local key="$1"
    local default="${2:-}"
    
    if [[ -v ARGS["$key"] ]]; then
        echo "${ARGS[$key]}"
    else
        echo "$default"
    fi
}

#######################################
# Check if argument exists
# Arguments:
#   $1 - argument key
# Returns:
#   0 if exists, 1 otherwise
#######################################
args::has() {
    local key="$1"
    [[ -v ARGS["$key"] ]]
}

#######################################
# Get all argument keys
# Returns:
#   Space-separated list of keys
#######################################
args::keys() {
    echo "${!ARGS[@]}"
}

# Export functions
export -f args::parse
export -f args::get
export -f args::has
export -f args::keys