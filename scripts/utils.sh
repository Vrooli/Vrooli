#!/bin/bash

HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Exit codes
E_NO_TPUT=1

# Set default terminal type if not set
export TERM=${TERM:-xterm}

# Helper function to get color code
get_color_code() {
    local color=$1
    case $color in
    RED) echo "1" ;;
    GREEN) echo "2" ;;
    YELLOW) echo "3" ;;
    BLUE) echo "4" ;;
    MAGENTA) echo "5" ;;
    CYAN) echo "6" ;;
    WHITE) echo "7" ;;
    *) echo "0" ;;
    esac
}

# Initialize a single color
initialize_color() {
    local color_name="$1"
    local color_code=$(get_color_code "$color_name")

    if [ -t 1 ] && command -v tput >/dev/null 2>&1; then
        eval "$color_name=$(tput setaf "$color_code")"
    else
        eval "$color_name=''"
    fi
}

# Initialize color reset
initialize_reset() {
    if [ -t 1 ] && command -v tput >/dev/null 2>&1; then
        RESET=$(tput sgr0)
    else
        RESET=''
    fi
}

# Echo colored text
echo_color() {
    local color="$1"
    local message="$2"

    initialize_color "$color"
    initialize_reset
    echo "${!color}${message}${RESET}"
}

# Print header message
header() {
    echo_color MAGENTA "$1"
}

# Print info message
info() {
    echo_color CYAN "$1"
}

# Print success message
success() {
    echo_color GREEN "$1"
}

# Print error message
error() {
    echo_color RED "$1"
}

# Print warning message
warning() {
    echo_color YELLOW "$1"
}

# Print input prompt message
prompt() {
    echo_color BLUE "$1"
}

# One-line confirmation prompt
prompt_confirm() {
    local message="$1"
    prompt "$message (y/n) "
    read -r -n 1 confirm
    echo
    case "$confirm" in
    [Yy]*) return 0 ;; # User confirmed
    *) return 1 ;;     # User did not confirm
    esac
}

# Exit with error message and code
exit_with_error() {
    local message="$1"
    local code="${2:-1}" # Default to exit code 1 if not provided
    error "$message"
    exit "$code"
}

# Only run a function if the script is executed (not sourced)
run_if_executed() {
    local callback="$1"
    shift # Remove the first argument
    if [[ "${BASH_SOURCE[1]}" == "${0}" ]]; then
        "$callback" "$@"
    fi
}

load_env_file() {
    local environment=$1
    local env_file="$HERE/../.env"

    if [ "$environment" != "development" ] && [ "$environment" != "production" ]; then
        error "Error: Environment must be either development or production."
        exit 1
    fi

    if [ "$environment" = "production" ]; then
        env_file="$HERE/../.env.prod"
    fi

    if [ ! -f "$env_file" ]; then
        error "Error: Environment file $env_file does not exist."
        exit 1
    fi

    . "$env_file"
}
