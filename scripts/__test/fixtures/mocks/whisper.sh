#!/usr/bin/env bash
# Whisper Mock Implementation
# Provides comprehensive mock for Whisper speech-to-text service including:
# - Docker container management 
# - HTTP API endpoints (transcription, translation, language detection)
# - Audio file processing simulation
# - Model management
# - Configuration and health status

# Prevent duplicate loading
if [[ "${WHISPER_MOCK_LOADED:-}" == "true" ]]; then
  return 0
fi
export WHISPER_MOCK_LOADED="true"

# Load dependencies
MOCK_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
[[ -f "$MOCK_DIR/logs.sh" ]] && source "$MOCK_DIR/logs.sh"

# ----------------------------
# Global mock state & configuration  
# ----------------------------

# Whisper service configuration
declare -gA WHISPER_MOCK_CONFIG=(
    [port]="8090"
    [container_name]="whisper"
    [base_url]="http://localhost:8090"
    [model]="base"
    [image]="onerahmet/openai-whisper-asr-webservice:latest-gpu"
    [cpu_image]="onerahmet/openai-whisper-asr-webservice:latest"
    [data_dir]="${HOME}/.whisper"
    [models_dir]="${HOME}/.whisper/models"
    [uploads_dir]="${HOME}/.whisper/uploads"
    [gpu_enabled]="false"
    [service_status]="running"
    [health_status]="healthy"
    [version]="1.0.0"
    [api_version]="openai_whisper"
    [current_step]="ready"
    [progress]="100"
)

# Mock data storage
declare -gA WHISPER_MOCK_ENDPOINTS=()        # endpoint -> state|response|status_code
declare -gA WHISPER_MOCK_RESPONSES=()        # endpoint -> response_body  
declare -gA WHISPER_MOCK_STATUS_CODES=()     # endpoint -> status_code
declare -gA WHISPER_MOCK_DELAYS=()           # endpoint -> delay_seconds
declare -gA WHISPER_MOCK_ERRORS=()           # command -> error_type
declare -gA WHISPER_MOCK_HEADERS=()          # endpoint -> headers
declare -gA WHISPER_MOCK_CALL_COUNT=()       # endpoint -> call_count
declare -gA WHISPER_MOCK_TRANSCRIPTS=()      # file_hash -> transcript_data
declare -gA WHISPER_MOCK_MODELS=()           # model_name -> model_info
declare -gA WHISPER_MOCK_CONTAINERS=()       # container_name -> container_state

# Available models with metadata
WHISPER_MOCK_MODELS=(
    [tiny]='{"size":"0.04GB","params":"39M","languages":"99+","speed":"very_fast"}'
    [base]='{"size":"0.07GB","params":"74M","languages":"99+","speed":"fast"}'
    [small]='{"size":"0.24GB","params":"244M","languages":"99+","speed":"medium"}'
    [medium]='{"size":"0.77GB","params":"769M","languages":"99+","speed":"medium"}'
    [large]='{"size":"1.55GB","params":"1550M","languages":"99+","speed":"slow"}'
    [large-v2]='{"size":"1.55GB","params":"1550M","languages":"99+","speed":"slow"}'
    [large-v3]='{"size":"1.55GB","params":"1550M","languages":"99+","speed":"slow"}'
)

# File-based state persistence - will be set after logging is initialized
export WHISPER_MOCK_STATE_FILE=""

# Initialize state file
_whisper_mock_init_state_file() {
  # Set state file path if not already set
  if [[ -z "${WHISPER_MOCK_STATE_FILE}" ]]; then
    export WHISPER_MOCK_STATE_FILE="${MOCK_LOG_DIR:-/tmp}/whisper_mock_state.$$"
  fi
  if [[ -n "${WHISPER_MOCK_STATE_FILE}" ]]; then
    # Ensure the directory exists
    local state_dir="$(dirname "$WHISPER_MOCK_STATE_FILE")"
    if [[ ! -d "$state_dir" ]]; then
      mkdir -p "$state_dir" || {
        echo "[MOCK] WARNING: Could not create state directory: $state_dir" >&2
        # Fallback to /tmp
        export WHISPER_MOCK_STATE_FILE="/tmp/whisper_mock_state.$$"
      }
    fi
    
    {
      echo "# Whisper mock state - $(date)"
      echo "declare -gA WHISPER_MOCK_CONFIG=()"
      echo "declare -gA WHISPER_MOCK_ENDPOINTS=()"
      echo "declare -gA WHISPER_MOCK_RESPONSES=()"
      echo "declare -gA WHISPER_MOCK_STATUS_CODES=()"
      echo "declare -gA WHISPER_MOCK_DELAYS=()"
      echo "declare -gA WHISPER_MOCK_ERRORS=()"
      echo "declare -gA WHISPER_MOCK_HEADERS=()"
      echo "declare -gA WHISPER_MOCK_CALL_COUNT=()"
      echo "declare -gA WHISPER_MOCK_TRANSCRIPTS=()"
      echo "declare -gA WHISPER_MOCK_MODELS=()"
      echo "declare -gA WHISPER_MOCK_CONTAINERS=()"
    } > "$WHISPER_MOCK_STATE_FILE"
  fi
}

