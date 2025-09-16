#!/bin/bash

# API Manager - Automated Fix Integration Tests
# Tests the automated fix functionality with safety controls

set -euo pipefail

# Test configuration
API_BASE_URL="${API_MANAGER_URL:-http://localhost:${API_PORT}}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🧪 Running API Manager Automated Fix Tests${NC}"
echo "API Base URL: $API_BASE_URL"
echo

# Test functions
test_fix_config_disabled_by_default() {
    echo -e "${YELLOW}Testing that automated fixes are disabled by default...${NC}"
    
    response=$(curl -s -w "%{http_code}" "$API_BASE_URL/api/v1/fix/config" -o /tmp/config_response.json)
    http_code="${response: -3}"
    
    if [[ "$http_code" == "200" ]]; then
        enabled=$(jq -r '.enabled' /tmp/config_response.json)
        safety_status=$(jq -r '.safety_status' /tmp/config_response.json)
        
        if [[ "$enabled" == "false" ]]; then
            echo -e "${GREEN}✓ Automated fixes disabled by default${NC}"
            echo -e "${BLUE}  Safety status: $safety_status${NC}"
        else
            echo -e "${RED}✗ CRITICAL: Automated fixes enabled by default (SAFETY VIOLATION)${NC}"
            return 1
        fi
        
        # Check required configuration fields
        if jq -e '.allowed_categories and .max_confidence and .require_approval and .backup_enabled' /tmp/config_response.json > /dev/null; then
            echo -e "${GREEN}✓ Configuration structure valid${NC}"
        else
            echo -e "${RED}✗ Configuration structure invalid${NC}"
            return 1
        fi
    else
        echo -e "${RED}✗ Fix config endpoint failed (HTTP $http_code)${NC}"
        return 1
    fi
    
    rm -f /tmp/config_response.json
}

test_fix_enable_requires_confirmation() {
    echo -e "${YELLOW}Testing that fix enable requires explicit confirmation...${NC}"
    
    # Test without confirmation - should fail
    response=$(curl -s -w "%{http_code}" "$API_BASE_URL/api/v1/fix/config/enable" \
        -H "Content-Type: application/json" \
        -d '{"allowed_categories": ["Resource Leak"], "max_confidence": "high"}' \
        -o /tmp/enable_response.json)
    http_code="${response: -3}"
    
    if [[ "$http_code" == "400" ]]; then
        error_message=$(jq -r '.error' /tmp/enable_response.json 2>/dev/null || echo "No error message")
        if [[ "$error_message" == *"confirmation"* ]]; then
            echo -e "${GREEN}✓ Enable without confirmation properly rejected${NC}"
            echo -e "${BLUE}  Error: $error_message${NC}"
        else
            echo -e "${RED}✗ Enable rejected but wrong error message${NC}"
            return 1
        fi
    else
        echo -e "${RED}✗ CRITICAL: Enable without confirmation allowed (SAFETY VIOLATION)${NC}"
        return 1
    fi
    
    rm -f /tmp/enable_response.json
}

test_fix_enable_with_confirmation() {
    echo -e "${YELLOW}Testing fix enable with proper confirmation...${NC}"
    
    # Test with confirmation - should succeed
    response=$(curl -s -w "%{http_code}" "$API_BASE_URL/api/v1/fix/config/enable" \
        -H "Content-Type: application/json" \
        -d '{"allowed_categories": ["Resource Leak"], "max_confidence": "high", "confirmation_understood": true}' \
        -o /tmp/enable_confirmed_response.json)
    http_code="${response: -3}"
    
    if [[ "$http_code" == "200" ]]; then
        status=$(jq -r '.status' /tmp/enable_confirmed_response.json)
        warning=$(jq -r '.safety_warning' /tmp/enable_confirmed_response.json)
        
        if [[ "$status" == "enabled" ]]; then
            echo -e "${GREEN}✓ Automated fixes enabled with confirmation${NC}"
            echo -e "${BLUE}  Warning: $warning${NC}"
        else
            echo -e "${RED}✗ Enable with confirmation failed${NC}"
            return 1
        fi
    else
        echo -e "${RED}✗ Enable with confirmation failed (HTTP $http_code)${NC}"
        return 1
    fi
    
    rm -f /tmp/enable_confirmed_response.json
}

