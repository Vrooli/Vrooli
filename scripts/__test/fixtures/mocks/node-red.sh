#!/usr/bin/env bash
# Node-RED Mock Implementation
# 
# Provides a comprehensive mock for Node-RED automation platform including:
# - Docker container management simulation
# - Node-RED Admin API endpoints
# - Flow management operations
# - Node installation/management
# - Health checking and monitoring
# - Debug and testing utilities
#
# This mock follows the same standards as redis.sh and docker.sh mocks with:
# - Comprehensive state management
# - File-based persistence for BATS compatibility
# - Integration with centralized logging
# - Test helper functions
# - Support for all manage.sh actions

# Prevent duplicate loading
[[ -n "${NODE_RED_MOCK_LOADED:-}" ]] && return 0
declare -g NODE_RED_MOCK_LOADED=1

# Load dependencies
MOCK_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
[[ -f "$MOCK_DIR/logs.sh" ]] && source "$MOCK_DIR/logs.sh"
[[ -f "$MOCK_DIR/docker.sh" ]] && source "$MOCK_DIR/docker.sh"
[[ -f "$MOCK_DIR/http.sh" ]] && source "$MOCK_DIR/http.sh"

# Global configuration
declare -g NODE_RED_MOCK_STATE_DIR="${NODE_RED_MOCK_STATE_DIR:-/tmp/node-red-mock-state}"
declare -g NODE_RED_MOCK_DEBUG="${NODE_RED_MOCK_DEBUG:-}"

# Node-RED specific configuration
declare -gA NODE_RED_MOCK_CONFIG=(
    [container_name]="vrooli_node-red"
    [port]="1880"
    [host]="localhost"
    [base_url]="http://localhost:1880"
    [version]="3.1.0"
    [user]="admin"
    [password]=""
    [state]="running"
    [health]="healthy"
    [api_timeout]="30"
    [error_mode]=""
    [settings_loaded]="true"
    [flows_loaded]="true"
)

# Node-RED data storage
declare -gA NODE_RED_MOCK_FLOWS=()           # Flow definitions by ID
declare -gA NODE_RED_MOCK_NODES=()           # Node type definitions
declare -gA NODE_RED_MOCK_SETTINGS=()        # Settings data
declare -gA NODE_RED_MOCK_FLOWS_METADATA=()  # Flow metadata (name, disabled state, etc.)
declare -gA NODE_RED_MOCK_DEBUG_MESSAGES=()  # Debug message history
declare -gA NODE_RED_MOCK_FLOW_STATS=()      # Flow execution statistics
declare -gA NODE_RED_MOCK_INSTALLED_NODES=() # Custom installed nodes

# Initialize state directory
mkdir -p "$NODE_RED_MOCK_STATE_DIR"

# State persistence functions
mock::node_red::save_state() {
    local state_file="$NODE_RED_MOCK_STATE_DIR/node-red-state.sh"
    {
        echo "# Node-RED mock state - $(date)"
        
        # Save arrays using declare -p for proper restoration
        declare -p NODE_RED_MOCK_CONFIG 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA NODE_RED_MOCK_CONFIG=()"
        declare -p NODE_RED_MOCK_FLOWS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA NODE_RED_MOCK_FLOWS=()"
        declare -p NODE_RED_MOCK_NODES 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA NODE_RED_MOCK_NODES=()"
        declare -p NODE_RED_MOCK_SETTINGS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA NODE_RED_MOCK_SETTINGS=()"
        declare -p NODE_RED_MOCK_FLOWS_METADATA 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA NODE_RED_MOCK_FLOWS_METADATA=()"
        declare -p NODE_RED_MOCK_DEBUG_MESSAGES 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA NODE_RED_MOCK_DEBUG_MESSAGES=()"
        declare -p NODE_RED_MOCK_FLOW_STATS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA NODE_RED_MOCK_FLOW_STATS=()"
        declare -p NODE_RED_MOCK_INSTALLED_NODES 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA NODE_RED_MOCK_INSTALLED_NODES=()"
        
    } > "$state_file"
    
    command -v mock::log_state >/dev/null 2>&1 && mock::log_state "node-red" "Saved Node-RED state to $state_file" || true
}

