#!/usr/bin/env bash
set -euo pipefail

# Set default terminal type if not set
export TERM=${TERM:-xterm}

# SMART CACHING: Initialize colors only once
declare -g LOG_COLORS_INITIALIZED="${LOG_COLORS_INITIALIZED:-0}"

# Pre-initialize all colors ONCE when sourced (eliminates repeated tput subshells)
if [[ "$LOG_COLORS_INITIALIZED" == "0" ]]; then
    if [ -t 1 ] && command -v tput >/dev/null 2>&1; then
        # Do all tput calls once and cache results
        RED=$(tput setaf 1)
        GREEN=$(tput setaf 2)
        YELLOW=$(tput setaf 3)
        BLUE=$(tput setaf 4)
        MAGENTA=$(tput setaf 5)
        CYAN=$(tput setaf 6)
        WHITE=$(tput setaf 7)
        RESET=$(tput sgr0)
    else
        # No colors in non-terminal or if tput unavailable
        RED=""
        GREEN=""
        YELLOW=""
        BLUE=""
        MAGENTA=""
        CYAN=""
        WHITE=""
        RESET=""
    fi
    export RED GREEN YELLOW BLUE MAGENTA CYAN WHITE RESET
    LOG_COLORS_INITIALIZED=1
fi

# Legacy functions kept for compatibility but now do nothing (colors already initialized)
log::initialize_color() {
    : # No-op - colors already initialized
}

log::initialize_reset() {
    : # No-op - reset already initialized
}

# Echo colored text (now much faster - no subshells!)
log::echo_color() {
    local color="$1"
    local message="$2"
    local color_value

    # Use indirect variable expansion (no eval needed)
    color_value="${!color}"

    printf '%s%s%s\n' "$color_value" "$message" "$RESET"
}

log::header() {
    log::echo_color MAGENTA "[HEADER]  $*"
}

log::subheader() {
    log::echo_color BLUE "[SECTION] $*"
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

# Aliases for compatibility
log::warn() {
    log::warning "$@"
}

log::debug() {
    # Only show debug messages if DEBUG is set
    if [[ "${DEBUG:-}" == "true" ]]; then
        log::echo_color WHITE "[DEBUG]   $*"
    fi
}
