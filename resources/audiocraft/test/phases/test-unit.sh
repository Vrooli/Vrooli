#!/usr/bin/env bash
################################################################################
# AudioCraft Unit Tests
# Test individual components and functions
################################################################################
set -euo pipefail

# Get directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(dirname "$SCRIPT_DIR")"
RESOURCE_DIR="$(dirname "$TEST_DIR")"

# Load libraries
source "${RESOURCE_DIR}/config/defaults.sh"

echo "📦 AudioCraft Unit Tests"
echo "======================="

# Track failures
FAILED=0

# Test 1: Configuration validation
echo -n "Testing configuration... "
if [[ -n "${AUDIOCRAFT_PORT:-}" ]] && [[ "$AUDIOCRAFT_PORT" =~ ^[0-9]+$ ]]; then
    echo "✅ PASSED"
else
    echo "❌ FAILED"
    FAILED=$((FAILED + 1))
fi

# Test 2: Directory paths
echo -n "Testing directory paths... "
if [[ -n "${AUDIOCRAFT_DATA_DIR:-}" ]] && [[ -n "${AUDIOCRAFT_MODELS_DIR:-}" ]]; then
    echo "✅ PASSED"
else
    echo "❌ FAILED"
    FAILED=$((FAILED + 1))
fi

# Test 3: Runtime.json validity
echo -n "Testing runtime.json... "
if [[ -f "${RESOURCE_DIR}/config/runtime.json" ]]; then
    if python3 -m json.tool "${RESOURCE_DIR}/config/runtime.json" > /dev/null 2>&1; then
        echo "✅ PASSED"
    else
        echo "❌ FAILED (invalid JSON)"
        FAILED=$((FAILED + 1))
    fi
else
    echo "❌ FAILED (file not found)"
    FAILED=$((FAILED + 1))
fi

# Test 4: Schema.json validity
echo -n "Testing schema.json... "
if [[ -f "${RESOURCE_DIR}/config/schema.json" ]]; then
    if python3 -m json.tool "${RESOURCE_DIR}/config/schema.json" > /dev/null 2>&1; then
        echo "✅ PASSED"
    else
        echo "❌ FAILED (invalid JSON)"
        FAILED=$((FAILED + 1))
    fi
else
    echo "❌ FAILED (file not found)"
    FAILED=$((FAILED + 1))
fi

# Test 5: CLI script executable
echo -n "Testing CLI script... "
if [[ -f "${RESOURCE_DIR}/cli.sh" ]] && [[ -x "${RESOURCE_DIR}/cli.sh" ]]; then
    echo "✅ PASSED"
else
    echo "❌ FAILED"
    FAILED=$((FAILED + 1))
fi

# Test 6: Port not conflicting
echo -n "Testing port availability... "
if [[ "$AUDIOCRAFT_PORT" != "7860" ]]; then  # Not conflicting with musicgen
    echo "✅ PASSED"
else
    echo "❌ FAILED (conflicts with musicgen)"
    FAILED=$((FAILED + 1))
fi

# Test 7: Model size validation
echo -n "Testing model size configuration... "
VALID_SIZES=("small" "medium" "large" "melody")
if [[ " ${VALID_SIZES[@]} " =~ " ${AUDIOCRAFT_MODEL_SIZE} " ]]; then
    echo "✅ PASSED"
else
    echo "❌ FAILED (invalid model size)"
    FAILED=$((FAILED + 1))
fi

# Summary
echo "======================="
if [[ $FAILED -eq 0 ]]; then
    echo "✅ All unit tests passed"
    exit 0
else
    echo "❌ $FAILED unit test(s) failed"
    exit 1
fi