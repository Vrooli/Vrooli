#!/usr/bin/env bash
# Mock Template - Tier 2 (Stateful)
# 
# This template provides the standard structure for Tier 2 mocks.
# Tier 2 mocks include:
# - In-memory state management
# - Basic error simulation
# - Convention-based test functions
# - Debug mode support
# 
# To create a new Tier 2 mock:
# 1. Copy this template to tier2/RESOURCE_NAME.sh
# 2. Replace __RESOURCE__ with your resource name (uppercase for variables)
# 3. Replace __resource__ with your resource name (lowercase for functions)
# 4. Replace __main_command__ with the actual command (e.g., redis-cli, psql)
# 5. Add specific operations to the case statement
# 6. Update test functions for your resource

# === Configuration ===
# State storage - use associative array for key-value pairs
declare -gA __RESOURCE___STATE=()
declare -gA __RESOURCE___METADATA=()  # For TTLs, types, etc.

# Debug mode - set __RESOURCE___DEBUG=1 to enable logging
declare -g __RESOURCE___DEBUG="${__RESOURCE___DEBUG:-}"

# Error injection - set __RESOURCE___ERROR_MODE to simulate failures
declare -g __RESOURCE___ERROR_MODE="${__RESOURCE___ERROR_MODE:-}"

# Configuration values
declare -g __RESOURCE___HOST="${__RESOURCE___HOST:-localhost}"
declare -g __RESOURCE___PORT="${__RESOURCE___PORT:-__DEFAULT_PORT__}"
declare -g __RESOURCE___VERSION="__VERSION__"

# === Helper Functions ===
__resource___debug() {
    [[ -n "$__RESOURCE___DEBUG" ]] && echo "[MOCK:__RESOURCE__] $*" >&2
}

__resource___check_error() {
    case "$__RESOURCE___ERROR_MODE" in
        "connection_failed")
            echo "Error: Connection refused" >&2
            return 1
            ;;
        "timeout")
            sleep 5  # Simulate timeout
            echo "Error: Operation timed out" >&2
            return 1
            ;;
        "auth_failed")
            echo "Error: Authentication failed" >&2
            return 1
            ;;
    esac
    return 0
}

# === Core Mock Function ===
# Replace __main_command__ with the actual command (e.g., redis-cli, psql, ollama)
__main_command__() {
    __resource___debug "Called with: $*"
    
    # Check for injected errors
    __resource___check_error || return $?
    
    # Parse arguments
    local operation="${1:-}"
    shift || true
    
    # Handle operations
    case "$operation" in
        # === Connection/Health Operations ===
        ping|status|health)
            echo "__HEALTHY_RESPONSE__"
            return 0
            ;;
            
        # === State Write Operations ===
        set|put|store|create)
            local key="${1:-}"
            local value="${2:-}"
            if [[ -z "$key" ]]; then
                echo "Error: Key required" >&2
                return 1
            fi
            __RESOURCE___STATE["$key"]="$value"
            __RESOURCE___METADATA["${key}_created"]="$(date +%s)"
            __resource___debug "Stored: $key = $value"
            echo "OK"
            return 0
            ;;
            
        # === State Read Operations ===
        get|fetch|read|show)
            local key="${1:-}"
            if [[ -z "$key" ]]; then
                echo "Error: Key required" >&2
                return 1
            fi
            local value="${__RESOURCE___STATE[$key]:-}"
            if [[ -n "$value" ]]; then
                echo "$value"
                return 0
            else
                echo "(nil)"
                return 0
            fi
            ;;
            
        # === State Delete Operations ===
        delete|remove|del|rm)
            local key="${1:-}"
            if [[ -z "$key" ]]; then
                echo "Error: Key required" >&2
                return 1
            fi
            if [[ -n "${__RESOURCE___STATE[$key]:-}" ]]; then
                unset "__RESOURCE___STATE[$key]"
                unset "__RESOURCE___METADATA[${key}_created]"
                echo "1"  # Number deleted
            else
                echo "0"  # Nothing to delete
            fi
            return 0
            ;;
            
        # === List Operations ===
        list|ls|keys)
            local pattern="${1:-*}"
            local count=0
            for key in "${!__RESOURCE___STATE[@]}"; do
                # Simple pattern matching (replace with proper pattern if needed)
                if [[ "$pattern" == "*" ]] || [[ "$key" == *"$pattern"* ]]; then
                    echo "$key"
                    ((count++))
                fi
            done
            [[ $count -eq 0 ]] && echo "(empty)"
            return 0
            ;;
            
        # === Info Operations ===
        info|version|about)
            echo "Resource: __RESOURCE__"
            echo "Version: $__RESOURCE___VERSION"
            echo "Host: $__RESOURCE___HOST:$__RESOURCE___PORT"
            echo "Keys: ${#__RESOURCE___STATE[@]}"
            return 0
            ;;
            
        # === Clear Operations ===
        flush|clear|reset)
            __RESOURCE___STATE=()
            __RESOURCE___METADATA=()
            echo "OK"
            return 0
            ;;
            
        # === Help/Unknown ===
        help|--help|-h)
            echo "Mock __RESOURCE__ - Available operations:"
            echo "  Connection: ping, status, health"
            echo "  Write: set, put, store, create"
            echo "  Read: get, fetch, read, show"
            echo "  Delete: delete, remove, del, rm"
            echo "  List: list, ls, keys"
            echo "  Info: info, version, about"
            echo "  Clear: flush, clear, reset"
            return 0
            ;;
            
        *)
            # Default response for unknown operations
            echo "OK"
            return 0
            ;;
    esac
}

