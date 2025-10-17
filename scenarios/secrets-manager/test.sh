#!/usr/bin/env bash
# Secrets Manager Integration Test Suite
# Dark Chrome Security Terminal Test Automation

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_NAME="secrets-manager"
API_PORT="${SECRETS_API_URL##*:}"
API_PORT="${API_PORT:-28000}"
UI_PORT="${SECRETS_UI_URL##*:}"
UI_PORT="${UI_PORT:-28250}"

# Cyberpunk color scheme for test output
RED='\033[1;31m'
GREEN='\033[1;32m'
YELLOW='\033[1;33m'
CYAN='\033[1;36m'
SILVER='\033[0;37m'
MATRIX_GREEN='\033[1;92m'
NC='\033[0m'

# Test results tracking
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0
FAILED_TESTS=()

echo -e "${SILVER}🔐 SECRETS MANAGER - Security Test Matrix${NC}"
echo -e "${SILVER}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}Testing Dark Chrome Security Dashboard${NC}"
echo -e "${SILVER}API Port: $API_PORT | UI Port: $UI_PORT${NC}"
echo -e "${SILVER}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"

# Test helper functions
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    echo -e "${CYAN}┌─ TESTING: $test_name${NC}"
    TESTS_RUN=$((TESTS_RUN + 1))
    
    if eval "$test_command"; then
        echo -e "${MATRIX_GREEN}│ ✅ PASSED${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "${RED}│ ❌ FAILED${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        FAILED_TESTS+=("$test_name")
    fi
    echo -e "${CYAN}└──${NC}\n"
}

wait_for_service() {
    local service_name="$1"
    local port="$2"
    local max_attempts=30
    local attempt=1
    
    echo -e "${YELLOW}⏳ Waiting for $service_name on port $port...${NC}"
    
    while [ $attempt -le $max_attempts ]; do
        if curl -sf "http://localhost:$port/health" > /dev/null 2>&1; then
            echo -e "${MATRIX_GREEN}✅ $service_name is ready${NC}"
            return 0
        fi
        
        echo -e "${SILVER}   Attempt $attempt/$max_attempts...${NC}"
        sleep 2
        attempt=$((attempt + 1))
    done
    
    echo -e "${RED}❌ $service_name failed to start${NC}"
    return 1
}

# Test functions
test_api_health() {
    local response
    response=$(curl -sf "http://localhost:$API_PORT/health")
    echo "$response" | jq -e '.status == "healthy"' > /dev/null
}

test_api_scan_endpoint() {
    local response
    response=$(curl -sf "http://localhost:$API_PORT/api/v1/secrets/scan")
    echo "$response" | jq -e '.discovered_secrets' > /dev/null
}

test_api_validate_endpoint() {
    local response
    response=$(curl -sf "http://localhost:$API_PORT/api/v1/secrets/validate")
    echo "$response" | jq -e '.total_secrets >= 0' > /dev/null
}

test_ui_loads() {
    curl -sf "http://localhost:$UI_PORT" | grep -q "Secrets Manager"
}

test_ui_health() {
    local response
    response=$(curl -sf "http://localhost:$UI_PORT/health")
    echo "$response" | jq -e '.service == "secrets-manager-ui"' > /dev/null
}

test_cli_available() {
    command -v secrets-manager > /dev/null
}

test_cli_status() {
    secrets-manager status --help > /dev/null
}

test_database_schema() {
    # Test if required tables exist in database
    PGPASSWORD="${POSTGRES_PASSWORD:-postgres}" psql -h localhost -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-vrooli}" \
        -c "SELECT 1 FROM resource_secrets LIMIT 1;" > /dev/null 2>&1 &&
    PGPASSWORD="${POSTGRES_PASSWORD:-postgres}" psql -h localhost -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-vrooli}" \
        -c "SELECT 1 FROM secret_validations LIMIT 1;" > /dev/null 2>&1
}

test_secret_discovery() {
    # Test actual secret discovery functionality
    local response
    response=$(curl -sf -X POST -H "Content-Type: application/json" \
        -d '{"scan_type": "quick", "resources": ["postgres"]}' \
        "http://localhost:$API_PORT/api/v1/secrets/scan")
    
    echo "$response" | jq -e '.discovered_secrets | length >= 0' > /dev/null
}

test_secret_validation() {
    # Test validation functionality
    local response
    response=$(curl -sf -X POST -H "Content-Type: application/json" \
        -d '{"resource": "postgres"}' \
        "http://localhost:$API_PORT/api/v1/secrets/validate")
    
    echo "$response" | jq -e '.validation_id' > /dev/null
}

test_dark_chrome_ui_elements() {
    # Test that the UI contains cyberpunk/dark chrome elements
    local html_content
    html_content=$(curl -sf "http://localhost:$UI_PORT")
    
    echo "$html_content" | grep -q "dark-chrome" &&
    echo "$html_content" | grep -q "SECURITY VAULT" &&
    echo "$html_content" | grep -q "matrix-green"
}

test_resource_filter() {
    # Test UI resource filtering functionality
    local html_content
    html_content=$(curl -sf "http://localhost:$UI_PORT")
    echo "$html_content" | grep -q "resource-filter"
}

test_security_alerts() {
    # Test security alert generation
    local response
    response=$(curl -sf -X POST -H "Content-Type: application/json" \
        -d '{}' "http://localhost:$API_PORT/api/v1/secrets/validate")
    
    # Should have security alerts structure
    echo "$response" | jq -e 'has("missing_secrets") and has("invalid_secrets")' > /dev/null
}

# Main test execution
main() {
    echo -e "${MATRIX_GREEN}🔍 INITIATING SECURITY SYSTEM SCAN${NC}\n"
    
    # Wait for services to be ready
    wait_for_service "Secrets Manager API" "$API_PORT" || exit 1
    wait_for_service "Dark Chrome UI" "$UI_PORT" || exit 1
    
    echo -e "${SILVER}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${MATRIX_GREEN}🔐 SECURITY VALIDATION MATRIX${NC}"
    echo -e "${SILVER}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
    
    # Core API Tests
    run_test "API Health Check" test_api_health
    run_test "Secrets Scan Endpoint" test_api_scan_endpoint  
    run_test "Secrets Validation Endpoint" test_api_validate_endpoint
    
    # UI Tests
    run_test "Dark Chrome UI Loads" test_ui_loads
    run_test "UI Health Check" test_ui_health
    run_test "Dark Chrome UI Elements" test_dark_chrome_ui_elements
    run_test "Resource Filter UI" test_resource_filter
    
    # CLI Tests
    run_test "CLI Command Available" test_cli_available
    run_test "CLI Status Command" test_cli_status
    
    # Database Tests
    run_test "Database Schema" test_database_schema
    
    # Functional Tests
    run_test "Secret Discovery Pipeline" test_secret_discovery
    run_test "Secret Validation Pipeline" test_secret_validation
    run_test "Security Alert Generation" test_security_alerts
    
    # Run custom tests if available
    if [[ -f "$SCRIPT_DIR/custom-tests.sh" ]]; then
        echo -e "${SILVER}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${MATRIX_GREEN}🧪 CUSTOM SECURITY TESTS${NC}"
        echo -e "${SILVER}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
        
        source "$SCRIPT_DIR/custom-tests.sh"
        
        run_test "Dark Chrome UI Accessibility" test_dark_chrome_ui_accessibility
        run_test "Resource Secret Discovery" test_resource_secret_discovery
        run_test "Vault Integration" test_vault_secret_storage
    fi
    
    # Test summary
    echo -e "${SILVER}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${MATRIX_GREEN}🔐 SECURITY TEST MATRIX COMPLETE${NC}"
    echo -e "${SILVER}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
    
    echo -e "${CYAN}📊 TEST RESULTS:${NC}"
    echo -e "   ${MATRIX_GREEN}✅ Passed: $TESTS_PASSED${NC}"
    echo -e "   ${RED}❌ Failed: $TESTS_FAILED${NC}"
    echo -e "   📈 Total:  $TESTS_RUN${NC}\n"
    
    if [[ $TESTS_FAILED -gt 0 ]]; then
        echo -e "${RED}🚨 FAILED TESTS:${NC}"
        for test in "${FAILED_TESTS[@]}"; do
            echo -e "   ${RED}• $test${NC}"
        done
        echo
        
        echo -e "${RED}❌ SECURITY VALIDATION FAILED${NC}"
        exit 1
    else
        echo -e "${MATRIX_GREEN}✅ ALL SECURITY TESTS PASSED${NC}"
        echo -e "${SILVER}🔐 Dark Chrome Security Terminal: OPERATIONAL${NC}"
        exit 0
    fi
}

# Execute main function
main "$@"