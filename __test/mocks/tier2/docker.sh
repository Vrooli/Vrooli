#!/usr/bin/env bash
# Docker Mock - Tier 2 (Stateful)
# 
# Provides stateful Docker command mock for testing:
# - Container lifecycle management (run, start, stop, rm)
# - Image management (images, pull, build)
# - Container inspection and logs
# - Basic network operations
# - Error injection for resilience testing
#
# Coverage: ~80% of common Docker use cases in 500 lines

# === Configuration ===
declare -gA DOCKER_CONTAINERS=()       # Container storage: name -> "state|image|created"
declare -gA DOCKER_IMAGES=()           # Image storage: name -> "available|size|created"
declare -gA DOCKER_NETWORKS=()         # Network storage: name -> "id|driver"
declare -gA DOCKER_CONFIG=(            # Docker configuration
    [version]="24.0.0"
    [mode]="normal"
    [error_mode]=""
)

# Debug mode
declare -g DOCKER_DEBUG="${DOCKER_DEBUG:-}"

# === Helper Functions ===
docker_debug() {
    [[ -n "$DOCKER_DEBUG" ]] && echo "[MOCK:DOCKER] $*" >&2
}

docker_check_error() {
    case "${DOCKER_CONFIG[error_mode]}" in
        "offline")
            echo "ERROR: Cannot connect to the Docker daemon" >&2
            return 1
            ;;
        "permission_denied")
            echo "ERROR: permission denied while trying to connect to the Docker daemon socket" >&2
            return 1
            ;;
        "network_timeout")
            echo "ERROR: Get \"https://registry-1.docker.io/v2/\": net/http: request canceled (Client.Timeout exceeded)" >&2
            return 1
            ;;
    esac
    return 0
}

docker_mock_id() {
    local name="$1"
    # Generate Docker-like 12-char hex ID
    local hex=$(printf "%s_%s" "$name" "$RANDOM" | sha256sum 2>/dev/null | cut -c1-12 || echo "${name:0:12}")
    echo "$hex"
}

docker_mock_time_ago() {
    local seconds=$(( (RANDOM % 3600) + 60 ))
    if ((seconds < 60)); then
        echo "$seconds seconds ago"
    elif ((seconds < 3600)); then
        echo "$((seconds / 60)) minutes ago" 
    else
        echo "$((seconds / 3600)) hours ago"
    fi
}

# === Main Docker Command Interceptor ===
docker() {
    docker_debug "docker called with: $*"
    
    # Check for error injection
    if ! docker_check_error; then
        return $?
    fi
    
    # Check mode
    case "${DOCKER_CONFIG[mode]}" in
        "offline")
            echo "ERROR: Cannot connect to the Docker daemon" >&2
            return 1
            ;;
        "error")
            echo "ERROR: Docker command failed" >&2
            return 1
            ;;
    esac
    
    if [[ $# -eq 0 ]]; then
        echo "Usage: docker [OPTIONS] COMMAND" >&2
        return 1
    fi
    
    # Handle version and help at top level
    case "$1" in
        --version)
            echo "Docker version ${DOCKER_CONFIG[version]}, build abc123"
            return 0
            ;;
        -h|--help)
            echo "Usage: docker [OPTIONS] COMMAND"
            echo "Commands:"
            echo "  ps        List containers"
            echo "  images    List images"
            echo "  run       Run a command in a new container"
            echo "  exec      Run a command in a running container"
            echo "  logs      Fetch logs of a container"
            echo "  inspect   Return low-level information"
            echo "  pull      Pull an image"
            echo "  start     Start containers"
            echo "  stop      Stop containers"
            echo "  rm        Remove containers"
            echo "  rmi       Remove images"
            echo "  build     Build an image from Dockerfile"
            echo "  version   Show Docker version"
            echo "  info      Display system information"
            return 0
            ;;
    esac
    
    # Route to command handlers
    local cmd="$1"
    shift
    
    case "$cmd" in
        ps) docker_cmd_ps "$@" ;;
        images) docker_cmd_images "$@" ;;
        run) docker_cmd_run "$@" ;;
        exec) docker_cmd_exec "$@" ;;
        logs) docker_cmd_logs "$@" ;;
        inspect) docker_cmd_inspect "$@" ;;
        pull) docker_cmd_pull "$@" ;;
        build) docker_cmd_build "$@" ;;
        start) docker_cmd_start "$@" ;;
        stop) docker_cmd_stop "$@" ;;
        rm) docker_cmd_rm "$@" ;;
        rmi) docker_cmd_rmi "$@" ;;
        version) echo "Docker version ${DOCKER_CONFIG[version]}, build abc123" ;;
        info) echo "Client: Docker Engine - Community"; echo "Server: Docker Engine - Community" ;;
        *)
            echo "docker: '$cmd' is not a docker command." >&2
            return 1
            ;;
    esac
}

