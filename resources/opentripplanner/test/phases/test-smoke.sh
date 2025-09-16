#!/usr/bin/env bash
# OpenTripPlanner Smoke Tests

set -euo pipefail

# Source configuration
source "$(dirname "${BASH_SOURCE[0]}")/../../config/defaults.sh"

echo "Running OpenTripPlanner smoke tests..."

# Test 1: Check if OpenTripPlanner is running
echo -n "1. Checking if service is running... "
if docker ps --format '{{.Names}}' | grep -q "^${OPENTRIPPLANNER_CONTAINER}$"; then
    echo "✓"
else
    echo "✗ (Service not running)"
    exit 1
fi

# Test 2: Check health endpoint
echo -n "2. Checking health endpoint... "
if timeout 5 curl -sf "http://localhost:${OTP_PORT}/otp" &>/dev/null; then
    echo "✓"
else
    echo "✗ (Health check failed)"
    exit 1
fi

# Test 3: Check API base endpoint
echo -n "3. Checking API base endpoint... "
response=$(timeout 5 curl -sf "http://localhost:${OTP_PORT}/otp" 2>/dev/null || echo "FAILED")
if [[ "$response" != "FAILED" ]]; then
    echo "✓"
else
    echo "✗ (API not responding)"
    exit 1
fi

# Test 4: Check routers endpoint
echo -n "4. Checking routers endpoint... "
if timeout 5 curl -sf "http://localhost:${OTP_PORT}/otp/routers" &>/dev/null; then
    echo "✓"
else
    echo "✗ (Routers endpoint failed)"
    exit 1
fi

# Test 5: Verify data directory exists
echo -n "5. Checking data directory... "
if [[ -d "$OTP_DATA_DIR" ]]; then
    echo "✓"
else
    echo "✗ (Data directory missing)"
    exit 1
fi

# Test 6: Check container logs for errors
echo -n "6. Checking for container errors... "
error_count=$(docker logs "${OPENTRIPPLANNER_CONTAINER}" 2>&1 | grep -iE "error|exception|fatal" | grep -v "No errors" | wc -l || echo "0")
if [[ "$error_count" -lt 5 ]]; then  # Allow some startup warnings
    echo "✓"
else
    echo "✗ (Found $error_count errors in logs)"
    exit 1
fi

echo ""
echo "Smoke tests completed successfully!"
exit 0