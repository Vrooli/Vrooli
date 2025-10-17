#!/usr/bin/env bash
# CNCjs Default Configuration
# Port 8194 allocated in scripts/resources/port_registry.sh

# Source guard to prevent re-sourcing
[[ -n "${_CNCJS_DEFAULTS_SOURCED:-}" ]] && return 0
export _CNCJS_DEFAULTS_SOURCED=1

# Container configuration
readonly CNCJS_CONTAINER_NAME="cncjs"
readonly CNCJS_IMAGE="cncjs/cncjs:latest"
readonly CNCJS_PORT="${CNCJS_PORT:-8194}"
readonly CNCJS_HOST="${CNCJS_HOST:-0.0.0.0}"

# Paths
readonly CNCJS_DATA_DIR="${CNCJS_DATA_DIR:-${HOME}/.cncjs}"
readonly CNCJS_CONFIG_FILE="${CNCJS_CONFIG_FILE:-${CNCJS_DATA_DIR}/.cncrc}"
readonly CNCJS_WATCH_DIR="${CNCJS_WATCH_DIR:-${CNCJS_DATA_DIR}/watch}"

# Controller configuration
readonly CNCJS_CONTROLLER="${CNCJS_CONTROLLER:-Grbl}"  # Grbl|Marlin|Smoothie|TinyG
readonly CNCJS_SERIAL_PORT="${CNCJS_SERIAL_PORT:-/dev/ttyUSB0}"
readonly CNCJS_BAUD_RATE="${CNCJS_BAUD_RATE:-115200}"

# Security
readonly CNCJS_ALLOW_REMOTE="${CNCJS_ALLOW_REMOTE:-true}"
readonly CNCJS_ACCESS_TOKEN_LIFETIME="${CNCJS_ACCESS_TOKEN_LIFETIME:-30d}"

# Performance
readonly CNCJS_STARTUP_TIMEOUT="${CNCJS_STARTUP_TIMEOUT:-60}"
readonly CNCJS_SHUTDOWN_TIMEOUT="${CNCJS_SHUTDOWN_TIMEOUT:-30}"
readonly CNCJS_HEALTH_CHECK_INTERVAL="${CNCJS_HEALTH_CHECK_INTERVAL:-5}"

# Export for use in other scripts
export CNCJS_CONTAINER_NAME CNCJS_IMAGE CNCJS_PORT CNCJS_HOST
export CNCJS_DATA_DIR CNCJS_CONFIG_FILE CNCJS_WATCH_DIR
export CNCJS_CONTROLLER CNCJS_SERIAL_PORT CNCJS_BAUD_RATE
export CNCJS_ALLOW_REMOTE CNCJS_ACCESS_TOKEN_LIFETIME
export CNCJS_STARTUP_TIMEOUT CNCJS_SHUTDOWN_TIMEOUT CNCJS_HEALTH_CHECK_INTERVAL