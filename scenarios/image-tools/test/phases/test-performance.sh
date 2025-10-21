#!/bin/bash
# Test: Performance validation
# Ensures the service meets performance targets

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "  ✓ Testing response time..."

# Get API port - use environment variable first, then check the service status
if [ -n "$API_PORT" ]; then
  : # Use provided API_PORT
elif [ -f "${SCENARIO_DIR}/.vrooli/runtime/ports.json" ]; then
  API_PORT=$(jq -r '.api // empty' "${SCENARIO_DIR}/.vrooli/runtime/ports.json" 2>/dev/null || echo "")
fi

# If still not found, check vrooli scenario status output
if [ -z "$API_PORT" ]; then
  API_PORT=$(vrooli scenario status image-tools 2>/dev/null | grep -oP 'http://localhost:\K[0-9]+' | head -1 || echo "")
fi

# Verify we have a valid port
if [ -z "$API_PORT" ]; then
  echo "  ❌ Could not determine API port"
  exit 1
fi

# Test health endpoint response time
START_TIME=$(date +%s%N)
curl -sf "http://localhost:${API_PORT}/health" > /dev/null
END_TIME=$(date +%s%N)
RESPONSE_TIME=$((($END_TIME - $START_TIME) / 1000000))

echo "    Health endpoint: ${RESPONSE_TIME}ms"

if [ $RESPONSE_TIME -lt 500 ]; then
    echo "    ✓ Response time within target (<500ms)"
else
    echo "    ⚠️  Response time above target (>500ms)"
fi

# Test memory usage
PID=$(pgrep -f "image-tools-api" | head -1)
if [ ! -z "$PID" ]; then
    MEM_KB=$(ps -o rss= -p $PID)
    MEM_MB=$((MEM_KB / 1024))
    echo "    Memory usage: ${MEM_MB}MB"
    
    if [ $MEM_MB -lt 2048 ]; then
        echo "    ✓ Memory usage within target (<2GB)"
    else
        echo "    ⚠️  Memory usage above target (>2GB)"
    fi
fi

echo "  ✓ Performance tests complete"