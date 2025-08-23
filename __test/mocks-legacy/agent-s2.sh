#!/usr/bin/env bash
# Agent-S2 Resource Mock Implementation
# Provides comprehensive mocking for Agent-S2 browser automation service including CLI commands and HTTP API

# Prevent duplicate loading
if [[ "${AGENTS2_MOCK_LOADED:-}" == "true" ]]; then
    return 0
fi
export AGENTS2_MOCK_LOADED="true"

# ----------------------------
# Mock Configuration & State Management
# ----------------------------

# Default configuration
readonly AGENTS2_DEFAULT_PORT="4113"
readonly AGENTS2_DEFAULT_BASE_URL="http://localhost:4113"
readonly AGENTS2_DEFAULT_CONTAINER_NAME="agent-s2"
readonly AGENTS2_DEFAULT_VNC_PORT="5900"

# Initialize mock state file for subshell access
if [[ -z "${AGENTS2_MOCK_STATE_FILE:-}" ]]; then
    if [[ -n "${BATS_TMPDIR:-}" ]]; then
        AGENTS2_MOCK_STATE_FILE="${BATS_TMPDIR}/agents2_mock_state_test"
    else
        AGENTS2_MOCK_STATE_FILE="${TMPDIR:-/tmp}/agents2_mock_state_$$"
    fi
fi
export AGENTS2_MOCK_STATE_FILE

# Global state arrays
declare -gA MOCK_AGENTS2_STATE=()           # container_name -> "running|stopped|installing|error"
declare -gA MOCK_AGENTS2_SESSIONS=()        # session_id -> "active|idle|terminated"
declare -gA MOCK_AGENTS2_TASKS=()          # task_id -> "status|result|created_at"
declare -gA MOCK_AGENTS2_CALL_COUNT=()     # action -> count
declare -gA MOCK_AGENTS2_ERRORS=()         # action -> error_type
declare -gA MOCK_AGENTS2_CONFIG=()         # config_key -> value
declare -gA MOCK_AGENTS2_STEALTH=()        # feature -> enabled

# Mock modes
export AGENTS2_MOCK_MODE="${AGENTS2_MOCK_MODE:-healthy}"  # healthy|unhealthy|installing|stopped|offline|error

# Configuration
export AGENTS2_PORT="${AGENTS2_PORT:-$AGENTS2_DEFAULT_PORT}"
export AGENTS2_BASE_URL="${AGENTS2_BASE_URL:-$AGENTS2_DEFAULT_BASE_URL}"
export AGENTS2_CONTAINER_NAME="${AGENTS2_CONTAINER_NAME:-$AGENTS2_DEFAULT_CONTAINER_NAME}"
export AGENTS2_VNC_PORT="${AGENTS2_VNC_PORT:-$AGENTS2_DEFAULT_VNC_PORT}"

#######################################
# Initialize state file for subshell access
#######################################
_agents2_mock_init_state_file() {
    if [[ ! -f "$AGENTS2_MOCK_STATE_FILE" ]]; then
        local state_dir
        state_dir=$(dirname "$AGENTS2_MOCK_STATE_FILE")
        mkdir -p "$state_dir" 2>/dev/null || true
        
        cat > "$AGENTS2_MOCK_STATE_FILE" << 'EOF'
# Agent-S2 mock state file
declare -gA MOCK_AGENTS2_STATE=()
declare -gA MOCK_AGENTS2_SESSIONS=()
declare -gA MOCK_AGENTS2_TASKS=()
declare -gA MOCK_AGENTS2_CALL_COUNT=()
declare -gA MOCK_AGENTS2_ERRORS=()
declare -gA MOCK_AGENTS2_CONFIG=()
declare -gA MOCK_AGENTS2_STEALTH=()
EOF
    fi
}

