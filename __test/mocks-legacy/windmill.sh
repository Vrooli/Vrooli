#!/usr/bin/env bash
# Windmill Mock Implementation
#
# Provides comprehensive mock for Windmill workflow automation platform including:
# - Docker Compose operations simulation
# - API endpoint mocking
# - Worker management
# - Database operations
# - Backup/restore operations
# - Status monitoring
# - Configuration management
#
# This mock follows the same standards as redis.sh and filesystem.sh with:
# - Comprehensive state management
# - File-based persistence for BATS compatibility
# - Integration with centralized logging
# - Test helper functions
# - Error injection capabilities

# Prevent duplicate loading
[[ -n "${WINDMILL_MOCK_LOADED:-}" ]] && return 0
declare -g WINDMILL_MOCK_LOADED=1

# Load dependencies
MOCK_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
[[ -f "$MOCK_DIR/logs.sh" ]] && source "$MOCK_DIR/logs.sh"

# Global configuration
declare -g WINDMILL_MOCK_STATE_DIR="${WINDMILL_MOCK_STATE_DIR:-/tmp/windmill-mock-state}"
declare -g WINDMILL_MOCK_DEBUG="${WINDMILL_MOCK_DEBUG:-}"

# Global state arrays and variables
declare -gA WINDMILL_MOCK_CONFIG=(
    [installed]="false"
    [status]="stopped"
    [server_port]="8000"
    [base_url]="http://localhost:8000"
    [admin_email]="admin@windmill.dev"
    [admin_password]="changeme"
    [db_status]="stopped"
    [worker_replicas]="3"
    [worker_memory]="2048M"
    [api_responding]="false"
    [version]="1.290.0"
    [error_mode]=""
    [backup_enabled]="true"
    [lsp_enabled]="true"
    [multiplayer_enabled]="false"
)

declare -gA WINDMILL_MOCK_SERVICES=(
    [windmill-app]="stopped"
    [windmill-worker]="stopped"
    [windmill-db]="stopped"
    [windmill-lsp]="stopped"
    [windmill-multiplayer]="stopped"
    [windmill-worker-native]="stopped"
)

declare -gA WINDMILL_MOCK_CONTAINERS=(
    [windmill-app]="container_windmill-vrooli-app-1"
    [windmill-worker]="container_windmill-vrooli-worker-1"
    [windmill-db]="container_windmill-vrooli-db-1"
    [windmill-lsp]="container_windmill-vrooli-lsp-1"
)

declare -gA WINDMILL_MOCK_APPS=()       # Available apps
declare -gA WINDMILL_MOCK_WORKSPACES=(  # Workspace data
    [demo]="active"
    [admin]="active"
)
declare -gA WINDMILL_MOCK_API_KEYS=()   # Stored API keys
declare -ga WINDMILL_MOCK_BACKUPS=()    # Available backups
declare -ga WINDMILL_MOCK_LOGS=()       # Service logs

# Initialize state directory
mkdir -p "$WINDMILL_MOCK_STATE_DIR"

#######################################
# State persistence functions
#######################################
mock::windmill::save_state() {
    local state_file="$WINDMILL_MOCK_STATE_DIR/windmill-state.sh"
    {
        echo "# Windmill mock state - $(date)"
        
        # Save arrays using declare -p for proper restoration
        declare -p WINDMILL_MOCK_CONFIG 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA WINDMILL_MOCK_CONFIG=()"
        declare -p WINDMILL_MOCK_SERVICES 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA WINDMILL_MOCK_SERVICES=()"
        declare -p WINDMILL_MOCK_CONTAINERS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA WINDMILL_MOCK_CONTAINERS=()"
        declare -p WINDMILL_MOCK_APPS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA WINDMILL_MOCK_APPS=()"
        declare -p WINDMILL_MOCK_WORKSPACES 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA WINDMILL_MOCK_WORKSPACES=()"
        declare -p WINDMILL_MOCK_API_KEYS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA WINDMILL_MOCK_API_KEYS=()"
        declare -p WINDMILL_MOCK_BACKUPS 2>/dev/null | sed 's/declare -a/declare -ga/' || echo "declare -ga WINDMILL_MOCK_BACKUPS=()"
        declare -p WINDMILL_MOCK_LOGS 2>/dev/null | sed 's/declare -a/declare -ga/' || echo "declare -ga WINDMILL_MOCK_LOGS=()"
    } > "$state_file"
    
    mock::log_state "windmill" "Saved Windmill state to $state_file"
}

mock::windmill::load_state() {
    local state_file="$WINDMILL_MOCK_STATE_DIR/windmill-state.sh"
    if [[ -f "$state_file" ]]; then
        source "$state_file"
        mock::log_state "windmill" "Loaded Windmill state from $state_file"
    fi
}

# Automatically load state when sourced
mock::windmill::load_state

#######################################
# Main command interceptors
#######################################

# Intercept docker-compose calls for windmill
docker-compose() {
    mock::log_and_verify "docker-compose" "$@"
    
    # Check if this is a windmill-related compose call
    local is_windmill_call=false
    for arg in "$@"; do
        if [[ "$arg" == *"windmill"* ]] || [[ "$arg" == *"docker-compose.yml"* ]]; then
            is_windmill_call=true
            break
        fi
    done
    
    if [[ "$is_windmill_call" == "true" ]]; then
        mock::windmill::load_state
        local result
        mock::windmill::handle_compose_command "$@"
        result=$?
        mock::windmill::save_state
        return $result
    else
        # Not windmill-related, pass through to real docker-compose
        command docker-compose "$@"
    fi
}

# Intercept curl calls for windmill API
curl() {
    mock::log_and_verify "curl" "$@"
    
    # Check if this is a windmill API call
    local is_windmill_api=false
    for arg in "$@"; do
        if [[ "$arg" == *"localhost:8000"* ]] || [[ "$arg" == *"windmill"* && "$arg" == *"/api/"* ]]; then
            is_windmill_api=true
            break
        fi
    done
    
    if [[ "$is_windmill_api" == "true" ]]; then
        mock::windmill::load_state
        local result
        mock::windmill::handle_api_call "$@"
        result=$?
        mock::windmill::save_state
        return $result
    else
        # Not windmill API, pass through
        command curl "$@"
    fi
}

#######################################
# Docker Compose command handler
#######################################
mock::windmill::handle_compose_command() {
    # Skip arguments until we find the actual command
    local cmd=""
    while [[ $# -gt 0 ]]; do
        case "$1" in
            "-f"|"--file")
                shift 2  # Skip -f and its argument
                ;;
            "-p"|"--project-name")
                shift 2  # Skip -p and its argument
                ;;
            "--env-file")
                shift 2  # Skip --env-file and its argument
                ;;
            "-*")
                shift    # Skip other flags
                ;;
            *)
                cmd="$1" # First non-flag argument is the command
                shift
                break
                ;;
        esac
    done
    
    # Check for error injection
    if [[ -n "${WINDMILL_MOCK_CONFIG[error_mode]}" ]]; then
        case "${WINDMILL_MOCK_CONFIG[error_mode]}" in
            "compose_failure")
                echo "ERROR: Docker Compose operation failed" >&2
                return 1
                ;;
            "service_unhealthy")
                # Let command proceed but mark services as unhealthy
                ;;
        esac
    fi
    
    case "$cmd" in
        "up")
            mock::windmill::compose_up "$@"
            ;;
        "down")
            mock::windmill::compose_down "$@"
            ;;
        "restart")
            mock::windmill::compose_restart "$@"
            ;;
        "ps")
            mock::windmill::compose_ps "$@"
            ;;
        "logs")
            mock::windmill::compose_logs "$@"
            ;;
        "exec")
            mock::windmill::compose_exec "$@"
            ;;
        "scale")
            mock::windmill::compose_scale "$@"
            ;;
        *)
            echo "Mock docker-compose: Unknown command $cmd" >&2
            return 1
            ;;
    esac
}

mock::windmill::compose_up() {
    local detached=false
    local scale_args=()
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            "-d"|"--detach")
                detached=true
                shift
                ;;
            "--scale")
                scale_args+=("$2")
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Start all windmill services
    WINDMILL_MOCK_SERVICES[windmill-app]="running"
    WINDMILL_MOCK_SERVICES[windmill-worker]="running"
    WINDMILL_MOCK_SERVICES[windmill-db]="running"
    
    if [[ "${WINDMILL_MOCK_CONFIG[lsp_enabled]}" == "true" ]]; then
        WINDMILL_MOCK_SERVICES[windmill-lsp]="running"
    fi
    
    if [[ "${WINDMILL_MOCK_CONFIG[multiplayer_enabled]}" == "true" ]]; then
        WINDMILL_MOCK_SERVICES[windmill-multiplayer]="running"
    fi
    
    # Handle scaling
    for scale_arg in "${scale_args[@]}"; do
        if [[ "$scale_arg" == windmill-worker=* ]]; then
            local count="${scale_arg#*=}"
            WINDMILL_MOCK_CONFIG[worker_replicas]="$count"
        fi
    done
    
    WINDMILL_MOCK_CONFIG[status]="running"
    WINDMILL_MOCK_CONFIG[db_status]="running"
    
    # API becomes responsive after startup
    if [[ "${WINDMILL_MOCK_CONFIG[error_mode]}" != "service_unhealthy" ]]; then
        WINDMILL_MOCK_CONFIG[api_responding]="true"
    fi
    
    # Simulate docker-compose output
    echo "Creating network \"windmill-vrooli_default\" with the default driver"
    echo "Creating windmill-vrooli-db-1 ... done"
    echo "Creating windmill-vrooli-app-1 ... done"
    echo "Creating windmill-vrooli-worker-1 ... done"
    if [[ "${WINDMILL_MOCK_CONFIG[lsp_enabled]}" == "true" ]]; then
        echo "Creating windmill-vrooli-lsp-1 ... done"
    fi
    
    return 0
}

mock::windmill::compose_down() {
    local remove_volumes=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            "-v"|"--volumes")
                remove_volumes=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Stop all services
    for service in "${!WINDMILL_MOCK_SERVICES[@]}"; do
        WINDMILL_MOCK_SERVICES[$service]="stopped"
    done
    
    WINDMILL_MOCK_CONFIG[status]="stopped"
    WINDMILL_MOCK_CONFIG[db_status]="stopped"
    WINDMILL_MOCK_CONFIG[api_responding]="false"
    
    # Clear data if volumes removed
    if [[ "$remove_volumes" == "true" ]]; then
        WINDMILL_MOCK_APPS=()
        WINDMILL_MOCK_WORKSPACES=(
            [demo]="active"
            [admin]="active"
        )
    fi
    
    echo "Stopping windmill-vrooli-worker-1 ... done"
    echo "Stopping windmill-vrooli-app-1 ... done"
    echo "Stopping windmill-vrooli-db-1 ... done"
    echo "Removing windmill-vrooli-worker-1 ... done"
    echo "Removing windmill-vrooli-app-1 ... done"
    echo "Removing windmill-vrooli-db-1 ... done"
    echo "Removing network windmill-vrooli_default"
    
    if [[ "$remove_volumes" == "true" ]]; then
        echo "Removing volume windmill-vrooli_db_data"
        echo "Removing volume windmill-vrooli_worker_data"
    fi
    
    return 0
}

mock::windmill::compose_restart() {
    local service="${1:-}"
    
    if [[ -n "$service" ]]; then
        if [[ -n "${WINDMILL_MOCK_SERVICES[$service]}" ]]; then
            echo "Restarting windmill-vrooli-${service}-1 ... done"
        else
            echo "No such service: $service" >&2
            return 1
        fi
    else
        echo "Restarting windmill-vrooli-db-1 ... done"
        echo "Restarting windmill-vrooli-app-1 ... done" 
        echo "Restarting windmill-vrooli-worker-1 ... done"
    fi
    
    return 0
}

mock::windmill::compose_ps() {
    local format="table"
    local services_filter=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            "--services")
                format="services"
                shift
                ;;
            *)
                services_filter="$1"
                shift
                ;;
        esac
    done
    
    if [[ "$format" == "services" ]]; then
        for service in "${!WINDMILL_MOCK_SERVICES[@]}"; do
            if [[ "${WINDMILL_MOCK_SERVICES[$service]}" == "running" ]]; then
                echo "$service"
            fi
        done
    else
        echo "Name                       Command                  State           Ports"
        echo "-------------------------------------------------------------------------------"
        for service in "${!WINDMILL_MOCK_SERVICES[@]}"; do
            local status="${WINDMILL_MOCK_SERVICES[$service]}"
            local state
            case "$status" in
                "running") state="Up" ;;
                "stopped") state="Exit 0" ;;
                *) state="Unknown" ;;
            esac
            echo "windmill-vrooli-${service}-1   /app/start.sh            $state           8000/tcp"
        done
    fi
    
    return 0
}

mock::windmill::compose_logs() {
    local follow=false
    local service=""
    local tail_lines=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            "-f"|"--follow")
                follow=true
                shift
                ;;
            "--tail")
                tail_lines="$2"
                shift 2
                ;;
            *)
                service="$1"
                shift
                ;;
        esac
    done
    
    # Generate mock log output
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    if [[ -n "$service" ]]; then
        echo "windmill-vrooli-${service}-1 | [$timestamp] INFO: Service $service is running"
        echo "windmill-vrooli-${service}-1 | [$timestamp] INFO: Ready to accept connections"
    else
        echo "windmill-vrooli-db-1     | [$timestamp] INFO: Database ready"
        echo "windmill-vrooli-app-1    | [$timestamp] INFO: Windmill server started on port ${WINDMILL_MOCK_CONFIG[server_port]}"
        echo "windmill-vrooli-worker-1 | [$timestamp] INFO: Worker started with ${WINDMILL_MOCK_CONFIG[worker_replicas]} replicas"
    fi
    
    if [[ "$follow" == "true" ]]; then
        echo "Following logs... (Press Ctrl+C to exit)"
        sleep 1
    fi
    
    return 0
}

mock::windmill::compose_exec() {
    local service="$1"
    shift
    local command="$*"
    
    if [[ -z "${WINDMILL_MOCK_SERVICES[$service]}" ]]; then
        echo "ERROR: No such service: $service" >&2
        return 1
    fi
    
    if [[ "${WINDMILL_MOCK_SERVICES[$service]}" != "running" ]]; then
        echo "ERROR: Service $service is not running" >&2
        return 1
    fi
    
    # Mock command execution
    case "$command" in
        *"psql"*)
            echo "psql (13.7)"
            echo "Type \"help\" for help."
            echo "windmill=#"
            ;;
        *"ls"*)
            echo "app  data  logs  temp"
            ;;
        *)
            echo "Executing: $command"
            ;;
    esac
    
    return 0
}

mock::windmill::compose_scale() {
    echo "Mock docker-compose scale: $*"
    
    # Extract worker count if specified
    for arg in "$@"; do
        if [[ "$arg" == windmill-worker=* ]]; then
            local count="${arg#*=}"
            WINDMILL_MOCK_CONFIG[worker_replicas]="$count"
            echo "Scaling windmill-worker to $count"
        fi
    done
    
    return 0
}

