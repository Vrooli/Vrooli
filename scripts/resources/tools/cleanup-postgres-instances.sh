#!/usr/bin/env bash
set -euo pipefail

# PostgreSQL Test Instance Cleanup Script
# Removes test/debug instances while preserving production instances

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
POSTGRES_DIR="$(dirname "$SCRIPT_DIR")/storage/postgres"
INSTANCES_DIR="$POSTGRES_DIR/instances"

# Source required utilities
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh"

echo "🧹 PostgreSQL Instance Cleanup"
echo "   Location: $INSTANCES_DIR"
echo

# Instances to remove (test/debug instances)
REMOVE_INSTANCES=(
    "test-client"
    "test-comprehensive"
    "test-comprehensive-2"
    "test-debug"
    "test-fix"
    "test-fix-2"
    "test-auto-port"
    "test-claude"
    "testing-template"
    "port-conflict-test"
    "port-conflict-test-2"
    "network-test"
    "network-test-new"
    "manual-network"
    "integration-test"
    "ecommerce-test"
    "client-test-ecommerce"
    "better-port-check"
)

# Instances to keep (production/template instances)
KEEP_INSTANCES=(
    "ai-realestate"
    "ecommerce-analytics"
    "real-estate"
    "minimal-template"
)

echo "📊 Instance Analysis:"
echo "   Instances to remove: ${#REMOVE_INSTANCES[@]}"
echo "   Instances to keep: ${#KEEP_INSTANCES[@]}"
echo

echo "🗑️  Removing test instances..."
echo "   Note: Some directories may require sudo due to container ownership"
echo
for instance in "${REMOVE_INSTANCES[@]}"; do
    if [ -d "$INSTANCES_DIR/$instance" ]; then
        echo "   Removing: $instance"
        # Try normal removal first, use sudo if it fails
        if ! trash::safe_remove "$INSTANCES_DIR/$instance" --no-confirm 2>/dev/null; then
            echo "     Using sudo for: $instance"
            sudo bash -c "source ${var_LIB_SYSTEM_DIR}/trash.sh && trash::safe_remove '$INSTANCES_DIR/$instance' --no-confirm"
        fi
    else
        echo "   Already gone: $instance"
    fi
done

echo
echo "✅ Cleanup complete!"
echo
echo "📦 Remaining instances:"
ls -la "$INSTANCES_DIR" | grep "^d" | grep -v "\.$" | awk '{print "   - " $NF}'