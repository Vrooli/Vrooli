#!/usr/bin/env bash
# Godot Engine Resource - Unit Tests

set -euo pipefail

# Get directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly RESOURCE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Source configuration and libraries
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"

# Test functions
test_configuration_loading() {
    echo "  Testing configuration loading..."
    
    # Check required environment variables
    if [[ -z "${GODOT_API_PORT:-}" ]]; then
        echo "    ❌ GODOT_API_PORT not set"
        return 1
    fi
    
    if [[ -z "${GODOT_LSP_PORT:-}" ]]; then
        echo "    ❌ GODOT_LSP_PORT not set"
        return 1
    fi
    
    if [[ -z "${GODOT_VERSION:-}" ]]; then
        echo "    ❌ GODOT_VERSION not set"
        return 1
    fi
    
    echo "    ✅ Configuration loaded correctly"
    return 0
}

test_runtime_json_validity() {
    echo "  Testing runtime.json validity..."
    
    local runtime_file="${RESOURCE_DIR}/config/runtime.json"
    
    if [[ ! -f "$runtime_file" ]]; then
        echo "    ❌ runtime.json missing"
        return 1
    fi
    
    # Check JSON validity
    if ! jq empty "$runtime_file" 2>/dev/null; then
        echo "    ❌ runtime.json is not valid JSON"
        return 1
    fi
    
    # Check required fields
    local startup_order=$(jq -r '.startup_order' "$runtime_file")
    if [[ "$startup_order" == "null" ]]; then
        echo "    ❌ startup_order field missing"
        return 1
    fi
    
    echo "    ✅ runtime.json is valid"
    return 0
}

test_schema_json_validity() {
    echo "  Testing schema.json validity..."
    
    local schema_file="${RESOURCE_DIR}/config/schema.json"
    
    if [[ ! -f "$schema_file" ]]; then
        echo "    ❌ schema.json missing"
        return 1
    fi
    
    # Check JSON validity
    if ! jq empty "$schema_file" 2>/dev/null; then
        echo "    ❌ schema.json is not valid JSON"
        return 1
    fi
    
    echo "    ✅ schema.json is valid"
    return 0
}

test_port_allocation() {
    echo "  Testing port allocation..."
    
    # Check port values are valid
    if [[ "$GODOT_API_PORT" -lt 1024 ]] || [[ "$GODOT_API_PORT" -gt 65535 ]]; then
        echo "    ❌ Invalid API port: $GODOT_API_PORT"
        return 1
    fi
    
    if [[ "$GODOT_LSP_PORT" -lt 1024 ]] || [[ "$GODOT_LSP_PORT" -gt 65535 ]]; then
        echo "    ❌ Invalid LSP port: $GODOT_LSP_PORT"
        return 1
    fi
    
    echo "    ✅ Port allocation valid"
    return 0
}

test_library_functions() {
    echo "  Testing library functions..."
    
    # Test health_check function exists
    if ! declare -f godot::health_check > /dev/null; then
        echo "    ❌ godot::health_check function not defined"
        return 1
    fi
    
    # Test is_running function exists
    if ! declare -f godot::is_running > /dev/null; then
        echo "    ❌ godot::is_running function not defined"
        return 1
    fi
    
    echo "    ✅ Library functions defined"
    return 0
}

# Main execution
main() {
    echo "🔬 Running Godot unit tests..."
    
    local failed=0
    
    test_configuration_loading || ((failed++))
    test_runtime_json_validity || ((failed++))
    test_schema_json_validity || ((failed++))
    test_port_allocation || ((failed++))
    test_library_functions || ((failed++))
    
    if [[ $failed -gt 0 ]]; then
        echo "❌ Unit tests failed ($failed failures)"
        return 1
    fi
    
    echo "✅ All unit tests passed"
    return 0
}

main "$@"