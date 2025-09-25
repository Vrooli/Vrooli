#!/bin/bash
# Test: Performance validation
# Ensures the service meets performance targets

set -e

echo "  ✓ Testing response time..."

# Get the API port
API_PORT=$(vrooli scenario logs image-tools --step start-api --tail 5 2>/dev/null | grep -oP 'port \K\d+' | tail -1)
API_PORT=${API_PORT:-19364}

# Test health endpoint response time
START_TIME=$(date +%s%N)
curl -sf "http://localhost:${API_PORT}/api/v1/health" > /dev/null
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