#!/usr/bin/env bash
# Test Helper - Provides compatibility layer for BATS tests
#
# This helper enables gradual migration from legacy to Tier 2 mocks
# by providing a unified interface that works with both architectures.
#
# Usage in BATS tests:
#   source __test/mocks/test_helper.sh
#   load_test_mock "redis"

# === Configuration ===
export MOCK_TEST_HELPER_VERSION="1.0.0"
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
export MOCK_BASE_DIR="${MOCK_BASE_DIR:-${APP_ROOT}/__test/mocks}"
export MOCK_MODE="${MOCK_MODE:-tier2}"  # Default to Tier 2

# Source the adapter
if ! source "${MOCK_BASE_DIR}/adapter.sh" 2>/dev/null; then
    # Only output error if not in BATS context or if debug is enabled
    if [[ -z "${BATS_VERSION:-}" ]] || [[ -n "${MOCK_DEBUG:-}" ]]; then
        echo "Error: Could not load mock adapter from ${MOCK_BASE_DIR}/adapter.sh" >&2
        echo "Current directory: $(pwd)" >&2
        echo "APP_ROOT: ${APP_ROOT}" >&2
    fi
    # In BATS context, return error code instead of exiting
    [[ -n "${BATS_VERSION:-}" ]] && return 1
    exit 1
fi

# Ensure MOCK_TIER2_DIR is set and exported (fix for BATS environment)
export MOCK_TIER2_DIR="${MOCK_BASE_DIR}/tier2"
export MOCK_LEGACY_DIR="${MOCK_BASE_DIR}/../mocks-legacy"

# === BATS Compatibility Functions ===

# Load a mock for testing (BATS-friendly wrapper)
load_test_mock() {
    local mock_name="$1"
    local force_mode="${2:-}"
    
    # Debug: Check paths
    if [[ -n "${BATS_TEST_FILENAME:-}" ]]; then
        echo "[DEBUG] load_test_mock called for: $mock_name" >&2
        echo "[DEBUG] MOCK_BASE_DIR: ${MOCK_BASE_DIR}" >&2
        echo "[DEBUG] MOCK_TIER2_DIR: ${MOCK_TIER2_DIR}" >&2
        echo "[DEBUG] APP_ROOT: ${APP_ROOT}" >&2
    fi
    
    # Set adapter mode from environment if specified
    [[ -n "$MOCK_MODE" ]] && export MOCK_ADAPTER_MODE="$MOCK_MODE"
    
    # Load the mock
    if load_mock "$mock_name" "$force_mode"; then
        # Add BATS-specific compatibility
        setup_bats_compatibility "$mock_name"
        return 0
    else
        echo "Failed to load mock: $mock_name" >&2
        return 1
    fi
}

# Setup BATS-specific compatibility shims
setup_bats_compatibility() {
    local mock_name="$1"
    
    # Add common test assertions
    case "$mock_name" in
        redis)
            # BATS tests often use these patterns
            assert_redis_running() {
                test_redis_connection || fail "Redis is not running"
            }
            
            assert_redis_key() {
                local key="$1"
                local expected="$2"
                local actual=$(redis get "$key")
                [[ "$actual" == "$expected" ]] || \
                    fail "Expected redis[$key]='$expected', got '$actual'"
            }
            ;;
            
        postgres)
            # Common PostgreSQL assertions
            assert_postgres_running() {
                test_postgres_connection || fail "PostgreSQL is not running"
            }
            
            assert_table_exists() {
                local table="$1"
                psql -c "SELECT 1 FROM $table LIMIT 1" >/dev/null 2>&1 || \
                    fail "Table $table does not exist"
            }
            ;;
            
        docker)
            # Docker-specific assertions
            assert_container_running() {
                local container="$1"
                docker ps --format "{{.Names}}" | grep -q "^${container}$" || \
                    fail "Container $container is not running"
            }
            ;;
    esac
}

# === Test Setup/Teardown Helpers ===

# Common setup for tests
test_setup() {
    # Reset all loaded mocks to clean state
    for mock in "${!LOADED_MOCKS[@]}"; do
        if declare -F "${mock}_mock_reset" >/dev/null 2>&1; then
            "${mock}_mock_reset"
        elif declare -F "mock::${mock}::reset" >/dev/null 2>&1; then
            "mock::${mock}::reset"
        fi
    done
}

