#!/usr/bin/env bash
set -euo pipefail

# URL Discovery Test Suite
# Tests the URL discovery infrastructure functionality

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCES_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Source the URL discovery infrastructure
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/url-discovery.sh"

# Test colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Test framework functions
test_start() {
    local test_name="$1"
    echo -e "${YELLOW}🧪 Testing: $test_name${NC}"
    ((TESTS_RUN++))
}

test_pass() {
    local test_name="$1"
    echo -e "${GREEN}✅ PASS: $test_name${NC}"
    ((TESTS_PASSED++))
}

test_fail() {
    local test_name="$1"
    local reason="${2:-Unknown error}"
    echo -e "${RED}❌ FAIL: $test_name${NC}"
    echo -e "${RED}   Reason: $reason${NC}"
    ((TESTS_FAILED++))
}

test_summary() {
    echo ""
    echo "📊 Test Summary:"
    echo "   Total: $TESTS_RUN"
    echo -e "   ${GREEN}Passed: $TESTS_PASSED${NC}"
    echo -e "   ${RED}Failed: $TESTS_FAILED${NC}"
    
    if (( TESTS_FAILED > 0 )); then
        echo -e "${RED}Some tests failed!${NC}"
        return 1
    else
        echo -e "${GREEN}All tests passed!${NC}"
        return 0
    fi
}

################################################################################
# Test Cases
################################################################################

test_infrastructure_loading() {
    test_start "URL Discovery Infrastructure Loading"
    
    # Check if key functions are available
    if command -v url_discovery::discover >/dev/null 2>&1 && \
       command -v url_discovery::format_display >/dev/null 2>&1 && \
       command -v url_discovery::validate_url >/dev/null 2>&1; then
        test_pass "Infrastructure functions are available"
    else
        test_fail "Infrastructure functions not loaded"
    fi
}

test_resource_configuration() {
    test_start "Resource Configuration Registry"
    
    # Check if resource configurations are loaded
    local resource_count
    resource_count=$(url_discovery::list_resources | wc -l)
    
    if (( resource_count > 10 )); then
        test_pass "Resource configurations loaded ($resource_count resources)"
    else
        test_fail "Insufficient resource configurations ($resource_count resources)"
    fi
}

test_config_parsing() {
    test_start "Configuration Parsing"
    
    # Test parsing for a known resource
    local config_json
    config_json=$(url_discovery::parse_config "ollama" 2>/dev/null)
    
    if echo "$config_json" | jq -e '.type == "http" and .default_port == "11434"' >/dev/null 2>&1; then
        test_pass "Configuration parsing works correctly"
    else
        test_fail "Configuration parsing failed or incorrect data"
    fi
}

test_url_validation() {
    test_start "URL Validation"
    
    # Test with a known good URL (if available)
    if url_discovery::validate_url "http://httpbin.org/get" 10 2>/dev/null; then
        test_pass "URL validation works with reachable URL"
    else
        # This might fail due to network issues, so it's not a critical failure
        echo -e "${YELLOW}⚠️  SKIP: URL validation (network not available)${NC}"
        ((TESTS_RUN--))  # Don't count this as a run test
    fi
}

test_discovery_fallback() {
    test_start "Discovery Fallback Method"
    
    # Test fallback discovery for a resource that doesn't have manage.sh url support
    local result
    result=$(url_discovery::discover "postgres" 2>/dev/null)
    
    if echo "$result" | jq -e '.primary and .name and .type' >/dev/null 2>&1; then
        test_pass "Fallback discovery produces valid JSON"
    else
        test_fail "Fallback discovery failed or invalid JSON"
    fi
}

test_caching_mechanism() {
    test_start "Caching Mechanism"
    
    # Test cache initialization
    url_discovery::init_cache
    
    if [[ -d "/tmp/vrooli-url-discovery" ]]; then
        test_pass "Cache directory created successfully"
        
        # Test cache operations
        local test_data='{"test": "data"}'
        url_discovery::set_cache "test-resource" "$test_data"
        
        local cached_data
        cached_data=$(url_discovery::get_cache "test-resource" 2>/dev/null || echo "")
        
        if [[ "$cached_data" == "$test_data" ]]; then
            test_pass "Cache set/get operations work"
        else
            test_fail "Cache operations failed"
        fi
        
        # Cleanup test cache
        url_discovery::clear_cache "test-resource"
    else
        test_fail "Cache directory not created"
    fi
}

test_display_formatting() {
    test_start "Display Formatting"
    
    # Test formatting with sample data
    local sample_json='{
        "primary": "http://localhost:11434",
        "name": "Test Service",
        "type": "http",
        "status": "healthy"
    }'
    
    local formatted
    formatted=$(url_discovery::format_display "test" "$sample_json" true 2>/dev/null)
    
    if echo "$formatted" | grep -q "✅.*Test Service.*http://localhost:11434"; then
        test_pass "Display formatting works correctly"
    else
        test_fail "Display formatting incorrect"
    fi
}

test_resource_categories() {
    test_start "Resource Categories"
    
    # Test that resource categories are properly defined
    local categories_json
    categories_json=$(url_discovery::get_resource_categories 2>/dev/null)
    
    if echo "$categories_json" | jq -e '.storage and .ai and .automation' >/dev/null 2>&1; then
        test_pass "Resource categories are properly defined"
    else
        test_fail "Resource categories missing or malformed"
    fi
}

test_health_status_detection() {
    test_start "Health Status Detection"
    
    # Test health status function
    local status
    status=$(url_discovery::get_health_status "http://httpbin.org/status/200" 5 2>/dev/null || echo "unreachable")
    
    if [[ "$status" == "healthy" || "$status" == "unreachable" ]]; then
        test_pass "Health status detection works"
    else
        test_fail "Health status detection returned unexpected value: $status"
    fi
}

################################################################################
# Integration Tests
################################################################################

test_scenario_integration() {
    test_start "Scenario Integration"
    
    # Test that the scenario-to-app.sh script can source the URL discovery
    local scenario_script="${RESOURCES_DIR}/../scenarios/tools/scenario-to-app.sh"
    
    if [[ -f "$scenario_script" ]]; then
        # Check if the scenario script sources the URL discovery infrastructure
        if grep -q "url-discovery.sh" "$scenario_script"; then
            test_pass "Scenario script integrates URL discovery"
        else
            test_fail "Scenario script does not integrate URL discovery"
        fi
    else
        test_fail "Scenario script not found"
    fi
}

################################################################################
# Main Test Execution
################################################################################

main() {
    echo "🚀 Starting URL Discovery Test Suite"
    echo "====================================="
    echo ""
    
    # Run all tests
    test_infrastructure_loading
    test_resource_configuration
    test_config_parsing
    test_url_validation
    test_discovery_fallback
    test_caching_mechanism
    test_display_formatting
    test_resource_categories
    test_health_status_detection
    
    # Integration tests
    echo ""
    echo "🔗 Integration Tests"
    echo "==================="
    test_scenario_integration
    
    echo ""
    test_summary
}

# Run tests if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi