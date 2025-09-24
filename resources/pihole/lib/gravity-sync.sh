#!/usr/bin/env bash
# Pi-hole Gravity Sync - Synchronization between multiple Pi-hole instances
set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Source core library
source "${SCRIPT_DIR}/core.sh"

# Sync configuration directory
SYNC_DIR="${PIHOLE_DATA_DIR}/sync"
SYNC_CONFIG="${SYNC_DIR}/sync.conf"

# Initialize sync configuration
init_sync() {
    mkdir -p "${SYNC_DIR}"
    
    if [[ ! -f "${SYNC_CONFIG}" ]]; then
        cat > "${SYNC_CONFIG}" << 'EOF'
# Pi-hole Gravity Sync Configuration
# List of remote Pi-hole instances to sync with
# Format: name|host|port|ssh_key
# Example: secondary|192.168.1.100|22|/path/to/key
EOF
        echo "Sync configuration initialized at: ${SYNC_CONFIG}"
    fi
}

# Add remote Pi-hole instance
add_remote() {
    local name="$1"
    local host="$2"
    local port="${3:-22}"
    local ssh_key="${4:-}"
    
    if [[ -z "$name" ]] || [[ -z "$host" ]]; then
        echo "Error: Name and host required" >&2
        return 1
    fi
    
    init_sync
    
    # Check if already exists
    if grep -q "^${name}|" "${SYNC_CONFIG}" 2>/dev/null; then
        echo "Error: Remote '${name}' already exists" >&2
        return 1
    fi
    
    # Add to config
    echo "${name}|${host}|${port}|${ssh_key}" >> "${SYNC_CONFIG}"
    
    echo "Remote Pi-hole '${name}' added"
    echo "Host: ${host}:${port}"
}

# Remove remote Pi-hole instance
remove_remote() {
    local name="$1"
    
    if [[ -z "$name" ]]; then
        echo "Error: Remote name required" >&2
        return 1
    fi
    
    if [[ ! -f "${SYNC_CONFIG}" ]]; then
        echo "Error: No sync configuration found" >&2
        return 1
    fi
    
    # Remove from config
    sed -i "/^${name}|/d" "${SYNC_CONFIG}"
    
    echo "Remote '${name}' removed"
}

