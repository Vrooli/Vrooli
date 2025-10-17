#!/bin/bash
# Elmer FEM Default Configuration

# Service configuration
export ELMER_FEM_PORT="${ELMER_FEM_PORT:-8192}"
export ELMER_FEM_HOST="${ELMER_FEM_HOST:-0.0.0.0}"
export ELMER_FEM_DATA_DIR="${VROOLI_DATA:-${HOME}/.vrooli/data}/elmer-fem"

# Solver configuration
export ELMER_MPI_PROCESSES="${ELMER_MPI_PROCESSES:-4}"
export ELMER_MAX_MEMORY="${ELMER_MAX_MEMORY:-4G}"
export ELMER_MAX_ITERATIONS="${ELMER_MAX_ITERATIONS:-1000}"
export ELMER_CONVERGENCE_TOL="${ELMER_CONVERGENCE_TOL:-1e-6}"

# Docker configuration
export ELMER_CONTAINER_NAME="${ELMER_CONTAINER_NAME:-vrooli-elmer-fem}"
export ELMER_IMAGE="${ELMER_IMAGE:-vrooli/elmer-fem:latest}"

# Timeout settings
export ELMER_STARTUP_TIMEOUT="${ELMER_STARTUP_TIMEOUT:-60}"
export ELMER_SOLVE_TIMEOUT="${ELMER_SOLVE_TIMEOUT:-600}"
export ELMER_SHUTDOWN_TIMEOUT="${ELMER_SHUTDOWN_TIMEOUT:-30}"

# Integration settings
export ELMER_MINIO_BUCKET="${ELMER_MINIO_BUCKET:-elmer-results}"
export ELMER_POSTGRES_DB="${ELMER_POSTGRES_DB:-elmer_metadata}"
export ELMER_QUESTDB_TABLE="${ELMER_QUESTDB_TABLE:-elmer_timeseries}"
export ELMER_QDRANT_COLLECTION="${ELMER_QDRANT_COLLECTION:-elmer_patterns}"

# Logging
export ELMER_LOG_LEVEL="${ELMER_LOG_LEVEL:-INFO}"
export ELMER_LOG_FILE="${ELMER_FEM_DATA_DIR}/elmer.log"