# === Convention-based Test Functions ===
# These functions are auto-discovered by the test framework

test___resource___connection() {
    __resource___debug "Testing connection..."
    
    # Test basic connectivity
    local result
    result=$(__main_command__ ping 2>&1)
    
    if [[ "$result" == "__HEALTHY_RESPONSE__" ]]; then
        __resource___debug "Connection test passed"
        return 0
    else
        __resource___debug "Connection test failed: $result"
        return 1
    fi
}

test___resource___health() {
    __resource___debug "Testing health..."
    
    # More comprehensive health check
    if ! test___resource___connection; then
        return 1
    fi
    
    # Test that we can write and read
    local test_key="health_check_$(date +%s)"
    local test_value="healthy"
    
    __main_command__ set "$test_key" "$test_value" >/dev/null 2>&1 || return 1
    local result
    result=$(__main_command__ get "$test_key" 2>/dev/null)
    __main_command__ del "$test_key" >/dev/null 2>&1
    
    if [[ "$result" == "$test_value" ]]; then
        __resource___debug "Health test passed"
        return 0
    else
        __resource___debug "Health test failed"
        return 1
    fi
}

test___resource___basic() {
    __resource___debug "Testing basic operations..."
    
    # Test write
    __main_command__ set "test_key" "test_value" >/dev/null 2>&1 || {
        __resource___debug "Basic test failed: couldn't set"
        return 1
    }
    
    # Test read
    local result
    result=$(__main_command__ get "test_key" 2>/dev/null)
    if [[ "$result" != "test_value" ]]; then
        __resource___debug "Basic test failed: couldn't get"
        return 1
    fi
    
    # Test delete
    __main_command__ del "test_key" >/dev/null 2>&1 || {
        __resource___debug "Basic test failed: couldn't delete"
        return 1
    }
    
    # Verify deletion
    result=$(__main_command__ get "test_key" 2>/dev/null)
    if [[ "$result" != "(nil)" ]]; then
        __resource___debug "Basic test failed: key still exists after delete"
        return 1
    fi
    
    __resource___debug "Basic test passed"
    return 0
}

# === State Management Functions ===
# These are helpers for testing and debugging

__resource___mock_reset() {
    __resource___debug "Resetting mock state"
    __RESOURCE___STATE=()
    __RESOURCE___METADATA=()
    __RESOURCE___ERROR_MODE=""
}

__resource___mock_set_state() {
    local key="$1"
    local value="$2"
    __RESOURCE___STATE["$key"]="$value"
    __RESOURCE___METADATA["${key}_created"]="$(date +%s)"
    __resource___debug "Set state: $key = $value"
}

__resource___mock_get_state() {
    local key="$1"
    echo "${__RESOURCE___STATE[$key]:-}"
}

__resource___mock_dump_state() {
    echo "=== __RESOURCE__ Mock State ==="
    echo "Configuration:"
    echo "  Host: $__RESOURCE___HOST:$__RESOURCE___PORT"
    echo "  Version: $__RESOURCE___VERSION"
    echo "  Debug: ${__RESOURCE___DEBUG:-off}"
    echo "  Error Mode: ${__RESOURCE___ERROR_MODE:-none}"
    echo ""
    echo "State (${#__RESOURCE___STATE[@]} keys):"
    for key in "${!__RESOURCE___STATE[@]}"; do
        echo "  $key: ${__RESOURCE___STATE[$key]}"
    done
    echo ""
    echo "Metadata:"
    for key in "${!__RESOURCE___METADATA[@]}"; do
        echo "  $key: ${__RESOURCE___METADATA[$key]}"
    done
    echo "=========================="
}

__resource___mock_set_error() {
    __RESOURCE___ERROR_MODE="$1"
    __resource___debug "Set error mode: $1"
}

# === Export Functions ===
# Export all functions that tests might need
export -f __main_command__
export -f test___resource___connection
export -f test___resource___health
export -f test___resource___basic
export -f __resource___mock_reset
export -f __resource___mock_set_state
export -f __resource___mock_get_state
export -f __resource___mock_dump_state
export -f __resource___mock_set_error
export -f __resource___debug
export -f __resource___check_error

# === Initialization ===
# Initialize state when sourced
__resource___debug "Mock initialized"