test_fix_apply_safety_checks() {
    echo -e "${YELLOW}Testing fix application safety checks...${NC}"
    
    # First disable fixes to test safety
    curl -s "$API_BASE_URL/api/v1/fix/config/disable" > /dev/null
    
    # Try to apply fix when disabled - should fail
    response=$(curl -s -w "%{http_code}" "$API_BASE_URL/api/v1/fix/apply/test-scenario" \
        -H "Content-Type: application/json" \
        -d '{"vulnerability_id": "test-vuln-123"}' \
        -o /tmp/apply_disabled_response.json)
    http_code="${response: -3}"
    
    if [[ "$http_code" == "403" ]]; then
        error_message=$(jq -r '.error' /tmp/apply_disabled_response.json 2>/dev/null || echo "No error message")
        if [[ "$error_message" == *"disabled"* ]]; then
            echo -e "${GREEN}✓ Fix application blocked when disabled${NC}"
            echo -e "${BLUE}  Error: $error_message${NC}"
        else
            echo -e "${RED}✗ Fix application blocked but wrong error message${NC}"
            return 1
        fi
    else
        echo -e "${RED}✗ CRITICAL: Fix application allowed when disabled (SAFETY VIOLATION)${NC}"
        return 1
    fi
    
    rm -f /tmp/apply_disabled_response.json
}

test_fix_disable_functionality() {
    echo -e "${YELLOW}Testing fix disable functionality...${NC}"
    
    # Enable first (for testing purposes)
    curl -s "$API_BASE_URL/api/v1/fix/config/enable" \
        -H "Content-Type: application/json" \
        -d '{"allowed_categories": ["Resource Leak"], "max_confidence": "high", "confirmation_understood": true}' \
        > /dev/null
    
    # Then disable
    response=$(curl -s -w "%{http_code}" "$API_BASE_URL/api/v1/fix/config/disable" -o /tmp/disable_response.json)
    http_code="${response: -3}"
    
    if [[ "$http_code" == "200" ]]; then
        status=$(jq -r '.status' /tmp/disable_response.json)
        safety_status=$(jq -r '.safety_status' /tmp/disable_response.json)
        
        if [[ "$status" == "disabled" ]]; then
            echo -e "${GREEN}✓ Automated fixes disabled successfully${NC}"
            echo -e "${BLUE}  Safety status: $safety_status${NC}"
            
            # Verify config shows disabled
            config_response=$(curl -s "$API_BASE_URL/api/v1/fix/config")
            enabled=$(echo "$config_response" | jq -r '.enabled')
            
            if [[ "$enabled" == "false" ]]; then
                echo -e "${GREEN}✓ Configuration confirms fixes are disabled${NC}"
            else
                echo -e "${RED}✗ Configuration still shows fixes enabled${NC}"
                return 1
            fi
        else
            echo -e "${RED}✗ Disable operation failed${NC}"
            return 1
        fi
    else
        echo -e "${RED}✗ Disable endpoint failed (HTTP $http_code)${NC}"
        return 1
    fi
    
    rm -f /tmp/disable_response.json
}

# Main test execution
main() {
    echo -e "${YELLOW}Starting Automated Fix Safety Tests...${NC}"
    echo
    
    local test_count=0
    local passed_count=0
    
    # Run tests
    tests=(
        "test_fix_config_disabled_by_default"
        "test_fix_enable_requires_confirmation"
        "test_fix_enable_with_confirmation"
        "test_fix_apply_safety_checks"
        "test_fix_disable_functionality"
    )
    
    for test in "${tests[@]}"; do
        test_count=$((test_count + 1))
        echo -e "${YELLOW}[Test $test_count] Running $test${NC}"
        
        if $test; then
            passed_count=$((passed_count + 1))
            echo -e "${GREEN}✓ $test PASSED${NC}"
        else
            echo -e "${RED}✗ $test FAILED${NC}"
        fi
        echo
    done
    
    # Clean up - ensure fixes are disabled at end
    echo -e "${YELLOW}Cleaning up - disabling automated fixes...${NC}"
    curl -s "$API_BASE_URL/api/v1/fix/config/disable" > /dev/null
    
    # Summary
    echo -e "${YELLOW}Test Summary:${NC}"
    echo "Total tests: $test_count"
    echo "Passed: $passed_count"
    echo "Failed: $((test_count - passed_count))"
    
    if [[ $passed_count -eq $test_count ]]; then
        echo -e "${GREEN}🎉 All safety tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}❌ Some tests failed${NC}"
        exit 1
    fi
}

# Run main function
main "$@"