#!/usr/bin/env bash
################################################################################
# FreeCAD Resource - Core Library Functions
################################################################################

# Get FreeCAD port from registry
freecad::get_port() {
    local port_registry="${APP_ROOT}/scripts/resources/port_registry.sh"
    if [[ -f "$port_registry" ]]; then
        # shellcheck disable=SC1090
        source "$port_registry"
        echo "${RESOURCE_PORTS[freecad]:-8195}"
    else
        echo "8195"
    fi
}

# Initialize FreeCAD configuration
freecad::init() {
    # Set port from registry if not already set
    if [[ -z "${FREECAD_PORT:-}" ]]; then
        export FREECAD_PORT="$(freecad::get_port)"
    fi
    
    # Create necessary directories
    local dirs=(
        "${FREECAD_DATA_DIR}"
        "${FREECAD_PROJECTS_DIR}"
        "${FREECAD_SCRIPTS_DIR}"
        "${FREECAD_EXPORTS_DIR}"
        "$(dirname "${FREECAD_LOG_FILE}")"
    )
    
    for dir in "${dirs[@]}"; do
        mkdir -p "$dir"
    done
}

# Install FreeCAD
freecad::install() {
    log::info "Installing FreeCAD resource..."
    
    freecad::init
    
    # Pull Python image if not exists
    log::info "Preparing FreeCAD Docker image: ${FREECAD_IMAGE}"
    
    if ! docker images | grep -q "${FREECAD_IMAGE}"; then
        log::info "Pulling Python image for FreeCAD..."
        docker pull "${FREECAD_IMAGE}" || {
            log::error "Failed to pull Docker image"
            return 1
        }
    else
        log::info "Docker image already exists"
    fi
    
    # Create Docker network if it doesn't exist
    if ! docker network ls | grep -q "${FREECAD_NETWORK}"; then
        log::info "Creating Docker network: ${FREECAD_NETWORK}"
        docker network create "${FREECAD_NETWORK}" || true
    fi
    
    log::info "FreeCAD installed successfully"
    return 0
}

# Uninstall FreeCAD
freecad::uninstall() {
    log::info "Uninstalling FreeCAD resource..."
    
    # Stop container if running
    freecad::stop
    
    # Remove container
    if docker ps -a | grep -q "${FREECAD_CONTAINER_NAME}"; then
        log::info "Removing FreeCAD container"
        docker rm -f "${FREECAD_CONTAINER_NAME}" &>/dev/null || true
    fi
    
    # Optionally remove data
    if [[ "${1:-}" != "--keep-data" ]]; then
        log::warning "Removing FreeCAD data directories"
        rm -rf "${FREECAD_DATA_DIR}" "${FREECAD_PROJECTS_DIR}" "${FREECAD_SCRIPTS_DIR}" "${FREECAD_EXPORTS_DIR}"
    fi
    
    log::info "FreeCAD uninstalled successfully"
    return 0
}

# Start FreeCAD service
freecad::start() {
    local wait_flag="${1:-}"
    
    freecad::init
    
    # Check if already running
    if freecad::is_running; then
        log::info "FreeCAD is already running"
        return 2
    fi
    
    log::info "Starting FreeCAD service on port ${FREECAD_PORT}..."
    
    # Start container with minimal FreeCAD API server
    docker run -d \
        --name "${FREECAD_CONTAINER_NAME}" \
        --network "${FREECAD_NETWORK}" \
        -p "${FREECAD_PORT}:8080" \
        -e "PORT=8080" \
        -v "${FREECAD_DATA_DIR}:/data" \
        -v "${FREECAD_PROJECTS_DIR}:/projects" \
        -v "${FREECAD_SCRIPTS_DIR}:/scripts" \
        -v "${FREECAD_EXPORTS_DIR}:/exports" \
        -v "${FREECAD_CLI_DIR}/api_server_minimal.py:/app/api_server.py:ro" \
        --memory="${FREECAD_MEMORY_LIMIT}" \
        --cpus="${FREECAD_CPU_LIMIT}" \
        --restart=unless-stopped \
        --workdir /app \
        "${FREECAD_IMAGE}" \
        python3 /app/api_server.py || {
        log::error "Failed to start FreeCAD container"
        return 1
    }
    
    # Wait for service to be ready if requested
    if [[ "$wait_flag" == "--wait" ]]; then
        log::info "Waiting for FreeCAD to be ready..."
        local timeout="${FREECAD_STARTUP_TIMEOUT}"
        local elapsed=0
        
        while [[ $elapsed -lt $timeout ]]; do
            if freecad::health_check; then
                log::info "FreeCAD is ready"
                return 0
            fi
            sleep 2
            elapsed=$((elapsed + 2))
        done
        
        log::error "FreeCAD failed to start within ${timeout} seconds"
        return 1
    fi
    
    return 0
}

