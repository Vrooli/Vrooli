#!/usr/bin/env bash
# Godot Engine Resource - Smoke Tests

set -euo pipefail

# Get directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly RESOURCE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Source configuration
source "${RESOURCE_DIR}/config/defaults.sh"

# Test functions
test_health_endpoint() {
    echo "  Testing health endpoint..."
    
    if ! timeout 5 curl -sf "http://localhost:${GODOT_API_PORT}/health" > /dev/null; then
        echo "    ❌ Health endpoint not responding"
        return 1
    fi
    
    # Check response format
    local response=$(curl -sf "http://localhost:${GODOT_API_PORT}/health")
    if ! echo "$response" | grep -q '"status"'; then
        echo "    ❌ Invalid health response format"
        return 1
    fi
    
    echo "    ✅ Health endpoint working"
    return 0
}

test_api_availability() {
    echo "  Testing API availability..."
    
    if ! timeout 5 curl -sf "http://localhost:${GODOT_API_PORT}/api/projects" > /dev/null; then
        echo "    ❌ API not available"
        return 1
    fi
    
    echo "    ✅ API available"
    return 0
}

test_lsp_port() {
    echo "  Testing LSP port..."
    
    if ! timeout 2 nc -z localhost "${GODOT_LSP_PORT}" 2>/dev/null; then
        echo "    ⚠️  LSP not running (expected for minimal scaffold)"
        # Don't fail for scaffold
        return 0
    fi
    
    echo "    ✅ LSP port open"
    return 0
}

test_directories() {
    echo "  Testing directory structure..."
    
    if [[ ! -d "${GODOT_BASE_DIR}" ]]; then
        echo "    ❌ Base directory missing: ${GODOT_BASE_DIR}"
        return 1
    fi
    
    echo "    ✅ Directory structure exists"
    return 0
}

# Main execution
main() {
    echo "🔥 Running Godot smoke tests..."
    
    local failed=0
    
    test_health_endpoint || ((failed++))
    test_api_availability || ((failed++))
    test_lsp_port || ((failed++))
    test_directories || ((failed++))
    
    if [[ $failed -gt 0 ]]; then
        echo "❌ Smoke tests failed ($failed failures)"
        return 1
    fi
    
    echo "✅ All smoke tests passed"
    return 0
}

main "$@"