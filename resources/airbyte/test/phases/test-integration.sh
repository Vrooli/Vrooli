#!/bin/bash
# Integration tests for Airbyte - End-to-end functionality (<120s)
# Tests API endpoints, connector management, and sync monitoring

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/config/defaults.sh"

echo "Running Airbyte integration tests..."

API_URL="http://localhost:${AIRBYTE_SERVER_PORT}/api/v1"

# Test 1: List source definitions
echo -n "  Testing source definitions API... "
response=$(curl -sf "${API_URL}/source_definitions/list" 2>/dev/null || echo "FAILED")
if [[ "$response" != "FAILED" ]] && echo "$response" | jq -e '.sourceDefinitions' > /dev/null 2>&1; then
    count=$(echo "$response" | jq '.sourceDefinitions | length')
    echo "OK ($count sources available)"
else
    echo "FAILED"
    echo "    Could not retrieve source definitions"
    exit 1
fi

# Test 2: List destination definitions
echo -n "  Testing destination definitions API... "
response=$(curl -sf "${API_URL}/destination_definitions/list" 2>/dev/null || echo "FAILED")
if [[ "$response" != "FAILED" ]] && echo "$response" | jq -e '.destinationDefinitions' > /dev/null 2>&1; then
    count=$(echo "$response" | jq '.destinationDefinitions | length')
    echo "OK ($count destinations available)"
else
    echo "FAILED"
    echo "    Could not retrieve destination definitions"
    exit 1
fi

# Test 3: List workspaces
echo -n "  Testing workspaces API... "
response=$(curl -sf "${API_URL}/workspaces/list" 2>/dev/null || echo "FAILED")
if [[ "$response" != "FAILED" ]] && echo "$response" | jq -e '.workspaces' > /dev/null 2>&1; then
    echo "OK"
else
    echo "FAILED"
    echo "    Could not retrieve workspaces"
    exit 1
fi

# Test 4: Check Temporal connectivity
echo -n "  Testing Temporal connectivity... "
if timeout 5 nc -zv localhost ${AIRBYTE_TEMPORAL_PORT:-8006} &> /dev/null; then
    echo "OK"
else
    echo "WARNING"
    echo "    Temporal not accessible (may affect job execution)"
fi

# Test 5: Database connectivity
echo -n "  Testing database connectivity... "
if docker exec airbyte-db pg_isready -U airbyte > /dev/null 2>&1; then
    echo "OK"
else
    echo "FAILED"
    echo "    Database not ready"
    exit 1
fi

# Test 6: Jobs API (sync history)
echo -n "  Testing jobs API... "
response=$(curl -sf "${API_URL}/jobs/list" 2>/dev/null || echo "FAILED")
if [[ "$response" != "FAILED" ]]; then
    echo "OK"
else
    echo "WARNING"
    echo "    Jobs API not accessible (expected if no syncs have run)"
fi

# Test 7: Enhanced health monitoring
echo -n "  Testing enhanced health monitoring... "
if check_health true > /dev/null 2>&1; then
    echo "OK"
else
    echo "WARNING"
    echo "    Some services may not be fully healthy"
fi

# Test 8: CLI content management
echo -n "  Testing CLI content commands... "
if "${RESOURCE_DIR}/cli.sh" content list --type sources > /dev/null 2>&1; then
    echo "OK"
else
    echo "FAILED"
    echo "    CLI content management failed"
    exit 1
fi

# Test 9: Sync status monitoring
echo -n "  Testing sync status monitoring... "
if check_sync_status > /dev/null 2>&1; then
    echo "OK"
else
    echo "WARNING"
    echo "    Sync monitoring returned warnings (expected if no syncs configured)"
fi

echo "Integration tests completed successfully"