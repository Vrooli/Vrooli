#!/usr/bin/env bash
# Windmill Mock - Tier 2 (Stateful)
# 
# Provides stateful Windmill workflow automation mocking for testing:
# - Workspace management (create, list, delete)
# - Script and flow execution
# - Worker management
# - API key management
# - Service lifecycle (start, stop, restart)
# - Error injection for resilience testing
#
# Coverage: ~80% of common Windmill operations in 500 lines

# === Configuration ===
declare -gA WINDMILL_WORKSPACES=()       # Workspace_name -> "status|owner|created"
declare -gA WINDMILL_SCRIPTS=()          # Script_id -> "workspace|name|language|status"
declare -gA WINDMILL_FLOWS=()            # Flow_id -> "workspace|name|status|runs"
declare -gA WINDMILL_JOBS=()             # Job_id -> "type|status|result"
declare -gA WINDMILL_API_KEYS=()         # Key_name -> "token|workspace|permissions"
declare -gA WINDMILL_CONFIG=(            # Service configuration
    [status]="running"
    [port]="8000"
    [workers]="3"
    [db_status]="running"
    [error_mode]=""
    [version]="1.290.0"
)

# Debug mode
declare -g WINDMILL_DEBUG="${WINDMILL_DEBUG:-}"

# === Helper Functions ===
windmill_debug() {
    [[ -n "$WINDMILL_DEBUG" ]] && echo "[MOCK:WINDMILL] $*" >&2
}

windmill_check_error() {
    case "${WINDMILL_CONFIG[error_mode]}" in
        "service_down")
            echo "Error: Windmill service is not running" >&2
            return 1
            ;;
        "db_error")
            echo "Error: Database connection failed" >&2
            return 1
            ;;
        "worker_busy")
            echo "Error: All workers are busy" >&2
            return 1
            ;;
        "auth_failed")
            echo "Error: Authentication failed" >&2
            return 1
            ;;
    esac
    return 0
}

windmill_generate_id() {
    printf "%08x" $RANDOM
}

# === Main Windmill Command ===
windmill() {
    windmill_debug "windmill called with: $*"
    
    if ! windmill_check_error; then
        return $?
    fi
    
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        workspace)
            windmill_cmd_workspace "$@"
            ;;
        script)
            windmill_cmd_script "$@"
            ;;
        flow)
            windmill_cmd_flow "$@"
            ;;
        run)
            windmill_cmd_run "$@"
            ;;
        worker)
            windmill_cmd_worker "$@"
            ;;
        apikey)
            windmill_cmd_apikey "$@"
            ;;
        status)
            windmill_cmd_status "$@"
            ;;
        start|stop|restart)
            windmill_cmd_service "$command" "$@"
            ;;
        *)
            echo "Windmill CLI - Workflow Automation Platform"
            echo "Commands:"
            echo "  workspace - Manage workspaces"
            echo "  script    - Manage scripts"
            echo "  flow      - Manage flows"
            echo "  run       - Execute scripts or flows"
            echo "  worker    - Manage workers"
            echo "  apikey    - Manage API keys"
            echo "  status    - Show service status"
            echo "  start     - Start service"
            echo "  stop      - Stop service"
            echo "  restart   - Restart service"
            ;;
    esac
}

# === Workspace Management ===
windmill_cmd_workspace() {
    local action="${1:-list}"
    shift || true
    
    case "$action" in
        list)
            echo "Workspaces:"
            if [[ ${#WINDMILL_WORKSPACES[@]} -eq 0 ]]; then
                echo "  (none)"
            else
                for name in "${!WINDMILL_WORKSPACES[@]}"; do
                    local data="${WINDMILL_WORKSPACES[$name]}"
                    IFS='|' read -r status owner created <<< "$data"
                    echo "  $name - Status: $status, Owner: $owner"
                done
            fi
            ;;
        create)
            local name="${1:-}"
            [[ -z "$name" ]] && { echo "Error: workspace name required" >&2; return 1; }
            
            WINDMILL_WORKSPACES[$name]="active|admin|$(date +%s)"
            windmill_debug "Created workspace: $name"
            echo "Workspace '$name' created"
            ;;
        delete)
            local name="${1:-}"
            [[ -z "$name" ]] && { echo "Error: workspace name required" >&2; return 1; }
            
            if [[ -n "${WINDMILL_WORKSPACES[$name]}" ]]; then
                unset WINDMILL_WORKSPACES[$name]
                echo "Workspace '$name' deleted"
            else
                echo "Error: workspace not found: $name" >&2
                return 1
            fi
            ;;
        *)
            echo "Usage: windmill workspace {list|create|delete} [args]"
            return 1
            ;;
    esac
}

