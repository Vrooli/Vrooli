#!/usr/bin/env bash
# MEEP Resource Configuration Defaults
# Port allocation: 8193 (from port_registry.sh)

# Service Configuration
export MEEP_PORT="${MEEP_PORT:-8193}"
export MEEP_HOST="${MEEP_HOST:-0.0.0.0}"
export MEEP_LOG_LEVEL="${MEEP_LOG_LEVEL:-info}"

# Container Configuration
export MEEP_CONTAINER_NAME="${MEEP_CONTAINER_NAME:-meep-server}"
export MEEP_IMAGE_NAME="${MEEP_IMAGE_NAME:-vrooli/meep}"
export MEEP_IMAGE_TAG="${MEEP_IMAGE_TAG:-latest}"

# Resource Limits
export MEEP_MEMORY_LIMIT="${MEEP_MEMORY_LIMIT:-4g}"
export MEEP_CPU_LIMIT="${MEEP_CPU_LIMIT:-4}"

# MPI Configuration
export MEEP_MPI_PROCESSES="${MEEP_MPI_PROCESSES:-4}"
export MEEP_OMP_NUM_THREADS="${MEEP_OMP_NUM_THREADS:-2}"

# Storage Configuration
export MEEP_DATA_DIR="${MEEP_DATA_DIR:-${HOME}/.vrooli/resources/meep/data}"
export MEEP_RESULTS_DIR="${MEEP_RESULTS_DIR:-${MEEP_DATA_DIR}/results}"
export MEEP_TEMPLATES_DIR="${MEEP_TEMPLATES_DIR:-${MEEP_DATA_DIR}/templates}"

# Simulation Defaults
export MEEP_DEFAULT_RESOLUTION="${MEEP_DEFAULT_RESOLUTION:-50}"
export MEEP_DEFAULT_RUNTIME="${MEEP_DEFAULT_RUNTIME:-100}"
export MEEP_MAX_SIMULATION_TIME="${MEEP_MAX_SIMULATION_TIME:-3600}"

# Integration Configuration
export MEEP_MINIO_ENABLED="${MEEP_MINIO_ENABLED:-false}"
export MEEP_POSTGRES_ENABLED="${MEEP_POSTGRES_ENABLED:-false}"
export MEEP_QUESTDB_ENABLED="${MEEP_QUESTDB_ENABLED:-false}"
export MEEP_QDRANT_ENABLED="${MEEP_QDRANT_ENABLED:-false}"

# GPU Configuration
export MEEP_GPU_ENABLED="${MEEP_GPU_ENABLED:-false}"
export MEEP_CUDA_DEVICE="${MEEP_CUDA_DEVICE:-0}"

# Health Check Configuration
export MEEP_HEALTH_CHECK_INTERVAL="${MEEP_HEALTH_CHECK_INTERVAL:-30}"
export MEEP_HEALTH_CHECK_TIMEOUT="${MEEP_HEALTH_CHECK_TIMEOUT:-5}"
export MEEP_STARTUP_TIMEOUT="${MEEP_STARTUP_TIMEOUT:-60}"

# API Configuration
export MEEP_API_MAX_UPLOAD_SIZE="${MEEP_API_MAX_UPLOAD_SIZE:-100M}"
export MEEP_API_REQUEST_TIMEOUT="${MEEP_API_REQUEST_TIMEOUT:-300}"

# Debug Configuration
export MEEP_DEBUG="${MEEP_DEBUG:-false}"
export MEEP_VERBOSE="${MEEP_VERBOSE:-false}"