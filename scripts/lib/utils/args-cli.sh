#!/usr/bin/env bash
# Utility for centralized argument handling
set -euo pipefail

# Array to hold registered arguments
declare -A ARG_FLAGS=()     # Short flags (-t)
declare -A ARG_NAMES=()     # Long names (--target)
declare -A ARG_DESCS=()     # Descriptions
declare -A ARG_TYPES=()     # Type (value|array)
declare -A ARG_DEFAULTS=()  # Default values
declare -A ARG_REQUIRED=()  # Whether required (yes|no)
declare -A ARG_VALUES=()    # Current values
declare -A ARG_OPTIONS=()   # Available options for display
declare -a ARG_ORDER=()     # Order of arguments for help display
declare -A ARG_NO_VALUE=()  # Arguments that don't require values (like help)

# Used for printing help/error messages
SCRIPT_NAME="$(basename "$0")"

# Reset all registered arguments
args::reset() {
    unset ARG_FLAGS ARG_NAMES ARG_DESCS ARG_TYPES ARG_DEFAULTS ARG_REQUIRED ARG_VALUES ARG_OPTIONS ARG_NO_VALUE ARG_ORDER
    declare -gA ARG_FLAGS ARG_NAMES ARG_DESCS ARG_TYPES ARG_DEFAULTS ARG_REQUIRED ARG_VALUES ARG_OPTIONS ARG_NO_VALUE
    declare -ga ARG_ORDER
}

# Register a new argument
# Usage:
#   args::register \
#     --name "target" \
#     --flag "t" \
#     --desc "Target environment" \
#     --type "value" \
#     --default "native-linux" \
#     --options "native-linux|native-macos|docker|k8s" \
#     --required "yes"
args::register() {
    local name="" flag="" desc="" type="value" default="" options="" required="no" no_value="no"
  
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)     name="$2";     shift 2 ;;
            --flag)     flag="$2";     shift 2 ;;
            --desc)     desc="$2";     shift 2 ;;
            --type)     type="$2";     shift 2 ;;
            --default)  default="$2";  shift 2 ;;
            --options)  options="$2";  shift 2 ;;
            --required) required="$2"; shift 2 ;;
            --no-value) no_value="$2"; shift 2 ;;
            *)
              echo "Error: Unknown parameter $1 for args::register" >&2
              return 1
              ;;
        esac
    done
  
    # Validate required parameters
    if [[ -z "$name" ]]; then
        echo "Error: Missing required parameter --name for args::register" >&2
        return 1
    fi
  
    # Only allow value or array types
    if [[ "$type" != "value" && "$type" != "array" ]]; then
        echo "Error: Type must be 'value' or 'array', got '$type'" >&2
        return 1
    fi
  
    ARG_FLAGS["$name"]="$flag"
    ARG_NAMES["$name"]="$name"
    ARG_DESCS["$name"]="$desc"
    ARG_TYPES["$name"]="$type"
    ARG_DEFAULTS["$name"]="$default"
    ARG_OPTIONS["$name"]="$options"
    ARG_REQUIRED["$name"]="$required"
    ARG_VALUES["$name"]="$default"
    ARG_NO_VALUE["$name"]="$no_value"
    ARG_ORDER+=("$name")
}

