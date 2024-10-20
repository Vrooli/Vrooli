#!/bin/bash

# Exit codes
export ERROR_USAGE=64
export ERROR_DOMAIN_RESOLVE=65
export ERROR_INVALID_SITE_IP=66
export ERROR_CURRENT_IP_FAIL=67
export ERROR_SITE_IP_MISMATCH=68

HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "${HERE}/utils.sh"

usage() {
    cat <<EOF
Usage: $(basename "$0") SITE_IP SERVER_URL

Checks if the domain resolves to this server's IP.

Arguments:
  SITE_IP     IP address (IPv4 or IPv6) of the server associated with the domain.
  SERVER_URL  URL of the server.

Exit Codes:
  0                     Success
  $ERROR_USAGE              Command line usage error
  $ERROR_DOMAIN_RESOLVE     Failed to resolve domain
  $ERROR_INVALID_SITE_IP    SITE_IP does not match domain IP
  $ERROR_CURRENT_IP_FAIL    Failed to retrieve current IP
  $ERROR_SITE_IP_MISMATCH   SITE_IP must be valid on production server

EOF
}

extract_domain() {
    local server_url="$1"
    local domain="${server_url#*://}" # Remove protocol and '://', if present
    domain="${domain%%/*}"            # Remove everything after the first '/'
    domain="${domain%%:*}"            # Remove port, if present
    echo "$domain"
}

get_domain_ip() {
    local domain="$1"
    local ipv4
    local ipv6

    ipv4="$(dig +short +time=5 +tries=1 A "$domain")"
    ipv6="$(dig +short +time=5 +tries=1 AAAA "$domain")"

    local all_ips="$ipv4"$'\n'"$ipv6"
    # Make sure output is not empty/whitespace
    if [ -z "$ipv4" ] && [ -z "$ipv6" ]; then
        error "Failed to resolve domain $domain"
        return $ERROR_DOMAIN_RESOLVE
    fi
    echo "$all_ips" | grep -v '^$'
}

get_current_ip() {
    local ipv4
    local ipv6

    ipv4=$(curl -s --max-time 10 -4 http://ipecho.net/plain)
    ipv6=$(curl -s --max-time 10 -6 http://ipecho.net/plain)

    if [[ -z "$ipv4" && -z "$ipv6" ]]; then
        error "Failed to retrieve current IP"
        return $ERROR_CURRENT_IP_FAIL
    fi

    echo "$ipv4"$'\n'"$ipv6" | grep -v '^$'
    return 0
}

validate_ip() {
    local ip="$1"
    if [[ $ip =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        return 0 # Valid IPv4
    elif [[ $ip =~ ^([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}$|^([0-9a-fA-F]{1,4}:){1,7}:|^([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}$|^([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}$|^([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}$|^([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}$|^([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}$|^[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})$|^:((:[0-9a-fA-F]{1,4}){1,7}|:)$ ]]; then
        return 0 # Valid IPv6
    else
        exit_with_error "Invalid IP address format: $ip" $ERROR_INVALID_SITE_IP
    fi
}

validate_url() {
    local server_url="$1"

    if ! [[ $server_url =~ ^(http|https):// ]]; then
        exit_with_error "Invalid URL format. Must start with http:// or https://" $ERROR_USAGE
    fi
}

main() {
    if [ "$#" -ne 2 ]; then
        error "Provided $# arguments. Expected 2"
        usage
        exit $ERROR_USAGE
    fi

    local site_ip="$1"
    local server_url="$2"
    validate_ip "$site_ip"
    validate_url "$server_url"

    local domain
    local domain_ip
    local current_ip

    domain="$(extract_domain "$server_url")"
    local extract_domain_status=$?
    if [ $extract_domain_status -ne 0 ]; then
        exit $extract_domain_status
    fi
    info "Domain: $domain"

    domain_ips="$(get_domain_ip "$domain")"
    local get_domain_ip_status=$?
    if [ $get_domain_ip_status -ne 0 ]; then
        exit $get_domain_ip_status
    fi
    info "Domain IPs:"
    echo "$domain_ips" | sed 's/^/  /'

    if [[ -z "$domain_ips" ]]; then
        exit_with_error "Failed to resolve domain $domain" $ERROR_DOMAIN_RESOLVE
    fi

    if ! echo "$domain_ips" | grep -q "^$site_ip$"; then
        exit_with_error "SITE_IP does not point to the server associated with $domain" $ERROR_INVALID_SITE_IP
    fi

    current_ips="$(get_current_ip)"
    local get_current_ip_status=$?
    if [ $get_current_ip_status -ne 0 ]; then
        exit $get_current_ip_status
    fi
    info "Current IPs:"
    echo "$current_ips" | sed 's/^/  /'

    if echo "$current_ips" | grep -q "^$site_ip$"; then
        echo "remote"
        exit 0
    else
        echo "local"
        exit 0
    fi
}

run_if_executed main "$@"
