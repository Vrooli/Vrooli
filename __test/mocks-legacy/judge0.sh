#!/usr/bin/env bash
# Judge0 Code Execution Mock Implementation
# Provides comprehensive mocking for Judge0 API endpoints and functionality
# Follows standardized mock patterns with proper state management and error injection

# Prevent duplicate loading
if [[ "${JUDGE0_MOCK_LOADED:-}" == "true" ]]; then
    return 0
fi
export JUDGE0_MOCK_LOADED="true"

# ----------------------------
# Global mock state & configuration
# ----------------------------
export JUDGE0_MOCK_MODE="${JUDGE0_MOCK_MODE:-normal}"
export JUDGE0_PORT="${JUDGE0_PORT:-2358}"
export JUDGE0_BASE_URL="http://localhost:${JUDGE0_PORT}"
export JUDGE0_CONTAINER_NAME="${TEST_NAMESPACE:-test}_judge0"
export JUDGE0_API_KEY="${JUDGE0_API_KEY:-mock-api-key-12345}"

# In-memory state
declare -A MOCK_JUDGE0_SUBMISSIONS=()     # token -> submission_json
declare -A MOCK_JUDGE0_RESULTS=()        # token -> result_json
declare -A MOCK_JUDGE0_LANGUAGES=()      # language_name -> id
declare -A MOCK_JUDGE0_CONFIG=()         # config_key -> value
declare -A MOCK_JUDGE0_ERRORS=()         # error_type -> enabled
declare -A MOCK_JUDGE0_WORKERS=()        # worker_id -> status

# File-based state persistence for subshell access (BATS compatibility)
export JUDGE0_MOCK_STATE_FILE="${MOCK_LOG_DIR:-/tmp}/judge0_mock_state.$$"

# Initialize state file
_judge0_mock_init_state_file() {
    if [[ -n "${JUDGE0_MOCK_STATE_FILE}" ]]; then
        {
            echo "declare -A MOCK_JUDGE0_SUBMISSIONS=()"
            echo "declare -A MOCK_JUDGE0_RESULTS=()"
            echo "declare -A MOCK_JUDGE0_LANGUAGES=()"
            echo "declare -A MOCK_JUDGE0_CONFIG=()"
            echo "declare -A MOCK_JUDGE0_ERRORS=()"
            echo "declare -A MOCK_JUDGE0_WORKERS=()"
        } > "$JUDGE0_MOCK_STATE_FILE"
    fi
}

# Save current state to file
_judge0_mock_save_state() {
    if [[ -n "${JUDGE0_MOCK_STATE_FILE}" ]]; then
        {
            declare -p MOCK_JUDGE0_SUBMISSIONS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_JUDGE0_SUBMISSIONS=()"
            declare -p MOCK_JUDGE0_RESULTS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_JUDGE0_RESULTS=()"
            declare -p MOCK_JUDGE0_LANGUAGES 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_JUDGE0_LANGUAGES=()"
            declare -p MOCK_JUDGE0_CONFIG 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_JUDGE0_CONFIG=()"
            declare -p MOCK_JUDGE0_ERRORS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_JUDGE0_ERRORS=()"
            declare -p MOCK_JUDGE0_WORKERS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_JUDGE0_WORKERS=()"
        } > "$JUDGE0_MOCK_STATE_FILE"
    fi
}

# Load state from file
_judge0_mock_load_state() {
    if [[ -n "${JUDGE0_MOCK_STATE_FILE}" && -f "$JUDGE0_MOCK_STATE_FILE" ]]; then
        eval "$(cat "$JUDGE0_MOCK_STATE_FILE")" 2>/dev/null || true
    fi
}

# Initialize state file
_judge0_mock_init_state_file

# ----------------------------
# Utilities
# ----------------------------
_judge0_generate_token() {
    echo "$(date +%s%N | sha256sum | head -c 16)"
}

