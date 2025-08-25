#!/bin/bash

# Installation script for OBS Studio
set -euo pipefail

# Define directory using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
OBS_INSTALL_DIR="${APP_ROOT}/resources/obs-studio/lib"
source "${OBS_INSTALL_DIR}/common.sh"

# Installation method (native or docker)
# Note: OBS Studio requires GUI access, so mock mode is used for testing
INSTALL_METHOD="${OBS_INSTALL_METHOD:-mock}"

# Install OBS Studio natively
install_obs_native() {
    echo "[INFO] Installing OBS Studio natively..."
    
    # Detect OS
    if [[ -f /etc/debian_version ]]; then
        # Debian/Ubuntu
        echo "[INFO] Detected Debian/Ubuntu system"
        
        # Add OBS Studio PPA if on Ubuntu
        if command -v lsb_release >/dev/null 2>&1; then
            local distro=$(lsb_release -si)
            if [[ "${distro}" == "Ubuntu" ]]; then
                sudo add-apt-repository -y ppa:obsproject/obs-studio || true
            fi
        fi
        
        # Update and install
        sudo apt-get update
        sudo apt-get install -y obs-studio
        
    elif [[ -f /etc/fedora-release ]]; then
        # Fedora
        echo "[INFO] Detected Fedora system"
        sudo dnf install -y obs-studio
        
    elif [[ -f /etc/arch-release ]]; then
        # Arch
        echo "[INFO] Detected Arch system"
        sudo pacman -Sy --noconfirm obs-studio
        
    else
        echo "[ERROR] Unsupported operating system"
        return 1
    fi
    
    echo "[SUCCESS] OBS Studio installed successfully"
}

# Install OBS Studio via Docker
install_obs_docker() {
    echo "[INFO] Installing OBS Studio via Docker..."
    
    # Create a Dockerfile for OBS Studio
    echo "[INFO] Creating OBS Studio Docker container..."
    
    cat > "${OBS_CONFIG_DIR}/Dockerfile" <<'EOF'
FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive
ENV DISPLAY=:99

# Install dependencies
RUN apt-get update && apt-get install -y \
    software-properties-common \
    wget \
    curl \
    xvfb \
    x11vnc \
    novnc \
    supervisor \
    net-tools \
    && add-apt-repository -y ppa:obsproject/obs-studio \
    && apt-get update \
    && apt-get install -y obs-studio \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Create user
RUN useradd -m -s /bin/bash obsuser

# Setup VNC
RUN mkdir -p /home/obsuser/.vnc \
    && x11vnc -storepasswd obspass /home/obsuser/.vnc/passwd \
    && chown -R obsuser:obsuser /home/obsuser

# Supervisor config
RUN mkdir -p /var/log/supervisor
COPY --chown=obsuser:obsuser supervisord.conf /etc/supervisor/conf.d/supervisord.conf

EXPOSE 4455 5900 6080

USER obsuser
WORKDIR /home/obsuser

CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
EOF

    # Create supervisor config
    cat > "${OBS_CONFIG_DIR}/supervisord.conf" <<'EOF'
[supervisord]
nodaemon=true
user=obsuser

[program:xvfb]
command=/usr/bin/Xvfb :99 -screen 0 1920x1080x24
autorestart=true
user=obsuser

[program:x11vnc]
command=/usr/bin/x11vnc -display :99 -forever -usepw -shared -rfbport 5900 -rfbauth /home/obsuser/.vnc/passwd
autorestart=true
user=obsuser

[program:novnc]
command=/usr/share/novnc/utils/launch.sh --vnc localhost:5900 --listen 6080
autorestart=true
user=obsuser
EOF

    # Build and run container
    cd "${OBS_CONFIG_DIR}"
    
    echo "[INFO] Building OBS Studio Docker image..."
    docker build -t obs-studio-vrooli .
    
    echo "[INFO] Starting OBS Studio container..."
    docker run -d \
        --name "${OBS_CONTAINER_NAME}" \
        -p "${OBS_PORT}:${OBS_PORT}" \
        -p 5900:5900 \
        -p 6080:6080 \
        -v "${OBS_CONFIG_DIR}:/home/obsuser/.config/obs-studio" \
        -v "${OBS_RECORDINGS_DIR}:/home/obsuser/recordings" \
        --restart unless-stopped \
        obs-studio-vrooli
    
    echo "[SUCCESS] OBS Studio Docker container created"
}

