#!/usr/bin/env bash
# Restic Resource - Smoke Tests

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
readonly TEST_DIR="$(dirname "$SCRIPT_DIR")"
readonly RESOURCE_DIR="$(dirname "$TEST_DIR")"

# Source the CLI
source "${RESOURCE_DIR}/lib/core.sh"

echo "Running restic smoke tests..."

# Test 1: Check if health check works
echo -n "  Testing health check... "
if restic::health_check; then
    echo "✓"
else
    # If not running, try to start it first
    restic::start --wait >/dev/null 2>&1 || true
    if restic::health_check; then
        echo "✓"
    else
        echo "✗"
        echo "    Health check failed"
        exit 1
    fi
fi

# Test 2: Check status command
echo -n "  Testing status command... "
if restic::status >/dev/null 2>&1; then
    echo "✓"
else
    echo "✗"
    echo "    Status command failed"
    exit 1
fi

# Test 3: Check info command
echo -n "  Testing info command... "
if restic::show_info >/dev/null 2>&1; then
    echo "✓"
else
    echo "✗"
    echo "    Info command failed"
    exit 1
fi

# Test 4: Check JSON output
echo -n "  Testing JSON output... "
if restic::status --json | jq . >/dev/null 2>&1; then
    echo "✓"
else
    echo "✗"
    echo "    JSON output invalid"
    exit 1
fi

echo "  All smoke tests passed!"
exit 0