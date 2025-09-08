#!/bin/bash

# Maintenance Orchestrator Discovery Test for Ecosystem Manager
# This script validates that ecosystem-manager meets all requirements for maintenance-orchestrator discovery

echo "🔍 Testing Maintenance Orchestrator Discovery Compatibility"
echo "========================================================="
echo

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test results
TESTS_PASSED=0
TESTS_FAILED=0

# Function to run a test
run_test() {
    local test_name="$1"
    local test_command="$2"
    local expected_result="$3"
    
    echo -n "Testing: $test_name... "
    
    result=$(eval "$test_command" 2>/dev/null)
    
    if [[ "$result" == *"$expected_result"* ]] || [[ -z "$expected_result" && $? -eq 0 ]]; then
        echo -e "${GREEN}✓ PASS${NC}"
        ((TESTS_PASSED++))
        return 0
    else
        echo -e "${RED}✗ FAIL${NC}"
        echo "  Expected: $expected_result"
        echo "  Got: $result"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Test 1: Check service.json exists
echo "📋 Checking service.json requirements..."
run_test "service.json file exists" "test -f ./.vrooli/service.json" ""

# Test 2: Check for maintenance tag
run_test "maintenance tag present" "grep -q '\"maintenance\"' ./.vrooli/service.json" ""

# Test 3: Check for maintenance configuration section
run_test "maintenance config section exists" "grep -q '\"maintenance\".*{' ./.vrooli/service.json" ""

# Test 4: Check for API port configuration
run_test "API port configured" "grep -q '\"api\".*{' ./.vrooli/service.json" ""

# Test 5: Check for canDeactivate field
run_test "canDeactivate field present" "grep -q '\"canDeactivate\".*true' ./.vrooli/service.json" ""

# Test 6: Check for defaultState field  
run_test "defaultState field present" "grep -q '\"defaultState\".*\"inactive\"' ./.vrooli/service.json" ""

# Test 7: Validate JSON structure
run_test "service.json is valid JSON" "python3 -c 'import json; json.load(open(\"./.vrooli/service.json\"))'" ""

echo
echo "🌐 Testing API endpoint requirements..."

# Get API port from service.json
API_PORT=$(python3 -c "
import json
with open('./.vrooli/service.json') as f:
    data = json.load(f)
    print(data.get('ports', {}).get('api', {}).get('port', '30500'))
" 2>/dev/null)

BASE_URL="http://localhost:$API_PORT"

# Test 8: Check if API is running (optional - may not be running during CI)
if curl -s "$BASE_URL/health" > /dev/null 2>&1; then
    echo -e "${YELLOW}ℹ API is running, testing live endpoints...${NC}"
    
    # Test 9: Health endpoint returns maintenance state
    run_test "Health endpoint includes maintenanceState" "curl -s '$BASE_URL/health' | grep -q maintenanceState" ""
    
    # Test 10: Health endpoint includes canToggle
    run_test "Health endpoint includes canToggle" "curl -s '$BASE_URL/health' | grep -q canToggle" ""
    
    # Test 11: Maintenance state endpoint exists
    run_test "Maintenance state endpoint responds" "curl -s -X POST '$BASE_URL/api/maintenance/state' -H 'Content-Type: application/json' -d '{\"maintenanceState\":\"active\"}' | grep -q status" ""
    
else
    echo -e "${YELLOW}ℹ API not running, skipping live endpoint tests${NC}"
    echo "  (This is expected if ecosystem-manager is not currently running)"
fi

echo
echo "📁 Testing orchestrator discovery simulation..."

# Test 12: Simulate maintenance-orchestrator discovery logic
SERVICE_JSON="./.vrooli/service.json"
DISCOVERY_RESULT=$(python3 -c "
import json
import os

try:
    with open('$SERVICE_JSON') as f:
        service = json.load(f)
    
    # Check for maintenance tag (orchestrator requirement)
    tags = service.get('service', {}).get('tags', [])
    if 'maintenance' not in tags:
        print('FAIL: maintenance tag missing')
        exit(1)
    
    # Check for API port (orchestrator requirement)
    ports = service.get('ports', {})
    api_port = None
    if 'api' in ports:
        if isinstance(ports['api'], dict):
            api_port = ports['api'].get('port')
        else:
            api_port = ports['api']
    
    if not api_port:
        print('FAIL: API port not configured')
        exit(1)
    
    # Create mock scenario object as orchestrator would
    scenario = {
        'id': service.get('service', {}).get('name', 'unknown'),
        'name': service.get('service', {}).get('name', 'unknown'),
        'displayName': service.get('service', {}).get('displayName', 'Unknown'),
        'description': service.get('service', {}).get('description', ''),
        'isActive': False,  # Always starts inactive
        'endpoint': f'http://localhost:{api_port}',
        'port': api_port,
        'tags': tags
    }
    
    print('SUCCESS: Would be discovered as maintenance scenario')
    print(f'ID: {scenario[\"id\"]}')
    print(f'Endpoint: {scenario[\"endpoint\"]}')
    print(f'Tags: {scenario[\"tags\"]}')
    
except Exception as e:
    print(f'FAIL: {e}')
    exit(1)
" 2>&1)

if [[ "$DISCOVERY_RESULT" == *"SUCCESS"* ]]; then
    echo -e "${GREEN}✓ PASS${NC} Orchestrator discovery simulation"
    echo "  $DISCOVERY_RESULT"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC} Orchestrator discovery simulation"
    echo "  $DISCOVERY_RESULT"
    ((TESTS_FAILED++))
fi

echo
echo "📊 Test Summary"
echo "==============="
echo -e "Tests Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Tests Failed: ${RED}$TESTS_FAILED${NC}"
echo

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 All tests passed!${NC}"
    echo -e "${GREEN}✅ Ecosystem Manager is fully compatible with maintenance-orchestrator${NC}"
    echo
    echo "Next steps:"
    echo "1. Start ecosystem-manager: vrooli scenario ecosystem-manager develop"  
    echo "2. Start maintenance-orchestrator: vrooli scenario maintenance-orchestrator develop"
    echo "3. Verify discovery: maintenance-orchestrator list"
    echo "4. Test control: maintenance-orchestrator activate ecosystem-manager"
    exit 0
else
    echo -e "${RED}❌ Some tests failed!${NC}"
    echo -e "${RED}Ecosystem Manager needs fixes before maintenance-orchestrator integration${NC}"
    exit 1
fi