# === Command Implementations ===

docker_cmd_ps() {
    docker_debug "ps command: $*"
    local all=false format="table"
    
    # Parse options
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -a|--all) all=true; shift ;;
            --format) format="$2"; shift 2 ;;
            *) shift ;;
        esac
    done
    
    if [[ "$format" == "table" ]]; then
        echo "CONTAINER ID   IMAGE           COMMAND          CREATED         STATUS                  PORTS     NAMES"
    fi
    
    for name in "${!DOCKER_CONTAINERS[@]}"; do
        local container_data="${DOCKER_CONTAINERS[$name]}"
        IFS='|' read -r state image created <<< "$container_data"
        
        # Filter running unless --all
        if [[ "$state" == "running" || "$all" == "true" ]]; then
            local id=$(docker_mock_id "$name")
            local status
            case "$state" in
                running) status="Up $(docker_mock_time_ago)" ;;
                stopped) status="Exited (0) $(docker_mock_time_ago)" ;;
                *) status="$state" ;;
            esac
            
            case "$format" in
                "table")
                    printf "%-14s %-15s %-16s %-15s %-22s %-9s %s\n" \
                        "$id" "${image:-test:latest}" '"/entrypoint.sh"' "$created" "$status" "8080/tcp" "$name"
                    ;;
                "{{.Names}}")
                    echo "$name"
                    ;;
                "{{.ID}}")
                    echo "$id"
                    ;;
                "{{.Status}}")
                    echo "$status"
                    ;;
                "json")
                    echo "{\"Names\":\"$name\",\"State\":\"$state\",\"ID\":\"$id\",\"Status\":\"$status\",\"Image\":\"${image:-test:latest}\"}"
                    ;;
                *)
                    echo "$name"
                    ;;
            esac
        fi
    done
}

docker_cmd_images() {
    docker_debug "images command: $*"
    local format="table"
    
    # Parse options
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --format) format="$2"; shift 2 ;;
            *) shift ;;
        esac
    done
    
    if [[ "$format" == "table" ]]; then
        echo "REPOSITORY      TAG       IMAGE ID       CREATED         SIZE"
    fi
    
    for img in "${!DOCKER_IMAGES[@]}"; do
        local img_data="${DOCKER_IMAGES[$img]}"
        IFS='|' read -r available size created <<< "$img_data"
        
        if [[ "$available" == "true" ]]; then
            local id=$(docker_mock_id "$img")
            case "$format" in
                "table")
                    printf "%-14s %-8s %-14s %-15s %s\n" "$img" "latest" "$id" "${created:-$(docker_mock_time_ago)}" "${size:-123MB}"
                    ;;
                "{{.Repository}}")
                    echo "$img"
                    ;;
                "{{.Repository}}:{{.Tag}}")
                    echo "$img:latest"
                    ;;
                *)
                    echo "$img"
                    ;;
            esac
        fi
    done
}

