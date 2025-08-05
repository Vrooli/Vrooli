#!/usr/bin/env bash
# Redis Mock Implementation
# 
# Provides a comprehensive mock for Redis operations including:
# - redis-cli command interception
# - Redis server state simulation
# - Connection management
# - Data storage and retrieval
# - Pub/Sub operations
# - Transaction support
#
# This mock follows the same standards as docker.sh mock with:
# - Comprehensive state management
# - File-based persistence for BATS compatibility
# - Integration with centralized logging
# - Test helper functions

# Prevent duplicate loading
[[ -n "${REDIS_MOCK_LOADED:-}" ]] && return 0
declare -g REDIS_MOCK_LOADED=1

# Load dependencies
MOCK_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
[[ -f "$MOCK_DIR/logs.sh" ]] && source "$MOCK_DIR/logs.sh"

# Global configuration
declare -g REDIS_MOCK_STATE_DIR="${REDIS_MOCK_STATE_DIR:-/tmp/redis-mock-state}"
declare -g REDIS_MOCK_DEBUG="${REDIS_MOCK_DEBUG:-}"

# Global state arrays
declare -gA REDIS_MOCK_DATA=()           # Key-value storage
declare -gA REDIS_MOCK_EXPIRES=()        # Key expiration times
declare -gA REDIS_MOCK_LISTS=()          # List data structures
declare -gA REDIS_MOCK_SETS=()           # Set data structures
declare -gA REDIS_MOCK_HASHES=()         # Hash data structures
declare -gA REDIS_MOCK_PUBSUB=()         # Pub/Sub channels
declare -gA REDIS_MOCK_CONFIG=(          # Redis configuration
    [host]="localhost"
    [port]="6379"
    [password]=""
    [db]="0"
    [connected]="true"
    [version]="7.0.11"
    [role]="master"
    [used_memory]="1048576"
    [connected_clients]="1"
    [error_mode]=""
)

# Transaction state
declare -g REDIS_MOCK_TRANSACTION_MODE=""
declare -ga REDIS_MOCK_TRANSACTION_QUEUE=()

# Initialize state directory
mkdir -p "$REDIS_MOCK_STATE_DIR"

# State persistence functions
mock::redis::save_state() {
    local state_file="$REDIS_MOCK_STATE_DIR/redis-state.sh"
    {
        echo "# Redis mock state - $(date)"
        
        # Save arrays using declare -p for proper restoration
        declare -p REDIS_MOCK_CONFIG 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA REDIS_MOCK_CONFIG=()"
        declare -p REDIS_MOCK_DATA 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA REDIS_MOCK_DATA=()"
        declare -p REDIS_MOCK_EXPIRES 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA REDIS_MOCK_EXPIRES=()"
        declare -p REDIS_MOCK_LISTS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA REDIS_MOCK_LISTS=()"
        declare -p REDIS_MOCK_SETS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA REDIS_MOCK_SETS=()"
        declare -p REDIS_MOCK_HASHES 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA REDIS_MOCK_HASHES=()"
        declare -p REDIS_MOCK_PUBSUB 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA REDIS_MOCK_PUBSUB=()"
        
        # Save transaction state
        echo "REDIS_MOCK_TRANSACTION_MODE=\"$REDIS_MOCK_TRANSACTION_MODE\""
        declare -p REDIS_MOCK_TRANSACTION_QUEUE 2>/dev/null | sed 's/declare -a/declare -ga/' || echo "declare -ga REDIS_MOCK_TRANSACTION_QUEUE=()"
    } > "$state_file"
    
    mock::log_state "redis" "Saved Redis state to $state_file"
}

mock::redis::load_state() {
    local state_file="$REDIS_MOCK_STATE_DIR/redis-state.sh"
    if [[ -f "$state_file" ]]; then
        source "$state_file"
        mock::log_state "redis" "Loaded Redis state from $state_file"
    fi
}

# Automatically load state when sourced
mock::redis::load_state

