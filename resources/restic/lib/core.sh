#!/usr/bin/env bash
# Restic Core Library - Core functionality for restic resource

set -euo pipefail

# Get script directory
if [[ -z "${SCRIPT_DIR:-}" ]]; then
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    readonly SCRIPT_DIR
fi
if [[ -z "${RESOURCE_DIR:-}" ]]; then
    RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"
    readonly RESOURCE_DIR
fi

# Source configuration
source "${RESOURCE_DIR}/config/defaults.sh"

# Source port registry
source "${VROOLI_ROOT:-${HOME}/Vrooli}/scripts/resources/port_registry.sh"

# Get restic port from registry
readonly RESTIC_PORT=$(ports::get_resource_port "restic")

# Container and volume names
readonly CONTAINER_NAME="vrooli-restic"
readonly REPOSITORY_VOLUME="restic-repository"
readonly CACHE_VOLUME="restic-cache"
readonly CONFIG_VOLUME="restic-config"

#######################################
# Show help information
#######################################
restic::show_help() {
    cat << EOF
Restic Resource - Encrypted backup and recovery service

Usage: resource-restic [COMMAND] [OPTIONS]

Commands:
  help                Show this help message
  info                Show resource information
  manage              Lifecycle management
    install           Install and initialize restic
    start             Start restic service
    stop              Stop restic service
    restart           Restart restic service
    uninstall         Remove restic (keeps repository)
  test                Run tests
    smoke             Quick health check
    integration       Full integration test
    unit              Unit tests
    all               Run all tests
  content             Content management
    add               Add backup target
    list              List backup targets
    get               Get backup target details
    remove            Remove backup target
    execute           Execute backup operation
  status              Show backup status
  logs                View restic logs
  
Resource-specific commands:
  backup              Trigger manual backup
  restore             Restore from snapshot
  snapshots           List available snapshots
  prune               Remove old snapshots
  verify              Verify backup integrity

Examples:
  resource-restic manage install        # Install and initialize
  resource-restic manage start          # Start service
  resource-restic backup --paths /data  # Create backup
  resource-restic snapshots             # List snapshots
  resource-restic restore --snapshot latest --target /restore

Configuration:
  Port: ${RESTIC_PORT}
  Repository: ${RESTIC_REPOSITORY:-/repository}
  Backend: ${RESTIC_BACKEND:-local}
EOF
}

#######################################
# Show resource information
#######################################
restic::show_info() {
    local json_output=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --json)
                json_output=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -f "${RESOURCE_DIR}/config/runtime.json" ]]; then
        if [[ "$json_output" == true ]]; then
            cat "${RESOURCE_DIR}/config/runtime.json"
        else
            echo "Restic Resource Information:"
            jq -r 'to_entries[] | "  \(.key): \(.value)"' "${RESOURCE_DIR}/config/runtime.json" 2>/dev/null || cat "${RESOURCE_DIR}/config/runtime.json"
        fi
    else
        echo "Runtime configuration not found"
        return 1
    fi
}

#######################################
# Manage lifecycle operations
#######################################
restic::manage() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        install)
            restic::install "$@"
            ;;
        uninstall)
            restic::uninstall "$@"
            ;;
        start)
            restic::start "$@"
            ;;
        stop)
            restic::stop "$@"
            ;;
        restart)
            restic::stop "$@"
            restic::start "$@"
            ;;
        *)
            echo "Error: Unknown manage command: $subcommand"
            echo "Valid commands: install, uninstall, start, stop, restart"
            return 1
            ;;
    esac
}

#######################################
# Install restic
#######################################
restic::install() {
    echo "Installing restic resource..."
    
    # Create volumes
    echo "Creating Docker volumes..."
    docker volume create "$REPOSITORY_VOLUME" 2>/dev/null || true
    docker volume create "$CACHE_VOLUME" 2>/dev/null || true
    docker volume create "$CONFIG_VOLUME" 2>/dev/null || true
    
    # Create network if it doesn't exist
    docker network create vrooli-network 2>/dev/null || true
    
    # Initialize repository if needed
    if [[ ! -f "/var/lib/docker/volumes/${REPOSITORY_VOLUME}/_data/config" ]]; then
        echo "Initializing restic repository..."
        docker run --rm \
            -v "${REPOSITORY_VOLUME}:/repository" \
            -e RESTIC_REPOSITORY=/repository \
            -e RESTIC_PASSWORD="${RESTIC_PASSWORD:-changeme}" \
            restic/restic:latest \
            init
    fi
    
    echo "Restic installed successfully"
    return 0
}

