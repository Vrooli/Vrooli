#!/usr/bin/env bash
# WireGuard Core Functions Library

set -euo pipefail

# Container and service configuration
CONTAINER_NAME="${CONTAINER_NAME:-vrooli-wireguard}"
WIREGUARD_IMAGE="${WIREGUARD_IMAGE:-lscr.io/linuxserver/wireguard:latest}"
WIREGUARD_PORT="${WIREGUARD_PORT:-51820}"
WIREGUARD_NETWORK="${WIREGUARD_NETWORK:-10.13.13.0/24}"
CONFIG_DIR="${HOME}/.vrooli/resources/wireguard"

# ====================
# Lifecycle Management
# ====================
handle_manage_command() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        install)
            install_wireguard "$@"
            ;;
        uninstall)
            uninstall_wireguard "$@"
            ;;
        start)
            start_wireguard "$@"
            ;;
        stop)
            stop_wireguard "$@"
            ;;
        restart)
            restart_wireguard "$@"
            ;;
        *)
            echo "Error: Unknown manage subcommand: $subcommand" >&2
            exit 1
            ;;
    esac
}

install_wireguard() {
    local force_flag=""
    local skip_validation=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --force) force_flag="--force"; shift ;;
            --skip-validation) skip_validation="true"; shift ;;
            *) shift ;;
        esac
    done
    
    echo "Installing WireGuard resource..."
    
    # Check if already installed
    if docker inspect "$CONTAINER_NAME" &>/dev/null; then
        echo "WireGuard is already installed"
        return 2  # Already installed exit code
    fi
    
    # Create config directory
    mkdir -p "$CONFIG_DIR"
    
    # Pull Docker image
    echo "Pulling WireGuard Docker image..."
    docker pull "$WIREGUARD_IMAGE" || {
        echo "Error: Failed to pull WireGuard image" >&2
        return 1
    }
    
    # Validate installation if not skipped
    if [[ "$skip_validation" != "true" ]]; then
        echo "Validating installation..."
        docker run --rm "$WIREGUARD_IMAGE" wg --version &>/dev/null || {
            echo "Error: WireGuard validation failed" >&2
            return 1
        }
    fi
    
    echo "WireGuard installed successfully"
    return 0
}

uninstall_wireguard() {
    local force_flag=""
    local keep_data=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --force) force_flag="--force"; shift ;;
            --keep-data) keep_data="true"; shift ;;
            *) shift ;;
        esac
    done
    
    echo "Uninstalling WireGuard resource..."
    
    # Stop container if running
    if docker ps -q -f name="$CONTAINER_NAME" | grep -q .; then
        echo "Stopping WireGuard container..."
        docker stop "$CONTAINER_NAME" || true
    fi
    
    # Remove container
    if docker inspect "$CONTAINER_NAME" &>/dev/null; then
        echo "Removing WireGuard container..."
        docker rm "$CONTAINER_NAME" || true
    fi
    
    # Remove data if not keeping
    if [[ "$keep_data" != "true" ]]; then
        echo "Removing WireGuard data..."
        rm -rf "$CONFIG_DIR"
    fi
    
    echo "WireGuard uninstalled successfully"
    return 0
}

start_wireguard() {
    local wait_flag=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --wait) wait_flag="true"; shift ;;
            *) shift ;;
        esac
    done
    
    echo "Starting WireGuard service..."
    
    # Check if already running
    if docker ps -q -f name="$CONTAINER_NAME" | grep -q .; then
        echo "WireGuard is already running"
        return 0
    fi
    
    # Check if container exists but stopped
    if docker inspect "$CONTAINER_NAME" &>/dev/null; then
        docker start "$CONTAINER_NAME"
    else
        # Create and start new container
        docker run -d \
            --name "$CONTAINER_NAME" \
            --cap-add=NET_ADMIN \
            --cap-add=SYS_MODULE \
            --cap-add=SYS_ADMIN \
            --privileged \
            --sysctl="net.ipv4.conf.all.src_valid_mark=1" \
            --sysctl="net.ipv4.ip_forward=1" \
            -p "${WIREGUARD_PORT}:51820/udp" \
            -v "${CONFIG_DIR}:/config" \
            -v /lib/modules:/lib/modules:ro \
            --restart unless-stopped \
            "$WIREGUARD_IMAGE"
    fi
    
    # Wait for health if requested
    if [[ "$wait_flag" == "true" ]]; then
        echo "Waiting for WireGuard to be healthy..."
        local max_attempts=30
        local attempt=0
        
        while [[ $attempt -lt $max_attempts ]]; do
            if check_health; then
                echo "WireGuard is healthy"
                return 0
            fi
            sleep 2
            ((attempt++))
        done
        
        echo "Error: WireGuard health check timeout" >&2
        return 1
    fi
    
    echo "WireGuard started successfully"
    return 0
}

stop_wireguard() {
    echo "Stopping WireGuard service..."
    
    if ! docker ps -q -f name="$CONTAINER_NAME" | grep -q .; then
        echo "WireGuard is not running"
        return 0
    fi
    
    docker stop "$CONTAINER_NAME" || {
        echo "Error: Failed to stop WireGuard" >&2
        return 1
    }
    
    echo "WireGuard stopped successfully"
    return 0
}

restart_wireguard() {
    echo "Restarting WireGuard service..."
    stop_wireguard || true
    sleep 2
    start_wireguard "$@"
}

# ====================
# Health Monitoring
# ====================
check_health() {
    # Check if container is running
    if ! docker ps -q -f name="$CONTAINER_NAME" | grep -q .; then
        return 1
    fi
    
    # Check WireGuard interface inside container
    docker exec "$CONTAINER_NAME" wg show &>/dev/null || return 1
    
    return 0
}

show_status() {
    echo "WireGuard Service Status"
    echo "========================"
    
    if docker ps -q -f name="$CONTAINER_NAME" | grep -q .; then
        echo "Status: Running"
        echo ""
        echo "Container Details:"
        docker ps --filter name="$CONTAINER_NAME" --format "table {{.Status}}\t{{.Ports}}"
        echo ""
        
        # Show active WireGuard interfaces with statistics
        echo "WireGuard Interfaces:"
        echo "--------------------"
        local interfaces=$(docker exec "$CONTAINER_NAME" wg show interfaces 2>/dev/null)
        
        if [[ -n "$interfaces" ]]; then
            for interface in $interfaces; do
                echo ""
                echo "Interface: $interface"
                
                # Get interface details
                docker exec "$CONTAINER_NAME" wg show "$interface" 2>/dev/null | while IFS= read -r line; do
                    echo "  $line"
                done
                
                # Get traffic statistics
                echo ""
                echo "  Traffic Statistics:"
                local rx_bytes=$(docker exec "$CONTAINER_NAME" cat "/sys/class/net/$interface/statistics/rx_bytes" 2>/dev/null || echo "0")
                local tx_bytes=$(docker exec "$CONTAINER_NAME" cat "/sys/class/net/$interface/statistics/tx_bytes" 2>/dev/null || echo "0")
                local rx_packets=$(docker exec "$CONTAINER_NAME" cat "/sys/class/net/$interface/statistics/rx_packets" 2>/dev/null || echo "0")
                local tx_packets=$(docker exec "$CONTAINER_NAME" cat "/sys/class/net/$interface/statistics/tx_packets" 2>/dev/null || echo "0")
                
                # Convert bytes to human-readable format
                echo "    RX: $(format_bytes $rx_bytes) ($rx_packets packets)"
                echo "    TX: $(format_bytes $tx_bytes) ($tx_packets packets)"
            done
        else
            echo "No active interfaces"
            echo ""
            echo "Available configurations:"
            list_tunnel_configs | grep "^- " || echo "None"
        fi
        
        # Show system resource usage
        echo ""
        echo "Resource Usage:"
        echo "--------------"
        docker stats --no-stream --format "  CPU: {{.CPUPerc}}\n  Memory: {{.MemUsage}}" "$CONTAINER_NAME" 2>/dev/null || echo "  Unable to retrieve stats"
        
    else
        echo "Status: Stopped"
        if docker inspect "$CONTAINER_NAME" &>/dev/null; then
            echo "Container exists but is not running"
        else
            echo "Container not found (run 'resource-wireguard manage install' first)"
        fi
    fi
}

# Helper function to format bytes to human-readable
format_bytes() {
    local bytes=$1
    if [[ $bytes -ge 1073741824 ]]; then
        echo "$(echo "scale=2; $bytes/1073741824" | bc) GB"
    elif [[ $bytes -ge 1048576 ]]; then
        echo "$(echo "scale=2; $bytes/1048576" | bc) MB"
    elif [[ $bytes -ge 1024 ]]; then
        echo "$(echo "scale=2; $bytes/1024" | bc) KB"
    else
        echo "$bytes B"
    fi
}

# ====================
# Traffic Statistics
# ====================
show_traffic_stats() {
    echo "WireGuard Traffic Statistics"
    echo "============================"
    
    if ! docker ps -q -f name="$CONTAINER_NAME" | grep -q .; then
        echo "Error: WireGuard container not running" >&2
        return 1
    fi
    
    local interfaces=$(docker exec "$CONTAINER_NAME" wg show interfaces 2>/dev/null)
    
    if [[ -z "$interfaces" ]]; then
        echo "No active WireGuard interfaces"
        return 0
    fi
    
    for interface in $interfaces; do
        echo ""
        echo "Interface: $interface"
        echo "--------------------"
        
        # Get peer statistics from wg show
        docker exec "$CONTAINER_NAME" wg show "$interface" 2>/dev/null | while IFS= read -r line; do
            if [[ "$line" =~ ^peer: ]]; then
                echo ""
                echo "$line"
            elif [[ "$line" =~ transfer: ]]; then
                echo "  $line"
            elif [[ "$line" =~ "latest handshake:" ]]; then
                echo "  $line"
            elif [[ "$line" =~ endpoint: ]]; then
                echo "  $line"
            fi
        done
        
        # Get interface-level statistics
        echo ""
        echo "Interface Statistics:"
        local rx_bytes=$(docker exec "$CONTAINER_NAME" cat "/sys/class/net/$interface/statistics/rx_bytes" 2>/dev/null || echo "0")
        local tx_bytes=$(docker exec "$CONTAINER_NAME" cat "/sys/class/net/$interface/statistics/tx_bytes" 2>/dev/null || echo "0")
        local rx_packets=$(docker exec "$CONTAINER_NAME" cat "/sys/class/net/$interface/statistics/rx_packets" 2>/dev/null || echo "0")
        local tx_packets=$(docker exec "$CONTAINER_NAME" cat "/sys/class/net/$interface/statistics/tx_packets" 2>/dev/null || echo "0")
        local rx_errors=$(docker exec "$CONTAINER_NAME" cat "/sys/class/net/$interface/statistics/rx_errors" 2>/dev/null || echo "0")
        local tx_errors=$(docker exec "$CONTAINER_NAME" cat "/sys/class/net/$interface/statistics/tx_errors" 2>/dev/null || echo "0")
        local rx_dropped=$(docker exec "$CONTAINER_NAME" cat "/sys/class/net/$interface/statistics/rx_dropped" 2>/dev/null || echo "0")
        local tx_dropped=$(docker exec "$CONTAINER_NAME" cat "/sys/class/net/$interface/statistics/tx_dropped" 2>/dev/null || echo "0")
        
        echo "  Total Received:    $(format_bytes $rx_bytes) ($rx_packets packets)"
        echo "  Total Transmitted: $(format_bytes $tx_bytes) ($tx_packets packets)"
        
        if [[ $rx_errors -gt 0 || $tx_errors -gt 0 || $rx_dropped -gt 0 || $tx_dropped -gt 0 ]]; then
            echo ""
            echo "  Errors/Drops:"
            [[ $rx_errors -gt 0 ]] && echo "    RX Errors: $rx_errors"
            [[ $tx_errors -gt 0 ]] && echo "    TX Errors: $tx_errors"
            [[ $rx_dropped -gt 0 ]] && echo "    RX Dropped: $rx_dropped"
            [[ $tx_dropped -gt 0 ]] && echo "    TX Dropped: $tx_dropped"
        fi
        
        # Calculate and show bandwidth if possible
        if [[ -f "/tmp/wireguard_stats_${interface}" ]]; then
            local old_stats=$(cat "/tmp/wireguard_stats_${interface}")
            local old_rx=$(echo "$old_stats" | cut -d' ' -f1)
            local old_tx=$(echo "$old_stats" | cut -d' ' -f2)
            local old_time=$(echo "$old_stats" | cut -d' ' -f3)
            local current_time=$(date +%s)
            local time_diff=$((current_time - old_time))
            
            if [[ $time_diff -gt 0 ]]; then
                local rx_rate=$(( (rx_bytes - old_rx) / time_diff ))
                local tx_rate=$(( (tx_bytes - old_tx) / time_diff ))
                echo ""
                echo "  Current Bandwidth:"
                echo "    RX: $(format_bytes $rx_rate)/s"
                echo "    TX: $(format_bytes $tx_rate)/s"
            fi
        fi
        
        # Save current stats for next calculation
        echo "$rx_bytes $tx_bytes $(date +%s)" > "/tmp/wireguard_stats_${interface}"
    done
    
    echo ""
    echo "Container Resource Usage:"
    echo "------------------------"
    docker stats --no-stream --format "CPU: {{.CPUPerc}}\nMemory: {{.MemUsage}}\nNetwork I/O: {{.NetIO}}" "$CONTAINER_NAME" 2>/dev/null || echo "Unable to retrieve container stats"
}

# ====================
# Content Management
# ====================
handle_content_command() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        add)
            add_tunnel_config "$@"
            ;;
        list)
            list_tunnel_configs "$@"
            ;;
        get)
            get_tunnel_config "$@"
            ;;
        remove)
            remove_tunnel_config "$@"
            ;;
        execute)
            execute_tunnel_config "$@"
            ;;
        *)
            echo "Error: Unknown content subcommand: $subcommand" >&2
            exit 1
            ;;
    esac
}

add_tunnel_config() {
    local name="${1:-}"
    
    if [[ -z "$name" ]]; then
        echo "Error: Tunnel name required" >&2
        return 1
    fi
    
    echo "Adding tunnel configuration: $name"
    
    # Ensure container is running for key generation
    if ! docker ps -q -f name="$CONTAINER_NAME" | grep -q .; then
        echo "Error: WireGuard container not running. Start it first with 'manage start'" >&2
        return 1
    fi
    
    # Create wg_confs directory if it doesn't exist
    mkdir -p "${CONFIG_DIR}/wg_confs"
    
    # Create config file path - using wg_confs subdirectory that container expects
    local config_file="${CONFIG_DIR}/wg_confs/wg${name}.conf"
    
    # Generate private key using Docker container
    local private_key=$(docker exec "$CONTAINER_NAME" wg genkey 2>/dev/null)
    if [[ -z "$private_key" ]]; then
        echo "Error: Failed to generate private key" >&2
        return 1
    fi
    
    local public_key=$(echo "$private_key" | docker exec -i "$CONTAINER_NAME" wg pubkey 2>/dev/null)
    if [[ -z "$public_key" ]]; then
        echo "Error: Failed to generate public key" >&2
        return 1
    fi
    
    # Determine next available IP address
    local ip_suffix=2  # Start from .2 (.1 is typically the server)
    for existing in "${CONFIG_DIR}/wg_confs"/wg*.conf; do
        if [[ -f "$existing" ]]; then
            ((ip_suffix++))
        fi
    done
    
    # Create the config using docker exec to avoid permission issues
    docker exec "$CONTAINER_NAME" bash -c "cat > /config/wg_confs/wg${name}.conf << EOF
[Interface]
PrivateKey = $private_key
Address = ${WIREGUARD_NETWORK%.*}.${ip_suffix}/32
ListenPort = 51820
DNS = 1.1.1.1

# Public Key: $public_key
# Server endpoint: <your-server-ip>:${WIREGUARD_PORT}

# Example peer configuration:
# [Peer]
# PublicKey = <peer-public-key>
# AllowedIPs = ${WIREGUARD_NETWORK%.*}.0/24
# Endpoint = <peer-ip>:51820
# PersistentKeepalive = 25  # Important for NAT traversal
EOF"
    
    echo "Tunnel configuration created: $name"
    echo "Public Key: $public_key"
    echo "IP Address: ${WIREGUARD_NETWORK%.*}.${ip_suffix}/32"
    echo ""
    echo "To activate this tunnel, run: resource-wireguard content execute $name"
    return 0
}