# === Script Management ===
windmill_cmd_script() {
    local action="${1:-list}"
    shift || true
    
    case "$action" in
        list)
            echo "Scripts:"
            if [[ ${#WINDMILL_SCRIPTS[@]} -eq 0 ]]; then
                echo "  (none)"
            else
                for id in "${!WINDMILL_SCRIPTS[@]}"; do
                    local data="${WINDMILL_SCRIPTS[$id]}"
                    IFS='|' read -r workspace name language status <<< "$data"
                    echo "  $id - $name ($language) in $workspace [$status]"
                done
            fi
            ;;
        create)
            local name="${1:-}"
            local language="${2:-python}"
            local workspace="${3:-default}"
            [[ -z "$name" ]] && { echo "Error: script name required" >&2; return 1; }
            
            local script_id="script_$(windmill_generate_id)"
            WINDMILL_SCRIPTS[$script_id]="$workspace|$name|$language|ready"
            windmill_debug "Created script: $script_id"
            echo "Script '$name' created with ID: $script_id"
            ;;
        delete)
            local script_id="${1:-}"
            [[ -z "$script_id" ]] && { echo "Error: script ID required" >&2; return 1; }
            
            if [[ -n "${WINDMILL_SCRIPTS[$script_id]}" ]]; then
                unset WINDMILL_SCRIPTS[$script_id]
                echo "Script '$script_id' deleted"
            else
                echo "Error: script not found: $script_id" >&2
                return 1
            fi
            ;;
        *)
            echo "Usage: windmill script {list|create|delete} [args]"
            return 1
            ;;
    esac
}

# === Flow Management ===
windmill_cmd_flow() {
    local action="${1:-list}"
    shift || true
    
    case "$action" in
        list)
            echo "Flows:"
            if [[ ${#WINDMILL_FLOWS[@]} -eq 0 ]]; then
                echo "  (none)"
            else
                for id in "${!WINDMILL_FLOWS[@]}"; do
                    local data="${WINDMILL_FLOWS[$id]}"
                    IFS='|' read -r workspace name status runs <<< "$data"
                    echo "  $id - $name in $workspace [$status] (runs: $runs)"
                done
            fi
            ;;
        create)
            local name="${1:-}"
            local workspace="${2:-default}"
            [[ -z "$name" ]] && { echo "Error: flow name required" >&2; return 1; }
            
            local flow_id="flow_$(windmill_generate_id)"
            WINDMILL_FLOWS[$flow_id]="$workspace|$name|ready|0"
            windmill_debug "Created flow: $flow_id"
            echo "Flow '$name' created with ID: $flow_id"
            ;;
        delete)
            local flow_id="${1:-}"
            [[ -z "$flow_id" ]] && { echo "Error: flow ID required" >&2; return 1; }
            
            if [[ -n "${WINDMILL_FLOWS[$flow_id]}" ]]; then
                unset WINDMILL_FLOWS[$flow_id]
                echo "Flow '$flow_id' deleted"
            else
                echo "Error: flow not found: $flow_id" >&2
                return 1
            fi
            ;;
        *)
            echo "Usage: windmill flow {list|create|delete} [args]"
            return 1
            ;;
    esac
}

# === Run Scripts and Flows ===
windmill_cmd_run() {
    local type="${1:-}"
    local id="${2:-}"
    
    [[ -z "$type" ]] && { echo "Error: type (script/flow) required" >&2; return 1; }
    [[ -z "$id" ]] && { echo "Error: ID required" >&2; return 1; }
    
    local job_id="job_$(windmill_generate_id)"
    
    case "$type" in
        script)
            if [[ -z "${WINDMILL_SCRIPTS[$id]}" ]]; then
                echo "Error: script not found: $id" >&2
                return 1
            fi
            WINDMILL_JOBS[$job_id]="script|running|"
            echo "Executing script $id..."
            WINDMILL_JOBS[$job_id]="script|completed|success"
            echo "Job completed: $job_id"
            ;;
        flow)
            if [[ -z "${WINDMILL_FLOWS[$id]}" ]]; then
                echo "Error: flow not found: $id" >&2
                return 1
            fi
            local data="${WINDMILL_FLOWS[$id]}"
            IFS='|' read -r workspace name status runs <<< "$data"
            WINDMILL_FLOWS[$id]="$workspace|$name|$status|$((runs+1))"
            WINDMILL_JOBS[$job_id]="flow|running|"
            echo "Executing flow $id..."
            WINDMILL_JOBS[$job_id]="flow|completed|success"
            echo "Job completed: $job_id"
            ;;
        *)
            echo "Error: unknown type: $type (use script or flow)" >&2
            return 1
            ;;
    esac
}

