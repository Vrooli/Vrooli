#!/usr/bin/env bash
# Core testing utilities - scenario detection, configuration, language detection
set -euo pipefail

# === Scenario Detection and Setup ===

# Detect the current scenario being tested
testing::core::detect_scenario() {
    local current_dir
    current_dir=$(pwd)
    
    # Check if we're in a scenario directory
    if [[ "$current_dir" =~ /scenarios/([^/]+) ]]; then
        echo "${BASH_REMATCH[1]}"
        return 0
    else
        # Try to detect from test file paths
        if [[ "$current_dir" =~ scenarios/([^/]+)/ ]]; then
            echo "${BASH_REMATCH[1]}"
            return 0
        fi
        
        echo "ERROR: Could not detect scenario name from current directory: $current_dir" >&2
        return 1
    fi
}

# Get scenario configuration from service.json
testing::core::get_scenario_config() {
    local scenario_name="$1"
    local config_key="${2:-}"
    
    local service_json=".vrooli/service.json"
    if [ ! -f "$service_json" ]; then
        echo "ERROR: No service.json found at $service_json" >&2
        return 1
    fi
    
    if [ -n "$config_key" ]; then
        jq -r ".$config_key // empty" "$service_json" 2>/dev/null || echo ""
    else
        cat "$service_json"
    fi
}

# === Language Detection ===

# Detect which languages/technologies are used in the scenario
testing::core::detect_languages() {
    local languages=()
    
    # Go detection
    if [ -f "go.mod" ] || find . -name "*.go" -not -path "./node_modules/*" | head -1 | grep -q ".*"; then
        languages+=("go")
    fi
    
    # Node.js detection
    if [ -f "package.json" ]; then
        languages+=("node")
    fi
    
    # Python detection
    if [ -f "requirements.txt" ] || [ -f "pyproject.toml" ] || [ -f "setup.py" ] || find . -name "*.py" -not -path "./node_modules/*" | head -1 | grep -q ".*"; then
        languages+=("python")
    fi
    
    # Rust detection
    if [ -f "Cargo.toml" ]; then
        languages+=("rust")
    fi
    
    printf '%s\n' "${languages[@]}"
}

# === Runtime Availability Helpers ===

# Check if runtime-dependent phases may be skipped when scenario is not running
testing::core::allow_skip_missing_runtime() {
    case "${TEST_ALLOW_SKIP_MISSING_RUNTIME:-}" in
        1|true|TRUE|yes|YES|on|ON)
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

# Ensure the scenario is running before executing a phase; optionally skip
testing::core::ensure_runtime_or_skip() {
    local scenario_name="${1:-$(testing::core::detect_scenario)}"
    local context="${2:-tests}"

    if testing::core::is_scenario_running "$scenario_name"; then
        return 0
    fi

    if testing::core::allow_skip_missing_runtime; then
        local message="Scenario '$scenario_name' is not running; skipping ${context}."
        if command -v log::warning >/dev/null 2>&1; then
            log::warning "⚠️  $message"
        else
            echo "⚠️  $message"
        fi
        if command -v log::info >/dev/null 2>&1; then
            log::info "   Run 'vrooli scenario run $scenario_name' to execute these checks."
        else
            echo "   Run 'vrooli scenario run $scenario_name' to execute these checks." >&2
        fi
        return 200
    fi

    if command -v log::error >/dev/null 2>&1; then
        log::error "❌ Scenario '$scenario_name' is not running"
        log::info "   Start it with: vrooli scenario run $scenario_name"
    else
        echo "❌ Scenario '$scenario_name' is not running" >&2
        echo "   Start it with: vrooli scenario run $scenario_name" >&2
    fi
    return 1
}

# === Utility Functions ===

# Check if scenario is running
testing::core::is_scenario_running() {
    local scenario_name="${1:-$(testing::core::detect_scenario)}"
    
    # Try to get API port
    local api_port
    api_port=$(vrooli scenario port "$scenario_name" API_PORT 2>/dev/null || echo "")
    
    if [ -n "$api_port" ]; then
        # Try to connect to the API health endpoint
        if curl -s --max-time 2 "http://localhost:$api_port/health" >/dev/null 2>&1; then
            return 0
        fi
    fi
    
    # Try UI port as fallback
    local ui_port
    ui_port=$(vrooli scenario port "$scenario_name" UI_PORT 2>/dev/null || echo "")
    
    if [ -n "$ui_port" ]; then
        if curl -s --max-time 2 "http://localhost:$ui_port" >/dev/null 2>&1; then
            return 0
        fi
    fi
    
    return 1
}

# Wait for scenario to be ready
testing::core::wait_for_scenario() {
    local scenario_name="${1:-$(testing::core::detect_scenario)}"
    local max_wait="${2:-30}"
    
    echo "⏳ Waiting for $scenario_name to be ready (max ${max_wait}s)..."
    
    local elapsed=0
    local sleep_interval=2
    
    while [[ $elapsed -lt $max_wait ]]; do
        if testing::core::is_scenario_running "$scenario_name"; then
            echo "$scenario_name is ready after ${elapsed}s"
            return 0
        fi
        
        sleep $sleep_interval
        elapsed=$((elapsed + sleep_interval))
        echo -n "."
    done
    
    echo ""
    echo "ERROR: $scenario_name did not become ready within ${max_wait}s"
    return 1
}

# Export functions for use by test scripts
if [[ "${BASH_SOURCE[0]}" != "${0}" ]]; then
    export -f testing::core::detect_scenario
    export -f testing::core::get_scenario_config
    export -f testing::core::detect_languages
    export -f testing::core::allow_skip_missing_runtime
    export -f testing::core::ensure_runtime_or_skip
    export -f testing::core::is_scenario_running
    export -f testing::core::wait_for_scenario
fi