# Save current state to file
_whisper_mock_save_state() {
  if [[ -n "${WHISPER_MOCK_STATE_FILE}" ]]; then
    # Ensure the directory exists
    local state_dir="$(dirname "$WHISPER_MOCK_STATE_FILE")"
    if [[ ! -d "$state_dir" ]]; then
      mkdir -p "$state_dir" 2>/dev/null || {
        echo "[MOCK] WARNING: Could not create state directory: $state_dir" >&2
        return 1
      }
    fi
    
    # Use file locking to prevent race conditions in concurrent access
    local lock_file="${WHISPER_MOCK_STATE_FILE}.lock"
    local max_wait=50  # 5 seconds total
    local wait_count=0
    
    # Acquire lock with timeout
    while ! (set -C; echo $$ > "$lock_file") 2>/dev/null; do
      if [[ $wait_count -ge $max_wait ]]; then
        echo "[MOCK] WARNING: Could not acquire state lock after $max_wait attempts" >&2
        return 1
      fi
      sleep 0.1
      ((wait_count++))
    done
    
    # Ensure lock cleanup on exit
    trap 'rm -f "$lock_file" 2>/dev/null' EXIT
    
    # For concurrent access safety, we need to merge call counts
    if [[ -f "$WHISPER_MOCK_STATE_FILE" ]]; then
      # Load current state from file and check call counts
      local -A temp_call_count
      # Extract just the call count line and evaluate it in temp array
      local call_count_line="$(grep '^declare -gA WHISPER_MOCK_CALL_COUNT=' "$WHISPER_MOCK_STATE_FILE" 2>/dev/null | head -1)" || true
      if [[ -n "$call_count_line" ]]; then
        # Replace the declaration to assign to our temp array
        local temp_line="${call_count_line/WHISPER_MOCK_CALL_COUNT=/temp_call_count=}"
        eval "$temp_line" 2>/dev/null || true
        
        # Merge call counts: take the maximum of current and new values
        for key in "${!temp_call_count[@]}"; do
          local file_val="${temp_call_count[$key]:-0}"
          local mem_val="${WHISPER_MOCK_CALL_COUNT[$key]:-0}"
          if [[ $file_val -gt $mem_val ]]; then
            WHISPER_MOCK_CALL_COUNT["$key"]=$file_val
          fi
        done
      fi
    fi
    
    {
      echo "# Whisper mock state - $(date)"
      declare -p WHISPER_MOCK_CONFIG 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA WHISPER_MOCK_CONFIG=()"
      declare -p WHISPER_MOCK_ENDPOINTS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA WHISPER_MOCK_ENDPOINTS=()"
      declare -p WHISPER_MOCK_RESPONSES 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA WHISPER_MOCK_RESPONSES=()"
      declare -p WHISPER_MOCK_STATUS_CODES 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA WHISPER_MOCK_STATUS_CODES=()"
      declare -p WHISPER_MOCK_DELAYS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA WHISPER_MOCK_DELAYS=()"
      declare -p WHISPER_MOCK_ERRORS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA WHISPER_MOCK_ERRORS=()"
      declare -p WHISPER_MOCK_HEADERS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA WHISPER_MOCK_HEADERS=()"
      declare -p WHISPER_MOCK_CALL_COUNT 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA WHISPER_MOCK_CALL_COUNT=()"
      declare -p WHISPER_MOCK_TRANSCRIPTS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA WHISPER_MOCK_TRANSCRIPTS=()"
      declare -p WHISPER_MOCK_MODELS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA WHISPER_MOCK_MODELS=()"
      declare -p WHISPER_MOCK_CONTAINERS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA WHISPER_MOCK_CONTAINERS=()"
    } > "$WHISPER_MOCK_STATE_FILE" 2>/dev/null || {
      rm -f "$lock_file" 2>/dev/null
      echo "[MOCK] WARNING: Could not save state to file: $WHISPER_MOCK_STATE_FILE" >&2
      return 1
    }
    
    # Release lock
    rm -f "$lock_file" 2>/dev/null
    trap - EXIT
  fi
}

# Load state from file
_whisper_mock_load_state() {
  if [[ -n "${WHISPER_MOCK_STATE_FILE}" && -f "$WHISPER_MOCK_STATE_FILE" ]]; then
    # Wait for any ongoing write operations to complete
    local lock_file="${WHISPER_MOCK_STATE_FILE}.lock"
    local max_wait=20  # 2 seconds total
    local wait_count=0
    
    while [[ -f "$lock_file" ]]; do
      if [[ $wait_count -ge $max_wait ]]; then
        break  # Proceed anyway after timeout
      fi
      sleep 0.1
      ((wait_count++))
    done
    
    eval "$(cat "$WHISPER_MOCK_STATE_FILE")" 2>/dev/null || true
  fi
}

# Initialize state file
_whisper_mock_init_state_file

# ----------------------------
# Utilities
# ----------------------------

_sanitize_endpoint_key() {
  local endpoint="$1"
  echo "$endpoint" | sed 's/[^a-zA-Z0-9_-]/_/g'
}

_generate_request_id() {
  local endpoint="$1"
  printf "%s_%s" "$endpoint" "$RANDOM" | sha256sum | cut -d' ' -f1 | head -c 8
}

_current_timestamp() {
  date -Iseconds 2>/dev/null || date '+%Y-%m-%dT%H:%M:%S%z'
}

_calculate_file_hash() {
  local file="$1"
  if [[ -f "$file" ]]; then
    sha256sum "$file" 2>/dev/null | cut -d' ' -f1
  else
    echo "${file}_${RANDOM}" | sha256sum | cut -d' ' -f1
  fi
}

_simulate_processing_time() {
  local file_size="${1:-1024}"  # bytes
  local model="${2:-base}"
  
  # Simple processing time based on model (avoid floating point arithmetic)
  case "$model" in
    tiny) echo "0.5" ;;
    base) echo "1.2" ;;
    small) echo "2.1" ;;
    medium) echo "4.3" ;;
    large*) echo "6.7" ;;
    *) echo "1.0" ;;
  esac
}

# ----------------------------
# Public mock configuration functions
# ----------------------------

mock::whisper::reset() {
  # Clear all state
  declare -gA WHISPER_MOCK_CONFIG=(
      [port]="8090"
      [container_name]="whisper"
      [base_url]="http://localhost:8090"
      [model]="base"
      [service_status]="running"
      [health_status]="healthy"
      [version]="1.0.0"
      [current_step]="ready"
      [progress]="100"
  )
  
  declare -gA WHISPER_MOCK_ENDPOINTS=()
  declare -gA WHISPER_MOCK_RESPONSES=()
  declare -gA WHISPER_MOCK_STATUS_CODES=()
  declare -gA WHISPER_MOCK_DELAYS=()
  declare -gA WHISPER_MOCK_ERRORS=()
  declare -gA WHISPER_MOCK_HEADERS=()
  declare -gA WHISPER_MOCK_CALL_COUNT=()
  declare -gA WHISPER_MOCK_TRANSCRIPTS=()
  declare -gA WHISPER_MOCK_CONTAINERS=()
  
  # Initialize default container state
  WHISPER_MOCK_CONTAINERS["whisper"]="running"
  
  # Reset models to defaults
  WHISPER_MOCK_MODELS=(
      [tiny]='{"size":"0.04GB","params":"39M","languages":"99+","speed":"very_fast"}'
      [base]='{"size":"0.07GB","params":"74M","languages":"99+","speed":"fast"}'
      [small]='{"size":"0.24GB","params":"244M","languages":"99+","speed":"medium"}'
      [medium]='{"size":"0.77GB","params":"769M","languages":"99+","speed":"medium"}'
      [large]='{"size":"1.55GB","params":"1550M","languages":"99+","speed":"slow"}'
      [large-v2]='{"size":"1.55GB","params":"1550M","languages":"99+","speed":"slow"}'
      [large-v3]='{"size":"1.55GB","params":"1550M","languages":"99+","speed":"slow"}'
  )
  
  # Set up healthy endpoints by default
  mock::whisper::setup_healthy_endpoints
  
  # Ensure logging is initialized (for proper state file path)
  if [[ -z "${MOCK_LOG_DIR:-}" ]]; then
    mock::init_logging >/dev/null 2>&1 || true
  fi
  
  # Reinitialize state file
  _whisper_mock_init_state_file
  _whisper_mock_save_state
  
  echo "[MOCK] Whisper state reset"
}

