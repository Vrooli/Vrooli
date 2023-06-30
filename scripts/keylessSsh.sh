#!/bin/bash
# This script sets up keyless SSH access to a remote server

HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
. "${HERE}/prettify.sh"

# Connection timeout in seconds
CONN_TIMEOUT=10

# Load variables from .env file
if [ -f "${HERE}/../.env" ]; then
    . "${HERE}/../.env"
else
    error "Could not find .env file. Exiting..."
    exit 1
fi

# Set the remote server location, using SITE_IP from .env
remote_server="root@${SITE_IP}"
info "Remote server: ${remote_server}"

# Remove local key pair if it was just generated within the last 5 minutes
# This allows us to try again without having to manually delete the key pair,
# while also keeping the key pair around if it was successful before
remove_key_file() {
    key_name="id_rsa_${SITE_IP}"
    if [ -f ~/.ssh/${key_name} ] && [ $(find ~/.ssh/${key_name} -mmin -5 | wc -l) -gt 0 ]; then
        rm ~/.ssh/${key_name}*
    fi
}

# Check if there is already a public key for the project
key_name="id_rsa_${SITE_IP}"
if [ ! -f ~/.ssh/${key_name} ]; then
    # Generate a new SSH key pair for the project
    ssh-keygen -t rsa -f ~/.ssh/${key_name} -N ""
    # Copy the public key to the remote host
    cat ~/.ssh/${key_name}.pub | ssh ${remote_server} 'chmod 700 ~/.ssh; chmod 600 ~/.ssh/authorized_keys; cat >> ~/.ssh/authorized_keys'
    if [ $? -ne 0 ]; then
        error "Failed to copy public key to remote host. Exiting..."
        rm ~/.ssh/${key_name}*
        exit 1
    fi
fi

# Test the SSH connection by running a command on the remote host
info "Testing SSH connection..."
ssh -i ~/.ssh/${key_name} -o "BatchMode=yes" -o "ConnectTimeout=${CONN_TIMEOUT}" ${remote_server} "echo 2>&1" >/dev/null
RET=$?
if [ ${RET} -ne 0 ]; then
    error "SSH connection failed: ${RET}. Retrying after removing old host key..."
    # Remove the known hosts entry for the remote server
    ssh-keygen -R ${SITE_IP}
    # Retry the SSH connection
    ssh -i ~/.ssh/${key_name} -o "BatchMode=yes" -o "ConnectTimeout=${CONN_TIMEOUT}" ${remote_server} "echo 2>&1" >/dev/null
    RET=$?
    if [ ${RET} -ne 0 ]; then
        error "SSH connection still failed: ${RET}. Exiting..."
        rm ~/.ssh/${key_name}*
        exit 1
    fi
fi
success "SSH connection successful."
