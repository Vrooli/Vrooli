#!/usr/bin/env bash
# Posix-compliant script to fix system clock. 
# An accurate system clock is often needed for installing/updating packages, and can 
# occasionally be set incorrectly.
set -euo pipefail

SETUP_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/flow.sh"
# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/log.sh"
# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/system.sh"

# Fix the system clock
clock::fix() {
    local can_sudo
    # Check for sudo capability; warn if unavailable but continue
    if ! flow::can_run_sudo; then
        log::warning "Skipping system clock accuracy check due to insufficient permissions"
        can_sudo=0
    else
        can_sudo=1
    fi

    # Always print header
    log::header "Making sure the system clock is accurate"
    # Only run hwclock if sudo is available
    if [ "$can_sudo" -eq 1 ]; then
        if system::is_command "hwclock"; then
            sudo hwclock -s
        else
            log::warning "hwclock command not found, skipping hwclock -s."
        fi
    fi

    # Print info
    log::info "System clock is now: $(date)"
}

# If this script is run directly, invoke its main function.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    clock::fix "$@"
fi