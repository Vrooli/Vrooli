#!/usr/bin/env bash
# Whisper Mock - Tier 2 (Stateful)
# 
# Provides stateful Whisper speech-to-text service mocking for testing:
# - Docker container management (ps, run, stop, start)
# - HTTP API endpoints (health, transcription)
# - Audio processing simulation with transcript responses
# - Model management (tiny, base, small, medium, large)
# - Error injection for resilience testing
#
# Coverage: ~80% of common Whisper operations in 550 lines

# === Configuration ===
declare -gA WHISPER_CONTAINERS=()        # Container management: name -> "state|model|port|image"
declare -gA WHISPER_MODELS=()            # Model information: name -> "size|params|speed"
declare -gA WHISPER_TRANSCRIPTS=()       # Transcription responses: file_hash -> transcript
declare -gA WHISPER_ENDPOINTS=()         # API endpoint tracking: endpoint -> "call_count|last_call"
declare -gA WHISPER_CONFIG=(             # Service configuration
    [port]="8090"
    [base_url]="http://localhost:8090"
    [current_model]="base"
    [service_status]="running"
    [error_mode]=""
    [version]="1.0.0"
)

# Debug mode
declare -g WHISPER_DEBUG="${WHISPER_DEBUG:-}"

# === Helper Functions ===
whisper_debug() {
    [[ -n "$WHISPER_DEBUG" ]] && echo "[MOCK:WHISPER] $*" >&2
}

whisper_check_error() {
    case "${WHISPER_CONFIG[error_mode]}" in
        "connection_refused")
            echo "Connection refused" >&2
            return 1
            ;;
        "service_unavailable")
            echo "Service unavailable" >&2
            return 503
            ;;
        "model_not_found")
            echo "Model not found" >&2
            return 404
            ;;
    esac
    return 0
}

whisper_mock_timestamp() {
    date '+%s' 2>/dev/null || echo '1704067200'
}

whisper_generate_container_id() {
    printf "%012x" $((RANDOM * RANDOM % 281474976710656))
}

whisper_file_hash() {
    local file="$1"
    echo "${file}_${RANDOM}" | md5sum 2>/dev/null | cut -d' ' -f1 || echo "mock_hash_${RANDOM}"
}

# === Docker Command Mock ===
docker() {
    whisper_debug "docker called with: $*"
    
    if ! whisper_check_error; then
        return $?
    fi
    
    if [[ $# -eq 0 ]]; then
        echo "Usage: docker COMMAND"
        return 1
    fi
    
    local cmd="$1"
    shift
    
    case "$cmd" in
        ps)
            whisper_cmd_ps "$@"
            ;;
        run)
            whisper_cmd_run "$@"
            ;;
        stop)
            whisper_cmd_stop "$@"
            ;;
        start)
            whisper_cmd_start "$@"
            ;;
        rm)
            whisper_cmd_rm "$@"
            ;;
        logs)
            whisper_cmd_logs "$@"
            ;;
        inspect)
            whisper_cmd_inspect "$@"
            ;;
        --version)
            echo "Docker version 20.10.8, build 3967b7d"
            ;;
        *)
            echo "docker: '$cmd' is not a docker command." >&2
            return 1
            ;;
    esac
}

# Docker command implementations
whisper_cmd_ps() {
    local show_all=false
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -a|--all) show_all=true; shift ;;
            -*) shift ;;
            *) shift ;;
        esac
    done
    
    echo "CONTAINER ID   IMAGE                                           COMMAND   CREATED       STATUS       PORTS                    NAMES"
    
    for name in "${!WHISPER_CONTAINERS[@]}"; do
        local container_data="${WHISPER_CONTAINERS[$name]}"
        IFS='|' read -r state model port image <<< "$container_data"
        
        if [[ "$show_all" == "true" || "$state" == "running" ]]; then
            local container_id=$(whisper_generate_container_id)
            local status="Up 2 hours"
            [[ "$state" != "running" ]] && status="Exited (0) 1 hour ago"
            local port_mapping="0.0.0.0:$port->$port/tcp"
            [[ "$state" != "running" ]] && port_mapping=""
            
            printf "%-13s %-47s %-9s %-13s %-12s %-24s %s\n" \
                "$container_id" "${image:0:47}" "\"/app\"" "2 hours ago" "$status" "$port_mapping" "$name"
        fi
    done
}