list_tunnel_configs() {
    echo "Available tunnel configurations:"
    echo "================================"
    
    if [[ -d "${CONFIG_DIR}/wg_confs" ]]; then
        local found=false
        for config in "${CONFIG_DIR}/wg_confs"/wg*.conf; do
            if [[ -f "$config" ]]; then
                found=true
                local name=$(basename "$config" | sed 's/^wg//;s/.conf$//')
                echo "- $name"
                
                # Show basic info from config if container is running
                if docker ps -q -f name="$CONTAINER_NAME" | grep -q .; then
                    local address=$(docker exec "$CONTAINER_NAME" grep "^Address" "/config/wg_confs/$(basename "$config")" 2>/dev/null | cut -d= -f2 | tr -d ' ')
                    [[ -n "$address" ]] && echo "  Address: $address"
                fi
            fi
        done
        
        if [[ "$found" == "false" ]]; then
            echo "No configurations found"
        fi
    else
        echo "No configurations found"
    fi
    
    # Show active tunnels if any
    if docker ps -q -f name="$CONTAINER_NAME" | grep -q .; then
        echo ""
        echo "Active tunnels:"
        echo "---------------"
        local active=$(docker exec "$CONTAINER_NAME" wg show interfaces 2>/dev/null)
        if [[ -n "$active" ]]; then
            echo "$active"
        else
            echo "None"
        fi
    fi
}

get_tunnel_config() {
    local name="${1:-}"
    
    if [[ -z "$name" ]]; then
        echo "Error: Tunnel name required" >&2
        return 1
    fi
    
    if ! docker ps -q -f name="$CONTAINER_NAME" | grep -q .; then
        echo "Error: WireGuard container not running" >&2
        return 1
    fi
    
    local config_file="/config/wg_confs/wg${name}.conf"
    
    # Check if config exists
    if ! docker exec "$CONTAINER_NAME" test -f "$config_file" 2>/dev/null; then
        echo "Error: Configuration not found: $name" >&2
        return 1
    fi
    
    # Display config using docker exec
    docker exec "$CONTAINER_NAME" cat "$config_file"
}

remove_tunnel_config() {
    local name="${1:-}"
    
    if [[ -z "$name" ]]; then
        echo "Error: Tunnel name required" >&2
        return 1
    fi
    
    if ! docker ps -q -f name="$CONTAINER_NAME" | grep -q .; then
        echo "Error: WireGuard container not running" >&2
        return 1
    fi
    
    local config_file="/config/wg_confs/wg${name}.conf"
    
    # Check if config exists
    if ! docker exec "$CONTAINER_NAME" test -f "$config_file" 2>/dev/null; then
        echo "Error: Configuration not found: $name" >&2
        return 1
    fi
    
    # First, deactivate tunnel if it's active
    local interface="wg${name}"
    if docker exec "$CONTAINER_NAME" ip link show "$interface" &>/dev/null; then
        echo "Deactivating tunnel before removal..."
        docker exec "$CONTAINER_NAME" wg-quick down "$interface" 2>/dev/null || true
    fi
    
    # Remove config file
    docker exec "$CONTAINER_NAME" rm "$config_file"
    echo "Tunnel configuration removed: $name"
}

execute_tunnel_config() {
    local name="${1:-}"
    
    if [[ -z "$name" ]]; then
        echo "Error: Tunnel name required" >&2
        return 1
    fi
    
    echo "Activating tunnel: $name"
    
    if ! docker ps -q -f name="$CONTAINER_NAME" | grep -q .; then
        echo "Error: WireGuard container not running" >&2
        return 1
    fi
    
    local config_file="/config/wg_confs/wg${name}.conf"
    local interface="wg${name}"
    
    # Check if config exists
    if ! docker exec "$CONTAINER_NAME" test -f "$config_file" 2>/dev/null; then
        echo "Error: Configuration not found: $name" >&2
        return 1
    fi
    
    # Check if interface already exists
    if docker exec "$CONTAINER_NAME" ip link show "$interface" &>/dev/null; then
        echo "Tunnel already active: $name"
        docker exec "$CONTAINER_NAME" wg show "$interface" 2>/dev/null || true
        return 0
    fi
    
    # Create symbolic link for wg-quick to find the config
    docker exec "$CONTAINER_NAME" ln -sf "$config_file" "/config/${interface}.conf" 2>/dev/null || true
    
    # Activate the tunnel
    docker exec "$CONTAINER_NAME" wg-quick up "$interface" || {
        echo "Error: Failed to activate tunnel" >&2
        return 1
    }
    
    echo "Tunnel activated: $name"
    
    # Show tunnel status
    docker exec "$CONTAINER_NAME" wg show "$interface" 2>/dev/null || true
    return 0
}

# ====================
# Logging
# ====================
show_logs() {
    local tail_lines=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --tail) tail_lines="$2"; shift 2 ;;
            *) shift ;;
        esac
    done
    
    if [[ -n "$tail_lines" ]]; then
        docker logs --tail "$tail_lines" "$CONTAINER_NAME" 2>&1
    else
        docker logs "$CONTAINER_NAME" 2>&1
    fi
}

# ====================
# Credentials
# ====================
show_credentials() {
    echo "WireGuard Connection Information"
    echo "================================"
    echo "UDP Port: $WIREGUARD_PORT"
    echo "Network: $WIREGUARD_NETWORK"
    echo "Config Directory: $CONFIG_DIR"
    echo ""
    echo "To connect a peer:"
    echo "1. Generate peer keys: wg genkey | tee private.key | wg pubkey > public.key"
    echo "2. Add peer to server config"
    echo "3. Configure peer with server public key"
}

# ====================
# Key Rotation System
# ====================
handle_rotate_command() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        keys)
            rotate_keys "$@"
            ;;
        schedule)
            schedule_rotation "$@"
            ;;
        status)
            rotation_status "$@"
            ;;
        *)
            echo "Usage: resource-wireguard rotate [keys|schedule|status] [options]"
            echo ""
            echo "Key rotation management for enhanced security"
            echo ""
            echo "Commands:"
            echo "  keys      - Rotate keys for a tunnel immediately"
            echo "  schedule  - Schedule automatic key rotation"
            echo "  status    - Show rotation status and history"
            echo ""
            echo "Examples:"
            echo "  resource-wireguard rotate keys mytunnel"
            echo "  resource-wireguard rotate schedule mytunnel --interval 30d"
            echo "  resource-wireguard rotate status"
            return 0
            ;;
    esac
}

rotate_keys() {
    local tunnel_name="${1:-}"
    local backup_old="${2:-true}"
    
    if [[ -z "$tunnel_name" ]]; then
        echo "Error: Tunnel name required" >&2
        echo "Usage: resource-wireguard rotate keys <tunnel-name> [--no-backup]" >&2
        return 1
    fi
    
    # Check if container is running
    if ! docker ps -q -f name="$CONTAINER_NAME" | grep -q .; then
        echo "Error: WireGuard container not running" >&2
        return 1
    fi
    
    # Check if tunnel config exists
    local config_file="/config/wg_confs/wg${tunnel_name}.conf"
    if ! docker exec "$CONTAINER_NAME" test -f "$config_file"; then
        echo "Error: Tunnel configuration '$tunnel_name' not found" >&2
        return 1
    fi
    
    echo "Rotating keys for tunnel: $tunnel_name"
    
    # Backup old configuration if requested
    if [[ "$backup_old" == "true" ]]; then
        local backup_dir="/config/backups"
        local timestamp=$(date +%Y%m%d_%H%M%S)
        docker exec "$CONTAINER_NAME" mkdir -p "$backup_dir"
        docker exec "$CONTAINER_NAME" cp "$config_file" "${backup_dir}/wg${tunnel_name}_${timestamp}.conf.bak"
        echo "Backed up old configuration to: wg${tunnel_name}_${timestamp}.conf.bak"
    fi
    
    # Generate new keys
    local new_private_key=$(docker exec "$CONTAINER_NAME" wg genkey 2>/dev/null)
    if [[ -z "$new_private_key" ]]; then
        echo "Error: Failed to generate new private key" >&2
        return 1
    fi
    
    local new_public_key=$(echo "$new_private_key" | docker exec -i "$CONTAINER_NAME" wg pubkey 2>/dev/null)
    if [[ -z "$new_public_key" ]]; then
        echo "Error: Failed to generate new public key" >&2
        return 1
    fi
    
    # Get current configuration
    local current_config=$(docker exec "$CONTAINER_NAME" cat "$config_file")
    local current_address=$(echo "$current_config" | grep "^Address" | cut -d'=' -f2 | xargs)
    local current_port=$(echo "$current_config" | grep "^ListenPort" | cut -d'=' -f2 | xargs)
    
    # Create new configuration with rotated keys
    docker exec "$CONTAINER_NAME" bash -c "cat > ${config_file}.tmp << EOF
[Interface]
PrivateKey = $new_private_key
Address = $current_address
ListenPort = ${current_port:-51820}
DNS = 1.1.1.1

# Public Key: $new_public_key
# Key Rotated: $(date -Iseconds)
# Previous rotation backup available in /config/backups/

$(echo \"$current_config\" | sed -n '/^\\[Peer\\]/,\$p')
EOF"
    
    # Apply new configuration
    docker exec "$CONTAINER_NAME" mv "${config_file}.tmp" "$config_file"
    
    # Store rotation metadata
    local metadata_file="/config/rotation_history.json"
    docker exec "$CONTAINER_NAME" bash -c "
        if [[ ! -f $metadata_file ]]; then
            echo '[]' > $metadata_file
        fi
        
        # Add rotation record
        jq '. += [{
            \"tunnel\": \"$tunnel_name\",
            \"timestamp\": \"$(date -Iseconds)\",
            \"old_public_key\": \"$(echo \"$current_config\" | grep '# Public Key:' | cut -d: -f2 | xargs)\",
            \"new_public_key\": \"$new_public_key\"
        }]' $metadata_file > ${metadata_file}.tmp && mv ${metadata_file}.tmp $metadata_file
    " 2>/dev/null || {
        # Fallback if jq is not available
        docker exec "$CONTAINER_NAME" bash -c "
            echo \"$(date -Iseconds): Rotated keys for $tunnel_name\" >> /config/rotation_log.txt
        "
    }
    
    # Reload WireGuard interface if active
    docker exec "$CONTAINER_NAME" bash -c "
        if wg show wg${tunnel_name} &>/dev/null; then
            wg-quick down wg${tunnel_name} 2>/dev/null || true
            wg-quick up wg${tunnel_name} 2>/dev/null || true
            echo 'Reloaded tunnel with new keys'
        fi
    " 2>/dev/null || true
    
    echo "Key rotation completed successfully"
    echo "New Public Key: $new_public_key"
    echo ""
    echo "IMPORTANT: Update all peers with the new public key"
    return 0
}

schedule_rotation() {
    local tunnel_name="${1:-}"
    shift || true
    
    local interval="30d"  # Default 30 days
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --interval)
                interval="${2:-30d}"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$tunnel_name" ]]; then
        echo "Error: Tunnel name required" >&2
        echo "Usage: resource-wireguard rotate schedule <tunnel-name> [--interval <time>]" >&2
        return 1
    fi
    
    echo "Scheduling key rotation for tunnel: $tunnel_name"
    echo "Rotation interval: $interval"
    
    # Create rotation schedule configuration
    local schedule_file="/config/rotation_schedules.json"
    docker exec "$CONTAINER_NAME" bash -c "
        if [[ ! -f $schedule_file ]]; then
            echo '{}' > $schedule_file
        fi
        
        # Parse interval (support formats like 30d, 1w, 720h)
        interval_seconds=0
        if [[ '$interval' =~ ([0-9]+)d ]]; then
            interval_seconds=\$((BASH_REMATCH[1] * 86400))
        elif [[ '$interval' =~ ([0-9]+)w ]]; then
            interval_seconds=\$((BASH_REMATCH[1] * 604800))
        elif [[ '$interval' =~ ([0-9]+)h ]]; then
            interval_seconds=\$((BASH_REMATCH[1] * 3600))
        else
            interval_seconds=2592000  # Default 30 days
        fi
        
        # Update schedule
        next_date=\$(date -d \"+\$interval_seconds seconds\" -Iseconds)
        jq --arg tunnel \"$tunnel_name\" \
           --arg interval \"$interval\" \
           --arg next_date \"\$next_date\" \
           --arg created \"$(date -Iseconds)\" \
           --argjson interval_sec \$interval_seconds \
           '.[\$tunnel] = {
            \"interval\": \$interval,
            \"interval_seconds\": \$interval_sec,
            \"next_rotation\": \$next_date,
            \"created\": \$created
        }' $schedule_file > ${schedule_file}.tmp && mv ${schedule_file}.tmp $schedule_file
    " 2>/dev/null || {
        # Fallback without jq
        docker exec "$CONTAINER_NAME" bash -c "
            echo \"$tunnel_name: $interval\" >> /config/rotation_schedule.txt
        "
        echo "Note: Install jq in container for full scheduling features"
    }
    
    echo "Rotation schedule created successfully"
    echo "Next rotation will occur in: $interval"
    echo ""
    echo "Note: Ensure the WireGuard container remains running for scheduled rotations"
    return 0
}

