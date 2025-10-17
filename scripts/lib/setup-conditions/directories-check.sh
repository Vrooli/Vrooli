#!/usr/bin/env bash
################################################################################
# Setup Condition: Directories Check
# 
# Checks for existence of directories
# Similar to files-check.sh but specifically for directories
#
# Input: JSON object with "targets" array of directory paths
# Returns: 0 if setup needed (directories missing), 1 if all exist
################################################################################

set -euo pipefail

# Get check configuration from argument
CHECK_CONFIG="${1:-{}}"
APP_ROOT="${APP_ROOT:-$(pwd)}"

# Extract targets (directory paths)
TARGETS=$(echo "$CHECK_CONFIG" | jq -r '.targets[]?' 2>/dev/null || echo "")

if [[ -z "$TARGETS" ]]; then
    # No targets specified - consider setup not needed
    exit 1
fi

# Check each directory
MISSING_COUNT=0
while IFS= read -r dir_path; do
    if [[ -z "$dir_path" ]]; then
        continue
    fi
    
    # Resolve path relative to APP_ROOT if not absolute
    if [[ ! "$dir_path" =~ ^/ ]]; then
        dir_path="${APP_ROOT}/$dir_path"
    fi
    
    # Check if directory exists
    if [[ ! -d "$dir_path" ]]; then
        MISSING_COUNT=$((MISSING_COUNT + 1))
        if [[ "${SETUP_DEBUG:-false}" == "true" ]]; then
            echo "Missing directory: $dir_path" >&2
        fi
    fi
done <<< "$TARGETS"

# Return 0 if any directories are missing (setup needed)
# Return 1 if all directories exist (no setup needed)
if [[ $MISSING_COUNT -gt 0 ]]; then
    exit 0  # Setup needed
else
    exit 1  # Setup not needed
fi