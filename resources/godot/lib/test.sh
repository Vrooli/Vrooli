#!/usr/bin/env bash
# Godot Engine Resource - Test Library

set -euo pipefail

# Test Functions

godot::test::all() {
    echo "🧪 Running all Godot tests..."
    
    local failed=0
    
    # Run test phases
    if ! godot::test::smoke; then
        ((failed++))
    fi
    
    if ! godot::test::integration; then
        ((failed++))
    fi
    
    if ! godot::test::unit; then
        ((failed++))
    fi
    
    if [[ $failed -gt 0 ]]; then
        echo "❌ $failed test suite(s) failed"
        return 1
    fi
    
    echo "✅ All tests passed"
    return 0
}

godot::test::smoke() {
    echo "🔥 Running smoke tests..."
    
    # Delegate to test script if exists
    if [[ -f "${SCRIPT_DIR}/test/phases/test-smoke.sh" ]]; then
        bash "${SCRIPT_DIR}/test/phases/test-smoke.sh"
        return $?
    fi
    
    # Basic smoke test
    echo "  Testing health endpoint..."
    if ! timeout 5 curl -sf "http://localhost:${GODOT_API_PORT}/health" > /dev/null; then
        echo "  ❌ Health check failed"
        return 1
    fi
    echo "  ✅ Health check passed"
    
    echo "  Testing API availability..."
    if ! timeout 5 curl -sf "http://localhost:${GODOT_API_PORT}/api/projects" > /dev/null; then
        echo "  ❌ API not responding"
        return 1
    fi
    echo "  ✅ API responding"
    
    echo "✅ Smoke tests passed"
    return 0
}

godot::test::integration() {
    echo "🔗 Running integration tests..."
    
    # Delegate to test script if exists
    if [[ -f "${SCRIPT_DIR}/test/phases/test-integration.sh" ]]; then
        bash "${SCRIPT_DIR}/test/phases/test-integration.sh"
        return $?
    fi
    
    # Basic integration test
    echo "  Testing project creation..."
    local response=$(curl -sf -X POST "http://localhost:${GODOT_API_PORT}/api/projects" \
        -H "Content-Type: application/json" \
        -d '{"name": "test-project"}' 2>/dev/null || echo "FAILED")
    
    if [[ "$response" == "FAILED" ]]; then
        echo "  ❌ Project creation failed"
        return 1
    fi
    echo "  ✅ Project creation works"
    
    echo "  Testing project listing..."
    local projects=$(curl -sf "http://localhost:${GODOT_API_PORT}/api/projects" 2>/dev/null || echo "FAILED")
    
    if [[ "$projects" == "FAILED" ]]; then
        echo "  ❌ Project listing failed"
        return 1
    fi
    echo "  ✅ Project listing works"
    
    echo "✅ Integration tests passed"
    return 0
}

godot::test::unit() {
    echo "🔬 Running unit tests..."
    
    # Delegate to test script if exists
    if [[ -f "${SCRIPT_DIR}/test/phases/test-unit.sh" ]]; then
        bash "${SCRIPT_DIR}/test/phases/test-unit.sh"
        return $?
    fi
    
    # Test configuration loading
    echo "  Testing configuration..."
    if [[ -z "${GODOT_API_PORT:-}" ]]; then
        echo "  ❌ Configuration not loaded"
        return 1
    fi
    echo "  ✅ Configuration loaded"
    
    # Test directory creation
    echo "  Testing directory structure..."
    if [[ ! -d "${GODOT_BASE_DIR}" ]]; then
        echo "  ❌ Base directory missing"
        return 1
    fi
    echo "  ✅ Directory structure valid"
    
    echo "✅ Unit tests passed"
    return 0
}