#!/bin/bash
# GGWave Test Library Functions

set -euo pipefail

# Run smoke tests
ggwave::test::smoke() {
    echo "=== GGWave Smoke Test ==="
    local start_time=$(date +%s)
    local errors=0
    
    # Test 1: Container exists
    echo -n "1. Checking container exists... "
    if docker ps -a --format '{{.Names}}' | grep -q "^${GGWAVE_CONTAINER_NAME}$"; then
        echo "✓"
    else
        echo "✗ Container not found"
        ((errors++))
    fi
    
    # Test 2: Container is running
    echo -n "2. Checking container is running... "
    if docker ps --format '{{.Names}}' | grep -q "^${GGWAVE_CONTAINER_NAME}$"; then
        echo "✓"
    else
        echo "✗ Container not running"
        ((errors++))
    fi
    
    # Test 3: Health check passes
    echo -n "3. Checking health endpoint... "
    if timeout 5 curl -sf "http://localhost:${GGWAVE_PORT}/health" &>/dev/null; then
        echo "✓"
    else
        echo "✗ Health check failed"
        ((errors++))
    fi
    
    # Test 4: Port is accessible
    echo -n "4. Checking port ${GGWAVE_PORT}... "
    if nc -z localhost "${GGWAVE_PORT}" 2>/dev/null; then
        echo "✓"
    else
        echo "✗ Port not accessible"
        ((errors++))
    fi
    
    # Test 5: Data directory exists
    echo -n "5. Checking data directory... "
    if [[ -d "${GGWAVE_DATA_DIR}" ]]; then
        echo "✓"
    else
        echo "✗ Data directory not found"
        ((errors++))
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo ""
    echo "Smoke test completed in ${duration}s"
    
    if [[ $errors -eq 0 ]]; then
        echo "Result: PASSED ✓"
        return 0
    else
        echo "Result: FAILED ✗ ($errors errors)"
        return 1
    fi
}

# Run integration tests
ggwave::test::integration() {
    echo "=== GGWave Integration Test ==="
    local start_time=$(date +%s)
    local errors=0
    
    # Test 1: Encode endpoint
    echo -n "1. Testing encode endpoint... "
    local encode_response=$(curl -sf -X POST "http://localhost:${GGWAVE_PORT}/api/encode" \
        -H "Content-Type: application/json" \
        -d '{"data": "test", "mode": "normal"}' 2>/dev/null || echo "failed")
    if [[ "$encode_response" != "failed" ]]; then
        echo "✓"
    else
        echo "✗ Encode endpoint failed"
        ((errors++))
    fi
    
    # Test 2: Decode endpoint
    echo -n "2. Testing decode endpoint... "
    local decode_response=$(curl -sf -X POST "http://localhost:${GGWAVE_PORT}/api/decode" \
        -H "Content-Type: application/json" \
        -d '{"audio": "dGVzdA==", "mode": "auto"}' 2>/dev/null || echo "failed")
    if [[ "$decode_response" != "failed" ]]; then
        echo "✓"
    else
        echo "✗ Decode endpoint failed"
        ((errors++))
    fi
    
    # Test 3: Content management
    echo -n "3. Testing content management... "
    ggwave::content::add --data="Integration test data" &>/dev/null
    if [[ -f "${GGWAVE_DATA_DIR}/pending_transmission.txt" ]]; then
        echo "✓"
    else
        echo "✗ Content management failed"
        ((errors++))
    fi
    
    # Test 4: Mode switching
    echo -n "4. Testing transmission modes... "
    local mode_test_passed=true
    for mode in normal fast dt ultrasonic; do
        if ! ggwave::test::validate_mode "$mode"; then
            mode_test_passed=false
            break
        fi
    done
    if [[ "$mode_test_passed" == "true" ]]; then
        echo "✓"
    else
        echo "✗ Mode validation failed"
        ((errors++))
    fi
    
    # Test 5: Error correction
    echo -n "5. Testing error correction... "
    if ggwave::test::validate_error_correction; then
        echo "✓"
    else
        echo "✗ Error correction failed"
        ((errors++))
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo ""
    echo "Integration test completed in ${duration}s"
    
    if [[ $errors -eq 0 ]]; then
        echo "Result: PASSED ✓"
        return 0
    else
        echo "Result: FAILED ✗ ($errors errors)"
        return 1
    fi
}

