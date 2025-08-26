#!/usr/bin/env bash
################################################################################
# SageMath Resource Configuration
# 
# Default configuration values for SageMath resource
################################################################################

# Container configuration
SAGEMATH_CONTAINER_NAME="${SAGEMATH_CONTAINER_NAME:-sagemath-main}"
SAGEMATH_IMAGE="${SAGEMATH_IMAGE:-sagemath/sagemath:latest}"
SAGEMATH_PORT_JUPYTER="${SAGEMATH_PORT_JUPYTER:-8888}"
SAGEMATH_PORT_API="${SAGEMATH_PORT_API:-8889}"

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