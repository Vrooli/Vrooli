#!/usr/bin/env bash
set -euo pipefail

UTILS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${UTILS_DIR}/log.sh"

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
    # Determine behavior based on SUDO_MODE
    local mode=${SUDO_MODE:-skip}

    # skip mode always returns non-zero (no sudo)
    if [[ "$mode" == "skip" ]]; then
        log::info "SUDO_MODE=skip: skipping privileged operations"
        return 1
    fi

    # If sudo is not installed, skip privileged ops
    # NOTE: Don't use system::is_command here, because it will cause a circular dependency
    if ! command -v sudo >/dev/null 2>&1; then
        log::info "sudo not found: skipping privileged operations"
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
        flow::exit_with_error "Privileged operations require sudo access, but unable to run sudo" "$ERROR_DEFAULT"
    else
        log::info "sudo requires password or is blocked, skipping privileged operations"
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