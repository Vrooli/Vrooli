#!/bin/bash

# OBS Studio Docker Management Module
# Provides Docker-based deployment with specific version tags

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
OBS_LIB_DIR="${APP_ROOT}/resources/obs-studio/lib"
OBS_DIR="${APP_ROOT}/resources/obs-studio"
OBS_SCRIPTS_DIR="${APP_ROOT}/scripts"

# Source dependencies
source "$OBS_SCRIPTS_DIR/lib/utils/var.sh" || exit 1
source "$OBS_SCRIPTS_DIR/lib/utils/log.sh" || exit 1
source "$OBS_SCRIPTS_DIR/lib/utils/format.sh" || exit 1
source "$OBS_SCRIPTS_DIR/resources/port_registry.sh" || exit 1

# Docker Configuration
OBS_DOCKER_IMAGE="${OBS_DOCKER_IMAGE:-obsproject/obs-studio}"
OBS_DOCKER_VERSION="${OBS_DOCKER_VERSION:-30.2.3}"  # Specific version, not 'latest'
OBS_CONTAINER_NAME="${OBS_CONTAINER_NAME:-vrooli-obs-studio}"
OBS_WEBSOCKET_PORT="$(ports::get_resource_port "obs-studio" 4455)"
OBS_VNC_PORT="${OBS_VNC_PORT:-5900}"
OBS_NOVNC_PORT="${OBS_NOVNC_PORT:-6080}"

# Directories
OBS_DATA_DIR="${HOME}/.vrooli/obs-studio"
OBS_CONFIG_DIR="${OBS_DATA_DIR}/config"
OBS_SCENES_DIR="${OBS_DATA_DIR}/scenes"
OBS_RECORDINGS_DIR="${OBS_DATA_DIR}/recordings"
OBS_DOCKER_DIR="${OBS_DATA_DIR}/docker"

# Check if Docker is available
docker::check_availability() {
    if ! command -v docker &>/dev/null; then
        log::error "Docker is not installed. Please install Docker first."
        return 1
    fi
    
    if ! docker info &>/dev/null; then
        log::error "Docker daemon is not running or you don't have permissions."
        return 1
    fi
    
    return 0
}

