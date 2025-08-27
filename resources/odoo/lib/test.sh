#!/bin/bash
# Test functions for Odoo resource

odoo_test() {
    echo "Running Odoo integration tests..."
    
    local test_results=()
    local total_tests=0
    local passed_tests=0
    
    # Test 1: Check if Odoo is installed
    echo -n "Testing Odoo installation... "
    ((total_tests++))
    if odoo_is_installed; then
        echo "✓"
        ((passed_tests++))
        test_results+=("install:passed")
    else
        echo "✗"
        test_results+=("install:failed")
    fi
    
    # Test 2: Check if Odoo can start
    echo -n "Testing Odoo startup... "
    ((total_tests++))
    if ! odoo_is_running; then
        odoo_start >/dev/null 2>&1
        sleep 10
    fi
    
    if odoo_is_running; then
        echo "✓"
        ((passed_tests++))
        test_results+=("startup:passed")
    else
        echo "✗"
        test_results+=("startup:failed")
    fi
    
    # Test 3: Check HTTP endpoint
    echo -n "Testing Odoo HTTP endpoint... "
    ((total_tests++))
    if curl -s -f "http://localhost:$ODOO_PORT/web/database/selector" | grep -q "Odoo" 2>/dev/null; then
        echo "✓"
        ((passed_tests++))
        test_results+=("http:passed")
    else
        echo "✗"
        test_results+=("http:failed")
    fi
    
    # Test 4: Check XML-RPC endpoint
    echo -n "Testing XML-RPC endpoint... "
    ((total_tests++))
    local xmlrpc_test=$(python3 -c "
import xmlrpc.client
try:
    common = xmlrpc.client.ServerProxy('http://localhost:$ODOO_PORT/xmlrpc/2/common')
    version = common.version()
    print('success' if version else 'failed')
except:
    print('failed')
" 2>/dev/null)
    
    if [[ "$xmlrpc_test" == "success" ]]; then
        echo "✓"
        ((passed_tests++))
        test_results+=("xmlrpc:passed")
    else
        echo "✗"
        test_results+=("xmlrpc:failed")
    fi
    
    # Test 5: Check database connectivity
    echo -n "Testing database connectivity... "
    ((total_tests++))
    if docker exec "$ODOO_PG_CONTAINER_NAME" psql -U "$ODOO_DB_USER" -d "$ODOO_DB_NAME" -c "SELECT 1;" &>/dev/null; then
        echo "✓"
        ((passed_tests++))
        test_results+=("database:passed")
    else
        echo "✗"
        test_results+=("database:failed")
    fi
    
    # Test 6: Check module installation capability
    echo -n "Testing module installation... "
    ((total_tests++))
    if odoo_install_module "base" &>/dev/null; then
        echo "✓"
        ((passed_tests++))
        test_results+=("module_install:passed")
    else
        echo "✗"
        test_results+=("module_install:failed")
    fi
    
    # Save test results
    local test_status="passed"
    if [[ $passed_tests -lt $total_tests ]]; then
        test_status="partial"
    fi
    if [[ $passed_tests -eq 0 ]]; then
        test_status="failed"
    fi
    
    echo "$test_status" > "$ODOO_DATA_DIR/.last_test"
    
    # Summary
    echo ""
    echo "Test Results: $passed_tests/$total_tests passed"
    echo "Test Status: $test_status"
    
    # Return based on results
    if [[ $passed_tests -eq $total_tests ]]; then
        return 0
    elif [[ $passed_tests -gt 0 ]]; then
        return 1
    else
        return 2
    fi
}

# v2.0 naming convention handlers
odoo::test::smoke() {
    # Smoke test is just a status check - tests the resource itself
    log::info "Running Odoo smoke test..."
    odoo::status::check
}

odoo::test::integration() {
    # Full integration test suite
    odoo_test "$@"
}

export -f odoo_test
export -f odoo::test::smoke
export -f odoo::test::integration