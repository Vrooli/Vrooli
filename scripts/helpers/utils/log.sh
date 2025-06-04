#!/usr/bin/env bash
set -euo pipefail

# Set default terminal type if not set
export TERM=${TERM:-xterm}

# Helper function to get color code
log::get_color_code() {
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
log::initialize_color() {
    local color_name="$1"
    local color_code
    color_code=$(log::get_color_code "$color_name")

    # NOTE: Don't use system::is_command here, because it will cause a circular dependency
    if [ -t 1 ] && command -v tput >/dev/null 2>&1; then
        eval "$color_name=$(tput setaf "$color_code")"
    else
        eval "$color_name=''"
    fi
}

# Initialize color reset
log::initialize_reset() {
    # NOTE: Don't use system::is_command here, because it will cause a circular dependency
    if [ -t 1 ] && command -v tput >/dev/null 2>&1; then
        RESET=$(tput sgr0)
    else
        RESET=''
    fi
}

# Echo colored text
log::echo_color() {
    local color="$1"
    local message="$2"
    local color_value

    log::initialize_color "$color"
    log::initialize_reset

    eval "color_value=\$$color"

    printf '%s%s%s\n' "$color_value" "$message" "$RESET"
}

log::header() {
    log::echo_color MAGENTA "[HEADER]  $*"
}

log::info() {
    log::echo_color CYAN "[INFO]    $*"
}

log::success() {
    log::echo_color GREEN "[SUCCESS] $*"
}

log::error() {
    log::echo_color RED "[ERROR]   $*"
}

log::warning() {
    log::echo_color YELLOW "[WARNING] $*"
}

log::prompt() {
    log::echo_color BLUE "[PROMPT]  $*"
}
