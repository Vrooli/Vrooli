#!/usr/bin/env bash
# Judge0 Mock - Tier 2 (Stateful)
# 
# Provides stateful Judge0 code execution mocking for testing:
# - Code submission and execution
# - Language support management
# - Batch submission handling
# - Result retrieval
# - Error injection for resilience testing
#
# Coverage: ~80% of common Judge0 operations in 350 lines

# === Configuration ===
declare -gA JUDGE0_SUBMISSIONS=()         # Token -> "language|status|output|time"
declare -gA JUDGE0_LANGUAGES=()           # Lang_id -> "name|version"
declare -gA JUDGE0_BATCHES=()             # Batch_id -> "tokens|status"
declare -gA JUDGE0_CONFIG=(               # Service configuration
    [status]="running"
    [port]="2358"
    [workers]="4"
    [max_cpu_time]="2.0"
    [max_memory]="128000"
    [error_mode]=""
    [version]="1.13.0"
)

# Debug mode
declare -g JUDGE0_DEBUG="${JUDGE0_DEBUG:-}"

# === Helper Functions ===
judge0_debug() {
    [[ -n "$JUDGE0_DEBUG" ]] && echo "[MOCK:JUDGE0] $*" >&2
}

judge0_check_error() {
    case "${JUDGE0_CONFIG[error_mode]}" in
        "service_down")
            echo "Error: Judge0 service is not running" >&2
            return 1
            ;;
        "compilation_error")
            echo "Error: Compilation failed" >&2
            return 1
            ;;
        "timeout")
            echo "Error: Time limit exceeded" >&2
            return 1
            ;;
    esac
    return 0
}

judge0_generate_token() {
    printf "%s-%08x" "$(date +%s)" "$RANDOM"
}

# === Main Judge0 Command ===
judge0() {
    judge0_debug "judge0 called with: $*"
    
    if ! judge0_check_error; then
        return $?
    fi
    
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        submit)
            judge0_cmd_submit "$@"
            ;;
        get)
            judge0_cmd_get "$@"
            ;;
        batch)
            judge0_cmd_batch "$@"
            ;;
        languages)
            judge0_cmd_languages "$@"
            ;;
        status)
            judge0_cmd_status "$@"
            ;;
        start|stop|restart)
            judge0_cmd_service "$command" "$@"
            ;;
        *)
            echo "Judge0 CLI - Code Execution System"
            echo "Commands:"
            echo "  submit    - Submit code for execution"
            echo "  get       - Get submission result"
            echo "  batch     - Batch submission operations"
            echo "  languages - List supported languages"
            echo "  status    - Show service status"
            echo "  start     - Start service"
            echo "  stop      - Stop service"
            echo "  restart   - Restart service"
            ;;
    esac
}

# === Submit Code ===
judge0_cmd_submit() {
    local language="${1:-}"
    local code="${2:-}"
    local input="${3:-}"
    
    [[ -z "$language" ]] && { echo "Error: language ID required" >&2; return 1; }
    [[ -z "$code" ]] && { echo "Error: source code required" >&2; return 1; }
    
    local token=$(judge0_generate_token)
    
    # Simulate execution based on language
    local output status exec_time
    case "$language" in
        71|74)  # Python
            output="Hello, World!"
            status="Accepted"
            exec_time="0.042"
            ;;
        62|63)  # Java
            output="Hello, World!"
            status="Accepted"
            exec_time="0.128"
            ;;
        52|54)  # C++
            output="Hello, World!"
            status="Accepted"
            exec_time="0.015"
            ;;
        *)
            output="Hello from language $language"
            status="Accepted"
            exec_time="0.050"
            ;;
    esac
    
    JUDGE0_SUBMISSIONS[$token]="$language|$status|$output|$exec_time"
    
    echo "Submission created"
    echo "Token: $token"
    echo "Status: In Queue"
    
    # Simulate processing
    sleep 0.1
    JUDGE0_SUBMISSIONS[$token]="$language|$status|$output|$exec_time"
}

