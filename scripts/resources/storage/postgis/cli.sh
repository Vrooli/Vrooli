#!/bin/bash
# PostGIS CLI Registration

# Get script directory
POSTGIS_CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Register with Vrooli CLI
register_postgis_cli() {
    local cli_dir="${HOME}/Vrooli/cli"
    
    if [ -d "$cli_dir" ]; then
        # Create symlink for CLI command
        ln -sf "${POSTGIS_CLI_DIR}/resource-postgis" "${cli_dir}/resource-postgis" 2>/dev/null || true
        
        # Register with Vrooli resource system
        if [ -f "${cli_dir}/vrooli" ]; then
            # Registration is automatic when CLI is in correct location
            return 0
        fi
    fi
    
    return 0
}

# Auto-register on source
register_postgis_cli

# Export registration function
export -f register_postgis_cli