whisper_cmd_run() {
    local container_name="" image="" port_mapping="" detached=false
    local env_vars=()
    
    # Parse Docker run arguments (simplified)
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name) container_name="$2"; shift 2 ;;
            -p) port_mapping="$2"; shift 2 ;;
            -d|--detach) detached=true; shift ;;
            -e) env_vars+=("$2"); shift 2 ;;
            -*) shift ;;
            *) image="$1"; shift ;;
        esac
    done
    
    # Default values
    container_name="${container_name:-whisper}"
    image="${image:-onerahmet/openai-whisper-asr-webservice:latest}"
    local port="8090"
    [[ "$port_mapping" =~ :([0-9]+)/ ]] && port="${BASH_REMATCH[1]}"
    
    # Create container
    local container_id=$(whisper_generate_container_id)
    WHISPER_CONTAINERS[$container_name]="running|${WHISPER_CONFIG[current_model]}|$port|$image"
    WHISPER_CONFIG[service_status]="running"
    
    whisper_debug "Started container: $container_name ($image)"
    
    if [[ "$detached" == "true" ]]; then
        echo "$container_id"
    fi
}

whisper_cmd_stop() {
    local container="$1"
    if [[ -z "$container" ]]; then
        echo "docker stop: missing container name" >&2
        return 1
    fi
    
    if [[ -n "${WHISPER_CONTAINERS[$container]}" ]]; then
        local container_data="${WHISPER_CONTAINERS[$container]}"
        IFS='|' read -r state model port image <<< "$container_data"
        WHISPER_CONTAINERS[$container]="stopped|$model|$port|$image"
        WHISPER_CONFIG[service_status]="stopped"
        whisper_debug "Stopped container: $container"
        echo "$container"
    else
        echo "Error response from daemon: No such container: $container" >&2
        return 1
    fi
}

whisper_cmd_start() {
    local container="$1"
    if [[ -z "$container" ]]; then
        echo "docker start: missing container name" >&2
        return 1
    fi
    
    if [[ -n "${WHISPER_CONTAINERS[$container]}" ]]; then
        local container_data="${WHISPER_CONTAINERS[$container]}"
        IFS='|' read -r state model port image <<< "$container_data"
        WHISPER_CONTAINERS[$container]="running|$model|$port|$image"
        WHISPER_CONFIG[service_status]="running"
        whisper_debug "Started container: $container"
        echo "$container"
    else
        echo "Error response from daemon: No such container: $container" >&2
        return 1
    fi
}

whisper_cmd_rm() {
    local force=false
    local containers=()
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -f|--force) force=true; shift ;;
            -*) shift ;;
            *) containers+=("$1"); shift ;;
        esac
    done
    
    for container in "${containers[@]}"; do
        if [[ -n "${WHISPER_CONTAINERS[$container]}" ]]; then
            local container_data="${WHISPER_CONTAINERS[$container]}"
            local state="${container_data%%|*}"
            
            if [[ "$state" == "running" && "$force" != "true" ]]; then
                echo "Error response from daemon: You cannot remove a running container $container. Stop the container before attempting removal or force remove" >&2
                return 1
            fi
            
            unset WHISPER_CONTAINERS[$container]
            whisper_debug "Removed container: $container"
            echo "$container"
        fi
    done
}

whisper_cmd_logs() {
    local container="$1"
    if [[ -z "$container" ]]; then
        echo "docker logs: missing container name" >&2
        return 1
    fi
    
    if [[ -n "${WHISPER_CONTAINERS[$container]}" ]]; then
        echo "INFO:     Started server process [1]"
        echo "INFO:     Waiting for application startup."
        echo "INFO:     Application startup complete."
        echo "INFO:     Uvicorn running on http://0.0.0.0:${WHISPER_CONFIG[port]} (Press CTRL+C to quit)"
    else
        echo "Error response from daemon: No such container: $container" >&2
        return 1
    fi
}

whisper_cmd_inspect() {
    local container="$1"
    if [[ -z "$container" ]]; then
        echo "docker inspect: missing container name" >&2
        return 1
    fi
    
    if [[ -n "${WHISPER_CONTAINERS[$container]}" ]]; then
        local container_data="${WHISPER_CONTAINERS[$container]}"
        IFS='|' read -r state model port image <<< "$container_data"
        
        cat <<EOF
[
    {
        "Id": "$(whisper_generate_container_id)",
        "State": {
            "Status": "$state",
            "Running": $([ "$state" == "running" ] && echo "true" || echo "false"),
            "ExitCode": 0
        },
        "Config": {
            "Image": "$image",
            "ExposedPorts": {
                "$port/tcp": {}
            }
        },
        "NetworkSettings": {
            "Ports": {
                "$port/tcp": [
                    {
                        "HostIp": "0.0.0.0",
                        "HostPort": "$port"
                    }
                ]
            }
        }
    }
]
EOF
    else
        echo "Error: No such container: $container" >&2
        return 1
    fi
}

