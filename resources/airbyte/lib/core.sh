#!/bin/bash
# Airbyte Core Library Functions

set -euo pipefail

# Resource metadata
export RESOURCE_NAME="airbyte"
RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONFIG_DIR="${RESOURCE_DIR}/config"
DATA_DIR="${RESOURCE_DIR}/data"

# Load configuration
source "${CONFIG_DIR}/defaults.sh"

# Logging functions
# shellcheck disable=SC2317  # Functions are used when file is sourced
log_info() {
    echo "[INFO] $*" >&2
}

# shellcheck disable=SC2317  # Functions are used when file is sourced
log_error() {
    echo "[ERROR] $*" >&2
}

log_debug() {
    if [[ "${DEBUG:-false}" == "true" ]]; then
        echo "[DEBUG] $*" >&2
    fi
}

# Command: info
cmd_info() {
    local format="${1:---text}"
    
    if [[ "$format" == "--json" ]]; then
        cat "${CONFIG_DIR}/runtime.json"
    else
        echo "Airbyte Resource Information:"
        echo "  Startup Order: 550"
        echo "  Dependencies: postgres"
        echo "  Startup Time: 30-60s"
        echo "  Startup Timeout: 120s"
        echo "  Recovery Attempts: 3"
        echo "  Priority: high"
        echo ""
        echo "Ports:"
        echo "  Webapp: ${AIRBYTE_WEBAPP_PORT}"
        echo "  API Server: ${AIRBYTE_SERVER_PORT}"
        echo "  Temporal: ${AIRBYTE_TEMPORAL_PORT}"
    fi
}

# Command: manage
cmd_manage() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        install)
            manage_install "$@"
            ;;
        start)
            manage_start "$@"
            ;;
        stop)
            manage_stop "$@"
            ;;
        restart)
            manage_restart "$@"
            ;;
        uninstall)
            manage_uninstall "$@"
            ;;
        *)
            echo "Error: Unknown manage subcommand: $subcommand" >&2
            exit 1
            ;;
    esac
}

# Detect deployment method
detect_deployment_method() {
    # Check if abctl is installed or should be used
    if [[ -f "${DATA_DIR}/abctl" ]] || [[ "${AIRBYTE_USE_ABCTL:-false}" == "true" ]]; then
        echo "abctl"
    else
        echo "docker-compose"
    fi
}

# Install abctl CLI
install_abctl() {
    log_info "Installing abctl CLI..."
    
    # Detect OS and architecture
    local os arch
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    arch=$(uname -m)
    
    # Map architecture names
    case "$arch" in
        x86_64)
            arch="amd64"
            ;;
        aarch64|arm64)
            arch="arm64"
            ;;
        *)
            log_error "Unsupported architecture: $arch"
            exit 1
            ;;
    esac
    
    # Download abctl
    local abctl_version
    abctl_version=$(curl -s https://api.github.com/repos/airbytehq/abctl/releases/latest | jq -r '.tag_name')
    local download_url="https://github.com/airbytehq/abctl/releases/download/${abctl_version}/abctl-${abctl_version}-${os}-${arch}.tar.gz"
    
    log_info "Downloading abctl ${abctl_version} from $download_url..."
    curl -L "$download_url" -o "${DATA_DIR}/abctl.tar.gz"
    tar -xzf "${DATA_DIR}/abctl.tar.gz" -C "${DATA_DIR}"
    # Move the binary to the expected location
    local extracted_dir="abctl-${abctl_version}-${os}-${arch}"
    if [[ -d "${DATA_DIR}/${extracted_dir}" ]]; then
        mv "${DATA_DIR}/${extracted_dir}/abctl" "${DATA_DIR}/abctl"
        rm -rf "${DATA_DIR:?}/${extracted_dir}"
    fi
    chmod +x "${DATA_DIR}/abctl"
    rm -f "${DATA_DIR}/abctl.tar.gz"
    
    # Add to PATH for this session
    export PATH="${DATA_DIR}:${PATH}"
    
    # Verify installation
    if ! "${DATA_DIR}/abctl" version &> /dev/null; then
        log_error "abctl installation failed"
        return 1
    fi
    
    log_info "abctl installed successfully"
}

