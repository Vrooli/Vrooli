#!/usr/bin/env bash
################################################################################
# Twilio Resource Configuration Defaults
# 
# Cloud communications platform for SMS, voice, and video
################################################################################

# Resource constants
export TWILIO_NAME="twilio"
export TWILIO_CATEGORY="execution"
export TWILIO_CONFIG_DIR="${var_vrooli_dir:-${HOME}/.vrooli}/twilio"
export TWILIO_DATA_DIR="${var_vrooli_data_dir:-${HOME}/.vrooli/data}/twilio"
export TWILIO_CREDENTIALS_FILE="${var_vrooli_dir:-${HOME}/.vrooli}/twilio-credentials.json"
export TWILIO_MONITOR_PID_FILE="${TWILIO_DATA_DIR}/monitor.pid"
export TWILIO_LOG_FILE="${TWILIO_DATA_DIR}/twilio.log"
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