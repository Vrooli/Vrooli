#!/usr/bin/env bash
################################################################################
# Zigbee2MQTT Default Configuration
# 
# Default values for Zigbee2MQTT resource configuration
################################################################################

# Service Configuration
export ZIGBEE2MQTT_VERSION="${ZIGBEE2MQTT_VERSION:-latest}"
export ZIGBEE2MQTT_PORT="${ZIGBEE2MQTT_PORT:-8090}"
export ZIGBEE2MQTT_CONTAINER="${ZIGBEE2MQTT_CONTAINER:-zigbee2mqtt}"
export ZIGBEE2MQTT_DATA_DIR="${ZIGBEE2MQTT_DATA_DIR:-${VROOLI_ROOT:-${HOME}/Vrooli}/data/zigbee2mqtt}"

# MQTT Configuration
export MQTT_HOST="${MQTT_HOST:-localhost}"
export MQTT_PORT="${MQTT_PORT:-1883}"
export MQTT_USER="${MQTT_USER:-}"
export MQTT_PASS="${MQTT_PASS:-}"
export MQTT_BASE_TOPIC="${MQTT_BASE_TOPIC:-zigbee2mqtt}"

# Zigbee Configuration
export ZIGBEE_ADAPTER="${ZIGBEE_ADAPTER:-/dev/ttyACM0}"
export ZIGBEE_ADAPTER_TYPE="${ZIGBEE_ADAPTER_TYPE:-auto}"
export ZIGBEE_CHANNEL="${ZIGBEE_CHANNEL:-11}"
export ZIGBEE_PAN_ID="${ZIGBEE_PAN_ID:-GENERATE}"
export ZIGBEE_NETWORK_KEY="${ZIGBEE_NETWORK_KEY:-GENERATE}"

# Advanced Configuration
export Z2M_PERMIT_JOIN_TIMEOUT="${Z2M_PERMIT_JOIN_TIMEOUT:-120}"
export Z2M_AVAILABILITY_TIMEOUT="${Z2M_AVAILABILITY_TIMEOUT:-10}"
export Z2M_LOG_LEVEL="${Z2M_LOG_LEVEL:-info}"
export Z2M_HOMEASSISTANT="${Z2M_HOMEASSISTANT:-true}"

# Frontend Configuration
export Z2M_FRONTEND_PORT="${Z2M_FRONTEND_PORT:-8080}"
export Z2M_FRONTEND_HOST="${Z2M_FRONTEND_HOST:-0.0.0.0}"
export Z2M_FRONTEND_AUTH="${Z2M_FRONTEND_AUTH:-false}"

# Performance Configuration
export Z2M_CACHE_STATE="${Z2M_CACHE_STATE:-true}"
export Z2M_CACHE_STATE_PERSISTENT="${Z2M_CACHE_STATE_PERSISTENT:-true}"
export Z2M_CACHE_STATE_SEND_ON_STARTUP="${Z2M_CACHE_STATE_SEND_ON_STARTUP:-true}"

# Experimental Features
export Z2M_EXPERIMENTAL="${Z2M_EXPERIMENTAL:-false}"
export Z2M_TRANSMIT_POWER="${Z2M_TRANSMIT_POWER:-}"