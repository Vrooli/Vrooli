#!/usr/bin/env bash
# Mathlib Resource - Default Configuration

# Port allocation
export MATHLIB_PORT="${MATHLIB_PORT:-11458}"

# Directories
export MATHLIB_WORK_DIR="${MATHLIB_WORK_DIR:-/tmp/mathlib}"
export MATHLIB_CACHE_DIR="${MATHLIB_CACHE_DIR:-${HOME}/.cache/mathlib}"
export MATHLIB_INSTALL_DIR="${MATHLIB_INSTALL_DIR:-${HOME}/.mathlib}"

# Timeouts
export MATHLIB_TIMEOUT="${MATHLIB_TIMEOUT:-30}"
export MATHLIB_STARTUP_TIMEOUT="${MATHLIB_STARTUP_TIMEOUT:-60}"
export MATHLIB_HEALTH_CHECK_TIMEOUT="${MATHLIB_HEALTH_CHECK_TIMEOUT:-5}"

# Performance settings
export MATHLIB_MAX_MEMORY="${MATHLIB_MAX_MEMORY:-4096}"  # MB
export MATHLIB_MAX_THREADS="${MATHLIB_MAX_THREADS:-4}"

# Lean configuration
export LEAN_PATH="${LEAN_PATH:-${MATHLIB_INSTALL_DIR}/lean}"
export LAKE_PATH="${LAKE_PATH:-${MATHLIB_INSTALL_DIR}/lake}"

# API configuration
export MATHLIB_API_KEY="${MATHLIB_API_KEY:-}"
export MATHLIB_ENABLE_API="${MATHLIB_ENABLE_API:-true}"

# Logging
export MATHLIB_LOG_LEVEL="${MATHLIB_LOG_LEVEL:-info}"
export MATHLIB_LOG_DIR="${MATHLIB_LOG_DIR:-/tmp/mathlib/logs}"
export MATHLIB_LOG_FILE="${MATHLIB_LOG_FILE:-/tmp/mathlib.log}"

# Health check
export MATHLIB_HEALTH_ENDPOINT="http://localhost:${MATHLIB_PORT}/health"

# Process management
export MATHLIB_PID_FILE="${MATHLIB_PID_FILE:-/tmp/mathlib.pid}"
export MATHLIB_PROCESS_NAME="mathlib-server"

# Feature flags
export MATHLIB_ENABLE_CACHE="${MATHLIB_ENABLE_CACHE:-true}"
export MATHLIB_ENABLE_METRICS="${MATHLIB_ENABLE_METRICS:-false}"

# Version requirements
export LEAN_MIN_VERSION="4.0.0"
export MATHLIB_MIN_VERSION="4.0.0"