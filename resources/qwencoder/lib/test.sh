#!/usr/bin/env bash
# QwenCoder Test Functions

set -euo pipefail

# Run smoke tests
run_smoke_tests() {
    echo "Running QwenCoder smoke tests..."
    
    # Test 1: Health check
    echo -n "Testing health endpoint... "
    if timeout 5 curl -sf "http://localhost:${QWENCODER_PORT}/health" > /dev/null 2>&1; then
        echo "✓"
    else
        echo "✗"
        return 1
    fi
    
    # Test 2: Models endpoint
    echo -n "Testing models endpoint... "
    if timeout 5 curl -sf "http://localhost:${QWENCODER_PORT}/models" > /dev/null 2>&1; then
        echo "✓"
    else
        echo "✗"
        return 1
    fi
    
    # Test 3: Basic completion
    echo -n "Testing completion endpoint... "
    local response=$(timeout 10 curl -sf -X POST "http://localhost:${QWENCODER_PORT}/v1/completions" \
        -H "Content-Type: application/json" \
        -d '{
            "model": "qwencoder-1.5b",
            "prompt": "def hello():",
            "max_tokens": 50
        }' 2>/dev/null || echo "")
    
    if [[ -n "${response}" ]]; then
        echo "✓"
    else
        echo "✗"
        return 1
    fi
    
    echo "Smoke tests passed!"
    return 0
}

# Run integration tests
run_integration_tests() {
    echo "Running QwenCoder integration tests..."
    
    # Test 1: Complete code generation
    echo -n "Testing code generation... "
    local response=$(curl -sf -X POST "http://localhost:${QWENCODER_PORT}/v1/completions" \
        -H "Content-Type: application/json" \
        -d '{
            "model": "qwencoder-1.5b",
            "prompt": "def fibonacci(n):",
            "max_tokens": 100,
            "language": "python"
        }' 2>/dev/null || echo "")
    
    if echo "${response}" | jq -e '.choices[0].text' > /dev/null 2>&1; then
        echo "✓"
    else
        echo "✗"
        return 1
    fi
    
    # Test 2: Chat completion
    echo -n "Testing chat completion... "
    response=$(curl -sf -X POST "http://localhost:${QWENCODER_PORT}/v1/chat/completions" \
        -H "Content-Type: application/json" \
        -d '{
            "model": "qwencoder-1.5b",
            "messages": [
                {"role": "user", "content": "Write a Python function to sort a list"}
            ],
            "max_tokens": 150
        }' 2>/dev/null || echo "")
    
    if echo "${response}" | jq -e '.choices[0].message.content' > /dev/null 2>&1; then
        echo "✓"
    else
        echo "✗"
        return 1
    fi
    
    # Test 3: Multi-language support
    echo -n "Testing multi-language support... "
    local languages=("python" "javascript" "go" "java")
    local all_passed=true
    
    for lang in "${languages[@]}"; do
        response=$(curl -sf -X POST "http://localhost:${QWENCODER_PORT}/v1/completions" \
            -H "Content-Type: application/json" \
            -d "{
                \"model\": \"qwencoder-1.5b\",
                \"prompt\": \"// Function to add two numbers\",
                \"max_tokens\": 50,
                \"language\": \"${lang}\"
            }" 2>/dev/null || echo "")
        
        if ! echo "${response}" | jq -e '.choices' > /dev/null 2>&1; then
            all_passed=false
            break
        fi
    done
    
    if ${all_passed}; then
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
    echo "Running QwenCoder unit tests..."
    
    # Test configuration loading
    echo -n "Testing configuration... "
    if [[ -f "${CONFIG_DIR}/defaults.sh" ]] && [[ -f "${CONFIG_DIR}/runtime.json" ]]; then
        echo "✓"
    else
        echo "✗"
        return 1
    fi
    
    # Test directory structure
    echo -n "Testing directory structure... "
    local required_dirs=("${QWENCODER_DATA_DIR}" "${QWENCODER_LOGS_DIR}" "${QWENCODER_API_DIR}")
    local all_exist=true
    
    for dir in "${required_dirs[@]}"; do
        if [[ ! -d "${dir}" ]]; then
            all_exist=false
            break
        fi
    done
    
    if ${all_exist}; then
        echo "✓"
    else
        echo "✗"
        return 1
    fi
    
    # Test Python environment
    echo -n "Testing Python environment... "
    # Check if Python is available and has minimum version
    if python3 -c "import sys; sys.exit(0 if sys.version_info >= (3,8) else 1)" 2>/dev/null; then
        echo "✓"
    else
        echo "✗ (Python 3.8+ required)"
        return 1
    fi
    
    echo "Unit tests passed!"
    return 0
}

# Run all tests
run_all_tests() {
    echo "Running all QwenCoder tests..."
    echo "=============================="
    
    local failed=0
    
    # Ensure service is running for tests
    if ! qwencoder_is_running; then
        echo "Starting QwenCoder for testing..."
        qwencoder_start
    fi
    
    # Run test suites
    if ! run_smoke_tests; then
        echo "Smoke tests failed"
        ((failed++))
    fi
    
    if ! run_integration_tests; then
        echo "Integration tests failed"
        ((failed++))
    fi
    
    if ! run_unit_tests; then
        echo "Unit tests failed"
        ((failed++))
    fi
    
    if [[ ${failed} -eq 0 ]]; then
        echo ""
        echo "All tests passed successfully!"
        return 0
    else
        echo ""
        echo "${failed} test suite(s) failed"
        return 1
    fi
}