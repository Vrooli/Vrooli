#!/usr/bin/env bash
################################################################################
# Start Vrooli App API Server
#
# Simple script to start the app management API server
################################################################################

set -euo pipefail

API_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PORT="${VROOLI_APP_API_PORT:-8094}"

# Check if already running
if lsof -i ":${PORT}" >/dev/null 2>&1; then
    echo "⚠️  API already running on port ${PORT}"
    exit 0
fi

# Install dependencies if needed
if [[ ! -f "$API_DIR/go.sum" ]]; then
    echo "📦 Installing dependencies..."
    cd "$API_DIR"
    go mod download
fi

# Start server
echo "🚀 Starting Vrooli App API on port ${PORT}..."
cd "$API_DIR"
exec go run main.go