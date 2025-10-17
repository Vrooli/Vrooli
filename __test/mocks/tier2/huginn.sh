#!/usr/bin/env bash
# Huginn Mock - Tier 2 (Stateful)
# 
# Provides stateful Huginn automation platform mocking for testing:
# - Agent creation and management
# - Event processing
# - Scenario execution
# - Webhook handling
# - Error injection for resilience testing
#
# Coverage: ~80% of common Huginn operations in 400 lines

# === Configuration ===
declare -gA HUGINN_AGENTS=()              # Agent_id -> "type|schedule|status"
declare -gA HUGINN_EVENTS=()              # Event_id -> "agent|payload|timestamp"
declare -gA HUGINN_SCENARIOS=()           # Scenario_name -> "agents|enabled"
declare -gA HUGINN_WEBHOOKS=()            # Webhook_id -> "url|method|headers"
declare -gA HUGINN_CONFIG=(               # Service configuration
    [status]="running"
    [port]="3000"
    [workers]="2"
    [error_mode]=""
    [version]="2.0.0"
)

# Debug mode
declare -g HUGINN_DEBUG="${HUGINN_DEBUG:-}"

# === Helper Functions ===
huginn_debug() {
    [[ -n "$HUGINN_DEBUG" ]] && echo "[MOCK:HUGINN] $*" >&2
}

huginn_check_error() {
    case "${HUGINN_CONFIG[error_mode]}" in
        "service_down")
            echo "Error: Huginn service is not running" >&2
            return 1
            ;;
        "agent_error")
            echo "Error: Agent execution failed" >&2
            return 1
            ;;
        "event_overflow")
            echo "Error: Event queue overflow" >&2
            return 1
            ;;
    esac
    return 0
}

huginn_generate_id() {
    printf "id_%08x" $RANDOM
}

# === Main Huginn Command ===
huginn() {
    huginn_debug "huginn called with: $*"
    
    if ! huginn_check_error; then
        return $?
    fi
    
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        agent)
            huginn_cmd_agent "$@"
            ;;
        event)
            huginn_cmd_event "$@"
            ;;
        scenario)
            huginn_cmd_scenario "$@"
            ;;
        webhook)
            huginn_cmd_webhook "$@"
            ;;
        run)
            huginn_cmd_run "$@"
            ;;
        status)
            huginn_cmd_status "$@"
            ;;
        start|stop|restart)
            huginn_cmd_service "$command" "$@"
            ;;
        *)
            echo "Huginn CLI - Build Agents that Monitor and Act"
            echo "Commands:"
            echo "  agent    - Manage agents"
            echo "  event    - Manage events"
            echo "  scenario - Manage scenarios"
            echo "  webhook  - Manage webhooks"
            echo "  run      - Run agent or scenario"
            echo "  status   - Show service status"
            echo "  start    - Start service"
            echo "  stop     - Stop service"
            echo "  restart  - Restart service"
            ;;
    esac
}

# === Agent Management ===
huginn_cmd_agent() {
    local action="${1:-list}"
    shift || true
    
    case "$action" in
        list)
            echo "Agents:"
            if [[ ${#HUGINN_AGENTS[@]} -eq 0 ]]; then
                echo "  (none)"
            else
                for agent_id in "${!HUGINN_AGENTS[@]}"; do
                    local data="${HUGINN_AGENTS[$agent_id]}"
                    IFS='|' read -r type schedule status <<< "$data"
                    echo "  $agent_id - Type: $type, Schedule: $schedule, Status: $status"
                done
            fi
            ;;
        create)
            local type="${1:-WebsiteAgent}"
            local schedule="${2:-every_1h}"
            local agent_id=$(huginn_generate_id)
            
            HUGINN_AGENTS[$agent_id]="$type|$schedule|active"
            huginn_debug "Created agent: $agent_id"
            echo "Agent created: $agent_id ($type)"
            ;;
        delete)
            local agent_id="${1:-}"
            [[ -z "$agent_id" ]] && { echo "Error: agent ID required" >&2; return 1; }
            
            if [[ -n "${HUGINN_AGENTS[$agent_id]}" ]]; then
                unset HUGINN_AGENTS[$agent_id]
                echo "Agent deleted: $agent_id"
            else
                echo "Error: agent not found: $agent_id" >&2
                return 1
            fi
            ;;
        *)
            echo "Usage: huginn agent {list|create|delete} [args]"
            return 1
            ;;
    esac
}

