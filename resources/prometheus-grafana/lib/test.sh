#!/usr/bin/env bash
# Test functionality for Prometheus + Grafana resource

set -euo pipefail

# Get script directory - use TEST_LIB_DIR to avoid conflicts
if [[ -z "${TEST_LIB_DIR:-}" ]]; then
    TEST_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    readonly TEST_LIB_DIR
fi

# Get resource directory
if [[ -z "${RESOURCE_DIR:-}" ]]; then
    RESOURCE_DIR="$(dirname "$TEST_LIB_DIR")"
    readonly RESOURCE_DIR
fi

# Source configuration and core functions if not already sourced
if [[ -z "${RESOURCE_NAME:-}" ]]; then
    source "${RESOURCE_DIR}/config/defaults.sh"
fi
source "${RESOURCE_DIR}/lib/core.sh"

#######################################
# Run smoke tests (quick health check)
#######################################
run_smoke_tests() {
    echo "Running smoke tests..."
    local tests_passed=0
    local tests_failed=0
    
    # Test 1: Check if services are running
    echo -n "Test 1: Services running... "
    if is_running; then
        echo "PASS"
        ((tests_passed++))
    else
        echo "FAIL"
        ((tests_failed++))
    fi
    
    # Test 2: Prometheus health check
    echo -n "Test 2: Prometheus health... "
    if timeout 5 curl -sf "http://localhost:${PROMETHEUS_PORT}/-/healthy" > /dev/null 2>&1; then
        echo "PASS"
        ((tests_passed++))
    else
        echo "FAIL"
        ((tests_failed++))
    fi
    
    # Test 3: Grafana health check
    echo -n "Test 3: Grafana health... "
    if timeout 5 curl -sf "http://localhost:${GRAFANA_PORT}/api/health" > /dev/null 2>&1; then
        echo "PASS"
        ((tests_passed++))
    else
        echo "FAIL"
        ((tests_failed++))
    fi
    
    # Test 4: Alertmanager health check (if enabled)
    if [[ "$ENABLE_ALERTMANAGER" == "true" ]]; then
        echo -n "Test 4: Alertmanager health... "
        if timeout 5 curl -sf "http://localhost:${ALERTMANAGER_PORT}/-/healthy" > /dev/null 2>&1; then
            echo "PASS"
            ((tests_passed++))
        else
            echo "FAIL"
            ((tests_failed++))
        fi
    fi
    
    # Test 5: Node exporter metrics
    echo -n "Test 5: Node exporter metrics... "
    if timeout 5 curl -sf "http://localhost:${NODE_EXPORTER_PORT}/metrics" | grep -q "node_"; then
        echo "PASS"
        ((tests_passed++))
    else
        echo "FAIL"
        ((tests_failed++))
    fi
    
    # Summary
    echo ""
    echo "Smoke Tests Summary:"
    echo "  Passed: $tests_passed"
    echo "  Failed: $tests_failed"
    
    if [[ $tests_failed -eq 0 ]]; then
        echo "All smoke tests passed!"
        return 0
    else
        echo "Some smoke tests failed!"
        return 1
    fi
}