# Install Airbyte using abctl
install_with_abctl() {
    log_info "Installing Airbyte with abctl..."
    
    # Install abctl if not present
    if [[ ! -f "${DATA_DIR}/abctl" ]]; then
        install_abctl
    fi
    
    # Set abctl path
    local ABCTL="${DATA_DIR}/abctl"
    
    # Performance optimization: Pre-pull images if docker is available
    if command -v docker &> /dev/null; then
        log_info "Pre-pulling Airbyte images for faster installation..."
        # Pull images in parallel for speed
        {
            docker pull airbyte/webapp:latest &
            docker pull airbyte/server:latest &
            docker pull airbyte/worker:latest &
            docker pull kindest/node:v1.27.3 &
        } 2>/dev/null
        wait # Wait for all pulls to complete
    fi
    
    # Install with abctl (simplified - abctl handles its own configuration)
    log_info "Running abctl installation (this may take 10-30 minutes on first run, 2-5 minutes on subsequent runs)..."
    log_info "Note: abctl will automatically install kind (Kubernetes in Docker) if needed"
    
    # Use default installation with port mapping and additional optimizations
    "${DATA_DIR}/abctl" local install \
        --port "${AIRBYTE_WEBAPP_PORT}" \
        --low-resource-mode \
        --insecure-cookies \
        --host localhost
    
    log_info "Airbyte installed successfully with abctl"
}

# Install Airbyte using docker-compose (legacy)
install_with_docker_compose() {
    log_error "Docker Compose deployment is no longer supported for Airbyte v1.x+"
    log_info ""
    log_info "Airbyte deprecated docker-compose in August 2024. Please use abctl instead:"
    log_info ""
    log_info "  export AIRBYTE_USE_ABCTL=true"
    log_info "  vrooli resource airbyte manage install"
    log_info ""
    log_info "The abctl installation will:"
    log_info "  - Install Kubernetes-in-Docker (kind)"
    log_info "  - Deploy Airbyte using Helm charts"
    log_info "  - Configure all necessary services"
    log_info ""
    log_info "Note: First installation may take 10-30 minutes"
    
    # Automatically switch to abctl
    export AIRBYTE_USE_ABCTL=true
    log_info "Switching to abctl installation..."
    install_with_abctl
}

# Install Airbyte (main entry point)
manage_install() {
    log_info "Installing Airbyte..."
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker is required but not installed"
        exit 1
    fi
    
    # Ensure data directory exists
    mkdir -p "${DATA_DIR}"
    
    # Detect and use appropriate method
    local method
    method=$(detect_deployment_method)
    
    # Check if already installed (for abctl) to speed up repeated installs
    if [[ "$method" == "abctl" ]] && [[ -f "${DATA_DIR}/abctl" ]]; then
        local status
        status=$("${DATA_DIR}/abctl" local status 2>&1)
        if ! echo "$status" | grep -q "does not appear to be installed"; then
            log_info "Airbyte is already installed and running. Skipping installation."
            log_info "Use 'manage stop' and 'manage start' to restart if needed."
            return 0
        fi
    fi
    
    log_info "Using deployment method: $method"
    
    if [[ "$method" == "abctl" ]]; then
        install_with_abctl
    else
        install_with_docker_compose
    fi
}

# Start Airbyte
manage_start() {
    local wait_flag="${1:---no-wait}"
    local method
    method=$(detect_deployment_method)
    
    log_info "Starting Airbyte services (method: $method)..."
    
    if [[ "$method" == "abctl" ]]; then
        # Check if abctl is installed
        local ABCTL="${DATA_DIR}/abctl"
        if [[ ! -f "$ABCTL" ]]; then
            log_error "abctl not found. Run 'manage install' first"
            exit 1
        fi
        
        # Check if Airbyte is installed
        local status
        status=$("${DATA_DIR}/abctl" local status 2>&1)
        if echo "$status" | grep -q "does not appear to be installed"; then
            log_info "Airbyte is not installed. Installing first..."
            install_with_abctl
        else
            log_info "Airbyte is already installed and running"
        fi
        
        if [[ "$wait_flag" == "--wait" ]]; then
            log_info "Waiting for services to be healthy..."
            local max_attempts=60  # More time for Kubernetes
            local attempt=0
            
            while [ $attempt -lt $max_attempts ]; do
                if check_health; then
                    log_info "Airbyte is ready!"
                    return 0
                fi
                
                attempt=$((attempt + 1))
                log_debug "Health check attempt $attempt/$max_attempts"
                sleep 5
            done
            
            log_error "Airbyte failed to start within timeout"
            return 1
        fi
    else
        # Legacy docker-compose method
        cd "${RESOURCE_DIR}"
        docker compose up -d
        
        if [[ "$wait_flag" == "--wait" ]]; then
            log_info "Waiting for services to be healthy..."
            local max_attempts=30
            local attempt=0
            
            while [ $attempt -lt $max_attempts ]; do
                if check_health; then
                    log_info "Airbyte is ready!"
                    return 0
                fi
                
                attempt=$((attempt + 1))
                log_debug "Health check attempt $attempt/$max_attempts"
                sleep 5
            done
            
            log_error "Airbyte failed to start within timeout"
            return 1
        fi
    fi
}

