#!/usr/bin/env bash
# farmOS Resource Configuration Defaults
# This file defines default environment variables for the farmOS resource

# Service configuration
export FARMOS_PORT="${FARMOS_PORT:-8004}"
export FARMOS_HOST="${FARMOS_HOST:-localhost}"
export FARMOS_SCHEME="${FARMOS_SCHEME:-http}"
export FARMOS_BASE_URL="${FARMOS_BASE_URL:-${FARMOS_SCHEME}://${FARMOS_HOST}:${FARMOS_PORT}}"

# Admin credentials
export FARMOS_ADMIN_USER="${FARMOS_ADMIN_USER:-admin}"
export FARMOS_ADMIN_PASSWORD="${FARMOS_ADMIN_PASSWORD:-admin}"
export FARMOS_ADMIN_EMAIL="${FARMOS_ADMIN_EMAIL:-admin@vrooli.local}"

# Database configuration
export FARMOS_DB_HOST="${FARMOS_DB_HOST:-farmos-db}"
export FARMOS_DB_PORT="${FARMOS_DB_PORT:-5432}"
export FARMOS_DB_NAME="${FARMOS_DB_NAME:-farmos}"
export FARMOS_DB_USER="${FARMOS_DB_USER:-farmos}"
export FARMOS_DB_PASSWORD="${FARMOS_DB_PASSWORD:-farmos_secure_pass}"

# Feature flags
export FARMOS_DEMO_DATA="${FARMOS_DEMO_DATA:-true}"
export FARMOS_OAUTH_ENABLED="${FARMOS_OAUTH_ENABLED:-true}"
export FARMOS_API_ENABLED="${FARMOS_API_ENABLED:-true}"

# API configuration
export FARMOS_API_VERSION="${FARMOS_API_VERSION:-2.x}"
export FARMOS_API_BASE="${FARMOS_API_BASE:-${FARMOS_BASE_URL}/api}"

# Docker configuration
export FARMOS_DOCKER_IMAGE="${FARMOS_DOCKER_IMAGE:-farmos/farmos:3.x}"
export FARMOS_COMPOSE_FILE="${FARMOS_COMPOSE_FILE:-${RESOURCE_DIR}/docker/docker-compose.yml}"

# Timeout configuration
export FARMOS_STARTUP_TIMEOUT="${FARMOS_STARTUP_TIMEOUT:-120}"
export FARMOS_STOP_TIMEOUT="${FARMOS_STOP_TIMEOUT:-30}"
export FARMOS_HEALTH_CHECK_INTERVAL="${FARMOS_HEALTH_CHECK_INTERVAL:-5}"

# IoT Integration
export FARMOS_MQTT_ENABLED="${FARMOS_MQTT_ENABLED:-false}"
export FARMOS_MQTT_BROKER="${FARMOS_MQTT_BROKER:-}"
export FARMOS_MQTT_PORT="${FARMOS_MQTT_PORT:-1883}"
export FARMOS_MQTT_TOPIC="${FARMOS_MQTT_TOPIC:-farmos/sensors/#}"

# Integration with other Vrooli resources
export FARMOS_QUESTDB_ENABLED="${FARMOS_QUESTDB_ENABLED:-false}"
export FARMOS_REDIS_ENABLED="${FARMOS_REDIS_ENABLED:-false}"
export FARMOS_NODERED_ENABLED="${FARMOS_NODERED_ENABLED:-false}"