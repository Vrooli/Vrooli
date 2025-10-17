#!/usr/bin/env bash
# Segment Anything Resource - Core Library Functions

set -euo pipefail

# Get script directory
if [[ -z "${SCRIPT_DIR:-}" ]]; then
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    readonly SCRIPT_DIR
fi
if [[ -z "${RESOURCE_DIR:-}" ]]; then
    RESOURCE_DIR="$(dirname "${SCRIPT_DIR}")"
    readonly RESOURCE_DIR
fi

# Load configuration
source "${RESOURCE_DIR}/config/defaults.sh"

# Load test functions if needed
[[ -f "${SCRIPT_DIR}/test.sh" ]] && source "${SCRIPT_DIR}/test.sh"

# Show resource information from runtime.json
show_info() {
    local format="${1:-text}"
    local runtime_file="${RESOURCE_DIR}/config/runtime.json"
    
    if [[ ! -f "$runtime_file" ]]; then
        echo "Error: runtime.json not found" >&2
        return 1
    fi
    
    if [[ "$format" == "--json" ]]; then
        cat "$runtime_file"
    else
        echo "Segment Anything Resource Information"
        echo "======================================"
        jq -r '
            "Startup Order: \(.startup_order)",
            "Dependencies: \(.dependencies | join(", "))",
            "Optional: \(.optional_dependencies | join(", "))",
            "Startup Time: \(.startup_time_estimate)",
            "Timeout: \(.startup_timeout)s",
            "Recovery Attempts: \(.recovery_attempts)",
            "Priority: \(.priority)",
            "Port: \(.api.port)",
            "Category: \(.category)",
            "Models: \(.models.available | join(", "))"
        ' "$runtime_file"
    fi
}

# Handle lifecycle management commands
handle_manage() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        install)
            install_resource "$@"
            ;;
        uninstall)
            uninstall_resource "$@"
            ;;
        start)
            start_service "$@"
            ;;
        stop)
            stop_service "$@"
            ;;
        restart)
            restart_service "$@"
            ;;
        *)
            echo "Error: Unknown manage subcommand: $subcommand" >&2
            echo "Valid subcommands: install, uninstall, start, stop, restart" >&2
            return 1
            ;;
    esac
}

# Install resource and dependencies
install_resource() {
    local force=false
    local skip_validation=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --force) force=true ;;
            --skip-validation) skip_validation=true ;;
            *) echo "Warning: Unknown option: $1" >&2 ;;
        esac
        shift
    done
    
    echo "Installing Segment Anything Resource..."
    
    # Check if Docker is installed
    if ! command -v docker &> /dev/null; then
        echo "Error: Docker is required but not installed" >&2
        echo "Please install Docker first: https://docs.docker.com/get-docker/" >&2
        return 1
    fi
    
    # Create required directories
    mkdir -p "${SEGMENT_ANYTHING_DATA_DIR}"
    mkdir -p "${SEGMENT_ANYTHING_MODEL_DIR}"
    mkdir -p "${SEGMENT_ANYTHING_CACHE_DIR}"
    
    # Build Docker image
    echo "Building Docker image..."
    
    # Copy Docker assets to build location
    if [[ -d "${RESOURCE_DIR}/docker" ]]; then
        cp -r "${RESOURCE_DIR}/docker/"* "${SEGMENT_ANYTHING_DATA_DIR}/"
    fi
    
    docker build -t "${SEGMENT_ANYTHING_IMAGE}" \
        -f "${SEGMENT_ANYTHING_DATA_DIR}/Dockerfile" \
        "${SEGMENT_ANYTHING_DATA_DIR}" || {
        echo "Error: Failed to build Docker image" >&2
        return 1
    }
    
    # Download default model if not exists
    if [[ ! -f "${SEGMENT_ANYTHING_MODEL_DIR}/sam2_${SEGMENT_ANYTHING_MODEL_SIZE}.pth" ]]; then
        echo "Downloading default model (${SEGMENT_ANYTHING_MODEL_SIZE})..."
        download_model "${SEGMENT_ANYTHING_MODEL_SIZE}"
    fi
    
    # Validate installation
    if [[ "$skip_validation" != "true" ]]; then
        echo "Validating installation..."
        if validate_installation; then
            echo "✓ Installation successful"
        else
            echo "⚠ Installation completed with warnings"
        fi
    fi
    
    echo "Segment Anything resource installed successfully"
}


# Download model weights
download_model() {
    local model_size="${1:-base}"
    local model_url=""
    
    case "$model_size" in
        tiny)
            model_url="https://dl.fbaipublicfiles.com/segment_anything_2/072824/sam2_hiera_tiny.pt"
            ;;
        small)
            model_url="https://dl.fbaipublicfiles.com/segment_anything_2/072824/sam2_hiera_small.pt"
            ;;
        base)
            model_url="https://dl.fbaipublicfiles.com/segment_anything_2/072824/sam2_hiera_base_plus.pt"
            ;;
        large)
            model_url="https://dl.fbaipublicfiles.com/segment_anything_2/072824/sam2_hiera_large.pt"
            ;;
        *)
            echo "Error: Unknown model size: $model_size" >&2
            return 1
            ;;
    esac
    
    wget -q --show-progress -O "${SEGMENT_ANYTHING_MODEL_DIR}/sam2_${model_size}.pth" "$model_url"
}

# Uninstall resource
uninstall_resource() {
    echo "Uninstalling Segment Anything resource..."
    
    # Stop service if running
    stop_service 2>/dev/null || true
    
    # Remove Docker container and image
    docker rm -f "${SEGMENT_ANYTHING_CONTAINER}" 2>/dev/null || true
    docker rmi "${SEGMENT_ANYTHING_IMAGE}" 2>/dev/null || true
    
    # Optionally remove data (prompt user)
    read -p "Remove data and models? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "${SEGMENT_ANYTHING_DATA_DIR}"
        rm -rf "${SEGMENT_ANYTHING_MODEL_DIR}"
        rm -rf "${SEGMENT_ANYTHING_CACHE_DIR}"
        echo "Data removed"
    fi
    
    echo "Segment Anything resource uninstalled"
}

# Start the service
start_service() {
    local wait_ready=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --wait) wait_ready=true ;;
            *) echo "Warning: Unknown option: $1" >&2 ;;
        esac
        shift
    done
    
    echo "Starting Segment Anything service..."
    
    # Check if already running
    if docker ps --format "table {{.Names}}" | grep -q "^${SEGMENT_ANYTHING_CONTAINER}$"; then
        echo "Service is already running"
        return 0
    fi
    
    # Copy API server to data directory if not exists
    if [[ ! -f "${SEGMENT_ANYTHING_DATA_DIR}/api_server.py" ]] && [[ -f "${RESOURCE_DIR}/docker/api_server.py" ]]; then
        cp "${RESOURCE_DIR}/docker/api_server.py" "${SEGMENT_ANYTHING_DATA_DIR}/"
    fi
    
    # Start Docker container
    docker run -d \
        --name "${SEGMENT_ANYTHING_CONTAINER}" \
        -p "${SEGMENT_ANYTHING_PORT}:11454" \
        -v "${SEGMENT_ANYTHING_MODEL_DIR}:/app/models:ro" \
        -v "${SEGMENT_ANYTHING_DATA_DIR}:/app/data" \
        -v "${SEGMENT_ANYTHING_CACHE_DIR}:/app/cache" \
        -v "${SEGMENT_ANYTHING_DATA_DIR}/api_server.py:/app/api_server.py:ro" \
        -e MODEL_SIZE="${SEGMENT_ANYTHING_MODEL_SIZE}" \
        -e MODEL_TYPE="${SEGMENT_ANYTHING_MODEL_TYPE}" \
        -e DEVICE="${SEGMENT_ANYTHING_DEVICE}" \
        -e REDIS_HOST="${SEGMENT_ANYTHING_REDIS_HOST}" \
        -e REDIS_PORT="${SEGMENT_ANYTHING_REDIS_PORT}" \
        --restart unless-stopped \
        "${SEGMENT_ANYTHING_IMAGE}" || {
        echo "Error: Failed to start Docker container" >&2
        return 1
    }
    
    if [[ "$wait_ready" == "true" ]]; then
        echo "Waiting for service to be ready..."
        wait_for_health
    fi
    
    echo "Segment Anything service started on port ${SEGMENT_ANYTHING_PORT}"
}

# Stop the service
stop_service() {
    echo "Stopping Segment Anything service..."
    
    if ! docker ps --format "table {{.Names}}" | grep -q "^${SEGMENT_ANYTHING_CONTAINER}$"; then
        echo "Service is not running"
        return 0
    fi
    
    # Stop gracefully
    docker stop --time="${SEGMENT_ANYTHING_SHUTDOWN_TIMEOUT}" "${SEGMENT_ANYTHING_CONTAINER}" || {
        echo "Warning: Failed to stop container gracefully" >&2
        docker kill "${SEGMENT_ANYTHING_CONTAINER}" 2>/dev/null || true
    }
    
    docker rm "${SEGMENT_ANYTHING_CONTAINER}" 2>/dev/null || true
    
    echo "Segment Anything service stopped"
}

# Restart the service
restart_service() {
    echo "Restarting Segment Anything service..."
    stop_service
    sleep 2
    start_service "$@"
}

# Wait for service health
wait_for_health() {
    local max_attempts=30
    local attempt=1
    
    while [[ $attempt -le $max_attempts ]]; do
        if timeout 5 curl -sf "http://localhost:${SEGMENT_ANYTHING_PORT}/health" >/dev/null 2>&1; then
            echo "✓ Service is healthy"
            return 0
        fi
        echo "Waiting for service... (attempt $attempt/$max_attempts)"
        sleep 2
        ((attempt++))
    done
    
    echo "Error: Service failed to become healthy" >&2
    return 1
}

# Handle test commands
handle_test() {
    local test_type="${1:-all}"
    
    # Source test library if not already loaded
    if ! declare -f run_smoke_test >/dev/null 2>&1; then
        source "${RESOURCE_DIR}/lib/test.sh"
    fi
    
    case "$test_type" in
        smoke)
            run_smoke_test
            ;;
        integration)
            run_integration_test
            ;;
        unit)
            run_unit_test
            ;;
        all)
            run_all_tests
            ;;
        *)
            echo "Error: Unknown test type: $test_type" >&2
            echo "Valid types: smoke, integration, unit, all" >&2
            return 1
            ;;
    esac
}

# Handle content operations
handle_content() {
    local action="${1:-}"
    shift || true
    
    case "$action" in
        list)
            list_models
            ;;
        add)
            add_model "$@"
            ;;
        get)
            get_result "$@"
            ;;
        remove)
            remove_result "$@"
            ;;
        execute)
            execute_segmentation "$@"
            ;;
        *)
            echo "Error: Unknown content action: $action" >&2
            echo "Valid actions: list, add, get, remove, execute" >&2
            return 1
            ;;
    esac
}

