#!/bin/bash
# Blender core functionality

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
BLENDER_CORE_DIR="${APP_ROOT}/resources/blender/lib"

# Source utilities (disable strict mode for compatibility)
# Only source if not already loaded
if [[ -z "${var_LIB_DIR:-}" ]]; then
    set +euo pipefail
    source "${BLENDER_CORE_DIR}/../../../../lib/utils/var.sh" || true
    set -euo pipefail 2>/dev/null || true
fi

# Configuration
export BLENDER_VERSION="${BLENDER_VERSION:-4.0}"
export BLENDER_DATA_DIR="${BLENDER_DATA_DIR:-${HOME}/.vrooli/blender}"
export BLENDER_SCRIPTS_DIR="${BLENDER_SCRIPTS_DIR:-${BLENDER_DATA_DIR}/scripts}"
export BLENDER_OUTPUT_DIR="${BLENDER_OUTPUT_DIR:-${BLENDER_DATA_DIR}/output}"
export BLENDER_CONTAINER_NAME="vrooli-blender"
export BLENDER_PORT="${BLENDER_PORT:-8093}"

# Initialize Blender directories and configuration
blender::init() {
    # Create required directories
    mkdir -p "$BLENDER_DATA_DIR"
    mkdir -p "$BLENDER_SCRIPTS_DIR"
    mkdir -p "$BLENDER_OUTPUT_DIR"
    mkdir -p "${BLENDER_DATA_DIR}/config"
    mkdir -p "${BLENDER_DATA_DIR}/addons"
    mkdir -p "${BLENDER_DATA_DIR}/temp"
    
    # Create initial configuration if not exists
    if [[ ! -f "${BLENDER_DATA_DIR}/config/blender.conf" ]]; then
        cat > "${BLENDER_DATA_DIR}/config/blender.conf" << EOF
# Blender Resource Configuration
version=${BLENDER_VERSION}
data_dir=${BLENDER_DATA_DIR}
scripts_dir=${BLENDER_SCRIPTS_DIR}
output_dir=${BLENDER_OUTPUT_DIR}
container_name=${BLENDER_CONTAINER_NAME}
port=${BLENDER_PORT}
EOF
    fi
    
    # Port registry is optional - skip if causes issues
    if [[ -f "${APP_ROOT}/scripts/resources/port_registry.sh" ]]; then
        source "${APP_ROOT}/scripts/resources/port_registry.sh" 2>/dev/null || true
        
        # Register our port if not already registered
        if type port_registry::is_registered &>/dev/null; then
            port_registry::is_registered "blender" || \
                port_registry::register "blender" "$BLENDER_PORT" "Blender 3D creation suite" 2>/dev/null || true
        fi
    fi
    
    return 0
}

# Check if Blender is installed
blender::is_installed() {
    # Check both native and docker installations
    if command -v blender &>/dev/null; then
        return 0
    elif docker inspect "$BLENDER_CONTAINER_NAME" &>/dev/null; then
        return 0
    fi
    return 1
}

# Check if Blender is running
blender::is_running() {
    # For native Blender, consider it "running" if installed
    if command -v blender &>/dev/null; then
        return 0
    elif docker ps --format "{{.Names}}" | grep -q "^${BLENDER_CONTAINER_NAME}$"; then
        return 0
    fi
    return 1
}

# Install Blender
blender::install() {
    echo "[INFO] Installing Blender ${BLENDER_VERSION}..."
    
    # Initialize directories
    blender::init
    
    # Try native installation first
    echo "[INFO] Installing Blender natively..."
    if command -v apt-get &>/dev/null; then
        # No sudo since we're in apply mode
        apt-get update && apt-get install -y blender && {
            echo "[SUCCESS] Blender installed natively"
            
            # Register the CLI with vrooli
            local resource_dir="${APP_ROOT}/resources/blender"
            if [[ -f "${APP_ROOT}/scripts/lib/resources/install-resource-cli.sh" ]]; then
                "${APP_ROOT}/scripts/lib/resources/install-resource-cli.sh" "${resource_dir}" 2>/dev/null || true
            fi
            
            return 0
        } || echo "[WARN] Native installation failed, trying Docker..."
    fi
    
    # Fallback to Docker if native fails
    echo "[INFO] Creating custom Blender Docker image..."
    
    # Create a Dockerfile for Blender
    local dockerfile="${BLENDER_DATA_DIR}/Dockerfile"
    cat > "$dockerfile" << 'DOCKERFILE'
FROM ubuntu:22.04

RUN apt-get update && apt-get install -y \
    blender \
    python3 \
    python3-pip \
    xvfb \
    && rm -rf /var/lib/apt/lists/*

# Create working directories
RUN mkdir -p /scripts /output /config

WORKDIR /scripts

# Set up display for headless operation
ENV DISPLAY=:99

# Entry point for running Blender
ENTRYPOINT ["xvfb-run", "-a", "blender"]
DOCKERFILE
    
    # Build the image
    docker build -t "vrooli/blender:latest" -f "$dockerfile" "${BLENDER_DATA_DIR}" || {
        echo "[ERROR] Failed to build Blender image"
        return 1
    }
    
    # Create container
    echo "[INFO] Creating Blender container..."
    local blender_image="vrooli/blender:latest"
    
    docker run -d \
        --name "$BLENDER_CONTAINER_NAME" \
        -e PUID=$(id -u) \
        -e PGID=$(id -g) \
        -e TZ=UTC \
        -v "${BLENDER_DATA_DIR}:/config" \
        -v "${BLENDER_SCRIPTS_DIR}:/scripts" \
        -v "${BLENDER_OUTPUT_DIR}:/output" \
        --restart unless-stopped \
        "$blender_image" \
        --background --python-console || {
        echo "[ERROR] Failed to create Blender container"
        return 1
    }
    
    echo "[SUCCESS] Blender installed successfully"
    
    # Register the CLI with vrooli
    local resource_dir="${APP_ROOT}/resources/blender"
    if [[ -f "${APP_ROOT}/scripts/lib/resources/install-resource-cli.sh" ]]; then
        "${APP_ROOT}/scripts/lib/resources/install-resource-cli.sh" "${resource_dir}" 2>/dev/null || true
    fi
    
    return 0
}

# Start Blender
blender::start() {
    if ! blender::is_installed; then
        echo "[INFO] Blender not installed, installing..."
        blender::install || return 1
    fi
    
    if blender::is_running; then
        echo "[INFO] Blender is already running"
        return 0
    fi
    
    echo "[INFO] Starting Blender..."
    docker start "$BLENDER_CONTAINER_NAME" || {
        echo "[ERROR] Failed to start Blender"
        return 1
    }
    
    # Wait for Blender to be ready
    echo "[INFO] Waiting for Blender to be ready..."
    local max_attempts=30
    local attempt=0
    
    while [[ $attempt -lt $max_attempts ]]; do
        if docker exec "$BLENDER_CONTAINER_NAME" python3 -c "import bpy; print(bpy.app.version_string)" &>/dev/null; then
            echo "[SUCCESS] Blender started successfully"
            return 0
        fi
        sleep 2
        ((attempt++))
    done
    
    echo "[ERROR] Blender failed to become ready"
    return 1
}

# Stop Blender
blender::stop() {
    if ! blender::is_running; then
        echo "[INFO] Blender is not running"
        return 0
    fi
    
    echo "[INFO] Stopping Blender..."
    docker stop "$BLENDER_CONTAINER_NAME" || {
        echo "[ERROR] Failed to stop Blender"
        return 1
    }
    
    echo "[SUCCESS] Blender stopped"
    return 0
}

# Run a Blender Python script
blender::run_script() {
    local script="$1"
    
    if [[ -z "$script" ]]; then
        echo "[ERROR] No script specified"
        return 1
    fi
    
    # Look for script in various locations
    local script_path=""
    if [[ -f "$script" ]]; then
        script_path="$script"
    elif [[ -f "${BLENDER_SCRIPTS_DIR}/$script" ]]; then
        script_path="${BLENDER_SCRIPTS_DIR}/$script"
    elif [[ -f "${BLENDER_SCRIPTS_DIR}/injected/$script" ]]; then
        script_path="${BLENDER_SCRIPTS_DIR}/injected/$script"
    else
        echo "[ERROR] Script not found: $script"
        return 1
    fi
    
    # Copy script to container if it's external
    if [[ "$script_path" != "${BLENDER_SCRIPTS_DIR}"* ]]; then
        cp "$script_path" "${BLENDER_SCRIPTS_DIR}/" || {
            echo "[ERROR] Failed to copy script to scripts directory"
            return 1
        }
        script_path="${BLENDER_SCRIPTS_DIR}/$(basename "$script")"
    fi
    
    # Run the script in Blender
    echo "[INFO] Running script: $(basename "$script")"
    
    # For Docker, determine the path inside container
    local container_script_path
    if [[ "$script_path" == "${BLENDER_SCRIPTS_DIR}/injected/"* ]]; then
        container_script_path="/scripts/injected/$(basename "$script")"
    elif [[ "$script_path" == "${BLENDER_SCRIPTS_DIR}/"* ]]; then
        container_script_path="/scripts/$(basename "$script")"
    else
        # External script, copy it first
        cp "$script_path" "${BLENDER_SCRIPTS_DIR}/" 
        container_script_path="/scripts/$(basename "$script")"
    fi
    
    # Check if using native or Docker Blender
    if docker ps --format "{{.Names}}" | grep -q "^${BLENDER_CONTAINER_NAME}$"; then
        # Docker Blender
        docker exec "$BLENDER_CONTAINER_NAME" blender \
            --background \
            --python "$container_script_path" || {
            echo "[ERROR] Script execution failed"
            return 1
        }
    elif command -v blender &>/dev/null; then
        # Native Blender fallback
        blender --background --python "$script_path" || {
            echo "[ERROR] Script execution failed"
            return 1
        }
    else
        echo "[ERROR] Blender is not available"
        return 1
    fi
    
    echo "[SUCCESS] Script executed successfully"
    return 0
}

# List injected scripts
blender::list_scripts() {
    if [[ ! -d "$BLENDER_SCRIPTS_DIR" ]]; then
        echo "[INFO] No scripts directory found"
        return 0
    fi
    
    local count=$(find "$BLENDER_SCRIPTS_DIR" -name "*.py" -type f | wc -l)
    echo "[INFO] Found $count Python scripts:"
    
    if [[ $count -gt 0 ]]; then
        find "$BLENDER_SCRIPTS_DIR" -name "*.py" -type f -printf "  - %f\n"
    fi
    
    return 0
}

# Export functions
export -f blender::init
export -f blender::is_installed
export -f blender::is_running
export -f blender::install
export -f blender::start
export -f blender::stop
export -f blender::run_script
export -f blender::list_scripts