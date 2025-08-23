#!/usr/bin/env bash
# Docker System Mocks for Bats tests.
# Provides comprehensive Docker command mocking with a clear mock::docker::* namespace.

# Prevent duplicate loading
if [[ "${DOCKER_MOCKS_LOADED:-}" == "true" ]]; then
  return 0
fi
export DOCKER_MOCKS_LOADED="true"

# ----------------------------
# Global mock state & options
# ----------------------------
# Modes: normal, offline, error
export DOCKER_MOCK_MODE="${DOCKER_MOCK_MODE:-normal}"

# Optional: directory to log calls/responses
# export MOCK_RESPONSES_DIR="${MOCK_RESPONSES_DIR:-}"

# In-memory state
declare -A MOCK_DOCKER_CONTAINERS=()  # name -> "state|image|metadata..."
declare -A MOCK_DOCKER_IMAGES=()      # image -> "true"/"false"
declare -A MOCK_DOCKER_NETWORKS=()    # name -> "id"
declare -A MOCK_DOCKER_VOLUMES=()     # name -> "driver"
declare -A MOCK_DOCKER_ERRORS=()      # command -> error_type
declare -A MOCK_DOCKER_CONTAINER_ENV=()  # container -> "KEY=VALUE KEY2=VALUE2..."
declare -A MOCK_DOCKER_CONTAINER_PORTS=() # container -> "8080:80 3000:3000..."
declare -A MOCK_DOCKER_CONTAINER_VOLUMES=() # container -> "/host:/container /data:/data..."
declare -A MOCK_DOCKER_CONTAINER_METADATA=() # container -> additional metadata

# File-based state persistence for subshell access (BATS compatibility)
export DOCKER_MOCK_STATE_FILE="${MOCK_LOG_DIR:-/tmp}/docker_mock_state.$$"

# Initialize state file
_docker_mock_init_state_file() {
  if [[ -n "${DOCKER_MOCK_STATE_FILE}" ]]; then
    {
      echo "declare -A MOCK_DOCKER_CONTAINERS=()"
      echo "declare -A MOCK_DOCKER_IMAGES=()"
      echo "declare -A MOCK_DOCKER_NETWORKS=()"
      echo "declare -A MOCK_DOCKER_VOLUMES=()"
      echo "declare -A MOCK_DOCKER_ERRORS=()"
      echo "declare -A MOCK_DOCKER_CONTAINER_ENV=()"
      echo "declare -A MOCK_DOCKER_CONTAINER_PORTS=()"
      echo "declare -A MOCK_DOCKER_CONTAINER_VOLUMES=()"
      echo "declare -A MOCK_DOCKER_CONTAINER_METADATA=()"
    } > "$DOCKER_MOCK_STATE_FILE"
  fi
}

# Save current state to file
_docker_mock_save_state() {
  if [[ -n "${DOCKER_MOCK_STATE_FILE}" ]]; then
    {
      # Use declare -gA for global associative arrays
      declare -p MOCK_DOCKER_CONTAINERS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_DOCKER_CONTAINERS=()"
      declare -p MOCK_DOCKER_IMAGES 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_DOCKER_IMAGES=()"
      declare -p MOCK_DOCKER_NETWORKS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_DOCKER_NETWORKS=()"
      declare -p MOCK_DOCKER_VOLUMES 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_DOCKER_VOLUMES=()"
      declare -p MOCK_DOCKER_ERRORS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_DOCKER_ERRORS=()"
      declare -p MOCK_DOCKER_CONTAINER_ENV 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_DOCKER_CONTAINER_ENV=()"
      declare -p MOCK_DOCKER_CONTAINER_PORTS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_DOCKER_CONTAINER_PORTS=()"
      declare -p MOCK_DOCKER_CONTAINER_VOLUMES 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_DOCKER_CONTAINER_VOLUMES=()"
      declare -p MOCK_DOCKER_CONTAINER_METADATA 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_DOCKER_CONTAINER_METADATA=()"
    } > "$DOCKER_MOCK_STATE_FILE"
  fi
}

# Load state from file  
_docker_mock_load_state() {
  if [[ -n "${DOCKER_MOCK_STATE_FILE}" && -f "$DOCKER_MOCK_STATE_FILE" ]]; then
    # Use eval to execute in global scope, not function scope
    eval "$(cat "$DOCKER_MOCK_STATE_FILE")" 2>/dev/null || true
  fi
}

# Initialize state file
_docker_mock_init_state_file

# ----------------------------
# Utilities
# ----------------------------
# mock::log_call is now provided by mocks/utils.bash - no need to redefine

_mock_container_id() {
  # Generate Docker-like 64-char hex ID (showing first 12)
  local name="$1"
  # Use a simpler hash that looks like Docker's format
  local hex=$(printf "%s_%s" "$name" "$RANDOM" | sha256sum | cut -d' ' -f1)
  echo "${hex:0:12}"
}

