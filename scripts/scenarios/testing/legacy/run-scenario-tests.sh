#!/usr/bin/env bash
# Universal scenario test runner - supports both legacy and new test formats
# This allows project-level tests to work with scenarios regardless of their test format

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# Run tests for a scenario, detecting format automatically
# Usage: run_scenario_tests <scenario_path> [timeout]
run_scenario_tests() {
    local scenario_path="$1"
    local timeout="${2:-60}"
    local scenario_name=$(basename "$scenario_path")
    
    # Check for new phased testing format first (preferred)
    if [[ -x "$scenario_path/test/run-tests.sh" ]]; then
        echo "🧪 Running phased tests for $scenario_name (new format)..."
        if timeout "$timeout" "$scenario_path/test/run-tests.sh" --quick; then
            return 0
        else
            return 1
        fi
    fi
    
    # Check for lifecycle test event in service.json (v2 format)
    if [[ -f "$scenario_path/.vrooli/service.json" ]]; then
        local has_test=$(jq -r '.lifecycle.test // null' "$scenario_path/.vrooli/service.json" 2>/dev/null)
        if [[ "$has_test" != "null" && -n "$has_test" ]]; then
            echo "🧪 Running lifecycle tests for $scenario_name (v2 format)..."
            if (cd "$scenario_path" && timeout "$timeout" "$APP_ROOT/scripts/manage.sh" test); then
                return 0
            else
                return 1
            fi
        fi
    fi
    
    # Check for legacy scenario-test.yaml format
    if [[ -f "$scenario_path/scenario-test.yaml" ]]; then
        echo "⚠️  Running legacy tests for $scenario_name (deprecated format)..."
        echo "   💡 Consider migrating to new phased testing architecture"
        
        # Run legacy integration test if tool exists
        local legacy_runner="$APP_ROOT/scripts/scenarios/tools/run-integration-test.sh"
        if [[ -x "$legacy_runner" ]]; then
            if timeout "$timeout" "$legacy_runner" "$scenario_name"; then
                return 0
            else
                return 1
            fi
        else
            echo "   ❌ Legacy test runner not found"
            return 1
        fi
    fi
    
    # No test format found
    echo "⚠️  No tests found for $scenario_name"
    echo "   💡 Add test/run-tests.sh or lifecycle.test in service.json"
    return 1
}

# Export for use by other scripts
export -f run_scenario_tests