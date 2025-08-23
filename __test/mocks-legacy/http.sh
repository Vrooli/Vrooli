#!/usr/bin/env bash
# HTTP System Mocks for Bats tests.
# Provides comprehensive HTTP command mocking with a clear mock::http::* namespace.

# Prevent duplicate loading
if [[ "${HTTP_MOCKS_LOADED:-}" == "true" ]]; then
  return 0
fi
export HTTP_MOCKS_LOADED="true"

# ----------------------------
# Global mock state & options
# ----------------------------
# Modes: normal, offline, slow, error, unreachable
export HTTP_MOCK_MODE="${HTTP_MOCK_MODE:-normal}"

# Optional: directory to log calls/responses
# export MOCK_RESPONSES_DIR="${MOCK_RESPONSES_DIR:-}"

# In-memory state
declare -A MOCK_HTTP_ENDPOINTS=()        # url -> "state|response|status_code"
declare -A MOCK_HTTP_RESPONSES=()        # url -> response_body
declare -A MOCK_HTTP_STATUS_CODES=()     # url -> status_code
declare -A MOCK_HTTP_DELAYS=()           # url -> delay_seconds
declare -A MOCK_HTTP_ERRORS=()           # command -> error_type
declare -A MOCK_HTTP_HEADERS=()          # url -> "header1:value1|header2:value2"
declare -A MOCK_HTTP_METHODS=()          # url -> expected_method
declare -A MOCK_HTTP_SEQUENCES=()        # url -> "response1,response2,response3"
declare -A MOCK_HTTP_SEQUENCE_INDEX=()   # url -> current_index
declare -A MOCK_HTTP_CALL_COUNT=()       # url -> call_count

# File-based state persistence for subshell access (BATS compatibility)
export HTTP_MOCK_STATE_FILE="${MOCK_LOG_DIR:-/tmp}/http_mock_state.$$"

# Initialize state file
_http_mock_init_state_file() {
  if [[ -n "${HTTP_MOCK_STATE_FILE}" ]]; then
    {
      echo "declare -A MOCK_HTTP_ENDPOINTS=()"
      echo "declare -A MOCK_HTTP_RESPONSES=()"
      echo "declare -A MOCK_HTTP_STATUS_CODES=()"
      echo "declare -A MOCK_HTTP_DELAYS=()"
      echo "declare -A MOCK_HTTP_ERRORS=()"
      echo "declare -A MOCK_HTTP_HEADERS=()"
      echo "declare -A MOCK_HTTP_METHODS=()"
      echo "declare -A MOCK_HTTP_SEQUENCES=()"
      echo "declare -A MOCK_HTTP_SEQUENCE_INDEX=()"
      echo "declare -A MOCK_HTTP_CALL_COUNT=()"
    } > "$HTTP_MOCK_STATE_FILE"
  fi
}

# Save current state to file
_http_mock_save_state() {
  if [[ -n "${HTTP_MOCK_STATE_FILE}" ]]; then
    {
      # Use declare -gA for global associative arrays
      declare -p MOCK_HTTP_ENDPOINTS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_HTTP_ENDPOINTS=()"
      declare -p MOCK_HTTP_RESPONSES 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_HTTP_RESPONSES=()"
      declare -p MOCK_HTTP_STATUS_CODES 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_HTTP_STATUS_CODES=()"
      declare -p MOCK_HTTP_DELAYS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_HTTP_DELAYS=()"
      declare -p MOCK_HTTP_ERRORS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_HTTP_ERRORS=()"
      declare -p MOCK_HTTP_HEADERS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_HTTP_HEADERS=()"
      declare -p MOCK_HTTP_METHODS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_HTTP_METHODS=()"
      declare -p MOCK_HTTP_SEQUENCES 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_HTTP_SEQUENCES=()"
      declare -p MOCK_HTTP_SEQUENCE_INDEX 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_HTTP_SEQUENCE_INDEX=()"
      declare -p MOCK_HTTP_CALL_COUNT 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_HTTP_CALL_COUNT=()"
    } > "$HTTP_MOCK_STATE_FILE"
  fi
}

# Load state from file
_http_mock_load_state() {
  if [[ -n "${HTTP_MOCK_STATE_FILE}" && -f "$HTTP_MOCK_STATE_FILE" ]]; then
    # Use eval to execute in global scope, not function scope
    eval "$(cat "$HTTP_MOCK_STATE_FILE")" 2>/dev/null || true
  fi
}

# Initialize state file
_http_mock_init_state_file

# ----------------------------
# Utilities
# ----------------------------
# Note: Centralized logging functions are provided by logs.sh
# which should always be loaded before this mock system

_sanitize_url_key() {
  local url="$1"
  # Replace all problematic characters with underscores, including dots and colons
  echo "$url" | sed 's/[^a-zA-Z0-9_-]/_/g'
}

