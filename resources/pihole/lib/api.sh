#!/usr/bin/env bash
# Pi-hole API Library - API interaction and authentication functions
set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Source core library if not already sourced
if [[ -z "${CONTAINER_NAME:-}" ]]; then
    source "${SCRIPT_DIR}/core.sh"
fi

# Get API authentication token
get_api_token() {
    # For new API v2, we use the pihole CLI command directly
    # which handles authentication internally
    return 0
}

# Make authenticated API call
api_call() {
    local endpoint="$1"
    shift
    
    # Use pihole CLI API command which handles auth internally
    docker exec "${CONTAINER_NAME}" pihole api "${endpoint}"
}

# Get blocking statistics
get_statistics() {
    local stats
    if ! stats=$(api_call "stats/summary"); then
        echo "Error: Failed to get statistics" >&2
        return 1
    fi
    
    echo "$stats"
}

# Get query log
get_query_log() {
    local limit="${1:-100}"
    local queries
    
    if ! queries=$(api_call "queries" "limit=${limit}"); then
        echo "Error: Failed to get query log" >&2
        return 1
    fi
    
    echo "$queries"
}

# Enable blocking
enable_blocking() {
    docker exec "${CONTAINER_NAME}" pihole enable
}

# Disable blocking
disable_blocking() {
    local duration="${1:-0}"  # 0 means permanent
    
    if [[ "$duration" -gt 0 ]]; then
        docker exec "${CONTAINER_NAME}" pihole disable "${duration}s"
    else
        docker exec "${CONTAINER_NAME}" pihole disable
    fi
}

# Add domain to blacklist
add_to_blacklist() {
    local domain="$1"
    
    if [[ -z "$domain" ]]; then
        echo "Error: Domain required" >&2
        return 1
    fi
    
    docker exec "${CONTAINER_NAME}" pihole -b "${domain}"
}

# Remove domain from blacklist
remove_from_blacklist() {
    local domain="$1"
    
    if [[ -z "$domain" ]]; then
        echo "Error: Domain required" >&2
        return 1
    fi
    
    docker exec "${CONTAINER_NAME}" pihole -b -d "${domain}"
}

# Add domain to whitelist
add_to_whitelist() {
    local domain="$1"
    
    if [[ -z "$domain" ]]; then
        echo "Error: Domain required" >&2
        return 1
    fi
    
    docker exec "${CONTAINER_NAME}" pihole -w "${domain}"
}

# Remove domain from whitelist
remove_from_whitelist() {
    local domain="$1"
    
    if [[ -z "$domain" ]]; then
        echo "Error: Domain required" >&2
        return 1
    fi
    
    docker exec "${CONTAINER_NAME}" pihole -w -d "${domain}"
}

# Get top items (domains, clients, etc.)
get_top_items() {
    local count="${1:-10}"
    local top_data
    
    if ! top_data=$(api_call "stats/top_domains" "limit=${count}"); then
        echo "Error: Failed to get top items" >&2
        return 1
    fi
    
    echo "$top_data"
}

# Get recent blocked domains
get_recent_blocked() {
    local count="${1:-10}"
    local blocked
    
    if ! blocked=$(api_call "queries" "limit=${count}&blocked=true"); then
        echo "Error: Failed to get recent blocked domains" >&2
        return 1
    fi
    
    echo "$blocked"
}

# Update gravity (blocklist database)
update_gravity() {
    echo "Updating Pi-hole gravity database..."
    docker exec "${CONTAINER_NAME}" pihole -g
}

# Get custom DNS records
get_custom_dns() {
    local dns_file="/etc/pihole/custom.list"
    
    echo "Custom DNS records:"
    docker exec "${CONTAINER_NAME}" cat "$dns_file" 2>/dev/null || echo "No custom DNS records found"
}

# Add custom DNS record
add_custom_dns() {
    local domain="$1"
    local ip="$2"
    
    if [[ -z "$domain" || -z "$ip" ]]; then
        echo "Error: Both domain and IP address required" >&2
        echo "Usage: add_custom_dns example.com 192.168.1.100" >&2
        return 1
    fi
    
    # Validate IP address format
    if ! echo "$ip" | grep -qE '^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$'; then
        echo "Error: Invalid IP address format" >&2
        return 1
    fi
    
    echo "Adding custom DNS record: $domain -> $ip"
    docker exec "${CONTAINER_NAME}" bash -c "echo '$ip $domain' >> /etc/pihole/custom.list"
    
    # Reload DNS to apply changes
    docker exec "${CONTAINER_NAME}" pihole reloaddns
}

# Remove custom DNS record
remove_custom_dns() {
    local domain="$1"
    
    if [[ -z "$domain" ]]; then
        echo "Error: Domain required" >&2
        return 1
    fi
    
    echo "Removing custom DNS record for: $domain"
    docker exec "${CONTAINER_NAME}" sed -i "/ ${domain}$/d" /etc/pihole/custom.list
    
    # Reload DNS to apply changes
    docker exec "${CONTAINER_NAME}" pihole reloaddns
}

# Get forward destinations (upstream DNS servers)
get_forward_destinations() {
    local destinations
    
    if ! destinations=$(api_call "stats/upstreams"); then
        echo "Error: Failed to get forward destinations" >&2
        return 1
    fi
    
    echo "$destinations"
}

# Get query types distribution
get_query_types() {
    local types
    
    if ! types=$(api_call "stats/summary"); then
        echo "Error: Failed to get query types" >&2
        return 1
    fi
    
    echo "$types" | jq '.queries.types'
}

# Flush DNS cache
flush_cache() {
    echo "Flushing Pi-hole DNS cache..."
    docker exec "${CONTAINER_NAME}" pihole reloaddns
}

# Export functions for use by other scripts
export -f get_api_token
export -f api_call
export -f get_statistics
export -f get_query_log
export -f enable_blocking
export -f disable_blocking
export -f add_to_blacklist
export -f remove_from_blacklist
export -f add_to_whitelist
export -f remove_from_whitelist
export -f get_top_items
export -f get_recent_blocked
export -f update_gravity
export -f get_custom_dns
export -f add_custom_dns
export -f remove_custom_dns
export -f get_forward_destinations
export -f get_query_types
export -f flush_cache