#!/bin/bash
# Blender default configuration

# Export default values
export BLENDER_VERSION="${BLENDER_VERSION:-4.0}"
export BLENDER_DATA_DIR="${BLENDER_DATA_DIR:-${HOME}/.vrooli/blender}"
export BLENDER_SCRIPTS_DIR="${BLENDER_SCRIPTS_DIR:-${BLENDER_DATA_DIR}/scripts}"
export BLENDER_OUTPUT_DIR="${BLENDER_OUTPUT_DIR:-${BLENDER_DATA_DIR}/output}"
export BLENDER_CONTAINER_NAME="${BLENDER_CONTAINER_NAME:-vrooli-blender}"
export BLENDER_PORT="${BLENDER_PORT:-8091}"

# Resource metadata
export BLENDER_RESOURCE_NAME="blender"
export BLENDER_RESOURCE_CATEGORY="execution"
export BLENDER_RESOURCE_DESCRIPTION="Professional 3D creation suite with Python API"