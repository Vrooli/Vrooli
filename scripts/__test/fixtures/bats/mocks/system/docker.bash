#!/usr/bin/env bash
# Docker System Mocks
# Provides comprehensive Docker command mocking for tests

# Prevent duplicate loading
if [[ "${DOCKER_MOCKS_LOADED:-}" == "true" ]]; then
    return 0
fi
export DOCKER_MOCKS_LOADED="true"

# Docker mock state storage
declare -A MOCK_DOCKER_CONTAINERS
declare -A MOCK_DOCKER_IMAGES
declare -A MOCK_DOCKER_NETWORKS
declare -A MOCK_DOCKER_VOLUMES

# Global Docker mock configuration
export DOCKER_MOCK_MODE="${DOCKER_MOCK_MODE:-normal}"  # normal, offline, error

#######################################
# Set mock container state
# Arguments: $1 - container name, $2 - state, $3 - additional info
#######################################
mock::docker::set_container_state() {
    local container="$1"
    local state="$2"
    local info="${3:-}"
    
    MOCK_DOCKER_CONTAINERS["$container"]="$state|$info"
    
    # Log mock usage for verification
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "docker_container_state:$container:$state" >> "${MOCK_RESPONSES_DIR}/used_mocks.log"
    fi
}

#######################################
# Set mock image availability
# Arguments: $1 - image name, $2 - available (true/false)
#######################################
mock::docker::set_image_available() {
    local image="$1"
    local available="$2"
    
    MOCK_DOCKER_IMAGES["$image"]="$available"
}

#######################################
# Main Docker command mock
#######################################
docker() {
    # Track command calls for verification
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "docker $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    # Handle different Docker mock modes
    case "$DOCKER_MOCK_MODE" in
        "offline")
            echo "ERROR: Cannot connect to the Docker daemon" >&2
            return 1
            ;;
        "error")
            echo "ERROR: Docker command failed" >&2
            return 1
            ;;
    esac
    
    # Handle global flags first
    case "$1" in
        "--version")
            echo "Docker version 24.0.0, build abc123"
            return 0
            ;;
        "--help"|"-h")
            echo "Usage: docker [OPTIONS] COMMAND"
            echo "Docker CLI commands:"
            echo "  ps      List containers"
            echo "  run     Run a command in a new container"
            echo "  version Show the Docker version information"
            return 0
            ;;
    esac
    
    case "$1" in
        "ps")
            docker_mock_ps "$@"
            ;;
        "images")
            docker_mock_images "$@"
            ;;
        "run")
            docker_mock_run "$@"
            ;;
        "exec")
            docker_mock_exec "$@"
            ;;
        "logs")
            docker_mock_logs "$@"
            ;;
        "inspect")
            docker_mock_inspect "$@"
            ;;
        "pull")
            docker_mock_pull "$@"
            ;;
        "build")
            docker_mock_build "$@"
            ;;
        "start"|"stop"|"restart")
            docker_mock_lifecycle "$@"
            ;;
        "rm"|"rmi")
            docker_mock_remove "$@"
            ;;
        "network")
            docker_mock_network "$@"
            ;;
        "volume")
            docker_mock_volume "$@"
            ;;
        "system")
            docker_mock_system "$@"
            ;;
        "version")
            echo "Docker version 24.0.0, build abc123"
            ;;
        "info")
            echo "Client: Docker Engine - Community"
            echo "Server: Docker Engine - Community"
            ;;
        *)
            echo "docker: '$1' is not a docker command." >&2
            return 1
            ;;
    esac
}

#######################################
# Docker ps mock
#######################################
docker_mock_ps() {
    shift  # Remove 'ps'
    
    local format="table"
    local all_containers=false
    local filters=()
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            "--format")
                format="$2"
                shift 2
                ;;
            "-a"|"--all")
                all_containers=true
                shift
                ;;
            "--filter")
                filters+=("$2")
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Generate output based on mock container states
    if [[ "$format" == "table" ]]; then
        echo "CONTAINER ID   IMAGE          COMMAND       CREATED        STATUS         PORTS                    NAMES"
    fi
    
    for container_name in "${!MOCK_DOCKER_CONTAINERS[@]}"; do
        local state_info="${MOCK_DOCKER_CONTAINERS[$container_name]}"
        local state="${state_info%%|*}"
        local info="${state_info##*|}"
        
        # Filter by running state unless --all is specified
        if [[ "$state" == "running" ]] || [[ "$all_containers" == true ]]; then
            local container_id="$(echo "$container_name" | md5sum | cut -c1-12)"
            local image="${info:-test:latest}"
            local command='"/entrypoint.sh"'
            local created="2 minutes ago"
            local ports="0.0.0.0:8080->8080/tcp"
            
            local status
            case "$state" in
                "running") status="Up 2 minutes" ;;
                "stopped") status="Exited (0) 1 minute ago" ;;
                "paused") status="Up 2 minutes (Paused)" ;;
                *) status="$state" ;;
            esac
            
            if [[ "$format" == "table" ]]; then
                printf "%-12s   %-12s   %-12s   %-12s   %-12s   %-22s   %s\n" \
                    "$container_id" "$image" "$command" "$created" "$status" "$ports" "$container_name"
            else
                echo "$container_name"
            fi
        fi
    done
}

