#!/usr/bin/env bash
# Redis Mock - Tier 2 (Stateful)
# 
# Provides stateful Redis mock with essential operations for testing:
# - Key-value operations with TTL support
# - List operations for queues
# - Basic pub/sub for event bus
# - Transaction support (MULTI/EXEC)
# - Error injection for resilience testing
#
# Coverage: ~80% of common Redis use cases in 400 lines

# === Configuration ===
declare -gA REDIS_STATE=()           # Key-value storage
declare -gA REDIS_LISTS=()          # List storage (pipe-separated values)
declare -gA REDIS_EXPIRES=()        # TTL storage (unix timestamps)
declare -gA REDIS_PUBSUB=()         # Channel subscriptions

# Debug and error modes
declare -g REDIS_DEBUG="${REDIS_DEBUG:-}"
declare -g REDIS_ERROR_MODE="${REDIS_ERROR_MODE:-}"

# Configuration
declare -g REDIS_HOST="${REDIS_HOST:-localhost}"
declare -g REDIS_PORT="${REDIS_PORT:-6379}"
declare -g REDIS_VERSION="7.0.0"

# Transaction state
declare -g REDIS_TRANSACTION_MODE=""
declare -ga REDIS_TRANSACTION_QUEUE=()

# === Helper Functions ===
redis_debug() {
    [[ -n "${REDIS_DEBUG:-}" ]] && echo "[MOCK:REDIS] $*" >&2 || true
}

redis_check_error() {
    case "$REDIS_ERROR_MODE" in
        "connection_failed")
            echo "Could not connect to Redis at $REDIS_HOST:$REDIS_PORT: Connection refused" >&2
            return 1
            ;;
        "timeout")
            sleep 5
            echo "Error: Operation timed out" >&2
            return 1
            ;;
        "auth_failed")
            echo "NOAUTH Authentication required." >&2
            return 1
            ;;
    esac
    return 0
}

redis_check_expiry() {
    local key="$1"
    if [[ -n "${REDIS_EXPIRES[$key]:-}" ]]; then
        local now=$(date +%s)
        if [[ $now -ge ${REDIS_EXPIRES[$key]} ]]; then
            redis_debug "Key expired: $key"
            unset "REDIS_STATE[$key]"
            unset "REDIS_EXPIRES[$key]"
            unset "REDIS_LISTS[$key]"
            return 1  # Key expired
        fi
    fi
    return 0  # Key valid
}

