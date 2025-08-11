#!/usr/bin/env bash
set -euo pipefail

# Source var.sh with relative path first
# shellcheck disable=SC1091
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/var.sh"
# shellcheck disable=SC1091
source "${var_EXIT_CODES_FILE}"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/sudo.sh"

# Returns 0 if the first argument is a recognized "yes" (y/yes), else 1.
flow::is_yes() {
    # Convert to lowercase. Note that [A-Z] and [a-z] in `tr` are POSIX,
    # but using '[:upper:]' and '[:lower:]' is typically safer for all locales.
    ans=$(echo "$1" | tr '[:upper:]' '[:lower:]')

    case "$ans" in
    y | yes | true | 1)
        return 0
        ;;
    *)
        return 1
        ;;
    esac
}

# One-line confirmation prompt
flow::confirm() {
    # If auto-confirm is enabled, skip the prompt
    if flow::is_yes "${YES:-}"; then
        log::info "Auto-confirm enabled, skipping prompt"
        return 0
    fi
    local message="$1"
    log::prompt "$message (y/n) "
    read -r -n 1 confirm
    echo
    case "$confirm" in
    [Yy]*) return 0 ;; # User confirmed
    *) return 1 ;;     # User did not confirm
    esac
}

# Exit with error message and code
flow::exit_with_error() {
    local message="$1"
    local code="${2:-1}" # Default to exit code 1 if not provided
    log::error "$message"
    exit "$code"
}

# Continue with error message and code
flow::continue_with_error() {
    local message="$1"
    local code="${2:-1}" # Default to exit code 1 if not provided
    log::error "$message"
    return "$code"
}

#  Checks if sudo can be invoked according to SUDO_MODE.
# Returns 0 if sudo operations should proceed, 1 if they should be skipped.
# Exits with error if SUDO_MODE is 'error' and sudo is unavailable.
flow::can_run_sudo() {
    # Track what operation is requesting sudo for better error messages
    local operation="${1:-privileged operations}"
    
    # Determine behavior based on SUDO_MODE
    local mode=${SUDO_MODE:-skip}

    # skip mode always returns non-zero (no sudo)
    if [[ "$mode" == "skip" ]]; then
        log::info "SUDO_MODE=skip: skipping $operation"
        return 1
    fi

    # If already running as root, we have privileges
    if [[ $EUID -eq 0 ]]; then
        return 0
    fi

    # If sudo is not installed, skip privileged ops
    # NOTE: Don't use system::is_command here, because it will cause a circular dependency
    if ! command -v sudo >/dev/null 2>&1; then
        log::info "sudo not found: skipping $operation"
        return 1
    fi

    # Test for passwordless sudo, but don't let 'set -e' kill us
    set +e
    sudo -n true >/dev/null 2>&1
    local status=$?
    set -e

    if [[ $status -eq 0 ]]; then
        return 0
    fi

    # If error mode, abort; otherwise just skip
    if [[ "$mode" == "error" ]]; then
        if sudo::is_running_as_sudo; then
            flow::exit_with_error "Running under sudo but can't execute privileged operations for: $operation. This usually means sudo has timed out or there's a permission issue." "$EXIT_GENERAL_ERROR"
        else
            flow::exit_with_error "Unable to run sudo for: $operation. Either run setup with 'sudo' prefix, or use --sudo-mode skip to bypass privileged operations." "$EXIT_GENERAL_ERROR"
        fi
    else
        log::info "sudo requires password or is blocked, skipping $operation"
        return 1
    fi
}

flow::maybe_run_sudo() {
    if flow::can_run_sudo; then
        sudo "$@"
    else
        "$@"
    fi
}