# === Get Submission ===
judge0_cmd_get() {
    local token="${1:-}"
    
    [[ -z "$token" ]] && { echo "Error: submission token required" >&2; return 1; }
    
    if [[ -z "${JUDGE0_SUBMISSIONS[$token]}" ]]; then
        echo "Error: submission not found: $token" >&2
        return 1
    fi
    
    local data="${JUDGE0_SUBMISSIONS[$token]}"
    IFS='|' read -r language status output exec_time <<< "$data"
    
    echo "Submission: $token"
    echo "Language: $language"
    echo "Status: $status"
    echo "Output: $output"
    echo "Time: ${exec_time}s"
    echo "Memory: 3124 KB"
}

# === Batch Operations ===
judge0_cmd_batch() {
    local action="${1:-submit}"
    shift || true
    
    case "$action" in
        submit)
            local count="${1:-2}"
            local batch_id="batch_$(judge0_generate_token)"
            local tokens=""
            
            echo "Creating batch submission..."
            for ((i=1; i<=count; i++)); do
                local token=$(judge0_generate_token)
                JUDGE0_SUBMISSIONS[$token]="71|Accepted|Output $i|0.05"
                tokens="$tokens$token,"
            done
            
            JUDGE0_BATCHES[$batch_id]="${tokens%,}|completed"
            echo "Batch created: $batch_id"
            echo "Submissions: $count"
            ;;
        get)
            local batch_id="${1:-}"
            [[ -z "$batch_id" ]] && { echo "Error: batch ID required" >&2; return 1; }
            
            if [[ -n "${JUDGE0_BATCHES[$batch_id]}" ]]; then
                local data="${JUDGE0_BATCHES[$batch_id]}"
                IFS='|' read -r tokens status <<< "$data"
                echo "Batch: $batch_id"
                echo "Status: $status"
                echo "Tokens: ${tokens//,/ }"
            else
                echo "Error: batch not found: $batch_id" >&2
                return 1
            fi
            ;;
        *)
            echo "Usage: judge0 batch {submit|get} [args]"
            return 1
            ;;
    esac
}

# === Languages Command ===
judge0_cmd_languages() {
    echo "Supported Languages:"
    echo "ID  | Name              | Version"
    echo "----|-------------------|----------"
    echo "71  | Python            | 3.9.7"
    echo "74  | Python            | 3.11.0"
    echo "62  | Java              | OpenJDK 17"
    echo "63  | JavaScript        | Node.js 18"
    echo "52  | C++               | GCC 11.2"
    echo "54  | C++               | GCC 12.1"
    echo "73  | Rust              | 1.65.0"
    echo "72  | Ruby              | 3.0.2"
    echo "68  | PHP               | 8.1.0"
    echo "51  | C#                | .NET 6.0"
    
    # Store for reference
    JUDGE0_LANGUAGES[71]="Python|3.9.7"
    JUDGE0_LANGUAGES[74]="Python|3.11.0"
    JUDGE0_LANGUAGES[62]="Java|OpenJDK 17"
}

# === Status Command ===
judge0_cmd_status() {
    echo "Judge0 Status"
    echo "============="
    echo "Service: ${JUDGE0_CONFIG[status]}"
    echo "Port: ${JUDGE0_CONFIG[port]}"
    echo "Workers: ${JUDGE0_CONFIG[workers]}"
    echo "Max CPU Time: ${JUDGE0_CONFIG[max_cpu_time]}s"
    echo "Max Memory: ${JUDGE0_CONFIG[max_memory]} KB"
    echo "Version: ${JUDGE0_CONFIG[version]}"
    echo ""
    echo "Submissions: ${#JUDGE0_SUBMISSIONS[@]}"
    echo "Batches: ${#JUDGE0_BATCHES[@]}"
    echo "Languages: ${#JUDGE0_LANGUAGES[@]}"
}

# === Service Management ===
judge0_cmd_service() {
    local action="$1"
    
    case "$action" in
        start)
            if [[ "${JUDGE0_CONFIG[status]}" == "running" ]]; then
                echo "Judge0 is already running"
            else
                JUDGE0_CONFIG[status]="running"
                echo "Judge0 started on port ${JUDGE0_CONFIG[port]}"
            fi
            ;;
        stop)
            JUDGE0_CONFIG[status]="stopped"
            echo "Judge0 stopped"
            ;;
        restart)
            JUDGE0_CONFIG[status]="stopped"
            JUDGE0_CONFIG[status]="running"
            echo "Judge0 restarted"
            ;;
    esac
}