mock::whisper::set_config() {
  local key="$1"
  local value="$2"
  
  WHISPER_MOCK_CONFIG["$key"]="$value"
  
  # Update base_url if port changes
  if [[ "$key" == "port" ]]; then
    WHISPER_MOCK_CONFIG["base_url"]="http://localhost:$value"
  fi
  
  # If version changes, clear the static health endpoint so dynamic response is used
  if [[ "$key" == "version" ]]; then
    local health_key="$(_sanitize_endpoint_key "/health")"
    unset WHISPER_MOCK_RESPONSES["$health_key"]
    unset WHISPER_MOCK_STATUS_CODES["$health_key"]
  fi
  
  _whisper_mock_save_state
  
  mock::log_state "whisper_config" "$key" "$value"
}

mock::whisper::set_service_status() {
  local status="$1"
  
  WHISPER_MOCK_CONFIG[service_status]="$status"
  
  # Set default container state based on service status
  local container_name="${WHISPER_MOCK_CONFIG[container_name]}"
  case "$status" in
    running)
      WHISPER_MOCK_CONTAINERS["$container_name"]="running"
      mock::whisper::setup_healthy_endpoints
      ;;
    stopped)
      WHISPER_MOCK_CONTAINERS["$container_name"]="exited"
      mock::whisper::setup_stopped_endpoints
      ;;
    installing)
      WHISPER_MOCK_CONTAINERS["$container_name"]="running"
      mock::whisper::setup_installing_endpoints
      ;;
    unhealthy)
      WHISPER_MOCK_CONTAINERS["$container_name"]="running" 
      mock::whisper::setup_unhealthy_endpoints
      ;;
  esac
  
  _whisper_mock_save_state
  mock::log_state "whisper_service_status" "$container_name" "$status"
}

mock::whisper::set_model() {
  local model="$1"
  
  if [[ -n "${WHISPER_MOCK_MODELS[$model]}" ]]; then
    WHISPER_MOCK_CONFIG[model]="$model"
    _whisper_mock_save_state
    mock::log_state "whisper_model" "current" "$model"
    echo "[MOCK] Whisper model set to: $model"
  else
    echo "[MOCK] ERROR: Unknown Whisper model: $model" >&2
    return 1
  fi
}

mock::whisper::inject_error() {
  local command="$1"
  local error_type="${2:-generic}"
  
  WHISPER_MOCK_ERRORS["$command"]="$error_type"
  _whisper_mock_save_state
  
  echo "[MOCK] Injected error for $command: $error_type"
}

mock::whisper::add_transcript() {
  local file_pattern="$1"
  local transcript_data="$2"
  
  WHISPER_MOCK_TRANSCRIPTS["$file_pattern"]="$transcript_data"
  _whisper_mock_save_state
  
  echo "[MOCK] Added transcript for pattern: $file_pattern"
}

# ----------------------------
# Endpoint setup functions
# ----------------------------

mock::whisper::setup_healthy_endpoints() {
  local base_url="${WHISPER_MOCK_CONFIG[base_url]}"
  local model="${WHISPER_MOCK_CONFIG[model]}"
  local version="${WHISPER_MOCK_CONFIG[version]}"
  
  # OpenAPI documentation endpoint
  mock::whisper::set_endpoint_response "/openapi.json" \
    '{"info":{"title":"Whisper ASR Webservice","version":"'$version'","description":"OpenAI Whisper Automatic Speech Recognition"},"paths":{"/asr":{"post":{"summary":"Transcribe audio"}},"/detect-language":{"post":{"summary":"Detect language"}}}}' 200
  
  # API documentation
  mock::whisper::set_endpoint_response "/docs" \
    '<!DOCTYPE html><html><head><title>Whisper ASR API</title></head><body><h1>Whisper ASR Webservice API</h1><p>OpenAI Whisper for audio transcription</p><ul><li><a href="/openapi.json">OpenAPI Spec</a></li><li><a href="/asr">Transcription endpoint</a></li></ul></body></html>' 200
  
  # NOTE: /health, /asr, and /detect-language endpoints are handled dynamically
  # Do not set static responses for these as they need to respond to different methods and states
}

mock::whisper::setup_unhealthy_endpoints() {
  local base_url="${WHISPER_MOCK_CONFIG[base_url]}"
  
  # OpenAPI still works but shows degraded status
  mock::whisper::set_endpoint_response "/openapi.json" \
    '{"info":{"title":"Whisper ASR Webservice","version":"1.0.0"},"error":"Service degraded"}' 200
  
  # NOTE: /health, /asr, and /detect-language endpoints are handled dynamically
  # They will return appropriate error responses based on service_status
}

mock::whisper::setup_installing_endpoints() {
  # OpenAPI may work during installation
  mock::whisper::set_endpoint_response "/openapi.json" \
    '{"info":{"title":"Whisper ASR Webservice","version":"1.0.0"},"status":"initializing"}' 200
  
  # NOTE: /health, /asr, and /detect-language endpoints are handled dynamically
  # They will return appropriate installing status responses based on service_status and progress
}

mock::whisper::setup_stopped_endpoints() {
  # All endpoints unreachable - set them to return connection refused
  for endpoint in "/openapi.json" "/docs" "/health" "/asr" "/detect-language" "/"; do
    mock::whisper::set_endpoint_unreachable "$endpoint"
  done
}

mock::whisper::set_endpoint_response() {
  local endpoint="$1"
  local response="$2" 
  local status_code="${3:-200}"
  local headers="${4:-}"
  
  local key="$(_sanitize_endpoint_key "$endpoint")"
  
  WHISPER_MOCK_RESPONSES["$key"]="$response"
  WHISPER_MOCK_STATUS_CODES["$key"]="$status_code"
  [[ -n "$headers" ]] && WHISPER_MOCK_HEADERS["$key"]="$headers"
  
  _whisper_mock_save_state
}

mock::whisper::set_endpoint_unreachable() {
  local endpoint="$1"
  
  # Skip empty endpoints
  [[ -z "$endpoint" ]] && return 0
  
  local key="$(_sanitize_endpoint_key "$endpoint")"
  
  # Skip empty keys  
  [[ -z "$key" ]] && return 0
  
  WHISPER_MOCK_RESPONSES["$key"]=""
  WHISPER_MOCK_STATUS_CODES["$key"]="0"  # Connection refused
  
  _whisper_mock_save_state
}

# ----------------------------
# Command implementations
# ----------------------------

# Docker command interception for Whisper
docker() {
  # Load state from file for subshell access
  _whisper_mock_load_state
  
  mock::log_and_verify "docker" "$*"
  
  # Check for injected errors first
  if [[ -n "${WHISPER_MOCK_ERRORS[docker]}" ]]; then
    local error_type="${WHISPER_MOCK_ERRORS[docker]}"
    case "$error_type" in
      connection_failed)
        echo "Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?" >&2
        return 1
        ;;
      *)
        echo "docker: Error response from daemon: $error_type" >&2
        return 1
        ;;
    esac
  fi
  
  local command="$1"
  shift
  
  case "$command" in
    ps)
      mock::whisper::docker_ps "$@"
      ;;
    run)
      mock::whisper::docker_run "$@"
      ;;
    stop)
      mock::whisper::docker_stop "$@"
      ;;
    start)
      mock::whisper::docker_start "$@"
      ;;
    rm)
      mock::whisper::docker_rm "$@"
      ;;
    logs)
      mock::whisper::docker_logs "$@"
      ;;
    pull)
      mock::whisper::docker_pull "$@"
      ;;
    inspect)
      mock::whisper::docker_inspect "$@"
      ;;
    image)
      mock::whisper::docker_image "$@"
      ;;
    stats)
      mock::whisper::docker_stats "$@"
      ;;
    *)
      echo "docker: '$command' is not a docker command. See 'docker --help'" >&2
      return 1
      ;;
  esac
}

