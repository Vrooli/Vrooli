#!/bin/bash
# WARNING: This is used by some shell scripts, so be careful not to use any
# Bash-specific features outside of functions.

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
    color="$1"
    message="$2"

    initialize_color "$color"
    initialize_reset

    eval "color_value=\$$color"

    printf '%s%s%s\n' "$color_value" "$message" "$RESET"
}

# Print header message
header() {
    echo_color MAGENTA "[HEADER]  $*"
}

# Print info message
info() {
    echo_color CYAN "[INFO]    $*"
}

# Print success message
success() {
    echo_color GREEN "[SUCCESS] $*"
}

# Print error message
error() {
    echo_color RED "[ERROR]   $*"
}

# Print warning message
warning() {
    echo_color YELLOW "[WARNING] $*"
}

# Print input prompt message
prompt() {
    echo_color BLUE "[PROMPT]  $*"
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
    HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

    local environment=$1
    local env_file="$HERE/../.env"

    if [ "$environment" != "development" ] && [ "$environment" != "production" ]; then
        error "Error: Environment must be either development or production."
        exit 1
    fi

    if [ "$environment" = "production" ]; then
        env_file="$HERE/../.env-prod"
    fi

    if [ ! -f "$env_file" ]; then
        error "Error: Environment file $env_file does not exist."
        exit 1
    fi

    . "$env_file"
}

is_running_in_ci() {
    [[ "$CI" == "true" ]]
}

# ------------------------------------------------------------------
# Runs a command, prints an error, and exits if the command fails.
# Usage:
#   run_step "Friendly step description" "actual_command_here"
# ------------------------------------------------------------------
run_step() {
    local step_description="$1"
    shift
    local cmd="$*"

    info "${step_description}..."
    if ! eval "${cmd}"; then
        error "Failed: ${step_description}"
        exit 1
    fi
    success "${step_description} - done!"
}

# ------------------------------------------------------------------
# Runs a command, prints an error if it fails, but does NOT exit.
# Usage:
#   run_step_noncritical "Friendly step description" "actual_command_here"
# ------------------------------------------------------------------
run_step_noncritical() {
    local step_description="$1"
    shift
    local cmd="$*"

    info "${step_description}..."
    if ! eval "${cmd}"; then
        warning "Non-critical step failed: ${step_description}"
        # We do not exit here
        return 1
    fi
    success "${step_description} - done!"
}

# ------------------------------------------------------------------------------
# is_yes: Returns 0 if the first argument is a recognized "yes" (y/yes), else 1.
# Usage:
#     if is_yes "$SOME_VAR"; then
#         echo "It's a yes!"
#     else
#         echo "It's a no!"
#     fi
# ------------------------------------------------------------------------------
is_yes() {
    # Convert to lowercase. Note that [A-Z] and [a-z] in `tr` are POSIX,
    # but using '[:upper:]' and '[:lower:]' is typically safer for all locales.
    ans=$(echo "$1" | tr '[:upper:]' '[:lower:]')

    case "$ans" in
    y | yes)
        return 0
        ;;
    *)
        return 1
        ;;
    esac
}