# === HTTP API Mock ===
curl() {
    judge0_debug "curl called with: $*"
    
    local url="" method="GET" data=""
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -X) method="$2"; shift 2 ;;
            -d|--data) data="$2"; shift 2 ;;
            http*) url="$1"; shift ;;
            *) shift ;;
        esac
    done
    
    if [[ "$url" =~ localhost:2358 || "$url" =~ 127.0.0.1:2358 ]]; then
        judge0_handle_api "$method" "$url" "$data"
        return $?
    fi
    
    echo "curl: Not a Judge0 endpoint"
    return 0
}

judge0_handle_api() {
    local method="$1" url="$2" data="$3"
    
    case "$url" in
        */submissions)
            if [[ "$method" == "POST" ]]; then
                local token=$(judge0_generate_token)
                echo '{"token":"'$token'"}'
            else
                echo '{"submissions":['$(( ${#JUDGE0_SUBMISSIONS[@]} ))']}'
            fi
            ;;
        */submissions/*)
            local token="${url##*/submissions/}"
            if [[ -n "${JUDGE0_SUBMISSIONS[$token]}" ]]; then
                echo '{"token":"'$token'","status":{"id":3,"description":"Accepted"},"stdout":"Hello, World!","time":"0.05"}'
            else
                echo '{"error":"Submission not found"}'
            fi
            ;;
        */languages)
            echo '[{"id":71,"name":"Python 3"},{"id":62,"name":"Java"},{"id":52,"name":"C++"}]'
            ;;
        */health)
            if [[ "${JUDGE0_CONFIG[status]}" == "running" ]]; then
                echo '{"status":"available","version":"'${JUDGE0_CONFIG[version]}'"}'
            else
                echo '{"status":"unavailable"}'
            fi
            ;;
        *)
            echo '{"status":"ok","version":"'${JUDGE0_CONFIG[version]}'"}'
            ;;
    esac
}

# === Mock Control Functions ===
judge0_mock_reset() {
    judge0_debug "Resetting mock state"
    
    JUDGE0_SUBMISSIONS=()
    JUDGE0_LANGUAGES=()
    JUDGE0_BATCHES=()
    JUDGE0_CONFIG[error_mode]=""
    JUDGE0_CONFIG[status]="running"
}

judge0_mock_set_error() {
    JUDGE0_CONFIG[error_mode]="$1"
    judge0_debug "Set error mode: $1"
}

judge0_mock_dump_state() {
    echo "=== Judge0 Mock State ==="
    echo "Status: ${JUDGE0_CONFIG[status]}"
    echo "Port: ${JUDGE0_CONFIG[port]}"
    echo "Workers: ${JUDGE0_CONFIG[workers]}"
    echo "Submissions: ${#JUDGE0_SUBMISSIONS[@]}"
    echo "Batches: ${#JUDGE0_BATCHES[@]}"
    echo "Languages: ${#JUDGE0_LANGUAGES[@]}"
    echo "Error Mode: ${JUDGE0_CONFIG[error_mode]:-none}"
    echo "====================="
}

# === Convention-based Test Functions ===
test_judge0_connection() {
    judge0_debug "Testing connection..."
    
    local result
    result=$(curl -s http://localhost:2358/health 2>&1)
    
    if [[ "$result" =~ "status" ]]; then
        judge0_debug "Connection test passed"
        return 0
    else
        judge0_debug "Connection test failed"
        return 1
    fi
}

test_judge0_health() {
    judge0_debug "Testing health..."
    
    test_judge0_connection || return 1
    
    judge0 languages >/dev/null 2>&1 || return 1
    
    judge0_debug "Health test passed"
    return 0
}

test_judge0_basic() {
    judge0_debug "Testing basic operations..."
    
    judge0 submit 71 'print("Hello")' >/dev/null 2>&1 || return 1
    judge0 batch submit 3 >/dev/null 2>&1 || return 1
    
    judge0_debug "Basic test passed"
    return 0
}

# === Export Functions ===
export -f judge0 curl
export -f test_judge0_connection test_judge0_health test_judge0_basic
export -f judge0_mock_reset judge0_mock_set_error judge0_mock_dump_state
export -f judge0_debug judge0_check_error

# Initialize
judge0_mock_reset
judge0_debug "Judge0 Tier 2 mock initialized"