# Stop FreeCAD service
freecad::stop() {
    if ! freecad::is_running; then
        log::info "FreeCAD is not running"
        return 2
    fi
    
    log::info "Stopping FreeCAD service..."
    
    docker stop -t "${FREECAD_STOP_TIMEOUT}" "${FREECAD_CONTAINER_NAME}" &>/dev/null || {
        log::error "Failed to stop FreeCAD gracefully, forcing..."
        docker kill "${FREECAD_CONTAINER_NAME}" &>/dev/null || true
    }
    
    docker rm "${FREECAD_CONTAINER_NAME}" &>/dev/null || true
    
    log::info "FreeCAD stopped successfully"
    return 0
}

# Restart FreeCAD service
freecad::restart() {
    log::info "Restarting FreeCAD service..."
    freecad::stop
    sleep 2
    freecad::start --wait
}

# Check if FreeCAD is running
freecad::is_running() {
    docker ps --format "{{.Names}}" | grep -q "^${FREECAD_CONTAINER_NAME}$"
}

# Health check
freecad::health_check() {
    freecad::init
    
    # Check container is running
    if ! freecad::is_running; then
        return 1
    fi
    
    # Check HTTP endpoint
    timeout 5 curl -sf "http://localhost:${FREECAD_PORT}/health" &>/dev/null || {
        # Fallback to basic HTTP check
        timeout 5 curl -sf "http://localhost:${FREECAD_PORT}/" &>/dev/null
    }
}

# Get FreeCAD status
freecad::status() {
    freecad::init
    
    echo "FreeCAD Resource Status"
    echo "======================="
    echo ""
    
    if freecad::is_running; then
        echo "Status: RUNNING"
        echo "Port: ${FREECAD_PORT}"
        echo "Container: ${FREECAD_CONTAINER_NAME}"
        echo "Display: ${FREECAD_DISPLAY}"
        
        # Get container stats
        local stats
        stats=$(docker stats --no-stream --format "json" "${FREECAD_CONTAINER_NAME}" 2>/dev/null || echo "{}")
        if [[ -n "$stats" ]] && [[ "$stats" != "{}" ]]; then
            echo ""
            echo "Resource Usage:"
            echo "  CPU: $(echo "$stats" | jq -r '.CPUPerc // "N/A"')"
            echo "  Memory: $(echo "$stats" | jq -r '.MemUsage // "N/A"')"
        fi
        
        # Check health
        echo ""
        if freecad::health_check; then
            echo "Health: HEALTHY"
        else
            echo "Health: UNHEALTHY"
        fi
    else
        echo "Status: STOPPED"
    fi
    
    echo ""
    echo "Configuration:"
    echo "  Image: ${FREECAD_IMAGE}"
    echo "  Memory Limit: ${FREECAD_MEMORY_LIMIT}"
    echo "  CPU Limit: ${FREECAD_CPU_LIMIT}"
    echo "  Thread Count: ${FREECAD_THREAD_COUNT}"
    echo ""
    echo "Directories:"
    echo "  Data: ${FREECAD_DATA_DIR}"
    echo "  Projects: ${FREECAD_PROJECTS_DIR}"
    echo "  Scripts: ${FREECAD_SCRIPTS_DIR}"
    echo "  Exports: ${FREECAD_EXPORTS_DIR}"
}

# Show FreeCAD logs
freecad::logs() {
    if freecad::is_running; then
        docker logs "${FREECAD_CONTAINER_NAME}" 2>&1
    else
        log::error "FreeCAD is not running"
        return 1
    fi
}

# Content management functions
freecad::content::add() {
    local file="${1:-}"
    
    if [[ -z "$file" ]]; then
        log::error "No file specified"
        return 1
    fi
    
    if [[ ! -f "$file" ]]; then
        log::error "File not found: $file"
        return 1
    fi
    
    local filename=$(basename "$file")
    local target="${FREECAD_SCRIPTS_DIR}/${filename}"
    
    cp "$file" "$target"
    log::info "Added script: $filename"
}