# Main redis-cli interceptor
redis-cli() {
    mock::log_and_verify "redis-cli" "$@"
    
    # Check if Redis is connected
    if [[ "${REDIS_MOCK_CONFIG[connected]}" != "true" ]]; then
        echo "Could not connect to Redis at ${REDIS_MOCK_CONFIG[host]}:${REDIS_MOCK_CONFIG[port]}: Connection refused" >&2
        return 1
    fi
    
    # Check for error injection
    if [[ -n "${REDIS_MOCK_CONFIG[error_mode]}" ]]; then
        case "${REDIS_MOCK_CONFIG[error_mode]}" in
            "connection_timeout")
                echo "Error: Connection timed out" >&2
                return 1
                ;;
            "auth_failed")
                echo "NOAUTH Authentication required." >&2
                return 1
                ;;
            "loading")
                echo "LOADING Redis is loading the dataset in memory" >&2
                return 1
                ;;
        esac
    fi
    
    # Parse command line arguments
    local host="${REDIS_MOCK_CONFIG[host]}"
    local port="${REDIS_MOCK_CONFIG[port]}"
    local password="${REDIS_MOCK_CONFIG[password]}"
    local db="${REDIS_MOCK_CONFIG[db]}"
    local command_args=()
    local raw_mode=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -h|--host)
                host="$2"
                shift 2
                ;;
            -p|--port)
                port="$2"
                shift 2
                ;;
            -a|--pass)
                password="$2"
                shift 2
                ;;
            -n|--db)
                db="$2"
                shift 2
                ;;
            --raw)
                raw_mode="true"
                shift
                ;;
            *)
                command_args+=("$1")
                shift
                ;;
        esac
    done
    
    # If no command provided, enter interactive mode (not supported in mock)
    if [[ ${#command_args[@]} -eq 0 ]]; then
        echo "Mock redis-cli: Interactive mode not supported" >&2
        return 1
    fi
    
    # Always reload state at the beginning to handle BATS subshells
    mock::redis::load_state
    
    # Execute the Redis command
    mock::redis::execute_command "${command_args[@]}"
    local result=$?
    
    # Save state after each command
    mock::redis::save_state
    
    return $result
}

# Execute a Redis command
mock::redis::execute_command() {
    local full_command="$*"
    local cmd="${1^^}"  # Uppercase first argument
    shift
    
    # Handle transaction mode
    if [[ "$REDIS_MOCK_TRANSACTION_MODE" == "active" ]] && [[ "$cmd" != "EXEC" ]] && [[ "$cmd" != "DISCARD" ]]; then
        REDIS_MOCK_TRANSACTION_QUEUE+=("$cmd $*")
        echo "QUEUED"
        return 0
    fi
    
    case "$cmd" in
        PING)
            mock::redis::cmd_ping "$@"
            ;;
        INFO)
            mock::redis::cmd_info "$@"
            ;;
        SET)
            mock::redis::cmd_set "$@"
            ;;
        GET)
            mock::redis::cmd_get "$@"
            ;;
        DEL)
            mock::redis::cmd_del "$@"
            ;;
        EXISTS)
            mock::redis::cmd_exists "$@"
            ;;
        EXPIRE)
            mock::redis::cmd_expire "$@"
            ;;
        TTL)
            mock::redis::cmd_ttl "$@"
            ;;
        KEYS)
            mock::redis::cmd_keys "$@"
            ;;
        FLUSHDB)
            mock::redis::cmd_flushdb "$@"
            ;;
        FLUSHALL)
            mock::redis::cmd_flushall "$@"
            ;;
        LPUSH)
            mock::redis::cmd_lpush "$@"
            ;;
        RPUSH)
            mock::redis::cmd_rpush "$@"
            ;;
        LPOP)
            mock::redis::cmd_lpop "$@"
            ;;
        RPOP)
            mock::redis::cmd_rpop "$@"
            ;;
        LLEN)
            mock::redis::cmd_llen "$@"
            ;;
        LRANGE)
            mock::redis::cmd_lrange "$@"
            ;;
        SADD)
            mock::redis::cmd_sadd "$@"
            ;;
        SREM)
            mock::redis::cmd_srem "$@"
            ;;
        SMEMBERS)
            mock::redis::cmd_smembers "$@"
            ;;
        SISMEMBER)
            mock::redis::cmd_sismember "$@"
            ;;
        HSET)
            mock::redis::cmd_hset "$@"
            ;;
        HGET)
            mock::redis::cmd_hget "$@"
            ;;
        HDEL)
            mock::redis::cmd_hdel "$@"
            ;;
        HGETALL)
            mock::redis::cmd_hgetall "$@"
            ;;
        PUBLISH)
            mock::redis::cmd_publish "$@"
            ;;
        SUBSCRIBE)
            mock::redis::cmd_subscribe "$@"
            ;;
        MULTI)
            mock::redis::cmd_multi "$@"
            ;;
        EXEC)
            mock::redis::cmd_exec "$@"
            ;;
        DISCARD)
            mock::redis::cmd_discard "$@"
            ;;
        CONFIG)
            mock::redis::cmd_config "$@"
            ;;
        *)
            echo "(error) ERR unknown command '$cmd'" >&2
            return 1
            ;;
    esac
}

# Redis command implementations
mock::redis::cmd_ping() {
    if [[ -n "$1" ]]; then
        echo "$1"
    else
        echo "PONG"
    fi
    return 0
}

mock::redis::cmd_info() {
    local section="${1:-}"
    
    if [[ -z "$section" ]] || [[ "$section" == "server" ]]; then
        echo "# Server"
        echo "redis_version:${REDIS_MOCK_CONFIG[version]}"
        echo "redis_mode:standalone"
        echo "tcp_port:${REDIS_MOCK_CONFIG[port]}"
        echo ""
    fi
    
    if [[ -z "$section" ]] || [[ "$section" == "clients" ]]; then
        echo "# Clients"
        echo "connected_clients:${REDIS_MOCK_CONFIG[connected_clients]}"
        echo ""
    fi
    
    if [[ -z "$section" ]] || [[ "$section" == "memory" ]]; then
        echo "# Memory"
        echo "used_memory:${REDIS_MOCK_CONFIG[used_memory]}"
        echo "used_memory_human:1M"
        echo ""
    fi
    
    if [[ -z "$section" ]] || [[ "$section" == "replication" ]]; then
        echo "# Replication"
        echo "role:${REDIS_MOCK_CONFIG[role]}"
        echo ""
    fi
    
    return 0
}

mock::redis::cmd_set() {
    if [[ $# -lt 2 ]]; then
        echo "(error) ERR wrong number of arguments for 'set' command" >&2
        return 1
    fi
    
    local key="$1"
    local value="$2"
    shift 2
    
    # Parse options first before storing
    local nx_mode=false
    local xx_mode=false
    local expire_seconds=""
    local expire_ms=""
    
    while [[ $# -gt 0 ]]; do
        case "${1^^}" in
            EX)
                expire_seconds="$2"
                shift 2
                ;;
            PX)
                expire_ms="$2"
                shift 2
                ;;
            NX)
                nx_mode=true
                shift
                ;;
            XX)
                xx_mode=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Check NX/XX conditions
    if [[ "$nx_mode" == "true" ]] && [[ -n "${REDIS_MOCK_DATA[$key]+x}" ]]; then
        echo "(nil)"
        return 0
    fi
    
    if [[ "$xx_mode" == "true" ]] && [[ -z "${REDIS_MOCK_DATA[$key]+x}" ]]; then
        echo "(nil)"
        return 0
    fi
    
    # Store the value
    REDIS_MOCK_DATA[$key]="$value"
    
    # Handle expiration
    if [[ -n "$expire_seconds" ]]; then
        local expire_time=$(($(date +%s) + expire_seconds))
        REDIS_MOCK_EXPIRES[$key]="$expire_time"
    elif [[ -n "$expire_ms" ]]; then
        local expire_time=$(($(date +%s) + (expire_ms / 1000)))
        REDIS_MOCK_EXPIRES[$key]="$expire_time"
    fi
    
    echo "OK"
    return 0
}

mock::redis::cmd_get() {
    local key="$1"
    
    if [[ -z "$key" ]]; then
        echo "(error) ERR wrong number of arguments for 'get' command" >&2
        return 1
    fi
    
    # Check expiration
    if [[ -n "${REDIS_MOCK_EXPIRES[$key]}" ]]; then
        local expire_time="${REDIS_MOCK_EXPIRES[$key]}"
        local current_time=$(date +%s)
        if [[ $current_time -gt $expire_time ]]; then
            unset REDIS_MOCK_DATA[$key]
            unset REDIS_MOCK_EXPIRES[$key]
        fi
    fi
    
    # Return value or nil
    if [[ -n "${REDIS_MOCK_DATA[$key]+x}" ]]; then
        echo "${REDIS_MOCK_DATA[$key]}"
    else
        echo "(nil)"
    fi
    return 0
}

mock::redis::cmd_del() {
    local count=0
    
    for key in "$@"; do
        local deleted=false
        
        if [[ -n "${REDIS_MOCK_DATA[$key]+x}" ]]; then
            unset REDIS_MOCK_DATA[$key]
            unset REDIS_MOCK_EXPIRES[$key]
            deleted=true
        fi
        if [[ -n "${REDIS_MOCK_LISTS[$key]+x}" ]]; then
            unset REDIS_MOCK_LISTS[$key]
            deleted=true
        fi
        if [[ -n "${REDIS_MOCK_SETS[$key]+x}" ]]; then
            unset REDIS_MOCK_SETS[$key]
            deleted=true
        fi
        if [[ -n "${REDIS_MOCK_HASHES[$key]+x}" ]]; then
            unset REDIS_MOCK_HASHES[$key]
            deleted=true
        fi
        
        if [[ "$deleted" == "true" ]]; then
            ((count++)) || true
        fi
    done
    
    echo "(integer) $count"
    return 0
}

mock::redis::cmd_exists() {
    local count=0
    
    for key in "$@"; do
        if [[ -n "${REDIS_MOCK_DATA[$key]+x}" ]] || \
           [[ -n "${REDIS_MOCK_LISTS[$key]+x}" ]] || \
           [[ -n "${REDIS_MOCK_SETS[$key]+x}" ]] || \
           [[ -n "${REDIS_MOCK_HASHES[$key]+x}" ]]; then
            ((count++)) || true
        fi
    done
    
    echo "(integer) $count"
    return 0
}

mock::redis::cmd_expire() {
    local key="$1"
    local seconds="$2"
    
    if [[ -z "$key" ]] || [[ -z "$seconds" ]]; then
        echo "(error) ERR wrong number of arguments for 'expire' command" >&2
        return 1
    fi
    
    if [[ -n "${REDIS_MOCK_DATA[$key]}" ]]; then
        local expire_time=$(($(date +%s) + seconds))
        REDIS_MOCK_EXPIRES[$key]="$expire_time"
        echo "(integer) 1"
    else
        echo "(integer) 0"
    fi
    return 0
}

mock::redis::cmd_ttl() {
    local key="$1"
    
    if [[ -z "$key" ]]; then
        echo "(error) ERR wrong number of arguments for 'ttl' command" >&2
        return 1
    fi
    
    if [[ -z "${REDIS_MOCK_DATA[$key]}" ]]; then
        echo "(integer) -2"  # Key does not exist
    elif [[ -z "${REDIS_MOCK_EXPIRES[$key]}" ]]; then
        echo "(integer) -1"  # Key exists but has no expire
    else
        local expire_time="${REDIS_MOCK_EXPIRES[$key]}"
        local current_time=$(date +%s)
        local ttl=$((expire_time - current_time))
        if [[ $ttl -lt 0 ]]; then
            unset REDIS_MOCK_DATA[$key]
            unset REDIS_MOCK_EXPIRES[$key]
            echo "(integer) -2"
        else
            echo "(integer) $ttl"
        fi
    fi
    return 0
}