#######################################
# Save state to file for subshell access
#######################################
_agents2_mock_save_state() {
    local state_dir
    state_dir=$(dirname "$AGENTS2_MOCK_STATE_FILE")
    mkdir -p "$state_dir" 2>/dev/null || true
    
    {
        echo "# Agent-S2 mock state file - auto generated"
        
        # MOCK_AGENTS2_STATE
        echo "declare -gA MOCK_AGENTS2_STATE"
        for key in "${!MOCK_AGENTS2_STATE[@]}"; do
            printf 'MOCK_AGENTS2_STATE[%q]=%q\n' "$key" "${MOCK_AGENTS2_STATE[$key]}"
        done
        
        # MOCK_AGENTS2_SESSIONS
        echo "declare -gA MOCK_AGENTS2_SESSIONS"
        for key in "${!MOCK_AGENTS2_SESSIONS[@]}"; do
            printf 'MOCK_AGENTS2_SESSIONS[%q]=%q\n' "$key" "${MOCK_AGENTS2_SESSIONS[$key]}"
        done
        
        # MOCK_AGENTS2_TASKS
        echo "declare -gA MOCK_AGENTS2_TASKS"
        for key in "${!MOCK_AGENTS2_TASKS[@]}"; do
            printf 'MOCK_AGENTS2_TASKS[%q]=%q\n' "$key" "${MOCK_AGENTS2_TASKS[$key]}"
        done
        
        # MOCK_AGENTS2_CALL_COUNT
        echo "declare -gA MOCK_AGENTS2_CALL_COUNT"
        for key in "${!MOCK_AGENTS2_CALL_COUNT[@]}"; do
            printf 'MOCK_AGENTS2_CALL_COUNT[%q]=%q\n' "$key" "${MOCK_AGENTS2_CALL_COUNT[$key]}"
        done
        
        # MOCK_AGENTS2_ERRORS
        echo "declare -gA MOCK_AGENTS2_ERRORS"
        for key in "${!MOCK_AGENTS2_ERRORS[@]}"; do
            printf 'MOCK_AGENTS2_ERRORS[%q]=%q\n' "$key" "${MOCK_AGENTS2_ERRORS[$key]}"
        done
        
        # MOCK_AGENTS2_CONFIG
        echo "declare -gA MOCK_AGENTS2_CONFIG"
        for key in "${!MOCK_AGENTS2_CONFIG[@]}"; do
            printf 'MOCK_AGENTS2_CONFIG[%q]=%q\n' "$key" "${MOCK_AGENTS2_CONFIG[$key]}"
        done
        
        # MOCK_AGENTS2_STEALTH
        echo "declare -gA MOCK_AGENTS2_STEALTH"
        for key in "${!MOCK_AGENTS2_STEALTH[@]}"; do
            printf 'MOCK_AGENTS2_STEALTH[%q]=%q\n' "$key" "${MOCK_AGENTS2_STEALTH[$key]}"
        done
        
        # Export scalar variables
        echo "export AGENTS2_MOCK_MODE='$AGENTS2_MOCK_MODE'"
        echo "export AGENTS2_PORT='$AGENTS2_PORT'"
        echo "export AGENTS2_BASE_URL='$AGENTS2_BASE_URL'"
        echo "export AGENTS2_CONTAINER_NAME='$AGENTS2_CONTAINER_NAME'"
        echo "export AGENTS2_VNC_PORT='$AGENTS2_VNC_PORT'"
    } > "$AGENTS2_MOCK_STATE_FILE"
}

# Initialize state file only if it doesn't exist
if [[ ! -f "$AGENTS2_MOCK_STATE_FILE" ]]; then
    _agents2_mock_init_state_file
fi

# ----------------------------
# Utilities
# ----------------------------

#######################################
# Load state from file - use everywhere state is needed
#######################################
_agents2_load_state() {
    if [[ -f "$AGENTS2_MOCK_STATE_FILE" && -s "$AGENTS2_MOCK_STATE_FILE" ]]; then
        source "$AGENTS2_MOCK_STATE_FILE" 2>/dev/null || true
    fi
}

_mock_current_timestamp() {
    date -u +"%Y-%m-%dT%H:%M:%S.%3NZ" 2>/dev/null || date -u +"%Y-%m-%dT%H:%M:%SZ"
}

_mock_task_id() {
    echo "task-$(date +%s)-$RANDOM"
}

_mock_session_id() {
    echo "session-$(uuidgen 2>/dev/null || echo "$(date +%s)-$RANDOM")"
}

# ----------------------------
# Public API Functions
# ----------------------------

