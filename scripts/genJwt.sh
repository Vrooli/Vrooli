#!/bin/bash
# This script generates a public/private key pair for JWT signing

HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "${HERE}/prettify.sh"

# Generate a new RSA key pair using ssh-keygen with 4096 bits and PEM format, output private key to jwtRS256.key
warning "Don't add a passphrase when prompted by ssh-keygen."
ssh-keygen -t rsa -b 4096 -m PEM -f jwtRS256.key

# Extract the public key from the private key using openssl, output public key to jwtRS256.key.pub
openssl rsa -in jwtRS256.key -pubout -outform PEM -out jwtRS256.key.pub

# Display the contents of the private key file jwtRS256.key
cat jwtRS256.key

# Display the contents of the public key file jwtRS256.key.pub
cat jwtRS256.key.pub