# === Main Redis CLI Mock ===
redis-cli() {
    redis_debug "Called with: $*"
    
    # Check for errors
    redis_check_error || return $?
    
    # Parse basic flags
    local args=()
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -h|--host) REDIS_HOST="$2"; shift 2 ;;
            -p|--port) REDIS_PORT="$2"; shift 2 ;;
            -n|--db) shift 2 ;;  # Ignore DB selection for simplicity
            *) args+=("$1"); shift ;;
        esac
    done
    
    # If no command, show error
    if [[ ${#args[@]} -eq 0 ]]; then
        echo "Interactive mode not supported in mock" >&2
        return 1
    fi
    
    # Get command and convert to uppercase
    local cmd="${args[0]^^}"
    local cmd_args=("${args[@]:1}")
    
    # Handle transaction mode
    if [[ "$REDIS_TRANSACTION_MODE" == "active" ]] && [[ "$cmd" != "EXEC" ]] && [[ "$cmd" != "DISCARD" ]]; then
        REDIS_TRANSACTION_QUEUE+=("$cmd ${cmd_args[*]}")
        echo "QUEUED"
        return 0
    fi
    
    # Execute command
    case "$cmd" in
        # === Connection ===
        PING)
            if [[ -n "${cmd_args[0]:-}" ]]; then
                echo "${cmd_args[0]}"
            else
                echo "PONG"
            fi
            ;;
            
        # === Key-Value Operations ===
        SET)
            local key="${cmd_args[0]:-}"
            local value="${cmd_args[1]:-}"
            if [[ -z "$key" ]] || [[ -z "$value" ]]; then
                echo "(error) ERR wrong number of arguments for 'set' command" >&2
                return 1
            fi
            
            # Handle options (EX, NX, XX)
            local ttl=""
            local nx=false xx=false
            for ((i=2; i<${#cmd_args[@]}; i++)); do
                case "${cmd_args[i]^^}" in
                    EX) ttl="${cmd_args[i+1]}"; ((i++)) ;;
                    NX) nx=true ;;
                    XX) xx=true ;;
                esac
            done
            
            # Check NX/XX conditions
            local exists=false
            [[ -n "${REDIS_STATE[$key]:-}" ]] && exists=true
            
            if [[ "$nx" == "true" ]] && [[ "$exists" == "true" ]]; then
                echo "(nil)"
                return 0
            fi
            
            if [[ "$xx" == "true" ]] && [[ "$exists" == "false" ]]; then
                echo "(nil)"
                return 0
            fi
            
            # Set value
            REDIS_STATE[$key]="$value"
            
            # Set TTL if specified
            if [[ -n "$ttl" ]]; then
                REDIS_EXPIRES[$key]=$(($(date +%s) + ttl))
            else
                unset "REDIS_EXPIRES[$key]" 2>/dev/null || true
            fi
            
            echo "OK"
            ;;
            
        GET)
            local key="${cmd_args[0]:-}"
            if [[ -z "$key" ]]; then
                echo "(error) ERR wrong number of arguments for 'get' command" >&2
                return 1
            fi
            
            redis_check_expiry "$key"
            
            if [[ -n "${REDIS_STATE[$key]:-}" ]]; then
                echo "${REDIS_STATE[$key]}"
            else
                echo "(nil)"
            fi
            ;;
            
        DEL)
            local count=0
            for key in "${cmd_args[@]}"; do
                if [[ -n "${REDIS_STATE[$key]:-}" ]] || [[ -n "${REDIS_LISTS[$key]:-}" ]]; then
                    unset "REDIS_STATE[$key]" 2>/dev/null || true
                    unset "REDIS_LISTS[$key]" 2>/dev/null || true
                    unset "REDIS_EXPIRES[$key]" 2>/dev/null || true
                    ((count++))
                fi
            done
            echo "(integer) $count"
            ;;
            
        EXISTS)
            local count=0
            for key in "${cmd_args[@]}"; do
                redis_check_expiry "$key"
                if [[ -n "${REDIS_STATE[$key]:-}" ]] || [[ -n "${REDIS_LISTS[$key]:-}" ]]; then
                    ((count++))
                fi
            done
            echo "(integer) $count"
            ;;
            
        EXPIRE)
            local key="${cmd_args[0]:-}"
            local seconds="${cmd_args[1]:-}"
            if [[ -z "$key" ]] || [[ -z "$seconds" ]]; then
                echo "(error) ERR wrong number of arguments" >&2
                return 1
            fi
            
            if [[ -n "${REDIS_STATE[$key]:-}" ]] || [[ -n "${REDIS_LISTS[$key]:-}" ]]; then
                REDIS_EXPIRES[$key]=$(($(date +%s) + seconds))
                echo "(integer) 1"
            else
                echo "(integer) 0"
            fi
            ;;
            
        TTL)
            local key="${cmd_args[0]:-}"
            if [[ -z "$key" ]]; then
                echo "(error) ERR wrong number of arguments" >&2
                return 1
            fi
            
            if [[ -z "${REDIS_STATE[$key]:-}" ]] && [[ -z "${REDIS_LISTS[$key]:-}" ]]; then
                echo "(integer) -2"  # Key doesn't exist
            elif [[ -z "${REDIS_EXPIRES[$key]:-}" ]]; then
                echo "(integer) -1"  # Key exists but no TTL
            else
                local ttl=$((REDIS_EXPIRES[$key] - $(date +%s)))
                if [[ $ttl -lt 0 ]]; then
                    unset "REDIS_STATE[$key]" 2>/dev/null || true
                    unset "REDIS_EXPIRES[$key]" 2>/dev/null || true
                    echo "(integer) -2"
                else
                    echo "(integer) $ttl"
                fi
            fi
            ;;
            
        KEYS)
            local pattern="${cmd_args[0]:-*}"
            # Simple pattern matching (not full glob)
            for key in "${!REDIS_STATE[@]}" "${!REDIS_LISTS[@]}"; do
                redis_check_expiry "$key"
                if [[ "$pattern" == "*" ]] || [[ "$key" == *"$pattern"* ]]; then
                    echo "$key"
                fi
            done | sort -u
            ;;
            
        # === List Operations ===
        LPUSH)
            local key="${cmd_args[0]:-}"
            local values=("${cmd_args[@]:1}")
            
            if [[ -z "$key" ]] || [[ ${#values[@]} -eq 0 ]]; then
                echo "(error) ERR wrong number of arguments" >&2
                return 1
            fi
            
            # Prepend values (in reverse order for LPUSH)
            local list="${REDIS_LISTS[$key]:-}"
            for ((i=${#values[@]}-1; i>=0; i--)); do
                if [[ -z "$list" ]]; then
                    list="${values[i]}"
                else
                    list="${values[i]}|$list"
                fi
            done
            REDIS_LISTS[$key]="$list"
            
            # Return new length
            local IFS='|'
            local array=($list)
            echo "(integer) ${#array[@]}"
            ;;
            
        RPUSH)
            local key="${cmd_args[0]:-}"
            local value="${cmd_args[1]:-}"
            
            if [[ -z "$key" ]] || [[ -z "$value" ]]; then
                echo "(error) ERR wrong number of arguments" >&2
                return 1
            fi
            
            # Append value
            if [[ -z "${REDIS_LISTS[$key]:-}" ]]; then
                REDIS_LISTS[$key]="$value"
            else
                REDIS_LISTS[$key]="${REDIS_LISTS[$key]}|$value"
            fi
            
            # Return new length
            local IFS='|'
            local array=(${REDIS_LISTS[$key]})
            echo "(integer) ${#array[@]}"
            ;;
            
        LPOP)
            local key="${cmd_args[0]:-}"
            if [[ -z "$key" ]]; then
                echo "(error) ERR wrong number of arguments" >&2
                return 1
            fi
            
            if [[ -z "${REDIS_LISTS[$key]:-}" ]]; then
                echo "(nil)"
                return 0
            fi
            
            local IFS='|'
            local array=(${REDIS_LISTS[$key]})
            
            if [[ ${#array[@]} -eq 0 ]]; then
                echo "(nil)"
            else
                echo "${array[0]}"
                # Rebuild list without first element
                if [[ ${#array[@]} -eq 1 ]]; then
                    unset "REDIS_LISTS[$key]"
                else
                    REDIS_LISTS[$key]=$(IFS='|'; echo "${array[*]:1}")
                fi
            fi
            ;;
            
        LLEN)
            local key="${cmd_args[0]:-}"
            if [[ -z "$key" ]]; then
                echo "(error) ERR wrong number of arguments" >&2
                return 1
            fi
            
            if [[ -z "${REDIS_LISTS[$key]:-}" ]]; then
                echo "(integer) 0"
            else
                local IFS='|'
                local array=(${REDIS_LISTS[$key]})
                echo "(integer) ${#array[@]}"
            fi
            ;;
            
        LRANGE)
            local key="${cmd_args[0]:-}"
            local start="${cmd_args[1]:-0}"
            local stop="${cmd_args[2]:--1}"
            
            if [[ -z "$key" ]]; then
                echo "(error) ERR wrong number of arguments" >&2
                return 1
            fi
            
            if [[ -z "${REDIS_LISTS[$key]:-}" ]]; then
                echo "(empty array)"
                return 0
            fi
            
            local IFS='|'
            local array=(${REDIS_LISTS[$key]})
            local len=${#array[@]}
            
            # Handle negative indices
            [[ $start -lt 0 ]] && start=$((len + start))
            [[ $stop -lt 0 ]] && stop=$((len + stop))
            [[ $stop -ge $len ]] && stop=$((len - 1))
            
            for ((i=start; i<=stop && i<len; i++)); do
                echo "${array[i]}"
            done
            ;;
            
        # === Pub/Sub (Basic) ===
        PUBLISH)
            local channel="${cmd_args[0]:-}"
            local message="${cmd_args[1]:-}"
            
            if [[ -z "$channel" ]] || [[ -z "$message" ]]; then
                echo "(error) ERR wrong number of arguments" >&2
                return 1
            fi
            
            # In mock, just return subscriber count (always 0 for simplicity)
            echo "(integer) 0"
            ;;
            
        SUBSCRIBE)
            echo "Reading messages... (press Ctrl-C to quit)"
            echo "1) \"subscribe\""
            echo "2) \"${cmd_args[0]:-channel}\""
            echo "3) (integer) 1"
            ;;
            
        # === Transactions ===
        MULTI)
            REDIS_TRANSACTION_MODE="active"
            REDIS_TRANSACTION_QUEUE=()
            echo "OK"
            ;;
            
        EXEC)
            if [[ "$REDIS_TRANSACTION_MODE" != "active" ]]; then
                echo "(error) ERR EXEC without MULTI" >&2
                return 1
            fi
            
            REDIS_TRANSACTION_MODE=""
            
            # Execute queued commands
            for queued_cmd in "${REDIS_TRANSACTION_QUEUE[@]}"; do
                redis-cli $queued_cmd
            done
            
            REDIS_TRANSACTION_QUEUE=()
            ;;
            
        DISCARD)
            REDIS_TRANSACTION_MODE=""
            REDIS_TRANSACTION_QUEUE=()
            echo "OK"
            ;;
            
        # === Server Info ===
        INFO)
            echo "# Server"
            echo "redis_version:$REDIS_VERSION"
            echo "tcp_port:$REDIS_PORT"
            echo ""
            echo "# Keyspace"
            echo "keys:${#REDIS_STATE[@]}"
            echo "expires:${#REDIS_EXPIRES[@]}"
            ;;
            
        FLUSHDB|FLUSHALL)
            REDIS_STATE=()
            REDIS_LISTS=()
            REDIS_EXPIRES=()
            echo "OK"
            ;;
            
        CONFIG)
            # Basic CONFIG GET/SET support
            case "${cmd_args[0]^^}" in
                GET)
                    echo "1) \"${cmd_args[1]}\""
                    echo "2) \"mock_value\""
                    ;;
                SET)
                    echo "OK"
                    ;;
                *)
                    echo "OK"
                    ;;
            esac
            ;;
            
        *)
            # Default response for unknown commands
            echo "OK"
            ;;
    esac
}

