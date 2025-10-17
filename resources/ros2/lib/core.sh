#!/usr/bin/env bash

# ROS2 Resource - Core Functionality

set -euo pipefail

# Ensure defaults are loaded
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/config/defaults.sh"

# Load DDS middleware functions
if [[ -f "${SCRIPT_DIR}/lib/dds.sh" ]]; then
    source "${SCRIPT_DIR}/lib/dds.sh"
fi

# Installation function
ros2_install() {
    echo "Installing ROS2 ${ROS2_DISTRO}..."
    
    # Check if already installed
    if ros2_is_installed; then
        echo "ROS2 is already installed"
        return 2  # Already installed (per v2.0 contract)
    fi
    
    # Create data directories
    mkdir -p "${ROS2_DATA_DIR}" "${ROS2_LOG_DIR}" "${ROS2_CONFIG_DIR}"
    
    if [[ "${ROS2_USE_DOCKER}" == "true" ]]; then
        echo "Installing ROS2 via Docker..."
        if ! command -v docker &>/dev/null; then
            echo "Error: Docker is not installed" >&2
            return 1
        fi
        
        # Pull ROS2 Docker image
        docker pull "${ROS2_DOCKER_IMAGE}"
        
        # Create marker file
        touch "${ROS2_DATA_DIR}/.installed"
    else
        echo "Installing ROS2 natively..."
        
        # Add ROS2 apt repository
        if [[ ! -f /etc/apt/sources.list.d/ros2.list ]]; then
            sudo apt-get update
            sudo apt-get install -y software-properties-common curl
            
            # Add ROS2 GPG key
            sudo curl -sSL https://raw.githubusercontent.com/ros/rosdistro/master/ros.key \
                -o /usr/share/keyrings/ros-archive-keyring.gpg
            
            # Add repository
            echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/ros-archive-keyring.gpg] http://packages.ros.org/ros2/ubuntu $(lsb_release -cs) main" | \
                sudo tee /etc/apt/sources.list.d/ros2.list > /dev/null
        fi
        
        # Install ROS2
        sudo apt-get update
        sudo apt-get install -y \
            ros-${ROS2_DISTRO}-desktop \
            python3-colcon-common-extensions \
            python3-rosdep \
            python3-argcomplete
        
        # Initialize rosdep
        if [[ ! -d /etc/ros/rosdep ]]; then
            sudo rosdep init
        fi
        rosdep update --rosdistro "${ROS2_DISTRO}"
        
        # Install Python dependencies for API server
        echo "Installing Python dependencies..."
        pip3 install --user fastapi uvicorn aiofiles python-multipart websockets 2>/dev/null || true
        
        # Create marker file
        touch "${ROS2_DATA_DIR}/.installed"
    fi
    
    echo "ROS2 installation complete"
    return 0
}

# Uninstall function
ros2_uninstall() {
    echo "Uninstalling ROS2..."
    
    # Stop services if running
    ros2_stop || true
    
    if [[ "${ROS2_USE_DOCKER}" == "true" ]]; then
        # Remove Docker container and image
        docker rm -f "${ROS2_CONTAINER_NAME}" 2>/dev/null || true
        
        if [[ "${1:-}" != "--keep-data" ]]; then
            docker rmi "${ROS2_DOCKER_IMAGE}" 2>/dev/null || true
        fi
    else
        # Remove native installation
        if [[ "${1:-}" != "--keep-data" ]]; then
            sudo apt-get remove -y ros-${ROS2_DISTRO}-* || true
            sudo rm -f /etc/apt/sources.list.d/ros2.list
        fi
    fi
    
    # Remove data directory unless --keep-data
    if [[ "${1:-}" != "--keep-data" ]]; then
        rm -rf "${ROS2_DATA_DIR}"
    else
        rm -f "${ROS2_DATA_DIR}/.installed"
    fi
    
    echo "ROS2 uninstalled"
    return 0
}

