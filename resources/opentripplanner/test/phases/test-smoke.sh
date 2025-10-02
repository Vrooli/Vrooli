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

# Test 2: Check health endpoint (root page serves debug UI)
echo -n "2. Checking health endpoint... "
if timeout 5 curl -sf "http://localhost:${OTP_PORT}/" | grep -q "OTP Debug" &>/dev/null; then
    echo "✓"
else
    echo "✗ (Health check failed)"
    exit 1
fi

# Test 3: Check if graph is loaded
echo -n "3. Checking graph loaded... "
if [[ -f "${OTP_DATA_DIR}/graph.obj" ]]; then
    echo "✓"
else
    echo "✗ (Graph not built)"
    exit 1
fi

# Test 4: Check container is healthy
echo -n "4. Checking container health... "
container_status=$(docker inspect "${OPENTRIPPLANNER_CONTAINER}" --format='{{.State.Status}}' 2>/dev/null || echo "not_found")
if [[ "$container_status" == "running" ]]; then
    echo "✓"
else
    echo "✗ (Container status: $container_status)"
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
# Count critical errors (exclude known safe ones)
# Use || true to handle grep not finding matches with pipefail
error_count=$(docker logs "${OPENTRIPPLANNER_CONTAINER}" 2>&1 | grep -iE "error|exception|fatal" | grep -v "No errors" | grep -v "Parameter error" | wc -l || true)
if [[ ${error_count} -lt 5 ]]; then  # Allow some startup warnings
    echo "✓"
else
    echo "✗ (Found ${error_count} errors in logs)"
    exit 1
fi

echo ""
echo "Smoke tests completed successfully!"
exit 0