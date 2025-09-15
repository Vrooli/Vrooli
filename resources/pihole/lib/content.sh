#!/usr/bin/env bash
# Pi-hole Content Management - Functions for managing DNS filtering content
set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source API library if not already sourced
if ! declare -f api_call &>/dev/null; then
    source "${SCRIPT_DIR}/api.sh"
fi

# Show blocking statistics
show_stats() {
    echo "Pi-hole Blocking Statistics"
    echo "==========================="
    
    local stats
    if ! stats=$(get_statistics); then
        echo "Error: Failed to retrieve statistics" >&2
        return 1
    fi
    
    # Parse and display key metrics
    echo "$stats" | jq -r '
        "Domains Blocked: \(.domains_being_blocked // 0)",
        "DNS Queries Today: \(.dns_queries_today // 0)",
        "Ads Blocked Today: \(.ads_blocked_today // 0)",
        "Ads Percentage: \(.ads_percentage_today // 0)%",
        "Unique Domains: \(.unique_domains // 0)",
        "Queries Forwarded: \(.queries_forwarded // 0)",
        "Queries Cached: \(.queries_cached // 0)",
        "Clients Ever Seen: \(.clients_ever_seen // 0)",
        "Unique Clients: \(.unique_clients // 0)",
        "Status: \(.status // "unknown")"
    '
}

# Update blocklists
update_blocklists() {
    echo "Updating Pi-hole blocklists..."
    update_gravity
    echo "Blocklists updated successfully"
}

# Manage blacklist
manage_blacklist() {
    local action="${1:-list}"
    shift || true
    
    case "$action" in
        add)
            local domain="$1"
            if [[ -z "$domain" ]]; then
                echo "Error: Domain required" >&2
                echo "Usage: vrooli resource pihole content blacklist add <domain>" >&2
                return 1
            fi
            echo "Adding $domain to blacklist..."
            add_to_blacklist "$domain"
            echo "Domain added to blacklist"
            ;;
        remove)
            local domain="$1"
            if [[ -z "$domain" ]]; then
                echo "Error: Domain required" >&2
                echo "Usage: vrooli resource pihole content blacklist remove <domain>" >&2
                return 1
            fi
            echo "Removing $domain from blacklist..."
            remove_from_blacklist "$domain"
            echo "Domain removed from blacklist"
            ;;
        list)
            echo "Current blacklist entries:"
            docker exec "${CONTAINER_NAME}" sqlite3 /etc/pihole/gravity.db \
                "SELECT domain FROM domainlist WHERE type = 1;" 2>/dev/null || \
                echo "No blacklist entries found"
            ;;
        *)
            echo "Error: Unknown blacklist action: $action" >&2
            echo "Valid actions: add, remove, list" >&2
            return 1
            ;;
    esac
}

# Manage whitelist
manage_whitelist() {
    local action="${1:-list}"
    shift || true
    
    case "$action" in
        add)
            local domain="$1"
            if [[ -z "$domain" ]]; then
                echo "Error: Domain required" >&2
                echo "Usage: vrooli resource pihole content whitelist add <domain>" >&2
                return 1
            fi
            echo "Adding $domain to whitelist..."
            add_to_whitelist "$domain"
            echo "Domain added to whitelist"
            ;;
        remove)
            local domain="$1"
            if [[ -z "$domain" ]]; then
                echo "Error: Domain required" >&2
                echo "Usage: vrooli resource pihole content whitelist remove <domain>" >&2
                return 1
            fi
            echo "Removing $domain from whitelist..."
            remove_from_whitelist "$domain"
            echo "Domain removed from whitelist"
            ;;
        list)
            echo "Current whitelist entries:"
            docker exec "${CONTAINER_NAME}" sqlite3 /etc/pihole/gravity.db \
                "SELECT domain FROM domainlist WHERE type = 0;" 2>/dev/null || \
                echo "No whitelist entries found"
            ;;
        *)
            echo "Error: Unknown whitelist action: $action" >&2
            echo "Valid actions: add, remove, list" >&2
            return 1
            ;;
    esac
}

# Query DNS logs
query_logs() {
    local limit="${1:-100}"
    
    echo "Recent DNS Queries (limit: $limit)"
    echo "=================================="
    
    local queries
    if ! queries=$(get_query_log "$limit"); then
        echo "Error: Failed to retrieve query log" >&2
        return 1
    fi
    
    # Display queries in readable format
    echo "$queries" | jq -r '.data[] | 
        "\(.time | strftime("%Y-%m-%d %H:%M:%S")) | \(.type) | \(.domain) | \(.client) | \(.status)"
    ' 2>/dev/null || echo "$queries" | jq '.data[:10]'
}

# Disable blocking temporarily
disable_blocking() {
    local duration="${1:-0}"
    
    if [[ "$duration" -gt 0 ]]; then
        echo "Disabling Pi-hole blocking for ${duration} seconds..."
    else
        echo "Disabling Pi-hole blocking permanently..."
    fi
    
    if disable_blocking "$duration"; then
        echo "Blocking disabled"
    else
        echo "Error: Failed to disable blocking" >&2
        return 1
    fi
}

# Enable blocking
enable_blocking() {
    echo "Enabling Pi-hole blocking..."
    
    if enable_blocking; then
        echo "Blocking enabled"
    else
        echo "Error: Failed to enable blocking" >&2
        return 1
    fi
}

# Show credentials
show_credentials() {
    echo "Pi-hole Credentials"
    echo "==================="
    
    # Get DNS port
    local dns_port="${PIHOLE_DNS_PORT}"
    if [[ -f "${PIHOLE_DATA_DIR}/.port_config" ]]; then
        source "${PIHOLE_DATA_DIR}/.port_config"
        dns_port="${DNS_PORT:-${PIHOLE_DNS_PORT}}"
    fi
    
    echo "Web Interface: http://localhost:${PIHOLE_API_PORT}/admin"
    echo "DNS Server: localhost:${dns_port}"
    
    if [[ -f "${PIHOLE_DATA_DIR}/.webpassword" ]]; then
        echo "Web Password: $(cat "${PIHOLE_DATA_DIR}/.webpassword")"
    else
        echo "Web Password: Not found"
    fi
    
    echo ""
    echo "Configuration:"
    echo "  DNS queries: dig @localhost -p ${dns_port} example.com"
    echo "  API endpoint: http://localhost:${PIHOLE_API_PORT}/admin/api.php"
    echo ""
    echo "Note: Use password for web interface login"
}

# Manage custom DNS records
manage_custom_dns() {
    local action="${1:-list}"
    shift || true
    
    case "$action" in
        add)
            local domain="$1"
            local ip="$2"
            if [[ -z "$domain" || -z "$ip" ]]; then
                echo "Error: Both domain and IP required" >&2
                echo "Usage: vrooli resource pihole content dns add <domain> <ip>" >&2
                return 1
            fi
            add_custom_dns "$domain" "$ip"
            ;;
        remove)
            local domain="$1"
            if [[ -z "$domain" ]]; then
                echo "Error: Domain required" >&2
                echo "Usage: vrooli resource pihole content dns remove <domain>" >&2
                return 1
            fi
            remove_custom_dns "$domain"
            ;;
        list)
            get_custom_dns
            ;;
        *)
            echo "Error: Unknown DNS action: $action" >&2
            echo "Valid actions: add, remove, list" >&2
            return 1
            ;;
    esac
}

# Export functions
export -f show_stats
export -f update_blocklists
export -f manage_blacklist
export -f manage_whitelist
export -f query_logs
export -f disable_blocking
export -f enable_blocking
export -f show_credentials
export -f manage_custom_dns