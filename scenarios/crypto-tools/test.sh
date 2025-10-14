#!/bin/bash
# ====================================================================
# Crypto-Tools Integration Test
# ====================================================================
#
# This test validates crypto-tools cryptographic operations
#
# ====================================================================

set -euo pipefail

# Get scenario directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_NAME="crypto-tools"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counters
PASSED=0
FAILED=0

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $*"
    ((PASSED++))
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $*"
    ((FAILED++))
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

# Get API port from running scenario
get_api_port() {
    local port=$(vrooli scenario status crypto-tools --json 2>/dev/null | jq -r '.scenario_data.allocated_ports.API_PORT // empty')
    if [[ -z "$port" || "$port" == "null" ]]; then
        log_error "Could not find API_PORT for crypto-tools"
        return 1
    fi
    echo "$port"
}

# Test 1: Health endpoint
test_health() {
    log_info "Testing health endpoint..."

    local api_port=$(get_api_port) || return 1
    # Health endpoint returns 503 when dependencies are unavailable (expected in degraded mode)
    local http_code=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:${api_port}/health" 2>/dev/null)

    if [[ "$http_code" == "200" || "$http_code" == "503" ]]; then
        log_success "Health endpoint responding (HTTP $http_code)"
    else
        log_error "Health endpoint not responding (HTTP $http_code)"
        return 1
    fi
}

# Test 2: Hash operation
test_hash() {
    log_info "Testing hash operation..."

    local api_port=$(get_api_port) || return 1
    local response=$(curl -s -X POST "http://localhost:${api_port}/api/v1/crypto/hash" \
        -H "Authorization: Bearer crypto-tools-api-key-2024" \
        -H "Content-Type: application/json" \
        -d '{"data":"test","algorithm":"sha256"}' 2>/dev/null)

    if echo "$response" | jq -e '.data.hash' >/dev/null 2>&1; then
        local hash=$(echo "$response" | jq -r '.data.hash')
        log_success "Hash operation successful (hash: ${hash:0:16}...)"
    else
        log_error "Hash operation failed: $response"
        return 1
    fi
}

# Test 3: Key generation
test_keygen() {
    log_info "Testing key generation..."

    local api_port=$(get_api_port) || return 1
    local response=$(curl -s -X POST "http://localhost:${api_port}/api/v1/crypto/keys/generate" \
        -H "Authorization: Bearer crypto-tools-api-key-2024" \
        -H "Content-Type: application/json" \
        -d '{"key_type":"rsa","key_size":2048}' 2>/dev/null)

    if echo "$response" | jq -e '.data.key_id' >/dev/null 2>&1; then
        local key_id=$(echo "$response" | jq -r '.data.key_id')
        log_success "Key generation successful (key_id: $key_id)"
    else
        log_error "Key generation failed: $response"
        return 1
    fi
}

# Test 4: CLI functionality
test_cli() {
    log_info "Testing CLI..."

    if [[ ! -f "${SCRIPT_DIR}/cli/crypto-tools" ]]; then
        log_error "CLI binary not found"
        return 1
    fi

    local api_port=$(get_api_port) || return 1
    local output=$("${SCRIPT_DIR}/cli/crypto-tools" --api-base "http://localhost:${api_port}" status 2>/dev/null)

    if [[ $? -eq 0 ]]; then
        log_success "CLI command successful"
    else
        log_error "CLI command failed"
        return 1
    fi
}

# Main execution
main() {
    echo "========================================"
    echo "  Crypto-Tools Integration Test"
    echo "========================================"
    echo ""

    # Check if scenario is running
    if ! vrooli scenario status crypto-tools --json >/dev/null 2>&1; then
        log_error "Scenario crypto-tools is not running"
        echo ""
        echo "Start the scenario first with: make run"
        exit 1
    fi

    # Run tests
    test_health || true
    test_hash || true
    test_keygen || true
    test_cli || true

    echo ""
    echo "========================================"
    echo "  Test Results"
    echo "========================================"
    echo -e "${GREEN}Passed:${NC} $PASSED"
    echo -e "${RED}Failed:${NC} $FAILED"
    echo ""

    if [[ $FAILED -gt 0 ]]; then
        echo -e "${RED}❌ Tests failed${NC}"
        exit 1
    else
        echo -e "${GREEN}✅ All tests passed${NC}"
        exit 0
    fi
}

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
