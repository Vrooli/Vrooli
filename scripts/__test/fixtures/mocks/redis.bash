#!/usr/bin/env bash
# Redis Resource Mock Implementation
# Provides realistic mock responses for Redis key-value store service

# Prevent duplicate loading
if [[ "${REDIS_MOCK_LOADED:-}" == "true" ]]; then
    return 0
fi
export REDIS_MOCK_LOADED="true"

#######################################
# Setup Redis mock environment
# Arguments: $1 - state (healthy, unhealthy, installing, stopped)
#######################################
mock::redis::setup() {
    local state="${1:-healthy}"
    
    # Configure Redis-specific environment
    export REDIS_PORT="${REDIS_PORT:-6379}"
    export REDIS_HOST="${REDIS_HOST:-localhost}"
    export REDIS_CONTAINER_NAME="${TEST_NAMESPACE}_redis"
    export REDIS_PASSWORD="${REDIS_PASSWORD:-}"
    export REDIS_URL="redis://${REDIS_HOST}:${REDIS_PORT}"
    
    # Set up Docker mock state
    mock::docker::set_container_state "$REDIS_CONTAINER_NAME" "$state"
    
    # Configure redis-cli mock based on state
    case "$state" in
        "healthy")
            mock::redis::setup_healthy_state
            ;;
        "unhealthy")
            mock::redis::setup_unhealthy_state
            ;;
        "installing")
            mock::redis::setup_installing_state
            ;;
        "stopped")
            mock::redis::setup_stopped_state
            ;;
        *)
            echo "[REDIS_MOCK] Unknown state: $state" >&2
            return 1
            ;;
    esac
    
    echo "[REDIS_MOCK] Redis mock configured with state: $state"
}

#######################################
# Setup healthy Redis state
#######################################
mock::redis::setup_healthy_state() {
    # Set container logs
    local log_file="${MOCK_RESPONSES_DIR}/redis_logs.txt"
    cat > "$log_file" << 'EOF'
1:C 15 Jan 2024 10:00:00.000 # oO0OoO0OoO0Oo Redis is starting oO0OoO0OoO0Oo
1:C 15 Jan 2024 10:00:00.000 # Redis version=7.2.3, bits=64, commit=00000000, modified=0, pid=1, just started
1:M 15 Jan 2024 10:00:00.001 * Ready to accept connections
EOF
}

#######################################
# Setup unhealthy Redis state
#######################################
mock::redis::setup_unhealthy_state() {
    # Set error logs
    local log_file="${MOCK_RESPONSES_DIR}/redis_logs.txt"
    cat > "$log_file" << 'EOF'
1:M 15 Jan 2024 10:00:00.000 # Warning: Could not create server TCP listening socket
1:M 15 Jan 2024 10:00:00.000 # Fatal error, can't open config file
EOF
}

#######################################
# Setup installing Redis state
#######################################
mock::redis::setup_installing_state() {
    # Set initialization logs
    local log_file="${MOCK_RESPONSES_DIR}/redis_logs.txt"
    cat > "$log_file" << 'EOF'
1:C 15 Jan 2024 10:00:00.000 # oO0OoO0OoO0Oo Redis is starting oO0OoO0OoO0Oo
1:C 15 Jan 2024 10:00:00.000 # Loading RDB snapshot...
1:C 15 Jan 2024 10:00:00.000 # Progress: 45%
EOF
}

#######################################
# Setup stopped Redis state
#######################################
mock::redis::setup_stopped_state() {
    # No logs when stopped
    local log_file="${MOCK_RESPONSES_DIR}/redis_logs.txt"
    echo "" > "$log_file"
}

