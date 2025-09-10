#!/bin/bash
# Integration tests for Airbyte - End-to-end functionality (<120s)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

source "${RESOURCE_DIR}/lib/core.sh"

echo "Running Airbyte integration tests..."

API_URL="http://localhost:${AIRBYTE_SERVER_PORT:-8001}/api/v1"

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

echo "Integration tests completed successfully"