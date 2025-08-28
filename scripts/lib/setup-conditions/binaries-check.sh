#!/usr/bin/env bash
################################################################################
# Setup Condition: Binary Files Check
# 
# Checks if specified binary files exist and are executable
# Used primarily for compiled languages (Go, Rust, C++)
#
# Input: JSON object with "targets" array
# Returns: 0 if setup needed (binaries missing), 1 if all present
################################################################################

set -euo pipefail

# Get check configuration from argument
CHECK_CONFIG="${1:-{}}"

# Extract targets array
TARGETS=$(echo "$CHECK_CONFIG" | jq -r '.targets[]?' 2>/dev/null || echo "")

if [[ -z "$TARGETS" ]]; then
    # No targets specified - consider setup not needed
    exit 1
fi

# Check each target binary
MISSING_COUNT=0
while IFS= read -r target; do
    if [[ -z "$target" ]]; then
        continue
    fi
    
    # Resolve path relative to APP_ROOT if not absolute
    if [[ ! "$target" =~ ^/ ]]; then
        target="${APP_ROOT:-$(pwd)}/$target"
    fi
    
    # Check if file exists and is executable
    if [[ ! -f "$target" ]] || [[ ! -x "$target" ]]; then
        echo "[DEBUG] Binary missing or not executable: $target" >&2
        ((MISSING_COUNT++))
    fi
done <<< "$TARGETS"

# Return 0 if any binaries are missing (setup needed)
[[ $MISSING_COUNT -eq 0 ]] && exit 1 || exit 0