mock::node_red::load_state() {
    local state_file="$NODE_RED_MOCK_STATE_DIR/node-red-state.sh"
    if [[ -f "$state_file" ]]; then
        source "$state_file"
        command -v mock::log_state >/dev/null 2>&1 && mock::log_state "node-red" "Loaded Node-RED state from $state_file" || true
    fi
}

# Automatically load state when sourced
mock::node_red::load_state

#######################################
# Docker Command Interceptor for Node-RED
#######################################

# Override docker commands specific to Node-RED container
docker() {
    command -v mock::log_and_verify >/dev/null 2>&1 && mock::log_and_verify "docker" "$@" || true
    
    # Always reload state at the beginning to handle BATS subshells
    mock::node_red::load_state
    
    local cmd="$1"
    local result=0
    
    case "$cmd" in
        "ps")
            mock::node_red::docker_ps "$@"
            result=$?
            ;;
        "container")
            shift
            mock::node_red::docker_container "$@"
            result=$?
            ;;
        "run")
            shift
            mock::node_red::docker_run "$@"
            result=$?
            ;;
        "stop"|"start"|"restart")
            shift
            mock::node_red::docker_lifecycle "$cmd" "$@"
            result=$?
            ;;
        "logs")
            shift
            mock::node_red::docker_logs "$@"
            result=$?
            ;;
        "exec")
            shift
            mock::node_red::docker_exec "$@"
            result=$?
            ;;
        "inspect")
            shift
            mock::node_red::docker_inspect "$@"
            result=$?
            ;;
        "pull")
            shift
            mock::node_red::docker_pull "$@"
            result=$?
            ;;
        *)
            # For commands we don't specifically mock, use the docker mock if available
            if declare -f mock::docker::handle_command &>/dev/null; then
                mock::docker::handle_command "$cmd" "$@"
                result=$?
            else
                echo "Mock docker: Unsupported command: $cmd" >&2
                result=1
            fi
            ;;
    esac
    
    # Save state after each docker command
    mock::node_red::save_state
    
    return $result
}

#######################################
# curl Command Interceptor for Node-RED API
#######################################

curl() {
    command -v mock::log_and_verify >/dev/null 2>&1 && mock::log_and_verify "curl" "$@" || true
    
    # Always reload state to handle BATS subshells
    mock::node_red::load_state
    
    local url=""
    local method="GET"
    local headers=()
    local data=""
    local output_file=""
    local silent=false
    local fail_fast=false
    local max_time=""
    local args=()
    
    # Parse curl arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -X|--request)
                method="$2"
                shift 2
                ;;
            -H|--header)
                headers+=("$2")
                shift 2
                ;;
            -d|--data)
                data="$2"
                # Handle @file syntax by reading file if it starts with @
                if [[ "$data" == @* ]]; then
                    local file_path="${data#@}"
                    if [[ -f "$file_path" ]]; then
                        data="$(<"$file_path")"
                    else
                        echo "curl: can't open '$file_path'!" >&2
                        return 2
                    fi
                fi
                shift 2
                ;;
            -o|--output)
                output_file="$2"
                shift 2
                ;;
            -s|--silent)
                silent=true
                shift
                ;;
            -f|--fail)
                fail_fast=true
                shift
                ;;
            --max-time)
                max_time="$2"
                shift 2
                ;;
            http*)
                url="$1"
                shift
                ;;
            *)
                args+=("$1")
                shift
                ;;
        esac
    done
    
    # Check if this is a Node-RED API call
    if [[ "$url" =~ ^http://localhost:${NODE_RED_MOCK_CONFIG[port]} ]]; then
        local result
        result=$(mock::node_red::handle_api_call "$method" "$url" "$data" "${headers[@]}")
        local exit_code=$?
        
        if [[ -n "$output_file" ]]; then
            echo "$result" > "$output_file"
        else
            # Always output the result - silent only suppresses progress/error messages
            echo "$result"
        fi
        
        mock::node_red::save_state
        return $exit_code
    fi
    
    # For other URLs, use http mock if available or simulate basic curl behavior
    if declare -f mock::http::handle_request &>/dev/null; then
        mock::http::handle_request "$method" "$url" "$data"
    else
        echo "Mock curl: URL not mocked: $url" >&2
        return 1
    fi
}

#######################################
# Docker Command Implementations
#######################################

mock::node_red::docker_ps() {
    shift # remove 'ps'
    local format=""
    local all=false
    
    # Parse ps arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -a|--all)
                all=true
                shift
                ;;
            --format)
                format="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    local container_name="${NODE_RED_MOCK_CONFIG[container_name]}"
    local state="${NODE_RED_MOCK_CONFIG[state]}"
    
    # Only show running containers by default, unless -a is specified
    if [[ "$state" == "running" ]] || [[ "$all" == "true" ]]; then
        if [[ -n "$format" ]]; then
            # Handle format strings like {{.Names}}
            if [[ "$format" == "{{.Names}}" ]]; then
                echo "$container_name"
            elif [[ "$format" == "{{.Status}}" ]]; then
                if [[ "$state" == "running" ]]; then
                    echo "Up 2 hours"
                else
                    echo "Exited (0) 5 minutes ago"
                fi
            elif [[ "$format" == "{{.Image}}" ]]; then
                echo "nodered/node-red:${NODE_RED_MOCK_CONFIG[version]}"
            else
                echo "mock-format-output"
            fi
        else
            # Standard ps output
            echo "CONTAINER ID   IMAGE                            COMMAND                  CREATED        STATUS                    PORTS                    NAMES"
            local status
            if [[ "$state" == "running" ]]; then
                status="Up 2 hours"
            else
                status="Exited (0) 5 minutes ago"
            fi
            echo "abc123def456   nodered/node-red:${NODE_RED_MOCK_CONFIG[version]}   \"npm start\"             2 hours ago    $status   0.0.0.0:${NODE_RED_MOCK_CONFIG[port]}->${NODE_RED_MOCK_CONFIG[port]}/tcp   $container_name"
        fi
    fi
    
    return 0
}

