#!/usr/bin/env bash
set -euo pipefail

# Source var.sh first with relative path
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Now source everything else using var_ variables
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/flow.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/system_commands.sh"
# shellcheck disable=SC1091
source "${var_EXIT_CODES_FILE}"

ports::preflight() {
    if ! system::is_command "lsof"; then
        flow::exit_with_error "Required command 'lsof' not found; please install 'lsof'" "$EXIT_DEPENDENCY_ERROR"
    fi
}

# Validates that the given port argument is a non-empty decimal number between 1 and 65535.
ports::validate_port() {
    local port=$1
    if [[ -z "$port" ]]; then
        flow::exit_with_error "Invalid port: '$port'" "$EXIT_INVALID_ARGUMENT"
    fi
    if ! [[ "$port" =~ ^[0-9]+$ ]]; then
        flow::exit_with_error "Invalid port: '$port'" "$EXIT_INVALID_ARGUMENT"
    fi
    # Force decimal interpretation to avoid octal parsing on leading zeros
    if (( 10#$port < 1 || 10#$port > 65535 )); then
        flow::exit_with_error "Port out of range: $port" "$EXIT_INVALID_ARGUMENT"
    fi
}

# Returns PIDs listening on TCP port $1, or exits on error
ports::get_listening_pids() {
    ports::preflight
    local port=$1
    ports::validate_port "$port"
    local output code=0
    output=$(lsof -tiTCP:"$port" -sTCP:LISTEN 2>&1) || code=$?
    if (( code == 1 )); then
        # No listening process
        echo ""
        return 0
    elif (( code != 0 )); then
        flow::exit_with_error "Error checking port $port: $output" "$EXIT_GENERAL_ERROR"
    fi
    echo "$output"
}

# Returns 0 if TCP port $1 has a listening process
ports::is_port_in_use() {
    ports::preflight
    local port=$1
    ports::validate_port "$port"
    local pids
    pids=$(ports::get_listening_pids "$port")
    [[ -n "$pids" ]]
}

# Kills processes listening on TCP port $1
ports::kill() {
    ports::preflight
    local port=$1
    ports::validate_port "$port"
    local pids
    pids=$(ports::get_listening_pids "$port")
    if [[ -n "$pids" ]]; then
        flow::maybe_run_sudo kill $pids
        local kill_status=$?
        if (( kill_status == 0 )); then
            log::success "Killed processes on port $port: $pids"
        else
            flow::exit_with_error "Failed to kill processes on port $port: $pids" "$EXIT_RESOURCE_ERROR"
        fi
    fi
}

# If port is in use, prompt user to kill blockers
ports::check_and_free() {
    ports::preflight
    local port=$1
    ports::validate_port "$port"
    # Fix default YES to avoid unbound-variable errors
    local yes=${2:-${YES:-}}
    local pids
    pids=$(ports::get_listening_pids "$port")
    if [[ -n "$pids" ]]; then
        log::warning "Port $port is in use by process(es): $pids"
        if flow::is_yes "$yes"; then
            ports::kill "$port"
        else
            if flow::confirm "Kill process(es) listening on port $port?"; then
                ports::kill "$port"
            else
                flow::exit_with_error "Please free port $port and retry" "$EXIT_RESOURCE_ERROR"
            fi
        fi
    fi
} 