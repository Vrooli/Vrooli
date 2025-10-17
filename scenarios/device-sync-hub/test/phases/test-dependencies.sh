#!/bin/bash
# Dependency Tests for Device Sync Hub
# Validates required resources and services

set -euo pipefail

echo "=== Device Sync Hub Dependency Tests ==="
echo "[INFO] Checking required dependencies..."

# Check PostgreSQL
if command -v psql >/dev/null 2>&1; then
    echo "[PASS] PostgreSQL client available"
else
    echo "[WARN] PostgreSQL client not installed (optional for testing)"
fi

# Check scenario-authenticator (optional - has fallback)
if curl -sf http://localhost:15785/health >/dev/null 2>&1; then
    echo "[PASS] scenario-authenticator is running"
else
    echo "[INFO] scenario-authenticator not running (will use test mode)"
fi

# Check Redis (optional)
if command -v redis-cli >/dev/null 2>&1; then
    echo "[PASS] Redis client available"
else
    echo "[INFO] Redis not available (optional dependency)"
fi

# Check Go runtime
if command -v go >/dev/null 2>&1; then
    echo "[PASS] Go runtime available"
else
    echo "[WARN] Go runtime not available"
fi

# Check Node.js
if command -v node >/dev/null 2>&1; then
    echo "[PASS] Node.js runtime available ($(node --version))"
else
    echo "[FAIL] Node.js runtime required for UI server"
    exit 1
fi

echo ""
echo "[PASS] All critical dependencies satisfied"
exit 0
