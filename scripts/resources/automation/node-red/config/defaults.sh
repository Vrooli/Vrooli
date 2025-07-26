#!/usr/bin/env bash
# Node-RED Configuration Defaults
# All configuration constants and default values

# Node-RED port configuration
readonly NODE_RED_PORT="${NODE_RED_CUSTOM_PORT:-1880}"
readonly NODE_RED_BASE_URL="http://localhost:${NODE_RED_PORT}"

# Resource metadata
readonly RESOURCE_NAME="node-red"
readonly RESOURCE_CATEGORY="automation"
readonly RESOURCE_DESC="Flow-based programming for event-driven applications"
readonly RESOURCE_PORT="${NODE_RED_PORT}"

# Container configuration
readonly CONTAINER_NAME="node-red"
readonly VOLUME_NAME="node-red-data"
readonly NETWORK_NAME="vrooli-network"

# Image configuration
readonly IMAGE_NAME="node-red-vrooli:latest"
readonly OFFICIAL_IMAGE="nodered/node-red:latest"

# Default settings
readonly DEFAULT_FLOW_FILE="flows.json"
readonly DEFAULT_SECRET=$(openssl rand -hex 32 2>/dev/null || echo "default-insecure-secret")

# Health check configuration
readonly NODE_RED_HEALTH_CHECK_INTERVAL=5
readonly NODE_RED_HEALTH_CHECK_MAX_ATTEMPTS=30
readonly NODE_RED_HEALTH_CHECK_TIMEOUT=5

# Docker health check settings
readonly DOCKER_HEALTH_INTERVAL="30s"
readonly DOCKER_HEALTH_TIMEOUT="5s"
readonly DOCKER_HEALTH_RETRIES=3

# API configuration
readonly NODE_RED_API_TIMEOUT=30

# Log configuration
readonly NODE_RED_LOG_LINES=100

# HTTP configuration
readonly HTTP_REQUEST_TIMEOUT=120000  # 2 minutes in milliseconds

# Resource configuration
readonly CONFIG_HEALTH_CHECK_INTERVAL_MS=60000
readonly CONFIG_HEALTH_CHECK_TIMEOUT_MS=5000
readonly CONFIG_BACKUP_INTERVAL="1h"

# Description for usage display
readonly DESCRIPTION="Node-RED management script for Vrooli flow-based programming"

# Feature flags
readonly NODE_RED_ENABLE_CUSTOM_IMAGE="${NODE_RED_ENABLE_CUSTOM_IMAGE:-yes}"
readonly NODE_RED_ENABLE_HOST_ACCESS="${NODE_RED_ENABLE_HOST_ACCESS:-yes}"
readonly NODE_RED_ENABLE_DOCKER_SOCKET="${NODE_RED_ENABLE_DOCKER_SOCKET:-yes}"

# Export function to make configuration available
node_red::export_config() {
    # Export all readonly variables
    export NODE_RED_PORT NODE_RED_BASE_URL
    export RESOURCE_NAME RESOURCE_CATEGORY RESOURCE_DESC RESOURCE_PORT
    export CONTAINER_NAME VOLUME_NAME NETWORK_NAME
    export IMAGE_NAME OFFICIAL_IMAGE
    export DEFAULT_FLOW_FILE DEFAULT_SECRET
    export NODE_RED_HEALTH_CHECK_INTERVAL NODE_RED_HEALTH_CHECK_MAX_ATTEMPTS NODE_RED_HEALTH_CHECK_TIMEOUT
    export DOCKER_HEALTH_INTERVAL DOCKER_HEALTH_TIMEOUT DOCKER_HEALTH_RETRIES
    export NODE_RED_API_TIMEOUT NODE_RED_LOG_LINES
    export HTTP_REQUEST_TIMEOUT
    export CONFIG_HEALTH_CHECK_INTERVAL_MS CONFIG_HEALTH_CHECK_TIMEOUT_MS CONFIG_BACKUP_INTERVAL
    export DESCRIPTION
    export NODE_RED_ENABLE_CUSTOM_IMAGE NODE_RED_ENABLE_HOST_ACCESS NODE_RED_ENABLE_DOCKER_SOCKET
}