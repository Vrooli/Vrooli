#!/bin/bash
# Checks if the domain resolves to this server's IP.
# Important for setting CORS.
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
. "${HERE}/prettify.sh"

# Check if sufficient arguments are provided
if [ "$#" -ne 2 ]; then
    echo "Usage: $0 SITE_IP SERVER_URL"
    exit 1
fi

# Assign arguments to variables
SITE_IP=$1    # IP of the server associated with the domain
SERVER_URL=$2 # URL of the server

# Extract the domain from SERVER_URL
domain=$(echo $SERVER_URL | awk -F[/:] '{print $4}')
info "Domain: $domain"

# Find what IP the domain resolves to
domain_ip=$(dig +short $domain)
info "Domain IP: $domain_ip"

# Make sure the domain resolves to the server's IP
if [[ "$domain_ip" != "$SITE_IP" ]]; then
    VALID_IP=false
    error "SITE_IP does not point to the server associated with $domain"
fi

# Find what IP we're currently using
current_ip=$(curl -s http://ipecho.net/plain)
info "Current IP: $current_ip"

# If the current IP is the same as the domain IP ,
# then we're running this script on a test/production server
if [[ "$current_ip" == "$SITE_IP" ]]; then
    # SITE_IP must be valid
    if [[ "$VALID_IP" == false ]]; then
        error "SITE_IP must be valid when running this script on a test/production server"
        exit 1
    fi
    echo "dns"
else
    echo "local"
fi
