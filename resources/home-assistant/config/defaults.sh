#!/bin/bash
# Home Assistant Configuration Defaults

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"

# Container settings
export HOME_ASSISTANT_CONTAINER_NAME="${HOME_ASSISTANT_CONTAINER_NAME:-home-assistant}"
export HOME_ASSISTANT_IMAGE="${HOME_ASSISTANT_IMAGE:-homeassistant/home-assistant:stable}"

# Network settings  
export HOME_ASSISTANT_PORT="${HOME_ASSISTANT_PORT:-8123}"
export HOME_ASSISTANT_BASE_URL="${HOME_ASSISTANT_BASE_URL:-http://localhost:${HOME_ASSISTANT_PORT}}"

# Canonical resource storage directories.
# RESOURCE_* is injected by the Go control plane; XDG fallbacks keep standalone shell usage off repo-local paths.
home_assistant_xdg_config_home="${XDG_CONFIG_HOME:-${HOME}/.config}"
home_assistant_xdg_data_home="${XDG_DATA_HOME:-${HOME}/.local/share}"

export HOME_ASSISTANT_DATA_DIR="${HOME_ASSISTANT_DATA_DIR:-${RESOURCE_DATA_DIR:-${home_assistant_xdg_data_home}/vrooli/resources/home-assistant}}"
export HOME_ASSISTANT_CONFIG_DIR="${HOME_ASSISTANT_CONFIG_DIR:-${RESOURCE_CONFIG_DIR:-${home_assistant_xdg_config_home}/vrooli/resources/home-assistant}}"

# Runtime settings
export HOME_ASSISTANT_TIME_ZONE="${HOME_ASSISTANT_TIME_ZONE:-America/New_York}"
export HOME_ASSISTANT_RESTART_POLICY="${HOME_ASSISTANT_RESTART_POLICY:-unless-stopped}"

# Installation settings
export HOME_ASSISTANT_INSTALL_TIMEOUT="${HOME_ASSISTANT_INSTALL_TIMEOUT:-300}"
export HOME_ASSISTANT_HEALTH_CHECK_TIMEOUT="${HOME_ASSISTANT_HEALTH_CHECK_TIMEOUT:-60}"
export HOME_ASSISTANT_HEALTH_CHECK_INTERVAL="${HOME_ASSISTANT_HEALTH_CHECK_INTERVAL:-5}"
