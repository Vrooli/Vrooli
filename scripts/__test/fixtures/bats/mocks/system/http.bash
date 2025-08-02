#!/usr/bin/env bash
# HTTP System Mocks
# Provides comprehensive HTTP command mocking (curl, wget, etc.)

# Prevent duplicate loading
if [[ "${HTTP_MOCKS_LOADED:-}" == "true" ]]; then
    return 0
fi
export HTTP_MOCKS_LOADED="true"

# HTTP mock state storage
declare -A MOCK_HTTP_ENDPOINTS
declare -A MOCK_HTTP_RESPONSES
declare -A MOCK_HTTP_STATUS_CODES
declare -A MOCK_HTTP_DELAYS

# Global HTTP mock configuration
export HTTP_MOCK_MODE="${HTTP_MOCK_MODE:-normal}"  # normal, offline, slow

#######################################
# Sanitize URL for use as associative array key
# Arguments: $1 - URL
# Returns: sanitized key
#######################################
sanitize_url_key() {
    local url="$1"
    # Replace all problematic characters with underscores, including dots
    echo "$url" | sed 's/[^a-zA-Z0-9_-]/_/g'
}

#######################################
# Set mock HTTP endpoint state
# Arguments: $1 - endpoint URL, $2 - state (healthy/unhealthy/unavailable)
#######################################
mock::http::set_endpoint_state() {
    local endpoint="$1"
    local state="$2"
    local key=$(sanitize_url_key "$endpoint")
    
    MOCK_HTTP_ENDPOINTS["$key"]="$state"
    
    # Set default response based on state
    case "$state" in
        "healthy")
            mock::http::set_endpoint_response "$endpoint" '{"status":"ok","health":"healthy"}' 200
            ;;
        "unhealthy")
            mock::http::set_endpoint_response "$endpoint" '{"status":"error","health":"unhealthy"}' 503
            ;;
        "unavailable")
            mock::http::set_endpoint_response "$endpoint" "" 0  # Connection refused
            ;;
    esac
    
    # Log mock usage
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "http_endpoint_state:$endpoint:$state" >> "${MOCK_RESPONSES_DIR}/used_mocks.log"
    fi
}

#######################################
# Set custom HTTP response for endpoint
# Arguments: $1 - endpoint URL, $2 - response body, $3 - status code
#######################################
mock::http::set_endpoint_response() {
    local endpoint="$1"
    local response="$2"
    local status_code="${3:-200}"
    local key=$(sanitize_url_key "$endpoint")
    
    MOCK_HTTP_RESPONSES["$key"]="$response"
    MOCK_HTTP_STATUS_CODES["$key"]="$status_code"
}

#######################################
# Set HTTP response delay
# Arguments: $1 - endpoint URL, $2 - delay in seconds
#######################################
mock::http::set_endpoint_delay() {
    local endpoint="$1"
    local delay="$2"
    local key=$(sanitize_url_key "$endpoint")
    
    MOCK_HTTP_DELAYS["$key"]="$delay"
}

#######################################
# Set HTTP response sequence (for multiple calls)
# Arguments: $1 - endpoint URL, $2 - comma-separated responses
#######################################
mock::http::set_endpoint_sequence() {
    local endpoint="$1"
    local responses="$2"
    local key=$(sanitize_url_key "$endpoint")
    
    # Store sequence in special format using underscore separators
    MOCK_HTTP_RESPONSES["${key}_sequence"]="$responses"
    MOCK_HTTP_RESPONSES["${key}_sequence_index"]="0"
}

