#!/usr/bin/env bash
################################################################################
# Neo4j Resource - Test Library Functions
# 
# Testing functions for Neo4j graph database validation
################################################################################

set -euo pipefail

# Source dependencies
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
NEO4J_RESOURCE_DIR="${APP_ROOT}/resources/neo4j"

# Source core functions
source "${NEO4J_RESOURCE_DIR}/lib/core.sh"

################################################################################
# Test Functions
################################################################################

#######################################
# Run smoke tests (quick health check)
# Returns:
#   0 if all pass, 1 if any fail
#######################################
neo4j_test_smoke() {
    local failed=0
    
    echo "=== Neo4j Smoke Tests ==="
    echo
    
    # Test 1: Container running
    echo -n "1. Container running... "
    if neo4j_is_running; then
        echo "✓ PASS"
    else
        echo "✗ FAIL"
        ((failed++))
    fi
    
    # Test 2: Health check responds
    echo -n "2. Health check... "
    if neo4j_health_check; then
        echo "✓ PASS"
    else
        echo "✗ FAIL"
        ((failed++))
    fi
    
    # Test 3: HTTP port accessible
    echo -n "3. HTTP port ${NEO4J_HTTP_PORT}... "
    if timeout 5 curl -sf "http://localhost:${NEO4J_HTTP_PORT}/" &>/dev/null; then
        echo "✓ PASS"
    else
        echo "✗ FAIL"
        ((failed++))
    fi
    
    # Test 4: Bolt port accessible
    echo -n "4. Bolt port ${NEO4J_BOLT_PORT}... "
    if timeout 5 nc -zv localhost "${NEO4J_BOLT_PORT}" &>/dev/null; then
        echo "✓ PASS"
    else
        echo "✗ FAIL"
        ((failed++))
    fi
    
    echo
    if [[ $failed -eq 0 ]]; then
        echo "✅ All smoke tests passed"
        return 0
    else
        echo "❌ $failed smoke tests failed"
        return 1
    fi
}

#######################################
# Run integration tests
# Returns:
#   0 if all pass, 1 if any fail
#######################################
neo4j_test_integration() {
    local failed=0
    
    echo "=== Neo4j Integration Tests ==="
    echo
    
    # Test 1: Create node
    echo -n "1. Create node... "
    if neo4j_query "CREATE (n:TestNode {name: 'test', created: timestamp()}) RETURN n" &>/dev/null; then
        echo "✓ PASS"
    else
        echo "✗ FAIL"
        ((failed++))
    fi
    
    # Test 2: Query node
    echo -n "2. Query node... "
    local result
    result=$(neo4j_query "MATCH (n:TestNode {name: 'test'}) RETURN count(n)" 2>/dev/null | tail -1)
    if [[ "$result" == "1" ]]; then
        echo "✓ PASS"
    else
        echo "✗ FAIL (expected 1, got $result)"
        ((failed++))
    fi
    
    # Test 3: Create relationship
    echo -n "3. Create relationship... "
    if neo4j_query "MATCH (n:TestNode {name: 'test'}) CREATE (n)-[:RELATES_TO]->(m:TestNode {name: 'related'}) RETURN m" &>/dev/null; then
        echo "✓ PASS"
    else
        echo "✗ FAIL"
        ((failed++))
    fi
    
    # Test 4: Query relationship
    echo -n "4. Query relationship... "
    local rel_count
    rel_count=$(neo4j_query "MATCH (n:TestNode)-[r:RELATES_TO]->() RETURN count(r)" 2>/dev/null | tail -1)
    if [[ "$rel_count" == "1" ]]; then
        echo "✓ PASS"
    else
        echo "✗ FAIL (expected 1, got $rel_count)"
        ((failed++))
    fi
    
    # Test 5: Delete test data
    echo -n "5. Cleanup test data... "
    if neo4j_query "MATCH (n:TestNode) DETACH DELETE n" &>/dev/null; then
        echo "✓ PASS"
    else
        echo "✗ FAIL"
        ((failed++))
    fi
    
    # Test 6: Verify cleanup
    echo -n "6. Verify cleanup... "
    local cleanup_count
    cleanup_count=$(neo4j_query "MATCH (n:TestNode) RETURN count(n)" 2>/dev/null | tail -1)
    if [[ "$cleanup_count" == "0" ]]; then
        echo "✓ PASS"
    else
        echo "✗ FAIL (expected 0, got $cleanup_count)"
        ((failed++))
    fi
    
    echo
    if [[ $failed -eq 0 ]]; then
        echo "✅ All integration tests passed"
        return 0
    else
        echo "❌ $failed integration tests failed"
        return 1
    fi
}