# Alias for backward compatibility
_mock_id12() {
  _mock_container_id "$1"
}

# Format output parser for --format strings
_mock_format_output() {
  local format="$1"
  local container="$2"
  local output="$format"
  
  # Simple implementation - just return the format string
  # In a real implementation, this would parse {{.Field}} patterns
  echo "mock-$container-value"
}

_mock_time_ago() {
  local seconds=$((RANDOM % 3600))
  if ((seconds < 60)); then
    echo "$seconds seconds ago"
  elif ((seconds < 3600)); then
    echo "$((seconds / 60)) minutes ago"
  else
    echo "$((seconds / 3600)) hours ago"
  fi
}


# ----------------------------
# Public functions used by tests
# ----------------------------
mock::docker::reset() {
  # Recreate as associative arrays (not indexed arrays)
  declare -gA MOCK_DOCKER_CONTAINERS=()
  declare -gA MOCK_DOCKER_IMAGES=()
  declare -gA MOCK_DOCKER_NETWORKS=()
  declare -gA MOCK_DOCKER_VOLUMES=()
  declare -gA MOCK_DOCKER_ERRORS=()
  declare -gA MOCK_DOCKER_CONTAINER_ENV=()
  declare -gA MOCK_DOCKER_CONTAINER_PORTS=()
  declare -gA MOCK_DOCKER_CONTAINER_VOLUMES=()
  declare -gA MOCK_DOCKER_CONTAINER_METADATA=()
  
  # Initialize state file for subshell access
  _docker_mock_init_state_file
  
  echo "[MOCK] Docker state reset"
}

# Enable automatic cleanup between tests
mock::docker::enable_auto_cleanup() {
  export DOCKER_MOCK_AUTO_CLEANUP=true
}

# Inject errors for testing failure scenarios
mock::docker::inject_error() {
  local cmd="$1"
  local error_type="${2:-generic}"
  MOCK_DOCKER_ERRORS["$cmd"]="$error_type"
  
  # Save state to file for subshell access
  _docker_mock_save_state
  
  echo "[MOCK] Injected error for docker $cmd: $error_type"
}

# ----------------------------
# Public setters used by tests
# ----------------------------
mock::docker::set_container_state() {
  local name="$1" state="$2" info="${3:-}"
  MOCK_DOCKER_CONTAINERS["$name"]="$state|$info"

  # Save state to file for subshell access
  _docker_mock_save_state

  # Use centralized state logging
  mock::log_state "docker_container_state" "$name" "$state"

  if command -v mock::verify::record_call &>/dev/null; then
    mock::verify::record_call "docker" "set_container_state $name $state"
  fi
  return 0
}

mock::docker::set_image_available() {
  local image="$1" available="$2"
  MOCK_DOCKER_IMAGES["$image"]="$available"
  
  # Save state to file for subshell access
  _docker_mock_save_state
  
  return 0
}

mock::docker::set_network() {
  local name="$1" id="$2"
  MOCK_DOCKER_NETWORKS["$name"]="$id"
  
  # Save state to file for subshell access
  _docker_mock_save_state
  
  return 0
}

# ----------------------------
# docker() main interceptor
# ----------------------------
docker() {
  # Load state from file for subshell access (inline to avoid function scoping issues)
  if [[ -n "${DOCKER_MOCK_STATE_FILE}" && -f "$DOCKER_MOCK_STATE_FILE" ]]; then
    eval "$(cat "$DOCKER_MOCK_STATE_FILE")" 2>/dev/null || true
  fi

  # Use centralized logging and verification
  mock::log_and_verify "docker" "$*"

  case "$DOCKER_MOCK_MODE" in
    offline) echo "ERROR: Cannot connect to the Docker daemon" >&2; return 1 ;;
    error)   echo "ERROR: Docker command failed" >&2; return 1 ;;
  esac

  # Check for injected errors
  local cmd_check="${1:-}"
  if [[ -n "${MOCK_DOCKER_ERRORS[$cmd_check]:-}" ]]; then
    local error_type="${MOCK_DOCKER_ERRORS[$cmd_check]}"
    case "$error_type" in
      network_timeout)
        echo "ERROR: Get \"https://registry-1.docker.io/v2/\": net/http: request canceled (Client.Timeout exceeded)" >&2
        return 1
        ;;
      permission_denied)
        echo "ERROR: permission denied while trying to connect to the Docker daemon socket" >&2
        return 1
        ;;
      container_not_found)
        echo "ERROR: No such container: ${2:-unknown}" >&2
        return 1
        ;;
      image_not_found)
        echo "ERROR: Unable to find image '${2:-unknown}:latest' locally" >&2
        return 125
        ;;
      port_already_in_use)
        echo "ERROR: bind: address already in use" >&2
        return 125
        ;;
      *)
        echo "ERROR: Docker command failed: $error_type" >&2
        return 1
        ;;
    esac
  fi

  # top-level flags
  case "$1" in
    --version) echo "Docker version 24.0.0, build abc123"; return 0 ;;
    -h|--help)
      cat <<'EOF'
