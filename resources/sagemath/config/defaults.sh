#!/usr/bin/env bash
################################################################################
# SageMath Resource Configuration
# 
# Default configuration values for SageMath resource
################################################################################

# Container configuration
SAGEMATH_CONTAINER_NAME="${SAGEMATH_CONTAINER_NAME:-sagemath-main}"
SAGEMATH_IMAGE="${SAGEMATH_IMAGE:-sagemath/sagemath:latest}"

# Source port registry for centralized port management
source "${VROOLI_ROOT:-${HOME}/Vrooli}/scripts/resources/port_registry.sh"

# Use ports from registry, with fallback to existing values for compatibility
SAGEMATH_PORT_JUPYTER="${SAGEMATH_PORT_JUPYTER:-$(ports::get_resource_port 'sagemath' || echo '8888')}"
SAGEMATH_PORT_API="${SAGEMATH_PORT_API:-$(ports::get_resource_port 'sagemath-api' || echo '8889')}"

# Data directories
SAGEMATH_DATA_DIR="${var_DATA_DIR:-${VROOLI_ROOT:-${HOME}/Vrooli}/data}/resources/sagemath"
SAGEMATH_SCRIPTS_DIR="$SAGEMATH_DATA_DIR/scripts"
SAGEMATH_NOTEBOOKS_DIR="$SAGEMATH_DATA_DIR/notebooks"
SAGEMATH_OUTPUTS_DIR="$SAGEMATH_DATA_DIR/outputs"
SAGEMATH_CONFIG_DIR="$SAGEMATH_DATA_DIR/config"

# Resource metadata
SAGEMATH_RESOURCE_NAME="sagemath"
SAGEMATH_RESOURCE_CATEGORY="execution"
SAGEMATH_RESOURCE_DESCRIPTION="Open-source mathematics software system"