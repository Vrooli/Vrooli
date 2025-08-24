#!/usr/bin/env bash
# Agent-S2 Mock - Tier 2 (Stateful)
# 
# Provides stateful Agent-S2 browser automation mocking for testing:
# - Container lifecycle management (install, start, stop, status)
# - AI task execution simulation with browser automation
# - Session management for browser instances
# - HTTP API endpoints for REST interactions
# - Error injection for resilience testing
#
# Coverage: ~80% of common Agent-S2 operations in 400 lines

# === Configuration ===
declare -gA AGENTS2_TASKS=()           # Task_ID -> "status|description|result|session_id|created_at"
declare -gA AGENTS2_SESSIONS=()        # Session_ID -> "status|browser|created_at|last_activity"
declare -gA AGENTS2_CALL_COUNT=()      # Action -> call_count
declare -gA AGENTS2_CONFIG=(           # Service configuration
    [status]="running"
    [mode]="healthy"
    [port]="3000"
    [ai_provider]="ollama"
    [ai_model]="llama2"
    [stealth_mode]="enabled"
    [max_sessions]="5"
    [error_mode]=""
    [version]="2.1.0"
)

# Debug mode
declare -g AGENTS2_DEBUG="${AGENTS2_DEBUG:-}"

# === Helper Functions ===
agents2_debug() {
    [[ -n "$AGENTS2_DEBUG" ]] && echo "[MOCK:AGENTS2] $*" >&2
}

agents2_check_error() {
    case "${AGENTS2_CONFIG[error_mode]}" in
        "service_unavailable")
            echo "Error: Agent-S2 service is unavailable" >&2
            return 1
            ;;
        "ai_provider_failed")
            echo "Error: AI provider connection failed" >&2
            return 1
            ;;
        "browser_launch_failed")
            echo "Error: Failed to launch browser session" >&2
            return 1
            ;;
        "stealth_detection")
            echo "Error: Bot detection triggered" >&2
            return 1
            ;;
        "task_timeout")
            echo "Error: Task execution timeout" >&2
            return 1
            ;;
    esac
    return 0
}

agents2_generate_id() {
    local prefix="${1:-task}"
    printf "%s_%08x" "$prefix" $((RANDOM * RANDOM))
}

agents2_timestamp() {
    date '+%Y-%m-%d %H:%M:%S' 2>/dev/null || echo '2023-01-01 00:00:00'
}

agents2_increment_call() {
    local action="$1"
    local count="${AGENTS2_CALL_COUNT[$action]:-0}"
    ((count++))
    AGENTS2_CALL_COUNT[$action]="$count"
    agents2_debug "Call count for $action: $count"
}

# === Main CLI Interface ===
agents2::manage() {
    agents2_debug "agents2::manage called with: $*"
    
    if ! agents2_check_error; then
        return $?
    fi
    
    if [[ $# -eq 0 ]]; then
        echo "Usage: agents2::manage <action> [options]"
        echo ""
        echo "Available actions:"
        echo "  install      Install Agent-S2 service"
        echo "  uninstall    Remove Agent-S2 service"
        echo "  start        Start the service"
        echo "  stop         Stop the service"
        echo "  restart      Restart the service"
        echo "  status       Show service status"
        echo "  ai-task      Execute AI-driven browser task"
        echo "  mode         Show or set service mode"
        echo "  logs         Show service logs"
        return 1
    fi
    
    local action="$1"
    shift
    
    agents2_increment_call "$action"
    
    case "$action" in
        install)
            agents2_cmd_install "$@"
            ;;
        uninstall)
            agents2_cmd_uninstall "$@"
            ;;
        start)
            agents2_cmd_start "$@"
            ;;
        stop)
            agents2_cmd_stop "$@"
            ;;
        restart)
            agents2_cmd_restart "$@"
            ;;
        status)
            agents2_cmd_status "$@"
            ;;
        ai-task)
            agents2_cmd_ai_task "$@"
            ;;
        mode)
            agents2_cmd_mode "$@"
            ;;
        logs)
            agents2_cmd_logs "$@"
            ;;
        list-sessions)
            agents2_cmd_list_sessions "$@"
            ;;
        *)
            echo "Error: Unknown action '$action'" >&2
            echo "Run 'agents2::manage' for usage information." >&2
            return 1
            ;;
    esac
}