_judge0_current_timestamp() {
    date -Iseconds 2>/dev/null || date +"%Y-%m-%dT%H:%M:%S.000Z"
}

# ----------------------------
# Public reset and state management
# ----------------------------
mock::judge0::reset() {
    # Reset all state arrays
    declare -gA MOCK_JUDGE0_SUBMISSIONS=()
    declare -gA MOCK_JUDGE0_RESULTS=()
    declare -gA MOCK_JUDGE0_LANGUAGES=()
    declare -gA MOCK_JUDGE0_CONFIG=()
    declare -gA MOCK_JUDGE0_ERRORS=()
    declare -gA MOCK_JUDGE0_WORKERS=()
    
    # Initialize default languages
    mock::judge0::init_default_languages
    
    # Initialize default config
    mock::judge0::init_default_config
    
    # Initialize state file for subshell access
    _judge0_mock_init_state_file
    _judge0_mock_save_state
    
    echo "[MOCK] Judge0 state reset"
}

# Initialize default languages mapping
mock::judge0::init_default_languages() {
    MOCK_JUDGE0_LANGUAGES["c"]="50"
    MOCK_JUDGE0_LANGUAGES["cpp"]="54"
    MOCK_JUDGE0_LANGUAGES["c++"]="54"
    MOCK_JUDGE0_LANGUAGES["java"]="62"
    MOCK_JUDGE0_LANGUAGES["python"]="71"
    MOCK_JUDGE0_LANGUAGES["javascript"]="63"
    MOCK_JUDGE0_LANGUAGES["js"]="63"
    MOCK_JUDGE0_LANGUAGES["kotlin"]="78"
    MOCK_JUDGE0_LANGUAGES["ruby"]="72"
    MOCK_JUDGE0_LANGUAGES["rust"]="73"
    MOCK_JUDGE0_LANGUAGES["go"]="60"
    MOCK_JUDGE0_LANGUAGES["bash"]="46"
    MOCK_JUDGE0_LANGUAGES["sh"]="46"
    
    _judge0_mock_save_state
}

# Initialize default configuration
mock::judge0::init_default_config() {
    MOCK_JUDGE0_CONFIG["cpu_time_limit"]="2"
    MOCK_JUDGE0_CONFIG["wall_time_limit"]="5"
    MOCK_JUDGE0_CONFIG["memory_limit"]="131072"
    MOCK_JUDGE0_CONFIG["stack_limit"]="65536"
    MOCK_JUDGE0_CONFIG["max_processes"]="60"
    MOCK_JUDGE0_CONFIG["max_file_size"]="1024"
    MOCK_JUDGE0_CONFIG["enable_network"]="false"
    MOCK_JUDGE0_CONFIG["workers_count"]="2"
    
    _judge0_mock_save_state
}

# ----------------------------
# Setup functions for different states
# ----------------------------
mock::judge0::setup() {
    local state="${1:-healthy}"
    
    # Configure environment
    export JUDGE0_PORT="${JUDGE0_PORT:-2358}"
    export JUDGE0_BASE_URL="http://localhost:${JUDGE0_PORT}"
    export JUDGE0_CONTAINER_NAME="${TEST_NAMESPACE:-test}_judge0"
    
    # Set up Docker mock state if docker mock is loaded
    if command -v mock::docker::set_container_state &>/dev/null; then
        mock::docker::set_container_state "$JUDGE0_CONTAINER_NAME" "$state"
    fi
    
    # Configure HTTP endpoints based on state
    case "$state" in
        "healthy")
            mock::judge0::setup_healthy_endpoints
            ;;
        "unhealthy")
            mock::judge0::setup_unhealthy_endpoints
            ;;
        "installing")
            mock::judge0::setup_installing_endpoints
            ;;
        "stopped")
            mock::judge0::setup_stopped_endpoints
            ;;
        "rate_limited")
            mock::judge0::setup_rate_limited_endpoints
            ;;
        *)
            echo "[JUDGE0_MOCK] Unknown state: $state" >&2
            return 1
            ;;
    esac
    
    echo "[JUDGE0_MOCK] Judge0 mock configured with state: $state"
}

