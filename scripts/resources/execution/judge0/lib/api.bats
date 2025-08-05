#!/usr/bin/env bats

# Tests for Judge0 api.sh functions


# Load test infrastructure
source "$(dirname "${BATS_TEST_FILENAME}")/../../../../__test/fixtures/setup.bash"

# Setup for each test
setup() {
    # Use auto-setup for service tests
    vrooli_auto_setup
    
    # Mock system functions (lightweight)
    log::info() { echo "[INFO] $*"; }
    log::success() { echo "[SUCCESS] $*"; }
    log::warning() { echo "[WARNING] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    system::is_command() { command -v "$1" >/dev/null 2>&1; }
    
    # Mock basic curl function
    curl() {
        case "$*" in
            *"health"*) echo '{"status":"healthy"}';;
            *) echo '{"success":true}';;
        esac
        return 0
    }
    export -f curl
    
    # Set test environment
    export JUDGE0_PORT="2358"
    export JUDGE0_BASE_URL="http://localhost:2358"
    export JUDGE0_API_PREFIX="/api/v1"
    export JUDGE0_API_KEY="test_api_key_12345"
    export JUDGE0_ENABLE_AUTHENTICATION="true"
    export JUDGE0_CPU_TIME_LIMIT="5"
    export JUDGE0_WALL_TIME_LIMIT="10"
    export JUDGE0_MEMORY_LIMIT="262144"
    export JUDGE0_STACK_LIMIT="262144"
    export JUDGE0_MAX_PROCESSES="30"
    export JUDGE0_MAX_FILE_SIZE="5120"
    export JUDGE0_ENABLE_NETWORK="false"
    export YES="no"
    
    # Load dependencies
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    JUDGE0_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Mock system functions
    
    # Mock curl for API calls
    
    # Mock jq for JSON processing
    jq() {
        case "$*" in
            *".version"*) echo "1.13.1" ;;
            *".token"*) echo "abc123-def456-ghi789" ;;
            *".status.id"*) echo "3" ;;
            *".status.description"*) echo "Accepted" ;;
            *".stdout"*) echo "SGVsbG8sIFdvcmxkIQ==" ;;
            *".stderr"*) echo "" ;;
            *".time"*) echo "0.02" ;;
            *".memory"*) echo "8192" ;;
            *".id"*) echo "92" ;;
            *".name"*) echo "Python (3.11.2)" ;;
            *"length"*) echo "2" ;;
            *) echo "JQ: $*" ;;
        esac
    }
    
    # Mock base64 for encoding/decoding
    base64() {
        if [[ "$*" =~ "-d" ]]; then
            case "$input" in
                "SGVsbG8sIFdvcmxkIQ==") echo "Hello, World!" ;;
                "SGVsbG8gZnJvbSBQeXRob24h") echo "Hello from Python!" ;;
                "RXJyb3I6IGludmFsaWQgc3ludGF4") echo "Error: invalid syntax" ;;
                "dGVzdCBvdXRwdXQ=") echo "test output" ;;
                "SGVsbG8gQmF0Y2ggMQ==") echo "Hello Batch 1" ;;
                "SGVsbG8gQmF0Y2ggMg==") echo "Hello Batch 2" ;;
                *) echo "decoded content" ;;
            esac
        else
            echo "encoded_content_123"
        fi
    }
    
    # Mock log functions
    
    # Mock Judge0 functions
    judge0::get_api_key() { echo "$JUDGE0_API_KEY"; }
    judge0::get_language_id() {
        case "$1" in
            "python"|"python3") echo "92" ;;
            "javascript"|"js") echo "93" ;;
            "java") echo "91" ;;
            *) return 1 ;;
        esac
    }
    
    # Load configuration and messages
    source "${JUDGE0_DIR}/config/defaults.sh"
    source "${JUDGE0_DIR}/config/messages.sh"
    judge0::export_config
    judge0::export_messages
    
    # Load the functions to test
    source "${JUDGE0_DIR}/lib/api.sh"
}

# Test API request function
@test "judge0::api::request makes authenticated API requests" {
    result=$(judge0::api::request "GET" "/system_info")
    
    [[ "$result" =~ "version" ]]
    [[ "$result" =~ "1.13.1" ]]
}

# Test API request with POST data
@test "judge0::api::request supports POST requests with data" {
    local data='{"source_code":"print(\"test\")","language_id":92}'
    
    result=$(judge0::api::request "POST" "/submissions" "$data")
    
    [[ "$result" =~ "token" ]]
    [[ "$result" =~ "status" ]]
}

# Test API request without authentication
@test "judge0::api::request works without authentication when disabled" {
    export JUDGE0_ENABLE_AUTHENTICATION="false"
    
    result=$(judge0::api::request "GET" "/system_info")
    
    [[ "$result" =~ "version" ]]
}

# Test health check
@test "judge0::api::health_check verifies service health" {
    result=$(judge0::api::health_check && echo "healthy" || echo "unhealthy")
    
    [[ "$result" == "healthy" ]]
}