#######################################
# Reset all mock state
#######################################
mock::agents2::reset() {
    # Clear all arrays completely
    unset MOCK_AGENTS2_STATE
    unset MOCK_AGENTS2_SESSIONS
    unset MOCK_AGENTS2_TASKS
    unset MOCK_AGENTS2_CALL_COUNT
    unset MOCK_AGENTS2_ERRORS
    unset MOCK_AGENTS2_CONFIG
    unset MOCK_AGENTS2_STEALTH
    
    # Redeclare arrays
    declare -gA MOCK_AGENTS2_STATE=()
    declare -gA MOCK_AGENTS2_SESSIONS=()
    declare -gA MOCK_AGENTS2_TASKS=()
    declare -gA MOCK_AGENTS2_CALL_COUNT=()
    declare -gA MOCK_AGENTS2_ERRORS=()
    declare -gA MOCK_AGENTS2_CONFIG=()
    declare -gA MOCK_AGENTS2_STEALTH=()
    
    # Reset to default state
    export AGENTS2_MOCK_MODE="healthy"
    
    # Initialize default state (not installed initially)
    # Don't set any state initially - this means "not_installed"
    MOCK_AGENTS2_CONFIG["llm_provider"]="ollama"
    MOCK_AGENTS2_CONFIG["llm_model"]="llama3.2-vision:11b"
    MOCK_AGENTS2_CONFIG["enable_ai"]="yes"
    MOCK_AGENTS2_CONFIG["enable_search"]="no"
    MOCK_AGENTS2_CONFIG["mode"]="sandbox"
    MOCK_AGENTS2_CONFIG["vnc_password"]="agents2vnc"
    MOCK_AGENTS2_CONFIG["enable_host_display"]="no"
    MOCK_AGENTS2_CONFIG["security_profile"]="moderate"
    
    # Initialize default stealth features
    MOCK_AGENTS2_STEALTH["enabled"]="no"
    MOCK_AGENTS2_STEALTH["webdriver_override"]="no"
    MOCK_AGENTS2_STEALTH["navigator_spoofing"]="no"
    MOCK_AGENTS2_STEALTH["canvas_fingerprint"]="no"
    
    # Save state to file for subshell access
    _agents2_mock_save_state
    
    echo "[MOCK] Agent-S2 state reset with default configuration"
}

#######################################
# Set mock mode
# Arguments: $1 - mode (healthy|unhealthy|installing|stopped|offline|error)
#######################################
mock::agents2::set_mode() {
    local mode="$1"
    
    # Load existing state first
    _agents2_load_state
    
    case "$mode" in
        healthy|unhealthy|installing|stopped|offline|error)
            export AGENTS2_MOCK_MODE="$mode"
            
            # Update container state based on mode
            case "$mode" in
                healthy)
                    MOCK_AGENTS2_STATE["$AGENTS2_CONTAINER_NAME"]="running"
                    ;;
                unhealthy)
                    MOCK_AGENTS2_STATE["$AGENTS2_CONTAINER_NAME"]="unhealthy"
                    ;;
                installing)
                    MOCK_AGENTS2_STATE["$AGENTS2_CONTAINER_NAME"]="installing"
                    ;;
                stopped)
                    MOCK_AGENTS2_STATE["$AGENTS2_CONTAINER_NAME"]="stopped"
                    ;;
                offline|error)
                    MOCK_AGENTS2_STATE["$AGENTS2_CONTAINER_NAME"]="error"
                    ;;
            esac
            ;;
        *)
            echo "[MOCK] Invalid Agent-S2 mode: $mode" >&2
            return 1
            ;;
    esac
    
    # Save state for subshell access
    _agents2_mock_save_state
    
    # Use centralized logging if available
    if command -v mock::log_state &>/dev/null; then
        mock::log_state "agents2_mode" "global" "$mode"
    fi
    
    # Setup HTTP endpoints based on mode if HTTP mock is available
    if command -v mock::http::set_endpoint_response &>/dev/null; then
        case "$mode" in
            healthy)
                mock::agents2::setup_healthy_endpoints
                ;;
            unhealthy)
                mock::agents2::setup_unhealthy_endpoints
                ;;
            installing)
                mock::agents2::setup_installing_endpoints
                ;;
            stopped|offline)
                mock::agents2::setup_stopped_endpoints
                ;;
        esac
    fi
    
    return 0
}

#######################################
# Create a new task
# Arguments: $1 - task type, $2 - parameters (optional)
#######################################
mock::agents2::create_task() {
    local task_type="$1"
    local params="${2:-}"
    local task_id=$(_mock_task_id)
    
    # Load existing state first to preserve other tasks
    _agents2_load_state
    
    MOCK_AGENTS2_TASKS["$task_id"]="accepted|pending|$(_mock_current_timestamp)"
    
    # Save state to file for subshell access
    _agents2_mock_save_state
    
    echo "$task_id"
}

#######################################
# Update task status
# Arguments: $1 - task id, $2 - status, $3 - result (optional)
#######################################
mock::agents2::update_task() {
    local task_id="$1"
    local status="$2"
    local result="${3:-}"
    
    # Load existing state first
    _agents2_load_state
    
    if [[ -n "${MOCK_AGENTS2_TASKS[$task_id]:-}" ]]; then
        local created_at=$(echo "${MOCK_AGENTS2_TASKS[$task_id]}" | cut -d'|' -f3)
        MOCK_AGENTS2_TASKS["$task_id"]="$status|$result|$created_at"
        _agents2_mock_save_state
        return 0
    fi
    return 1
}