rotation_status() {
    echo "=== Key Rotation Status ==="
    echo ""
    
    # Check if container is running
    if ! docker ps -q -f name="$CONTAINER_NAME" | grep -q .; then
        echo "Error: WireGuard container not running" >&2
        return 1
    fi
    
    # Show rotation history
    echo "Recent Rotations:"
    if docker exec "$CONTAINER_NAME" test -f /config/rotation_history.json 2>/dev/null; then
        docker exec "$CONTAINER_NAME" bash -c "
            jq -r '.[] | \"  \\(.timestamp): Tunnel \\(.tunnel) - Old key: \\(.old_public_key[0:8])... New key: \\(.new_public_key[0:8])...\"' /config/rotation_history.json 2>/dev/null | tail -10
        " || {
            # Fallback to log file
            docker exec "$CONTAINER_NAME" bash -c "
                if [[ -f /config/rotation_log.txt ]]; then
                    tail -10 /config/rotation_log.txt | sed 's/^/  /'
                else
                    echo '  No rotation history available'
                fi
            "
        }
    else
        echo "  No rotation history available"
    fi
    
    echo ""
    echo "Scheduled Rotations:"
    if docker exec "$CONTAINER_NAME" test -f /config/rotation_schedules.json 2>/dev/null; then
        docker exec "$CONTAINER_NAME" bash -c "
            jq -r 'to_entries[] | \"  \\(.key): Every \\(.value.interval) - Next: \\(.value.next_rotation)\"' /config/rotation_schedules.json 2>/dev/null
        " || {
            # Fallback to schedule file
            docker exec "$CONTAINER_NAME" bash -c "
                if [[ -f /config/rotation_schedule.txt ]]; then
                    cat /config/rotation_schedule.txt | sed 's/^/  /'
                else
                    echo '  No scheduled rotations'
                fi
            "
        }
    else
        echo "  No scheduled rotations"
    fi
    
    echo ""
    echo "Backup Keys:"
    docker exec "$CONTAINER_NAME" bash -c "
        if [[ -d /config/backups ]]; then
            ls -1 /config/backups/*.conf.bak 2>/dev/null | tail -5 | sed 's/^/  /' || echo '  No backups available'
        else
            echo '  No backups available'
        fi
    "
    
    return 0
}

# ====================
# NAT Traversal Management
# ====================
handle_nat_command() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        enable)
            enable_nat_traversal "$@"
            ;;
        disable)
            disable_nat_traversal "$@"
            ;;
        status)
            nat_traversal_status "$@"
            ;;
        test)
            test_nat_traversal "$@"
            ;;
        *)
            echo "Error: Unknown NAT subcommand: $subcommand" >&2
            echo "Usage: resource-wireguard nat {enable|disable|status|test}" >&2
            exit 1
            ;;
    esac
}

enable_nat_traversal() {
    local tunnel_name="${1:-}"
    local persistent_keepalive="${2:-25}"
    
    if [[ -z "$tunnel_name" ]]; then
        echo "Error: Tunnel name required" >&2
        echo "Usage: resource-wireguard nat enable <tunnel-name> [keepalive-interval]" >&2
        return 1
    fi
    
    echo "Enabling NAT traversal for tunnel: $tunnel_name"
    echo "Setting PersistentKeepalive to ${persistent_keepalive} seconds"
    
    # Check if tunnel exists
    if ! docker exec "$CONTAINER_NAME" test -f "/config/wg_confs/wg${tunnel_name}.conf" 2>/dev/null; then
        echo "Error: Tunnel configuration not found: wg${tunnel_name}.conf" >&2
        return 1
    fi
    
    # Enable NAT traversal with PersistentKeepalive
    docker exec "$CONTAINER_NAME" bash -c "
        # Backup current config
        cp /config/wg_confs/wg${tunnel_name}.conf /config/wg_confs/wg${tunnel_name}.conf.nat-backup
        
        # Add or update PersistentKeepalive for all peers
        sed -i '/\[Peer\]/,/\[/{
            /PersistentKeepalive/d
        }' /config/wg_confs/wg${tunnel_name}.conf
        
        # Add PersistentKeepalive to each peer section
        awk '/\[Peer\]/{print; print \"PersistentKeepalive = ${persistent_keepalive}\"; next}1' \
            /config/wg_confs/wg${tunnel_name}.conf > /config/wg_confs/wg${tunnel_name}.conf.tmp
        mv /config/wg_confs/wg${tunnel_name}.conf.tmp /config/wg_confs/wg${tunnel_name}.conf
        
        # Enable IP forwarding for NAT
        echo 1 > /proc/sys/net/ipv4/ip_forward
        
        # Add NAT rules for the tunnel network
        tunnel_subnet=\$(grep 'Address' /config/wg_confs/wg${tunnel_name}.conf | head -1 | cut -d'=' -f2 | xargs | cut -d'/' -f1)
        tunnel_subnet_prefix=\$(echo \$tunnel_subnet | cut -d'.' -f1-3)
        
        # Setup iptables rules for NAT
        iptables -t nat -A POSTROUTING -s \${tunnel_subnet_prefix}.0/24 -o eth0 -j MASQUERADE 2>/dev/null || true
        iptables -A FORWARD -i wg${tunnel_name} -j ACCEPT 2>/dev/null || true
        iptables -A FORWARD -o wg${tunnel_name} -j ACCEPT 2>/dev/null || true
        
        # Save NAT configuration
        echo '{
            \"tunnel\": \"${tunnel_name}\",
            \"persistent_keepalive\": ${persistent_keepalive},
            \"nat_enabled\": true,
            \"timestamp\": \"'\$(date -Iseconds)'\"
        }' > /config/nat_${tunnel_name}.json
        
        # Reload tunnel if it's active
        if ip link show wg${tunnel_name} &>/dev/null; then
            wg-quick down wg${tunnel_name} 2>/dev/null || true
            wg-quick up wg${tunnel_name} 2>/dev/null || true
            echo 'Reloaded tunnel with NAT traversal enabled'
        fi
    " || {
        echo "Error: Failed to enable NAT traversal" >&2
        return 1
    }
    
    echo "NAT traversal enabled successfully"
    echo ""
    echo "Benefits:"
    echo "  - Automatic keep-alive packets every ${persistent_keepalive} seconds"
    echo "  - Maintains connection through NAT/firewall"
    echo "  - Automatic hole-punching for bidirectional traffic"
    echo "  - IP forwarding and masquerading enabled"
    return 0
}

disable_nat_traversal() {
    local tunnel_name="${1:-}"
    
    if [[ -z "$tunnel_name" ]]; then
        echo "Error: Tunnel name required" >&2
        echo "Usage: resource-wireguard nat disable <tunnel-name>" >&2
        return 1
    fi
    
    echo "Disabling NAT traversal for tunnel: $tunnel_name"
    
    docker exec "$CONTAINER_NAME" bash -c "
        # Restore backup if exists
        if [[ -f /config/wg_confs/wg${tunnel_name}.conf.nat-backup ]]; then
            cp /config/wg_confs/wg${tunnel_name}.conf.nat-backup /config/wg_confs/wg${tunnel_name}.conf
        else
            # Remove PersistentKeepalive manually
            sed -i '/PersistentKeepalive/d' /config/wg_confs/wg${tunnel_name}.conf
        fi
        
        # Remove NAT configuration
        rm -f /config/nat_${tunnel_name}.json
        
        # Reload tunnel if active
        if ip link show wg${tunnel_name} &>/dev/null; then
            wg-quick down wg${tunnel_name} 2>/dev/null || true
            wg-quick up wg${tunnel_name} 2>/dev/null || true
            echo 'Reloaded tunnel with NAT traversal disabled'
        fi
    " || {
        echo "Error: Failed to disable NAT traversal" >&2
        return 1
    }
    
    echo "NAT traversal disabled successfully"
    return 0
}

nat_traversal_status() {
    echo "=== NAT Traversal Status ==="
    echo ""
    
    # Check if container is running
    if ! docker ps -q -f name="$CONTAINER_NAME" | grep -q .; then
        echo "Error: WireGuard container not running" >&2
        return 1
    fi
    
    # Check IP forwarding
    echo "IP Forwarding:"
    docker exec "$CONTAINER_NAME" bash -c "
        if [[ \$(cat /proc/sys/net/ipv4/ip_forward) -eq 1 ]]; then
            echo '  ✓ Enabled'
        else
            echo '  ✗ Disabled'
        fi
    "
    
    echo ""
    echo "NAT-Enabled Tunnels:"
    docker exec "$CONTAINER_NAME" bash -c "
        for nat_file in /config/nat_*.json; do
            if [[ -f \$nat_file ]]; then
                tunnel=\$(basename \$nat_file | sed 's/nat_//;s/.json//')
                keepalive=\$(grep persistent_keepalive \$nat_file | cut -d':' -f2 | tr -d ' ,')
                echo \"  \$tunnel: PersistentKeepalive=\${keepalive}s\"
            fi
        done
        
        # If no NAT configs found
        if ! ls /config/nat_*.json &>/dev/null; then
            echo '  None configured'
        fi
    " 2>/dev/null
    
    echo ""
    echo "Active NAT Rules:"
    docker exec "$CONTAINER_NAME" bash -c "
        iptables -t nat -L POSTROUTING -n -v 2>/dev/null | grep MASQUERADE | head -3 | sed 's/^/  /' || echo '  No NAT rules active'
    "
    
    return 0
}

test_nat_traversal() {
    local tunnel_name="${1:-}"
    
    if [[ -z "$tunnel_name" ]]; then
        echo "Error: Tunnel name required" >&2
        echo "Usage: resource-wireguard nat test <tunnel-name>" >&2
        return 1
    fi
    
    echo "Testing NAT traversal for tunnel: $tunnel_name"
    echo ""
    
    # Check if tunnel is active
    if ! docker exec "$CONTAINER_NAME" ip link show "wg${tunnel_name}" &>/dev/null; then
        echo "Error: Tunnel not active. Start it first with:"
        echo "  resource-wireguard content execute $tunnel_name"
        return 1
    fi
    
    # Run connectivity tests
    docker exec "$CONTAINER_NAME" bash -c "
        echo 'Checking tunnel interface...'
        if ip link show wg${tunnel_name} &>/dev/null; then
            echo '  ✓ Tunnel interface active'
        else
            echo '  ✗ Tunnel interface not found'
            exit 1
        fi
        
        echo ''
        echo 'Checking PersistentKeepalive...'
        keepalive=\$(wg show wg${tunnel_name} dump | tail -n +2 | awk '{print \$7}' | head -1)
        if [[ \$keepalive -gt 0 ]]; then
            echo \"  ✓ PersistentKeepalive enabled: \${keepalive}s\"
        else
            echo '  ✗ PersistentKeepalive not configured'
        fi
        
        echo ''
        echo 'Checking latest handshake...'
        latest_handshake=\$(wg show wg${tunnel_name} latest-handshakes | tail -n +2 | awk '{print \$2}' | head -1)
        if [[ -n \$latest_handshake ]] && [[ \$latest_handshake -ne 0 ]]; then
            now=\$(date +%s)
            diff=\$((now - latest_handshake))
            echo \"  ✓ Last handshake: \${diff}s ago\"
            if [[ \$diff -lt 180 ]]; then
                echo '  ✓ Connection is fresh (< 3 minutes)'
            else
                echo '  ⚠ Connection might be stale (> 3 minutes)'
            fi
        else
            echo '  ✗ No handshake recorded'
        fi
        
        echo ''
        echo 'Checking transfer statistics...'
        transfer=\$(wg show wg${tunnel_name} transfer)
        if [[ -n \$transfer ]]; then
            echo '  Transfer data available:'
            echo \"\$transfer\" | sed 's/^/    /'
        else
            echo '  No transfer data available'
        fi
    " || {
        echo "Error: NAT traversal test failed" >&2
        return 1
    }
    
    echo ""
    echo "NAT Traversal Test Complete"
    echo ""
    echo "Tips for troubleshooting NAT issues:"
    echo "  1. Ensure PersistentKeepalive is set (25-30 seconds recommended)"
    echo "  2. Check firewall allows UDP port ${WIREGUARD_PORT}"
    echo "  3. Verify both peers have correct endpoint addresses"
    echo "  4. Monitor handshake times - should update regularly"
    
    return 0
}

# ====================
# Network Isolation Management (Docker-based)
# ====================
handle_namespace_command() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        create)
            create_isolated_network "$@"
            ;;
        list)
            list_isolated_networks "$@"
            ;;
        delete)
            delete_isolated_network "$@"
            ;;
        connect)
            connect_to_network "$@"
            ;;
        status)
            network_isolation_status "$@"
            ;;
        *)
            echo "Error: Unknown namespace subcommand: $subcommand" >&2
            echo "Usage: resource-wireguard namespace {create|list|delete|connect|status}" >&2
            exit 1
            ;;
    esac
}

create_isolated_network() {
    local network_name="${1:-}"
    local tunnel_name="${2:-}"
    
    if [[ -z "$network_name" ]]; then
        echo "Error: Network name required" >&2
        echo "Usage: resource-wireguard namespace create <network-name> [tunnel-name]" >&2
        return 1
    fi
    
    echo "Creating isolated network: wg-${network_name}"
    
    # Generate unique subnet for this network
    NS_ID=$(echo "$network_name" | cksum | cut -d' ' -f1)
    SUBNET_NUM=$((NS_ID % 250 + 1))
    
    # Create Docker network with isolation
    docker network create \
        --driver bridge \
        --subnet="10.200.${SUBNET_NUM}.0/24" \
        --gateway="10.200.${SUBNET_NUM}.1" \
        --opt "com.docker.network.bridge.enable_icc=false" \
        --opt "com.docker.network.bridge.enable_ip_masquerade=true" \
        --label "wireguard.network=true" \
        --label "wireguard.tunnel=${tunnel_name:-none}" \
        "wg-${network_name}" 2>/dev/null || {
        echo "Error: Network wg-${network_name} already exists or subnet conflict" >&2
        return 1
    }
    
    # Connect WireGuard container to this network if tunnel specified
    if [[ -n "$tunnel_name" ]]; then
        docker network connect "wg-${network_name}" "$CONTAINER_NAME" 2>/dev/null || true
        
        # Configure routing inside WireGuard container
        docker exec "$CONTAINER_NAME" bash -c "
            # Add routing rule for this network through WireGuard tunnel
            if [[ -f /config/wg${tunnel_name}.conf ]]; then
                # Extract tunnel interface details
                wg_interface='wg${tunnel_name}'
                
                # Check if tunnel is active
                if wg show \"\$wg_interface\" &>/dev/null; then
                    # Route this network's traffic through WireGuard
                    ip route add 10.200.${SUBNET_NUM}.0/24 dev \"\$wg_interface\" 2>/dev/null || true
                    echo 'Network traffic will be routed through WireGuard tunnel: $tunnel_name'
                else
                    echo 'Warning: Tunnel $tunnel_name not active'
                fi
            fi
        "
    fi
    
    # Store network config inside container
    docker exec "$CONTAINER_NAME" bash -c "
        mkdir -p /config/networks
        cat > /config/networks/${network_name}.json << 'NETEOF'
{
    \"name\": \"${network_name}\",
    \"docker_network\": \"wg-${network_name}\",
    \"tunnel\": \"${tunnel_name:-none}\",
    \"subnet\": \"10.200.${SUBNET_NUM}.0/24\",
    \"gateway\": \"10.200.${SUBNET_NUM}.1\",
    \"created\": \"$(date -Iseconds)\"
}
NETEOF
    "
    
    echo "Isolated network wg-${network_name} created successfully"
    echo "Subnet: 10.200.${SUBNET_NUM}.0/24"
    echo ""
    echo "To run a container in this isolated network:"
    echo "  docker run --network=wg-${network_name} <image>"
    
    return 0
}

list_isolated_networks() {
    echo "=== Isolated Networks ==="
    echo ""
    
    # List Docker networks with WireGuard label
    local networks=$(docker network ls --filter "label=wireguard.network=true" --format "{{.Name}}")
    
    if [[ -z "$networks" ]]; then
        echo "No isolated networks found"
        return 0
    fi
    
    echo "Network            Tunnel            Subnet              Containers"
    echo "-------            ------            ------              ----------"
    
    for network in $networks; do
        # Get network details
        local network_info=$(docker network inspect "$network" 2>/dev/null)
        
        if [[ -n "$network_info" ]]; then
            local tunnel=$(echo "$network_info" | jq -r '.[0].Labels."wireguard.tunnel" // "none"')
            local subnet=$(echo "$network_info" | jq -r '.[0].IPAM.Config[0].Subnet // "unknown"')
            local container_count=$(echo "$network_info" | jq -r '.[0].Containers | length')
            
            # Remove wg- prefix for display
            local display_name="${network#wg-}"
            
            printf "%-18s %-17s %-19s %s\n" "$display_name" "$tunnel" "$subnet" "$container_count"
        fi
    done
    
    return 0
}

delete_isolated_network() {
    local network_name="${1:-}"
    
    if [[ -z "$network_name" ]]; then
        echo "Error: Network name required" >&2
        echo "Usage: resource-wireguard namespace delete <network-name>" >&2
        return 1
    fi
    
    local docker_network="wg-${network_name}"
    
    echo "Deleting isolated network: $docker_network"
    
    # Check if network exists
    if ! docker network ls | grep -q "$docker_network"; then
        echo "Error: Network $docker_network does not exist" >&2
        return 1
    fi
    
    # Check for connected containers
    local containers=$(docker network inspect "$docker_network" 2>/dev/null | jq -r '.[0].Containers | length')
    if [[ "$containers" -gt 0 ]]; then
        echo "Error: Network has $containers connected container(s)" >&2
        echo "Disconnect all containers before deleting the network" >&2
        return 1
    fi
    
    # Delete the network
    docker network rm "$docker_network" || {
        echo "Error: Failed to delete network" >&2
        return 1
    }
    
    # Remove config file from container
    docker exec "$CONTAINER_NAME" rm -f "/config/networks/${network_name}.json" 2>/dev/null || true
    
    echo "Isolated network $docker_network deleted successfully"
    return 0
}

connect_to_network() {
    local container_name="${1:-}"
    local network_name="${2:-}"
    
    if [[ -z "$container_name" ]] || [[ -z "$network_name" ]]; then
        echo "Error: Container and network names required" >&2
        echo "Usage: resource-wireguard namespace connect <container> <network>" >&2
        return 1
    fi
    
    local docker_network="wg-${network_name}"
    
    # Check if network exists
    if ! docker network ls | grep -q "$docker_network"; then
        echo "Error: Network $docker_network does not exist" >&2
        return 1
    fi
    
    # Connect container to network
    docker network connect "$docker_network" "$container_name" || {
        echo "Error: Failed to connect container to network" >&2
        return 1
    }
    
    echo "Container $container_name connected to isolated network $docker_network"
    return 0
}

network_isolation_status() {
    local network_name="${1:-}"
    
    if [[ -z "$network_name" ]]; then
        echo "Error: Network name required" >&2
        echo "Usage: resource-wireguard namespace status <network-name>" >&2
        return 1
    fi
    
    local docker_network="wg-${network_name}"
    
    echo "=== Network Status: $network_name ==="
    echo ""
    
    # Check if network exists
    if ! docker network ls | grep -q "$docker_network"; then
        echo "Error: Network $docker_network does not exist" >&2
        return 1
    fi
    
    # Get network details
    local network_info=$(docker network inspect "$docker_network" 2>/dev/null)
    
    if [[ -z "$network_info" ]]; then
        echo "Error: Failed to get network information" >&2
        return 1
    fi
    
    echo "Docker Network: $docker_network"
    echo "Subnet: $(echo "$network_info" | jq -r '.[0].IPAM.Config[0].Subnet')"
    echo "Gateway: $(echo "$network_info" | jq -r '.[0].IPAM.Config[0].Gateway')"
    echo "WireGuard Tunnel: $(echo "$network_info" | jq -r '.[0].Labels."wireguard.tunnel" // "none"')"
    echo ""
    
    echo "Connected Containers:"
    local containers=$(echo "$network_info" | jq -r '.[0].Containers')
    if [[ "$containers" == "{}" ]]; then
        echo "  None"
    else
        echo "$containers" | jq -r 'to_entries[] | "  \(.value.Name): \(.value.IPv4Address)"'
    fi
    
    # Show config if available from container
    if docker exec "$CONTAINER_NAME" test -f "/config/networks/${network_name}.json" 2>/dev/null; then
        echo ""
        echo "Configuration:"
        docker exec "$CONTAINER_NAME" jq '.' "/config/networks/${network_name}.json" | sed 's/^/  /'
    fi
    
    return 0
}

# ====================
# Multi-Interface Management
# ====================
handle_interface_command() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        create)
            create_interface "$@"
            ;;
        delete)
            delete_interface "$@"
            ;;
        list)
            list_interfaces "$@"
            ;;
        config)
            configure_interface "$@"
            ;;
        status)
            interface_status "$@"
            ;;
        *)
            echo "Error: Unknown interface subcommand: $subcommand" >&2
            echo "Usage: resource-wireguard interface {create|delete|list|config|status}" >&2
            exit 1
            ;;
    esac
}

create_interface() {
    local interface_name="${1:-}"
    local port="${2:-}"
    
    if [[ -z "$interface_name" ]]; then
        echo "Error: Interface name required" >&2
        echo "Usage: resource-wireguard interface create <name> [port]" >&2
        return 1
    fi
    
    # Check if container is running
    if ! docker inspect "$CONTAINER_NAME" &>/dev/null; then
        echo "Error: WireGuard container is not running" >&2
        return 1
    fi
    
    echo "Creating WireGuard interface: $interface_name"
    
    # Auto-assign port if not provided
    if [[ -z "$port" ]]; then
        # Find next available port starting from 51821
        port=51821
        while docker exec "$CONTAINER_NAME" test -f "/config/${interface_name}_port_${port}.conf" 2>/dev/null; do
            port=$((port + 1))
        done
    fi
    
    # Generate keys for the interface
    local private_key=$(docker exec "$CONTAINER_NAME" wg genkey)
    local public_key=$(echo "$private_key" | docker exec -i "$CONTAINER_NAME" wg pubkey)
    
    # Calculate unique subnet for this interface
    local interface_id=$(echo "$interface_name" | cksum | cut -d' ' -f1)
    local subnet_num=$((interface_id % 250 + 1))
    local interface_subnet="10.${subnet_num}.0.0/24"
    
    # Create interface configuration
    docker exec "$CONTAINER_NAME" sh -c "
        cat > /config/${interface_name}.conf << EOF
[Interface]
Address = ${interface_subnet%.*}.1/24
ListenPort = $port
PrivateKey = $private_key
PostUp = iptables -A FORWARD -i %i -j ACCEPT; iptables -A FORWARD -o %i -j ACCEPT; iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
PostDown = iptables -D FORWARD -i %i -j ACCEPT; iptables -D FORWARD -o %i -j ACCEPT; iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE

# Peers will be added here
EOF
        
        # Create metadata file
        cat > /config/interfaces/${interface_name}.json << JSON
{
    \"name\": \"$interface_name\",
    \"port\": $port,
    \"subnet\": \"$interface_subnet\",
    \"public_key\": \"$public_key\",
    \"created\": \"$(date -Iseconds)\",
    \"status\": \"configured\",
    \"peers\": []
}
JSON
        
        # Create the directory if it doesn't exist
        mkdir -p /config/interfaces
        
        # Bring up the interface
        wg-quick up /config/${interface_name}.conf 2>/dev/null || {
            # Try alternative method
            ip link add $interface_name type wireguard
            ip addr add ${interface_subnet%.*}.1/24 dev $interface_name
            wg setconf $interface_name /config/${interface_name}.conf
            ip link set $interface_name up
        }
    " || {
        echo "Error: Failed to create interface" >&2
        return 1
    }
    
    echo "Interface $interface_name created successfully"
    echo "  Port: $port"
    echo "  Subnet: $interface_subnet"
    echo "  Public Key: $public_key"
    
    return 0
}

delete_interface() {
    local interface_name="${1:-}"
    
    if [[ -z "$interface_name" ]]; then
        echo "Error: Interface name required" >&2
        echo "Usage: resource-wireguard interface delete <name>" >&2
        return 1
    fi
    
    echo "Deleting WireGuard interface: $interface_name"
    
    # Check if interface exists
    if ! docker exec "$CONTAINER_NAME" test -f "/config/${interface_name}.conf" 2>/dev/null; then
        echo "Error: Interface $interface_name does not exist" >&2
        return 1
    fi
    
    # Bring down the interface
    docker exec "$CONTAINER_NAME" sh -c "
        # Try wg-quick first
        wg-quick down /config/${interface_name}.conf 2>/dev/null || {
            # Fallback to manual removal
            ip link del $interface_name 2>/dev/null || true
        }
        
        # Backup configuration before deletion
        mkdir -p /config/backups/interfaces
        cp /config/${interface_name}.conf /config/backups/interfaces/${interface_name}_$(date +%Y%m%d_%H%M%S).conf
        
        # Remove configuration files
        rm -f /config/${interface_name}.conf
        rm -f /config/interfaces/${interface_name}.json
    " || {
        echo "Error: Failed to delete interface" >&2
        return 1
    }
    
    echo "Interface $interface_name deleted successfully"
    echo "Configuration backed up to /config/backups/interfaces/"
    
    return 0
}

list_interfaces() {
    local format="${1:---table}"
    
    echo "WireGuard Interfaces"
    echo "===================="
    
    # Check running interfaces
    local running_interfaces=$(docker exec "$CONTAINER_NAME" sh -c "
        wg show interfaces 2>/dev/null || echo ''
    ")
    
    if [[ "$format" == "--json" ]]; then
        # JSON output
        docker exec "$CONTAINER_NAME" sh -c "
            interfaces='[]'
            for config in /config/*.conf; do
                if [[ -f \"\$config\" ]] && [[ \"\$config\" != \"/config/wg0.conf\" ]]; then
                    name=\$(basename \"\$config\" .conf)
                    if [[ -f /config/interfaces/\${name}.json ]]; then
                        interface_data=\$(cat /config/interfaces/\${name}.json)
                    else
                        interface_data='{\"name\": \"'\$name'\", \"status\": \"unknown\"}'
                    fi
                    
                    # Check if running
                    if echo \"$running_interfaces\" | grep -q \"\$name\"; then
                        interface_data=\$(echo \"\$interface_data\" | jq '.status = \"running\"')
                    else
                        interface_data=\$(echo \"\$interface_data\" | jq '.status = \"stopped\"')
                    fi
                    
                    interfaces=\$(echo \"\$interfaces\" | jq \". + [\$interface_data]\")
                fi
            done
            echo \"\$interfaces\" | jq '.'
        "
    else
        # Table output
        if [[ -z "$running_interfaces" ]]; then
            echo "No interfaces currently running"
        else
            echo "Running interfaces: $running_interfaces"
        fi
        
        echo ""
        echo "Configured Interfaces:"
        docker exec "$CONTAINER_NAME" sh -c "
            found=0
            for config in /config/*.conf; do
                if [[ -f \"\$config\" ]]; then
                    name=\$(basename \"\$config\" .conf)
                    found=1
                    
                    # Get metadata if available
                    if [[ -f /config/interfaces/\${name}.json ]]; then
                        port=\$(jq -r '.port' /config/interfaces/\${name}.json)
                        subnet=\$(jq -r '.subnet' /config/interfaces/\${name}.json)
                        peer_count=\$(jq '.peers | length' /config/interfaces/\${name}.json)
                    else
                        port='N/A'
                        subnet='N/A'
                        peer_count=0
                    fi
                    
                    # Check if running
                    if echo \"$running_interfaces\" | grep -q \"\$name\"; then
                        status='running'
                    else
                        status='stopped'
                    fi
                    
                    echo \"  - \$name: port=\$port, subnet=\$subnet, peers=\$peer_count, status=\$status\"
                fi
            done
            
            if [[ \$found -eq 0 ]]; then
                echo '  None'
            fi
        "
    fi
    
    return 0
}

configure_interface() {
    local interface_name="${1:-}"
    local action="${2:-}"
    shift 2 || true
    
    if [[ -z "$interface_name" ]] || [[ -z "$action" ]]; then
        echo "Error: Interface name and action required" >&2
        echo "Usage: resource-wireguard interface config <name> {add-peer|remove-peer|update} [options]" >&2
        return 1
    fi
    
    case "$action" in
        add-peer)
            local peer_public_key="${1:-}"
            local peer_endpoint="${2:-}"
            local peer_allowed_ips="${3:-0.0.0.0/0}"
            
            if [[ -z "$peer_public_key" ]]; then
                echo "Error: Peer public key required" >&2
                return 1
            fi
            
            echo "Adding peer to interface $interface_name"
            
            docker exec "$CONTAINER_NAME" sh -c "
                # Add peer to config
                cat >> /config/${interface_name}.conf << EOF

[Peer]
PublicKey = $peer_public_key
AllowedIPs = $peer_allowed_ips
EOF
                
                if [[ -n \"$peer_endpoint\" ]]; then
                    echo \"Endpoint = $peer_endpoint\" >> /config/${interface_name}.conf
                fi
                
                # Update metadata
                if [[ -f /config/interfaces/${interface_name}.json ]]; then
                    jq '.peers += [{\"public_key\": \"$peer_public_key\", \"endpoint\": \"$peer_endpoint\", \"allowed_ips\": \"$peer_allowed_ips\"}]' \
                        /config/interfaces/${interface_name}.json > /tmp/interface.json
                    mv /tmp/interface.json /config/interfaces/${interface_name}.json
                fi
                
                # Apply configuration if interface is running
                if ip link show $interface_name &>/dev/null; then
                    wg setconf $interface_name /config/${interface_name}.conf
                fi
            " || {
                echo "Error: Failed to add peer" >&2
                return 1
            }
            
            echo "Peer added successfully"
            ;;
            
        remove-peer)
            local peer_public_key="${1:-}"
            
            if [[ -z "$peer_public_key" ]]; then
                echo "Error: Peer public key required" >&2
                return 1
            fi
            
            echo "Removing peer from interface $interface_name"
            
            docker exec "$CONTAINER_NAME" sh -c "
                # Remove peer from running interface
                if ip link show $interface_name &>/dev/null; then
                    wg set $interface_name peer $peer_public_key remove
                fi
                
                # Update config file
                # This is complex, so we'll rebuild the config without the peer
                cp /config/${interface_name}.conf /tmp/interface_backup.conf
                awk '/^\\[Peer\\]/{p=1} p && /^PublicKey = $peer_public_key/{p=0;next} !p || /^\\[/{p=1}1' \
                    /tmp/interface_backup.conf > /config/${interface_name}.conf
                
                # Update metadata
                if [[ -f /config/interfaces/${interface_name}.json ]]; then
                    jq '.peers |= map(select(.public_key != \"$peer_public_key\"))' \
                        /config/interfaces/${interface_name}.json > /tmp/interface.json
                    mv /tmp/interface.json /config/interfaces/${interface_name}.json
                fi
            " || {
                echo "Error: Failed to remove peer" >&2
                return 1
            }
            
            echo "Peer removed successfully"
            ;;
            
        update)
            echo "Reloading interface configuration: $interface_name"
            
            docker exec "$CONTAINER_NAME" sh -c "
                if ip link show $interface_name &>/dev/null; then
                    wg setconf $interface_name /config/${interface_name}.conf
                else
                    echo 'Interface is not running. Start it with: resource-wireguard interface start $interface_name'
                fi
            " || {
                echo "Error: Failed to update configuration" >&2
                return 1
            }
            
            echo "Configuration updated"
            ;;
            
        *)
            echo "Error: Unknown config action: $action" >&2
            return 1
            ;;
    esac
    
    return 0
}

interface_status() {
    local interface_name="${1:-}"
    
    if [[ -z "$interface_name" ]]; then
        echo "Error: Interface name required" >&2
        echo "Usage: resource-wireguard interface status <name>" >&2
        return 1
    fi
    
    echo "Interface Status: $interface_name"
    echo "================================"
    
    # Check if interface exists
    if ! docker exec "$CONTAINER_NAME" test -f "/config/${interface_name}.conf" 2>/dev/null; then
        echo "Status: Not configured"
        return 1
    fi
    
    # Check if interface is running
    local is_running=$(docker exec "$CONTAINER_NAME" sh -c "
        if ip link show $interface_name &>/dev/null; then
            echo 'yes'
        else
            echo 'no'
        fi
    ")
    
    if [[ "$is_running" == "no" ]]; then
        echo "Status: Configured but not running"
        echo "Start with: resource-wireguard interface start $interface_name"
        return 0
    fi
    
    # Show detailed status
    docker exec "$CONTAINER_NAME" sh -c "
        echo 'Status: Running'
        echo ''
        
        # Show interface details
        wg show $interface_name
        
        # Show metadata if available
        if [[ -f /config/interfaces/${interface_name}.json ]]; then
            echo ''
            echo 'Configuration Metadata:'
            jq '.' /config/interfaces/${interface_name}.json | sed 's/^/  /'
        fi
        
        # Show traffic stats
        echo ''
        echo 'Traffic Statistics:'
        ip -s link show $interface_name | grep -A 1 'RX:\\|TX:' | sed 's/^/  /'
    "
    
    return 0
}

# ====================
# Monitoring Dashboard
# ====================
handle_monitor_command() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        start)
            start_monitor_dashboard "$@"
            ;;
        stop)
            stop_monitor_dashboard "$@"
            ;;
        status)
            monitor_dashboard_status "$@"
            ;;
        *)
            echo "Error: Unknown monitor subcommand: $subcommand" >&2
            echo "Usage: resource-wireguard monitor {start|stop|status}" >&2
            exit 1
            ;;
    esac
}

start_monitor_dashboard() {
    local port="${1:-8080}"
    
    echo "Starting WireGuard monitoring dashboard on port $port"
    
    # Check if container is running
    if ! docker inspect "$CONTAINER_NAME" &>/dev/null; then
        echo "Error: WireGuard container is not running" >&2
        return 1
    fi
    
    # Create dashboard HTML
    docker exec "$CONTAINER_NAME" sh -c "
        mkdir -p /config/dashboard
        
        cat > /config/dashboard/index.html << 'HTML'
<!DOCTYPE html>
<html lang=\"en\">
<head>
    <meta charset=\"UTF-8\">
    <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">
    <title>WireGuard Monitoring Dashboard</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 0; padding: 20px; background: #f5f5f5; }
        h1 { color: #333; border-bottom: 2px solid #ff9000; padding-bottom: 10px; }
        .container { max-width: 1200px; margin: 0 auto; }
        .card { background: white; border-radius: 8px; padding: 20px; margin-bottom: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .card h2 { margin-top: 0; color: #666; font-size: 18px; }
        .stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 15px; }
        .stat { background: #f8f9fa; padding: 15px; border-radius: 6px; }
        .stat-label { color: #666; font-size: 12px; text-transform: uppercase; }
        .stat-value { font-size: 24px; font-weight: bold; color: #333; margin-top: 5px; }
        .interface { margin-bottom: 15px; padding: 15px; background: #f8f9fa; border-radius: 6px; }
        .interface-name { font-weight: bold; color: #333; }
        .status { display: inline-block; padding: 3px 8px; border-radius: 4px; font-size: 12px; }
        .status.running { background: #28a745; color: white; }
        .status.stopped { background: #dc3545; color: white; }
        .peer { margin-left: 20px; padding: 10px; background: white; border-left: 3px solid #ff9000; margin-top: 10px; }
        .refresh-btn { background: #ff9000; color: white; border: none; padding: 10px 20px; border-radius: 4px; cursor: pointer; }
        .refresh-btn:hover { background: #e67e00; }
        #last-update { color: #666; font-size: 12px; margin-top: 10px; }
    </style>
</head>
<body>
    <div class=\"container\">
        <h1>🔒 WireGuard Monitoring Dashboard</h1>
        <button class=\"refresh-btn\" onclick=\"refreshData()\">Refresh</button>
        <span id=\"last-update\"></span>
        
        <div class=\"card\">
            <h2>System Overview</h2>
            <div class=\"stats\" id=\"system-stats\">
                <div class=\"stat\">
                    <div class=\"stat-label\">Total Interfaces</div>
                    <div class=\"stat-value\" id=\"total-interfaces\">-</div>
                </div>
                <div class=\"stat\">
                    <div class=\"stat-label\">Active Tunnels</div>
                    <div class=\"stat-value\" id=\"active-tunnels\">-</div>
                </div>
                <div class=\"stat\">
                    <div class=\"stat-label\">Total Peers</div>
                    <div class=\"stat-value\" id=\"total-peers\">-</div>
                </div>
                <div class=\"stat\">
                    <div class=\"stat-label\">Data Transferred</div>
                    <div class=\"stat-value\" id=\"data-transferred\">-</div>
                </div>
            </div>
        </div>
        
        <div class=\"card\">
            <h2>Interfaces</h2>
            <div id=\"interfaces-list\"></div>
        </div>
        
        <div class=\"card\">
            <h2>Recent Activity</h2>
            <div id=\"activity-log\"></div>
        </div>
    </div>
    
    <script>
        async function fetchStatus() {
            try {
                const response = await fetch('/api/status');
                const data = await response.json();
                return data;
            } catch (e) {
                console.error('Failed to fetch status:', e);
                return null;
            }
        }
        
        function formatBytes(bytes) {
            if (bytes === 0) return '0 B';
            const k = 1024;
            const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
        }
        
        async function refreshData() {
            const data = await fetchStatus();
            if (!data) return;
            
            // Update system stats
            document.getElementById('total-interfaces').textContent = data.interfaces.length;
            document.getElementById('active-tunnels').textContent = data.active_count;
            document.getElementById('total-peers').textContent = data.total_peers;
            document.getElementById('data-transferred').textContent = formatBytes(data.total_bytes);
            
            // Update interfaces list
            const interfacesList = document.getElementById('interfaces-list');
            interfacesList.innerHTML = '';
            
            data.interfaces.forEach(iface => {
                const div = document.createElement('div');
                div.className = 'interface';
                div.innerHTML = \`
                    <div class=\"interface-name\">\${iface.name} <span class=\"status \${iface.status}\">\${iface.status}</span></div>
                    <div>Port: \${iface.port} | Subnet: \${iface.subnet}</div>
                    <div>Peers: \${iface.peers.length}</div>
                \`;
                
                if (iface.peers.length > 0) {
                    iface.peers.forEach(peer => {
                        const peerDiv = document.createElement('div');
                        peerDiv.className = 'peer';
                        peerDiv.innerHTML = \`
                            <div>Peer: \${peer.public_key.substring(0, 8)}...</div>
                            <div>Endpoint: \${peer.endpoint || 'N/A'}</div>
                            <div>Transfer: ↑ \${formatBytes(peer.tx_bytes)} ↓ \${formatBytes(peer.rx_bytes)}</div>
                        \`;
                        div.appendChild(peerDiv);
                    });
                }
                
                interfacesList.appendChild(div);
            });
            
            // Update activity log
            const activityLog = document.getElementById('activity-log');
            activityLog.innerHTML = '';
            
            data.recent_activity.forEach(activity => {
                const div = document.createElement('div');
                div.style.padding = '10px';
                div.style.borderBottom = '1px solid #eee';
                div.innerHTML = \`<strong>\${activity.time}</strong>: \${activity.message}\`;
                activityLog.appendChild(div);
            });
            
            // Update timestamp
            document.getElementById('last-update').textContent = 'Last updated: ' + new Date().toLocaleTimeString();
        }
        
        // Initial load and auto-refresh every 5 seconds
        refreshData();
        setInterval(refreshData, 5000);
    </script>
</body>
</html>
HTML
        
        # Create API endpoint script
        cat > /config/dashboard/api.sh << 'API'
#!/bin/sh
# Simple API endpoint for dashboard

get_status() {
    # Get interfaces
    interfaces='[]'
    total_peers=0
    active_count=0
    total_bytes=0
    
    for config in /config/*.conf; do
        if [ -f \"\$config\" ]; then
            name=\$(basename \"\$config\" .conf)
            
            # Check if running
            if ip link show \$name 2>/dev/null | grep -q UP; then
                status=\"running\"
                active_count=\$((active_count + 1))
                
                # Get peer count
                peer_count=\$(wg show \$name peers 2>/dev/null | wc -l)
                total_peers=\$((total_peers + peer_count))
                
                # Get traffic stats
                tx_bytes=\$(wg show \$name transfer 2>/dev/null | awk '{s+=\$2} END {print s}')
                rx_bytes=\$(wg show \$name transfer 2>/dev/null | awk '{s+=\$3} END {print s}')
                total_bytes=\$((total_bytes + tx_bytes + rx_bytes))
            else
                status=\"stopped\"
                peer_count=0
            fi
            
            # Get metadata
            if [ -f \"/config/interfaces/\${name}.json\" ]; then
                port=\$(jq -r '.port' /config/interfaces/\${name}.json)
                subnet=\$(jq -r '.subnet' /config/interfaces/\${name}.json)
            else
                port=\"N/A\"
                subnet=\"N/A\"
            fi
            
            interface_json=\"{\\\"name\\\": \\\"\$name\\\", \\\"status\\\": \\\"\$status\\\", \\\"port\\\": \\\"\$port\\\", \\\"subnet\\\": \\\"\$subnet\\\", \\\"peers\\\": []}\"
            interfaces=\$(echo \"\$interfaces\" | jq \". + [\$interface_json]\")
        fi
    done
    
    # Get recent activity (last 5 log entries)
    recent_activity='[]'
    if [ -f /var/log/wireguard.log ]; then
        tail -5 /var/log/wireguard.log | while read line; do
            activity=\"{\\\"time\\\": \\\"\$(date +%H:%M:%S)\\\", \\\"message\\\": \\\"\$line\\\"}\"
            recent_activity=\$(echo \"\$recent_activity\" | jq \". + [\$activity]\")
        done
    fi
    
    # Output JSON
    cat << JSON
{
    \"interfaces\": \$interfaces,
    \"total_peers\": \$total_peers,
    \"active_count\": \$active_count,
    \"total_bytes\": \${total_bytes:-0},
    \"recent_activity\": \$recent_activity
}
JSON
}

# Handle request
if [ \"\$REQUEST_METHOD\" = \"GET\" ] && [ \"\$REQUEST_URI\" = \"/api/status\" ]; then
    echo \"HTTP/1.1 200 OK\"
    echo \"Content-Type: application/json\"
    echo \"Access-Control-Allow-Origin: *\"
    echo \"\"
    get_status
else
    echo \"HTTP/1.1 404 Not Found\"
    echo \"\"
    echo \"Not Found\"
fi
API
        chmod +x /config/dashboard/api.sh
        
        # Start simple web server with busybox httpd
        if pgrep -f 'httpd.*dashboard' > /dev/null; then
            pkill -f 'httpd.*dashboard'
        fi
        
        cd /config/dashboard
        busybox httpd -p $port -h /config/dashboard &
        echo \$! > /var/run/wireguard-dashboard.pid
        
        # Create CGI wrapper for API
        mkdir -p /config/dashboard/cgi-bin
        cat > /config/dashboard/cgi-bin/api << 'CGI'
#!/bin/sh
export REQUEST_METHOD=\"GET\"
export REQUEST_URI=\"/api/status\"
/config/dashboard/api.sh
CGI
        chmod +x /config/dashboard/cgi-bin/api
    " || {
        echo "Error: Failed to start monitoring dashboard" >&2
        return 1
    }
    
    echo "Monitoring dashboard started on http://localhost:$port"
    echo "Access the dashboard in your browser to view WireGuard status"
    
    return 0
}

stop_monitor_dashboard() {
    echo "Stopping WireGuard monitoring dashboard"
    
    docker exec "$CONTAINER_NAME" sh -c "
        if [ -f /var/run/wireguard-dashboard.pid ]; then
            kill \$(cat /var/run/wireguard-dashboard.pid) 2>/dev/null || true
            rm -f /var/run/wireguard-dashboard.pid
            echo 'Dashboard stopped'
        else
            echo 'Dashboard is not running'
        fi
    " || {
        echo "Warning: Issues stopping dashboard" >&2
    }
    
    return 0
}

monitor_dashboard_status() {
    echo "Monitoring Dashboard Status"
    echo "=========================="
    
    local status=$(docker exec "$CONTAINER_NAME" sh -c "
        if [ -f /var/run/wireguard-dashboard.pid ]; then
            if kill -0 \$(cat /var/run/wireguard-dashboard.pid) 2>/dev/null; then
                echo 'running'
            else
                echo 'stopped'
            fi
        else
            echo 'stopped'
        fi
    " 2>/dev/null || echo "error")
    
    if [[ "$status" == "running" ]]; then
        echo "Status: Running"
        echo "URL: http://localhost:8080 (or configured port)"
        echo "Stop with: resource-wireguard monitor stop"
    else
        echo "Status: Stopped"
        echo "Start with: resource-wireguard monitor start [port]"
    fi
    
    return 0
}

# ====================
# Auto-Discovery Management (mDNS/DNS-SD)
# ====================
handle_discovery_command() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        enable)
            enable_discovery "$@"
            ;;
        disable)
            disable_discovery "$@"
            ;;
        scan)
            scan_peers "$@"
            ;;
        advertise)
            advertise_peer "$@"
            ;;
        status)
            discovery_status "$@"
            ;;
        *)
            echo "Error: Unknown discovery subcommand: $subcommand" >&2
            echo "Usage: resource-wireguard discovery {enable|disable|scan|advertise|status}" >&2
            exit 1
            ;;
    esac
}

enable_discovery() {
    local tunnel_name="${1:-wg0}"
    local service_name="${2:-vrooli-wg}"
    
    echo "Enabling auto-discovery for tunnel: $tunnel_name"
    
    # Check if container is running
    if ! docker inspect "$CONTAINER_NAME" &>/dev/null; then
        echo "Error: WireGuard container is not running" >&2
        return 1
    fi
    
    # Install avahi-daemon in the container for mDNS
    docker exec "$CONTAINER_NAME" sh -c "
        apk add --no-cache avahi avahi-tools dbus 2>/dev/null || true
        
        # Start dbus if not running
        if ! pgrep dbus-daemon > /dev/null; then
            dbus-daemon --system --fork
        fi
        
        # Configure avahi
        cat > /etc/avahi/avahi-daemon.conf << 'AVAHI'
[server]
host-name=$service_name
domain-name=local
use-ipv4=yes
use-ipv6=no
allow-interfaces=eth0
deny-interfaces=wg0

[wide-area]
enable-wide-area=yes

[publish]
publish-addresses=yes
publish-hinfo=yes
publish-workstation=no
publish-domain=yes
AVAHI
        
        # Start avahi-daemon
        if ! pgrep avahi-daemon > /dev/null; then
            avahi-daemon -D
        fi
    " || {
        echo "Error: Failed to setup mDNS discovery" >&2
        return 1
    }
    
    # Create discovery config
    local discovery_config="/config/discovery/${tunnel_name}.json"
    docker exec "$CONTAINER_NAME" sh -c "
        mkdir -p /config/discovery
        cat > '$discovery_config' << 'CONFIG'
{
    \"tunnel\": \"$tunnel_name\",
    \"service_name\": \"$service_name\",
    \"enabled\": true,
    \"scan_interval\": 60,
    \"last_scan\": \"$(date -Iseconds)\",
    \"discovered_peers\": []
}
CONFIG
    "
    
    # Register the WireGuard service with mDNS
    local public_key=$(docker exec "$CONTAINER_NAME" wg show "$tunnel_name" public-key 2>/dev/null || echo "pending")
    local endpoint_port="${WIREGUARD_PORT:-51820}"
    
    docker exec "$CONTAINER_NAME" sh -c "
        # Register service with mDNS
        avahi-publish-service \
            '${service_name}' \
            _wireguard._udp \
            $endpoint_port \
            \"public_key=$public_key\" \
            \"tunnel=$tunnel_name\" \
            \"version=1.0\" &
        
        # Save PID for cleanup
        echo \$! > /var/run/avahi-publish-${tunnel_name}.pid
    " || {
        echo "Warning: Could not register mDNS service" >&2
    }
    
    echo "Auto-discovery enabled for tunnel $tunnel_name"
    echo "Service advertised as: ${service_name}._wireguard._udp.local"
    return 0
}

disable_discovery() {
    local tunnel_name="${1:-wg0}"
    
    echo "Disabling auto-discovery for tunnel: $tunnel_name"
    
    # Stop avahi publishing
    docker exec "$CONTAINER_NAME" sh -c "
        if [[ -f /var/run/avahi-publish-${tunnel_name}.pid ]]; then
            kill \$(cat /var/run/avahi-publish-${tunnel_name}.pid) 2>/dev/null || true
            rm -f /var/run/avahi-publish-${tunnel_name}.pid
        fi
        
        # Update config
        if [[ -f /config/discovery/${tunnel_name}.json ]]; then
            jq '.enabled = false' /config/discovery/${tunnel_name}.json > /tmp/discovery.json
            mv /tmp/discovery.json /config/discovery/${tunnel_name}.json
        fi
    " || {
        echo "Warning: Issues disabling discovery" >&2
    }
    
    echo "Auto-discovery disabled for tunnel $tunnel_name"
    return 0
}

scan_peers() {
    local timeout="${1:-5}"
    
    echo "Scanning for WireGuard peers (timeout: ${timeout}s)..."
    
    # Check if discovery is enabled
    docker exec "$CONTAINER_NAME" sh -c "
        if ! pgrep avahi-daemon > /dev/null; then
            echo 'Error: mDNS discovery is not enabled' >&2
            echo 'Run: resource-wireguard discovery enable' >&2
            exit 1
        fi
    " || return 1
    
    # Scan for _wireguard._udp services
    local discovered_peers=$(docker exec "$CONTAINER_NAME" timeout "$timeout" sh -c "
        avahi-browse -t -r -p _wireguard._udp 2>/dev/null | grep '^=' | while IFS=';' read -r iface proto name type domain hostname addr port txt; do
            if [[ -n \"\$hostname\" ]] && [[ -n \"\$addr\" ]] && [[ -n \"\$port\" ]]; then
                echo \"{\\\"hostname\\\": \\\"\$hostname\\\", \\\"address\\\": \\\"\$addr\\\", \\\"port\\\": \\\"\$port\\\", \\\"txt\\\": \\\"\$txt\\\"}\"
            fi
        done
    " 2>/dev/null || echo "")
    
    if [[ -z "$discovered_peers" ]]; then
        echo "No WireGuard peers discovered"
        return 0
    fi
    
    echo "Discovered peers:"
    echo "$discovered_peers" | while read -r peer; do
        if [[ -n "$peer" ]]; then
            hostname=$(echo "$peer" | jq -r '.hostname')
            address=$(echo "$peer" | jq -r '.address')
            port=$(echo "$peer" | jq -r '.port')
            echo "  - $hostname at $address:$port"
        fi
    done
    
    # Update discovery config with found peers
    docker exec "$CONTAINER_NAME" sh -c "
        for tunnel_json in /config/discovery/*.json; do
            if [[ -f \"\$tunnel_json\" ]]; then
                peers='[]'
                while read -r peer; do
                    if [[ -n \"\$peer\" ]]; then
                        peers=\$(echo \"\$peers\" | jq \". + [\$peer]\")
                    fi
                done <<< \"$discovered_peers\"
                
                jq \".discovered_peers = \$peers | .last_scan = \\\"$(date -Iseconds)\\\"\" \"\$tunnel_json\" > /tmp/discovery.json
                mv /tmp/discovery.json \"\$tunnel_json\"
            fi
        done
    " 2>/dev/null || true
    
    return 0
}

advertise_peer() {
    local tunnel_name="${1:-wg0}"
    local custom_name="${2:-}"
    
    echo "Advertising peer information for tunnel: $tunnel_name"
    
    # Get tunnel information
    local public_key=$(docker exec "$CONTAINER_NAME" wg show "$tunnel_name" public-key 2>/dev/null)
    if [[ -z "$public_key" ]]; then
        echo "Error: Tunnel $tunnel_name not found" >&2
        return 1
    fi
    
    local service_name="${custom_name:-vrooli-wg-$tunnel_name}"
    local endpoint_port="${WIREGUARD_PORT:-51820}"
    
    # Re-advertise or update advertisement
    docker exec "$CONTAINER_NAME" sh -c "
        # Stop existing advertisement if any
        if [[ -f /var/run/avahi-publish-${tunnel_name}.pid ]]; then
            kill \$(cat /var/run/avahi-publish-${tunnel_name}.pid) 2>/dev/null || true
        fi
        
        # Start new advertisement
        avahi-publish-service \
            '${service_name}' \
            _wireguard._udp \
            $endpoint_port \
            \"public_key=$public_key\" \
            \"tunnel=$tunnel_name\" \
            \"version=1.0\" \
            \"updated=$(date -Iseconds)\" &
        
        echo \$! > /var/run/avahi-publish-${tunnel_name}.pid
    " || {
        echo "Error: Failed to advertise service" >&2
        return 1
    }
    
    echo "Peer advertised as: ${service_name}._wireguard._udp.local"
    echo "Public key: $public_key"
    return 0
}

discovery_status() {
    echo "Auto-Discovery Status"
    echo "===================="
    
    # Check if avahi is running
    local avahi_status=$(docker exec "$CONTAINER_NAME" sh -c "
        if pgrep avahi-daemon > /dev/null; then
            echo 'running'
        else
            echo 'stopped'
        fi
    " 2>/dev/null || echo "error")
    
    echo "mDNS Service: $avahi_status"
    
    if [[ "$avahi_status" != "running" ]]; then
        echo "Discovery is not enabled"
        echo "Run: resource-wireguard discovery enable"
        return 0
    fi
    
    # Show configured discoveries
    echo ""
    echo "Configured Tunnels:"
    docker exec "$CONTAINER_NAME" sh -c "
        for config in /config/discovery/*.json; do
            if [[ -f \"\$config\" ]]; then
                tunnel=\$(jq -r '.tunnel' \"\$config\")
                enabled=\$(jq -r '.enabled' \"\$config\")
                last_scan=\$(jq -r '.last_scan' \"\$config\")
                peer_count=\$(jq '.discovered_peers | length' \"\$config\")
                echo \"  - \$tunnel: enabled=\$enabled, peers=\$peer_count, last_scan=\$last_scan\"
            fi
        done
    " 2>/dev/null || echo "  None configured"
    
    # Show active advertisements
    echo ""
    echo "Active Advertisements:"
    docker exec "$CONTAINER_NAME" sh -c "
        for pidfile in /var/run/avahi-publish-*.pid; do
            if [[ -f \"\$pidfile\" ]]; then
                tunnel=\$(basename \"\$pidfile\" .pid | sed 's/avahi-publish-//')
                if kill -0 \$(cat \"\$pidfile\") 2>/dev/null; then
                    echo \"  - Tunnel \$tunnel: advertising\"
                fi
            fi
        done
    " 2>/dev/null || echo "  None active"
    
    return 0
}

# ====================
# Mesh Networking Management
# ====================
handle_mesh_command() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        create)
            create_mesh_network "$@"
            ;;
        join)
            join_mesh_network "$@"
            ;;
        leave)
            leave_mesh_network "$@"
            ;;
        status)
            mesh_network_status "$@"
            ;;
        sync)
            sync_mesh_peers "$@"
            ;;
        *)
            echo "Error: Unknown mesh subcommand: $subcommand" >&2
            echo "Usage: resource-wireguard mesh {create|join|leave|status|sync}" >&2
            exit 1
            ;;
    esac
}

create_mesh_network() {
    local mesh_name="${1:-}"
    
    if [[ -z "$mesh_name" ]]; then
        echo "Error: Mesh network name required" >&2
        echo "Usage: resource-wireguard mesh create <name>" >&2
        return 1
    fi
    
    echo "Creating mesh network: $mesh_name"
    
    # Check if container is running
    if ! docker inspect "$CONTAINER_NAME" &>/dev/null; then
        echo "Error: WireGuard container is not running" >&2
        return 1
    fi
    
    # Generate mesh configuration
    docker exec "$CONTAINER_NAME" sh -c "
        mkdir -p /config/mesh/$mesh_name
        
        # Generate unique mesh ID
        mesh_id=\$(head -c 32 /dev/urandom | base64 | tr -d '=' | head -c 16)
        
        # Store mesh metadata
        cat > /config/mesh/$mesh_name/metadata.json << JSON
{
    \"name\": \"$mesh_name\",
    \"id\": \"\$mesh_id\",
    \"created\": \"\$(date -Iseconds)\",
    \"topology\": \"full-mesh\",
    \"peers\": [],
    \"subnet\": \"10.200.\$(shuf -i 1-254 -n 1).0/24\"
}
JSON
        
        # Generate initial peer config for this node
        wg genkey > /config/mesh/$mesh_name/privatekey 2>/dev/null
        wg pubkey < /config/mesh/$mesh_name/privatekey > /config/mesh/$mesh_name/publickey 2>/dev/null
        
        # Create mesh interface configuration
        cat > /config/mesh/$mesh_name/wg-mesh.conf << CONF
[Interface]
PrivateKey = \$(cat /config/mesh/$mesh_name/privatekey)
ListenPort = \$(shuf -i 52000-53000 -n 1)
Address = \$(jq -r '.subnet' /config/mesh/$mesh_name/metadata.json | sed 's/0\/24/1\/24/')

# Mesh peers will be added here
CONF
        
        echo 'Mesh network created successfully'
        echo 'Mesh ID: '\$mesh_id
        echo 'Public Key: '\$(cat /config/mesh/$mesh_name/publickey)
        
    " || {
        echo "Error: Failed to create mesh network" >&2
        return 1
    }
    
    return 0
}

join_mesh_network() {
    local mesh_name="${1:-}"
    local peer_endpoint="${2:-}"
    
    if [[ -z "$mesh_name" ]] || [[ -z "$peer_endpoint" ]]; then
        echo "Error: Mesh name and peer endpoint required" >&2
        echo "Usage: resource-wireguard mesh join <name> <peer_endpoint>" >&2
        return 1
    fi
    
    echo "Joining mesh network: $mesh_name"
    echo "Connecting to peer: $peer_endpoint"
    
    # Check if container is running
    if ! docker inspect "$CONTAINER_NAME" &>/dev/null; then
        echo "Error: WireGuard container is not running" >&2
        return 1
    fi
    
    # Join mesh network
    docker exec "$CONTAINER_NAME" sh -c "
        mkdir -p /config/mesh/$mesh_name
        
        # Generate keys for this node if not exists
        if [ ! -f /config/mesh/$mesh_name/privatekey ]; then
            wg genkey > /config/mesh/$mesh_name/privatekey 2>/dev/null
            wg pubkey < /config/mesh/$mesh_name/privatekey > /config/mesh/$mesh_name/publickey 2>/dev/null
        fi
        
        # Parse peer endpoint (format: pubkey@host:port)
        peer_pubkey=\$(echo '$peer_endpoint' | cut -d@ -f1)
        peer_host=\$(echo '$peer_endpoint' | cut -d@ -f2)
        
        # Create initial configuration
        if [ ! -f /config/mesh/$mesh_name/wg-mesh.conf ]; then
            cat > /config/mesh/$mesh_name/wg-mesh.conf << CONF
[Interface]
PrivateKey = \$(cat /config/mesh/$mesh_name/privatekey)
ListenPort = \$(shuf -i 52000-53000 -n 1)
Address = 10.200.\$(shuf -i 1-254 -n 1).\$(shuf -i 2-254 -n 1)/24
CONF
        fi
        
        # Add peer to configuration
        cat >> /config/mesh/$mesh_name/wg-mesh.conf << PEER

[Peer]
PublicKey = \$peer_pubkey
Endpoint = \$peer_host
AllowedIPs = 10.200.0.0/16
PersistentKeepalive = 25
PEER
        
        # Store peer information
        if [ ! -f /config/mesh/$mesh_name/peers.json ]; then
            echo '[]' > /config/mesh/$mesh_name/peers.json
        fi
        
        jq '. += [{\"pubkey\": \"'\$peer_pubkey'\", \"endpoint\": \"'\$peer_host'\", \"joined\": \"'\$(date -Iseconds)'\"}]' \
           /config/mesh/$mesh_name/peers.json > /tmp/peers.json && mv /tmp/peers.json /config/mesh/$mesh_name/peers.json
        
        echo 'Joined mesh network successfully'
        echo 'Local Public Key: '\$(cat /config/mesh/$mesh_name/publickey)
        
    " || {
        echo "Error: Failed to join mesh network" >&2
        return 1
    }
    
    # Activate the mesh interface
    echo "Activating mesh interface..."
    docker exec "$CONTAINER_NAME" sh -c "
        # Bring up the mesh interface
        wg-quick up /config/mesh/$mesh_name/wg-mesh.conf 2>/dev/null || {
            echo 'Mesh interface already up or activation failed'
        }
    "
    
    return 0
}

leave_mesh_network() {
    local mesh_name="${1:-}"
    
    if [[ -z "$mesh_name" ]]; then
        echo "Error: Mesh network name required" >&2
        echo "Usage: resource-wireguard mesh leave <name>" >&2
        return 1
    fi
    
    echo "Leaving mesh network: $mesh_name"
    
    # Check if container is running
    if ! docker inspect "$CONTAINER_NAME" &>/dev/null; then
        echo "Error: WireGuard container is not running" >&2
        return 1
    fi
    
    # Leave mesh network
    docker exec "$CONTAINER_NAME" sh -c "
        if [ -f /config/mesh/$mesh_name/wg-mesh.conf ]; then
            # Bring down the interface
            wg-quick down /config/mesh/$mesh_name/wg-mesh.conf 2>/dev/null || true
            
            # Backup configuration before removal
            mkdir -p /config/mesh/backups
            cp -r /config/mesh/$mesh_name /config/mesh/backups/$mesh_name-\$(date +%Y%m%d-%H%M%S)
            
            # Remove mesh configuration
            rm -rf /config/mesh/$mesh_name
            
            echo 'Left mesh network successfully'
        else
            echo 'Not a member of mesh network: $mesh_name'
            return 1
        fi
    " || {
        echo "Error: Failed to leave mesh network" >&2
        return 1
    }
    
    return 0
}

mesh_network_status() {
    local mesh_name="${1:-}"
    
    if [[ -z "$mesh_name" ]]; then
        echo "Mesh Networks Status"
        echo "===================="
        
        # List all mesh networks
        docker exec "$CONTAINER_NAME" sh -c "
            if [ -d /config/mesh ]; then
                for mesh in /config/mesh/*/; do
                    if [ -f \"\$mesh/metadata.json\" ] || [ -f \"\$mesh/wg-mesh.conf\" ]; then
                        mesh_name=\$(basename \"\$mesh\")
                        echo \"Mesh: \$mesh_name\"
                        if [ -f \"\$mesh/metadata.json\" ]; then
                            echo \"  ID: \$(jq -r '.id' \"\$mesh/metadata.json\" 2>/dev/null || echo 'N/A')\"
                            echo \"  Subnet: \$(jq -r '.subnet' \"\$mesh/metadata.json\" 2>/dev/null || echo 'N/A')\"
                            echo \"  Topology: \$(jq -r '.topology' \"\$mesh/metadata.json\" 2>/dev/null || echo 'full-mesh')\"
                        fi
                        if [ -f \"\$mesh/peers.json\" ]; then
                            peer_count=\$(jq '. | length' \"\$mesh/peers.json\" 2>/dev/null || echo '0')
                            echo \"  Peers: \$peer_count\"
                        fi
                        # Check if interface is up
                        if ip link show wg-mesh 2>/dev/null | grep -q 'UP'; then
                            echo \"  Status: Active\"
                        else
                            echo \"  Status: Inactive\"
                        fi
                        echo
                    fi
                done
            else
                echo 'No mesh networks configured'
            fi
        " 2>/dev/null || echo "Error retrieving mesh status"
    else
        echo "Mesh Network Status: $mesh_name"
        echo "================================"
        
        docker exec "$CONTAINER_NAME" sh -c "
            if [ -d /config/mesh/$mesh_name ]; then
                # Show metadata
                if [ -f /config/mesh/$mesh_name/metadata.json ]; then
                    echo 'Metadata:'
                    jq '.' /config/mesh/$mesh_name/metadata.json 2>/dev/null || echo '  No metadata available'
                fi
                
                echo
                echo 'Local Node:'
                if [ -f /config/mesh/$mesh_name/publickey ]; then
                    echo \"  Public Key: \$(cat /config/mesh/$mesh_name/publickey)\"
                fi
                
                # Show interface status
                echo
                echo 'Interface Status:'
                if wg show wg-mesh 2>/dev/null; then
                    wg show wg-mesh 2>/dev/null | sed 's/^/  /'
                else
                    echo '  Interface not active'
                fi
                
                # Show peers
                echo
                echo 'Mesh Peers:'
                if [ -f /config/mesh/$mesh_name/peers.json ]; then
                    jq -r '.[] | \"  - \" + .pubkey[0:16] + \"... @ \" + .endpoint' /config/mesh/$mesh_name/peers.json 2>/dev/null || echo '  No peers'
                else
                    echo '  No peers configured'
                fi
                
                # Show traffic stats if interface is up
                if ip link show wg-mesh 2>/dev/null | grep -q 'UP'; then
                    echo
                    echo 'Traffic Statistics:'
                    wg show wg-mesh transfer 2>/dev/null | sed 's/^/  /' || echo '  No traffic data'
                fi
            else
                echo 'Mesh network not found: $mesh_name'
                return 1
            fi
        " || {
            echo "Error: Failed to get mesh status" >&2
            return 1
        }
    fi
    
    return 0
}

sync_mesh_peers() {
    local mesh_name="${1:-}"
    
    if [[ -z "$mesh_name" ]]; then
        echo "Error: Mesh network name required" >&2
        echo "Usage: resource-wireguard mesh sync <name>" >&2
        return 1
    fi
    
    echo "Syncing mesh network peers: $mesh_name"
    
    # Check if container is running
    if ! docker inspect "$CONTAINER_NAME" &>/dev/null; then
        echo "Error: WireGuard container is not running" >&2
        return 1
    fi
    
    # Sync mesh peers (full mesh topology)
    docker exec "$CONTAINER_NAME" sh -c "
        if [ ! -f /config/mesh/$mesh_name/peers.json ]; then
            echo 'No peers to sync'
            return 0
        fi
        
        # Get local public key
        local_pubkey=\$(cat /config/mesh/$mesh_name/publickey 2>/dev/null)
        
        # Count peers
        peer_count=\$(jq '. | length' /config/mesh/$mesh_name/peers.json)
        echo \"Syncing with \$peer_count peers...\"
        
        # Rebuild configuration with all peers
        cat > /config/mesh/$mesh_name/wg-mesh-new.conf << CONF
[Interface]
PrivateKey = \$(cat /config/mesh/$mesh_name/privatekey)
ListenPort = \$(grep ListenPort /config/mesh/$mesh_name/wg-mesh.conf | cut -d= -f2 | tr -d ' ')
Address = \$(grep Address /config/mesh/$mesh_name/wg-mesh.conf | cut -d= -f2 | tr -d ' ')
CONF
        
        # Add all peers from peers.json
        jq -r '.[] | \"[Peer]\\nPublicKey = \" + .pubkey + \"\\nEndpoint = \" + .endpoint + \"\\nAllowedIPs = 10.200.0.0/16\\nPersistentKeepalive = 25\\n\"' \
           /config/mesh/$mesh_name/peers.json >> /config/mesh/$mesh_name/wg-mesh-new.conf
        
        # Backup old configuration
        cp /config/mesh/$mesh_name/wg-mesh.conf /config/mesh/$mesh_name/wg-mesh.conf.bak
        
        # Replace configuration
        mv /config/mesh/$mesh_name/wg-mesh-new.conf /config/mesh/$mesh_name/wg-mesh.conf
        
        # Reload interface
        wg-quick down /config/mesh/$mesh_name/wg-mesh.conf 2>/dev/null || true
        wg-quick up /config/mesh/$mesh_name/wg-mesh.conf 2>/dev/null || {
            echo 'Failed to reload mesh interface'
            # Restore backup on failure
            mv /config/mesh/$mesh_name/wg-mesh.conf.bak /config/mesh/$mesh_name/wg-mesh.conf
            return 1
        }
        
        echo 'Mesh peers synchronized successfully'
        echo 'Active peers:'
        wg show wg-mesh peers 2>/dev/null | head -20
        
    " || {
        echo "Error: Failed to sync mesh peers" >&2
        return 1
    }
    
    return 0
}

# ====================
# Load Balancing Management
# ====================
handle_balance_command() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        enable)
            enable_load_balancing "$@"
            ;;
        disable)
            disable_load_balancing "$@"
            ;;
        add-path)
            add_balance_path "$@"
            ;;
        remove-path)
            remove_balance_path "$@"
            ;;
        status)
            load_balance_status "$@"
            ;;
        policy)
            set_balance_policy "$@"
            ;;
        *)
            echo "Error: Unknown balance subcommand: $subcommand" >&2
            echo "Usage: resource-wireguard balance {enable|disable|add-path|remove-path|status|policy}" >&2
            exit 1
            ;;
    esac
}

