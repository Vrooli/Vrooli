#!/usr/bin/env bash
# OpenMRS Test Implementation

set -euo pipefail

# Source core functions
source "${SCRIPT_DIR}/lib/core.sh"

# Main test runner
openmrs::test() {
    local phase="${1:-all}"
    shift || true
    
    case "$phase" in
        smoke)
            openmrs::test::smoke "$@"
            ;;
        integration)
            openmrs::test::integration "$@"
            ;;
        unit)
            openmrs::test::unit "$@"
            ;;
        all)
            openmrs::test::smoke "$@" && \
            openmrs::test::integration "$@" && \
            openmrs::test::unit "$@"
            ;;
        *)
            echo "Unknown test phase: $phase" >&2
            echo "Available phases: smoke, integration, unit, all" >&2
            exit 1
            ;;
    esac
}

# Smoke test - quick health validation
openmrs::test::smoke() {
    log::info "Running OpenMRS smoke tests..."
    
    local failed=0
    
    # Test 1: Check if containers are running
    echo -n "Testing container status... "
    if docker ps --format '{{.Names}}' | grep -q "^${OPENMRS_APP_CONTAINER}$"; then
        echo "✓"
    else
        echo "✗ (container not running)"
        ((failed++))
    fi
    
    # Test 2: Check web interface
    echo -n "Testing web interface... "
    if timeout 5 curl -sf "http://localhost:${OPENMRS_PORT}/openmrs" &>/dev/null; then
        echo "✓"
    else
        echo "✗ (not responding)"
        ((failed++))
    fi
    
    # Test 3: Check REST API
    echo -n "Testing REST API... "
    if timeout 5 curl -sf "http://localhost:${OPENMRS_API_PORT}/openmrs/ws/rest/v1/session" &>/dev/null; then
        echo "✓"
    else
        echo "✗ (API not responding)"
        ((failed++))
    fi
    
    # Test 4: Check database connectivity
    echo -n "Testing database connectivity... "
    if docker exec "${OPENMRS_DB_CONTAINER}" pg_isready -U "${OPENMRS_DB_USER}" &>/dev/null; then
        echo "✓"
    else
        echo "✗ (database not ready)"
        ((failed++))
    fi
    
    if [[ $failed -eq 0 ]]; then
        log::success "All smoke tests passed!"
        return 0
    else
        log::error "$failed smoke tests failed"
        return 1
    fi
}

# Integration test - end-to-end functionality
openmrs::test::integration() {
    log::info "Running OpenMRS integration tests..."
    
    local failed=0
    
    # Test 1: Authenticate and get session
    echo -n "Testing authentication... "
    local session_response=$(curl -s -u "${OPENMRS_ADMIN_USER}:${OPENMRS_ADMIN_PASS}" \
        "http://localhost:${OPENMRS_API_PORT}/openmrs/ws/rest/v1/session" 2>/dev/null)
    
    if echo "$session_response" | jq -e '.sessionId' &>/dev/null; then
        echo "✓"
    else
        echo "✗ (authentication failed)"
        ((failed++))
    fi
    
    # Test 2: List patients
    echo -n "Testing patient listing... "
    local patients_response=$(curl -s -u "${OPENMRS_ADMIN_USER}:${OPENMRS_ADMIN_PASS}" \
        "http://localhost:${OPENMRS_API_PORT}/openmrs/ws/rest/v1/patient" 2>/dev/null)
    
    if echo "$patients_response" | jq -e '.results' &>/dev/null; then
        echo "✓"
    else
        echo "✗ (failed to list patients)"
        ((failed++))
    fi
    
    # Test 3: Check for demo data
    echo -n "Testing demo data presence... "
    local patient_count=$(echo "$patients_response" | jq '.results | length' 2>/dev/null || echo "0")
    
    if [[ "$patient_count" -gt 0 ]]; then
        echo "✓ (found $patient_count patients)"
    else
        echo "✗ (no demo patients found)"
        ((failed++))
    fi
    
    if [[ $failed -eq 0 ]]; then
        log::success "All integration tests passed!"
        return 0
    else
        log::error "$failed integration tests failed"
        return 1
    fi
}

# Unit test - test library functions
openmrs::test::unit() {
    log::info "Running OpenMRS unit tests..."
    
    local failed=0
    
    # Test 1: Port configuration
    echo -n "Testing port configuration... "
    if [[ -n "${OPENMRS_PORT}" ]] && [[ -n "${OPENMRS_API_PORT}" ]] && [[ -n "${OPENMRS_FHIR_PORT}" ]]; then
        echo "✓"
    else
        echo "✗ (ports not configured)"
        ((failed++))
    fi
    
    # Test 2: Directory creation
    echo -n "Testing directory structure... "
    if [[ -d "${OPENMRS_DATA_DIR}" ]] || mkdir -p "${OPENMRS_DATA_DIR}" 2>/dev/null; then
        echo "✓"
    else
        echo "✗ (cannot create directories)"
        ((failed++))
    fi
    
    # Test 3: Docker availability
    echo -n "Testing Docker availability... "
    if docker version &>/dev/null; then
        echo "✓"
    else
        echo "✗ (Docker not available)"
        ((failed++))
    fi
    
    if [[ $failed -eq 0 ]]; then
        log::success "All unit tests passed!"
        return 0
    else
        log::error "$failed unit tests failed"
        return 1
    fi
}