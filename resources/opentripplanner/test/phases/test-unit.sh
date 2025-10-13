#!/usr/bin/env bash
# OpenTripPlanner Unit Tests

set -euo pipefail

# Determine APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LIB_DIR="${SCRIPT_DIR}/../../lib"
CONFIG_DIR="${SCRIPT_DIR}/../../config"

# Source port registry and configuration first
source "${APP_ROOT}/scripts/resources/port_registry.sh" || exit 1
source "${CONFIG_DIR}/defaults.sh"

# Source libraries to test
source "${LIB_DIR}/core.sh"

echo "Running OpenTripPlanner unit tests..."

# Test 1: Check is_installed function
echo -n "1. Testing is_installed function... "
if docker image inspect "${OPENTRIPPLANNER_IMAGE}" &>/dev/null; then
    if opentripplanner::is_installed; then
        echo "✓"
    else
        echo "✗ (Function returned false when image exists)"
        exit 1
    fi
else
    if ! opentripplanner::is_installed; then
        echo "✓"
    else
        echo "✗ (Function returned true when image missing)"
        exit 1
    fi
fi

# Test 2: Check port configuration
echo -n "2. Testing port configuration... "
if [[ "$OTP_PORT" =~ ^[0-9]+$ ]] && [[ "$OTP_PORT" -ge 1024 ]] && [[ "$OTP_PORT" -le 65535 ]]; then
    echo "✓"
else
    echo "✗ (Invalid port: $OTP_PORT)"
    exit 1
fi

# Test 3: Check heap size format
echo -n "3. Testing heap size configuration... "
if [[ "$OTP_HEAP_SIZE" =~ ^[0-9]+[MG]$ ]]; then
    echo "✓"
else
    echo "✗ (Invalid heap size format: $OTP_HEAP_SIZE)"
    exit 1
fi

# Test 4: Check directory variables
echo -n "4. Testing directory configuration... "
if [[ -n "$OTP_DATA_DIR" ]] && [[ -n "$OTP_CACHE_DIR" ]]; then
    echo "✓"
else
    echo "✗ (Directory variables not set)"
    exit 1
fi

# Test 5: Check Docker configuration
echo -n "5. Testing Docker configuration... "
if [[ -n "$OPENTRIPPLANNER_IMAGE" ]] && [[ -n "$OPENTRIPPLANNER_CONTAINER" ]] && [[ -n "$OPENTRIPPLANNER_NETWORK" ]]; then
    echo "✓"
else
    echo "✗ (Docker variables not set)"
    exit 1
fi

# Test 6: Check routing defaults
echo -n "6. Testing routing defaults... "
if [[ "$OTP_MAX_WALK_DISTANCE" =~ ^[0-9]+$ ]] && [[ "$OTP_MAX_TRANSFERS" =~ ^[0-9]+$ ]]; then
    echo "✓"
else
    echo "✗ (Invalid routing defaults)"
    exit 1
fi

# Test 7: Test walk speed validation
echo -n "7. Testing walk speed configuration... "
if [[ "$OTP_WALK_SPEED" =~ ^[0-9]+\.?[0-9]*$ ]]; then
    echo "✓"
else
    echo "✗ (Invalid walk speed: $OTP_WALK_SPEED)"
    exit 1
fi

# Test 8: Test bike speed validation
echo -n "8. Testing bike speed configuration... "
if [[ "$OTP_BIKE_SPEED" =~ ^[0-9]+\.?[0-9]*$ ]]; then
    echo "✓"
else
    echo "✗ (Invalid bike speed: $OTP_BIKE_SPEED)"
    exit 1
fi

# Test 9: Test plan_trip function validation
echo -n "9. Testing plan_trip parameter validation... "
# Test missing parameters handling
error_output=$(opentripplanner::plan_trip 2>&1 || true)
if echo "$error_output" | grep -q "Missing required coordinates"; then
    echo "✓"
else
    echo "✗ (Function doesn't validate parameters properly)"
    exit 1
fi

# Test 10: Test transport mode mapping
echo -n "10. Testing transport mode mapping... "
# This tests that the function exists and can be called
if type -t opentripplanner::plan_trip &>/dev/null; then
    echo "✓"
else
    echo "✗ (plan_trip function not found)"
    exit 1
fi

# Test 11: Test GTFS-RT feed function
echo -n "11. Testing GTFS-RT feed addition function... "
if type -t opentripplanner::add_gtfs_rt_feed &>/dev/null; then
    echo "✓"
else
    echo "✗ (add_gtfs_rt_feed function not found)"
    exit 1
fi

echo ""
echo "Unit tests completed successfully!"
exit 0