# === Event Management ===
huginn_cmd_event() {
    local action="${1:-list}"
    shift || true
    
    case "$action" in
        list)
            echo "Events:"
            if [[ ${#HUGINN_EVENTS[@]} -eq 0 ]]; then
                echo "  (none)"
            else
                local count=0
                for event_id in "${!HUGINN_EVENTS[@]}"; do
                    local data="${HUGINN_EVENTS[$event_id]}"
                    IFS='|' read -r agent payload timestamp <<< "$data"
                    echo "  $event_id - Agent: $agent, Time: $timestamp"
                    ((count++))
                    [[ $count -ge 5 ]] && { echo "  ... (${#HUGINN_EVENTS[@]} total)"; break; }
                done
            fi
            ;;
        create)
            local agent="${1:-}"
            local payload="${2:-{}}"
            [[ -z "$agent" ]] && { echo "Error: agent ID required" >&2; return 1; }
            
            local event_id=$(huginn_generate_id)
            HUGINN_EVENTS[$event_id]="$agent|$payload|$(date +%s)"
            
            huginn_debug "Created event: $event_id"
            echo "Event created: $event_id"
            ;;
        clear)
            HUGINN_EVENTS=()
            echo "Events cleared"
            ;;
        *)
            echo "Usage: huginn event {list|create|clear} [args]"
            return 1
            ;;
    esac
}

# === Scenario Management ===
huginn_cmd_scenario() {
    local action="${1:-list}"
    shift || true
    
    case "$action" in
        list)
            echo "Scenarios:"
            if [[ ${#HUGINN_SCENARIOS[@]} -eq 0 ]]; then
                echo "  (none)"
            else
                for name in "${!HUGINN_SCENARIOS[@]}"; do
                    local data="${HUGINN_SCENARIOS[$name]}"
                    IFS='|' read -r agents enabled <<< "$data"
                    echo "  $name - Agents: $agents, Enabled: $enabled"
                done
            fi
            ;;
        create)
            local name="${1:-}"
            [[ -z "$name" ]] && { echo "Error: scenario name required" >&2; return 1; }
            
            HUGINN_SCENARIOS[$name]="0|true"
            echo "Scenario created: $name"
            ;;
        enable|disable)
            local name="${1:-}"
            [[ -z "$name" ]] && { echo "Error: scenario name required" >&2; return 1; }
            
            if [[ -n "${HUGINN_SCENARIOS[$name]}" ]]; then
                local data="${HUGINN_SCENARIOS[$name]}"
                IFS='|' read -r agents _ <<< "$data"
                local enabled="false"
                [[ "$action" == "enable" ]] && enabled="true"
                HUGINN_SCENARIOS[$name]="$agents|$enabled"
                echo "Scenario $action: $name"
            else
                echo "Error: scenario not found: $name" >&2
                return 1
            fi
            ;;
        *)
            echo "Usage: huginn scenario {list|create|enable|disable} [args]"
            return 1
            ;;
    esac
}

# === Webhook Management ===
huginn_cmd_webhook() {
    local action="${1:-list}"
    shift || true
    
    case "$action" in
        list)
            echo "Webhooks:"
            if [[ ${#HUGINN_WEBHOOKS[@]} -eq 0 ]]; then
                echo "  (none)"
            else
                for webhook_id in "${!HUGINN_WEBHOOKS[@]}"; do
                    local data="${HUGINN_WEBHOOKS[$webhook_id]}"
                    IFS='|' read -r url method headers <<< "$data"
                    echo "  $webhook_id - $method $url"
                done
            fi
            ;;
        create)
            local url="${1:-}"
            local method="${2:-POST}"
            [[ -z "$url" ]] && { echo "Error: webhook URL required" >&2; return 1; }
            
            local webhook_id=$(huginn_generate_id)
            HUGINN_WEBHOOKS[$webhook_id]="$url|$method|Content-Type: application/json"
            
            echo "Webhook created: $webhook_id"
            echo "Endpoint: /webhooks/$webhook_id"
            ;;
        delete)
            local webhook_id="${1:-}"
            [[ -z "$webhook_id" ]] && { echo "Error: webhook ID required" >&2; return 1; }
            
            if [[ -n "${HUGINN_WEBHOOKS[$webhook_id]}" ]]; then
                unset HUGINN_WEBHOOKS[$webhook_id]
                echo "Webhook deleted: $webhook_id"
            else
                echo "Error: webhook not found: $webhook_id" >&2
                return 1
            fi
            ;;
        *)
            echo "Usage: huginn webhook {list|create|delete} [args]"
            return 1
            ;;
    esac
}

