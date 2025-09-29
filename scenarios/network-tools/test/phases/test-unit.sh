#!/bin/bash
# Unit tests for network-tools scenario

set -euo pipefail

# Colors for output
readonly GREEN='\033[0;32m'
readonly RED='\033[0;31m'
readonly YELLOW='\033[1;33m'
readonly NC='\033[0m' # No Color

# Test counter
TESTS_TOTAL=0
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
test_start() {
    local test_name="$1"
    echo -e "\n${YELLOW}Testing: ${test_name}...${NC}"
    ((TESTS_TOTAL++))
}

test_pass() {
    echo -e "${GREEN}✓ PASSED${NC}"
    ((TESTS_PASSED++))
}

test_fail() {
    local reason="${1:-Unknown reason}"
    echo -e "${RED}✗ FAILED: ${reason}${NC}"
    ((TESTS_FAILED++))
}

# Test Go API build
test_go_build() {
    test_start "Go API compilation"
    
    if [ -f "api/go.mod" ]; then
        cd api
        if go build -o test-build ./cmd/server/main.go 2>/dev/null; then
            rm -f test-build
            test_pass
        else
            test_fail "Go compilation failed"
        fi
        cd ..
    else
        test_fail "Go module not found"
    fi
}

# Test Go unit tests
test_go_unit() {
    test_start "Go unit tests"
    
    if [ -f "api/go.mod" ]; then
        cd api
        if go test ./... -v 2>/dev/null | grep -q "PASS"; then
            test_pass
        else
            # If no tests exist yet, that's okay
            if go test ./... -v 2>&1 | grep -q "no test files"; then
                echo -e "${YELLOW}⚠ No unit tests found (not a failure)${NC}"
                test_pass
            else
                test_fail "Go tests failed"
            fi
        fi
        cd ..
    else
        test_fail "Go module not found"
    fi
}

# Test database schema
test_database_schema() {
    test_start "Database schema validity"
    
    if [ -f "initialization/storage/postgres/schema.sql" ]; then
        # Check for basic SQL syntax (simplified check)
        if grep -q "CREATE TABLE" initialization/storage/postgres/schema.sql; then
            test_pass
        else
            test_fail "No CREATE TABLE statements found"
        fi
    else
        test_fail "Database schema not found"
    fi
}

# Test service.json structure
test_service_json() {
    test_start "service.json structure"
    
    if [ -f ".vrooli/service.json" ]; then
        # Validate JSON syntax
        if jq '.' .vrooli/service.json > /dev/null 2>&1; then
            # Check for required fields
            if jq -e '.version and .service and .components and .resources and .lifecycle' \
                .vrooli/service.json > /dev/null 2>&1; then
                test_pass
            else
                test_fail "Missing required fields in service.json"
            fi
        else
            test_fail "Invalid JSON in service.json"
        fi
    else
        test_fail "service.json not found"
    fi
}

# Test CLI script
test_cli_script() {
    test_start "CLI script syntax"
    
    if [ -f "cli/network-tools" ]; then
        # Check bash syntax
        if bash -n cli/network-tools 2>/dev/null; then
            test_pass
        else
            test_fail "CLI script has syntax errors"
        fi
    else
        test_fail "CLI script not found"
    fi
}

# Test Makefile
test_makefile() {
    test_start "Makefile structure"
    
    if [ -f "Makefile" ]; then
        # Check for essential targets
        if grep -q "^run:" Makefile && \
           grep -q "^test:" Makefile && \
           grep -q "^stop:" Makefile; then
            test_pass
        else
            test_fail "Missing essential Makefile targets"
        fi
    else
        test_fail "Makefile not found"
    fi
}

# Test configuration files
test_config_files() {
    test_start "Configuration files"
    
    errors=0
    
    # Check go.mod
    if [ ! -f "api/go.mod" ]; then
        echo "  Missing: api/go.mod"
        ((errors++))
    fi
    
    # Check PRD.md
    if [ ! -f "PRD.md" ]; then
        echo "  Missing: PRD.md"
        ((errors++))
    fi
    
    # Check README.md
    if [ ! -f "README.md" ]; then
        echo "  Missing: README.md"
        ((errors++))
    fi
    
    if [ $errors -eq 0 ]; then
        test_pass
    else
        test_fail "$errors configuration files missing"
    fi
}

# Test API endpoints are defined
test_api_endpoints() {
    test_start "API endpoint definitions"
    
    if [ -f "api/cmd/server/main.go" ]; then
        endpoints_found=0
        
        # Check for key endpoints
        grep -q "handleHTTPRequest" api/cmd/server/main.go && ((endpoints_found++))
        grep -q "handleDNSQuery" api/cmd/server/main.go && ((endpoints_found++))
        grep -q "handleConnectivityTest" api/cmd/server/main.go && ((endpoints_found++))
        grep -q "handleNetworkScan" api/cmd/server/main.go && ((endpoints_found++))
        grep -q "handleSSLValidation" api/cmd/server/main.go && ((endpoints_found++))
        grep -q "handleAPITest" api/cmd/server/main.go && ((endpoints_found++))
        
        if [ $endpoints_found -ge 5 ]; then
            test_pass
        else
            test_fail "Only $endpoints_found/6 endpoints found"
        fi
    else
        test_fail "API main.go not found"
    fi
}

# Test security features
test_security_features() {
    test_start "Security features"
    
    if [ -f "api/cmd/server/main.go" ]; then
        security_features=0
        
        # Check for security middleware
        grep -q "authMiddleware" api/cmd/server/main.go && ((security_features++))
        grep -q "corsMiddleware" api/cmd/server/main.go && ((security_features++))
        grep -q "rateLimitMiddleware" api/cmd/server/main.go && ((security_features++))
        
        if [ $security_features -eq 3 ]; then
            test_pass
        else
            test_fail "Only $security_features/3 security features found"
        fi
    else
        test_fail "API main.go not found"
    fi
}

# Test imports and dependencies
test_dependencies() {
    test_start "Go dependencies"
    
    if [ -f "api/go.mod" ]; then
        cd api
        if go mod verify 2>/dev/null; then
            test_pass
        else
            test_fail "Go module verification failed"
        fi
        cd ..
    else
        test_fail "Go module not found"
    fi
}

# Main test execution
main() {
    echo -e "${YELLOW}=== Network Tools Unit Tests ===${NC}"
    
    # Run tests
    test_go_build
    test_go_unit
    test_database_schema
    test_service_json
    test_cli_script
    test_makefile
    test_config_files
    test_api_endpoints
    test_security_features
    test_dependencies
    
    # Print summary
    echo -e "\n${YELLOW}=== Test Summary ===${NC}"
    echo -e "Total tests: ${TESTS_TOTAL}"
    echo -e "${GREEN}Passed: ${TESTS_PASSED}${NC}"
    echo -e "${RED}Failed: ${TESTS_FAILED}${NC}"
    
    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "\n${GREEN}All tests passed!${NC}"
        exit 0
    else
        echo -e "\n${RED}Some tests failed${NC}"
        exit 1
    fi
}

# Run tests
main "$@"