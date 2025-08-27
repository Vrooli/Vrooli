#!/usr/bin/env bash
################################################################################
# OBS Studio Resource Configuration Defaults
# 
# Default configuration values for the OBS Studio resource
################################################################################

# Core Configuration
OBS_CONFIG_DIR="${HOME}/.vrooli/obs-studio"
OBS_CONFIG_FILE="${OBS_CONFIG_DIR}/config.json"
OBS_PORT="${OBS_PORT:-4455}"
OBS_PASSWORD_FILE="${OBS_CONFIG_DIR}/websocket-password"
OBS_SCENE_CONFIG="${OBS_CONFIG_DIR}/scenes.json"
OBS_RECORDINGS_DIR="${HOME}/Videos/obs-recordings"
OBS_CONTAINER_NAME="obs-studio-main"
OBS_WEBSOCKET_VERSION="5.5.0"

# Installation method
OBS_INSTALL_METHOD="${OBS_INSTALL_METHOD:-mock}"

# Resource directories
OBS_CLI_DIR="${APP_ROOT}/resources/obs-studio"
OBS_LIB_DIR="${OBS_CLI_DIR}/lib"
OBS_RESOURCE_DIR="${OBS_CLI_DIR}"

# Ensure configuration directories exist
mkdir -p "${OBS_CONFIG_DIR}"
mkdir -p "${OBS_RECORDINGS_DIR}"