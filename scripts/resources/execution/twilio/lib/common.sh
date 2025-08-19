#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
TWILIO_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TWILIO_DIR="$(dirname "$TWILIO_LIB_DIR")"

# Source utilities
source "$TWILIO_DIR/../../../lib/utils/var.sh"
source "$TWILIO_DIR/../../../lib/utils/format.sh"
source "$TWILIO_DIR/../../../lib/utils/log.sh"
source "$TWILIO_DIR/../../port_registry.sh"

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

# Create directories if needed
twilio::ensure_dirs() {
    mkdir -p "$TWILIO_CONFIG_DIR"
    mkdir -p "$TWILIO_DATA_DIR"
    mkdir -p "$TWILIO_WORKFLOWS_DIR"
}

# Check if Twilio CLI is installed
twilio::is_installed() {
    # Check global install first
    if command -v twilio >/dev/null 2>&1; then
        return 0
    fi
    # Check local install
    if [[ -x "$TWILIO_DATA_DIR/node_modules/.bin/twilio" ]]; then
        return 0
    fi
    return 1
}

# Get Twilio command path
twilio::get_command() {
    if command -v twilio >/dev/null 2>&1; then
        echo "twilio"
    elif [[ -x "$TWILIO_DATA_DIR/node_modules/.bin/twilio" ]]; then
        echo "$TWILIO_DATA_DIR/node_modules/.bin/twilio"
    else
        return 1
    fi
}

# Get Twilio version
twilio::get_version() {
    if twilio::is_installed; then
        local cmd
        cmd=$(twilio::get_command)
        "$cmd" --version 2>/dev/null | head -1 || echo "unknown"
    else
        echo "not installed"
    fi
}

# Check if credentials are configured
twilio::has_credentials() {
    [[ -f "$TWILIO_CREDENTIALS_FILE" ]] && \
    [[ -n "$(jq -r '.account_sid // empty' "$TWILIO_CREDENTIALS_FILE" 2>/dev/null)" ]] && \
    [[ -n "$(jq -r '.auth_token // empty' "$TWILIO_CREDENTIALS_FILE" 2>/dev/null)" ]]
}

# Get account SID
twilio::get_account_sid() {
    if [[ -f "$TWILIO_CREDENTIALS_FILE" ]]; then
        jq -r '.account_sid // empty' "$TWILIO_CREDENTIALS_FILE" 2>/dev/null || echo ""
    else
        echo ""
    fi
}

# Get auth token
twilio::get_auth_token() {
    if [[ -f "$TWILIO_CREDENTIALS_FILE" ]]; then
        jq -r '.auth_token // empty' "$TWILIO_CREDENTIALS_FILE" 2>/dev/null || echo ""
    else
        echo ""
    fi
}

# Get default phone number
twilio::get_default_number() {
    if [[ -f "$TWILIO_PHONE_NUMBERS_FILE" ]]; then
        jq -r '.numbers[0].number // empty' "$TWILIO_PHONE_NUMBERS_FILE" 2>/dev/null || echo ""
    else
        echo ""
    fi
}

# Check if monitor is running
twilio::is_monitor_running() {
    if [[ -f "$TWILIO_MONITOR_PID_FILE" ]]; then
        local pid
        pid=$(cat "$TWILIO_MONITOR_PID_FILE")
        if kill -0 "$pid" 2>/dev/null; then
            return 0
        fi
    fi
    return 1
}

# Set up Twilio CLI auth
twilio::setup_auth() {
    if ! twilio::has_credentials; then
        return 1
    fi
    
    local account_sid auth_token
    account_sid=$(twilio::get_account_sid)
    auth_token=$(twilio::get_auth_token)
    
    # Export for CLI to use
    export TWILIO_ACCOUNT_SID="$account_sid"
    export TWILIO_AUTH_TOKEN="$auth_token"
    
    return 0
}

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