# Install WebSocket plugin
install_websocket_plugin() {
    echo "[INFO] Installing obs-websocket plugin..."
    
    local plugin_dir
    local download_url="https://github.com/obsproject/obs-websocket/releases/download/${OBS_WEBSOCKET_VERSION}"
    
    if [[ "${INSTALL_METHOD}" == "native" ]]; then
        plugin_dir="${HOME}/.config/obs-studio/plugins"
    else
        plugin_dir="${OBS_CONFIG_DIR}/plugins"
    fi
    
    mkdir -p "${plugin_dir}"
    
    # Download appropriate version based on system
    local system_arch=$(uname -m)
    local plugin_file
    
    case "${system_arch}" in
        x86_64)
            plugin_file="obs-websocket-${OBS_WEBSOCKET_VERSION}-linux-x86_64.tar.gz"
            ;;
        aarch64|arm64)
            plugin_file="obs-websocket-${OBS_WEBSOCKET_VERSION}-linux-aarch64.tar.gz"
            ;;
        *)
            echo "[WARNING] Unsupported architecture: ${system_arch}"
            echo "[INFO] WebSocket plugin must be installed manually"
            return 0
            ;;
    esac
    
    # Download and extract
    local temp_dir=$(mktemp -d)
    cd "${temp_dir}"
    
    echo "[INFO] Downloading ${plugin_file}..."
    if curl -L -o "${plugin_file}" "${download_url}/${plugin_file}" 2>/dev/null; then
        tar -xzf "${plugin_file}"
        
        # Copy to plugin directory
        if [[ -d "obs-websocket" ]]; then
            cp -r obs-websocket "${plugin_dir}/"
            echo "[SUCCESS] obs-websocket plugin installed"
        else
            echo "[WARNING] Plugin extraction structure unexpected"
        fi
    else
        echo "[WARNING] Failed to download obs-websocket plugin"
        echo "[INFO] Plugin can be installed manually from: ${download_url}"
    fi
    
    # Cleanup
    cd - >/dev/null
    rm -rf "${temp_dir}"
}

# Configure WebSocket plugin
configure_websocket() {
    echo "[INFO] Configuring obs-websocket..."
    
    local password=$(obs_generate_password)
    
    # Create WebSocket configuration
    local ws_config_dir
    if [[ "${INSTALL_METHOD}" == "native" ]]; then
        ws_config_dir="${HOME}/.config/obs-studio"
    else
        ws_config_dir="${OBS_CONFIG_DIR}"
    fi
    
    mkdir -p "${ws_config_dir}/plugin_config/obs-websocket"
    
    cat > "${ws_config_dir}/plugin_config/obs-websocket/config.json" <<EOF
{
    "server_enabled": true,
    "server_port": ${OBS_PORT},
    "enable_authentication": true,
    "auth_password": "${password}",
    "enable_system_tray_alerts": false,
    "alert_on_broadcast_stream_start": false,
    "alert_on_broadcast_stream_stop": false,
    "alert_on_broadcast_record_start": false,
    "alert_on_broadcast_record_stop": false
}
EOF
    
    echo "[SUCCESS] WebSocket plugin configured"
}

# Install OBS Studio in mock mode for testing
install_obs_mock() {
    echo "[INFO] Installing OBS Studio in mock mode (for testing)..."
    
    # Create mock installation marker
    touch "${OBS_CONFIG_DIR}/.obs-installed-mock"
    
    # Create mock configuration
    obs_create_default_config
    obs_create_default_scenes
    
    echo "[SUCCESS] OBS Studio mock installation complete"
}

# Main installation function
main() {
    echo "[HEADER] 🎬 Installing OBS Studio..."
    
    # Check if already installed
    if obs_is_installed; then
        echo "[INFO] OBS Studio is already installed"
        
        # Check WebSocket plugin
        if ! obs_websocket_installed; then
            install_websocket_plugin
            configure_websocket
        else
            echo "[INFO] obs-websocket plugin already installed"
        fi
        
        return 0
    fi
    
    # Install based on method
    if [[ "${INSTALL_METHOD}" == "docker" ]]; then
        install_obs_docker
    elif [[ "${INSTALL_METHOD}" == "mock" ]]; then
        install_obs_mock
    else
        install_obs_native
    fi
    
    # Install WebSocket plugin (skip for mock)
    if [[ "${INSTALL_METHOD}" != "mock" ]]; then
        install_websocket_plugin
    else
        # Create mock WebSocket marker
        mkdir -p "${OBS_CONFIG_DIR}/plugins"
        touch "${OBS_CONFIG_DIR}/plugins/obs-websocket-mock"
    fi
    
    # Configure
    configure_websocket
    obs_create_default_config
    obs_create_default_scenes
    
    echo "[SUCCESS] ✅ OBS Studio installation complete"
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi