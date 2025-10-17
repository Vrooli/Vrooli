#!/bin/bash

set -e

echo "=== Performance Tests ==="

# Get API port dynamically
API_PORT=$(lsof -ti:19313 2>/dev/null | head -1 | xargs -I {} lsof -nP -p {} 2>/dev/null | grep LISTEN | awk '{print $9}' | cut -d: -f2 | head -1)
if [ -z "$API_PORT" ]; then
    API_PORT="19313"  # Fallback
fi

API_URL="http://localhost:${API_PORT}"

# Function to measure response time
measure_response_time() {
    local endpoint=$1
    local method=${2:-GET}
    local data=${3:-}

    local start=$(date +%s%N)
    if [ -z "$data" ]; then
        curl -sf -X "$method" "${API_URL}${endpoint}" > /dev/null
    else
        curl -sf -X "$method" "${API_URL}${endpoint}" \
            -H "Content-Type: application/json" \
            -d "$data" > /dev/null
    fi
    local end=$(date +%s%N)

    # Calculate milliseconds
    echo $(( (end - start) / 1000000 ))
}

# Test health endpoint performance
echo "Testing health endpoint performance..."
health_time=$(measure_response_time "/health")
echo "Health endpoint: ${health_time}ms"
if [ "$health_time" -lt 200 ]; then
    echo "✅ Health endpoint under 200ms target"
else
    echo "⚠️  Health endpoint slower than 200ms target (${health_time}ms)"
fi

# Test contacts endpoint performance
echo "Testing contacts endpoint performance..."
contacts_time=$(measure_response_time "/api/v1/contacts?limit=50")
echo "Contacts endpoint (50 items): ${contacts_time}ms"
if [ "$contacts_time" -lt 200 ]; then
    echo "✅ Contacts endpoint under 200ms target"
else
    echo "⚠️  Contacts endpoint slower than 200ms target (${contacts_time}ms)"
fi

# Test search endpoint performance
echo "Testing search endpoint performance..."
search_time=$(measure_response_time "/api/v1/search" "POST" '{"query":"test","limit":10}')
echo "Search endpoint: ${search_time}ms"
if [ "$search_time" -lt 500 ]; then
    echo "✅ Search endpoint under 500ms target"
else
    echo "⚠️  Search endpoint slower than 500ms target (${search_time}ms)"
fi

# Test CLI performance
echo "Testing CLI performance..."
if command -v contact-book > /dev/null 2>&1; then
    start=$(date +%s%N)
    contact-book list --limit 10 > /dev/null 2>&1
    end=$(date +%s%N)
    cli_time=$(( (end - start) / 1000000 ))
    echo "CLI list command: ${cli_time}ms"
    if [ "$cli_time" -lt 2000 ]; then
        echo "✅ CLI under 2s target"
    else
        echo "⚠️  CLI slower than 2s target (${cli_time}ms)"
    fi
else
    echo "⚠️  CLI not available, skipping CLI performance test"
fi

echo "✅ Performance tests complete"