enable_load_balancing() {
    local interface="${1:-wg0}"
    
    echo "Enabling load balancing for interface: $interface"
    
    # Check if container is running
    if ! docker inspect "$CONTAINER_NAME" &>/dev/null; then
        echo "Error: WireGuard container is not running" >&2
        return 1
    fi
    
    # Enable load balancing
    docker exec "$CONTAINER_NAME" sh -c "
        mkdir -p /config/loadbalance
        
        # Initialize load balancing configuration
        cat > /config/loadbalance/${interface}.json << JSON
{
    \"interface\": \"$interface\",
    \"enabled\": true,
    \"policy\": \"round-robin\",
    \"paths\": [],
    \"metrics\": {
        \"packets_sent\": 0,
        \"packets_received\": 0,
        \"bytes_sent\": 0,
        \"bytes_received\": 0
    },
    \"created\": \"\$(date -Iseconds)\"
}
JSON
        
        # Install required tools for load balancing
        apk add --no-cache iptables iproute2 iputils 2>/dev/null || true
        
        # Enable IP forwarding
        echo 1 > /proc/sys/net/ipv4/ip_forward
        
        # Create routing tables for multipath
        if ! grep -q 'loadbalance' /etc/iproute2/rt_tables 2>/dev/null; then
            echo '100 loadbalance' >> /etc/iproute2/rt_tables
        fi
        
        echo 'Load balancing enabled for $interface'
        echo 'Use \"balance add-path\" to add alternative paths'
        
    " || {
        echo "Error: Failed to enable load balancing" >&2
        return 1
    }
    
    return 0
}