# === Worker Management ===
windmill_cmd_worker() {
    local action="${1:-status}"
    
    case "$action" in
        status)
            echo "Workers: ${WINDMILL_CONFIG[workers]}"
            echo "Status: active"
            echo "Load: medium"
            ;;
        scale)
            local count="${2:-}"
            [[ -z "$count" ]] && { echo "Error: worker count required" >&2; return 1; }
            WINDMILL_CONFIG[workers]="$count"
            echo "Scaled workers to $count"
            ;;
        *)
            echo "Usage: windmill worker {status|scale} [args]"
            return 1
            ;;
    esac
}

# === API Key Management ===
windmill_cmd_apikey() {
    local action="${1:-list}"
    shift || true
    
    case "$action" in
        list)
            echo "API Keys:"
            if [[ ${#WINDMILL_API_KEYS[@]} -eq 0 ]]; then
                echo "  (none)"
            else
                for name in "${!WINDMILL_API_KEYS[@]}"; do
                    local data="${WINDMILL_API_KEYS[$name]}"
                    IFS='|' read -r token workspace permissions <<< "$data"
                    echo "  $name - Workspace: $workspace, Permissions: $permissions"
                done
            fi
            ;;
        create)
            local name="${1:-}"
            local workspace="${2:-default}"
            [[ -z "$name" ]] && { echo "Error: key name required" >&2; return 1; }
            
            local token="wm_$(windmill_generate_id)_$(windmill_generate_id)"
            WINDMILL_API_KEYS[$name]="$token|$workspace|read,write"
            windmill_debug "Created API key: $name"
            echo "API key created: $token"
            ;;
        delete)
            local name="${1:-}"
            [[ -z "$name" ]] && { echo "Error: key name required" >&2; return 1; }
            
            if [[ -n "${WINDMILL_API_KEYS[$name]}" ]]; then
                unset WINDMILL_API_KEYS[$name]
                echo "API key '$name' deleted"
            else
                echo "Error: key not found: $name" >&2
                return 1
            fi
            ;;
        *)
            echo "Usage: windmill apikey {list|create|delete} [args]"
            return 1
            ;;
    esac
}

# === Status Command ===
windmill_cmd_status() {
    echo "Windmill Status"
    echo "==============="
    echo "Service: ${WINDMILL_CONFIG[status]}"
    echo "Port: ${WINDMILL_CONFIG[port]}"
    echo "Database: ${WINDMILL_CONFIG[db_status]}"
    echo "Workers: ${WINDMILL_CONFIG[workers]}"
    echo "Version: ${WINDMILL_CONFIG[version]}"
    echo ""
    echo "Workspaces: ${#WINDMILL_WORKSPACES[@]}"
    echo "Scripts: ${#WINDMILL_SCRIPTS[@]}"
    echo "Flows: ${#WINDMILL_FLOWS[@]}"
    echo "Jobs: ${#WINDMILL_JOBS[@]}"
    echo "API Keys: ${#WINDMILL_API_KEYS[@]}"
}

# === Service Management ===
windmill_cmd_service() {
    local action="$1"
    
    case "$action" in
        start)
            if [[ "${WINDMILL_CONFIG[status]}" == "running" ]]; then
                echo "Windmill is already running"
            else
                WINDMILL_CONFIG[status]="running"
                WINDMILL_CONFIG[db_status]="running"
                echo "Windmill started on port ${WINDMILL_CONFIG[port]}"
            fi
            ;;
        stop)
            WINDMILL_CONFIG[status]="stopped"
            WINDMILL_CONFIG[db_status]="stopped"
            echo "Windmill stopped"
            ;;
        restart)
            WINDMILL_CONFIG[status]="stopped"
            WINDMILL_CONFIG[db_status]="stopped"
            WINDMILL_CONFIG[status]="running"
            WINDMILL_CONFIG[db_status]="running"
            echo "Windmill restarted"
            ;;
    esac
}

# === HTTP API Mock (via curl interceptor) ===
curl() {
    windmill_debug "curl called with: $*"
    
    local url="" method="GET" data=""
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -X) method="$2"; shift 2 ;;
            -d|--data) data="$2"; shift 2 ;;
            http*) url="$1"; shift ;;
            *) shift ;;
        esac
    done
    
    # Check if this is a Windmill API call
    if [[ "$url" =~ localhost:8000 || "$url" =~ 127.0.0.1:8000 ]]; then
        windmill_handle_api "$method" "$url" "$data"
        return $?
    fi
    
    echo "curl: Not a Windmill endpoint"
    return 0
}