#######################################
# Set endpoint as unreachable (connection refused)
# Arguments: $1 - endpoint URL or base URL
#######################################
mock::http::set_endpoint_unreachable() {
    local endpoint="$1"
    
    # If it's a base URL, mark all common endpoints as unreachable
    if [[ "$endpoint" =~ ^https?://[^/]+/?$ ]]; then
        # Remove trailing slash if present
        endpoint="${endpoint%/}"
        
        # Mark common endpoints as unreachable
        local common_endpoints=("/" "/health" "/api" "/api/v1" "/status")
        for path in "${common_endpoints[@]}"; do
            mock::http::set_endpoint_state "${endpoint}${path}" "unavailable"
        done
        
        # Also mark the base URL itself
        mock::http::set_endpoint_state "$endpoint" "unavailable"
    else
        # Mark specific endpoint as unreachable
        mock::http::set_endpoint_state "$endpoint" "unavailable"
    fi
}

#######################################
# Main curl command mock
#######################################
curl() {
    # Track command calls for verification
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "curl $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    # Handle different HTTP mock modes
    case "$HTTP_MOCK_MODE" in
        "offline")
            echo "curl: (6) Could not resolve host" >&2
            return 6
            ;;
    esac
    
    # Parse curl arguments
    local url=""
    local method="GET"
    local output_file=""
    local headers=()
    local data=""
    local follow_redirects=false
    local silent=false
    local show_headers=false
    local output_http_code=false
    local max_time=""
    local connect_timeout=""
    local fail_on_error=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            "-X"|"--request")
                method="$2"
                shift 2
                ;;
            "-o"|"--output")
                output_file="$2"
                shift 2
                ;;
            "-H"|"--header")
                headers+=("$2")
                shift 2
                ;;
            "-d"|"--data")
                data="$2"
                method="POST"
                shift 2
                ;;
            "-L"|"--location")
                follow_redirects=true
                shift
                ;;
            "-s"|"--silent")
                silent=true
                shift
                ;;
            "-i"|"--include")
                show_headers=true
                shift
                ;;
            "-w"|"--write-out")
                if [[ "$2" == "%{http_code}" ]]; then
                    output_http_code=true
                fi
                shift 2
                ;;
            "--max-time")
                max_time="$2"
                shift 2
                ;;
            "--connect-timeout")
                connect_timeout="$2"
                shift 2
                ;;
            "-f"|"--fail")
                fail_on_error=true
                shift
                ;;
            *)
                if [[ -z "$url" && ! "$1" =~ ^- ]]; then
                    url="$1"
                fi
                shift
                ;;
        esac
    done
    
    # Handle missing URL
    if [[ -z "$url" ]]; then
        echo "curl: no URL specified!" >&2
        return 2
    fi
    
    # Apply delay if configured
    local delay=""
    local url_key=$(sanitize_url_key "$url")
    if [[ -n "${MOCK_HTTP_DELAYS[$url_key]:-}" ]]; then
        delay="${MOCK_HTTP_DELAYS[$url_key]}"
    fi
    if [[ -n "$delay" && "$HTTP_MOCK_MODE" == "slow" ]]; then
        sleep "$delay"
    fi
    
    # Get response for URL
    local response
    local status_code
    curl_get_mock_response "$url" response status_code
    
    # Handle connection timeouts
    if [[ "$status_code" == "0" ]]; then
        if [[ "$silent" != true ]]; then
            echo "curl: (7) Failed to connect to $(echo "$url" | cut -d/ -f3) port 80: Connection refused" >&2
        fi
        return 7
    fi
    
    # Handle HTTP errors with --fail
    if [[ "$fail_on_error" == true && "$status_code" -ge 400 ]]; then
        if [[ "$silent" != true ]]; then
            echo "curl: (22) The requested URL returned error: $status_code" >&2
        fi
        return 22
    fi
    
    # Generate output
    local output=""
    
    if [[ "$show_headers" == true ]]; then
        output+="HTTP/1.1 $status_code OK"$'\n'
        output+="Content-Type: application/json"$'\n'
        output+="Content-Length: ${#response}"$'\n'
        output+=""$'\n'
    fi
    
    output+="$response"
    
    # Write to file or stdout
    if [[ -n "$output_file" ]]; then
        echo "$output" > "$output_file"
    else
        if [[ "$output_http_code" == true ]]; then
            echo "$status_code"
        else
            echo "$output"
        fi
    fi
    
    # Return appropriate exit code
    if [[ "$status_code" -ge 400 ]]; then
        return 1
    else
        return 0
    fi
}

