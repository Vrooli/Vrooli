#!/usr/bin/env bash
# Pi-hole Group Management - Simplified configuration-based management
set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Source core library
source "${SCRIPT_DIR}/core.sh"

# Groups configuration file
GROUPS_CONFIG="${PIHOLE_DATA_DIR}/groups.conf"
CLIENT_GROUPS="${PIHOLE_DATA_DIR}/client-groups.conf"

# Initialize group configuration
init_groups() {
    mkdir -p "${PIHOLE_DATA_DIR}"
    
    if [[ ! -f "${GROUPS_CONFIG}" ]]; then
        cat > "${GROUPS_CONFIG}" << 'EOF'
# Pi-hole Groups Configuration
# Format: group_name|description|enabled
Default|Default group for all clients|1
EOF
    fi
    
    if [[ ! -f "${CLIENT_GROUPS}" ]]; then
        cat > "${CLIENT_GROUPS}" << 'EOF'
# Client to Group Mapping
# Format: client_ip|group_name|comment
EOF
    fi
    
    echo "Groups configuration initialized"
}

# Create a new group
create_group() {
    local group_name="$1"
    local description="${2:-}"
    
    if [[ -z "$group_name" ]]; then
        echo "Error: Group name required" >&2
        return 1
    fi
    
    init_groups
    
    # Check if group exists
    if grep -q "^${group_name}|" "${GROUPS_CONFIG}" 2>/dev/null; then
        echo "Error: Group '${group_name}' already exists" >&2
        return 1
    fi
    
    # Add to configuration
    echo "${group_name}|${description}|1" >> "${GROUPS_CONFIG}"
    
    # Create corresponding blocklist file
    touch "${PIHOLE_DATA_DIR}/group-${group_name}.list"
    
    echo "Group '${group_name}' created"
    echo "Blocklist file: ${PIHOLE_DATA_DIR}/group-${group_name}.list"
}

# Delete a group
delete_group() {
    local group_name="$1"
    
    if [[ -z "$group_name" ]]; then
        echo "Error: Group name required" >&2
        return 1
    fi
    
    if [[ "$group_name" == "Default" ]]; then
        echo "Error: Cannot delete Default group" >&2
        return 1
    fi
    
    init_groups
    
    # Remove from groups config
    sed -i "/^${group_name}|/d" "${GROUPS_CONFIG}"
    
    # Remove from client mappings
    sed -i "/|${group_name}|/d" "${CLIENT_GROUPS}"
    
    # Remove blocklist file
    rm -f "${PIHOLE_DATA_DIR}/group-${group_name}.list"
    
    echo "Group '${group_name}' deleted"
}

