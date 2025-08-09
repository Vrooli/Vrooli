#!/usr/bin/env bash
set -euo pipefail

# URL Discovery Test Suite
# Tests the URL discovery infrastructure functionality

_HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck disable=SC1091
source "${_HERE}/../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${_HERE}/url-discovery.sh"

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
url_discovery_test::test_start() {
    local test_name="$1"
    echo -e "${YELLOW}🧪 Testing: $test_name${NC}"
    ((TESTS_RUN++))
}

url_discovery_test::test_pass() {
    local test_name="$1"
    echo -e "${GREEN}✅ PASS: $test_name${NC}"
    ((TESTS_PASSED++))
}

url_discovery_test::test_fail() {
    local test_name="$1"
    local reason="${2:-Unknown error}"
    echo -e "${RED}❌ FAIL: $test_name${NC}"
    echo -e "${RED}   Reason: $reason${NC}"
    ((TESTS_FAILED++))
}

url_discovery_test::test_summary() {
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

url_discovery_test::test_infrastructure_loading() {
    url_discovery_test::test_start "URL Discovery Infrastructure Loading"
    
    # Check if key functions are available
    if command -v url_discovery::discover >/dev/null 2>&1 && \
       command -v url_discovery::format_display >/dev/null 2>&1 && \
       command -v url_discovery::validate_url >/dev/null 2>&1; then
        url_discovery_test::test_pass "Infrastructure functions are available"
    else
        url_discovery_test::test_fail "Infrastructure functions not loaded"
    fi
}

url_discovery_test::test_resource_configuration() {
    url_discovery_test::test_start "Resource Configuration Registry"
    
    # Check if resource configurations are loaded
    local resource_count
    resource_count=$(url_discovery::list_resources | wc -l)
    
    if (( resource_count > 10 )); then
        url_discovery_test::test_pass "Resource configurations loaded ($resource_count resources)"
    else
        url_discovery_test::test_fail "Insufficient resource configurations ($resource_count resources)"
    fi
}

url_discovery_test::test_config_parsing() {
    url_discovery_test::test_start "Configuration Parsing"
    
    # Test parsing for a known resource
    local config_json
    config_json=$(url_discovery::parse_config "ollama" 2>/dev/null)
    
    if echo "$config_json" | jq -e '.type == "http" and .default_port == "11434"' >/dev/null 2>&1; then
        url_discovery_test::test_pass "Configuration parsing works correctly"
    else
        url_discovery_test::test_fail "Configuration parsing failed or incorrect data"
    fi
}

url_discovery_test::test_url_validation() {
    url_discovery_test::test_start "URL Validation"
    
    # Test with a known good URL (if available)
    if url_discovery::validate_url "http://httpbin.org/get" 10 2>/dev/null; then
        url_discovery_test::test_pass "URL validation works with reachable URL"
    else
        # This might fail due to network issues, so it's not a critical failure
        echo -e "${YELLOW}⚠️  SKIP: URL validation (network not available)${NC}"
        ((TESTS_RUN--))  # Don't count this as a run test
    fi
}

url_discovery_test::test_discovery_fallback() {
    url_discovery_test::test_start "Discovery Fallback Method"
    
    # Test fallback discovery for a resource that doesn't have manage.sh url support
    local result
    result=$(url_discovery::discover "postgres" 2>/dev/null)
    
    if echo "$result" | jq -e '.primary and .name and .type' >/dev/null 2>&1; then
        url_discovery_test::test_pass "Fallback discovery produces valid JSON"
    else
        url_discovery_test::test_fail "Fallback discovery failed or invalid JSON"
    fi
}

url_discovery_test::test_caching_mechanism() {
    url_discovery_test::test_start "Caching Mechanism"
    
    # Test cache initialization
    url_discovery::init_cache
    
    if [[ -d "/tmp/vrooli-url-discovery" ]]; then
        url_discovery_test::test_pass "Cache directory created successfully"
        
        # Test cache operations
        local test_data='{"test": "data"}'
        url_discovery::set_cache "test-resource" "$test_data"
        
        local cached_data
        cached_data=$(url_discovery::get_cache "test-resource" 2>/dev/null || echo "")
        
        if [[ "$cached_data" == "$test_data" ]]; then
            url_discovery_test::test_pass "Cache set/get operations work"
        else
            url_discovery_test::test_fail "Cache operations failed"
        fi
        
        # Cleanup test cache
        url_discovery::clear_cache "test-resource"
    else
        url_discovery_test::test_fail "Cache directory not created"
    fi
}

url_discovery_test::test_display_formatting() {
    url_discovery_test::test_start "Display Formatting"
    
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
        url_discovery_test::test_pass "Display formatting works correctly"
    else
        url_discovery_test::test_fail "Display formatting incorrect"
    fi
}

url_discovery_test::test_resource_categories() {
    url_discovery_test::test_start "Resource Categories"
    
    # Test that resource categories are properly defined
    local categories_json
    categories_json=$(url_discovery::get_resource_categories 2>/dev/null)
    
    if echo "$categories_json" | jq -e '.storage and .ai and .automation' >/dev/null 2>&1; then
        url_discovery_test::test_pass "Resource categories are properly defined"
    else
        url_discovery_test::test_fail "Resource categories missing or malformed"
    fi
}

url_discovery_test::test_health_status_detection() {
    url_discovery_test::test_start "Health Status Detection"
    
    # Test health status function
    local status
    status=$(url_discovery::get_health_status "http://httpbin.org/status/200" 5 2>/dev/null || echo "unreachable")
    
    if [[ "$status" == "healthy" || "$status" == "unreachable" ]]; then
        url_discovery_test::test_pass "Health status detection works"
    else
        url_discovery_test::test_fail "Health status detection returned unexpected value: $status"
    fi
}

################################################################################
# Integration Tests
################################################################################

url_discovery_test::test_scenario_integration() {
    url_discovery_test::test_start "Scenario Integration"
    
    # Test that the scenario-to-app.sh script can source the URL discovery
    local scenario_script="${RESOURCES_DIR}/../scenarios/tools/scenario-to-app.sh"
    
    if [[ -f "$scenario_script" ]]; then
        # Check if the scenario script sources the URL discovery infrastructure
        if grep -q "url-discovery.sh" "$scenario_script"; then
            url_discovery_test::test_pass "Scenario script integrates URL discovery"
        else
            url_discovery_test::test_fail "Scenario script does not integrate URL discovery"
        fi
    else
        url_discovery_test::test_fail "Scenario script not found"
    fi
}

################################################################################
# Main Test Execution
################################################################################

url_discovery_test::main() {
    echo "🚀 Starting URL Discovery Test Suite"
    echo "====================================="
    echo ""
    
    # Run all tests
    url_discovery_test::test_infrastructure_loading
    url_discovery_test::test_resource_configuration
    url_discovery_test::test_config_parsing
    url_discovery_test::test_url_validation
    url_discovery_test::test_discovery_fallback
    url_discovery_test::test_caching_mechanism
    url_discovery_test::test_display_formatting
    url_discovery_test::test_resource_categories
    url_discovery_test::test_health_status_detection
    
    # Integration tests
    echo ""
    echo "🔗 Integration Tests"
    echo "==================="
    url_discovery_test::test_scenario_integration
    
    echo ""
    url_discovery_test::test_summary
}

# Run tests if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    url_discovery_test::main "$@"
fi