mock::whisper::docker_ps() {
  local container_name="${WHISPER_MOCK_CONFIG[container_name]}"
  local format=""
  local all=false
  
  # Parse docker ps arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --format) format="$2"; shift 2 ;;
      -a|--all) all=true; shift ;;
      *) shift ;;
    esac
  done
  
  local container_state="${WHISPER_MOCK_CONTAINERS[$container_name]:-}"
  
  # Only show running containers by default, all with -a
  if [[ "$container_state" == "running" ]] || [[ "$all" == true && -n "$container_state" ]]; then
    if [[ -n "$format" ]]; then
      case "$format" in
        "{{.Names}}")
          echo "$container_name"
          ;;
        "table {{.Container}}"*"{{.CPUPerc}}"*"{{.MemUsage}}")
          echo "CONTAINER   CPU %   MEM USAGE"
          echo "$container_name   0.5%    128MiB / 2GiB"
          ;;
        *)
          echo "$container_name"
          ;;
      esac
    else
      local status="Up 2 hours"
      [[ "$container_state" != "running" ]] && status="Exited (0) 1 minute ago"
      
      echo "CONTAINER ID   IMAGE     COMMAND   CREATED   STATUS   PORTS   NAMES"
      echo "abc123def456   whisper   \"/start\"   1 hour ago   $status   0.0.0.0:${WHISPER_MOCK_CONFIG[port]}->9000/tcp   $container_name"
    fi
  fi
  
  return 0
}

mock::whisper::docker_run() {
  local container_name=""
  local port_mapping=""
  local detached=false
  local image=""
  
  # Parse docker run arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -d|--detach) detached=true; shift ;;
      --name) container_name="$2"; shift 2 ;;
      -p) port_mapping="$2"; shift 2 ;;
      -v) shift 2 ;;  # Skip volume mounts
      -e) shift 2 ;;  # Skip environment variables
      --gpus) shift 2 ;;  # Skip GPU args
      --restart) shift 2 ;;  # Skip restart policy
      -*) shift ;;  # Skip other flags
      *)
        if [[ -z "$image" ]]; then
          image="$1"
        fi
        shift
        ;;
    esac
  done
  
  # Default container name if not specified
  [[ -z "$container_name" ]] && container_name="${WHISPER_MOCK_CONFIG[container_name]}"
  
  # Update state
  WHISPER_MOCK_CONTAINERS["$container_name"]="running"
  WHISPER_MOCK_CONFIG[service_status]="running"
  
  # Set up healthy endpoints
  mock::whisper::setup_healthy_endpoints
  
  _whisper_mock_save_state
  
  if [[ "$detached" == true ]]; then
    echo "$(_generate_request_id "$container_name")"
  else
    echo "Starting Whisper container: $container_name"
    echo "Model: ${WHISPER_MOCK_CONFIG[model]}"
    echo "Port: ${WHISPER_MOCK_CONFIG[port]}"
    echo "Ready to accept requests"
  fi
  
  return 0
}

mock::whisper::docker_stop() {
  local container_name="$1"
  [[ -z "$container_name" ]] && container_name="${WHISPER_MOCK_CONFIG[container_name]}"
  
  if [[ "${WHISPER_MOCK_CONTAINERS[$container_name]}" == "running" ]]; then
    WHISPER_MOCK_CONTAINERS["$container_name"]="exited"
    WHISPER_MOCK_CONFIG[service_status]="stopped"
    mock::whisper::setup_stopped_endpoints
    _whisper_mock_save_state
    echo "$container_name"
    return 0
  else
    echo "Error response from daemon: No such container: $container_name" >&2
    return 1
  fi
}

mock::whisper::docker_start() {
  local container_name="$1"
  [[ -z "$container_name" ]] && container_name="${WHISPER_MOCK_CONFIG[container_name]}"
  
  if [[ -n "${WHISPER_MOCK_CONTAINERS[$container_name]}" ]]; then
    WHISPER_MOCK_CONTAINERS["$container_name"]="running"
    WHISPER_MOCK_CONFIG[service_status]="running"
    mock::whisper::setup_healthy_endpoints
    _whisper_mock_save_state
    
    # Small delay to ensure state is written before subsequent operations
    sleep 0.05
    
    echo "$container_name"
    return 0
  else
    echo "Error response from daemon: No such container: $container_name" >&2
    return 1
  fi
}

mock::whisper::docker_rm() {
  local container_name="$1"
  [[ -z "$container_name" ]] && container_name="${WHISPER_MOCK_CONFIG[container_name]}"
  
  if [[ -n "${WHISPER_MOCK_CONTAINERS[$container_name]}" ]]; then
    unset WHISPER_MOCK_CONTAINERS["$container_name"]
    WHISPER_MOCK_CONFIG[service_status]="stopped"
    _whisper_mock_save_state
    echo "$container_name"
    return 0
  else
    echo "Error response from daemon: No such container: $container_name" >&2
    return 1
  fi
}

mock::whisper::docker_logs() {
  local container_name="$1"
  local follow=false
  local tail_lines=""
  
  # Parse logs arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -f|--follow) follow=true; shift ;;
      --tail) tail_lines="$2"; shift 2 ;;
      *) 
        if [[ -z "$container_name" ]]; then
          container_name="$1"
        fi
        shift 
        ;;
    esac
  done
  
  [[ -z "$container_name" ]] && container_name="${WHISPER_MOCK_CONFIG[container_name]}"
  
  if [[ -n "${WHISPER_MOCK_CONTAINERS[$container_name]}" ]]; then
    local model="${WHISPER_MOCK_CONFIG[model]}"
    local status="${WHISPER_MOCK_CONFIG[service_status]}"
    
    echo "Starting Whisper ASR Webservice..."
    echo "Loading model: $model"
    case "$status" in
      installing)
        echo "Downloading model files..."
        echo "Progress: ${WHISPER_MOCK_CONFIG[progress]:-50}%"
        echo "Current step: ${WHISPER_MOCK_CONFIG[current_step]:-Downloading model}"
        ;;
      running)
        echo "Model loaded successfully"
        echo "Server ready on port ${WHISPER_MOCK_CONFIG[port]}"
        echo "ASR engine: openai_whisper"
        echo "Available at: ${WHISPER_MOCK_CONFIG[base_url]}"
        ;;
      unhealthy)
        echo "Error loading model: Model initialization failed"
        echo "Check model files and GPU availability"
        ;;
    esac
    
    return 0
  else
    echo "Error response from daemon: No such container: $container_name" >&2
    return 1
  fi
}

