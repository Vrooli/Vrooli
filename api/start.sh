#!/usr/bin/env bash
################################################################################
# Start Vrooli Unified API Server
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/.." && builtin pwd)}"
PORT="${VROOLI_API_PORT:-8092}"
export VROOLI_ROOT="${VROOLI_ROOT:-$APP_ROOT}"
export VROOLI_SOURCE_ROOT="${VROOLI_SOURCE_ROOT:-$APP_ROOT}"
export VROOLI_FINGERPRINT_PATHS="${VROOLI_FINGERPRINT_PATHS:-cmd/vrooli-api,internal}"

BUILD_BINARY="$APP_ROOT/.vrooli/build/vrooli-api"
INSTALLED_BINARY="${HOME:-}/.vrooli/bin/vrooli-api"

# Check if already running
if lsof -i ":${PORT}" >/dev/null 2>&1; then
    echo "⚠️  Vrooli API already running on port ${PORT}"
    exit 0
fi

# Start server
echo "🚀 Starting Vrooli Unified API on port ${PORT}..."
cd "$APP_ROOT"
if [[ -x "$BUILD_BINARY" ]]; then
    exec "$BUILD_BINARY"
fi

if [[ -x "$INSTALLED_BINARY" ]]; then
    exec "$INSTALLED_BINARY"
fi

# Install dependencies if needed for go run fallback
if [[ ! -f "$APP_ROOT/go.sum" ]]; then
    echo "📦 Installing dependencies..."
    go mod download
fi

exec go run ./cmd/vrooli-api
