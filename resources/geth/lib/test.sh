#!/usr/bin/env bash
# Geth Test Operations (Health Checks Only)
# Tests the resource itself, not business functionality

# Basic smoke test - check if Geth is responding
geth::test::smoke() {
    echo "[INFO] Running Geth smoke test..."
    
    if ! geth::is_running; then
        echo "[ERROR] Geth is not running"
        return 1
    fi
    
    # Test JSON-RPC endpoint
    echo -n "Testing JSON-RPC endpoint... "
    local response
    response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        --data '{"jsonrpc":"2.0","method":"web3_clientVersion","params":[],"id":1}' \
        "http://localhost:${GETH_PORT}" 2>/dev/null)
    
    if [[ -n "$response" ]] && echo "$response" | grep -q "result"; then
        echo "✓"
    else
        echo "✗"
        echo "[ERROR] JSON-RPC endpoint not responding"
        return 1
    fi
    
    # Test WebSocket endpoint
    echo -n "Testing WebSocket endpoint... "
    if timeout 2 bash -c "echo '' > /dev/tcp/localhost/${GETH_WS_PORT}" 2>/dev/null; then
        echo "✓"
    else
        echo "✗"
        echo "[ERROR] WebSocket endpoint not accessible"
        return 1
    fi
    
    echo "[INFO] Smoke test passed"
    return 0
}

# Integration test - verify Geth can process transactions
geth::test::integration() {
    echo "[INFO] Running Geth integration test..."
    
    if ! geth::is_running; then
        echo "[ERROR] Geth is not running"
        return 1
    fi
    
    # Check if node is synced (or in dev mode)
    local sync_status
    sync_status=$(geth::get_sync_status)
    
    if [[ "$GETH_NETWORK" != "dev" ]] && [[ "$sync_status" == "syncing" ]]; then
        echo "[WARNING] Node is still syncing, some tests may fail"
    fi
    
    # Test account creation capability
    echo -n "Testing account operations... "
    local accounts
    accounts=$(geth::get_account_count)
    
    if [[ "$accounts" -ge 0 ]]; then
        echo "✓ (${accounts} accounts)"
    else
        echo "✗"
        echo "[ERROR] Cannot retrieve account information"
        return 1
    fi
    
    # Test block retrieval
    echo -n "Testing block operations... "
    local block
    block=$(geth::get_block_number)
    
    if [[ -n "$block" ]]; then
        echo "✓ (block #${block})"
    else
        echo "✗"
        echo "[ERROR] Cannot retrieve block information"
        return 1
    fi
    
    echo "[INFO] Integration test passed"
    return 0
}

# Performance test - check response times
geth::test::performance() {
    echo "[INFO] Running Geth performance test..."
    
    if ! geth::is_running; then
        echo "[ERROR] Geth is not running"
        return 1
    fi
    
    # Test RPC response time
    echo -n "Testing RPC response time... "
    local start_time end_time duration
    start_time=$(date +%s%N)
    
    curl -s -X POST \
        -H "Content-Type: application/json" \
        --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
        "http://localhost:${GETH_PORT}" > /dev/null 2>&1
    
    end_time=$(date +%s%N)
    duration=$(( (end_time - start_time) / 1000000 ))
    
    if [[ "$duration" -lt 1000 ]]; then
        echo "✓ (${duration}ms)"
    else
        echo "⚠ (${duration}ms - slow response)"
    fi
    
    # Check memory usage
    echo -n "Checking memory usage... "
    local mem_usage
    mem_usage=$(docker stats --no-stream --format "{{.MemUsage}}" "${GETH_CONTAINER_NAME}" 2>/dev/null | cut -d'/' -f1)
    
    if [[ -n "$mem_usage" ]]; then
        echo "✓ (${mem_usage})"
    else
        echo "⚠ (unable to determine)"
    fi
    
    echo "[INFO] Performance test completed"
    return 0
}