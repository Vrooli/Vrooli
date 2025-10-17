#!/usr/bin/env bash
# DeepStack Resource - Core Library Functions

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
readonly RESOURCE_DIR="$(dirname "${SCRIPT_DIR}")"

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
        echo "DeepStack Computer Vision Resource Information"
        echo "=============================================="
        jq -r '
            "Startup Order: \(.startup_order)",
            "Dependencies: \(.dependencies | join(", "))",
            "Optional: \(.optional_dependencies | join(", "))",
            "Startup Time: \(.startup_time_estimate)",
            "Timeout: \(.startup_timeout)s",
            "Recovery Attempts: \(.recovery_attempts)",
            "Priority: \(.priority)",
            "Port: \(.api.port)",
            "Category: \(.category)"
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
    
    echo "Installing DeepStack Computer Vision Resource..."
    
    # Check if Docker is installed
    if ! command -v docker &> /dev/null; then
        echo "Error: Docker is required but not installed" >&2
        echo "Please install Docker first: https://docs.docker.com/get-docker/" >&2
        return 1
    fi
    
    # Create required directories
    mkdir -p "${DEEPSTACK_DATA_DIR}"
    mkdir -p "${DEEPSTACK_MODEL_DIR}"
    mkdir -p "${DEEPSTACK_TEMP_DIR}"
    mkdir -p "$(dirname "${DEEPSTACK_LOG_FILE}")"
    
    # Pull Docker image
    echo "Pulling DeepStack Docker image..."
    docker pull "${DEEPSTACK_IMAGE}"
    
    # Check for GPU support
    if [[ "${DEEPSTACK_ENABLE_GPU}" == "auto" ]] || [[ "${DEEPSTACK_ENABLE_GPU}" == "true" ]]; then
        if command -v nvidia-smi &> /dev/null && nvidia-smi &> /dev/null; then
            echo "GPU detected - enabling GPU acceleration"
            export DEEPSTACK_ENABLE_GPU="true"
        else
            echo "No GPU detected - using CPU mode"
            export DEEPSTACK_ENABLE_GPU="false"
        fi
    fi
    
    # Create Docker network if it doesn't exist
    if ! docker network ls | grep -q "${DEEPSTACK_NETWORK}"; then
        echo "Creating Docker network: ${DEEPSTACK_NETWORK}"
        docker network create "${DEEPSTACK_NETWORK}" || true
    fi
    
    if [[ "$skip_validation" != "true" ]]; then
        echo "Validating installation..."
        if docker images | grep -q "${DEEPSTACK_IMAGE%%:*}"; then
            echo "✓ DeepStack Docker image installed successfully"
        else
            echo "✗ Failed to install DeepStack Docker image" >&2
            return 1
        fi
    fi
    
    echo "DeepStack installation complete!"
    return 0
}

# Uninstall resource
uninstall_resource() {
    local force=false
    local keep_data=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --force) force=true ;;
            --keep-data) keep_data=true ;;
            *) echo "Warning: Unknown option: $1" >&2 ;;
        esac
        shift
    done
    
    echo "Uninstalling DeepStack Resource..."
    
    # Stop service if running
    stop_service --force || true
    
    # Remove container if exists
    if docker ps -a | grep -q "${DEEPSTACK_CONTAINER_NAME}"; then
        echo "Removing Docker container..."
        docker rm -f "${DEEPSTACK_CONTAINER_NAME}" || true
    fi
    
    # Optionally remove data
    if [[ "$keep_data" != "true" ]]; then
        echo "Removing data directories..."
        rm -rf "${DEEPSTACK_DATA_DIR}"
        rm -rf "${DEEPSTACK_MODEL_DIR}"
        rm -rf "${DEEPSTACK_TEMP_DIR}"
    else
        echo "Keeping data directories (--keep-data specified)"
    fi
    
    echo "DeepStack uninstalled successfully"
    return 0
}

