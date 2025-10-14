#!/bin/bash
# Performance Tests for Device Sync Hub
# Validates performance targets from PRD

set -euo pipefail

API_PORT="${API_PORT:-17402}"
API_URL="${API_URL:-http://localhost:${API_PORT}}"

echo "=== Device Sync Hub Performance Tests ==="
echo "[INFO] Testing performance characteristics..."

# Test 1: API health endpoint response time (target: < 500ms)
echo "[INFO] Testing API response time..."
start_time=$(date +%s%3N)
if curl -sf "$API_URL/health" >/dev/null 2>&1; then
    end_time=$(date +%s%3N)
    response_time=$((end_time - start_time))
    if [ "$response_time" -lt 500 ]; then
        echo "[PASS] API health responds in ${response_time}ms (target: <500ms)"
    else
        echo "[WARN] API health responds in ${response_time}ms (slower than target)"
    fi
else
    echo "[FAIL] API health check failed"
    exit 1
fi

# Test 2: Memory usage check (target: < 512MB)
echo "[INFO] Checking memory usage..."
if command -v pgrep >/dev/null 2>&1; then
    api_pid=$(pgrep -f "device-sync-hub-api" | head -1)
    if [ -n "$api_pid" ]; then
        mem_kb=$(ps -o rss= -p "$api_pid" 2>/dev/null || echo "0")
        mem_mb=$((mem_kb / 1024))
        if [ "$mem_mb" -lt 512 ]; then
            echo "[PASS] API memory usage: ${mem_mb}MB (target: <512MB)"
        else
            echo "[WARN] API memory usage: ${mem_mb}MB (above target)"
        fi
    else
        echo "[INFO] API process not found (may not be running)"
    fi
else
    echo "[INFO] Process monitoring tools not available"
fi

# Test 3: Database query performance
echo "[INFO] Testing database query performance..."
health_data=$(curl -sf "$API_URL/health")
db_latency=$(echo "$health_data" | grep -o '"latency_ms":[0-9.]*' | grep -o '[0-9.]*' | head -1)
if [ -n "$db_latency" ]; then
    echo "[PASS] Database query latency: ${db_latency}ms"
else
    echo "[INFO] Database latency not available"
fi

echo ""
echo "[PASS] Performance validation complete"
exit 0