# === HTTP API Mock (curl interceptor) ===
curl() {
    whisper_debug "curl called with: $*"
    
    if ! whisper_check_error; then
        return $?
    fi
    
    # Parse curl arguments (simplified)
    local url="" method="GET" data="" output_file=""
    local silent=false headers=()
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -X) method="$2"; shift 2 ;;
            -d|--data) data="$2"; shift 2 ;;
            -s|--silent) silent=true; shift ;;
            -o|--output) output_file="$2"; shift 2 ;;
            -H|--header) headers+=("$2"); shift 2 ;;
            http*) url="$1"; shift ;;
            *) shift ;;
        esac
    done
    
    # Check if this is a Whisper API call
    if [[ "$url" =~ localhost:${WHISPER_CONFIG[port]} || "$url" =~ localhost:8090 ]]; then
        whisper_handle_api_request "$method" "$url" "$data" "$output_file" "$silent"
        return $?
    fi
    
    # Not a Whisper call - return empty response for non-Whisper URLs
    if [[ "$silent" != "true" ]]; then
        echo "curl: Not a Whisper API endpoint"
    fi
    return 0
}

whisper_handle_api_request() {
    local method="$1" url="$2" data="$3" output_file="$4" silent="$5"
    local endpoint=""
    
    whisper_debug "Processing API request: $method $url"
    
    # Extract endpoint from URL
    if [[ "$url" =~ /([a-zA-Z0-9_.-]+)$ ]]; then
        endpoint="${BASH_REMATCH[1]}"
        whisper_debug "Extracted endpoint: $endpoint"
    elif [[ "$url" =~ localhost:[0-9]+/$ || "$url" =~ localhost:[0-9]+$ ]]; then
        endpoint="root"
        whisper_debug "Root endpoint detected"
    else
        endpoint="unknown"
        whisper_debug "Unknown endpoint pattern: $url"
    fi
    
    # Track call count
    local count="${WHISPER_ENDPOINTS[$endpoint]:-0|0}"
    local call_count="${count%%|*}"
    ((call_count++))
    local timestamp=$(whisper_mock_timestamp)
    WHISPER_ENDPOINTS[$endpoint]="$call_count|$timestamp"
    
    whisper_debug "API request: $method $endpoint (call #$call_count)"
    
    # Route to appropriate handler
    case "$endpoint" in
        "health")
            whisper_api_health "$method" "$silent" "$output_file"
            ;;
        "asr")
            whisper_api_transcribe "$method" "$data" "$silent" "$output_file"
            ;;
        "openapi.json")
            whisper_api_openapi "$silent" "$output_file"
            ;;
        "root")
            whisper_api_info "$silent" "$output_file"
            ;;
        *)
            echo "HTTP/1.1 404 Not Found" >&2
            return 404
            ;;
    esac
}

whisper_api_health() {
    local method="$1" silent="$2" output_file="$3"
    
    whisper_debug "Health endpoint called (silent=$silent, output_file=$output_file)"
    
    local response="{\"status\":\"${WHISPER_CONFIG[service_status]}\",\"model\":\"${WHISPER_CONFIG[current_model]}\",\"version\":\"${WHISPER_CONFIG[version]}\"}"
    
    if [[ -n "$output_file" ]]; then
        echo "$response" > "$output_file"
        whisper_debug "Written response to file: $output_file"
    else
        echo "$response"
        whisper_debug "Returned response: $response"
    fi
    
    return 0
}