# List configured remotes
list_remotes() {
    if [[ ! -f "${SYNC_CONFIG}" ]]; then
        echo "No sync configuration found"
        return 0
    fi
    
    echo "Configured remote Pi-hole instances:"
    echo "===================================="
    
    while IFS='|' read -r name host port ssh_key; do
        if [[ "$name" =~ ^#.*$ ]] || [[ -z "$name" ]]; then
            continue
        fi
        echo "Name: ${name}"
        echo "Host: ${host}:${port}"
        echo "SSH Key: ${ssh_key:-default}"
        echo "---"
    done < "${SYNC_CONFIG}"
}

# Export gravity database
export_gravity() {
    local export_file="${1:-${SYNC_DIR}/gravity_export_$(date +%Y%m%d_%H%M%S).db}"
    
    echo "Exporting gravity database..."
    
    # Copy gravity database from container
    docker cp "${CONTAINER_NAME}:/etc/pihole/gravity.db" "${export_file}"
    
    # Also export custom configurations
    docker cp "${CONTAINER_NAME}:/etc/pihole/custom.list" "${export_file}.custom" 2>/dev/null || true
    docker cp "${CONTAINER_NAME}:/etc/pihole/regex.list" "${export_file}.regex" 2>/dev/null || true
    
    echo "Gravity database exported to: ${export_file}"
    
    # Create checksum for verification
    sha256sum "${export_file}" > "${export_file}.sha256"
    
    echo "${export_file}"
}

# Import gravity database
import_gravity() {
    local import_file="$1"
    local backup="${2:-true}"
    
    if [[ ! -f "$import_file" ]]; then
        echo "Error: Import file not found: ${import_file}" >&2
        return 1
    fi
    
    # Verify checksum if available
    if [[ -f "${import_file}.sha256" ]]; then
        echo "Verifying checksum..."
        if ! sha256sum -c "${import_file}.sha256" > /dev/null 2>&1; then
            echo "Error: Checksum verification failed" >&2
            return 1
        fi
    fi
    
    # Backup current database if requested
    if [[ "$backup" == "true" ]]; then
        echo "Backing up current database..."
        export_gravity "${SYNC_DIR}/gravity_backup_$(date +%Y%m%d_%H%M%S).db"
    fi
    
    echo "Importing gravity database..."
    
    # Stop Pi-hole DNS
    docker exec "${CONTAINER_NAME}" pihole disable
    
    # Import database
    docker cp "${import_file}" "${CONTAINER_NAME}:/etc/pihole/gravity_new.db"
    docker exec "${CONTAINER_NAME}" bash -c "mv /etc/pihole/gravity_new.db /etc/pihole/gravity.db"
    
    # Import custom configurations if available
    if [[ -f "${import_file}.custom" ]]; then
        docker cp "${import_file}.custom" "${CONTAINER_NAME}:/etc/pihole/custom.list"
    fi
    
    if [[ -f "${import_file}.regex" ]]; then
        docker cp "${import_file}.regex" "${CONTAINER_NAME}:/etc/pihole/regex.list"
    fi
    
    # Restart Pi-hole
    docker exec "${CONTAINER_NAME}" pihole enable
    docker exec "${CONTAINER_NAME}" pihole restartdns
    
    echo "Gravity database imported successfully"
}

# Sync with remote Pi-hole
sync_with_remote() {
    local remote_name="$1"
    local direction="${2:-pull}"  # pull, push, or bidirectional
    
    if [[ -z "$remote_name" ]]; then
        echo "Error: Remote name required" >&2
        return 1
    fi
    
    # Get remote configuration
    local remote_config
    remote_config=$(grep "^${remote_name}|" "${SYNC_CONFIG}" 2>/dev/null || echo "")
    
    if [[ -z "$remote_config" ]]; then
        echo "Error: Remote '${remote_name}' not found" >&2
        return 1
    fi
    
    IFS='|' read -r name host port ssh_key <<< "$remote_config"
    
    case "$direction" in
        "pull")
            echo "Pulling from remote '${name}'..."
            pull_from_remote "$host" "$port" "$ssh_key"
            ;;
        "push")
            echo "Pushing to remote '${name}'..."
            push_to_remote "$host" "$port" "$ssh_key"
            ;;
        "bidirectional")
            echo "Bidirectional sync with '${name}'..."
            bidirectional_sync "$host" "$port" "$ssh_key"
            ;;
        *)
            echo "Error: Unknown sync direction: ${direction}" >&2
            return 1
            ;;
    esac
}

# Pull gravity database from remote
pull_from_remote() {
    local host="$1"
    local port="$2"
    local ssh_key="$3"
    
    local temp_file="${SYNC_DIR}/remote_gravity_$(date +%Y%m%d_%H%M%S).db"
    
    echo "Connecting to remote Pi-hole at ${host}:${port}..."
    
    # SSH options
    local ssh_opts="-o StrictHostKeyChecking=no -o ConnectTimeout=10"
    if [[ -n "$ssh_key" ]]; then
        ssh_opts="${ssh_opts} -i ${ssh_key}"
    fi
    
    # Copy remote gravity database
    if scp -P "${port}" ${ssh_opts} "root@${host}:/etc/pihole/gravity.db" "${temp_file}" 2>/dev/null; then
        echo "Remote database downloaded"
        import_gravity "${temp_file}"
    else
        echo "Error: Failed to connect to remote Pi-hole" >&2
        return 1
    fi
}