Usage: docker [OPTIONS] COMMAND
Commands:
  ps        List containers
  images    List images
  run       Run a command in a new container
  exec      Run a command in a running container
  logs      Fetch logs of a container
  inspect   Return low-level information
  pull      Pull an image
  build     Build an image from a Dockerfile
  start     Start one or more containers
  stop      Stop one or more containers
  restart   Restart one or more containers
  rm        Remove one or more containers
  rmi       Remove one or more images
  network   Manage networks
  volume    Manage volumes
  system    Manage Docker
  compose   Docker Compose (v2)
  version   Show the Docker version information
  info      Display system-wide information
EOF
      return 0
      ;;
  esac

  local cmd="$1"; shift || true
  local exit_code=0
  
  case "$cmd" in
    ps)        mock::docker::ps        "$@" ;;
    images)    mock::docker::images    "$@" ;;
    run)       mock::docker::run       "$@" || exit_code=$? ;;
    exec)      mock::docker::exec      "$@" || exit_code=$? ;;
    logs)      mock::docker::logs      "$@" || exit_code=$? ;;
    inspect)   mock::docker::inspect   "$@" || exit_code=$? ;;
    pull)      mock::docker::pull      "$@" || exit_code=$? ;;
    build)     mock::docker::build     "$@" || exit_code=$? ;;
    start|stop|restart)
               mock::docker::lifecycle "$cmd" "$@" || exit_code=$? ;;
    rm|rmi)    mock::docker::remove    "$cmd" "$@" || exit_code=$? ;;
    network)   mock::docker::network   "$@" || exit_code=$? ;;
    volume)    mock::docker::volume    "$@" || exit_code=$? ;;
    system)    mock::docker::system    "$@" || exit_code=$? ;;
    compose)   mock::docker::compose   "$@" || exit_code=$? ;;   # docker compose ...
    version)   echo "Docker version 24.0.0, build abc123" ;;
    info)
      echo "Client: Docker Engine - Community"
      echo "Server: Docker Engine - Community"
      ;;
    *)
      echo "docker: '$cmd' is not a docker command." >&2
      exit_code=1
      ;;
  esac
  
  # Save state after any command that might have modified it
  _docker_mock_save_state
  
  return $exit_code
}

# ----------------------------
# Subcommand implementations
# ----------------------------
mock::docker::ps() {
  mock::log_and_verify "docker" "ps $*"
  local all=false format="table" filters=()
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -a|--all) all=true; shift ;;
      --format) format="$2"; shift 2 ;;
      --filter) filters+=("$2"); shift 2 ;;
      *) shift ;;
    esac
  done

  if [[ "$format" == "table" ]]; then
    echo "CONTAINER ID   IMAGE           COMMAND          CREATED         STATUS                  PORTS                    NAMES"
  fi

  for name in "${!MOCK_DOCKER_CONTAINERS[@]}"; do
    IFS='|' read -r state image <<<"${MOCK_DOCKER_CONTAINERS[$name]}"
    [[ -z "$image" ]] && image="test:latest"
    # filter running unless --all
    if [[ "$state" == "running" || "$all" == true ]]; then
      local id="$(_mock_id12 "$name")"
      local created="$(_mock_time_ago)"
      local ports="0.0.0.0:8080->8080/tcp"
      local status
      case "$state" in
        running) status="Up $(_mock_time_ago)" ;;
        stopped) status="Exited (0) $(_mock_time_ago)" ;;
        paused)  status="Up $(_mock_time_ago) (Paused)" ;;
        *)       status="$state" ;;
      esac

      if [[ "$format" == "table" ]]; then
        printf "%-14s %-15s %-16s %-15s %-22s %-24s %s\n" \
          "$id" "$image" '"/entrypoint.sh"' "$created" "$status" "$ports" "$name"
      else
        # naive support for non-table formats
        case "$format" in
          "{{.Names}}")          echo "$name" ;;
          "{{.Image}}")          echo "$image" ;;
          "{{.Status}}")         echo "$status" ;;
          "{{.ID}}")             echo "$id" ;;
          "table {{.ID}}\t{{.Names}}\t{{.Status}}") printf "%s\t%s\t%s\n" "$id" "$name" "$status" ;;
          "{{json .}}")          echo "{\"ID\":\"$id\",\"Names\":\"$name\",\"Status\":\"$status\",\"Image\":\"$image\"}" ;;
          "json")                
            # Full JSON format for docker ps --format json (what instance_manager expects)
            local created_at="$(_mock_systemd_time)"
            echo "{\"Names\":\"$name\",\"State\":\"$state\",\"ID\":\"$id\",\"CreatedAt\":\"$created_at\",\"Ports\":\"$ports\",\"Image\":\"$image\",\"Status\":\"$status\"}"
            ;;
          *)                     echo "$name" ;;
        esac
      fi
    fi
  done
  return 0
}