#######################################
# Create a new session
# Arguments: $1 - profile (optional)
#######################################
mock::agents2::create_session() {
    local profile="${1:-default}"
    local session_id=$(_mock_session_id)
    
    # Load existing state first to preserve other sessions
    _agents2_load_state
    
    MOCK_AGENTS2_SESSIONS["$session_id"]="active"
    
    # Save state to file for subshell access
    _agents2_mock_save_state
    
    echo "$session_id"
}

#######################################
# Update configuration
# Arguments: $1 - key, $2 - value
#######################################
mock::agents2::set_config() {
    local key="$1"
    local value="$2"
    
    # Load existing state first
    _agents2_load_state
    
    MOCK_AGENTS2_CONFIG["$key"]="$value"
    
    # Save state to file for subshell access
    _agents2_mock_save_state
}

#######################################
# Configure stealth feature
# Arguments: $1 - feature, $2 - enabled (yes/no)
#######################################
mock::agents2::set_stealth() {
    local feature="$1"
    local enabled="$2"
    
    # Load existing state first
    _agents2_load_state
    
    MOCK_AGENTS2_STEALTH["$feature"]="$enabled"
    
    # Save state to file for subshell access
    _agents2_mock_save_state
}

#######################################
# Inject error for testing failure scenarios
# Arguments: $1 - action, $2 - error type (optional)
#######################################
mock::agents2::inject_error() {
    local action="$1"
    local error_type="${2:-generic}"
    
    # Load existing state first
    _agents2_load_state
    
    MOCK_AGENTS2_ERRORS["$action"]="$error_type"
    
    # Save state to file for subshell access
    _agents2_mock_save_state
    
    echo "[MOCK] Injected error for $action: $error_type"
}

# ----------------------------
# Setup HTTP Endpoints
# ----------------------------

#######################################
# Setup healthy Agent-S2 endpoints
#######################################
mock::agents2::setup_healthy_endpoints() {
    if ! command -v mock::http::set_endpoint_response &>/dev/null; then
        return 0
    fi
    
    # Health endpoint
    mock::http::set_endpoint_response "$AGENTS2_BASE_URL/health" \
        '{"status":"ok","version":"1.0.0","browser_ready":true,"display_available":true,"vnc_port":5900}'
    
    # Task execution endpoint
    local task_id=$(_mock_task_id)
    mock::http::set_endpoint_response "$AGENTS2_BASE_URL/api/v1/tasks" \
        "{
            \"task_id\": \"$task_id\",
            \"status\": \"accepted\",
            \"estimated_duration\": 30,
            \"created_at\": \"$(_mock_current_timestamp)\"
        }" \
        "POST"
    
    # Task status endpoint (generic)
    mock::http::set_endpoint_response "$AGENTS2_BASE_URL/api/v1/tasks/.*" \
        '{
            "task_id": "task-123",
            "status": "completed",
            "progress": 100,
            "result": {
                "success": true,
                "data": {
                    "screenshot": "base64encodedimage...",
                    "extracted_data": {
                        "title": "Example Page",
                        "text_content": "This is the extracted text"
                    }
                }
            },
            "created_at": "2024-01-15T12:00:00Z",
            "completed_at": "2024-01-15T12:00:30Z"
        }'
    
    # Sessions endpoint
    mock::http::set_endpoint_response "$AGENTS2_BASE_URL/api/v1/sessions" \
        '[
            {
                "session_id": "session-abc123",
                "status": "active",
                "created_at": "2024-01-15T12:00:00Z",
                "pages": 3,
                "profile": "default"
            }
        ]'
    
    # Capabilities endpoint
    mock::http::set_endpoint_response "$AGENTS2_BASE_URL/api/v1/capabilities" \
        '{
            "browser_automation": true,
            "screenshot_capture": true,
            "text_extraction": true,
            "form_interaction": true,
            "javascript_execution": true,
            "proxy_support": true,
            "stealth_mode": true,
            "ai_planning": true,
            "multi_step_execution": true
        }'
    
    # Screenshot endpoint
    mock::http::set_endpoint_response "$AGENTS2_BASE_URL/screenshot" \
        '{"data":"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg==","format":"png","timestamp":"'$(_mock_current_timestamp)'"}' \
        "POST"
    
    # Execute endpoint for various task types
    mock::http::set_endpoint_response "$AGENTS2_BASE_URL/execute" \
        '{"task_id":"'$(_mock_task_id)'","status":"running","message":"Task execution started"}' \
        "POST"
    
    # Mouse position endpoint
    mock::http::set_endpoint_response "$AGENTS2_BASE_URL/mouse/position" \
        '{"x":500,"y":300,"timestamp":"'$(_mock_current_timestamp)'"}'
}