mock::whisper::docker_pull() {
  local image="$1"
  
  echo "Pulling from library/${image##*/}"
  echo "latest: Pulling from ${image##*/}"
  echo "abc123def456: Pull complete"
  echo "def456ghi789: Pull complete" 
  echo "ghi789jkl012: Pull complete"
  echo "Digest: sha256:$(_generate_request_id "$image")"
  echo "Status: Downloaded newer image for $image"
  
  return 0
}

mock::whisper::docker_inspect() {
  local container_name="$1"
  local format=""
  
  # Parse inspect arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --format) format="$2"; shift 2 ;;
      *) 
        if [[ -z "$container_name" ]]; then
          container_name="$1"
        fi
        shift 
        ;;
    esac
  done
  
  [[ -z "$container_name" ]] && container_name="${WHISPER_MOCK_CONFIG[container_name]}"
  
  local container_state="${WHISPER_MOCK_CONTAINERS[$container_name]:-}"
  
  if [[ -n "$container_state" ]]; then
    if [[ -n "$format" ]]; then
      case "$format" in
        "{{.State.Status}}")
          echo "$container_state"
          ;;
        "{{.NetworkSettings.IPAddress}}")
          echo "172.17.0.2"
          ;;
        *)
          echo "$container_state"
          ;;
      esac
    else
      echo '[
  {
    "Id": "'$(_generate_request_id "$container_name")'",
    "Name": "/'$container_name'",
    "State": {
      "Status": "'$container_state'",
      "Running": '$([[ "$container_state" == "running" ]] && echo "true" || echo "false")',
      "Pid": '$([[ "$container_state" == "running" ]] && echo "1234" || echo "0")'
    },
    "NetworkSettings": {
      "IPAddress": "172.17.0.2",
      "Ports": {
        "9000/tcp": [{"HostIp": "0.0.0.0", "HostPort": "'${WHISPER_MOCK_CONFIG[port]}'"}]
      }
    }
  }
]'
    fi
    return 0
  else
    echo "Error response from daemon: No such container: $container_name" >&2
    return 1
  fi
}

mock::whisper::docker_image() {
  local subcommand="$1"
  shift
  
  case "$subcommand" in
    inspect)
      local image="$1"
      echo '[
  {
    "Id": "sha256:'$(_generate_request_id "$image")'", 
    "Size": 2000000000,
    "Created": "'$(_current_timestamp)'"
  }
]'
      ;;
    ls)
      echo "REPOSITORY                                  TAG       IMAGE ID       CREATED        SIZE"
      echo "onerahmet/openai-whisper-asr-webservice    latest    abc123def456   2 hours ago    2GB"
      ;;
    *)
      echo "docker image: '$subcommand' is not a valid subcommand" >&2
      return 1
      ;;
  esac
}

mock::whisper::docker_stats() {
  local container_name="${WHISPER_MOCK_CONFIG[container_name]}"
  local no_stream=false
  local format=""
  
  # Parse stats arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --no-stream) no_stream=true; shift ;;
      --format) format="$2"; shift 2 ;;
      *) 
        if [[ -z "$container_name" ]]; then
          container_name="$1"
        fi
        shift 
        ;;
    esac
  done
  
  if [[ -n "${WHISPER_MOCK_CONTAINERS[$container_name]}" ]]; then
    if [[ -n "$format" ]]; then
      echo "$format" | sed 's/{{\.Container}}/'"$container_name"'/g; s/{{\.CPUPerc}}/0.5%/g; s/{{\.MemUsage}}/128MiB \/ 2GiB/g'
    else
      echo "CONTAINER ID   NAME        CPU %     MEM USAGE / LIMIT     MEM %     NET I/O       BLOCK I/O     PIDS"
      echo "abc123def456   $container_name   0.50%     128MiB / 2GiB         6.25%     1.2kB / 648B   0B / 0B       1"
    fi
    return 0
  else
    echo "Error response from daemon: No such container: $container_name" >&2
    return 1
  fi
}

# cURL command interception for Whisper API
curl() {
  # Load state from file for subshell access
  _whisper_mock_load_state
  
  mock::log_and_verify "curl" "$*"
  
  local url="" method="GET" output_file="" data="" form_data=()
  local silent=false show_headers=false output_http_code=false
  local write_out="" max_time=""
  
  # Parse curl arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -X|--request) method="$2"; shift 2 ;;
      -o|--output) output_file="$2"; shift 2 ;;
      -d|--data) data="$2"; method="POST"; shift 2 ;;
      -F|--form) form_data+=("$2"); method="POST"; shift 2 ;;
      -s|--silent) silent=true; shift ;;
      -i|--include) show_headers=true; shift ;;
      -w|--write-out) write_out="$2"; output_http_code=true; shift 2 ;;
      --max-time) max_time="$2"; shift 2 ;;
      -*) shift ;;  # Ignore other flags
      *)
        if [[ -z "$url" && ! "$1" =~ ^- ]]; then
          url="$1"
        fi
        shift
        ;;
    esac
  done
  
  if [[ -z "$url" ]]; then
    echo "curl: no URL specified!" >&2
    return 2
  fi
  
  # Extract endpoint from URL
  local endpoint
  if [[ "$url" =~ ^https?://[^/]+(.*)$ ]]; then
    endpoint="${BASH_REMATCH[1]}"
  else
    endpoint="$url"
  fi
  [[ -z "$endpoint" ]] && endpoint="/"
  
  # Track call count
  local key="$(_sanitize_endpoint_key "$endpoint")"
  local count="${WHISPER_MOCK_CALL_COUNT[$key]:-0}"
  WHISPER_MOCK_CALL_COUNT["$key"]=$((count + 1))
  _whisper_mock_save_state
  
  # Handle special Whisper endpoints
  local response status_code
  # Pass variable names as strings for the function to use with eval
  local form_args=("${form_data[@]}")
  form_args+=("response" "status_code")
  mock::whisper::get_endpoint_response "$endpoint" "$method" "${form_args[@]}"
  
  # Check for injected errors for this specific command
  local cmd_check="curl"
  if [[ -n "${WHISPER_MOCK_ERRORS[$cmd_check]}" ]]; then
    local error_type="${WHISPER_MOCK_ERRORS[$cmd_check]}"
    case "$error_type" in
      connection_failed|connection_refused)
        if [[ "$silent" != true ]]; then
          echo "curl: (7) Failed to connect to localhost port ${WHISPER_MOCK_CONFIG[port]}: Connection refused" >&2
        fi
        return 7
        ;;
      *)
        if [[ "$silent" != true ]]; then
          echo "curl: Generic error: $error_type" >&2
        fi
        return 1
        ;;
    esac
  fi
  
  # Handle connection failures
  if [[ "$status_code" == "0" ]]; then
    # Always output connection refused errors, even with silent flag
    # This matches real curl behavior where connection errors are still shown
    echo "curl: (7) Failed to connect to localhost port ${WHISPER_MOCK_CONFIG[port]}: Connection refused" >&2
    return 7
  fi
  
  # Generate output
  local output=""
  
  # Add headers if requested
  if [[ "$show_headers" == true ]]; then
    local status_text="OK"
    case "$status_code" in
      200) status_text="OK" ;;
      201) status_text="Created" ;;
      400) status_text="Bad Request" ;;
      422) status_text="Unprocessable Entity" ;;
      503) status_text="Service Unavailable" ;;
    esac
    
    output+="HTTP/1.1 $status_code $status_text"$'\n'
    output+="Content-Type: application/json"$'\n' 
    output+="Content-Length: ${#response}"$'\n'
    output+="Server: Whisper-ASR/1.0"$'\n'
    output+=""$'\n'
  fi
  
  output+="$response"
  
  # Handle output
  if [[ -n "$output_file" ]]; then
    echo "$output" > "$output_file"
  else
    if [[ "$output_http_code" == true ]]; then
      echo "$output"
      case "$write_out" in
        "%{http_code}"|"\\n%{http_code}"|"\n%{http_code}")
          echo "$status_code"
          ;;
        *)
          echo "$status_code"
          ;;
      esac
    else
      echo "$output"
    fi
  fi
  
  # Save state after any command that might have modified it
  _whisper_mock_save_state

  # Return appropriate exit code based on HTTP status
  if [[ "$status_code" -ge 400 ]]; then
    return 1
  else
    return 0
  fi
}

