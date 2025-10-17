#!/usr/bin/env bash
# Pi-hole Group Management - Client group configuration and policies
set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Source core library
source "${SCRIPT_DIR}/core.sh"

# Create a new group
create_group() {
    local group_name="$1"
    local description="${2:-}"
    
    if [[ -z "$group_name" ]]; then
        echo "Error: Group name required" >&2
        return 1
    fi
    
    echo "Creating group: ${group_name}"
    
    # Use Pi-hole's group management
    docker exec "${CONTAINER_NAME}" sqlite3 /etc/pihole/gravity.db \
        "INSERT OR IGNORE INTO 'group' (name, description, enabled) VALUES ('${group_name}', '${description}', 1);"
    
    echo "Group '${group_name}' created"
}

# Delete a group
delete_group() {
    local group_name="$1"
    
    if [[ -z "$group_name" ]]; then
        echo "Error: Group name required" >&2
        return 1
    fi
    
    echo "Deleting group: ${group_name}"
    
    # Get group ID
    local group_id=$(docker exec "${CONTAINER_NAME}" sqlite3 /etc/pihole/gravity.db \
        "SELECT id FROM 'group' WHERE name='${group_name}';")
    
    if [[ -z "$group_id" ]]; then
        echo "Error: Group '${group_name}' not found" >&2
        return 1
    fi
    
    # Remove group
    docker exec "${CONTAINER_NAME}" sqlite3 /etc/pihole/gravity.db \
        "DELETE FROM 'group' WHERE id=${group_id};"
    
    # Remove group associations
    docker exec "${CONTAINER_NAME}" sqlite3 /etc/pihole/gravity.db \
        "DELETE FROM client_by_group WHERE group_id=${group_id};"
    
    docker exec "${CONTAINER_NAME}" sqlite3 /etc/pihole/gravity.db \
        "DELETE FROM domainlist_by_group WHERE group_id=${group_id};"
    
    echo "Group '${group_name}' deleted"
}

# List all groups
list_groups() {
    echo "Available groups:"
    echo "================"
    
    docker exec "${CONTAINER_NAME}" sqlite3 /etc/pihole/gravity.db \
        "SELECT id, name, description, CASE enabled WHEN 1 THEN 'enabled' ELSE 'disabled' END FROM 'group';" \
        | column -t -s '|'
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
    
    echo "Adding client ${client_ip} to group ${group_name}"
    
    # Get group ID
    local group_id=$(docker exec "${CONTAINER_NAME}" sqlite3 /etc/pihole/gravity.db \
        "SELECT id FROM 'group' WHERE name='${group_name}';")
    
    if [[ -z "$group_id" ]]; then
        echo "Error: Group '${group_name}' not found" >&2
        return 1
    fi
    
    # Add client
    docker exec "${CONTAINER_NAME}" sqlite3 /etc/pihole/gravity.db \
        "INSERT OR IGNORE INTO client (ip, comment) VALUES ('${client_ip}', '${comment}');"
    
    # Get client ID
    local client_id=$(docker exec "${CONTAINER_NAME}" sqlite3 /etc/pihole/gravity.db \
        "SELECT id FROM client WHERE ip='${client_ip}';")
    
    # Associate client with group
    docker exec "${CONTAINER_NAME}" sqlite3 /etc/pihole/gravity.db \
        "INSERT OR IGNORE INTO client_by_group (client_id, group_id) VALUES (${client_id}, ${group_id});"
    
    echo "Client ${client_ip} added to group ${group_name}"
}

# Remove client from group
remove_client_from_group() {
    local client_ip="$1"
    local group_name="$2"
    
    if [[ -z "$client_ip" ]] || [[ -z "$group_name" ]]; then
        echo "Error: Client IP and group name required" >&2
        return 1
    fi
    
    echo "Removing client ${client_ip} from group ${group_name}"
    
    # Get group ID
    local group_id=$(docker exec "${CONTAINER_NAME}" sqlite3 /etc/pihole/gravity.db \
        "SELECT id FROM 'group' WHERE name='${group_name}';")
    
    if [[ -z "$group_id" ]]; then
        echo "Error: Group '${group_name}' not found" >&2
        return 1
    fi
    
    # Get client ID
    local client_id=$(docker exec "${CONTAINER_NAME}" sqlite3 /etc/pihole/gravity.db \
        "SELECT id FROM client WHERE ip='${client_ip}';")
    
    if [[ -z "$client_id" ]]; then
        echo "Error: Client '${client_ip}' not found" >&2
        return 1
    fi
    
    # Remove association
    docker exec "${CONTAINER_NAME}" sqlite3 /etc/pihole/gravity.db \
        "DELETE FROM client_by_group WHERE client_id=${client_id} AND group_id=${group_id};"
    
    echo "Client ${client_ip} removed from group ${group_name}"
}