#######################################
# Get mock response for URL
# Arguments: $1 - URL, $2 - response var name, $3 - status code var name
#######################################
curl_get_mock_response() {
    local url="$1"
    local -n response_ref=$2
    local -n status_ref=$3
    local url_key=$(sanitize_url_key "$url")
    
    # Check for sequence responses first
    local sequence_key="${url_key}_sequence"
    local index_key="${url_key}_sequence_index"
    if [[ -n "${MOCK_HTTP_RESPONSES[$sequence_key]:-}" ]]; then
        local sequence="${MOCK_HTTP_RESPONSES[$sequence_key]}"
        local index="${MOCK_HTTP_RESPONSES[$index_key]:-0}"
        
        # Parse sequence
        IFS=',' read -ra responses <<< "$sequence"
        if [[ "$index" -lt "${#responses[@]}" ]]; then
            response_ref="${responses[$index]}"
            # Increment index for next call
            MOCK_HTTP_RESPONSES["$index_key"]=$((index + 1))
        else
            # Use last response if sequence exhausted
            response_ref="${responses[-1]}"
        fi
        status_ref="${MOCK_HTTP_STATUS_CODES[$url_key]:-200}"
        return
    fi
    
    # Check for exact URL match
    if [[ -n "${MOCK_HTTP_RESPONSES[$url_key]:-}" ]]; then
        response_ref="${MOCK_HTTP_RESPONSES[$url_key]}"
        status_ref="${MOCK_HTTP_STATUS_CODES[$url_key]:-200}"
        return
    fi
    
    # Check for pattern matches
    curl_match_url_pattern "$url" response_ref status_ref
}

#######################################
# Match URL against patterns and generate appropriate response
# Arguments: $1 - URL, $2 - response var name, $3 - status code var name
#######################################
curl_match_url_pattern() {
    local url="$1"
    local -n response_ref=$2
    local -n status_ref=$3
    
    # Extract components
    local host=$(echo "$url" | sed 's|https\?://||' | cut -d/ -f1)
    local path=$(echo "$url" | sed 's|https\?://[^/]*||')
    
    # Default response
    response_ref='{"status":"ok"}'
    status_ref="200"
    
    # Pattern matching for common endpoints
    case "$path" in
        *"/health"*|*"/healthz"*)
            response_ref='{"status":"healthy","timestamp":"'$(date -Iseconds)'"}'
            ;;
        *"/status"*)
            response_ref='{"status":"ok","version":"1.0.0"}'
            ;;
        *"/version"*)
            response_ref='{"version":"1.0.0","build":"abc123"}'
            ;;
        *"/api/"*)
            response_ref='{"data":[],"total":0}'
            ;;
        *"/metrics"*)
            response_ref='# HELP test_metric A test metric\ntest_metric 42'
            ;;
        *"/ping"*)
            response_ref='pong'
            ;;
        *"/ready"*)
            response_ref='{"ready":true}'
            ;;
        *"/config"*)
            response_ref='{"config":{"setting1":"value1","setting2":"value2"}}'
            ;;
        *)
            # Resource-specific patterns
            case "$host" in
                *"ollama"*|*"11434"*)
                    curl_ollama_response "$path" response_ref status_ref
                    ;;
                *"whisper"*|*"8090"*)
                    curl_whisper_response "$path" response_ref status_ref
                    ;;
                *"n8n"*|*"5678"*)
                    curl_n8n_response "$path" response_ref status_ref
                    ;;
                *"qdrant"*|*"6333"*)
                    curl_qdrant_response "$path" response_ref status_ref
                    ;;
                *"minio"*|*"9000"*)
                    curl_minio_response "$path" response_ref status_ref
                    ;;
                *)
                    response_ref='{"message":"Mock response for '$(basename "$path")'"}'
                    ;;
            esac
            ;;
    esac
}

#######################################
# Resource-specific response generators
#######################################

curl_ollama_response() {
    local path="$1"
    local -n response_ref=$2
    local -n status_ref=$3
    
    case "$path" in
        *"/api/tags")
            response_ref='{"models":[{"name":"llama3.1:8b","size":4900000000},{"name":"deepseek-r1:8b","size":4700000000}]}'
            ;;
        *"/api/generate")
            response_ref='{"response":"Hello! This is a mock response.","done":true}'
            ;;
        *"/api/pull")
            response_ref='{"status":"downloading","digest":"sha256:abc123","total":1000000,"completed":500000}'
            ;;
        *)
            response_ref='{"version":"0.1.45"}'
            ;;
    esac
}