#######################################
# Run unit tests
# Returns:
#   0 if all pass, 1 if any fail
#######################################
neo4j_test_unit() {
    local failed=0
    
    echo "=== Neo4j Unit Tests ==="
    echo
    
    # Test 1: Version retrieval
    echo -n "1. Get version... "
    local version
    version=$(neo4j_get_version)
    if [[ "$version" != "unknown" ]]; then
        echo "✓ PASS (version: $version)"
    else
        echo "✗ FAIL"
        ((failed++))
    fi
    
    # Test 2: Stats retrieval
    echo -n "2. Get stats... "
    local stats
    stats=$(neo4j_get_stats)
    if [[ "$stats" =~ "nodes" ]] && [[ "$stats" =~ "relationships" ]]; then
        echo "✓ PASS"
    else
        echo "✗ FAIL"
        ((failed++))
    fi
    
    # Test 3: Container name
    echo -n "3. Container name... "
    if [[ "$NEO4J_CONTAINER_NAME" == "vrooli-neo4j" ]]; then
        echo "✓ PASS"
    else
        echo "✗ FAIL (expected vrooli-neo4j, got $NEO4J_CONTAINER_NAME)"
        ((failed++))
    fi
    
    # Test 4: Port configuration
    echo -n "4. Port configuration... "
    if [[ "$NEO4J_HTTP_PORT" == "7474" ]] && [[ "$NEO4J_BOLT_PORT" == "7687" ]]; then
        echo "✓ PASS"
    else
        echo "✗ FAIL"
        ((failed++))
    fi
    
    echo
    if [[ $failed -eq 0 ]]; then
        echo "✅ All unit tests passed"
        return 0
    else
        echo "❌ $failed unit tests failed"
        return 1
    fi
}

#######################################
# Run all tests
# Returns:
#   0 if all pass, 1 if any fail
#######################################
neo4j_test_all() {
    local overall_failed=0
    
    echo "Running all Neo4j tests..."
    echo "=========================="
    echo
    
    # Run smoke tests
    if neo4j_test_smoke; then
        echo "✓ Smoke tests passed"
    else
        echo "✗ Smoke tests failed"
        ((overall_failed++))
    fi
    echo
    
    # Run integration tests
    if neo4j_test_integration; then
        echo "✓ Integration tests passed"
    else
        echo "✗ Integration tests failed"
        ((overall_failed++))
    fi
    echo
    
    # Run unit tests
    if neo4j_test_unit; then
        echo "✓ Unit tests passed"
    else
        echo "✗ Unit tests failed"
        ((overall_failed++))
    fi
    echo
    
    echo "=========================="
    if [[ $overall_failed -eq 0 ]]; then
        echo "✅ ALL TESTS PASSED"
        return 0
    else
        echo "❌ $overall_failed TEST SUITES FAILED"
        return 1
    fi
}

# Create wrapper functions for v2.0 framework compatibility
neo4j::test::smoke() {
    neo4j_test_smoke "$@"
}

neo4j::test::integration() {
    neo4j_test_integration "$@"
}

neo4j::test::unit() {
    neo4j_test_unit "$@"
}

neo4j::test::all() {
    neo4j_test_all "$@"
}

# Export test functions
export -f neo4j_test_smoke neo4j_test_integration neo4j_test_unit neo4j_test_all
export -f neo4j::test::smoke neo4j::test::integration neo4j::test::unit neo4j::test::all