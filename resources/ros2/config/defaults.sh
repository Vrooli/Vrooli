#!/usr/bin/env bash

# ROS2 Resource - Default Configuration

# Port configuration (use port_registry.sh pattern)
export ROS2_PORT="${ROS2_PORT:-11501}"
export ROS2_HEALTH_PORT="${ROS2_PORT}"

# ROS2 specific configuration
export ROS2_DOMAIN_ID="${ROS2_DOMAIN_ID:-0}"
export ROS2_MIDDLEWARE="${ROS2_MIDDLEWARE:-fastdds}"
export ROS2_DISTRO="${ROS2_DISTRO:-humble}"
export ROS2_WORKSPACE="${ROS2_WORKSPACE:-/opt/ros2/workspace}"

# Installation paths
export ROS2_INSTALL_DIR="${ROS2_INSTALL_DIR:-/opt/ros/${ROS2_DISTRO}}"
export ROS2_DATA_DIR="${ROS2_DATA_DIR:-${HOME}/.vrooli/ros2}"
export ROS2_LOG_DIR="${ROS2_LOG_DIR:-${ROS2_DATA_DIR}/logs}"
export ROS2_CONFIG_DIR="${ROS2_CONFIG_DIR:-${ROS2_DATA_DIR}/config}"

# Process management
export ROS2_DAEMON_PID_FILE="${ROS2_DATA_DIR}/daemon.pid"
export ROS2_API_PID_FILE="${ROS2_DATA_DIR}/api.pid"

# Timeouts
export ROS2_STARTUP_TIMEOUT="${ROS2_STARTUP_TIMEOUT:-30}"
export ROS2_SHUTDOWN_TIMEOUT="${ROS2_SHUTDOWN_TIMEOUT:-10}"
export ROS2_HEALTH_CHECK_TIMEOUT="${ROS2_HEALTH_CHECK_TIMEOUT:-5}"

# Performance settings
export ROS2_MAX_NODES="${ROS2_MAX_NODES:-100}"
export ROS2_MAX_TOPICS="${ROS2_MAX_TOPICS:-500}"
export ROS2_BUFFER_SIZE="${ROS2_BUFFER_SIZE:-65536}"

# Docker settings (if using containerized ROS2)
export ROS2_USE_DOCKER="${ROS2_USE_DOCKER:-false}"
export ROS2_DOCKER_IMAGE="${ROS2_DOCKER_IMAGE:-osrf/ros:${ROS2_DISTRO}-desktop}"
export ROS2_CONTAINER_NAME="${ROS2_CONTAINER_NAME:-vrooli-ros2}"

# Security settings
export ROS2_SECURITY_ENABLE="${ROS2_SECURITY_ENABLE:-false}"
export ROS2_SECURITY_KEYSTORE="${ROS2_SECURITY_KEYSTORE:-${ROS2_DATA_DIR}/keystore}"

# API Server settings
export ROS2_API_HOST="${ROS2_API_HOST:-0.0.0.0}"
export ROS2_API_WORKERS="${ROS2_API_WORKERS:-2}"

# Development settings
export ROS2_DEBUG="${ROS2_DEBUG:-false}"
export ROS2_VERBOSE="${ROS2_VERBOSE:-false}"

# Create necessary directories
mkdir -p "${ROS2_DATA_DIR}" "${ROS2_LOG_DIR}" "${ROS2_CONFIG_DIR}"