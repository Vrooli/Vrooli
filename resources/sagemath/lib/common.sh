#!/bin/bash

# Common variables and functions for SageMath resource

# Container configuration
SAGEMATH_CONTAINER_NAME="sagemath-main"
SAGEMATH_IMAGE="sagemath/sagemath:latest"
SAGEMATH_PORT_JUPYTER="${SAGEMATH_PORT_JUPYTER:-8888}"
SAGEMATH_PORT_API="${SAGEMATH_PORT_API:-8889}"

# Data directories
SAGEMATH_DATA_DIR="${var_DATA_DIR:-/home/matthalloran8/Vrooli/data}/sagemath"
SAGEMATH_SCRIPTS_DIR="$SAGEMATH_DATA_DIR/scripts"
SAGEMATH_NOTEBOOKS_DIR="$SAGEMATH_DATA_DIR/notebooks"
SAGEMATH_OUTPUTS_DIR="$SAGEMATH_DATA_DIR/outputs"
SAGEMATH_CONFIG_DIR="$SAGEMATH_DATA_DIR/config"

# Resource metadata
SAGEMATH_RESOURCE_NAME="sagemath"
SAGEMATH_RESOURCE_CATEGORY="execution"
SAGEMATH_RESOURCE_DESCRIPTION="Open-source mathematics software system"

# Ensure data directories exist
sagemath_ensure_directories() {
    mkdir -p "$SAGEMATH_SCRIPTS_DIR"
    mkdir -p "$SAGEMATH_NOTEBOOKS_DIR"
    mkdir -p "$SAGEMATH_OUTPUTS_DIR"
    mkdir -p "$SAGEMATH_CONFIG_DIR"
}

# Check if container exists
sagemath_container_exists() {
    docker ps -a --format "{{.Names}}" | grep -q "^${SAGEMATH_CONTAINER_NAME}$"
}

# Check if container is running
sagemath_container_running() {
    docker ps --format "{{.Names}}" | grep -q "^${SAGEMATH_CONTAINER_NAME}$"
}

# Get container ID
sagemath_get_container_id() {
    docker ps -aq -f "name=${SAGEMATH_CONTAINER_NAME}"
}

# Format output based on requested format
sagemath_format_output() {
    local format="${1:-text}"
    local content="$2"
    
    if [[ "$format" == "json" ]]; then
        echo "$content"
    else
        echo "$content" | jq -r '.'
    fi
}