mock::judge0::setup_healthy_endpoints() {
    # Load HTTP mock if available
    if ! command -v mock::http::set_endpoint_response &>/dev/null; then
        echo "[JUDGE0_MOCK] Warning: HTTP mock not loaded" >&2
        return 0
    fi
    
    # Version endpoint (used by health check - returns plain text)
    mock::http::set_endpoint_response "$JUDGE0_BASE_URL/version" \
        "1.13.0" "200"
    
    # Languages endpoint
    mock::http::set_endpoint_response "$JUDGE0_BASE_URL/languages" \
        '[
            {"id": 46, "name": "Bash (5.0.0)"},
            {"id": 50, "name": "C (GCC 9.2.0)"},
            {"id": 54, "name": "C++ (GCC 9.2.0)"},
            {"id": 60, "name": "Go (1.13.5)"},
            {"id": 62, "name": "Java (OpenJDK 13.0.1)"},
            {"id": 63, "name": "JavaScript (Node.js 12.14.0)"},
            {"id": 71, "name": "Python (3.8.1)"},
            {"id": 72, "name": "Ruby (2.7.0)"},
            {"id": 73, "name": "Rust (1.40.0)"},
            {"id": 78, "name": "Kotlin (1.3.70)"}
        ]' "200"
    
    # Specific language endpoints
    mock::http::set_endpoint_response "$JUDGE0_BASE_URL/languages/71" \
        '{"id": 71, "name": "Python (3.8.1)", "source_file": "script.py"}' "200"
    
    # Submissions endpoint with wait parameter
    mock::http::set_endpoint_response "$JUDGE0_BASE_URL/submissions?wait=true" \
        '{"token":"'$(_judge0_generate_token)'","status":{"id":3,"description":"Accepted"},"stdout":"Hello, World!\n"}' "201"
    
    # Batch submissions endpoint
    mock::http::set_endpoint_response "$JUDGE0_BASE_URL/submissions/batch?wait=true" \
        '[{"token":"batch-1","status":{"id":3}},{"token":"batch-2","status":{"id":3}}]' "201"
    
    # Statistics endpoint
    mock::http::set_endpoint_response "$JUDGE0_BASE_URL/statistics" \
        '{"submissions":{"total":12345,"today":156},"languages":{"most_popular":{"id":71,"name":"Python"}}}' "200"
    
    # System info endpoint
    mock::http::set_endpoint_response "$JUDGE0_BASE_URL/system_info" \
        '{"version":"1.13.0","architecture":"x86_64","cpu_count":4,"memory":"4096 MB"}' "200"
    
    # Config endpoints
    mock::http::set_endpoint_response "$JUDGE0_BASE_URL/config" \
        '{"enable_wait_result":true,"cpu_time_limit":2,"memory_limit":131072}' "200"
}

mock::judge0::setup_unhealthy_endpoints() {
    if ! command -v mock::http::set_endpoint_response &>/dev/null; then
        return 0
    fi
    
    # Version endpoint returns error
    mock::http::set_endpoint_response "$JUDGE0_BASE_URL/version" \
        "" "503"
    
    # All other endpoints return 503
    mock::http::set_endpoint_response "$JUDGE0_BASE_URL/submissions?wait=true" \
        '{"error":"Service temporarily unavailable"}' "503"
    
    mock::http::set_endpoint_response "$JUDGE0_BASE_URL/languages" \
        '{"error":"Execution environment unavailable"}' "503"
}

mock::judge0::setup_installing_endpoints() {
    if ! command -v mock::http::set_endpoint_response &>/dev/null; then
        return 0
    fi
    
    # Version endpoint returns installing message
    mock::http::set_endpoint_response "$JUDGE0_BASE_URL/version" \
        "Installing..." "200"
    
    # Other endpoints return 503 with installing message
    mock::http::set_endpoint_response "$JUDGE0_BASE_URL/submissions?wait=true" \
        '{"error":"Judge0 is still initializing","progress":90}' "503"
}

mock::judge0::setup_stopped_endpoints() {
    if ! command -v mock::http::set_endpoint_unreachable &>/dev/null; then
        return 0
    fi
    
    # All endpoints are unreachable
    mock::http::set_endpoint_unreachable "$JUDGE0_BASE_URL"
}

mock::judge0::setup_rate_limited_endpoints() {
    if ! command -v mock::http::set_endpoint_response &>/dev/null; then
        return 0
    fi
    
    # All endpoints return rate limit error
    mock::http::set_endpoint_response "$JUDGE0_BASE_URL/submissions?wait=true" \
        '{"error":"Rate limit exceeded. Please try again later."}' "429"
    
    mock::http::set_endpoint_response "$JUDGE0_BASE_URL/languages" \
        '{"error":"Rate limit exceeded"}' "429"
}

# ----------------------------
# Error injection
# ----------------------------
mock::judge0::inject_error() {
    local error_type="$1"
    
    # Load current state first
    _judge0_mock_load_state
    
    MOCK_JUDGE0_ERRORS["$error_type"]="true"
    _judge0_mock_save_state
    
    echo "[MOCK] Injected Judge0 error: $error_type"
}

mock::judge0::clear_errors() {
    declare -gA MOCK_JUDGE0_ERRORS=()
    _judge0_mock_save_state
    
    echo "[MOCK] Cleared all Judge0 errors"
}

# ----------------------------
# Submission management
# ----------------------------
mock::judge0::create_submission() {
    local source_code="$1"
    local language_id="$2"
    local stdin="${3:-}"
    local expected_output="${4:-}"
    
    # Load current state first
    _judge0_mock_load_state
    
    local token="$(_judge0_generate_token)"
    local submission_json=$(cat <<EOF
{
    "token": "$token",
    "source_code": "$(echo "$source_code" | sed 's/"/\\"/g')",
    "language_id": $language_id,
    "stdin": "$stdin",
    "expected_output": $([ -n "$expected_output" ] && echo "\"$expected_output\"" || echo "null")
}
EOF
    )
    
    MOCK_JUDGE0_SUBMISSIONS["$token"]="$submission_json"
    _judge0_mock_save_state
    
    echo "$token"
}

mock::judge0::set_submission_result() {
    local token="$1"
    local status_id="${2:-3}"
    local status_desc="${3:-Accepted}"
    local stdout="${4:-}"
    local stderr="${5:-}"
    local time="${6:-0.01}"
    local memory="${7:-2048}"
    
    # Load current state first
    _judge0_mock_load_state
    
    local result_json=$(cat <<EOF
{
    "token": "$token",
    "status": {
        "id": $status_id,
        "description": "$status_desc"
    },
    "stdout": $([ -n "$stdout" ] && echo "\"$stdout\"" || echo "null"),
    "stderr": $([ -n "$stderr" ] && echo "\"$stderr\"" || echo "null"),
    "time": "$time",
    "memory": $memory,
    "created_at": "$(_judge0_current_timestamp)",
    "finished_at": "$(_judge0_current_timestamp)"
}
EOF
    )
    
    MOCK_JUDGE0_RESULTS["$token"]="$result_json"
    _judge0_mock_save_state
}

# ----------------------------
# Configuration management
# ----------------------------
mock::judge0::set_config() {
    local key="$1"
    local value="$2"
    
    # Load current state first
    _judge0_mock_load_state
    
    MOCK_JUDGE0_CONFIG["$key"]="$value"
    _judge0_mock_save_state
}

mock::judge0::get_config() {
    local key="$1"
    
    _judge0_mock_load_state
    echo "${MOCK_JUDGE0_CONFIG[$key]:-}"
}