docker_cmd_run() {
    docker_debug "run command: $*"
    local name="" image="" detach=false rm=false
    
    # Parse basic options
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name) name="$2"; shift 2 ;;
            -d|--detach) detach=true; shift ;;
            --rm) rm=true; shift ;;
            -e|--env) shift 2 ;;  # Skip env vars for simplicity
            -p|--publish) shift 2 ;;  # Skip port mapping for simplicity
            -v|--volume) shift 2 ;;   # Skip volumes for simplicity
            --network) shift 2 ;;     # Skip network for simplicity
            -*) shift ;;  # Skip other flags
            *)
                if [[ -z "$image" ]]; then
                    image="$1"
                else
                    break  # Rest are command args
                fi
                shift
                ;;
        esac
    done
    
    # Generate container name if not provided
    if [[ -z "$name" ]]; then
        name="$(echo -n "$image" | tr ':/' '_')_$RANDOM"
    fi
    
    # Check for duplicate container name
    if [[ -n "${DOCKER_CONTAINERS[$name]}" ]]; then
        echo "docker: Error response from daemon: Conflict. Container name \"/$name\" is already in use" >&2
        return 125
    fi
    
    # Check image exists (basic check)
    if [[ -z "${DOCKER_IMAGES[$image]}" && "$image" != *"test"* ]]; then
        echo "Unable to find image '$image' locally" >&2
        return 125
    fi
    
    # Create container
    local id=$(docker_mock_id "$name")
    local created=$(docker_mock_time_ago)
    DOCKER_CONTAINERS[$name]="running|$image|$created"
    
    docker_debug "Created container: $name ($id) from image $image"
    
    if [[ "$detach" == "true" ]]; then
        echo "$id"
    else
        echo "Container started successfully"
        echo "Mock container output for: $image"
        
        # If --rm, remove after execution
        if [[ "$rm" == "true" ]]; then
            unset DOCKER_CONTAINERS[$name]
        else
            DOCKER_CONTAINERS[$name]="stopped|$image|$created"
        fi
    fi
}

docker_cmd_exec() {
    docker_debug "exec command: $*"
    local container="" cmd=()
    
    # Parse: first non-flag is container, rest is command
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -*) shift ;;  # Skip flags like -it
            *)
                if [[ -z "$container" ]]; then
                    container="$1"
                    shift
                    cmd+=("$@")
                    break
                else
                    cmd+=("$1")
                    shift
                fi
                ;;
        esac
    done
    
    # Check container exists
    if [[ -z "${DOCKER_CONTAINERS[$container]}" ]]; then
        echo "Error: No such container: $container" >&2
        return 1
    fi
    
    # Check container is running
    local container_data="${DOCKER_CONTAINERS[$container]}"
    local state="${container_data%%|*}"
    if [[ "$state" != "running" ]]; then
        echo "Error: Container $container is not running" >&2
        return 126
    fi
    
    # Simulate command execution
    local full_cmd="${cmd[*]}"
    case "$full_cmd" in
        *health*|*status*) echo "healthy" ;;
        *version*) echo "1.0.0" ;;
        *ps*|*list*) echo "process1 process2" ;;
        *) echo "Mock exec output for: $full_cmd" ;;
    esac
}

docker_cmd_logs() {
    docker_debug "logs command: $*"
    local container="" follow=false
    
    # Parse options
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -f|--follow) follow=true; shift ;;
            --tail) shift 2 ;;  # Skip tail value
            *) container="$1"; shift ;;
        esac
    done
    
    # Check container exists
    if [[ -z "${DOCKER_CONTAINERS[$container]}" ]]; then
        echo "Error: No such container: $container" >&2
        return 1
    fi
    
    echo "$(date): [INFO] Container $container started"
    echo "$(date): [INFO] Service initialization complete"
    echo "$(date): [INFO] Ready to accept connections"
    
    if [[ "$follow" == "true" ]]; then
        echo "$(date): [INFO] Streaming logs..."
    fi
}

docker_cmd_inspect() {
    docker_debug "inspect command: $*"
    local container="" format=""
    
    # Parse options
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --format) format="$2"; shift 2 ;;
            *) container="$1"; shift ;;
        esac
    done
    
    # Check container exists
    if [[ -z "${DOCKER_CONTAINERS[$container]}" ]]; then
        echo "Error: No such container: $container" >&2
        return 1
    fi
    
    local container_data="${DOCKER_CONTAINERS[$container]}"
    IFS='|' read -r state image created <<< "$container_data"
    
    if [[ -n "$format" ]]; then
        case "$format" in
            "{{.State.Status}}") echo "$state" ;;
            "{{.State.Running}}") [[ "$state" == "running" ]] && echo "true" || echo "false" ;;
            "{{.Config.Image}}") echo "${image:-test:latest}" ;;
            "{{.State.Health.Status}}") [[ "$state" == "running" ]] && echo "healthy" || echo "unhealthy" ;;
            *) echo "mock-format-output" ;;
        esac
    else
        # Return basic JSON
        cat <<EOF
