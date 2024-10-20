#!/bin/bash
# This script generates a public/private key pair for JWT signing

HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
. "${HERE}/utils.sh"

# Go to root directory of the project
cd "${HERE}/.."

PUB_KEY_NAME="jwt_pub.pem"
PRIV_KEY_NAME="jwt_priv.pem"

# If keys already exist, inform user and soft exit
if [ -f "${PUB_KEY_NAME}" ] || [ -f "${PRIV_KEY_NAME}" ]; then
    # If one key exists, but not the other, show error and exit
    if [ ! -f "${PUB_KEY_NAME}" ] || [ ! -f "${PRIV_KEY_NAME}" ]; then
        error "One JWT key exists, but not the other. Delete them both and try again."
        exit 1
    fi
    info "JWT keys already exist. Delete them first if you want to generate new ones."
    return 0
fi

# Use openssl to generate private key and public key
info "Generating new JWT keys..."
openssl genpkey -algorithm RSA -out "${PRIV_KEY_NAME}" -pkeyopt rsa_keygen_bits:2048
openssl rsa -pubout -in "${PRIV_KEY_NAME}" -out "${PUB_KEY_NAME}"
info "JWT keys generated and saved to ${PRIV_KEY_NAME} and ${PUB_KEY_NAME} in the root directory of the project."
