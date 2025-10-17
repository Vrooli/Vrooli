#!/bin/bash

# Test script for scenario analysis functionality
# Tests the branding and pricing analysis capabilities

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="${SCRIPT_DIR}/.."
TEST_DATA_DIR="$SCRIPT_DIR/data"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Test tracking
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_TOTAL=0

# Logging functions
log_info() {
    echo -e "${YELLOW}[TEST]${NC} $1"
}

log_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((TESTS_PASSED++))
}

log_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((TESTS_FAILED++))
}

# Test helper functions
run_test() {
    local test_name="$1"
    local test_function="$2"
    
    log_info "Running test: $test_name"
    ((TESTS_TOTAL++))
    
    if $test_function; then
        log_pass "$test_name"
    else
        log_fail "$test_name"
    fi
}

# Create test scenario directory
setup_test_scenario() {
    local test_scenario_dir="$TEST_DATA_DIR/test-scenario"
    mkdir -p "$test_scenario_dir"/{api,ui,initialization}
    
    # Create test package.json
    cat > "$test_scenario_dir/ui/package.json" << 'EOF'
{
  "name": "test-scenario",
  "version": "1.0.0",
  "dependencies": {
    "react": "^18.0.0"
  }
}
EOF

    # Create test CSS with colors
    cat > "$test_scenario_dir/ui/styles.css" << 'EOF'
:root {
  --primary-color: #007bff;
  --secondary-color: #6c757d;
}

.primary {
  color: var(--primary-color);
}

body {
  font-family: "Inter", sans-serif;
}
EOF

    # Create test HTML with title
    cat > "$test_scenario_dir/ui/index.html" << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <title>Test Scenario App</title>
</head>
<body>
    <h1>Welcome to Test Scenario</h1>
</body>
</html>
EOF

    # Create test API file with pricing hints
    cat > "$test_scenario_dir/api/main.go" << 'EOF'
package main

import "fmt"

const (
    SUBSCRIPTION_PRICE = 29.99
    MONTHLY_BILLING = "monthly"
)

func main() {
    fmt.Println("Test scenario API")
}
EOF

    # Create service configuration
    cat > "$test_scenario_dir/.vrooli/service.json" << 'EOF'
{
  "service": {
    "name": "test-scenario",
    "displayName": "Test Scenario"
  }
}
EOF

    echo "$test_scenario_dir"
}

# Test: Analysis script exists and is executable
test_analysis_script_exists() {
    local analysis_script="$PROJECT_ROOT/scripts/analyze-scenario.sh"
    
    if [[ -f "$analysis_script" && -x "$analysis_script" ]]; then
        return 0
    else
        echo "Analysis script not found or not executable: $analysis_script"
        return 1
    fi
}

# Test: Local scenario analysis
test_local_scenario_analysis() {
    local test_scenario_dir
    test_scenario_dir=$(setup_test_scenario)
    local analysis_script="$PROJECT_ROOT/scripts/analyze-scenario.sh"
    
    # Run analysis
    local output
    if output=$("$analysis_script" --mode local --output json "$test_scenario_dir" 2>/dev/null); then
        # Check if output is valid JSON
        if echo "$output" | jq . >/dev/null 2>&1; then
            # Check for expected fields
            local brand_name
            brand_name=$(echo "$output" | jq -r '.branding.brand_name // empty')
            
            local primary_color
            primary_color=$(echo "$output" | jq -r '.branding.colors.primary // empty')
            
            local has_api
            has_api=$(echo "$output" | jq -r '.structure.has_api')
            
            local has_ui
            has_ui=$(echo "$output" | jq -r '.structure.has_ui')
            
            # Validate results
            if [[ "$brand_name" == "Test Scenario App" && 
                  "$primary_color" == "#007bff" && 
                  "$has_api" == "true" && 
                  "$has_ui" == "true" ]]; then
                return 0
            else
                echo "Analysis results don't match expected values"
                echo "Brand: '$brand_name' (expected 'Test Scenario App')"
                echo "Color: '$primary_color' (expected '#007bff')"
                echo "API: '$has_api' (expected 'true')"
                echo "UI: '$has_ui' (expected 'true')"
                return 1
            fi
        else
            echo "Analysis output is not valid JSON"
            echo "Output: $output"
            return 1
        fi
    else
        echo "Analysis script failed to run"
        return 1
    fi
}

# Test: Branding extraction
test_branding_extraction() {
    local test_scenario_dir
    test_scenario_dir=$(setup_test_scenario)
    local analysis_script="$PROJECT_ROOT/scripts/analyze-scenario.sh"
    
    # Run analysis and extract branding info
    local output
    output=$("$analysis_script" --mode local --output json "$test_scenario_dir" 2>/dev/null)
    
    # Check color extraction
    local primary_color
    primary_color=$(echo "$output" | jq -r '.branding.colors.primary')
    
    if [[ "$primary_color" == "#007bff" ]]; then
        return 0
    else
        echo "Primary color extraction failed: got '$primary_color', expected '#007bff'"
        return 1
    fi
}

