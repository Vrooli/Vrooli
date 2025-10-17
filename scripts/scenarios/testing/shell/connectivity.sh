#!/usr/bin/env bash
# Connectivity testing utilities - API and UI endpoint testing
set -euo pipefail

# Source core utilities
SHELL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SHELL_DIR/core.sh"

# === URL Building ===

# Get API URL for the scenario
testing::connectivity::get_api_url() {
    local scenario_name="${1:-$(testing::core::detect_scenario)}"
    
    # Get API port using vrooli command
    local api_port
    api_port=$(vrooli scenario port "$scenario_name" API_PORT 2>/dev/null || echo "")
    
    if [ -n "$api_port" ]; then
        echo "http://localhost:$api_port"
        return 0
    else
        return 1
    fi
}

# Get UI URL for the scenario
testing::connectivity::get_ui_url() {
    local scenario_name="${1:-$(testing::core::detect_scenario)}"
    
    # Get UI port using vrooli command
    local ui_port
    ui_port=$(vrooli scenario port "$scenario_name" UI_PORT 2>/dev/null || echo "")
    
    if [ -n "$ui_port" ]; then
        echo "http://localhost:$ui_port"
        return 0
    else
        return 1
    fi
}

# === Connectivity Testing ===

# Test API connectivity
testing::connectivity::test_api() {
    local scenario_name="${1:-$(testing::core::detect_scenario)}"
    local endpoint="${2:-/health}"
    local timeout="${3:-10}"
    
    echo "🔌 Testing API connectivity for $scenario_name..."
    
    # Get API URL
    local api_url
    api_url=$(testing::connectivity::get_api_url "$scenario_name")
    
    if [ -z "$api_url" ]; then
        echo "❌ Could not discover API URL for $scenario_name"
        return 1
    fi
    
    # Test connectivity with curl
    if curl -s --max-time "$timeout" "${api_url}${endpoint}" >/dev/null 2>&1; then
        echo "✅ API is responding at $api_url"
        return 0
    else
        echo "❌ API is not responding at $api_url"
        return 1
    fi
}

# Test UI connectivity
testing::connectivity::test_ui() {
    local scenario_name="${1:-$(testing::core::detect_scenario)}"
    local timeout="${2:-10}"
    
    echo "🔌 Testing UI connectivity for $scenario_name..."
    
    # Get UI URL
    local ui_url
    ui_url=$(testing::connectivity::get_ui_url "$scenario_name")
    
    if [ -z "$ui_url" ]; then
        echo "❌ Could not discover UI URL for $scenario_name"
        return 1
    fi
    
    # Test connectivity with curl
    if curl -s --max-time "$timeout" "$ui_url" >/dev/null 2>&1; then
        echo "✅ UI is responding at $ui_url"
        return 0
    else
        echo "❌ UI is not responding at $ui_url"
        return 1
    fi
}

# Test both API and UI connectivity
testing::connectivity::test_all() {
    local scenario_name="${1:-$(testing::core::detect_scenario)}"
    local api_success=false
    local ui_success=false
    
    echo "🔌 Testing all connectivity for $scenario_name..."
    
    if testing::connectivity::test_api "$scenario_name"; then
        api_success=true
    fi
    
    if testing::connectivity::test_ui "$scenario_name"; then
        ui_success=true
    fi
    
    if $api_success && $ui_success; then
        echo "✅ All connectivity tests passed"
        return 0
    elif $api_success || $ui_success; then
        echo "⚠️  Partial connectivity (API: $api_success, UI: $ui_success)"
        return 0  # Partial success is still success
    else
        echo "❌ All connectivity tests failed"
        return 1
    fi
}

# Export functions
if [[ "${BASH_SOURCE[0]}" != "${0}" ]]; then
    export -f testing::connectivity::get_api_url
    export -f testing::connectivity::get_ui_url
    export -f testing::connectivity::test_api
    export -f testing::connectivity::test_ui
    export -f testing::connectivity::test_all
fi