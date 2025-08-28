#!/usr/bin/env bash
################################################################################
# PID Directory Setup Tool
# Ensures all PID tracking directories exist for proper process management
################################################################################

set -euo pipefail

# PID directories
PID_DIRS=(
    "/tmp/vrooli-apps"
    "/tmp/vrooli-ui-pids"
    "/tmp/vrooli-api-pids"
    "/tmp/vrooli-processes"
)

echo "🔧 Setting up PID tracking directories..."

for dir in "${PID_DIRS[@]}"; do
    if [[ ! -d "$dir" ]]; then
        mkdir -p "$dir"
        chmod 755 "$dir"
        echo "  ✓ Created: $dir"
    else
        echo "  • Exists: $dir"
    fi
done

# Clean up stale PID files
echo ""
echo "🧹 Cleaning stale PID files..."
CLEANED=0

for dir in "${PID_DIRS[@]}"; do
    if [[ -d "$dir" ]]; then
        for pidfile in "$dir"/*.pid; do
            if [[ -f "$pidfile" ]]; then
                pid=$(<"$pidfile")
                if ! kill -0 "$pid" 2>/dev/null; then
                    rm -f "$pidfile"
                    echo "  ✓ Removed stale: $(basename "$pidfile")"
                    ((CLEANED++))
                fi
            fi
        done
    fi
done

if [[ $CLEANED -eq 0 ]]; then
    echo "  No stale PID files found"
fi

echo ""
echo "✅ PID directory setup complete!"