# === Convention-based Test Functions ===
test_redis_connection() {
    redis_debug "Testing connection..."
    local result
    result=$(redis-cli ping 2>&1)
    
    if [[ "$result" == "PONG" ]]; then
        redis_debug "Connection test passed"
        return 0
    else
        redis_debug "Connection test failed: $result"
        return 1
    fi
}

test_redis_health() {
    redis_debug "Testing health..."
    
    # Test connection
    test_redis_connection || return 1
    
    # Test write/read/delete cycle
    local test_key="health_$(date +%s)"
    redis-cli set "$test_key" "healthy" >/dev/null 2>&1 || return 1
    local result
    result=$(redis-cli get "$test_key" 2>/dev/null)
    redis-cli del "$test_key" >/dev/null 2>&1
    
    if [[ "$result" == "healthy" ]]; then
        redis_debug "Health test passed"
        return 0
    else
        redis_debug "Health test failed"
        return 1
    fi
}

test_redis_basic() {
    redis_debug "Testing basic operations..."
    
    # Test SET/GET
    redis-cli set "test_key" "test_value" >/dev/null 2>&1 || return 1
    [[ "$(redis-cli get "test_key")" == "test_value" ]] || return 1
    
    # Test EXISTS
    [[ "$(redis-cli exists "test_key")" == "(integer) 1" ]] || return 1
    
    # Test DEL
    redis-cli del "test_key" >/dev/null 2>&1 || return 1
    [[ "$(redis-cli exists "test_key")" == "(integer) 0" ]] || return 1
    
    # Test LIST operations
    redis-cli rpush "test_list" "item1" >/dev/null 2>&1 || return 1
    redis-cli rpush "test_list" "item2" >/dev/null 2>&1 || return 1
    [[ "$(redis-cli llen "test_list")" == "(integer) 2" ]] || return 1
    [[ "$(redis-cli lpop "test_list")" == "item1" ]] || return 1
    redis-cli del "test_list" >/dev/null 2>&1
    
    redis_debug "Basic test passed"
    return 0
}

