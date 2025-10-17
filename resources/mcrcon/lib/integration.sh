#!/bin/bash
# Integration helpers for mcrcon with other Vrooli resources

# Source base directory logic
if [[ -z "${SCRIPT_DIR:-}" ]]; then
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
fi
if [[ -z "${RESOURCE_DIR:-}" ]]; then
    RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"
fi

# Source configuration
source "${RESOURCE_DIR}/config/defaults.sh"

# Auto-detect and configure PaperMC integration
detect_papermc() {
    local papermc_config="$HOME/.papermc/docker-compose.yml"
    
    if [[ -f "$papermc_config" ]]; then
        echo "PaperMC installation detected!"
        
        # Extract RCON password from docker-compose.yml
        local password=$(grep "RCON_PASSWORD=" "$papermc_config" | sed 's/.*RCON_PASSWORD=//' | tr -d ' ')
        
        if [[ -n "$password" ]]; then
            echo "RCON password found: $password"
            echo ""
            echo "To use mcrcon with PaperMC, set the environment variable:"
            echo "  export MCRCON_PASSWORD=$password"
            echo ""
            echo "Or add PaperMC as a server:"
            echo "  vrooli resource mcrcon content add papermc localhost 25575 $password"
            
            return 0
        else
            echo "Warning: Could not extract RCON password from PaperMC config"
            return 1
        fi
    else
        return 1
    fi
}

# Check if PaperMC is running
check_papermc_status() {
    if docker ps --format "table {{.Names}}" | grep -q "papermc-server"; then
        echo "✓ PaperMC server is running"
        
        # Get server status
        if [[ -n "$MCRCON_PASSWORD" ]]; then
            echo ""
            echo "Server Status:"
            timeout 5 "${MCRCON_BINARY}" -H localhost -P 25575 -p "$MCRCON_PASSWORD" "list" 2>/dev/null || \
                echo "  Unable to connect (check MCRCON_PASSWORD)"
        else
            echo "  Set MCRCON_PASSWORD to connect"
        fi
        
        return 0
    else
        echo "✗ PaperMC server is not running"
        echo "  Start it with: vrooli resource papermc manage start"
        return 1
    fi
}

# Auto-configure for detected servers
auto_configure() {
    echo "=== Auto-Configuration ==="
    echo ""
    
    local configured=0
    
    # Check for PaperMC
    if detect_papermc; then
        configured=$((configured + 1))
    fi
    
    # Check for other Minecraft servers (future expansion)
    # detect_vanilla_minecraft
    # detect_spigot
    # detect_forge
    
    if [[ $configured -eq 0 ]]; then
        echo "No Minecraft servers detected for auto-configuration"
        echo ""
        echo "You can manually configure servers with:"
        echo "  vrooli resource mcrcon content add <name> <host> <port> <password>"
    else
        echo ""
        echo "Auto-configuration complete!"
    fi
}

# Quick start helper
quick_start() {
    echo "=== mcrcon Quick Start ==="
    echo ""
    
    # Check if mcrcon is installed
    if [[ ! -f "${MCRCON_BINARY}" ]]; then
        echo "Installing mcrcon..."
        "${RESOURCE_DIR}/cli.sh" manage install
    fi
    
    # Check if service is running
    if ! pgrep -f "health_server.py.*mcrcon" > /dev/null; then
        echo "Starting mcrcon service..."
        "${RESOURCE_DIR}/cli.sh" manage start --wait
    fi
    
    # Check for PaperMC
    if check_papermc_status; then
        if detect_papermc > /dev/null; then
            local password=$(grep "RCON_PASSWORD=" "$HOME/.papermc/docker-compose.yml" | sed 's/.*RCON_PASSWORD=//' | tr -d ' ')
            export MCRCON_PASSWORD="$password"
            
            echo ""
            echo "Testing connection..."
            if timeout 5 "${MCRCON_BINARY}" -H localhost -P 25575 -p "$MCRCON_PASSWORD" "list" &>/dev/null; then
                echo "✓ Successfully connected to PaperMC server!"
                echo ""
                echo "Example commands:"
                echo "  vrooli resource mcrcon content execute \"list\""
                echo "  vrooli resource mcrcon content execute \"say Hello!\""
                echo "  vrooli resource mcrcon world info"
                echo "  vrooli resource mcrcon player list"
            else
                echo "✗ Could not connect to server"
            fi
        fi
    else
        echo ""
        echo "To start PaperMC server:"
        echo "  vrooli resource papermc manage start --wait"
    fi
}

# Export functions
export -f detect_papermc
export -f check_papermc_status
export -f auto_configure
export -f quick_start