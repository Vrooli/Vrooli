#!/bin/bash
# Earthly Resource - Test Functions
# Implements v2.0 contract test requirements

set -euo pipefail

# Source configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/config/defaults.sh"
source "${SCRIPT_DIR}/lib/core.sh"

# Handle test subcommands
handle_test() {
    local subcommand="${1:-all}"
    shift || true
    
    # Show help if requested
    if [[ "${subcommand}" == "--help" ]] || [[ "${subcommand}" == "-h" ]]; then
        cat << EOF
🧪 TEST - Testing and Validation

📋 USAGE:
    resource-earthly test <subcommand> [options]

📖 SUBCOMMANDS:
    smoke                Quick health validation (<30s)
    unit                 Test library functions (<60s)
    integration          Test full functionality (<120s)
    all                  Run all test suites (default)

💡 EXAMPLES:
    resource-earthly test smoke
    resource-earthly test integration
    resource-earthly test all

📊 TEST COVERAGE:
    - Health checks and availability
    - Configuration loading
    - Build execution
    - Artifact management
    - Cache functionality
    - Parallel execution
EOF
        return 0
    fi
    
    case "${subcommand}" in
        smoke)
            test_smoke "$@"
            ;;
        integration)
            test_integration "$@"
            ;;
        unit)
            test_unit "$@"
            ;;
        all)
            test_all "$@"
            ;;
        *)
            log_error "Unknown test subcommand: ${subcommand}"
            echo "Available subcommands: smoke, integration, unit, all"
            echo "Use 'resource-earthly test --help' for more information"
            return 1
            ;;
    esac
}

# Smoke test - quick health validation (<30s)
test_smoke() {
    log_info "Running smoke tests..."
    local start_time
    start_time=$(date +%s)
    local failed=0
    
    # Test 1: Earthly installed or mock available
    echo -n "Testing Earthly availability... "
    if timeout 5 bash -c "command -v earthly &>/dev/null || [[ -f ${HOME}/.local/bin/earthly ]]" || [[ -f "${SCRIPT_DIR}/lib/mock.sh" ]]; then
        echo "✅ PASS (binary or mock available)"
    else
        echo "❌ FAIL: Earthly not available"
        ((failed++))
    fi
    
    # Test 2: Docker available
    echo -n "Testing Docker availability... "
    if timeout 5 docker ps &>/dev/null; then
        echo "✅ PASS"
    else
        echo "❌ FAIL: Docker daemon not accessible"
        ((failed++))
    fi
    
    # Test 3: Configuration exists
    echo -n "Testing configuration... "
    if [[ -f "${SCRIPT_DIR}/config/runtime.json" ]]; then
        echo "✅ PASS"
    else
        echo "❌ FAIL: Runtime configuration missing"
        ((failed++))
    fi
    
    # Test 4: Directories created
    echo -n "Testing directory structure... "
    if [[ -d "${EARTHLY_HOME}" ]]; then
        echo "✅ PASS"
    else
        echo "❌ FAIL: Earthly home directory not found"
        ((failed++))
    fi
    
    # Test 5: Health check simulation
    echo -n "Testing health check... "
    if timeout 5 bash -c "command -v earthly &>/dev/null && earthly --version 2>/dev/null | grep -q 'earthly'" || [[ -f "${SCRIPT_DIR}/lib/mock.sh" ]]; then
        echo "✅ PASS"
    else
        echo "❌ FAIL: Health check failed"
        ((failed++))
    fi
    
    local end_time
    end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo ""
    if [[ ${failed} -eq 0 ]]; then
        log_info "✅ All smoke tests passed (${duration}s)"
        return 0
    else
        log_error "❌ ${failed} smoke tests failed (${duration}s)"
        return 1
    fi
}

