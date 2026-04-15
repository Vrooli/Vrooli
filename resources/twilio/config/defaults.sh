#!/usr/bin/env bash
################################################################################
# Twilio Resource Configuration Defaults
# 
# Cloud communications platform for SMS, voice, and video
################################################################################

# Resource constants
export TWILIO_NAME="twilio"
export TWILIO_CATEGORY="execution"
twilio_xdg_config_home="${XDG_CONFIG_HOME:-${HOME}/.config}"
twilio_xdg_data_home="${XDG_DATA_HOME:-${HOME}/.local/share}"
twilio_xdg_state_home="${XDG_STATE_HOME:-${HOME}/.local/state}"
export TWILIO_CONFIG_DIR="${RESOURCE_CONFIG_DIR:-${twilio_xdg_config_home}/vrooli/resources/twilio}"
export TWILIO_DATA_DIR="${RESOURCE_DATA_DIR:-${twilio_xdg_data_home}/vrooli/resources/twilio}"
export TWILIO_STATE_DIR="${RESOURCE_STATE_DIR:-${twilio_xdg_state_home}/vrooli/resources/twilio}"
export TWILIO_CREDENTIALS_FILE="${TWILIO_CONFIG_DIR}/credentials.json"
export TWILIO_MONITOR_PID_FILE="${TWILIO_STATE_DIR}/monitor.pid"
export TWILIO_LOG_FILE="${TWILIO_STATE_DIR}/twilio.log"
export TWILIO_PHONE_NUMBERS_FILE="${TWILIO_CONFIG_DIR}/phone-numbers.json"
export TWILIO_WORKFLOWS_DIR="${TWILIO_CONFIG_DIR}/workflows"

# Messages
export MSG_TWILIO_INSTALLING="Installing Twilio CLI..."
export MSG_TWILIO_INSTALLED="Twilio CLI installed successfully"
export MSG_TWILIO_ALREADY_INSTALLED="Twilio CLI is already installed"
export MSG_TWILIO_INSTALL_FAILED="Failed to install Twilio CLI"
export MSG_TWILIO_NO_CREDENTIALS="Twilio credentials not configured"
export MSG_TWILIO_CONFIGURED="Twilio configured successfully"
export MSG_TWILIO_MONITOR_STARTED="Twilio monitor started"
export MSG_TWILIO_MONITOR_STOPPED="Twilio monitor stopped"
export MSG_TWILIO_NOT_INSTALLED="Twilio CLI not installed"