[{
  "Id": "$(docker_mock_id "$container")",
  "Name": "/$container",
  "State": {
    "Status": "$state",
    "Running": $([[ "$state" == "running" ]] && echo "true" || echo "false"),
    "Health": {"Status": "$([[ "$state" == "running" ]] && echo "healthy" || echo "unhealthy")"}
  },
  "Config": {
    "Image": "${image:-test:latest}"
  }
}]
EOF
    fi
}

docker_cmd_pull() {
    docker_debug "pull command: $*"
    local image="$1"
    
    if [[ -z "$image" ]]; then
        echo "Error: missing image name" >&2
        return 1
    fi
    
    echo "Using default tag: latest"
    echo "latest: Pulling from $image"
    echo "Digest: sha256:$(docker_mock_id "$image")abcdef"
    echo "Status: Downloaded newer image for $image:latest"
    
    # Add image to available images
    DOCKER_IMAGES[$image]="true|$(( RANDOM % 500 + 50 ))MB|$(docker_mock_time_ago)"
    docker_debug "Pulled image: $image"
}

docker_cmd_build() {
    docker_debug "build command: $*"
    local tag="" path="."
    
    # Parse basic options
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -t|--tag) tag="$2"; shift 2 ;;
            *) path="$1"; shift ;;
        esac
    done
    
    if [[ -z "$tag" ]]; then
        tag="build-$(date +%s)"
    fi
    
    echo "Step 1/3 : FROM base"
    echo "Step 2/3 : COPY . /app"  
    echo "Step 3/3 : CMD [\"/app/start.sh\"]"
    echo "Successfully built $(docker_mock_id "$tag")"
    echo "Successfully tagged $tag"
    
    # Add built image
    DOCKER_IMAGES[$tag]="true|$(( RANDOM % 200 + 100 ))MB|$(docker_mock_time_ago)"
    docker_debug "Built image: $tag"
}

docker_cmd_start() {
    docker_debug "start command: $*"
    local container="$1"
    
    if [[ -z "$container" ]]; then
        echo "Error: missing container name" >&2
        return 1
    fi
    
    if [[ -z "${DOCKER_CONTAINERS[$container]}" ]]; then
        echo "Error: No such container: $container" >&2
        return 1
    fi
    
    local container_data="${DOCKER_CONTAINERS[$container]}"
    IFS='|' read -r state image created <<< "$container_data"
    DOCKER_CONTAINERS[$container]="running|$image|$created"
    
    echo "$container"
    docker_debug "Started container: $container"
}

docker_cmd_stop() {
    docker_debug "stop command: $*"
    local container="$1"
    
    if [[ -z "$container" ]]; then
        echo "Error: missing container name" >&2
        return 1
    fi
    
    if [[ -z "${DOCKER_CONTAINERS[$container]}" ]]; then
        echo "Error: No such container: $container" >&2
        return 1
    fi
    
    local container_data="${DOCKER_CONTAINERS[$container]}"
    IFS='|' read -r state image created <<< "$container_data"
    DOCKER_CONTAINERS[$container]="stopped|$image|$created"
    
    echo "$container"
    docker_debug "Stopped container: $container"
}

docker_cmd_rm() {
    docker_debug "rm command: $*"
    local force=false
    local containers=()
    
    # Parse options and containers
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -f|--force) force=true; shift ;;
            *) containers+=("$1"); shift ;;
        esac
    done
    
    for container in "${containers[@]}"; do
        if [[ -z "${DOCKER_CONTAINERS[$container]}" ]]; then
            echo "Error: No such container: $container" >&2
            return 1
        fi
        
        local container_data="${DOCKER_CONTAINERS[$container]}"
        local state="${container_data%%|*}"
        
        if [[ "$state" == "running" && "$force" != "true" ]]; then
            echo "Error: Cannot remove running container $container: container is running: stop the container before removing or force remove" >&2
            return 1
        fi
        
        unset DOCKER_CONTAINERS[$container]
        echo "$container"
        docker_debug "Removed container: $container"
    done
}

docker_cmd_rmi() {
    docker_debug "rmi command: $*"
    local images=("$@")
    
    for image in "${images[@]}"; do
        if [[ -z "${DOCKER_IMAGES[$image]}" ]]; then
            echo "Error: No such image: $image" >&2
            return 1
        fi
        
        unset DOCKER_IMAGES[$image]
        echo "Untagged: $image:latest"
        echo "Deleted: sha256:$(docker_mock_id "$image")abcdef"
        docker_debug "Removed image: $image"
    done
}

