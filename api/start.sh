#!/usr/bin/env bash
################################################################################
# Start Vrooli Unified API Server
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/.." && builtin pwd)}"
API_DIR="${APP_ROOT}/api"
PORT="${VROOLI_API_PORT:-8092}"

# Check if already running
if lsof -i ":${PORT}" >/dev/null 2>&1; then
    echo "⚠️  Vrooli API already running on port ${PORT}"
    exit 0
fi

# Install dependencies if needed
if [[ ! -f "$API_DIR/go.sum" ]]; then
    echo "📦 Installing dependencies..."
    cd "$API_DIR"
    go mod download
fi

# Start server
echo "🚀 Starting Vrooli Unified API on port ${PORT}..."
cd "$API_DIR"
exec go run main.go