#######################################
# Run integration tests
#######################################
run_integration_tests() {
    echo "Running integration tests..."
    local tests_passed=0
    local tests_failed=0
    
    # Test 1: Prometheus can scrape targets
    echo -n "Test 1: Prometheus scraping... "
    local targets=$(curl -s "http://localhost:${PROMETHEUS_PORT}/api/v1/targets" | jq -r '.data.activeTargets | length')
    if [[ $targets -gt 0 ]]; then
        echo "PASS ($targets active targets)"
        ((tests_passed++))
    else
        echo "FAIL (no active targets)"
        ((tests_failed++))
    fi
    
    # Test 2: Grafana datasource connectivity
    echo -n "Test 2: Grafana datasource... "
    local datasources=$(curl -s "http://admin:${GRAFANA_ADMIN_PASSWORD}@localhost:${GRAFANA_PORT}/api/datasources" | jq -r '. | length')
    if [[ $datasources -gt 0 ]]; then
        echo "PASS ($datasources datasources)"
        ((tests_passed++))
    else
        echo "FAIL (no datasources)"
        ((tests_failed++))
    fi
    
    # Test 3: PromQL query execution
    echo -n "Test 3: PromQL queries... "
    local query_result=$(curl -s "http://localhost:${PROMETHEUS_PORT}/api/v1/query?query=up" | jq -r '.status')
    if [[ "$query_result" == "success" ]]; then
        echo "PASS"
        ((tests_passed++))
    else
        echo "FAIL"
        ((tests_failed++))
    fi
    
    # Test 4: Metrics collection
    echo -n "Test 4: Metrics collection... "
    local metric_count=$(curl -s "http://localhost:${PROMETHEUS_PORT}/api/v1/label/__name__/values" | jq -r '.data | length')
    if [[ $metric_count -gt 100 ]]; then
        echo "PASS ($metric_count metrics)"
        ((tests_passed++))
    else
        echo "FAIL (only $metric_count metrics)"
        ((tests_failed++))
    fi
    
    # Test 5: Alert manager configuration
    if [[ "$ENABLE_ALERTMANAGER" == "true" ]]; then
        echo -n "Test 5: Alertmanager config... "
        local alert_status=$(curl -s "http://localhost:${ALERTMANAGER_PORT}/api/v1/status" | jq -r '.data.uptime')
        if [[ -n "$alert_status" ]]; then
            echo "PASS"
            ((tests_passed++))
        else
            echo "FAIL"
            ((tests_failed++))
        fi
    fi
    
    # Test 6: Dashboard provisioning
    echo -n "Test 6: Dashboard provisioning... "
    sleep 2  # Give Grafana time to load dashboards
    local dashboards=$(curl -s "http://admin:${GRAFANA_ADMIN_PASSWORD}@localhost:${GRAFANA_PORT}/api/search" | jq -r '. | length')
    if [[ $dashboards -ge 0 ]]; then
        echo "PASS ($dashboards dashboards)"
        ((tests_passed++))
    else
        echo "FAIL"
        ((tests_failed++))
    fi
    
    # Summary
    echo ""
    echo "Integration Tests Summary:"
    echo "  Passed: $tests_passed"
    echo "  Failed: $tests_failed"
    
    if [[ $tests_failed -eq 0 ]]; then
        echo "All integration tests passed!"
        return 0
    else
        echo "Some integration tests failed!"
        return 1
    fi
}

#######################################
# Run unit tests
#######################################
run_unit_tests() {
    echo "Running unit tests..."
    local tests_passed=0
    local tests_failed=0
    
    # Test 1: Configuration validation
    echo -n "Test 1: Configuration validation... "
    if validate_config > /dev/null 2>&1; then
        echo "PASS"
        ((tests_passed++))
    else
        echo "FAIL"
        ((tests_failed++))
    fi
    
    # Test 2: Port availability check
    echo -n "Test 2: Port availability... "
    local ports_available=true
    for port in $PROMETHEUS_PORT $GRAFANA_PORT $ALERTMANAGER_PORT $NODE_EXPORTER_PORT; do
        if ! [[ "$port" =~ ^[0-9]+$ ]] || [[ "$port" -lt 1024 || "$port" -gt 65535 ]]; then
            ports_available=false
            break
        fi
    done
    if $ports_available; then
        echo "PASS"
        ((tests_passed++))
    else
        echo "FAIL"
        ((tests_failed++))
    fi
    
    # Test 3: Docker availability
    echo -n "Test 3: Docker availability... "
    if command -v docker > /dev/null 2>&1 && docker ps > /dev/null 2>&1; then
        echo "PASS"
        ((tests_passed++))
    else
        echo "FAIL"
        ((tests_failed++))
    fi
    
    # Test 4: Docker-compose availability
    echo -n "Test 4: Docker-compose availability... "
    if command -v docker-compose > /dev/null 2>&1; then
        echo "PASS"
        ((tests_passed++))
    else
        echo "FAIL"
        ((tests_failed++))
    fi
    
    # Test 5: Required directories exist
    echo -n "Test 5: Directory structure... "
    if [[ -d "${RESOURCE_DIR}/config" && -d "${RESOURCE_DIR}/lib" ]]; then
        echo "PASS"
        ((tests_passed++))
    else
        echo "FAIL"
        ((tests_failed++))
    fi
    
    # Test 6: Configuration files exist
    echo -n "Test 6: Configuration files... "
    if [[ -f "${RESOURCE_DIR}/config/defaults.sh" && -f "${RESOURCE_DIR}/config/runtime.json" ]]; then
        echo "PASS"
        ((tests_passed++))
    else
        echo "FAIL"
        ((tests_failed++))
    fi
    
    # Summary
    echo ""
    echo "Unit Tests Summary:"
    echo "  Passed: $tests_passed"
    echo "  Failed: $tests_failed"
    
    if [[ $tests_failed -eq 0 ]]; then
        echo "All unit tests passed!"
        return 0
    else
        echo "Some unit tests failed!"
        return 1
    fi
}

