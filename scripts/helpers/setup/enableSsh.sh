#!/usr/bin/env bash
# Posix-compliant script to make sure they keyless ssh login is enabled
set -euo pipefail

SETUP_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/flow.sh"
# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/log.sh"

# enable PasswordAuthentication for ssh
enableSsh::enable_password_authentication() {
    if ! flow::can_run_sudo; then
        log::warning "Skipping PasswordAuthentication setup due to sudo mode"
        return
    fi

    # Determine sed inline-edit options for portability
    local sed_opts
    if sed --version >/dev/null 2>&1; then
        sed_opts=(-i)
    else
        sed_opts=(-i '')
    fi

    log::header "Enabling PasswordAuthentication"
    sudo sed "${sed_opts[@]}" 's/#\?PasswordAuthentication .*/PasswordAuthentication yes/g' /etc/ssh/sshd_config
    sudo sed "${sed_opts[@]}" 's/#\?PubkeyAuthentication .*/PubkeyAuthentication yes/g' /etc/ssh/sshd_config
    sudo sed "${sed_opts[@]}" 's/#\?AuthorizedKeysFile .*/AuthorizedKeysFile .ssh\/authorized_keys/g' /etc/ssh/sshd_config
}

# Ensure .ssh directory and authorized_keys file exist with correct permissions
enableSsh::ensure_ssh_files() {
    mkdir -p ~/.ssh
    touch ~/.ssh/authorized_keys
    chmod 700 ~/.ssh
    chmod 600 ~/.ssh/authorized_keys
}

# Try restarting SSH service, checking for both common service names
enableSsh::restart_ssh() {
    if ! flow::can_run_sudo; then
        log::warning "Skipping SSH restart due to sudo mode"
        return
    fi

    if ! sudo systemctl restart sshd 2>/dev/null; then
        if ! sudo systemctl restart ssh 2>/dev/null; then
            log::error "Failed to restart ssh. Exiting with error."
            exit 1
        fi
    fi
}

enableSsh::setup_ssh() {
    log::header "Setting up SSH"

    enableSsh::ensure_ssh_files
    enableSsh::enable_password_authentication
    enableSsh::restart_ssh
}

# If this script is run directly, invoke its main function.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    enableSsh::setup_ssh "$@"
fi