curl_whisper_response() {
    local path="$1"
    local -n response_ref=$2
    local -n status_ref=$3
    
    case "$path" in
        *"/transcribe")
            response_ref='{"text":"This is a mock transcription result.","language":"en","duration":5.2}'
            ;;
        *"/translate")
            response_ref='{"text":"This is a mock translation result.","language":"en","duration":4.8}'
            ;;
        *)
            response_ref='{"status":"ready","model":"whisper-1"}'
            ;;
    esac
}

curl_n8n_response() {
    local path="$1"
    local -n response_ref=$2
    local -n status_ref=$3
    
    case "$path" in
        *"/api/v1/workflows")
            response_ref='[{"id":"1","name":"Test Workflow","active":true,"nodes":3}]'
            ;;
        *"/healthz")
            response_ref='{"status":"ok"}'
            ;;
        *)
            response_ref='{"version":"1.19.0"}'
            ;;
    esac
}

curl_qdrant_response() {
    local path="$1"
    local -n response_ref=$2
    local -n status_ref=$3
    
    case "$path" in
        *"/collections")
            response_ref='{"result":{"collections":[{"name":"test_collection"}]},"status":"ok"}'
            ;;
        *"/health")
            response_ref='{"status":"ok","version":"1.7.0"}'
            ;;
        *)
            response_ref='{"result":{},"status":"ok"}'
            ;;
    esac
}

curl_minio_response() {
    local path="$1"
    local -n response_ref=$2
    local -n status_ref=$3
    
    case "$path" in
        *"/minio/health/live")
            response_ref='{"status":"ok"}'
            ;;
        *)
            response_ref='{"status":"ok"}'
            ;;
    esac
}

#######################################
# wget command mock
#######################################
wget() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "wget $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    local url=""
    local output_file=""
    local quiet=false
    
    # Parse basic wget arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            "-O"|"--output-document")
                output_file="$2"
                shift 2
                ;;
            "-q"|"--quiet")
                quiet=true
                shift
                ;;
            *)
                if [[ -z "$url" && ! "$1" =~ ^- ]]; then
                    url="$1"
                fi
                shift
                ;;
        esac
    done
    
    if [[ -z "$url" ]]; then
        echo "wget: missing URL" >&2
        return 1
    fi
    
    # Get mock response using curl logic
    local response
    local status_code
    curl_get_mock_response "$url" response status_code
    
    if [[ "$status_code" == "0" ]]; then
        echo "wget: unable to resolve host address" >&2
        return 4
    fi
    
    # Output to file or stdout
    if [[ -n "$output_file" ]]; then
        echo "$response" > "$output_file"
    fi
    
    if [[ "$quiet" != true ]]; then
        echo "HTTP request sent, awaiting response... $status_code OK"
        echo "Saving to: '${output_file:-index.html}'"
    fi
    
    return 0
}

#######################################
# nc (netcat) command mock for port testing
#######################################
nc() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "nc $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    # Simple port connectivity test
    local host=""
    local port=""
    
    # Parse basic nc arguments
    while [[ $# -gt 0 ]]; do
        if [[ "$1" =~ ^[0-9]+$ ]]; then
            port="$1"
        elif [[ ! "$1" =~ ^- ]]; then
            host="$1"
        fi
        shift
    done
    
    # Simulate connection test
    if [[ -n "$host" && -n "$port" ]]; then
        local endpoint="http://${host}:${port}"
        if [[ -n "${MOCK_HTTP_ENDPOINTS[$endpoint]:-}" ]]; then
            local state="${MOCK_HTTP_ENDPOINTS[$endpoint]}"
            if [[ "$state" == "healthy" ]]; then
                return 0
            fi
        fi
    fi
    
    return 1
}

# Export functions
export -f curl wget nc
export -f mock::http::set_endpoint_state mock::http::set_endpoint_response
export -f mock::http::set_endpoint_delay mock::http::set_endpoint_sequence
export -f mock::http::set_endpoint_unreachable
export -f curl_get_mock_response curl_match_url_pattern
export -f curl_ollama_response curl_whisper_response curl_n8n_response
export -f curl_qdrant_response curl_minio_response

echo "[HTTP_MOCKS] HTTP system mocks loaded"