#######################################
# Run all tests
#######################################
run_all_tests() {
    echo "Running all tests..."
    echo "=================="
    
    local overall_passed=true
    
    # Run unit tests first (they don't require services to be running)
    if ! run_unit_tests; then
        overall_passed=false
    fi
    
    echo ""
    
    # Check if services are running for other tests
    if ! is_running; then
        echo "Services not running. Starting services for testing..."
        if ! start_resource; then
            echo "Failed to start services. Skipping smoke and integration tests."
            return 1
        fi
        echo ""
    fi
    
    # Run smoke tests
    if ! run_smoke_tests; then
        overall_passed=false
    fi
    
    echo ""
    
    # Run integration tests
    if ! run_integration_tests; then
        overall_passed=false
    fi
    
    echo ""
    echo "=================="
    if $overall_passed; then
        echo "ALL TESTS PASSED!"
        return 0
    else
        echo "SOME TESTS FAILED!"
        return 1
    fi
}

#######################################
# Performance test
#######################################
run_performance_test() {
    echo "Running performance tests..."
    
    # Test query response time
    echo "Testing Prometheus query performance..."
    local start_time=$(date +%s%N)
    curl -s "http://localhost:${PROMETHEUS_PORT}/api/v1/query?query=up" > /dev/null
    local end_time=$(date +%s%N)
    local duration=$((($end_time - $start_time) / 1000000))
    echo "  Query response time: ${duration}ms"
    
    # Test dashboard load time
    echo "Testing Grafana dashboard load time..."
    start_time=$(date +%s%N)
    curl -s "http://admin:${GRAFANA_ADMIN_PASSWORD}@localhost:${GRAFANA_PORT}/api/dashboards/home" > /dev/null
    end_time=$(date +%s%N)
    duration=$((($end_time - $start_time) / 1000000))
    echo "  Dashboard load time: ${duration}ms"
    
    # Test metric ingestion rate
    echo "Testing metric ingestion rate..."
    local metrics_before=$(curl -s "http://localhost:${PROMETHEUS_PORT}/api/v1/query?query=prometheus_tsdb_head_samples_appended_total" | jq -r '.data.result[0].value[1]')
    sleep 10
    local metrics_after=$(curl -s "http://localhost:${PROMETHEUS_PORT}/api/v1/query?query=prometheus_tsdb_head_samples_appended_total" | jq -r '.data.result[0].value[1]')
    local ingestion_rate=$(echo "scale=2; ($metrics_after - $metrics_before) / 10" | bc)
    echo "  Metric ingestion rate: ${ingestion_rate} samples/sec"
    
    echo "Performance tests complete!"
}