#######################################
# Setup unhealthy Agent-S2 endpoints
#######################################
mock::agents2::setup_unhealthy_endpoints() {
    if ! command -v mock::http::set_endpoint_response &>/dev/null; then
        return 0
    fi
    
    # Health endpoint returns error
    mock::http::set_endpoint_response "$AGENTS2_BASE_URL/health" \
        '{"status":"error","error":"Browser engine unavailable","browser_ready":false,"display_available":false}' \
        "GET" \
        "503"
    
    # Task endpoint returns error
    mock::http::set_endpoint_response "$AGENTS2_BASE_URL/api/v1/tasks" \
        '{"error":"Service temporarily unavailable"}' \
        "POST" \
        "503"
}

#######################################
# Setup installing Agent-S2 endpoints
#######################################
mock::agents2::setup_installing_endpoints() {
    if ! command -v mock::http::set_endpoint_response &>/dev/null; then
        return 0
    fi
    
    # Health endpoint returns installing status
    mock::http::set_endpoint_response "$AGENTS2_BASE_URL/health" \
        '{"status":"installing","progress":95,"current_step":"Starting browser engine","browser_ready":false}'
    
    # Other endpoints return not ready
    mock::http::set_endpoint_response "$AGENTS2_BASE_URL/api/v1/tasks" \
        '{"error":"Agent-S2 is still initializing"}' \
        "POST" \
        "503"
}

#######################################
# Setup stopped Agent-S2 endpoints
#######################################
mock::agents2::setup_stopped_endpoints() {
    if ! command -v mock::http::set_endpoint_unreachable &>/dev/null; then
        return 0
    fi
    
    # All endpoints fail to connect
    mock::http::set_endpoint_unreachable "$AGENTS2_BASE_URL"
}

# ----------------------------
# Agent-S2 Management Script Mock
# ----------------------------