_mock_http_response_id() {
  # Generate HTTP-like response ID (showing first 8 chars)
  local url="$1"
  local hex=$(printf "%s_%s" "$url" "$RANDOM" | sha256sum | cut -d' ' -f1)
  echo "${hex:0:8}"
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

_mock_current_timestamp() {
  date -Iseconds
}

# ----------------------------
# Public functions used by tests
# ----------------------------
mock::http::reset() {
  # Recreate as associative arrays (not indexed arrays)
  declare -gA MOCK_HTTP_ENDPOINTS=()
  declare -gA MOCK_HTTP_RESPONSES=()
  declare -gA MOCK_HTTP_STATUS_CODES=()
  declare -gA MOCK_HTTP_DELAYS=()
  declare -gA MOCK_HTTP_ERRORS=()
  declare -gA MOCK_HTTP_HEADERS=()
  declare -gA MOCK_HTTP_METHODS=()
  declare -gA MOCK_HTTP_SEQUENCES=()
  declare -gA MOCK_HTTP_SEQUENCE_INDEX=()
  declare -gA MOCK_HTTP_CALL_COUNT=()
  
  # Initialize state file for subshell access
  _http_mock_init_state_file
  
  # Save the cleared state to file for subshell access
  _http_mock_save_state
  
  echo "[MOCK] HTTP state reset"
}

# Enable automatic cleanup between tests
mock::http::enable_auto_cleanup() {
  export HTTP_MOCK_AUTO_CLEANUP=true
}

# Inject errors for testing failure scenarios
mock::http::inject_error() {
  local cmd="$1"
  local error_type="${2:-generic}"
  MOCK_HTTP_ERRORS["$cmd"]="$error_type"
  
  # Save state to file for subshell access
  _http_mock_save_state
  
  echo "[MOCK] Injected error for $cmd: $error_type"
}

# ----------------------------
# Public setters used by tests
# ----------------------------
mock::http::set_endpoint_state() {
  local url="$1" state="$2" response="${3:-}" status_code="${4:-200}"
  local key="$(_sanitize_url_key "$url")"
  
  MOCK_HTTP_ENDPOINTS["$key"]="$state"
  
  # Set default responses based on state
  case "$state" in
    healthy)
      [[ -z "$response" ]] && response='{"status":"healthy","timestamp":"'$(_mock_current_timestamp)'"}'
      [[ "$status_code" == "200" ]] && status_code="200"
      ;;
    unhealthy)
      [[ -z "$response" ]] && response='{"status":"unhealthy","error":"Service degraded"}'
      [[ "$status_code" == "200" ]] && status_code="503"
      ;;
    unavailable)
      [[ -z "$response" ]] && response=""
      status_code="0"  # Connection refused
      ;;
    timeout)
      [[ -z "$response" ]] && response=""
      status_code="0"  # Connection timeout
      ;;
  esac
  
  MOCK_HTTP_RESPONSES["$key"]="$response"
  MOCK_HTTP_STATUS_CODES["$key"]="$status_code"
  
  # Save state to file for subshell access
  _http_mock_save_state
  
  # Use centralized state logging
  mock::log_state "http_endpoint_state" "$url" "$state"
  
  if command -v mock::verify::record_call &>/dev/null; then
    mock::verify::record_call "http" "set_endpoint_state $url $state"
  fi
  return 0
}

mock::http::set_endpoint_response() {
  local url="$1" response="$2" status_code="${3:-200}" headers="${4:-}"
  local key="$(_sanitize_url_key "$url")"
  
  MOCK_HTTP_RESPONSES["$key"]="$response"
  MOCK_HTTP_STATUS_CODES["$key"]="$status_code"
  [[ -n "$headers" ]] && MOCK_HTTP_HEADERS["$key"]="$headers"
  
  # Save state to file for subshell access
  _http_mock_save_state
  
  return 0
}

mock::http::set_endpoint_delay() {
  local url="$1" delay="$2"
  local key="$(_sanitize_url_key "$url")"
  
  MOCK_HTTP_DELAYS["$key"]="$delay"
  
  # Save state to file for subshell access
  _http_mock_save_state
  
  return 0
}

mock::http::set_endpoint_sequence() {
  local url="$1" responses="$2"
  local key="$(_sanitize_url_key "$url")"
  
  MOCK_HTTP_SEQUENCES["$key"]="$responses"
  MOCK_HTTP_SEQUENCE_INDEX["$key"]="0"
  
  # Save state to file for subshell access
  _http_mock_save_state
  
  return 0
}

mock::http::set_endpoint_method() {
  local url="$1" method="$2"
  local key="$(_sanitize_url_key "$url")"
  
  MOCK_HTTP_METHODS["$key"]="$method"
  
  # Save state to file for subshell access
  _http_mock_save_state
  
  return 0
}

