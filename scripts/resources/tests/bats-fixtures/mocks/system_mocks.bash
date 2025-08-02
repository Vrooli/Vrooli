#!/usr/bin/env bash
# System Command Mocks for Bats Tests
# Provides configurable mocks for common system commands used across resource tests

# Initialize mock response directory
MOCK_RESPONSES_DIR="${BATS_TEST_TMPDIR:-/tmp}/mock_responses"

#######################################
# Docker mock with configurable behaviors
# Supports common docker commands with test-specific responses
#######################################
docker() {
    local mock_file="$MOCK_RESPONSES_DIR/docker_${1}_response"
    local exit_file="$MOCK_RESPONSES_DIR/docker_${1}_exitcode"
    
    # Check for test-specific mock configuration
    if [[ -f "$mock_file" ]]; then
        cat "$mock_file"
        return $(cat "$exit_file" 2>/dev/null || echo 0)
    fi
    
    # Default behaviors based on command
    case "$1" in
        "ps")
            _handle_docker_ps "$@"
            ;;
        "inspect")
            _handle_docker_inspect "$@"
            ;;
        "container")
            # Handle docker container subcommands
            case "$2" in
                "inspect")
                    shift # remove "container"
                    _handle_docker_inspect "$@"
                    ;;
                *)
                    echo "Mock docker container: $*" >&2
                    return 0
                    ;;
            esac
            ;;
        "run")
            _handle_docker_run "$@"
            ;;
        "exec")
            _handle_docker_exec "$@"
            ;;
        "logs")
            _handle_docker_logs "$@"
            ;;
        "images")
            _handle_docker_images "$@"
            ;;
        "stop"|"start"|"restart")
            _handle_docker_lifecycle "$@"
            ;;
        "pull")
            echo "Using default tag: latest"
            echo "latest: Pulling from library/mock"
            echo "Status: Downloaded newer image for mock:latest"
            return 0
            ;;
        "version")
            echo "Docker version 20.10.0, build mock"
            return 0
            ;;
        *)
            echo "Mock docker: $*" >&2
            return 0
            ;;
    esac
}

# Docker ps handler
_handle_docker_ps() {
    local show_all=false
    local filter=""
    local format=""
    
    # Parse arguments
    shift # skip "ps"
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -a|--all) 
                show_all=true
                shift 
                ;;
            --filter) 
                filter="$2"
                shift 2 
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
    
    # List all containers based on their state files
    for state_file in "$MOCK_RESPONSES_DIR"/container_*_state; do
        if [[ -f "$state_file" ]]; then
            local container_name=$(basename "$state_file" | sed 's/container_//; s/_state$//')
            local state=$(cat "$state_file")
            
            # Show container if running, or if show_all is true and it exists
            if [[ "$state" == "running" ]] || ([[ "$show_all" == "true" ]] && [[ "$state" != "missing" ]]); then
                if [[ -n "$format" ]]; then
                    # Handle format strings
                    case "$format" in
                        "{{.Names}}")
                            echo "$container_name"
                            ;;
                        *)
                            # Default to showing container name for other formats
                            echo "$container_name"
                            ;;
                    esac
                else
                    # Default table format
                    echo "CONTAINER ID   IMAGE         COMMAND   CREATED   STATUS    PORTS   NAMES"
                    echo "abc123def456   agent-s2:latest   \"\"   1 hour ago   Up 1 hour   4113/tcp   $container_name"
                fi
            fi
        fi
    done
    
    return 0
}

# Docker inspect handler
_handle_docker_inspect() {
    local container=""
    local format=""
    
    # Parse arguments
    shift # skip "inspect"
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --format=*)
                format="${1#--format=}"
                shift
                ;;
            --format)
                format="$2"
                shift 2
                ;;
            *)
                container="$1"
                shift
                ;;
        esac
    done
    
    # Check if container exists
    local state_file="$MOCK_RESPONSES_DIR/container_${container}_state"
    if [[ ! -f "$state_file" ]]; then
        echo "Error: No such object: $container" >&2
        return 1
    fi
    
    # Handle format string for health status
    if [[ "$format" == "{{.State.Health.Status}}" ]]; then
        local health_file="$MOCK_RESPONSES_DIR/container_${container}_health"
        if [[ -f "$health_file" ]]; then
            cat "$health_file"
        else
            echo "unknown"
        fi
        return 0
    fi
    
    # Check for custom inspect file
    local inspect_file="$MOCK_RESPONSES_DIR/container_${container}_inspect"
    if [[ -f "$inspect_file" ]]; then
        cat "$inspect_file"
        return 0
    fi
    
    # Default full inspect output
    echo '[{"State": {"Running": true, "Health": {"Status": "healthy"}}}]'
    return 0
}