# Start the service
start_service() {
    local wait=false
    local timeout=60
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --wait) wait=true ;;
            --timeout) 
                shift
                timeout="${1:-60}"
                ;;
            *) echo "Warning: Unknown option: $1" >&2 ;;
        esac
        shift
    done
    
    # Check if already running
    if docker ps | grep -q "${DEEPSTACK_CONTAINER_NAME}"; then
        echo "DeepStack is already running"
        return 2
    fi
    
    echo "Starting DeepStack service..."
    
    # Build Docker run command
    local docker_cmd="docker run -d"
    docker_cmd+=" --name ${DEEPSTACK_CONTAINER_NAME}"
    docker_cmd+=" --network ${DEEPSTACK_NETWORK}"
    docker_cmd+=" -p ${DEEPSTACK_PORT}:5000"
    docker_cmd+=" -v ${DEEPSTACK_DATA_DIR}:/datastore"
    docker_cmd+=" -v ${DEEPSTACK_MODEL_DIR}:/modelstore"
    docker_cmd+=" -v ${DEEPSTACK_TEMP_DIR}:/tempstore"
    docker_cmd+=" -e VISION-DETECTION=${DEEPSTACK_DETECTION_MODEL}"
    docker_cmd+=" -e VISION-FACE=${DEEPSTACK_FACE_MODEL}"
    docker_cmd+=" -e VISION-SCENE=${DEEPSTACK_SCENE_MODEL}"
    docker_cmd+=" -e MODE=${DEEPSTACK_MODE}"
    
    # Add GPU support if enabled
    if [[ "${DEEPSTACK_ENABLE_GPU}" == "true" ]]; then
        docker_cmd+=" --gpus all"
    fi
    
    # Add API key if configured
    if [[ -n "${DEEPSTACK_API_KEY}" ]]; then
        docker_cmd+=" -e API-KEY=${DEEPSTACK_API_KEY}"
    fi
    
    docker_cmd+=" ${DEEPSTACK_IMAGE}"
    
    # Start container
    eval "${docker_cmd}"
    
    if [[ "$wait" == "true" ]]; then
        echo "Waiting for DeepStack to be ready (timeout: ${timeout}s)..."
        local elapsed=0
        while [[ $elapsed -lt $timeout ]]; do
            if timeout 5 curl -sf "http://${DEEPSTACK_HOST}:${DEEPSTACK_PORT}/v1/vision/detection" &> /dev/null; then
                echo "✓ DeepStack is ready!"
                return 0
            fi
            sleep 2
            ((elapsed += 2))
            echo -n "."
        done
        echo ""
        echo "Warning: DeepStack did not become ready within ${timeout}s" >&2
        return 1
    fi
    
    echo "DeepStack service started"
    return 0
}

# Stop the service
stop_service() {
    local force=false
    local timeout=30
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --force) force=true ;;
            --timeout) 
                shift
                timeout="${1:-30}"
                ;;
            *) echo "Warning: Unknown option: $1" >&2 ;;
        esac
        shift
    done
    
    # Check if running
    if ! docker ps | grep -q "${DEEPSTACK_CONTAINER_NAME}"; then
        echo "DeepStack is not running"
        return 2
    fi
    
    echo "Stopping DeepStack service..."
    
    if [[ "$force" == "true" ]]; then
        docker kill "${DEEPSTACK_CONTAINER_NAME}"
    else
        docker stop --time="${timeout}" "${DEEPSTACK_CONTAINER_NAME}"
    fi
    
    docker rm "${DEEPSTACK_CONTAINER_NAME}" || true
    
    echo "DeepStack service stopped"
    return 0
}

# Restart the service
restart_service() {
    local force=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --force) force=true ;;
            *) echo "Warning: Unknown option: $1" >&2 ;;
        esac
        shift
    done
    
    echo "Restarting DeepStack service..."
    
    if [[ "$force" == "true" ]]; then
        stop_service --force || true
    else
        stop_service || true
    fi
    
    start_service --wait
    return $?
}

# Handle test commands
handle_test() {
    local subcommand="${1:-}"
    shift || true
    
    # Delegate to test runner
    local test_runner="${RESOURCE_DIR}/test/run-tests.sh"
    
    if [[ ! -f "$test_runner" ]]; then
        echo "Error: Test runner not found at $test_runner" >&2
        return 1
    fi
    
    case "$subcommand" in
        all|smoke|integration|unit)
            bash "$test_runner" "$subcommand"
            ;;
        *)
            echo "Error: Unknown test subcommand: $subcommand" >&2
            echo "Valid subcommands: all, smoke, integration, unit" >&2
            return 1
            ;;
    esac
}

# Handle content management commands
handle_content() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        add)
            add_content "$@"
            ;;
        list)
            list_content "$@"
            ;;
        get)
            get_content "$@"
            ;;
        remove)
            remove_content "$@"
            ;;
        execute)
            execute_content "$@"
            ;;
        *)
            echo "Error: Unknown content subcommand: $subcommand" >&2
            echo "Valid subcommands: add, list, get, remove, execute" >&2
            return 1
            ;;
    esac
}

# Add content (model or face registration)
add_content() {
    local file=""
    local type=""
    local name=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --file) shift; file="$1" ;;
            --type) shift; type="$1" ;;
            --name) shift; name="$1" ;;
            *) echo "Warning: Unknown option: $1" >&2 ;;
        esac
        shift
    done
    
    if [[ -z "$type" ]]; then
        echo "Error: --type is required (face, model)" >&2
        return 1
    fi
    
    case "$type" in
        face)
            if [[ -z "$name" ]] || [[ -z "$file" ]]; then
                echo "Error: --name and --file are required for face registration" >&2
                return 1
            fi
            echo "Registering face: $name from $file"
            curl -X POST "http://${DEEPSTACK_HOST}:${DEEPSTACK_PORT}/v1/vision/face/register" \
                -F "userid=${name}" \
                -F "image=@${file}"
            ;;
        model)
            echo "Custom model upload not yet implemented"
            return 2
            ;;
        *)
            echo "Error: Unknown content type: $type" >&2
            return 1
            ;;
    esac
}

# List available content
list_content() {
    local format="${1:-text}"
    
    echo "Available Models:"
    echo "- Object Detection: ${DEEPSTACK_DETECTION_MODEL}"
    echo "- Face Detection: ${DEEPSTACK_FACE_MODEL}"
    echo "- Scene Classification: ${DEEPSTACK_SCENE_MODEL}"
    
    if [[ -d "${DEEPSTACK_MODEL_DIR}" ]]; then
        local custom_models=$(ls -1 "${DEEPSTACK_MODEL_DIR}" 2>/dev/null | wc -l)
        if [[ $custom_models -gt 0 ]]; then
            echo ""
            echo "Custom Models:"
            ls -1 "${DEEPSTACK_MODEL_DIR}"
        fi
    fi
}

# Get content information
get_content() {
    local name=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name) shift; name="$1" ;;
            *) echo "Warning: Unknown option: $1" >&2 ;;
        esac
        shift
    done
    
    if [[ -z "$name" ]]; then
        echo "Error: --name is required" >&2
        return 1
    fi
    
    echo "Model information for: $name"
    echo "Not yet implemented"
    return 2
}

# Remove content
remove_content() {
    local name=""
    local force=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name) shift; name="$1" ;;
            --force) force=true ;;
            *) echo "Warning: Unknown option: $1" >&2 ;;
        esac
        shift
    done
    
    if [[ -z "$name" ]]; then
        echo "Error: --name is required" >&2
        return 1
    fi
    
    echo "Removing content: $name"
    echo "Not yet implemented"
    return 2
}

