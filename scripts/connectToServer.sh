#!/bin/bash
# This script connects to the remote server using the SSH key set up by keylessSsh.sh
# Example usage:
#  ./scripts/connectToServer.sh              # Connect using IP from .env-prod
#  ./scripts/connectToServer.sh 123.456.789.012  # Connect to specific IP
#  ./scripts/connectToServer.sh -e /path/to/.env # Use custom .env file

HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
. "${HERE}/utils.sh"

# Read arguments
ENV_FILE="${HERE}/../.env-prod"
while getopts "he:" opt; do
    case $opt in
    h)
        echo "Usage: $0 [-h] [-e ENV_FILE] [SITE_IP]"
        echo "  -h --help: Show this help message"
        echo "  -e --env-file: .env file location (e.g. \"/root/my-folder/.env\")"
        echo "  SITE_IP: Optional IP address to connect to (overrides .env)"
        exit 0
        ;;
    e)
        ENV_FILE=$OPTARG
        ;;
    \?)
        echo "Invalid option: -$OPTARG" >&2
        exit 1
        ;;
    :)
        echo "Option -$OPTARG requires an argument." >&2
        exit 1
        ;;
    esac
done
shift $((OPTIND-1))

# Check for IP address as positional argument
if [ $# -ge 1 ]; then
    SITE_IP=$1
    info "Using provided IP address: ${SITE_IP}"
else
    # Load variables from .env file
    if [ -f "${ENV_FILE}" ]; then
        info "Loading variables from ${ENV_FILE}..."
        . "${ENV_FILE}"
    else
        error "Could not find .env file at ${ENV_FILE}. Exiting..."
        exit 1
    fi

    # Check if SITE_IP was loaded from .env
    if [ -z "${SITE_IP}" ]; then
        error "Could not find SITE_IP in .env file. Please provide IP as argument or in .env file."
        exit 1
    fi
fi

remote_server="root@${SITE_IP}"
key_name="id_rsa_${SITE_IP}"

if [ ! -f ~/.ssh/${key_name} ]; then
    error "SSH key not found. Please run keylessSsh.sh first to set up the connection."
    exit 1
fi

info "Connecting to ${remote_server}..."
ssh -i ~/.ssh/${key_name} ${remote_server}
