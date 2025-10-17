#!/usr/bin/env bash
# Apache Airflow Resource - Test Implementation

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "${SCRIPT_DIR}")"

# Source configuration
source "${RESOURCE_DIR}/config/defaults.sh"

# Run smoke tests (quick health validation)
smoke() {
    echo "Running Apache Airflow smoke tests..."
    local failures=0
    
    # Test 1: Check if service is running
    echo -n "  [1/4] Checking if Airflow webserver is running... "
    if timeout 5 curl -sf "http://localhost:${AIRFLOW_WEBSERVER_PORT}/health" &>/dev/null; then
        echo "✓"
    else
        echo "✗"
        ((failures++))
    fi
    
    # Test 2: Check scheduler
    echo -n "  [2/4] Checking if scheduler is running... "
    if docker ps --format "table {{.Names}}" 2>/dev/null | grep -q "airflow-scheduler"; then
        echo "✓"
    else
        echo "✗"
        ((failures++))
    fi
    
    # Test 3: Check Redis connectivity
    echo -n "  [3/4] Checking Redis connectivity... "
    if docker ps --format "table {{.Names}}" 2>/dev/null | grep -q "airflow-redis"; then
        echo "✓"
    else
        echo "✗"
        ((failures++))
    fi
    
    # Test 4: Check PostgreSQL connectivity
    echo -n "  [4/4] Checking PostgreSQL connectivity... "
    if docker ps --format "table {{.Names}}" 2>/dev/null | grep -q "airflow-postgres"; then
        echo "✓"
    else
        echo "✗"
        ((failures++))
    fi
    
    if [[ "$failures" -eq 0 ]]; then
        echo "All smoke tests passed!"
        return 0
    else
        echo "$failures smoke test(s) failed"
        return 1
    fi
}

# Run integration tests (full functionality)
integration() {
    echo "Running Apache Airflow integration tests..."
    local failures=0
    
    # Test 1: API health endpoint
    echo -n "  [1/5] Testing API health endpoint... "
    local health_response
    if health_response=$(timeout 5 curl -sf "http://localhost:${AIRFLOW_WEBSERVER_PORT}/health" 2>/dev/null); then
        echo "✓"
    else
        echo "✗"
        ((failures++))
    fi
    
    # Test 2: DAG parsing
    echo -n "  [2/5] Testing DAG parsing... "
    if docker exec airflow-scheduler airflow dags list &>/dev/null; then
        echo "✓"
    else
        echo "✗"
        ((failures++))
    fi
    
    # Test 3: Worker availability
    echo -n "  [3/5] Testing worker availability... "
    local worker_count
    worker_count=$(docker ps --format "table {{.Names}}" 2>/dev/null | grep -c "airflow-worker" || true)
    if [[ "$worker_count" -gt 0 ]]; then
        echo "✓ ($worker_count workers)"
    else
        echo "✗"
        ((failures++))
    fi
    
    # Test 4: Database connectivity
    echo -n "  [4/5] Testing database connectivity... "
    if docker exec airflow-postgres pg_isready -U "${AIRFLOW_POSTGRES_USER}" &>/dev/null; then
        echo "✓"
    else
        echo "✗"
        ((failures++))
    fi
    
    # Test 5: Example DAG execution (if enabled)
    if [[ "${AIRFLOW_ENABLE_EXAMPLE_DAGS}" == "true" ]]; then
        echo -n "  [5/5] Testing example DAG execution... "
        if docker exec airflow-scheduler airflow dags test example_etl_pipeline 2025-01-01 &>/dev/null; then
            echo "✓"
        else
            echo "✗"
            ((failures++))
        fi
    else
        echo "  [5/5] Example DAGs disabled, skipping execution test"
    fi
    
    if [[ "$failures" -eq 0 ]]; then
        echo "All integration tests passed!"
        return 0
    else
        echo "$failures integration test(s) failed"
        return 1
    fi
}

# Run unit tests (library functions)
unit() {
    echo "Running Apache Airflow unit tests..."
    local failures=0
    
    # Test 1: Configuration loading
    echo -n "  [1/3] Testing configuration loading... "
    if [[ -f "${RESOURCE_DIR}/config/defaults.sh" ]] && source "${RESOURCE_DIR}/config/defaults.sh"; then
        if [[ -n "${AIRFLOW_WEBSERVER_PORT}" ]]; then
            echo "✓"
        else
            echo "✗ (configuration not loaded)"
            ((failures++))
        fi
    else
        echo "✗ (configuration file missing)"
        ((failures++))
    fi
    
    # Test 2: Runtime.json validation
    echo -n "  [2/3] Testing runtime.json validation... "
    if [[ -f "${RESOURCE_DIR}/config/runtime.json" ]]; then
        if jq -e . "${RESOURCE_DIR}/config/runtime.json" &>/dev/null; then
            echo "✓"
        else
            echo "✗ (invalid JSON)"
            ((failures++))
        fi
    else
        echo "✗ (file missing)"
        ((failures++))
    fi
    
    # Test 3: Docker compose file validation
    echo -n "  [3/3] Testing docker-compose.yml validation... "
    if [[ -f "${RESOURCE_DIR}/docker/docker-compose.yml" ]]; then
        if docker-compose -f "${RESOURCE_DIR}/docker/docker-compose.yml" config &>/dev/null; then
            echo "✓"
        else
            echo "✗ (invalid compose file)"
            ((failures++))
        fi
    else
        echo "✗ (file missing)"
        ((failures++))
    fi
    
    if [[ "$failures" -eq 0 ]]; then
        echo "All unit tests passed!"
        return 0
    else
        echo "$failures unit test(s) failed"
        return 1
    fi
}

# Run all tests
all() {
    echo "Running all Apache Airflow tests..."
    echo "================================"
    
    local overall_failures=0
    
    # Run smoke tests
    if ! smoke; then
        ((overall_failures++))
    fi
    echo ""
    
    # Run integration tests
    if ! integration; then
        ((overall_failures++))
    fi
    echo ""
    
    # Run unit tests
    if ! unit; then
        ((overall_failures++))
    fi
    
    echo "================================"
    if [[ "$overall_failures" -eq 0 ]]; then
        echo "All test suites passed!"
        return 0
    else
        echo "$overall_failures test suite(s) failed"
        return 1
    fi
}

# Export functions
export -f smoke
export -f integration
export -f unit
export -f all

# Handle direct script execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    command="${1:-all}"
    shift || true
    
    case "$command" in
        smoke|integration|unit|all)
            "$command" "$@"
            ;;
        *)
            echo "Usage: $0 {smoke|integration|unit|all}"
            exit 1
            ;;
    esac
fi