# === State Management ===
redis_mock_reset() {
    redis_debug "Resetting mock state"
    REDIS_STATE=()
    REDIS_LISTS=()
    REDIS_EXPIRES=()
    REDIS_PUBSUB=()
    REDIS_TRANSACTION_MODE=""
    REDIS_TRANSACTION_QUEUE=()
    REDIS_ERROR_MODE=""
}

redis_mock_set_error() {
    REDIS_ERROR_MODE="$1"
    redis_debug "Set error mode: $1"
}

redis_mock_dump_state() {
    echo "=== Redis Mock State ==="
    echo "Keys: ${#REDIS_STATE[@]}"
    echo "Lists: ${#REDIS_LISTS[@]}"
    echo "Expires: ${#REDIS_EXPIRES[@]}"
    echo "Transaction: ${REDIS_TRANSACTION_MODE:-inactive}"
    echo "Error Mode: ${REDIS_ERROR_MODE:-none}"
    echo "========================"
}

# === Export Functions ===
export -f redis-cli
export -f test_redis_connection
export -f test_redis_health
export -f test_redis_basic
export -f redis_mock_reset
export -f redis_mock_set_error
export -f redis_mock_dump_state
export -f redis_debug
export -f redis_check_error
export -f redis_check_expiry

# Initialize
redis_debug "Redis Tier 2 mock initialized"

# Ensure we return success when sourced
return 0 2>/dev/null || true