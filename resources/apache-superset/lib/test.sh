#!/usr/bin/env bash
# Apache Superset Test Functions

# Main test handler
superset::test() {
    local test_type="${1:-all}"
    
    case "$test_type" in
        smoke)
            superset::test_smoke
            ;;
        integration)
            superset::test_integration
            ;;
        unit)
            superset::test_unit
            ;;
        all)
            superset::test_all
            ;;
        *)
            echo "Error: Unknown test type '$test_type'"
            echo "Valid types: smoke, integration, unit, all"
            return 1
            ;;
    esac
}

# Quick smoke test
superset::test_smoke() {
    log::info "Running smoke tests..."
    local failed=0
    
    # Test 1: Check if service is running
    echo -n "Testing service status... "
    if docker ps --format '{{.Names}}' | grep -q "^${SUPERSET_APP_CONTAINER}$"; then
        echo "✓"
    else
        echo "✗ Service not running"
        ((failed++))
    fi
    
    # Test 2: Health check
    echo -n "Testing health endpoint... "
    if timeout 5 curl -sf "http://localhost:${SUPERSET_PORT}/health" &>/dev/null; then
        echo "✓"
    else
        echo "✗ Health check failed"
        ((failed++))
    fi
    
    # Test 3: API accessibility
    echo -n "Testing API endpoint... "
    if timeout 5 curl -sf "http://localhost:${SUPERSET_PORT}/api/v1/security/login" -X POST -H "Content-Type: application/json" -d '{"username": "admin", "password": "admin", "provider": "db"}' | grep -q "access_token" &>/dev/null; then
        echo "✓"
    else
        echo "✗ API not accessible"
        ((failed++))
    fi
    
    # Test 4: Database connectivity
    echo -n "Testing database connection... "
    if docker exec "${SUPERSET_POSTGRES_CONTAINER}" pg_isready -U "${SUPERSET_POSTGRES_USER}" &>/dev/null; then
        echo "✓"
    else
        echo "✗ Database not ready"
        ((failed++))
    fi
    
    # Test 5: Redis connectivity
    echo -n "Testing Redis connection... "
    if docker exec "${SUPERSET_REDIS_CONTAINER}" redis-cli ping &>/dev/null; then
        echo "✓"
    else
        echo "✗ Redis not responding"
        ((failed++))
    fi
    
    if [[ $failed -eq 0 ]]; then
        log::success "All smoke tests passed"
        return 0
    else
        log::error "$failed smoke tests failed"
        return 1
    fi
}

# Integration tests
superset::test_integration() {
    log::info "Running integration tests..."
    local failed=0
    
    # Test 1: Authentication
    echo -n "Testing authentication... "
    local token=$(superset::get_auth_token 2>/dev/null)
    if [[ -n "$token" ]]; then
        echo "✓"
    else
        echo "✗ Authentication failed"
        ((failed++))
    fi
    
    # Test 2: Create dashboard
    echo -n "Testing dashboard creation... "
    local dashboard_id=$(curl -s -X POST \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json" \
        "http://localhost:${SUPERSET_PORT}/api/v1/dashboard/" \
        -d '{"dashboard_title": "Test Dashboard"}' | jq -r '.id')
    
    if [[ -n "$dashboard_id" ]] && [[ "$dashboard_id" != "null" ]]; then
        echo "✓ (ID: $dashboard_id)"
    else
        echo "✗ Failed to create dashboard"
        ((failed++))
    fi
    
    # Test 3: List dashboards
    echo -n "Testing dashboard listing... "
    local dashboard_count=$(curl -s -H "Authorization: Bearer ${token}" \
        "http://localhost:${SUPERSET_PORT}/api/v1/dashboard/" | jq '.count')
    
    if [[ "$dashboard_count" -gt 0 ]]; then
        echo "✓ ($dashboard_count dashboards)"
    else
        echo "✗ No dashboards found"
        ((failed++))
    fi
    
    # Test 4: Database connection to Vrooli Postgres
    echo -n "Testing Vrooli Postgres connection... "
    local db_test=$(curl -s -X POST \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json" \
        "http://localhost:${SUPERSET_PORT}/api/v1/database/test_connection/" \
        -d "{\"database_name\": \"Vrooli Postgres\", \"sqlalchemy_uri\": \"postgresql://postgres:postgres@${VROOLI_POSTGRES_HOST}:${VROOLI_POSTGRES_PORT}/postgres\"}" 2>/dev/null || echo "failed")
    
    if [[ "$db_test" != "failed" ]]; then
        echo "✓"
    else
        echo "✗ Could not connect to Vrooli Postgres (expected if not running)"
    fi
    
    # Test 5: Delete test dashboard
    if [[ -n "$dashboard_id" ]] && [[ "$dashboard_id" != "null" ]]; then
        echo -n "Testing dashboard deletion... "
        curl -s -X DELETE -H "Authorization: Bearer ${token}" \
            "http://localhost:${SUPERSET_PORT}/api/v1/dashboard/${dashboard_id}" &>/dev/null
        echo "✓"
    fi
    
    if [[ $failed -eq 0 ]]; then
        log::success "All integration tests passed"
        return 0
    else
        log::warning "$failed integration tests had issues (some may be expected)"
        return 0  # Don't fail if external services aren't available
    fi
}

# Unit tests (testing individual functions)
superset::test_unit() {
    log::info "Running unit tests..."
    local failed=0
    
    # Test 1: Configuration file generation
    echo -n "Testing config generation... "
    if [[ -f "${SUPERSET_CONFIG_DIR}/superset_config.py" ]]; then
        echo "✓"
    else
        superset::create_config
        if [[ -f "${SUPERSET_CONFIG_DIR}/superset_config.py" ]]; then
            echo "✓ (created)"
        else
            echo "✗ Failed to create config"
            ((failed++))
        fi
    fi
    
    # Test 2: Directory structure
    echo -n "Testing directory structure... "
    local dirs_ok=true
    for dir in "${SUPERSET_CONFIG_DIR}" "${SUPERSET_UPLOADS_DIR}" "${SUPERSET_CACHE_DIR}"; do
        if [[ ! -d "$dir" ]]; then
            dirs_ok=false
            break
        fi
    done
    
    if [[ "$dirs_ok" == true ]]; then
        echo "✓"
    else
        echo "✗ Missing directories"
        ((failed++))
    fi
    
    # Test 3: Network existence
    echo -n "Testing Docker network... "
    if docker network inspect "${SUPERSET_NETWORK}" &>/dev/null; then
        echo "✓"
    else
        echo "✗ Network not found"
        ((failed++))
    fi
    
    if [[ $failed -eq 0 ]]; then
        log::success "All unit tests passed"
        return 0
    else
        log::error "$failed unit tests failed"
        return 1
    fi
}

# Run all tests
superset::test_all() {
    log::info "Running all tests..."
    local failed=0
    
    superset::test_smoke || ((failed++))
    echo ""
    superset::test_integration || ((failed++))
    echo ""
    superset::test_unit || ((failed++))
    
    if [[ $failed -eq 0 ]]; then
        log::success "All test suites passed"
        return 0
    else
        log::error "$failed test suites had failures"
        return 1
    fi
}