# Start function
ros2_start() {
    local wait_flag="${1:-}"
    echo "Starting ROS2 services..."
    
    if ! ros2_is_installed; then
        echo "Error: ROS2 is not installed. Run 'manage install' first" >&2
        return 1
    fi
    
    if ros2_is_running; then
        echo "ROS2 is already running"
        return 0
    fi
    
    # Source ROS2 environment
    if [[ -f "${ROS2_INSTALL_DIR}/setup.bash" ]]; then
        source "${ROS2_INSTALL_DIR}/setup.bash"
    fi
    
    # Configure DDS middleware
    if command -v dds_configure &>/dev/null; then
        dds_configure "${ROS2_MIDDLEWARE}"
        dds_init_discovery
    else
        # Fallback to basic configuration
        export ROS_DOMAIN_ID="${ROS2_DOMAIN_ID}"
    fi
    
    # Start ROS2 daemon or container
    if [[ "${ROS2_USE_DOCKER}" == "true" ]] && command -v docker &>/dev/null; then
        echo "Starting ROS2 Docker container..."
        docker run -d \
            --name "${ROS2_CONTAINER_NAME}" \
            --network host \
            -e ROS_DOMAIN_ID="${ROS2_DOMAIN_ID}" \
            -v "${ROS2_DATA_DIR}:/ros2_data" \
            "${ROS2_DOCKER_IMAGE}" \
            sleep infinity 2>/dev/null || {
                echo "Note: Using simulated ROS2 daemon"
                touch "${ROS2_DAEMON_PID_FILE}"
                echo "$$" > "${ROS2_DAEMON_PID_FILE}"
            }
    else
        echo "Starting ROS2 daemon (simulated)..."
        # In real implementation, would start actual ROS2 daemon
        # For now, just create a marker
        touch "${ROS2_DAEMON_PID_FILE}"
        echo "$$" > "${ROS2_DAEMON_PID_FILE}"
    fi
    
    # Start API server
    echo "Starting ROS2 API server on port ${ROS2_PORT}..."
    nohup python3 "${SCRIPT_DIR}/lib/api_server.py" > "${ROS2_LOG_DIR}/api.log" 2>&1 &
    local api_pid=$!
    echo "${api_pid}" > "${ROS2_API_PID_FILE}"
    
    # If --wait flag provided, wait for services to be ready
    if [[ "${wait_flag}" == "--wait" ]]; then
        local wait_time=0
        while [[ ${wait_time} -lt ${ROS2_STARTUP_TIMEOUT} ]]; do
            if ros2_health_check; then
                echo "ROS2 services started successfully"
                return 0
            fi
            sleep 1
            ((wait_time++))
        done
        
        echo "Error: ROS2 failed to start within ${ROS2_STARTUP_TIMEOUT} seconds" >&2
        ros2_stop
        return 1
    else
        # Give it a moment to start
        sleep 2
        if ros2_health_check; then
            echo "ROS2 services started successfully"
            return 0
        else
            echo "ROS2 services started (check status for health)"
            return 0
        fi
    fi
}

# Stop function  
ros2_stop() {
    echo "Stopping ROS2 services..."
    
    # Stop API server
    if [[ -f "${ROS2_API_PID_FILE}" ]]; then
        local api_pid=$(cat "${ROS2_API_PID_FILE}")
        if kill -0 "${api_pid}" 2>/dev/null; then
            kill "${api_pid}"
            rm -f "${ROS2_API_PID_FILE}"
        fi
    fi
    
    # Stop ROS2 daemon or container
    if [[ "${ROS2_USE_DOCKER}" == "true" ]] && command -v docker &>/dev/null; then
        echo "Stopping ROS2 Docker container..."
        docker stop "${ROS2_CONTAINER_NAME}" 2>/dev/null || true
        docker rm "${ROS2_CONTAINER_NAME}" 2>/dev/null || true
    else
        echo "Stopping ROS2 daemon..."
        if [[ -f "${ROS2_DAEMON_PID_FILE}" ]]; then
            local daemon_pid=$(cat "${ROS2_DAEMON_PID_FILE}")
            if kill -0 "${daemon_pid}" 2>/dev/null; then
                kill "${daemon_pid}" 2>/dev/null || true
            fi
            rm -f "${ROS2_DAEMON_PID_FILE}"
        fi
    fi
    
    echo "ROS2 services stopped"
    return 0
}

