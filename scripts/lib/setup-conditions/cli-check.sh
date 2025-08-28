#!/usr/bin/env bash
################################################################################
# Setup Condition: CLI Command Check
# 
# Checks if specified CLI commands are available in PATH
# Used for verifying CLI tools are installed
#
# Input: JSON object with "command" field
# Returns: 0 if setup needed (command missing), 1 if available
################################################################################

set -euo pipefail

# Get check configuration from argument
CHECK_CONFIG="${1:-{}}"

# Extract command to check
COMMAND=$(echo "$CHECK_CONFIG" | jq -r '.command // empty' 2>/dev/null)

if [[ -z "$COMMAND" ]]; then
    # No command specified - consider setup not needed
    exit 1
fi

# Check if command is available
if command -v "$COMMAND" &>/dev/null; then
    # Command found - no setup needed
    exit 1
else
    echo "[DEBUG] CLI command not found: $COMMAND" >&2
    # Command missing - setup needed
    exit 0
fi