# Parse command line arguments according to registered arguments
args::parse() {
    local pos_args=()
    local show_usage=false
    
    while [[ $# -gt 0 ]]; do
        local arg="$1"
        local matched=false
        
        # Special handling for help flag
        if [[ "$arg" == "--help" || "$arg" == "-h" ]]; then
            ARG_VALUES["help"]="yes"
            show_usage=true
            matched=true
            shift
            continue
        fi
        
        # Handle flags and values
        if [[ "$arg" == --* ]]; then
            # Long format (--name)
            local name="${arg:2}"
            
            # Find matching registered argument
            for key in "${!ARG_NAMES[@]}"; do
                if [[ "${ARG_NAMES[$key]}" == "$name" ]]; then
                    # Handle arguments that don't require values
                    if [[ "${ARG_NO_VALUE[$key]}" == "yes" ]]; then
                        ARG_VALUES["$key"]="yes"
                        matched=true
                        shift
                        break
                    fi
                    
                    # Regular arguments require values
                    if [[ $# -lt 2 ]]; then
                        echo "Error: Option $arg requires a value" >&2
                        args::usage
                        exit 1
                    fi
                    
                    if [[ "${ARG_TYPES[$key]}" == "value" ]]; then
                        ARG_VALUES["$key"]="$2"
                        matched=true
                        shift 2
                    elif [[ "${ARG_TYPES[$key]}" == "array" ]]; then
                        # Append to array with space separator
                        if [[ -z "${ARG_VALUES[$key]}" ]]; then
                            ARG_VALUES["$key"]="$2"
                        else
                            ARG_VALUES["$key"]="${ARG_VALUES[$key]} $2"
                        fi
                        matched=true
                        shift 2
                    fi
                    break
                fi
            done
        elif [[ "$arg" == -* ]]; then
            # Short format (-f)
            local flag="${arg:1:1}"
            
            # Find matching registered argument
            for key in "${!ARG_FLAGS[@]}"; do
                if [[ "${ARG_FLAGS[$key]}" == "$flag" ]]; then
                    # Handle arguments that don't require values
                    if [[ "${ARG_NO_VALUE[$key]}" == "yes" ]]; then
                        ARG_VALUES["$key"]="yes"
                        matched=true
                        shift
                        break
                    fi
                    
                    # Regular arguments require values
                    if [[ $# -lt 2 ]]; then
                        echo "Error: Option $arg requires a value" >&2
                        args::usage
                        exit 1
                    fi
                    
                    if [[ "${ARG_TYPES[$key]}" == "value" ]]; then
                        ARG_VALUES["$key"]="$2"
                        matched=true
                        shift 2
                    elif [[ "${ARG_TYPES[$key]}" == "array" ]]; then
                        # Append to array with space separator
                        if [[ -z "${ARG_VALUES[$key]}" ]]; then
                            ARG_VALUES["$key"]="$2"
                        else
                            ARG_VALUES["$key"]="${ARG_VALUES[$key]} $2"
                        fi
                        matched=true
                        shift 2
                    fi
                  break
                fi
            done
        fi
        
        # If not matched as an option, treat as positional argument
        if ! $matched; then
            pos_args+=("$arg")
            shift
        fi
    done
    
    # If help was requested, show usage and exit
    if $show_usage; then
        args::usage
        exit 0
    fi
    
    # Check required arguments
    for key in "${!ARG_REQUIRED[@]}"; do
        if [[ "${ARG_REQUIRED[$key]}" == "yes" && -z "${ARG_VALUES[$key]}" ]]; then
            echo "Error: Required argument --${ARG_NAMES[$key]} is missing" >&2
            args::usage
            exit 1
        fi
    done
    
    # Return positional arguments
    echo "${pos_args[@]:-}"
}

# Get value of a specific argument
# args::get <name>
args::get() {
    local name="$1"
    echo "${ARG_VALUES[$name]:-}"
}

# Export all argument values as environment variables with prefix
# args::export [prefix]
args::export() {
    local prefix="${1:-}"
    
    for key in "${!ARG_VALUES[@]}"; do
        local varname
        if [[ -n "$prefix" ]]; then
            varname="${prefix}_${key}"
        else
            varname="${key}"
        fi
        varname=$(echo "$varname" | tr '[:lower:]' '[:upper:]' | tr '-' '_')
        export "$varname"="${ARG_VALUES[$key]}"
    done
}

# Generate and print a usage message
args::usage() {
    local script_desc="${1:-}"
    
    echo "Usage: $SCRIPT_NAME [OPTIONS]"
    if [[ -n "$script_desc" ]]; then
      echo "$script_desc"
    fi
    echo
    echo "Options:"
    
    # Calculate padding for formatting
    local max_length=0
    for name in "${ARG_ORDER[@]}"; do
      local flag="${ARG_FLAGS[$name]}"
      local options="${ARG_OPTIONS[$name]}"
      local length
      
      if [[ -n "$options" ]]; then
        # Include the space needed for options
        length=$((${#name} + ${#flag} + ${#options} + 10)) # 10 for "-f, --" and " <...>"
      else
        length=$((${#name} + ${#flag} + 7)) # 7 for "-f, --" and space
      fi
      
      if [[ $length -gt $max_length ]]; then
        max_length=$length
      fi
    done
    
    # Print each option with description
    for name in "${ARG_ORDER[@]}"; do
        local flag="${ARG_FLAGS[$name]}"
        local desc="${ARG_DESCS[$name]}"
        local type="${ARG_TYPES[$name]}"
        local default="${ARG_DEFAULTS[$name]}"
        local options="${ARG_OPTIONS[$name]}"
        local required="${ARG_REQUIRED[$name]}"
        local no_value="${ARG_NO_VALUE[$name]}"
        
        local option
        if [[ -n "$flag" ]]; then
            option="  -$flag, --$name"
        else
            option="      --$name"
        fi
        
        # Add value hint with options if specified and not a no-value arg
        if [[ "$no_value" != "yes" ]]; then
            if [[ -n "$options" ]]; then
                option="$option <$options>"
            else
                option="$option <value>"
            fi
        fi
        
        # Add padding
        local padding_length=$((max_length - ${#option}))
        local padding
        printf -v padding "%*s" "$padding_length" ""
        option="$option$padding"
        
        # Add default value if exists
        if [[ -n "$default" ]]; then
            desc="$desc (default: $default)"
        fi
        
        # Add required flag if needed
        if [[ "$required" == "yes" ]]; then
            desc="$desc [REQUIRED]"
        fi
        
        echo "$option $desc"
    done
}  
  
# Returns true if argument is enabled (value is "yes", "true", "1", etc.)
# args::is_enabled <name>
args::is_enabled() {
    local name="$1"
    local value
    value=$(args::get "$name")
    value=$(echo "$value" | tr '[:upper:]' '[:lower:]')
    [[ "$value" == "yes" || "$value" == "true" || "$value" == "1" || "$value" == "on" ]]
}

# Returns true if we should show the help message
args::is_asking_for_help() {
    local args=("$@")
    for arg in "${args[@]}"; do
        if [[ "$arg" == "--help" || "$arg" == "-h" ]]; then
            return 0
        fi
    done
    return 1
}

# ===============================================
# ===== Common arguments for most scripts ===== #
# ===============================================

args::register_help() {
    args::register \
        --name "help" \
        --flag "h" \
        --desc "Show this help message" \
        --type "value" \
        --default "no" \
        --no-value "yes" 
}

args::register_sudo_mode() {  
    args::register \
        --name "sudo-mode" \
        --flag "m" \
        --desc "What to do when encountering sudo commands without elevated privileges" \
        --type "value" \
        --options "error|skip" \
        --default "${SUDO_MODE:-error}"
}

args::register_yes() {      
    args::register \
        --name "yes" \
        --flag "y" \
        --desc "Automatically answer yes to all confirmation prompts" \
        --type "value" \
        --options "yes|no" \
        --default "no"
}

args::register_location() {      
    args::register \
        --name "location" \
        --flag "l" \
        --desc "Override automatic server location detection" \
        --type "value" \
        --options "local|remote" \
        --default "${LOCATION:-}"
}

args::register_target() {
    args::register \
        --name "target" \
        --flag "t" \
        --desc "How the app is being started" \
        --type "value" \
        --options "native-linux|native-macos|docker|k8s" \
        --default "native-linux" \
        --required "yes"
}

args::register_environment() {
    args::register \
        --name "environment" \
        --flag "e" \
        --desc "Skips development-only steps and uses production environment variables" \
        --type "value" \
        --options "development|production" \
        --default "${NODE_ENV:-development}"
}

# Registers --detached flag for skipping teardown in reverse proxy setup
args::register_detached() {
    args::register \
        --name "detached" \
        --flag "x" \
        --desc "Run in detached mode. Frees the terminal and disables teardown events" \
        --type "value" \
        --options "yes|no" \
        --default "no"
}