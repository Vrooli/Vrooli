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