#######################################
# Uninstall restic
#######################################
restic::uninstall() {
    local keep_repository=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --keep-repository)
                keep_repository=true
                shift
                ;;
            --force)
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    echo "Uninstalling restic..."
    
    # Stop container if running
    restic::stop
    
    # Remove volumes
    if [[ "$keep_repository" == false ]]; then
        echo "Removing repository volume..."
        docker volume rm "$REPOSITORY_VOLUME" 2>/dev/null || true
    else
        echo "Keeping repository volume as requested"
    fi
    
    docker volume rm "$CACHE_VOLUME" 2>/dev/null || true
    docker volume rm "$CONFIG_VOLUME" 2>/dev/null || true
    
    echo "Restic uninstalled"
    return 0
}

#######################################
# Start restic service
#######################################
restic::start() {
    local wait_for_ready=false
    local timeout=60
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --wait)
                wait_for_ready=true
                shift
                ;;
            --timeout)
                timeout="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Check if already running
    if docker ps --format "table {{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        echo "Restic is already running"
        return 2
    fi
    
    echo "Starting restic service..."
    
    # Build the Docker image if it doesn't exist
    if ! docker images | grep -q "vrooli-restic-api"; then
        echo "Building restic API image..."
        docker build -t vrooli-restic-api "${RESOURCE_DIR}/api"
    fi
    
    # Run container with the API server
    docker run -d \
        --name "$CONTAINER_NAME" \
        --network vrooli-network \
        -p "${RESTIC_PORT}:8000" \
        -v "${REPOSITORY_VOLUME}:/repository" \
        -v "${CACHE_VOLUME}:/cache" \
        -v "${CONFIG_VOLUME}:/config" \
        -v /home:/backup/home:ro \
        -v /var/lib/docker/volumes:/backup/volumes:ro \
        -e RESTIC_REPOSITORY=/repository \
        -e RESTIC_PASSWORD="${RESTIC_PASSWORD:-changeme}" \
        -e RESTIC_CACHE_DIR=/cache \
        -e API_PORT=8000 \
        vrooli-restic-api
    
    if [[ "$wait_for_ready" == true ]]; then
        echo "Waiting for restic to be ready..."
        local elapsed=0
        while [[ $elapsed -lt $timeout ]]; do
            if restic::health_check; then
                echo "Restic is ready"
                return 0
            fi
            sleep 1
            ((elapsed++))
        done
        echo "Timeout waiting for restic to be ready"
        return 1
    fi
    
    echo "Restic started successfully"
    return 0
}

#######################################
# Stop restic service
#######################################
restic::stop() {
    local force=false
    local timeout=30
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --force)
                force=true
                shift
                ;;
            --timeout)
                timeout="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Check if running
    if ! docker ps --format "table {{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        echo "Restic is not running"
        return 2
    fi
    
    echo "Stopping restic service..."
    
    if [[ "$force" == true ]]; then
        docker kill "$CONTAINER_NAME" 2>/dev/null || true
    else
        docker stop --time="$timeout" "$CONTAINER_NAME" 2>/dev/null || true
    fi
    
    docker rm "$CONTAINER_NAME" 2>/dev/null || true
    
    echo "Restic stopped successfully"
    return 0
}

#######################################
# Run tests
#######################################
restic::test() {
    local test_type="${1:-all}"
    
    case "$test_type" in
        smoke)
            restic::test_smoke
            ;;
        integration)
            restic::test_integration
            ;;
        unit)
            restic::test_unit
            ;;
        all)
            restic::test_smoke
            restic::test_integration
            restic::test_unit
            ;;
        *)
            echo "Error: Unknown test type: $test_type"
            echo "Valid types: smoke, integration, unit, all"
            return 1
            ;;
    esac
}

#######################################
# Smoke test - quick health check
#######################################
restic::test_smoke() {
    echo "Running smoke test..."
    
    if restic::health_check; then
        echo "✓ Health check passed"
        return 0
    else
        echo "✗ Health check failed"
        return 1
    fi
}

#######################################
# Integration test
#######################################
restic::test_integration() {
    echo "Running integration test..."
    
    if [[ -f "${RESOURCE_DIR}/test/run-tests.sh" ]]; then
        "${RESOURCE_DIR}/test/run-tests.sh" integration
    else
        echo "Integration tests not yet implemented"
        return 0
    fi
}

#######################################
# Unit tests
#######################################
restic::test_unit() {
    echo "Running unit tests..."
    
    if [[ -f "${RESOURCE_DIR}/test/run-tests.sh" ]]; then
        "${RESOURCE_DIR}/test/run-tests.sh" unit
    else
        echo "Unit tests not yet implemented"
        return 0
    fi
}

#######################################
# Content management
#######################################
restic::content() {
    local subcommand="${1:-list}"
    shift || true
    
    case "$subcommand" in
        add)
            restic::content_add "$@"
            ;;
        list)
            restic::content_list "$@"
            ;;
        get)
            restic::content_get "$@"
            ;;
        remove)
            restic::content_remove "$@"
            ;;
        execute)
            restic::backup "$@"
            ;;
        *)
            echo "Error: Unknown content command: $subcommand"
            echo "Valid commands: add, list, get, remove, execute"
            return 1
            ;;
    esac
}

#######################################
# Add backup target
#######################################
restic::content_add() {
    local name=""
    local path=""
    local tags=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            --path)
                path="$2"
                shift 2
                ;;
            --tags)
                tags="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" || -z "$path" ]]; then
        echo "Error: --name and --path are required"
        echo "Usage: resource-restic content add --name <name> --path <path> [--tags <tags>]"
        return 1
    fi
    
    # Store backup target configuration
    local config_file="${CONFIG_VOLUME}/_data/targets.json"
    local targets_json="{}"
    
    # Read existing targets if file exists
    if docker exec "$CONTAINER_NAME" test -f /config/targets.json 2>/dev/null; then
        targets_json=$(docker exec "$CONTAINER_NAME" cat /config/targets.json 2>/dev/null || echo "{}")
    fi
    
    # Add new target
    local new_target="{\"path\":\"$path\",\"tags\":\"$tags\"}"
    targets_json=$(echo "$targets_json" | jq --arg name "$name" --argjson target "$new_target" '.[$name] = $target')
    
    # Save updated targets
    echo "$targets_json" | docker exec -i "$CONTAINER_NAME" sh -c 'cat > /config/targets.json'
    
    echo "Added backup target: $name -> $path"
    return 0
}

#######################################
# List backup targets
#######################################
restic::content_list() {
    local json_output=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --json)
                json_output=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Check if container is running
    if ! docker ps --format "table {{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        echo "Error: Restic container is not running"
        return 1
    fi
    
    # Read targets configuration
    local targets_json="{}"
    if docker exec "$CONTAINER_NAME" test -f /config/targets.json 2>/dev/null; then
        targets_json=$(docker exec "$CONTAINER_NAME" cat /config/targets.json 2>/dev/null || echo "{}")
    fi
    
    if [[ "$json_output" == true ]]; then
        echo "$targets_json"
    else
        echo "Backup Targets:"
        echo "$targets_json" | jq -r 'to_entries[] | "  \(.key): \(.value.path) (tags: \(.value.tags // "none"))"' 2>/dev/null || echo "  No targets configured"
    fi
    return 0
}

#######################################
# Get backup target details
#######################################
restic::content_get() {
    local name="${1:-}"
    local json_output=false
    
    shift || true
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --json)
                json_output=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        echo "Error: Target name required"
        echo "Usage: resource-restic content get <name> [--json]"
        return 1
    fi
    
    # Check if container is running
    if ! docker ps --format "table {{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        echo "Error: Restic container is not running"
        return 1
    fi
    
    # Read targets configuration
    local targets_json="{}"
    if docker exec "$CONTAINER_NAME" test -f /config/targets.json 2>/dev/null; then
        targets_json=$(docker exec "$CONTAINER_NAME" cat /config/targets.json 2>/dev/null || echo "{}")
    fi
    
    # Get specific target
    local target=$(echo "$targets_json" | jq --arg name "$name" '.[$name]' 2>/dev/null)
    
    if [[ "$target" == "null" || -z "$target" ]]; then
        echo "Error: Target '$name' not found"
        return 1
    fi
    
    if [[ "$json_output" == true ]]; then
        echo "$target"
    else
        echo "Target: $name"
        echo "$target" | jq -r '"  Path: \(.path)\n  Tags: \(.tags // "none")"' 2>/dev/null
    fi
    return 0
}

#######################################
# Remove backup target
#######################################
restic::content_remove() {
    local name="${1:-}"
    
    if [[ -z "$name" ]]; then
        echo "Error: Target name required"
        echo "Usage: resource-restic content remove <name>"
        return 1
    fi
    
    # Check if container is running
    if ! docker ps --format "table {{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        echo "Error: Restic container is not running"
        return 1
    fi
    
    # Read targets configuration
    local targets_json="{}"
    if docker exec "$CONTAINER_NAME" test -f /config/targets.json 2>/dev/null; then
        targets_json=$(docker exec "$CONTAINER_NAME" cat /config/targets.json 2>/dev/null || echo "{}")
    fi
    
    # Remove target
    targets_json=$(echo "$targets_json" | jq --arg name "$name" 'del(.[$name])')
    
    # Save updated targets
    echo "$targets_json" | docker exec -i "$CONTAINER_NAME" sh -c 'cat > /config/targets.json'
    
    echo "Removed backup target: $name"
    return 0
}

#######################################
# Show status
#######################################
restic::status() {
    local json_output=false
    local verbose=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --json)
                json_output=true
                shift
                ;;
            --verbose)
                verbose=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Check if container is running
    local status="stopped"
    if docker ps --format "table {{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        status="running"
    fi
    
    # Check health
    local health="unknown"
    if [[ "$status" == "running" ]]; then
        if restic::health_check; then
            health="healthy"
        else
            health="unhealthy"
        fi
    fi
    
    if [[ "$json_output" == true ]]; then
        echo "{\"status\":\"$status\",\"health\":\"$health\",\"port\":\"$RESTIC_PORT\"}"
    else
        echo "Restic Status:"
        echo "  Status: $status"
        echo "  Health: $health"
        echo "  Port: $RESTIC_PORT"
        
        if [[ "$verbose" == true && "$status" == "running" ]]; then
            echo ""
            echo "Container Details:"
            docker ps --filter "name=${CONTAINER_NAME}" --format "table {{.ID}}\t{{.Status}}\t{{.Ports}}"
        fi
    fi
}

#######################################
# View logs
#######################################
restic::logs() {
    if docker ps -a --format "table {{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        docker logs "$CONTAINER_NAME" "$@"
    else
        echo "Container not found"
        return 1
    fi
}

#######################################
# Health check
#######################################
restic::health_check() {
    timeout 5 curl -sf "http://localhost:${RESTIC_PORT}/health" >/dev/null 2>&1
}

#######################################
# Create backup
#######################################
restic::backup() {
    echo "Creating backup..."
    
    local paths=()
    local tags=()
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --paths)
                IFS=',' read -ra paths <<< "$2"
                shift 2
                ;;
            --tags)
                IFS=',' read -ra tags <<< "$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ ${#paths[@]} -eq 0 ]]; then
        # Default to backing up Vrooli data
        paths=("/backup/home/matthalloran8/Vrooli" "/backup/volumes")
    fi
    
    echo "Backing up: ${paths[*]}"
    
    # Create JSON payload
    local json_paths=$(printf '"%s",' "${paths[@]}" | sed 's/,$//')
    local json_tags=$(printf '"%s",' "${tags[@]}" | sed 's/,$//')
    local payload="{\"paths\":[$json_paths]"
    if [[ -n "$json_tags" ]]; then
        payload+=",\"tags\":[$json_tags]"
    fi
    payload+="}"
    
    # Call API endpoint
    response=$(curl -sf -X POST \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "http://localhost:${RESTIC_PORT}/backup" 2>&1)
    
    if [[ $? -eq 0 ]]; then
        echo "Backup completed successfully"
        echo "$response" | jq -r '.snapshot // "Backup completed"' 2>/dev/null || echo "$response"
        return 0
    else
        echo "Backup failed: $response"
        return 1
    fi
}

#######################################
# List snapshots
#######################################
restic::snapshots() {
    echo "Listing snapshots..."
    
    response=$(curl -sf "http://localhost:${RESTIC_PORT}/snapshots" 2>&1)
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | jq -r '.[] | "\(.id[0:8]) \(.time) \(.hostname) \(.paths | join(","))"' 2>/dev/null || echo "$response"
        return 0
    else
        echo "Failed to list snapshots: $response"
        return 1
    fi
}

#######################################
# Restore from backup
#######################################
restic::restore() {
    echo "Restoring from backup..."
    
    local snapshot="latest"
    local target="/restore"
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --snapshot)
                snapshot="$2"
                shift 2
                ;;
            --target)
                target="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    echo "Restoring snapshot '$snapshot' to '$target'"
    
    # Execute restore in container
    docker exec "$CONTAINER_NAME" restic restore "$snapshot" --target "$target"
    
    if [[ $? -eq 0 ]]; then
        echo "Restore completed successfully"
        return 0
    else
        echo "Restore failed"
        return 1
    fi
}


#######################################
# Verify backup integrity
#######################################
restic::verify() {
    echo "Verifying backup integrity..."
    
    local snapshot="${1:-latest}"
    local read_data=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --snapshot)
                snapshot="$2"
                shift 2
                ;;
            --read-data)
                read_data=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    echo "Checking snapshot: $snapshot"
    
    # Run check command in container
    local check_args="check"
    if [[ "$read_data" == true ]]; then
        check_args="$check_args --read-data"
    fi
    
    if docker exec "$CONTAINER_NAME" restic $check_args 2>&1; then
        echo "✓ Repository integrity verified"
        
        # Also verify specific snapshot if not checking all
        if [[ "$snapshot" != "all" ]]; then
            if docker exec "$CONTAINER_NAME" restic check --read-data-subset="$snapshot" 2>&1; then
                echo "✓ Snapshot $snapshot verified"
                return 0
            else
                echo "✗ Snapshot $snapshot has issues"
                return 1
            fi
        fi
        return 0
    else
        echo "✗ Repository integrity check failed"
        return 1
    fi
}

#######################################
# Prune old snapshots
#######################################
restic::prune() {
    echo "Pruning old snapshots..."
    
    local dry_run=""
    local keep_daily=7
    local keep_weekly=4
    local keep_monthly=12
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dry-run)
                dry_run="--dry-run"
                shift
                ;;
            --keep-daily)
                keep_daily="$2"
                shift 2
                ;;
            --keep-weekly)
                keep_weekly="$2"
                shift 2
                ;;
            --keep-monthly)
                keep_monthly="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -n "$dry_run" ]]; then
        echo "Dry run - no snapshots will be removed"
    fi
    
    echo "Retention policy: daily=$keep_daily, weekly=$keep_weekly, monthly=$keep_monthly"
    
    # Execute prune in container
    docker exec "$CONTAINER_NAME" restic forget \
        --keep-daily "$keep_daily" \
        --keep-weekly "$keep_weekly" \
        --keep-monthly "$keep_monthly" \
        --prune \
        $dry_run
    
    if [[ $? -eq 0 ]]; then
        echo "Prune completed successfully"
        return 0
    else
        echo "Prune failed"
        return 1
    fi
}