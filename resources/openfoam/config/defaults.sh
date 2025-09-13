#!/bin/bash

# OpenFOAM Resource Default Configuration
# Defines standard settings for CFD simulation platform

# Source port registry for correct port allocation
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
if [[ -f "${APP_ROOT}/scripts/resources/port_registry.sh" ]]; then
    source "${APP_ROOT}/scripts/resources/port_registry.sh"
fi

# Use port from registry or fallback (per registry: openfoam=8090)
export OPENFOAM_PORT="${OPENFOAM_PORT:-${RESOURCE_PORTS[openfoam]:-8090}}"
export OPENFOAM_HOST="${OPENFOAM_HOST:-localhost}"
export OPENFOAM_CONTAINER_NAME="${OPENFOAM_CONTAINER_NAME:-openfoam}"

# Storage paths
export OPENFOAM_DATA_DIR="${OPENFOAM_DATA_DIR:-${HOME}/.vrooli/openfoam/data}"
export OPENFOAM_CASES_DIR="${OPENFOAM_CASES_DIR:-${HOME}/.vrooli/openfoam/cases}"
export OPENFOAM_RESULTS_DIR="${OPENFOAM_RESULTS_DIR:-${HOME}/.vrooli/openfoam/results}"
export OPENFOAM_TEMPLATES_DIR="${OPENFOAM_TEMPLATES_DIR:-${RESOURCE_DIR}/examples}"

# Resource limits
export OPENFOAM_MEMORY_LIMIT="${OPENFOAM_MEMORY_LIMIT:-4g}"
export OPENFOAM_CPU_LIMIT="${OPENFOAM_CPU_LIMIT:-2}"
export OPENFOAM_STORAGE_LIMIT="${OPENFOAM_STORAGE_LIMIT:-20G}"

# Solver settings
export OPENFOAM_DEFAULT_SOLVER="${OPENFOAM_DEFAULT_SOLVER:-simpleFoam}"
export OPENFOAM_MAX_ITERATIONS="${OPENFOAM_MAX_ITERATIONS:-1000}"
export OPENFOAM_CONVERGENCE_THRESHOLD="${OPENFOAM_CONVERGENCE_THRESHOLD:-1e-5}"
export OPENFOAM_PARALLEL_CORES="${OPENFOAM_PARALLEL_CORES:-4}"

# Mesh settings
export OPENFOAM_MAX_CELLS="${OPENFOAM_MAX_CELLS:-1000000}"
export OPENFOAM_MESH_QUALITY_CHECK="${OPENFOAM_MESH_QUALITY_CHECK:-true}"

# Timeouts
export OPENFOAM_STARTUP_TIMEOUT="${OPENFOAM_STARTUP_TIMEOUT:-60}"
export OPENFOAM_HEALTH_CHECK_TIMEOUT="${OPENFOAM_HEALTH_CHECK_TIMEOUT:-5}"
export OPENFOAM_SOLVER_TIMEOUT="${OPENFOAM_SOLVER_TIMEOUT:-3600}"
export OPENFOAM_MESH_TIMEOUT="${OPENFOAM_MESH_TIMEOUT:-300}"

# Docker settings
export OPENFOAM_DOCKER_IMAGE="${OPENFOAM_DOCKER_IMAGE:-opencfd/openfoam:v2312}"
export OPENFOAM_DOCKER_NETWORK="${OPENFOAM_DOCKER_NETWORK:-openfoam-net}"

# API settings
export OPENFOAM_API_ENABLED="${OPENFOAM_API_ENABLED:-true}"
export OPENFOAM_API_MAX_CONCURRENT="${OPENFOAM_API_MAX_CONCURRENT:-5}"

# ParaView integration (optional)
export OPENFOAM_PARAVIEW_ENABLED="${OPENFOAM_PARAVIEW_ENABLED:-false}"
export OPENFOAM_PARAVIEW_PORT="${OPENFOAM_PARAVIEW_PORT:-11111}"

# Logging
export OPENFOAM_LOG_LEVEL="${OPENFOAM_LOG_LEVEL:-INFO}"
export OPENFOAM_LOG_DIR="${OPENFOAM_LOG_DIR:-${HOME}/.vrooli/openfoam/logs}"

# Feature flags
export OPENFOAM_ENABLE_MPI="${OPENFOAM_ENABLE_MPI:-false}"
export OPENFOAM_ENABLE_GPU="${OPENFOAM_ENABLE_GPU:-false}"
export OPENFOAM_ENABLE_CLOUD="${OPENFOAM_ENABLE_CLOUD:-false}"
export OPENFOAM_AUTO_CLEANUP="${OPENFOAM_AUTO_CLEANUP:-true}"

# Health check settings
export OPENFOAM_HEALTH_CHECK_INTERVAL="${OPENFOAM_HEALTH_CHECK_INTERVAL:-30}"
export OPENFOAM_HEALTH_CHECK_RETRIES="${OPENFOAM_HEALTH_CHECK_RETRIES:-3}"

# Create required directories
mkdir -p "${OPENFOAM_DATA_DIR}"
mkdir -p "${OPENFOAM_CASES_DIR}"
mkdir -p "${OPENFOAM_RESULTS_DIR}"
mkdir -p "${OPENFOAM_LOG_DIR}"