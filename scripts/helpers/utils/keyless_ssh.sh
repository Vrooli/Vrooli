#!/usr/bin/env bash
# This script sets up keyless SSH access to a remote server.

UTILS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${UTILS_DIR}/log.sh"
# shellcheck disable=SC1091
source "${UTILS_DIR}/var.sh"

CONN_TIMEOUT_S=10
SETUP_MODE=false

# Returns the SSH key path to use for connections.
# Uses SSH_KEY_PATH if set, otherwise defaults to ~/.ssh/id_rsa_${SITE_IP}
keyless_ssh::get_key_path() {
    if [ -n "${SSH_KEY_PATH:-}" ]; then
        echo "$SSH_KEY_PATH"
    elif [ -n "${SITE_IP:-}" ]; then
        echo "$HOME/.ssh/id_rsa_${SITE_IP}"
    else
        echo "No SSH key path or SITE_IP set" >&2
        return 1
    fi
}

keyless_ssh::get_remote_server() {
    if [ -n "${SITE_IP:-}" ]; then
        echo "root@${SITE_IP}"
    else
        log::error "No SITE_IP set"
        return 1
    fi
}

# Checks if there is already a public key for the project
keyless_ssh::check_key() {
    local key_name=$(keyless_ssh::get_key_path)
    local remote_server=$(keyless_ssh::get_remote_server)
    if [ ! -f "${key_name}" ]; then
        # Generate a new SSH key pair for the project
        ssh-keygen -t rsa -f "${key_name}" -N ""
        # Copy the public key to the remote host
        cat "${key_name}.pub" | ssh ${remote_server} 'chmod 700 ~/.ssh; chmod 600 ~/.ssh/authorized_keys; cat >> ~/.ssh/authorized_keys'
        if [ $? -ne 0 ]; then
            log::error "Failed to copy public key to remote host. Exiting..."
            rm "${key_name}"*
            exit 1
        fi
    fi
}

keyless_ssh::remove_key_file() {
    local key_name=$(keyless_ssh::get_key_path)
    if [ -f "${key_name}" ] && [ $(find "${key_name}" -mmin -5 | wc -l) -gt 0 ]; then
        rm "${key_name}"*
    fi
}

keyless_ssh::connect() {
    local key_name=$(keyless_ssh::get_key_path)
    local remote_server=$(keyless_ssh::get_remote_server)
    log::info "Testing SSH connection to ${remote_server}..."
    ssh -i "${key_name}" -o "BatchMode=yes" -o "ConnectTimeout=${CONN_TIMEOUT_S}" ${remote_server} "echo 2>&1" >/dev/null
    RET=$?
    if [ ${RET} -ne 0 ]; then
        log::error "SSH connection failed: ${RET}. Retrying after removing old host key..."
        # Remove the known hosts entry for the remote server
        ssh-keygen -R ${SITE_IP}
        # Retry the SSH connection
        ssh -i "${key_name}" -o "BatchMode=yes" -o "ConnectTimeout=${CONN_TIMEOUT_S}" ${remote_server} "echo 2>&1" >/dev/null
        RET=$?
        if [ ${RET} -ne 0 ]; then
            log::error "SSH connection still failed: ${RET}. Exiting..."
            keyless_ssh::remove_key_file
            exit 1
        fi
    fi
    log::success "✅ SSH connection successful."
}