# Execute detection on content
execute_content() {
    local file=""
    local type="object"
    local options=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --file) shift; file="$1" ;;
            --type) shift; type="$1" ;;
            --options) shift; options="$1" ;;
            *) echo "Warning: Unknown option: $1" >&2 ;;
        esac
        shift
    done
    
    if [[ -z "$file" ]]; then
        echo "Error: --file is required" >&2
        return 1
    fi
    
    if [[ ! -f "$file" ]]; then
        echo "Error: File not found: $file" >&2
        return 1
    fi
    
    case "$type" in
        object)
            echo "Detecting objects in: $file"
            curl -X POST "http://${DEEPSTACK_HOST}:${DEEPSTACK_PORT}/v1/vision/detection" \
                -F "image=@${file}" \
                -F "min_confidence=${DEEPSTACK_CONFIDENCE_THRESHOLD}"
            ;;
        face)
            echo "Detecting faces in: $file"
            curl -X POST "http://${DEEPSTACK_HOST}:${DEEPSTACK_PORT}/v1/vision/face" \
                -F "image=@${file}"
            ;;
        face-recognize)
            echo "Recognizing faces in: $file"
            curl -X POST "http://${DEEPSTACK_HOST}:${DEEPSTACK_PORT}/v1/vision/face/recognize" \
                -F "image=@${file}"
            ;;
        scene)
            echo "Classifying scene in: $file"
            curl -X POST "http://${DEEPSTACK_HOST}:${DEEPSTACK_PORT}/v1/vision/scene" \
                -F "image=@${file}"
            ;;
        *)
            echo "Error: Unknown detection type: $type" >&2
            return 1
            ;;
    esac
}

# Show service status
show_status() {
    local format="${1:-text}"
    
    # Check if container exists
    if ! docker ps -a | grep -q "${DEEPSTACK_CONTAINER_NAME}"; then
        if [[ "$format" == "--json" ]]; then
            echo '{"status":"not_installed","running":false}'
        else
            echo "Status: Not installed"
        fi
        return 2
    fi
    
    # Check if running
    if docker ps | grep -q "${DEEPSTACK_CONTAINER_NAME}"; then
        # Check health
        if timeout "${DEEPSTACK_HEALTH_TIMEOUT}" curl -sf "http://${DEEPSTACK_HOST}:${DEEPSTACK_PORT}/v1/vision/detection" &> /dev/null; then
            if [[ "$format" == "--json" ]]; then
                echo '{"status":"healthy","running":true,"port":'${DEEPSTACK_PORT}',"gpu":"'${DEEPSTACK_ENABLE_GPU}'"}'
            else
                echo "Status: Running and healthy"
                echo "Port: ${DEEPSTACK_PORT}"
                echo "GPU: ${DEEPSTACK_ENABLE_GPU}"
                echo "Mode: ${DEEPSTACK_MODE}"
            fi
            return 0
        else
            if [[ "$format" == "--json" ]]; then
                echo '{"status":"unhealthy","running":true,"port":'${DEEPSTACK_PORT}'}'
            else
                echo "Status: Running but unhealthy"
                echo "Port: ${DEEPSTACK_PORT}"
            fi
            return 1
        fi
    else
        if [[ "$format" == "--json" ]]; then
            echo '{"status":"stopped","running":false}'
        else
            echo "Status: Stopped"
        fi
        return 2
    fi
}

# Show logs
show_logs() {
    local tail_lines=50
    local follow=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --tail) shift; tail_lines="${1:-50}" ;;
            --follow) follow=true ;;
            *) echo "Warning: Unknown option: $1" >&2 ;;
        esac
        shift
    done
    
    if ! docker ps -a | grep -q "${DEEPSTACK_CONTAINER_NAME}"; then
        echo "Error: DeepStack container not found" >&2
        return 2
    fi
    
    if [[ "$follow" == "true" ]]; then
        docker logs -f --tail "${tail_lines}" "${DEEPSTACK_CONTAINER_NAME}"
    else
        docker logs --tail "${tail_lines}" "${DEEPSTACK_CONTAINER_NAME}"
    fi
}