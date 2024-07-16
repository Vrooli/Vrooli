#!/bin/bash
# This script connects to the remote server using the SSH key set up by keylessSsh.sh

HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
. "${HERE}/prettify.sh"

# Load variables from .env file
ENV_FILE="${HERE}/../.env-prod"
if [ -f "${ENV_FILE}" ]; then
    . "${ENV_FILE}"
else
    error "Could not find .env file at ${ENV_FILE}. Exiting..."
    exit 1
fi

if [ -z "${SITE_IP}" ]; then
    error "Could not find SITE_IP in .env. Exiting..."
    exit 1
fi

remote_server="root@${SITE_IP}"
key_name="id_rsa_${SITE_IP}"

if [ ! -f ~/.ssh/${key_name} ]; then
    error "SSH key not found. Please run keylessSsh.sh first to set up the connection."
    exit 1
fi

info "Connecting to ${remote_server}..."
ssh -i ~/.ssh/${key_name} ${remote_server}