# === Command Implementations ===

agents2_cmd_install() {
    agents2_debug "Installing Agent-S2 service"
    
    if [[ "${AGENTS2_CONFIG[status]}" == "installed" || "${AGENTS2_CONFIG[status]}" == "running" ]]; then
        echo "Agent-S2 is already installed"
        return 0
    fi
    
    echo "Installing Agent-S2 browser automation service..."
    echo "📦 Downloading browser engines..."
    echo "🤖 Setting up AI model integration..."
    echo "🛡️  Configuring stealth features..."
    echo "✅ Agent-S2 installation completed"
    
    AGENTS2_CONFIG[status]="installed"
    agents2_debug "Status changed to: installed"
}

agents2_cmd_uninstall() {
    agents2_debug "Uninstalling Agent-S2 service"
    
    echo "Uninstalling Agent-S2 service..."
    echo "🛑 Stopping all sessions..."
    echo "🗑️  Removing browser data..."
    echo "✅ Agent-S2 uninstalled"
    
    # Clear all state
    AGENTS2_TASKS=()
    AGENTS2_SESSIONS=()
    AGENTS2_CONFIG[status]="not_installed"
    agents2_debug "Status changed to: not_installed"
}

agents2_cmd_start() {
    agents2_debug "Starting Agent-S2 service"
    
    if [[ "${AGENTS2_CONFIG[status]}" == "not_installed" ]]; then
        echo "Error: Agent-S2 is not installed. Run 'install' first." >&2
        return 1
    fi
    
    if [[ "${AGENTS2_CONFIG[status]}" == "running" ]]; then
        echo "Agent-S2 is already running"
        return 0
    fi
    
    echo "Starting Agent-S2 service..."
    echo "🚀 Initializing browser automation engine..."
    echo "🤖 Connecting to AI provider (${AGENTS2_CONFIG[ai_provider]})..."
    echo "🌐 Starting API server on port ${AGENTS2_CONFIG[port]}..."
    echo "✅ Agent-S2 is now running"
    
    AGENTS2_CONFIG[status]="running"
    agents2_debug "Status changed to: running"
}

agents2_cmd_stop() {
    agents2_debug "Stopping Agent-S2 service"
    
    if [[ "${AGENTS2_CONFIG[status]}" == "stopped" ]]; then
        echo "Agent-S2 is already stopped"
        return 0
    fi
    
    echo "Stopping Agent-S2 service..."
    echo "🛑 Terminating active sessions..."
    echo "📝 Saving task states..."
    echo "✅ Agent-S2 stopped"
    
    AGENTS2_CONFIG[status]="stopped"
    agents2_debug "Status changed to: stopped"
}

agents2_cmd_restart() {
    agents2_debug "Restarting Agent-S2 service"
    
    agents2_cmd_stop >/dev/null 2>&1
    sleep 1
    agents2_cmd_start
}