mock::whisper::get_endpoint_response() {
  local endpoint="$1"
  local method="${2:-GET}"
  shift 2
  
  # Get response and status variable names (last two parameters)
  local num_args=$#
  local response_var_name="${@: -2:1}"
  local status_var_name="${@: -1:1}"
  
  # Remove response and status var names from form_data
  local form_data=("${@:1:$((num_args-2))}")
  
  # Use eval for variable assignment 
  local response_value status_value
  
  local key="$(_sanitize_endpoint_key "$endpoint")"
  
  # Check for exact endpoint match (including empty responses for unreachable endpoints)
  if [[ -v "WHISPER_MOCK_RESPONSES[$key]" ]]; then
    response_value="${WHISPER_MOCK_RESPONSES[$key]}"
    status_value="${WHISPER_MOCK_STATUS_CODES[$key]:-200}"
  else
    # Handle Whisper-specific endpoints with dynamic responses
    case "$endpoint" in
      "/asr")
        # Pass the original variable names, not local variable names
        local asr_args=("${form_data[@]}")
        asr_args+=("$response_var_name" "$status_var_name")
        mock::whisper::handle_transcription_request "$method" "${asr_args[@]}"
        return
        ;;
      "/detect-language")
        # Pass the original variable names, not local variable names
        local detect_args=("${form_data[@]}")
        detect_args+=("$response_var_name" "$status_var_name")
        mock::whisper::handle_language_detection "$method" "${detect_args[@]}"
        return
        ;;
      "/openapi.json")
        response_value='{"info":{"title":"Whisper ASR Webservice","version":"'${WHISPER_MOCK_CONFIG[version]}'","description":"OpenAI Whisper for speech recognition"},"paths":{"/asr":{"post":{"summary":"Audio transcription"}},"/detect-language":{"post":{"summary":"Language detection"}}}}'
        status_value="200"
        ;;
      "/docs")
        response_value='<!DOCTYPE html><html><head><title>Whisper ASR API</title></head><body><h1>Whisper ASR Webservice</h1><p>OpenAI Whisper automatic speech recognition API</p><ul><li><a href="/openapi.json">OpenAPI Spec</a></li><li><a href="/asr">Transcription endpoint</a></li></ul></body></html>'
        status_value="200"
        ;;
      "/health")
        local service_status="${WHISPER_MOCK_CONFIG[service_status]}"
        case "$service_status" in
          running)
            response_value='{"status":"healthy","model":"'${WHISPER_MOCK_CONFIG[model]}'","version":"'${WHISPER_MOCK_CONFIG[version]}'","timestamp":"'$(_current_timestamp)'"}'
            status_value="200"
            ;;
          installing)
            response_value='{"status":"installing","progress":'${WHISPER_MOCK_CONFIG[progress]}',"current_step":"'${WHISPER_MOCK_CONFIG[current_step]}'","timestamp":"'$(_current_timestamp)'"}'
            status_value="200"
            ;;
          *)
            response_value='{"status":"unhealthy","error":"Service not available","timestamp":"'$(_current_timestamp)'"}'
            status_value="503"
            ;;
        esac
        ;;
      "/" | "")
        response_value='{"service":"Whisper ASR","version":"'${WHISPER_MOCK_CONFIG[version]}'","model":"'${WHISPER_MOCK_CONFIG[model]}'","status":"'${WHISPER_MOCK_CONFIG[service_status]}'","endpoints":["/asr","/detect-language","/docs","/openapi.json"]}'
        status_value="200"
        ;;
      *)
        # Default response for unknown endpoints
        response_value='{"error":"Not found","message":"Endpoint not found: '$endpoint'","available_endpoints":["/asr","/detect-language","/health","/docs"]}'
        status_value="404"
        ;;
    esac
  fi
  
  # Assign to the caller's variables using eval
  eval "$response_var_name=\"\$response_value\""
  eval "$status_var_name=\"\$status_value\""
}