#######################################
# API call handler
#######################################
mock::windmill::handle_api_call() {
    local url=""
    local method="GET"
    local data=""
    local silent=false
    local fail_on_error=false
    local headers=()
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            "-s"|"--silent")
                silent=true
                shift
                ;;
            "-f"|"--fail")
                fail_on_error=true
                shift
                ;;
            "-X"|"--request")
                method="$2"
                shift 2
                ;;
            "-d"|"--data")
                data="$2"
                shift 2
                ;;
            "-H"|"--header")
                headers+=("$2")
                shift 2
                ;;
            http*)
                url="$1"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Check if API is responding
    if [[ "${WINDMILL_MOCK_CONFIG[api_responding]}" != "true" ]]; then
        echo "curl: (7) Failed to connect to localhost port ${WINDMILL_MOCK_CONFIG[server_port]}: Connection refused" >&2
        return 7  # Connection refused always returns 7, regardless of -f flag
    fi
    
    # Extract endpoint from URL
    local endpoint=""
    if [[ "$url" =~ /api/(.+)$ ]]; then
        endpoint="${BASH_REMATCH[1]}"
    fi
    
    # Handle different API endpoints
    case "$endpoint" in
        "version")
            echo "\"v${WINDMILL_MOCK_CONFIG[version]}\""
            ;;
        "health")
            if [[ "${WINDMILL_MOCK_CONFIG[error_mode]}" == "api_unhealthy" ]]; then
                return 1
            fi
            echo "{\"status\":\"healthy\"}"
            ;;
        "w/list")
            echo "[{\"id\":\"demo\",\"name\":\"Demo Workspace\"},{\"id\":\"admin\",\"name\":\"Admin Workspace\"}]"
            ;;
        "users/whoami")
            # Check for auth header
            local has_auth=false
            for header in "${headers[@]}"; do
                if [[ "$header" == "Authorization:"* ]]; then
                    has_auth=true
                    break
                fi
            done
            
            if [[ "$has_auth" == "false" ]]; then
                echo "{\"error\":\"Unauthorized\"}" >&2
                return 22  # curl uses exit code 22 for HTTP errors like 401
            fi
            
            echo "{\"username\":\"admin\",\"email\":\"${WINDMILL_MOCK_CONFIG[admin_email]}\"}"
            ;;
        "workers/list")
            local worker_count="${WINDMILL_MOCK_CONFIG[worker_replicas]}"
            echo "["
            for ((i=1; i<=worker_count; i++)); do
                echo "  {\"name\":\"worker-$i\",\"status\":\"running\",\"memory\":\"${WINDMILL_MOCK_CONFIG[worker_memory]}\"}"
                if [[ $i -lt $worker_count ]]; then
                    echo ","
                fi
            done
            echo "]"
            ;;
        w/*/scripts/list|w/*/jobs/list|w/*/resources/list|w/*/variables/list|w/*/schedules/list)
            echo "[]"
            ;;
        w/*/jobs/run/*)
            if [[ "$method" == "POST" ]]; then
                echo "{\"id\":\"job-$(date +%s)\",\"status\":\"running\"}"
            else
                return 1
            fi
            ;;
        *)
            echo "{\"error\":\"Not found\"}" >&2
            return 404
            ;;
    esac
    
    return 0
}

#######################################
# Test helper functions
#######################################
mock::windmill::reset() {
    local save_state="${1:-true}"
    
    # Reset all state to initial values
    WINDMILL_MOCK_CONFIG=(
        [installed]="false"
        [status]="stopped"
        [server_port]="8000"
        [base_url]="http://localhost:8000"
        [admin_email]="admin@windmill.dev"
        [admin_password]="changeme"
        [db_status]="stopped"
        [worker_replicas]="3"
        [worker_memory]="2048M"
        [api_responding]="false"
        [version]="1.290.0"
        [error_mode]=""
        [backup_enabled]="true"
        [lsp_enabled]="true"
        [multiplayer_enabled]="false"
    )
    
    WINDMILL_MOCK_SERVICES=(
        [windmill-app]="stopped"
        [windmill-worker]="stopped"
        [windmill-db]="stopped"
        [windmill-lsp]="stopped"
        [windmill-multiplayer]="stopped"
        [windmill-worker-native]="stopped"
    )
    
    WINDMILL_MOCK_APPS=()
    WINDMILL_MOCK_WORKSPACES=(
        [demo]="active"
        [admin]="active"
    )
    WINDMILL_MOCK_API_KEYS=()
    WINDMILL_MOCK_BACKUPS=()
    WINDMILL_MOCK_LOGS=()
    
    if [[ "$save_state" == "true" ]]; then
        mock::windmill::save_state
    fi
    
    mock::log_state "windmill" "Windmill mock reset to initial state"
}

mock::windmill::set_error() {
    local error_mode="$1"
    WINDMILL_MOCK_CONFIG[error_mode]="$error_mode"
    mock::windmill::save_state
    mock::log_state "windmill" "Set Windmill error mode: $error_mode"
}

mock::windmill::install() {
    WINDMILL_MOCK_CONFIG[installed]="true"
    mock::windmill::save_state
    mock::log_state "windmill" "Windmill installed"
}

mock::windmill::start() {
    if [[ "${WINDMILL_MOCK_CONFIG[installed]}" != "true" ]]; then
        echo "Error: Windmill is not installed" >&2
        return 1
    fi
    
    # Simulate startup
    WINDMILL_MOCK_CONFIG[status]="running"
    WINDMILL_MOCK_CONFIG[db_status]="running" 
    WINDMILL_MOCK_CONFIG[api_responding]="true"
    
    for service in "${!WINDMILL_MOCK_SERVICES[@]}"; do
        if [[ "$service" == "windmill-lsp" && "${WINDMILL_MOCK_CONFIG[lsp_enabled]}" != "true" ]]; then
            continue
        fi
        if [[ "$service" == "windmill-multiplayer" && "${WINDMILL_MOCK_CONFIG[multiplayer_enabled]}" != "true" ]]; then
            continue
        fi
        WINDMILL_MOCK_SERVICES[$service]="running"
    done
    
    mock::windmill::save_state
    mock::log_state "windmill" "Windmill started"
}

mock::windmill::stop() {
    WINDMILL_MOCK_CONFIG[status]="stopped"
    WINDMILL_MOCK_CONFIG[db_status]="stopped"
    WINDMILL_MOCK_CONFIG[api_responding]="false"
    
    for service in "${!WINDMILL_MOCK_SERVICES[@]}"; do
        WINDMILL_MOCK_SERVICES[$service]="stopped"
    done
    
    mock::windmill::save_state
    mock::log_state "windmill" "Windmill stopped"
}

mock::windmill::add_app() {
    local app_name="$1"
    local app_data="$2"
    WINDMILL_MOCK_APPS[$app_name]="$app_data"
    mock::windmill::save_state
    mock::log_state "windmill" "Added app: $app_name"
}

mock::windmill::add_workspace() {
    local workspace_name="$1"
    local status="${2:-active}"
    WINDMILL_MOCK_WORKSPACES[$workspace_name]="$status"
    mock::windmill::save_state
    mock::log_state "windmill" "Added workspace: $workspace_name"
}

mock::windmill::store_api_key() {
    local key_name="$1"
    local key_value="$2"
    WINDMILL_MOCK_API_KEYS[$key_name]="$key_value"
    mock::windmill::save_state
    mock::log_state "windmill" "Stored API key: $key_name"
}

mock::windmill::add_backup() {
    local backup_name="$1"
    WINDMILL_MOCK_BACKUPS+=("$backup_name")
    mock::windmill::save_state
    mock::log_state "windmill" "Added backup: $backup_name"
}

mock::windmill::scale_workers() {
    local count="$1"
    WINDMILL_MOCK_CONFIG[worker_replicas]="$count"
    mock::windmill::save_state
    mock::log_state "windmill" "Scaled workers to: $count"
}

#######################################
# Test assertions
#######################################
mock::windmill::assert_installed() {
    if [[ "${WINDMILL_MOCK_CONFIG[installed]}" == "true" ]]; then
        return 0
    else
        echo "Assertion failed: Windmill is not installed" >&2
        return 1
    fi
}

mock::windmill::assert_running() {
    if [[ "${WINDMILL_MOCK_CONFIG[status]}" == "running" ]]; then
        return 0
    else
        echo "Assertion failed: Windmill is not running (status: ${WINDMILL_MOCK_CONFIG[status]})" >&2
        return 1
    fi
}

mock::windmill::assert_stopped() {
    if [[ "${WINDMILL_MOCK_CONFIG[status]}" == "stopped" ]]; then
        return 0
    else
        echo "Assertion failed: Windmill is not stopped (status: ${WINDMILL_MOCK_CONFIG[status]})" >&2
        return 1
    fi
}

mock::windmill::assert_service_running() {
    local service="$1"
    if [[ "${WINDMILL_MOCK_SERVICES[$service]}" == "running" ]]; then
        return 0
    else
        echo "Assertion failed: Service '$service' is not running" >&2
        return 1
    fi
}

mock::windmill::assert_worker_count() {
    local expected_count="$1"
    local actual_count="${WINDMILL_MOCK_CONFIG[worker_replicas]}"
    if [[ "$actual_count" == "$expected_count" ]]; then
        return 0
    else
        echo "Assertion failed: Worker count mismatch" >&2
        echo "  Expected: $expected_count" >&2
        echo "  Actual: $actual_count" >&2
        return 1
    fi
}

mock::windmill::assert_app_exists() {
    local app_name="$1"
    if [[ -n "${WINDMILL_MOCK_APPS[$app_name]}" ]]; then
        return 0
    else
        echo "Assertion failed: App '$app_name' does not exist" >&2
        return 1
    fi
}

mock::windmill::assert_workspace_exists() {
    local workspace_name="$1"
    if [[ -n "${WINDMILL_MOCK_WORKSPACES[$workspace_name]}" ]]; then
        return 0
    else
        echo "Assertion failed: Workspace '$workspace_name' does not exist" >&2
        return 1
    fi
}

mock::windmill::assert_api_responding() {
    if [[ "${WINDMILL_MOCK_CONFIG[api_responding]}" == "true" ]]; then
        return 0
    else
        echo "Assertion failed: API is not responding" >&2
        return 1
    fi
}

#######################################
# Debug functions
#######################################
mock::windmill::dump_state() {
    echo "=== Windmill Mock State ==="
    echo "Configuration:"
    for key in "${!WINDMILL_MOCK_CONFIG[@]}"; do
        echo "  $key: ${WINDMILL_MOCK_CONFIG[$key]}"
    done
    
    echo "Services:"
    for service in "${!WINDMILL_MOCK_SERVICES[@]}"; do
        echo "  $service: ${WINDMILL_MOCK_SERVICES[$service]}"
    done
    
    echo "Apps:"
    for app in "${!WINDMILL_MOCK_APPS[@]}"; do
        echo "  $app: ${WINDMILL_MOCK_APPS[$app]}"
    done
    
    echo "Workspaces:"
    for workspace in "${!WINDMILL_MOCK_WORKSPACES[@]}"; do
        echo "  $workspace: ${WINDMILL_MOCK_WORKSPACES[$workspace]}"
    done
    
    echo "API Keys:"
    for key in "${!WINDMILL_MOCK_API_KEYS[@]}"; do
        echo "  $key: [REDACTED]"
    done
    
    echo "Backups:"
    for backup in "${WINDMILL_MOCK_BACKUPS[@]}"; do
        echo "  $backup"
    done
    echo "=========================="
}

# Export all functions
export -f docker-compose
export -f curl
export -f mock::windmill::save_state
export -f mock::windmill::load_state
export -f mock::windmill::handle_compose_command
export -f mock::windmill::handle_api_call
export -f mock::windmill::compose_up
export -f mock::windmill::compose_down
export -f mock::windmill::compose_restart
export -f mock::windmill::compose_ps
export -f mock::windmill::compose_logs
export -f mock::windmill::compose_exec
export -f mock::windmill::compose_scale
export -f mock::windmill::reset
export -f mock::windmill::set_error
export -f mock::windmill::install
export -f mock::windmill::start
export -f mock::windmill::stop
export -f mock::windmill::add_app
export -f mock::windmill::add_workspace
export -f mock::windmill::store_api_key
export -f mock::windmill::add_backup
export -f mock::windmill::scale_workers
export -f mock::windmill::assert_installed
export -f mock::windmill::assert_running
export -f mock::windmill::assert_stopped
export -f mock::windmill::assert_service_running
export -f mock::windmill::assert_worker_count
export -f mock::windmill::assert_app_exists
export -f mock::windmill::assert_workspace_exists
export -f mock::windmill::assert_api_responding
export -f mock::windmill::dump_state

# Save initial state
mock::windmill::save_state