# List available models
list_models() {
    echo "Available models:"
    echo "  - sam2-tiny: Fastest, smallest model (632M parameters)"
    echo "  - sam2-small: Balanced speed/quality (46M parameters)"
    echo "  - sam2-base: Default model (80M parameters)"
    echo "  - sam2-large: Highest quality (224M parameters)"
    echo "  - hq-sam: High-quality variant"
    
    echo -e "\nInstalled models:"
    ls -lh "${SEGMENT_ANYTHING_MODEL_DIR}"/*.pth 2>/dev/null || echo "  No models installed"
}

# Execute segmentation
execute_segmentation() {
    local image_path=""
    local prompt_type="auto"
    local prompt_data=""
    local output_format="png"
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --image)
                image_path="$2"
                shift 2
                ;;
            --prompt)
                prompt_data="$2"
                shift 2
                ;;
            --format)
                output_format="$2"
                shift 2
                ;;
            *)
                echo "Warning: Unknown option: $1" >&2
                shift
                ;;
        esac
    done
    
    if [[ -z "$image_path" ]]; then
        echo "Error: --image path is required" >&2
        return 1
    fi
    
    # Send request to API
    curl -X POST "http://localhost:${SEGMENT_ANYTHING_PORT}/api/v1/segment" \
        -F "image=@${image_path}" \
        -F "prompt=${prompt_data}" \
        -F "format=${output_format}"
}

# Show service status
show_status() {
    local format="${1:-text}"
    
    if docker ps --format "table {{.Names}}" | grep -q "^${SEGMENT_ANYTHING_CONTAINER}$"; then
        local health_status="unknown"
        if timeout 5 curl -sf "http://localhost:${SEGMENT_ANYTHING_PORT}/health" >/dev/null 2>&1; then
            health_status="healthy"
        else
            health_status="unhealthy"
        fi
        
        if [[ "$format" == "--json" ]]; then
            echo "{\"status\":\"running\",\"health\":\"${health_status}\",\"port\":${SEGMENT_ANYTHING_PORT}}"
        else
            echo "Service Status: running"
            echo "Health: ${health_status}"
            echo "Port: ${SEGMENT_ANYTHING_PORT}"
            echo "Container: ${SEGMENT_ANYTHING_CONTAINER}"
        fi
    else
        if [[ "$format" == "--json" ]]; then
            echo "{\"status\":\"stopped\"}"
        else
            echo "Service Status: stopped"
        fi
    fi
}

# Show logs
show_logs() {
    local follow=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --follow|-f) follow=true ;;
            *) echo "Warning: Unknown option: $1" >&2 ;;
        esac
        shift
    done
    
    if [[ "$follow" == "true" ]]; then
        docker logs -f "${SEGMENT_ANYTHING_CONTAINER}"
    else
        docker logs --tail 100 "${SEGMENT_ANYTHING_CONTAINER}"
    fi
}

# Show credentials
show_credentials() {
    echo "Segment Anything Integration Credentials"
    echo "========================================"
    echo "API Endpoint: http://localhost:${SEGMENT_ANYTHING_PORT}/api/v1"
    echo "Health Check: http://localhost:${SEGMENT_ANYTHING_PORT}/health"
    echo "Model Type: ${SEGMENT_ANYTHING_MODEL_TYPE}"
    echo "Model Size: ${SEGMENT_ANYTHING_MODEL_SIZE}"
    echo ""
    echo "Optional Integrations:"
    echo "  Redis: ${SEGMENT_ANYTHING_REDIS_HOST}:${SEGMENT_ANYTHING_REDIS_PORT}"
    echo "  MinIO: ${SEGMENT_ANYTHING_MINIO_ENDPOINT}"
    echo "  PostgreSQL: ${SEGMENT_ANYTHING_POSTGRES_HOST}:${SEGMENT_ANYTHING_POSTGRES_PORT}"
}

# Validate installation
validate_installation() {
    local valid=true
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        echo "  ✗ Docker not installed"
        valid=false
    else
        echo "  ✓ Docker installed"
    fi
    
    # Check directories
    if [[ -d "${SEGMENT_ANYTHING_DATA_DIR}" ]]; then
        echo "  ✓ Data directory exists"
    else
        echo "  ✗ Data directory missing"
        valid=false
    fi
    
    # Check Docker image
    if docker images --format "{{.Repository}}:{{.Tag}}" | grep -q "^${SEGMENT_ANYTHING_IMAGE}$"; then
        echo "  ✓ Docker image built"
    else
        echo "  ✗ Docker image not built"
        valid=false
    fi
    
    [[ "$valid" == "true" ]]
}

