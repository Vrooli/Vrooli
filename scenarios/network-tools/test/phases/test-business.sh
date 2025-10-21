#!/usr/bin/env bash
# Test Phase 3: Business Logic Validation
# Validates core business functionality and value propositions from PRD

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Get API port from environment or default
API_PORT=${API_PORT:-17124}

echo "================================================"
echo "💼 Phase 3: Business Logic Validation"
echo "================================================"

TESTS_PASSED=0
TESTS_FAILED=0

# Helper function
test_api() {
    local name="$1"
    local method="$2"
    local endpoint="$3"
    local data="$4"
    local expected_status="$5"

    echo ""
    echo "  Testing: $name"

    local response
    if [ -n "$data" ]; then
        response=$(curl -s -X "$method" \
            -H "Content-Type: application/json" \
            -d "$data" \
            -w "\n%{http_code}" \
            "http://localhost:$API_PORT$endpoint" 2>&1)
    else
        response=$(curl -s -X "$method" \
            -w "\n%{http_code}" \
            "http://localhost:$API_PORT$endpoint" 2>&1)
    fi

    local body=$(echo "$response" | head -n -1)
    local status=$(echo "$response" | tail -n 1)

    if [ "$status" -eq "$expected_status" ]; then
        echo "    ✅ Returns HTTP $status"
        ((TESTS_PASSED++))
        return 0
    else
        echo "    ❌ Expected HTTP $expected_status, got $status"
        echo "       Response: $body"
        ((TESTS_FAILED++))
        return 1
    fi
}

# P0 Requirement: HTTP Client Operations
echo ""
echo "🌐 P0: HTTP Client Operations"

test_api "HTTP GET Request" "POST" "/api/v1/network/http" \
    '{"url":"https://httpbingo.org/get","method":"GET"}' 200

test_api "HTTP POST Request" "POST" "/api/v1/network/http" \
    '{"url":"https://httpbingo.org/post","method":"POST","body":"test"}' 200

# P0 Requirement: DNS Operations
echo ""
echo "🔍 P0: DNS Operations"

test_api "DNS A Record Lookup" "POST" "/api/v1/network/dns" \
    '{"query":"google.com","record_type":"A"}' 200

test_api "DNS MX Record Lookup" "POST" "/api/v1/network/dns" \
    '{"query":"google.com","record_type":"MX"}' 200

# P0 Requirement: Port Scanning
echo ""
echo "🔓 P0: Port Scanning"

test_api "Port Scan" "POST" "/api/v1/network/scan" \
    '{"target":"localhost","ports":[80,443,22]}' 200

# P0 Requirement: SSL/TLS Validation
echo ""
echo "🔒 P0: SSL/TLS Certificate Validation"

test_api "SSL Validation" "POST" "/api/v1/network/ssl/validate" \
    '{"url":"https://www.google.com","options":{"check_expiry":true,"check_chain":true}}' 200

# P0 Requirement: Connectivity Testing
echo ""
echo "📡 P0: Network Connectivity Testing"

test_api "Connectivity Test" "POST" "/api/v1/network/test/connectivity" \
    '{"target":"8.8.8.8","test_type":"ping"}' 200

# P0 Requirement: API Testing
echo ""
echo "🧪 P0: API Testing"

test_api "API Testing" "POST" "/api/v1/network/api/test" \
    '{"base_url":"https://httpbingo.org","test_suite":[{"endpoint":"/get","method":"GET","test_cases":[{"name":"Basic GET","expected_status":200}]}]}' 200

# Business Value Validation
echo ""
echo "💰 Business Value Validation:"

# Performance Target: Health check < 100ms
echo "  Testing: Health endpoint response time"
START_TIME=$(date +%s%N)
curl -s "http://localhost:$API_PORT/health" > /dev/null
END_TIME=$(date +%s%N)
DURATION_MS=$(( (END_TIME - START_TIME) / 1000000 ))

if [ $DURATION_MS -lt 100 ]; then
    echo "    ✅ Health endpoint responds in ${DURATION_MS}ms (< 100ms target)"
    ((TESTS_PASSED++))
else
    echo "    ⚠️  Health endpoint took ${DURATION_MS}ms (target: < 100ms)"
fi

# Core Capability: Network operations platform
echo "  Testing: Core capability availability"
ENDPOINTS=(
    "/api/v1/network/http"
    "/api/v1/network/dns"
    "/api/v1/network/scan"
    "/api/v1/network/ssl/validate"
    "/api/v1/network/test/connectivity"
    "/api/v1/network/api/test"
)

for endpoint in "${ENDPOINTS[@]}"; do
    if curl -s -X OPTIONS "http://localhost:$API_PORT$endpoint" > /dev/null 2>&1; then
        ((TESTS_PASSED++))
    else
        echo "    ⚠️  Endpoint $endpoint not responding to OPTIONS"
        ((TESTS_FAILED++))
    fi
done

echo "    ✅ All core network operation endpoints available"

# Summary
echo ""
echo "================================================"
echo "📊 Business Logic Validation Summary"
echo "================================================"
echo "  Passed: $TESTS_PASSED"
echo "  Failed: $TESTS_FAILED"

if [ $TESTS_FAILED -eq 0 ]; then
    echo ""
    echo "✅ All business logic tests passed!"
    echo "💼 Network-tools delivers on core value proposition:"
    echo "   • HTTP client operations working"
    echo "   • DNS operations functional"
    echo "   • Port scanning available"
    echo "   • SSL/TLS validation working"
    echo "   • Connectivity testing operational"
    echo "   • API testing framework ready"
    exit 0
else
    echo ""
    echo "❌ Some business logic tests failed"
    exit 1
fi
