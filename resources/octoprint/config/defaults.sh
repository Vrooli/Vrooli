#!/bin/bash

# OctoPrint Resource - Default Configuration
# Web-based 3D printer management platform

# Service Configuration
export OCTOPRINT_PORT="${OCTOPRINT_PORT:-8197}"
export OCTOPRINT_HOST="${OCTOPRINT_HOST:-0.0.0.0}"
export OCTOPRINT_NAME="${OCTOPRINT_NAME:-octoprint}"

# Printer Configuration
export OCTOPRINT_PRINTER_PORT="${OCTOPRINT_PRINTER_PORT:-/dev/ttyUSB0}"
export OCTOPRINT_BAUDRATE="${OCTOPRINT_BAUDRATE:-115200}"
export OCTOPRINT_VIRTUAL_PRINTER="${OCTOPRINT_VIRTUAL_PRINTER:-true}"  # Default to virtual for testing

# API Configuration
export OCTOPRINT_API_KEY="${OCTOPRINT_API_KEY:-auto}"
export OCTOPRINT_API_ENABLED="${OCTOPRINT_API_ENABLED:-true}"

# Camera Configuration (optional)
export OCTOPRINT_CAMERA_ENABLED="${OCTOPRINT_CAMERA_ENABLED:-false}"
export OCTOPRINT_CAMERA_URL="${OCTOPRINT_CAMERA_URL:-http://localhost:8080/?action=stream}"
export OCTOPRINT_CAMERA_SNAPSHOT_URL="${OCTOPRINT_CAMERA_SNAPSHOT_URL:-http://localhost:8080/?action=snapshot}"

# File Management
export OCTOPRINT_UPLOAD_MAX_SIZE="${OCTOPRINT_UPLOAD_MAX_SIZE:-524288000}"  # 500MB
export OCTOPRINT_GCODE_DIR="${OCTOPRINT_GCODE_DIR:-${HOME}/.vrooli/resources/octoprint/gcode}"
export OCTOPRINT_TIMELAPSE_DIR="${OCTOPRINT_TIMELAPSE_DIR:-${HOME}/.vrooli/resources/octoprint/timelapse}"

# Performance Settings
export OCTOPRINT_POLL_INTERVAL="${OCTOPRINT_POLL_INTERVAL:-1}"  # Temperature polling in seconds
export OCTOPRINT_WEBSOCKET_ENABLED="${OCTOPRINT_WEBSOCKET_ENABLED:-true}"

# Docker Settings
export OCTOPRINT_DOCKER_IMAGE="${OCTOPRINT_DOCKER_IMAGE:-octoprint/octoprint:latest}"
export OCTOPRINT_CONTAINER_NAME="${OCTOPRINT_CONTAINER_NAME:-vrooli-octoprint}"

# Installation Settings
export OCTOPRINT_INSTALL_DIR="${OCTOPRINT_INSTALL_DIR:-${HOME}/.vrooli/resources/octoprint}"
export OCTOPRINT_DATA_DIR="${OCTOPRINT_DATA_DIR:-${HOME}/.vrooli/resources/octoprint/data}"
export OCTOPRINT_CONFIG_DIR="${OCTOPRINT_CONFIG_DIR:-${HOME}/.vrooli/resources/octoprint/config}"

# Timeout Settings
export OCTOPRINT_STARTUP_TIMEOUT="${OCTOPRINT_STARTUP_TIMEOUT:-60}"
export OCTOPRINT_SHUTDOWN_TIMEOUT="${OCTOPRINT_SHUTDOWN_TIMEOUT:-30}"
export OCTOPRINT_HEALTH_CHECK_TIMEOUT="${OCTOPRINT_HEALTH_CHECK_TIMEOUT:-5}"

# Logging
export OCTOPRINT_LOG_LEVEL="${OCTOPRINT_LOG_LEVEL:-INFO}"
export OCTOPRINT_LOG_FILE="${OCTOPRINT_LOG_FILE:-${HOME}/.vrooli/logs/octoprint.log}"

# Plugin Settings
export OCTOPRINT_PLUGINS_ENABLED="${OCTOPRINT_PLUGINS_ENABLED:-true}"
export OCTOPRINT_PLUGIN_DIR="${OCTOPRINT_PLUGIN_DIR:-${HOME}/.vrooli/resources/octoprint/plugins}"