#!/usr/bin/env bash
# Huginn Resource Mock Implementation
# Provides comprehensive mocking for Huginn agent automation platform

# Prevent duplicate loading
if [[ "${HUGINN_MOCK_LOADED:-}" == "true" ]]; then
    return 0
fi
export HUGINN_MOCK_LOADED="true"

# ----------------------------
# Mock Configuration & State Management
# ----------------------------

# Default configuration
readonly HUGINN_DEFAULT_PORT="3000"
readonly HUGINN_DEFAULT_BASE_URL="http://localhost:3000"
readonly HUGINN_DEFAULT_DB_PORT="5432"
readonly HUGINN_DEFAULT_CONTAINER_NAME="huginn_app"
readonly HUGINN_DEFAULT_DB_CONTAINER_NAME="huginn_db"

# Initialize mock state file for subshell access
if [[ -z "${HUGINN_MOCK_STATE_FILE:-}" ]]; then
    if [[ -n "${BATS_TMPDIR:-}" ]]; then
        HUGINN_MOCK_STATE_FILE="${BATS_TMPDIR}/huginn_mock_state_test"
    else
        HUGINN_MOCK_STATE_FILE="${TMPDIR:-/tmp}/huginn_mock_state_$$"
    fi
fi
export HUGINN_MOCK_STATE_FILE

# Global state arrays
declare -gA MOCK_HUGINN_AGENTS=()
declare -gA MOCK_HUGINN_SCENARIOS=()
declare -gA MOCK_HUGINN_EVENTS=()
declare -gA MOCK_HUGINN_USERS=()
declare -gA MOCK_HUGINN_CALL_COUNT=()
declare -gA MOCK_HUGINN_ERRORS=()
declare -gA MOCK_HUGINN_AGENT_STATES=()
declare -gA MOCK_HUGINN_AGENT_MEMORY=()
declare -gA MOCK_HUGINN_SCENARIO_AGENTS=()

# Mock modes
export HUGINN_MOCK_MODE="${HUGINN_MOCK_MODE:-healthy}"  # healthy|unhealthy|installing|stopped|offline|error

# Configuration
export HUGINN_PORT="${HUGINN_PORT:-$HUGINN_DEFAULT_PORT}"
export HUGINN_BASE_URL="${HUGINN_BASE_URL:-$HUGINN_DEFAULT_BASE_URL}"
export HUGINN_DB_PORT="${HUGINN_DB_PORT:-$HUGINN_DEFAULT_DB_PORT}"
export HUGINN_CONTAINER_NAME="${HUGINN_CONTAINER_NAME:-$HUGINN_DEFAULT_CONTAINER_NAME}"
export HUGINN_DB_CONTAINER_NAME="${HUGINN_DB_CONTAINER_NAME:-$HUGINN_DEFAULT_DB_CONTAINER_NAME}"
export HUGINN_USERNAME="${HUGINN_USERNAME:-admin}"
export HUGINN_PASSWORD="${HUGINN_PASSWORD:-password}"

# Test namespace for isolation
export TEST_NAMESPACE="${TEST_NAMESPACE:-test}"

#######################################
# Initialize state file for subshell access
#######################################
_huginn_mock_init_state_file() {
    if [[ ! -f "$HUGINN_MOCK_STATE_FILE" ]]; then
        local state_dir
        state_dir=$(dirname "$HUGINN_MOCK_STATE_FILE")
        mkdir -p "$state_dir" 2>/dev/null || true
        
        cat > "$HUGINN_MOCK_STATE_FILE" << 'EOF'
# Huginn mock state file
declare -gA MOCK_HUGINN_AGENTS=()
declare -gA MOCK_HUGINN_SCENARIOS=()
declare -gA MOCK_HUGINN_EVENTS=()
declare -gA MOCK_HUGINN_USERS=()
declare -gA MOCK_HUGINN_CALL_COUNT=()
declare -gA MOCK_HUGINN_ERRORS=()
declare -gA MOCK_HUGINN_AGENT_STATES=()
declare -gA MOCK_HUGINN_AGENT_MEMORY=()
declare -gA MOCK_HUGINN_SCENARIO_AGENTS=()
EOF
    fi
}

#######################################
# Save state to file for subshell access
#######################################
_huginn_mock_save_state() {
    local state_dir
    state_dir=$(dirname "$HUGINN_MOCK_STATE_FILE")
    mkdir -p "$state_dir" 2>/dev/null || true
    
    {
        echo "# Huginn mock state file - auto generated"
        
        # MOCK_HUGINN_AGENTS
        echo "declare -gA MOCK_HUGINN_AGENTS"
        for key in "${!MOCK_HUGINN_AGENTS[@]}"; do
            printf 'MOCK_HUGINN_AGENTS[%q]=%q\n' "$key" "${MOCK_HUGINN_AGENTS[$key]}"
        done
        
        # MOCK_HUGINN_SCENARIOS
        echo "declare -gA MOCK_HUGINN_SCENARIOS"
        for key in "${!MOCK_HUGINN_SCENARIOS[@]}"; do
            printf 'MOCK_HUGINN_SCENARIOS[%q]=%q\n' "$key" "${MOCK_HUGINN_SCENARIOS[$key]}"
        done
        
        # MOCK_HUGINN_EVENTS
        echo "declare -gA MOCK_HUGINN_EVENTS"
        for key in "${!MOCK_HUGINN_EVENTS[@]}"; do
            printf 'MOCK_HUGINN_EVENTS[%q]=%q\n' "$key" "${MOCK_HUGINN_EVENTS[$key]}"
        done
        
        # MOCK_HUGINN_USERS
        echo "declare -gA MOCK_HUGINN_USERS"
        for key in "${!MOCK_HUGINN_USERS[@]}"; do
            printf 'MOCK_HUGINN_USERS[%q]=%q\n' "$key" "${MOCK_HUGINN_USERS[$key]}"
        done
        
        # MOCK_HUGINN_CALL_COUNT
        echo "declare -gA MOCK_HUGINN_CALL_COUNT"
        for key in "${!MOCK_HUGINN_CALL_COUNT[@]}"; do
            printf 'MOCK_HUGINN_CALL_COUNT[%q]=%q\n' "$key" "${MOCK_HUGINN_CALL_COUNT[$key]}"
        done
        
        # MOCK_HUGINN_ERRORS
        echo "declare -gA MOCK_HUGINN_ERRORS"
        for key in "${!MOCK_HUGINN_ERRORS[@]}"; do
            printf 'MOCK_HUGINN_ERRORS[%q]=%q\n' "$key" "${MOCK_HUGINN_ERRORS[$key]}"
        done
        
        # MOCK_HUGINN_AGENT_STATES
        echo "declare -gA MOCK_HUGINN_AGENT_STATES"
        for key in "${!MOCK_HUGINN_AGENT_STATES[@]}"; do
            printf 'MOCK_HUGINN_AGENT_STATES[%q]=%q\n' "$key" "${MOCK_HUGINN_AGENT_STATES[$key]}"
        done
        
        # MOCK_HUGINN_AGENT_MEMORY
        echo "declare -gA MOCK_HUGINN_AGENT_MEMORY"
        for key in "${!MOCK_HUGINN_AGENT_MEMORY[@]}"; do
            printf 'MOCK_HUGINN_AGENT_MEMORY[%q]=%q\n' "$key" "${MOCK_HUGINN_AGENT_MEMORY[$key]}"
        done
        
        # MOCK_HUGINN_SCENARIO_AGENTS
        echo "declare -gA MOCK_HUGINN_SCENARIO_AGENTS"
        for key in "${!MOCK_HUGINN_SCENARIO_AGENTS[@]}"; do
            printf 'MOCK_HUGINN_SCENARIO_AGENTS[%q]=%q\n' "$key" "${MOCK_HUGINN_SCENARIO_AGENTS[$key]}"
        done
    } > "$HUGINN_MOCK_STATE_FILE"
}

#######################################
# Load state from file
#######################################
_huginn_mock_load_state() {
    if [[ -f "$HUGINN_MOCK_STATE_FILE" ]]; then
        # shellcheck disable=SC1090
        source "$HUGINN_MOCK_STATE_FILE" 2>/dev/null || true
    fi
}

# Initialize state file
_huginn_mock_init_state_file

# ----------------------------
# Utility Functions
# ----------------------------

_huginn_mock_log() {
    local level="$1"
    shift
    if [[ -n "${MOCK_LOG_DIR:-}" ]]; then
        # Ensure the log directory exists
        mkdir -p "${MOCK_LOG_DIR}" 2>/dev/null || true
        echo "[$(date '+%Y-%m-%d %H:%M:%S' 2>/dev/null || echo 'TIMESTAMP')] [$level] [HUGINN_MOCK] $*" >> "${MOCK_LOG_DIR}/huginn.log" 2>/dev/null || true
    fi
    [[ "${MOCK_DEBUG:-}" == "true" ]] && echo "[HUGINN_MOCK] [$level] $*" >&2 || true
}

_huginn_mock_increment_call_count() {
    local command="$1"
    _huginn_mock_load_state
    local count="${MOCK_HUGINN_CALL_COUNT[$command]:-0}"
    MOCK_HUGINN_CALL_COUNT[$command]=$((count + 1))
    _huginn_mock_save_state
}

# ----------------------------
# Public Mock Functions
# ----------------------------

#######################################
# Reset mock to clean state
#######################################
mock::huginn::reset() {
    declare -gA MOCK_HUGINN_AGENTS=()
    declare -gA MOCK_HUGINN_SCENARIOS=()
    declare -gA MOCK_HUGINN_EVENTS=()
    declare -gA MOCK_HUGINN_USERS=()
    declare -gA MOCK_HUGINN_CALL_COUNT=()
    declare -gA MOCK_HUGINN_ERRORS=()
    declare -gA MOCK_HUGINN_AGENT_STATES=()
    declare -gA MOCK_HUGINN_AGENT_MEMORY=()
    declare -gA MOCK_HUGINN_SCENARIO_AGENTS=()
    
    # Reset mode to healthy
    export HUGINN_MOCK_MODE="healthy"
    
    # Initialize default data
    mock::huginn::setup_default_data
    
    # Save state
    _huginn_mock_save_state
    
    _huginn_mock_log "INFO" "Mock state reset"
}

#######################################
# Setup default mock data
#######################################
mock::huginn::setup_default_data() {
    # Default user
    MOCK_HUGINN_USERS["admin"]='{"id":1,"username":"admin","email":"admin@localhost","admin":true}'
    
    # Default agents
    MOCK_HUGINN_AGENTS["1"]='{"id":1,"name":"Weather Agent","type":"Agents::WeatherAgent","schedule":"every_1h","disabled":false,"guid":"weather-agent-001"}'
    MOCK_HUGINN_AGENTS["2"]='{"id":2,"name":"Email Digest","type":"Agents::EmailDigestAgent","schedule":"9am","disabled":false,"guid":"email-digest-001"}'
    MOCK_HUGINN_AGENTS["3"]='{"id":3,"name":"RSS Feed","type":"Agents::RssAgent","schedule":"every_30m","disabled":false,"guid":"rss-feed-001"}'
    
    # Agent states
    MOCK_HUGINN_AGENT_STATES["1"]="ok"
    MOCK_HUGINN_AGENT_STATES["2"]="ok"
    MOCK_HUGINN_AGENT_STATES["3"]="warning"
    
    # Agent memory
    MOCK_HUGINN_AGENT_MEMORY["1"]='{"last_weather":{"temperature":68,"conditions":"sunny"}}'
    MOCK_HUGINN_AGENT_MEMORY["2"]='{"last_digest_sent":"2024-01-15T09:00:00Z"}'
    MOCK_HUGINN_AGENT_MEMORY["3"]='{"last_items":["item1","item2"]}'
    
    # Default scenarios
    MOCK_HUGINN_SCENARIOS["1"]='{"id":1,"name":"Weather Monitoring","description":"Monitor weather and send alerts","enabled":true}'
    MOCK_HUGINN_SCENARIO_AGENTS["1"]="1,2"
    
    # Default events
    MOCK_HUGINN_EVENTS["100"]='{"id":100,"agent_id":1,"created_at":"2024-01-15T15:00:00Z","payload":{"temperature":68,"conditions":"sunny"}}'
    MOCK_HUGINN_EVENTS["101"]='{"id":101,"agent_id":2,"created_at":"2024-01-15T09:00:00Z","payload":{"subject":"Daily Digest"}}'
}

#######################################
# Set mock mode
#######################################
mock::huginn::set_mode() {
    local mode="$1"
    export HUGINN_MOCK_MODE="$mode"
    _huginn_mock_log "INFO" "Mode set to: $mode"
    
    # Configure Docker state based on mode (if docker mock is available)
    # Only try to set docker state if the docker mock is loaded
    # We check this by looking for the DOCKER_MOCKS_LOADED variable
    if [[ "${DOCKER_MOCKS_LOADED:-}" == "true" ]]; then
        case "$mode" in
            healthy)
                mock::docker::set_container_state "$HUGINN_CONTAINER_NAME" "running" 2>/dev/null || true
                mock::docker::set_container_state "$HUGINN_DB_CONTAINER_NAME" "running" 2>/dev/null || true
                ;;
            unhealthy)
                mock::docker::set_container_state "$HUGINN_CONTAINER_NAME" "running" 2>/dev/null || true
                mock::docker::set_container_state "$HUGINN_DB_CONTAINER_NAME" "unhealthy" 2>/dev/null || true
                ;;
            installing)
                mock::docker::set_container_state "$HUGINN_CONTAINER_NAME" "starting" 2>/dev/null || true
                mock::docker::set_container_state "$HUGINN_DB_CONTAINER_NAME" "running" 2>/dev/null || true
                ;;
            stopped)
                mock::docker::set_container_state "$HUGINN_CONTAINER_NAME" "stopped" 2>/dev/null || true
                mock::docker::set_container_state "$HUGINN_DB_CONTAINER_NAME" "stopped" 2>/dev/null || true
                ;;
            offline|error)
                mock::docker::set_container_state "$HUGINN_CONTAINER_NAME" "error" 2>/dev/null || true
                mock::docker::set_container_state "$HUGINN_DB_CONTAINER_NAME" "error" 2>/dev/null || true
                ;;
        esac
    fi
}

#######################################
# Add an agent
#######################################
mock::huginn::add_agent() {
    local id="$1"
    local name="$2"
    local type="${3:-Agents::WebsiteAgent}"
    local schedule="${4:-never}"
    
    _huginn_mock_load_state
    
    local agent_json=$(cat <<EOF
{
    "id": $id,
    "name": "$name",
    "type": "$type",
    "schedule": "$schedule",
    "disabled": false,
    "guid": "$name-$(date +%s)"
}
EOF
)
    
    MOCK_HUGINN_AGENTS["$id"]="$agent_json"
    MOCK_HUGINN_AGENT_STATES["$id"]="ok"
    MOCK_HUGINN_AGENT_MEMORY["$id"]='{}'
    
    _huginn_mock_save_state
    _huginn_mock_log "INFO" "Added agent: $id - $name"
}

#######################################
# Add a scenario
#######################################
mock::huginn::add_scenario() {
    local id="$1"
    local name="$2"
    local description="${3:-Test scenario}"
    local agent_ids="${4:-}"
    
    _huginn_mock_load_state
    
    local scenario_json=$(cat <<EOF
{
    "id": $id,
    "name": "$name",
    "description": "$description",
    "enabled": true
}
EOF
)
    
    MOCK_HUGINN_SCENARIOS["$id"]="$scenario_json"
    [[ -n "$agent_ids" ]] && MOCK_HUGINN_SCENARIO_AGENTS["$id"]="$agent_ids"
    
    _huginn_mock_save_state
    _huginn_mock_log "INFO" "Added scenario: $id - $name"
}

#######################################
# Add an event
#######################################
mock::huginn::add_event() {
    local id="$1"
    local agent_id="$2"
    local payload="${3:-{}}"
    
    _huginn_mock_load_state
    
    local event_json=$(cat <<EOF
{
    "id": $id,
    "agent_id": $agent_id,
    "created_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || echo '2024-01-15T12:00:00Z')",
    "payload": $payload
}
EOF
)
    
    MOCK_HUGINN_EVENTS["$id"]="$event_json"
    
    _huginn_mock_save_state
    _huginn_mock_log "INFO" "Added event: $id for agent $agent_id"
}

#######################################
# Inject error for specific operation
#######################################
mock::huginn::inject_error() {
    local operation="$1"
    local error_type="${2:-generic}"
    
    _huginn_mock_load_state
    MOCK_HUGINN_ERRORS["$operation"]="$error_type"
    _huginn_mock_save_state
    
    _huginn_mock_log "INFO" "Injected error for operation: $operation ($error_type)"
}

#######################################
# Clear error for specific operation
#######################################
mock::huginn::clear_error() {
    local operation="$1"
    
    _huginn_mock_load_state
    unset MOCK_HUGINN_ERRORS["$operation"]
    _huginn_mock_save_state
    
    _huginn_mock_log "INFO" "Cleared error for operation: $operation"
}

#######################################
# Get call count for command
#######################################
mock::huginn::get_call_count() {
    local command="$1"
    _huginn_mock_load_state
    echo "${MOCK_HUGINN_CALL_COUNT[$command]:-0}"
}

#######################################
# Set agent state
#######################################
mock::huginn::set_agent_state() {
    local agent_id="$1"
    local state="$2"
    
    _huginn_mock_load_state
    MOCK_HUGINN_AGENT_STATES["$agent_id"]="$state"
    _huginn_mock_save_state
    
    _huginn_mock_log "INFO" "Set agent $agent_id state to: $state"
}

#######################################
# Set agent memory
#######################################
mock::huginn::set_agent_memory() {
    local agent_id="$1"
    local memory="$2"
    
    _huginn_mock_load_state
    MOCK_HUGINN_AGENT_MEMORY["$agent_id"]="$memory"
    _huginn_mock_save_state
    
    _huginn_mock_log "INFO" "Updated agent $agent_id memory"
}

# ----------------------------
# Command Mock Functions
# ----------------------------

#######################################
# Mock docker exec for Rails runner
#######################################
docker() {
    _huginn_mock_increment_call_count "docker"
    _huginn_mock_log "DEBUG" "docker called with: $*"
    
    # Handle docker exec for Rails runner commands
    if [[ "$1" == "exec" ]]; then
        shift
        local interactive=""
        local tty=""
        local container=""
        
        while [[ $# -gt 0 ]]; do
            case "$1" in
                -i) interactive="true"; shift ;;
                -t) tty="true"; shift ;;
                -*) shift ;;
                *)
                    container="$1"
                    shift
                    break
                    ;;
            esac
        done
        
        # Check if this is a Rails runner command
        if [[ "$1" == "bundle" && "$2" == "exec" && "$3" == "rails" && "$4" == "runner" ]]; then
            shift 4  # Remove bundle exec rails runner
            local ruby_code="$1"
            
            _huginn_mock_log "DEBUG" "Rails runner executing code"
            
            # Check for errors
            _huginn_mock_load_state
            if [[ -n "${MOCK_HUGINN_ERRORS[rails_runner]:-}" ]]; then
                echo "Error: ${MOCK_HUGINN_ERRORS[rails_runner]}" >&2
                return 1
            fi
            
            # Simulate Rails runner output based on mode
            case "$HUGINN_MOCK_MODE" in
                healthy)
                    # Parse the Ruby code to determine what to return
                    if [[ "$ruby_code" == *"Agent.count"* ]]; then
                        echo "Total agents: ${#MOCK_HUGINN_AGENTS[@]}"
                    elif [[ "$ruby_code" == *"Scenario.count"* ]]; then
                        echo "Total scenarios: ${#MOCK_HUGINN_SCENARIOS[@]}"
                    elif [[ "$ruby_code" == *"Event.count"* ]]; then
                        echo "Total events: ${#MOCK_HUGINN_EVENTS[@]}"
                    elif [[ "$ruby_code" == *"Agent.all"* ]]; then
                        # List all agents
                        for agent_data in "${MOCK_HUGINN_AGENTS[@]}"; do
                            echo "$agent_data"
                        done
                    elif [[ "$ruby_code" == *"Backup Summary"* ]]; then
                        # Backup operation output
                        echo "📊 Backup Summary:"
                        echo "   Timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
                        echo "   Users: ${#MOCK_HUGINN_USERS[@]}"
                        echo "   Agents: ${#MOCK_HUGINN_AGENTS[@]}"
                        echo "   Scenarios: ${#MOCK_HUGINN_SCENARIOS[@]}"
                        echo "   Events: ${#MOCK_HUGINN_EVENTS[@]}"
                        echo "   Links: 5"
                        echo ""
                        echo "💾 Backup data prepared"
                        echo "   Size: 4096 bytes"
                        echo "   File: /tmp/huginn_backup_test.json"
                        echo ""
                        echo "✅ Backup complete!"
                    else
                        # Generic success output
                        echo "Operation completed successfully"
                    fi
                    return 0
                    ;;
                unhealthy)
                    echo "Error: Database connection failed" >&2
                    return 1
                    ;;
                *)
                    echo "Error: Service unavailable" >&2
                    return 1
                    ;;
            esac
        fi
        
        # Fall back to regular docker mock if available
        if [[ "${DOCKER_MOCKS_LOADED:-}" == "true" ]]; then
            mock::docker::docker exec "$container" "$@" 2>/dev/null || echo "Mock container execution"
        else
            echo "Mock container execution"
        fi
        return 0
    fi
    
    # Handle other docker commands
    case "$1" in
        ps)
            # Simple container listing  
            if [[ "$HUGINN_MOCK_MODE" == "healthy" || "$HUGINN_MOCK_MODE" == "running" ]]; then
                echo "$HUGINN_CONTAINER_NAME"
                echo "$HUGINN_DB_CONTAINER_NAME"
            fi
            return 0
            ;;
        logs)
            echo "[$(date)] Huginn mock container logs"
            return 0
            ;;
        *)
            # Delegate to docker mock if available
            if [[ "${DOCKER_MOCKS_LOADED:-}" == "true" ]]; then
                mock::docker::docker "$@" 2>/dev/null || true
            else
                echo "Mock docker command: $*"
            fi
            ;;
    esac
}