mock::docker::images() {
  mock::log_and_verify "docker" "images $*"
  local format="table"
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --format) format="$2"; shift 2 ;;
      *) shift ;;
    esac
  done

  if [[ "$format" == "table" ]]; then
    echo "REPOSITORY      TAG       IMAGE ID       CREATED         SIZE"
  fi

  for img in "${!MOCK_DOCKER_IMAGES[@]}"; do
    [[ "${MOCK_DOCKER_IMAGES[$img]}" != "true" ]] && continue
    local id="$(_mock_id12 "$img")"
    if [[ "$format" == "table" ]]; then
      printf "%-14s %-8s %-14s %-15s %s\n" "$img" "latest" "$id" "$(_mock_time_ago)" "123MB"
    else
      case "$format" in
        "{{.Repository}}:{{.Tag}}") echo "$img:latest" ;;
        "{{.Repository}}")          echo "$img" ;;
        *)                          echo "$img" ;;
      esac
    fi
  done
  return 0
}

mock::docker::run() {
  mock::log_and_verify "docker" "run $*"
  local name="" image="" detach=false rm=false network=""
  local -a env_vars=() ports=() volumes=() cmd_args=()
  
  # Parse all docker run flags
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --name)  name="$2"; shift 2 ;;
      -d|--detach) detach=true; shift ;;
      --rm) rm=true; shift ;;
      -e|--env)
        env_vars+=("$2")
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
      --network)
        network="$2"
        shift 2
        ;;
      --) shift; cmd_args=("$@"); break ;;
      -*) shift ;;  # ignore other flags for now
      *) 
        if [[ -z "$image" ]]; then 
          image="$1"
        else
          cmd_args+=("$1")
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
  if [[ -n "${MOCK_DOCKER_CONTAINERS[$name]}" ]]; then
    echo "docker: Error response from daemon: Conflict. Container name \"/$name\" is already in use" >&2
    return 125
  fi
  
  # Check for port conflicts
  for port in "${ports[@]}"; do
    local host_port="${port%%:*}"
    # Simple check - in real mock might track all used ports
    if [[ "$host_port" == "80" || "$host_port" == "443" ]]; then
      echo "docker: Error response from daemon: bind: address already in use" >&2
      return 125
    fi
  done
  
  # Check image exists (unless it's being pulled implicitly)
  if [[ "${MOCK_DOCKER_IMAGES[$image]}" != "true" && "$image" != *"test"* ]]; then
    echo "Unable to find image '$image' locally" >&2
    echo "docker: Error response from daemon: pull access denied for $image" >&2
    return 125
  fi
  
  # Store container state and metadata
  local id="$(_mock_id12 "$name")"
  mock::docker::set_container_state "$name" "running" "$image"
  
  # Store container metadata
  [[ ${#env_vars[@]} -gt 0 ]] && MOCK_DOCKER_CONTAINER_ENV["$name"]="${env_vars[*]}"
  [[ ${#ports[@]} -gt 0 ]] && MOCK_DOCKER_CONTAINER_PORTS["$name"]="${ports[*]}"
  [[ ${#volumes[@]} -gt 0 ]] && MOCK_DOCKER_CONTAINER_VOLUMES["$name"]="${volumes[*]}"
  
  # Store additional metadata
  local metadata="rm=$rm"
  [[ -n "$network" ]] && metadata="$metadata|network=$network"
  [[ ${#cmd_args[@]} -gt 0 ]] && metadata="$metadata|cmd=${cmd_args[*]}"
  MOCK_DOCKER_CONTAINER_METADATA["$name"]="$metadata"

  if [[ "$detach" == true ]]; then
    echo "$id"
  else
    # Simulate container output
    echo "Container started with:"
    [[ ${#env_vars[@]} -gt 0 ]] && echo "  Environment: ${env_vars[*]}"
    [[ ${#ports[@]} -gt 0 ]] && echo "  Ports: ${ports[*]}"
    [[ ${#volumes[@]} -gt 0 ]] && echo "  Volumes: ${volumes[*]}"
    echo "Container output simulation"
    
    # If --rm was specified, remove container after "execution"
    if [[ "$rm" == true ]]; then
      unset "MOCK_DOCKER_CONTAINERS[$name]"
      unset "MOCK_DOCKER_CONTAINER_ENV[$name]"
      unset "MOCK_DOCKER_CONTAINER_PORTS[$name]"
      unset "MOCK_DOCKER_CONTAINER_VOLUMES[$name]"
      unset "MOCK_DOCKER_CONTAINER_METADATA[$name]"
    else
      # Mark as stopped since it ran interactively
      mock::docker::set_container_state "$name" "stopped" "$image"
    fi
  fi
  return 0
}

mock::docker::exec() {
  mock::log_and_verify "docker" "exec $*"
  local container="" cmd=()
  # parse: first non-flag is container, rest = command
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -*) shift ;;  # ignore flags (-i -t etc.)
      *)  container="$1"; shift; cmd+=("$@"); break ;;
    esac
  done
  
  # Validate container exists
  if [[ -z "${MOCK_DOCKER_CONTAINERS[$container]}" ]]; then
    echo "Error: No such container: $container" >&2
    return 1
  fi
  
  # Check if running
  IFS='|' read -r state _ <<<"${MOCK_DOCKER_CONTAINERS[$container]}"
  if [[ "$state" != "running" ]]; then
    echo "Error: Container $container is not running" >&2
    return 126
  fi
  
  local full="${cmd[*]}"
  case "$full" in
    *health*|*status*) echo "healthy" ;;
    *version*)         echo "1.0.0" ;;
    *ps*|*list*)       echo "process1 process2" ;;
    *)                 echo "Mock exec output for: $full" ;;
  esac
  return 0
}

mock::docker::logs() {
  mock::log_and_verify "docker" "logs $*"
  local follow=false tail="" container=""
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -f|--follow) follow=true; shift ;;
      --tail) tail="$2"; shift 2 ;;
      *) container="$1"; shift ;;
    esac
  done
  
  # Validate container exists
  if [[ -z "${MOCK_DOCKER_CONTAINERS[$container]}" ]]; then
    echo "Error: No such container: $container" >&2
    return 1
  fi
  echo "$(date): [INFO] Container $container started"
  echo "$(date): [INFO] Service initialization complete"
  echo "$(date): [INFO] Ready to accept connections"
  [[ "$follow" == true ]] && echo "$(date): [INFO] Streaming logs..."
  return 0
}

mock::docker::inspect() {
  mock::log_and_verify "docker" "inspect $*"
  local container="" format=""
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --format) format="$2"; shift 2 ;;
      *) container="$1"; shift ;;
    esac
  done
  
  # Validate container exists
  if [[ -z "${MOCK_DOCKER_CONTAINERS[$container]}" ]]; then
    echo "Error: No such container: $container" >&2
    return 1
  fi

  IFS='|' read -r state image <<<"${MOCK_DOCKER_CONTAINERS[$container]:-stopped|}"
  [[ -z "$image" ]] && image="test:latest"

  if [[ -n "$format" ]]; then
    # Retrieve container metadata
    local env_vars="${MOCK_DOCKER_CONTAINER_ENV[$container]:-}"
    local ports="${MOCK_DOCKER_CONTAINER_PORTS[$container]:-}"
    local volumes="${MOCK_DOCKER_CONTAINER_VOLUMES[$container]:-}"
    local metadata="${MOCK_DOCKER_CONTAINER_METADATA[$container]:-}"
    
    case "$format" in
      "{{.State.Status}}")        echo "$state" ;;
      "{{.State.Health.Status}}") [[ "$state" == "running" ]] && echo "healthy" || echo "unhealthy" ;;
      "{{.State.Running}}")       [[ "$state" == "running" ]] && echo "true" || echo "false" ;;
      "{{.Config.Image}}")        echo "$image" ;;
      "{{.Config.Env}}")          echo "$env_vars" ;;
      "{{.NetworkSettings.Ports}}") echo "$ports" ;;
      "{{.Mounts}}")              echo "$volumes" ;;
      "{{.Name}}")                echo "/$container" ;;
      "{{.Id}}")                  echo "$(_mock_id12 "$container")" ;;
      *)                          
        # Try to extract nested values
        _mock_format_output "$format" "$container"
        ;;
    esac
  else
    # full JSON-ish output
    cat <<EOF
