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

# Canonical resource storage directories.
# RESOURCE_* is injected by the Go control plane; XDG fallbacks keep standalone shell usage off repo-local paths.
sagemath_xdg_config_home="${XDG_CONFIG_HOME:-${HOME}/.config}"
sagemath_xdg_data_home="${XDG_DATA_HOME:-${HOME}/.local/share}"
sagemath_xdg_cache_home="${XDG_CACHE_HOME:-${HOME}/.cache}"
sagemath_xdg_state_home="${XDG_STATE_HOME:-${HOME}/.local/state}"

SAGEMATH_DATA_DIR="${RESOURCE_DATA_DIR:-${sagemath_xdg_data_home}/vrooli/resources/sagemath}"
SAGEMATH_CONFIG_DIR="${RESOURCE_CONFIG_DIR:-${sagemath_xdg_config_home}/vrooli/resources/sagemath}"
SAGEMATH_CACHE_DIR="${RESOURCE_CACHE_DIR:-${sagemath_xdg_cache_home}/vrooli/resources/sagemath}"
SAGEMATH_LOGS_DIR="${RESOURCE_LOGS_DIR:-${sagemath_xdg_state_home}/logs/vrooli/resources/sagemath}"
SAGEMATH_STATE_DIR="${RESOURCE_STATE_DIR:-${sagemath_xdg_state_home}/vrooli/resources/sagemath}"
SAGEMATH_SCRIPTS_DIR="${SAGEMATH_DATA_DIR}/scripts"
SAGEMATH_NOTEBOOKS_DIR="${SAGEMATH_DATA_DIR}/notebooks"
SAGEMATH_OUTPUTS_DIR="${SAGEMATH_DATA_DIR}/outputs"

# Resource metadata
SAGEMATH_RESOURCE_NAME="sagemath"
SAGEMATH_RESOURCE_CATEGORY="execution"
SAGEMATH_RESOURCE_DESCRIPTION="Open-source mathematics software system"
