#!/usr/bin/env bash
set -euo pipefail

# Source var.sh first with relative path
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../utils/var.sh"

# Now source everything else using var_ variables
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_EXIT_CODES_FILE}"
# shellcheck disable=SC1091
source "${var_FLOW_FILE}"

domainCheck::extract_domain() {
    local server_url="$1"
    local domain="${server_url#*://}" # Remove protocol and '://', if present
    domain="${domain#*@}"             # Remove user credentials (user:pass@), if present
    domain="${domain%%/*}"            # Remove everything after the first '/'
    domain="${domain%%:*}"            # Remove port, if present
    echo "$domain"
}

domainCheck::get_domain_ip() {
    local domain="$1"
    local ipv4
    local ipv6

    ipv4="$(dig +short +time=5 +tries=1 A "$domain")"
    ipv6="$(dig +short +time=5 +tries=1 AAAA "$domain")"

    local all_ips="$ipv4"$'\n'"$ipv6"
    # Make sure output is not empty/whitespace
    if [ -z "$ipv4" ] && [ -z "$ipv6" ]; then
        log::error "Failed to resolve domain $domain"
        return "$EXIT_NETWORK_ERROR"
    fi
    echo "$all_ips" | grep -v '^$'
}

domainCheck::get_current_ip() {
    local ipv4
    local ipv6

    ipv4=$(curl -s --max-time 10 -4 http://ipecho.net/plain)
    ipv6=$(curl -s --max-time 10 -6 http://ipecho.net/plain)

    if [[ -z "$ipv4" && -z "$ipv6" ]]; then
        log::error "Failed to retrieve current IP"
        return "$EXIT_NETWORK_ERROR"
    fi

    echo "$ipv4"$'\n'"$ipv6" | grep -v '^$'
    return 0
}

domainCheck::validate_ip() {
    local ip="$1"
    
    # Handle localhost as a special case
    if [[ "$ip" == "localhost" ]]; then
        return 0 # Valid localhost
    fi
    
    if [[ $ip =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        return 0 # Valid IPv4
    elif [[ $ip =~ ^([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}$|^([0-9a-fA-F]{1,4}:){1,7}:|^([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}$|^([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}$|^([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}$|^([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}$|^([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}$|^[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})$|^:((:[0-9a-fA-F]{1,4}){1,7}|:)$ ]]; then
        return 0 # Valid IPv6
    else
        flow::exit_with_error "Invalid IP address format: $ip" "$EXIT_INVALID_ARGUMENT"
    fi
}

domainCheck::validate_url() {
    local server_url="$1"

    if ! [[ $server_url =~ ^(http|https):// ]]; then
        flow::exit_with_error "Invalid URL format. Must start with http:// or https://" "$EXIT_INVALID_ARGUMENT"
    fi
}

# Returns "remote" if the domain resolves to the current IP address, or "local" otherwise
domainCheck::check_location() {
    # Get values from parsed arguments or environment
    local site_ip="${1:-${SITE_IP:-}}"
    local server_url="${2:-${API_URL:-}}"
    
    # Check required parameters 
    if [[ -z "$site_ip" ]]; then
        flow::exit_with_error "Required parameter: SITE_IP" "$EXIT_INVALID_ARGUMENT"
    fi
    if [[ -z "$server_url" ]]; then
        flow::exit_with_error "Required parameter: SERVER_URL" "$EXIT_INVALID_ARGUMENT"
    fi

    domainCheck::validate_ip "$site_ip"
    domainCheck::validate_url "$server_url"

    # Normalize localhost to 127.0.0.1 for comparisons
    local normalized_site_ip="$site_ip"
    if [[ "$site_ip" == "localhost" ]]; then
        normalized_site_ip="127.0.0.1"
    fi

    local domain

    domain="$(domainCheck::extract_domain "$server_url")"
    local extract_domain_status=$?
    if [ $extract_domain_status -ne 0 ]; then
        exit $extract_domain_status
    fi
    log::info "Domain: $domain"

    local domain_ips
    domain_ips="$(domainCheck::get_domain_ip "$domain" 2>&1)"
    local get_domain_ip_status=$?
    if [ $get_domain_ip_status -ne 0 ]; then
        # Re-log the error so it's visible in the output
        if [[ "$domain_ips" == *"ERROR"* ]]; then
            echo "$domain_ips" >&2
        else
            log::error "Failed to resolve domain $domain"
        fi
        exit $get_domain_ip_status
    fi
    log::info "Domain IPs:"
    # shellcheck disable=SC2001 # sed is appropriate here for multi-line prepending
    echo "$domain_ips" | sed 's/^/  /'

    if [[ -z "$domain_ips" ]]; then
        flow::exit_with_error "Failed to resolve domain $domain" "$EXIT_NETWORK_ERROR"
    fi

    if ! echo "$domain_ips" | grep -q "^$normalized_site_ip$"; then
        flow::continue_with_error "SITE_IP does not point to the server associated with $domain. Check DNS settings." "$EXIT_CONFIGURATION_ERROR"
        local error_status=$?
    fi

    local current_ips
    current_ips="$(domainCheck::get_current_ip)"
    local get_current_ip_status=$?
    if [ $get_current_ip_status -ne 0 ]; then
        exit $get_current_ip_status
    fi
    log::info "Current IPs:"
    # shellcheck disable=SC2001 # sed is appropriate here for multi-line prepending
    echo "$current_ips" | sed 's/^/  /'

    if echo "$current_ips" | grep -q "^$normalized_site_ip$"; then
        echo "remote"
    else
        echo "local"
    fi
    
    # Return error status if SITE_IP didn't match domain
    if [[ -n "${error_status:-}" ]]; then
        return "$error_status"
    fi
    return 0
}

domainCheck::is_location_valid() {
    local server_location="$1"
    if [[ "$server_location" != "local" && "$server_location" != "remote" ]]; then
        return 1
    fi
    echo "Location is valid: $server_location"
    return 0
}

domainCheck::check_location_if_not_set() {
    # Get values from parsed arguments or environment
    local server_location="${1:-}"
    if [[ -z "$server_location" ]]; then
        server_location="$LOCATION"
    fi

    # Detect location if not set or invalid
    if ! domainCheck::is_location_valid "$server_location"; then
        log::info "Detecting server location..."
        LOCATION=$(domainCheck::check_location | tail -n 1)
        export LOCATION
        log::info "Detected server location: $LOCATION"
    fi
}