[{
  "Id": "$(_mock_id12 "$container")",
  "Created": "$(date -Iseconds)",
  "State": {
    "Status": "$state",
    "Running": $([[ "$state" == "running" ]] && echo "true" || echo "false"),
    "Health": { "Status": $([[ "$state" == "running" ]] && echo "\"healthy\"" || echo "\"unhealthy\"") }
  },
  "Config": { "Image": "$image", "Env": ["PATH=/usr/local/bin:/usr/bin:/bin"] },
  "NetworkSettings": { "Ports": { "8080/tcp": [{"HostIp": "0.0.0.0", "HostPort": "8080"}] } }
}]
EOF
  fi
  return 0
}

mock::docker::pull() {
  mock::log_and_verify "docker" "pull $*"
  local image="$1"
  echo "Pulling $image..."
  echo "latest: Pulling from ${image%%:*}"
  echo "Digest: sha256:abc123..."
  echo "Status: Downloaded newer image for $image"
  mock::docker::set_image_available "$image" "true"
  return 0
}

mock::docker::build() {
  mock::log_and_verify "docker" "build $*"
  local tag="" context="."
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -t|--tag) tag="$2"; shift 2 ;;
      *) context="$1"; shift ;;
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
  return 0
}

mock::docker::lifecycle() {
  local action="$1" name="$2"
  mock::log_and_verify "docker" "$action $name"
  
  # Validate container exists
  if [[ -z "${MOCK_DOCKER_CONTAINERS[$name]}" ]]; then
    echo "Error: No such container: $name" >&2
    return 1
  fi
  
  case "$action" in
    start)   mock::docker::set_container_state "$name" "running" ;;
    stop)    mock::docker::set_container_state "$name" "stopped" ;;
    restart) mock::docker::set_container_state "$name" "running" ;;
  esac
  echo "$name"
  return 0
}

