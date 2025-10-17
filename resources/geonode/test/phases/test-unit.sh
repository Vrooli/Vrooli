#!/bin/bash

# GeoNode Unit Tests

set -euo pipefail

# Force unbuffered output
exec 2>&1

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

# Test counter
TESTS_RUN=0
TESTS_FAILED=0

# Test function
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    ((TESTS_RUN++))
    echo -n "  Testing ${test_name}... "
    
    if eval "$test_command" &>/dev/null; then
        echo -e "${GREEN}✓${NC}"
        return 0
    else
        echo -e "${RED}✗${NC}"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Main unit tests
main() {
    echo "Running GeoNode unit tests..."
    echo ""
    
    # Source the library and config
    source "${RESOURCE_DIR}/lib/core.sh"
    load_config
    
    echo "Configuration tests:"
    
    run_test "Configuration loading" "[[ -n \$GEONODE_PORT ]]"
    run_test "Port configuration" "[[ \$GEONODE_PORT -gt 1024 ]]"
    run_test "Database configuration" "[[ -n \$GEONODE_DB_NAME ]]"
    run_test "Admin configuration" "[[ -n \$GEONODE_ADMIN_USER ]]"
    
    echo ""
    echo "Function tests:"
    
    run_test "Status function exists" "type -t get_status >/dev/null 2>&1"
    run_test "Health check function exists" "type -t is_healthy >/dev/null 2>&1"
    run_test "Service health function exists" "type -t is_service_healthy >/dev/null 2>&1"
    
    echo ""
    echo "File structure tests:"
    
    run_test "CLI script exists" "[[ -f ${RESOURCE_DIR}/cli.sh ]]"
    run_test "Core library exists" "[[ -f ${RESOURCE_DIR}/lib/core.sh ]]"
    run_test "Test library exists" "[[ -f ${RESOURCE_DIR}/lib/test.sh ]]"
    run_test "Config directory exists" "[[ -d ${RESOURCE_DIR}/config ]]"
    run_test "Defaults config exists" "[[ -f ${RESOURCE_DIR}/config/defaults.sh ]]"
    run_test "Schema config exists" "[[ -f ${RESOURCE_DIR}/config/schema.json ]]"
    run_test "Runtime config exists" "[[ -f ${RESOURCE_DIR}/config/runtime.json ]]"
    
    echo ""
    echo "Docker configuration tests:"
    
    run_test "Docker directory exists" "[[ -d ${RESOURCE_DIR}/docker ]]"
    run_test "Docker Compose creation" "create_docker_compose && [[ -f \$COMPOSE_FILE ]]"
    
    echo ""
    echo "JSON validation tests:"
    
    run_test "Schema JSON valid" "jq empty ${RESOURCE_DIR}/config/schema.json"
    run_test "Runtime JSON valid" "jq empty ${RESOURCE_DIR}/config/runtime.json"
    
    echo ""
    echo "Unit test summary:"
    echo "  Tests run: ${TESTS_RUN}"
    echo "  Tests failed: ${TESTS_FAILED}"
    
    if [[ $TESTS_FAILED -gt 0 ]]; then
        echo -e "${RED}Unit tests failed${NC}"
        exit 1
    else
        echo -e "${GREEN}All unit tests passed${NC}"
        exit 0
    fi
}

main "$@"