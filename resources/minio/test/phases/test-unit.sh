#!/usr/bin/env bash
################################################################################
# MinIO Unit Tests - Library Function Validation
#
# Tests MinIO library functions and utilities
################################################################################

set -euo pipefail

# Setup paths
SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
MINIO_DIR="$(builtin cd "${SCRIPT_DIR}/../.." && builtin pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${MINIO_DIR}/../.." && builtin pwd)}"

# Source dependencies
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/log.sh"
# shellcheck disable=SC1091
source "${MINIO_DIR}/config/defaults.sh"

# Export configuration
minio::export_config 2>/dev/null || true

# Source libraries to test
# shellcheck disable=SC1091
source "${MINIO_DIR}/lib/core.sh" 2>/dev/null || true

################################################################################
# Unit Tests
################################################################################

log::info "Starting MinIO unit tests..."

FAILED=0

# Test 1: Configuration loading
log::info "Test 1: Configuration loading..."

if [[ -n "${MINIO_PORT:-}" ]]; then
    log::success "✓ Configuration variables loaded (MINIO_PORT=${MINIO_PORT})"
else
    log::error "✗ Configuration variables not loaded"
    ((FAILED++))
fi

# Test 2: Default values
log::info "Test 2: Default configuration values..."

if [[ "${MINIO_PORT:-9000}" == "9000" ]] && [[ "${MINIO_CONSOLE_PORT:-9001}" == "9001" ]]; then
    log::success "✓ Default port values are correct"
else
    log::error "✗ Default port values incorrect"
    ((FAILED++))
fi

# Test 3: Container name configuration
log::info "Test 3: Container name configuration..."

EXPECTED_CONTAINER="${MINIO_CONTAINER_NAME:-minio}"
if [[ "$EXPECTED_CONTAINER" == "minio" ]]; then
    log::success "✓ Container name configured correctly"
else
    log::error "✗ Container name misconfigured: $EXPECTED_CONTAINER"
    ((FAILED++))
fi

# Test 4: Function availability
log::info "Test 4: Core function availability..."

CORE_FUNCTIONS=(
    "minio::is_installed"
    "minio::is_running"
    "minio::is_healthy"
    "minio::health_check"
)

for func in "${CORE_FUNCTIONS[@]}"; do
    if command -v "$func" &>/dev/null; then
        log::success "✓ Function available: $func"
    else
        log::warning "⚠ Function not available: $func"
    fi
done

# Test 5: Credential file permissions
log::info "Test 5: Credential file security..."

CREDS_FILE="${HOME}/.minio/config/credentials"
if [[ -f "$CREDS_FILE" ]]; then
    PERMS=$(stat -c "%a" "$CREDS_FILE" 2>/dev/null || stat -f "%OLp" "$CREDS_FILE" 2>/dev/null || echo "unknown")
    
    if [[ "$PERMS" == "600" ]]; then
        log::success "✓ Credential file has secure permissions (600)"
    else
        log::error "✗ Credential file has insecure permissions: $PERMS"
        ((FAILED++))
    fi
else
    log::info "ℹ Credential file not yet created (expected for fresh install)"
fi

# Test 6: Runtime configuration
log::info "Test 6: Runtime configuration..."

RUNTIME_FILE="${MINIO_DIR}/config/runtime.json"
if [[ -f "$RUNTIME_FILE" ]]; then
    # Check required fields
    if jq -e '.startup_order' "$RUNTIME_FILE" &>/dev/null && \
       jq -e '.startup_timeout' "$RUNTIME_FILE" &>/dev/null && \
       jq -e '.dependencies' "$RUNTIME_FILE" &>/dev/null; then
        log::success "✓ Runtime configuration is valid"
    else
        log::error "✗ Runtime configuration missing required fields"
        ((FAILED++))
    fi
else
    log::error "✗ Runtime configuration file missing"
    ((FAILED++))
fi

# Test 7: CLI script executable
log::info "Test 7: CLI script permissions..."

CLI_SCRIPT="${MINIO_DIR}/cli.sh"
if [[ -x "$CLI_SCRIPT" ]]; then
    log::success "✓ CLI script is executable"
else
    log::error "✗ CLI script not executable"
    ((FAILED++))
fi

# Test 8: Test runner executable
log::info "Test 8: Test runner permissions..."

TEST_RUNNER="${MINIO_DIR}/test/run-tests.sh"
if [[ -x "$TEST_RUNNER" ]]; then
    log::success "✓ Test runner is executable"
else
    log::warning "⚠ Test runner not executable"
fi

################################################################################
# Results
################################################################################

if [[ $FAILED -gt 0 ]]; then
    log::error "MinIO unit tests failed: $FAILED tests failed"
    exit 1
else
    log::success "All MinIO unit tests passed"
    exit 0
fi