disable_load_balancing() {
    local interface="${1:-wg0}"
    
    echo "Disabling load balancing for interface: $interface"
    
    # Check if container is running
    if ! docker inspect "$CONTAINER_NAME" &>/dev/null; then
        echo "Error: WireGuard container is not running" >&2
        return 1
    fi
    
    # Disable load balancing
    docker exec "$CONTAINER_NAME" sh -c "
        if [ -f /config/loadbalance/${interface}.json ]; then
            # Backup configuration
            cp /config/loadbalance/${interface}.json /config/loadbalance/${interface}.json.bak
            
            # Remove multipath routes
            ip route del default table loadbalance 2>/dev/null || true
            
            # Update configuration
            jq '.enabled = false' /config/loadbalance/${interface}.json > /tmp/lb.json && \
                mv /tmp/lb.json /config/loadbalance/${interface}.json
            
            echo 'Load balancing disabled for $interface'
        else
            echo 'Load balancing not configured for $interface'
            return 1
        fi
    " || {
        echo "Error: Failed to disable load balancing" >&2
        return 1
    }
    
    return 0
}

add_balance_path() {
    local interface="${1:-wg0}"
    local gateway="${2:-}"
    local weight="${3:-1}"
    
    if [[ -z "$gateway" ]]; then
        echo "Error: Gateway IP required" >&2
        echo "Usage: resource-wireguard balance add-path <interface> <gateway> [weight]" >&2
        return 1
    fi
    
    echo "Adding path for $interface via $gateway (weight: $weight)"
    
    # Check if container is running
    if ! docker inspect "$CONTAINER_NAME" &>/dev/null; then
        echo "Error: WireGuard container is not running" >&2
        return 1
    fi
    
    # Add balance path
    docker exec "$CONTAINER_NAME" sh -c "
        if [ ! -f /config/loadbalance/${interface}.json ]; then
            echo 'Error: Load balancing not enabled for $interface'
            echo 'Run \"balance enable\" first'
            return 1
        fi
        
        # Add path to configuration
        jq '.paths += [{\"gateway\": \"$gateway\", \"weight\": $weight, \"status\": \"active\", \"added\": \"'\$(date -Iseconds)'\"}]' \
           /config/loadbalance/${interface}.json > /tmp/lb.json && \
           mv /tmp/lb.json /config/loadbalance/${interface}.json
        
        # Get policy
        policy=\$(jq -r '.policy' /config/loadbalance/${interface}.json)
        
        # Apply routing based on policy
        if [ \"\$policy\" = \"round-robin\" ]; then
            # Add multipath route for round-robin
            ip route add default scope global nexthop via $gateway weight $weight dev $interface 2>/dev/null || {
                ip route append default scope global nexthop via $gateway weight $weight dev $interface 2>/dev/null || true
            }
        elif [ \"\$policy\" = \"weighted\" ]; then
            # Add weighted multipath
            ip route add default table loadbalance nexthop via $gateway weight $weight dev $interface 2>/dev/null || {
                ip route append default table loadbalance nexthop via $gateway weight $weight dev $interface 2>/dev/null || true
            }
        elif [ \"\$policy\" = \"failover\" ]; then
            # Add as backup route with metric
            metric=\$(jq '.paths | length' /config/loadbalance/${interface}.json)
            ip route add default via $gateway metric \$((100 + \$metric * 10)) dev $interface 2>/dev/null || true
        fi
        
        echo 'Path added successfully'
        echo 'Current paths:'
        jq -r '.paths[] | \"  - Gateway: \" + .gateway + \" Weight: \" + (.weight|tostring) + \" Status: \" + .status' \
           /config/loadbalance/${interface}.json
        
    " || {
        echo "Error: Failed to add balance path" >&2
        return 1
    }
    
    return 0
}

remove_balance_path() {
    local interface="${1:-wg0}"
    local gateway="${2:-}"
    
    if [[ -z "$gateway" ]]; then
        echo "Error: Gateway IP required" >&2
        echo "Usage: resource-wireguard balance remove-path <interface> <gateway>" >&2
        return 1
    fi
    
    echo "Removing path for $interface via $gateway"
    
    # Check if container is running
    if ! docker inspect "$CONTAINER_NAME" &>/dev/null; then
        echo "Error: WireGuard container is not running" >&2
        return 1
    fi
    
    # Remove balance path
    docker exec "$CONTAINER_NAME" sh -c "
        if [ ! -f /config/loadbalance/${interface}.json ]; then
            echo 'Error: Load balancing not configured for $interface'
            return 1
        fi
        
        # Remove path from configuration
        jq '.paths = [.paths[] | select(.gateway != \"$gateway\")]' \
           /config/loadbalance/${interface}.json > /tmp/lb.json && \
           mv /tmp/lb.json /config/loadbalance/${interface}.json
        
        # Remove route
        ip route del default via $gateway 2>/dev/null || true
        ip route del default via $gateway table loadbalance 2>/dev/null || true
        
        echo 'Path removed successfully'
        echo 'Remaining paths:'
        jq -r '.paths[] | \"  - Gateway: \" + .gateway + \" Weight: \" + (.weight|tostring) + \" Status: \" + .status' \
           /config/loadbalance/${interface}.json
        
    " || {
        echo "Error: Failed to remove balance path" >&2
        return 1
    }
    
    return 0
}

load_balance_status() {
    echo "Load Balancing Status"
    echo "===================="
    
    # Check if container is running
    if ! docker inspect "$CONTAINER_NAME" &>/dev/null; then
        echo "Error: WireGuard container is not running" >&2
        return 1
    fi
    
    docker exec "$CONTAINER_NAME" sh -c "
        if [ -d /config/loadbalance ]; then
            for config in /config/loadbalance/*.json; do
                if [ -f \"\$config\" ]; then
                    interface=\$(basename \"\$config\" .json)
                    echo \"Interface: \$interface\"
                    
                    # Show configuration
                    enabled=\$(jq -r '.enabled' \"\$config\")
                    policy=\$(jq -r '.policy' \"\$config\")
                    path_count=\$(jq '.paths | length' \"\$config\")
                    
                    echo \"  Enabled: \$enabled\"
                    echo \"  Policy: \$policy\"
                    echo \"  Paths: \$path_count\"
                    
                    if [ \"\$path_count\" -gt 0 ]; then
                        echo \"  Active Paths:\"
                        jq -r '.paths[] | \"    - \" + .gateway + \" (weight: \" + (.weight|tostring) + \", status: \" + .status + \")\"' \"\$config\"
                    fi
                    
                    # Show metrics if enabled
                    if [ \"\$enabled\" = \"true\" ]; then
                        echo \"  Traffic Distribution:\"
                        # Check actual route usage
                        ip route show | grep -E \"^default|nexthop\" | head -5 | sed 's/^/    /'
                    fi
                    echo
                fi
            done
        else
            echo 'No load balancing configurations found'
        fi
        
        # Show routing tables
        echo 'Routing Tables:'
        ip route show table all | grep -E \"table|nexthop|default\" | head -10
        
    " 2>/dev/null || echo "Error retrieving load balance status"
    
    return 0
}

set_balance_policy() {
    local interface="${1:-wg0}"
    local policy="${2:-}"
    
    if [[ -z "$policy" ]]; then
        echo "Error: Policy required (round-robin|weighted|failover)" >&2
        echo "Usage: resource-wireguard balance policy <interface> <policy>" >&2
        return 1
    fi
    
    # Validate policy
    if [[ "$policy" != "round-robin" ]] && [[ "$policy" != "weighted" ]] && [[ "$policy" != "failover" ]]; then
        echo "Error: Invalid policy. Choose: round-robin, weighted, or failover" >&2
        return 1
    fi
    
    echo "Setting load balance policy for $interface to: $policy"
    
    # Check if container is running
    if ! docker inspect "$CONTAINER_NAME" &>/dev/null; then
        echo "Error: WireGuard container is not running" >&2
        return 1
    fi
    
    # Set policy
    docker exec "$CONTAINER_NAME" sh -c "
        if [ ! -f /config/loadbalance/${interface}.json ]; then
            echo 'Error: Load balancing not configured for $interface'
            echo 'Run \"balance enable\" first'
            return 1
        fi
        
        # Update policy
        old_policy=\$(jq -r '.policy' /config/loadbalance/${interface}.json)
        jq '.policy = \"$policy\"' /config/loadbalance/${interface}.json > /tmp/lb.json && \
           mv /tmp/lb.json /config/loadbalance/${interface}.json
        
        echo \"Policy changed from \$old_policy to $policy\"
        
        # Reconfigure routes based on new policy
        paths=\$(jq -r '.paths[] | .gateway + \":\" + (.weight|tostring)' /config/loadbalance/${interface}.json)
        
        if [ -n \"\$paths\" ]; then
            echo 'Reconfiguring routes for new policy...'
            
            # Clear existing routes
            ip route del default 2>/dev/null || true
            ip route del default table loadbalance 2>/dev/null || true
            
            # Apply new policy
            if [ \"$policy\" = \"round-robin\" ]; then
                echo 'Configuring round-robin load balancing...'
                for path in \$paths; do
                    gw=\$(echo \$path | cut -d: -f1)
                    weight=\$(echo \$path | cut -d: -f2)
                    ip route add default scope global nexthop via \$gw weight \$weight dev $interface 2>/dev/null || {
                        ip route append default scope global nexthop via \$gw weight \$weight dev $interface 2>/dev/null || true
                    }
                done
            elif [ \"$policy\" = \"weighted\" ]; then
                echo 'Configuring weighted load balancing...'
                for path in \$paths; do
                    gw=\$(echo \$path | cut -d: -f1)
                    weight=\$(echo \$path | cut -d: -f2)
                    ip route add default table loadbalance nexthop via \$gw weight \$weight dev $interface 2>/dev/null || {
                        ip route append default table loadbalance nexthop via \$gw weight \$weight dev $interface 2>/dev/null || true
                    }
                done
                # Add rule to use loadbalance table
                ip rule add from all table loadbalance 2>/dev/null || true
            elif [ \"$policy\" = \"failover\" ]; then
                echo 'Configuring failover routing...'
                metric=100
                for path in \$paths; do
                    gw=\$(echo \$path | cut -d: -f1)
                    ip route add default via \$gw metric \$metric dev $interface 2>/dev/null || true
                    metric=\$((metric + 10))
                done
            fi
            
            echo 'Routes reconfigured successfully'
        fi
        
    " || {
        echo "Error: Failed to set balance policy" >&2
        return 1
    }
    
    return 0
}

# ====================
# QoS Management
# ====================
handle_qos_command() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        enable)
            enable_qos "$@"
            ;;
        disable)
            disable_qos "$@"
            ;;
        set-limit)
            set_bandwidth_limit "$@"
            ;;
        priority)
            set_traffic_priority "$@"
            ;;
        class)
            define_traffic_class "$@"
            ;;
        status)
            qos_status "$@"
            ;;
        *)
            echo "Error: Unknown qos subcommand: $subcommand" >&2
            echo "Usage: resource-wireguard qos {enable|disable|set-limit|priority|class|status}" >&2
            exit 1
            ;;
    esac
}

enable_qos() {
    local interface="${1:-wg0}"
    
    echo "Enabling QoS for interface: $interface"
    
    # Check if container is running
    if ! docker inspect "$CONTAINER_NAME" &>/dev/null; then
        echo "Error: WireGuard container is not running" >&2
        return 1
    fi
    
    # Enable QoS
    docker exec "$CONTAINER_NAME" sh -c "
        mkdir -p /config/qos
        
        # Initialize QoS configuration
        cat > /config/qos/${interface}.json << JSON
{
    \"interface\": \"$interface\",
    \"enabled\": true,
    \"bandwidth_limit\": \"unlimited\",
    \"traffic_classes\": [],
    \"priority_rules\": [],
    \"statistics\": {
        \"packets_shaped\": 0,
        \"packets_dropped\": 0,
        \"bytes_shaped\": 0
    },
    \"created\": \"\$(date -Iseconds)\"
}
JSON
        
        # Install tc (traffic control) for QoS
        apk add --no-cache iproute2 iptables 2>/dev/null || true
        
        # Create root qdisc (queuing discipline)
        tc qdisc add dev $interface root handle 1: htb default 30 2>/dev/null || {
            tc qdisc del dev $interface root 2>/dev/null || true
            tc qdisc add dev $interface root handle 1: htb default 30
        }
        
        # Create default class with no limit
        tc class add dev $interface parent 1: classid 1:1 htb rate 1000mbit
        tc class add dev $interface parent 1:1 classid 1:30 htb rate 100mbit ceil 1000mbit
        
        echo 'QoS enabled for $interface'
        echo 'Use \"qos set-limit\" to configure bandwidth limits'
        echo 'Use \"qos priority\" to set traffic priorities'
        
    " || {
        echo "Error: Failed to enable QoS" >&2
        return 1
    }
    
    return 0
}

disable_qos() {
    local interface="${1:-wg0}"
    
    echo "Disabling QoS for interface: $interface"
    
    # Check if container is running
    if ! docker inspect "$CONTAINER_NAME" &>/dev/null; then
        echo "Error: WireGuard container is not running" >&2
        return 1
    fi
    
    # Disable QoS
    docker exec "$CONTAINER_NAME" sh -c "
        if [ -f /config/qos/${interface}.json ]; then
            # Backup configuration
            cp /config/qos/${interface}.json /config/qos/${interface}.json.bak
            
            # Remove traffic control
            tc qdisc del dev $interface root 2>/dev/null || true
            
            # Clear iptables marks
            iptables -t mangle -F 2>/dev/null || true
            
            # Update configuration
            jq '.enabled = false' /config/qos/${interface}.json > /tmp/qos.json && \
                mv /tmp/qos.json /config/qos/${interface}.json
            
            echo 'QoS disabled for $interface'
        else
            echo 'QoS not configured for $interface'
            return 1
        fi
    " || {
        echo "Error: Failed to disable QoS" >&2
        return 1
    }
    
    return 0
}

set_bandwidth_limit() {
    local interface="${1:-wg0}"
    local rate="${2:-}"
    local burst="${3:-15k}"
    
    if [[ -z "$rate" ]]; then
        echo "Error: Bandwidth rate required (e.g., 100mbit, 50mbit)" >&2
        echo "Usage: resource-wireguard qos set-limit <interface> <rate> [burst]" >&2
        return 1
    fi
    
    echo "Setting bandwidth limit for $interface to: $rate"
    
    # Check if container is running
    if ! docker inspect "$CONTAINER_NAME" &>/dev/null; then
        echo "Error: WireGuard container is not running" >&2
        return 1
    fi
    
    # Set bandwidth limit
    docker exec "$CONTAINER_NAME" sh -c "
        if [ ! -f /config/qos/${interface}.json ]; then
            echo 'Error: QoS not enabled for $interface'
            echo 'Run \"qos enable\" first'
            return 1
        fi
        
        # Update configuration
        jq '.bandwidth_limit = \"$rate\"' /config/qos/${interface}.json > /tmp/qos.json && \
           mv /tmp/qos.json /config/qos/${interface}.json
        
        # Apply bandwidth limit with tc
        # Remove existing root qdisc
        tc qdisc del dev $interface root 2>/dev/null || true
        
        # Add HTB root qdisc
        tc qdisc add dev $interface root handle 1: htb default 30
        
        # Add root class with bandwidth limit
        tc class add dev $interface parent 1: classid 1:1 htb rate $rate burst $burst
        
        # Add leaf classes for different priorities
        tc class add dev $interface parent 1:1 classid 1:10 htb rate \$(echo $rate | sed 's/mbit//')mbit ceil $rate prio 1
        tc class add dev $interface parent 1:1 classid 1:20 htb rate \$(echo $rate | sed 's/mbit//')mbit ceil $rate prio 2
        tc class add dev $interface parent 1:1 classid 1:30 htb rate \$(echo $rate | sed 's/mbit//')mbit ceil $rate prio 3
        
        # Add SFQ for fairness
        tc qdisc add dev $interface parent 1:10 handle 10: sfq perturb 10
        tc qdisc add dev $interface parent 1:20 handle 20: sfq perturb 10
        tc qdisc add dev $interface parent 1:30 handle 30: sfq perturb 10
        
        echo 'Bandwidth limit set successfully'
        echo 'Rate: $rate, Burst: $burst'
        
    " || {
        echo "Error: Failed to set bandwidth limit" >&2
        return 1
    }
    
    return 0
}

set_traffic_priority() {
    local interface="${1:-wg0}"
    local port="${2:-}"
    local priority="${3:-}"
    local protocol="${4:-tcp}"
    
    if [[ -z "$port" ]] || [[ -z "$priority" ]]; then
        echo "Error: Port and priority required" >&2
        echo "Usage: resource-wireguard qos priority <interface> <port> <priority> [protocol]" >&2
        echo "Priority: 1 (highest) to 3 (lowest)" >&2
        return 1
    fi
    
    echo "Setting priority $priority for $protocol port $port on $interface"
    
    # Check if container is running
    if ! docker inspect "$CONTAINER_NAME" &>/dev/null; then
        echo "Error: WireGuard container is not running" >&2
        return 1
    fi
    
    # Set traffic priority
    docker exec "$CONTAINER_NAME" sh -c "
        if [ ! -f /config/qos/${interface}.json ]; then
            echo 'Error: QoS not enabled for $interface'
            echo 'Run \"qos enable\" first'
            return 1
        fi
        
        # Add priority rule to configuration
        jq '.priority_rules += [{\"port\": $port, \"protocol\": \"$protocol\", \"priority\": $priority, \"added\": \"'\$(date -Iseconds)'\"}]' \
           /config/qos/${interface}.json > /tmp/qos.json && \
           mv /tmp/qos.json /config/qos/${interface}.json
        
        # Map priority to class
        class_id=\$((10 * $priority))
        
        # Add iptables rule to mark packets
        mark_value=\$priority
        
        if [ \"$protocol\" = \"tcp\" ]; then
            iptables -t mangle -A POSTROUTING -o $interface -p tcp --dport $port -j MARK --set-mark \$mark_value 2>/dev/null || true
            iptables -t mangle -A POSTROUTING -o $interface -p tcp --sport $port -j MARK --set-mark \$mark_value 2>/dev/null || true
        elif [ \"$protocol\" = \"udp\" ]; then
            iptables -t mangle -A POSTROUTING -o $interface -p udp --dport $port -j MARK --set-mark \$mark_value 2>/dev/null || true
            iptables -t mangle -A POSTROUTING -o $interface -p udp --sport $port -j MARK --set-mark \$mark_value 2>/dev/null || true
        fi
        
        # Add tc filter to classify marked packets
        tc filter add dev $interface parent 1:0 protocol ip prio \$priority handle \$mark_value fw classid 1:\$class_id 2>/dev/null || true
        
        echo 'Priority rule added successfully'
        echo 'Port $port ($protocol) -> Priority $priority'
        
    " || {
        echo "Error: Failed to set traffic priority" >&2
        return 1
    }
    
    return 0
}

define_traffic_class() {
    local interface="${1:-wg0}"
    local class_name="${2:-}"
    local rate="${3:-}"
    local ceil="${4:-}"
    
    if [[ -z "$class_name" ]] || [[ -z "$rate" ]]; then
        echo "Error: Class name and rate required" >&2
        echo "Usage: resource-wireguard qos class <interface> <name> <rate> [ceil]" >&2
        echo "Example: resource-wireguard qos class wg0 voip 10mbit 20mbit" >&2
        return 1
    fi
    
    ceil="${ceil:-$rate}"
    
    echo "Defining traffic class '$class_name' on $interface"
    echo "Guaranteed rate: $rate, Maximum rate: $ceil"
    
    # Check if container is running
    if ! docker inspect "$CONTAINER_NAME" &>/dev/null; then
        echo "Error: WireGuard container is not running" >&2
        return 1
    fi
    
    # Define traffic class
    docker exec "$CONTAINER_NAME" sh -c "
        if [ ! -f /config/qos/${interface}.json ]; then
            echo 'Error: QoS not enabled for $interface'
            echo 'Run \"qos enable\" first'
            return 1
        fi
        
        # Generate class ID
        class_count=\$(jq '.traffic_classes | length' /config/qos/${interface}.json)
        class_id=\$((40 + class_count))
        
        # Add class to configuration
        jq '.traffic_classes += [{\"name\": \"$class_name\", \"class_id\": \"1:'\$class_id'\", \"rate\": \"$rate\", \"ceil\": \"$ceil\", \"added\": \"'\$(date -Iseconds)'\"}]' \
           /config/qos/${interface}.json > /tmp/qos.json && \
           mv /tmp/qos.json /config/qos/${interface}.json
        
        # Create traffic class with tc
        tc class add dev $interface parent 1:1 classid 1:\$class_id htb rate $rate ceil $ceil
        
        # Add SFQ for fairness within the class
        tc qdisc add dev $interface parent 1:\$class_id handle \$class_id: sfq perturb 10
        
        echo 'Traffic class created successfully'
        echo 'Class: $class_name (1:'\$class_id')'
        echo 'To assign traffic to this class, use iptables marking and tc filters'
        
    " || {
        echo "Error: Failed to define traffic class" >&2
        return 1
    }
    
    return 0
}

qos_status() {
    echo "QoS Configuration Status"
    echo "======================="
    
    # Check if container is running
    if ! docker inspect "$CONTAINER_NAME" &>/dev/null; then
        echo "Error: WireGuard container is not running" >&2
        return 1
    fi
    
    docker exec "$CONTAINER_NAME" sh -c "
        if [ -d /config/qos ]; then
            for config in /config/qos/*.json; do
                if [ -f \"\$config\" ]; then
                    interface=\$(basename \"\$config\" .json)
                    echo \"Interface: \$interface\"
                    echo \"-------------------\"
                    
                    # Show configuration
                    enabled=\$(jq -r '.enabled' \"\$config\")
                    limit=\$(jq -r '.bandwidth_limit' \"\$config\")
                    class_count=\$(jq '.traffic_classes | length' \"\$config\")
                    rule_count=\$(jq '.priority_rules | length' \"\$config\")
                    
                    echo \"  QoS Enabled: \$enabled\"
                    echo \"  Bandwidth Limit: \$limit\"
                    echo \"  Traffic Classes: \$class_count\"
                    echo \"  Priority Rules: \$rule_count\"
                    
                    # Show traffic classes
                    if [ \"\$class_count\" -gt 0 ]; then
                        echo
                        echo \"  Traffic Classes:\"
                        jq -r '.traffic_classes[] | \"    - \" + .name + \" (\" + .class_id + \"): \" + .rate + \" (max: \" + .ceil + \")\"' \"\$config\"
                    fi
                    
                    # Show priority rules
                    if [ \"\$rule_count\" -gt 0 ]; then
                        echo
                        echo \"  Priority Rules:\"
                        jq -r '.priority_rules[] | \"    - Port \" + (.port|tostring) + \" (\" + .protocol + \"): Priority \" + (.priority|tostring)' \"\$config\"
                    fi
                    
                    # Show live statistics if enabled
                    if [ \"\$enabled\" = \"true\" ]; then
                        echo
                        echo \"  Live Statistics:\"
                        echo \"    Traffic Control:\"
                        tc -s class show dev \$interface 2>/dev/null | head -20 | sed 's/^/      /'
                        
                        echo
                        echo \"    Queue Disciplines:\"
                        tc -s qdisc show dev \$interface 2>/dev/null | head -10 | sed 's/^/      /'
                    fi
                    echo
                fi
            done
        else
            echo 'No QoS configurations found'
        fi
    " 2>/dev/null || echo "Error retrieving QoS status"
    
    return 0
}