#######################################
# Mock redis-cli command
#######################################
redis-cli() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "redis-cli $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    # Parse arguments
    local host="$REDIS_HOST"
    local port="$REDIS_PORT"
    local password="$REDIS_PASSWORD"
    local command=""
    local remaining_args=()
    
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
            --scan)
                command="SCAN"
                shift
                ;;
            *)
                remaining_args+=("$1")
                shift
                ;;
        esac
    done
    
    # Check container state
    local container_state=$(docker inspect "$REDIS_CONTAINER_NAME" --format '{{.State.Status}}' 2>/dev/null || echo "stopped")
    
    if [[ "$container_state" != "running" ]]; then
        echo "Could not connect to Redis at $host:$port: Connection refused" >&2
        return 1
    fi
    
    # If no command specified, check remaining args
    if [[ -z "$command" && ${#remaining_args[@]} -gt 0 ]]; then
        command="${remaining_args[0]}"
        remaining_args=("${remaining_args[@]:1}")
    fi
    
    # Handle different Redis commands
    case "$command" in
        "PING")
            echo "PONG"
            ;;
        "INFO"|"info")
            echo "# Server"
            echo "redis_version:7.2.3"
            echo "redis_git_sha1:00000000"
            echo "redis_mode:standalone"
            echo "os:Linux 5.15.0-1 x86_64"
            echo "arch_bits:64"
            echo "tcp_port:$port"
            echo "uptime_in_seconds:3600"
            echo ""
            echo "# Clients"
            echo "connected_clients:1"
            echo ""
            echo "# Memory"
            echo "used_memory:1048576"
            echo "used_memory_human:1M"
            echo ""
            echo "# Stats"
            echo "total_connections_received:100"
            echo "total_commands_processed:1000"
            ;;
        "SET"|"set")
            local key="${remaining_args[0]}"
            local value="${remaining_args[1]}"
            if [[ -n "$key" && -n "$value" ]]; then
                echo "OK"
            else
                echo "(error) ERR wrong number of arguments for 'set' command" >&2
                return 1
            fi
            ;;
        "GET"|"get")
            local key="${remaining_args[0]}"
            if [[ -n "$key" ]]; then
                # Return mock value based on key
                case "$key" in
                    "test:key")
                        echo "test:value"
                        ;;
                    "counter")
                        echo "42"
                        ;;
                    *)
                        echo "(nil)"
                        ;;
                esac
            else
                echo "(error) ERR wrong number of arguments for 'get' command" >&2
                return 1
            fi
            ;;
        "KEYS"|"keys")
            local pattern="${remaining_args[0]:-*}"
            if [[ "$pattern" == "*" ]]; then
                echo "1) \"test:key\""
                echo "2) \"counter\""
                echo "3) \"session:abc123\""
            else
                echo "(empty array)"
            fi
            ;;
        "SCAN"|"scan")
            echo "1) \"0\""
            echo "2) 1) \"test:key\""
            echo "   2) \"counter\""
            echo "   3) \"session:abc123\""
            ;;
        "DBSIZE"|"dbsize")
            echo "(integer) 3"
            ;;
        "FLUSHDB"|"flushdb")
            echo "OK"
            ;;
        "CONFIG"|"config")
            local subcommand="${remaining_args[0]}"
            case "$subcommand" in
                "GET"|"get")
                    local param="${remaining_args[1]}"
                    echo "1) \"$param\""
                    echo "2) \"default\""
                    ;;
                *)
                    echo "OK"
                    ;;
            esac
            ;;
        *)
            # Unknown command
            echo "(error) ERR unknown command '$command'" >&2
            return 1
            ;;
    esac
    
    return 0
}

#######################################
# Mock Redis-specific operations
#######################################

# Mock Redis pub/sub
mock::redis::publish() {
    local channel="$1"
    local message="$2"
    
    echo "(integer) 1"  # Number of clients that received the message
}

# Mock Redis transactions
mock::redis::multi_exec() {
    echo "OK"  # MULTI
    echo "QUEUED"  # Commands
    echo "QUEUED"
    echo "1) OK"  # EXEC results
    echo "2) (integer) 1"
}

# Mock Redis cluster info
mock::redis::cluster_info() {
    echo "cluster_state:ok"
    echo "cluster_slots_assigned:16384"
    echo "cluster_slots_ok:16384"
    echo "cluster_slots_pfail:0"
    echo "cluster_slots_fail:0"
    echo "cluster_known_nodes:1"
    echo "cluster_size:1"
}

# Mock Redis backup
mock::redis::bgsave() {
    echo "Background saving started"
}

#######################################
# Export mock functions
#######################################
export -f redis-cli
export -f mock::redis::setup
export -f mock::redis::setup_healthy_state
export -f mock::redis::setup_unhealthy_state
export -f mock::redis::setup_installing_state
export -f mock::redis::setup_stopped_state
export -f mock::redis::publish
export -f mock::redis::multi_exec
export -f mock::redis::cluster_info
export -f mock::redis::bgsave

echo "[REDIS_MOCK] Redis mock implementation loaded"