#######################################
# Mock the manage.sh script behavior
#######################################
agents2::manage() {
    # Load state
    _agents2_load_state
    
    local action="${1:-install}"
    shift || true
    
    # Track call count
    local count="${MOCK_AGENTS2_CALL_COUNT[$action]:-0}"
    MOCK_AGENTS2_CALL_COUNT["$action"]=$((count + 1))
    _agents2_mock_save_state
    
    # Check for injected errors
    if [[ -n "${MOCK_AGENTS2_ERRORS[$action]:-}" ]]; then
        local error_type="${MOCK_AGENTS2_ERRORS[$action]}"
        case "$error_type" in
            connection) echo "Error: Failed to connect to Agent-S2 server" >&2; return 1 ;;
            docker) echo "Error: Docker is not running" >&2; return 1 ;;
            permission) echo "Error: Permission denied" >&2; return 1 ;;
            *) echo "Error: $error_type" >&2; return 1 ;;
        esac
    fi
    
    # Handle global mock modes
    case "$AGENTS2_MOCK_MODE" in
        offline|stopped)
            echo "Error: Agent-S2 is not running" >&2
            return 1
            ;;
        error)
            echo "Error: Agent-S2 encountered an error" >&2
            return 1
            ;;
    esac
    
    # Handle actions
    case "$action" in
        install)
            echo "Installing Agent-S2..."
            echo "Setting up browser automation environment..."
            echo "Configuring display server..."
            echo "Agent-S2 installed successfully"
            MOCK_AGENTS2_STATE["$AGENTS2_CONTAINER_NAME"]="stopped"
            _agents2_mock_save_state
            return 0
            ;;
            
        uninstall)
            echo "Removing Agent-S2..."
            echo "Cleaning up resources..."
            echo "Agent-S2 uninstalled successfully"
            # Remove the container from state array
            unset "MOCK_AGENTS2_STATE[$AGENTS2_CONTAINER_NAME]"
            _agents2_mock_save_state
            return 0
            ;;
            
        start)
            if [[ "${MOCK_AGENTS2_STATE[$AGENTS2_CONTAINER_NAME]:-}" == "running" ]]; then
                echo "Agent-S2 is already running"
                return 0
            fi
            echo "Starting Agent-S2..."
            echo "Starting browser engine..."
            echo "Agent-S2 started successfully"
            echo "API available at: $AGENTS2_BASE_URL"
            echo "VNC available at: vnc://localhost:$AGENTS2_VNC_PORT"
            MOCK_AGENTS2_STATE["$AGENTS2_CONTAINER_NAME"]="running"
            _agents2_mock_save_state
            return 0
            ;;
            
        stop)
            if [[ "${MOCK_AGENTS2_STATE[$AGENTS2_CONTAINER_NAME]:-}" != "running" ]]; then
                echo "Agent-S2 is not running"
                return 0
            fi
            echo "Stopping Agent-S2..."
            echo "Agent-S2 stopped successfully"
            MOCK_AGENTS2_STATE["$AGENTS2_CONTAINER_NAME"]="stopped"
            _agents2_mock_save_state
            return 0
            ;;
            
        restart)
            agents2::manage stop
            agents2::manage start
            return $?
            ;;
            
        status)
            local state="${MOCK_AGENTS2_STATE[$AGENTS2_CONTAINER_NAME]:-not_installed}"
            echo "Agent-S2 Status: $state"
            if [[ "$state" == "running" ]]; then
                echo "  API: $AGENTS2_BASE_URL"
                echo "  VNC: vnc://localhost:$AGENTS2_VNC_PORT"
                echo "  Mode: ${MOCK_AGENTS2_CONFIG[mode]:-sandbox}"
                echo "  LLM Provider: ${MOCK_AGENTS2_CONFIG[llm_provider]:-ollama}"
                echo "  AI Enabled: ${MOCK_AGENTS2_CONFIG[enable_ai]:-yes}"
                echo "  Stealth Mode: ${MOCK_AGENTS2_STEALTH[enabled]:-no}"
            fi
            return 0
            ;;
            
        logs)
            echo "[2024-01-15 10:00:00] Agent-S2 started"
            echo "[2024-01-15 10:00:01] Browser engine initialized"
            echo "[2024-01-15 10:00:02] Display server ready"
            echo "[2024-01-15 10:00:03] API server listening on port $AGENTS2_PORT"
            return 0
            ;;
            
        mode)
            echo "Current mode: ${MOCK_AGENTS2_CONFIG[mode]:-sandbox}"
            return 0
            ;;
            
        switch-mode)
            local target_mode="${1:-sandbox}"
            echo "Switching to $target_mode mode..."
            MOCK_AGENTS2_CONFIG["mode"]="$target_mode"
            _agents2_mock_save_state
            echo "Switched to $target_mode mode successfully"
            return 0
            ;;
            
        ai-task)
            local task="$*"
            # Trim whitespace and check if empty
            task=$(echo "$task" | xargs)
            if [[ -z "$task" ]]; then
                echo "Error: Task description is required" >&2
                return 1
            fi
            local task_id=$(mock::agents2::create_task "ai_task" "$task")
            echo "Executing AI task: $task"
            echo "Task ID: $task_id"
            echo "Task accepted and running..."
            sleep 1
            echo "Task completed successfully"
            mock::agents2::update_task "$task_id" "completed" "success"
            return 0
            ;;
            
        configure-stealth)
            local enabled="${1:-yes}"
            echo "Configuring stealth mode: $enabled"
            mock::agents2::set_stealth "enabled" "$enabled"
            echo "Stealth mode configured successfully"
            return 0
            ;;
            
        test-stealth)
            local url="${1:-https://bot.sannysoft.com/}"
            echo "Testing stealth mode at: $url"
            if [[ "${MOCK_AGENTS2_STEALTH[enabled]}" == "yes" ]]; then
                echo "Stealth test: PASSED"
                echo "  WebDriver: Hidden"
                echo "  Navigator: Spoofed"
                echo "  Canvas: Randomized"
            else
                echo "Stealth test: FAILED"
                echo "  WebDriver: Detected"
            fi
            return 0
            ;;
            
        list-sessions)
            echo "Active sessions:"
            for session_id in "${!MOCK_AGENTS2_SESSIONS[@]}"; do
                echo "  $session_id: ${MOCK_AGENTS2_SESSIONS[$session_id]}"
            done
            return 0
            ;;
            
        *)
            echo "Unknown action: $action" >&2
            return 1
            ;;
    esac
}

# ----------------------------
# Test Scenario Functions
# ----------------------------