mock::node_red::docker_container() {
    local subcmd="$1"
    shift
    
    case "$subcmd" in
        "inspect")
            mock::node_red::docker_inspect "$@"
            ;;
        *)
            echo "Mock docker container: Unsupported subcommand: $subcmd" >&2
            return 1
            ;;
    esac
}

mock::node_red::docker_run() {
    local container_name=""
    local ports=()
    local volumes=()
    local env_vars=()
    local image=""
    local detached=false
    
    # Parse run arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                container_name="$2"
                shift 2
                ;;
            -p|--publish)
                ports+=("$2")
                shift 2
                ;;
            -v|--volume)
                volumes+=("$2")
                shift 2
                ;;
            -e|--env)
                env_vars+=("$2")
                shift 2
                ;;
            -d|--detach)
                detached=true
                shift
                ;;
            *)
                image="$1"
                shift
                break
                ;;
        esac
    done
    
    # Check if this is Node-RED container
    if [[ "$container_name" == "${NODE_RED_MOCK_CONFIG[container_name]}" ]]; then
        NODE_RED_MOCK_CONFIG[state]="running"
        NODE_RED_MOCK_CONFIG[health]="healthy"
        echo "Container $container_name started successfully"
        
        # Set up Docker mock state if available
        if declare -f mock::docker::set_container_state &>/dev/null; then
            mock::docker::set_container_state "$container_name" "running" "$image"
        fi
        
        return 0
    fi
    
    echo "Mock docker run: Container $container_name created" >&2
    return 0
}

mock::node_red::docker_lifecycle() {
    local action="$1"
    local container_name="$2"
    
    if [[ "$container_name" == "${NODE_RED_MOCK_CONFIG[container_name]}" ]]; then
        case "$action" in
            "start")
                NODE_RED_MOCK_CONFIG[state]="running"
                NODE_RED_MOCK_CONFIG[health]="healthy"
                ;;
            "stop")
                NODE_RED_MOCK_CONFIG[state]="stopped"
                NODE_RED_MOCK_CONFIG[health]="unhealthy"
                ;;
            "restart")
                NODE_RED_MOCK_CONFIG[state]="running"
                NODE_RED_MOCK_CONFIG[health]="healthy"
                ;;
        esac
        
        # Update Docker mock state if available
        if declare -f mock::docker::set_container_state &>/dev/null; then
            mock::docker::set_container_state "$container_name" "${NODE_RED_MOCK_CONFIG[state]}"
        fi
        
        echo "$container_name"
        return 0
    fi
    
    echo "Mock docker $action: Container not found: $container_name" >&2
    return 1
}

