#!/usr/bin/env bash
set -euo pipefail

UTILS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${UTILS_DIR}/log.sh"
# shellcheck disable=SC1091
source "${UTILS_DIR}/exit_codes.sh"

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
        log::error "Failed to resolve domain $domain"
        return "$ERROR_DOMAIN_RESOLVE"
    fi
    echo "$all_ips" | grep -v '^$'
}

get_current_ip() {
    local ipv4
    local ipv6

    ipv4=$(curl -s --max-time 10 -4 http://ipecho.net/plain)
    ipv6=$(curl -s --max-time 10 -6 http://ipecho.net/plain)

    if [[ -z "$ipv4" && -z "$ipv6" ]]; then
        log::error "Failed to retrieve current IP"
        return "$ERROR_CURRENT_IP_FAIL"
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
        flow::exit_with_error "Invalid IP address format: $ip" "$ERROR_INVALID_SITE_IP"
    fi
}

validate_url() {
    local server_url="$1"

    if ! [[ $server_url =~ ^(http|https):// ]]; then
        flow::exit_with_error "Invalid URL format. Must start with http:// or https://" "$ERROR_USAGE"
    fi
}

# Returns "remote" if the domain resolves to the current IP address, or "local" otherwise
check_location() {
    # Get values from parsed arguments or environment
    local site_ip="${1:-$SITE_IP}"
    local server_url="${2:-$API_URL}"
    
    # Check required parameters 
    if [[ -z "$site_ip" ]]; then
        flow::exit_with_error "Required parameter: SITE_IP" "$ERROR_USAGE"
    fi
    if [[ -z "$server_url" ]]; then
        flow::exit_with_error "Required parameter: SERVER_URL" "$ERROR_USAGE"
    fi

    validate_ip "$site_ip"
    validate_url "$server_url"

    local domain

    domain="$(extract_domain "$server_url")"
    local extract_domain_status=$?
    if [ $extract_domain_status -ne 0 ]; then
        exit $extract_domain_status
    fi
    log::info "Domain: $domain"

    domain_ips="$(get_domain_ip "$domain")"
    local get_domain_ip_status=$?
    if [ $get_domain_ip_status -ne 0 ]; then
        exit $get_domain_ip_status
    fi
    log::info "Domain IPs:"
    # shellcheck disable=SC2001 # sed is appropriate here for multi-line prepending
    echo "$domain_ips" | sed 's/^/  /'

    if [[ -z "$domain_ips" ]]; then
        flow::exit_with_error "Failed to resolve domain $domain" "$ERROR_DOMAIN_RESOLVE"
    fi

    if ! echo "$domain_ips" | grep -q "^$site_ip$"; then
        flow::continue_with_error "SITE_IP does not point to the server associated with $domain. Check DNS settings." "$ERROR_INVALID_SITE_IP"
    fi

    current_ips="$(get_current_ip)"
    local get_current_ip_status=$?
    if [ $get_current_ip_status -ne 0 ]; then
        exit $get_current_ip_status
    fi
    log::info "Current IPs:"
    # shellcheck disable=SC2001 # sed is appropriate here for multi-line prepending
    echo "$current_ips" | sed 's/^/  /'

    if echo "$current_ips" | grep -q "^$site_ip$"; then
        echo "remote"
        return 0 # Return success status
    else
        echo "local"
        return 0 # Return success status
    fi
}

is_location_valid() {
    local server_location="$1"
    if [[ "$server_location" != "local" && "$server_location" != "remote" ]]; then
        return 1
    fi
    echo "Location is valid: $server_location"
    return 0
}

check_location_if_not_set() {
    # Get values from parsed arguments or environment
    local server_location="${1:-}"
    if [[ -z "$server_location" ]]; then
        server_location="$LOCATION"
    fi

    # Detect location if not set or invalid
    if ! is_location_valid "$server_location"; then
        log::info "Detecting server location..."
        LOCATION=$(check_location | tail -n 1)
        export LOCATION
        log::info "Detected server location: $LOCATION"
    fi
}