mock::docker::remove() {
  local cmd="$1"; shift
  mock::log_and_verify "docker" "$cmd $*"
  case "$cmd" in
    rm)
      local name
      for name in "$@"; do
        if [[ -z "${MOCK_DOCKER_CONTAINERS[$name]}" ]]; then
          echo "Error: No such container: $name" >&2
          return 1
        fi
        unset "MOCK_DOCKER_CONTAINERS[$name]"
        echo "$name"
      done
      ;;
    rmi)
      local img
      for img in "$@"; do
        if [[ "${MOCK_DOCKER_IMAGES[$img]}" != "true" ]]; then
          echo "Error: No such image: $img" >&2
          return 1
        fi
        unset "MOCK_DOCKER_IMAGES[$img]"
        echo "Untagged: $img"
      done
      ;;
  esac
  return 0
}

mock::docker::network() {
  local sub="$1"; shift || true
  mock::log_and_verify "docker" "network $sub $*"
  case "$sub" in
    ls)
      echo "NETWORK ID     NAME        DRIVER    SCOPE"
      # print either stored nets or a default
      if ((${#MOCK_DOCKER_NETWORKS[@]})); then
        local k
        for k in "${!MOCK_DOCKER_NETWORKS[@]}"; do
          printf "%-14s %-10s %-9s %s\n" "${MOCK_DOCKER_NETWORKS[$k]}" "$k" "bridge" "local"
        done
      else
        echo "abc123def456   bridge      bridge    local"
        echo "def456ghi789   host        host      local"
      fi
      ;;
    create)
      local name="$1"
      local id="$(_mock_id12 "$name")"
      MOCK_DOCKER_NETWORKS["$name"]="$id"
      echo "$id"
      ;;
    rm)
      local name
      for name in "$@"; do
        unset "MOCK_DOCKER_NETWORKS[$name]"
        echo "$name"
      done
      ;;
    inspect)
      local name="$1"
      local id="${MOCK_DOCKER_NETWORKS[$name]:-$(_mock_id12 "$name")}"
      cat <<EOF
[{"Name":"$name","Id":"$id","Driver":"bridge","Scope":"local"}]
EOF
      ;;
    *)
      echo "Mock network command: $sub $*" ;;
  esac
  return 0
}

mock::docker::volume() {
  local sub="$1"; shift || true
  mock::log_and_verify "docker" "volume $sub $*"
  case "$sub" in
    ls)
      echo "DRIVER    VOLUME NAME"
      if ((${#MOCK_DOCKER_VOLUMES[@]})); then
        local k
        for k in "${!MOCK_DOCKER_VOLUMES[@]}"; do
          printf "%-9s %s\n" "${MOCK_DOCKER_VOLUMES[$k]}" "$k"
        done
      else
        echo "local     test_volume"
      fi
      ;;
    create)
      local name="$1" driver="${2:-local}"
      MOCK_DOCKER_VOLUMES["$name"]="$driver"
      echo "$name"
      ;;
    rm)
      local name
      for name in "$@"; do
        unset "MOCK_DOCKER_VOLUMES[$name]"
        echo "$name"
      done
      ;;
    inspect)
      local name="$1"
      local driver="${MOCK_DOCKER_VOLUMES[$name]:-local}"
      cat <<EOF
[{"Name":"$name","Driver":"$driver"}]
EOF
      ;;
    *)
      echo "Mock volume command: $sub $*" ;;
  esac
  return 0
}

mock::docker::system() {
  local sub="$1"; shift || true
  mock::log_and_verify "docker" "system $sub $*"
  case "$sub" in
    prune) echo "Total reclaimed space: 123MB" ;;
    df)
      cat <<'EOF'
TYPE            TOTAL     ACTIVE    SIZE      RECLAIMABLE
Images          5         2         1.2GB     500MB (41%)
Containers      3         1         100MB     80MB (80%)
EOF
      ;;
    *) echo "Mock system command: $sub $*" ;;
  esac
  return 0
}