windmill_handle_api() {
    local method="$1" url="$2" data="$3"
    
    case "$url" in
        */api/version)
            echo '{"version":"'${WINDMILL_CONFIG[version]}'"}'
            ;;
        */api/workspaces)
            echo '{"workspaces":['
            local first=true
            for ws in "${!WINDMILL_WORKSPACES[@]}"; do
                [[ "$first" != "true" ]] && echo ","
                echo '{"id":"'$ws'","name":"'$ws'"}'
                first=false
            done
            echo ']}'
            ;;
        */api/scripts)
            echo '{"scripts":['$(( ${#WINDMILL_SCRIPTS[@]} ))']}'
            ;;
        */api/flows)
            echo '{"flows":['$(( ${#WINDMILL_FLOWS[@]} ))']}'
            ;;
        */health)
            if [[ "${WINDMILL_CONFIG[status]}" == "running" ]]; then
                echo '{"status":"healthy","db":"connected","workers":'${WINDMILL_CONFIG[workers]}'}'
            else
                echo '{"status":"unhealthy","error":"Service not running"}'
            fi
            ;;
        *)
            echo '{"status":"ok","version":"'${WINDMILL_CONFIG[version]}'"}'
            ;;
    esac
}

# === Mock Control Functions ===
windmill_mock_reset() {
    windmill_debug "Resetting mock state"
    
    WINDMILL_WORKSPACES=()
    WINDMILL_SCRIPTS=()
    WINDMILL_FLOWS=()
    WINDMILL_JOBS=()
    WINDMILL_API_KEYS=()
    WINDMILL_CONFIG[error_mode]=""
    WINDMILL_CONFIG[status]="running"
    WINDMILL_CONFIG[db_status]="running"
    
    # Initialize defaults
    windmill_mock_init_defaults
}

windmill_mock_init_defaults() {
    # Default workspaces
    WINDMILL_WORKSPACES["demo"]="active|admin|$(date +%s)"
    WINDMILL_WORKSPACES["admin"]="active|admin|$(date +%s)"
    
    # Default script
    WINDMILL_SCRIPTS["script_default"]="demo|hello_world|python|ready"
}

windmill_mock_set_error() {
    WINDMILL_CONFIG[error_mode]="$1"
    windmill_debug "Set error mode: $1"
}

windmill_mock_dump_state() {
    echo "=== Windmill Mock State ==="
    echo "Status: ${WINDMILL_CONFIG[status]}"
    echo "Port: ${WINDMILL_CONFIG[port]}"
    echo "Database: ${WINDMILL_CONFIG[db_status]}"
    echo "Workers: ${WINDMILL_CONFIG[workers]}"
    echo "Workspaces: ${#WINDMILL_WORKSPACES[@]}"
    echo "Scripts: ${#WINDMILL_SCRIPTS[@]}"
    echo "Flows: ${#WINDMILL_FLOWS[@]}"
    echo "Jobs: ${#WINDMILL_JOBS[@]}"
    echo "API Keys: ${#WINDMILL_API_KEYS[@]}"
    echo "Error Mode: ${WINDMILL_CONFIG[error_mode]:-none}"
    echo "===================="
}

# === Convention-based Test Functions ===
test_windmill_connection() {
    windmill_debug "Testing connection..."
    
    local result
    result=$(curl -s http://localhost:8000/health 2>&1)
    
    if [[ "$result" =~ "status" ]]; then
        windmill_debug "Connection test passed"
        return 0
    else
        windmill_debug "Connection test failed"
        return 1
    fi
}

test_windmill_health() {
    windmill_debug "Testing health..."
    
    test_windmill_connection || return 1
    
    windmill workspace create test-health >/dev/null 2>&1 || return 1
    windmill workspace list | grep -q "test-health" || return 1
    windmill workspace delete test-health >/dev/null 2>&1 || return 1
    
    windmill_debug "Health test passed"
    return 0
}

test_windmill_basic() {
    windmill_debug "Testing basic operations..."
    
    windmill script create test-script python demo >/dev/null 2>&1 || return 1
    windmill flow create test-flow demo >/dev/null 2>&1 || return 1
    windmill apikey create test-key demo >/dev/null 2>&1 || return 1
    
    windmill_debug "Basic test passed"
    return 0
}

# === Export Functions ===
export -f windmill curl
export -f test_windmill_connection test_windmill_health test_windmill_basic
export -f windmill_mock_reset windmill_mock_set_error windmill_mock_dump_state
export -f windmill_debug windmill_check_error

# Initialize with defaults
windmill_mock_reset
windmill_debug "Windmill Tier 2 mock initialized"
# Ensure we return success when sourced
return 0 2>/dev/null || true