mock::whisper::handle_transcription_request() {
  local method="$1"
  shift
  
  # Get variable names from last two parameters
  local num_args=$#
  local response_var_name="${@: -2:1}"
  local status_var_name="${@: -1:1}"
  
  # Remove variable names from form_data
  local form_data=("${@:1:$((num_args-2))}")
  
  local response_value status_value
  
  if [[ "$method" != "POST" ]]; then
    response_value='{"error":"Method not allowed","detail":"Transcription requires POST method","code":405}'
    status_value="405"
    eval "$response_var_name=\"\$response_value\""
    eval "$status_var_name=\"\$status_value\""
    return
  fi
  
  local audio_file=""
  local task="transcribe"
  local language=""
  local model="${WHISPER_MOCK_CONFIG[model]}"
  
  # Parse form data
  for field in "${form_data[@]}"; do
    if [[ "$field" =~ ^audio_file=@(.+)$ ]]; then
      audio_file="${BASH_REMATCH[1]}"
    elif [[ "$field" =~ ^task=(.+)$ ]]; then
      task="${BASH_REMATCH[1]}"
    elif [[ "$field" =~ ^language=(.+)$ ]]; then  
      language="${BASH_REMATCH[1]}"
    elif [[ "$field" =~ ^model=(.+)$ ]]; then
      model="${BASH_REMATCH[1]}"
    fi
  done
  
  # Validate audio file
  if [[ -z "$audio_file" ]]; then
    response_value='{"error":"Bad Request","detail":"Missing required field: audio_file","code":400}'
    status_value="400"
    eval "$response_var_name=\"\$response_value\""
    eval "$status_var_name=\"\$status_value\""
    return
  fi
  
  # Check service status  
  local service_status="${WHISPER_MOCK_CONFIG[service_status]}"
  if [[ "$service_status" != "running" ]]; then
    if [[ "$service_status" == "installing" ]]; then
      response_value='{"error":"Service initializing","detail":"Whisper service is currently installing","code":503}'
    else
      response_value='{"error":"Service Unavailable","detail":"Whisper service is not ready","code":503}'
    fi
    status_value="503"
    eval "$response_var_name=\"\$response_value\""
    eval "$status_var_name=\"\$status_value\""
    return
  fi
  
  # Generate response based on inputs
  local transcription_text="This is a test audio file for transcription validation."
  local duration=5.2
  local filename=$(basename "$audio_file")
  
  # First, check for custom transcript patterns
  local custom_transcript_found=false
  for pattern in "${!WHISPER_MOCK_TRANSCRIPTS[@]}"; do
    if [[ "$filename" == "$pattern" ]]; then
      # Return custom transcript directly
      response_value="${WHISPER_MOCK_TRANSCRIPTS[$pattern]}"
      status_value="200"
      eval "$response_var_name=\"\$response_value\""
      eval "$status_var_name=\"\$status_value\""
      return
    elif [[ "$pattern" == *"*"* || "$pattern" == *"."* || "$pattern" == *"["* ]] && [[ "$filename" =~ $pattern ]]; then
      # Handle regex patterns like ".*pattern.*", "*.mp3", "[abc].wav", etc.
      response_value="${WHISPER_MOCK_TRANSCRIPTS[$pattern]}"
      status_value="200"
      eval "$response_var_name=\"\$response_value\""
      eval "$status_var_name=\"\$status_value\""
      return
    fi
  done
  
  # Only use default logic if no custom transcript was found
  # Handle translation task
  if [[ "$task" == "translate" ]]; then
    transcription_text="This is the English translation of the provided audio."
  fi
  
  # Handle different file types based on filename
  if [[ "$filename" =~ (speech|voice) ]]; then
    transcription_text="Hello, this is a speech recognition test. The audio quality is good and the transcription should be accurate."
    duration=8.5
  elif [[ "$filename" =~ (silent|quiet) ]]; then
    transcription_text=""
    duration=1.0
  fi
  
  response_value='{"text":"'$transcription_text'","segments":[{"start":0.0,"end":'$duration',"text":"'$transcription_text'"}],"language":"'${language:-en}'","duration":'$duration',"model":"'$model'","task":"'$task'","timestamp":"2025-08-06T00:15:00-04:00"}'
  status_value="200"
  
  eval "$response_var_name=\"\$response_value\""
  eval "$status_var_name=\"\$status_value\""
}

mock::whisper::handle_language_detection() {
  local method="$1"
  shift
  
  # Get variable names from last two parameters
  local num_args=$#
  local response_var_name="${@: -2:1}"
  local status_var_name="${@: -1:1}"
  
  # Remove variable names from form_data
  local form_data=("${@:1:$((num_args-2))}")
  
  local response_value status_value
  
  if [[ "$method" != "POST" ]]; then
    response_value='{"error":"Method not allowed","detail":"Language detection requires POST method","code":405}'
    status_value="405"
    eval "$response_var_name=\"\$response_value\""
    eval "$status_var_name=\"\$status_value\""
    return
  fi
  
  local audio_file=""
  
  # Parse form data to find audio file
  for field in "${form_data[@]}"; do
    if [[ "$field" =~ ^audio_file=@(.+)$ ]]; then
      audio_file="${BASH_REMATCH[1]}"
      break
    fi
  done
  
  if [[ -z "$audio_file" ]]; then
    response_value='{"error":"Bad Request","detail":"Missing required field: audio_file","code":400}'
    status_value="400"
    eval "$response_var_name=\"\$response_value\""
    eval "$status_var_name=\"\$status_value\""
    return
  fi
  
  # Check service status
  local service_status="${WHISPER_MOCK_CONFIG[service_status]}"
  if [[ "$service_status" != "running" ]]; then
    if [[ "$service_status" == "installing" ]]; then
      response_value='{"error":"Service initializing","detail":"Language detection service is currently installing","code":503}'
    else
      response_value='{"error":"Service Unavailable","detail":"Language detection service not ready","code":503}'
    fi
    status_value="503"
    eval "$response_var_name=\"\$response_value\""
    eval "$status_var_name=\"\$status_value\""
    return
  fi
  
  # Simulate language detection based on filename
  local detected_lang="en"
  local confidence=0.95
  local file_pattern=$(basename "$audio_file")
  
  # Detect language hints from filename - only match specific patterns to avoid false positives
  if [[ "$file_pattern" == *"spanish"* ]] || [[ "$file_pattern" == *"español"* ]] || [[ "$file_pattern" == *"_es."* ]] || [[ "$file_pattern" == *"-es."* ]]; then
    detected_lang="es"
  elif [[ "$file_pattern" == *"french"* ]] || [[ "$file_pattern" == *"français"* ]] || [[ "$file_pattern" == *"_fr."* ]] || [[ "$file_pattern" == *"-fr."* ]]; then
    detected_lang="fr"
  elif [[ "$file_pattern" == *"german"* ]] || [[ "$file_pattern" == *"deutsch"* ]] || [[ "$file_pattern" == *"_de."* ]] || [[ "$file_pattern" == *"-de."* ]]; then
    detected_lang="de"
  elif [[ "$file_pattern" == *"italian"* ]] || [[ "$file_pattern" == *"italiano"* ]] || [[ "$file_pattern" == *"_it."* ]] || [[ "$file_pattern" == *"-it."* ]]; then
    detected_lang="it"
  elif [[ "$file_pattern" == *"portuguese"* ]] || [[ "$file_pattern" == *"português"* ]] || [[ "$file_pattern" == *"_pt."* ]] || [[ "$file_pattern" == *"-pt."* ]]; then
    detected_lang="pt"
  elif [[ "$file_pattern" == *"japanese"* ]] || [[ "$file_pattern" == *"日本語"* ]] || [[ "$file_pattern" == *"_ja."* ]] || [[ "$file_pattern" == *"-ja."* ]]; then
    detected_lang="ja"
    confidence=0.88
  elif [[ "$file_pattern" == *"chinese"* ]] || [[ "$file_pattern" == *"中文"* ]] || [[ "$file_pattern" == *"_zh."* ]] || [[ "$file_pattern" == *"-zh."* ]]; then
    detected_lang="zh"
    confidence=0.88
  elif [[ "$file_pattern" == *"russian"* ]] || [[ "$file_pattern" == *"русский"* ]] || [[ "$file_pattern" == *"_ru."* ]] || [[ "$file_pattern" == *"-ru."* ]]; then
    detected_lang="ru"
    confidence=0.90
  fi
  
  # Simple probability distribution - avoid bc dependency
  local probabilities='{"'$detected_lang'": '$confidence', "en": 0.02, "es": 0.01}'
  
  response_value='{"language_code":"'$detected_lang'","confidence":'$confidence',"probabilities":'$probabilities',"timestamp":"2025-08-06T00:15:00-04:00"}'
  status_value="200"
  
  # Assign to caller variables
  eval "$response_var_name=\"\$response_value\""
  eval "$status_var_name=\"\$status_value\""
}