# Stop Airbyte
manage_stop() {
    local method
    method=$(detect_deployment_method)
    
    log_info "Stopping Airbyte services (method: $method)..."
    
    # Stop port forwarding if running
    stop_api_port_forward
    
    if [[ "$method" == "abctl" ]]; then
        local ABCTL="${DATA_DIR}/abctl"
        if [[ -f "$ABCTL" ]]; then
            # abctl doesn't have a "down" command, uninstall stops and removes the installation
            log_info "Note: abctl doesn't support stopping without uninstalling. Services remain running."
            log_info "To fully stop, use 'manage uninstall --keep-data'"
        fi
    else
        if [[ -f "${RESOURCE_DIR}/docker-compose.yml" ]]; then
            cd "${RESOURCE_DIR}"
            docker compose down
        fi
    fi
    
    log_info "Airbyte stopped"
}

# Restart Airbyte
manage_restart() {
    manage_stop
    sleep 2
    manage_start "$@"
}

# Uninstall Airbyte
manage_uninstall() {
    local keep_data="${1:---remove-data}"
    local method
    method=$(detect_deployment_method)
    
    log_info "Uninstalling Airbyte (method: $method)..."
    
    # Stop services
    manage_stop
    
    if [[ "$method" == "abctl" ]]; then
        local ABCTL="${DATA_DIR}/abctl"
        if [[ -f "$ABCTL" ]]; then
            "${DATA_DIR}/abctl" local uninstall
            # Remove abctl binary from data dir
            rm -f "$ABCTL"
        fi
    else
        if [[ -f "${RESOURCE_DIR}/docker-compose.yml" ]]; then
            # Remove containers and images
            cd "${RESOURCE_DIR}"
            docker compose down --rmi all --volumes
            # Remove docker-compose file
            rm -f "${RESOURCE_DIR}/docker-compose.yml"
        fi
    fi
    
    # Remove data if requested
    if [[ "$keep_data" != "--keep-data" ]]; then
        log_info "Removing data directory..."
        rm -rf "${DATA_DIR}"
    fi
    
    log_info "Airbyte uninstalled"
}

# Make API call through kubectl exec (abctl deployment)
# This is used instead of port-forwarding for better reliability
api_call() {
    local method="${1}"
    local endpoint="${2}"
    local data="${3:-}"
    local deployment_method
    deployment_method=$(detect_deployment_method)
    
    if [[ "$deployment_method" == "abctl" ]]; then
        # Use kubectl exec to make API call directly on the server pod
        if [[ -n "$data" ]]; then
            docker exec airbyte-abctl-control-plane kubectl -n airbyte-abctl exec deploy/airbyte-abctl-server -- \
                curl -s -X "${method}" -H "Content-Type: application/json" -d "${data}" \
                "http://localhost:8001/api/public/v1/${endpoint}"
        else
            docker exec airbyte-abctl-control-plane kubectl -n airbyte-abctl exec deploy/airbyte-abctl-server -- \
                curl -s "http://localhost:8001/api/public/v1/${endpoint}"
        fi
    else
        # For docker-compose deployment, use direct curl
        if [[ -n "$data" ]]; then
            curl -sf -X "${method}" \
                -H "Content-Type: application/json" \
                -d "${data}" \
                "http://localhost:${AIRBYTE_SERVER_PORT}/api/v1/${endpoint}"
        else
            curl -sf "http://localhost:${AIRBYTE_SERVER_PORT}/api/v1/${endpoint}"
        fi
    fi
}

# Setup port forwarding for API access (abctl deployment) - simplified version
setup_api_port_forward() {
    # This function is kept for backward compatibility but is now a no-op
    # API calls are made directly through kubectl exec
    return 0
}

# Stop port forwarding
stop_api_port_forward() {
    if [[ -f /tmp/airbyte-port-forward.pid ]]; then
        local pid
        pid=$(cat /tmp/airbyte-port-forward.pid)
        if ps -p "$pid" > /dev/null 2>&1; then
            kill "$pid" 2>/dev/null || true
            log_debug "Stopped API port forward (PID: $pid)"
        fi
        rm -f /tmp/airbyte-port-forward.pid
    fi
}

# Check health with detailed status
check_health() {
    local verbose="${1:-false}"
    local method
    method=$(detect_deployment_method)
    
    # Health check based on deployment method
    if [[ "$method" == "abctl" ]]; then
        # For abctl, check if cluster and pods are running
        if ! "${DATA_DIR}/abctl" local status &> /dev/null; then
            [[ "$verbose" == "true" ]] && log_error "Airbyte cluster not running"
            return 1
        fi
        # Check pod health - all non-completed pods should be running
        local unhealthy_pods
        unhealthy_pods=$(docker exec airbyte-abctl-control-plane sh -c 'kubectl -n airbyte-abctl get pods --no-headers 2>/dev/null' | grep -v Completed | grep -vc Running)
        # Remove whitespace and leading zeros to avoid octal interpretation
        unhealthy_pods=$(echo "$unhealthy_pods" | sed 's/^0*//' | tr -d '[:space:]')
        [[ -z "$unhealthy_pods" ]] && unhealthy_pods="0"
        if [[ "$unhealthy_pods" -gt 0 ]]; then
            [[ "$verbose" == "true" ]] && log_error "Some pods are not healthy"
            return 1
        fi
        # If we get here, consider it healthy for abctl
        return 0
    else
        # For docker-compose, use traditional health check
        if ! timeout 5 api_call GET "health" > /dev/null 2>&1; then
            [[ "$verbose" == "true" ]] && log_error "API health check failed"
            return 1
        fi
    fi
    
    # Check critical services if verbose
    if [[ "$verbose" == "true" ]]; then
        local services=("server" "worker" "temporal" "db")
        local failed=0
        
        for service in "${services[@]}"; do
            container="airbyte-${service}"
            [[ "$service" == "db" ]] && container="airbyte-db"
            
            if ! docker ps --format "{{.Names}}" | grep -q "^${container}$"; then
                log_error "Service ${service} is not running"
                failed=$((failed + 1))
            fi
        done
        
        if [[ $failed -gt 0 ]]; then
            return 1
        fi
    fi
    
    return 0
}

# Check sync status
check_sync_status() {
    local connection_id="${1:-}"
    
    if [[ -z "$connection_id" ]]; then
        # Get overall sync health
        local result
        if ! result=$(api_call GET "jobs/list" 2>/dev/null); then
            log_error "Failed to retrieve job list"
            return 1
        fi
        
        # Check for failed jobs in last hour
        local failed_count
        failed_count=$(echo "$result" | jq '[.jobs[] | select(.status == "failed" and (.updatedAt | fromdateiso8601) > (now - 3600))] | length' 2>/dev/null || echo "0")
        
        if [[ "$failed_count" -gt 0 ]]; then
            log_error "Found $failed_count failed sync jobs in the last hour"
            return 1
        fi
    else
        # Check specific connection
        local result
        if ! result=$(curl -sf -X POST \
            -H "Content-Type: application/json" \
            -d "{\"connectionId\":\"$connection_id\"}" \
            "http://localhost:${AIRBYTE_SERVER_PORT}/api/v1/connections/get" 2>/dev/null); then
            log_error "Failed to retrieve connection status"
            return 1
        fi
        
        local status
        status=$(echo "$result" | jq -r '.status' 2>/dev/null)
        if [[ "$status" != "active" ]]; then
            log_error "Connection $connection_id is not active (status: $status)"
            return 1
        fi
    fi
    
    return 0
}

# Command: test
cmd_test() {
    local test_type="${1:-all}"
    shift || true
    
    source "${RESOURCE_DIR}/lib/test.sh"
    
    case "$test_type" in
        smoke)
            test_smoke "$@"
            ;;
        integration)
            test_integration "$@"
            ;;
        unit)
            test_unit "$@"
            ;;
        all)
            test_all "$@"
            ;;
        *)
            echo "Error: Unknown test type: $test_type" >&2
            exit 1
            ;;
    esac
}

# Command: content
cmd_content() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        list)
            content_list "$@"
            ;;
        add)
            content_add "$@"
            ;;
        get)
            content_get "$@"
            ;;
        remove)
            content_remove "$@"
            ;;
        execute)
            content_execute "$@"
            ;;
        *)
            echo "Error: Unknown content subcommand: $subcommand" >&2
            exit 1
            ;;
    esac
}

# List content (sources, destinations, connections)
content_list() {
    local type="sources"
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --type)
                type="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Setup port forwarding if needed
    setup_api_port_forward
    
    local endpoint
    case "$type" in
        sources)
            endpoint="source_definitions/list"
            ;;
        destinations)
            endpoint="destination_definitions/list"
            ;;
        connections)
            endpoint="connections/list"
            ;;
        *)
            log_error "Unknown type: $type"
            exit 1
            ;;
    esac
    
    api_call GET "${endpoint}" | jq '.'
}

# Add content
content_add() {
    local type=""
    local config_file=""
    local credential=""
    
    # Setup port forwarding if needed
    setup_api_port_forward
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --type)
                type="$2"
                shift 2
                ;;
            --config)
                config_file="$2"
                shift 2
                ;;
            --credential)
                credential="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$type" || -z "$config_file" ]]; then
        log_error "Both --type and --config are required"
        exit 1
    fi
    
    if [[ ! -f "$config_file" ]]; then
        log_error "Config file not found: $config_file"
        exit 1
    fi
    
    local config_data
    config_data=$(cat "$config_file")
    
    # Apply credential if specified
    if [[ -n "$credential" ]]; then
        if ! config_data=$(apply_credential "$config_data" "$credential"); then
            log_error "Failed to apply credential: $credential"
            exit 1
        fi
    fi
    
    local endpoint
    case "$type" in
        source)
            endpoint="sources/create"
            ;;
        destination)
            endpoint="destinations/create"
            ;;
        connection)
            endpoint="connections/create"
            ;;
        *)
            log_error "Unknown type: $type"
            exit 1
            ;;
    esac
    
    echo "$config_data" | curl -X POST \
        -H "Content-Type: application/json" \
        -d @- \
        "http://localhost:${AIRBYTE_SERVER_PORT}/api/v1/${endpoint}" | jq '.'
}

# Get content details
content_get() {
    local id=""
    
    # Setup port forwarding if needed
    setup_api_port_forward
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --id)
                id="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$id" ]]; then
        log_error "--id is required"
        exit 1
    fi
    
    # Try different endpoints to find the resource
    for endpoint in "sources/get" "destinations/get" "connections/get"; do
        if result=$(curl -sf -X POST \
            -H "Content-Type: application/json" \
            -d "{\"sourceId\":\"$id\"}" \
            "http://localhost:${AIRBYTE_SERVER_PORT}/api/v1/${endpoint}" 2>/dev/null); then
            echo "$result" | jq '.'
            return 0
        fi
    done
    
    log_error "Resource not found: $id"
    exit 1
}

# Remove content
content_remove() {
    local id=""
    
    # Setup port forwarding if needed
    setup_api_port_forward
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --id)
                id="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$id" ]]; then
        log_error "--id is required"
        exit 1
    fi
    
    # Try different endpoints to delete the resource
    for endpoint in "sources/delete" "destinations/delete" "connections/delete"; do
        if result=$(curl -sf -X POST \
            -H "Content-Type: application/json" \
            -d "{\"sourceId\":\"$id\"}" \
            "http://localhost:${AIRBYTE_SERVER_PORT}/api/v1/${endpoint}" 2>/dev/null); then
            echo "Resource deleted: $id"
            return 0
        fi
    done
    
    log_error "Failed to delete resource: $id"
    exit 1
}

# Execute sync with retry logic
content_execute() {
    local connection_id=""
    local max_retries=3
    local retry_delay=5
    local wait_for_completion="false"
    
    # Setup port forwarding if needed
    setup_api_port_forward
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --connection-id)
                connection_id="$2"
                shift 2
                ;;
            --max-retries)
                max_retries="$2"
                shift 2
                ;;
            --wait)
                wait_for_completion="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$connection_id" ]]; then
        log_error "--connection-id is required"
        exit 1
    fi
    
    # Trigger sync with retry logic
    local attempt=0
    local job_id=""
    
    while [[ $attempt -lt $max_retries ]]; do
        attempt=$((attempt + 1))
        log_info "Triggering sync for connection $connection_id (attempt $attempt/$max_retries)..."
        
        local result
        if result=$(curl -sf -X POST \
            -H "Content-Type: application/json" \
            -d "{\"connectionId\":\"$connection_id\"}" \
            "http://localhost:${AIRBYTE_SERVER_PORT}/api/v1/connections/sync" 2>/dev/null); then
            job_id=$(echo "$result" | jq -r '.job.id' 2>/dev/null)
            if [[ -n "$job_id" && "$job_id" != "null" ]]; then
                echo "$result" | jq '.'
                log_info "Sync job started successfully: $job_id"
                break
            fi
        fi
        
        if [[ $attempt -lt $max_retries ]]; then
            log_error "Failed to start sync, retrying in ${retry_delay} seconds..."
            sleep "$retry_delay"
            retry_delay=$((retry_delay * 2))  # Exponential backoff
        else
            log_error "Failed to start sync after $max_retries attempts"
            exit 1
        fi
    done
    
    # Wait for completion if requested
    if [[ "$wait_for_completion" == "true" && -n "$job_id" ]]; then
        monitor_sync_job "$job_id"
    fi
}

# Monitor sync job status
monitor_sync_job() {
    local job_id="$1"
    local timeout="${2:-3600}"  # Default 1 hour timeout
    local check_interval=10
    local elapsed=0
    
    log_info "Monitoring sync job $job_id..."
    
    while [[ $elapsed -lt $timeout ]]; do
        local result
        if ! result=$(curl -sf -X POST \
            -H "Content-Type: application/json" \
            -d "{\"id\":\"$job_id\"}" \
            "http://localhost:${AIRBYTE_SERVER_PORT}/api/v1/jobs/get" 2>/dev/null); then
            log_error "Failed to get job status"
            return 1
        fi
        
        local status
        status=$(echo "$result" | jq -r '.job.status' 2>/dev/null)
        
        case "$status" in
            "succeeded")
                log_info "Sync job completed successfully"
                echo "$result" | jq '.job'
                return 0
                ;;
            "failed")
                log_error "Sync job failed"
                echo "$result" | jq '.job'
                return 1
                ;;
            "cancelled")
                log_error "Sync job was cancelled"
                return 1
                ;;
            "running"|"pending")
                # Still in progress
                ;;
            *)
                log_error "Unknown job status: $status"
                return 1
                ;;
        esac
        
        sleep "$check_interval"
        elapsed=$((elapsed + check_interval))
        
        # Log progress every minute
        if [[ $((elapsed % 60)) -eq 0 ]]; then
            log_info "Job still running... (${elapsed}s elapsed)"
        fi
    done
    
    log_error "Sync job timed out after ${timeout} seconds"
    return 1
}

# Source credential management library
source "${RESOURCE_DIR}/lib/credentials.sh"

# Source schedule management library
source "${RESOURCE_DIR}/lib/schedule.sh"

# Source webhook notification library
source "${RESOURCE_DIR}/lib/webhooks.sh"

# Source transformation support library
source "${RESOURCE_DIR}/lib/transformations.sh"

# Source pipeline optimization library
source "${RESOURCE_DIR}/lib/pipeline.sh"

# Command: pipeline
cmd_pipeline() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        performance)
            local connection_id=""
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --connection-id)
                        connection_id="$2"
                        shift 2
                        ;;
                    *)
                        shift
                        ;;
                esac
            done
            monitor_sync_performance "$connection_id"
            ;;
        optimize)
            local connection_id=""
            local batch_size="10000"
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --connection-id)
                        connection_id="$2"
                        shift 2
                        ;;
                    --batch-size)
                        batch_size="$2"
                        shift 2
                        ;;
                    *)
                        shift
                        ;;
                esac
            done
            optimize_sync_config "$connection_id" "$batch_size"
            ;;
        quality)
            local connection_id=""
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --connection-id)
                        connection_id="$2"
                        shift 2
                        ;;
                    *)
                        shift
                        ;;
                esac
            done
            analyze_data_quality "$connection_id"
            ;;
        batch)
            local connections_file=""
            local parallel="false"
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --file)
                        connections_file="$2"
                        shift 2
                        ;;
                    --parallel)
                        parallel="true"
                        shift
                        ;;
                    *)
                        shift
                        ;;
                esac
            done
            batch_sync_orchestrator "$connections_file" "$parallel"
            ;;
        resources)
            analyze_resource_usage "$@"
            ;;
        *)
            echo "Error: Unknown pipeline subcommand: $subcommand" >&2
            echo "Available subcommands: performance, optimize, quality, batch, resources" >&2
            exit 1
            ;;
    esac
}

# Command: status
cmd_status() {
    local format="${1:---text}"
    local verbose="${2:---verbose}"
    local method
    method=$(detect_deployment_method)
    
    if [[ "$format" == "--json" ]]; then
        status_json
    else
        echo "Airbyte Service Status:"
        echo "  Deployment Method: $method"
        echo ""
        
        if [[ "$method" == "abctl" ]]; then
            # For abctl, use its status command
            local ABCTL="${DATA_DIR}/abctl"
            if [[ -f "$ABCTL" ]]; then
                # Check if abctl cluster is running (no need to capture output)
                if "${DATA_DIR}/abctl" local status &>/dev/null; then
                    echo "  Cluster: running ✓"
                    echo ""
                    echo "Health Check:"
                    if check_health true; then
                        echo "  API: healthy ✓"
                        echo "  Webapp: accessible on port ${AIRBYTE_WEBAPP_PORT}"
                    else
                        echo "  API: unhealthy ✗"
                    fi
                else
                    echo "  Cluster: stopped ✗"
                fi
            else
                echo "  abctl: not installed ✗"
            fi
        else
            # Legacy docker-compose status
            local services_running=0
            local services_total=5  # Updated count without scheduler
            
            for service in webapp server worker temporal db; do
                container_name="airbyte-${service}"
                if [[ "$service" == "db" ]]; then
                    container_name="airbyte-db"
                fi
                
                if docker ps --format "table {{.Names}}" | grep -q "^${container_name}$"; then
                    echo "  ${service}: running ✓"
                    services_running=$((services_running + 1))
                else
                    echo "  ${service}: stopped ✗"
                fi
            done
            
            echo ""
            echo "Health Check:"
            if check_health true; then
                echo "  API: healthy ✓"
            else
                echo "  API: unhealthy ✗"
            fi
            
            echo ""
            echo "Summary: ${services_running}/${services_total} services running"
        fi
        
        # Show sync status if services are running and verbose
        if [[ "$verbose" == "--verbose" ]] && check_health; then
            echo ""
            echo "Sync Status:"
            show_sync_status
        fi
    fi
}

# Show sync status summary
show_sync_status() {
    local result
    if ! result=$(curl -sf "http://localhost:${AIRBYTE_SERVER_PORT}/api/v1/jobs/list" 2>/dev/null); then
        echo "  Unable to retrieve sync status"
        return
    fi
    
    # Count jobs by status
    local running
    running=$(echo "$result" | jq '[.jobs[] | select(.status == "running")] | length' 2>/dev/null || echo "0")
    local succeeded
    succeeded=$(echo "$result" | jq '[.jobs[] | select(.status == "succeeded" and (.updatedAt | fromdateiso8601) > (now - 3600))] | length' 2>/dev/null || echo "0")
    local failed
    failed=$(echo "$result" | jq '[.jobs[] | select(.status == "failed" and (.updatedAt | fromdateiso8601) > (now - 3600))] | length' 2>/dev/null || echo "0")
    
    echo "  Running syncs: $running"
    echo "  Successful (last hour): $succeeded"
    echo "  Failed (last hour): $failed"
    
    if [[ "$failed" -gt 0 ]]; then
        echo "  ⚠️  Warning: Recent sync failures detected"
    fi
}

# Status in JSON format
status_json() {
    local services_status="{}"
    
    for service in webapp server worker temporal db; do
        container_name="airbyte-${service}"
        if [[ "$service" == "db" ]]; then
            container_name="airbyte-db"
        fi
        
        if docker ps --format "table {{.Names}}" | grep -q "^${container_name}$"; then
            services_status=$(echo "$services_status" | jq ". + {\"$service\": \"running\"}")
        else
            services_status=$(echo "$services_status" | jq ". + {\"$service\": \"stopped\"}")
        fi
    done
    
    local health="unhealthy"
    check_health && health="healthy"
    
    echo "{
        \"services\": $services_status,
        \"health\": \"$health\",
        \"ports\": {
            \"webapp\": ${AIRBYTE_WEBAPP_PORT},
            \"server\": ${AIRBYTE_SERVER_PORT},
            \"temporal\": ${AIRBYTE_TEMPORAL_PORT}
        }
    }" | jq '.'
}

# Command: logs
cmd_logs() {
    local service="server"
    local follow=""
    local method
    method=$(detect_deployment_method)
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --service)
                service="$2"
                shift 2
                ;;
            -f|--follow)
                follow="-f"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ "$method" == "abctl" ]]; then
        # For abctl, use its logs command
        if command -v abctl &> /dev/null; then
            if [[ -n "$follow" ]]; then
                abctl local logs --follow
            else
                abctl local logs
            fi
        else
            log_error "abctl not found"
            exit 1
        fi
    else
        # Legacy docker-compose method
        container_name="airbyte-${service}"
        if [[ "$service" == "db" ]]; then
            container_name="airbyte-db"
        fi
        
        docker logs $follow "$container_name"
    fi
}

# Command: credentials
cmd_credentials() {
    local subcommand="${1:-show}"
    shift || true
    
    case "$subcommand" in
        show)
            echo "Airbyte API Credentials:"
            echo ""
            echo "  API URL: http://localhost:${AIRBYTE_SERVER_PORT}/api/v1/"
            echo "  Webapp URL: http://localhost:${AIRBYTE_WEBAPP_PORT}"
            echo ""
            echo "Default credentials (first-time setup):"
            echo "  Username: airbyte"
            echo "  Password: password"
            echo ""
            echo "Note: Change default credentials after first login"
            ;;
        store)
            credential_store "$@"
            ;;
        get)
            credential_get "$@"
            ;;
        list)
            credential_list "$@"
            ;;
        remove)
            credential_remove "$@"
            ;;
        rotate)
            credential_rotate "$@"
            ;;
        *)
            echo "Error: Unknown credentials subcommand: $subcommand" >&2
            echo "Usage: credentials [show|store|get|list|remove|rotate]" >&2
            exit 1
            ;;
    esac
}

# Store a credential securely
credential_store() {
    local name=""
    local type=""
    local file=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            --type)
                type="$2"
                shift 2
                ;;
            --file)
                file="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" || -z "$type" || -z "$file" ]]; then
        log_error "Usage: credentials store --name NAME --type TYPE --file FILE"
        exit 1
    fi
    
    if [[ ! -f "$file" ]]; then
        log_error "File not found: $file"
        exit 1
    fi
    
    local data
    data=$(cat "$file")
    
    # Validate credential format
    if ! validate_credential "$data" "$type"; then
        log_error "Invalid credential format for type: $type"
        exit 1
    fi
    
    store_credential "$name" "$type" "$data"
}

# Get a stored credential
credential_get() {
    local name=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        log_error "Usage: credentials get --name NAME"
        exit 1
    fi
    
    get_credential "$name"
}

# List stored credentials
credential_list() {
    list_credentials
}

# Remove a stored credential
credential_remove() {
    local name=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        log_error "Usage: credentials remove --name NAME"
        exit 1
    fi
    
    remove_credential "$name"
}

# Rotate the master encryption key
credential_rotate() {
    rotate_master_key
}