# Test: Pricing model detection
test_pricing_detection() {
    local test_scenario_dir
    test_scenario_dir=$(setup_test_scenario)
    local analysis_script="$PROJECT_ROOT/scripts/analyze-scenario.sh"
    
    # Run analysis and check pricing detection
    local output
    output=$("$analysis_script" --mode local --output json "$test_scenario_dir" 2>/dev/null)
    
    local pricing_model
    pricing_model=$(echo "$output" | jq -r '.pricing.model')
    
    # Should detect subscription model from the Go code
    if [[ "$pricing_model" == "subscription" ]]; then
        return 0
    else
        echo "Pricing model detection failed: got '$pricing_model', expected 'subscription'"
        return 1
    fi
}

# Test: Structure analysis
test_structure_analysis() {
    local test_scenario_dir
    test_scenario_dir=$(setup_test_scenario)
    local analysis_script="$PROJECT_ROOT/scripts/analyze-scenario.sh"
    
    # Run analysis
    local output
    output=$("$analysis_script" --mode local --output json "$test_scenario_dir" 2>/dev/null)
    
    # Check structure detection
    local has_api
    has_api=$(echo "$output" | jq -r '.structure.has_api')
    local has_ui
    has_ui=$(echo "$output" | jq -r '.structure.has_ui')
    local api_framework
    api_framework=$(echo "$output" | jq -r '.structure.api_framework')
    local ui_framework
    ui_framework=$(echo "$output" | jq -r '.structure.ui_framework')
    
    if [[ "$has_api" == "true" && "$has_ui" == "true" && 
          "$api_framework" == "go" && "$ui_framework" == "javascript" ]]; then
        return 0
    else
        echo "Structure analysis failed:"
        echo "  API: $has_api (framework: $api_framework)"
        echo "  UI: $has_ui (framework: $ui_framework)"
        return 1
    fi
}

# Test: Summary output format
test_summary_output() {
    local test_scenario_dir
    test_scenario_dir=$(setup_test_scenario)
    local analysis_script="$PROJECT_ROOT/scripts/analyze-scenario.sh"
    
    # Run analysis with summary output
    local output
    if output=$("$analysis_script" --mode local --output summary "$test_scenario_dir" 2>/dev/null); then
        # Check if output contains expected summary sections
        if echo "$output" | grep -q "BRANDING:" && 
           echo "$output" | grep -q "PRICING:" && 
           echo "$output" | grep -q "STRUCTURE:"; then
            return 0
        else
            echo "Summary output missing expected sections"
            echo "Output: $output"
            return 1
        fi
    else
        echo "Summary output failed to generate"
        return 1
    fi
}

# Test: API endpoint (if running)
test_api_integration() {
    local api_port="${API_PORT:-8080}"
    local api_url="http://localhost:$api_port"
    
    # Check if API is running
    if ! curl -sf "$api_url/health" >/dev/null 2>&1; then
        echo "API not running, skipping API integration test"
        return 0  # Skip test if API not available
    fi
    
    local test_scenario_dir
    test_scenario_dir=$(setup_test_scenario)
    
    # Test API analysis endpoint
    local payload
    payload=$(jq -n \
        --arg mode "local" \
        --arg scenario_path "$test_scenario_dir" \
        '{ mode: $mode, scenario_path: $scenario_path }')
    
    local response
    if response=$(curl -sf -X POST \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "$api_url/api/v1/referral/analyze" 2>/dev/null); then
        
        # Validate response
        if echo "$response" | jq -e '.branding.brand_name' >/dev/null; then
            return 0
        else
            echo "API response missing expected fields"
            return 1
        fi
    else
        echo "API analysis request failed"
        return 1
    fi
}

# Cleanup function
cleanup() {
    if [[ -d "$TEST_DATA_DIR" ]]; then
        rm -rf "$TEST_DATA_DIR"
    fi
}

# Main test runner
main() {
    echo "Referral Program Generator - Analysis Tests"
    echo "==========================================="
    echo ""
    
    # Create test data directory
    mkdir -p "$TEST_DATA_DIR"
    
    # Run tests
    run_test "Analysis script exists" test_analysis_script_exists
    run_test "Local scenario analysis" test_local_scenario_analysis
    run_test "Branding extraction" test_branding_extraction
    run_test "Pricing detection" test_pricing_detection
    run_test "Structure analysis" test_structure_analysis
    run_test "Summary output format" test_summary_output
    run_test "API integration" test_api_integration
    
    # Cleanup
    cleanup
    
    # Report results
    echo ""
    echo "Test Results:"
    echo "============="
    echo "Total tests: $TESTS_TOTAL"
    echo "Passed: $TESTS_PASSED"
    echo "Failed: $TESTS_FAILED"
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        log_pass "All tests passed!"
        exit 0
    else
        log_fail "Some tests failed!"
        exit 1
    fi
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi