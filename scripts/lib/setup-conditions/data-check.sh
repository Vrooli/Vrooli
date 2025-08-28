#!/usr/bin/env bash
################################################################################
# Setup Condition: Data Directory Check
# 
# Checks if data directory exists with expected content
# Used as a simple "first run" detection
#
# Input: JSON object with optional "path" field
# Returns: 0 if setup needed (data missing), 1 if exists
################################################################################

set -euo pipefail

# Get check configuration from argument
CHECK_CONFIG="${1:-{}}"
APP_ROOT="${APP_ROOT:-$(pwd)}"

# Extract path to check (default: data/)
DATA_PATH=$(echo "$CHECK_CONFIG" | jq -r '.path // "data"' 2>/dev/null)

# Resolve path relative to APP_ROOT if not absolute
if [[ ! "$DATA_PATH" =~ ^/ ]]; then
    DATA_PATH="${APP_ROOT}/$DATA_PATH"
fi

# Check if data directory exists and has content
if [[ -d "$DATA_PATH" ]] && [[ "$(ls -A "$DATA_PATH" 2>/dev/null)" ]]; then
    # Data directory exists with content - no setup needed
    exit 1
else
    echo "[DEBUG] Data directory missing or empty: $DATA_PATH" >&2
    # Data missing - setup needed
    exit 0
fi