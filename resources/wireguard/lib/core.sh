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