#######################################
# Docker images mock
#######################################
docker_mock_images() {
    shift  # Remove 'images'
    
    echo "REPOSITORY    TAG       IMAGE ID       CREATED        SIZE"
    for image_name in "${!MOCK_DOCKER_IMAGES[@]}"; do
        if [[ "${MOCK_DOCKER_IMAGES[$image_name]}" == "true" ]]; then
            local image_id="$(echo "$image_name" | md5sum | cut -c1-12)"
            printf "%-12s  %-8s  %-12s  %-12s  %s\n" \
                "$image_name" "latest" "$image_id" "2 hours ago" "123MB"
        fi
    done
}

#######################################
# Docker run mock
#######################################
docker_mock_run() {
    shift  # Remove 'run'
    
    local container_name=""
    local image=""
    local detach=false
    
    # Parse basic arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            "--name")
                container_name="$2"
                shift 2
                ;;
            "-d"|"--detach")
                detach=true
                shift
                ;;
            *)
                if [[ -z "$image" && ! "$1" =~ ^- ]]; then
                    image="$1"
                fi
                shift
                ;;
        esac
    done
    
    # Generate container ID
    local container_id="$(echo "${container_name:-$image}" | md5sum | cut -c1-64)"
    
    # Set default state if container name provided
    if [[ -n "$container_name" ]]; then
        mock::docker::set_container_state "$container_name" "running" "$image"
    fi
    
    if [[ "$detach" == true ]]; then
        echo "$container_id"
    else
        echo "Container output simulation"
    fi
}

#######################################
# Docker exec mock
#######################################
docker_mock_exec() {
    shift  # Remove 'exec'
    
    local container=""
    local command=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        if [[ ! "$1" =~ ^- ]]; then
            if [[ -z "$container" ]]; then
                container="$1"
            else
                command="$*"
                break
            fi
        fi
        shift
    done
    
    # Simulate command execution
    case "$command" in
        *"health"*|*"status"*)
            echo "healthy"
            ;;
        *"version"*)
            echo "1.0.0"
            ;;
        *"ps"*|*"list"*)
            echo "process1 process2"
            ;;
        *)
            echo "Mock exec output for: $command"
            ;;
    esac
}

#######################################
# Docker logs mock
#######################################
docker_mock_logs() {
    shift  # Remove 'logs'
    
    local container=""
    local follow=false
    local tail=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            "-f"|"--follow")
                follow=true
                shift
                ;;
            "--tail")
                tail="$2"
                shift 2
                ;;
            *)
                container="$1"
                shift
                ;;
        esac
    done
    
    # Generate mock log output
    echo "$(date): [INFO] Container $container started"
    echo "$(date): [INFO] Service initialization complete"
    echo "$(date): [INFO] Ready to accept connections"
    
    if [[ "$follow" == true ]]; then
        # In a real scenario, this would stream logs
        echo "$(date): [INFO] Streaming logs..."
    fi
}

#######################################
# Docker inspect mock
#######################################
docker_mock_inspect() {
    shift  # Remove 'inspect'
    
    local container=""
    local format=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            "--format")
                format="$2"
                shift 2
                ;;
            *)
                container="$1"
                shift
                ;;
        esac
    done
    
    # Get container state
    local state_info="${MOCK_DOCKER_CONTAINERS[$container]:-stopped|}"
    local state="${state_info%%|*}"
    local image="${state_info##*|}"
    
    if [[ -n "$format" ]]; then
        # Handle specific format requests
        case "$format" in
            "{{.State.Status}}")
                echo "$state"
                ;;
            "{{.State.Health.Status}}")
                case "$state" in
                    "running") echo "healthy" ;;
                    *) echo "unhealthy" ;;
                esac
                ;;
            "{{.Config.Image}}")
                echo "${image:-test:latest}"
                ;;
            *)
                echo "mock-value"
                ;;
        esac
    else
        # Return full JSON
        cat << EOF
[{
    "Id": "$(echo "$container" | md5sum | cut -c1-64)",
    "Created": "$(date -Iseconds)",
    "State": {
        "Status": "$state",
        "Running": $([ "$state" = "running" ] && echo "true" || echo "false"),
        "Health": {
            "Status": $([ "$state" = "running" ] && echo "\"healthy\"" || echo "\"unhealthy\"")
        }
    },
    "Config": {
        "Image": "${image:-test:latest}",
        "Env": ["PATH=/usr/local/bin:/usr/bin:/bin"]
    },
    "NetworkSettings": {
        "Ports": {
            "8080/tcp": [{"HostIp": "0.0.0.0", "HostPort": "8080"}]
        }
    }
}]
EOF
    fi
}

