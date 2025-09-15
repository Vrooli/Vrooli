#!/usr/bin/env bash
# Pi-hole DHCP Library - DHCP server management functions
set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Source core library if not already sourced
if [[ -z "${CONTAINER_NAME:-}" ]]; then
    source "${SCRIPT_DIR}/core.sh"
fi

# Enable DHCP server
enable_dhcp() {
    local start_ip="${1:-192.168.1.100}"
    local end_ip="${2:-192.168.1.200}"
    local gateway="${3:-192.168.1.1}"
    local lease_time="${4:-24}"  # hours
    
    echo "Enabling DHCP server..."
    
    # Configure DHCP settings
    docker exec "${CONTAINER_NAME}" bash -c "cat > /etc/dnsmasq.d/02-pihole-dhcp.conf << EOF
dhcp-range=${start_ip},${end_ip},${lease_time}h
dhcp-option=option:router,${gateway}
dhcp-option=option:dns-server,0.0.0.0
dhcp-leasefile=/etc/pihole/dhcp.leases
dhcp-rapid-commit
dhcp-authoritative
EOF"
    
    # Restart dnsmasq to apply changes
    docker exec "${CONTAINER_NAME}" pihole reloaddns
    
    echo "DHCP server enabled with range ${start_ip} - ${end_ip}"
    return 0
}

# Disable DHCP server
disable_dhcp() {
    echo "Disabling DHCP server..."
    
    # Remove DHCP configuration
    docker exec "${CONTAINER_NAME}" rm -f /etc/dnsmasq.d/02-pihole-dhcp.conf
    
    # Restart dnsmasq to apply changes
    docker exec "${CONTAINER_NAME}" pihole reloaddns
    
    echo "DHCP server disabled"
    return 0
}

# Get DHCP status
get_dhcp_status() {
    if docker exec "${CONTAINER_NAME}" test -f /etc/dnsmasq.d/02-pihole-dhcp.conf; then
        echo "DHCP server is ENABLED"
        
        # Show configuration
        echo ""
        echo "Configuration:"
        docker exec "${CONTAINER_NAME}" cat /etc/dnsmasq.d/02-pihole-dhcp.conf | grep -E "^dhcp-"
    else
        echo "DHCP server is DISABLED"
    fi
}

# Get DHCP leases
get_dhcp_leases() {
    local leases_file="/etc/pihole/dhcp.leases"
    
    echo "Active DHCP leases:"
    echo "==================="
    
    if docker exec "${CONTAINER_NAME}" test -f "$leases_file"; then
        # Parse leases file (format: timestamp mac ip hostname client-id)
        docker exec "${CONTAINER_NAME}" bash -c "
            if [[ -s $leases_file ]]; then
                echo 'IP Address      MAC Address        Hostname           Expires'
                echo '-----------------------------------------------------------'
                while IFS=' ' read -r timestamp mac ip hostname clientid; do
                    if [[ -n \"\$timestamp\" ]]; then
                        expire_date=\$(date -d \"@\$timestamp\" '+%Y-%m-%d %H:%M')
                        printf '%-15s %-17s %-18s %s\n' \"\$ip\" \"\$mac\" \"\${hostname:-unknown}\" \"\$expire_date\"
                    fi
                done < $leases_file
            else
                echo 'No active leases'
            fi
        "
    else
        echo "No active leases"
    fi
}

# Add static DHCP reservation
add_dhcp_reservation() {
    local mac="$1"
    local ip="$2"
    local hostname="${3:-}"
    
    if [[ -z "$mac" || -z "$ip" ]]; then
        echo "Error: MAC address and IP address required" >&2
        echo "Usage: add_dhcp_reservation <mac> <ip> [hostname]" >&2
        return 1
    fi
    
    # Validate MAC address format
    if ! echo "$mac" | grep -qE '^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$'; then
        echo "Error: Invalid MAC address format" >&2
        return 1
    fi
    
    # Validate IP address format
    if ! echo "$ip" | grep -qE '^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$'; then
        echo "Error: Invalid IP address format" >&2
        return 1
    fi
    
    echo "Adding DHCP reservation: $mac -> $ip"
    
    # Add reservation to configuration
    local reservation="dhcp-host=${mac},${ip}"
    if [[ -n "$hostname" ]]; then
        reservation="${reservation},${hostname}"
    fi
    
    docker exec "${CONTAINER_NAME}" bash -c "echo '${reservation}' >> /etc/dnsmasq.d/04-pihole-static-dhcp.conf"
    
    # Restart dnsmasq to apply changes
    docker exec "${CONTAINER_NAME}" pihole reloaddns
    
    echo "DHCP reservation added"
    return 0
}

# Remove static DHCP reservation
remove_dhcp_reservation() {
    local mac="$1"
    
    if [[ -z "$mac" ]]; then
        echo "Error: MAC address required" >&2
        return 1
    fi
    
    echo "Removing DHCP reservation for: $mac"
    
    # Remove reservation from configuration
    docker exec "${CONTAINER_NAME}" sed -i "/dhcp-host=${mac}/d" /etc/dnsmasq.d/04-pihole-static-dhcp.conf 2>/dev/null || true
    
    # Restart dnsmasq to apply changes
    docker exec "${CONTAINER_NAME}" pihole reloaddns
    
    echo "DHCP reservation removed"
    return 0
}

# List DHCP reservations
list_dhcp_reservations() {
    echo "Static DHCP reservations:"
    echo "========================"
    
    if docker exec "${CONTAINER_NAME}" test -f /etc/dnsmasq.d/04-pihole-static-dhcp.conf; then
        docker exec "${CONTAINER_NAME}" grep "^dhcp-host=" /etc/dnsmasq.d/04-pihole-static-dhcp.conf 2>/dev/null || echo "No reservations configured"
    else
        echo "No reservations configured"
    fi
}

# Configure DHCP options
configure_dhcp_options() {
    local option="$1"
    local value="$2"
    
    if [[ -z "$option" || -z "$value" ]]; then
        echo "Error: Option and value required" >&2
        echo "Usage: configure_dhcp_options <option> <value>" >&2
        echo "Example: configure_dhcp_options ntp-server 192.168.1.1" >&2
        return 1
    fi
    
    echo "Setting DHCP option: ${option} = ${value}"
    
    # Add option to configuration
    docker exec "${CONTAINER_NAME}" bash -c "echo 'dhcp-option=option:${option},${value}' >> /etc/dnsmasq.d/03-pihole-dhcp-options.conf"
    
    # Restart dnsmasq to apply changes
    docker exec "${CONTAINER_NAME}" pihole reloaddns
    
    echo "DHCP option configured"
    return 0
}

# Export functions for use by other scripts
export -f enable_dhcp
export -f disable_dhcp
export -f get_dhcp_status
export -f get_dhcp_leases
export -f add_dhcp_reservation
export -f remove_dhcp_reservation
export -f list_dhcp_reservations
export -f configure_dhcp_options