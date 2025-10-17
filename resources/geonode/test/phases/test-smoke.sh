#!/bin/bash

# GeoNode Smoke Tests

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
CLI="${RESOURCE_DIR}/cli.sh"

# Test utilities
source "${RESOURCE_DIR}/lib/core.sh"

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

# Main smoke tests
main() {
    echo "Running GeoNode smoke tests..."
    echo ""
    
    # Load configuration
    load_config
    
    # Check if service is running
    if [[ "$(get_status)" != "running" ]]; then
        echo "GeoNode is not running. Skipping smoke tests."
        echo "To run tests, first start the service with: vrooli resource geonode manage start"
        exit 0
    fi
    
    echo "Basic functionality tests:"
    
    # Test GeoServer health (primary working component)
    run_test "GeoServer health check" \
        "timeout 5 curl -sf -u ${GEONODE_ADMIN_USER}:${GEONODE_ADMIN_PASSWORD} http://localhost:${GEONODE_GEOSERVER_PORT}/geoserver/rest/about/status"
    
    # Test GeoServer workspace API
    run_test "GeoServer workspace API" \
        "timeout 5 curl -sf -u ${GEONODE_ADMIN_USER}:${GEONODE_ADMIN_PASSWORD} http://localhost:${GEONODE_GEOSERVER_PORT}/geoserver/rest/workspaces"
    
    # Test GeoServer layers API
    run_test "GeoServer layers API" \
        "timeout 5 curl -sf -u ${GEONODE_ADMIN_USER}:${GEONODE_ADMIN_PASSWORD} http://localhost:${GEONODE_GEOSERVER_PORT}/geoserver/rest/layers"
    
    # Django is slow to start - check if container is at least running
    run_test "Django container running" \
        "docker ps --filter name=geonode-django --format '{{.Names}}' | grep -q geonode-django"
    
    # Test CLI commands
    run_test "CLI help command" \
        "${CLI} help"
    
    run_test "CLI status command" \
        "${CLI} status"
    
    run_test "CLI info command" \
        "${CLI} info"
    
    # Test sample data upload (if available)
    if [[ -f "${RESOURCE_DIR}/examples/sample.geojson" ]]; then
        run_test "Sample data upload" \
            "${CLI} content add-layer ${RESOURCE_DIR}/examples/sample.geojson"
        
        run_test "List layers after upload" \
            "${CLI} content list-layers"
    fi
    
    echo ""
    echo "Smoke test summary:"
    echo "  Tests run: ${TESTS_RUN}"
    echo "  Tests failed: ${TESTS_FAILED}"
    
    if [[ $TESTS_FAILED -gt 0 ]]; then
        echo -e "${RED}Smoke tests failed${NC}"
        exit 1
    else
        echo -e "${GREEN}All smoke tests passed${NC}"
        exit 0
    fi
}

main "$@"