whisper_api_transcribe() {
    local method="$1" data="$2" silent="$3" output_file="$4"
    
    whisper_debug "Transcribe endpoint called: $method, data=$data, silent=$silent"
    
    if [[ "$method" != "POST" ]]; then
        echo "HTTP/1.1 405 Method Not Allowed" >&2
        return 405
    fi
    
    # Extract audio file from form data (simplified)
    local audio_file="unknown"
    if [[ "$data" =~ audio_file=([^&]+) ]]; then
        audio_file="${BASH_REMATCH[1]}"
        whisper_debug "Extracted audio file: $audio_file"
    fi
    
    # Get or generate transcript
    local file_hash=$(whisper_file_hash "$audio_file")
    local transcript="${WHISPER_TRANSCRIPTS[$file_hash]:-Hello, this is a mock transcription of the audio file.}"
    whisper_debug "Using transcript: $transcript"
    
    local response="{\"text\":\"$transcript\",\"language\":\"en\",\"model\":\"${WHISPER_CONFIG[current_model]}\",\"duration\":2.5}"
    
    if [[ -n "$output_file" ]]; then
        echo "$response" > "$output_file"
        whisper_debug "Written transcription to file: $output_file"
    else
        echo "$response"
        whisper_debug "Returned transcription: $response"
    fi
    
    return 0
}

whisper_api_openapi() {
    local silent="$1" output_file="$2"
    
    local response="{\"openapi\":\"3.0.0\",\"info\":{\"title\":\"OpenAI Whisper ASR\",\"version\":\"${WHISPER_CONFIG[version]}\"},\"paths\":{\"/asr\":{\"post\":{\"summary\":\"Audio transcription\"}},\"/health\":{\"get\":{\"summary\":\"Health check\"}}}}"
    
    if [[ -n "$output_file" ]]; then
        echo "$response" > "$output_file"
    elif [[ "$silent" != "true" ]]; then
        echo "$response"
    fi
    
    return 0
}

whisper_api_info() {
    local silent="$1" output_file="$2"
    
    local response="{\"service\":\"Whisper ASR\",\"version\":\"${WHISPER_CONFIG[version]}\",\"model\":\"${WHISPER_CONFIG[current_model]}\",\"endpoints\":[\"/health\",\"/asr\",\"/openapi.json\"]}"
    
    if [[ -n "$output_file" ]]; then
        echo "$response" > "$output_file"
    elif [[ "$silent" != "true" ]]; then
        echo "$response"
    fi
    
    return 0
}

# === Convention-based Test Functions ===
test_whisper_connection() {
    whisper_debug "Testing connection..."
    
    # Test health endpoint
    local result
    result=$(curl -s "http://localhost:${WHISPER_CONFIG[port]}/health" 2>&1)
    
    if [[ "$result" =~ "status" ]]; then
        whisper_debug "Connection test passed"
        return 0
    else
        whisper_debug "Connection test failed: $result"
        return 1
    fi
}

test_whisper_health() {
    whisper_debug "Testing health..."
    
    # Test connection
    test_whisper_connection || return 1
    
    # Test container operations
    docker run --name health-test -d onerahmet/openai-whisper-asr-webservice:latest >/dev/null 2>&1 || return 1
    docker ps | grep -q "health-test" || return 1
    docker stop health-test >/dev/null 2>&1 || return 1
    docker rm health-test >/dev/null 2>&1 || return 1
    
    whisper_debug "Health test passed"
    return 0
}

test_whisper_basic() {
    whisper_debug "Testing basic operations..."
    
    # Test transcription API
    local result
    result=$(curl -s -X POST "http://localhost:${WHISPER_CONFIG[port]}/asr" -d "audio_file=test.wav" 2>&1)
    [[ "$result" =~ "text" ]] || return 1
    
    # Test container lifecycle
    docker run --name basic-test -d onerahmet/openai-whisper-asr-webservice:latest >/dev/null 2>&1 || return 1
    docker ps | grep -q "basic-test" || return 1
    docker stop basic-test >/dev/null 2>&1 || return 1
    
    # Test model info
    local health_result
    health_result=$(curl -s "http://localhost:${WHISPER_CONFIG[port]}/health" 2>&1)
    [[ "$health_result" =~ "${WHISPER_CONFIG[current_model]}" ]] || return 1
    
    # Cleanup
    docker rm basic-test >/dev/null 2>&1
    
    whisper_debug "Basic test passed"
    return 0
}

# === State Management ===
whisper_mock_reset() {
    whisper_debug "Resetting mock state (called from: ${BASH_SOURCE[1]:-unknown}:${BASH_LINENO[0]:-unknown})"
    
    WHISPER_CONTAINERS=()
    WHISPER_TRANSCRIPTS=()
    WHISPER_ENDPOINTS=()
    WHISPER_CONFIG[error_mode]=""
    WHISPER_CONFIG[service_status]="running"
    WHISPER_CONFIG[current_model]="base"
    
    # Initialize defaults
    whisper_mock_init_defaults
}