#######################################
# Docker pull mock
#######################################
docker_mock_pull() {
    shift  # Remove 'pull'
    local image="$1"
    
    echo "Pulling $image..."
    echo "latest: Pulling from library/${image%%:*}"
    echo "Digest: sha256:abc123..."
    echo "Status: Downloaded newer image for $image"
    
    # Mark image as available
    mock::docker::set_image_available "$image" "true"
}

#######################################
# Docker build mock
#######################################
docker_mock_build() {
    shift  # Remove 'build'
    
    local tag=""
    local context="."
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            "-t"|"--tag")
                tag="$2"
                shift 2
                ;;
            *)
                context="$1"
                shift
                ;;
        esac
    done
    
    echo "Building image from context: $context"
    echo "Step 1/3 : FROM ubuntu:latest"
    echo "Step 2/3 : COPY . /app"
    echo "Step 3/3 : CMD [\"/app/start.sh\"]"
    echo "Successfully built abc123def456"
    
    if [[ -n "$tag" ]]; then
        echo "Successfully tagged $tag"
        mock::docker::set_image_available "$tag" "true"
    fi
}

#######################################
# Docker lifecycle mock (start/stop/restart)
#######################################
docker_mock_lifecycle() {
    local action="$1"
    shift
    local container="$1"
    
    case "$action" in
        "start")
            mock::docker::set_container_state "$container" "running"
            echo "$container"
            ;;
        "stop")
            mock::docker::set_container_state "$container" "stopped"
            echo "$container"
            ;;
        "restart")
            mock::docker::set_container_state "$container" "running"
            echo "$container"
            ;;
    esac
}

#######################################
# Docker remove mock (rm/rmi)
#######################################
docker_mock_remove() {
    local action="$1"
    shift
    
    case "$action" in
        "rm")
            local container="$1"
            unset MOCK_DOCKER_CONTAINERS["$container"]
            echo "$container"
            ;;
        "rmi")
            local image="$1"
            unset MOCK_DOCKER_IMAGES["$image"]
            echo "Untagged: $image"
            ;;
    esac
}

#######################################
# Docker network mock
#######################################
docker_mock_network() {
    shift  # Remove 'network'
    local subcommand="$1"
    
    case "$subcommand" in
        "ls")
            echo "NETWORK ID     NAME      DRIVER    SCOPE"
            echo "abc123def456   bridge    bridge    local"
            echo "def456ghi789   host      host      local"
            ;;
        "create")
            local network_name="$2"
            echo "$(echo "$network_name" | md5sum | cut -c1-12)"
            ;;
        *)
            echo "Mock network command: $subcommand"
            ;;
    esac
}

#######################################
# Docker volume mock
#######################################
docker_mock_volume() {
    shift  # Remove 'volume'
    local subcommand="$1"
    
    case "$subcommand" in
        "ls")
            echo "DRIVER    VOLUME NAME"
            echo "local     test_volume"
            ;;
        "create")
            local volume_name="$2"
            echo "$volume_name"
            ;;
        *)
            echo "Mock volume command: $subcommand"
            ;;
    esac
}

#######################################
# Docker system mock
#######################################
docker_mock_system() {
    shift  # Remove 'system'
    local subcommand="$1"
    
    case "$subcommand" in
        "prune")
            echo "Total reclaimed space: 123MB"
            ;;
        "df")
            echo "TYPE           TOTAL     ACTIVE    SIZE      RECLAIMABLE"
            echo "Images         5         2         1.2GB     500MB (41%)"
            echo "Containers     3         1         100MB     80MB (80%)"
            ;;
        *)
            echo "Mock system command: $subcommand"
            ;;
    esac
}

#######################################
# Docker compose mock (if docker-compose is called)
#######################################
docker-compose() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "docker-compose $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    case "$1" in
        "up")
            echo "Creating network \"test_default\" with the default driver"
            echo "Creating test_service_1 ... done"
            ;;
        "down")
            echo "Stopping test_service_1 ... done"
            echo "Removing test_service_1 ... done"
            ;;
        "ps")
            echo "Name            Command     State    Ports"
            echo "test_service_1  /start.sh   Up       0.0.0.0:8080->8080/tcp"
            ;;
        *)
            echo "Mock docker-compose command: $*"
            ;;
    esac
}

# Export functions
export -f docker docker-compose
export -f mock::docker::set_container_state mock::docker::set_image_available
export -f docker_mock_ps docker_mock_images docker_mock_run docker_mock_exec
export -f docker_mock_logs docker_mock_inspect docker_mock_pull docker_mock_build
export -f docker_mock_lifecycle docker_mock_remove docker_mock_network
export -f docker_mock_volume docker_mock_system

echo "[DOCKER_MOCKS] Docker system mocks loaded"