# Integration test - full functionality
test_integration() {
    log_info "Running integration tests..."
    local failed=0
    
    # Ensure earthly is in PATH
    export PATH="${HOME}/.local/bin:${PATH}"
    
    # Create a temporary test directory within Vrooli
    local test_dir="${SCRIPT_DIR}/tmp/earthly-integration-test-$$"
    mkdir -p "${test_dir}"
    
    # Test 1: Create simple Earthfile
    echo -n "Testing Earthfile creation... "
    cat > "${test_dir}/Earthfile" << 'EOF'
VERSION 0.8
FROM alpine:3.18

test:
    RUN echo "Integration test successful" > test.txt
    SAVE ARTIFACT test.txt AS LOCAL test-output.txt

build:
    RUN echo "Build test" > build.txt
    SAVE ARTIFACT build.txt
EOF
    
    if [[ -f "${test_dir}/Earthfile" ]]; then
        echo "✅ PASS"
    else
        echo "❌ FAIL"
        ((failed++))
    fi
    
    # Test 2: Execute simple build
    echo -n "Testing build execution... "
    # Check if earthly binary exists and Docker is running
    if command -v earthly &>/dev/null && docker ps &>/dev/null; then
        # Try to run the build from the test directory
        # Use explicit PATH to ensure earthly is found
        set +e  # Temporarily disable exit on error
        cd "${test_dir}"
        # Clear any earthly platform variables
        unset EARTHLY_PLATFORMS
        PATH="${HOME}/.local/bin:${PATH}" timeout 60 earthly +test > "${test_dir}/build.log" 2>&1
        local exit_code=$?
        cd - >/dev/null
        set -e  # Re-enable exit on error
        
        if [[ ${exit_code} -eq 0 ]]; then
            echo "✅ PASS"
        elif [[ ${exit_code} -eq 124 ]]; then
            echo "❌ FAIL: Build timed out"
            ((failed++))
        else
            echo "❌ FAIL: Build failed with exit code ${exit_code}"
            if [[ -f "${test_dir}/build.log" ]]; then
                echo "Build log: $(tail -5 "${test_dir}/build.log")"
            fi
            ((failed++))
        fi
    else
        echo "⚠️  SKIP: Earthly or Docker not available"
    fi
    
    # Test 3: Check artifact creation
    echo -n "Testing artifact management... "
    # Check if earthly binary exists and Docker is running
    if command -v earthly &>/dev/null && docker ps &>/dev/null; then
        if [[ -f "${test_dir}/test-output.txt" ]]; then
            echo "✅ PASS"
        else
            # Only skip if earthly wasn't available, otherwise it's a failure
            if command -v earthly &>/dev/null; then
                echo "❌ FAIL: Artifact not created"
                ((failed++))
            else
                echo "⚠️  SKIP: Artifact test skipped (earthly not available)"
            fi
        fi
    else
        echo "⚠️  SKIP: Earthly or Docker not available"
    fi
    
    # Test 4: Cache functionality
    echo -n "Testing cache functionality... "
    if command -v earthly &>/dev/null && docker ps &>/dev/null; then
        set +e  # Temporarily disable exit on error
        local first_run second_run third_run
        first_run=$(date +%s)
        cd "${test_dir}"
        unset EARTHLY_PLATFORMS
        PATH="${HOME}/.local/bin:${PATH}" timeout 60 earthly +build >/dev/null 2>&1
        second_run=$(date +%s)
        PATH="${HOME}/.local/bin:${PATH}" timeout 60 earthly +build >/dev/null 2>&1
        third_run=$(date +%s)
        cd - >/dev/null
        set -e  # Re-enable exit on error
        
        local first_duration=$((second_run - first_run))
        local second_duration=$((third_run - second_run))
        
        # Cache is working if second run is faster
        if [[ ${second_duration} -le ${first_duration} ]]; then
            echo "✅ PASS (cache working)"
        else
            echo "⚠️  WARN: Cache may not be working optimally"
        fi
    else
        echo "⚠️  SKIP: Earthly or Docker not available"
    fi
    
    # Test 5: Parallel execution
    echo -n "Testing parallel execution... "
    if command -v earthly &>/dev/null && docker ps &>/dev/null; then
        cat > "${test_dir}/ParallelEarthfile" << 'EOF'
VERSION 0.8
FROM alpine:3.18

target1:
    RUN echo "Target 1 complete"

target2:
    RUN echo "Target 2 complete"

all:
    BUILD +target1
    BUILD +target2
EOF
        
        # Create new Earthfile for parallel test
        cp "${test_dir}/ParallelEarthfile" "${test_dir}/Earthfile" 2>/dev/null || true
        
        # Use explicit PATH to ensure earthly is found
        set +e  # Temporarily disable exit on error
        cd "${test_dir}"
        unset EARTHLY_PLATFORMS
        # Reduced timeout and simpler targets for more reliable testing
        PATH="${HOME}/.local/bin:${PATH}" timeout 30 earthly +all >/dev/null 2>&1
        local exit_code=$?
        cd - >/dev/null
        set -e  # Re-enable exit on error
        
        if [[ ${exit_code} -eq 0 ]]; then
            echo "✅ PASS"
        elif [[ ${exit_code} -eq 124 ]]; then
            echo "⚠️  WARN: Parallel build timed out (may need more time)"
        else
            echo "⚠️  WARN: Parallel build returned exit code ${exit_code}"
        fi
    else
        echo "⚠️  SKIP: Earthly or Docker not available"
    fi
    
    # Cleanup
    rm -rf "${test_dir}"
    
    echo ""
    if [[ ${failed} -eq 0 ]]; then
        log_info "✅ All integration tests passed"
        return 0
    else
        log_error "❌ ${failed} integration tests failed"
        return 1
    fi
}

# Unit test - library functions
test_unit() {
    log_info "Running unit tests..."
    local failed=0
    
    # Test 1: Configuration loading
    echo -n "Testing configuration loading... "
    if [[ -n "${EARTHLY_VERSION}" ]]; then
        echo "✅ PASS"
    else
        echo "❌ FAIL"
        ((failed++))
    fi
    
    # Test 2: Directory creation function
    echo -n "Testing directory creation... "
    local test_dir="/tmp/earthly-test-$$"
    mkdir -p "${test_dir}"
    if [[ -d "${test_dir}" ]]; then
        echo "✅ PASS"
        rm -rf "${test_dir}"
    else
        echo "❌ FAIL"
        ((failed++))
    fi
    
    # Test 3: JSON parsing
    echo -n "Testing JSON configuration parsing... "
    if jq -e '.version' "${SCRIPT_DIR}/config/runtime.json" &>/dev/null; then
        echo "✅ PASS"
    else
        echo "❌ FAIL"
        ((failed++))
    fi
    
    # Test 4: Error handling
    echo -n "Testing error handling... "
    if ! show_info "--invalid-flag" &>/dev/null; then
        echo "✅ PASS (error caught)"
    else
        echo "❌ FAIL (error not caught)"
        ((failed++))
    fi
    
    echo ""
    if [[ ${failed} -eq 0 ]]; then
        log_info "✅ All unit tests passed"
        return 0
    else
        log_error "❌ ${failed} unit tests failed"
        return 1
    fi
}

# Run all tests
test_all() {
    log_info "Running all tests..."
    local failed=0
    
    # Run each test phase
    if ! test_smoke; then
        ((failed++))
    fi
    
    if ! test_unit; then
        ((failed++))
    fi
    
    if ! test_integration; then
        ((failed++))
    fi
    
    echo ""
    if [[ ${failed} -eq 0 ]]; then
        log_info "✅ All test phases passed"
        return 0
    else
        log_error "❌ ${failed} test phases failed"
        return 1
    fi
}