whisper_mock_init_defaults() {
    # Default models
    WHISPER_MODELS[tiny]="0.04GB|39M|very_fast"
    WHISPER_MODELS[base]="0.07GB|74M|fast"
    WHISPER_MODELS[small]="0.24GB|244M|medium"
    WHISPER_MODELS[medium]="0.77GB|769M|medium"
    WHISPER_MODELS[large]="1.55GB|1550M|slow"
    WHISPER_MODELS[large-v2]="1.55GB|1550M|slow"
    WHISPER_MODELS[large-v3]="1.55GB|1550M|slow"
    
    # Default container
    WHISPER_CONTAINERS[whisper]="running|base|8090|onerahmet/openai-whisper-asr-webservice:latest"
    
    # Default transcripts
    WHISPER_TRANSCRIPTS[default]="Hello, this is a mock transcription of the audio file."
    WHISPER_TRANSCRIPTS[test]="This is a test transcription for audio processing validation."
}

whisper_mock_set_error() {
    WHISPER_CONFIG[error_mode]="$1"
    whisper_debug "Set error mode: $1"
}

whisper_mock_set_model() {
    local model="$1"
    if [[ -n "${WHISPER_MODELS[$model]}" ]]; then
        WHISPER_CONFIG[current_model]="$model"
        whisper_debug "Set model: $model"
        return 0
    else
        echo "Model not found: $model" >&2
        return 1
    fi
}

whisper_mock_set_service_status() {
    WHISPER_CONFIG[service_status]="$1"
    whisper_debug "Set service status: $1"
}

whisper_mock_add_transcript() {
    local file_hash="$1"
    local transcript="$2"
    WHISPER_TRANSCRIPTS[$file_hash]="$transcript"
    whisper_debug "Added transcript for: $file_hash"
}

whisper_mock_dump_state() {
    echo "=== Whisper Mock State ==="
    echo "Service Status: ${WHISPER_CONFIG[service_status]}"
    echo "Current Model: ${WHISPER_CONFIG[current_model]}"
    echo "Port: ${WHISPER_CONFIG[port]}"
    echo "Containers: ${#WHISPER_CONTAINERS[@]}"
    for name in "${!WHISPER_CONTAINERS[@]}"; do
        echo "  $name: ${WHISPER_CONTAINERS[$name]}"
    done
    echo "Models: ${#WHISPER_MODELS[@]}"
    for model in "${!WHISPER_MODELS[@]}"; do
        echo "  $model: ${WHISPER_MODELS[$model]}"
    done
    echo "Transcripts: ${#WHISPER_TRANSCRIPTS[@]}"
    for hash in "${!WHISPER_TRANSCRIPTS[@]}"; do
        echo "  $hash: ${WHISPER_TRANSCRIPTS[$hash]:0:50}..."
    done
    echo "API Calls: ${#WHISPER_ENDPOINTS[@]}"
    for endpoint in "${!WHISPER_ENDPOINTS[@]}"; do
        echo "  $endpoint: ${WHISPER_ENDPOINTS[$endpoint]}"
    done
    echo "Error Mode: ${WHISPER_CONFIG[error_mode]:-none}"
    echo "=================="
}

whisper_mock_create_container() {
    local name="${1:-whisper}"
    local model="${2:-base}"
    local port="${3:-8090}"
    local state="${4:-running}"
    
    WHISPER_CONTAINERS[$name]="$state|$model|$port|onerahmet/openai-whisper-asr-webservice:latest"
    whisper_debug "Created container: $name ($model, $state)"
    echo "$name"
}

whisper_mock_get_stats() {
    local container="$1"
    if [[ -n "${WHISPER_CONTAINERS[$container]}" ]]; then
        echo "{\"read\":\"$(date)\",\"networks\":{\"eth0\":{\"rx_bytes\":1024,\"tx_bytes\":2048}},\"memory\":{\"usage\":134217728,\"limit\":2147483648}}"
    else
        echo "Error: No such container: $container" >&2
        return 1
    fi
}

# === Export Functions ===
export -f docker curl
export -f test_whisper_connection test_whisper_health test_whisper_basic
export -f whisper_mock_reset whisper_mock_set_error whisper_mock_set_model
export -f whisper_mock_set_service_status whisper_mock_add_transcript
export -f whisper_mock_dump_state whisper_mock_create_container whisper_mock_get_stats
export -f whisper_debug whisper_check_error

# Initialize with defaults
whisper_mock_reset
whisper_debug "Whisper Tier 2 mock initialized"