mock::node_red::docker_logs() {
    local container_name=""
    local follow=false
    local tail=""
    
    # Parse logs arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -f|--follow)
                follow=true
                shift
                ;;
            --tail)
                tail="$2"
                shift 2
                ;;
            *)
                container_name="$1"
                shift
                ;;
        esac
    done
    
    if [[ "$container_name" == "${NODE_RED_MOCK_CONFIG[container_name]}" ]]; then
        # Simulate Node-RED logs
        cat << EOF
$(date '+%Y-%m-%d %H:%M:%S') - [info] Node-RED version: v${NODE_RED_MOCK_CONFIG[version]}
$(date '+%Y-%m-%d %H:%M:%S') - [info] Node.js  version: v18.17.0
$(date '+%Y-%m-%d %H:%M:%S') - [info] Starting flows
$(date '+%Y-%m-%d %H:%M:%S') - [info] Started flows
$(date '+%Y-%m-%d %H:%M:%S') - [info] Server now running at http://0.0.0.0:${NODE_RED_MOCK_CONFIG[port]}/
EOF
        
        if [[ "$follow" == "true" ]]; then
            # Simulate continuous logs
            while true; do
                echo "$(date '+%Y-%m-%d %H:%M:%S') - [debug] Flow execution: messages processed"
                sleep 5
            done
        fi
        
        return 0
    fi
    
    echo "Mock docker logs: Container not found: $container_name" >&2
    return 1
}

mock::node_red::docker_exec() {
    local container_name=""
    local cmd_args=()
    
    # Parse exec arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -it)
                shift
                ;;
            *)
                if [[ -z "$container_name" ]]; then
                    container_name="$1"
                else
                    cmd_args+=("$1")
                fi
                shift
                ;;
        esac
    done
    
    if [[ "$container_name" == "${NODE_RED_MOCK_CONFIG[container_name]}" ]]; then
        # Simulate command execution in container
        echo "Mock exec in $container_name: ${cmd_args[*]}"
        return 0
    fi
    
    echo "Mock docker exec: Container not found: $container_name" >&2
    return 1
}

mock::node_red::docker_inspect() {
    local container_name="$1"
    local format=""
    
    # Parse inspect arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -f|--format)
                format="$2"
                shift 2
                ;;
            *)
                container_name="$1"
                shift
                ;;
        esac
    done
    
    if [[ "$container_name" == "${NODE_RED_MOCK_CONFIG[container_name]}" ]]; then
        local state="${NODE_RED_MOCK_CONFIG[state]}"
        local running="false"
        [[ "$state" == "running" ]] && running="true"
        
        if [[ -n "$format" ]]; then
            # Handle specific format requests
            if [[ "$format" == "{{.State.Running}}" ]]; then
                echo "$running"
            elif [[ "$format" == "{{.State.Status}}" ]]; then
                echo "$state"
            elif [[ "$format" == "{{.Config.Image}}" ]]; then
                echo "nodered/node-red:${NODE_RED_MOCK_CONFIG[version]}"
            else
                echo "mock-format-value"
            fi
        else
            # Full JSON inspect output
            cat << EOF
[
    {
        "Id": "abc123def456",
        "Created": "$(date -u +%Y-%m-%dT%H:%M:%S.000000000Z)",
        "State": {
            "Status": "$state",
            "Running": $running,
            "Paused": false,
            "Restarting": false,
            "OOMKilled": false,
            "Dead": false,
            "Pid": 12345,
            "ExitCode": 0,
            "Error": "",
            "StartedAt": "$(date -u +%Y-%m-%dT%H:%M:%S.000000000Z)",
            "FinishedAt": "0001-01-01T00:00:00Z"
        },
        "Image": "sha256:xyz789",
        "Name": "/$container_name",
        "Config": {
            "Image": "nodered/node-red:${NODE_RED_MOCK_CONFIG[version]}",
            "ExposedPorts": {
                "${NODE_RED_MOCK_CONFIG[port]}/tcp": {}
            }
        },
        "NetworkSettings": {
            "Ports": {
                "${NODE_RED_MOCK_CONFIG[port]}/tcp": [
                    {
                        "HostIp": "0.0.0.0",
                        "HostPort": "${NODE_RED_MOCK_CONFIG[port]}"
                    }
                ]
            }
        }
    }
]
EOF
        fi
        return 0
    fi
    
    echo "[]"
    return 1
}

