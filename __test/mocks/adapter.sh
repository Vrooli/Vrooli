#!/usr/bin/env bash
# Mock Adapter Layer - Provides compatibility between legacy and Tier 2 mocks
#
# This adapter enables gradual migration from legacy mocks to Tier 2 architecture
# by detecting which mock system to use and providing a compatibility layer.
#
# Usage:
#   source mock_adapter.sh
#   load_mock "redis"     # Automatically loads appropriate mock version
#   load_mock "postgres" --tier2  # Force Tier 2
#   load_mock "docker" --legacy   # Force legacy

# === Configuration ===
declare -g MOCK_ADAPTER_VERSION="1.0.0"
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
declare -g MOCK_BASE_DIR="${MOCK_BASE_DIR:-${APP_ROOT}/__test/mocks}"
declare -g MOCK_TIER2_DIR="${MOCK_BASE_DIR}/tier2"
declare -g MOCK_LEGACY_DIR="${MOCK_BASE_DIR}/../mocks-legacy"
declare -g MOCK_ADAPTER_DEBUG="${MOCK_ADAPTER_DEBUG:-}"
declare -g MOCK_ADAPTER_MODE="${MOCK_ADAPTER_MODE:-auto}"  # auto|tier2|legacy

# Track loaded mocks to prevent duplicates
declare -gA LOADED_MOCKS=()

# === Debug Functions ===
mock_adapter_debug() {
    [[ -n "$MOCK_ADAPTER_DEBUG" ]] && echo "[MOCK_ADAPTER] $*" >&2
}

# === Mock Detection ===
detect_mock_availability() {
    local mock_name="$1"
    local tier2_path="${MOCK_TIER2_DIR}/${mock_name}.sh"
    local legacy_path="${MOCK_LEGACY_DIR}/${mock_name}.sh"
    
    local result=""
    [[ -f "$tier2_path" ]] && result="${result}tier2:"
    [[ -f "$legacy_path" ]] && result="${result}legacy:"
    
    echo "${result%:}"  # Remove trailing colon
}

# === Main Load Function ===
load_mock() {
    local mock_name="$1"
    local force_mode="${2:-}"
    
    # Check if already loaded
    if [[ -n "${LOADED_MOCKS[$mock_name]}" ]]; then
        mock_adapter_debug "Mock '$mock_name' already loaded from ${LOADED_MOCKS[$mock_name]}"
        return 0
    fi
    
    # Determine which version to load
    local load_mode="$MOCK_ADAPTER_MODE"
    if [[ -n "$force_mode" ]]; then
        case "$force_mode" in
            --tier2) load_mode="tier2" ;;
            --legacy) load_mode="legacy" ;;
            *) mock_adapter_debug "Unknown force mode: $force_mode" ;;
        esac
    fi
    
    # Auto-detect if needed
    if [[ "$load_mode" == "auto" ]]; then
        local available=$(detect_mock_availability "$mock_name")
        case "$available" in
            tier2:legacy|tier2)
                load_mode="tier2"
                ;;
            legacy)
                load_mode="legacy"
                ;;
            *)
                # Only show error if not in BATS context or if debug is enabled
                if [[ -z "${BATS_VERSION:-}" ]] || [[ -n "${MOCK_ADAPTER_DEBUG:-}" ]]; then
                    echo "Error: Mock '$mock_name' not found in either tier2 or legacy" >&2
                fi
                return 1
                ;;
        esac
    fi
    
    # Load the mock
    local mock_path=""
    case "$load_mode" in
        tier2)
            mock_path="${MOCK_TIER2_DIR}/${mock_name}.sh"
            ;;
        legacy)
            mock_path="${MOCK_LEGACY_DIR}/${mock_name}.sh"
            ;;
    esac
    
    if [[ ! -f "$mock_path" ]]; then
        # Only show error if not in BATS context or if debug is enabled
        if [[ -z "${BATS_VERSION:-}" ]] || [[ -n "${MOCK_ADAPTER_DEBUG:-}" ]]; then
            echo "Error: Mock file not found: $mock_path" >&2
        fi
        return 1
    fi
    
    mock_adapter_debug "Loading $load_mode mock: $mock_name from $mock_path"
    
    # Source the mock
    source "$mock_path"
    
    # Apply compatibility layer if needed
    if [[ "$load_mode" == "tier2" ]]; then
        apply_tier2_compatibility "$mock_name"
    elif [[ "$load_mode" == "legacy" ]]; then
        apply_legacy_compatibility "$mock_name"
    fi
    
    # Mark as loaded
    LOADED_MOCKS[$mock_name]="$load_mode"
    
    mock_adapter_debug "Successfully loaded $load_mode mock: $mock_name"
    return 0
}

