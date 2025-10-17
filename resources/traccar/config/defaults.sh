#!/bin/bash
# Traccar Resource Default Configuration

# Service Configuration
export TRACCAR_PORT="${TRACCAR_PORT:-8082}"
export TRACCAR_HOST="${TRACCAR_HOST:-localhost}"
export TRACCAR_VERSION="${TRACCAR_VERSION:-5.12}"

# Database Configuration
export TRACCAR_DB_HOST="${TRACCAR_DB_HOST:-localhost}"
export TRACCAR_DB_PORT="${TRACCAR_DB_PORT:-5433}"
export TRACCAR_DB_NAME="${TRACCAR_DB_NAME:-traccar}"
export TRACCAR_DB_USER="${TRACCAR_DB_USER:-traccar}"
export TRACCAR_DB_PASSWORD="${TRACCAR_DB_PASSWORD:-traccar123}"

# Admin Credentials
export TRACCAR_ADMIN_EMAIL="${TRACCAR_ADMIN_EMAIL:-admin@example.com}"
export TRACCAR_ADMIN_PASSWORD="${TRACCAR_ADMIN_PASSWORD:-admin}"

# Storage Paths
export TRACCAR_DATA_DIR="${TRACCAR_DATA_DIR:-${HOME}/.vrooli/data/traccar}"
export TRACCAR_CONFIG_DIR="${TRACCAR_CONFIG_DIR:-${TRACCAR_DATA_DIR}/config}"
export TRACCAR_LOGS_DIR="${TRACCAR_LOGS_DIR:-${TRACCAR_DATA_DIR}/logs}"
export TRACCAR_MEDIA_DIR="${TRACCAR_MEDIA_DIR:-${TRACCAR_DATA_DIR}/media}"

# Demo Data
export TRACCAR_DEMO_DEVICES="${TRACCAR_DEMO_DEVICES:-5}"
export TRACCAR_DEMO_HISTORY_DAYS="${TRACCAR_DEMO_HISTORY_DAYS:-7}"

# Docker Configuration
export TRACCAR_CONTAINER_NAME="${TRACCAR_CONTAINER_NAME:-vrooli-traccar}"
export TRACCAR_DOCKER_IMAGE="${TRACCAR_DOCKER_IMAGE:-traccar/traccar:${TRACCAR_VERSION}}"

# API Configuration
export TRACCAR_API_TIMEOUT="${TRACCAR_API_TIMEOUT:-30}"

# Webhook Configuration
export TRACCAR_WEBHOOK_ENABLED="${TRACCAR_WEBHOOK_ENABLED:-false}"
export TRACCAR_WEBHOOK_URL="${TRACCAR_WEBHOOK_URL:-}"

# WebSocket Configuration
export TRACCAR_WEBSOCKET_ENABLED="${TRACCAR_WEBSOCKET_ENABLED:-true}"