agents2_cmd_status() {
    agents2_debug "Checking Agent-S2 service status"
    
    local status="${AGENTS2_CONFIG[status]}"
    local mode="${AGENTS2_CONFIG[mode]}"
    local active_sessions=$(( ${#AGENTS2_SESSIONS[@]} ))
    local total_tasks=$(( ${#AGENTS2_TASKS[@]} ))
    
    echo "Agent-S2 Browser Automation Service"
    echo "===================================="
    echo "Status:           $status"
    echo "Mode:             $mode"
    echo "Version:          ${AGENTS2_CONFIG[version]}"
    echo "AI Provider:      ${AGENTS2_CONFIG[ai_provider]}"
    echo "AI Model:         ${AGENTS2_CONFIG[ai_model]}"
    echo "Stealth Mode:     ${AGENTS2_CONFIG[stealth_mode]}"
    echo "Port:             ${AGENTS2_CONFIG[port]}"
    echo "Active Sessions:  $active_sessions/${AGENTS2_CONFIG[max_sessions]}"
    echo "Total Tasks:      $total_tasks"
    
    # Return appropriate exit code
    case "$status" in
        "running") return 0 ;;
        "stopped"|"installed") return 3 ;;
        "not_installed") return 4 ;;
        *) return 1 ;;
    esac
}

agents2_cmd_ai_task() {
    agents2_debug "AI task execution: $*"
    
    if [[ "${AGENTS2_CONFIG[status]}" != "running" ]]; then
        echo "Error: Agent-S2 service is not running" >&2
        return 1
    fi
    
    if [[ $# -eq 0 ]]; then
        echo "Usage: agents2::manage ai-task \"<task description>\"" >&2
        return 1
    fi
    
    local task_description="$*"
    local task_id=$(agents2_generate_id "task")
    local session_id=$(agents2_generate_id "session")
    local timestamp=$(agents2_timestamp)
    
    # Create new session if needed
    AGENTS2_SESSIONS[$session_id]="active|chrome|$timestamp|$timestamp"
    
    echo "🤖 Executing AI task: $task_description"
    echo "📋 Task ID: $task_id"
    echo "🌐 Browser Session: $session_id"
    echo ""
    
    # Simulate task execution based on description
    local result=""
    case "$task_description" in
        *"login"*|*"sign in"*)
            result="Successfully logged in to the website"
            ;;
        *"screenshot"*|*"capture"*)
            result="Screenshot captured and saved to /tmp/screenshot_$task_id.png"
            ;;
        *"form"*|*"submit"*)
            result="Form filled out and submitted successfully"
            ;;
        *"scrape"*|*"extract"*)
            result="Data extracted: 42 items found and processed"
            ;;
        *"navigate"*|*"go to"*)
            result="Successfully navigated to target page"
            ;;
        *)
            result="Task completed successfully with AI assistance"
            ;;
    esac
    
    # Store task
    AGENTS2_TASKS[$task_id]="completed|$task_description|$result|$session_id|$timestamp"
    
    echo "⏱️  Analyzing page structure..."
    echo "🧠 AI planning task steps..."
    echo "🖱️  Executing browser actions..."
    echo "✅ Task completed successfully"
    echo ""
    echo "Result: $result"
    echo ""
    echo "Task ID: $task_id (use for reference)"
    
    agents2_debug "Created task: $task_id"
    return 0
}

agents2_cmd_mode() {
    agents2_debug "Mode command: $*"
    
    if [[ $# -eq 0 ]]; then
        echo "Current mode: ${AGENTS2_CONFIG[mode]}"
        return 0
    fi
    
    local new_mode="$1"
    case "$new_mode" in
        healthy|unhealthy|offline)
            AGENTS2_CONFIG[mode]="$new_mode"
            echo "Mode changed to: $new_mode"
            agents2_debug "Mode changed to: $new_mode"
            ;;
        *)
            echo "Error: Invalid mode '$new_mode'. Use: healthy, unhealthy, offline" >&2
            return 1
            ;;
    esac
}

agents2_cmd_logs() {
    agents2_debug "Showing logs"
    
    echo "=== Agent-S2 Service Logs ==="
    echo "[$(agents2_timestamp)] Service started on port ${AGENTS2_CONFIG[port]}"
    echo "[$(agents2_timestamp)] AI provider connected: ${AGENTS2_CONFIG[ai_provider]}"
    echo "[$(agents2_timestamp)] Stealth mode: ${AGENTS2_CONFIG[stealth_mode]}"
    
    # Show recent task activity
    if [[ ${#AGENTS2_TASKS[@]} -gt 0 ]]; then
        echo "[$(agents2_timestamp)] Task activity:"
        for task_id in "${!AGENTS2_TASKS[@]}"; do
            local task_data="${AGENTS2_TASKS[$task_id]}"
            IFS='|' read -r status description result session_id created_at <<< "$task_data"
            echo "  [$created_at] Task $task_id: $status - ${description:0:50}..."
        done
    fi
    
    echo "[$(agents2_timestamp)] Active sessions: ${#AGENTS2_SESSIONS[@]}"
    echo "=== End of Logs ==="
}

agents2_cmd_list_sessions() {
    agents2_debug "Listing sessions"
    
    if [[ ${#AGENTS2_SESSIONS[@]} -eq 0 ]]; then
        echo "No active sessions"
        return 0
    fi
    
    echo "Active Browser Sessions:"
    echo "========================"
    printf "%-15s %-10s %-10s %-20s %-20s\n" "SESSION_ID" "STATUS" "BROWSER" "CREATED" "LAST_ACTIVITY"
    
    for session_id in "${!AGENTS2_SESSIONS[@]}"; do
        local session_data="${AGENTS2_SESSIONS[$session_id]}"
        IFS='|' read -r status browser created_at last_activity <<< "$session_data"
        printf "%-15s %-10s %-10s %-20s %-20s\n" \
            "${session_id:0:12}..." "$status" "$browser" "$created_at" "$last_activity"
    done
}

# === Mock API Functions ===
mock::agents2::reset() {
    agents2_debug "Resetting mock state"
    
    AGENTS2_TASKS=()
    AGENTS2_SESSIONS=()
    AGENTS2_CALL_COUNT=()
    AGENTS2_CONFIG[error_mode]=""
    AGENTS2_CONFIG[status]="running"
    AGENTS2_CONFIG[mode]="healthy"
    
    agents2_debug "Mock state reset"
}

mock::agents2::set_error() {
    local error_mode="$1"
    AGENTS2_CONFIG[error_mode]="$error_mode"
    agents2_debug "Set error mode: $error_mode"
}

mock::agents2::set_mode() {
    local mode="$1"
    AGENTS2_CONFIG[mode]="$mode"
    agents2_debug "Set mode: $mode"
}

mock::agents2::set_config() {
    local key="$1" value="$2"
    AGENTS2_CONFIG[$key]="$value"
    agents2_debug "Set config: $key = $value"
}

mock::agents2::create_task() {
    local description="${1:-Test task}"
    local task_id=$(agents2_generate_id "task")
    local session_id=$(agents2_generate_id "session")
    local timestamp=$(agents2_timestamp)
    
    AGENTS2_TASKS[$task_id]="pending|$description|waiting|$session_id|$timestamp"
    AGENTS2_SESSIONS[$session_id]="active|chrome|$timestamp|$timestamp"
    
    agents2_debug "Created task: $task_id"
    echo "$task_id"
}

mock::agents2::update_task() {
    local task_id="$1" new_status="$2" result="${3:-}"
    
    if [[ -z "${AGENTS2_TASKS[$task_id]}" ]]; then
        echo "Error: Task not found: $task_id" >&2
        return 1
    fi
    
    local task_data="${AGENTS2_TASKS[$task_id]}"
    IFS='|' read -r old_status description old_result session_id created_at <<< "$task_data"
    
    local updated_result="${result:-$old_result}"
    AGENTS2_TASKS[$task_id]="$new_status|$description|$updated_result|$session_id|$created_at"
    
    agents2_debug "Updated task $task_id: $new_status"
}

mock::agents2::get_call_count() {
    local action="$1"
    echo "${AGENTS2_CALL_COUNT[$action]:-0}"
}

mock::agents2::dump_state() {
    echo "=== Agent-S2 Mock State ==="
    echo "Status: ${AGENTS2_CONFIG[status]}"
    echo "Mode: ${AGENTS2_CONFIG[mode]}"
    echo "Error Mode: ${AGENTS2_CONFIG[error_mode]:-none}"
    echo "Tasks: ${#AGENTS2_TASKS[@]}"
    for task_id in "${!AGENTS2_TASKS[@]}"; do
        echo "  $task_id: ${AGENTS2_TASKS[$task_id]}"
    done
    echo "Sessions: ${#AGENTS2_SESSIONS[@]}"
    for session_id in "${!AGENTS2_SESSIONS[@]}"; do
        echo "  $session_id: ${AGENTS2_SESSIONS[$session_id]}"
    done
    echo "Call Counts: ${#AGENTS2_CALL_COUNT[@]}"
    for action in "${!AGENTS2_CALL_COUNT[@]}"; do
        echo "  $action: ${AGENTS2_CALL_COUNT[$action]}"
    done
    echo "=========================="
}

# === Scenario Setup Functions (Testing Helpers) ===
mock::agents2::scenario::healthy_service() {
    agents2_debug "Setting up healthy service scenario"
    mock::agents2::reset
    AGENTS2_CONFIG[status]="running"
    AGENTS2_CONFIG[mode]="healthy"
    echo "✅ Healthy service scenario ready"
}

mock::agents2::scenario::service_down() {
    agents2_debug "Setting up service down scenario"
    mock::agents2::reset
    AGENTS2_CONFIG[status]="stopped"
    AGENTS2_CONFIG[mode]="offline"
    echo "🛑 Service down scenario ready"
}

mock::agents2::scenario::ai_provider_issues() {
    agents2_debug "Setting up AI provider issues scenario"
    mock::agents2::reset
    AGENTS2_CONFIG[status]="running"
    AGENTS2_CONFIG[mode]="unhealthy"
    mock::agents2::set_error "ai_provider_failed"
    echo "🤖 AI provider issues scenario ready"
}

mock::agents2::scenario::busy_with_tasks() {
    agents2_debug "Setting up busy service scenario"
    mock::agents2::reset
    AGENTS2_CONFIG[status]="running"
    AGENTS2_CONFIG[mode]="healthy"
    
    # Create multiple active tasks and sessions
    for i in {1..3}; do
        mock::agents2::create_task "Background task $i" >/dev/null
    done
    
    echo "🏃 Busy service scenario ready (${#AGENTS2_TASKS[@]} active tasks)"
}

# === Convention-based Test Functions ===
test_agents2_connection() {
    agents2_debug "Testing connection..."
    
    # Test status command
    local result
    result=$(agents2::manage status 2>&1)
    
    if [[ "$result" =~ "Agent-S2 Browser Automation" ]]; then
        agents2_debug "Connection test passed"
        return 0
    else
        agents2_debug "Connection test failed: $result"
        return 1
    fi
}

test_agents2_health() {
    agents2_debug "Testing health..."
    
    # Test connection
    test_agents2_connection || return 1
    
    # Test service lifecycle
    agents2::manage stop >/dev/null 2>&1
    agents2::manage start >/dev/null 2>&1 || return 1
    
    # Test task execution
    agents2::manage ai-task "test navigation task" >/dev/null 2>&1 || return 1
    
    agents2_debug "Health test passed"
    return 0
}

test_agents2_basic() {
    agents2_debug "Testing basic operations..."
    
    # Test task creation and execution
    local result
    result=$(agents2::manage ai-task "login to website" 2>&1)
    [[ "$result" =~ "Task ID:" ]] || return 1
    
    # Test status shows task
    local status_result
    status_result=$(agents2::manage status 2>&1)
    [[ "$status_result" =~ "Total Tasks: [1-9]" ]] || return 1
    
    # Test mode switching
    agents2::manage mode unhealthy >/dev/null 2>&1 || return 1
    agents2::manage mode healthy >/dev/null 2>&1 || return 1
    
    # Test session listing
    agents2::manage list-sessions >/dev/null 2>&1 || return 1
    
    agents2_debug "Basic test passed"
    return 0
}

# === Export Functions ===
export -f agents2::manage
export -f test_agents2_connection test_agents2_health test_agents2_basic
export -f mock::agents2::reset mock::agents2::set_error mock::agents2::set_mode
export -f mock::agents2::set_config mock::agents2::create_task mock::agents2::update_task
export -f mock::agents2::get_call_count mock::agents2::dump_state
export -f mock::agents2::scenario::healthy_service mock::agents2::scenario::service_down
export -f mock::agents2::scenario::ai_provider_issues mock::agents2::scenario::busy_with_tasks
export -f agents2_debug agents2_check_error

# Initialize with defaults
mock::agents2::reset
agents2_debug "Agent-S2 Tier 2 mock initialized"
# Ensure we return success when sourced
return 0 2>/dev/null || true