# Docker Compose (v2: `docker compose ...`)
mock::docker::compose() {
  mock::log_and_verify "docker" "compose $*"
  local sub="$1"; shift || true
  case "$sub" in
    up)
      echo "Creating network \"test_default\" with the default driver"
      echo "Creating test_service_1 ... done"
      ;;
    down)
      echo "Stopping test_service_1 ... done"
      echo "Removing test_service_1 ... done"
      ;;
    ps)
      echo "NAME              COMMAND        SERVICE        STATUS   PORTS"
      echo "test_service_1    \"/start.sh\"   test_service   running  0.0.0.0:8080->8080/tcp"
      ;;
    *)
      echo "Mock docker compose: $sub $*"
      ;;
  esac
  return 0
}

# ----------------------------
# Optional docker-compose shim
# ----------------------------
# Because function names with hyphens are unreliable, we can optionally
# create a real `docker-compose` binary on PATH that delegates to `docker compose`.
if [[ "${DOCKER_MOCK_CREATE_COMPOSE_SHIM:-false}" == "true" ]]; then
  # pick dir
  DOCKER_MOCK_SHIM_DIR="${DOCKER_MOCK_SHIM_DIR:-"$(mktemp -d)"}"
  mkdir -p "$DOCKER_MOCK_SHIM_DIR"

  cat > "$DOCKER_MOCK_SHIM_DIR/docker-compose" <<'SHIM'
#!/usr/bin/env bash
# delegate to docker compose; relies on exported `docker` function in this env
exec bash -lc 'docker compose "$@"' bash "$@"
SHIM
  chmod +x "$DOCKER_MOCK_SHIM_DIR/docker-compose"
  case ":$PATH:" in
    *":$DOCKER_MOCK_SHIM_DIR:"*) : ;; # already in PATH
    *) export PATH="$DOCKER_MOCK_SHIM_DIR:$PATH" ;;
  esac
fi

# ----------------------------
# Test Helper Functions
# ----------------------------
# Scenario builders for common test patterns
mock::docker::scenario::create_running_stack() {
  local stack_name="${1:-test}"
  local prefix="${2:-}"
  
  # Create typical microservices stack
  mock::docker::set_container_state "${prefix}${stack_name}_db" "running" "postgres:14"
  mock::docker::set_container_state "${prefix}${stack_name}_app" "running" "${stack_name}:latest"
  mock::docker::set_container_state "${prefix}${stack_name}_cache" "running" "redis:alpine"
  
  # Set up networking
  MOCK_DOCKER_NETWORKS["${stack_name}_network"]="$(_mock_container_id "${stack_name}_network")"
  
  # Add some metadata
  MOCK_DOCKER_CONTAINER_ENV["${prefix}${stack_name}_app"]="NODE_ENV=production PORT=3000"
  MOCK_DOCKER_CONTAINER_PORTS["${prefix}${stack_name}_app"]="3000:3000"
  
  echo "[MOCK] Created running stack: $stack_name"
}

# Create a stopped stack (for testing startup scenarios)
mock::docker::scenario::create_stopped_stack() {
  local stack_name="${1:-test}"
  mock::docker::scenario::create_running_stack "$stack_name"
  
  # Stop all containers
  for container in "${!MOCK_DOCKER_CONTAINERS[@]}"; do
    if [[ "$container" == *"${stack_name}"* ]]; then
      IFS='|' read -r _ image _ <<<"${MOCK_DOCKER_CONTAINERS[$container]}"
      mock::docker::set_container_state "$container" "stopped" "$image"
    fi
  done
  
  echo "[MOCK] Created stopped stack: $stack_name"
}

# Assertion helpers
mock::docker::assert::container_running() {
  local name="$1"
  local state="${MOCK_DOCKER_CONTAINERS[$name]%%|*}"
  
  if [[ "$state" != "running" ]]; then
    echo "ASSERTION FAILED: Container '$name' is not running (state: ${state:-not found})" >&2
    return 1
  fi
  return 0
}

mock::docker::assert::container_stopped() {
  local name="$1"
  local state="${MOCK_DOCKER_CONTAINERS[$name]%%|*}"
  
  if [[ "$state" != "stopped" ]]; then
    echo "ASSERTION FAILED: Container '$name' is not stopped (state: ${state:-not found})" >&2
    return 1
  fi
  return 0
}

mock::docker::assert::container_exists() {
  local name="$1"
  
  if [[ -z "${MOCK_DOCKER_CONTAINERS[$name]}" ]]; then
    echo "ASSERTION FAILED: Container '$name' does not exist" >&2
    return 1
  fi
  return 0
}

mock::docker::assert::container_not_exists() {
  local name="$1"
  
  if [[ -n "${MOCK_DOCKER_CONTAINERS[$name]}" ]]; then
    echo "ASSERTION FAILED: Container '$name' exists but should not" >&2
    return 1
  fi
  return 0
}

mock::docker::assert::image_exists() {
  local image="$1"
  
  if [[ "${MOCK_DOCKER_IMAGES[$image]}" != "true" ]]; then
    echo "ASSERTION FAILED: Image '$image' does not exist" >&2
    return 1
  fi
  return 0
}

# Get container info helpers
mock::docker::get::container_env() {
  local name="$1"
  echo "${MOCK_DOCKER_CONTAINER_ENV[$name]:-}"
}

