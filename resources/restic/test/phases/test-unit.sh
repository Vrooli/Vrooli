#!/usr/bin/env bash
# Restic Resource - Unit Tests

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
readonly TEST_DIR="$(dirname "$SCRIPT_DIR")"
readonly RESOURCE_DIR="$(dirname "$TEST_DIR")"

echo "Running restic unit tests..."

# Test 1: Configuration loading
echo -n "  Testing configuration loading... "
if source "${RESOURCE_DIR}/config/defaults.sh" 2>/dev/null; then
    if [[ -n "${RESTIC_REPOSITORY:-}" ]]; then
        echo "✓"
    else
        echo "✗"
        echo "    Configuration variables not set"
        exit 1
    fi
else
    echo "✗"
    echo "    Failed to load configuration"
    exit 1
fi

# Test 2: Port registry integration
echo -n "  Testing port registry integration... "
source "${VROOLI_ROOT:-${HOME}/Vrooli}/scripts/resources/port_registry.sh"
RESTIC_PORT=$(ports::get_resource_port "restic")
if [[ "$RESTIC_PORT" == "8085" ]]; then
    echo "✓"
else
    echo "✗"
    echo "    Port registry integration failed (expected 8085, got $RESTIC_PORT)"
    exit 1
fi

# Test 3: Runtime configuration
echo -n "  Testing runtime configuration... "
if [[ -f "${RESOURCE_DIR}/config/runtime.json" ]]; then
    if jq -e '.startup_order' "${RESOURCE_DIR}/config/runtime.json" >/dev/null 2>&1; then
        echo "✓"
    else
        echo "✗"
        echo "    Invalid runtime.json structure"
        exit 1
    fi
else
    echo "✗"
    echo "    runtime.json not found"
    exit 1
fi

# Test 4: Schema validation
echo -n "  Testing schema validation... "
if [[ -f "${RESOURCE_DIR}/config/schema.json" ]]; then
    if jq -e '."$schema"' "${RESOURCE_DIR}/config/schema.json" >/dev/null 2>&1; then
        echo "✓"
    else
        echo "✗"
        echo "    Invalid schema.json structure"
        exit 1
    fi
else
    echo "✗"
    echo "    schema.json not found"
    exit 1
fi

echo "  All unit tests passed!"
exit 0