# Common teardown for tests
test_teardown() {
    # Clean up any test data
    for mock in "${!LOADED_MOCKS[@]}"; do
        case "$mock" in
            redis)
                redis flushall 2>/dev/null || true
                ;;
            postgres)
                psql -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;" 2>/dev/null || true
                ;;
        esac
    done
}

# === Mock Verification Helpers ===

# Verify a mock is working correctly
verify_mock() {
    local mock_name="$1"
    
    echo "Verifying $mock_name mock..."
    
    # Run standard tests
    if declare -F "test_${mock_name}_connection" >/dev/null 2>&1; then
        if test_${mock_name}_connection >/dev/null 2>&1; then
            echo "  ✓ Connection test passed"
        else
            echo "  ✗ Connection test failed" >&2
            return 1
        fi
    fi
    
    if declare -F "test_${mock_name}_health" >/dev/null 2>&1; then
        if test_${mock_name}_health >/dev/null 2>&1; then
            echo "  ✓ Health test passed"
        else
            echo "  ✗ Health test failed" >&2
            return 1
        fi
    fi
    
    if declare -F "test_${mock_name}_basic" >/dev/null 2>&1; then
        if test_${mock_name}_basic >/dev/null 2>&1; then
            echo "  ✓ Basic test passed"
        else
            echo "  ✗ Basic test failed" >&2
            return 1
        fi
    fi
    
    echo "  ✓ All tests passed for $mock_name"
    return 0
}

# === Error Injection Helpers ===

# Inject errors for testing error handling
inject_test_error() {
    local mock_name="$1"
    local error_type="$2"
    
    # Try Tier 2 method first
    if declare -F "${mock_name}_mock_set_error" >/dev/null 2>&1; then
        "${mock_name}_mock_set_error" "$error_type"
    # Fall back to legacy method
    elif declare -F "mock::${mock_name}::inject_error" >/dev/null 2>&1; then
        "mock::${mock_name}::inject_error" "$error_type"
    else
        echo "Warning: Cannot inject error for $mock_name" >&2
    fi
}

# Clear injected errors
clear_test_errors() {
    local mock_name="$1"
    
    # Try Tier 2 method
    if declare -F "${mock_name}_mock_set_error" >/dev/null 2>&1; then
        "${mock_name}_mock_set_error" ""
    # Try legacy method
    elif declare -F "mock::${mock_name}::inject_error" >/dev/null 2>&1; then
        "mock::${mock_name}::inject_error" ""
    fi
}

# === Batch Operations ===

# Load all mocks for a resource category
load_resource_test_mocks() {
    local category="$1"
    
    case "$category" in
        storage)
            load_test_mock "redis"
            load_test_mock "postgres"
            load_test_mock "minio"
            load_test_mock "qdrant"
            ;;
        ai)
            load_test_mock "ollama"
            load_test_mock "whisper"
            load_test_mock "claude-code"
            ;;
        automation)
            load_test_mock "n8n"
            load_test_mock "node-red"
            load_test_mock "windmill"
            ;;
        core)
            load_test_mock "docker"
            load_test_mock "filesystem"
            load_test_mock "http"
            ;;
        *)
            echo "Unknown resource category: $category" >&2
            return 1
            ;;
    esac
}

# === BATS-Specific Utilities ===

# Skip test if mock not available
skip_if_mock_unavailable() {
    local mock_name="$1"
    local available=$(detect_mock_availability "$mock_name")
    
    if [[ -z "$available" ]]; then
        skip "Mock '$mock_name' not available"
    fi
}

# Run only if using Tier 2 mocks
require_tier2() {
    if [[ "$MOCK_MODE" != "tier2" ]]; then
        skip "Test requires Tier 2 mocks"
    fi
}

# Run only if using legacy mocks
require_legacy() {
    if [[ "$MOCK_MODE" != "legacy" ]]; then
        skip "Test requires legacy mocks"
    fi
}

# === Export Functions ===
export -f load_test_mock
export -f setup_bats_compatibility
export -f test_setup
export -f test_teardown
export -f verify_mock
export -f inject_test_error
export -f clear_test_errors
export -f load_resource_test_mocks
export -f skip_if_mock_unavailable
export -f require_tier2
export -f require_legacy

# === Initialize ===
# Only output when explicitly in debug mode to avoid breaking test output
[[ -n "${MOCK_DEBUG:-}" ]] && echo "Test helper loaded (v${MOCK_TEST_HELPER_VERSION}, mode: ${MOCK_MODE})"

# Ensure we return success
return 0 2>/dev/null || true