freecad::content::list() {
    echo "FreeCAD Scripts:"
    echo "==============="
    if [[ -d "${FREECAD_SCRIPTS_DIR}" ]]; then
        ls -la "${FREECAD_SCRIPTS_DIR}"
    else
        echo "No scripts directory found"
    fi
    
    echo ""
    echo "FreeCAD Projects:"
    echo "================"
    if [[ -d "${FREECAD_PROJECTS_DIR}" ]]; then
        ls -la "${FREECAD_PROJECTS_DIR}"
    else
        echo "No projects directory found"
    fi
}

freecad::content::get() {
    local name="${1:-}"
    
    if [[ -z "$name" ]]; then
        log::error "No script name specified"
        return 1
    fi
    
    local script="${FREECAD_SCRIPTS_DIR}/${name}"
    if [[ -f "$script" ]]; then
        cat "$script"
    else
        log::error "Script not found: $name"
        return 1
    fi
}

freecad::content::remove() {
    local name="${1:-}"
    
    if [[ -z "$name" ]]; then
        log::error "No script name specified"
        return 1
    fi
    
    local script="${FREECAD_SCRIPTS_DIR}/${name}"
    if [[ -f "$script" ]]; then
        rm "$script"
        log::info "Removed script: $name"
    else
        log::error "Script not found: $name"
        return 1
    fi
}

freecad::content::execute() {
    local script="${1:-}"
    
    if [[ -z "$script" ]]; then
        log::error "No script specified"
        return 1
    fi
    
    if ! freecad::is_running; then
        log::error "FreeCAD is not running"
        return 1
    fi
    
    log::info "Executing FreeCAD script: $script"
    
    # Read script content
    local script_content
    if [[ -f "$script" ]]; then
        script_content=$(cat "$script")
    else
        # Check in scripts directory
        if [[ -f "${FREECAD_SCRIPTS_DIR}/${script}" ]]; then
            script_content=$(cat "${FREECAD_SCRIPTS_DIR}/${script}")
        else
            log::error "Script file not found: $script"
            return 1
        fi
    fi
    
    # Send script to API for execution
    local response
    response=$(curl -sf -X POST "http://localhost:${FREECAD_PORT}/generate" \
        -H "Content-Type: application/json" \
        -d "$(jq -n --arg script "$script_content" '{script: $script}')" 2>&1) || {
        log::error "Failed to execute script: $response"
        return 1
    }
    
    echo "$response" | jq '.'
}

freecad::content::generate() {
    freecad::content::execute "$@"
}

freecad::content::export() {
    local input="${1:-}"
    local format="${2:-${FREECAD_DEFAULT_EXPORT_FORMAT}}"
    
    if [[ -z "$input" ]]; then
        log::error "No input file specified"
        return 1
    fi
    
    if ! freecad::is_running; then
        log::error "FreeCAD is not running"
        return 1
    fi
    
    log::info "Exporting $input to $format format"
    
    # Send export request to API
    local response
    response=$(curl -sf -X POST "http://localhost:${FREECAD_PORT}/export" \
        -H "Content-Type: application/json" \
        -d "$(jq -n --arg input "$input" --arg format "$format" '{input: $input, format: $format}')" 2>&1) || {
        log::error "Failed to export file: $response"
        return 1
    }
    
    echo "$response" | jq '.'
}

freecad::content::analyze() {
    local input="${1:-}"
    local analysis_type="${2:-properties}"
    
    if [[ -z "$input" ]]; then
        log::error "No input file specified"
        return 1
    fi
    
    if ! freecad::is_running; then
        log::error "FreeCAD is not running"
        return 1
    fi
    
    log::info "Analyzing $input (type: $analysis_type)"
    
    # Send analyze request to API
    local response
    response=$(curl -sf -X POST "http://localhost:${FREECAD_PORT}/analyze" \
        -H "Content-Type: application/json" \
        -d "$(jq -n --arg input "$input" --arg type "$analysis_type" '{input: $input, type: $type}')" 2>&1) || {
        log::error "Failed to analyze file: $response"
        return 1
    }
    
    echo "$response" | jq '.'
}