mock::http::set_endpoint_unreachable() {
  local endpoint="$1"
  
  # If it's a base URL, mark common endpoints as unreachable
  if [[ "$endpoint" =~ ^https?://[^/]+/?$ ]]; then
    # Remove trailing slash if present
    endpoint="${endpoint%/}"
    
    # Mark common endpoints as unreachable
    local common_endpoints=("/" "/health" "/healthz" "/api" "/api/v1" "/status" "/ping" "/ready" "/metrics")
    for path in "${common_endpoints[@]}"; do
      mock::http::set_endpoint_state "${endpoint}${path}" "unavailable"
    done
    
    # Also mark the base URL itself
    mock::http::set_endpoint_state "$endpoint" "unavailable"
  else
    # Mark specific endpoint as unreachable
    mock::http::set_endpoint_state "$endpoint" "unavailable"
  fi
  
  return 0
}

# ----------------------------
# HTTP Tools Implementation
# ----------------------------

# curl() main interceptor
curl() {
  # Load state from file for subshell access (inline to avoid function scoping issues)
  if [[ -n "${HTTP_MOCK_STATE_FILE}" && -f "$HTTP_MOCK_STATE_FILE" ]]; then
    eval "$(cat "$HTTP_MOCK_STATE_FILE")" 2>/dev/null || true
  fi

  # Use centralized logging and verification
  mock::log_and_verify "curl" "$*"

  case "$HTTP_MOCK_MODE" in
    offline) echo "curl: (6) Could not resolve host" >&2; return 6 ;;
    error)   echo "curl: (7) Failed to connect" >&2; return 7 ;;
  esac

  local url="" method="GET" output_file="" headers=() data="" 
  local follow_redirects=false silent=false show_headers=false
  local output_http_code=false max_time="" connect_timeout=""
  local fail_on_error=false user_agent="curl/8.0.0" write_out=""
  local compressed=false insecure=false location_trusted=false
  local proxy="" cookie_jar="" cookie=""

  # Parse curl arguments with comprehensive flag support
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -X|--request)  method="$2"; shift 2 ;;
      -o|--output)   output_file="$2"; shift 2 ;;
      -H|--header)   headers+=("$2"); shift 2 ;;
      -d|--data|--data-raw) data="$2"; method="POST"; shift 2 ;;
      --data-binary) data="$2"; method="POST"; shift 2 ;;
      -L|--location) follow_redirects=true; shift ;;
      -s|--silent)   silent=true; shift ;;
      -i|--include)  show_headers=true; shift ;;
      -I|--head)     method="HEAD"; shift ;;
      -w|--write-out) write_out="$2"; output_http_code=true; shift 2 ;;
      --max-time)    max_time="$2"; shift 2 ;;
      --connect-timeout) connect_timeout="$2"; shift 2 ;;
      -f|--fail)     fail_on_error=true; shift ;;
      -A|--user-agent) user_agent="$2"; shift 2 ;;
      --compressed)  compressed=true; shift ;;
      -k|--insecure) insecure=true; shift ;;
      --location-trusted) location_trusted=true; shift ;;
      --proxy)       proxy="$2"; shift 2 ;;
      -c|--cookie-jar) cookie_jar="$2"; shift 2 ;;
      -b|--cookie)   cookie="$2"; shift 2 ;;
      -v|--verbose)  shift ;;  # Ignore verbose flag
      -*)            shift ;;  # Ignore other flags
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

  # Track call count
  local key="$(_sanitize_url_key "$url")"
  local count="${MOCK_HTTP_CALL_COUNT[$key]:-0}"
  MOCK_HTTP_CALL_COUNT["$key"]=$((count + 1))
  
  # Save state immediately after call count increment
  _http_mock_save_state

  # Check for injected errors (after tracking call count)
  local cmd_check="curl"
  if [[ -n "${MOCK_HTTP_ERRORS[$cmd_check]}" ]]; then
    local error_type="${MOCK_HTTP_ERRORS[$cmd_check]}"
    case "$error_type" in
      dns_resolution)
        echo "curl: (6) Could not resolve host: ${2:-unknown}" >&2
        return 6
        ;;
      connection_timeout)
        echo "curl: (28) Connection timed out after ${3:-10000} milliseconds" >&2
        return 28
        ;;
      connection_refused)
        echo "curl: (7) Failed to connect to ${2:-localhost} port ${3:-80}: Connection refused" >&2
        return 7
        ;;
      ssl_error)
        echo "curl: (35) SSL connect error" >&2
        return 35
        ;;
      http_error)
        echo "curl: (22) The requested URL returned error: ${3:-500}" >&2
        return 22
        ;;
      *)
        echo "curl: Generic error: $error_type" >&2
        return 1
        ;;
    esac
  fi

  # Apply delay if configured
  local delay="${MOCK_HTTP_DELAYS[$key]:-}"
  if [[ -n "$delay" && "$HTTP_MOCK_MODE" == "slow" ]]; then
    sleep "$delay"
  fi

  # Get response for URL
  local response status_code
  mock::http::get_response "$url" "$method" response status_code

  # Handle connection timeouts and unreachable endpoints
  if [[ "$status_code" == "0" ]]; then
    if [[ "$silent" != true ]]; then
      local host=$(echo "$url" | sed 's|https\?://||' | cut -d/ -f1)
      local port="80"
      [[ "$url" =~ https:// ]] && port="443"
      [[ "$host" =~ :([0-9]+) ]] && port="${BASH_REMATCH[1]}"
      echo "curl: (7) Failed to connect to ${host} port ${port}: Connection refused" >&2
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

  # Handle HEAD requests
  if [[ "$method" == "HEAD" ]]; then
    show_headers=true
    response=""
  fi

  # Add headers if requested
  if [[ "$show_headers" == true ]]; then
    local status_text="OK"
    case "$status_code" in
      200) status_text="OK" ;;
      201) status_text="Created" ;;
      204) status_text="No Content" ;;
      400) status_text="Bad Request" ;;
      401) status_text="Unauthorized" ;;
      403) status_text="Forbidden" ;;
      404) status_text="Not Found" ;;
      500) status_text="Internal Server Error" ;;
      503) status_text="Service Unavailable" ;;
    esac
    
    output+="HTTP/1.1 $status_code $status_text"$'\n'
    output+="Content-Type: application/json"$'\n'
    output+="Content-Length: ${#response}"$'\n'
    output+="Server: MockServer/1.0"$'\n'
    
    # Add custom headers if configured
    local custom_headers="${MOCK_HTTP_HEADERS[$key]:-}"
    if [[ -n "$custom_headers" ]]; then
      # Parse header format: "header1:value1|header2:value2"
      IFS='|' read -ra header_array <<< "$custom_headers"
      for header in "${header_array[@]}"; do
        output+="$header"$'\n'
      done
    fi
    
    output+=""$'\n'
  fi

  # Add response body (unless HEAD request)
  if [[ "$method" != "HEAD" ]]; then
    output+="$response"
  fi

  # Handle output options
  if [[ -n "$output_file" ]]; then
    echo "$output" > "$output_file"
    if [[ "$silent" != true ]]; then
      local size=${#output}
      echo "  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current"
      echo "                                 Dload  Upload   Total   Spent    Left  Speed"
      printf "%3d %5d  %3d %5d    0     0  %5d      0 --:--:-- --:--:-- --:--:-- %5d\n" \
        100 "$size" 100 "$size" "$size" "$size"
    fi
  else
    if [[ "$output_http_code" == true ]]; then
      echo "$output"
      # Handle write-out format
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
  _http_mock_save_state

  # Return appropriate exit code
  if [[ "$status_code" -ge 400 ]]; then
    return 1
  else
    return 0
  fi
}

# wget() implementation
wget() {
  # Load state from file for subshell access
  if [[ -n "${HTTP_MOCK_STATE_FILE}" && -f "$HTTP_MOCK_STATE_FILE" ]]; then
    eval "$(cat "$HTTP_MOCK_STATE_FILE")" 2>/dev/null || true
  fi

  # Use centralized logging and verification
  mock::log_and_verify "wget" "$*"

  case "$HTTP_MOCK_MODE" in
    offline) echo "wget: unable to resolve host address" >&2; return 4 ;;
    error)   echo "wget: connection failed" >&2; return 4 ;;
  esac

  local url="" output_file="" quiet=false continue_partial=false
  local no_check_certificate=false user_agent="Wget/1.21.3"
  local timeout="" tries="20" wait_seconds="" headers=()

  # Parse wget arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -O|--output-document) output_file="$2"; shift 2 ;;
      -q|--quiet)           quiet=true; shift ;;
      -c|--continue)        continue_partial=true; shift ;;
      --no-check-certificate) no_check_certificate=true; shift ;;
      -U|--user-agent)      user_agent="$2"; shift 2 ;;
      -T|--timeout)         timeout="$2"; shift 2 ;;
      -t|--tries)           tries="$2"; shift 2 ;;
      -w|--wait)            wait_seconds="$2"; shift 2 ;;
      --header)             headers+=("$2"); shift 2 ;;
      -*)                   shift ;;  # Ignore other flags
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

  # Track call count
  local key="$(_sanitize_url_key "$url")"
  local count="${MOCK_HTTP_CALL_COUNT[$key]:-0}"
  MOCK_HTTP_CALL_COUNT["$key"]=$((count + 1))

  # Get mock response using HTTP logic
  local response status_code
  mock::http::get_response "$url" "GET" response status_code

  if [[ "$status_code" == "0" ]]; then
    echo "wget: unable to resolve host address" >&2
    return 4
  fi

  # Determine output filename
  local filename="${output_file:-index.html}"
  if [[ "$output_file" == "-" ]]; then
    filename="stdout"
  fi

  # Generate output
  if [[ "$quiet" != true ]]; then
    echo "--$(date '+%Y-%m-%d %H:%M:%S')--  $url"
    echo "Resolving $(echo "$url" | sed 's|https\?://||' | cut -d/ -f1)... 127.0.0.1"
    echo "Connecting to $(echo "$url" | sed 's|https\?://||' | cut -d/ -f1)|127.0.0.1|:80... connected."
    echo "HTTP request sent, awaiting response... $status_code OK"
    echo "Length: ${#response} [application/json]"
    echo "Saving to: '$filename'"
    echo ""
    echo "     0K $(printf '%-40s' '.' | tr ' ' '.') 100% $(date '+%H:%M:%S') (999 KB/s) - '$filename' saved [${#response}/${#response}]"
  fi

  # Output to file or stdout
  if [[ "$output_file" == "-" ]]; then
    echo "$response"
  elif [[ -n "$output_file" ]]; then
    echo "$response" > "$output_file"
  else
    echo "$response" > "index.html"
  fi

  # Save state
  _http_mock_save_state

  return 0
}

# nc (netcat) command mock for port testing
nc() {
  # Load state from file for subshell access
  if [[ -n "${HTTP_MOCK_STATE_FILE}" && -f "$HTTP_MOCK_STATE_FILE" ]]; then
    eval "$(cat "$HTTP_MOCK_STATE_FILE")" 2>/dev/null || true
  fi

  # Use centralized logging and verification
  mock::log_and_verify "nc" "$*"

  local host="" port="" zero_io=false verbose=false timeout=""
  local udp=false listen=false

  # Parse nc arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -z)  zero_io=true; shift ;;
      -v)  verbose=true; shift ;;
      -zv|-vz) zero_io=true; verbose=true; shift ;;  # Handle combined flags
      -w)  timeout="$2"; shift 2 ;;
      -u)  udp=true; shift ;;
      -l)  listen=true; shift ;;
      -*)  shift ;;  # Ignore other flags
      *)
        if [[ -z "$host" ]]; then
          host="$1"
        elif [[ -z "$port" ]]; then
          port="$1"
        fi
        shift
        ;;
    esac
  done

  # Simulate connection test
  if [[ -n "$host" && -n "$port" ]]; then
    local endpoint="http://${host}:${port}"
    local key="$(_sanitize_url_key "$endpoint")"
    
    if [[ -n "${MOCK_HTTP_ENDPOINTS[$key]}" ]]; then
      local state="${MOCK_HTTP_ENDPOINTS[$key]}"
      case "$state" in
        healthy|running)
          [[ "$verbose" == true ]] && echo "Connection to $host $port port [tcp/*] succeeded!"
          return 0
          ;;
        *)
          [[ "$verbose" == true ]] && echo "nc: connect to $host port $port (tcp) failed: Connection refused" >&2
          return 1
          ;;
      esac
    else
      # Default behavior - check for common ports
      case "$port" in
        80|443|8080|3000|5000|8000)
          [[ "$verbose" == true ]] && echo "Connection to $host $port port [tcp/*] succeeded!"
          return 0
          ;;
        *)
          [[ "$verbose" == true ]] && echo "nc: connect to $host port $port (tcp) failed: Connection refused" >&2
          return 1
          ;;
      esac
    fi
  fi

  return 1
}

