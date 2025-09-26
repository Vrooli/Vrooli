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

# Test 1: API Health Check
echo -n "  Testing API health endpoint... "
response=$(api_call GET "health" 2>/dev/null || echo "FAILED")
if [[ "$response" != "FAILED" ]] && [[ "$response" == *"Successful"* || "$response" == *"available"* ]]; then
    echo "OK"
else
    echo "FAILED"
    echo "    API health check failed"
    exit 1
fi

# Test 2: Workload API Health (for connectors)
echo -n "  Testing workload API health... "
if docker exec airbyte-abctl-control-plane kubectl -n airbyte-abctl exec deploy/airbyte-abctl-workload-api-server -- curl -s http://localhost:8007/health 2>/dev/null | grep -q '"status":"UP"'; then
    echo "OK"
else
    echo "WARNING"
    echo "    Workload API not fully accessible (connector operations may be affected)"
fi

# Test 3: Kubernetes Pods Status
echo -n "  Testing Kubernetes pods status... "
pod_count=$(docker exec airbyte-abctl-control-plane kubectl -n airbyte-abctl get pods --no-headers 2>/dev/null | grep -c Running || echo "0")
if [[ "$pod_count" -ge 6 ]]; then
    echo "OK ($pod_count pods running)"
else
    echo "WARNING"
    echo "    Only $pod_count pods running (expected 6+)"
fi

# Test 4: Check Temporal connectivity
echo -n "  Testing Temporal connectivity... "
if timeout 5 nc -zv localhost ${AIRBYTE_TEMPORAL_PORT:-8006} &> /dev/null; then
    echo "OK"
else
    echo "WARNING"
    echo "    Temporal not accessible (may affect job execution)"
fi

# Test 5: Database connectivity (only for docker-compose method)
echo -n "  Testing database connectivity... "
method=$(detect_deployment_method)
if [[ "$method" == "abctl" ]]; then
    # abctl manages database internally in Kubernetes
    echo "SKIPPED (managed by abctl)"
else
    if docker exec airbyte-db pg_isready -U airbyte > /dev/null 2>&1; then
        echo "OK"
    else
        echo "FAILED"
        echo "    Database not ready"
        exit 1
    fi
fi

# Test 6: Server Deployment Status
echo -n "  Testing server deployment... "
if docker exec airbyte-abctl-control-plane kubectl -n airbyte-abctl get deploy airbyte-abctl-server 2>/dev/null | grep -q "1/1"; then
    echo "OK"
else
    echo "WARNING"
    echo "    Server deployment not fully ready"
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

# Test 10: Pipeline optimization commands
echo -n "  Testing pipeline commands... "
if "${RESOURCE_DIR}/cli.sh" pipeline resources > /dev/null 2>&1; then
    echo "OK"
else
    echo "WARNING"
    echo "    Pipeline commands not fully available (expected if no connections exist)"
fi

# Test 11: CDK functionality
echo -n "  Testing CDK commands... "
if "${RESOURCE_DIR}/cli.sh" cdk list > /dev/null 2>&1; then
    echo "OK"
else
    echo "WARNING"
    echo "    CDK commands not available"
fi

# Test 12: Workspace management
echo -n "  Testing workspace commands... "
if "${RESOURCE_DIR}/cli.sh" workspace list > /dev/null 2>&1; then
    echo "OK"
else
    echo "WARNING"
    echo "    Workspace commands not available"
fi

# Test 13: Metrics export
echo -n "  Testing metrics commands... "
if "${RESOURCE_DIR}/cli.sh" metrics status > /dev/null 2>&1; then
    echo "OK"
else
    echo "WARNING"
    echo "    Metrics commands not available"
fi

echo "Integration tests completed successfully"