# Push gravity database to remote
push_to_remote() {
    local host="$1"
    local port="$2"
    local ssh_key="$3"
    
    # Export local database
    local export_file
    export_file=$(export_gravity)
    
    echo "Connecting to remote Pi-hole at ${host}:${port}..."
    
    # SSH options
    local ssh_opts="-o StrictHostKeyChecking=no -o ConnectTimeout=10"
    if [[ -n "$ssh_key" ]]; then
        ssh_opts="${ssh_opts} -i ${ssh_key}"
    fi
    
    # Copy to remote
    if scp -P "${port}" ${ssh_opts} "${export_file}" "root@${host}:/tmp/gravity_import.db" 2>/dev/null; then
        echo "Database uploaded to remote"
        
        # Import on remote
        ssh -p "${port}" ${ssh_opts} "root@${host}" \
            "pihole disable && mv /tmp/gravity_import.db /etc/pihole/gravity.db && pihole enable && pihole restartdns"
        
        echo "Remote Pi-hole updated"
    else
        echo "Error: Failed to connect to remote Pi-hole" >&2
        return 1
    fi
}

# Bidirectional sync (merge databases)
bidirectional_sync() {
    local host="$1"
    local port="$2"
    local ssh_key="$3"
    
    echo "Performing bidirectional sync..."
    
    # Export local database
    local local_db
    local_db=$(export_gravity)
    
    # Pull remote database
    pull_from_remote "$host" "$port" "$ssh_key"
    
    # Merge would require more complex logic
    # For now, this is a placeholder for future enhancement
    echo "Note: Full bidirectional merge not yet implemented"
    echo "Current implementation pulls from remote (remote wins conflicts)"
}

# Schedule automatic sync
schedule_sync() {
    local interval="${1:-daily}"  # hourly, daily, weekly
    local remote="${2:-all}"
    
    local cron_schedule
    case "$interval" in
        "hourly")
            cron_schedule="0 * * * *"
            ;;
        "daily")
            cron_schedule="0 2 * * *"
            ;;
        "weekly")
            cron_schedule="0 2 * * 0"
            ;;
        *)
            echo "Error: Invalid interval. Use: hourly, daily, or weekly" >&2
            return 1
            ;;
    esac
    
    # Create cron job
    local cron_cmd="${RESOURCE_DIR}/cli.sh gravity-sync sync ${remote} pull"
    local cron_entry="${cron_schedule} ${cron_cmd}"
    
    # Add to crontab
    (crontab -l 2>/dev/null | grep -v "pihole.*gravity-sync"; echo "${cron_entry}") | crontab -
    
    echo "Scheduled ${interval} sync for remote: ${remote}"
}

# Main entry point for CLI
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "${1:-}" in
    "init")
        init_sync
        ;;
    "add-remote")
        add_remote "${2:-}" "${3:-}" "${4:-}" "${5:-}"
        ;;
    "remove-remote")
        remove_remote "${2:-}"
        ;;
    "list-remotes")
        list_remotes
        ;;
    "export")
        export_gravity "${2:-}"
        ;;
    "import")
        import_gravity "${2:-}" "${3:-true}"
        ;;
    "sync")
        sync_with_remote "${2:-}" "${3:-pull}"
        ;;
    "schedule")
        schedule_sync "${2:-}" "${3:-}"
        ;;
    *)
        echo "Usage: $0 {init|add-remote|remove-remote|list-remotes|export|import|sync|schedule}"
        echo ""
        echo "Commands:"
        echo "  init                              - Initialize sync configuration"
        echo "  add-remote <name> <host> [port]  - Add remote Pi-hole instance"
        echo "  remove-remote <name>              - Remove remote instance"
        echo "  list-remotes                      - List configured remotes"
        echo "  export [file]                     - Export gravity database"
        echo "  import <file> [backup]            - Import gravity database"
        echo "  sync <remote> [direction]         - Sync with remote (pull/push/bidirectional)"
        echo "  schedule [interval] [remote]      - Schedule automatic sync"
        exit 1
        ;;
    esac
fi