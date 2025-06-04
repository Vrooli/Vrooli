#!/usr/bin/env bash
# Bash script to check if host has internet access
set -euo pipefail

SETUP_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/log.sh"
# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/exit_codes.sh"

# Check if host has internet access.
# Usage: internet::check_connection [HOST]
#   HOST: Hostname or IP to ping (default: $INTERNET_TEST_HOST or google.com)
internet::check_connection() {
    log::header "Checking host internet access..."

    # Ensure 'ping' command is available
    if ! command -v ping >/dev/null 2>&1; then
        log::error "ping command not found"
        exit "${ERROR_DEPENDENCY_MISSING}"
    fi

    # Determine target host from argument or environment variable
    local host="${1:-${INTERNET_TEST_HOST:-google.com}}"

    # Check DNS resolution if possible
    if command -v getent >/dev/null 2>&1; then
        if ! getent hosts "$host" >/dev/null 2>&1; then
            log::error "Failed to resolve domain $host"
            exit "${ERROR_DOMAIN_RESOLVE}"
        fi
    elif command -v host >/dev/null 2>&1; then
        if ! host "$host" >/dev/null 2>&1; then
            log::error "Failed to resolve domain $host"
            exit "${ERROR_DOMAIN_RESOLVE}"
        fi
    else
        log::warning "Cannot verify DNS resolution: neither 'getent' nor 'host' is available"
    fi

    # Determine appropriate ping command with timeout support
    # Prefer external timeout utilities to avoid ping flag inconsistencies
    local ping_cmd
    if command -v timeout >/dev/null 2>&1; then
        ping_cmd=(timeout 1 ping -c1 "$host")
    elif command -v gtimeout >/dev/null 2>&1; then
        ping_cmd=(gtimeout 1 ping -c1 "$host")
    else
        # Fallback to ping flags
        if ping -c1 -W1 127.0.0.1 >/dev/null 2>&1; then
            ping_cmd=(ping -c1 -W1 "$host")
        elif ping -c1 -w1 127.0.0.1 >/dev/null 2>&1; then
            ping_cmd=(ping -c1 -w1 "$host")
        else
            log::warning "Could not determine ping timeout flags; defaulting to plain ping"
            ping_cmd=(ping -c1 "$host")
        fi
    fi

    # Ping the target host once with timeout
    if "${ping_cmd[@]}" >/dev/null 2>&1; then
        log::success "Host internet access to $host: OK"
    else
        log::error "Host internet access to $host: FAILED"
        exit "${ERROR_NO_INTERNET}"
    fi
}

# If this script is run directly, invoke its main function.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    internet::check_connection "$@"
fi