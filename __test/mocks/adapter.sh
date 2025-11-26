#!/usr/bin/env bash
# Mock Adapter Layer - Tier 2 Mock System Interface
#
# This adapter provides a unified interface for loading Tier 2 mocks
# with compatibility functions for BATS and shell script testing.
#
# Usage:
#   source __test/mocks/adapter.sh
#   load_mock "redis"     # Load Tier 2 mock
#   load_mock "postgres"  # Load Tier 2 mock

# === Configuration ===
declare -g MOCK_ADAPTER_VERSION="1.0.0"
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../.." 2>/dev/null && pwd)}"
declare -g MOCK_BASE_DIR="${MOCK_BASE_DIR:-${APP_ROOT}/__test/mocks}"
declare -g MOCK_TIER2_DIR="${MOCK_BASE_DIR}/tier2"
declare -g MOCK_ADAPTER_DEBUG="${MOCK_ADAPTER_DEBUG:-}"
declare -g MOCK_ADAPTER_MODE="${MOCK_ADAPTER_MODE:-tier2}"

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
    
    if [[ -f "$tier2_path" ]]; then
        echo "tier2"
    else
        echo ""
    fi
}

# === Main Load Function ===
load_mock() {
    local mock_name="$1"
    
    mock_adapter_debug "Loading mock: $mock_name"
    
    # Check if already loaded
    if [[ -n "${LOADED_MOCKS[$mock_name]:-}" ]]; then
        mock_adapter_debug "Mock $mock_name already loaded"
        return 0
    fi
    
    # Check availability
    local available
    available=$(detect_mock_availability "$mock_name")
    
    if [[ "$available" != "tier2" ]]; then
        if [[ -z "${BATS_VERSION:-}" ]] || [[ -n "${MOCK_ADAPTER_DEBUG:-}" ]]; then
            echo "Error: Mock '$mock_name' not found" >&2
        fi
        return 1
    fi
    
    # Load Tier 2 mock
    local mock_path="${MOCK_TIER2_DIR}/${mock_name}.sh"
    
    if [[ ! -f "$mock_path" ]]; then
        if [[ -z "${BATS_VERSION:-}" ]] || [[ -n "${MOCK_ADAPTER_DEBUG:-}" ]]; then
            echo "Error: Mock file not found: $mock_path" >&2
        fi
        return 1
    fi
    
    # Source the mock
    mock_adapter_debug "Sourcing mock: $mock_path"
    # shellcheck disable=SC1090
    source "$mock_path" || {
        echo "Error: Failed to source mock: $mock_path" >&2
        return 1
    }
    
    # Apply compatibility layer
    apply_tier2_compatibility "$mock_name"
    
    # Mark as loaded
    LOADED_MOCKS[$mock_name]="tier2"
    
    mock_adapter_debug "Successfully loaded $mock_name (tier2)"
    return 0
}

# === Compatibility Layer ===
apply_tier2_compatibility() {
    local mock_name="$1"
    
    # Add compatibility shims for Tier 2 mocks
    case "$mock_name" in
        redis)
            # Some tests might expect 'redis' command as alias to 'redis-cli'
            if declare -F redis-cli >/dev/null 2>&1 && ! declare -F redis >/dev/null 2>&1; then
                redis() {
                    redis-cli "$@"
                }
                export -f redis
            fi
            ;;
        postgres)
            # PostgreSQL alias compatibility
            if declare -F psql >/dev/null 2>&1 && ! declare -F postgres_query >/dev/null 2>&1; then
                postgres_query() {
                    psql -c "$1" 2>/dev/null
                }
                export -f postgres_query
            fi
            ;;
    esac
}

# === Utility Functions ===
list_all_mocks() {
    echo "Available mocks:"
    
    local tier2_count=0
    
    if [[ -d "$MOCK_TIER2_DIR" ]]; then
        for file in "$MOCK_TIER2_DIR"/*.sh; do
            [[ -f "$file" ]] || continue
            local mock_name=$(basename "$file" .sh)
            echo "  [TIER2] $mock_name"
            ((tier2_count++))
        done
    fi
    
    echo ""
    echo "Summary:"
    echo "  Tier 2 mocks: $tier2_count"
}

check_mock_migration_status() {
    echo "Mock Migration Status:"
    echo "  Migration completed: ✅ (August 2025)"
    echo "  Tier 2 mocks: $(find "$MOCK_TIER2_DIR" -name "*.sh" 2>/dev/null | wc -l)"
    echo "  Architecture: Pure Tier 2 (legacy removed)"
}

# === Batch Operations ===
load_resource_mocks() {
    local category="$1"
    
    case "$category" in
        storage)
            load_mock "redis"
            load_mock "postgres"
            load_mock "minio"
            load_mock "qdrant"
            ;;
        ai)
            load_mock "ollama"
            load_mock "whisper"
            load_mock "claude-code"
            ;;
        automation)
            load_mock "node-red"
            ;;
        core)
            load_mock "docker"
            load_mock "filesystem"
            load_mock "http"
            ;;
        *)
            echo "Unknown resource category: $category" >&2
            return 1
            ;;
    esac
}

# === Testing Functions ===
run_mock_tests() {
    local mock_name="$1"
    
    echo "Testing mock: $mock_name"
    
    # Load the mock
    if ! load_mock "$mock_name"; then
        echo "  ✗ Failed to load"
        return 1
    fi
    
    # Test connection if function exists
    if declare -F "test_${mock_name}_connection" >/dev/null 2>&1; then
        if "test_${mock_name}_connection"; then
            echo "  ✓ Connection test passed"
        else
            echo "  ✗ Connection test failed"
            return 1
        fi
    fi
    
    # Test health if function exists  
    if declare -F "test_${mock_name}_health" >/dev/null 2>&1; then
        if "test_${mock_name}_health"; then
            echo "  ✓ Health test passed"
        else
            echo "  ✗ Health test failed"
            return 1
        fi
    fi
    
    echo "  ✓ All tests passed"
    return 0
}

# === Export Functions ===
export -f load_mock
export -f detect_mock_availability
export -f apply_tier2_compatibility
export -f list_all_mocks
export -f check_mock_migration_status
export -f load_resource_mocks
export -f run_mock_tests
export -f mock_adapter_debug

# Initialize
mock_adapter_debug "Mock adapter initialized (version $MOCK_ADAPTER_VERSION)"
mock_adapter_debug "Using Tier 2 mock system (migration completed)"

# Return success
return 0 2>/dev/null || true
