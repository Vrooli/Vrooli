#!/usr/bin/env bash
#
# Browserless uninstall script
#
set -euo pipefail

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source common utilities
source "${SCRIPT_DIR}/common.sh"

# Uninstall browserless
function uninstall_browserless() {
    echo "🗑️ Uninstalling browserless..."
    
    # Stop container if running
    if is_running; then
        echo "  Stopping browserless container..."
        docker stop "$BROWSERLESS_CONTAINER_NAME" 2>/dev/null || true
    fi
    
    # Remove container
    if docker ps -a --format "{{.Names}}" | grep -q "^${BROWSERLESS_CONTAINER_NAME}$"; then
        echo "  Removing browserless container..."
        docker rm "$BROWSERLESS_CONTAINER_NAME" 2>/dev/null || true
    fi
    
    # Clean up data directory if requested
    if [[ "${1:-}" == "--clean" ]]; then
        echo "  Cleaning browserless data directory..."
        rm -rf "$BROWSERLESS_DATA_DIR"
    fi
    
    echo "✅ Browserless uninstalled successfully"
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    uninstall_browserless "$@"
fi