# Create Docker image with specific version
docker::build_image() {
    local version="${1:-$OBS_DOCKER_VERSION}"
    
    log::info "Building OBS Studio Docker image with version $version"
    
    # Create Docker directory
    mkdir -p "$OBS_DOCKER_DIR"
    
    # Create optimized Dockerfile with specific versions
    cat > "$OBS_DOCKER_DIR/Dockerfile" <<EOF
FROM ubuntu:22.04

ARG OBS_VERSION=${version}
ENV DEBIAN_FRONTEND=noninteractive
ENV DISPLAY=:99
ENV OBS_VERSION=\${OBS_VERSION}

# Install dependencies with version pinning
RUN apt-get update && apt-get install -y \\
    software-properties-common=0.99.22.* \\
    wget=1.21.* \\
    curl=7.81.* \\
    xvfb \\
    x11vnc \\
    novnc \\
    supervisor \\
    net-tools \\
    ffmpeg \\
    v4l2loopback-dkms \\
    v4l2loopback-utils \\
    python3-pip \\
    && add-apt-repository -y ppa:obsproject/obs-studio \\
    && apt-get update \\
    && apt-get install -y obs-studio=\${OBS_VERSION}* \\
    && apt-get clean \\
    && rm -rf /var/lib/apt/lists/*

# Install obs-websocket Python client
RUN pip3 install obs-websocket-py==1.0.2 simpleobsws==1.3.5

# Create non-root user
RUN groupadd -g 1000 obs && \\
    useradd -m -u 1000 -g obs -s /bin/bash obs

# Setup VNC with secure password
RUN mkdir -p /home/obs/.vnc && \\
    x11vnc -storepasswd \${OBS_VNC_PASSWORD:-obspass123} /home/obs/.vnc/passwd && \\
    chown -R obs:obs /home/obs

# Create necessary directories
RUN mkdir -p /home/obs/.config/obs-studio && \\
    mkdir -p /home/obs/recordings && \\
    mkdir -p /home/obs/scenes && \\
    mkdir -p /var/log/supervisor && \\
    chown -R obs:obs /home/obs

# Copy supervisor configuration
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf

# Copy health check script
COPY health_check.sh /usr/local/bin/health_check.sh
RUN chmod +x /usr/local/bin/health_check.sh

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=60s --retries=3 \\
    CMD /usr/local/bin/health_check.sh || exit 1

# Expose ports
EXPOSE ${OBS_WEBSOCKET_PORT} ${OBS_VNC_PORT} ${OBS_NOVNC_PORT}

# Volume mounts
VOLUME ["/home/obs/.config/obs-studio", "/home/obs/recordings", "/home/obs/scenes"]

# Set user and working directory
USER obs
WORKDIR /home/obs

# Entry point
ENTRYPOINT ["/usr/bin/supervisord"]
CMD ["-c", "/etc/supervisor/conf.d/supervisord.conf"]
EOF

    # Create supervisor configuration
    cat > "$OBS_DOCKER_DIR/supervisord.conf" <<EOF
[supervisord]
nodaemon=true
user=obs
logfile=/home/obs/supervisord.log
pidfile=/home/obs/supervisord.pid

[program:xvfb]
command=/usr/bin/Xvfb :99 -screen 0 1920x1080x24 -ac +extension GLX +render -noreset
autorestart=true
user=obs
priority=1
stdout_logfile=/home/obs/xvfb.log
stderr_logfile=/home/obs/xvfb.err

[program:x11vnc]
command=/usr/bin/x11vnc -display :99 -forever -usepw -shared -rfbport ${OBS_VNC_PORT} -rfbauth /home/obs/.vnc/passwd -bg -o /home/obs/x11vnc.log
autorestart=true
user=obs
priority=2
stdout_logfile=/home/obs/x11vnc.log
stderr_logfile=/home/obs/x11vnc.err

[program:novnc]
command=/usr/share/novnc/utils/launch.sh --vnc localhost:${OBS_VNC_PORT} --listen ${OBS_NOVNC_PORT}
autorestart=true
user=obs
priority=3
stdout_logfile=/home/obs/novnc.log
stderr_logfile=/home/obs/novnc.err

[program:obs]
command=/usr/bin/obs --startstreaming --minimize-to-tray
autorestart=true
user=obs
environment=DISPLAY=":99",HOME="/home/obs"
priority=10
stdout_logfile=/home/obs/obs.log
stderr_logfile=/home/obs/obs.err
EOF

    # Create health check script
    cat > "$OBS_DOCKER_DIR/health_check.sh" <<'EOF'
#!/bin/bash
# Health check for OBS Studio Docker container

# Check if X server is running
if ! pgrep -x Xvfb > /dev/null; then
    echo "Xvfb is not running"
    exit 1
fi

# Check if VNC server is running
if ! pgrep -x x11vnc > /dev/null; then
    echo "x11vnc is not running"
    exit 1
fi

# Check if OBS is running
if ! pgrep -x obs > /dev/null; then
    echo "OBS Studio is not running"
    exit 1
fi

# Check WebSocket port
if ! nc -z localhost ${OBS_WEBSOCKET_PORT:-4455} 2>/dev/null; then
    echo "WebSocket port is not accessible"
    exit 1
fi

echo "Health check passed"
exit 0
EOF

    # Build the Docker image
    cd "$OBS_DOCKER_DIR"
    if docker build -t "vrooli/obs-studio:${version}" -t "vrooli/obs-studio:latest" .; then
        log::success "Docker image built successfully: vrooli/obs-studio:${version}"
        return 0
    else
        log::error "Failed to build Docker image"
        return 1
    fi
}

# Run OBS Studio container
docker::run_container() {
    local version="${1:-$OBS_DOCKER_VERSION}"
    
    if ! docker::check_availability; then
        return 1
    fi
    
    # Check if container already exists
    if docker ps -a --format '{{.Names}}' | grep -q "^${OBS_CONTAINER_NAME}$"; then
        log::info "Container ${OBS_CONTAINER_NAME} already exists"
        
        # Check if running
        if docker ps --format '{{.Names}}' | grep -q "^${OBS_CONTAINER_NAME}$"; then
            log::info "Container is already running"
            return 0
        else
            log::info "Starting existing container"
            docker start "${OBS_CONTAINER_NAME}"
            return $?
        fi
    fi
    
    # Create container with specific version
    log::info "Creating new OBS Studio container with version ${version}"
    
    docker run -d \
        --name "${OBS_CONTAINER_NAME}" \
        --restart unless-stopped \
        -p "${OBS_WEBSOCKET_PORT}:${OBS_WEBSOCKET_PORT}" \
        -p "${OBS_VNC_PORT}:${OBS_VNC_PORT}" \
        -p "${OBS_NOVNC_PORT}:${OBS_NOVNC_PORT}" \
        -v "${OBS_CONFIG_DIR}:/home/obs/.config/obs-studio" \
        -v "${OBS_RECORDINGS_DIR}:/home/obs/recordings" \
        -v "${OBS_SCENES_DIR}:/home/obs/scenes" \
        -e "OBS_WEBSOCKET_PORT=${OBS_WEBSOCKET_PORT}" \
        -e "OBS_VNC_PORT=${OBS_VNC_PORT}" \
        -e "OBS_NOVNC_PORT=${OBS_NOVNC_PORT}" \
        -e "OBS_VNC_PASSWORD=${OBS_VNC_PASSWORD:-obspass123}" \
        --shm-size=2g \
        --device=/dev/dri:/dev/dri \
        "vrooli/obs-studio:${version}"
    
    if [[ $? -eq 0 ]]; then
        log::success "OBS Studio container started successfully"
        log::info "VNC access: vnc://localhost:${OBS_VNC_PORT}"
        log::info "Web VNC access: http://localhost:${OBS_NOVNC_PORT}"
        log::info "WebSocket API: ws://localhost:${OBS_WEBSOCKET_PORT}"
        return 0
    else
        log::error "Failed to start OBS Studio container"
        return 1
    fi
}

# Stop OBS Studio container
docker::stop_container() {
    if ! docker::check_availability; then
        return 1
    fi
    
    if docker ps --format '{{.Names}}' | grep -q "^${OBS_CONTAINER_NAME}$"; then
        log::info "Stopping OBS Studio container"
        docker stop "${OBS_CONTAINER_NAME}"
        return $?
    else
        log::info "Container ${OBS_CONTAINER_NAME} is not running"
        return 0
    fi
}

# Remove OBS Studio container
docker::remove_container() {
    if ! docker::check_availability; then
        return 1
    fi
    
    if docker ps -a --format '{{.Names}}' | grep -q "^${OBS_CONTAINER_NAME}$"; then
        log::info "Removing OBS Studio container"
        docker::stop_container
        docker rm "${OBS_CONTAINER_NAME}"
        return $?
    else
        log::info "Container ${OBS_CONTAINER_NAME} does not exist"
        return 0
    fi
}

# Get container status
docker::get_status() {
    if ! docker::check_availability; then
        echo "Docker not available"
        return 1
    fi
    
    if docker ps --format '{{.Names}}' | grep -q "^${OBS_CONTAINER_NAME}$"; then
        echo "Running"
        
        # Get container health status
        local health=$(docker inspect --format='{{.State.Health.Status}}' "${OBS_CONTAINER_NAME}" 2>/dev/null)
        if [[ -n "$health" ]]; then
            echo "Health: $health"
        fi
        
        # Get resource usage
        docker stats --no-stream --format "CPU: {{.CPUPerc}}, Memory: {{.MemUsage}}" "${OBS_CONTAINER_NAME}"
        
        return 0
    elif docker ps -a --format '{{.Names}}' | grep -q "^${OBS_CONTAINER_NAME}$"; then
        echo "Stopped"
        return 1
    else
        echo "Not installed"
        return 1
    fi
}

# Get container logs
docker::get_logs() {
    local lines="${1:-50}"
    
    if ! docker::check_availability; then
        return 1
    fi
    
    if docker ps -a --format '{{.Names}}' | grep -q "^${OBS_CONTAINER_NAME}$"; then
        docker logs --tail "$lines" "${OBS_CONTAINER_NAME}"
        return 0
    else
        log::error "Container ${OBS_CONTAINER_NAME} does not exist"
        return 1
    fi
}

# Cleanup Docker resources
docker::cleanup() {
    log::info "Cleaning up OBS Studio Docker resources"
    
    # Stop and remove container
    docker::remove_container
    
    # Remove images
    docker rmi "vrooli/obs-studio:${OBS_DOCKER_VERSION}" 2>/dev/null || true
    docker rmi "vrooli/obs-studio:latest" 2>/dev/null || true
    
    # Clean up build cache
    docker builder prune -f 2>/dev/null || true
    
    log::success "Docker cleanup completed"
    return 0
}

# Export functions
export -f docker::check_availability
export -f docker::build_image
export -f docker::run_container
export -f docker::stop_container
export -f docker::remove_container
export -f docker::get_status
export -f docker::get_logs
export -f docker::cleanup