# Docker run handler
_handle_docker_run() {
    echo "Mock container started"
    return 0
}

# Docker exec handler
_handle_docker_exec() {
    local exec_file="$MOCK_RESPONSES_DIR/docker_exec_response"
    if [[ -f "$exec_file" ]]; then
        cat "$exec_file"
        return $(cat "$MOCK_RESPONSES_DIR/docker_exec_exitcode" 2>/dev/null || echo 0)
    fi
    return 0
}

# Docker logs handler
_handle_docker_logs() {
    local container="${2:-}"
    local logs_file="$MOCK_RESPONSES_DIR/container_${container}_logs"
    
    if [[ -f "$logs_file" ]]; then
        cat "$logs_file"
    else
        echo "Mock logs for $container"
    fi
    return 0
}

# Docker lifecycle commands handler
_handle_docker_lifecycle() {
    local action="$1"
    local container="${2:-}"
    
    case "$action" in
        "start")
            echo "$container"
            ;;
        "stop")
            echo "$container"
            ;;
        "restart")
            echo "$container"
            ;;
    esac
    return 0
}

# Docker images handler
_handle_docker_images() {
    local image_filter=""
    local format=""
    
    # Parse arguments for image filtering
    while [[ $# -gt 1 ]]; do
        case "$2" in
            --filter) 
                image_filter="$3"
                shift 2
                ;;
            --format)
                format="$3"
                shift 2
                ;;
            *) 
                # Assume it's an image name if no filter
                image_filter="$2"
                shift
                ;;
        esac
    done
    
    # Check all images that should exist
    local has_images=false
    for mock_file in "$MOCK_RESPONSES_DIR"/image_*_exists; do
        if [[ -f "$mock_file" ]]; then
            has_images=true
            local image_name=$(basename "$mock_file" | sed 's/image_//; s/_exists$//')
            # Replace underscores back to colons for image names like agent-s2:latest
            image_name="${image_name//__/:}"
            
            if [[ -n "$format" ]]; then
                # Handle format option - just output the formatted string
                echo "$image_name"
            else
                # Default table format
                if [[ "$has_images" == "true" ]]; then
                    echo "REPOSITORY    TAG       IMAGE ID       CREATED        SIZE"
                    has_images="header_printed"
                fi
                local repo="${image_name%:*}"
                local tag="${image_name#*:}"
                echo "$repo   $tag    abc123def456   2 hours ago    1.2GB"
            fi
        fi
    done
    
    # If no images exist and not using format, show headers
    if [[ "$has_images" == "false" && -z "$format" ]]; then
        echo "REPOSITORY    TAG       IMAGE ID       CREATED        SIZE"
    fi
    
    return 0
}

#######################################
# Curl mock for HTTP requests
#######################################
curl() {
    local url=""
    local method="GET"
    local data=""
    local silent=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -X) method="$2"; shift 2 ;;
            -d|--data) data="$2"; shift 2 ;;
            -s|--silent) silent=true; shift ;;
            -f|--fail) shift ;; # Ignore fail flag in mock
            -H|--header) shift 2 ;; # Ignore headers in basic mock
            http*) url="$1"; shift ;;
            *) shift ;;
        esac
    done
    
    # Check for URL-specific mock
    local url_hash=$(echo -n "$url" | md5sum | cut -d' ' -f1)
    
    # Check for sequence responses first
    local sequence_count_file="$MOCK_RESPONSES_DIR/curl_${url_hash}_sequence_count"
    local sequence_index_file="$MOCK_RESPONSES_DIR/curl_${url_hash}_sequence_index"
    
    if [[ -f "$sequence_count_file" && -f "$sequence_index_file" ]]; then
        local count=$(cat "$sequence_count_file")
        local index=$(cat "$sequence_index_file")
        
        if [[ "$index" -lt "$count" ]]; then
            local seq_response_file="$MOCK_RESPONSES_DIR/curl_${url_hash}_sequence_${index}_response"
            local seq_exit_file="$MOCK_RESPONSES_DIR/curl_${url_hash}_sequence_${index}_exitcode"
            
            # Increment index for next call
            echo $((index + 1)) > "$sequence_index_file"
            
            # Return sequence response
            if [[ -f "$seq_response_file" ]]; then
                [[ "$silent" != "true" ]] && cat "$seq_response_file" || cat "$seq_response_file" 2>/dev/null
                return $(cat "$seq_exit_file" 2>/dev/null || echo 0)
            fi
        fi
    fi
    
    # Fall back to single response
    local response_file="$MOCK_RESPONSES_DIR/curl_${url_hash}_response"
    local exit_file="$MOCK_RESPONSES_DIR/curl_${url_hash}_exitcode"
    
    if [[ -f "$response_file" ]]; then
        [[ "$silent" != "true" ]] && cat "$response_file" || cat "$response_file" 2>/dev/null
        return $(cat "$exit_file" 2>/dev/null || echo 0)
    fi
    
    # Default responses based on common patterns
    if [[ "$url" =~ health|status ]]; then
        echo '{"status":"ok","timestamp":"2024-01-01T00:00:00Z"}'
        return 0
    elif [[ "$url" =~ /api/tags ]]; then
        echo '{"models":[{"name":"llama2:latest","size":4000000000}]}'
        return 0
    elif [[ "$url" =~ /version ]]; then
        echo '{"version":"1.0.0"}'
        return 0
    fi
    
    # Generic success response
    echo '{"success":true}'
    return 0
}