# === Run Command ===
huginn_cmd_run() {
    local target="${1:-}"
    
    [[ -z "$target" ]] && { echo "Error: agent or scenario required" >&2; return 1; }
    
    if [[ -n "${HUGINN_AGENTS[$target]}" ]]; then
        echo "Running agent: $target"
        local event_id=$(huginn_generate_id)
        HUGINN_EVENTS[$event_id]="$target|{\"status\":\"success\"}|$(date +%s)"
        echo "Agent execution complete. Event created: $event_id"
    elif [[ -n "${HUGINN_SCENARIOS[$target]}" ]]; then
        echo "Running scenario: $target"
        echo "Scenario execution complete. 3 events created."
    else
        echo "Error: target not found: $target" >&2
        return 1
    fi
}

# === Status Command ===
huginn_cmd_status() {
    echo "Huginn Status"
    echo "============="
    echo "Service: ${HUGINN_CONFIG[status]}"
    echo "Port: ${HUGINN_CONFIG[port]}"
    echo "Workers: ${HUGINN_CONFIG[workers]}"
    echo "Version: ${HUGINN_CONFIG[version]}"
    echo ""
    echo "Agents: ${#HUGINN_AGENTS[@]}"
    echo "Events: ${#HUGINN_EVENTS[@]}"
    echo "Scenarios: ${#HUGINN_SCENARIOS[@]}"
    echo "Webhooks: ${#HUGINN_WEBHOOKS[@]}"
}

# === Service Management ===
huginn_cmd_service() {
    local action="$1"
    
    case "$action" in
        start)
            if [[ "${HUGINN_CONFIG[status]}" == "running" ]]; then
                echo "Huginn is already running"
            else
                HUGINN_CONFIG[status]="running"
                echo "Huginn started on port ${HUGINN_CONFIG[port]}"
            fi
            ;;
        stop)
            HUGINN_CONFIG[status]="stopped"
            echo "Huginn stopped"
            ;;
        restart)
            HUGINN_CONFIG[status]="stopped"
            HUGINN_CONFIG[status]="running"
            echo "Huginn restarted"
            ;;
    esac
}

# === Mock Control Functions ===
huginn_mock_reset() {
    huginn_debug "Resetting mock state"
    
    HUGINN_AGENTS=()
    HUGINN_EVENTS=()
    HUGINN_SCENARIOS=()
    HUGINN_WEBHOOKS=()
    HUGINN_CONFIG[error_mode]=""
    HUGINN_CONFIG[status]="running"
}

huginn_mock_set_error() {
    HUGINN_CONFIG[error_mode]="$1"
    huginn_debug "Set error mode: $1"
}

huginn_mock_dump_state() {
    echo "=== Huginn Mock State ==="
    echo "Status: ${HUGINN_CONFIG[status]}"
    echo "Port: ${HUGINN_CONFIG[port]}"
    echo "Agents: ${#HUGINN_AGENTS[@]}"
    echo "Events: ${#HUGINN_EVENTS[@]}"
    echo "Scenarios: ${#HUGINN_SCENARIOS[@]}"
    echo "Webhooks: ${#HUGINN_WEBHOOKS[@]}"
    echo "Error Mode: ${HUGINN_CONFIG[error_mode]:-none}"
    echo "====================="
}

# === Convention-based Test Functions ===
test_huginn_connection() {
    huginn_debug "Testing connection..."
    
    if [[ "${HUGINN_CONFIG[status]}" == "running" ]]; then
        huginn_debug "Connection test passed"
        return 0
    else
        huginn_debug "Connection test failed"
        return 1
    fi
}

test_huginn_health() {
    huginn_debug "Testing health..."
    
    test_huginn_connection || return 1
    
    huginn agent list >/dev/null 2>&1 || return 1
    huginn scenario list >/dev/null 2>&1 || return 1
    
    huginn_debug "Health test passed"
    return 0
}

test_huginn_basic() {
    huginn_debug "Testing basic operations..."
    
    huginn agent create WebsiteAgent every_1h >/dev/null 2>&1 || return 1
    huginn scenario create test-scenario >/dev/null 2>&1 || return 1
    huginn webhook create https://example.com/hook POST >/dev/null 2>&1 || return 1
    
    huginn_debug "Basic test passed"
    return 0
}

# === Export Functions ===
export -f huginn
export -f test_huginn_connection test_huginn_health test_huginn_basic
export -f huginn_mock_reset huginn_mock_set_error huginn_mock_dump_state
export -f huginn_debug huginn_check_error

# Initialize
huginn_mock_reset
huginn_debug "Huginn Tier 2 mock initialized"
# Ensure we return success when sourced
return 0 2>/dev/null || true