# ----------------------------
# Response Resolution System
# ----------------------------
mock::http::get_response() {
  local url="$1" method="${2:-GET}"
  local -n response_ref=$3
  local -n status_ref=$4
  local key="$(_sanitize_url_key "$url")"

  # Check for sequence responses first
  if [[ -n "${MOCK_HTTP_SEQUENCES[$key]}" ]]; then
    local sequence="${MOCK_HTTP_SEQUENCES[$key]}"
    local index="${MOCK_HTTP_SEQUENCE_INDEX[$key]:-0}"
    
    # Parse sequence (format: "response1:status1,response2:status2,...")
    IFS=',' read -ra responses <<< "$sequence"
    if [[ "$index" -lt "${#responses[@]}" ]]; then
      local response_entry="${responses[$index]}"
      if [[ "$response_entry" =~ ^(.*):(.*?)$ ]]; then
        response_ref="${BASH_REMATCH[1]}"
        status_ref="${BASH_REMATCH[2]}"
      else
        response_ref="$response_entry"
        status_ref="200"
      fi
      # Increment index for next call
      MOCK_HTTP_SEQUENCE_INDEX["$key"]=$((index + 1))
    else
      # Use last response if sequence exhausted
      local last_response="${responses[-1]}"
      if [[ "$last_response" =~ ^(.*):(.*?)$ ]]; then
        response_ref="${BASH_REMATCH[1]}"
        status_ref="${BASH_REMATCH[2]}"
      else
        response_ref="$last_response"
        status_ref="200"
      fi
    fi
    return
  fi

  # Check for exact URL match (including empty responses for unavailable endpoints)
  if [[ "${MOCK_HTTP_RESPONSES[$key]+isset}" == "isset" ]]; then
    response_ref="${MOCK_HTTP_RESPONSES[$key]}"
    status_ref="${MOCK_HTTP_STATUS_CODES[$key]:-200}"
    return
  fi

  # Check method expectations
  if [[ -n "${MOCK_HTTP_METHODS[$key]}" ]]; then
    local expected_method="${MOCK_HTTP_METHODS[$key]}"
    if [[ "$method" != "$expected_method" ]]; then
      response_ref='{"error":"Method not allowed","expected":"'$expected_method'","received":"'$method'"}'
      status_ref="405"
      return
    fi
  fi

  # Fall back to pattern matching
  mock::http::match_url_pattern "$url" "$method" response_ref status_ref
}

mock::http::match_url_pattern() {
  local url="$1" method="${2:-GET}"
  local -n pattern_response_ref=$3
  local -n pattern_status_ref=$4

  # Extract components
  local host=$(echo "$url" | sed 's|https\?://||' | cut -d/ -f1)
  local path=$(echo "$url" | sed 's|https\?://[^/]*||')
  [[ -z "$path" ]] && path="/"

  # Default response
  pattern_response_ref='{"status":"ok"}'
  pattern_status_ref="200"

  # Check resource-specific patterns first (by host/port)
  case "$host" in
    *"ollama"*|*"11434"*)
      mock::http::pattern_ollama "$path" "$method" pattern_response_ref pattern_status_ref
      return
      ;;
    *"whisper"*|*"8090"*)
      mock::http::pattern_whisper "$path" "$method" pattern_response_ref pattern_status_ref
      return
      ;;
    *"n8n"*|*"5678"*)
      mock::http::pattern_n8n "$path" "$method" pattern_response_ref pattern_status_ref
      return
      ;;
    *"qdrant"*|*"6333"*)
      mock::http::pattern_qdrant "$path" "$method" pattern_response_ref pattern_status_ref
      return
      ;;
    *"minio"*|*"9000"*)
      mock::http::pattern_minio "$path" "$method" pattern_response_ref pattern_status_ref
      return
      ;;
    *"redis"*|*"6379"*)
      mock::http::pattern_redis "$path" "$method" pattern_response_ref pattern_status_ref
      return
      ;;
    *"postgres"*|*"5432"*)
      mock::http::pattern_postgres "$path" "$method" pattern_response_ref pattern_status_ref
      return
      ;;
  esac
  
  # Pattern matching for common endpoints (by path)
  case "$path" in
    *"/health"*|*"/healthz"*)
      pattern_response_ref='{"status":"healthy","timestamp":"'$(_mock_current_timestamp)'","method":"'$method'"}'
      ;;
    *"/status"*)
      pattern_response_ref='{"status":"ok","version":"1.0.0","uptime":"'$(_mock_time_ago)'"}'
      ;;
    *"/version"*)
      pattern_response_ref='{"version":"1.0.0","build":"'$(_mock_http_response_id "$url")'","commit":"abc123"}'
      ;;
    *"/api/v"[0-9]*)
      case "$method" in
        GET)  pattern_response_ref='{"data":[],"total":0,"page":1,"limit":10}' ;;
        POST) pattern_response_ref='{"id":"'$(_mock_http_response_id "$url")'","created":"'$(_mock_current_timestamp)'","status":"created"}'; pattern_status_ref="201" ;;
        PUT|PATCH) pattern_response_ref='{"id":"'$(_mock_http_response_id "$url")'","updated":"'$(_mock_current_timestamp)'","status":"updated"}' ;;
        DELETE) pattern_response_ref='{"id":"'$(_mock_http_response_id "$url")'","deleted":"'$(_mock_current_timestamp)'","status":"deleted"}'; pattern_status_ref="204" ;;
      esac
      ;;
    *"/api/"*)
      pattern_response_ref='{"data":[],"total":0,"timestamp":"'$(_mock_current_timestamp)'"}'
      ;;
    *"/metrics"*)
      pattern_response_ref='# HELP http_requests_total Total number of HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="'$method'",status="200"} 42
# HELP http_request_duration_seconds HTTP request duration
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_sum 0.123
http_request_duration_seconds_count 42'
      ;;
    *"/ping"*)
      pattern_response_ref='pong'
      ;;
    *"/ready"*|*"/readiness"*)
      pattern_response_ref='{"ready":true,"timestamp":"'$(_mock_current_timestamp)'"}'
      ;;
    *"/live"*|*"/liveness"*)
      pattern_response_ref='{"alive":true,"timestamp":"'$(_mock_current_timestamp)'"}'
      ;;
    *"/config"*)
      pattern_response_ref='{"config":{"environment":"mock","debug":false,"version":"1.0.0"}}'
      ;;
    *"/info"*)
      pattern_response_ref='{"service":"mock-service","version":"1.0.0","environment":"test","timestamp":"'$(_mock_current_timestamp)'"}'
      ;;
    "/")
      pattern_response_ref='{"message":"Welcome to Mock HTTP Service","version":"1.0.0","endpoints":["/health","/status","/api"]}'
      ;;
    *)
      # Default generic response
      pattern_response_ref='{"message":"Mock response for '$(basename "$path")'","url":"'$url'","method":"'$method'","timestamp":"'$(_mock_current_timestamp)'"}'
      ;;
  esac
}

# ----------------------------
# Resource-specific patterns
# ----------------------------
mock::http::pattern_ollama() {
  local path="$1" method="$2"
  local -n response_ref=$3
  local -n status_ref=$4

  case "$path" in
    *"/api/tags")
      response_ref='{"models":[{"name":"llama3.1:8b","size":4900000000,"modified_at":"'$(_mock_current_timestamp)'"},{"name":"deepseek-r1:8b","size":4700000000,"modified_at":"'$(_mock_current_timestamp)'"}]}'
      ;;
    *"/api/generate")
      case "$method" in
        POST) response_ref='{"model":"llama3.1:8b","created_at":"'$(_mock_current_timestamp)'","response":"Hello! This is a mock response from Ollama.","done":true,"context":[1,2,3],"total_duration":123456789}' ;;
        *) response_ref='{"error":"Method not allowed"}'; status_ref="405" ;;
      esac
      ;;
    *"/api/pull")
      case "$method" in
        POST) response_ref='{"status":"pulling manifest","digest":"sha256:abc123def456","total":1000000000,"completed":500000000}' ;;
        *) response_ref='{"error":"Method not allowed"}'; status_ref="405" ;;
      esac
      ;;
    *"/api/push")
      case "$method" in
        POST) response_ref='{"status":"pushing manifest","digest":"sha256:def456ghi789"}' ;;
        *) response_ref='{"error":"Method not allowed"}'; status_ref="405" ;;
      esac
      ;;
    *)
      response_ref='{"version":"0.1.45","api_version":"1.0"}'
      ;;
  esac
}

mock::http::pattern_whisper() {
  local path="$1" method="$2"
  local -n response_ref=$3
  local -n status_ref=$4

  case "$path" in
    *"/transcribe")
      case "$method" in
        POST) response_ref='{"text":"This is a mock transcription result from Whisper.","language":"en","duration":5.2,"segments":[{"start":0.0,"end":5.2,"text":"This is a mock transcription result from Whisper."}]}' ;;
        *) response_ref='{"error":"Method not allowed"}'; status_ref="405" ;;
      esac
      ;;
    *"/translate")
      case "$method" in
        POST) response_ref='{"text":"This is a mock translation result.","source_language":"es","target_language":"en","duration":4.8}' ;;
        *) response_ref='{"error":"Method not allowed"}'; status_ref="405" ;;
      esac
      ;;
    *)
      response_ref='{"status":"ready","model":"whisper-1","version":"20231117"}'
      ;;
  esac
}

mock::http::pattern_n8n() {
  local path="$1" method="$2"
  local -n response_ref=$3
  local -n status_ref=$4

  case "$path" in
    *"/api/v1/workflows")
      case "$method" in
        GET) response_ref='{"data":[{"id":"1","name":"Test Workflow","active":true,"nodes":3,"connections":{"Start":{"main":[[{"node":"End","type":"main","index":0}]]}},"createdAt":"'$(_mock_current_timestamp)'"}],"nextCursor":null}' ;;
        POST) response_ref='{"data":{"id":"'$(_mock_http_response_id "$path")'","name":"New Workflow","active":false,"nodes":1,"createdAt":"'$(_mock_current_timestamp)'"}}'; status_ref="201" ;;
        *) response_ref='{"error":"Method not allowed"}'; status_ref="405" ;;
      esac
      ;;
    *"/api/v1/executions")
      response_ref='{"data":[{"id":"exec_123","workflowId":"1","mode":"trigger","startedAt":"'$(_mock_current_timestamp)'","status":"success"}],"nextCursor":null}'
      ;;
    *"/healthz")
      response_ref='{"status":"ok","timestamp":"'$(_mock_current_timestamp)'"}'
      ;;
    *)
      response_ref='{"version":"1.19.0","instanceId":"'$(_mock_http_response_id "$path")'"}'
      ;;
  esac
}

mock::http::pattern_qdrant() {
  local path="$1" method="$2"
  local -n response_ref=$3
  local -n status_ref=$4

  case "$path" in
    *"/collections")
      case "$method" in
        GET) response_ref='{"result":{"collections":[{"name":"test_collection","vectors_count":100,"indexed_vectors_count":100,"points_count":100}]},"status":"ok","time":0.001}' ;;
        POST) response_ref='{"result":true,"status":"ok","time":0.002}'; status_ref="201" ;;
        *) response_ref='{"status":"error","message":"Method not allowed"}'; status_ref="405" ;;
      esac
      ;;
    *"/collections/"*"/points")
      case "$method" in
        GET) response_ref='{"result":{"points":[{"id":1,"vector":[0.1,0.2,0.3],"payload":{"field":"value"}}]},"status":"ok","time":0.001}' ;;
        POST) response_ref='{"result":{"operation_id":123,"status":"acknowledged"},"status":"ok","time":0.002}'; status_ref="201" ;;
        *) response_ref='{"status":"error","message":"Method not allowed"}'; status_ref="405" ;;
      esac
      ;;
    *"/health")
      response_ref='{"title":"qdrant - vector search engine","version":"1.7.0","commit":"abc123def456"}'
      ;;
    *)
      response_ref='{"result":{},"status":"ok","time":0.001}'
      ;;
  esac
}

mock::http::pattern_minio() {
  local path="$1" method="$2"
  local -n response_ref=$3
  local -n status_ref=$4

  case "$path" in
    *"/minio/health/live")
      response_ref='{"status":"ok","timestamp":"'$(_mock_current_timestamp)'"}'
      ;;
    *"/minio/health/ready")
      response_ref='{"status":"ok","timestamp":"'$(_mock_current_timestamp)'"}'
      ;;
    *"/minio/health/cluster")
      response_ref='{"status":"online","nodes":1,"drives":4,"timestamp":"'$(_mock_current_timestamp)'"}'
      ;;
    *)
      response_ref='{"version":"RELEASE.2023-11-20T22-40-07Z","region":"us-east-1"}'
      ;;
  esac
}

mock::http::pattern_redis() {
  local path="$1" method="$2"
  local -n response_ref=$3
  local -n status_ref=$4

  case "$path" in
    *"/health")
      response_ref='{"status":"ok","redis_version":"7.0.0","uptime":"'$(_mock_time_ago)'"}'
      ;;
    *"/info")
      response_ref='{"redis_version":"7.0.0","redis_mode":"standalone","role":"master","connected_clients":1,"used_memory_human":"1.2M"}'
      ;;
    *)
      response_ref='{"status":"ready","version":"7.0.0"}'
      ;;
  esac
}

mock::http::pattern_postgres() {
  local path="$1" method="$2"
  local -n response_ref=$3
  local -n status_ref=$4

  case "$path" in
    *"/health")
      response_ref='{"status":"ok","version":"15.0","connections":{"active":1,"idle":5,"max":100}}'
      ;;
    *"/stats")
      response_ref='{"databases":3,"tables":25,"connections":{"current":1,"max":100},"uptime":"'$(_mock_time_ago)'"}'
      ;;
    *)
      response_ref='{"version":"PostgreSQL 15.0","status":"ready"}'
      ;;
  esac
}

# ----------------------------
# Test Helper Functions
# ----------------------------

# Scenario builders for common test patterns
mock::http::scenario::create_healthy_services() {
  local base_url="${1:-"http://localhost"}"
  local services=("api:8080" "db:5432" "cache:6379" "search:9200")
  
  for service in "${services[@]}"; do
    IFS=':' read -r name port <<< "$service"
    mock::http::set_endpoint_state "${base_url}:${port}/health" "healthy"
    mock::http::set_endpoint_response "${base_url}:${port}/health" \
      '{"service":"'$name'","status":"healthy","port":'$port',"timestamp":"'$(_mock_current_timestamp)'"}' 200
  done
  
  echo "[MOCK] Created healthy services scenario"
}

# Create mixed health scenario (some healthy, some not)
mock::http::scenario::create_mixed_health() {
  local base_url="${1:-"http://localhost"}"
  
  # Healthy services
  mock::http::set_endpoint_state "${base_url}:8080/health" "healthy"
  mock::http::set_endpoint_state "${base_url}:6379/health" "healthy"
  
  # Unhealthy services
  mock::http::set_endpoint_state "${base_url}:5432/health" "unhealthy"
  mock::http::set_endpoint_state "${base_url}:9200/health" "unavailable"
  
  echo "[MOCK] Created mixed health scenario"
}

# Create API versioning scenario
mock::http::scenario::create_api_versions() {
  local base_url="${1:-"http://localhost:8080"}"
  
  # v1 API - deprecated
  mock::http::set_endpoint_response "${base_url}/api/v1/users" \
    '{"data":[],"deprecated":true,"migrate_to":"/api/v2/users"}' 200
    
  # v2 API - current
  mock::http::set_endpoint_response "${base_url}/api/v2/users" \
    '{"data":[{"id":1,"name":"Test User","email":"test@example.com"}],"version":"2.0"}' 200
    
  # v3 API - beta
  mock::http::set_endpoint_response "${base_url}/api/v3/users" \
    '{"data":[{"id":1,"name":"Test User","email":"test@example.com","profile":{"avatar":"url"}}],"version":"3.0-beta"}' 200
  
  echo "[MOCK] Created API versioning scenario"
}

# Assertion helpers
mock::http::assert::endpoint_called() {
  local url="$1"
  local expected_count="${2:-1}"
  local key="$(_sanitize_url_key "$url")"
  local actual_count="${MOCK_HTTP_CALL_COUNT[$key]:-0}"
  
  if [[ "$actual_count" -lt "$expected_count" ]]; then
    echo "ASSERTION FAILED: Endpoint '$url' called $actual_count times, expected at least $expected_count" >&2
    return 1
  fi
  return 0
}

mock::http::assert::endpoint_not_called() {
  local url="$1"
  local key="$(_sanitize_url_key "$url")"
  local actual_count="${MOCK_HTTP_CALL_COUNT[$key]:-0}"
  
  if [[ "$actual_count" -gt 0 ]]; then
    echo "ASSERTION FAILED: Endpoint '$url' was called $actual_count times but should not have been called" >&2
    return 1
  fi
  return 0
}

mock::http::assert::endpoint_healthy() {
  local url="$1"
  local key="$(_sanitize_url_key "$url")"
  local state="${MOCK_HTTP_ENDPOINTS[$key]:-unknown}"
  
  if [[ "$state" != "healthy" ]]; then
    echo "ASSERTION FAILED: Endpoint '$url' is not healthy (state: $state)" >&2
    return 1
  fi
  return 0
}

mock::http::assert::endpoint_unavailable() {
  local url="$1"
  local key="$(_sanitize_url_key "$url")"
  local state="${MOCK_HTTP_ENDPOINTS[$key]:-unknown}"
  
  if [[ "$state" != "unavailable" ]]; then
    echo "ASSERTION FAILED: Endpoint '$url' is not unavailable (state: $state)" >&2
    return 1
  fi
  return 0
}

mock::http::assert::response_contains() {
  local url="$1"
  local expected_content="$2"
  local key="$(_sanitize_url_key "$url")"
  local response="${MOCK_HTTP_RESPONSES[$key]:-}"
  
  if [[ ! "$response" =~ $expected_content ]]; then
    echo "ASSERTION FAILED: Response for '$url' does not contain '$expected_content'" >&2
    echo "Actual response: $response" >&2
    return 1
  fi
  return 0
}

mock::http::assert::status_code() {
  local url="$1"
  local expected_status="$2"
  local key="$(_sanitize_url_key "$url")"
  local actual_status="${MOCK_HTTP_STATUS_CODES[$key]:-200}"
  
  if [[ "$actual_status" != "$expected_status" ]]; then
    echo "ASSERTION FAILED: Status code for '$url' is $actual_status, expected $expected_status" >&2
    return 1
  fi
  return 0
}

