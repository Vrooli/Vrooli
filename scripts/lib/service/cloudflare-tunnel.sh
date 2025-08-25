#!/bin/bash

# Cloudflare Tunnel setup service
# Installs cloudflared package and dependencies for Cloudflare tunnels

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/scripts/lib/service"
if [[ -f "${APP_ROOT}/scripts/lib/utils/common.sh" ]]; then
    source "${APP_ROOT}/scripts/lib/utils/common.sh"
fi

# Main setup function for Cloudflare tunnels
cloudflare_tunnel_setup() {
    echo "Setting up Cloudflare tunnel packages..."
    
    # Add cloudflare gpg key
    echo "Adding Cloudflare GPG key..."
    sudo mkdir -p --mode=0755 /usr/share/keyrings
    curl -fsSL https://pkg.cloudflare.com/cloudflare-main.gpg | sudo tee /usr/share/keyrings/cloudflare-main.gpg >/dev/null
    
    # Add this repo to your apt repositories
    echo "Adding Cloudflare repository..."
    echo 'deb [signed-by=/usr/share/keyrings/cloudflare-main.gpg] https://pkg.cloudflare.com/cloudflared any main' | sudo tee /etc/apt/sources.list.d/cloudflared.list
    
    # install cloudflared
    echo "Installing cloudflared..."
    sudo apt-get update && sudo apt-get install -y cloudflared
    
    echo "Cloudflare tunnel setup completed successfully"
}

# Check if cloudflared is already installed
check_cloudflared_installed() {
    if command -v cloudflared &> /dev/null; then
        echo "cloudflared is already installed (version: $(cloudflared --version))"
        return 0
    else
        return 1
    fi
}

# Main entry point
main() {
    if check_cloudflared_installed; then
        echo "Cloudflare tunnel packages already installed, skipping setup"
        return 0
    fi
    
    cloudflare_tunnel_setup
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi