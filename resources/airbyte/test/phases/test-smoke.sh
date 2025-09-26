#!/bin/bash
# Smoke tests for Airbyte - Quick health validation (<30s)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

source "${RESOURCE_DIR}/lib/core.sh"

echo "Running Airbyte smoke tests..."

# Test 1: Check if Docker is available
echo -n "  Checking Docker availability... "
if command -v docker &> /dev/null; then
    echo "OK"
else
    echo "FAILED"
    echo "    Docker is not installed or not in PATH"
    exit 1
fi

# Test 2: Check if services are running
echo -n "  Checking Airbyte services... "
method=$(detect_deployment_method)
if [[ "$method" == "abctl" ]]; then
    # For abctl, check if cluster is running
    ABCTL="${DATA_DIR}/abctl"
    if [[ -f "$ABCTL" ]] && "$ABCTL" local status &> /dev/null; then
        echo "OK (Kubernetes cluster running)"
    else
        echo "FAILED"
        echo "    Airbyte cluster is not running"
        exit 1
    fi
else
    # For docker-compose, check containers
    running_count=$(docker ps --filter "name=airbyte-" --format "{{.Names}}" | wc -l)
    if [[ $running_count -gt 0 ]]; then
        echo "OK ($running_count services running)"
    else
        echo "FAILED"
        echo "    No Airbyte services are running"
        exit 1
    fi
fi

# Test 3: Health check
echo -n "  Checking API health... "
if [[ "$method" == "abctl" ]]; then
    # For abctl, check pod health  
    unhealthy_pods=0
    while read -r pod; do
        if [[ ! "$pod" =~ (Running|Completed) ]]; then
            ((unhealthy_pods++))
        fi
    done < <(docker exec airbyte-abctl-control-plane kubectl -n airbyte-abctl get pods --no-headers)
    if [[ "$unhealthy_pods" -eq 0 ]]; then
        echo "OK (all pods healthy)"
    else
        echo "FAILED"
        echo "    Some pods are not running (count: $unhealthy_pods)"
        exit 1
    fi
else
    # For docker-compose, check traditional endpoint
    if timeout 5 curl -sf "http://localhost:${AIRBYTE_SERVER_PORT:-8003}/api/v1/health" > /dev/null 2>&1; then
        echo "OK"
    else
        echo "FAILED"
        echo "    Health endpoint not responding"
        exit 1
    fi
fi

# Test 4: Webapp accessibility
echo -n "  Checking webapp accessibility... "
if timeout 5 curl -sf "http://localhost:${AIRBYTE_WEBAPP_PORT:-8002}" > /dev/null 2>&1; then
    echo "OK"
else
    echo "WARNING"
    echo "    Webapp not accessible (may still be starting)"
fi

echo "Smoke tests completed successfully"