mock::node_red::docker_pull() {
    local image="$1"
    echo "Pulling $image... (mocked)"
    echo "Status: Downloaded newer image for $image"
    return 0
}

#######################################
# Node-RED API Handler
#######################################

mock::node_red::handle_api_call() {
    local method="$1"
    local url="$2"
    local data="$3"
    shift 3
    local headers=("$@")
    
    # Check if Node-RED is in error mode
    if [[ -n "${NODE_RED_MOCK_CONFIG[error_mode]}" ]]; then
        case "${NODE_RED_MOCK_CONFIG[error_mode]}" in
            "connection_refused")
                echo "curl: (7) Failed to connect to localhost port ${NODE_RED_MOCK_CONFIG[port]}: Connection refused" >&2
                return 7
                ;;
            "timeout")
                echo "curl: (28) Operation timed out after ${NODE_RED_MOCK_CONFIG[api_timeout]} seconds" >&2
                return 28
                ;;
            "not_found")
                echo '{"error":"Not found"}' >&2
                return 22
                ;;
        esac
    fi
    
    # Check if Node-RED is not running
    if [[ "${NODE_RED_MOCK_CONFIG[state]}" != "running" ]]; then
        echo "curl: (7) Failed to connect to localhost port ${NODE_RED_MOCK_CONFIG[port]}: Connection refused" >&2
        return 7
    fi
    
    # Parse URL path
    local base_url="${NODE_RED_MOCK_CONFIG[base_url]}"
    local path="${url#$base_url}"
    
    case "$path" in
        "/"|"")
            mock::node_red::api_root "$method"
            ;;
        "/settings")
            mock::node_red::api_settings "$method"
            ;;
        "/flows")
            mock::node_red::api_flows "$method" "$data" "${headers[@]}"
            ;;
        "/nodes")
            mock::node_red::api_nodes "$method" "$data"
            ;;
        "/library/"*)
            mock::node_red::api_library "$method" "$path"
            ;;
        "/health")
            mock::node_red::api_health "$method"
            ;;
        "/test"*|"/api/"*|"/webhook/"*|"/endpoint/"*)
            # Known flow execution endpoints (more restrictive patterns)
            mock::node_red::api_flow_endpoint "$method" "$path" "$data"
            ;;
        *)
            echo '{"error":"Unknown endpoint"}' >&2
            return 22
            ;;
    esac
}

#######################################
# API Endpoint Implementations
#######################################

mock::node_red::api_root() {
    local method="$1"
    
    if [[ "$method" == "GET" ]]; then
        echo "<!DOCTYPE html><html><head><title>Node-RED</title></head><body>Node-RED Mock Server</body></html>"
        return 0
    fi
    
    echo '{"error":"Method not allowed"}' >&2
    return 22
}

mock::node_red::api_settings() {
    local method="$1"
    
    if [[ "$method" == "GET" ]]; then
        cat << EOF
{
    "httpNodeRoot": "/",
    "version": "${NODE_RED_MOCK_CONFIG[version]}",
    "user": {
        "username": "${NODE_RED_MOCK_CONFIG[user]}"
    },
    "context": {
        "default": "memory",
        "stores": ["memory", "file"]
    },
    "paletteCategories": [
        "subflows",
        "common",
        "function",
        "network",
        "sequence",
        "parser",
        "storage"
    ],
    "flowFilePretty": true,
    "editorTheme": {
        "projects": {
            "enabled": true
        }
    }
}
EOF
        return 0
    fi
    
    echo '{"error":"Method not allowed"}' >&2
    return 22
}