# Status function
ros2_status() {
    local json_output="${1:-}"
    
    if [[ "${json_output}" == "--json" ]]; then
        local status="stopped"
        local health="unknown"
        
        if ros2_is_running; then
            status="running"
            if ros2_health_check; then
                health="healthy"
            else
                health="unhealthy"
            fi
        fi
        
        cat <<EOF
{
  "status": "${status}",
  "health": "${health}",
  "port": ${ROS2_PORT},
  "domain_id": ${ROS2_DOMAIN_ID},
  "middleware": "${ROS2_MIDDLEWARE}",
  "distro": "${ROS2_DISTRO}"
}
EOF
    else
        echo "ROS2 Status"
        echo "==========="
        
        if ros2_is_installed; then
            echo "Installation: Installed (${ROS2_DISTRO})"
        else
            echo "Installation: Not installed"
        fi
        
        if ros2_is_running; then
            echo "Status: Running"
            echo "API Port: ${ROS2_PORT}"
            echo "Domain ID: ${ROS2_DOMAIN_ID}"
            echo "Middleware: ${ROS2_MIDDLEWARE}"
            
            if ros2_health_check; then
                echo "Health: Healthy"
                
                # Show DDS stats if available
                if command -v dds_get_stats &>/dev/null; then
                    echo ""
                    dds_get_stats
                fi
            else
                echo "Health: Unhealthy"
            fi
        else
            echo "Status: Stopped"
        fi
    fi
}

# Logs function
ros2_logs() {
    local follow="${1:-}"
    
    if [[ ! -d "${ROS2_LOG_DIR}" ]]; then
        echo "No logs available"
        return 0
    fi
    
    if [[ "${follow}" == "-f" || "${follow}" == "--follow" ]]; then
        tail -f "${ROS2_LOG_DIR}"/*.log 2>/dev/null || echo "No logs available"
    else
        tail -n 100 "${ROS2_LOG_DIR}"/*.log 2>/dev/null || echo "No logs available"
    fi
}

# Content management function
ros2_content() {
    local action="${1:-}"
    shift || true
    
    case "${action}" in
        list-nodes)
            # In real implementation, would use: ros2 node list
            echo "Active nodes (simulated):"
            echo "  /talker"
            echo "  /listener"
            ;;
        launch-node)
            local node_name="${1:-}"
            if [[ -z "${node_name}" ]]; then
                echo "Error: Node name required" >&2
                return 1
            fi
            echo "Launching node: ${node_name}"
            # This would launch actual ROS2 nodes in real implementation
            echo "Node ${node_name} launched (simulated)"
            ;;
        list-topics)
            # In real implementation, would use: ros2 topic list
            echo "Active topics (simulated):"
            echo "  /chatter [std_msgs/String]"
            echo "  /rosout [rcl_interfaces/msg/Log]"
            ;;
        publish)
            local topic="${1:-}"
            local message="${2:-}"
            if [[ -z "${topic}" || -z "${message}" ]]; then
                echo "Error: Topic and message required" >&2
                return 1
            fi
            echo "Publishing to ${topic}: ${message}"
            # This would publish actual messages in real implementation
            ;;
        list-services)
            # In real implementation, would use: ros2 service list
            echo "Available services (simulated):"
            echo "  /clear [std_srvs/srv/Empty]"
            echo "  /spawn [turtlesim/srv/Spawn]"
            ;;
        call-service)
            local service="${1:-}"
            if [[ -z "${service}" ]]; then
                echo "Error: Service name required" >&2
                return 1
            fi
            echo "Calling service: ${service}"
            # This would call actual services in real implementation
            ;;
        *)
            echo "Error: Unknown content action: ${action}" >&2
            return 1
            ;;
    esac
}

# Helper functions
ros2_is_installed() {
    [[ -f "${ROS2_DATA_DIR}/.installed" ]] || [[ -d "${ROS2_INSTALL_DIR}" ]]
}

ros2_is_running() {
    if [[ -f "${ROS2_API_PID_FILE}" ]]; then
        local pid=$(cat "${ROS2_API_PID_FILE}")
        kill -0 "${pid}" 2>/dev/null
    else
        false
    fi
}

ros2_health_check() {
    timeout "${ROS2_HEALTH_CHECK_TIMEOUT}" curl -sf "http://localhost:${ROS2_PORT}/health" &>/dev/null
}