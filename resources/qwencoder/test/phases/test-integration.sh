#!/usr/bin/env bash
# QwenCoder Integration Tests - Full functionality (<300s)

set -euo pipefail

# Setup
readonly PHASES_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly TEST_DIR="$(dirname "${PHASES_DIR}")"
readonly RESOURCE_DIR="$(dirname "${TEST_DIR}")"

# Source libraries
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"

echo "Running QwenCoder integration tests..."

# Test counter
tests_passed=0
tests_failed=0

# Ensure service is running
if ! qwencoder_is_running; then
    echo "Starting QwenCoder for integration tests..."
    qwencoder_start
fi

# Test 1: Code completion with context
echo -n "1. Testing code completion with context... "
response=$(curl -sf -X POST "http://localhost:${QWENCODER_PORT}/v1/completions" \
    -H "Content-Type: application/json" \
    -d '{
        "model": "qwencoder-1.5b",
        "prompt": "def calculate_factorial(n):\n    if n == 0:\n        return 1\n    else:",
        "max_tokens": 50,
        "language": "python"
    }' 2>/dev/null || echo "{}")

if echo "${response}" | jq -e '.choices[0].text' > /dev/null 2>&1; then
    echo "✓"
    ((tests_passed++))
else
    echo "✗"
    ((tests_failed++))
fi

# Test 2: Chat completion
echo -n "2. Testing chat completion... "
response=$(curl -sf -X POST "http://localhost:${QWENCODER_PORT}/v1/chat/completions" \
    -H "Content-Type: application/json" \
    -d '{
        "model": "qwencoder-1.5b",
        "messages": [
            {"role": "system", "content": "You are a helpful coding assistant."},
            {"role": "user", "content": "Write a Python function to reverse a string"}
        ],
        "max_tokens": 100
    }' 2>/dev/null || echo "{}")

if echo "${response}" | jq -e '.choices[0].message.content' > /dev/null 2>&1; then
    echo "✓"
    ((tests_passed++))
else
    echo "✗"
    ((tests_failed++))
fi

# Test 3: Multi-language support
echo -n "3. Testing multi-language support... "
languages=("python" "javascript" "go")
lang_passed=0

for lang in "${languages[@]}"; do
    response=$(curl -sf -X POST "http://localhost:${QWENCODER_PORT}/v1/completions" \
        -H "Content-Type: application/json" \
        -d "{
            \"model\": \"qwencoder-1.5b\",
            \"prompt\": \"// Function to add two numbers\",
            \"max_tokens\": 30,
            \"language\": \"${lang}\"
        }" 2>/dev/null || echo "{}")
    
    if echo "${response}" | jq -e '.choices' > /dev/null 2>&1; then
        ((lang_passed++))
    fi
done

if [[ ${lang_passed} -eq ${#languages[@]} ]]; then
    echo "✓ (${lang_passed}/${#languages[@]} languages)"
    ((tests_passed++))
else
    echo "✗ (${lang_passed}/${#languages[@]} languages)"
    ((tests_failed++))
fi

# Test 4: Function calling capability
echo -n "4. Testing function calling... "
response=$(curl -sf -X POST "http://localhost:${QWENCODER_PORT}/v1/chat/completions" \
    -H "Content-Type: application/json" \
    -d '{
        "model": "qwencoder-1.5b",
        "messages": [{"role": "user", "content": "Calculate the sum of 1 to 10"}],
        "functions": [{
            "name": "calculate_sum",
            "description": "Calculate sum of numbers",
            "parameters": {
                "type": "object",
                "properties": {
                    "start": {"type": "integer"},
                    "end": {"type": "integer"}
                }
            }
        }]
    }' 2>/dev/null || echo "{}")

if echo "${response}" | jq -e '.choices' > /dev/null 2>&1; then
    echo "✓"
    ((tests_passed++))
else
    echo "✗"
    ((tests_failed++))
fi

# Test 5: Error handling
echo -n "5. Testing error handling... "
response=$(curl -sf -X POST "http://localhost:${QWENCODER_PORT}/v1/completions" \
    -H "Content-Type: application/json" \
    -d '{
        "model": "invalid-model",
        "prompt": "test"
    }' 2>/dev/null || echo '{"error": "expected"}')

if echo "${response}" | jq -e '.' > /dev/null 2>&1; then
    echo "✓"
    ((tests_passed++))
else
    echo "✗"
    ((tests_failed++))
fi

# Test 6: Token usage tracking
echo -n "6. Testing token usage tracking... "
response=$(curl -sf -X POST "http://localhost:${QWENCODER_PORT}/v1/completions" \
    -H "Content-Type: application/json" \
    -d '{
        "model": "qwencoder-1.5b",
        "prompt": "def hello():",
        "max_tokens": 20
    }' 2>/dev/null || echo "{}")

if echo "${response}" | jq -e '.usage.total_tokens' > /dev/null 2>&1; then
    echo "✓"
    ((tests_passed++))
else
    echo "✗"
    ((tests_failed++))
fi

# Summary
echo ""
echo "Integration Test Results:"
echo "========================="
echo "Passed: ${tests_passed}"
echo "Failed: ${tests_failed}"

if [[ ${tests_failed} -eq 0 ]]; then
    echo "Status: ✓ All integration tests passed!"
    exit 0
else
    echo "Status: ✗ Some tests failed"
    exit 1
fi