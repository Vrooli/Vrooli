#!/usr/bin/env bash
# Restic Resource - Integration Tests

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
readonly TEST_DIR="$(dirname "$SCRIPT_DIR")"
readonly RESOURCE_DIR="$(dirname "$TEST_DIR")"

# Source the CLI
source "${RESOURCE_DIR}/lib/core.sh"

echo "Running restic integration tests..."

# Test 1: Backup operation
echo -n "  Testing backup operation... "
if restic::backup --paths /tmp --tags test >/dev/null 2>&1; then
    echo "✓"
else
    echo "✗"
    echo "    Backup operation failed"
    exit 1
fi

# Test 2: List snapshots
echo -n "  Testing snapshot listing... "
if restic::snapshots >/dev/null 2>&1; then
    echo "✓"
else
    echo "✗"
    echo "    Snapshot listing failed"
    exit 1
fi

# Test 3: Restore operation
echo -n "  Testing restore operation... "
if restic::restore --snapshot latest --target /tmp/restore-test >/dev/null 2>&1; then
    echo "✓"
else
    echo "✗"
    echo "    Restore operation failed"
    exit 1
fi

# Test 4: Prune operation
echo -n "  Testing prune operation... "
if restic::prune --dry-run --keep-daily 7 >/dev/null 2>&1; then
    echo "✓"
else
    echo "✗"
    echo "    Prune operation failed"
    exit 1
fi

echo "  All integration tests passed!"
exit 0