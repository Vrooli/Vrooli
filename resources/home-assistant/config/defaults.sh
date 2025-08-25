#!/bin/bash
# Home Assistant Configuration Defaults

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Container settings
export HOME_ASSISTANT_CONTAINER_NAME="${HOME_ASSISTANT_CONTAINER_NAME:-home-assistant}"
export HOME_ASSISTANT_IMAGE="${HOME_ASSISTANT_IMAGE:-homeassistant/home-assistant:stable}"

# Network settings  
export HOME_ASSISTANT_PORT="${HOME_ASSISTANT_PORT:-8123}"
export HOME_ASSISTANT_BASE_URL="${HOME_ASSISTANT_BASE_URL:-http://localhost:${HOME_ASSISTANT_PORT}}"

# Data directories
export HOME_ASSISTANT_DATA_DIR="${HOME_ASSISTANT_DATA_DIR:-${var_DATA_DIR}/resources/home-assistant}"
export HOME_ASSISTANT_CONFIG_DIR="${HOME_ASSISTANT_CONFIG_DIR:-${HOME_ASSISTANT_DATA_DIR}/config}"

# Runtime settings
export HOME_ASSISTANT_TIME_ZONE="${HOME_ASSISTANT_TIME_ZONE:-America/New_York}"
export HOME_ASSISTANT_RESTART_POLICY="${HOME_ASSISTANT_RESTART_POLICY:-unless-stopped}"

# Installation settings
export HOME_ASSISTANT_INSTALL_TIMEOUT="${HOME_ASSISTANT_INSTALL_TIMEOUT:-300}"
export HOME_ASSISTANT_HEALTH_CHECK_TIMEOUT="${HOME_ASSISTANT_HEALTH_CHECK_TIMEOUT:-60}"
export HOME_ASSISTANT_HEALTH_CHECK_INTERVAL="${HOME_ASSISTANT_HEALTH_CHECK_INTERVAL:-5}"