# ----------------------------
# Worker management
# ----------------------------
mock::judge0::set_worker_status() {
    local worker_id="$1"
    local status="$2"
    
    # Load current state first
    _judge0_mock_load_state
    
    MOCK_JUDGE0_WORKERS["$worker_id"]="$status"
    _judge0_mock_save_state
}

mock::judge0::get_worker_count() {
    _judge0_mock_load_state
    echo "${#MOCK_JUDGE0_WORKERS[@]}"
}

# ----------------------------
# Test helper functions
# ----------------------------
mock::judge0::assert_submission_exists() {
    local token="$1"
    
    _judge0_mock_load_state
    if [[ -z "${MOCK_JUDGE0_SUBMISSIONS[$token]:-}" ]]; then
        echo "ASSERTION FAILED: Submission with token '$token' does not exist" >&2
        return 1
    fi
    return 0
}

mock::judge0::assert_submission_completed() {
    local token="$1"
    
    _judge0_mock_load_state
    if [[ -z "${MOCK_JUDGE0_RESULTS[$token]:-}" ]]; then
        echo "ASSERTION FAILED: No result for submission token '$token'" >&2
        return 1
    fi
    return 0
}

mock::judge0::assert_language_supported() {
    local language="$1"
    
    _judge0_mock_load_state
    local lang_lower="$(echo "$language" | tr '[:upper:]' '[:lower:]')"
    if [[ -z "${MOCK_JUDGE0_LANGUAGES[$lang_lower]:-}" ]]; then
        echo "ASSERTION FAILED: Language '$language' is not supported" >&2
        return 1
    fi
    return 0
}

mock::judge0::get_submission_count() {
    _judge0_mock_load_state
    echo "${#MOCK_JUDGE0_SUBMISSIONS[@]}"
}

mock::judge0::dump_state() {
    _judge0_mock_load_state
    
    echo "=== Judge0 Mock State ==="
    echo "Submissions: ${#MOCK_JUDGE0_SUBMISSIONS[@]}"
    for token in "${!MOCK_JUDGE0_SUBMISSIONS[@]}"; do
        echo "  Token: $token"
    done
    
    echo "Results: ${#MOCK_JUDGE0_RESULTS[@]}"
    echo "Languages: ${#MOCK_JUDGE0_LANGUAGES[@]}"
    echo "Config entries: ${#MOCK_JUDGE0_CONFIG[@]}"
    echo "Workers: ${#MOCK_JUDGE0_WORKERS[@]}"
    echo "Active errors: ${#MOCK_JUDGE0_ERRORS[@]}"
    for error in "${!MOCK_JUDGE0_ERRORS[@]}"; do
        echo "  - $error"
    done
    echo "========================"
}

# ----------------------------
# Export functions for subshell access
# ----------------------------
export -f _judge0_generate_token
export -f _judge0_current_timestamp
export -f _judge0_mock_init_state_file
export -f _judge0_mock_save_state
export -f _judge0_mock_load_state

export -f mock::judge0::reset
export -f mock::judge0::init_default_languages
export -f mock::judge0::init_default_config
export -f mock::judge0::setup
export -f mock::judge0::setup_healthy_endpoints
export -f mock::judge0::setup_unhealthy_endpoints
export -f mock::judge0::setup_installing_endpoints
export -f mock::judge0::setup_stopped_endpoints
export -f mock::judge0::setup_rate_limited_endpoints

export -f mock::judge0::inject_error
export -f mock::judge0::clear_errors
export -f mock::judge0::create_submission
export -f mock::judge0::set_submission_result
export -f mock::judge0::set_config
export -f mock::judge0::get_config
export -f mock::judge0::set_worker_status
export -f mock::judge0::get_worker_count

export -f mock::judge0::assert_submission_exists
export -f mock::judge0::assert_submission_completed
export -f mock::judge0::assert_language_supported
export -f mock::judge0::get_submission_count
export -f mock::judge0::dump_state

echo "[JUDGE0_MOCK] Judge0 mock implementation loaded"