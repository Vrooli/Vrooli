#!/bin/bash
# Performance Test Suite for Network Tools
# Tests response times, throughput, and resource usage

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
API_PORT="${API_PORT:-17125}"
API_BASE="http://localhost:${API_PORT}"
TESTS_PASSED=0
TESTS_FAILED=0

echo "======================================================================"
echo "Network Tools Performance Test Suite"
echo "API Endpoint: ${API_BASE}"
echo "======================================================================"
echo ""

# Helper function to measure response time
measure_response_time() {
    local endpoint="$1"
    local method="$2"
    local data="${3:-}"

    if [[ -n "$data" ]]; then
        time_ms=$(curl -s -o /dev/null -w "%{time_total}" -X "$method" \
            "${API_BASE}${endpoint}" \
            -H "Content-Type: application/json" \
            -d "$data" 2>/dev/null)
    else
        time_ms=$(curl -s -o /dev/null -w "%{time_total}" -X "$method" \
            "${API_BASE}${endpoint}" 2>/dev/null)
    fi

    echo "$time_ms"
}

# Test 1: Health check response time
echo -e "${BLUE}Testing: Health endpoint performance${NC}"
health_time=$(measure_response_time "/health" "GET")
health_time_ms=$(echo "$health_time * 1000" | bc)

if (( $(echo "$health_time_ms < 100" | bc -l) )); then
    echo -e "${GREEN}  ✓ Health check < 100ms (${health_time_ms}ms)${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}  ✗ Health check too slow (${health_time_ms}ms > 100ms)${NC}"
    ((TESTS_FAILED++))
fi

# Test 2: DNS query performance
echo -e "${BLUE}Testing: DNS query performance${NC}"
dns_time=$(measure_response_time "/api/v1/network/dns" "POST" \
    '{"query":"google.com","record_type":"A"}')
dns_time_ms=$(echo "$dns_time * 1000" | bc)

if (( $(echo "$dns_time_ms < 500" | bc -l) )); then
    echo -e "${GREEN}  ✓ DNS query < 500ms (${dns_time_ms}ms)${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${YELLOW}  ⚠ DNS query slow (${dns_time_ms}ms > 500ms)${NC}"
    ((TESTS_FAILED++))
fi

# Test 3: HTTP request performance
echo -e "${BLUE}Testing: HTTP request performance${NC}"
http_time=$(measure_response_time "/api/v1/network/http" "POST" \
    '{"url":"https://httpbin.org/get","method":"GET"}')
http_time_ms=$(echo "$http_time * 1000" | bc)

if (( $(echo "$http_time_ms < 2000" | bc -l) )); then
    echo -e "${GREEN}  ✓ HTTP request < 2000ms (${http_time_ms}ms)${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${YELLOW}  ⚠ HTTP request slow (${http_time_ms}ms > 2000ms)${NC}"
    ((TESTS_FAILED++))
fi

# Test 4: Concurrent request handling
echo -e "${BLUE}Testing: Concurrent request handling${NC}"
start_time=$(date +%s%N)

# Send 10 concurrent health requests
for i in {1..10}; do
    curl -s "${API_BASE}/health" >/dev/null 2>&1 &
done
wait

end_time=$(date +%s%N)
total_time_ms=$(( (end_time - start_time) / 1000000 ))

if (( total_time_ms < 1000 )); then
    echo -e "${GREEN}  ✓ 10 concurrent requests < 1s (${total_time_ms}ms)${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${YELLOW}  ⚠ Concurrent requests slow (${total_time_ms}ms > 1000ms)${NC}"
    ((TESTS_FAILED++))
fi

# Test 5: Port scan performance (small range)
echo -e "${BLUE}Testing: Port scan performance${NC}"
scan_time=$(measure_response_time "/api/v1/network/scan" "POST" \
    '{"target":"localhost","scan_type":"port","ports":[80,443,8080]}')
scan_time_ms=$(echo "$scan_time * 1000" | bc)

if (( $(echo "$scan_time_ms < 3500" | bc -l) )); then
    echo -e "${GREEN}  ✓ Port scan (3 ports) < 3.5s (${scan_time_ms}ms)${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${YELLOW}  ⚠ Port scan slow (${scan_time_ms}ms > 3500ms)${NC}"
    ((TESTS_FAILED++))
fi

# Test 6: SSL validation performance
echo -e "${BLUE}Testing: SSL validation performance${NC}"
ssl_time=$(measure_response_time "/api/v1/network/ssl/validate" "POST" \
    '{"url":"https://github.com"}')
ssl_time_ms=$(echo "$ssl_time * 1000" | bc)

if (( $(echo "$ssl_time_ms < 1000" | bc -l) )); then
    echo -e "${GREEN}  ✓ SSL validation < 1s (${ssl_time_ms}ms)${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${YELLOW}  ⚠ SSL validation slow (${ssl_time_ms}ms > 1000ms)${NC}"
    ((TESTS_FAILED++))
fi

# Test 7: Throughput test - requests per second
echo -e "${BLUE}Testing: Throughput (requests per second)${NC}"
start_time=$(date +%s)
request_count=0

# Send requests for 5 seconds
while [ $(($(date +%s) - start_time)) -lt 5 ]; do
    curl -s "${API_BASE}/health" >/dev/null 2>&1 &
    ((request_count++))

    # Limit to prevent overwhelming
    if [ $((request_count % 10)) -eq 0 ]; then
        sleep 0.1
    fi
done
wait

rps=$((request_count / 5))
echo -e "${GREEN}  ✓ Throughput: ${rps} requests/second${NC}"

if [ $rps -gt 50 ]; then
    echo -e "${GREEN}  ✓ Throughput exceeds 50 rps target${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${YELLOW}  ⚠ Throughput below target (${rps} < 50 rps)${NC}"
    ((TESTS_FAILED++))
fi

echo ""
echo "======================================================================"
echo "Performance Test Summary"
echo "======================================================================"
echo -e "${GREEN}Passed: ${TESTS_PASSED}${NC}"
echo -e "${YELLOW}Warnings: ${TESTS_FAILED}${NC}"

echo ""
echo "Performance Benchmarks:"
echo "  Health Check: ${health_time_ms}ms (target: <100ms)"
echo "  DNS Query: ${dns_time_ms}ms (target: <500ms)"
echo "  HTTP Request: ${http_time_ms}ms (target: <2000ms)"
echo "  SSL Validation: ${ssl_time_ms}ms (target: <1000ms)"
echo "  Port Scan (3): ${scan_time_ms}ms (target: <3500ms)"
echo "  Throughput: ${rps} rps (target: >50 rps)"

if [[ ${TESTS_FAILED} -gt 3 ]]; then
    echo -e "${RED}⚠️  Multiple performance issues detected. Optimization needed.${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Performance tests completed${NC}"
exit 0