# Command: schedule
cmd_schedule() {
    local subcommand="${1:-list}"
    shift || true
    
    case "$subcommand" in
        create)
            schedule_create "$@"
            ;;
        list)
            list_schedules
            ;;
        get)
            schedule_get "$@"
            ;;
        enable)
            schedule_enable_cmd "$@"
            ;;
        disable)
            schedule_disable_cmd "$@"
            ;;
        delete)
            schedule_delete_cmd "$@"
            ;;
        execute)
            schedule_execute_cmd "$@"
            ;;
        status)
            schedule_status_cmd "$@"
            ;;
        validate)
            validate_schedules
            ;;
        *)
            echo "Error: Unknown schedule subcommand: $subcommand" >&2
            echo "Usage: schedule [create|list|get|enable|disable|delete|execute|status|validate]" >&2
            exit 1
            ;;
    esac
}

# Create a new schedule
schedule_create() {
    local name=""
    local connection_id=""
    local cron=""
    local enabled="true"
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            --connection-id)
                connection_id="$2"
                shift 2
                ;;
            --cron)
                cron="$2"
                shift 2
                ;;
            --disabled)
                enabled="false"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" || -z "$connection_id" || -z "$cron" ]]; then
        log_error "Usage: schedule create --name NAME --connection-id ID --cron 'CRON_EXPRESSION' [--disabled]"
        exit 1
    fi
    
    create_schedule "$name" "$connection_id" "$cron" "$enabled"
}

# Get schedule details
schedule_get() {
    local name=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        log_error "Usage: schedule get --name NAME"
        exit 1
    fi
    
    get_schedule "$name"
}

# Enable a schedule
schedule_enable_cmd() {
    local name=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        log_error "Usage: schedule enable --name NAME"
        exit 1
    fi
    
    enable_schedule "$name"
}

# Disable a schedule
schedule_disable_cmd() {
    local name=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        log_error "Usage: schedule disable --name NAME"
        exit 1
    fi
    
    disable_schedule "$name"
}

# Delete a schedule
schedule_delete_cmd() {
    local name=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        log_error "Usage: schedule delete --name NAME"
        exit 1
    fi
    
    delete_schedule "$name"
}

# Execute a scheduled sync
schedule_execute_cmd() {
    local name=""
    local connection_id=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            --connection-id)
                connection_id="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" || -z "$connection_id" ]]; then
        log_error "Usage: schedule execute --name NAME --connection-id ID"
        exit 1
    fi
    
    execute_scheduled_sync "$name" "$connection_id"
}

# Get schedule status
schedule_status_cmd() {
    local name=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        log_error "Usage: schedule status --name NAME"
        exit 1
    fi
    
    schedule_status "$name"
}

# Command: webhook
cmd_webhook() {
    local subcommand="${1:-list}"
    shift || true
    
    case "$subcommand" in
        register)
            webhook_register "$@"
            ;;
        list)
            list_webhooks
            ;;
        get)
            webhook_get "$@"
            ;;
        enable)
            webhook_enable_cmd "$@"
            ;;
        disable)
            webhook_disable_cmd "$@"
            ;;
        delete)
            webhook_delete_cmd "$@"
            ;;
        test)
            webhook_test_cmd "$@"
            ;;
        stats)
            webhook_stats "$@"
            ;;
        *)
            echo "Error: Unknown webhook subcommand: $subcommand" >&2
            echo "Usage: webhook [register|list|get|enable|disable|delete|test|stats]" >&2
            exit 1
            ;;
    esac
}

# Register a new webhook
webhook_register() {
    local name=""
    local url=""
    local events=""
    local enabled="true"
    local auth_type="none"
    local auth_value=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            --url)
                url="$2"
                shift 2
                ;;
            --events)
                events="$2"
                shift 2
                ;;
            --disabled)
                enabled="false"
                shift
                ;;
            --auth-type)
                auth_type="$2"
                shift 2
                ;;
            --auth-value)
                auth_value="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" || -z "$url" || -z "$events" ]]; then
        log_error "Usage: webhook register --name NAME --url URL --events 'event1,event2' [--disabled] [--auth-type TYPE] [--auth-value VALUE]"
        exit 1
    fi
    
    register_webhook "$name" "$url" "$events" "$enabled" "$auth_type" "$auth_value"
}

# Get webhook details
webhook_get() {
    local name=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        log_error "Usage: webhook get --name NAME"
        exit 1
    fi
    
    get_webhook "$name"
}

# Enable a webhook
webhook_enable_cmd() {
    local name=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        log_error "Usage: webhook enable --name NAME"
        exit 1
    fi
    
    enable_webhook "$name"
}

# Disable a webhook
webhook_disable_cmd() {
    local name=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        log_error "Usage: webhook disable --name NAME"
        exit 1
    fi
    
    disable_webhook "$name"
}

# Delete a webhook
webhook_delete_cmd() {
    local name=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        log_error "Usage: webhook delete --name NAME"
        exit 1
    fi
    
    delete_webhook "$name"
}

# Test a webhook
webhook_test_cmd() {
    local name=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        log_error "Usage: webhook test --name NAME"
        exit 1
    fi
    
    test_webhook "$name"
}