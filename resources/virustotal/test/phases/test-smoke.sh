#!/bin/bash
# VirusTotal Smoke Test Phase
# Quick validation that service is alive and responding

set -euo pipefail

# Configuration
RESOURCE_NAME="virustotal"
MAX_DURATION=30  # v2.0 contract requirement
HEALTH_URL="http://localhost:${VIRUSTOTAL_PORT:-8290}/api/health"

# Start timer
START_TIME=$(date +%s)

echo "Starting smoke tests for ${RESOURCE_NAME}..."

# Test 1: Check if service is running
echo -n "1. Checking if service container is running... "
if docker ps --format "table {{.Names}}" | grep -q "vrooli-${RESOURCE_NAME}"; then
    echo "PASS"
else
    echo "FAIL: Service container not found"
    exit 1
fi

# Test 2: Health endpoint responds
echo -n "2. Testing health endpoint availability... "
if timeout 5 curl -sf "${HEALTH_URL}" >/dev/null 2>&1; then
    echo "PASS"
else
    echo "FAIL: Health endpoint not responding"
    exit 1
fi

# Test 3: Health endpoint returns valid JSON
echo -n "3. Validating health endpoint response format... "
HEALTH_RESPONSE=$(timeout 5 curl -sf "${HEALTH_URL}" 2>/dev/null || echo "{}")
if echo "$HEALTH_RESPONSE" | jq -e '.status' >/dev/null 2>&1; then
    echo "PASS"
else
    echo "FAIL: Health endpoint returned invalid JSON"
    echo "Response: $HEALTH_RESPONSE"
    exit 1
fi

# Test 4: Check service status
echo -n "4. Checking service status... "
STATUS=$(echo "$HEALTH_RESPONSE" | jq -r '.status')
if [[ "$STATUS" == "healthy" ]] || [[ "$STATUS" == "degraded" ]]; then
    echo "PASS (Status: $STATUS)"
    if [[ "$STATUS" == "degraded" ]]; then
        echo "   Warning: Service is degraded. Check API key configuration."
    fi
else
    echo "FAIL: Unexpected status: $STATUS"
    exit 1
fi

# Test 5: API stats endpoint
echo -n "5. Testing stats endpoint... "
if timeout 5 curl -sf "http://localhost:${VIRUSTOTAL_PORT:-8290}/api/stats" >/dev/null 2>&1; then
    echo "PASS"
else
    echo "FAIL: Stats endpoint not responding"
    exit 1
fi

# Test 6: Port accessibility
echo -n "6. Verifying port ${VIRUSTOTAL_PORT:-8290} is accessible... "
if nc -z localhost ${VIRUSTOTAL_PORT:-8290} 2>/dev/null; then
    echo "PASS"
else
    echo "FAIL: Port not accessible"
    exit 1
fi

# Test 7: Docker health check
echo -n "7. Checking Docker health status... "
DOCKER_HEALTH=$(docker inspect --format='{{.State.Health.Status}}' "vrooli-${RESOURCE_NAME}" 2>/dev/null || echo "none")
if [[ "$DOCKER_HEALTH" == "healthy" ]] || [[ "$DOCKER_HEALTH" == "starting" ]]; then
    echo "PASS (Docker health: $DOCKER_HEALTH)"
elif [[ "$DOCKER_HEALTH" == "none" ]]; then
    echo "PASS (No Docker health check configured)"
else
    echo "FAIL: Docker health check failed: $DOCKER_HEALTH"
    exit 1
fi

# End timer and check duration
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

echo ""
echo "Smoke tests completed in ${DURATION} seconds"

if [ $DURATION -gt $MAX_DURATION ]; then
    echo "ERROR: Smoke tests exceeded ${MAX_DURATION} second limit"
    exit 1
fi

echo "All smoke tests passed successfully!"
exit 0