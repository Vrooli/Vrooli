#!/usr/bin/env bash
# Godot Engine Resource - Integration Tests

set -euo pipefail

# Get directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly RESOURCE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Source configuration
source "${RESOURCE_DIR}/config/defaults.sh"

# Test functions
test_project_creation() {
    echo "  Testing project creation..."
    
    local project_name="test-project-$(date +%s)"
    local response=$(curl -sf -X POST "http://localhost:${GODOT_API_PORT}/api/projects" \
        -H "Content-Type: application/json" \
        -d "{\"name\": \"${project_name}\"}" 2>/dev/null || echo "FAILED")
    
    if [[ "$response" == "FAILED" ]]; then
        echo "    ❌ Failed to create project"
        return 1
    fi
    
    if ! echo "$response" | grep -q "$project_name"; then
        echo "    ❌ Project not created correctly"
        return 1
    fi
    
    echo "    ✅ Project creation successful"
    return 0
}

test_project_listing() {
    echo "  Testing project listing..."
    
    local response=$(curl -sf "http://localhost:${GODOT_API_PORT}/api/projects" 2>/dev/null || echo "FAILED")
    
    if [[ "$response" == "FAILED" ]]; then
        echo "    ❌ Failed to list projects"
        return 1
    fi
    
    if ! echo "$response" | grep -q '"projects"'; then
        echo "    ❌ Invalid projects response format"
        return 1
    fi
    
    echo "    ✅ Project listing successful"
    return 0
}

test_health_details() {
    echo "  Testing health endpoint details..."
    
    local response=$(curl -sf "http://localhost:${GODOT_API_PORT}/health" 2>/dev/null || echo "FAILED")
    
    if [[ "$response" == "FAILED" ]]; then
        echo "    ❌ Health endpoint failed"
        return 1
    fi
    
    # Check for required fields
    if ! echo "$response" | grep -q '"version"'; then
        echo "    ❌ Version field missing in health response"
        return 1
    fi
    
    if ! echo "$response" | grep -q '"lsp_port"'; then
        echo "    ❌ LSP port field missing in health response"
        return 1
    fi
    
    echo "    ✅ Health endpoint provides complete information"
    return 0
}

test_invalid_endpoints() {
    echo "  Testing invalid endpoint handling..."
    
    local status_code=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:${GODOT_API_PORT}/invalid")
    
    if [[ "$status_code" != "404" ]]; then
        echo "    ❌ Invalid endpoint returned $status_code instead of 404"
        return 1
    fi
    
    echo "    ✅ Invalid endpoints handled correctly"
    return 0
}

# Main execution
main() {
    echo "🔗 Running Godot integration tests..."
    
    local failed=0
    
    test_project_creation || ((failed++))
    test_project_listing || ((failed++))
    test_health_details || ((failed++))
    test_invalid_endpoints || ((failed++))
    
    if [[ $failed -gt 0 ]]; then
        echo "❌ Integration tests failed ($failed failures)"
        return 1
    fi
    
    echo "✅ All integration tests passed"
    return 0
}

main "$@"