#######################################
# System command mocks
#######################################
which() {
    local cmd="$1"
    case "$cmd" in
        docker|curl|jq|nc|systemctl|sudo)
            echo "/usr/bin/$cmd"
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

command() {
    if [[ "$1" == "-v" ]]; then
        which "$2"
    fi
}

systemctl() {
    local action="$1"
    local service="${2:-}"
    
    case "$action" in
        "status")
            if [[ -f "$MOCK_RESPONSES_DIR/systemctl_${service}_status" ]]; then
                cat "$MOCK_RESPONSES_DIR/systemctl_${service}_status"
                return $(cat "$MOCK_RESPONSES_DIR/systemctl_${service}_exitcode" 2>/dev/null || echo 0)
            fi
            echo "● $service.service - Mock Service"
            echo "   Active: active (running)"
            return 0
            ;;
        "start"|"stop"|"restart")
            return 0
            ;;
        *)
            return 0
            ;;
    esac
}

#######################################
# Network utilities
#######################################
nc() {
    # Netcat mock - always succeed for port checks
    return 0
}

ping() {
    # Ping mock - configurable
    if [[ -f "$MOCK_RESPONSES_DIR/network_offline" ]]; then
        return 1
    fi
    return 0
}

#######################################
# File system utilities
#######################################
sudo() {
    # Pass through all commands in test environment
    shift  # Remove 'sudo'
    "$@"
}

# Port checking mock
lsof() {
    if [[ "$1" == "-i" ]] && [[ "$2" =~ ^:[0-9]+$ ]]; then
        local port="${2#:}"
        local port_file="$MOCK_RESPONSES_DIR/port_${port}_in_use"
        
        if [[ -f "$port_file" ]]; then
            echo "COMMAND   PID USER   FD   TYPE DEVICE SIZE/OFF NODE NAME"
            echo "mockapp  1234 user    3u  IPv4  12345      0t0  TCP *:$port (LISTEN)"
            return 0
        fi
    fi
    return 1
}

#######################################
# Logging function mocks
# Standard logging interface used across all resources
#######################################
log::info() {
    echo "[INFO]    $*"
    return 0
}

log::success() {
    echo "[SUCCESS] $*"
    return 0
}

log::error() {
    echo "[ERROR]   $*"
    return 1
}

log::warn() {
    echo "[WARNING] $*"
    return 0
}

log::warning() {
    echo "[WARNING] $*"
    return 0
}

log::header() {
    echo "HEADER: $*"
    return 0
}

log::debug() {
    # Only output if DEBUG mode is enabled
    if [[ "${MOCK_DEBUG:-false}" == "true" ]]; then
        echo "DEBUG: $*"
    fi
    return 0
}

#######################################
# System utility mocks
#######################################
system::is_command() {
    local cmd="$1"
    case "$cmd" in
        # Always report these common commands as available in tests
        docker|curl|jq|nc|systemctl|sudo|which|command|date|wc|tr|sed|cat|echo|grep|awk)
            return 0
            ;;
        # For other commands, check if a mock override exists
        *)
            if [[ -f "$MOCK_RESPONSES_DIR/command_${cmd}_available" ]]; then
                return 0
            fi
            return 1
            ;;
    esac
}

# Export functions for use in subshells
export -f docker _handle_docker_ps _handle_docker_inspect _handle_docker_run
export -f _handle_docker_exec _handle_docker_logs _handle_docker_lifecycle
export -f curl which command systemctl nc ping sudo lsof
export -f log::info log::success log::error log::warn log::header log::debug
export -f system::is_command