mock::node_red::api_flows() {
    local method="$1"
    local data="$2"
    shift 2
    local headers=("$@")
    
    case "$method" in
        "GET")
            mock::node_red::get_flows
            ;;
        "POST")
            mock::node_red::deploy_flows "$data" "${headers[@]}"
            ;;
        *)
            echo '{"error":"Method not allowed"}' >&2
            return 22
            ;;
    esac
}

mock::node_red::get_flows() {
    # Return default flows if none configured
    if [[ ${#NODE_RED_MOCK_FLOWS[@]} -eq 0 ]]; then
        cat << 'EOF'
[
    {
        "id": "f1",
        "type": "tab",
        "label": "Flow 1",
        "disabled": false,
        "info": ""
    },
    {
        "id": "n1",
        "type": "inject",
        "z": "f1",
        "name": "",
        "props": [{"p": "payload"}, {"p": "topic", "vt": "str"}],
        "repeat": "",
        "crontab": "",
        "once": false,
        "x": 150,
        "y": 100,
        "wires": [["n2"]]
    },
    {
        "id": "n2",
        "type": "debug",
        "z": "f1",
        "name": "",
        "active": true,
        "tosidebar": true,
        "console": false,
        "x": 350,
        "y": 100,
        "wires": []
    }
]
EOF
    else
        # Return stored flows
        local flows_json="["
        local first=true
        for flow_id in "${!NODE_RED_MOCK_FLOWS[@]}"; do
            if [[ "$first" == "true" ]]; then
                first=false
            else
                flows_json+=","
            fi
            flows_json+="${NODE_RED_MOCK_FLOWS[$flow_id]}"
        done
        flows_json+="]"
        echo "$flows_json"
    fi
    
    return 0
}

mock::node_red::deploy_flows() {
    local data="$1"
    shift
    local headers=("$@")
    
    # Check for deployment type in headers
    local deployment_type="full"
    for header in "${headers[@]}"; do
        if [[ "$header" =~ Node-RED-Deployment-Type:* ]]; then
            deployment_type="${header#*:}"
            deployment_type="${deployment_type// /}"
        fi
    done
    
    # Parse JSON data and update flows (simplified)
    if [[ -n "$data" ]]; then
        # Generate a revision ID
        local rev
        rev="$(date +%s)-$(echo "$data" | md5sum | cut -c1-6 2>/dev/null || echo "abc123")"
        
        # Count flows in data (simplified)
        local flow_count
        flow_count=$(echo "$data" | grep -o '"type":"tab"' | wc -l 2>/dev/null || echo "1")
        
        # Store flow data (simplified - in real implementation would parse JSON properly)
        NODE_RED_MOCK_FLOWS["deployed"]="$data"
        
        echo "{\"rev\":\"$rev\",\"flows\":$flow_count,\"deploy_type\":\"$deployment_type\"}"
        return 0
    fi
    
    echo '{"error":"No flow data provided"}' >&2
    return 22
}

mock::node_red::api_nodes() {
    local method="$1"
    local data="$2"
    
    case "$method" in
        "GET")
            cat << EOF
[
    {
        "id": "node-red/inject",
        "name": "inject",
        "types": ["inject"],
        "enabled": true,
        "module": "node-red",
        "version": "${NODE_RED_MOCK_CONFIG[version]}"
    },
    {
        "id": "node-red/debug",
        "name": "debug",
        "types": ["debug"],
        "enabled": true,
        "module": "node-red",
        "version": "${NODE_RED_MOCK_CONFIG[version]}"
    },
    {
        "id": "node-red/http",
        "name": "http",
        "types": ["http in", "http response"],
        "enabled": true,
        "module": "node-red",
        "version": "${NODE_RED_MOCK_CONFIG[version]}"
    }
]
EOF
            ;;
        "POST")
            # Node installation
            echo '{"success":true,"message":"Node installed successfully"}'
            ;;
        *)
            echo '{"error":"Method not allowed"}' >&2
            return 22
            ;;
    esac
    
    return 0
}

mock::node_red::api_library() {
    local method="$1"
    local path="$2"
    
    if [[ "$method" == "GET" ]]; then
        cat << 'EOF'
[
    {
        "name": "Example Flow",
        "description": "A simple example flow"
    }
]
EOF
        return 0
    fi
    
    echo '{"error":"Method not allowed"}' >&2
    return 22
}

