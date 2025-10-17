#!/usr/bin/env bash
# Pi-hole Web UI Management - Web interface access and configuration
set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Source core library
source "${SCRIPT_DIR}/core.sh"

# Get web interface URL
get_web_url() {
    local api_port="${PIHOLE_API_PORT:-8087}"
    echo "http://localhost:${api_port}/admin"
}

# Get web interface password
get_web_password() {
    if [[ -f "${PIHOLE_DATA_DIR}/.webpassword" ]]; then
        cat "${PIHOLE_DATA_DIR}/.webpassword"
    else
        echo "Error: Web password file not found" >&2
        return 1
    fi
}

# Enable web interface
enable_web_interface() {
    echo "Enabling Pi-hole web interface..."
    
    # Check if container is running
    if ! docker ps --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        echo "Error: Pi-hole container is not running" >&2
        return 1
    fi
    
    # Enable lighttpd if disabled
    docker exec "${CONTAINER_NAME}" bash -c "lighttpd-enable-mod fastcgi fastcgi-php 2>/dev/null || true"
    docker exec "${CONTAINER_NAME}" bash -c "service lighttpd restart 2>/dev/null || true"
    
    local url=$(get_web_url)
    local password=$(get_web_password)
    
    echo "Web interface enabled at: ${url}"
    echo "Login password: ${password}"
    echo ""
    echo "To access the web interface:"
    echo "1. Open your browser to: ${url}"
    echo "2. Click 'Login' in the left menu"
    echo "3. Enter the password shown above"
}

# Disable web interface
disable_web_interface() {
    echo "Disabling Pi-hole web interface..."
    
    # Stop lighttpd service
    docker exec "${CONTAINER_NAME}" bash -c "service lighttpd stop 2>/dev/null || true"
    
    echo "Web interface disabled"
}

# Check web interface status
check_web_status() {
    local api_port="${PIHOLE_API_PORT:-8087}"
    
    if timeout 5 curl -sf "http://localhost:${api_port}/admin/" > /dev/null 2>&1; then
        echo "Web interface is accessible"
        local url=$(get_web_url)
        echo "URL: ${url}"
        return 0
    else
        echo "Web interface is not accessible"
        return 1
    fi
}

# Reset web password
reset_web_password() {
    local new_password="${1:-}"
    
    if [[ -z "$new_password" ]]; then
        # Generate new random password
        new_password=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-16)
    fi
    
    echo "Setting new web interface password..."
    
    # Update password in container
    docker exec "${CONTAINER_NAME}" pihole -a -p "${new_password}"
    
    # Save to file
    echo "$new_password" > "${PIHOLE_DATA_DIR}/.webpassword"
    chmod 600 "${PIHOLE_DATA_DIR}/.webpassword"
    
    echo "New password set: ${new_password}"
}

# Open web interface in browser (if available)
open_web_interface() {
    local url=$(get_web_url)
    
    echo "Web interface URL: ${url}"
    
    # Try to open in browser if available
    if command -v xdg-open &> /dev/null; then
        xdg-open "${url}" 2>/dev/null || true
        echo "Opening in default browser..."
    elif command -v open &> /dev/null; then
        open "${url}" 2>/dev/null || true
        echo "Opening in default browser..."
    else
        echo "Please manually open the URL in your browser"
    fi
    
    # Show password
    echo ""
    echo "Login password: $(get_web_password 2>/dev/null || echo 'Run: vrooli resource pihole web password')"
}

# Configure web interface settings
configure_web_interface() {
    local setting="$1"
    local value="$2"
    
    case "$setting" in
        "theme")
            # Set theme (default, dark, etc)
            docker exec "${CONTAINER_NAME}" bash -c "echo 'WEBTHEME=${value}' >> /etc/pihole/setupVars.conf"
            docker exec "${CONTAINER_NAME}" bash -c "service lighttpd restart"
            echo "Web interface theme set to: ${value}"
            ;;
        "boxed_layout")
            # Enable/disable boxed layout
            docker exec "${CONTAINER_NAME}" bash -c "echo 'BOXEDLAYOUT=${value}' >> /etc/pihole/setupVars.conf"
            docker exec "${CONTAINER_NAME}" bash -c "service lighttpd restart"
            echo "Boxed layout: ${value}"
            ;;
        *)
            echo "Unknown setting: ${setting}"
            echo "Available settings: theme, boxed_layout"
            return 1
            ;;
    esac
}

# Main entry point for CLI
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "${1:-}" in
    "enable")
        enable_web_interface
        ;;
    "disable")
        disable_web_interface
        ;;
    "status")
        check_web_status
        ;;
    "password")
        if [[ -n "${2:-}" ]]; then
            reset_web_password "$2"
        else
            get_web_password
        fi
        ;;
    "open")
        open_web_interface
        ;;
    "configure")
        configure_web_interface "${2:-}" "${3:-}"
        ;;
    "url")
        get_web_url
        ;;
    *)
        echo "Usage: $0 {enable|disable|status|password|open|configure|url}"
        echo ""
        echo "Commands:"
        echo "  enable    - Enable web interface"
        echo "  disable   - Disable web interface"
        echo "  status    - Check web interface status"
        echo "  password  - Show or reset web password"
        echo "  open      - Open web interface in browser"
        echo "  configure - Configure web interface settings"
        echo "  url       - Show web interface URL"
        exit 1
        ;;
    esac
fi