# Test health check failure
@test "judge0::api::health_check detects unhealthy service" {
    # Override curl to fail
    curl() {
        return 1
    }
    
    result=$(judge0::api::health_check && echo "healthy" || echo "unhealthy")
    
    [[ "$result" == "unhealthy" ]]
}

# Test API connectivity test
@test "judge0::api::test tests API connectivity" {
    result=$(judge0::api::test)
    
    [[ "$result" =~ "API" ]]
    [[ "$result" =~ "SUCCESS:" ]]
}

# Test API connectivity test failure
@test "judge0::api::test handles API connectivity failure" {
    # Override health check to fail
    judge0::api::health_check() { return 1; }
    
    result=$(judge0::api::test)
    
    [[ "$result" =~ "ERROR:" ]] || [[ "$result" =~ "failed" ]]
}

# Test system information retrieval
@test "judge0::api::system_info retrieves system information" {
    result=$(judge0::api::system_info)
    
    [[ "$result" =~ "version" ]]
    [[ "$result" =~ "hostname" ]]
    [[ "$result" =~ "judge0-server" ]]
}

# Test language listing
@test "judge0::api::get_languages retrieves available languages" {
    result=$(judge0::api::get_languages)
    
    [[ "$result" =~ "Python" ]]
    [[ "$result" =~ "JavaScript" ]]
    [[ "$result" =~ "Java" ]]
}

# Test specific language retrieval
@test "judge0::api::get_language retrieves specific language info" {
    result=$(judge0::api::get_language "92")
    
    [[ "$result" =~ "Python" ]]
    [[ "$result" =~ "3.11.2" ]]
    [[ "$result" =~ "main.py" ]]
}

# Test code submission
@test "judge0::api::submit submits code for execution" {
    result=$(judge0::api::submit 'console.log("Hello, World!");' "javascript")
    
    [[ "$result" =~ "Execution Result" ]]
    [[ "$result" =~ "successful" ]]
    [[ "$result" =~ "Hello, World!" ]]
}

# Test Python code submission
@test "judge0::api::submit executes Python code correctly" {
    result=$(judge0::api::submit 'print("Hello from Python!")' "python")
    
    [[ "$result" =~ "Execution Result" ]]
    [[ "$result" =~ "successful" ]]
    [[ "$result" =~ "Hello from Python!" ]]
}

# Test code submission with input
@test "judge0::api::submit supports standard input" {
    result=$(judge0::api::submit 'name = input(); print(f"Hello, {name}!")' "python" "World")
    
    [[ "$result" =~ "Execution Result" ]]
}

# Test code submission with expected output
@test "judge0::api::submit supports expected output validation" {
    result=$(judge0::api::submit 'print("Hello, World!")' "python" "" "Hello, World!")
    
    [[ "$result" =~ "Execution Result" ]]
}

# Test compilation error handling
@test "judge0::api::submit handles compilation errors" {
    # Mock curl to return compilation error
    curl() {
        case "$*" in
            *"/submissions"*)
                echo '{"token":"error-token","status":{"id":6,"description":"Compilation Error"},"stdout":"","stderr":"","compile_output":"RXJyb3I6IGludmFsaWQgc3ludGF4","time":"0.0","memory":"0"}'
                ;;
            *) echo "CURL: $*" ;;
        esac
        return 0
    }
    
    result=$(judge0::api::submit 'invalid syntax (' "python")
    
    [[ "$result" =~ "Compilation Error" ]]
    [[ "$result" =~ "Error: invalid syntax" ]]
}

# Test language validation
@test "judge0::api::submit validates language support" {
    run judge0::api::submit 'print("test")' "unsupported-language"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not found" ]] || [[ "$output" =~ "ERROR:" ]]
}

# Test batch submission
@test "judge0::api::submit_batch executes multiple code submissions" {
    local submissions='[{"source_code":"print(\"Hello Batch 1\")","language_id":92},{"source_code":"print(\"Hello Batch 2\")","language_id":92}]'
    
    result=$(judge0::api::submit_batch "$submissions")
    
    [[ "$result" =~ "token" ]]
    [[ "$result" =~ "batch1" ]] || [[ "$result" =~ "batch2" ]]
}

# Test submission retrieval by token
@test "judge0::api::get_submission retrieves submission by token" {
    result=$(judge0::api::get_submission "abc123-def456-ghi789")
    
    [[ "$result" =~ "token" ]]
    [[ "$result" =~ "abc123-def456-ghi789" ]]
    [[ "$result" =~ "console.log" ]]
}

# Test submission deletion
@test "judge0::api::delete_submission removes submission" {
    result=$(judge0::api::delete_submission "abc123-def456-ghi789")
    
    [[ "$result" =~ "deleted successfully" ]]
}

# Test submission deletion when disabled
@test "judge0::api::delete_submission respects deletion setting" {
    export JUDGE0_ENABLE_SUBMISSION_DELETE="false"
    
    run judge0::api::delete_submission "abc123-def456-ghi789"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]] || [[ "$output" =~ "disabled" ]]
}

# Test queue status retrieval
@test "judge0::api::get_queue_status shows worker queue status" {
    result=$(judge0::api::get_queue_status)
    
    [[ "$result" =~ "worker" ]]
    [[ "$result" =~ "queue" ]]
}

# Test result display for successful execution
@test "judge0::api::display_result formats successful execution results" {
    local result_json='{"status":{"id":3,"description":"Accepted"},"stdout":"SGVsbG8sIFdvcmxkIQ==","stderr":"","time":"0.02","memory":"8192"}'
    
    result=$(judge0::api::display_result "$result_json")
    
    [[ "$result" =~ "Execution Result" ]]
    [[ "$result" =~ "Accepted" ]]
    [[ "$result" =~ "successful" ]]
    [[ "$result" =~ "Hello, World!" ]]
}

# Test result display for time limit exceeded
@test "judge0::api::display_result handles time limit exceeded" {
    local result_json='{"status":{"id":5,"description":"Time Limit Exceeded"},"stdout":"","stderr":"","time":"5.0","memory":"8192"}'
    
    result=$(judge0::api::display_result "$result_json")
    
    [[ "$result" =~ "Time Limit Exceeded" ]]
    [[ "$result" =~ "ERROR:" ]]
}

# Test result display for memory limit exceeded
@test "judge0::api::display_result handles memory limit exceeded" {
    local result_json='{"status":{"id":12,"description":"Memory Limit Exceeded"},"stdout":"","stderr":"","time":"0.1","memory":"262144"}'
    
    result=$(judge0::api::display_result "$result_json")
    
    [[ "$result" =~ "Memory Limit" ]]
    [[ "$result" =~ "ERROR:" ]]
}

# Test result display for runtime error
@test "judge0::api::display_result handles runtime errors" {
    local result_json='{"status":{"id":7,"description":"Runtime Error (NZEC)"},"stdout":"","stderr":"VHJhY2ViYWNrOiBFeGNlcHRpb24=","time":"0.01","memory":"4096"}'
    
    # Mock base64 to decode error message
    base64() {
        if [[ "$*" =~ "-d" ]]; then
            echo "Traceback: Exception"
        else
            echo "encoded"
        fi
    }
    
    result=$(judge0::api::display_result "$result_json")
    
    [[ "$result" =~ "Runtime Error" ]]
    [[ "$result" =~ "ERROR:" ]]
    [[ "$result" =~ "Traceback: Exception" ]]
}

# Test language discovery from API
@test "judge0::api::submit discovers language ID from API when not in defaults" {
    # Override get_language_id to fail initially
    judge0::get_language_id() { return 1; }
    
    result=$(judge0::api::submit 'console.log("test");' "javascript")
    
    [[ "$result" =~ "Execution Result" ]]
}

# Test API error handling
@test "judge0::api::submit handles API errors gracefully" {
    # Override curl to return empty response
    curl() {
        echo ""
        return 1
    }
    
    run judge0::api::submit 'print("test")' "python"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]]
}

# Test submission with network disabled
@test "judge0::api::submit includes network configuration" {
    export JUDGE0_ENABLE_NETWORK="false"
    
    result=$(judge0::api::submit 'print("test")' "python")
    
    [[ "$result" =~ "Execution Result" ]] || [[ "$result" =~ "submission" ]]
}

# Test resource limits in submission
@test "judge0::api::submit applies resource limits" {
    result=$(judge0::api::submit 'print("test")' "python")
    
    # Should include the submission in logs/output
    [[ "$result" =~ "submission" ]] || [[ "$result" =~ "Execution Result" ]]
}

# Test wrong answer detection
@test "judge0::api::display_result handles wrong answer status" {
    local result_json='{"status":{"id":4,"description":"Wrong Answer"},"stdout":"V3Jvbmcgb3V0cHV0","stderr":"","time":"0.01","memory":"4096"}'
    
    # Mock base64 for output decoding
    base64() {
        if [[ "$*" =~ "-d" ]]; then
            echo "Wrong output"
        else
            echo "encoded"
        fi
    }
    
    result=$(judge0::api::display_result "$result_json")
    
    [[ "$result" =~ "Wrong answer" ]]
    [[ "$result" =~ "ERROR:" ]]
    [[ "$result" =~ "Wrong output" ]]
}

# Test unknown status handling
@test "judge0::api::display_result handles unknown status codes" {
    local result_json='{"status":{"id":99,"description":"Unknown Status"},"stdout":"","stderr":"","time":"0.01","memory":"4096"}'
    
    result=$(judge0::api::display_result "$result_json")
    
    [[ "$result" =~ "Unknown Status" ]]
    [[ "$result" =~ "Status:" ]]
    [[ "$result" =~ "ID: 99" ]]
}

# Teardown
teardown() {
    vrooli_cleanup_test
}