# === Compatibility Layers ===
apply_tier2_compatibility() {
    local mock_name="$1"
    
    # Add compatibility shims for Tier 2 mocks to work with legacy tests
    case "$mock_name" in
        redis)
            # Legacy tests might expect mock::redis::* functions
            if declare -F redis_mock_reset >/dev/null 2>&1; then
                mock::redis::reset() { redis_mock_reset "$@"; }
                mock::redis::set() { redis set "$@"; }
                mock::redis::get() { redis get "$@"; }
                export -f mock::redis::reset mock::redis::set mock::redis::get
            fi
            ;;
        postgres)
            # Legacy tests might expect mock::postgres::* functions
            if declare -F postgres_mock_reset >/dev/null 2>&1; then
                mock::postgres::reset() { postgres_mock_reset "$@"; }
                mock::postgres::query() { psql -c "$@"; }
                export -f mock::postgres::reset mock::postgres::query
            fi
            ;;
        docker)
            # Legacy tests might expect mock::docker::* functions
            if declare -F docker_mock_reset >/dev/null 2>&1; then
                mock::docker::reset() { docker_mock_reset "$@"; }
                mock::docker::ps() { docker ps "$@"; }
                export -f mock::docker::reset mock::docker::ps
            fi
            ;;
    esac
}

apply_legacy_compatibility() {
    local mock_name="$1"
    
    # Add compatibility shims for legacy mocks to work with Tier 2 tests
    case "$mock_name" in
        redis)
            # Tier 2 tests expect test_redis_* functions
            if declare -F mock::redis::health_check >/dev/null 2>&1; then
                test_redis_connection() { mock::redis::health_check "$@"; }
                test_redis_health() { mock::redis::health_check "$@"; }
                test_redis_basic() { 
                    mock::redis::set "test" "value" && \
                    [[ "$(mock::redis::get "test")" == "value" ]]
                }
                redis_mock_reset() { mock::redis::reset "$@"; }
                export -f test_redis_connection test_redis_health test_redis_basic redis_mock_reset
            fi
            ;;
        postgres)
            # Tier 2 tests expect test_postgres_* functions
            if declare -F mock::postgres::health_check >/dev/null 2>&1; then
                test_postgres_connection() { mock::postgres::health_check "$@"; }
                test_postgres_health() { mock::postgres::health_check "$@"; }
                test_postgres_basic() {
                    mock::postgres::query "SELECT 1" >/dev/null 2>&1
                }
                postgres_mock_reset() { mock::postgres::reset "$@"; }
                export -f test_postgres_connection test_postgres_health test_postgres_basic postgres_mock_reset
            fi
            ;;
    esac
}

# === Migration Helpers ===
check_mock_migration_status() {
    echo "=== Mock Migration Status ==="
    echo "Adapter Version: $MOCK_ADAPTER_VERSION"
    echo "Mode: $MOCK_ADAPTER_MODE"
    echo ""
    
    local tier2_count=0
    local legacy_count=0
    local both_count=0
    
    # Check all available mocks
    for mock_file in "$MOCK_TIER2_DIR"/*.sh "$MOCK_LEGACY_DIR"/*.sh; do
        [[ ! -f "$mock_file" ]] && continue
        local mock_name=$(basename "$mock_file" .sh)
        local available=$(detect_mock_availability "$mock_name")
        
        case "$available" in
            tier2:legacy)
                echo "  [BOTH] $mock_name"
                ((both_count++))
                ;;
            tier2)
                echo "  [TIER2] $mock_name"
                ((tier2_count++))
                ;;
            legacy)
                echo "  [LEGACY] $mock_name"
                ((legacy_count++))
                ;;
        esac
    done | sort -u
    
    echo ""
    echo "Summary:"
    echo "  Tier 2 only: $tier2_count"
    echo "  Legacy only: $legacy_count"
    echo "  Both available: $both_count"
    echo ""
    echo "Loaded mocks in this session:"
    for mock in "${!LOADED_MOCKS[@]}"; do
        echo "  $mock: ${LOADED_MOCKS[$mock]}"
    done
}

# === Batch Loading ===
load_resource_mocks() {
    local resource="$1"
    
    # Load common mocks for a resource
    case "$resource" in
        ai)
            load_mock "ollama"
            load_mock "whisper"
            load_mock "claude-code"
            ;;
        storage)
            load_mock "postgres"
            load_mock "redis"
            load_mock "minio"
            load_mock "qdrant"
            ;;
        automation)
            load_mock "n8n"
            load_mock "node-red"
            load_mock "windmill"
            ;;
        *)
            echo "Unknown resource category: $resource" >&2
            return 1
            ;;
    esac
}

# === Test Helpers ===
run_mock_tests() {
    local mock_name="$1"
    
    if [[ -z "${LOADED_MOCKS[$mock_name]}" ]]; then
        echo "Mock '$mock_name' not loaded" >&2
        return 1
    fi
    
    echo "Running tests for $mock_name (${LOADED_MOCKS[$mock_name]} version)..."
    
    # Run standard test functions
    local test_functions=(
        "test_${mock_name}_connection"
        "test_${mock_name}_health"
        "test_${mock_name}_basic"
    )
    
    for test_func in "${test_functions[@]}"; do
        if declare -F "$test_func" >/dev/null 2>&1; then
            echo -n "  $test_func: "
            if $test_func >/dev/null 2>&1; then
                echo "✓"
            else
                echo "✗"
            fi
        fi
    done
}

# === Export Functions ===
export -f load_mock
export -f detect_mock_availability
export -f check_mock_migration_status
export -f load_resource_mocks
export -f run_mock_tests
export -f mock_adapter_debug
export -f apply_tier2_compatibility
export -f apply_legacy_compatibility

# Initialize
mock_adapter_debug "Mock adapter initialized (version $MOCK_ADAPTER_VERSION)"

# Return success
true