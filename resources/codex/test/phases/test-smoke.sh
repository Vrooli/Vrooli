#!/usr/bin/env bash
################################################################################
# Codex Smoke Tests - Quick validation (<30s)
################################################################################

set -euo pipefail

# Setup paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CODEX_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
APP_ROOT="$(cd "${CODEX_DIR}/../.." && pwd)"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/log.sh" || exit 1

# Main tests
log::info "Running Codex smoke tests..."

TESTS_PASSED=0
TESTS_FAILED=0
TESTS_TOTAL=10

# Test 1: CLI exists
if [[ -f "${CODEX_DIR}/cli.sh" ]]; then
    log::success "✓ CLI script exists"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    log::error "✗ CLI script exists: File not found"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Test 2: CLI is executable
if [[ -x "${CODEX_DIR}/cli.sh" ]]; then
    log::success "✓ CLI is executable"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    log::error "✗ CLI is executable: Not executable"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Test 3: Config files exist
if [[ -f "${CODEX_DIR}/config/defaults.sh" ]] && \
   [[ -f "${CODEX_DIR}/config/runtime.json" ]] && \
   [[ -f "${CODEX_DIR}/config/schema.json" ]]; then
    log::success "✓ Config files exist"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    log::error "✗ Config files exist: Missing config files"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Test 4: Test directory structure
if [[ -d "${CODEX_DIR}/test" ]] && \
   [[ -d "${CODEX_DIR}/test/phases" ]] && \
   [[ -f "${CODEX_DIR}/test/run-tests.sh" ]]; then
    log::success "✓ Test structure exists"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    log::error "✗ Test structure exists: Missing test directories"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Test 5: Libraries exist
if [[ -f "${CODEX_DIR}/lib/common.sh" ]] && \
   [[ -f "${CODEX_DIR}/lib/core.sh" ]] && \
   [[ -f "${CODEX_DIR}/lib/test.sh" ]]; then
    log::success "✓ Core libraries exist"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    log::error "✗ Core libraries exist: Missing library files"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Test 6: Directories exist
if [[ -d "${HOME}/.codex/scripts" ]] && \
   [[ -d "${HOME}/.codex/outputs" ]]; then
    log::success "✓ Working directories exist"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    # Try to create them
    mkdir -p "${HOME}/.codex/scripts" "${HOME}/.codex/outputs" 2>/dev/null || true
    if [[ -d "${HOME}/.codex/scripts" ]] && [[ -d "${HOME}/.codex/outputs" ]]; then
        log::success "✓ Working directories created"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        log::error "✗ Working directories: Cannot create"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
fi

# Test 7: Help command works
if timeout 2 "${CODEX_DIR}/cli.sh" help &>/dev/null; then
    log::success "✓ Help command works"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    log::error "✗ Help command: Failed or timed out"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Test 8: Status command works
if timeout 2 "${CODEX_DIR}/cli.sh" status &>/dev/null; then
    log::success "✓ Status command works"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    log::error "✗ Status command: Failed or timed out"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Test 9: Info command works
if timeout 2 "${CODEX_DIR}/cli.sh" info &>/dev/null; then
    log::success "✓ Info command works"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    log::error "✗ Info command: Failed or timed out"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Test 10: Secrets config exists
if [[ -f "${CODEX_DIR}/config/secrets.yaml" ]]; then
    log::success "✓ Secrets configuration exists"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    log::error "✗ Secrets configuration: Missing secrets.yaml"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Report results
echo ""
log::info "Test Results: $TESTS_PASSED/$TESTS_TOTAL passed"

if [ $TESTS_FAILED -gt 0 ]; then
    log::error "Smoke tests failed: $TESTS_FAILED test(s) failed"
    exit 1
fi

log::success "All smoke tests passed!"
exit 0