# List all groups
list_groups() {
    init_groups
    
    echo "Available groups:"
    echo "================"
    echo "Name | Description | Status"
    echo "------|-------------|--------"
    
    while IFS='|' read -r name desc enabled; do
        if [[ "$name" =~ ^#.*$ ]] || [[ -z "$name" ]]; then
            continue
        fi
        local status="disabled"
        if [[ "$enabled" == "1" ]]; then
            status="enabled"
        fi
        printf "%-20s | %-40s | %s\n" "$name" "$desc" "$status"
    done < "${GROUPS_CONFIG}"
}

# Add client to group
add_client_to_group() {
    local client_ip="$1"
    local group_name="$2"
    local comment="${3:-}"
    
    if [[ -z "$client_ip" ]] || [[ -z "$group_name" ]]; then
        echo "Error: Client IP and group name required" >&2
        return 1
    fi
    
    init_groups
    
    # Check if group exists
    if ! grep -q "^${group_name}|" "${GROUPS_CONFIG}" 2>/dev/null; then
        echo "Error: Group '${group_name}' not found" >&2
        return 1
    fi
    
    # Check if mapping exists
    if grep -q "^${client_ip}|" "${CLIENT_GROUPS}" 2>/dev/null; then
        # Update existing mapping
        sed -i "/^${client_ip}|/d" "${CLIENT_GROUPS}"
    fi
    
    # Add new mapping
    echo "${client_ip}|${group_name}|${comment}" >> "${CLIENT_GROUPS}"
    
    # Apply group-specific blocklist to client
    local group_list="${PIHOLE_DATA_DIR}/group-${group_name}.list"
    if [[ -f "$group_list" ]]; then
        # Add custom DNS entries for this client
        while read -r domain; do
            if [[ -n "$domain" ]] && ! [[ "$domain" =~ ^#.*$ ]]; then
                echo "${client_ip} ${domain}" >> "${PIHOLE_DATA_DIR}/etc-pihole/custom.list"
            fi
        done < "$group_list"
        
        # Copy to container
        docker cp "${PIHOLE_DATA_DIR}/etc-pihole/custom.list" "${CONTAINER_NAME}:/etc/pihole/custom.list"
        docker exec "${CONTAINER_NAME}" pihole restartdns
    fi
    
    echo "Client ${client_ip} added to group ${group_name}"
}

# Remove client from group
remove_client_from_group() {
    local client_ip="$1"
    
    if [[ -z "$client_ip" ]]; then
        echo "Error: Client IP required" >&2
        return 1
    fi
    
    init_groups
    
    # Remove mapping
    sed -i "/^${client_ip}|/d" "${CLIENT_GROUPS}"
    
    # Remove custom DNS entries for this client
    sed -i "/^${client_ip} /d" "${PIHOLE_DATA_DIR}/etc-pihole/custom.list" 2>/dev/null || true
    
    # Update container
    if [[ -f "${PIHOLE_DATA_DIR}/etc-pihole/custom.list" ]]; then
        docker cp "${PIHOLE_DATA_DIR}/etc-pihole/custom.list" "${CONTAINER_NAME}:/etc/pihole/custom.list"
        docker exec "${CONTAINER_NAME}" pihole restartdns
    fi
    
    echo "Client ${client_ip} removed from groups"
}

# List clients in a group
list_group_clients() {
    local group_name="$1"
    
    if [[ -z "$group_name" ]]; then
        echo "Error: Group name required" >&2
        return 1
    fi
    
    init_groups
    
    echo "Clients in group '${group_name}':"
    echo "================================"
    echo "IP Address | Comment"
    echo "-----------|--------"
    
    while IFS='|' read -r ip group comment; do
        if [[ "$group" == "$group_name" ]]; then
            printf "%-15s | %s\n" "$ip" "$comment"
        fi
    done < "${CLIENT_GROUPS}"
}

# Add domain to group blocklist
add_to_group_blocklist() {
    local group_name="$1"
    local domain="$2"
    
    if [[ -z "$group_name" ]] || [[ -z "$domain" ]]; then
        echo "Error: Group name and domain required" >&2
        return 1
    fi
    
    init_groups
    
    local group_list="${PIHOLE_DATA_DIR}/group-${group_name}.list"
    
    # Check if group exists
    if ! grep -q "^${group_name}|" "${GROUPS_CONFIG}" 2>/dev/null; then
        echo "Error: Group '${group_name}' not found" >&2
        return 1
    fi
    
    # Add domain to group blocklist
    echo "$domain" >> "$group_list"
    
    # Apply to all clients in this group
    while IFS='|' read -r ip group comment; do
        if [[ "$group" == "$group_name" ]]; then
            echo "${ip} ${domain}" >> "${PIHOLE_DATA_DIR}/etc-pihole/custom.list"
        fi
    done < "${CLIENT_GROUPS}"
    
    # Update container
    if [[ -f "${PIHOLE_DATA_DIR}/etc-pihole/custom.list" ]]; then
        docker cp "${PIHOLE_DATA_DIR}/etc-pihole/custom.list" "${CONTAINER_NAME}:/etc/pihole/custom.list"
        docker exec "${CONTAINER_NAME}" pihole restartdns
    fi
    
    echo "Domain '${domain}' added to group '${group_name}' blocklist"
}

# Enable/disable group
toggle_group() {
    local group_name="$1"
    local enabled="${2:-1}"  # 1=enable, 0=disable
    
    if [[ -z "$group_name" ]]; then
        echo "Error: Group name required" >&2
        return 1
    fi
    
    init_groups
    
    local status="enabled"
    if [[ "$enabled" == "0" ]]; then
        status="disabled"
    fi
    
    # Update group status
    local temp_file=$(mktemp)
    while IFS='|' read -r name desc current_enabled; do
        if [[ "$name" == "$group_name" ]]; then
            echo "${name}|${desc}|${enabled}" >> "$temp_file"
        else
            echo "${name}|${desc}|${current_enabled}" >> "$temp_file"
        fi
    done < "${GROUPS_CONFIG}"
    
    mv "$temp_file" "${GROUPS_CONFIG}"
    
    echo "Group '${group_name}' ${status}"
}

# Main entry point for CLI
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "${1:-}" in
    "create")
        create_group "${2:-}" "${3:-}"
        ;;
    "delete")
        delete_group "${2:-}"
        ;;
    "list")
        list_groups
        ;;
    "add-client")
        add_client_to_group "${2:-}" "${3:-}" "${4:-}"
        ;;
    "remove-client")
        remove_client_from_group "${2:-}"
        ;;
    "list-clients")
        list_group_clients "${2:-}"
        ;;
    "add-domain")
        add_to_group_blocklist "${2:-}" "${3:-}"
        ;;
    "enable")
        toggle_group "${2:-}" "1"
        ;;
    "disable")
        toggle_group "${2:-}" "0"
        ;;
    *)
        echo "Usage: $0 {create|delete|list|add-client|remove-client|list-clients|add-domain|enable|disable}"
        echo ""
        echo "Commands:"
        echo "  create <name> [description]     - Create a new group"
        echo "  delete <name>                   - Delete a group"
        echo "  list                            - List all groups"
        echo "  add-client <ip> <group> [note]  - Add client to group"
        echo "  remove-client <ip>              - Remove client from groups"
        echo "  list-clients <group>            - List clients in group"
        echo "  add-domain <group> <domain>     - Add domain to group blocklist"
        echo "  enable <name>                   - Enable group"
        echo "  disable <name>                  - Disable group"
        exit 1
        ;;
    esac
fi