#######################################
# Scenario: Setup healthy Agent-S2 with active session
#######################################
mock::agents2::scenario::healthy_with_session() {
    mock::agents2::reset
    mock::agents2::set_mode "healthy"
    
    # Create an active session
    local session_id=$(mock::agents2::create_session "test-profile")
    
    # Create some completed tasks
    local task1=$(mock::agents2::create_task "screenshot" "https://example.com")
    mock::agents2::update_task "$task1" "completed" "success"
    
    local task2=$(mock::agents2::create_task "automation" "click_button")
    mock::agents2::update_task "$task2" "completed" "success"
    
    echo "[MOCK] Created healthy Agent-S2 scenario with session: $session_id"
}

#######################################
# Scenario: Setup Agent-S2 in installing state
#######################################
mock::agents2::scenario::installing() {
    mock::agents2::reset
    if ! mock::agents2::set_mode "installing"; then
        echo "[MOCK] ERROR: Failed to set installing mode" >&2
        return 1
    fi
    
    echo "[MOCK] Created installing Agent-S2 scenario"
}

#######################################
# Scenario: Setup offline Agent-S2
#######################################
mock::agents2::scenario::offline() {
    mock::agents2::reset
    if ! mock::agents2::set_mode "offline"; then
        echo "[MOCK] ERROR: Failed to set offline mode" >&2
        return 1
    fi
    
    echo "[MOCK] Created offline Agent-S2 scenario"
}

#######################################
# Scenario: Setup Agent-S2 with stealth mode
#######################################
mock::agents2::scenario::stealth_enabled() {
    mock::agents2::reset
    mock::agents2::set_mode "healthy"
    
    # Enable all stealth features
    mock::agents2::set_stealth "enabled" "yes"
    mock::agents2::set_stealth "webdriver_override" "yes"
    mock::agents2::set_stealth "navigator_spoofing" "yes"
    mock::agents2::set_stealth "canvas_fingerprint" "yes"
    
    echo "[MOCK] Created Agent-S2 scenario with stealth mode enabled"
}

# ----------------------------
# Assertion Helper Functions
# ----------------------------

#######################################
# Assert that Agent-S2 is in a specific state
# Arguments: $1 - expected state
#######################################
mock::agents2::assert::state() {
    local expected_state="$1"
    
    # Load current state
    _agents2_load_state
    
    local actual_state="${MOCK_AGENTS2_STATE[$AGENTS2_CONTAINER_NAME]:-not_installed}"
    
    if [[ "$actual_state" != "$expected_state" ]]; then
        echo "ASSERTION FAILED: Agent-S2 state is '$actual_state', expected '$expected_state'" >&2
        return 1
    fi
    return 0
}

#######################################
# Assert that a task exists with specific status
# Arguments: $1 - task id, $2 - expected status
#######################################
mock::agents2::assert::task_status() {
    local task_id="$1"
    local expected_status="$2"
    
    # Load current state
    _agents2_load_state
    
    if [[ -z "${MOCK_AGENTS2_TASKS[$task_id]:-}" ]]; then
        echo "ASSERTION FAILED: Task '$task_id' does not exist" >&2
        return 1
    fi
    
    local actual_status=$(echo "${MOCK_AGENTS2_TASKS[$task_id]}" | cut -d'|' -f1)
    
    if [[ "$actual_status" != "$expected_status" ]]; then
        echo "ASSERTION FAILED: Task '$task_id' status is '$actual_status', expected '$expected_status'" >&2
        return 1
    fi
    return 0
}

#######################################
# Assert that an action was called specific number of times
# Arguments: $1 - action, $2 - expected count
#######################################
mock::agents2::assert::action_called() {
    local action="$1"
    local expected_count="${2:-1}"
    
    # Load state from file for subshell access
    _agents2_load_state
    
    local actual_count="${MOCK_AGENTS2_CALL_COUNT[$action]:-0}"
    
    if [[ "$actual_count" -lt "$expected_count" ]]; then
        echo "ASSERTION FAILED: Action '$action' called $actual_count times, expected at least $expected_count" >&2
        return 1
    fi
    return 0
}

# ----------------------------
# Get Information Functions
# ----------------------------

#######################################
# Get call count for an action
# Arguments: $1 - action name
#######################################
mock::agents2::get::call_count() {
    _agents2_load_state
    
    local action="$1"
    echo "${MOCK_AGENTS2_CALL_COUNT[$action]:-0}"
}

#######################################
# Get current mode
#######################################
mock::agents2::get::mode() {
    _agents2_load_state
    
    echo "$AGENTS2_MOCK_MODE"
}

#######################################
# Get configuration value
# Arguments: $1 - config key
#######################################
mock::agents2::get::config() {
    _agents2_load_state
    
    local key="$1"
    echo "${MOCK_AGENTS2_CONFIG[$key]:-}"
}

#######################################
# List active sessions
#######################################
mock::agents2::get::sessions() {
    _agents2_load_state
    
    local sessions=()
    for session_id in "${!MOCK_AGENTS2_SESSIONS[@]}"; do
        if [[ "${MOCK_AGENTS2_SESSIONS[$session_id]}" == "active" ]]; then
            sessions+=("$session_id")
        fi
    done
    
    printf '%s\n' "${sessions[@]}"
}

# ----------------------------
# Debug Functions
# ----------------------------

#######################################
# Dump all mock state for debugging
#######################################
mock::agents2::debug::dump_state() {
    # Load state from file for subshell access
    _agents2_load_state
    
    echo "=== Agent-S2 Mock State Dump ==="
    echo "Mode: $AGENTS2_MOCK_MODE"
    echo "Port: $AGENTS2_PORT"
    echo "Base URL: $AGENTS2_BASE_URL"
    echo "Container: $AGENTS2_CONTAINER_NAME"
    echo
    echo "Container State:"
    for container in "${!MOCK_AGENTS2_STATE[@]}"; do
        echo "  $container: ${MOCK_AGENTS2_STATE[$container]}"
    done
    echo
    echo "Sessions:"
    for session in "${!MOCK_AGENTS2_SESSIONS[@]}"; do
        echo "  $session: ${MOCK_AGENTS2_SESSIONS[$session]}"
    done
    echo
    echo "Tasks:"
    for task in "${!MOCK_AGENTS2_TASKS[@]}"; do
        echo "  $task: ${MOCK_AGENTS2_TASKS[$task]}"
    done
    echo
    echo "Configuration:"
    for key in "${!MOCK_AGENTS2_CONFIG[@]}"; do
        echo "  $key: ${MOCK_AGENTS2_CONFIG[$key]}"
    done
    echo
    echo "Stealth Features:"
    for feature in "${!MOCK_AGENTS2_STEALTH[@]}"; do
        echo "  $feature: ${MOCK_AGENTS2_STEALTH[$feature]}"
    done
    echo
    echo "Action Call Counts:"
    for action in "${!MOCK_AGENTS2_CALL_COUNT[@]}"; do
        echo "  $action: ${MOCK_AGENTS2_CALL_COUNT[$action]}"
    done
    echo
    echo "Injected Errors:"
    for action in "${!MOCK_AGENTS2_ERRORS[@]}"; do
        echo "  $action: ${MOCK_AGENTS2_ERRORS[$action]}"
    done
    echo "============================="
}

# ----------------------------
# Export functions for subshell availability
# ----------------------------

# Export utilities
export -f _mock_current_timestamp _mock_task_id _mock_session_id
export -f _agents2_mock_init_state_file _agents2_mock_save_state _agents2_load_state

# Export public API functions
export -f mock::agents2::reset mock::agents2::set_mode
export -f mock::agents2::create_task mock::agents2::update_task
export -f mock::agents2::create_session
export -f mock::agents2::set_config mock::agents2::set_stealth
export -f mock::agents2::inject_error

# Export HTTP endpoint setup functions
export -f mock::agents2::setup_healthy_endpoints mock::agents2::setup_unhealthy_endpoints
export -f mock::agents2::setup_installing_endpoints mock::agents2::setup_stopped_endpoints

# Export management function
export -f agents2::manage

# Export scenarios
export -f mock::agents2::scenario::healthy_with_session mock::agents2::scenario::installing
export -f mock::agents2::scenario::offline mock::agents2::scenario::stealth_enabled

# Export assertions
export -f mock::agents2::assert::state mock::agents2::assert::task_status
export -f mock::agents2::assert::action_called

# Export getters
export -f mock::agents2::get::call_count mock::agents2::get::mode
export -f mock::agents2::get::config mock::agents2::get::sessions

# Export debug functions
export -f mock::agents2::debug::dump_state

# Load state if state file exists and has content, otherwise use defaults
if [[ -f "$AGENTS2_MOCK_STATE_FILE" ]] && [[ -s "$AGENTS2_MOCK_STATE_FILE" ]] && grep -q "MOCK_AGENTS2_STATE\[" "$AGENTS2_MOCK_STATE_FILE" 2>/dev/null; then
    # State file exists with actual state, load it
    eval "$(cat "$AGENTS2_MOCK_STATE_FILE")" 2>/dev/null || true
fi

echo "[MOCK] Agent-S2 mock implementation loaded successfully"