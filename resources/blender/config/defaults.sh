#!/bin/bash
# Blender default configuration

# Source port registry for correct port allocation
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
if [[ -f "${APP_ROOT}/scripts/resources/port_registry.sh" ]]; then
    source "${APP_ROOT}/scripts/resources/port_registry.sh"
fi

# Export default values
export BLENDER_VERSION="${BLENDER_VERSION:-4.0}"
export BLENDER_DATA_DIR="${BLENDER_DATA_DIR:-${HOME}/.vrooli/blender}"
export BLENDER_SCRIPTS_DIR="${BLENDER_SCRIPTS_DIR:-${BLENDER_DATA_DIR}/scripts}"
export BLENDER_OUTPUT_DIR="${BLENDER_OUTPUT_DIR:-${BLENDER_DATA_DIR}/output}"
export BLENDER_CONTAINER_NAME="${BLENDER_CONTAINER_NAME:-vrooli-blender}"
# Use port from registry or fallback (per registry: blender=8093)
export BLENDER_PORT="${BLENDER_PORT:-${RESOURCE_PORTS[blender]:-8093}}"

# Resource metadata
export BLENDER_RESOURCE_NAME="blender"
export BLENDER_RESOURCE_CATEGORY="execution"
export BLENDER_RESOURCE_DESCRIPTION="Professional 3D creation suite with Python API"