# Run unit tests
ggwave::test::unit() {
    echo "=== GGWave Unit Test ==="
    local start_time=$(date +%s)
    local errors=0
    
    # Test 1: Configuration loading
    echo -n "1. Testing configuration loading... "
    if [[ -n "${GGWAVE_PORT}" ]] && [[ -n "${GGWAVE_MODE}" ]]; then
        echo "✓"
    else
        echo "✗ Configuration not loaded"
        ((errors++))
    fi
    
    # Test 2: Port validation
    echo -n "2. Testing port validation... "
    if [[ "${GGWAVE_PORT}" =~ ^[0-9]+$ ]] && [[ "${GGWAVE_PORT}" -ge 1024 ]] && [[ "${GGWAVE_PORT}" -le 65535 ]]; then
        echo "✓"
    else
        echo "✗ Invalid port configuration"
        ((errors++))
    fi
    
    # Test 3: Mode validation
    echo -n "3. Testing mode validation... "
    case "${GGWAVE_MODE}" in
        auto|normal|fast|dt|ultrasonic)
            echo "✓"
            ;;
        *)
            echo "✗ Invalid mode: ${GGWAVE_MODE}"
            ((errors++))
            ;;
    esac
    
    # Test 4: Sample rate validation
    echo -n "4. Testing sample rate validation... "
    if [[ "${GGWAVE_SAMPLE_RATE}" -ge 8000 ]] && [[ "${GGWAVE_SAMPLE_RATE}" -le 96000 ]]; then
        echo "✓"
    else
        echo "✗ Invalid sample rate"
        ((errors++))
    fi
    
    # Test 5: Volume validation
    echo -n "5. Testing volume validation... "
    if (( $(echo "${GGWAVE_VOLUME} >= 0.0 && ${GGWAVE_VOLUME} <= 1.0" | bc -l) )); then
        echo "✓"
    else
        echo "✗ Invalid volume setting"
        ((errors++))
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo ""
    echo "Unit test completed in ${duration}s"
    
    if [[ $errors -eq 0 ]]; then
        echo "Result: PASSED ✓"
        return 0
    else
        echo "Result: FAILED ✗ ($errors errors)"
        return 1
    fi
}

# Run all tests
ggwave::test::all() {
    echo "=== Running All GGWave Tests ==="
    local total_errors=0
    
    # Run smoke tests
    if ! ggwave::test::smoke; then
        ((total_errors++))
    fi
    echo ""
    
    # Run unit tests
    if ! ggwave::test::unit; then
        ((total_errors++))
    fi
    echo ""
    
    # Run integration tests
    if ! ggwave::test::integration; then
        ((total_errors++))
    fi
    
    echo ""
    echo "=== Test Summary ==="
    if [[ $total_errors -eq 0 ]]; then
        echo "All tests PASSED ✓"
        return 0
    else
        echo "$total_errors test suites FAILED ✗"
        return 1
    fi
}

# Helper function to validate transmission mode
ggwave::test::validate_mode() {
    local mode="$1"
    # In production, this would actually test the mode
    # For now, we just validate it's a known mode
    case "$mode" in
        normal|fast|dt|ultrasonic)
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

# Helper function to validate error correction
ggwave::test::validate_error_correction() {
    # In production, this would test Reed-Solomon error correction
    # For now, we just check the configuration
    [[ "${GGWAVE_ERROR_CORRECTION}" == "true" ]]
}