mock::docker::get::container_ports() {
  local name="$1"
  echo "${MOCK_DOCKER_CONTAINER_PORTS[$name]:-}"
}

mock::docker::get::container_volumes() {
  local name="$1"
  echo "${MOCK_DOCKER_CONTAINER_VOLUMES[$name]:-}"
}

# Advanced assertions
mock::docker::assert::port_mapped() {
  local container="$1"
  local port="$2"
  local ports="${MOCK_DOCKER_CONTAINER_PORTS[$container]:-}"
  
  if [[ ! "$ports" =~ $port ]]; then
    echo "ASSERTION FAILED: Port $port not mapped for container '$container'" >&2
    return 1
  fi
  return 0
}

mock::docker::assert::env_set() {
  local container="$1"
  local env_var="$2"
  local env_vars="${MOCK_DOCKER_CONTAINER_ENV[$container]:-}"
  
  if [[ ! "$env_vars" =~ $env_var ]]; then
    echo "ASSERTION FAILED: Environment variable '$env_var' not set for container '$container'" >&2
    return 1
  fi
  return 0
}

# Debug helper
mock::docker::debug::dump_state() {
  echo "=== Docker Mock State Dump ==="
  echo "Containers:"
  for name in "${!MOCK_DOCKER_CONTAINERS[@]}"; do
    echo "  $name: ${MOCK_DOCKER_CONTAINERS[$name]}"
    [[ -n "${MOCK_DOCKER_CONTAINER_ENV[$name]}" ]] && echo "    ENV: ${MOCK_DOCKER_CONTAINER_ENV[$name]}"
    [[ -n "${MOCK_DOCKER_CONTAINER_PORTS[$name]}" ]] && echo "    PORTS: ${MOCK_DOCKER_CONTAINER_PORTS[$name]}"
    [[ -n "${MOCK_DOCKER_CONTAINER_VOLUMES[$name]}" ]] && echo "    VOLUMES: ${MOCK_DOCKER_CONTAINER_VOLUMES[$name]}"
  done
  echo "Images:"
  for img in "${!MOCK_DOCKER_IMAGES[@]}"; do
    echo "  $img: ${MOCK_DOCKER_IMAGES[$img]}"
  done
  echo "Networks:"
  for net in "${!MOCK_DOCKER_NETWORKS[@]}"; do
    echo "  $net: ${MOCK_DOCKER_NETWORKS[$net]}"
  done
  echo "Errors:"
  for cmd in "${!MOCK_DOCKER_ERRORS[@]}"; do
    echo "  $cmd: ${MOCK_DOCKER_ERRORS[$cmd]}"
  done
  echo "=========================="
}

#######################################
# Compatibility aliases for lifecycle tests
#######################################

#######################################
# Alias for set_image_available to match test expectations
# Arguments: $1 - image name, $2 - availability (true/false)
#######################################
mock::docker::set_image_exists() {
    local image="$1"
    local exists="${2:-true}"
    
    if [[ "$exists" == "true" ]]; then
        mock::docker::set_image_available "$image"
    else
        # Remove image from available list
        unset MOCK_DOCKER_IMAGES["$image"]
        _docker_mock_save_state
    fi
}

# ----------------------------
# Export functions into subshells
# ----------------------------
# Exporting lets child bash processes (spawned by scripts under test) inherit mocks.
export -f docker
export -f _mock_id12 _mock_container_id _mock_time_ago _mock_format_output
export -f _docker_mock_init_state_file _docker_mock_save_state _docker_mock_load_state
export -f mock::docker::reset mock::docker::enable_auto_cleanup mock::docker::inject_error
export -f mock::docker::set_container_state
export -f mock::docker::set_image_available
export -f mock::docker::set_network
export -f mock::docker::ps
export -f mock::docker::images
export -f mock::docker::run
export -f mock::docker::exec
export -f mock::docker::logs
export -f mock::docker::inspect
export -f mock::docker::pull
export -f mock::docker::build
export -f mock::docker::lifecycle
export -f mock::docker::remove
export -f mock::docker::network
export -f mock::docker::volume
export -f mock::docker::system
export -f mock::docker::compose

# Export test helper functions
export -f mock::docker::scenario::create_running_stack
export -f mock::docker::scenario::create_stopped_stack
export -f mock::docker::assert::container_running
export -f mock::docker::assert::container_stopped
export -f mock::docker::assert::container_exists
export -f mock::docker::assert::container_not_exists
export -f mock::docker::assert::image_exists
export -f mock::docker::assert::port_mapped
export -f mock::docker::assert::env_set
export -f mock::docker::get::container_env
export -f mock::docker::get::container_ports
export -f mock::docker::get::container_volumes
export -f mock::docker::debug::dump_state
export -f mock::docker::set_image_exists

echo "[MOCK] Docker mocks loaded successfully"