# ----------------------------
# Test helper functions  
# ----------------------------

mock::whisper::assert::service_running() {
  _whisper_mock_load_state
  local container_name="${WHISPER_MOCK_CONFIG[container_name]}"
  if [[ "${WHISPER_MOCK_CONTAINERS[$container_name]}" == "running" ]]; then
    return 0
  else
    echo "ASSERTION FAILED: Whisper service is not running (container: $container_name, state: ${WHISPER_MOCK_CONTAINERS[$container_name]:-unset})" >&2
    return 1
  fi
}

mock::whisper::assert::service_stopped() {
  _whisper_mock_load_state
  local container_name="${WHISPER_MOCK_CONFIG[container_name]}"
  if [[ "${WHISPER_MOCK_CONTAINERS[$container_name]}" != "running" ]]; then
    return 0
  else
    echo "ASSERTION FAILED: Whisper service should be stopped (container: $container_name, state: ${WHISPER_MOCK_CONTAINERS[$container_name]:-unset})" >&2
    return 1
  fi
}

mock::whisper::assert::endpoint_called() {
  local endpoint="$1"
  local expected_count="${2:-1}"
  local key="$(_sanitize_endpoint_key "$endpoint")"
  local actual_count="${WHISPER_MOCK_CALL_COUNT[$key]:-0}"
  
  if [[ "$actual_count" -ge "$expected_count" ]]; then
    return 0
  else
    echo "ASSERTION FAILED: Endpoint '$endpoint' called $actual_count times, expected at least $expected_count" >&2
    return 1
  fi
}

mock::whisper::assert::transcription_contains() {
  local pattern="$1"
  local expected_text="$2"
  
  if [[ -n "${WHISPER_MOCK_TRANSCRIPTS[$pattern]}" ]]; then
    local transcript="${WHISPER_MOCK_TRANSCRIPTS[$pattern]}"
    if [[ "$transcript" =~ $expected_text ]]; then
      return 0
    else
      echo "ASSERTION FAILED: Transcript for '$pattern' does not contain '$expected_text'" >&2
      return 1
    fi
  else
    echo "ASSERTION FAILED: No transcript found for pattern '$pattern'" >&2
    return 1
  fi
}

# Get information helpers
mock::whisper::get::call_count() {
  _whisper_mock_load_state
  local endpoint="$1"
  local key="$(_sanitize_endpoint_key "$endpoint")"
  echo "${WHISPER_MOCK_CALL_COUNT[$key]:-0}"
}

mock::whisper::get::service_status() {
  _whisper_mock_load_state  
  echo "${WHISPER_MOCK_CONFIG[service_status]}"
}

mock::whisper::get::current_model() {
  _whisper_mock_load_state
  echo "${WHISPER_MOCK_CONFIG[model]}"
}

# Debug helpers
mock::whisper::debug::dump_state() {
  _whisper_mock_load_state
  
  echo "=== Whisper Mock State Dump ==="
  echo "Configuration:"
  for key in "${!WHISPER_MOCK_CONFIG[@]}"; do
    echo "  $key: ${WHISPER_MOCK_CONFIG[$key]}"
  done
  
  echo "Containers:"
  for name in "${!WHISPER_MOCK_CONTAINERS[@]}"; do
    echo "  $name: ${WHISPER_MOCK_CONTAINERS[$name]}"
  done
  
  echo "Endpoints:"
  for endpoint in "${!WHISPER_MOCK_RESPONSES[@]}"; do
    echo "  $endpoint: ${WHISPER_MOCK_STATUS_CODES[$endpoint]:-200}"
    [[ -n "${WHISPER_MOCK_CALL_COUNT[$endpoint]}" ]] && echo "    calls: ${WHISPER_MOCK_CALL_COUNT[$endpoint]}"
  done
  
  echo "Transcripts:"
  for pattern in "${!WHISPER_MOCK_TRANSCRIPTS[@]}"; do
    echo "  $pattern: ${WHISPER_MOCK_TRANSCRIPTS[$pattern]:0:50}..."
  done
  
  echo "Errors:"
  for cmd in "${!WHISPER_MOCK_ERRORS[@]}"; do
    echo "  $cmd: ${WHISPER_MOCK_ERRORS[$cmd]}"
  done
  
  echo "==============================="
}

mock::whisper::debug::list_models() {
  echo "=== Available Whisper Models ==="
  for model in "${!WHISPER_MOCK_MODELS[@]}"; do
    echo "  $model: ${WHISPER_MOCK_MODELS[$model]}"
  done  
  echo "Current model: ${WHISPER_MOCK_CONFIG[model]}"
  echo "================================"
}

# ----------------------------
# Export functions
# ----------------------------

# Export private utility functions for subshell access
export -f _sanitize_endpoint_key _generate_request_id _current_timestamp
export -f _calculate_file_hash _simulate_processing_time
export -f _whisper_mock_init_state_file _whisper_mock_save_state _whisper_mock_load_state

# Export main command overrides
export -f docker curl

# Export public functions
export -f mock::whisper::reset
export -f mock::whisper::set_config mock::whisper::set_service_status
export -f mock::whisper::set_model mock::whisper::inject_error mock::whisper::add_transcript
export -f mock::whisper::setup_healthy_endpoints mock::whisper::setup_unhealthy_endpoints
export -f mock::whisper::setup_installing_endpoints mock::whisper::setup_stopped_endpoints
export -f mock::whisper::set_endpoint_response mock::whisper::set_endpoint_unreachable
export -f mock::whisper::get_endpoint_response
export -f mock::whisper::handle_transcription_request mock::whisper::handle_language_detection
export -f mock::whisper::docker_ps mock::whisper::docker_run mock::whisper::docker_stop
export -f mock::whisper::docker_start mock::whisper::docker_rm mock::whisper::docker_logs
export -f mock::whisper::docker_pull mock::whisper::docker_inspect
export -f mock::whisper::docker_image mock::whisper::docker_stats
export -f mock::whisper::assert::service_running mock::whisper::assert::service_stopped
export -f mock::whisper::assert::endpoint_called mock::whisper::assert::transcription_contains
export -f mock::whisper::get::call_count mock::whisper::get::service_status
export -f mock::whisper::get::current_model
export -f mock::whisper::debug::dump_state mock::whisper::debug::list_models

# Initialize mock state if not already done
# Load any existing state first
_whisper_mock_load_state

# Ensure associative array is declared first
declare -gA WHISPER_MOCK_CONFIG 2>/dev/null || true
if [[ -z "${WHISPER_MOCK_CONFIG[service_status]:-}" ]]; then
  echo "[MOCK] Initializing Whisper mock state..."
  mock::whisper::reset
fi

echo "[MOCK] Whisper mock loaded successfully"