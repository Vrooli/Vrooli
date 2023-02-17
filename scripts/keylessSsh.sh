#!/bin/bash
# This script sets up keyless SSH access to a remote server

HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "${HERE}/prettify.sh"

# Load variables from .env file
if [ -f "${HERE}/../.env" ]; then
    source "${HERE}/../.env"
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
    if [ -f ~/.ssh/id_rsa ] && [ $(find ~/.ssh/id_rsa -mmin -5 | wc -l) -gt 0 ]; then
        rm ~/.ssh/id_rsa*
    fi
}

# Check if there is already a public key on the local machine
if [ ! -f ~/.ssh/id_rsa ]; then
    # Generate a new SSH key pair
    ssh-keygen -t rsa -f ~/.ssh/id_rsa -N ""
    # Copy the public key to the remote host
    cat ~/.ssh/id_rsa.pub | ssh ${remote_server} 'chmod 700 ~/.ssh; chmod 600 ~/.ssh/authorized_keys; cat >> ~/.ssh/authorized_keys'
    if [ $? -ne 0 ]; then
        error "Failed to copy public key to remote host. Exiting..."
        remove_key_file
        exit 1
    fi
fi

# Test the SSH connection by running a command on the remote host
info "Testing SSH connection..."
ssh -o "BatchMode=yes" ${remote_server} "echo 2>&1" >/dev/null
RET=$?
if [ ${RET} -ne 0 ]; then
    error "SSH connection failed: ${RET}. Exiting..."
    remove_key_file
    exit 1
else
    success "SSH connection successful."
fi