# === Convention-based Test Functions ===
test_docker_connection() {
    docker_debug "Testing connection..."
    
    local result
    result=$(docker version 2>&1)
    
    if [[ "$result" =~ "Docker version" ]]; then
        docker_debug "Connection test passed"
        return 0
    else
        docker_debug "Connection test failed: $result"
        return 1
    fi
}

test_docker_health() {
    docker_debug "Testing health..."
    
    # Test connection
    test_docker_connection || return 1
    
    # Test basic container operations
    docker pull nginx:latest >/dev/null 2>&1 || return 1
    docker run --name health-test -d nginx:latest >/dev/null 2>&1 || return 1
    docker ps | grep -q "health-test" || return 1
    docker stop health-test >/dev/null 2>&1 || return 1
    docker rm health-test >/dev/null 2>&1 || return 1
    
    docker_debug "Health test passed"
    return 0
}

test_docker_basic() {
    docker_debug "Testing basic operations..."
    
    # Test version
    docker version >/dev/null 2>&1 || return 1
    
    # Test image operations
    docker pull alpine:latest >/dev/null 2>&1 || return 1
    docker images | grep -q "alpine" || return 1
    
    # Test container operations
    docker run --name basic-test -d alpine:latest sleep 10 >/dev/null 2>&1 || return 1
    docker ps | grep -q "basic-test" || return 1
    docker exec basic-test echo "test" >/dev/null 2>&1 || return 1
    docker logs basic-test >/dev/null 2>&1 || return 1
    docker stop basic-test >/dev/null 2>&1 || return 1
    docker rm basic-test >/dev/null 2>&1 || return 1
    
    docker_debug "Basic test passed"
    return 0
}

# === State Management ===
docker_mock_reset() {
    docker_debug "Resetting mock state (called from: ${BASH_SOURCE[1]:-unknown}:${BASH_LINENO[0]:-unknown})"
    
    DOCKER_CONTAINERS=()
    DOCKER_IMAGES=()
    DOCKER_NETWORKS=()
    DOCKER_CONFIG[error_mode]=""
    DOCKER_CONFIG[mode]="normal"
    
    # Create default images for testing
    DOCKER_IMAGES["nginx"]="true|142MB|$(docker_mock_time_ago)"
    DOCKER_IMAGES["alpine"]="true|5.6MB|$(docker_mock_time_ago)"
}

docker_mock_set_error() {
    DOCKER_CONFIG[error_mode]="$1"
    docker_debug "Set error mode: $1"
}

docker_mock_set_mode() {
    DOCKER_CONFIG[mode]="$1"
    docker_debug "Set mode: $1"
}

docker_mock_dump_state() {
    echo "=== Docker Mock State ==="
    echo "Version: ${DOCKER_CONFIG[version]}"
    echo "Mode: ${DOCKER_CONFIG[mode]}"
    echo "Containers: ${#DOCKER_CONTAINERS[@]}"
    for name in "${!DOCKER_CONTAINERS[@]}"; do
        echo "  $name: ${DOCKER_CONTAINERS[$name]}"
    done
    echo "Images: ${#DOCKER_IMAGES[@]}"
    for name in "${!DOCKER_IMAGES[@]}"; do
        echo "  $name: ${DOCKER_IMAGES[$name]}"
    done
    echo "Error Mode: ${DOCKER_CONFIG[error_mode]:-none}"
    echo "==================="
}

docker_mock_create_container() {
    local name="${1:-test_container}"
    local image="${2:-nginx:latest}"
    local state="${3:-running}"
    
    DOCKER_CONTAINERS[$name]="$state|$image|$(docker_mock_time_ago)"
    docker_debug "Added container: $name ($state, $image)"
    echo "$name"
}

# === Export Functions ===
export -f docker
export -f test_docker_connection
export -f test_docker_health  
export -f test_docker_basic
export -f docker_mock_reset
export -f docker_mock_set_error
export -f docker_mock_set_mode
export -f docker_mock_dump_state
export -f docker_mock_create_container
export -f docker_debug
export -f docker_check_error

# Initialize with default state
docker_mock_reset
docker_debug "Docker Tier 2 mock initialized"
# Ensure we return success when sourced
return 0 2>/dev/null || true