# List clients in a group
list_group_clients() {
    local group_name="$1"
    
    if [[ -z "$group_name" ]]; then
        echo "Error: Group name required" >&2
        return 1
    fi
    
    # Get group ID
    local group_id=$(docker exec "${CONTAINER_NAME}" sqlite3 /etc/pihole/gravity.db \
        "SELECT id FROM 'group' WHERE name='${group_name}';")
    
    if [[ -z "$group_id" ]]; then
        echo "Error: Group '${group_name}' not found" >&2
        return 1
    fi
    
    echo "Clients in group '${group_name}':"
    echo "================================"
    
    docker exec "${CONTAINER_NAME}" sqlite3 /etc/pihole/gravity.db \
        "SELECT c.ip, c.comment FROM client c 
         JOIN client_by_group cbg ON c.id = cbg.client_id 
         WHERE cbg.group_id = ${group_id};" \
        | column -t -s '|'
}

# Apply blocklist to group
apply_blocklist_to_group() {
    local blocklist_id="$1"
    local group_name="$2"
    
    if [[ -z "$blocklist_id" ]] || [[ -z "$group_name" ]]; then
        echo "Error: Blocklist ID and group name required" >&2
        return 1
    fi
    
    # Get group ID
    local group_id=$(docker exec "${CONTAINER_NAME}" sqlite3 /etc/pihole/gravity.db \
        "SELECT id FROM 'group' WHERE name='${group_name}';")
    
    if [[ -z "$group_id" ]]; then
        echo "Error: Group '${group_name}' not found" >&2
        return 1
    fi
    
    # Associate blocklist with group
    docker exec "${CONTAINER_NAME}" sqlite3 /etc/pihole/gravity.db \
        "INSERT OR IGNORE INTO domainlist_by_group (domainlist_id, group_id) VALUES (${blocklist_id}, ${group_id});"
    
    echo "Blocklist ${blocklist_id} applied to group ${group_name}"
}

# Enable/disable group
toggle_group() {
    local group_name="$1"
    local enabled="${2:-1}"  # 1=enable, 0=disable
    
    if [[ -z "$group_name" ]]; then
        echo "Error: Group name required" >&2
        return 1
    fi
    
    local status="enabled"
    if [[ "$enabled" == "0" ]]; then
        status="disabled"
    fi
    
    echo "Setting group '${group_name}' to ${status}"
    
    docker exec "${CONTAINER_NAME}" sqlite3 /etc/pihole/gravity.db \
        "UPDATE 'group' SET enabled=${enabled} WHERE name='${group_name}';"
    
    echo "Group '${group_name}' ${status}"
}

# Update gravity database after changes
update_gravity() {
    echo "Updating gravity database..."
    docker exec "${CONTAINER_NAME}" pihole -g
    echo "Gravity database updated"
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
        remove_client_from_group "${2:-}" "${3:-}"
        ;;
    "list-clients")
        list_group_clients "${2:-}"
        ;;
    "apply-blocklist")
        apply_blocklist_to_group "${2:-}" "${3:-}"
        ;;
    "enable")
        toggle_group "${2:-}" "1"
        ;;
    "disable")
        toggle_group "${2:-}" "0"
        ;;
    "update")
        update_gravity
        ;;
    *)
        echo "Usage: $0 {create|delete|list|add-client|remove-client|list-clients|apply-blocklist|enable|disable|update}"
        echo ""
        echo "Commands:"
        echo "  create <name> [description]     - Create a new group"
        echo "  delete <name>                   - Delete a group"
        echo "  list                            - List all groups"
        echo "  add-client <ip> <group> [note]  - Add client to group"
        echo "  remove-client <ip> <group>      - Remove client from group"
        echo "  list-clients <group>            - List clients in group"
        echo "  apply-blocklist <id> <group>    - Apply blocklist to group"
        echo "  enable <name>                   - Enable group"
        echo "  disable <name>                  - Disable group"
        echo "  update                          - Update gravity database"
        exit 1
        ;;
    esac
fi