mock::node_red::api_health() {
    local method="$1"
    
    if [[ "$method" == "GET" ]]; then
        local health="${NODE_RED_MOCK_CONFIG[health]}"
        local status="ok"
        
        if [[ "$health" != "healthy" ]]; then
            status="error"
            echo "{\"status\":\"$status\",\"error\":\"Service unhealthy\",\"version\":\"${NODE_RED_MOCK_CONFIG[version]}\"}" >&2
            return 22
        fi
        
        echo "{\"status\":\"$status\",\"version\":\"${NODE_RED_MOCK_CONFIG[version]}\"}"
        return 0
    fi
    
    echo '{"error":"Method not allowed"}' >&2
    return 22
}

mock::node_red::api_flow_endpoint() {
    local method="$1"
    local path="$2"
    local data="$3"
    
    # Simulate flow execution
    if [[ "$method" == "POST" ]]; then
        echo "{\"success\":true,\"message\":\"Flow executed\",\"endpoint\":\"$path\"}"
        return 0
    elif [[ "$method" == "GET" ]]; then
        echo "{\"status\":\"ready\",\"endpoint\":\"$path\"}"
        return 0
    fi
    
    echo '{"error":"Method not allowed"}' >&2
    return 22
}

#######################################
# Test helper functions
#######################################

mock::node_red::reset() {
    local save_state="${1:-true}"
    
    # Clear all data
    NODE_RED_MOCK_FLOWS=()
    NODE_RED_MOCK_NODES=()
    NODE_RED_MOCK_SETTINGS=()
    NODE_RED_MOCK_FLOWS_METADATA=()
    NODE_RED_MOCK_DEBUG_MESSAGES=()
    NODE_RED_MOCK_FLOW_STATS=()
    NODE_RED_MOCK_INSTALLED_NODES=()
    
    # Reset configuration to defaults
    NODE_RED_MOCK_CONFIG=(
        [container_name]="vrooli_node-red"
        [port]="1880"
        [host]="localhost"
        [base_url]="http://localhost:1880"
        [version]="3.1.0"
        [user]="admin"
        [password]=""
        [state]="running"
        [health]="healthy"
        [api_timeout]="30"
        [error_mode]=""
        [settings_loaded]="true"
        [flows_loaded]="true"
    )
    
    # Save the reset state to file if requested (default: true)
    if [[ "$save_state" == "true" ]]; then
        mock::node_red::save_state
    fi
    
    command -v mock::log_state >/dev/null 2>&1 && mock::log_state "node-red" "Node-RED mock reset to initial state" || true
}

mock::node_red::set_state() {
    local state="$1"
    NODE_RED_MOCK_CONFIG[state]="$state"
    
    # Update health based on state
    case "$state" in
        "running")
            NODE_RED_MOCK_CONFIG[health]="healthy"
            ;;
        "stopped"|"exited")
            NODE_RED_MOCK_CONFIG[health]="unhealthy"
            ;;
        "starting")
            NODE_RED_MOCK_CONFIG[health]="starting"
            ;;
    esac
    
    mock::node_red::save_state
    command -v mock::log_state >/dev/null 2>&1 && mock::log_state "node-red" "Set Node-RED state: $state" || true
}

mock::node_red::set_error() {
    local error_mode="$1"
    NODE_RED_MOCK_CONFIG[error_mode]="$error_mode"
    mock::node_red::save_state
    command -v mock::log_state >/dev/null 2>&1 && mock::log_state "node-red" "Set Node-RED error mode: $error_mode" || true
}

mock::node_red::set_config() {
    local key="$1"
    local value="$2"
    NODE_RED_MOCK_CONFIG[$key]="$value"
    mock::node_red::save_state
    command -v mock::log_state >/dev/null 2>&1 && mock::log_state "node-red" "Set Node-RED config: $key=$value" || true
}

mock::node_red::add_flow() {
    local flow_id="$1"
    local flow_json="$2"
    NODE_RED_MOCK_FLOWS[$flow_id]="$flow_json"
    mock::node_red::save_state
    command -v mock::log_state >/dev/null 2>&1 && mock::log_state "node-red" "Added flow: $flow_id" || true
}

mock::node_red::install_node() {
    local node_name="$1"
    local node_version="${2:-1.0.0}"
    NODE_RED_MOCK_INSTALLED_NODES[$node_name]="$node_version"
    mock::node_red::save_state
    command -v mock::log_state >/dev/null 2>&1 && mock::log_state "node-red" "Installed node: $node_name@$node_version" || true
}

#######################################
# Test assertions
#######################################

mock::node_red::assert_running() {
    if [[ "${NODE_RED_MOCK_CONFIG[state]}" == "running" ]]; then
        return 0
    else
        echo "Assertion failed: Node-RED is not running (state: ${NODE_RED_MOCK_CONFIG[state]})" >&2
        return 1
    fi
}

mock::node_red::assert_stopped() {
    if [[ "${NODE_RED_MOCK_CONFIG[state]}" != "running" ]]; then
        return 0
    else
        echo "Assertion failed: Node-RED is still running" >&2
        return 1
    fi
}

mock::node_red::assert_healthy() {
    if [[ "${NODE_RED_MOCK_CONFIG[health]}" == "healthy" ]]; then
        return 0
    else
        echo "Assertion failed: Node-RED is not healthy (health: ${NODE_RED_MOCK_CONFIG[health]})" >&2
        return 1
    fi
}

mock::node_red::assert_flow_exists() {
    local flow_id="$1"
    if [[ -n "${NODE_RED_MOCK_FLOWS[$flow_id]}" ]]; then
        return 0
    else
        echo "Assertion failed: Flow '$flow_id' does not exist" >&2
        return 1
    fi
}

mock::node_red::assert_node_installed() {
    local node_name="$1"
    if [[ -n "${NODE_RED_MOCK_INSTALLED_NODES[$node_name]}" ]]; then
        return 0
    else
        echo "Assertion failed: Node '$node_name' is not installed" >&2
        return 1
    fi
}

#######################################
# Debug functions
#######################################

mock::node_red::dump_state() {
    echo "=== Node-RED Mock State ==="
    echo "Configuration:"
    for key in "${!NODE_RED_MOCK_CONFIG[@]}"; do
        echo "  $key: ${NODE_RED_MOCK_CONFIG[$key]}"
    done
    
    echo "Flows:"
    for flow_id in "${!NODE_RED_MOCK_FLOWS[@]}"; do
        echo "  $flow_id: ${NODE_RED_MOCK_FLOWS[$flow_id]:0:100}..."
    done
    
    echo "Installed Nodes:"
    for node_name in "${!NODE_RED_MOCK_INSTALLED_NODES[@]}"; do
        echo "  $node_name: ${NODE_RED_MOCK_INSTALLED_NODES[$node_name]}"
    done
    echo "=========================="
}

# Export all functions
export -f docker
export -f curl
export -f mock::node_red::save_state
export -f mock::node_red::load_state
export -f mock::node_red::docker_ps
export -f mock::node_red::docker_container
export -f mock::node_red::docker_run
export -f mock::node_red::docker_lifecycle
export -f mock::node_red::docker_logs
export -f mock::node_red::docker_exec
export -f mock::node_red::docker_inspect
export -f mock::node_red::docker_pull
export -f mock::node_red::handle_api_call
export -f mock::node_red::api_root
export -f mock::node_red::api_settings
export -f mock::node_red::api_flows
export -f mock::node_red::get_flows
export -f mock::node_red::deploy_flows
export -f mock::node_red::api_nodes
export -f mock::node_red::api_library
export -f mock::node_red::api_health
export -f mock::node_red::api_flow_endpoint
export -f mock::node_red::reset
export -f mock::node_red::set_state
export -f mock::node_red::set_error
export -f mock::node_red::set_config
export -f mock::node_red::add_flow
export -f mock::node_red::install_node
export -f mock::node_red::assert_running
export -f mock::node_red::assert_stopped
export -f mock::node_red::assert_healthy
export -f mock::node_red::assert_flow_exists
export -f mock::node_red::assert_node_installed
export -f mock::node_red::dump_state

# Save initial state
mock::node_red::save_state

echo "[NODE_RED_MOCK] Node-RED mock implementation loaded"