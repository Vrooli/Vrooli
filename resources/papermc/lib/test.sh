#!/usr/bin/env bash
# PaperMC test implementations

set -euo pipefail

# Get the directory of this script if not already set
if [[ -z "${SCRIPT_DIR:-}" ]]; then
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
fi

# Source core functionality if not already sourced
if ! declare -f start_health_service &>/dev/null; then
    source "${SCRIPT_DIR}/core.sh"
fi

# Run tests based on type
run_tests() {
    local test_type="${1:-all}"
    
    case "${test_type}" in
        smoke)
            run_smoke_tests
            ;;
        integration)
            run_integration_tests
            ;;
        unit)
            run_unit_tests
            ;;
        all)
            run_all_tests
            ;;
        *)
            echo "Error: Unknown test type: ${test_type}" >&2
            echo "Valid types: smoke, integration, unit, all" >&2
            return 1
            ;;
    esac
}

# Run smoke tests
run_smoke_tests() {
    echo "Running smoke tests..."
    
    # Clean up any leftover processes on the health port first
    lsof -t -i:${PAPERMC_HEALTH_PORT} 2>/dev/null | xargs -r kill -9 2>/dev/null || true
    sleep 1
    
    # Start health service temporarily for testing
    start_health_service || {
        echo "✗ Failed to start health service"
        return 1
    }
    sleep 2
    
    # Check health endpoint (just check if port responds)
    echo -n "Testing health endpoint... "
    if timeout 5 curl -sf "http://localhost:${PAPERMC_HEALTH_PORT}/" > /dev/null 2>&1; then
        echo "✓ Passed"
    else
        echo "✗ Failed"
        stop_health_service 2>/dev/null || true
        return 1
    fi
    
    # Stop health service after test
    stop_health_service 2>/dev/null || true
    
    # Check if Docker/Java is available
    echo -n "Testing runtime availability... "
    if [[ "${PAPERMC_SERVER_TYPE}" == "docker" ]]; then
        if command -v docker &> /dev/null; then
            echo "✓ Docker available"
        else
            echo "✗ Docker not found"
            return 1
        fi
    else
        if command -v java &> /dev/null; then
            echo "✓ Java available"
        else
            echo "✗ Java not found"
            return 1
        fi
    fi
    
    # Check data directory
    echo -n "Testing data directory... "
    if [[ -d "${PAPERMC_DATA_DIR}" ]] || mkdir -p "${PAPERMC_DATA_DIR}" 2>/dev/null; then
        echo "✓ Accessible"
    else
        echo "✗ Cannot create/access"
        return 1
    fi
    
    echo "Smoke tests passed!"
    return 0
}

# Run integration tests
run_integration_tests() {
    echo "Running integration tests..."
    
    # Test server lifecycle
    echo "Testing server lifecycle..."
    
    # Test installation
    echo -n "  Installing server... "
    if install_papermc > /dev/null 2>&1; then
        echo "✓"
    else
        echo "✗"
        return 1
    fi
    
    # Test start
    echo -n "  Starting server... "
    if start_papermc > /dev/null 2>&1; then
        echo "✓"
        sleep 10  # Give server time to initialize
    else
        echo "✗"
        return 1
    fi
    
    # Test status check
    echo -n "  Checking status... "
    if is_server_running; then
        echo "✓"
    else
        echo "✗"
        return 1
    fi
    
    # Test RCON connectivity (if server is ready)
    echo -n "  Testing RCON... "
    if is_server_ready; then
        echo "✓ Ready"
        
        # Try to execute a command if mcrcon is available
        if command -v mcrcon &> /dev/null; then
            echo -n "  Executing test command... "
            if execute_rcon_command "say Integration test" > /dev/null 2>&1; then
                echo "✓"
            else
                echo "✗ (non-critical)"
            fi
        fi
    else
        echo "⚠ Not ready yet (server still starting)"
    fi
    
    # Test backup
    echo -n "  Testing backup... "
    if backup_world > /dev/null 2>&1; then
        echo "✓"
    else
        echo "✗"
        return 1
    fi
    
    # Test stop
    echo -n "  Stopping server... "
    if stop_papermc > /dev/null 2>&1; then
        echo "✓"
    else
        echo "✗"
        return 1
    fi
    
    echo "Integration tests passed!"
    return 0
}

# Run unit tests
run_unit_tests() {
    echo "Running unit tests..."
    
    # Test configuration functions
    echo -n "Testing configuration creation... "
    local temp_dir=$(mktemp -d)
    PAPERMC_DATA_DIR="${temp_dir}" create_server_properties
    if [[ -f "${temp_dir}/server.properties" ]]; then
        echo "✓"
    else
        echo "✗"
        rm -rf "${temp_dir}"
        return 1
    fi
    rm -rf "${temp_dir}"
    
    # Test docker-compose generation
    echo -n "Testing Docker compose generation... "
    temp_dir=$(mktemp -d)
    PAPERMC_DATA_DIR="${temp_dir}" create_docker_compose
    if [[ -f "${temp_dir}/docker-compose.yml" ]]; then
        echo "✓"
    else
        echo "✗"
        rm -rf "${temp_dir}"
        return 1
    fi
    rm -rf "${temp_dir}"
    
    # Test server readiness check
    echo -n "Testing readiness check... "
    # When server is not running, is_server_ready should return false
    set +e  # Temporarily disable exit on error
    is_server_ready 2>/dev/null
    local result=$?
    set -e  # Re-enable exit on error
    if [[ ${result} -ne 0 ]]; then
        echo "✓ (correctly returns false when not running)"
    else
        echo "✗"
        return 1
    fi
    
    echo "Unit tests passed!"
    return 0
}

# Run all tests
run_all_tests() {
    echo "Running all tests..."
    echo "===================="
    
    local failed=0
    
    # Run each test suite
    run_smoke_tests || ((failed++))
    echo ""
    run_unit_tests || ((failed++))
    echo ""
    run_integration_tests || ((failed++))
    
    echo ""
    echo "===================="
    if [[ ${failed} -eq 0 ]]; then
        echo "All tests passed!"
        return 0
    else
        echo "${failed} test suite(s) failed"
        return 1
    fi
}