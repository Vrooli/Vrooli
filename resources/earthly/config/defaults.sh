#!/bin/bash
# Earthly Resource - Default Configuration
# v2.0 Contract Compliant Configuration

# Earthly version and installation
export EARTHLY_VERSION="${EARTHLY_VERSION:-0.8.15}"
export EARTHLY_INSTALL_DIR="${EARTHLY_INSTALL_DIR:-/usr/local/bin}"
export EARTHLY_DOWNLOAD_URL="https://github.com/earthly/earthly/releases/download/v${EARTHLY_VERSION}/earthly-linux-amd64"

# Earthly directories
export EARTHLY_HOME="${EARTHLY_HOME:-${HOME}/.earthly}"
export EARTHLY_CONFIG_DIR="${EARTHLY_CONFIG_DIR:-${EARTHLY_HOME}/config}"
export EARTHLY_CACHE_DIR="${EARTHLY_CACHE_DIR:-${EARTHLY_HOME}/cache}"
export EARTHLY_ARTIFACTS_DIR="${EARTHLY_ARTIFACTS_DIR:-${EARTHLY_HOME}/artifacts}"
export EARTHLY_LOGS_DIR="${EARTHLY_LOGS_DIR:-${EARTHLY_HOME}/logs}"

# Build configuration
export EARTHLY_PARALLEL_LIMIT="${EARTHLY_PARALLEL_LIMIT:-$(nproc)}"
export EARTHLY_BUILD_TIMEOUT="${EARTHLY_BUILD_TIMEOUT:-3600}"
export EARTHLY_CACHE_SIZE_MB="${EARTHLY_CACHE_SIZE_MB:-10240}"
export EARTHLY_USE_INLINE_CACHE="${EARTHLY_USE_INLINE_CACHE:-true}"

# Docker integration
export EARTHLY_DOCKER_HOST="${EARTHLY_DOCKER_HOST:-unix:///var/run/docker.sock}"
export EARTHLY_BUILDKIT_HOST="${EARTHLY_BUILDKIT_HOST:-}"

# Remote cache configuration (optional)
export EARTHLY_REMOTE_CACHE="${EARTHLY_REMOTE_CACHE:-}"
export EARTHLY_REMOTE_CACHE_ENDPOINT="${EARTHLY_REMOTE_CACHE_ENDPOINT:-}"

# Satellite configuration (optional)
export EARTHLY_SATELLITE_NAME="${EARTHLY_SATELLITE_NAME:-}"
export EARTHLY_SATELLITE_ORG="${EARTHLY_SATELLITE_ORG:-}"

# Security settings
export EARTHLY_DISABLE_ANALYTICS="${EARTHLY_DISABLE_ANALYTICS:-true}"
export EARTHLY_DISABLE_REMOTE_REGISTRY_PROXY="${EARTHLY_DISABLE_REMOTE_REGISTRY_PROXY:-false}"

# Logging configuration
export EARTHLY_LOG_LEVEL="${EARTHLY_LOG_LEVEL:-info}"
export EARTHLY_LOG_FILE="${EARTHLY_LOG_FILE:-${EARTHLY_LOGS_DIR}/earthly.log}"
export EARTHLY_MAX_LOG_SIZE="${EARTHLY_MAX_LOG_SIZE:-100M}"
export EARTHLY_MAX_LOG_AGE="${EARTHLY_MAX_LOG_AGE:-7}"

# Performance tuning
export EARTHLY_MAX_PARALLEL_STEPS="${EARTHLY_MAX_PARALLEL_STEPS:-20}"
export EARTHLY_CACHE_INLINE_SIZE_MB="${EARTHLY_CACHE_INLINE_SIZE_MB:-100}"
export EARTHLY_FRONTEND_IMAGE="${EARTHLY_FRONTEND_IMAGE:-}"

# Health check settings
export EARTHLY_HEALTH_CHECK_INTERVAL="${EARTHLY_HEALTH_CHECK_INTERVAL:-30}"
export EARTHLY_HEALTH_CHECK_TIMEOUT="${EARTHLY_HEALTH_CHECK_TIMEOUT:-5}"

# Artifact management
export EARTHLY_MAX_ARTIFACT_SIZE_MB="${EARTHLY_MAX_ARTIFACT_SIZE_MB:-1024}"
export EARTHLY_ARTIFACT_RETENTION_DAYS="${EARTHLY_ARTIFACT_RETENTION_DAYS:-7}"

# Platform support
# Platform specification moved to individual builds to avoid command-line conflicts
# export EARTHLY_PLATFORMS="${EARTHLY_PLATFORMS:-linux/amd64,linux/arm64}"

# Resource limits
export EARTHLY_CPU_LIMIT="${EARTHLY_CPU_LIMIT:-}"
export EARTHLY_MEMORY_LIMIT="${EARTHLY_MEMORY_LIMIT:-}"

# Default Earthfile location
export EARTHLY_DEFAULT_EARTHFILE="${EARTHLY_DEFAULT_EARTHFILE:-./Earthfile}"