#######################################
# Mock curl for HTTP API calls
#######################################
curl() {
    _huginn_mock_increment_call_count "curl"
    _huginn_mock_log "DEBUG" "curl called with: $*"
    
    local url=""
    local method="GET"
    local data=""
    local headers=()
    local silent=""
    
    # Parse curl arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -X|--request)
                method="$2"
                shift 2
                ;;
            -d|--data)
                data="$2"
                shift 2
                ;;
            -H|--header)
                headers+=("$2")
                shift 2
                ;;
            -s|--silent)
                silent="true"
                shift
                ;;
            http://*)
                url="$1"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Check for errors
    _huginn_mock_load_state
    if [[ -n "${MOCK_HUGINN_ERRORS[curl]:-}" ]]; then
        echo "curl: (7) Failed to connect" >&2
        return 7
    fi
    
    # Handle different endpoints
    case "$url" in
        */health)
            case "$HUGINN_MOCK_MODE" in
                healthy)
                    echo '{"status":"ok","version":"2023.03.20","database":"connected"}'
                    ;;
                unhealthy)
                    echo '{"status":"unhealthy","error":"Database connection failed"}'
                    return 1
                    ;;
                installing)
                    echo '{"status":"installing","progress":40,"current_step":"Running database migrations"}'
                    ;;
                *)
                    return 7
                    ;;
            esac
            ;;
        */agents)
            if [[ "$HUGINN_MOCK_MODE" == "healthy" ]]; then
                if [[ ${#MOCK_HUGINN_AGENTS[@]} -eq 0 ]]; then
                    echo '[]'
                else
                    echo '['
                    local first=true
                    for agent_data in "${MOCK_HUGINN_AGENTS[@]}"; do
                        [[ "$first" == "true" ]] && first=false || echo ","
                        echo "$agent_data"
                    done
                    echo ']'
                fi
            else
                echo '{"error":"Service unavailable"}'
                return 1
            fi
            ;;
        */agents/*)
            if [[ "$HUGINN_MOCK_MODE" == "healthy" ]]; then
                local agent_id="${url##*/agents/}"
                agent_id="${agent_id%%/*}"
                if [[ -n "${MOCK_HUGINN_AGENTS[$agent_id]:-}" ]]; then
                    local agent="${MOCK_HUGINN_AGENTS[$agent_id]}"
                    local memory="${MOCK_HUGINN_AGENT_MEMORY[$agent_id]:-{}}"
                    echo "${agent%\}},\"memory\":$memory}"
                else
                    echo '{"error":"Agent not found"}'
                    return 1
                fi
            else
                echo '{"error":"Service unavailable"}'
                return 1
            fi
            ;;
        */scenarios)
            if [[ "$HUGINN_MOCK_MODE" == "healthy" ]]; then
                if [[ ${#MOCK_HUGINN_SCENARIOS[@]} -eq 0 ]]; then
                    echo '[]'
                else
                    echo '['
                    local first=true
                    for scenario_data in "${MOCK_HUGINN_SCENARIOS[@]}"; do
                        [[ "$first" == "true" ]] && first=false || echo ","
                        local scenario_id
                        scenario_id=$(echo "$scenario_data" | grep -o '"id":[0-9]*' | cut -d: -f2)
                        local agents="${MOCK_HUGINN_SCENARIO_AGENTS[$scenario_id]:-}"
                        if [[ -n "$agents" ]]; then
                            echo "${scenario_data%\}},\"agents\":[${agents}]}"
                        else
                            echo "$scenario_data"
                        fi
                    done
                    echo ']'
                fi
            else
                echo '{"error":"Service unavailable"}'
                return 1
            fi
            ;;
        */events)
            if [[ "$HUGINN_MOCK_MODE" == "healthy" ]]; then
                if [[ ${#MOCK_HUGINN_EVENTS[@]} -eq 0 ]]; then
                    echo '[]'
                else
                    echo '['
                    local first=true
                    for event_data in "${MOCK_HUGINN_EVENTS[@]}"; do
                        [[ "$first" == "true" ]] && first=false || echo ","
                        echo "$event_data"
                    done
                    echo ']'
                fi
            else
                echo '{"error":"Service unavailable"}'
                return 1
            fi
            ;;
        *)
            echo '{"error":"Unknown endpoint"}'
            return 1
            ;;
    esac
}

# ----------------------------
# Export Functions
# ----------------------------

export -f mock::huginn::reset
export -f mock::huginn::setup_default_data
export -f mock::huginn::set_mode
export -f mock::huginn::add_agent
export -f mock::huginn::add_scenario
export -f mock::huginn::add_event
export -f mock::huginn::inject_error
export -f mock::huginn::clear_error
export -f mock::huginn::get_call_count
export -f mock::huginn::set_agent_state
export -f mock::huginn::set_agent_memory
export -f docker
export -f curl
export -f _huginn_mock_init_state_file
export -f _huginn_mock_save_state
export -f _huginn_mock_load_state
export -f _huginn_mock_log
export -f _huginn_mock_increment_call_count

# Only log if we're in debug mode or log directory is already set up
[[ "${MOCK_DEBUG:-}" == "true" || -n "${MOCK_LOG_DIR:-}" ]] && _huginn_mock_log "INFO" "Huginn mock implementation loaded" || true