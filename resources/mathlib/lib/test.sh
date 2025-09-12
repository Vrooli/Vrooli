#!/usr/bin/env bash
# Mathlib Resource - Test Library

set -euo pipefail

# Source configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../config/defaults.sh"

# Run specified test suite
mathlib::test::run() {
    local test_type="${1:-all}"
    local test_dir="${SCRIPT_DIR}/../test"
    
    echo "Running Mathlib ${test_type} tests..."
    
    case "${test_type}" in
        smoke)
            "${test_dir}/phases/test-smoke.sh"
            ;;
        integration)
            "${test_dir}/phases/test-integration.sh"
            ;;
        unit)
            "${test_dir}/phases/test-unit.sh"
            ;;
        all)
            "${test_dir}/run-tests.sh"
            ;;
        *)
            echo "Error: Unknown test type '${test_type}'"
            return 1
            ;;
    esac
}

# Validate health endpoint
mathlib::test::health() {
    echo "Testing health endpoint..."
    
    if ! timeout 5 curl -sf "${MATHLIB_HEALTH_ENDPOINT}" > /dev/null; then
        echo "FAIL: Health check failed"
        return 1
    fi
    
    # Validate response format
    local response=$(timeout 5 curl -sf "${MATHLIB_HEALTH_ENDPOINT}")
    if ! echo "${response}" | jq -e '.status' > /dev/null 2>&1; then
        echo "FAIL: Invalid health response format"
        return 1
    fi
    
    echo "PASS: Health endpoint working"
    return 0
}

# Validate installation
mathlib::test::installation() {
    echo "Testing installation..."
    
    # Check directories exist
    if [[ ! -d "${MATHLIB_INSTALL_DIR}" ]]; then
        echo "FAIL: Install directory missing"
        return 1
    fi
    
    if [[ ! -d "${MATHLIB_WORK_DIR}" ]]; then
        echo "FAIL: Work directory missing"
        return 1
    fi
    
    echo "PASS: Installation validated"
    return 0
}

# Validate configuration
mathlib::test::config() {
    echo "Testing configuration..."
    
    # Check runtime.json exists
    if [[ ! -f "${SCRIPT_DIR}/../config/runtime.json" ]]; then
        echo "FAIL: runtime.json missing"
        return 1
    fi
    
    # Validate JSON format
    if ! jq -e . "${SCRIPT_DIR}/../config/runtime.json" > /dev/null 2>&1; then
        echo "FAIL: Invalid runtime.json format"
        return 1
    fi
    
    echo "PASS: Configuration valid"
    return 0
}

# Test lifecycle operations
mathlib::test::lifecycle() {
    echo "Testing lifecycle operations..."
    
    # Test stop (ensure clean state)
    mathlib::stop > /dev/null 2>&1 || true
    
    # Test start
    if ! mathlib::start --wait; then
        echo "FAIL: Service failed to start"
        return 1
    fi
    
    # Test restart
    if ! mathlib::restart --wait; then
        echo "FAIL: Service failed to restart"
        return 1
    fi
    
    # Test stop
    if ! mathlib::stop; then
        echo "FAIL: Service failed to stop"
        return 1
    fi
    
    echo "PASS: Lifecycle operations working"
    return 0
}

# Test CLI commands
mathlib::test::cli() {
    echo "Testing CLI commands..."
    
    # Test help
    if ! "${SCRIPT_DIR}/../cli.sh" help > /dev/null; then
        echo "FAIL: Help command failed"
        return 1
    fi
    
    # Test info
    if ! "${SCRIPT_DIR}/../cli.sh" info > /dev/null; then
        echo "FAIL: Info command failed"
        return 1
    fi
    
    # Test status
    if ! "${SCRIPT_DIR}/../cli.sh" status > /dev/null; then
        echo "FAIL: Status command failed"
        return 1
    fi
    
    echo "PASS: CLI commands working"
    return 0
}