# Get information helpers
mock::http::get::call_count() {
  # Load state from file for subshell access
  if [[ -n "${HTTP_MOCK_STATE_FILE}" && -f "$HTTP_MOCK_STATE_FILE" ]]; then
    eval "$(cat "$HTTP_MOCK_STATE_FILE")" 2>/dev/null || true
  fi
  
  local url="$1"
  local key="$(_sanitize_url_key "$url")"
  echo "${MOCK_HTTP_CALL_COUNT[$key]:-0}"
}

mock::http::get::response() {
  # Load state from file for subshell access
  if [[ -n "${HTTP_MOCK_STATE_FILE}" && -f "$HTTP_MOCK_STATE_FILE" ]]; then
    eval "$(cat "$HTTP_MOCK_STATE_FILE")" 2>/dev/null || true
  fi
  
  local url="$1"
  local key="$(_sanitize_url_key "$url")"
  echo "${MOCK_HTTP_RESPONSES[$key]:-}"
}

mock::http::get::status_code() {
  # Load state from file for subshell access
  if [[ -n "${HTTP_MOCK_STATE_FILE}" && -f "$HTTP_MOCK_STATE_FILE" ]]; then
    eval "$(cat "$HTTP_MOCK_STATE_FILE")" 2>/dev/null || true
  fi
  
  local url="$1"
  local key="$(_sanitize_url_key "$url")"
  echo "${MOCK_HTTP_STATUS_CODES[$key]:-200}"
}

mock::http::get::endpoint_state() {
  # Load state from file for subshell access
  if [[ -n "${HTTP_MOCK_STATE_FILE}" && -f "$HTTP_MOCK_STATE_FILE" ]]; then
    eval "$(cat "$HTTP_MOCK_STATE_FILE")" 2>/dev/null || true
  fi
  
  local url="$1"
  local key="$(_sanitize_url_key "$url")"
  echo "${MOCK_HTTP_ENDPOINTS[$key]:-unknown}"
}

# Debug helper
mock::http::debug::dump_state() {
  # Load state from file for subshell access
  if [[ -n "${HTTP_MOCK_STATE_FILE}" && -f "$HTTP_MOCK_STATE_FILE" ]]; then
    eval "$(cat "$HTTP_MOCK_STATE_FILE")" 2>/dev/null || true
  fi
  
  echo "=== HTTP Mock State Dump ==="
  echo "Endpoints:"
  for url in "${!MOCK_HTTP_ENDPOINTS[@]}"; do
    echo "  $url: ${MOCK_HTTP_ENDPOINTS[$url]}"
    [[ -n "${MOCK_HTTP_RESPONSES[$url]}" ]] && echo "    RESPONSE: ${MOCK_HTTP_RESPONSES[$url]:0:100}..."
    [[ -n "${MOCK_HTTP_STATUS_CODES[$url]}" ]] && echo "    STATUS: ${MOCK_HTTP_STATUS_CODES[$url]}"
    [[ -n "${MOCK_HTTP_CALL_COUNT[$url]}" ]] && echo "    CALLS: ${MOCK_HTTP_CALL_COUNT[$url]}"
  done
  echo "Errors:"
  for cmd in "${!MOCK_HTTP_ERRORS[@]}"; do
    echo "  $cmd: ${MOCK_HTTP_ERRORS[$cmd]}"
  done
  echo "=========================="
}

mock::http::debug::list_endpoints() {
  # Load state from file for subshell access
  if [[ -n "${HTTP_MOCK_STATE_FILE}" && -f "$HTTP_MOCK_STATE_FILE" ]]; then
    eval "$(cat "$HTTP_MOCK_STATE_FILE")" 2>/dev/null || true
  fi
  
  echo "=== Configured HTTP Endpoints ==="
  for url in "${!MOCK_HTTP_ENDPOINTS[@]}"; do
    local state="${MOCK_HTTP_ENDPOINTS[$url]}"
    local status="${MOCK_HTTP_STATUS_CODES[$url]:-200}"
    local calls="${MOCK_HTTP_CALL_COUNT[$url]:-0}"
    printf "%-50s %-12s %-6s %s\n" "$url" "$state" "$status" "($calls calls)"
  done
  echo "================================"
}

# ----------------------------
# Export functions into subshells
# ----------------------------
# Exporting lets child bash processes (spawned by scripts under test) inherit mocks.
export -f curl wget nc
export -f _sanitize_url_key _mock_http_response_id _mock_time_ago _mock_current_timestamp
export -f _http_mock_init_state_file _http_mock_save_state _http_mock_load_state
export -f mock::http::reset mock::http::enable_auto_cleanup mock::http::inject_error
export -f mock::http::set_endpoint_state mock::http::set_endpoint_response
export -f mock::http::set_endpoint_delay mock::http::set_endpoint_sequence
export -f mock::http::set_endpoint_method mock::http::set_endpoint_unreachable
export -f mock::http::get_response mock::http::match_url_pattern
export -f mock::http::pattern_ollama mock::http::pattern_whisper mock::http::pattern_n8n
export -f mock::http::pattern_qdrant mock::http::pattern_minio mock::http::pattern_redis
export -f mock::http::pattern_postgres

# Export test helper functions
export -f mock::http::scenario::create_healthy_services
export -f mock::http::scenario::create_mixed_health
export -f mock::http::scenario::create_api_versions
export -f mock::http::assert::endpoint_called mock::http::assert::endpoint_not_called
export -f mock::http::assert::endpoint_healthy mock::http::assert::endpoint_unavailable
export -f mock::http::assert::response_contains mock::http::assert::status_code
export -f mock::http::get::call_count mock::http::get::response
export -f mock::http::get::status_code mock::http::get::endpoint_state
export -f mock::http::debug::dump_state mock::http::debug::list_endpoints

echo "[MOCK] HTTP mocks loaded successfully"