mock::redis::cmd_keys() {
    local pattern="${1:-*}"
    
    # Convert Redis pattern to bash pattern
    pattern="${pattern//\*/.*}"
    pattern="${pattern//\?/.}"
    
    local keys=()
    for key in "${!REDIS_MOCK_DATA[@]}" "${!REDIS_MOCK_LISTS[@]}" "${!REDIS_MOCK_SETS[@]}" "${!REDIS_MOCK_HASHES[@]}"; do
        if [[ "$key" =~ ^${pattern}$ ]]; then
            keys+=("$key")
        fi
    done
    
    # Remove duplicates
    local unique_keys=($(printf "%s\n" "${keys[@]}" | sort -u))
    
    if [[ ${#unique_keys[@]} -eq 0 ]]; then
        echo "(empty array)"
    else
        for i in "${!unique_keys[@]}"; do
            echo "$((i + 1))) \"${unique_keys[$i]}\""
        done
    fi
    return 0
}

mock::redis::cmd_flushdb() {
    REDIS_MOCK_DATA=()
    REDIS_MOCK_EXPIRES=()
    REDIS_MOCK_LISTS=()
    REDIS_MOCK_SETS=()
    REDIS_MOCK_HASHES=()
    echo "OK"
    return 0
}

mock::redis::cmd_flushall() {
    mock::redis::cmd_flushdb
    return 0
}

# List operations
mock::redis::cmd_lpush() {
    local key="$1"
    shift
    
    if [[ -z "$key" ]] || [[ $# -eq 0 ]]; then
        echo "(error) ERR wrong number of arguments for 'lpush' command" >&2
        return 1
    fi
    
    # Initialize list if not exists
    [[ -z "${REDIS_MOCK_LISTS[$key]}" ]] && REDIS_MOCK_LISTS[$key]=""
    
    # Add elements to the beginning
    for value in "$@"; do
        if [[ -z "${REDIS_MOCK_LISTS[$key]}" ]]; then
            REDIS_MOCK_LISTS[$key]="$value"
        else
            REDIS_MOCK_LISTS[$key]="$value|${REDIS_MOCK_LISTS[$key]}"
        fi
    done
    
    # Return list length
    local IFS='|'
    local list_array=(${REDIS_MOCK_LISTS[$key]})
    echo "(integer) ${#list_array[@]}"
    return 0
}

mock::redis::cmd_rpush() {
    local key="$1"
    shift
    
    if [[ -z "$key" ]] || [[ $# -eq 0 ]]; then
        echo "(error) ERR wrong number of arguments for 'rpush' command" >&2
        return 1
    fi
    
    # Initialize list if not exists
    [[ -z "${REDIS_MOCK_LISTS[$key]}" ]] && REDIS_MOCK_LISTS[$key]=""
    
    # Add elements to the end
    for value in "$@"; do
        if [[ -z "${REDIS_MOCK_LISTS[$key]}" ]]; then
            REDIS_MOCK_LISTS[$key]="$value"
        else
            REDIS_MOCK_LISTS[$key]="${REDIS_MOCK_LISTS[$key]}|$value"
        fi
    done
    
    # Return list length
    local IFS='|'
    local list_array=(${REDIS_MOCK_LISTS[$key]})
    echo "(integer) ${#list_array[@]}"
    return 0
}

mock::redis::cmd_lpop() {
    local key="$1"
    
    if [[ -z "$key" ]]; then
        echo "(error) ERR wrong number of arguments for 'lpop' command" >&2
        return 1
    fi
    
    if [[ -z "${REDIS_MOCK_LISTS[$key]}" ]]; then
        echo "(nil)"
        return 0
    fi
    
    local IFS='|'
    local list_array=(${REDIS_MOCK_LISTS[$key]})
    
    if [[ ${#list_array[@]} -eq 0 ]]; then
        echo "(nil)"
        return 0
    fi
    
    # Pop first element
    local value="${list_array[0]}"
    
    # Rebuild list without first element
    if [[ ${#list_array[@]} -eq 1 ]]; then
        unset REDIS_MOCK_LISTS[$key]
    else
        REDIS_MOCK_LISTS[$key]=$(IFS='|'; echo "${list_array[*]:1}")
    fi
    
    echo "$value"
    return 0
}

mock::redis::cmd_rpop() {
    local key="$1"
    
    if [[ -z "$key" ]]; then
        echo "(error) ERR wrong number of arguments for 'rpop' command" >&2
        return 1
    fi
    
    if [[ -z "${REDIS_MOCK_LISTS[$key]}" ]]; then
        echo "(nil)"
        return 0
    fi
    
    local IFS='|'
    local list_array=(${REDIS_MOCK_LISTS[$key]})
    
    if [[ ${#list_array[@]} -eq 0 ]]; then
        echo "(nil)"
        return 0
    fi
    
    # Pop last element
    local value="${list_array[-1]}"
    
    # Rebuild list without last element
    if [[ ${#list_array[@]} -eq 1 ]]; then
        unset REDIS_MOCK_LISTS[$key]
    else
        REDIS_MOCK_LISTS[$key]=$(IFS='|'; echo "${list_array[*]:0:${#list_array[@]}-1}")
    fi
    
    echo "$value"
    return 0
}

mock::redis::cmd_llen() {
    local key="$1"
    
    if [[ -z "$key" ]]; then
        echo "(error) ERR wrong number of arguments for 'llen' command" >&2
        return 1
    fi
    
    if [[ -z "${REDIS_MOCK_LISTS[$key]}" ]]; then
        echo "(integer) 0"
        return 0
    fi
    
    local IFS='|'
    local list_array=(${REDIS_MOCK_LISTS[$key]})
    echo "(integer) ${#list_array[@]}"
    return 0
}

mock::redis::cmd_lrange() {
    local key="$1"
    local start="$2"
    local stop="$3"
    
    if [[ -z "$key" ]] || [[ -z "$start" ]] || [[ -z "$stop" ]]; then
        echo "(error) ERR wrong number of arguments for 'lrange' command" >&2
        return 1
    fi
    
    if [[ -z "${REDIS_MOCK_LISTS[$key]}" ]]; then
        echo "(empty array)"
        return 0
    fi
    
    local IFS='|'
    local list_array=(${REDIS_MOCK_LISTS[$key]})
    local len=${#list_array[@]}
    
    # Handle negative indices
    [[ $start -lt 0 ]] && start=$((len + start))
    [[ $stop -lt 0 ]] && stop=$((len + stop))
    
    # Bound checking
    [[ $start -lt 0 ]] && start=0
    [[ $stop -ge $len ]] && stop=$((len - 1))
    
    if [[ $start -gt $stop ]]; then
        echo "(empty array)"
        return 0
    fi
    
    local i
    for ((i = start; i <= stop; i++)); do
        echo "$((i - start + 1))) \"${list_array[$i]}\""
    done
    return 0
}

# Set operations
mock::redis::cmd_sadd() {
    local key="$1"
    shift
    
    if [[ -z "$key" ]] || [[ $# -eq 0 ]]; then
        echo "(error) ERR wrong number of arguments for 'sadd' command" >&2
        return 1
    fi
    
    # Initialize set if not exists
    if [[ -z "${REDIS_MOCK_SETS[$key]+x}" ]]; then
        REDIS_MOCK_SETS[$key]=""
    fi
    
    local count=0
    
    for value in "$@"; do
        # Check if value already exists in set
        local found=false
        if [[ -n "${REDIS_MOCK_SETS[$key]}" ]]; then
            local old_IFS="$IFS"
            IFS='|'
            local set_members=(${REDIS_MOCK_SETS[$key]})
            IFS="$old_IFS"
            
            for member in "${set_members[@]}"; do
                if [[ "$member" == "$value" ]]; then
                    found=true
                    break
                fi
            done
        fi
        
        if [[ "$found" == "false" ]]; then
            if [[ -z "${REDIS_MOCK_SETS[$key]}" ]]; then
                REDIS_MOCK_SETS[$key]="$value"
            else
                REDIS_MOCK_SETS[$key]="${REDIS_MOCK_SETS[$key]}|$value"
            fi
            ((count++)) || true
        fi
    done
    
    echo "(integer) $count"
    return 0
}

mock::redis::cmd_srem() {
    local key="$1"
    shift
    
    if [[ -z "$key" ]] || [[ $# -eq 0 ]]; then
        echo "(error) ERR wrong number of arguments for 'srem' command" >&2
        return 1
    fi
    
    if [[ -z "${REDIS_MOCK_SETS[$key]}" ]]; then
        echo "(integer) 0"
        return 0
    fi
    
    local count=0
    local IFS='|'
    local set_array=(${REDIS_MOCK_SETS[$key]})
    local new_set=()
    
    for member in "${set_array[@]}"; do
        local remove=0
        for value in "$@"; do
            if [[ "$member" == "$value" ]]; then
                remove=1
                ((count++)) || true
                break
            fi
        done
        
        if [[ $remove -eq 0 ]]; then
            new_set+=("$member")
        fi
    done
    
    if [[ ${#new_set[@]} -eq 0 ]]; then
        unset REDIS_MOCK_SETS[$key]
    else
        REDIS_MOCK_SETS[$key]=$(IFS='|'; echo "${new_set[*]}")
    fi
    
    echo "(integer) $count"
    return 0
}

mock::redis::cmd_smembers() {
    local key="$1"
    
    if [[ -z "$key" ]]; then
        echo "(error) ERR wrong number of arguments for 'smembers' command" >&2
        return 1
    fi
    
    if [[ -z "${REDIS_MOCK_SETS[$key]}" ]]; then
        echo "(empty array)"
        return 0
    fi
    
    local IFS='|'
    local set_array=(${REDIS_MOCK_SETS[$key]})
    
    for i in "${!set_array[@]}"; do
        echo "$((i + 1))) \"${set_array[$i]}\""
    done
    return 0
}

mock::redis::cmd_sismember() {
    local key="$1"
    local member="$2"
    
    if [[ -z "$key" ]] || [[ -z "$member" ]]; then
        echo "(error) ERR wrong number of arguments for 'sismember' command" >&2
        return 1
    fi
    
    if [[ -z "${REDIS_MOCK_SETS[$key]}" ]]; then
        echo "(integer) 0"
        return 0
    fi
    
    local IFS='|'
    local set_array=(${REDIS_MOCK_SETS[$key]})
    
    for value in "${set_array[@]}"; do
        if [[ "$value" == "$member" ]]; then
            echo "(integer) 1"
            return 0
        fi
    done
    
    echo "(integer) 0"
    return 0
}

# Hash operations
mock::redis::cmd_hset() {
    local key="$1"
    local field="$2"
    local value="$3"
    
    if [[ -z "$key" ]] || [[ -z "$field" ]] || [[ -z "$value" ]]; then
        echo "(error) ERR wrong number of arguments for 'hset' command" >&2
        return 1
    fi
    
    # Initialize hash if not exists
    if [[ -z "${REDIS_MOCK_HASHES[$key]+x}" ]]; then
        REDIS_MOCK_HASHES[$key]=""
    fi
    
    # Store fields in format field1|value1|field2|value2|...
    local current="${REDIS_MOCK_HASHES[$key]}"
    local new_hash=""
    local found=false
    
    if [[ -n "$current" ]]; then
        # Parse existing fields
        local old_IFS="$IFS"
        IFS='|'
        local fields=($current)
        IFS="$old_IFS"
        
        for ((i=0; i<${#fields[@]}; i+=2)); do
            if [[ "${fields[i]}" == "$field" ]]; then
                fields[i+1]="$value"
                found=true
            fi
            if [[ -n "$new_hash" ]]; then
                new_hash="${new_hash}|${fields[i]}|${fields[i+1]}"
            else
                new_hash="${fields[i]}|${fields[i+1]}"
            fi
        done
    fi
    
    if [[ "$found" == "false" ]]; then
        if [[ -n "$new_hash" ]]; then
            new_hash="${new_hash}|${field}|${value}"
        else
            new_hash="${field}|${value}"
        fi
    fi
    
    REDIS_MOCK_HASHES[$key]="$new_hash"
    
    echo "(integer) 1"
    return 0
}

mock::redis::cmd_hget() {
    local key="$1"
    local field="$2"
    
    if [[ -z "$key" ]] || [[ -z "$field" ]]; then
        echo "(error) ERR wrong number of arguments for 'hget' command" >&2
        return 1
    fi
    
    if [[ -z "${REDIS_MOCK_HASHES[$key]+x}" ]]; then
        echo "(nil)"
        return 0
    fi
    
    # Parse fields in format field1|value1|field2|value2|...
    local current="${REDIS_MOCK_HASHES[$key]}"
    local old_IFS="$IFS"
    IFS='|'
    local fields=($current)
    IFS="$old_IFS"
    
    for ((i=0; i<${#fields[@]}; i+=2)); do
        if [[ "${fields[i]}" == "$field" ]]; then
            echo "${fields[i+1]}"
            return 0
        fi
    done
    
    echo "(nil)"
    return 0
}

mock::redis::cmd_hdel() {
    local key="$1"
    shift
    
    if [[ -z "$key" ]] || [[ $# -eq 0 ]]; then
        echo "(error) ERR wrong number of arguments for 'hdel' command" >&2
        return 1
    fi
    
    if [[ -z "${REDIS_MOCK_HASHES[$key]+x}" ]]; then
        echo "(integer) 0"
        return 0
    fi
    
    # Parse and rebuild hash without deleted fields
    local current="${REDIS_MOCK_HASHES[$key]}"
    local old_IFS="$IFS"
    IFS='|'
    local fields=($current)
    IFS="$old_IFS"
    
    local new_hash=""
    local count=0
    
    for ((i=0; i<${#fields[@]}; i+=2)); do
        local field_found=false
        for del_field in "$@"; do
            if [[ "${fields[i]}" == "$del_field" ]]; then
                field_found=true
                ((count++)) || true
                break
            fi
        done
        
        if [[ "$field_found" == "false" ]]; then
            if [[ -n "$new_hash" ]]; then
                new_hash="${new_hash}|${fields[i]}|${fields[i+1]}"
            else
                new_hash="${fields[i]}|${fields[i+1]}"
            fi
        fi
    done
    
    if [[ -z "$new_hash" ]]; then
        unset REDIS_MOCK_HASHES[$key]
    else
        REDIS_MOCK_HASHES[$key]="$new_hash"
    fi
    
    echo "(integer) $count"
    return 0
}

mock::redis::cmd_hgetall() {
    local key="$1"
    
    if [[ -z "$key" ]]; then
        echo "(error) ERR wrong number of arguments for 'hgetall' command" >&2
        return 1
    fi
    
    if [[ -z "${REDIS_MOCK_HASHES[$key]+x}" ]]; then
        echo "(empty array)"
        return 0
    fi
    
    # Parse fields in format field1|value1|field2|value2|...
    local current="${REDIS_MOCK_HASHES[$key]}"
    if [[ -z "$current" ]]; then
        echo "(empty array)"
        return 0
    fi
    
    local old_IFS="$IFS"
    IFS='|'
    local fields=($current)
    IFS="$old_IFS"
    
    local index=1
    for ((i=0; i<${#fields[@]}; i+=2)); do
        echo "$index) \"${fields[i]}\""
        ((index++))
        echo "$index) \"${fields[i+1]}\""
        ((index++))
    done
    
    return 0
}

# Pub/Sub operations
mock::redis::cmd_publish() {
    local channel="$1"
    local message="$2"
    
    if [[ -z "$channel" ]] || [[ -z "$message" ]]; then
        echo "(error) ERR wrong number of arguments for 'publish' command" >&2
        return 1
    fi
    
    # In a real implementation, this would notify subscribers
    # For mock, we just return the number of subscribers (always 0)
    echo "(integer) 0"
    return 0
}

mock::redis::cmd_subscribe() {
    # Subscribe command blocks in real Redis
    # For mock, we just return a subscription confirmation
    echo "Reading messages... (press Ctrl-C to quit)"
    echo "1) \"subscribe\""
    echo "2) \"$1\""
    echo "3) (integer) 1"
    
    # In tests, this should be interrupted
    return 0
}

# Transaction operations
mock::redis::cmd_multi() {
    REDIS_MOCK_TRANSACTION_MODE="active"
    REDIS_MOCK_TRANSACTION_QUEUE=()
    echo "OK"
    return 0
}

mock::redis::cmd_exec() {
    if [[ "$REDIS_MOCK_TRANSACTION_MODE" != "active" ]]; then
        echo "(error) ERR EXEC without MULTI" >&2
        return 1
    fi
    
    REDIS_MOCK_TRANSACTION_MODE=""
    
    # Execute queued commands
    local results=()
    local temp_file
    temp_file=$(mktemp)
    
    for cmd in "${REDIS_MOCK_TRANSACTION_QUEUE[@]}"; do
        # Execute command in current shell and capture output
        mock::redis::execute_command $cmd > "$temp_file" 2>&1
        local result=$(<"$temp_file")
        results+=("$result")
    done
    
    rm -f "$temp_file"
    
    REDIS_MOCK_TRANSACTION_QUEUE=()
    
    # Return results
    if [[ ${#results[@]} -eq 0 ]]; then
        echo "(empty array)"
    else
        for i in "${!results[@]}"; do
            echo "$((i + 1))) ${results[$i]}"
        done
    fi
    return 0
}

mock::redis::cmd_discard() {
    if [[ "$REDIS_MOCK_TRANSACTION_MODE" != "active" ]]; then
        echo "(error) ERR DISCARD without MULTI" >&2
        return 1
    fi
    
    REDIS_MOCK_TRANSACTION_MODE=""
    REDIS_MOCK_TRANSACTION_QUEUE=()
    echo "OK"
    return 0
}

# Config operations
mock::redis::cmd_config() {
    local subcommand="${1^^}"
    shift
    
    case "$subcommand" in
        GET)
            local param="$1"
            case "$param" in
                maxmemory)
                    echo "1) \"maxmemory\""
                    echo "2) \"0\""
                    ;;
                *)
                    echo "(empty array)"
                    ;;
            esac
            ;;
        SET)
            echo "OK"
            ;;
        *)
            echo "(error) ERR Unknown CONFIG subcommand" >&2
            return 1
            ;;
    esac
    return 0
}

# Test helper functions
mock::redis::reset() {
    # Optional parameter to control whether to save state after reset
    local save_state="${1:-true}"
    
    # Clear all data
    REDIS_MOCK_DATA=()
    REDIS_MOCK_EXPIRES=()
    REDIS_MOCK_LISTS=()
    REDIS_MOCK_SETS=()
    REDIS_MOCK_HASHES=()
    REDIS_MOCK_PUBSUB=()
    
    # Reset configuration
    REDIS_MOCK_CONFIG=(
        [host]="localhost"
        [port]="6379"
        [password]=""
        [db]="0"
        [connected]="true"
        [version]="7.0.11"
        [role]="master"
        [used_memory]="1048576"
        [connected_clients]="1"
        [error_mode]=""
    )
    
    # Clear transaction state
    REDIS_MOCK_TRANSACTION_MODE=""
    REDIS_MOCK_TRANSACTION_QUEUE=()
    
    # Save the reset state to file if requested (default: true)
    # This ensures subsequent redis-cli calls in subshells get clean state
    if [[ "$save_state" == "true" ]]; then
        mock::redis::save_state
    fi
    
    mock::log_state "redis" "Redis mock reset to initial state"
}

mock::redis::set_error() {
    local error_mode="$1"
    REDIS_MOCK_CONFIG[error_mode]="$error_mode"
    mock::redis::save_state
    mock::log_state "redis" "Set Redis error mode: $error_mode"
}

mock::redis::set_connected() {
    local connected="$1"
    REDIS_MOCK_CONFIG[connected]="$connected"
    mock::redis::save_state
    mock::log_state "redis" "Set Redis connected: $connected"
}

mock::redis::set_config() {
    local key="$1"
    local value="$2"
    REDIS_MOCK_CONFIG[$key]="$value"
    mock::redis::save_state
    mock::log_state "redis" "Set Redis config: $key=$value"
}

# Test assertions
mock::redis::assert_key_exists() {
    local key="$1"
    if [[ -n "${REDIS_MOCK_DATA[$key]}" ]] || \
       [[ -n "${REDIS_MOCK_LISTS[$key]}" ]] || \
       [[ -n "${REDIS_MOCK_SETS[$key]}" ]] || \
       [[ -n "${REDIS_MOCK_HASHES[$key]}" ]]; then
        return 0
    else
        echo "Assertion failed: Key '$key' does not exist" >&2
        return 1
    fi
}

mock::redis::assert_key_value() {
    local key="$1"
    local expected_value="$2"
    local actual_value="${REDIS_MOCK_DATA[$key]}"
    
    if [[ "$actual_value" == "$expected_value" ]]; then
        return 0
    else
        echo "Assertion failed: Key '$key' value mismatch" >&2
        echo "  Expected: '$expected_value'" >&2
        echo "  Actual: '$actual_value'" >&2
        return 1
    fi
}

mock::redis::assert_list_length() {
    local key="$1"
    local expected_length="$2"
    
    if [[ -z "${REDIS_MOCK_LISTS[$key]}" ]]; then
        [[ "$expected_length" -eq 0 ]] && return 0
        echo "Assertion failed: List '$key' does not exist" >&2
        return 1
    fi
    
    local IFS='|'
    local list_array=(${REDIS_MOCK_LISTS[$key]})
    local actual_length=${#list_array[@]}
    
    if [[ "$actual_length" -eq "$expected_length" ]]; then
        return 0
    else
        echo "Assertion failed: List '$key' length mismatch" >&2
        echo "  Expected: $expected_length" >&2
        echo "  Actual: $actual_length" >&2
        return 1
    fi
}

mock::redis::assert_set_member() {
    local key="$1"
    local member="$2"
    
    if [[ -z "${REDIS_MOCK_SETS[$key]}" ]]; then
        echo "Assertion failed: Set '$key' does not exist" >&2
        return 1
    fi
    
    local IFS='|'
    local set_array=(${REDIS_MOCK_SETS[$key]})
    
    for value in "${set_array[@]}"; do
        if [[ "$value" == "$member" ]]; then
            return 0
        fi
    done
    
    echo "Assertion failed: Member '$member' not in set '$key'" >&2
    return 1
}

# Debug functions
mock::redis::dump_state() {
    echo "=== Redis Mock State ==="
    echo "Configuration:"
    for key in "${!REDIS_MOCK_CONFIG[@]}"; do
        echo "  $key: ${REDIS_MOCK_CONFIG[$key]}"
    done
    
    echo "Data:"
    for key in "${!REDIS_MOCK_DATA[@]}"; do
        echo "  $key: ${REDIS_MOCK_DATA[$key]}"
    done
    
    echo "Lists:"
    for key in "${!REDIS_MOCK_LISTS[@]}"; do
        echo "  $key: ${REDIS_MOCK_LISTS[$key]}"
    done
    
    echo "Sets:"
    for key in "${!REDIS_MOCK_SETS[@]}"; do
        echo "  $key: ${REDIS_MOCK_SETS[$key]}"
    done
    
    echo "Hashes:"
    for key in "${!REDIS_MOCK_HASHES[@]}"; do
        echo "  $key: ${REDIS_MOCK_HASHES[$key]}"
    done
    
    echo "Transaction:"
    echo "  Mode: $REDIS_MOCK_TRANSACTION_MODE"
    echo "  Queue: ${#REDIS_MOCK_TRANSACTION_QUEUE[@]} commands"
    echo "======================="
}

# Export all functions
export -f redis-cli
export -f mock::redis::save_state
export -f mock::redis::load_state
export -f mock::redis::execute_command
export -f mock::redis::cmd_ping
export -f mock::redis::cmd_info
export -f mock::redis::cmd_set
export -f mock::redis::cmd_get
export -f mock::redis::cmd_del
export -f mock::redis::cmd_exists
export -f mock::redis::cmd_expire
export -f mock::redis::cmd_ttl
export -f mock::redis::cmd_keys
export -f mock::redis::cmd_flushdb
export -f mock::redis::cmd_flushall
export -f mock::redis::cmd_lpush
export -f mock::redis::cmd_rpush
export -f mock::redis::cmd_lpop
export -f mock::redis::cmd_rpop
export -f mock::redis::cmd_llen
export -f mock::redis::cmd_lrange
export -f mock::redis::cmd_sadd
export -f mock::redis::cmd_srem
export -f mock::redis::cmd_smembers
export -f mock::redis::cmd_sismember
export -f mock::redis::cmd_hset
export -f mock::redis::cmd_hget
export -f mock::redis::cmd_hdel
export -f mock::redis::cmd_hgetall
export -f mock::redis::cmd_publish
export -f mock::redis::cmd_subscribe
export -f mock::redis::cmd_multi
export -f mock::redis::cmd_exec
export -f mock::redis::cmd_discard
export -f mock::redis::cmd_config
export -f mock::redis::reset
export -f mock::redis::set_error
export -f mock::redis::set_connected
export -f mock::redis::set_config
export -f mock::redis::assert_key_exists
export -f mock::redis::assert_key_value
export -f mock::redis::assert_list_length
export -f mock::redis::assert_set_member
export -f mock::redis::dump_state

# Save initial state
mock::redis::save_state